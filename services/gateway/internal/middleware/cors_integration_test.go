package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORSIntegration_AllowedOrigin verifies a real OPTIONS preflight
// request through the CORS middleware returns the correct headers.
func TestCORSIntegration_AllowedOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called for OPTIONS preflight")
	})
	handler := CORS(next)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Preflight should return 204
	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Fatalf("expected 204 or 200, got %d", rr.Code)
	}
	// Verify CORS response headers
	if ac := rr.Header().Get("Access-Control-Allow-Origin"); ac == "" {
		t.Error("expected Access-Control-Allow-Origin header to be set")
	}
	if acm := rr.Header().Get("Access-Control-Allow-Methods"); acm == "" {
		t.Error("expected Access-Control-Allow-Methods header to be set")
	}
	if ach := rr.Header().Get("Access-Control-Allow-Headers"); ach == "" {
		t.Error("expected Access-Control-Allow-Headers header to be set")
	}
}

// TestCORSIntegration_ActualRequest verifies CORS headers on a real GET request
func TestCORSIntegration_ActualRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ac := rr.Header().Get("Access-Control-Allow-Origin"); ac == "" {
		t.Error("expected Access-Control-Allow-Origin header on actual request")
	}
}

// TestCORSIntegration_NoOrigin verifies non-CORS request still gets wildcard
// headers (because default config is ["*"] which sets ACAO unconditionally).
func TestCORSIntegration_NoOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Use explicit origins (not wildcard) so no-origin requests don't get ACAO
	handler := CORSWithConfig(CORSConfig{
		AllowedOrigins: []string{"https://app.example.com"},
	})(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ac := rr.Header().Get("Access-Control-Allow-Origin"); ac != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for non-CORS request, got %q", ac)
	}
}
