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

// TestCheckRouteScope_EmptyRolesDeniedOnAdminPaths: P0 regression — a
// newly registered user with roles=[] and only OIDC scopes must be denied
// on admin-only paths including the bare /oauth/clients route (which the
// gateway proxies without the /api/v1 prefix).
func TestCheckRouteScope_EmptyRolesDeniedOnAdminPaths(t *testing.T) {
	token := mkTestJWT(map[string]any{
		"sub":   "newuser",
		"scope": "openid profile email",
		"roles": []string{},
	})

	for _, path := range []string{
		"/api/v1/oauth/clients",
		"/oauth/clients",
		"/api/v1/users",
		"/api/v1/users/123e4567-e89b-12d3-a456-426614174000",
	} {
		rec := checkScopeRequest(t, path, token)
		if rec.Code != http.StatusForbidden {
			t.Errorf("path %s: expected 403 for empty-roles user, got %d", path, rec.Code)
		}
	}
}

func TestCheckRouteScope_OAuthRolesClaim(t *testing.T) {
	// OAuth-issued token with platform:admin in both scope and roles.
	// Platform access requires the scope-style key, not the display name.
	token := mkTestJWT(map[string]any{
		"sub":       "u1",
		"scope":     "openid profile email platform:admin",
		"roles":     []string{"platform:admin"},
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	})
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code == http.StatusForbidden {
		t.Error("OAuth token with platform:admin must access /api/v1/users")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", token); rec.Code == http.StatusForbidden {
		t.Error("platform:admin scope must access /api/v1/system/")
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
	// Non-platform tenant with only a role name — no admin access at all.
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code != http.StatusForbidden {
		t.Error("forged Administrator role without admin scope must NOT access admin paths")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", token); rec.Code != http.StatusForbidden {
		t.Error("forged Administrator role must NOT access platform paths")
	}
}

func TestCheckRouteScope_TenantAdminRole(t *testing.T) {
	token := mkTestJWT(map[string]any{
		"sub":   "u2",
		"scope": "openid profile email tenant:admin",
	})
	if rec := checkScopeRequest(t, "/api/v1/users", token); rec.Code == http.StatusForbidden {
		t.Error("tenant:admin scope must access /api/v1/users")
	}
	if rec := checkScopeRequest(t, "/api/v1/system/config", token); rec.Code != http.StatusForbidden {
		t.Error("tenant:admin must NOT access platform-only paths")
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
