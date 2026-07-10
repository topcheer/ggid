package middleware

import (
	"testing"
	"time"
)

func TestHealthScore_InitialScore(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	if s := hs.Score("backend1"); s != 100.0 {
		t.Errorf("expected 100 for unknown backend, got %f", s)
	}
}

func TestHealthScore_AllSuccess(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("b1", 50*time.Millisecond)
	}
	s := hs.Score("b1")
	if s < 90 {
		t.Errorf("expected high score for all-success, got %f", s)
	}
}

func TestHealthScore_AllErrors(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 10; i++ {
		hs.RecordError("b1")
	}
	s := hs.Score("b1")
	// 0% success rate → base score 0, but decay applies multiplicatively
	if s > 5 {
		t.Errorf("expected very low score for all-errors, got %f", s)
	}
}

func TestHealthScore_MixedResults(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	// 8 success, 2 errors = 80% success rate
	for i := 0; i < 8; i++ {
		hs.RecordSuccess("b1", 50*time.Millisecond)
	}
	hs.RecordError("b1")
	hs.RecordError("b1")

	s := hs.Score("b1")
	// 80% success → base ~56, latency bonus ~30, decay 0.95 → ~82
	if s < 30 || s > 85 {
		t.Errorf("expected 30-80 for 80%% success, got %f", s)
	}
}

func TestHealthScore_Weight(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 5; i++ {
		hs.RecordError("b1")
	}
	w := hs.Weight("b1")
	// Should have low weight but not zero (minimum 0.1)
	if w < 0.1 || w > 0.5 {
		t.Errorf("expected 0.1-0.5 weight for failing backend, got %f", w)
	}
}

func TestHealthScore_HealthyBackend(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("b1", 30*time.Millisecond)
	}
	if !hs.IsHealthy("b1", 50) {
		t.Error("expected b1 to be healthy")
	}
}

func TestHealthScore_UnhealthyBackend(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 10; i++ {
		hs.RecordError("b1")
	}
	if hs.IsHealthy("b1", 50) {
		t.Error("expected b1 to be unhealthy")
	}
}

func TestHealthScore_AllScores(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	hs.RecordSuccess("b1", 10*time.Millisecond)
	hs.RecordSuccess("b2", 10*time.Millisecond)
	hs.RecordError("b3")

	scores := hs.AllScores()
	if len(scores) != 3 {
		t.Errorf("expected 3 backends, got %d", len(scores))
	}
	if scores["b3"] >= scores["b1"] {
		t.Error("expected b3 score < b1 score")
	}
}

func TestHealthScore_AllWeights(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	hs.RecordSuccess("b1", 10*time.Millisecond)
	for i := 0; i < 5; i++ {
		hs.RecordError("b2")
	}

	weights := hs.AllWeights()
	if weights["b1"] <= weights["b2"] {
		t.Error("expected b1 weight > b2 weight")
	}
}

func TestHealthScore_Reset(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	for i := 0; i < 5; i++ {
		hs.RecordError("b1")
	}
	hs.Reset("b1")
	// After reset, backend is unknown → score 100
	if s := hs.Score("b1"); s != 100 {
		t.Errorf("expected 100 after reset, got %f", s)
	}
}

func TestHealthScore_Prune(t *testing.T) {
	hs := NewHealthScore(1*time.Millisecond, 0.95)
	hs.RecordSuccess("b1", 10*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	hs.Prune()
	scores := hs.AllScores()
	if len(scores) != 0 {
		t.Errorf("expected 0 after prune, got %d", len(scores))
	}
}

func TestHealthScore_LatencyImpact(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	// Fast backend
	hs.RecordSuccess("fast", 10*time.Millisecond)
	// Slow backend (still success but >2s)
	hs.RecordSuccess("slow", 3*time.Second)

	fastScore := hs.Score("fast")
	slowScore := hs.Score("slow")
	if slowScore >= fastScore {
		t.Errorf("expected slow < fast: slow=%f fast=%f", slowScore, fastScore)
	}
}

func TestNewHealthScore_Defaults(t *testing.T) {
	hs := NewHealthScore(0, 0)
	if hs.successWindow != 5*time.Minute {
		t.Errorf("expected 5m default, got %v", hs.successWindow)
	}
	if hs.decayFactor != 0.95 {
		t.Errorf("expected 0.95 default, got %f", hs.decayFactor)
	}
}
