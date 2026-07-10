package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestStatsCollector_Record(t *testing.T) {
	sc := NewStatsCollector()
	sc.Record("/api/v1/users", "GET", 200, 1024, 10*time.Millisecond)
	sc.Record("/api/v1/auth", "POST", 200, 512, 5*time.Millisecond)
	sc.Record("/api/v1/users", "GET", 404, 0, 1*time.Millisecond)
	sc.Record("/api/v1/audit", "GET", 502, 0, 50*time.Millisecond)

	snap := sc.Snapshot()
	if snap.TotalRequests != 4 {
		t.Errorf("expected 4 total requests, got %d", snap.TotalRequests)
	}
	if snap.TotalErrors != 1 { // 500 = 1 proxy error
		t.Errorf("expected 1 error, got %d", snap.TotalErrors)
	}
	if snap.StatusBreakdown["2xx"] != 2 {
		t.Errorf("expected 2 2xx, got %d", snap.StatusBreakdown["2xx"])
	}
	if snap.StatusBreakdown["4xx"] != 1 {
		t.Errorf("expected 1 4xx, got %d", snap.StatusBreakdown["4xx"])
	}
	if snap.StatusBreakdown["5xx"] != 1 {
		t.Errorf("expected 1 5xx, got %d", snap.StatusBreakdown["5xx"])
	}
}

func TestStatsCollector_PerRoute(t *testing.T) {
	sc := NewStatsCollector()
	sc.Record("/api/v1/users", "GET", 200, 100, 5*time.Millisecond)
	sc.Record("/api/v1/users", "GET", 404, 0, 1*time.Millisecond)

	snap := sc.Snapshot()
	rs, ok := snap.Routes["/api/v1/users"]
	if !ok {
		t.Fatal("expected per-route stats for /api/v1/users")
	}
	if rs.Requests != 2 {
		t.Errorf("expected 2 requests, got %d", rs.Requests)
	}
}

func TestStatsMiddleware_RecordsRequests(t *testing.T) {
	sc := NewStatsCollector()
	mw := StatsMiddleware(sc, func(path string) string {
		return ExtractRoutePrefix(path)
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello"))
	})).ServeHTTP(w, req)

	snap := sc.Snapshot()
	if snap.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", snap.TotalRequests)
	}
	if snap.TotalBytesSent != 5 { // "hello" = 5 bytes
		t.Errorf("expected 5 bytes sent, got %d", snap.TotalBytesSent)
	}
}

func TestStatsHandler_ReturnsJSON(t *testing.T) {
	sc := NewStatsCollector()
	sc.Record("/test", "GET", 200, 100, 5*time.Millisecond)

	req := httptest.NewRequest("GET", "/api/v1/gateway/stats", nil)
	w := httptest.NewRecorder()
	sc.StatsHandler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp StatsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", resp.TotalRequests)
	}
}

func TestDefaultMiddlewareChain(t *testing.T) {
	chain := DefaultMiddlewareChain()
	if chain.Count < 10 {
		t.Errorf("expected at least 10 middleware, got %d", chain.Count)
	}
	if len(chain.Outer) == 0 {
		t.Error("expected outer middleware chain")
	}
	if len(chain.Inner) == 0 {
		t.Error("expected inner middleware chain")
	}
	if chain.Outer[0].Name != "PanicRecovery" {
		t.Errorf("expected PanicRecovery first, got %s", chain.Outer[0].Name)
	}
}

func TestMiddlewareChainHandler_ReturnsJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/gateway/middleware", nil)
	w := httptest.NewRecorder()
	MiddlewareChainHandler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var chain MiddlewareChain
	if err := json.NewDecoder(w.Body).Decode(&chain); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if chain.Count == 0 {
		t.Error("expected non-zero middleware count")
	}
}

func TestExtractRoutePrefix(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users", "/api/v1/users"},
		{"/api/v1/users/123", "/api/v1/users"},
		{"/api/v1/users/123/roles", "/api/v1/users"},
		{"/healthz", "/healthz"},
		{"/", "/"},
	}
	for _, tt := range tests {
		if got := ExtractRoutePrefix(tt.path); got != tt.want {
			t.Errorf("ExtractRoutePrefix(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestStatsCollector_ConcurrentAccess(t *testing.T) {
	sc := NewStatsCollector()
	var done atomic.Int64
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done.Add(1) }()
			sc.Record("/test", "GET", 200, 100, 1*time.Millisecond)
		}()
	}
	for done.Load() < 100 {
		time.Sleep(time.Millisecond)
	}
	snap := sc.Snapshot()
	if snap.TotalRequests != 100 {
		t.Errorf("expected 100 requests, got %d", snap.TotalRequests)
	}
}
