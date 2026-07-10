package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Tracing Tests ---

func TestTracing_GeneratesTraceID(t *testing.T) {
	var traceID string
	handler := Tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, ok := FromContext(r.Context())
		if ok {
			traceID = tc.TraceID
		}
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if traceID == "" {
		t.Error("expected non-empty trace ID in context")
	}
	if w.Header().Get("X-Trace-ID") == "" {
		t.Error("expected X-Trace-ID response header")
	}
	if w.Header().Get("X-Span-ID") == "" {
		t.Error("expected X-Span-ID response header")
	}
}

func TestTracing_PropagatesExistingTrace(t *testing.T) {
	existingTrace := "00-" + generateTraceID() + "-" + generateSpanID() + "-01"
	var parentID string

	handler := Tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, _ := FromContext(r.Context())
		parentID = tc.ParentID
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", existingTrace)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if parentID == "" {
		t.Error("expected non-empty parent ID from incoming traceparent")
	}
}

func TestTracing_DifferentRequestsDifferentTraces(t *testing.T) {
	var traces []string
	handler := Tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, _ := FromContext(r.Context())
		traces = append(traces, tc.TraceID)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	if traces[0] == traces[1] || traces[1] == traces[2] {
		t.Error("expected different trace IDs for different requests")
	}
}

// --- ParseMaxBodySize Tests (additional) ---

func TestParseMaxBodySize_Lowercase(t *testing.T) {
	if ParseMaxBodySize("5mb") != 5<<20 {
		t.Error("expected 5MB")
	}
	if ParseMaxBodySize("2gb") != 2<<30 {
		t.Error("expected 2GB")
	}
}

// --- IncAuthFailure / SetActiveSessions (additional) ---

func TestIncAuthFailure_MultipleCalls(t *testing.T) {
	for i := 0; i < 10; i++ {
		IncAuthFailure("test-reason")
	}
}

// --- Cache Invalidate (additional) ---

func TestCache_InvalidateClearsAll(t *testing.T) {
	cache := NewCache(60 * 1000000000)
	// Add entry manually
	cache.mu.Lock()
	cache.entries["GET:/test"] = &cacheEntry{
		body:      []byte("cached"),
		etag:      "abc",
		expiresAt: cacheEntryExpiry(cache),
	}
	cache.mu.Unlock()

	if len(cache.entries) != 1 {
		t.Fatal("expected 1 entry before invalidate")
	}

	cache.Invalidate()
	if len(cache.entries) != 0 {
		t.Error("expected 0 entries after invalidate")
	}
}

func cacheEntryExpiry(c *Cache) time.Time {
	return time.Now().Add(c.ttl)
}

// --- BotDetect edge cases ---

func TestBotDetect_EmptyUserAgent(t *testing.T) {
	called := false
	handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	// No User-Agent header
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("empty UA should pass through")
	}
}

func TestBotDetect_MultiplePatterns(t *testing.T) {
	tests := []string{
		"Googlebot/2.1",
		"bingbot/2.0",
		"duckduckbot",
		"linkedinbot/2.0",
	}
	for _, ua := range tests {
		handler := BotDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("User-Agent", ua)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Header().Get("X-Bot-Detected") == "" {
			t.Errorf("expected bot detection for %s", ua)
		}
	}
}

// --- BehavioralBotDetect additional ---

func TestBehavioralBotDetect_DifferentIPs(t *testing.T) {
	detector := NewBehavioralBotDetect(2, 60*1000000000)
	handler := detector.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	// IP A: 2 requests (at limit)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.1.1.1:1"
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// IP B: should still work
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "2.2.2.2:2"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("different IP should not be limited, got %d", w.Code)
	}
}
