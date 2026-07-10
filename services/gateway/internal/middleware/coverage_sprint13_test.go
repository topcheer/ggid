package middleware

import (
	"testing"
	"time"
)

// === HealthScore coverage for recomputeScore branches ===

func TestHealthScore_V2_NoRequests(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	if hs.Score("unknown") != 100 {
		t.Errorf("Score for unknown backend: want 100, got %.2f", hs.Score("unknown"))
	}
}

func TestHealthScore_V2_LatencyBonusFull(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("fast", 5*time.Millisecond)
	}
	if score := hs.Score("fast"); score < 90 {
		t.Errorf("Fast score: want >=90, got %.2f", score)
	}
}

func TestHealthScore_V2_LatencyBonusPartial(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("med", 500*time.Millisecond)
	}
	score := hs.Score("med")
	// With 500ms latency: success rate 100% → 70 pts, latency bonus ≈ 22 pts
	// total ≈ 92 * 0.9 decay ≈ 82.8. But error count is 0 so should be decent.
	if score < 50 || score > 100 {
		t.Errorf("Medium latency score: want 50-100, got %.2f", score)
	}
}

func TestHealthScore_V2_LatencyBonusZero(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("slow", 3*time.Second)
	}
	if score := hs.Score("slow"); score > 70 {
		t.Errorf("Slow score: want <=70, got %.2f", score)
	}
}

func TestHealthScore_V2_AllErrors(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	for i := 0; i < 10; i++ {
		hs.RecordError("err")
	}
	if score := hs.Score("err"); score > 10 {
		t.Errorf("All errors: want <=10, got %.2f", score)
	}
}

func TestHealthScore_V2_Prune(t *testing.T) {
	hs := NewHealthScore(1*time.Millisecond, 0.9)
	hs.RecordSuccess("old", 5*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	hs.Prune()
	if hs.Score("old") != 100 {
		t.Error("Old backend should be pruned to default 100")
	}
}

func TestHealthScore_V2_IsHealthy_Threshold(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	for i := 0; i < 10; i++ {
		hs.RecordSuccess("ok", 5*time.Millisecond)
	}
	if !hs.IsHealthy("ok", 50) {
		t.Error("Healthy backend should pass threshold 50")
	}
	for i := 0; i < 10; i++ {
		hs.RecordError("bad")
	}
	if hs.IsHealthy("bad", 50) {
		t.Error("Unhealthy backend should fail threshold 50")
	}
}

func TestHealthScore_V2_AllScores(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	hs.RecordSuccess("b1", 5*time.Millisecond)
	hs.RecordError("b2")
	scores := hs.AllScores()
	if len(scores) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(scores))
	}
}

// === AdaptiveRateLimiter coverage ===

func TestAdaptiveRateLimiter_V2_BasicAllow(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 200)
	if !al.Allow("k1") {
		t.Error("First Allow should succeed")
	}
}

func TestAdaptiveRateLimiter_V2_RecordLatencyDecrease(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 200)
	al.adjustInterval = 10 * time.Millisecond
	for i := 0; i < 5; i++ {
		al.Allow("k1")
		al.RecordLatency("k1", 600*time.Millisecond)
		time.Sleep(15 * time.Millisecond)
	}
	limit := al.Limit("k1")
	if limit >= 100 {
		t.Errorf("Limit should decrease: got %.2f", limit)
	}
}

func TestAdaptiveRateLimiter_V2_RecordLatencyIncrease(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 200)
	al.adjustInterval = 10 * time.Millisecond
	al.SetLimit("k1", 10)
	for i := 0; i < 5; i++ {
		al.Allow("k1")
		al.RecordLatency("k1", 5*time.Millisecond)
		time.Sleep(15 * time.Millisecond)
	}
	limit := al.Limit("k1")
	if limit <= 10 {
		t.Errorf("Limit should increase: got %.2f", limit)
	}
}

func TestAdaptiveRateLimiter_V2_SetLimit(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 200)
	al.SetLimit("k1", 50)
	if al.Limit("k1") != 50 {
		t.Errorf("SetLimit: want 50, got %.2f", al.Limit("k1"))
	}
}

func TestAdaptiveRateLimiter_V2_AllLimits(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 200)
	al.Allow("k1")
	al.Allow("k2")
	limits := al.AllLimits()
	if len(limits) != 2 {
		t.Errorf("Expected 2 limits, got %d", len(limits))
	}
}

func TestAdaptiveRateLimiter_V2_ExhaustTokens(t *testing.T) {
	al := NewAdaptiveRateLimiter(2, 1, 5)
	// Use all tokens
	al.Allow("k1")
	al.Allow("k1")
	// Third should be denied
	if al.Allow("k1") {
		t.Error("Third call should be rate limited")
	}
}

// === RequestDeduplicator ===

func TestNewRequestDeduplicator_V2(t *testing.T) {
	rd := NewRequestDeduplicator(5 * time.Second)
	if rd == nil {
		t.Fatal("nil")
	}
}
