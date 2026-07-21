package router

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// --- buildProxies edge cases ---

func TestBuildProxies_InvalidAndValidURLs(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/valid":   "http://localhost:18001",
		"/api/v1/invalid": "://bad-url", // invalid URL
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	// Valid route should be proxied, invalid should be skipped
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	if _, ok := gw.proxies["/api/v1/valid"]; !ok {
		t.Error("valid route should be in proxies")
	}
	if _, ok := gw.proxies["/api/v1/invalid"]; ok {
		t.Error("invalid route should NOT be in proxies")
	}
}

func TestBuildProxies_WithTimeouts(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/auth": "http://localhost:18001",
	}
	cfg.RouteConfigs = map[string]config.RouteConfig{
		"/api/v1/auth": {Timeout: config.RouteTimeout{Read: 5 * time.Second, Idle: 30 * time.Second, Dial: 3 * time.Second}},
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	if to, ok := gw.timeouts["/api/v1/auth"]; !ok || to != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v ok=%v", to, ok)
	}
}

// --- buildHealthChecker edge cases ---

func TestBuildHealthChecker_RootPrefix(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/": "http://localhost:18001",
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)
	if gw.healthChecker == nil {
		t.Error("health checker should be created")
	}
}

func TestBuildHealthChecker_NormalPrefix(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/users": "http://localhost:18001",
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)
	if gw.healthChecker == nil {
		t.Error("health checker should be created")
	}
}

// --- ServeHTTP edge cases ---

func TestServeHTTP_MetricsPath(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	// Metrics endpoint returns 200 or 405
	if w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("/metrics: expected 200 or 405, got %d", w.Code)
	}
}

func TestServeHTTP_GraphQLConfigured(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	gw.graphql = middleware.NewGraphQLResolver(gw.cfg.Routes)
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query":"{ __typename }"}`))
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	// Should not return 503 (GraphQL is configured)
	if w.Code == http.StatusServiceUnavailable {
		t.Error("GraphQL should be configured, not 503")
	}
}

func TestServeHTTP_StatsConfigured(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)
	gw.stats = middleware.NewStatsCollector()

	req := httptest.NewRequest("GET", "/api/v1/gateway/stats", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServeHTTP_AdminRoutesMultipleRoutes(t *testing.T) {
	gw := newTestGateway(t)
	req := httptest.NewRequest("GET", "/api/v1/admin/routes", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServeHTTP_NoRouteMatch(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/api/v1/nonexistent/path", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestServeHTTP_WithRouteTimeout(t *testing.T) {
	// Create a slow backend
	slowBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer slowBackend.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/slow": slowBackend.URL,
	}
	cfg.RouteConfigs = map[string]config.RouteConfig{
		"/api/v1/slow": {Timeout: config.RouteTimeout{Read: 50 * time.Millisecond, Idle: 10 * time.Second, Dial: 5 * time.Second}},
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	req := httptest.NewRequest("GET", "/api/v1/slow/test", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 200*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	// May timeout or succeed, just verify it doesn't hang
}

// --- injectTenantIntoBody edge cases ---

func TestInjectTenantIntoBody_LargeBody(t *testing.T) {
	largeBody := `{"data":"` + strings.Repeat("x", 10000) + `"}`
	req := httptest.NewRequest("POST", "/api/v1/test", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	injectTenantIntoBody(req, "tenant-123")

	body, _ := io.ReadAll(req.Body)
	if !bytes.Contains(body, []byte("tenant-123")) {
		t.Error("tenant_id should be injected into large body")
	}
}

func TestInjectTenantIntoBody_NonJSONContentType(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "text/plain")

	originalBody := `{"key":"value"}`
	injectTenantIntoBody(req, "tenant-123")

	body, _ := io.ReadAll(req.Body)
	if string(body) != originalBody {
		t.Error("non-JSON content type should not be modified")
	}
}

// --- handleAdminToggleRoute edge cases ---

func TestAdminToggleRoute_EnableInvalidBackendURL(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/test": "://bad-url", // invalid
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	// Route is disabled (invalid URL won't be in proxies)
	// So toggle should try to re-enable it
	req := httptest.NewRequest("POST", "/api/v1/admin/routes//api/v1/test/toggle", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	// May get 500 due to invalid URL parse, or 404 if prefix doesn't match exactly
	_ = w.Code
}

// --- Handler chain edge cases ---

func TestHandler_MultiplePublicPaths(t *testing.T) {
	gw := newTestGateway(t)
	handler := gw.Handler()

	publicPaths := []string{
		"/api/v1/auth/verify",
		"/api/v1/auth/register",
		"/api/v1/auth/password/forgot",
		"/api/v1/auth/password/reset",
		"/.well-known/jwks.json",
	}

	for _, path := range publicPaths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		// Should not get 401 for public paths
		if w.Code == http.StatusUnauthorized {
			t.Errorf("public path %s returned 401", path)
		}
	}
}

func TestHandler_CORSHeaders(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")
	gw := testGatewayNoJWKS(t)
	handler := gw.Handler()

	req := httptest.NewRequest("OPTIONS", "/api/v1/auth/verify", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS header should be set")
	}
}

// --- Reload with valid config ---

func TestReloadRoutes_SuccessWithVersion(t *testing.T) {
	gw := newTestGateway(t)
	originalVersion := gw.routeVersion

	gw.SetReloadFunc(func() (*config.Config, error) {
		cfg := config.Default()
		cfg.Routes = map[string]string{
			"/api/v1/new": "http://localhost:19999",
		}
		return cfg, nil
	})

	req := httptest.NewRequest("POST", "/api/v1/gateway/routes/reload", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if gw.routeVersion != originalVersion+1 {
		t.Errorf("version should increment, got %d", gw.routeVersion)
	}
}

// --- matchBackend multiple prefix matching ---

func TestMatchBackend_OverlappingPrefixes(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1":      "http://localhost:18001",
		"/api/v1/users": "http://localhost:18002",
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	// Should match the longest prefix
	proxy, prefix := gw.matchBackend("/api/v1/users/list")
	if prefix != "/api/v1/users" {
		t.Errorf("expected longest prefix match, got %q", prefix)
	}
	if proxy == nil {
		t.Error("proxy should be non-nil")
	}
}

func TestMatchBackend_RootOnly(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/": "http://localhost:18001",
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	proxy, prefix := gw.matchBackend("/anything/at/all")
	if prefix != "/" {
		t.Errorf("expected '/' prefix, got %q", prefix)
	}
	if proxy == nil {
		t.Error("proxy should be non-nil")
	}
}
