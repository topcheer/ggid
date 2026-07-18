package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// --- Helpers ---

// newSecurityTestGateway creates a Gateway with test routes for all protected endpoints.
func newSecurityTestGateway(t *testing.T) *Gateway {
	t.Helper()
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/users":          "http://localhost:18001",
		"/api/v1/roles":          "http://localhost:18002",
		"/api/v1/policies":       "http://localhost:18003",
		"/api/v1/audit":          "http://localhost:18004",
		"/api/v1/auth":           "http://localhost:18005",
		"/api/v1/oauth":          "http://localhost:18006",
		"/api/v1/policy":         "http://localhost:18003",
	}
	jwks, err := middleware.NewJWKSClient("", "")
	if err != nil {
		t.Fatalf("failed to create JWKS client: %v", err)
	}
	return New(cfg, jwks)
}

// protectedPaths is the list of protected API endpoints to test.
var protectedPaths = []struct {
	path   string
	method string
}{
	{"/api/v1/users", http.MethodGet},
	{"/api/v1/users", http.MethodPost},
	{"/api/v1/users/550e8400-e29b-41d4-a716-446655440000", http.MethodGet},
	{"/api/v1/users/550e8400-e29b-41d4-a716-446655440000", http.MethodPut},
	{"/api/v1/users/550e8400-e29b-41d4-a716-446655440000", http.MethodDelete},
	{"/api/v1/roles", http.MethodGet},
	{"/api/v1/roles", http.MethodPost},
	{"/api/v1/roles/550e8400-e29b-41d4-a716-446655440000", http.MethodDelete},
	{"/api/v1/policies", http.MethodGet},
	{"/api/v1/policies", http.MethodPost},
	{"/api/v1/policies/check", http.MethodPost},
	{"/api/v1/audit/events", http.MethodGet},
	{"/api/v1/audit/integrity", http.MethodGet},
	{"/api/v1/auth/sessions", http.MethodGet},
	{"/api/v1/auth/sessions/abc123", http.MethodDelete},
	{"/api/v1/auth/conditional-access/policies", http.MethodGet},
	{"/api/v1/auth/conditional-access/policies", http.MethodPost},
	{"/api/v1/auth/webauthn/aaguid", http.MethodGet},
	{"/api/v1/auth/webauthn/aaguid", http.MethodPost},
	{"/api/v1/auth/webauthn/aaguid/test-aaguid-id", http.MethodDelete},
	{"/api/v1/auth/mfa/enroll", http.MethodPost},
	{"/api/v1/auth/mfa/verify", http.MethodPost},
	{"/api/v1/auth/profile", http.MethodGet},
	{"/api/v1/oauth/clients", http.MethodGet},
	{"/api/v1/oauth/clients", http.MethodPost},
}

// --- No Token Tests (20 endpoints → 401) ---

func TestAPISecurity_NoToken_Users(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/users", http.MethodGet},
		{"/api/v1/users", http.MethodPost},
		{"/api/v1/users/abc-def-123", http.MethodGet},
		{"/api/v1/users/abc-def-123", http.MethodPut},
		{"/api/v1/users/abc-def-123", http.MethodDelete},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_Roles(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/roles", http.MethodGet},
		{"/api/v1/roles", http.MethodPost},
		{"/api/v1/roles/abc-def-123", http.MethodDelete},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_Policies(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/policies", http.MethodGet},
		{"/api/v1/policies", http.MethodPost},
		{"/api/v1/policies/check", http.MethodPost},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_Audit(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/audit/events", http.MethodGet},
		{"/api/v1/audit/integrity", http.MethodGet},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_AuthSessions(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/auth/sessions", http.MethodGet},
		{"/api/v1/auth/sessions/sess-123", http.MethodDelete},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_OAuthClients(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/oauth/clients", http.MethodGet},
		{"/api/v1/oauth/clients", http.MethodPost},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_ConditionalAccess(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/auth/conditional-access/policies", http.MethodGet},
		{"/api/v1/auth/conditional-access/policies", http.MethodPost},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_WebAuthnAAGUID(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/auth/webauthn/aaguid", http.MethodGet},
		{"/api/v1/auth/webauthn/aaguid", http.MethodPost},
		{"/api/v1/auth/webauthn/aaguid/test-id", http.MethodDelete},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_MFA(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range []struct{ path, method string }{
		{"/api/v1/auth/mfa/enroll", http.MethodPost},
		{"/api/v1/auth/mfa/verify", http.MethodPost},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_NoToken_Profile(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- Invalid Token Tests ---

func TestAPISecurity_InvalidToken_BearerPrefix(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range protectedPaths[:5] {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer invalid.jwt.token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with invalid token: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_InvalidToken_MalformedHeader(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range protectedPaths[5:10] {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "NotBearer some-token")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with malformed auth header: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

func TestAPISecurity_InvalidToken_EmptyBearer(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("empty bearer token: expected 401, got %d", rr.Code)
	}
}

func TestAPISecurity_InvalidToken_RawTokenNoBearer(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	req.Header.Set("Authorization", "rawtoken123456")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("token without Bearer prefix: expected 401, got %d", rr.Code)
	}
}

func TestAPISecurity_InvalidToken_GarbageJWT(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, tc := range protectedPaths[10:15] {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer aaa.bbb.ccc")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s %s with garbage JWT: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

// --- Public Path Tests (should NOT require token) ---

func TestAPISecurity_PublicPaths_Login(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should not get 401 for login (public path). May get 400/422 for empty body.
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("login should be public, got 401")
	}
}

func TestAPISecurity_PublicPaths_Register(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("register should be public, got 401")
	}
}

func TestAPISecurity_PublicPaths_Healthz(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("/healthz: expected 200, got %d", rr.Code)
	}
}

func TestAPISecurity_PublicPaths_OAuthToken(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/oauth/token", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("oauth/token should be public, got 401")
	}
}

func TestAPISecurity_PublicPaths_OAuthAuthorize(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/oauth/authorize", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("oauth/authorize should be public, got 401")
	}
}

func TestAPISecurity_PublicPaths_SystemInitialized(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/initialized", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("system/initialized should be public, got 401")
	}
}

// --- All Protected Paths Require Auth ---

func TestAPISecurity_AllProtectedPaths_Return401(t *testing.T) {
	for _, tc := range protectedPaths {
		gw := newSecurityTestGateway(t)
		handler := gw.Handler()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		// Accept 401 (unauthorized) or 429 (rate limited) — both prove access is denied
		if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusTooManyRequests {
			t.Errorf("%s %s without token: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}

// --- Method Override Tests ---

func TestAPISecurity_WrongMethod_Login(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	// Login with GET instead of POST - should not be treated as login
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Public path may still accept it or return error, but not 401
	if rr.Code == http.StatusUnauthorized {
		t.Errorf("GET login on public path should not 401, got %d", rr.Code)
	}
}

// --- X-Tenant-ID Header Tests ---

func TestAPISecurity_NoTenantID_StillAuthed(t *testing.T) {
	// Without X-Tenant-ID, the request should still go through JWT check first.
	// The tenant header is validated by backend services, not the gateway.
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	// No X-Tenant-ID header, no Authorization
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without any auth, got %d", rr.Code)
	}
}

// --- Comprehensive Invalid Auth Variations ---

func TestAPISecurity_AuthVariations_AllBlocked(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	path := "/api/v1/users"

	invalidAuthHeaders := []string{
		"Bearer ",                          // empty
		"Bearer null",                      // literal null
		"Bearer undefined",                 // literal undefined
		"Bearer 00000000.000000.00000000", // zeros
		"Basic dXNlcjpwYXNz",              // basic auth instead of bearer
		"bearer lowercase.prefix",         // lowercase bearer
		"Digest username=admin",           // digest auth
		"",                                 // empty header
	}

	for _, authHeader := range invalidAuthHeaders {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("auth=%q: expected 401, got %d", authHeader, rr.Code)
		}
	}
}

// --- Audit Chain Integrity Without Auth ---

func TestAPISecurity_AuditIntegrity_RequiresAuth(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/integrity", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("audit integrity without auth: expected 401, got %d", rr.Code)
	}
}

// --- WebAuthn Endpoints Auth ---

func TestAPISecurity_WebAuthnRegister_RequiresAuth(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, path := range []string{
		"/api/v1/auth/webauthn/register/begin",
		"/api/v1/auth/webauthn/register/finish",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s without auth: expected 401, got %d", path, rr.Code)
		}
	}
}

// --- Break-Glass Requires Auth ---

func TestAPISecurity_BreakGlass_RequiresAuth(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	for _, path := range []string{
		"/api/v1/auth/break-glass/activate",
		"/api/v1/auth/break-glass/history",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s without auth: expected 401, got %d", path, rr.Code)
		}
	}
}

// --- Privileged Operations Requires Auth ---

func TestAPISecurity_PrivilegedOps_RequiresAuth(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/identity/privileged-operations", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("privileged-operations without auth: expected 401, got %d", rr.Code)
	}
}

// --- CAE Evaluation Requires Auth ---

func TestAPISecurity_CAEEvaluation_RequiresAuth(t *testing.T) {
	gw := newSecurityTestGateway(t)
	handler := gw.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/cae/run", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("cae/run without auth: expected 401, got %d", rr.Code)
	}
}
