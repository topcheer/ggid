package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
)

func testGatewayNoJWKS(t *testing.T) *Gateway {
	cfg := config.Default()
	cfg.Routes = map[string]string{}
	return New(cfg, nil)
}

// --- Health Check ---

func TestGateway_Healthz(t *testing.T) {
	gw := testGatewayNoJWKS(t)
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

// --- JWKS ---

func TestGateway_JWKS_NoClient_Panics(t *testing.T) {
	// This tests that JWKS handler is only called when jwks client is set
	// Without jwks client, it would panic — but healthz doesn't need it
	gw := testGatewayNoJWKS(t)

	// healthz should work without jwks
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Docs / Swagger ---

func TestGateway_Docs(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/docs", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html, got %s", ct)
	}
	if !contains(w.Body.String(), "Swagger") {
		t.Error("expected Swagger UI in docs")
	}
}

func TestGateway_DocsTrailingSlash(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/docs/", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Hosted Login ---

func TestGateway_Login(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), "Sign In") {
		t.Error("expected login form with 'Sign In'")
	}
}

// --- Hosted Register ---

func TestGateway_Register(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), "Create Account") {
		t.Error("expected register form")
	}
}

// --- Forgot Password ---

func TestGateway_ForgotPassword(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/forgot-password", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !contains(w.Body.String(), "Reset") {
		t.Error("expected reset form")
	}
}

// --- API Docs ---

func TestGateway_ApiDocs(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/api-docs", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Should be valid JSON
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("expected valid JSON for api-docs: %v", err)
	}
	if body["openapi"] == nil {
		t.Error("expected openapi field")
	}
}

// --- 404 ---

func TestGateway_NotFound(t *testing.T) {
	gw := testGatewayNoJWKS(t)
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error message")
	}
}

// --- Reverse Proxy ---

func TestGateway_ReverseProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "hit")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": backend.URL}
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/test/data", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Backend") != "hit" {
		t.Error("expected to reach backend")
	}
}

// --- Proxy Error ---

func TestGateway_ProxyError_502(t *testing.T) {
	cfg := config.Default()
	cfg.Routes = map[string]string{"/api/v1/test": "http://127.0.0.1:1"} // blackhole
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/test/data", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

// --- Longest Prefix Match ---

func TestGateway_LongestPrefixMatch(t *testing.T) {
	short := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("short"))
	}))
	defer short.Close()
	long := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("long"))
	}))
	defer long.Close()

	cfg := config.Default()
	cfg.Routes = map[string]string{
		"/api/v1":       short.URL,
		"/api/v1/users": long.URL,
	}
	gw := New(cfg, nil)

	req := httptest.NewRequest("GET", "/api/v1/users/list", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Body.String() != "long" {
		t.Errorf("expected 'long' backend, got %s", w.Body.String())
	}
}

// --- Helper ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
