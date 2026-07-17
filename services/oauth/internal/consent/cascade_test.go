package consent

import (
	"context"
	"testing"
)

func TestWithdrawCascade_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	result, err := engine.WithdrawCascade(context.Background(), "user-1", "t1", "profile.read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TriggerType != "consent_withdrawal" {
		t.Fatalf("expected trigger consent_withdrawal, got %s", result.TriggerType)
	}
	if result.AffectedTokens == 0 {
		t.Fatal("expected some affected tokens")
	}
	if result.AffectedSessions == 0 {
		t.Fatal("expected some affected sessions")
	}
	// Should have audit_log action.
	found := false
	for _, a := range result.Actions {
		if a.Type == "audit_log" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected audit_log action in cascade")
	}
}

func TestGDPRErase_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	result, err := engine.GDPRErase(context.Background(), "user-1", "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TriggerType != "gdpr_erase" {
		t.Fatalf("expected trigger gdpr_erase, got %s", result.TriggerType)
	}
	// Should have delete_pii actions for 3 tables.
	piiCount := 0
	for _, a := range result.Actions {
		if a.Type == "delete_pii" {
			piiCount++
		}
	}
	if piiCount != 3 {
		t.Fatalf("expected 3 delete_pii actions, got %d", piiCount)
	}
	// Should have revoke_all_tokens.
	foundTokens := false
	for _, a := range result.Actions {
		if a.Type == "revoke_all_tokens" {
			foundTokens = true
		}
	}
	if !foundTokens {
		t.Fatal("expected revoke_all_tokens action")
	}
}

func TestGetCascadeLog_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	results, err := engine.GetCascadeLog(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatal("nil pool should return nil results")
	}
}

func TestEnsureSchema_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	err := engine.EnsureSchema(context.Background())
	if err != nil {
		t.Fatalf("nil pool EnsureSchema should not error: %v", err)
	}
}

func TestWithdrawCascade_ScopePropagation(t *testing.T) {
	engine := NewEngine(nil)
	result, _ := engine.WithdrawCascade(context.Background(), "user-1", "t1", "admin.write")
	if result.Scope != "admin.write" {
		t.Fatalf("expected scope admin.write, got %s", result.Scope)
	}
}
