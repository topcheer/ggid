package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// === RateLimiter coverage ===

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.LoginLimit != 5 {
		t.Errorf("LoginLimit: want 5, got %d", cfg.LoginLimit)
	}
	if cfg.RegisterLimit != 3 {
		t.Errorf("RegisterLimit: want 3, got %d", cfg.RegisterLimit)
	}
	if cfg.APILimit != 100 {
		t.Errorf("APILimit: want 100, got %d", cfg.APILimit)
	}
	if cfg.Window != time.Minute {
		t.Errorf("Window: want 1m, got %v", cfg.Window)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	cfg := RateLimitConfig{APILimit: 100, Window: time.Minute}
	rl := NewRateLimiter(cfg)

	var wg sync.WaitGroup
	var allowed int64
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/api/v1/data", nil)
			req.RemoteAddr = "10.0.0.1:1234"
			rr := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			})
			rl.Middleware(next).ServeHTTP(rr, req)
			if rr.Code == 200 {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()
	if allowed == 0 {
		t.Error("Some requests should be allowed")
	}
}

func TestRateLimiter_RefillTiming(t *testing.T) {
	cfg := RateLimitConfig{APILimit: 2, Window: 50 * time.Millisecond}
	rl := NewRateLimiter(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := rl.Middleware(next)

	// Use 2 tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/v1/data", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
	// Third should be limited
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 429 {
		t.Errorf("Should be rate limited: got %d", rr.Code)
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Errorf("Should be allowed after window: got %d", rr2.Code)
	}
}


func TestRateLimiter_LoginEndpoint(t *testing.T) {
	cfg := RateLimitConfig{LoginLimit: 2, Window: time.Minute}
	rl := NewRateLimiter(cfg)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := rl.Middleware(next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("attempt %d: want 200, got %d", i, rr.Code)
		}
	}
	// Third should be limited
	req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 429 {
		t.Errorf("login limit exceeded: want 429, got %d", rr.Code)
	}
}

func TestRateLimiter_RegisterEndpoint(t *testing.T) {
	cfg := RateLimitConfig{RegisterLimit: 1, Window: time.Minute}
	rl := NewRateLimiter(cfg)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := rl.Middleware(next)

	req := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
	req.RemoteAddr = "10.0.0.6:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("first register: want 200, got %d", rr.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
	req2.RemoteAddr = "10.0.0.6:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 429 {
		t.Errorf("register limit: want 429, got %d", rr2.Code)
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	cfg := RateLimitConfig{APILimit: 1, Window: time.Minute}
	rl := NewRateLimiter(cfg)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := rl.Middleware(next)

	// IP1
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("IP1: want 200, got %d", rr.Code)
	}
	// IP2 should also be allowed (separate bucket)
	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Errorf("IP2: want 200 (separate bucket), got %d", rr2.Code)
	}
}

// === CORS coverage ===

func TestCORSPreflight_OptionsHandling(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("OPTIONS should not call next")
	}))

	req := httptest.NewRequest("OPTIONS", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("OPTIONS: want 204, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("Missing ACAO header")
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Missing ACAM header")
	}
}

func TestCORSPreflight_OriginValidation(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Allowed origin
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("Allowed origin should get ACAO")
	}

	// Disallowed origin
	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	req2.Header.Set("Origin", "https://evil.com")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Disallowed origin should not get ACAO")
	}
}

func TestCORSPreflight_CredentialsHeaders(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Missing ACA-Credentials")
	}
}

func TestCORSPreflight_WildcardOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Origin", "https://anything.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Wildcard should allow any origin")
	}
}

func TestCORSPreflight_NoOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
	}
	handler := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	// No Origin header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("No origin: want 200, got %d", rr.Code)
	}
}

// === Request logging coverage ===

func TestRequestLogging_LatencyRecorded(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if cl.Entries[0].Latency == "" {
		t.Error("Latency should be recorded")
	}
}

func TestRequestLogging_MethodRecorded(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		req := httptest.NewRequest(method, "/api/v1/data", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
	if len(cl.Entries) != 5 {
		t.Errorf("want 5 entries, got %d", len(cl.Entries))
	}
	for i, e := range cl.Entries {
		expected := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}[i]
		if e.Method != expected {
			t.Errorf("entry %d: method want %s, got %s", i, expected, e.Method)
		}
	}
}

func TestRequestLogging_IPRecorded(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if cl.Entries[0].IP != "203.0.113.50" {
		t.Errorf("IP: want '203.0.113.50', got '%s'", cl.Entries[0].IP)
	}
}

func TestRateLimiter_TokenEndpoint(t *testing.T) {
	cfg := RateLimitConfig{TokenLimit: 3, Window: time.Minute}
	rl := NewRateLimiter(cfg)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	handler := rl.Middleware(next)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/oauth/token", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Errorf("attempt %d: want 200, got %d", i, rr.Code)
		}
	}
	// 4th should be rate limited (brute-force protection)
	req := httptest.NewRequest("POST", "/oauth/token", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 429 {
		t.Errorf("token endpoint limit exceeded: want 429, got %d", rr.Code)
	}
}

func TestDefaultRateLimitConfig_TokenLimit(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.TokenLimit != 10 {
		t.Errorf("TokenLimit: want 10, got %d", cfg.TokenLimit)
	}
}
