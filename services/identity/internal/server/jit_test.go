package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestRunJITPipeline_CreateUser(t *testing.T) {
	mapping := &JITMapping{
		Protocol:    "saml",
		AttributeMap: map[string]any{"email": "mail", "username": "uid"},
		GroupMap:    map[string]any{"admin": "admins"},
		DefaultRoleID: "viewer",
	}
	attrs := map[string]any{
		"mail":   "alice@example.com",
		"uid":    "alice",
		"groups": []any{"admins"},
	}
	result := RunJITPipeline(mapping, attrs, false)
	if result.Action != "created" {
		t.Errorf("expected created, got %s", result.Action)
	}
	if result.Username != "alice" {
		t.Errorf("expected alice, got %s", result.Username)
	}
	found := false
	for _, r := range result.AssignedRoles {
		if r == "admin" {
			found = true
		}
	}
	if !found {
		t.Error("should map admins group to admin role")
	}
}

func TestRunJITPipeline_DefaultRole(t *testing.T) {
	mapping := &JITMapping{
		AttributeMap:  map[string]any{"email": "mail"},
		DefaultRoleID: "viewer",
	}
	attrs := map[string]any{"mail": "bob@example.com"}
	result := RunJITPipeline(mapping, attrs, false)
	if len(result.AssignedRoles) != 1 || result.AssignedRoles[0] != "viewer" {
		t.Error("should assign default role when no group match")
	}
}

func TestRunJITPipeline_DryRun(t *testing.T) {
	mapping := &JITMapping{
		AttributeMap: map[string]any{"email": "mail"},
	}
	attrs := map[string]any{"mail": "carol@example.com"}
	result := RunJITPipeline(mapping, attrs, true)
	if result.Action != "no_change" {
		t.Errorf("dry-run should return no_change, got %s", result.Action)
	}
	if !result.DryRun {
		t.Error("dry_run flag should be true")
	}
}

func TestRunJITPipeline_NoEmail(t *testing.T) {
	mapping := &JITMapping{
		AttributeMap: map[string]any{"email": "mail"},
	}
	attrs := map[string]any{"name": "dave"}
	result := RunJITPipeline(mapping, attrs, false)
	if result.Action != "error" {
		t.Errorf("missing email should error, got %s", result.Action)
	}
}

func TestRunJITPipeline_MultipleGroupMapping(t *testing.T) {
	mapping := &JITMapping{
		AttributeMap: map[string]any{"email": "mail"},
		GroupMap: map[string]any{
			"developer": "eng",
			"viewer":    "staff",
		},
	}
	attrs := map[string]any{
		"mail":   "eve@example.com",
		"groups": []any{"eng", "staff"},
	}
	result := RunJITPipeline(mapping, attrs, false)
	if len(result.AssignedRoles) != 2 {
		t.Errorf("expected 2 roles, got %d: %v", len(result.AssignedRoles), result.AssignedRoles)
	}
}

func TestJITRepo_NilPool(t *testing.T) {
	repo := newJITRepo(nil)
	mappings, err := repo.List(nil, uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(mappings) != 0 {
		t.Error("nil pool should return empty")
	}
}
