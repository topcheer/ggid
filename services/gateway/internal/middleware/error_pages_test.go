package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- ErrorHandler tests ---

func TestErrorHandler_Timeout(t *testing.T) {
	handler := ErrorHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	r.Header.Set("X-Request-ID", "req-123")

	handler(w, r, errors.New("context deadline exceeded"))

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d", w.Code)
	}
	var gwErr GatewayError
	if err := json.NewDecoder(w.Body).Decode(&gwErr); err != nil {
		t.Fatal(err)
	}
	if gwErr.RequestID != "req-123" {
		t.Errorf("expected req-123, got %s", gwErr.RequestID)
	}
	if gwErr.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d", gwErr.Code)
	}
}

func TestErrorHandler_TimeoutString(t *testing.T) {
	handler := ErrorHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	handler(w, r, errors.New("dial timeout"))

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504 for timeout, got %d", w.Code)
	}
}

func TestErrorHandler_ConnectionRefused(t *testing.T) {
	handler := ErrorHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	handler(w, r, errors.New("connection refused"))

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
	var gwErr GatewayError
	if err := json.NewDecoder(w.Body).Decode(&gwErr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gwErr.Message, "unavailable") {
		t.Errorf("expected unavailable message, got %s", gwErr.Message)
	}
}

func TestErrorHandler_Default(t *testing.T) {
	handler := ErrorHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	handler(w, r, errors.New("some other error"))

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 default, got %d", w.Code)
	}
}

func TestErrorHandler_NoRequestID(t *testing.T) {
	handler := ErrorHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	handler(w, r, errors.New("generic error"))

	var gwErr GatewayError
	if err := json.NewDecoder(w.Body).Decode(&gwErr); err != nil {
		t.Fatal(err)
	}
	if gwErr.RequestID != "unknown" {
		t.Errorf("expected unknown, got %s", gwErr.RequestID)
	}
}

// --- WriteGatewayError tests ---

func TestWriteGatewayError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("X-Request-ID", "req-456")

	WriteGatewayError(w, r, http.StatusServiceUnavailable, "policy", "overloaded")

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	var gwErr GatewayError
	if err := json.NewDecoder(w.Body).Decode(&gwErr); err != nil {
		t.Fatal(err)
	}
	if gwErr.Backend != "policy" {
		t.Errorf("expected policy, got %s", gwErr.Backend)
	}
	if gwErr.RequestID != "req-456" {
		t.Errorf("expected req-456, got %s", gwErr.RequestID)
	}
}

func TestWriteGatewayError_NoRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	WriteGatewayError(w, r, http.StatusBadGateway, "", "failed")

	var gwErr GatewayError
	if err := json.NewDecoder(w.Body).Decode(&gwErr); err != nil {
		t.Fatal(err)
	}
	if gwErr.RequestID != "unknown" {
		t.Errorf("expected unknown, got %s", gwErr.RequestID)
	}
}

// --- PreflightCache tests ---

func TestNewPreflightCache_DefaultTTL(t *testing.T) {
	cache := NewPreflightCache(0)
	if cache.ttl != 5*time.Minute {
		t.Errorf("expected 5m default, got %v", cache.ttl)
	}
}

func TestNewPreflightCache_CustomTTL(t *testing.T) {
	cache := NewPreflightCache(10 * time.Minute)
	if cache.ttl != 10*time.Minute {
		t.Errorf("expected 10m, got %v", cache.ttl)
	}
}

func TestPreflightCache_SetGet(t *testing.T) {
	cache := NewPreflightCache(5 * time.Minute)
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r.Header.Set("Origin", "https://example.com")
	r.Header.Set("Access-Control-Request-Method", "POST")

	hdr := http.Header{}
	hdr.Set("Access-Control-Allow-Origin", "https://example.com")
	cache.Set(r, http.StatusOK, hdr)

	entry, ok := cache.Get(r)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.status != http.StatusOK {
		t.Errorf("expected 200, got %d", entry.status)
	}
	if entry.header.Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("header not cloned")
	}
}

func TestPreflightCache_Expired(t *testing.T) {
	cache := NewPreflightCache(1 * time.Millisecond)
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r.Header.Set("Origin", "https://example.com")

	cache.Set(r, http.StatusOK, http.Header{})
	time.Sleep(10 * time.Millisecond)

	_, ok := cache.Get(r)
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestPreflightCache_NoEntry(t *testing.T) {
	cache := NewPreflightCache(5 * time.Minute)
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)

	_, ok := cache.Get(r)
	if ok {
		t.Error("expected cache miss")
	}
}

func TestPreflightCacheMiddleware_NonOptions(t *testing.T) {
	cache := NewPreflightCache(5 * time.Minute)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := PreflightCacheMiddleware(cache, next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("next handler should be called for non-OPTIONS")
	}
}

func TestPreflightCacheMiddleware_CacheMiss(t *testing.T) {
	cache := NewPreflightCache(5 * time.Minute)
	backendCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
	})

	handler := PreflightCacheMiddleware(cache, next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w, r)

	if !backendCalled {
		t.Error("backend should be called on cache miss")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPreflightCacheMiddleware_CacheHit(t *testing.T) {
	cache := NewPreflightCache(5 * time.Minute)
	backendCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
	})

	handler := PreflightCacheMiddleware(cache, next)

	// First request: cache miss
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r1.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w1, r1)
	if !backendCalled {
		t.Fatal("backend should be called first time")
	}

	// Second request: should hit cache
	backendCalled = false
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r2.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w2, r2)
	if backendCalled {
		t.Error("backend should NOT be called on cache hit")
	}
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}
}

func TestPreflightCacheMiddleware_Non2xxNotCached(t *testing.T) {
	cache := NewPreflightCache(5 * time.Minute)
	callCount := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := PreflightCacheMiddleware(cache, next)

	// First request: 500 should NOT be cached
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r1.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w1, r1)

	// Second request: should still call backend since 500 not cached
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	r2.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(w2, r2)

	if callCount != 2 {
		t.Errorf("expected 2 backend calls, got %d", callCount)
	}
}

// --- RouteBodySizeConfig tests ---

func TestNewRouteBodySizeConfig(t *testing.T) {
	cfg := NewRouteBodySizeConfig()
	if cfg.Default != 10*1024*1024 {
		t.Errorf("expected 10MB default, got %d", cfg.Default)
	}
}

func TestRouteBodySizeConfig_GetLimit(t *testing.T) {
	cfg := NewRouteBodySizeConfig()

	tests := []struct {
		path     string
		expected int64
	}{
		{"/api/v1/auth/login", 1024},
		{"/api/v1/auth/register", 2048},
		{"/api/v1/audit/events", 0}, // unlimited
		{"/api/v1/users", 10 * 1024 * 1024}, // default
		{"/unknown", 10 * 1024 * 1024},      // default
	}

	for _, tt := range tests {
		got := cfg.GetLimit(tt.path)
		if got != tt.expected {
			t.Errorf("GetLimit(%q) = %d, want %d", tt.path, got, tt.expected)
		}
	}
}

func TestRouteBodySizeMiddleware(t *testing.T) {
	cfg := NewRouteBodySizeConfig()
	mw := RouteBodySizeMiddleware(cfg)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := mw(next)

	// Normal-sized body should pass
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("small body"))
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRouteBodySizeMiddleware_UnlimitedRoute(t *testing.T) {
	cfg := NewRouteBodySizeConfig()
	cfg.Limits["/api/v1/audit"] = 0 // unlimited

	mw := RouteBodySizeMiddleware(cfg)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := mw(next)

	// Large body on unlimited route should pass
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/audit/events", strings.NewReader(strings.Repeat("x", 1024*1024)))
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for unlimited route, got %d", w.Code)
	}
}
