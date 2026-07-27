package service

import (
	"testing"
	"time"
)

func TestRiskEngine_NoRiskFactors(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:      "user-1",
		IPAddress:   "1.2.3.4",
		GeoLocation: "US",
		DeviceKnown: true,
		HourOfDay:   14,
	}
	score := engine.EvaluateRisk(ctx)

	if score.Score != 0 {
		t.Errorf("expected score 0 for safe context, got %f", score.Score)
	}
	if score.Level != RiskLow {
		t.Errorf("expected RiskLow, got %s", score.Level)
	}
	if score.Action != ActionAllow {
		t.Errorf("expected ActionAllow, got %s", score.Action)
	}
}

func TestRiskEngine_GeoVelocity(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:          "user-1",
		GeoLocation:     "US",
		LastGeoLocation: "CN",
		DeviceKnown:     true,
		HourOfDay:       14,
	}
	score := engine.EvaluateRisk(ctx)

	if score.Score < 0.25 {
		t.Errorf("expected score >= 0.25 for geo velocity, got %f", score.Score)
	}
	found := false
	for _, f := range score.TriggeredFactors {
		if f == "geo_velocity" {
			found = true
		}
	}
	if !found {
		t.Error("expected geo_velocity in triggered factors")
	}
}

func TestRiskEngine_ImpossibleTravel(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:          "user-1",
		GeoLocation:     "US",
		LastGeoLocation: "CN",
		LastLoginAt:     time.Now().Add(10 * time.Minute), // 10 min ago
		DeviceKnown:     true,
		HourOfDay:       14,
	}
	score := engine.EvaluateRisk(ctx)

	// Should trigger both geo_velocity (0.25) + impossible_travel (0.35) = 0.60
	if score.Level != RiskHigh {
		t.Errorf("expected RiskHigh for impossible travel, got %s (score %f)", score.Level, score.Score)
	}
}

func TestRiskEngine_NewDevice(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:      "user-1",
		DeviceKnown: false,
		HourOfDay:   14,
	}
	score := engine.EvaluateRisk(ctx)

	if score.Score < 0.15 {
		t.Errorf("expected score >= 0.15 for new device, got %f", score.Score)
	}
}

func TestRiskEngine_AnomalousTime_Night(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:      "user-1",
		DeviceKnown: true,
		HourOfDay:   3, // 3 AM
	}
	score := engine.EvaluateRisk(ctx)

	if score.Score < 0.10 {
		t.Errorf("expected score >= 0.10 for anomalous time, got %f", score.Score)
	}
}

func TestRiskEngine_AnomalousTime_LateNight(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:      "user-1",
		DeviceKnown: true,
		HourOfDay:   23, // 11 PM
	}
	score := engine.EvaluateRisk(ctx)

	if score.Score < 0.10 {
		t.Errorf("expected score >= 0.10 for late night, got %f", score.Score)
	}
}

func TestRiskEngine_AnomalousTime_Daytime(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:      "user-1",
		DeviceKnown: true,
		HourOfDay:   10, // 10 AM — normal
	}
	score := engine.EvaluateRisk(ctx)

	for _, f := range score.TriggeredFactors {
		if f == "anomalous_time" {
			t.Error("did not expect anomalous_time trigger at 10 AM")
		}
	}
}

func TestRiskEngine_FailedAttempts(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:         "user-1",
		DeviceKnown:    true,
		FailedAttempts: 3,
		HourOfDay:      14,
	}
	score := engine.EvaluateRisk(ctx)

	if score.Score < 0.15 {
		t.Errorf("expected score >= 0.15 for 3 failed attempts, got %f", score.Score)
	}
}

func TestRiskEngine_FailedAttempts_BelowThreshold(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:         "user-1",
		DeviceKnown:    true,
		FailedAttempts: 2, // below threshold of 3
		HourOfDay:      14,
	}
	score := engine.EvaluateRisk(ctx)

	for _, f := range score.TriggeredFactors {
		if f == "failed_attempts" {
			t.Error("did not expect failed_attempts trigger for 2 attempts")
		}
	}
}

func TestRiskEngine_AllFactorsTriggered_Critical(t *testing.T) {
	engine := NewRiskEngine()
	ctx := RiskContext{
		UserID:          "user-1",
		GeoLocation:     "US",
		LastGeoLocation: "CN",
		LastLoginAt:     time.Now().Add(10 * time.Minute),
		DeviceKnown:     false,
		FailedAttempts:  5,
		HourOfDay:       3,
	}
	score := engine.EvaluateRisk(ctx)

	// All 5 factors: 0.25+0.35+0.15+0.10+0.15 = 1.00
	if score.Level != RiskCritical {
		t.Errorf("expected RiskCritical, got %s (score %f)", score.Level, score.Score)
	}
	if score.Action != ActionBlock {
		t.Errorf("expected ActionBlock, got %s", score.Action)
	}
	if len(score.TriggeredFactors) != 5 {
		t.Errorf("expected 5 triggered factors, got %d", len(score.TriggeredFactors))
	}
}

func TestRiskEngine_StepUpBoundary(t *testing.T) {
	engine := NewRiskEngine()
	// Score exactly 0.30 should trigger step_up
	// geo_velocity (0.25) + new_device (0.15) = 0.40 → medium step_up
	ctx := RiskContext{
		UserID:          "user-1",
		GeoLocation:     "US",
		LastGeoLocation: "UK",
		DeviceKnown:     false,
		HourOfDay:       14,
	}
	score := engine.EvaluateRisk(ctx)

	if score.Level != RiskMedium {
		t.Errorf("expected RiskMedium for score %f, got %s", score.Score, score.Level)
	}
	if score.Action != ActionStepUp {
		t.Errorf("expected ActionStepUp, got %s", score.Action)
	}
}

func TestScoreToLevel_Boundaries(t *testing.T) {
	tests := []struct {
		score     float64
		wantLevel RiskLevel
		wantAction RiskAction
	}{
		{0.0, RiskLow, ActionAllow},
		{0.29, RiskLow, ActionAllow},
		{0.30, RiskMedium, ActionStepUp},
		{0.49, RiskMedium, ActionStepUp},
		{0.50, RiskHigh, ActionBlock},
		{0.69, RiskHigh, ActionBlock},
		{0.70, RiskCritical, ActionBlock},
		{1.00, RiskCritical, ActionBlock},
	}
	for _, tt := range tests {
		level, action := scoreToLevel(tt.score)
		if level != tt.wantLevel {
			t.Errorf("scoreToLevel(%.2f) level = %s, want %s", tt.score, level, tt.wantLevel)
		}
		if action != tt.wantAction {
			t.Errorf("scoreToLevel(%.2f) action = %s, want %s", tt.score, action, tt.wantAction)
		}
	}
}
