package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// claimsToVerified converts a raw claim map into the signature-verified
// JWTCClaims representation. Since R226 P0 the gateway never parses unsigned
// JWTs, so tests inject verified claims directly (as JWTAuth would).
func claimsToVerified(claims map[string]any) middleware.JWTCClaims {
	c := middleware.JWTCClaims{}
	if v, ok := claims["sub"].(string); ok {
		c.Subject = v
	}
	if v, ok := claims["tenant_id"].(string); ok {
		c.TenantID = v
	}
	if v, ok := claims["scope"].(string); ok {
		c.Scopes = strings.Fields(v)
	}
	switch v := claims["roles"].(type) {
	case []string:
		c.Roles = v
	case []any:
		for _, r := range v {
			if str, ok := r.(string); ok {
				c.Roles = append(c.Roles, str)
			}
		}
	}
	return c
}

func checkScopeRequest(t *testing.T, path string, claims map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	gw := &Gateway{}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(middleware.WithVerifiedClaims(req.Context(), claimsToVerified(claims)))
	rec := httptest.NewRecorder()
	gw.checkRouteScope(rec, req)
	return rec
}

// TestCheckRouteScope_EmptyRolesDeniedOnAdminPaths: P0 regression — a
// newly registered user with roles=[] and only OIDC scopes must be denied
// on admin-only paths including the bare /oauth/clients route (which the
// gateway proxies without the /api/v1 prefix).
func TestCheckRouteScope_EmptyRolesDeniedOnAdminPaths(t *testing.T) {
	claims := map[string]any{
		"sub":   "newuser",
		"scope": "openid profile email",
		"roles": []string{},
	}

	for _, path := range []string{
		"/api/v1/oauth/clients",
		"/oauth/clients",
		"/api/v1/users",
		"/api/v1/users/123e4567-e89b-12d3-a456-426614174000",
	} {
		rec := checkScopeRequest(t, path, claims)
		if rec.Code != http.StatusForbidden {
			t.Errorf("path %s: expected 403 for empty-roles user, got %d", path, rec.Code)
		}
	}
}

func TestCheckRouteScope_OAuthRolesClaim(t *testing.T) {
	// OAuth-issued token with BOTH platform:admin AND tenant:admin.
	// Setup wizard assigns both roles to bootstrap admin users.
	claims := map[string]any{
		"sub":       "u1",
		"scope":     "openid profile email platform:admin tenant:admin",
		"roles":     []string{"platform:admin", "tenant:admin"},
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	}
	if rec := checkScopeRequest(t, "/api/v1/users", claims); rec.Code == http.StatusForbidden {
		t.Error("OAuth token with platform:admin + tenant:admin must access /api/v1/users")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", claims); rec.Code == http.StatusForbidden {
		t.Error("platform:admin scope must access /api/v1/system/")
	}
}

// TestCheckRouteScope_PlatformOnlyWithoutTenantDenied: platform:admin
// WITHOUT tenant:admin must NOT access tenant-level admin paths.
func TestCheckRouteScope_PlatformOnlyWithoutTenantDenied(t *testing.T) {
	claims := map[string]any{
		"sub":       "u5",
		"scope":     "openid profile email platform:admin",
		"roles":     []string{"platform:admin"},
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	}
	// Platform paths still accessible
	if rec := checkScopeRequest(t, "/api/v1/system/config", claims); rec.Code == http.StatusForbidden {
		t.Error("platform:admin must access platform-only paths")
	}
	// Tenant admin paths now DENIED (no auto-inheritance)
	if rec := checkScopeRequest(t, "/api/v1/users", claims); rec.Code != http.StatusForbidden {
		t.Error("platform:admin WITHOUT tenant:admin must be denied /api/v1/users")
	}
}

// TestCheckRouteScope_ForgedAdminRoleDenied: a user in a non-platform tenant
// whose tenant admin created a role named "Administrator" must NOT gain
// platform admin access (privilege-escalation regression).
func TestCheckRouteScope_ForgedAdminRoleDenied(t *testing.T) {
	claims := map[string]any{
		"sub":       "attacker",
		"scope":     "openid profile email",
		"roles":     []string{"Administrator"},
		"tenant_id": "00000007-0000-0000-0000-000000000001",
	}
	// Non-platform tenant with only a role name — no admin access at all.
	if rec := checkScopeRequest(t, "/api/v1/users", claims); rec.Code != http.StatusForbidden {
		t.Error("forged Administrator role without admin scope must NOT access admin paths")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", claims); rec.Code != http.StatusForbidden {
		t.Error("forged Administrator role must NOT access platform paths")
	}
}

func TestCheckRouteScope_TenantAdminRole(t *testing.T) {
	claims := map[string]any{
		"sub":   "u2",
		"scope": "openid profile email tenant:admin",
	}
	if rec := checkScopeRequest(t, "/api/v1/users", claims); rec.Code == http.StatusForbidden {
		t.Error("tenant:admin scope must access /api/v1/users")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", claims); rec.Code != http.StatusForbidden {
		t.Error("tenant:admin must NOT access platform-only paths")
	}
}
