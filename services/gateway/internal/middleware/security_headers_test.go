package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()
	SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).ServeHTTP(w, req)
	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-XSS-Protection":      "1; mode=block",
	}
	for hdr, want := range checks {
		if w.Header().Get(hdr) != want { t.Errorf("%s: got %q want %q", hdr, w.Header().Get(hdr), want) }
	}
	if w.Header().Get("Strict-Transport-Security") == "" { t.Error("missing HSTS") }
	if w.Header().Get("Content-Security-Policy") == "" { t.Error("missing CSP") }
}

func TestTenantCORSMiddleware_Preflight(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/v1/users", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	called := false
	TenantCORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)
	if called { t.Error("should not call next on preflight") }
	if w.Code != http.StatusNoContent { t.Errorf("expected 204, got %d", w.Code) }
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" { t.Error("should allow origin") }
}

func TestTenantCORSMiddleware_RejectedOrigin(t *testing.T) {
	SetTenantCORS("t1", TenantCORSConfig{AllowedOrigins: []string{"https://allowed.com"}, AllowedMethods: []string{"GET"}})
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()
	TenantCORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "" { t.Error("should not allow rejected origin") }
}

func TestTenantCORSMiddleware_Wildcard(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Origin", "https://anything.com")
	w := httptest.NewRecorder()
	TenantCORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") == "" { t.Error("wildcard should allow") }
}

func TestSetSecureCookie(t *testing.T) {
	w := httptest.NewRecorder()
	SetSecureCookie(w, "session", "abc", "/", 3600)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 { t.Fatal("expected 1 cookie") }
	c := cookies[0]
	if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Error("cookie not hardened")
	}
}

func TestGetTenantCORS_Default(t *testing.T) {
	cfg := GetTenantCORS("unknown")
	if cfg.AllowedOrigins[0] != "*" { t.Error("default should be wildcard") }
}
