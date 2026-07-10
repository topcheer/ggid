package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGzipBrotli_GzipResponse_V2(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Repeat("hello world ", 100)))
	})
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected gzip, got %q", w.Header().Get("Content-Encoding"))
	}
}

func TestGzipBrotli_BrotliResponse_V2(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Repeat("hello world ", 100)))
	})
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	req.Header.Set("Accept-Encoding", "br")
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") == "" {
		t.Error("expected encoding")
	}
}

func TestGzipBrotli_NoEncodingHeader_V2(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain response"))
	})
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") != "" {
		t.Error("should not compress without Accept-Encoding")
	}
}

func TestGzipBrotli_AlreadyEncoded_V2(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.Write([]byte("already encoded"))
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Error("should not double-encode")
	}
}

func TestHealthScore_IsHealthy_Unknown_V2(t *testing.T) {
	hc := NewHealthScore(10*time.Second, 0.5)
	if !hc.IsHealthy("unknown", 0.5) {
		t.Error("unknown backend should be healthy")
	}
}

func TestHealthScore_IsHealthy_Degraded_V2(t *testing.T) {
	hc := NewHealthScore(10*time.Second, 0.5)
	for i := 0; i < 3; i++ {
		hc.RecordSuccess("b1", 50*time.Millisecond)
	}
	for i := 0; i < 6; i++ {
		hc.RecordError("b1")
	}
	if hc.IsHealthy("b1", 0.5) {
		t.Error("degraded backend should be unhealthy")
	}
}

func TestHealthScore_AllScores_V2(t *testing.T) {
	hc := NewHealthScore(10*time.Second, 0.5)
	hc.RecordSuccess("b1", 10*time.Millisecond)
	hc.RecordError("b2")
	scores := hc.AllScores()
	if len(scores) != 2 {
		t.Errorf("expected 2 scores, got %d", len(scores))
	}
}

func TestAdaptiveRateLimiter_Allow_V2(t *testing.T) {
	al := NewAdaptiveRateLimiter(10, 1, 100)
	for i := 0; i < 5; i++ {
		al.Allow("k1")
	}
}

func TestAdaptiveRateLimiter_RecordLatency_V2(t *testing.T) {
	al := NewAdaptiveRateLimiter(10, 1, 100)
	al.RecordLatency("b1", 50*time.Millisecond)
	if al.Limit("b1") <= 0 {
		t.Error("limit should be > 0")
	}
}

func TestAdaptiveRateLimiter_SetLimit_V2(t *testing.T) {
	al := NewAdaptiveRateLimiter(10, 1, 100)
	al.SetLimit("b1", 20)
	if al.Limit("b1") != 20 {
		t.Errorf("expected 20, got %v", al.Limit("b1"))
	}
}

func TestAdaptiveRateLimiter_AllLimits_V2(t *testing.T) {
	al := NewAdaptiveRateLimiter(10, 1, 100)
	al.Allow("k1")
	al.Allow("k2")
	limits := al.AllLimits()
	if len(limits) < 2 {
		t.Errorf("expected >= 2 limits, got %d", len(limits))
	}
}

func TestRotatableAPIKeyValidator_V2(t *testing.T) {
	v := NewRotatableAPIKeyValidator(5 * time.Minute)
	if v == nil {
		t.Fatal("validator should not be nil")
	}
}
