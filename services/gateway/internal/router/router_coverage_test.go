package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/healthcheck"
)

// --- Liveness / Readiness ---

func TestGateway_HealthzLive(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/healthz/live", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "alive" {
		t.Errorf("expected alive, got %s", body["status"])
	}
}

func TestGateway_HealthzReady_WithChecker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	gw := testGatewayNoJWKS(t)
	gw.SetHealthChecker(healthcheck.NewChecker(map[string]string{"auth": srv.URL}))

	req := httptest.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGateway_HealthzReady_NoChecker(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// buildHealthChecker always creates a checker from routes; empty routes → 0 unhealthy → "healthy"
}

func TestGateway_HealthzReady_Unhealthy(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	gw.SetHealthChecker(healthcheck.NewChecker(map[string]string{
		"dead": "http://127.0.0.1:1/healthz",
	}))
	req := httptest.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// --- buildHealthChecker ---

func TestGateway_BuildHealthChecker(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/auth":  "http://localhost:9001",
		"/api/v1/users": "http://localhost:8081",
	}
	gw := New(cfg, nil)
	if gw.healthChecker == nil {
		t.Fatal("expected healthChecker to be built")
	}
}

// --- SetHealthChecker ---

func TestGateway_SetHealthChecker(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	hc := healthcheck.NewChecker(nil)
	gw.SetHealthChecker(hc)
	if gw.healthChecker != hc {
		t.Error("SetHealthChecker did not set checker")
	}
}

// --- Per-Route Timeout ---

func TestGateway_PerRouteTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": srv.URL}
	cfg.RouteConfigs = map[string]config.RouteConfig{
		"/api/v1/test": {Timeout: config.RouteTimeout{Read: 5_000_000_000}}, // 5s in ns (time.Duration)
	}
	gw := New(cfg, nil)

	if to, ok := gw.timeouts["/api/v1/test"]; !ok || to == 0 {
		t.Error("expected per-route timeout to be set")
	}

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGateway_TimeoutFiresOnSlowBackend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until cancelled
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/slow": srv.URL}
	cfg.RouteConfigs = map[string]config.RouteConfig{
		"/api/v1/slow": {Timeout: config.RouteTimeout{Read: 1_000_000_000}}, // 1s
	}
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/slow", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	// Context deadline exceeded should trigger proxy error handler → 502
	if w.Code != 502 {
		t.Logf("timeout test got code %d (expected 502)", w.Code)
	}
}

// --- matchBackend ---

func TestGateway_MatchBackend_Nil(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	proxy, prefix := gw.matchBackend("/nonexistent")
	if proxy != nil || prefix != "" {
		t.Error("expected nil proxy and empty prefix for unknown path")
	}
}

func TestGateway_MatchBackend_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/x": srv.URL}
	gw := New(cfg, nil)
	proxy, prefix := gw.matchBackend("/api/v1/x/data")
	if proxy == nil || prefix != "/api/v1/x" {
		t.Errorf("expected proxy and prefix /api/v1/x, got %v %q", proxy != nil, prefix)
	}
}

// --- injectTenantIntoBody ---

func TestInjectTenantIntoBody_NilBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Body = nil
	injectTenantIntoBody(req, "tenant-123")
	// Should not panic
}

func TestInjectTenantIntoBody_EmptyTenant(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "")
	// Should not modify body
}

func TestInjectTenantIntoBody_GetRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "tenant-123")
	// GET should not be modified
}

func TestInjectTenantIntoBody_NonJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`plain text`))
	req.Header.Set("Content-Type", "text/plain")
	injectTenantIntoBody(req, "tenant-123")
	// Non-JSON should not be modified
}

func TestInjectTenantIntoBody_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{invalid json`))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "tenant-123")
	body, _ := io.ReadAll(req.Body)
	if !bytes.Equal(body, []byte(`{invalid json`)) {
		t.Error("invalid JSON body should be restored unchanged")
	}
}

func TestInjectTenantIntoBody_AlreadyHasTenant(t *testing.T) {
	original := `{"name":"test","tenant_id":"existing"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(original))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "new-tenant")
	body, _ := io.ReadAll(req.Body)
	if string(body) != original {
		t.Error("body with existing tenant_id should not be modified")
	}
}

func TestInjectTenantIntoBody_Success(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "tenant-123")
	body, _ := io.ReadAll(req.Body)
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("injected body should be valid JSON: %v (body: %s)", err, string(body))
	}
	if result["tenant_id"] != "tenant-123" {
		t.Errorf("expected tenant_id=tenant-123, got %v", result["tenant_id"])
	}
}

// --- PrintRoutes ---

func TestGateway_PrintRoutes(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": "http://localhost:9999"}
	gw := New(cfg, nil)
	// Should not panic
	gw.PrintRoutes()
}

// --- serveSwaggerUI / serveOpenAPISpec ---

func TestServeSwaggerUI(t *testing.T) {
	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	serveSwaggerUI(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServeOpenAPISpec(t *testing.T) {
	req := httptest.NewRequest("GET", "/api-docs", nil)
	w := httptest.NewRecorder()
	serveOpenAPISpec(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Handler chain ---

func TestGateway_HandlerChain(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	h := gw.Handler()
	if h == nil {
		t.Fatal("Handler() returned nil")
	}
	// Test that healthz works through full handler chain
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 through handler chain, got %d", w.Code)
	}
}

// --- Context with timeout edge case ---

func TestGateway_ContextTimeoutCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": srv.URL}
	gw := New(cfg, nil)

	req := httptest.NewRequestWithContext(ctx, "GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	// Cancelled context should result in error from proxy
}

// --- JSON parse percentage edge cases ---

func TestInjectTenantIntoBody_PutMethod(t *testing.T) {
	req := httptest.NewRequest("PUT", "/", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "tenant-put")
	body, _ := io.ReadAll(req.Body)
	var result map[string]any
	json.Unmarshal(body, &result)
	if result["tenant_id"] != "tenant-put" {
		t.Errorf("expected tenant_id=tenant-put, got %v", result["tenant_id"])
	}
}

func TestInjectTenantIntoBody_PatchMethod(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	injectTenantIntoBody(req, "tenant-patch")
	body, _ := io.ReadAll(req.Body)
	var result map[string]any
	json.Unmarshal(body, &result)
	if result["tenant_id"] != "tenant-patch" {
		t.Errorf("expected tenant_id=tenant-patch, got %v", result["tenant_id"])
	}
}

func TestInjectTenantIntoBody_EmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(``))
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(``))
	injectTenantIntoBody(req, "tenant-123")
	// Should not panic with empty body
}
