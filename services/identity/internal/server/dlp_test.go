package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestEvaluateDLP_BlockCoreByNonAdmin(t *testing.T) {
	policies := []*DLPPolicy{
		{Name: "Block core export", Trigger: "export", Action: "block", Enabled: true,
			Conditions: map[string]any{"and": []any{
				map[string]any{"$data.classification": "core"},
				map[string]any{"$user.role": map[string]any{"$ne": "admin"}},
			}}},
	}
	result := EvaluateDLP(policies, "export", "audit_events", "core", "viewer")
	if !result.Matched || result.Action != "block" {
		t.Errorf("expected matched+block, got matched=%v action=%s", result.Matched, result.Action)
	}
}

func TestEvaluateDLP_AllowAdminExportCore(t *testing.T) {
	policies := []*DLPPolicy{
		{Name: "Block core export", Trigger: "export", Action: "block", Enabled: true,
			Conditions: map[string]any{"and": []any{
				map[string]any{"$data.classification": "core"},
				map[string]any{"$user.role": map[string]any{"$ne": "admin"}},
			}}},
	}
	result := EvaluateDLP(policies, "export", "audit_events", "core", "admin")
	if result.Matched && result.Action == "block" {
		t.Error("admin should not be blocked from core export")
	}
}

func TestEvaluateDLP_DefaultBlockCore(t *testing.T) {
	result := EvaluateDLP([]*DLPPolicy{}, "export", "audit_events", "core", "viewer")
	if !result.Matched || result.Action != "block" {
		t.Error("core data should default to block for non-admin")
	}
}

func TestEvaluateDLP_DefaultMaskImportant(t *testing.T) {
	result := EvaluateDLP([]*DLPPolicy{}, "download", "user_attribute", "important", "viewer")
	if !result.Matched || result.Action != "mask" {
		t.Error("important data should default to mask")
	}
}

func TestEvaluateDLP_DefaultLogGeneral(t *testing.T) {
	result := EvaluateDLP([]*DLPPolicy{}, "api_call", "config", "general", "viewer")
	if result.Matched {
		t.Error("general data should not match (log only)")
	}
}

func TestDLPRepo_NilPool(t *testing.T) {
	repo := newDLPRepo(nil)
	policies, err := repo.List(nil, uuid.Nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(policies) != 0 { t.Error("nil pool should return empty") }
}
