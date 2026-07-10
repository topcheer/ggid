package router

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
	"github.com/ggid/ggid/services/gateway/internal/middleware"
)

func TestRouter_ProxyErrorBackend(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/test": "http://127.0.0.1:1", // unreachable
		},
	}
	gw := New(cfg, nil)
	req := httptest.NewRequest("GET", "/api/v1/test/data", nil)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Errorf("want 502, got %d", rr.Code)
	}
	body, _ := io.ReadAll(rr.Body)
	var resp map[string]string
	json.Unmarshal(body, &resp)
	if resp["error"] == "" {
		t.Error("Should have error message")
	}
}

func TestRouter_InvalidBackendURL(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/bad": "://invalid-url",
		},
	}
	gw := New(cfg, nil)
	// Should not panic
	req := httptest.NewRequest("GET", "/api/v1/bad/data", nil)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("invalid backend: want 404, got %d", rr.Code)
	}
}

func TestRouter_BuildHealthChecker(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:8081",
			"/api/v1/orgs":  "http://localhost:8071",
		},
	}
	gw := New(cfg, nil)
	gw.buildHealthChecker()
	if gw.healthChecker == nil {
		t.Error("healthChecker should be set")
	}
}

func TestRouter_ServeHTTPHealthz(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{},
	}
	gw := New(cfg, nil)
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("healthz: want 200, got %d", rr.Code)
	}
}

func TestRouter_ServeHTTPRouteNotFound(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:8081",
		},
	}
	gw := New(cfg, nil)
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("not found: want 404, got %d", rr.Code)
	}
}

func TestRouter_TenantIDInjectionInBody(t *testing.T) {
	var capturedBody []byte
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/test": backend.URL,
		},
	}
	gw := New(cfg, nil)

	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/test/create", io.NopCloser(io.Reader(nil)))
	req.Body = io.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, "tenant-123")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)

	var data map[string]any
	json.Unmarshal(capturedBody, &data)
	if data["tenant_id"] != "tenant-123" {
		t.Errorf("tenant_id not injected: got %v", data["tenant_id"])
	}
}

func TestRouter_RouteMatchBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/test": backend.URL,
		},
	}
	gw := New(cfg, nil)
	req := httptest.NewRequest("GET", "/api/v1/test/path", nil)
	rr := httptest.NewRecorder()
	gw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestRouter_HandleGetRoutes(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:8081",
		},
	}
	gw := New(cfg, nil)
	req := httptest.NewRequest("GET", "/admin/routes", nil)
	rr := httptest.NewRecorder()
	gw.handleGetRoutes(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestRouter_HandleReloadRoutes(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:8081",
		},
	}
	gw := New(cfg, nil)
	gw.SetReloadFunc(func() (*config.Config, error) {
		return cfg, nil
	})
	req := httptest.NewRequest("POST", "/admin/reload", nil)
	rr := httptest.NewRecorder()
	gw.handleReloadRoutes(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestRouter_MatchBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": backend.URL,
		},
	}
	gw := New(cfg, nil)

	proxy, prefix := gw.matchBackend("/api/v1/users/123")
	if proxy == nil {
		t.Error("Should match backend")
	}
	if prefix != "/api/v1/users" {
		t.Errorf("prefix: want '/api/v1/users', got '%s'", prefix)
	}

	proxy2, _ := gw.matchBackend("/unknown")
	if proxy2 != nil {
		t.Error("Should not match unknown route")
	}
}

func TestRouter_PrintRoutes(t *testing.T) {
	cfg := &config.Config{
		Routes: map[string]string{
			"/api/v1/users": "http://localhost:8081",
		},
	}
	gw := New(cfg, nil)
	gw.PrintRoutes() // Should not panic
}
