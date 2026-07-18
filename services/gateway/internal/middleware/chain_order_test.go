package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMiddlewareChainOrder verifies middleware execution order by checking
// observable side effects in the response: security headers from
// SecurityHeaders, CORS headers from CORS, and request ID from RequestID.
func TestMiddlewareChainOrder(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Build chain in the same order as router.go Handler()
	// PanicRecovery > SecurityHeaders > CORS > RequestID > inner
	h := RequestID(inner)
	h = CORSWithConfig(CORSConfig{AllowedOrigins: []string{"*"}})(h)
	h = SecurityHeaders(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// SecurityHeaders runs first → sets headers on the response
	if xcto := rr.Header().Get("X-Content-Type-Options"); xcto != "nosniff" {
		t.Errorf("expected X-Content-Type-Options=nosniff, got %q", xcto)
	}
	if sts := rr.Header().Get("Strict-Transport-Security"); sts == "" {
		t.Error("expected Strict-Transport-Security header from SecurityHeaders")
	}
	// CORS runs → sets Access-Control-Allow-Origin
	if acao := rr.Header().Get("Access-Control-Allow-Origin"); acao == "" {
		t.Error("expected Access-Control-Allow-Origin from CORS middleware")
	}
	// RequestID runs → sets X-Request-ID
	if rid := rr.Header().Get("X-Request-ID"); rid == "" {
		t.Error("expected X-Request-ID from RequestID middleware")
	}
}

// TestMiddlewareChainOrder_PanicRecoveryWraps verifies a panic in the inner
// handler is caught by PanicRecovery and returns 500, not a crash.
func TestMiddlewareChainOrder_PanicRecoveryWraps(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic from inner handler")
	})

	logger := NewStructuredLogger("test-service")
	h := RequestID(inner)
	h = CORS(h)
	h = SecurityHeaders(h)
	h = PanicRecovery(logger)(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from panic recovery, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "error") {
		t.Errorf("expected error in response body, got: %s", rr.Body.String())
	}
}

// TestMiddlewareChainOrder_RateLimitBlocks verifies rate limiter blocks
// before the inner handler is reached.
func TestMiddlewareChainOrder_RateLimitBlocks(t *testing.T) {
	reached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	rl := NewTenantBucketLimiter(&BucketRateLimitConfig{
		DefaultMaxTokens:    1,
		DefaultRefillPerSec: 0,
	})

	logger := NewStructuredLogger("test-service")
	h := rl.Middleware(inner)
	h = RequestID(h)
	h = SecurityHeaders(h)
	h = PanicRecovery(logger)(h)

	// First request: passes
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rr1.Code)
	}

	// Second request: blocked
	reached = false
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rr2.Code)
	}
	if reached {
		t.Error("inner handler should NOT be reached when rate limited")
	}
}
