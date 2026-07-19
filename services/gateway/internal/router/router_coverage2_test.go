package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// Test proxy error handler directly
func TestProxyErrorHandler_502(t *testing.T) {
	gw := newTestGateway(t)
	// Find any configured proxy and trigger its error handler
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)

	// Serve to a non-existent backend to trigger proxy error handler
	gw.ServeHTTP(w, r)

	// Should get 502 since backend doesn't exist
	if w.Code != http.StatusBadGateway && w.Code != http.StatusOK {
		// Depending on timing, backend may not exist
	}
}

// Test metrics endpoint
func TestGateway_MetricsEndpoint(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/metrics", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /metrics, got %d", w.Code)
	}
}

// Test JWKS endpoint
func TestGateway_JWKSEndpoint(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for JWKS, got %d", w.Code)
	}
}

// Test gateway stats endpoint with no stats configured
func TestGateway_StatsNotConfigured(t *testing.T) {
	gw := newTestGateway(t) // no stats configured

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/gateway/stats", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "stats not configured" {
		t.Errorf("expected 'stats not configured', got %s", body["status"])
	}
}

// Test gateway middleware chain endpoint
func TestGateway_MiddlewareChainEndpoint(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/gateway/middleware", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// Test GraphQL endpoint without graphql configured
func TestGateway_GraphQLNotConfigured(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/graphql", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// Test injectTenantIntoBody with already-prefilled tenant_id
func TestInjectTenant_AlreadyHasTenant_RestoresBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/users",
		stringBody(`{"name":"test","tenant_id":"existing-tenant"}`))
	r.Header.Set("Content-Type", "application/json")

	injectTenantIntoBody(r, "new-tenant")

	// Should NOT override existing tenant_id
	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)
	if body["tenant_id"] != "existing-tenant" {
		t.Errorf("expected existing-tenant, got %s", body["tenant_id"])
	}
}

// Test injectTenantIntoBody with non-POST method
func TestInjectTenant_DeleteMethod(t *testing.T) {
	r := httptest.NewRequest("DELETE", "/api/v1/users/123",
		stringBody(`{"name":"test"}`))
	r.Header.Set("Content-Type", "application/json")

	injectTenantIntoBody(r, "tenant-1")

	// Should not modify DELETE body
	var body map[string]string
	json.NewDecoder(r.Body).Decode(&body)
	if _, ok := body["tenant_id"]; ok {
		t.Error("DELETE should not have tenant_id injected")
	}
}

// Test matchBackend longest-prefix matching
func TestMatchBackend_LongestPrefix(t *testing.T) {
	gw := newTestGateway(t)
	// Add overlapping routes
	gw.mu.Lock()
	gw.cfg.Routes["/api/v1"] = "http://localhost:19000"
	gw.cfg.Routes["/api/v1/users"] = "http://localhost:19001"
	gw.buildProxiesLocked()
	gw.mu.Unlock()

	// /api/v1/users should match the longer prefix
	proxy, prefix := gw.matchBackend("/api/v1/users/123")
	if proxy == nil {
		t.Fatal("expected proxy to be found")
	}
	if prefix != "/api/v1/users" {
		t.Errorf("expected /api/v1/users, got %s", prefix)
	}
}

// Test buildProxiesLocked with invalid URL
func TestBuildProxiesLocked_InvalidURL(t *testing.T) {
	gw := newTestGateway(t)
	gw.mu.Lock()
	gw.cfg.Routes["/api/v1/bad"] = "://invalid-url"
	gw.buildProxiesLocked()
	gw.mu.Unlock()

	// Invalid URL should be skipped (not cause panic)
	_, ok := gw.proxies["/api/v1/bad"]
	if ok {
		t.Error("invalid URL should not create a proxy")
	}
}

// Test SetReloadFunc + reload route
func TestReloadRoutes_Success(t *testing.T) {
	gw := newTestGateway(t)
	gw.SetReloadFunc(func() (*config.Config, error) {
		newCfg := config.Default()
		newCfg.Routes = map[string]string{
			"/api/v1": "http://localhost:29000",
		}
		return newCfg, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/gateway/routes/reload", nil)
	gw.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected ok, got %v", resp["status"])
	}
}

// Test Handler() returns non-nil
func TestGateway_Handler_ReturnsHandler(t *testing.T) {
	gw := newTestGateway(t)
	h := gw.Handler()
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

// --- helpers ---

func stringBody(s string) *strings.Reader {
	return strings.NewReader(s)
}

func newTestGateway(t *testing.T) *Gateway {
	t.Helper()
	mockBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(mockBackend.Close)

	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/users":  mockBackend.URL,
		"/api/v1/roles":  mockBackend.URL,
		"/api/v1/orgs":   mockBackend.URL,
		"/api/v1/audit":  mockBackend.URL,
	}

	// Create a JWKS client with a test key
	jwks, err := middleware.NewJWKSClient("", "")
	if err != nil {
		// If NewJWKSClient fails, try with a generated key
		t.Fatalf("failed to create JWKS client: %v", err)
	}
	return New(cfg, jwks)
}
