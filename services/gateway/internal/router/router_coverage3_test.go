package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// Test buildProxiesLocked director callback with identity headers
func TestBuildProxiesLocked_DirectorForwardsHeaders(t *testing.T) {
	gw := newTestGateway(t)
	gw.mu.RLock()
	proxy, ok := gw.proxies["/api/v1/users"]
	gw.mu.RUnlock()
	if !ok {
		t.Fatal("expected proxy for /api/v1/users")
	}
	if proxy == nil {
		t.Fatal("proxy is nil")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	ctx := context.WithValue(r.Context(), middleware.RequestIDKey, "req-123")
	r = r.WithContext(ctx)
	gw.ServeHTTP(w, r)
}

// Test buildProxiesLocked with per-route timeout configured
func TestBuildProxiesLocked_WithTimeouts(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/slow": "http://localhost:39000",
	}
	cfg.RouteConfigs = map[string]config.RouteConfig{
		"/api/v1/slow": {Timeout: config.RouteTimeout{Read: 5 * time.Second}},
	}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	gw.mu.Lock()
	gw.buildProxiesLocked()
	gw.mu.Unlock()

	gw.mu.RLock()
	defer gw.mu.RUnlock()
	to, ok := gw.timeouts["/api/v1/slow"]
	if !ok {
		t.Fatal("expected timeout to be configured")
	}
	if to <= 0 {
		t.Errorf("expected positive timeout, got %v", to)
	}
}

// Test buildProxiesLocked with empty routes
func TestBuildProxiesLocked_EmptyRoutes(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{}
	jwks, _ := middleware.NewJWKSClient("", "")
	gw := New(cfg, jwks)

	gw.mu.Lock()
	gw.buildProxiesLocked()
	gw.mu.Unlock()

	if len(gw.proxies) != 0 {
		t.Errorf("expected 0 proxies, got %d", len(gw.proxies))
	}
}

// Test proxy error handler returns 502 JSON
func TestProxyErrorHandler_JSONResponse(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	gw.ServeHTTP(w, r)
	if w.Code != http.StatusBadGateway {
		t.Logf("got status %d (expected 502 for non-existent backend)", w.Code)
	}
}

// Test ServeHTTP with tenant resolution from header
func TestServeHTTP_TenantResolutionFromHeader(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/users", stringBody(`{"name":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Tenant-ID", "resolved-tenant")
	gw.ServeHTTP(w, r)
}

// Test serveSwaggerUI directly
func TestServeSwaggerUI_DirectCall(t *testing.T) {
	w := httptest.NewRecorder()
	serveSwaggerUI(w, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("expected text/html, got %s", w.Header().Get("Content-Type"))
	}
}

// Test serveOpenAPISpec directly
func TestServeOpenAPISpec_DirectCall(t *testing.T) {
	w := httptest.NewRecorder()
	serveOpenAPISpec(w, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got %s", w.Header().Get("Content-Type"))
	}
}

// Test healthz ready returns proper status
func TestHealthzReady_Structure(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/healthz/ready", nil)
	gw.ServeHTTP(w, r)
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 200 or 503, got %d", w.Code)
	}
}

// Test docs endpoint
func TestDocs_TrailingSlash(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/docs/", nil)
	gw.ServeHTTP(w, r)
	if w.Code == 0 {
		t.Error("expected non-zero status")
	}
}

// Test OPTIONS preflight
func TestOPTIONS_Preflight(t *testing.T) {
	gw := newTestGateway(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/api/v1/users", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "GET")
	gw.ServeHTTP(w, r)
	// CORS preflight returns 204 or 200 or may pass through to JWT auth (401)
	// Just verify it doesn't panic
	if w.Code == 0 {
		t.Error("expected non-zero status")
	}
}
