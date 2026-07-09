package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
)

func TestGateway_ServeHTTP_HealthCheck(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{} // no routes needed for health check
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestGateway_ServeHTTP_NoRoute(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{}
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/nonexistent/path", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGateway_MatchBackend_EmptyRoutes(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{}
	gw := New(cfg, nil)

	backend := gw.matchBackend("/any/path")
	if backend != nil {
		t.Error("expected nil when no routes configured")
	}
}

func TestGateway_MatchBackend_LongestPrefix(t *testing.T) {
	// Start two mock backends
	shortBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"backend":"short"}`))
	}))
	defer shortBackend.Close()
	longBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"backend":"long"}`))
	}))
	defer longBackend.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1":       shortBackend.URL,
		"/api/v1/users": longBackend.URL,
	}
	gw := New(cfg, nil)

	// Path /api/v1/users/list should match the longer prefix
	backend := gw.matchBackend("/api/v1/users/list")
	if backend == nil {
		t.Fatal("expected non-nil backend for /api/v1/users/list")
	}

	// Verify it's the longer-prefix backend by making a request
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/users/list", nil)
	backend.ServeHTTP(rec, req)

	// The response should contain "long" from the longBackend
	var body map[string]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body["backend"] != "long" {
		t.Errorf("expected long backend, got %s", body["backend"])
	}
}

func TestGateway_MatchBackend_ShortPrefix(t *testing.T) {
	shortBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer shortBackend.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1": shortBackend.URL,
	}
	gw := New(cfg, nil)

	backend := gw.matchBackend("/api/v1/anything")
	if backend == nil {
		t.Fatal("expected non-nil backend")
	}
}

func TestNew_CreatesGatewayWithProxies(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1/test": "http://127.0.0.1:9999",
	}
	gw := New(cfg, nil)
	if gw == nil {
		t.Fatal("expected non-nil gateway")
	}
	if len(gw.proxies) == 0 {
		t.Error("expected proxies to be built")
	}
}
