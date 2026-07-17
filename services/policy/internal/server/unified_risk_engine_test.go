package httpserver

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestRiskRepo_NilPool(t *testing.T) {
	repo := NewRiskRepo(nil)
	policy, err := repo.GetPolicy(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if policy.AllowThreshold != 30 {
		t.Error("default policy should have allow=30")
	}
}

func TestSignalRegistry_Count(t *testing.T) {
	if len(signalRegistry) != 26 {
		t.Errorf("expected 26 signals, got %d", len(signalRegistry))
	}
	// Check categories.
	cats := map[SignalCategory]int{}
	for _, s := range signalRegistry {
		cats[s.Category]++
	}
	if cats[SigDevice] != 6 { t.Errorf("expected 6 device signals, got %d", cats[SigDevice]) }
	if cats[SigGeo] != 5 { t.Errorf("expected 5 geo signals, got %d", cats[SigGeo]) }
	if cats[SigNetwork] != 5 { t.Errorf("expected 5 network signals, got %d", cats[SigNetwork]) }
	if cats[SigBehavior] != 6 { t.Errorf("expected 6 behavior signals, got %d", cats[SigBehavior]) }
	if cats[SigSession] != 4 { t.Errorf("expected 4 session signals, got %d", cats[SigSession]) }
}

func TestEvaluateRisk_AllowByDefault(t *testing.T) {
	s := &HTTPServer{riskRepo: NewRiskRepo(nil)}
	resp := s.EvaluateRisk(context.Background(), &RiskEvaluationRequest{
		UserID: "test-user",
		Context: map[string]any{}, // no risk signals
	})
	if resp.Score != 0 {
		t.Errorf("expected score 0 with no signals, got %d", resp.Score)
	}
	if resp.Decision != "allow" {
		t.Errorf("expected allow, got %s", resp.Decision)
	}
	if resp.Level != "low" {
		t.Errorf("expected low, got %s", resp.Level)
	}
}

func TestEvaluateRisk_HighRisk(t *testing.T) {
	s := &HTTPServer{riskRepo: NewRiskRepo(nil)}
	resp := s.EvaluateRisk(context.Background(), &RiskEvaluationRequest{
		UserID: "test-user",
		Context: map[string]any{
			"device_jailbreak":     1.0,
			"geo_impossible_travel": 1.0,
			"net_threat_intel":      1.0,
			"beh_privilege_escalation": 1.0,
		},
	})
	if resp.Score <= 0 {
		t.Error("expected non-zero score with risk signals")
	}
	if resp.Decision == "allow" {
		t.Error("expected non-allow decision with high risk")
	}
}

func TestDefaultPolicy(t *testing.T) {
	p := defaultPolicy(uuid.New())
	if p.AllowThreshold != 30 || p.StepUpThreshold != 60 || p.StrongThreshold != 85 {
		t.Error("default thresholds mismatch")
	}
}
