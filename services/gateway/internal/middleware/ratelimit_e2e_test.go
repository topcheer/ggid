package middleware

// Rate Limiter E2E Test
// Verifies: 100 requests → first N succeed → subsequent get 429.
// Date: 2026-07-25

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestRateLimiter_E2E_100Requests verifies that sending 100 requests with a
// limit of 10 results in 10x200 + 90x429.
func TestRateLimiter_E2E_100Requests(t *testing.T) {
	cfg := RateLimitConfig{
		APILimit:      10,
		LoginLimit:    5,
		RegisterLimit: 3,
		Window:        time.Minute,
	}
	rl := NewRateLimiter(cfg)

	var okCount, limited int
	var mu sync.Mutex

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rl.Middleware(next)

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.RemoteAddr = "10.0.0.100:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		mu.Lock()
		switch rr.Code {
		case http.StatusOK:
			okCount++
		case http.StatusTooManyRequests:
			limited++
		default:
			t.Errorf("unexpected status %d on request %d", rr.Code, i+1)
		}
		mu.Unlock()
	}

	if okCount != 10 {
		t.Errorf("expected 10 successful requests, got %d", okCount)
	}
	if limited != 90 {
		t.Errorf("expected 90 rate-limited requests, got %d", limited)
	}
}

// TestRateLimiter_E2E_429HasJSONBody verifies 429 response includes JSON error.
func TestRateLimiter_E2E_429HasJSONBody(t *testing.T) {
	cfg := RateLimitConfig{APILimit: 1, Window: time.Minute}
	rl := NewRateLimiter(cfg)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request: OK
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	req1.RemoteAddr = "10.0.0.200:1234"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != 200 {
		t.Fatalf("first request should succeed, got %d", rr1.Code)
	}

	// Second request: 429
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	req2.RemoteAddr = "10.0.0.200:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 429 {
		t.Fatalf("second request should be 429, got %d", rr2.Code)
	}

	// Verify JSON body
	body := rr2.Body.String()
	if !contains(body, "rate limit") {
		t.Errorf("429 body should contain 'rate limit', got: %s", body)
	}

	// Verify Retry-After header
	if rr2.Header().Get("Retry-After") == "" {
		t.Error("429 response should have Retry-After header")
	}

	// Verify Content-Type
	ct := rr2.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type should be application/json, got %s", ct)
	}
}

// TestRateLimiter_E2E_DifferentIPsNotShared verifies that different IPs have separate limits.
func TestRateLimiter_E2E_DifferentIPsNotShared(t *testing.T) {
	cfg := RateLimitConfig{APILimit: 5, Window: time.Minute}
	rl := NewRateLimiter(cfg)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// IP A sends 5 requests (all OK)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("IP A request %d should succeed, got %d", i+1, rr.Code)
		}
	}

	// IP B sends 5 requests (should also succeed — separate bucket)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("IP B request %d should succeed (separate bucket), got %d", i+1, rr.Code)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOfStr(s, sub) >= 0)
}

func indexOfStr(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
