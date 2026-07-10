package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoalesce_PostWithIdempotencyKey(t *testing.T) {
	var callCount int32
	rc := NewRequestCoalescer(50 * time.Millisecond)
	mw := CoalesceMiddleware(rc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/resource", nil)
			req.Header.Set("Idempotency-Key", "abc-123")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call for coalesced POST, got %d", callCount)
	}
}

func TestCoalesce_PostWithoutIdempotencyKey_PassThrough(t *testing.T) {
	var callCount int32
	rc := NewRequestCoalescer(50 * time.Millisecond)
	mw := CoalesceMiddleware(rc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/resource", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}()
	}
	wg.Wait()

	// Without idempotency key, each request should pass through
	if atomic.LoadInt32(&callCount) != 5 {
		t.Errorf("expected 5 calls without idempotency key, got %d", callCount)
	}
}

func TestCoalesce_PostDifferentIdempotencyKeys(t *testing.T) {
	var callCount int32
	rc := NewRequestCoalescer(50 * time.Millisecond)
	mw := CoalesceMiddleware(rc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusCreated)
	}))

	// Two POSTs with different idempotency keys should both execute
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/resource", nil)
	req1.Header.Set("Idempotency-Key", "key-1")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/resource", nil)
	req2.Header.Set("Idempotency-Key", "key-2")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls for different keys, got %d", callCount)
	}
}

func TestCoalesce_PostIdempotency_CacheHit(t *testing.T) {
	var callCount int32
	rc := NewRequestCoalescer(100 * time.Millisecond)
	mw := CoalesceMiddleware(rc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"created":true}`))
	}))

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	req1.Header.Set("Idempotency-Key", "cache-test")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	// Second request with same key — should hit cache
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	req2.Header.Set("Idempotency-Key", "cache-test")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Errorf("expected 201 from cache, got %d", w2.Code)
	}
	if w2.Body.String() != `{"created":true}` {
		t.Errorf("expected cached body, got %s", w2.Body.String())
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call with cache hit, got %d", callCount)
	}
}

func TestCoalesce_PutWithIdempotencyKey(t *testing.T) {
	var callCount int32
	rc := NewRequestCoalescer(0) // no cache, but inflight dedup works
	mw := CoalesceMiddleware(rc)

	// Barrier ensures all goroutines have entered the middleware before the
	// handler starts, so they all find the inflight entry and coalesce.
	const numGoroutines = 5
	barrier := make(chan struct{})
	var entered int32

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond) // ensure overlap
		w.WriteHeader(http.StatusOK)
	}))

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPut, "/api/v1/resource/1", nil)
			req.Header.Set("Idempotency-Key", "put-key-1")
			w := httptest.NewRecorder()

			// Signal that this goroutine is about to enter ServeHTTP
			if atomic.AddInt32(&entered, 1) == numGoroutines {
				close(barrier) // all goroutines are ready
			}
			<-barrier // wait for all goroutines to be ready

			handler.ServeHTTP(w, req)
		}()
	}
	wg.Wait()

	// All concurrent PUTs with same idempotency key should coalesce to 1
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call for coalesced PUT, got %d", callCount)
	}
}
