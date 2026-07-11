package router

// Gap Regression Verification Test
// Verifies: Gap #14 — Admin API Role Check (DONE, MEDIUM → HIGH confidence)
// Method: Dedicated unit tests for hasAdminScope covering admin scope present/absent,
//         non-admin scope, empty context, multiple scopes, wildcard.
// Date: 2026-07-25

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// makeFakeJWT builds a minimal unsigned JWT with the given payload.
// The signature is a dummy "." — ExtractJWTClaims only decodes the payload.
func makeFakeJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	header := map[string]string{"alg": "none", "typ": "JWT"}
	headerB, _ := json.Marshal(header)
	payloadB, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(headerB) + "." +
		base64.RawURLEncoding.EncodeToString(payloadB) + "."
}

// makeRequestWithScopes creates a request with a Bearer JWT containing the given scopes.
func makeRequestWithScopes(t *testing.T, scopes []string) *http.Request {
	t.Helper()
	payload := map[string]any{
		"sub":   "user-123",
		"scope": scopes,
	}
	token := makeFakeJWT(t, payload)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/routes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// makeRequestWithScopeString creates a request with scopes as a space-delimited string.
func makeRequestWithScopeString(t *testing.T, scopeStr string) *http.Request {
	t.Helper()
	payload := map[string]any{
		"sub":   "user-123",
		"scope": scopeStr,
	}
	token := makeFakeJWT(t, payload)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/routes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// ========== GAP #14: hasAdminScope — Dedicated Unit Tests ==========

// TestHasAdminScope_AdminScopePresent verifies that "admin" scope grants access.
func TestHasAdminScope_AdminScopePresent(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"read", "write", "admin"})

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return true when 'admin' scope is present")
	}
}

// TestHasAdminScope_GgidAdminScopePresent verifies that "ggid:admin" scope grants access.
func TestHasAdminScope_GgidAdminScopePresent(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"read", "ggid:admin"})

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return true when 'ggid:admin' scope is present")
	}
}

// TestHasAdminScope_NonAdminScope verifies that non-admin scopes are rejected.
func TestHasAdminScope_NonAdminScope(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"read", "write", "delete"})

	if gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return false for non-admin scopes")
	}
}

// TestHasAdminScope_EmptyContext verifies that a request with no JWT returns false.
func TestHasAdminScope_EmptyContext(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/routes", nil)
	// No Authorization header

	if gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return false when no JWT is present")
	}
}

// TestHasAdminScope_EmptyScopes verifies that a JWT with empty scopes returns false.
func TestHasAdminScope_EmptyScopes(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{})

	if gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return false when scopes list is empty")
	}
}

// TestHasAdminScope_ScopeString verifies that scopes as a space-delimited string work.
func TestHasAdminScope_ScopeString(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopeString(t, "read write admin delete")

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return true when 'admin' is in scope string")
	}
}

// TestHasAdminScope_MalformedJWT verifies that a malformed JWT is rejected.
func TestHasAdminScope_MalformedJWT(t *testing.T) {
	gw := &Gateway{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/routes", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.jwt.token")

	if gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return false for malformed JWT")
	}
}

// TestHasAdminScope_WildcardNotAdmin verifies that "*" wildcard does NOT
// bypass the admin scope check (hasAdminScope is strict — requires explicit admin).
func TestHasAdminScope_WildcardNotAdmin(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"*"})

	// hasAdminScope checks for literal "admin" or "ggid:admin" — not "*"
	// This is correct behavior: admin scope must be explicitly granted.
	if gw.hasAdminScope(req) {
		t.Error("hasAdminScope should NOT return true for wildcard '*' scope — admin must be explicit")
	}
}

// TestHasAdminScope_AdminAmongManyScopes verifies admin works even with many scopes.
func TestHasAdminScope_AdminAmongManyScopes(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopeString(t, "read write delete users:create users:read roles:manage admin organizations:read settings:write audit:read")

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should find 'admin' among 10+ scopes")
	}
}
