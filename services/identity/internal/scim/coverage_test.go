package scim

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

func TestCovSCIM_MapSCIMSortAttr(t *testing.T) {
	cases := map[string]string{
		"userName":     "username",
		"email":        "email",
		"created":      "created_at",
		"lastModified": "updated_at",
	}
	for input, expected := range cases {
		if result := mapSCIMSortAttr(input); result != expected {
			t.Errorf("mapSCIMSortAttr(%q) = %q, expected %q", input, result, expected)
		}
	}
}

func TestCovSCIM_ToSCIMUser(t *testing.T) {
	u := &domain.User{
		ID:          uuid.New(),
		Username:    "testuser",
		Email:       "test@example.com",
		Status:      domain.UserStatusActive,
		DisplayName: "Test User",
	}
	su := toSCIMUser(u)
	if su.ID != u.ID.String() {
		t.Error("expected matching ID")
	}
	if su.UserName != "testuser" {
		t.Error("expected testuser")
	}
}

func TestCovSCIM_FormatSCIMTime(t *testing.T) {
	s := formatSCIMTime(time.Now())
	if s == "" {
		t.Error("expected non-empty time")
	}
}

func TestCovSCIM_ParseExternalIdFilter(t *testing.T) {
	result := parseExternalIdFilter(`externalId eq "ext-123"`)
	if result != "ext-123" {
		t.Errorf("expected ext-123, got %s", result)
	}
}

func TestCovSCIM_ParseAttrList(t *testing.T) {
	m := parseAttrList("userName,email,displayName")
	if len(m) != 3 {
		t.Errorf("expected 3 attrs, got %d", len(m))
	}
}

func TestCovSCIM_ApplyAttributeFilter(t *testing.T) {
	u := SCIMUser{ID: "u1", UserName: "test", DisplayName: "Test", Active: true}
	filtered := applyAttributeFilter(u, "userName", "")
	if filtered.UserName != "test" {
		t.Error("expected userName")
	}
	if filtered.DisplayName != "" {
		t.Error("expected displayName filtered out")
	}
}

func TestCovSCIM_ApplyPatch_Replace(t *testing.T) {
	attrs := map[string]any{"userName": "original@test.com"}
	ops := []PatchOperation{
		{Op: "replace", Path: "userName", Value: json.RawMessage(`"replaced@test.com"`)},
	}
	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch: %v", err)
	}
	if result["userName"] != "replaced@test.com" {
		t.Errorf("expected replaced, got %v", result["userName"])
	}
}

func TestCovSCIM_ApplyPatch_Add(t *testing.T) {
	attrs := map[string]any{}
	ops := []PatchOperation{
		{Op: "add", Path: "displayName", Value: json.RawMessage(`"New Name"`)},
	}
	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch: %v", err)
	}
	if result["displayName"] != "New Name" {
		t.Errorf("expected New Name, got %v", result["displayName"])
	}
}

func TestCovSCIM_ApplyPatch_Remove(t *testing.T) {
	attrs := map[string]any{"displayName": "Remove Me", "userName": "keep@test.com"}
	ops := []PatchOperation{
		{Op: "remove", Path: "displayName"},
	}
	result, err := ApplyPatch(attrs, ops)
	if err != nil {
		t.Fatalf("ApplyPatch: %v", err)
	}
	if _, exists := result["displayName"]; exists {
		t.Error("expected displayName removed")
	}
	if result["userName"] != "keep@test.com" {
		t.Error("userName should remain")
	}
}

func TestCovSCIM_PatchedAttrsToSCIMUser(t *testing.T) {
	attrs := map[string]any{
		"userName":    "patched@test.com",
		"displayName": "Patched User",
		"active":      true,
	}
	user := PatchedAttrsToSCIMUser(attrs)
	if user.UserName != "patched@test.com" {
		t.Error("expected userName")
	}
}
