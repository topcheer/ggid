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

// TestHasAdminScope_PlatformAdmin verifies that "platform:admin" scope grants access.
func TestHasAdminScope_PlatformAdmin(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"read", "write", "platform:admin"})

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return true when 'platform:admin' scope is present")
	}
}

// TestHasAdminScope_TenantAdmin verifies that "tenant:admin" scope grants access.
func TestHasAdminScope_TenantAdmin(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"read", "tenant:admin"})

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return true when 'tenant:admin' scope is present")
	}
}

// TestHasAdminScope_ForgeableNamesRejected verifies that raw role names
// like "admin"/"administrator" do NOT grant access (privilege escalation).
func TestHasAdminScope_ForgeableNamesRejected(t *testing.T) {
	gw := &Gateway{}
	for _, scope := range []string{"admin", "administrator", "ggid:admin", "superadmin", "*"} {
		req := makeRequestWithScopes(t, []string{scope})
		if gw.hasAdminScope(req) {
			t.Errorf("hasAdminScope should reject forgeable scope %q", scope)
		}
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
	req := makeRequestWithScopeString(t, "read write platform:admin delete")

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should return true when 'platform:admin' is in scope string")
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
// bypass the admin scope check.
func TestHasAdminScope_WildcardNotAdmin(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopes(t, []string{"*"})

	if gw.hasAdminScope(req) {
		t.Error("hasAdminScope should NOT return true for wildcard '*' scope")
	}
}

// TestHasAdminScope_AdminAmongManyScopes verifies admin works even with many scopes.
func TestHasAdminScope_AdminAmongManyScopes(t *testing.T) {
	gw := &Gateway{}
	req := makeRequestWithScopeString(t, "read write delete users:create users:read roles:manage platform:admin organizations:read settings:write audit:read")

	if !gw.hasAdminScope(req) {
		t.Error("hasAdminScope should find 'platform:admin' among 10+ scopes")
	}
}
