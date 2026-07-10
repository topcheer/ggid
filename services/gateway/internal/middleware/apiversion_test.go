package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users", "1"},
		{"/api/v2/users", "2"},
		{"/api/v10/data", "10"},
		{"/api/users", ""},
		{"/api/vx/users", ""},
		{"/users", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := VersionFromPath(tt.path)
		if got != tt.want {
			t.Errorf("VersionFromPath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestExtractAPIVersion_URLPath(t *testing.T) {
	cfg := DefaultAPIVersionConfig()
	tests := []struct {
		path    string
		header  string
		query   string
		want    string
	}{
		{"/api/v2/users", "", "", "2"},
		{"/api/v1/users", "3", "", "1"}, // URL takes precedence
		{"/users", "3", "", "3"},      // header
		{"/users", "", "4", "4"},      // query
		{"/users", "", "", "1"},       // default
	}
	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		if tt.header != "" {
			req.Header.Set("Api-Version", tt.header)
		}
		if tt.query != "" {
			q := req.URL.Query()
			q.Set("api_version", tt.query)
			req.URL.RawQuery = q.Encode()
		}
		got := ExtractAPIVersion(req, cfg)
		if got != tt.want {
			t.Errorf("path=%q header=%q query=%q: got %q, want %q", tt.path, tt.header, tt.query, got, tt.want)
		}
	}
}

func TestExtractAPIVersion_CustomHeader(t *testing.T) {
	cfg := APIVersionConfig{
		DefaultVersion: "1",
		HeaderName:     "X-API-Version",
	}
	req := httptest.NewRequest("GET", "/users", nil)
	req.Header.Set("X-API-Version", "5")
	got := ExtractAPIVersion(req, cfg)
	if got != "5" {
		t.Errorf("expected 5 from custom header, got %s", got)
	}
}

func TestAPIVersioning_DifferentBackends(t *testing.T) {
	v1Called := false
	v2Called := false

	handlers := map[string]http.Handler{
		"1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v1Called = true
			w.WriteHeader(200)
		}),
		"2": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v2Called = true
			w.WriteHeader(200)
		}),
	}

	cfg := DefaultAPIVersionConfig()
	mw := APIVersioning(cfg, func(version string) http.Handler {
		return handlers[version]
	})

	// v1 request
	req1 := httptest.NewRequest("GET", "/api/v1/users", nil)
	w1 := httptest.NewRecorder()
	mw.ServeHTTP(w1, req1)
	if !v1Called {
		t.Error("v1 handler should be called")
	}

	// v2 request
	req2 := httptest.NewRequest("GET", "/api/v2/users", nil)
	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, req2)
	if !v2Called {
		t.Error("v2 handler should be called")
	}
}

func TestAPIVersioning_HeaderVersion(t *testing.T) {
	v3Called := false
	handlers := map[string]http.Handler{
		"1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"3": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v3Called = true
		}),
	}

	cfg := DefaultAPIVersionConfig()
	mw := APIVersioning(cfg, func(version string) http.Handler {
		return handlers[version]
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Api-Version", "3")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !v3Called {
		t.Error("v3 handler should be called via header")
	}
}

func TestAPIVersioning_DefaultFallback(t *testing.T) {
	defaultCalled := false
	handlers := map[string]http.Handler{
		"1": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defaultCalled = true
		}),
	}

	cfg := DefaultAPIVersionConfig()
	mw := APIVersioning(cfg, func(version string) http.Handler {
		return handlers[version]
	})

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if !defaultCalled {
		t.Error("default handler should be called")
	}
}

func TestAPIVersioning_NilHandler(t *testing.T) {
	cfg := DefaultAPIVersionConfig()
	mw := APIVersioning(cfg, func(version string) http.Handler {
		return nil
	})

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for nil handler, got %d", w.Code)
	}
}

func TestAPIVersioning_StripPrefix(t *testing.T) {
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
	})

	cfg := APIVersionConfig{
		DefaultVersion: "1",
		StripPrefix:    true,
	}
	mw := APIVersioning(cfg, func(version string) http.Handler {
		return handler
	})

	req := httptest.NewRequest("GET", "/api/v2/users/123", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
	if receivedPath != "/api/users/123" {
		t.Errorf("expected stripped path /api/users/123, got %s", receivedPath)
	}
}
