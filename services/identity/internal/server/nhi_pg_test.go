package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestNHIPGRepo_NilPool(t *testing.T) {
	// Verify constructors work with nil pool (won't crash until methods called).
	repo := NewNHIPGRepo(nil)
	if repo == nil {
		t.Fatal("NewNHIPGRepo returned nil")
	}
	riskRepo := NewNHIRiskPGRepo(nil)
	if riskRepo == nil {
		t.Fatal("NewNHIRiskPGRepo returned nil")
	}
}

func TestNHIRiskEngine_NewEngine(t *testing.T) {
	// Verify in-memory engine still works (backward compat).
	e := NewNHIRiskEngine()
	if e == nil {
		t.Fatal("NewNHIRiskEngine returned nil")
	}
}

func TestNHIRiskEngine_RecordAndEvaluate(t *testing.T) {
	e := NewNHIRiskEngine()

	// Record baseline.
	e.RecordBaseline("nhi-test-1", "/api/v1/users", 10.0, "10.0.0.1", 14)

	// Evaluate: normal activity → low risk.
	score := e.EvaluateRisk(uuidMust(), CurrentActivity{
		NHIID:        "nhi-test-1",
		Endpoint:     "/api/v1/users",
		CallsPerHour: 12.0,
		IP:           "10.0.0.1",
		Hour:         14,
	})
	if score == nil {
		t.Fatal("expected non-nil score")
	}
	if score.Score > 30 {
		t.Errorf("normal activity should be low risk, got score=%d", score.Score)
	}
}

func TestNHIRiskEngine_FrequencySpike(t *testing.T) {
	e := NewNHIRiskEngine()
	e.RecordBaseline("nhi-spike", "/api/v1/data", 5.0, "10.0.0.1", 10)

	// 10x normal → frequency spike.
	score := e.EvaluateRisk(uuidMust(), CurrentActivity{
		NHIID:        "nhi-spike",
		Endpoint:     "/api/v1/data",
		CallsPerHour: 50.0,
		IP:           "10.0.0.1",
		Hour:         10,
	})
	if score.Score < 25 {
		t.Errorf("spike should have elevated score, got %d", score.Score)
	}
}

func TestRiskLevel(t *testing.T) {
	tests := []struct {
		score int
		level string
	}{
		{0, "low"}, {24, "low"},
		{25, "medium"}, {49, "medium"},
		{50, "high"}, {69, "high"},
		{70, "critical"}, {100, "critical"},
	}
	for _, tt := range tests {
		if got := riskLevel(tt.score); got != tt.level {
			t.Errorf("riskLevel(%d) = %s, want %s", tt.score, got, tt.level)
		}
	}
}

func uuidMust() uuid.UUID {
	return uuid.New()
}
