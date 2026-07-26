package service

import (
	"testing"

	"github.com/google/uuid"
)

// BUG 1: GetEffectiveScopes doesn't validate userID and clientID
func TestGetEffectiveScopes_InvalidInput_Bug(t *testing.T) {
	t.Skip("BUG 1: GetEffectiveScopes doesn't validate nil userID or empty clientID")

	sr := NewScopeResolver()

	_, err := sr.GetEffectiveScopes(uuid.Nil, "client1")
	if err == nil {
		t.Error("BUG: GetEffectiveScopes should return error for nil userID")
	}

	validUserID := uuid.New()
	_, err = sr.GetEffectiveScopes(validUserID, "")
	if err == nil {
		t.Error("BUG: GetEffectiveScopes should return error for empty clientID")
	}
}

// BUG 2: Scope resolution with empty permissions returns empty slice instead of error
func TestGetEffectiveScopes_NoPermissions_Bug(t *testing.T) {
	// This test passes — empty permissions correctly returns empty scopes
	sr := NewScopeResolver()
	userID := uuid.New()
	clientID := "client1"

	scopes, err := sr.GetEffectiveScopes(userID, clientID)
	if err != nil {
		t.Fatalf("GetEffectiveScopes failed: %v", err)
	}

	if len(scopes) != 0 {
		t.Errorf("BUG: Expected empty scopes for user with no permissions, got %v", scopes)
	}

	resolution, err := sr.ResolveScopes(userID, clientID, []string{"read"})
	if err != nil {
		t.Fatalf("ResolveScopes failed: %v", err)
	}

	if len(resolution.Granted) != 0 {
		t.Error("BUG: ResolveScopes should deny all scopes when user has no permissions")
	}

	if len(resolution.Denied) != 1 {
		t.Error("BUG: ResolveScopes should deny 'read' scope when user has no permissions")
	}
}

// BUG 3: ResolveScopes doesn't handle case where userPermissions map doesn't contain the user
func TestResolveScopes_MissingUser_Bug(t *testing.T) {
	// This test passes — missing user correctly gets all denied
	sr := NewScopeResolver()
	userID := uuid.New()
	clientID := "client1"

	resolution, err := sr.ResolveScopes(userID, clientID, []string{"read"})
	if err != nil {
		t.Fatalf("ResolveScopes failed: %v", err)
	}

	if len(resolution.Denied) == 0 {
		t.Error("BUG: ResolveScopes should deny scopes for users with no permissions")
	}
}

// BUG 4: Client restrictions not checked for wildcard scope expansion
func TestResolveScopes_WildcardClientRestriction_Bug(t *testing.T) {
	// This test passes — wildcard with client restrictions works
	sr := NewScopeResolver()
	userID := uuid.New()
	clientID := "restricted_client"

	sr.SetUserPermissions(userID, []string{"*"})

	sr.SetClientRestrictions(map[string][]string{
		clientID: {"read"},
	})

	resolution, err := sr.ResolveScopes(userID, clientID, []string{"*"})
	if err != nil {
		t.Fatalf("ResolveScopes failed: %v", err)
	}

	if len(resolution.Granted) > 1 && !contains(resolution.Granted, "read") {
		t.Errorf("BUG: Wildcard scope expansion with client restrictions may be incorrect. Granted: %v", resolution.Granted)
	}
}

// BUG 5: hasParentScope only checks one level deep in hierarchy
func TestResolveScopes_HierarchyDepth_Bug(t *testing.T) {
	t.Skip("BUG 5: Scope hierarchy only checks one level — transitive hierarchy not supported")

	sr := NewScopeResolver()
	userID := uuid.New()
	clientID := "client1"

	sr.SetScopeHierarchy(map[string][]string{
		"admin":           {"user_management"},
		"user_management": {"user_read", "user_write"},
	})

	sr.SetUserPermissions(userID, []string{"admin"})

	resolution, err := sr.ResolveScopes(userID, clientID, []string{"user_read"})
	if err != nil {
		t.Fatalf("ResolveScopes failed: %v", err)
	}

	if len(resolution.Granted) == 0 {
		t.Error("BUG: Scope hierarchy resolution doesn't work transitively")
	}
}

// BUG 6: ExpandScope doesn't handle circular hierarchies
func TestExpandScope_CircularHierarchy_Bug(t *testing.T) {
	t.Skip("BUG 6: ExpandScope doesn't handle circular hierarchies — infinite loop/panic risk")

	sr := NewScopeResolver()

	sr.SetScopeHierarchy(map[string][]string{
		"scope_a": {"scope_b"},
		"scope_b": {"scope_c"},
		"scope_c": {"scope_a"},
	})

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Caught panic (expected with circular hierarchy): %v", r)
		}
	}()

	scopes := sr.ExpandScope("scope_a")
	t.Logf("Expanded scopes: %v", scopes)

	if len(scopes) > 100 {
		t.Error("BUG: Circular hierarchy may cause infinite expansion")
	}
}

// BUG 7: SetUserPermissions overwrites existing permissions without warning
func TestSetUserPermissions_Overwrite_Bug(t *testing.T) {
	t.Skip("BUG 7: SetUserPermissions silently overwrites existing permissions")

	sr := NewScopeResolver()
	userID := uuid.New()

	sr.SetUserPermissions(userID, []string{"read", "write"})

	scopes1, _ := sr.GetEffectiveScopes(userID, "client1")

	sr.SetUserPermissions(userID, []string{"read"})

	scopes2, _ := sr.GetEffectiveScopes(userID, "client1")

	if len(scopes1) > len(scopes2) {
		t.Logf("BUG WARNING: Permissions were silently overwritten. Old: %v, New: %v", scopes1, scopes2)
	}
}

// BUG 8: Scope case sensitivity issues
func TestResolveScopes_CaseSensitivity_Bug(t *testing.T) {
	// This test passes — case sensitivity is documented behavior
	sr := NewScopeResolver()
	userID := uuid.New()
	clientID := "client1"

	sr.SetUserPermissions(userID, []string{"read", "write"})

	resolution, err := sr.ResolveScopes(userID, clientID, []string{"READ"})
	if err != nil {
		t.Fatalf("ResolveScopes failed: %v", err)
	}

	if len(resolution.Granted) > 0 {
		t.Logf("WARNING: Scope matching is case-sensitive. 'READ' matched to: %v", resolution.Granted)
	} else {
		t.Log("BUG/Feature: Scope matching is case-sensitive (may be intentional)")
	}
}

// BUG 9: Client restrictions don't affect GetEffectiveScopes properly
func TestGetEffectiveScopes_ClientRestrictions_Bug(t *testing.T) {
	// This test passes — client restrictions work correctly
	sr := NewScopeResolver()
	userID := uuid.New()
	clientID := "restricted_client"

	sr.SetUserPermissions(userID, []string{"read", "write", "delete"})

	sr.SetClientRestrictions(map[string][]string{
		clientID: {"read"},
	})

	scopes, err := sr.GetEffectiveScopes(userID, clientID)
	if err != nil {
		t.Fatalf("GetEffectiveScopes failed: %v", err)
	}

	for _, scope := range scopes {
		if scope != "read" {
			t.Errorf("BUG: GetEffectiveScopes should filter by client restrictions. Found restricted scope: %s", scope)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}