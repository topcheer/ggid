package soar

import (
	"context"
	"testing"
	"time"
)

func TestEvaluateTrigger_RuleMatch(t *testing.T) {
	engine := NewEngine(nil)
	trigger := PlaybookTrigger{Rule: "mfa_fatigue", Severity: "high"}
	event := TriggerEvent{RuleID: "mfa_fatigue", Severity: "high", UserID: "u1"}

	if !engine.EvaluateTrigger(trigger, event) {
		t.Fatal("trigger should match mfa_fatigue + high")
	}
}

func TestEvaluateTrigger_RuleMismatch(t *testing.T) {
	engine := NewEngine(nil)
	trigger := PlaybookTrigger{Rule: "mfa_fatigue"}
	event := TriggerEvent{RuleID: "brute_force"}

	if engine.EvaluateTrigger(trigger, event) {
		t.Fatal("trigger should not match different rule")
	}
}

func TestEvaluateTrigger_SeverityBelowThreshold(t *testing.T) {
	engine := NewEngine(nil)
	trigger := PlaybookTrigger{Severity: "critical"}
	event := TriggerEvent{Severity: "medium"}

	if engine.EvaluateTrigger(trigger, event) {
		t.Fatal("medium should not trigger critical threshold")
	}
}

func TestExecute_RateLimited(t *testing.T) {
	engine := NewEngine(nil)
	pb := &Playbook{ID: "pb1", Actions: []PlaybookAction{{Type: "lock_account"}}}
	event := TriggerEvent{UserID: "user-1", RuleID: "mfa_fatigue"}

	// First execution should work.
	exec1, err := engine.Execute(context.Background(), pb, event)
	if err != nil {
		t.Fatal(err)
	}
	if exec1.Status != "completed" {
		t.Fatalf("expected completed, got %s", exec1.Status)
	}

	// Second within 5 min should be rate limited.
	exec2, _ := engine.Execute(context.Background(), pb, event)
	if exec2.Status != "rate_limited" {
		t.Fatalf("expected rate_limited, got %s", exec2.Status)
	}
}

func TestExecute_DifferentUsersNotRateLimited(t *testing.T) {
	engine := NewEngine(nil)
	pb := &Playbook{ID: "pb1", Actions: []PlaybookAction{{Type: "lock_account"}}}

	exec1, _ := engine.Execute(context.Background(), pb, TriggerEvent{UserID: "user-a"})
	exec2, _ := engine.Execute(context.Background(), pb, TriggerEvent{UserID: "user-b"})

	if exec1.Status != "completed" || exec2.Status != "completed" {
		t.Fatal("different users should not be rate limited")
	}
}

func TestExecute_ActionResults(t *testing.T) {
	engine := NewEngine(nil)
	pb := &Playbook{
		ID: "pb1",
		Actions: []PlaybookAction{
			{Type: "lock_account"},
			{Type: "step_up_mfa"},
			{Type: "unknown_action"},
		},
	}

	exec, err := engine.Execute(context.Background(), pb, TriggerEvent{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(exec.ActionsTaken) != 3 {
		t.Fatalf("expected 3 action results, got %d", len(exec.ActionsTaken))
	}
	if exec.ActionsTaken[0] != "lock_account:OK" {
		t.Fatalf("expected lock_account:OK, got %s", exec.ActionsTaken[0])
	}
	if exec.ActionsTaken[2] != "unknown_action:FAILED" {
		t.Fatalf("expected FAILED for unknown, got %s", exec.ActionsTaken[2])
	}
}

func TestSeverityAtLeast(t *testing.T) {
	if !severityAtLeast("critical", "high") {
		t.Error("critical >= high")
	}
	if !severityAtLeast("high", "high") {
		t.Error("high >= high")
	}
	if severityAtLeast("low", "high") {
		t.Error("low < high")
	}
}

func TestExecute_CompletionTime(t *testing.T) {
	engine := NewEngine(nil)
	pb := &Playbook{ID: "pb1", Actions: []PlaybookAction{{Type: "create_incident"}}}

	exec, _ := engine.Execute(context.Background(), pb, TriggerEvent{UserID: "u1"})
	if exec.CompletedAt == nil {
		t.Fatal("completed_at should be set")
	}
	if exec.CompletedAt.Before(exec.StartedAt) {
		t.Fatal("completed_at should be after started_at")
	}
	if time.Since(*exec.CompletedAt) > 5*time.Second {
		t.Fatal("execution should complete quickly")
	}
}
