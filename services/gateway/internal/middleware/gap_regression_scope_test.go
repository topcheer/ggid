package middleware

// Gap Regression Verification Test
// Verifies: Gap #13 — HasScope Enforcement (DONE, MEDIUM confidence → now HIGH)
// Method: Functional tests covering wildcard scopes, API key vs JWT priority,
//         empty/nil scopes, and negative cases.
// Date: 2026-07-24

import (
	"context"
	"testing"
)

// ========== GAP #13: HasScope Enforcement — Functional Verification ==========

// TestGapRegression_HasScope_WildcardScope verifies that a "*" scope grants
// access to any requested scope.
func TestGapRegression_HasScope_WildcardScope(t *testing.T) {
	ctx := context.WithValue(context.Background(), APIKeyScopesKey, []string{"*"})

	if !HasScope(ctx, "read") {
		t.Error("wildcard scope '*' should grant 'read'")
	}
	if !HasScope(ctx, "admin") {
		t.Error("wildcard scope '*' should grant 'admin'")
	}
	if !HasScope(ctx, "any-scope-imaginable") {
		t.Error("wildcard scope '*' should grant any scope")
	}
}

// TestGapRegression_HasScope_MultipleScopes verifies that having multiple
// scopes grants access to each individually but not to missing ones.
func TestGapRegression_HasScope_MultipleScopes(t *testing.T) {
	ctx := context.WithValue(context.Background(), APIKeyScopesKey, []string{"read", "write", "delete"})

	if !HasScope(ctx, "read") {
		t.Error("should have 'read' scope")
	}
	if !HasScope(ctx, "write") {
		t.Error("should have 'write' scope")
	}
	if !HasScope(ctx, "delete") {
		t.Error("should have 'delete' scope")
	}
	if HasScope(ctx, "admin") {
		t.Error("should NOT have 'admin' scope")
	}
	if HasScope(ctx, "execute") {
		t.Error("should NOT have 'execute' scope")
	}
}

// TestGapRegression_HasScope_APIKeyTakesPriority verifies that when both API key
// scopes and JWT scopes exist, API key scopes take priority.
func TestGapRegression_HasScope_APIKeyTakesPriority(t *testing.T) {
	ctx := context.WithValue(context.Background(), APIKeyScopesKey, []string{"read"})
	ctx = context.WithValue(ctx, jwtScopesKey, []string{"admin"})

	// Should use API key scopes, not JWT scopes
	if !HasScope(ctx, "read") {
		t.Error("should have 'read' from API key scopes (priority)")
	}
	if HasScope(ctx, "admin") {
		t.Error("should NOT have 'admin' — API key scopes take priority over JWT scopes")
	}
}

// TestGapRegression_HasScope_FallbackToJWT verifies that when no API key scopes
// exist, JWT scopes are checked as fallback.
func TestGapRegression_HasScope_FallbackToJWT(t *testing.T) {
	ctx := context.WithValue(context.Background(), jwtScopesKey, []string{"openid", "profile", "email"})

	if !HasScope(ctx, "openid") {
		t.Error("should have 'openid' from JWT scopes")
	}
	if !HasScope(ctx, "profile") {
		t.Error("should have 'profile' from JWT scopes")
	}
	if HasScope(ctx, "admin") {
		t.Error("should NOT have 'admin' from JWT scopes")
	}
}

// TestGapRegression_HasScope_DenyByDefault verifies that empty context
// results in denial (P0 security fix: was always-true before fix).
func TestGapRegression_HasScope_DenyByDefault(t *testing.T) {
	// Completely empty context
	if HasScope(context.Background(), "read") {
		t.Error("empty context should deny scope (P0 security fix — was always-true)")
	}

	// Context with unrelated values
	ctx := context.WithValue(context.Background(), "unrelated-key", "value")
	if HasScope(ctx, "read") {
		t.Error("context without scope keys should deny scope")
	}
}

// TestGapRegression_HasScope_EmptyScopesList verifies that an empty scope list
// in context denies all scopes.
func TestGapRegression_HasScope_EmptyScopesList(t *testing.T) {
	ctx := context.WithValue(context.Background(), APIKeyScopesKey, []string{})

	if HasScope(ctx, "read") {
		t.Error("empty scope list should deny all scopes")
	}
}

// TestGapRegression_HasScope_WildcardInJWTScopes verifies wildcard works
// in JWT scopes too.
func TestGapRegression_HasScope_WildcardInJWTScopes(t *testing.T) {
	ctx := context.WithValue(context.Background(), jwtScopesKey, []string{"*"})

	if !HasScope(ctx, "admin") {
		t.Error("wildcard '*' in JWT scopes should grant 'admin'")
	}
}

// TestGapRegression_HasScope_SecurityRegression verifies that the specific
// vulnerability (always returning true) cannot be reintroduced.
// This is the critical regression test for the P0 fix.
func TestGapRegression_HasScope_SecurityRegression(t *testing.T) {
	// Test with 20 random scope names — none should be granted from empty context
	scopes := []string{
		"admin", "superuser", "root", "write", "delete",
		"execute", "manage", "create", "update", "purge",
		"import", "export", "audit", "config", "deploy",
		"billing", "users", "roles", "policies", "tokens",
	}

	for _, scope := range scopes {
		ctx := context.WithValue(context.Background(), APIKeyScopesKey, []string{"read-only"})
		if HasScope(ctx, scope) {
			t.Fatalf("SECURITY REGRESSION: HasScope returned true for '%s' when only 'read-only' was granted — P0 bypass!", scope)
		}
	}
}
