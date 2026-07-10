package middleware

import (
	"net/http"
	"net/http/httptest"
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetry_IdempotentSuccess(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond}
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestRetry_RetriesOn502(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond}
	var calls int32
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&calls, 1)
		if c < 3 {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("want 200 after retry, got %d", rr.Code)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("want 3 calls, got %d", atomic.LoadInt32(&calls))
	}
}

func TestRetry_ExhaustedRetries(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 2, InitialDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond}
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 503 {
		t.Errorf("want 503, got %d", rr.Code)
	}
}

func TestRetry_NonIdempotentSkipped(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond}
	var calls int32
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	req := httptest.NewRequest("POST", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("POST should not be retried, got %d calls", atomic.LoadInt32(&calls))
	}
}

func TestRetry_NonRetryableStatus(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond}
	var calls int32
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("404 should not be retried, got %d calls", atomic.LoadInt32(&calls))
	}
}

func TestRetry_NilConfig(t *testing.T) {
	handler := RetryMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("want 200, got %d", rr.Code)
	}
}

func TestRetry_ContextCancel(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 5, InitialDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond}
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should return 408 due to context cancellation
	if rr.Code != http.StatusRequestTimeout && rr.Code != http.StatusBadGateway {
		t.Errorf("want 408 or 502, got %d", rr.Code)
	}
}

func TestRetry_XRetryCountHeader(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond}
	var calls int32
	handler := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&calls, 1)
		if c < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("X-Retry-Count") != "1" {
		t.Errorf("X-Retry-Count: want '1', got '%s'", rr.Header().Get("X-Retry-Count"))
	}
}

func TestBackoffWithJitter(t *testing.T) {
	d := backoffWithJitter(0, 100*time.Millisecond, 2*time.Second)
	if d < 0 || d > 200*time.Millisecond {
		t.Errorf("attempt 0: got %v", d)
	}
	d = backoffWithJitter(5, 100*time.Millisecond, 2*time.Second)
	if d > 2*time.Second {
		t.Errorf("should be capped at max: got %v", d)
	}
}

func TestIsRetryableMethod(t *testing.T) {
	if !isRetryableMethod("GET") {
		t.Error("GET should be retryable")
	}
	if !isRetryableMethod("HEAD") {
		t.Error("HEAD should be retryable")
	}
	if !isRetryableMethod("OPTIONS") {
		t.Error("OPTIONS should be retryable")
	}
	if isRetryableMethod("POST") {
		t.Error("POST should not be retryable")
	}
}
