package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ControlStatus represents the compliance state of a single control.
const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// NHIBehaviorBaseline records the normal usage pattern for an NHI per endpoint.
type NHIBehaviorBaseline struct {
	NHIID           string    `json:"nhi_id"`
	Endpoint        string    `json:"endpoint"`
	AvgCallsPerHour float64   `json:"avg_calls_per_hour"`
	StdCallsPerHour float64   `json:"std_calls_per_hour"`
	KnownIPs        []string  `json:"known_ips"`
	KnownHours      []int     `json:"known_hours"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
	TotalCalls      int64     `json:"total_calls"`
}

// NHIRiskScore represents the evaluated risk for an NHI.
type NHIRiskScore struct {
	NHIID       uuid.UUID      `json:"nhi_id"`
	Score       int            `json:"score"`
	Level       string         `json:"level"`
	Signals     map[string]any `json:"signals"`
	EvaluatedAt time.Time      `json:"evaluated_at"`
}

// NHIRiskEngine evaluates NHI behavior against baselines.
// Uses PG-backed repo for persistence (in-memory fallback in repo when nil pool).
type NHIRiskEngine struct {
	pgRepo *NHIRiskPGRepo
}

func NewNHIRiskEngine() *NHIRiskEngine {
	return &NHIRiskEngine{}
}

// SetPGRepo wires a PostgreSQL-backed repo.
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
// Persists via PG repo (in-memory fallback when pool is nil).
func (e *NHIRiskEngine) RecordBaseline(nhiID, endpoint string, callsPerHour float64, ip string, hour int) {
	b := &NHIBehaviorBaseline{
		NHIID:           nhiID,
		Endpoint:        endpoint,
		AvgCallsPerHour: callsPerHour,
		KnownIPs:        []string{ip},
		KnownHours:      []int{hour},
		FirstSeen:       time.Now(),
		LastSeen:        time.Now(),
		TotalCalls:      1,
	}
	if e.pgRepo != nil {
		_ = e.pgRepo.SaveBaseline(context.Background(), b)
	}
}

// RiskSignals represents detected anomalies for an evaluation.
type RiskSignals struct {
	FrequencySpike     bool    `json:"frequency_spike"`
	NewEndpoint        bool    `json:"new_endpoint"`
	OffHoursAccess     bool    `json:"off_hours_access"`
	NewIP              bool    `json:"new_ip"`
	SpikeRatio         float64 `json:"spike_ratio,omitempty"`
	UnexpectedEndpoint string  `json:"unexpected_endpoint,omitempty"`
	OffHour            int     `json:"off_hour,omitempty"`
	UnexpectedIP       string  `json:"unexpected_ip,omitempty"`
}

// CurrentActivity represents the current activity being evaluated.
type CurrentActivity struct {
	NHIID        string  `json:"nhi_id"`
	Endpoint     string  `json:"endpoint"`
	CallsPerHour float64 `json:"calls_per_hour"`
	IP           string  `json:"ip"`
	Hour         int     `json:"hour"`
}

// EvaluateRisk computes a risk score (0-100) for an NHI.
func (e *NHIRiskEngine) EvaluateRisk(nhiID uuid.UUID, currentActivity CurrentActivity) *NHIRiskScore {
	var baselines []*NHIBehaviorBaseline
	if e.pgRepo != nil {
		baselines, _ = e.pgRepo.GetBaselines(context.Background(), currentActivity.NHIID)
	}
	// Fall back to in-memory if PG returned nothing (nil pool, error, etc.).
	if len(baselines) == 0 {
		e.mu.RLock()
		baselines = e.baselines[currentActivity.NHIID]
		e.mu.RUnlock()
	}

	signals := RiskSignals{}
	score := 0

	// No baseline → moderate risk.
	if len(baselines) == 0 {
		result := &NHIRiskScore{
			NHIID:       nhiID,
			Score:       20,
			Level:       riskLevel(20),
			Signals:     map[string]any{"no_baseline": true, "message": "new NHI with no behavior baseline"},
			EvaluatedAt: time.Now(),
		}
		e.persistScore(result)
		return result
	}

	// 1. Frequency spike.
	for _, b := range baselines {
		if b.Endpoint == currentActivity.Endpoint && b.AvgCallsPerHour > 0 {
			ratio := currentActivity.CallsPerHour / b.AvgCallsPerHour
			if ratio > 5 {
				signals.FrequencySpike = true
				signals.SpikeRatio = ratio
				score += 30
			}
			break
		}
	}

	// 2. New endpoint.
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

	// 3. Off-hours.
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

	// 4. New IP.
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

	e.persistScore(result)
	return result
}

func (e *NHIRiskEngine) persistScore(result *NHIRiskScore) {
	// Always save to in-memory.
	// Also persist to PG if configured.
	if e.pgRepo != nil {
		_ = e.pgRepo.SaveRiskScore(context.Background(), result)
	}
}

// GetRiskScore returns the latest risk score from PG.
func (e *NHIRiskEngine) GetRiskScore(nhiID uuid.UUID) *NHIRiskScore {
	if e.pgRepo != nil {
		if score, err := e.pgRepo.GetRiskScore(context.Background(), nhiID); err == nil && score != nil {
			return score
		}
	}
	// No PG or not found.
	return nil
}

// ListHighRisk returns all NHIs with score >= threshold from PG or in-memory.
func (e *NHIRiskEngine) ListHighRisk(threshold int) []*NHIRiskScore {
	if e.pgRepo != nil {
		if high, err := e.pgRepo.ListHighRisk(context.Background(), threshold); err == nil && high != nil {
			return high
		}
	}
	// PG not configured or returned nothing.
	return []*NHIRiskScore{}
}

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
