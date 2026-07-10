package middleware

import (
	"math"
	"sync"
	"time"
)

// HealthScore tracks a rolling health score for each backend.
// The score is based on success rate, average latency, and error rate.
// Higher score = healthier backend. Range: 0.0 to 100.0.
type HealthScore struct {
	mu              sync.RWMutex
	backends        map[string]*backendHealth
	successWindow   time.Duration
	decayFactor     float64
}

type backendHealth struct {
	// Rolling counters (reset every window)
	totalReqs    int64
	successReqs  int64
	errorReqs    int64
	totalLatency time.Duration
	lastUpdate   time.Time
	// Computed score (0-100)
	score float64
	// Weight multiplier for load balancing (derived from score)
	weight float64
}

// NewHealthScore creates a health scorer.
// successWindow: how far back to count requests (default 5m)
// decayFactor: how much old errors decay (0-1, default 0.95)
func NewHealthScore(successWindow time.Duration, decayFactor float64) *HealthScore {
	if successWindow <= 0 {
		successWindow = 5 * time.Minute
	}
	if decayFactor <= 0 || decayFactor > 1 {
		decayFactor = 0.95
	}
	return &HealthScore{
		backends:      make(map[string]*backendHealth),
		successWindow: successWindow,
		decayFactor:   decayFactor,
	}
}

// RecordSuccess records a successful request to the given backend.
func (hs *HealthScore) RecordSuccess(backend string, latency time.Duration) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	bh := hs.getOrCreate(backend)
	bh.totalReqs++
	bh.successReqs++
	bh.totalLatency += latency
	bh.lastUpdate = time.Now()
	hs.recomputeScore(bh)
}

// RecordError records a failed request to the given backend.
func (hs *HealthScore) RecordError(backend string) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	bh := hs.getOrCreate(backend)
	bh.totalReqs++
	bh.errorReqs++
	bh.lastUpdate = time.Now()
	hs.recomputeScore(bh)
}

// Score returns the current health score for a backend (0-100).
// Returns 100 for backends with no data (optimistic default).
func (hs *HealthScore) Score(backend string) float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	bh, ok := hs.backends[backend]
	if !ok || bh.totalReqs == 0 {
		return 100.0
	}
	return bh.score
}

// Weight returns the load-balancing weight for a backend.
// Healthier backends get higher weights.
func (hs *HealthScore) Weight(backend string) float64 {
	score := hs.Score(backend)
	// Weight = score/100, but with a minimum floor of 0.1 for recovery
	return math.Max(score/100.0, 0.1)
}

// AllScores returns scores for all tracked backends.
func (hs *HealthScore) AllScores() map[string]float64 {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	result := make(map[string]float64, len(hs.backends))
	for name, bh := range hs.backends {
		if bh.totalReqs > 0 {
			result[name] = bh.score
		} else {
			result[name] = 100.0
		}
	}
	return result
}

// AllWeights returns weights for all tracked backends.
func (hs *HealthScore) AllWeights() map[string]float64 {
	result := hs.AllScores()
	for k, v := range result {
		result[k] = math.Max(v/100.0, 0.1)
	}
	return result
}

// IsHealthy returns true if the backend's score is above the threshold.
func (hs *HealthScore) IsHealthy(backend string, threshold float64) bool {
	if threshold <= 0 {
		threshold = 50.0
	}
	return hs.Score(backend) >= threshold
}

// Reset clears all health data for a backend (e.g. after recovery).
func (hs *HealthScore) Reset(backend string) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	delete(hs.backends, backend)
}

// Prune removes backends that haven't been updated within the window.
func (hs *HealthScore) Prune() {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	cutoff := time.Now().Add(-hs.successWindow)
	for name, bh := range hs.backends {
		if bh.lastUpdate.Before(cutoff) {
			delete(hs.backends, name)
		}
	}
}

// getOrCreate must be called with write lock held.
func (hs *HealthScore) getOrCreate(backend string) *backendHealth {
	bh, ok := hs.backends[backend]
	if !ok {
		bh = &backendHealth{
			score:  100.0,
			weight: 1.0,
		}
		hs.backends[backend] = bh
	}
	return bh
}

// recomputeScore calculates the health score based on current counters.
// Must be called with write lock held.
func (hs *HealthScore) recomputeScore(bh *backendHealth) {
	if bh.totalReqs == 0 {
		bh.score = 100.0
		bh.weight = 1.0
		return
	}

	successRate := float64(bh.successReqs) / float64(bh.totalReqs)

	// Base score from success rate (0-70 points)
	score := successRate * 70.0

	// Latency bonus (0-30 points)
	// <100ms = full 30 points, >2s = 0 points
	// Only award latency bonus if there are successful requests
	latencyScore := 0.0
	if bh.successReqs > 0 {
		avgLatency := bh.totalLatency / time.Duration(bh.successReqs)
		latencyScore = 30.0
		if avgLatency >= 2*time.Second {
			latencyScore = 0
		} else if avgLatency > 100*time.Millisecond {
			latencyScore = 30.0 * (1.0 - float64(avgLatency-100*time.Millisecond)/float64(1900*time.Millisecond))
		}
	}
	score += latencyScore

	// Apply decay for older errors
	score *= hs.decayFactor

	// Clamp to [0, 100]
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	bh.score = score
	bh.weight = math.Max(score/100.0, 0.1)
}
