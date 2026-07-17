package service

import "testing"

// fakeThreatChecker implements ThreatIntelChecker for testing.
type fakeThreatChecker struct {
	severity string
	confidence int
	hit bool
}

func (f *fakeThreatChecker) CheckThreat(tenantID, indicatorType, value string) (string, int, bool) {
	return f.severity, f.confidence, f.hit
}

func TestRiskEngine_ThreatIntelHitCritical(t *testing.T) {
	engine := NewRiskEngine()
	engine.SetThreatIntelChecker(&fakeThreatChecker{
		severity:   "critical",
		confidence: 95,
		hit:        true,
	})

	score := engine.Evaluate("user-1", "fp-1", "203.0.113.50", "US")
	if !score.ThreatIntelHit {
		t.Fatal("expected ThreatIntelHit=true")
	}
	if score.ThreatSeverity != "critical" {
		t.Fatalf("expected severity critical, got %s", score.ThreatSeverity)
	}
	if score.Score < 0.5 {
		t.Fatalf("expected score >= 0.5 with critical threat hit, got %.2f", score.Score)
	}
	found := false
	for _, rec := range score.Recommendations {
		if rec == "block_session" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected block_session in recommendations for critical threat hit")
	}
}

func TestRiskEngine_ThreatIntelNoHit(t *testing.T) {
	engine := NewRiskEngine()
	engine.SetThreatIntelChecker(&fakeThreatChecker{
		hit: false,
	})

	score := engine.Evaluate("user-1", "fp-1", "192.168.1.1", "US")
	if score.ThreatIntelHit {
		t.Fatal("expected ThreatIntelHit=false")
	}
	if score.ThreatSeverity != "" {
		t.Fatalf("expected empty ThreatSeverity, got %s", score.ThreatSeverity)
	}
}

func TestRiskEngine_NoChecker(t *testing.T) {
	engine := NewRiskEngine()
	// No threat checker injected — should work fine.
	score := engine.Evaluate("user-1", "fp-1", "192.168.1.1", "US")
	if score.ThreatIntelHit {
		t.Fatal("expected ThreatIntelHit=false when no checker")
	}
}
