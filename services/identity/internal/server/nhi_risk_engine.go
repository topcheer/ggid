package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NHIBehaviorBaseline records the normal usage pattern for an NHI per endpoint.
type NHIBehaviorBaseline struct {
	NHIID           string    `json:"nhi_id"`
	Endpoint        string    `json:"endpoint"`
	AvgCallsPerHour float64   `json:"avg_calls_per_hour"`
	StdCallsPerHour float64   `json:"std_calls_per_hour"`
	KnownIPs        []string  `json:"known_ips"`
	KnownHours      []int     `json:"known_hours"` // 0-23 hours seen before
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	TotalCalls      int64     `json:"total_calls"`
}

// NHIRiskScore represents the evaluated risk for an NHI.
type NHIRiskScore struct {
	NHIID       uuid.UUID      `json:"nhi_id"`
	Score       int            `json:"score"`        // 0-100
	Level       string         `json:"level"`         // low/medium/high/critical
	Signals     map[string]any `json:"signals"`       // detected anomaly signals
	EvaluatedAt time.Time      `json:"evaluated_at"`
}

// NHIRiskEngine evaluates NHI behavior against baselines.
// Uses PG-backed repo when configured (pool != nil), falls back to
// in-memory maps when no DB is available (tests/dev).
type NHIRiskEngine struct {
	mu        sync.RWMutex
	baselines map[string][]*NHIBehaviorBaseline // nhi_id → baselines
	scores    map[uuid.UUID]*NHIRiskScore       // nhi_id → latest score
	pgRepo    *NHIRiskPGRepo                     // PG persistence (nil = in-memory)
}

func NewNHIRiskEngine() *NHIRiskEngine {
	return &NHIRiskEngine{
		baselines: make(map[string][]*NHIBehaviorBaseline),
		scores:    make(map[uuid.UUID]*NHIRiskScore),
	}
}

// SetPGRepo wires a PostgreSQL-backed repo for persistent storage.
func (e *NHIRiskEngine) SetPGRepo(repo *NHIRiskPGRepo) {
	e.pgRepo = repo
}

// EnsureSchema creates DB tables if PG repo is configured.
func (e *NHIRiskEngine) EnsureSchema(ctx context.Context) error {
	if e.pgRepo != nil {
		return e.pgRepo.EnsureSchema(ctx)
	}
	return nil
}

// RecordBaseline adds or updates a behavior baseline entry for an NHI.
func (e *NHIRiskEngine) RecordBaseline(nhiID, endpoint string, callsPerHour float64, ip string, hour int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find or create baseline for this endpoint.
	for _, b := range e.baselines[nhiID] {
		if b.Endpoint == endpoint {
			b.TotalCalls++
			b.AvgCallsPerHour = (b.AvgCallsPerHour*float64(b.TotalCalls-1) + callsPerHour) / float64(b.TotalCalls)
			b.LastSeen = time.Now()
			// Add IP if new.
			if !containsStrSlice(b.KnownIPs, ip) {
				b.KnownIPs = append(b.KnownIPs, ip)
			}
			// Add hour if new.
			if !containsIntSlice(b.KnownHours, hour) {
				b.KnownHours = append(b.KnownHours, hour)
			}
			return
		}
	}

	// New baseline.
	if e.baselines[nhiID] == nil {
		e.baselines[nhiID] = []*NHIBehaviorBaseline{}
	}
	e.baselines[nhiID] = append(e.baselines[nhiID], &NHIBehaviorBaseline{
		NHIID:           nhiID,
		Endpoint:        endpoint,
		AvgCallsPerHour: callsPerHour,
		KnownIPs:        []string{ip},
		KnownHours:      []int{hour},
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		TotalCalls:      1,
	})

	// Persist new baseline to PG.
	if e.pgRepo != nil {
		rec := e.baselines[nhiID][len(e.baselines[nhiID])-1]
		_ = e.pgRepo.SaveBaseline(context.Background(), rec)
	}
}

// RiskSignals represents detected anomalies for an evaluation.
type RiskSignals struct {
	FrequencySpike   bool    `json:"frequency_spike"`
	NewEndpoint      bool    `json:"new_endpoint"`
	OffHoursAccess   bool    `json:"off_hours_access"`
	NewIP            bool    `json:"new_ip"`
	SpikeRatio       float64 `json:"spike_ratio,omitempty"`
	UnexpectedEndpoint string  `json:"unexpected_endpoint,omitempty"`
	OffHour          int     `json:"off_hour,omitempty"`
	UnexpectedIP     string  `json:"unexpected_ip,omitempty"`
}

// EvaluateRisk computes a risk score (0-100) for an NHI based on current
// activity compared to established baselines.
func (e *NHIRiskEngine) EvaluateRisk(nhiID uuid.UUID, currentActivity CurrentActivity) *NHIRiskScore {
	e.mu.RLock()
	baselines := e.baselines[currentActivity.NHIID]
	e.mu.RUnlock()

	signals := RiskSignals{}
	score := 0

	// If no baseline exists, this is a new NHI — moderate risk.
	if len(baselines) == 0 {
		return &NHIRiskScore{
			NHIID:       nhiID,
			Score:       20,
			Level:       riskLevel(20),
			Signals:     map[string]any{"no_baseline": true, "message": "new NHI with no behavior baseline"},
			EvaluatedAt: time.Now(),
		}
	}

	// 1. Frequency spike detection.
	for _, b := range baselines {
		if b.Endpoint == currentActivity.Endpoint {
			if b.AvgCallsPerHour > 0 {
				ratio := currentActivity.CallsPerHour / b.AvgCallsPerHour
				if ratio > 5 {
					signals.FrequencySpike = true
				signals.SpikeRatio = ratio
					score += 30
				}
			}
			break
		}
	}

	// 2. New endpoint detection.
	endpointKnown := false
	for _, b := range baselines {
		if b.Endpoint == currentActivity.Endpoint {
			endpointKnown = true
			break
		}
	}
	if !endpointKnown {
		signals.NewEndpoint = true
		signals.UnexpectedEndpoint = currentActivity.Endpoint
		score += 25
	}

	// 3. Off-hours access detection.
	hourKnown := false
	for _, b := range baselines {
		for _, h := range b.KnownHours {
			if h == currentActivity.Hour {
				hourKnown = true
				break
			}
		}
		if hourKnown {
			break
		}
	}
	if !hourKnown && (currentActivity.Hour < 6 || currentActivity.Hour >= 22) {
		signals.OffHoursAccess = true
		signals.OffHour = currentActivity.Hour
		score += 20
	}

	// 4. New IP detection.
	ipKnown := false
	for _, b := range baselines {
		for _, ip := range b.KnownIPs {
			if ip == currentActivity.IP {
				ipKnown = true
				break
			}
		}
		if ipKnown {
			break
		}
	}
	if !ipKnown {
		signals.NewIP = true
		signals.UnexpectedIP = currentActivity.IP
		score += 15
	}

	if score > 100 {
		score = 100
	}

	signalsMap := map[string]any{}
	signalsJSON, _ := json.Marshal(signals)
	_ = json.Unmarshal(signalsJSON, &signalsMap)

	result := &NHIRiskScore{
		NHIID:       nhiID,
		Score:       score,
		Level:       riskLevel(score),
		Signals:     signalsMap,
		EvaluatedAt: time.Now(),
	}

	e.mu.Lock()
	e.scores[nhiID] = result
	e.mu.Unlock()

	// Persist to PG if configured.
	if e.pgRepo != nil {
		_ = e.pgRepo.SaveRiskScore(context.Background(), result)
	}

	return result
}

// GetRiskScore returns the latest risk score for an NHI.
// Tries PG first, falls back to in-memory.
func (e *NHIRiskEngine) GetRiskScore(nhiID uuid.UUID) *NHIRiskScore {
	if e.pgRepo != nil {
		if score, err := e.pgRepo.GetRiskScore(context.Background(), nhiID); err == nil && score != nil {
			return score
		}
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.scores[nhiID]
}

// ListHighRisk returns all NHIs with score >= threshold.
// Queries PG if configured, otherwise scans in-memory.
func (e *NHIRiskEngine) ListHighRisk(threshold int) []*NHIRiskScore {
	if e.pgRepo != nil {
		if high, err := e.pgRepo.ListHighRisk(context.Background(), threshold); err == nil && high != nil {
			return high
		}
		// PG returned nil/error — fall through to in-memory.
	}
	e.mu.RLock()
	defer e.mu.RUnlock()

	var high []*NHIRiskScore
	for _, s := range e.scores {
		if s.Score >= threshold {
			high = append(high, s)
		}
	}
	return high
}

// CurrentActivity represents the current activity being evaluated.
type CurrentActivity struct {
	NHIID         string  `json:"nhi_id"`
	Endpoint      string  `json:"endpoint"`
	CallsPerHour  float64 `json:"calls_per_hour"`
	IP            string  `json:"ip"`
	Hour          int     `json:"hour"` // 0-23
}

// riskLevel converts a numeric score to a risk level.
func riskLevel(score int) string {
	switch {
	case score >= 70:
		return "critical"
	case score >= 50:
		return "high"
	case score >= 25:
		return "medium"
	default:
		return "low"
	}
}

func containsStrSlice(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func containsIntSlice(slice []int, val int) bool {
	for _, i := range slice {
		if i == val {
			return true
		}
	}
	return false
}
