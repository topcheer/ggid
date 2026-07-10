package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// --- ResponseCache.get with expired entry ---

func TestResponseCache_GetExpiredEntry(t *testing.T) {
	rc := NewResponseCache(DefaultResponseCacheConfig())
	// Manually insert an expired entry
	key := "GET|/api/v1/data||tenant1"
	rc.put(key, &rcCachedResponse{
		status:    200,
		body:      []byte("old data"),
		cachedAt:  time.Now().Add(-2 * time.Minute),
		expiresAt: time.Now().Add(-1 * time.Minute), // already expired
		etag:      "abc123",
	})

	result := rc.get(key)
	if result != nil {
		t.Error("expired entry should return nil")
	}
}

func TestResponseCache_GetValidEntry(t *testing.T) {
	rc := NewResponseCache(DefaultResponseCacheConfig())
	key := "GET|/api/v1/data||tenant1"
	rc.put(key, &rcCachedResponse{
		status:    200,
		body:      []byte("data"),
		cachedAt:  time.Now(),
		expiresAt: time.Now().Add(30 * time.Second),
		etag:      "abc123",
	})

	result := rc.get(key)
	if result == nil {
		t.Error("valid entry should not return nil")
	}
	if result.etag != "abc123" {
		t.Errorf("etag = %q", result.etag)
	}
}

// --- ResponseCache.cleanup ---

func TestResponseCache_CleanupRemovesExpired(t *testing.T) {
	// Create cache with very short TTL for fast cleanup
	cfg := ResponseCacheConfig{
		TTL:            50 * time.Millisecond,
		MaxBodySize:    64 * 1024,
		EnabledMethods: map[string]bool{http.MethodGet: true},
	}
	rc := NewResponseCache(cfg)

	// Insert entries with short expiry
	rc.put("key1", &rcCachedResponse{
		status:    200,
		body:      []byte("data1"),
		expiresAt: time.Now().Add(30 * time.Millisecond),
	})
	rc.put("key2", &rcCachedResponse{
		status:    200,
		body:      []byte("data2"),
		expiresAt: time.Now().Add(30 * time.Millisecond),
	})

	// Wait for expiry + cleanup tick
	time.Sleep(200 * time.Millisecond)

	// Both should have been cleaned up
	rc.mu.RLock()
	count := len(rc.cache)
	rc.mu.RUnlock()

	if count > 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", count)
	}
}

// --- rcCaptureWriter.Header() ---

func TestRcCaptureWriter_Header_CreatesClone(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Custom", "value1")

	cw := &rcCaptureWriter{
		ResponseWriter: rec,
		status:         200,
	}

	// First call should clone headers
	h1 := cw.Header()
	if h1.Get("X-Custom") != "value1" {
		t.Errorf("Header() missing custom header: %q", h1.Get("X-Custom"))
	}

	// Verify internal header was populated
	if len(cw.header) == 0 {
		t.Error("cw.header should be populated after first Header() call")
	}

	// Second call should use the stored header
	h2 := cw.Header()
	if h2.Get("X-Custom") != "value1" {
		t.Errorf("second Header() missing custom header")
	}
}

func TestRcCaptureWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	cw := &rcCaptureWriter{
		ResponseWriter: rec,
		status:         200,
	}
	cw.WriteHeader(201)
	if cw.status != 201 {
		t.Errorf("status = %d", cw.status)
	}
	if rec.Code != 201 {
		t.Errorf("recorder code = %d", rec.Code)
	}
}

func TestRcCaptureWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	cw := &rcCaptureWriter{
		ResponseWriter: rec,
	}
	n, err := cw.Write([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 11 {
		t.Errorf("wrote %d bytes", n)
	}
	if string(cw.body) != "hello world" {
		t.Errorf("captured body = %q", cw.body)
	}
	if rec.Body.String() != "hello world" {
		t.Errorf("response body = %q", rec.Body.String())
	}
}

// --- ResponseCache Middleware edge cases ---

func TestResponseCache_Middleware_PostNotCached(t *testing.T) {
	rc := NewResponseCache(DefaultResponseCacheConfig())
	callCount := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte("post result"))
	})

	handler := rc.Middleware(next)

	// POST should not be cached
	req1 := httptest.NewRequest("POST", "/api/v1/data", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("POST", "/api/v1/data", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if callCount != 2 {
		t.Errorf("POST should not be cached, expected 2 calls, got %d", callCount)
	}
}

func TestResponseCache_Middleware_LargeBodyNotCached(t *testing.T) {
	cfg := DefaultResponseCacheConfig()
	cfg.MaxBodySize = 10 // very small limit
	rc := NewResponseCache(cfg)

	callCount := 0
	largeBody := make([]byte, 100)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write(largeBody)
	})

	handler := rc.Middleware(next)

	// Large response should not be cached
	req1 := httptest.NewRequest("GET", "/api/v1/large", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/api/v1/large", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if callCount != 2 {
		t.Errorf("large response should not be cached, expected 2 calls, got %d", callCount)
	}
}

func TestResponseCache_Middleware_OldCacheEntry(t *testing.T) {
	// Test that an old (non-ETag-able) cached entry is still served
	rc := NewResponseCache(DefaultResponseCacheConfig())

	callCount := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		w.Write([]byte("data"))
	})

	handler := rc.Middleware(next)

	// First request - should be cached
	req1 := httptest.NewRequest("GET", "/api/v1/data", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	// Second request - should be served from cache
	req2 := httptest.NewRequest("GET", "/api/v1/data", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if callCount != 1 {
		t.Errorf("second request should be served from cache, expected 1 call, got %d", callCount)
	}
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected X-Cache: HIT, got %q", w2.Header().Get("X-Cache"))
	}
}

func TestResponseCache_Invalidate_ByPrefix(t *testing.T) {
	rc := NewResponseCache(DefaultResponseCacheConfig())

	rc.put("GET|/api/v1/users/list|", &rcCachedResponse{status: 200, expiresAt: time.Now().Add(time.Hour)})
	rc.put("GET|/api/v1/users/get|", &rcCachedResponse{status: 200, expiresAt: time.Now().Add(time.Hour)})
	rc.put("GET|/api/v1/roles/list|", &rcCachedResponse{status: 200, expiresAt: time.Now().Add(time.Hour)})

	// Invalidate all /api/v1/users entries
	rc.Invalidate("GET|/api/v1/users/")

	rc.mu.RLock()
	count := len(rc.cache)
	rc.mu.RUnlock()

	if count != 1 {
		t.Errorf("expected 1 entry remaining, got %d", count)
	}
}

func TestResponseCache_ConcurrentAccess(t *testing.T) {
	rc := NewResponseCache(DefaultResponseCacheConfig())
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("concurrent data"))
	})

	handler := rc.Middleware(next)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/api/v1/data/%d", i)
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}(i)
	}
	wg.Wait()
}
