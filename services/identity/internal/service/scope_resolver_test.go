package service

import (
	"testing"

	"github.com/google/uuid"
)

func TestScopeResolver_ResolveScopes_AllGranted(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"read", "write", "delete"})

	res, err := sr.ResolveScopes(userID, "client-1", []string{"read", "write"})
	if err != nil {
		t.Fatalf("ResolveScopes: %v", err)
	}
	if len(res.Granted) != 2 {
		t.Errorf("expected 2 granted, got %d", len(res.Granted))
	}
	if len(res.Denied) != 0 {
		t.Errorf("expected 0 denied, got %d", len(res.Denied))
	}
}

func TestScopeResolver_ResolveScopes_SomeDenied(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"read"})

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"read", "delete"})
	if len(res.Granted) != 1 {
		t.Errorf("expected 1 granted, got %d", len(res.Granted))
	}
	if len(res.Denied) != 1 {
		t.Errorf("expected 1 denied, got %d", len(res.Denied))
	}
}

func TestScopeResolver_ResolveScopes_NilUser(t *testing.T) {
	sr := NewScopeResolver()
	_, err := sr.ResolveScopes(uuid.Nil, "client-1", []string{"read"})
	if err == nil {
		t.Error("should error on nil user")
	}
}

func TestScopeResolver_ResolveScopes_EmptyClient(t *testing.T) {
	sr := NewScopeResolver()
	_, err := sr.ResolveScopes(uuid.New(), "", []string{"read"})
	if err == nil {
		t.Error("should error on empty client")
	}
}

func TestScopeResolver_ClientRestriction_Denied(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"read", "write", "delete"})
	sr.SetClientRestrictions(map[string][]string{
		"client-1": {"read"}, // client only allows read
	})

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"read", "write"})
	if len(res.Granted) != 1 {
		t.Errorf("expected 1 granted (client restricts), got %d", len(res.Granted))
	}
	if len(res.Denied) != 1 {
		t.Errorf("expected 1 denied, got %d", len(res.Denied))
	}
}

func TestScopeResolver_ClientRestriction_Wildcard(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"read", "write"})
	sr.SetClientRestrictions(map[string][]string{
		"client-1": {"*"},
	})

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"read", "write"})
	if len(res.Granted) != 2 {
		t.Errorf("expected 2 granted with wildcard client, got %d", len(res.Granted))
	}
}

func TestScopeResolver_ScopeHierarchy(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"admin"})
	sr.SetScopeHierarchy(map[string][]string{
		"admin": {"read", "write", "delete"},
	})

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"read"})
	if len(res.Granted) != 1 {
		t.Errorf("expected 1 granted via hierarchy, got %d", len(res.Granted))
	}
}

func TestScopeResolver_WildcardScopeExpansion(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"read", "write", "delete"})

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"*"})
	if len(res.Granted) != 3 {
		t.Errorf("expected 3 granted from wildcard, got %d", len(res.Granted))
	}
}

func TestScopeResolver_NoPermissions(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"read"})
	if len(res.Granted) != 0 {
		t.Errorf("expected 0 granted with no permissions, got %d", len(res.Granted))
	}
	if len(res.Denied) != 1 {
		t.Errorf("expected 1 denied, got %d", len(res.Denied))
	}
}

func TestScopeResolver_ExpandScope(t *testing.T) {
	sr := NewScopeResolver()
	sr.SetScopeHierarchy(map[string][]string{
		"admin": {"read", "write"},
		"write": {"write:own"},
	})

	expanded := sr.ExpandScope("admin")
	if len(expanded) < 3 {
		t.Errorf("expected at least 3 expanded scopes, got %d", len(expanded))
	}
}

func TestScopeResolver_GetEffectiveScopes(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"admin"})
	sr.SetScopeHierarchy(map[string][]string{
		"admin": {"read", "write"},
	})

	effective, err := sr.GetEffectiveScopes(userID, "client-1")
	if err != nil {
		t.Fatalf("GetEffectiveScopes: %v", err)
	}
	if len(effective) < 3 {
		t.Errorf("expected at least 3 effective scopes, got %d", len(effective))
	}
}

func TestScopeResolver_GetEffectiveScopes_WithClientRestriction(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"admin", "read"})
	sr.SetScopeHierarchy(map[string][]string{
		"admin": {"read", "write"},
	})
	sr.SetClientRestrictions(map[string][]string{
		"client-1": {"read"}, // only read allowed
	})

	effective, _ := sr.GetEffectiveScopes(userID, "client-1")
	for _, s := range effective {
		if s != "read" {
			t.Errorf("expected only 'read', got '%s'", s)
		}
	}
}

func TestScopeResolver_Reset(t *testing.T) {
	sr := NewScopeResolver()
	userID := uuid.New()
	sr.SetUserPermissions(userID, []string{"read"})
	sr.Reset()

	res, _ := sr.ResolveScopes(userID, "client-1", []string{"read"})
	if len(res.Granted) != 0 {
		t.Error("should have 0 granted after reset")
	}
}