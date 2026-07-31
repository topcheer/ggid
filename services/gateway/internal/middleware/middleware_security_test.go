package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TestTenantResolver_HeaderPriority tests that X-Tenant-ID header takes priority over JWT tenant_id.
// This demonstrates the potential for tenant confusion attacks.
func TestTenantResolver_HeaderPriority(t *testing.T) {
	_, pubPath := generateTestRSAKey(t)
	defer os.Remove(pubPath)

	_, err := NewJWKSClient("", pubPath)
	if err != nil {
		t.Fatalf("create JWKS: %v", err)
	}

	// Create a JWT for tenant A
	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	privKey, _ := generateTestRSAKey(t)
	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"tenant_id": tenantA.String(),
		"scope":     "openid profile",
		"exp":       jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tokenA := signTestJWT(t, privKey, claims)

	// Test: Request with JWT for tenantA but X-Tenant-ID header set to tenantB
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("X-Tenant-ID", tenantB.String()) // Try to access tenant B's data!

	w := httptest.NewRecorder()

	// Apply TenantResolver middleware
	handler := TenantResolver("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check which tenant ID was resolved
		tenantID, ok := TenantIDFromRequest(r)
		if !ok {
			t.Error("expected tenant ID in context")
		}

		// SECURITY (R25 P0 fix): JWT claim (tenantA) MUST take priority
		// over client-supplied X-Tenant-ID header (tenantB).
		if tenantID != tenantA.String() {
			t.Errorf("SECURITY: expected JWT tenant %s to override header tenant %s, got %s",
				tenantA, tenantB, tenantID)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("unexpected status: %d", w.Code)
	}

	t.Logf("PASS: Request with JWT for tenant %s correctly ignored spoofed X-Tenant-ID header for tenant %s",
		tenantA, tenantB)
}

// TestHasAdminScope_RoleVsScope tests that hasAdminScope checks scopes correctly.
// Note: Roles should NEVER be passed to hasAdminScope after the security fix.
func TestHasAdminScope_RoleVsScope(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		expected bool
	}{
		{
			name:     "platform:admin scope returns true",
			scopes:   []string{"platform:admin"},
			expected: true,
		},
		{
			name:     "tenant:admin scope returns true",
			scopes:   []string{"tenant:admin"},
			expected: true,
		},
		{
			name:     "non-admin scope returns false",
			scopes:   []string{"openid", "profile"},
			expected: false,
		},
		{
			name:     "admin scope (not namespaced) returns false",
			scopes:   []string{"admin", "administrator"},
			expected: false, // This is correct - loose names should not be trusted
		},
		{
			name:     "platform:admin with other scopes returns true",
			scopes:   []string{"openid", "profile", "platform:admin"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := hasAdminScope(tt.scopes); result != tt.expected {
				t.Errorf("hasAdminScope(scopes=%v) = %v, want %v", tt.scopes, result, tt.expected)
			}
		})
	}

	// SECURITY NOTE: The hasAdminScope function now ONLY accepts scopes.
	// Roles are never checked - they are tenant-controlled and can be forged.
	// Platform:admin in roles no longer grants cross-tenant access.
}

// TestTenantIDOverride_Vulnerability demonstrates the tenant ID override vulnerability.
func TestTenantIDOverride_Vulnerability(t *testing.T) {
	_, pubPath := generateTestRSAKey(t)
	defer os.Remove(pubPath)

	// Create a regular tenant admin JWT (no platform:admin)
	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	privKey, _ := generateTestRSAKey(t)
	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"tenant_id": tenantA.String(),
		"scope":     "openid profile tenant:admin",
		"exp":       jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token := signTestJWT(t, privKey, claims)

	// Test: Tenant A admin tries to access Tenant B's data
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantB.String())

	w := httptest.NewRecorder()

	// Apply middleware chain
	jwks, _ := NewJWKSClient("", pubPath)
	handler := TenantResolver("")(JWTAuth(jwks, true, "", "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := TenantIDFromRequest(r)
		w.Write([]byte(tenantID))
	})))

	handler.ServeHTTP(w, req)

	// The context now has tenantB even though the JWT is for tenantA
	if w.Code == http.StatusOK {
		respBody := w.Body.String()
		t.Logf("Context tenant_id: %s", respBody)
		t.Logf("SECURITY ISSUE: Tenant %s admin was able to set context to tenant %s",
			tenantA, tenantB)

		if respBody == tenantB.String() {
			t.Error("VULNERABILITY: Tenant ID from header overrode JWT tenant_id!")
		}
	}
}

// TestJWTAuth_PlatformAdminInRolesClaim tests that platform:admin in roles claim allows cross-tenant.
func TestJWTAuth_PlatformAdminInRolesClaim(t *testing.T) {
	privKey, pubPath := generateTestRSAKey(t)
	defer os.Remove(pubPath)

	jwks, err := NewJWKSClient("", pubPath)
	if err != nil {
		t.Fatalf("create JWKS: %v", err)
	}

	// Create a JWT with platform:admin in the ROLES claim (not scope)
	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"tenant_id": tenantA.String(),
		"scope":     "openid profile",                // No platform:admin in scopes
		"roles":     []interface{}{"platform:admin"}, // But in roles!
		"exp":       jwt.NewNumericDate(time.Now().Add(time.Hour)),
		"iss":       "test-issuer",
		"aud":       "test-audience",
	}
	token := signTestJWT(t, privKey, claims)

	// Test: Try to access tenant B's data
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantB.String())

	w := httptest.NewRecorder()

	handler := JWTAuth(jwks, true, "test-issuer", "test-audience")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	// BUG: The request might succeed if the roles claim is trusted for platform:admin
	// This is a critical security vulnerability!
	if w.Code == http.StatusOK {
		t.Error("CRITICAL BUG: platform:admin in roles claim (tenant-controlled) allowed cross-tenant access!")
		t.Logf("SECURITY ISSUE: User from tenant %s accessed tenant %s using platform:admin in roles claim",
			tenantA, tenantB)
	} else {
		t.Logf("Good: Request was blocked (status %d)", w.Code)
	}
}
