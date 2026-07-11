package service

import (
	"testing"
)

func TestRiskEngine_LowRisk(t *testing.T) {
	e := NewRiskEngine()
	e.RecordEvent("user1", "device-known", "1.2.3.4")
	e.SetKnownLocation("user1", "US")

	score := e.Evaluate("user1", "device-known", "1.2.3.4", "US")
	if score.Score > 0.3 {
		t.Errorf("known user/device/IP/geo should be low risk, got %f", score.Score)
	}
	if len(score.Recommendations) > 0 {
		t.Error("low risk should have no recommendations")
	}
}

func TestRiskEngine_UnknownDevice(t *testing.T) {
	e := NewRiskEngine()
	e.RecordEvent("user1", "device-A", "1.2.3.4")

	score := e.Evaluate("user1", "device-UNKNOWN", "1.2.3.4", "US")
	if !score.DeviceKnown && score.Score == 0 {
		t.Error("unknown device should increase risk")
	}
}

func TestRiskEngine_NewIP(t *testing.T) {
	e := NewRiskEngine()
	score := e.Evaluate("user1", "dev", "5.6.7.8", "US")
	if !score.NewIP {
		t.Error("should detect new IP")
	}
}

func TestRiskEngine_GeoAnomaly(t *testing.T) {
	e := NewRiskEngine()
	e.RecordEvent("user1", "dev", "1.2.3.4")
	e.SetKnownLocation("user1", "US")

	score := e.Evaluate("user1", "dev", "1.2.3.4", "CN")
	if !score.GeoAnomaly {
		t.Error("should detect geo anomaly (US → CN)")
	}
}

func TestRiskEngine_HighVelocity(t *testing.T) {
	e := NewRiskEngine()
	for i := 0; i < 25; i++ {
		e.RecordEvent("user1", "dev", "1.2.3.4")
	}
	score := e.Evaluate("user1", "dev", "1.2.3.4", "US")
	if score.Velocity < 20 {
		t.Errorf("should show high velocity, got %d", score.Velocity)
	}
}

func TestRiskEngine_BlockRecommendation(t *testing.T) {
	e := NewRiskEngine()
	// Trigger multiple risk factors
	for i := 0; i < 30; i++ {
		e.RecordEvent("user1", "dev", "1.2.3.4")
	}
	e.SetKnownLocation("user1", "US")
	score := e.Evaluate("user1", "unknown-dev", "new-ip", "RU")

	if score.Score < 0.8 {
		t.Errorf("should be high risk with multiple factors, got %f", score.Score)
	}
	found := false
	for _, r := range score.Recommendations {
		if r == "block" {
			found = true
		}
	}
	if !found {
		t.Error("should recommend block for very high risk")
	}
}

func TestRiskEngine_Reset(t *testing.T) {
	e := NewRiskEngine()
	e.RecordEvent("user1", "dev", "1.2.3.4")
	e.Reset()
	score := e.Evaluate("user1", "dev", "1.2.3.4", "")
	if score.DeviceKnown {
		t.Error("after reset, device should not be known")
	}
}
