package ueba

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BehavioralSample represents a single user event for model training.
type BehavioralSample struct {
	Hour       float64 `json:"hour"`        // 0-23
	DayOfWeek  float64 `json:"day_of_week"` // 0-6
	IPHash     float64 `json:"ip_hash"`     // normalized hash 0-1
	DeviceHash float64 `json:"device_hash"` // normalized hash 0-1
	GeoLat     float64 `json:"geo_lat"`     // latitude normalized -1 to 1
	GeoLng     float64 `json:"geo_lng"`     // longitude normalized -1 to 1
	IsNewIP    float64 `json:"is_new_ip"`   // 0 or 1
	IsNewDevice float64 `json:"is_new_device"` // 0 or 1
}

// Features returns the behavioral sample as a feature vector.
func (s BehavioralSample) Features() []float64 {
	return []float64{s.Hour / 23.0, s.DayOfWeek / 6.0, s.IPHash, s.DeviceHash, s.GeoLat, s.GeoLng, s.IsNewIP, s.IsNewDevice}
}

// NumFeatures returns the number of features.
const NumFeatures = 8

// IsolationTree is a single isolation tree node.
type IsolationTree struct {
	SplitIndex int     // feature index to split on (-1 = leaf)
	SplitValue float64 // threshold
	Left       *IsolationTree
	Right      *IsolationTree
	Size       int // number of samples at this node (leaf only)
}

// IsolationForest is an ensemble of isolation trees for anomaly detection.
type IsolationForest struct {
	Trees       []*IsolationTree
	SubsampleSize int
	NumTrees     int
}

// BehavioralBaseline is the trained model for a user.
type BehavioralBaseline struct {
	UserID      uuid.UUID       `json:"user_id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Forest      *IsolationForest `json:"-"`
	SampleCount int             `json:"sample_count"`
	TrainedAt   time.Time       `json:"trained_at"`
	Stats       BaselineStats   `json:"stats"`
}

// BaselineStats summarizes the training data distribution.
type BaselineStats struct {
	MeanHour     float64 `json:"mean_hour"`
	StdHour      float64 `json:"std_hour"`
	UniqueIPs    int     `json:"unique_ips"`
	UniqueDevices int    `json:"unique_devices"`
	CommonHours  []int   `json:"common_hours"`
}

// Engine manages per-user isolation forest models.
type Engine struct {
	pool     *pgxpool.Pool
	mu       sync.RWMutex
	baselines map[string]*BehavioralBaseline // key: tenant:user
	rng      *rand.Rand
}

// NewEngine creates a new UEBA engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		pool:      pool,
		baselines: make(map[string]*BehavioralBaseline),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// EnsureSchema creates the user_behavioral_baselines table.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_behavioral_baselines (
			user_id UUID NOT NULL,
			tenant_id UUID NOT NULL,
			model_params JSONB NOT NULL DEFAULT '{}',
			sample_count INT NOT NULL DEFAULT 0,
			trained_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (tenant_id, user_id)
		);
	`)
	return err
}

// Train builds an isolation forest from behavioral samples.
// Requires at least minSamples (default 30) — falls back to nil model with stats only.
func (e *Engine) Train(tenantID, userID uuid.UUID, samples []BehavioralSample) *BehavioralBaseline {
	stats := computeStats(samples)
	baseline := &BehavioralBaseline{
		UserID:      userID,
		TenantID:    tenantID,
		SampleCount: len(samples),
		TrainedAt:   time.Now(),
		Stats:       stats,
	}

	// Need at least 30 samples for isolation forest.
	if len(samples) >= 30 {
		subsampleSize := 256
		if len(samples) < subsampleSize {
			subsampleSize = len(samples)
		}
		numTrees := 100

		forest := &IsolationForest{
			SubsampleSize: subsampleSize,
			NumTrees:      numTrees,
		}

		e.mu.Lock()
		for i := 0; i < numTrees; i++ {
			subsample := e.randomSubsample(samples, subsampleSize)
			tree := e.buildTree(subsample, 0, int(math.Ceil(math.Log2(float64(subsampleSize)))))
			forest.Trees = append(forest.Trees, tree)
		}
		e.mu.Unlock()

		baseline.Forest = forest
	}

	key := tenantID.String() + ":" + userID.String()
	e.mu.Lock()
	e.baselines[key] = baseline
	e.mu.Unlock()

	return baseline
}

// Score evaluates a sample against the user's isolation forest.
// Returns anomaly score 0-1 (1 = highly anomalous).
func (e *Engine) Score(tenantID, userID uuid.UUID, sample BehavioralSample) float64 {
	key := tenantID.String() + ":" + userID.String()
	e.mu.RLock()
	baseline, ok := e.baselines[key]
	e.mu.RUnlock()

	if !ok || baseline == nil || baseline.Forest == nil {
		// Fallback: use 3-sigma on hour.
		return e.sigmaScore(sample, baseline)
	}

	// Isolation forest scoring: average path length → anomaly score.
	features := sample.Features()
	totalPathLength := 0.0
	for _, tree := range baseline.Forest.Trees {
		pathLen := e.pathLength(tree, features, 0)
		totalPathLength += pathLen
	}
	avgPath := totalPathLength / float64(len(baseline.Forest.Trees))

	// Anomaly score: s = 2^(-E(h(x)) / c(n))
	// where c(n) = average path length of unsuccessful BST search
	cn := avgPathLengthFunc(baseline.Forest.SubsampleSize)
	score := math.Pow(2, -avgPath/cn)

	return score
}

// sigmaScore is the fallback 3-sigma deviation score.
func (e *Engine) sigmaScore(sample BehavioralSample, baseline *BehavioralBaseline) float64 {
	if baseline == nil || baseline.Stats.StdHour == 0 {
		return 0
	}
	deviation := math.Abs(sample.Hour - baseline.Stats.StdHour)
	if deviation > 3*baseline.Stats.StdHour {
		return 0.8
	} else if deviation > 2*baseline.Stats.StdHour {
		return 0.5
	}
	return 0.1
}

// buildTree recursively builds an isolation tree.
func (e *Engine) buildTree(samples []BehavioralSample, depth, maxDepth int) *IsolationTree {
	if depth >= maxDepth || len(samples) <= 1 {
		return &IsolationTree{SplitIndex: -1, Size: len(samples)}
	}

	// Pick random feature.
	featIdx := e.rng.Intn(NumFeatures)
	values := make([]float64, len(samples))
	for i, s := range samples {
		values[i] = s.Features()[featIdx]
	}
	sort.Float64s(values)

	// Random split point between min and max.
	minVal, maxVal := values[0], values[len(values)-1]
	if maxVal == minVal {
		return &IsolationTree{SplitIndex: -1, Size: len(samples)}
	}
	splitVal := minVal + e.rng.Float64()*(maxVal-minVal)

	var left, right []BehavioralSample
	for _, s := range samples {
		if s.Features()[featIdx] < splitVal {
			left = append(left, s)
		} else {
			right = append(right, s)
		}
	}

	if len(left) == 0 || len(right) == 0 {
		return &IsolationTree{SplitIndex: -1, Size: len(samples)}
	}

	return &IsolationTree{
		SplitIndex: featIdx,
		SplitValue: splitVal,
		Left:       e.buildTree(left, depth+1, maxDepth),
		Right:      e.buildTree(right, depth+1, maxDepth),
	}
}

// pathLength traverses the tree and returns the path length for a sample.
func (e *Engine) pathLength(tree *IsolationTree, features []float64, depth int) float64 {
	if tree.SplitIndex == -1 {
		// Leaf node: path length = depth + c(tree.Size)
		return float64(depth) + avgPathLengthFunc(tree.Size)
	}
	if features[tree.SplitIndex] < tree.SplitValue {
		return e.pathLength(tree.Left, features, depth+1)
	}
	return e.pathLength(tree.Right, features, depth+1)
}

// randomSubsample picks n random samples (with replacement).
func (e *Engine) randomSubsample(samples []BehavioralSample, n int) []BehavioralSample {
	result := make([]BehavioralSample, n)
	for i := 0; i < n; i++ {
		result[i] = samples[e.rng.Intn(len(samples))]
	}
	return result
}

// avgPathLengthFunc computes c(n) = 2*H(n-1) - 2*(n-1)/n
// where H(i) = ln(i) + 0.5772157 (Euler-Mascheroni constant).
func avgPathLengthFunc(n int) float64 {
	if n <= 1 {
		return 0
	}
	if n == 2 {
		return 1
	}
	hn := math.Log(float64(n-1)) + 0.5772156649
	return 2*hn - 2*float64(n-1)/float64(n)
}

// computeStats calculates summary statistics from samples.
func computeStats(samples []BehavioralSample) BaselineStats {
	if len(samples) == 0 {
		return BaselineStats{}
	}

	// Mean and std of login hour.
	sum := 0.0
	for _, s := range samples {
		sum += s.Hour
	}
	mean := sum / float64(len(samples))

	variance := 0.0
	for _, s := range samples {
		variance += (s.Hour - mean) * (s.Hour - mean)
	}
	std := math.Sqrt(variance / float64(len(samples)))

	// Unique IPs and devices.
	ipSet := map[float64]bool{}
	deviceSet := map[float64]bool{}
	for _, s := range samples {
		ipSet[s.IPHash] = true
		deviceSet[s.DeviceHash] = true
	}

	// Common hours (top 3).
	hourCount := make(map[int]int)
	for _, s := range samples {
		h := int(math.Round(s.Hour)) % 24
		hourCount[h]++
	}
	type hourFreq struct {
		hour int
		freq int
	}
	var freqs []hourFreq
	for h, f := range hourCount {
		freqs = append(freqs, hourFreq{h, f})
	}
	sort.Slice(freqs, func(i, j int) bool { return freqs[i].freq > freqs[j].freq })
	commonHours := []int{}
	for i := 0; i < 3 && i < len(freqs); i++ {
		commonHours = append(commonHours, freqs[i].hour)
	}

	return BaselineStats{
		MeanHour:      mean,
		StdHour:       std,
		UniqueIPs:     len(ipSet),
		UniqueDevices: len(deviceSet),
		CommonHours:   commonHours,
	}
}

// GetBaseline returns the current baseline for a user.
func (e *Engine) GetBaseline(tenantID, userID uuid.UUID) *BehavioralBaseline {
	key := tenantID.String() + ":" + userID.String()
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.baselines[key]
}

// PersistBaseline stores the baseline in PostgreSQL.
func (e *Engine) PersistBaseline(ctx context.Context, b *BehavioralBaseline) error {
	if e.pool == nil {
		return nil
	}
	statsJSON, _ := json.Marshal(b.Stats)
	_, err := e.pool.Exec(ctx, `
		INSERT INTO user_behavioral_baselines (user_id, tenant_id, model_params, sample_count, trained_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET model_params = EXCLUDED.model_params, sample_count = EXCLUDED.sample_count, trained_at = EXCLUDED.trained_at`,
		b.UserID, b.TenantID, statsJSON, b.SampleCount, b.TrainedAt)
	return err
}

// GenerateSamplesFromHours creates training samples from login hour data.
// Helper for converting audit events into behavioral samples.
func GenerateSamplesFromHours(hours []int, ipHashes, deviceHashes []float64) []BehavioralSample {
	var samples []BehavioralSample
	for i, h := range hours {
		ip := 0.5
		if i < len(ipHashes) {
			ip = ipHashes[i]
		}
		dev := 0.5
		if i < len(deviceHashes) {
			dev = deviceHashes[i]
		}
		samples = append(samples, BehavioralSample{
			Hour:       float64(h),
			DayOfWeek:  float64(i % 7),
			IPHash:     ip,
			DeviceHash: dev,
			IsNewIP:    0,
			IsNewDevice: 0,
		})
	}
	return samples
}
