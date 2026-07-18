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
	SetTenantCORS("preflight-tenant", TenantCORSConfig{AllowedOrigins: []string{"https://example.com"}, AllowedMethods: []string{"GET"}})
	req := httptest.NewRequest("OPTIONS", "/api/v1/users", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("X-Tenant-ID", "preflight-tenant")
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

func TestTenantCORSMiddleware_StrictDefault(t *testing.T) {
	// No explicit config: origin should NOT be allowed (strict default).
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Origin", "https://anything.com")
	req.Header.Set("X-Tenant-ID", "strict-default-tenant")
	w := httptest.NewRecorder()
	TenantCORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "" { t.Error("strict default should not allow origin") }
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
	if len(cfg.AllowedOrigins) != 0 { t.Errorf("default should be empty (strict), got %v", cfg.AllowedOrigins) }
}
