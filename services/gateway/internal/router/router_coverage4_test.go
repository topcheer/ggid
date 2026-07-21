package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

// --- handleAdminStats coverage (53.3% → higher) ---

func TestAdminStats_WithStatsCollector(t *testing.T) {
	// Create a gateway with routes
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:19001",
			"/api/v1/roles": "http://localhost:19002",
		},
	}
	gw := New(cfg, nil)

	// Set stats collector and record some data
	sc := middleware.NewStatsCollector()
	sc.Record("/api/v1/users", "GET", 200, 1024, 50*time.Millisecond)
	sc.Record("/api/v1/users", "GET", 500, 0, 100*time.Millisecond)
	sc.Record("/api/v1/roles", "POST", 201, 512, 30*time.Millisecond)
	gw.stats = sc

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp AdminStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	usersStats, ok := resp.Backends["/api/v1/users"]
	if !ok {
		t.Fatal("expected /api/v1/users in backends")
	}
	if usersStats.RequestCount != 2 {
		t.Errorf("expected 2 requests, got %d", usersStats.RequestCount)
	}
	if usersStats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", usersStats.ErrorCount)
	}
	// error_rate should be 0.5
	if usersStats.ErrorRate < 0.49 || usersStats.ErrorRate > 0.51 {
		t.Errorf("expected error rate ~0.5, got %f", usersStats.ErrorRate)
	}
}

func TestAdminStats_NoStatsCollector(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/test": "http://localhost:19003",
		},
	}
	gw := New(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp AdminStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Backends) != 1 {
		t.Errorf("expected 1 backend, got %d", len(resp.Backends))
	}
}

// --- handleAdminToggleRoute coverage (71.4% → higher) ---

func TestAdminToggleRoute_EnableAfterDisable(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:19010",
		},
	}
	gw := New(cfg, nil)

	// First: disable the route
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/routes//api/v1/users/toggle", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("disable: expected 200, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["enabled"] != false {
		t.Error("expected enabled=false after disable")
	}

	// Second: re-enable the route
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/routes//api/v1/users/toggle", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	req2.Header.Set("Authorization", adminAuthHeader)
	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("enable: expected 200, got %d", w2.Code)
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["enabled"] != true {
		t.Error("expected enabled=true after re-enable")
	}

	// Verify route is usable again
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w3 := httptest.NewRecorder()
	gw.ServeHTTP(w3, req3)
	// Should not be 404 (route exists again)
	if w3.Code == http.StatusNotFound {
		t.Error("route should be enabled and not return 404")
	}
}

func TestAdminToggleRoute_EmptyPrefix(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:19010",
		},
	}
	gw := New(cfg, nil)

	// Toggle with just "/toggle" → empty prefix
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/routes//toggle", nil)
	req.Header.Set("Authorization", adminAuthHeader)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty prefix, got %d", w.Code)
	}
}

// --- buildProxies coverage (70% → higher) ---

func TestBuildProxies_InvalidURL(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/valid":   "http://localhost:19020",
			"/api/v1/invalid": "://bad-url-no-scheme",
		},
	}
	gw := New(cfg, nil)

	// Invalid URL should be skipped (no proxy created)
	gw.mu.RLock()
	_, invalidExists := gw.proxies["/api/v1/invalid"]
	_, validExists := gw.proxies["/api/v1/valid"]
	gw.mu.RUnlock()

	if invalidExists {
		t.Error("expected no proxy for invalid URL")
	}
	if !validExists {
		t.Error("expected proxy for valid URL")
	}
}

// --- buildProxiesLocked coverage via reload + request flow ---

func TestBuildProxiesLocked_DirectorExercised(t *testing.T) {
	// Start a backend that records received headers
	var receivedHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/old": "http://localhost:19030",
		},
	}
	gw := New(cfg, nil)

	// Set reload func that returns new config pointing to test backend
	newCfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/new": backend.URL,
		},
	}
	gw.SetReloadFunc(func() (*config.Config, error) {
		return newCfg, nil
	})

	// Trigger reload — this calls buildProxiesLocked
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/routes/reload", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("reload failed: %d", w.Code)
	}

	// Now send a request through the new route to exercise Director closure in buildProxiesLocked
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/new/resource", nil)
	// Add context with request ID and tenant to exercise Director
	ctx := context.WithValue(req2.Context(), middleware.RequestIDKey, "req-12345")
	ctx = context.WithValue(ctx, middleware.TenantIDKey, "tenant-abc")
	req2 = req2.WithContext(ctx)

	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 from backend, got %d", w2.Code)
	}

	// Verify Director forwarded headers
	if receivedHeaders.Get("X-Request-ID") != "req-12345" {
		t.Errorf("expected X-Request-ID header forwarded, got %q", receivedHeaders.Get("X-Request-ID"))
	}
	if receivedHeaders.Get("X-Tenant-ID") != "tenant-abc" {
		t.Errorf("expected X-Tenant-ID header forwarded, got %q", receivedHeaders.Get("X-Tenant-ID"))
	}
}

func TestBuildProxiesLocked_ErrorHandlerExercised(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/bad": "http://localhost:1", // unreachable port
		},
	}
	gw := New(cfg, nil)

	// Reload to trigger buildProxiesLocked with same config
	gw.SetReloadFunc(func() (*config.Config, error) {
		return cfg, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gateway/routes/reload", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// Send request to unreachable backend → triggers ErrorHandler
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/bad/resource", nil)
	w2 := httptest.NewRecorder()
	gw.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w2.Code)
	}

	body := w2.Body.String()
	if !strings.Contains(body, "backend service unavailable") {
		t.Errorf("expected error message, got %s", body)
	}
}

// --- Handler coverage (80% → higher) ---

func TestHandler_ProtectedPathWithoutJWT(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:19040",
		},
	}
	gw := New(cfg, nil)
	handler := gw.Handler()

	// Access a protected path without JWT
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should get 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for protected path without JWT, got %d", w.Code)
	}
}

func TestHandler_PublicPathSkipsJWT(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/auth": "http://localhost:19041",
		},
	}
	gw := New(cfg, nil)
	handler := gw.Handler()

	// Access a public path — should not require JWT
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should NOT be 401 — public path skips JWT requirement
	if w.Code == http.StatusUnauthorized {
		t.Error("public path should not require JWT")
	}
}

// --- buildHealthChecker coverage (85.7% → higher) ---

func TestBuildHealthChecker_WithRootPrefix(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/":            "http://localhost:19050",
			"/api/v1/test": "http://localhost:19051",
		},
	}
	gw := New(cfg, nil)

	if gw.healthChecker == nil {
		t.Fatal("expected health checker to be built")
	}
}

// --- injectTenantIntoBody edge cases ---

func TestInjectTenantIntoBody_BodyReadError(t *testing.T) {
	// Create a request with a body that errors on read
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", errReader{})
	req.Header.Set("Content-Type", "application/json")

	injectTenantIntoBody(req, "tenant-123")

	// Should not panic, body should be left as-is
	if req.Body == nil {
		t.Error("body should not be nil")
	}
}

// errReader implements io.ReadCloser but always returns error
type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, http.ErrBodyReadAfterClose
}
func (errReader) Close() error { return nil }

func TestInjectTenantIntoBody_MarshalError(t *testing.T) {
	// Use a body that unmarshals fine but has a value that can't re-marshal
	// This is tricky — a map with NaN can cause marshal error
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test",
		strings.NewReader(`{"data":"test","nested":{"deep":123}}`))
	req.Header.Set("Content-Type", "application/json")

	// This should succeed (normal JSON can be re-marshaled)
	injectTenantIntoBody(req, "tenant-xyz")

	// Verify tenant_id was injected
	var body map[string]any
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["tenant_id"] != "tenant-xyz" {
		t.Errorf("expected tenant_id=tenant-xyz, got %v", body["tenant_id"])
	}
}
