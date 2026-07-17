package httpserver

import (
	"testing"
	"time"
)

func TestEvaluateComposite_ThresholdMet(t *testing.T) {
	rule := &CompositeRule{
		Signals:    []string{"brute_force", "impossible_travel", "malware_c2"},
		MinSignals: 2,
		WindowMin:  30,
	}
	triggered := map[string]time.Time{
		"brute_force":      time.Now().Add(-5 * time.Minute),
		"impossible_travel": time.Now().Add(-10 * time.Minute),
	}
	if !EvaluateComposite(rule, triggered) {
		t.Error("2 of 3 signals in window should trigger composite rule")
	}
}

func TestEvaluateComposite_ThresholdNotMet(t *testing.T) {
	rule := &CompositeRule{
		Signals:    []string{"brute_force", "impossible_travel", "malware_c2"},
		MinSignals: 3,
		WindowMin:  30,
	}
	triggered := map[string]time.Time{
		"brute_force": time.Now().Add(-5 * time.Minute),
	}
	if EvaluateComposite(rule, triggered) {
		t.Error("1 of 3 signals should not trigger rule with min_signals=3")
	}
}

func TestEvaluateComposite_OutsideWindow(t *testing.T) {
	rule := &CompositeRule{
		Signals:    []string{"brute_force", "impossible_travel"},
		MinSignals: 2,
		WindowMin:  30,
	}
	triggered := map[string]time.Time{
		"brute_force":      time.Now().Add(-5 * time.Minute),
		"impossible_travel": time.Now().Add(-2 * time.Hour), // outside 30min window
	}
	if EvaluateComposite(rule, triggered) {
		t.Error("signal outside time window should not count")
	}
}

func TestCompositeRuleRepo_NilPool(t *testing.T) {
	repo := newCompositeRuleRepo(nil)
	rules, err := repo.List(nil)
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(rules) != 0 {
		t.Error("nil pool should return empty list")
	}
}
