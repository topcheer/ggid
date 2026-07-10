package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestResponseCache_BasicGetHit(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))

	// First request → MISS
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req)
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if w1.Header().Get("X-Cache") != "MISS" {
		t.Error("expected X-Cache: MISS")
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected non-empty ETag")
	}

	// Second request → HIT
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
	req2.Header.Set("X-Tenant-ID", "t1")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("expected still 1 call (cached), got %d", calls)
	}
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Error("expected X-Cache: HIT")
	}
}

func TestResponseCache_IfNoneMatch_304(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":"test"}`))
	}))

	// First request to populate cache
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	etag := w1.Header().Get("ETag")

	// Second request with If-None-Match
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %d", w2.Code)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Error("should not call handler on 304")
	}
}

func TestResponseCache_IfModifiedSince_304(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`hello`))
	}))

	// Populate cache
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/lm", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	lastMod := w1.Header().Get("Last-Modified")

	// Request with If-Modified-Since
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/lm", nil)
	req2.Header.Set("If-Modified-Since", lastMod)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Errorf("expected 304, got %d", w2.Code)
	}
}

func TestResponseCache_PostNotCached(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	}))

	// POST should not be cached
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if atomic.LoadInt32(&calls) != 2 {
		t.Error("POST should not be cached, expected 2 calls")
	}
}

func TestResponseCache_NoCacheHeaderBypasses(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))

	// First request populates cache
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/nc", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	// Second request with Cache-Control: no-cache bypasses cache
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/nc", nil)
	req2.Header.Set("Cache-Control", "no-cache")
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if atomic.LoadInt32(&calls) != 2 {
		t.Error("no-cache should bypass, expected 2 calls")
	}
}

func TestResponseCache_DifferentTenantsSeparate(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))

	// Tenant A
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/same", nil)
	req1.Header.Set("X-Tenant-ID", "tenant-a")
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	// Tenant B
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/same", nil)
	req2.Header.Set("X-Tenant-ID", "tenant-b")
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if atomic.LoadInt32(&calls) != 2 {
		t.Error("different tenants should have separate cache entries")
	}
}

func TestResponseCache_ErrorNotCached(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/err", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req1)
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/err", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if atomic.LoadInt32(&calls) != 2 {
		t.Error("500 responses should not be cached")
	}
}

func TestResponseCache_Invalidate(t *testing.T) {
	var calls int32
	rc := NewResponseCache(DefaultResponseCacheConfig())
	handler := rc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`x`))
	}))

	// Populate
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Should be cached
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatal("expected 1 call before invalidate")
	}

	// Invalidate
	rc.Invalidate("GET|/api/v1/x")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if atomic.LoadInt32(&calls) != 2 {
		t.Error("expected 2 calls after invalidate")
	}
}

func TestResponseCache_Clear(t *testing.T) {
	rc := NewResponseCache(ResponseCacheConfig{
		TTL: time.Minute, MaxBodySize: 1024,
		EnabledMethods: map[string]bool{http.MethodGet: true},
	})
	rc.put("key1", &rcCachedResponse{status: 200, expiresAt: time.Now().Add(time.Hour)})
	rc.put("key2", &rcCachedResponse{status: 200, expiresAt: time.Now().Add(time.Hour)})
	rc.Clear()
	if len(rc.cache) != 0 {
		t.Error("expected empty cache after Clear")
	}
}

func TestCacheKey_Unique(t *testing.T) {
	k1 := rcCacheKey("GET", "/api/v1/users", "", "t1")
	k2 := rcCacheKey("GET", "/api/v1/users", "", "t2")
	if k1 == k2 {
		t.Error("different tenants should have different keys")
	}
}

func TestGenerateETag_Consistent(t *testing.T) {
	etag1 := rcGenerateETag([]byte(`{"id":1}`))
	etag2 := rcGenerateETag([]byte(`{"id":1}`))
	if etag1 != etag2 {
		t.Error("ETags for same body should match")
	}
	etag3 := rcGenerateETag([]byte(`{"id":2}`))
	if etag1 == etag3 {
		t.Error("ETags for different bodies should differ")
	}
}
