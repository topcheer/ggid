package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- RateLimiter Tests ---

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 5, RegisterLimit: 3, APILimit: 100, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("attempt %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 3, RegisterLimit: 3, APILimit: 100, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 429 {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
	if w.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("expected X-RateLimit-Limit header")
	}
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 2, RegisterLimit: 2, APILimit: 100, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Exhaust IP A
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
		req.RemoteAddr = "1.1.1.1:1"
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// IP B should still work
	req := httptest.NewRequest("POST", "/api/v1/auth/verify", nil)
	req.RemoteAddr = "2.2.2.2:2"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 for different IP, got %d", w.Code)
	}
}

func TestRateLimiter_SkipsHealthCheck(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 1, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// healthz should never be rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("healthz attempt %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_SetsHeaders(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{APILimit: 50, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") != "50" {
		t.Errorf("expected limit 50, got %s", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "49" {
		t.Errorf("expected remaining 49, got %s", w.Header().Get("X-RateLimit-Remaining"))
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestRateLimiter_RegisterStricterThanLogin(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{LoginLimit: 5, RegisterLimit: 2, Window: time.Minute})
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Register limit is 2
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
		req.RemoteAddr = "1.1.1.1:1"
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	req := httptest.NewRequest("POST", "/api/v1/auth/register", nil)
	req.RemoteAddr = "1.1.1.1:1"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("expected 429 after 2 register attempts, got %d", w.Code)
	}
}

// --- SessionManager Tests ---

func TestSessionManager_NilRedis_PassesThrough(t *testing.T) {
	sm := NewSessionManager(nil)
	called := false
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))

	// Public path
	req := httptest.NewRequest("GET", "/api/v1/auth/verify", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Error("handler should be called for public path")
	}
}
