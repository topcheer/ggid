package router

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mkTestJWT builds an unsigned JWT string (ExtractJWTClaims does not verify
// signatures; JWTAuth does that upstream).
func mkTestJWT(claims map[string]any) string {
	payload, _ := json.Marshal(claims)
	return "x." + base64.RawURLEncoding.EncodeToString(payload) + ".y"
}

func checkScopeRequest(t *testing.T, path, authHeader string) *httptest.ResponseRecorder {
	t.Helper()
	gw := &Gateway{}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", "Bearer "+authHeader)
	}
	rec := httptest.NewRecorder()
	gw.checkRouteScope(rec, req)
	return rec
}

func TestCheckRouteScope_OAuthRolesClaim(t *testing.T) {
	// OAuth-issued token: scope contains only OIDC scopes; admin role is in
	// the roles claim. Platform-level role names only grant platform access
	// when the token belongs to the platform tenant (role names are
	// forgeable by tenant admins in other tenants).
	token := mkTestJWT(map[string]any{
		"sub":       "u1",
		"scope":     "openid profile email",
		"roles":     []string{"Administrator"},
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	})
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code == http.StatusForbidden {
		t.Error("OAuth token with roles=[Administrator] must access /api/v1/users")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", token); rec.Code == http.StatusForbidden {
		t.Error("Administrator (platform-level) must access /api/v1/system/")
	}
}

// TestCheckRouteScope_ForgedAdminRoleDenied: a user in a non-platform tenant
// whose tenant admin created a role named "Administrator" must NOT gain
// platform admin access (privilege-escalation regression).
func TestCheckRouteScope_ForgedAdminRoleDenied(t *testing.T) {
	token := mkTestJWT(map[string]any{
		"sub":       "attacker",
		"scope":     "openid profile email",
		"roles":     []string{"Administrator"},
		"tenant_id": "00000007-0000-0000-0000-000000000001",
	})
	// Tenant-level paths still work (tenant admins are sovereign in their tenant).
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code == http.StatusForbidden {
		t.Error("tenant-local Administrator role should still access tenant admin paths")
	}
	// Platform-only paths must be denied.
	if rec := checkScopeRequest(t, "/api/v1/system/config", token); rec.Code != http.StatusForbidden {
		t.Error("forged Administrator role in non-platform tenant must NOT access /api/v1/system/")
	}
}

func TestCheckRouteScope_TenantAdminRole(t *testing.T) {
	token := mkTestJWT(map[string]any{
		"sub":   "u2",
		"scope": "openid profile email",
		"roles": []string{"Tenant Administrator"},
	})
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code == http.StatusForbidden {
		t.Error("Tenant Administrator must access /api/v1/users")
	}
	// But not platform-only paths.
	if rec := checkScopeRequest(t, "/api/v1/system/config", token); rec.Code != http.StatusForbidden {
		t.Error("Tenant Administrator must NOT access /api/v1/system/")
	}
}

func TestCheckRouteScope_LegacyScopeStillWorks(t *testing.T) {
	token := mkTestJWT(map[string]any{
		"sub":   "u3",
		"scope": "platform:admin",
	})
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code == http.StatusForbidden {
		t.Error("legacy platform:admin scope must still work")
	}
}

func TestCheckRouteScope_RegularUserDenied(t *testing.T) {
	token := mkTestJWT(map[string]any{
		"sub":   "u4",
		"scope": "openid profile email",
		"roles": []string{"Viewer"},
	})
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code != http.StatusForbidden {
		t.Error("Viewer must be denied on /api/v1/users")
	}
	// Non-admin path is allowed.
	if rec := checkScopeRequest(t, "/api/v1/flows", token); rec.Code == http.StatusForbidden {
		t.Error("Viewer must access non-admin paths")
	}
}

func TestCheckRouteScope_NoClaimsPasses(t *testing.T) {
	// No JWT → defer to JWTAuth (401 upstream), never 403 here.
	if rec := checkScopeRequest(t, "/api/v1/users", ""); rec.Code == http.StatusForbidden {
		t.Error("anonymous request should defer to JWTAuth, not 403")
	}
}
