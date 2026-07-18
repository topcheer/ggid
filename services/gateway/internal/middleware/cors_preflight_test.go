package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_PreflightOptions(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowCredentials: true,
	}

	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should NOT be called for OPTIONS preflight")
	}))

	req := httptest.NewRequest("OPTIONS", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 NoContent, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected Access-Control-Allow-Credentials: true")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}
	if w.Header().Get("Access-Control-Expose-Headers") == "" {
		t.Error("expected Access-Control-Expose-Headers header")
	}
	if w.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("Access-Control-Max-Age = %q, want '3600'", w.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORS_WildcardOrigin(t *testing.T) {
	handler := CORSWithConfig(CORSConfig{AllowedOrigins: []string{"*"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://anywhere.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected wildcard origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_SpecificOriginAllowed(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://app1.com", "https://app2.com"},
	}

	for _, origin := range []string{"https://app1.com", "https://app2.com"} {
		handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))

		req := httptest.NewRequest("GET", "/api/v1/data", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Errorf("origin %q: expected Allow-Origin to match, got %q", origin, w.Header().Get("Access-Control-Allow-Origin"))
		}
		if w.Header().Get("Vary") != "Origin" {
			t.Errorf("expected Vary: Origin header")
		}
	}
}

func TestCORS_SpecificOriginNotAllowed(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://allowed.com"},
	}

	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") == "https://evil.com" {
		t.Error("evil.com should NOT be in Allow-Origin")
	}
}

func TestCORS_CredentialsWithWildcard(t *testing.T) {
	// When AllowCredentials=true with wildcard origins, credentials header should be set
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}

	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected Access-Control-Allow-Credentials: true")
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected * origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	handler := CORSWithConfig(CORSConfig{AllowedOrigins: []string{"*"}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	// No Origin header
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// With wildcard config, should still set *
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected * for missing Origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_OptionsRequestReturnsNoContent(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest("OPTIONS", "/api/v1/data", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestCORS_GetRequestPassesThrough(t *testing.T) {
	called := false
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("GET request should pass through to handler")
	}
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	// Empty allowed origins = strict default, no CORS headers should be set
	cfg := CORSConfig{
		AllowedOrigins: []string{},
	}

	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("empty origins should not set ACAO (strict default), got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
