package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestMemoryRateLimitStore_CountAndAdd(t *testing.T) {
	store := NewMemoryRateLimitStore()
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// First request
	count, err := store.CountAndAdd(context.Background(), "tenant1", windowStart, now)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("first request count = %d, want 0", count)
	}

	// Second request
	count, err = store.CountAndAdd(context.Background(), "tenant1", windowStart, now.Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("second request count = %d, want 1", count)
	}

	// Third request
	count, err = store.CountAndAdd(context.Background(), "tenant1", windowStart, now.Add(2*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("third request count = %d, want 2", count)
	}
}

func TestMemoryRateLimitStore_WindowEviction(t *testing.T) {
	store := NewMemoryRateLimitStore()
	old := time.Now().Add(-2 * time.Minute)
	recent := time.Now()

	// Add an old entry
	_, _ = store.CountAndAdd(context.Background(), "k", old.Add(-time.Minute), old)

	// Now add with a window that excludes the old entry
	windowStart := recent.Add(-time.Minute)
	count, _ := store.CountAndAdd(context.Background(), "k", windowStart, recent)
	if count != 0 {
		t.Errorf("evicted old entry: count = %d, want 0", count)
	}
}

func TestMemoryRateLimitStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryRateLimitStore()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			now := time.Now().Add(time.Duration(n) * time.Microsecond)
			_, _ = store.CountAndAdd(context.Background(), "concurrent", now.Add(-time.Minute), now)
		}(i)
	}
	wg.Wait()
	// Should not panic or race
}

func TestRedisRateLimitStore_CountAndAdd(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Skipf("miniredis unavailable: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	store := NewRedisRateLimitStore(client)
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	count, err := store.CountAndAdd(context.Background(), "tenant1", windowStart, now)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("first request count = %d, want 0", count)
	}

	count, err = store.CountAndAdd(context.Background(), "tenant1", windowStart, now.Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("second request count = %d, want 1", count)
	}
}

func TestRedisRateLimitStore_WindowEviction(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Skipf("miniredis unavailable: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	store := NewRedisRateLimitStore(client)

	// Add 3 requests in the first minute
	t0 := time.Now().Add(-90 * time.Second)
	for i := 0; i < 3; i++ {
		_, _ = store.CountAndAdd(context.Background(), "evict", t0.Add(-time.Minute), t0.Add(time.Duration(i)*time.Second))
	}

	// Now request with window that only includes recent entries
	recent := time.Now()
	count, _ := store.CountAndAdd(context.Background(), "evict", recent.Add(-30*time.Second), recent)
	if count != 0 {
		t.Errorf("expected 0 entries in 30s window (old ones evicted), got %d", count)
	}
}

func TestSlidingWindowLimiter_AllowsUnderLimit(t *testing.T) {
	store := NewMemoryRateLimitStore()
	cfg := SlidingWindowConfig{
		Tiers: map[Tier]SlidingWindowTierLimit{
			TierFree: {Requests: 5, Window: time.Minute},
		},
		KeyPrefix: "test:",
	}
	limiter := NewSlidingWindowLimiter(store, cfg)

	called := 0
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(200)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/v1/data", nil)
		req = req.WithContext(WithTier(req.Context(), TierFree))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}
	if called != 5 {
		t.Errorf("handler called %d times, want 5", called)
	}
}

func TestSlidingWindowLimiter_BlocksOverLimit(t *testing.T) {
	store := NewMemoryRateLimitStore()
	cfg := SlidingWindowConfig{
		Tiers: map[Tier]SlidingWindowTierLimit{
			TierFree: {Requests: 3, Window: time.Minute},
		},
	}
	limiter := NewSlidingWindowLimiter(store, cfg)

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// Exhaust the limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/v1/data", nil)
		req = req.WithContext(WithTier(req.Context(), TierFree))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// 4th request should be blocked
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req = req.WithContext(WithTier(req.Context(), TierFree))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
	if w.Header().Get("X-RateLimit-Tier") != string(TierFree) {
		t.Error("expected X-RateLimit-Tier header")
	}
}

func TestSlidingWindowLimiter_EnterpriseUnlimited(t *testing.T) {
	store := NewMemoryRateLimitStore()
	cfg := DefaultSlidingWindowConfig()
	limiter := NewSlidingWindowLimiter(store, cfg)

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest("GET", "/api/v1/data", nil)
		req = req.WithContext(WithTier(req.Context(), TierEnterprise))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("enterprise request %d: expected 200, got %d", i, w.Code)
			break
		}
	}
}

func TestSlidingWindowLimiter_SkipsHealthCheck(t *testing.T) {
	store := NewMemoryRateLimitStore()
	cfg := DefaultSlidingWindowConfig()
	limiter := NewSlidingWindowLimiter(store, cfg)

	for _, path := range []string{"/healthz", "/healthz/live", "/healthz/ready", "/docs"} {
		handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("health check %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSlidingWindowLimiter_RateLimitHeaders(t *testing.T) {
	store := NewMemoryRateLimitStore()
	cfg := SlidingWindowConfig{
		Tiers: map[Tier]SlidingWindowTierLimit{
			TierPro: {Requests: 100, Window: time.Minute},
		},
	}
	limiter := NewSlidingWindowLimiter(store, cfg)

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req = req.WithContext(WithTier(req.Context(), TierPro))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("X-RateLimit-Limit = %q, want '100'", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header")
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}
