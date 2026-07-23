package ueba

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsolationForest_NormalBehavior_LowScore(t *testing.T) {
	engine := NewEngineWithSeed(nil, 42) // Fixed seed for deterministic test
	tenantID := uuid.New()
	userID := uuid.New()

	// Generate 50 samples: logins at 9-11am, same IP/device.
	samples := []BehavioralSample{}
	for i := 0; i < 50; i++ {
		samples = append(samples, BehavioralSample{
			Hour:        9 + float64(i%3),
			DayOfWeek:   float64(i % 5),
			IPHash:      0.5,
			DeviceHash:  0.3,
			IsNewIP:     0,
			IsNewDevice: 0,
		})
	}

	engine.Train(tenantID, userID, samples)

	// Normal event: 10am, same IP/device.
	score := engine.Score(tenantID, userID, BehavioralSample{
		Hour: 10, IPHash: 0.5, DeviceHash: 0.3,
	})

	if score > 0.7 {
		t.Fatalf("normal behavior should have low anomaly score, got %.2f", score)
	}
}

func TestIsolationForest_AnomalousBehavior_HighScore(t *testing.T) {
	engine := NewEngineWithSeed(nil, 42) // Fixed seed for deterministic test
	tenantID := uuid.New()
	userID := uuid.New()

	// Train with normal hours (9-17, weekdays).
	samples := []BehavioralSample{}
	for i := 0; i < 50; i++ {
		samples = append(samples, BehavioralSample{
			Hour:      9 + float64(i%8),
			DayOfWeek: float64(i % 5),
			IPHash:    0.5,
			DeviceHash: 0.3,
		})
	}
	engine.Train(tenantID, userID, samples)

	// Anomalous event: 3am, new IP + new device.
	score := engine.Score(tenantID, userID, BehavioralSample{
		Hour:        3,
		DayOfWeek:   6, // weekend
		IPHash:      0.99,
		DeviceHash:  0.88,
		IsNewIP:     1,
		IsNewDevice: 1,
	})

	if score < 0.5 {
		t.Fatalf("anomalous behavior should have high anomaly score, got %.2f", score)
	}
}

func TestIsolationForest_InsufficientData_Fallback(t *testing.T) {
	engine := NewEngine(nil)
	tenantID := uuid.New()
	userID := uuid.New()

	// Only 5 samples — below threshold of 30.
	samples := []BehavioralSample{}
	for i := 0; i < 5; i++ {
		samples = append(samples, BehavioralSample{
			Hour: float64(9 + i),
			IPHash: 0.5, DeviceHash: 0.3,
		})
	}

	baseline := engine.Train(tenantID, userID, samples)
	if baseline.Forest != nil {
		t.Fatal("forest should be nil with <30 samples")
	}
	if baseline.SampleCount != 5 {
		t.Fatalf("expected 5 samples, got %d", baseline.SampleCount)
	}

	// Score should use 3-sigma fallback.
	score := engine.Score(tenantID, userID, BehavioralSample{Hour: 9})
	if score < 0 {
		t.Fatal("fallback score should be non-negative")
	}
}

func TestIsolationForest_ComputeStats(t *testing.T) {
	samples := []BehavioralSample{}
	for _, h := range []int{9, 9, 10, 10, 11, 11, 14, 15} {
		samples = append(samples, BehavioralSample{
			Hour: float64(h), IPHash: 0.5, DeviceHash: 0.3,
		})
	}

	stats := computeStats(samples)
	if stats.MeanHour < 9 || stats.MeanHour > 13 {
		t.Fatalf("unexpected mean hour: %.2f", stats.MeanHour)
	}
	if len(stats.CommonHours) == 0 {
		t.Fatal("expected common hours")
	}
	if stats.UniqueIPs != 1 {
		t.Fatalf("expected 1 unique IP, got %d", stats.UniqueIPs)
	}
}

func TestIsolationForest_AvgPathLengthFunc(t *testing.T) {
	// c(1) = 0
	if v := avgPathLengthFunc(1); v != 0 {
		t.Fatalf("c(1) should be 0, got %f", v)
	}
	// c(2) = 1
	if v := avgPathLengthFunc(2); v != 1 {
		t.Fatalf("c(2) should be 1, got %f", v)
	}
	// c(256) should be reasonable (~12)
	v := avgPathLengthFunc(256)
	if v < 10 || v > 15 {
		t.Fatalf("c(256) expected ~12, got %f", v)
	}
}

func TestIsolationForest_PersistBaseline(t *testing.T) {
	engine := NewEngine(nil) // nil pool
	baseline := &BehavioralBaseline{
		UserID: uuid.New(), TenantID: uuid.New(),
		SampleCount: 50,
	}
	// Should not error with nil pool.
	err := engine.PersistBaseline(nil, baseline)
	if err != nil {
		t.Fatalf("PersistBaseline with nil pool should not error: %v", err)
	}
}

func TestGenerateSamplesFromHours(t *testing.T) {
	hours := []int{9, 10, 11, 14}
	ipHashes := []float64{0.5, 0.6, 0.5, 0.7}
	samples := GenerateSamplesFromHours(hours, ipHashes, nil)
	if len(samples) != 4 {
		t.Fatalf("expected 4 samples, got %d", len(samples))
	}
	if samples[0].Hour != 9 {
		t.Fatalf("expected first hour 9, got %f", samples[0].Hour)
	}
	if samples[1].IPHash != 0.6 {
		t.Fatalf("expected second IP hash 0.6, got %f", samples[1].IPHash)
	}
}
