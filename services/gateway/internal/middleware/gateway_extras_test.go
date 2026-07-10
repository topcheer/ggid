package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// --- Slow Request Detection ---

func TestSlowRequestMiddleware_FastRequest(t *testing.T) {
	var mu sync.Mutex
	called := false
	cfg := &SlowRequestConfig{
		Threshold: 1 * time.Second,
		OnSlow: func(_ *SlowRequestInfo) {
			mu.Lock()
			called = true
			mu.Unlock()
		},
	}
	h := SlowRequestMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/fast", nil))

	mu.Lock()
	defer mu.Unlock()
	if called {
		t.Error("OnSlow should not be called for fast request")
	}
}

func TestSlowRequestMiddleware_SlowRequest(t *testing.T) {
	var mu sync.Mutex
	var info *SlowRequestInfo
	cfg := &SlowRequestConfig{
		Threshold: 50 * time.Millisecond,
		OnSlow: func(i *SlowRequestInfo) {
			mu.Lock()
			info = i
			mu.Unlock()
		},
	}
	h := SlowRequestMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/slow", nil))

	mu.Lock()
	defer mu.Unlock()
	if info == nil {
		t.Fatal("OnSlow should have been called")
	}
	if info.Path != "/slow" {
		t.Errorf("expected /slow, got %s", info.Path)
	}
}

func TestDefaultSlowRequestConfig(t *testing.T) {
	cfg := DefaultSlowRequestConfig()
	if cfg.Threshold != 5*time.Second {
		t.Errorf("expected 5s, got %v", cfg.Threshold)
	}
}

// --- WebSocket Connection Limits ---

func TestWSConnLimiter_AllowWithinLimit(t *testing.T) {
	l := NewWSConnLimiter(5)
	allowed, evict := l.Allow("t1", "s1")
	if !allowed || evict != "" {
		t.Error("expected allowed with no eviction")
	}
	if l.Count("t1") != 1 {
		t.Errorf("expected 1, got %d", l.Count("t1"))
	}
}

func TestWSConnLimiter_EvictOldest(t *testing.T) {
	l := NewWSConnLimiter(2)
	l.Allow("t1", "s1")
	l.Allow("t1", "s2")

	// Third connection should evict s1
	allowed, evict := l.Allow("t1", "s3")
	if !allowed {
		t.Error("expected allowed")
	}
	if evict != "s1" {
		t.Errorf("expected s1 evicted, got %s", evict)
	}
	if l.Count("t1") != 2 {
		t.Errorf("expected 2, got %d", l.Count("t1"))
	}
}

func TestWSConnLimiter_Release(t *testing.T) {
	l := NewWSConnLimiter(10)
	l.Allow("t1", "s1")
	l.Allow("t1", "s2")
	l.Release("s1")
	if l.Count("t1") != 1 {
		t.Errorf("expected 1 after release, got %d", l.Count("t1"))
	}
}

func TestWSConnLimiter_ReleaseUnknown(t *testing.T) {
	l := NewWSConnLimiter(5)
	l.Release("nonexistent") // should not panic
}

func TestWSConnLimiter_TotalCount(t *testing.T) {
	l := NewWSConnLimiter(10)
	l.Allow("t1", "s1")
	l.Allow("t1", "s2")
	l.Allow("t2", "s3")
	if l.TotalCount() != 3 {
		t.Errorf("expected 3 total, got %d", l.TotalCount())
	}
}

func TestWSConnLimiter_DefaultMax(t *testing.T) {
	l := NewWSConnLimiter(0)
	if l.maxPerTenant != 100 {
		t.Errorf("expected default 100, got %d", l.maxPerTenant)
	}
}

// --- Fallback Response ---

func TestFallbackConfig_SetGet(t *testing.T) {
	fc := NewFallbackConfig()
	fc.Set("/api/v1/users", &CachedResponse{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"users":[]}`),
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
	})

	resp := fc.Get("/api/v1/users?page=1")
	if resp == nil {
		t.Fatal("expected fallback found")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestFallbackConfig_NoMatch(t *testing.T) {
	fc := NewFallbackConfig()
	if fc.Get("/unknown") != nil {
		t.Error("expected nil for unconfigured route")
	}
}

func TestFallbackMiddleware_ServesOn502(t *testing.T) {
	fc := NewFallbackConfig()
	fc.Set("/api/v1/users", &CachedResponse{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"fallback":true}`),
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	h := FallbackMiddleware(fc)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	h.ServeHTTP(w, r)

	if w.Body.String() != `{"fallback":true}` {
		t.Errorf("expected fallback body, got %s", w.Body.String())
	}
	if w.Header().Get("X-Fallback") != "true" {
		t.Error("expected X-Fallback header")
	}
}

func TestFallbackMiddleware_PassesOn200(t *testing.T) {
	fc := NewFallbackConfig()
	fc.Set("/api/v1/users", &CachedResponse{StatusCode: 200, Body: []byte("fb")})

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("real"))
	})
	h := FallbackMiddleware(fc)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	h.ServeHTTP(w, r)

	if w.Body.String() != "real" {
		t.Errorf("expected real response, got %s", w.Body.String())
	}
}

func TestFallbackMiddleware_NoFallbackConfigured(t *testing.T) {
	fc := NewFallbackConfig()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	h := FallbackMiddleware(fc)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/users", nil)
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 passthrough, got %d", w.Code)
	}
}
