package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGraphQLHandler_NoFields_C21(t *testing.T) {
	r := &GraphQLResolver{BackendURLs: make(map[string]string)}
	h := r.GraphQLHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(`{"query":"query { }"}`))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("code=%d", rr.Code)
	}
}

func TestStatsMiddleware_Basic_C21(t *testing.T) {
	col := NewStatsCollector()
	called := false
	h := StatsMiddleware(col, func(p string) string { return "/api" })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/users", nil))
	if !called {
		t.Error("not called")
	}
	if col.Snapshot().TotalRequests != 1 {
		t.Error("requests != 1")
	}
}

func TestStatsMiddleware_NilResolver_C21(t *testing.T) {
	col := NewStatsCollector()
	h := StatsMiddleware(col, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

func TestStatsHandler_C21(t *testing.T) {
	rr := httptest.NewRecorder()
	NewStatsCollector().StatsHandler().ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 200 {
		t.Errorf("code=%d", rr.Code)
	}
}

func TestNormalizePath_UUID_C21(t *testing.T) {
	got := normalizePath("/api/users/550e8400-e29b-41d4-a716-446655440000")
	if !strings.Contains(got, "{id}") {
		t.Errorf("got=%q want {id}", got)
	}
}

func TestIsID_UUID_C21(t *testing.T) {
	if !isID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("UUID should be ID")
	}
	if isID("users") {
		t.Error("users not ID")
	}
}

func TestGetTimeoutForRoute_Custom_C21(t *testing.T) {
	cfg := &TimeoutConfig{
		Default:      30 * time.Second,
		RouteConfigs: map[string]time.Duration{"/slow": 120 * time.Second},
	}
	if d := cfg.GetTimeoutForRoute("/slow"); d != 120*time.Second {
		t.Errorf("got=%v", d)
	}
	if d := cfg.GetTimeoutForRoute("/unknown"); d != 30*time.Second {
		t.Errorf("got=%v", d)
	}
}

func TestTokenBucket_RetryAfter_C21(t *testing.T) {
	tb := NewTokenBucket(5, 1)
	for i := 0; i < 5; i++ {
		tb.Allow()
	}
	if tb.RetryAfter() <= 0 {
		t.Error("RetryAfter should be positive")
	}
}

func TestClaimsFromContext_Nil_C21(t *testing.T) {
	if c := ClaimsFromContext(nil); c.Subject != "" {
		t.Error("nil ctx should have empty Subject")
	}
}

func TestTenantResolver_Header_C21(t *testing.T) {
	called := false
	h := TenantResolver("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("not called")
	}
}

func TestHealthScore_C21(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.9)
	hs.RecordSuccess("b1", 50*time.Millisecond)
	hs.RecordError("b1")
	if len(hs.AllScores()) == 0 {
		t.Error("no scores")
	}
}

func TestWasmUnload_C21(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())
	if host.UnloadPlugin(context.Background(), "nope") == nil {
		t.Error("want error")
	}
}

func TestParseMaxBodySize_GB_C21(t *testing.T) {
	if v := ParseMaxBodySize("2GB"); v != 2<<30 {
		t.Errorf("got=%d", v)
	}
}

func TestMaxBodySize_Allow_C21(t *testing.T) {
	h := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("POST", "/", strings.NewReader("ok")))
	if rr.Code != 200 {
		t.Errorf("code=%d", rr.Code)
	}
}

func TestTimeoutMiddleware_504_C21(t *testing.T) {
	cfg := &TimeoutConfig{Default: 50 * time.Millisecond}
	h := TimeoutMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/slow", nil))
	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("code=%d want 504", rr.Code)
	}
}

func TestTimeoutMiddleware_Fast_C21(t *testing.T) {
	cfg := &TimeoutConfig{Default: 5 * time.Second}
	h := TimeoutMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 200 {
		t.Errorf("code=%d", rr.Code)
	}
}
