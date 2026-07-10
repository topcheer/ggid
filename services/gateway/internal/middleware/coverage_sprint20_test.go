package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Task 1: coalesce recorder
func TestCoalesceRecorder_Header_C20(t *testing.T) {
	w := httptest.NewRecorder()
	cr := &coalesceRecorder{ResponseWriter: w, status: http.StatusOK, header: make(http.Header)}
	if cr.Header() == nil { t.Fatal("nil header") }
	cr.Header().Set("X-Test", "v")
	if cr.header.Get("X-Test") != "v" { t.Error("header mismatch") }
}
func TestCoalesceRecorder_WriteHeader_C20(t *testing.T) {
	w := httptest.NewRecorder()
	cr := &coalesceRecorder{ResponseWriter: w, status: http.StatusOK, header: make(http.Header)}
	cr.WriteHeader(http.StatusAccepted)
	if cr.status != http.StatusAccepted { t.Errorf("status=%d", cr.status) }
}
func TestCoalesceRecorder_Write_C20(t *testing.T) {
	w := httptest.NewRecorder()
	cr := &coalesceRecorder{ResponseWriter: w, status: http.StatusOK, header: make(http.Header)}
	n, _ := cr.Write([]byte("hello"))
	if n != 5 || cr.body.String() != "hello" { t.Errorf("n=%d body=%q", n, cr.body.String()) }
}

// Task 1: coalesce inflight + cache
func TestCoalesceMiddleware_InflightWait_C20(t *testing.T) {
	rc := NewRequestCoalescer(100 * time.Millisecond)
	mu := sync.Mutex{}
	cc := 0
	h := CoalesceMiddleware(rc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock(); cc++; mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() { defer wg.Done()
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/d", nil))
		}()
	}
	wg.Wait()
	if cc > 2 { t.Errorf("callCount=%d", cc) }
}
func TestCoalesceMiddleware_CacheHit_C20(t *testing.T) {
	rc := NewRequestCoalescer(500 * time.Millisecond)
	cc := 0
	h := CoalesceMiddleware(rc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cc++; w.WriteHeader(http.StatusOK); w.Write([]byte("c"))
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	if cc != 1 { t.Errorf("callCount=%d want 1", cc) }
}

// Task 1: graphql
func TestResolveField_NoBackend_C20(t *testing.T) {
	r := &GraphQLResolver{BackendURLs: make(map[string]string)}
	_, err := r.resolveField(context.Background(), graphqlField{Name: "u", Type: "u", Path: "/u"}, "t", "")
	if err == nil { t.Error("expected error") }
}
func TestResolveField_BackendError_C20(t *testing.T) {
	r := &GraphQLResolver{BackendURLs: map[string]string{"u": "http://127.0.0.1:1"}, HTTPClient: &http.Client{Timeout: 100 * time.Millisecond}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := r.resolveField(ctx, graphqlField{Name: "u", Type: "u", Path: "/u"}, "t", "")
	if err == nil { t.Error("expected error") }
}
func TestParseGraphQLFields_Depth_C20(t *testing.T) {
	f := parseGraphQLFields("query { user { name } org { t } }")
	if len(f) < 1 { t.Errorf("fields=%d", len(f)) }
}

// Task 1: JSONLogger
func TestJSONLogger_AllLevels_C20(t *testing.T) {
	var out []string
	mu := sync.Mutex{}
	l := &JSONLogger{writer: func(s string) { mu.Lock(); out = append(out, s); mu.Unlock() }}
	l.Info(LogEntry{Method: "GET", Path: "/", Status: 200})
	l.Warn(LogEntry{Method: "POST", Path: "/", Status: 404})
	l.Error(LogEntry{Method: "PUT", Path: "/", Status: 500})
	if len(out) != 3 { t.Errorf("out=%d", len(out)) }
}

// Task 2: WASM
func TestLoadPlugin_MissingName_C20(t *testing.T) {
	h := NewWasmPluginHost(); defer h.Close(context.Background())
	if err := h.LoadPlugin(context.Background(), WasmPluginConfig{Name: "", WasmPath: "/t.wasm"}); err == nil { t.Error("want error") }
}
func TestLoadPlugin_MissingPath_C20(t *testing.T) {
	h := NewWasmPluginHost(); defer h.Close(context.Background())
	if err := h.LoadPlugin(context.Background(), WasmPluginConfig{Name: "t", WasmPath: ""}); err == nil { t.Error("want error") }
}
func TestLoadPlugin_InvalidPath_C20(t *testing.T) {
	h := NewWasmPluginHost(); defer h.Close(context.Background())
	if err := h.LoadPlugin(context.Background(), WasmPluginConfig{Name: "t", WasmPath: "/nope.wasm"}); err == nil { t.Error("want error") }
}
func TestListPlugins_Empty_C20(t *testing.T) {
	h := NewWasmPluginHost(); defer h.Close(context.Background())
	if len(h.ListPlugins()) != 0 { t.Error("want 0 plugins") }
}
func TestWasmMiddleware_NoPlugins_C20(t *testing.T) {
	h := NewWasmPluginHost(); defer h.Close(context.Background())
	called := false
	mw := WasmMiddleware(h, nil)
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) })).
		ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if !called { t.Error("not called") }
}

// Task 3: Circuit Breaker
func TestCB_Closed_C20(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitConfig())
	if cb.State() != CircuitClosed { t.Error("not closed") }
	if !cb.Allow() { t.Error("should allow") }
}
func TestCB_OpenOnFailures_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 3
	cb := NewCircuitBreaker(cfg)
	for i := 0; i < 3; i++ { cb.Allow(); cb.RecordFailure() }
	if cb.State() != CircuitOpen { t.Error("not open") }
}
func TestCB_RejectWhenOpen_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 1
	cb := NewCircuitBreaker(cfg)
	cb.Allow(); cb.RecordFailure()
	if cb.Allow() { t.Error("should reject when open") }
}
func TestCB_HalfOpenAfterTimeout_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 1; cfg.Timeout = 50 * time.Millisecond
	cb := NewCircuitBreaker(cfg)
	cb.Allow(); cb.RecordFailure()
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() { t.Error("should allow after timeout") }
}
func TestCB_HalfOpenSuccess_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 1; cfg.Timeout = 30 * time.Millisecond; cfg.HalfOpenSuccess = 1
	cb := NewCircuitBreaker(cfg)
	cb.Allow(); cb.RecordFailure(); time.Sleep(40 * time.Millisecond)
	cb.Allow(); cb.RecordSuccess()
	if cb.State() != CircuitClosed { t.Error("should close") }
}
func TestCB_HalfOpenFailureReopens_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 1; cfg.Timeout = 30 * time.Millisecond
	cb := NewCircuitBreaker(cfg)
	cb.Allow(); cb.RecordFailure(); time.Sleep(40 * time.Millisecond)
	cb.Allow(); cb.RecordFailure()
	if cb.State() != CircuitOpen { t.Error("should reopen") }
}
func TestCB_SuccessResets_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 3
	cb := NewCircuitBreaker(cfg)
	cb.Allow(); cb.RecordFailure(); cb.Allow(); cb.RecordFailure()
	cb.Allow(); cb.RecordSuccess()
	cb.Allow(); cb.RecordFailure()
	if cb.State() != CircuitClosed { t.Error("should stay closed") }
}
func TestCB_Stats_C20(t *testing.T) {
	cfg := DefaultCircuitConfig(); cfg.MaxFailures = 2
	cb := NewCircuitBreaker(cfg)
	cb.Allow(); cb.RecordFailure()
	s := cb.Stats()
	if s.Failures != 1 { t.Errorf("failures=%d", s.Failures) }
}
func TestCB_StateString_C20(t *testing.T) {
	if CircuitClosed.String() != "closed" { t.Error() }
	if CircuitOpen.String() != "open" { t.Error() }
	if CircuitHalfOpen.String() != "half-open" { t.Error() }
}
func TestCB_Registry_Get_C20(t *testing.T) {
	r := NewCircuitRegistry(DefaultCircuitConfig())
	a := r.Get("b1"); b := r.Get("b1")
	if a != b { t.Error("same backend should return same breaker") }
}
func TestCB_Registry_AllStats_C20(t *testing.T) {
	r := NewCircuitRegistry(DefaultCircuitConfig())
	r.Get("b1"); r.Get("b2")
	if len(r.AllStats()) < 2 { t.Error("want >=2") }
}

// Task 4: CORS
func TestCORS_Preflight_C20(t *testing.T) {
	h := CORSWithConfig(DefaultCORSConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("should not call next") }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://x.com")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent { t.Errorf("code=%d", rr.Code) }
	if rr.Header().Get("Access-Control-Max-Age") == "" { t.Error("no max-age") }
}
func TestCORS_Wildcard_C20(t *testing.T) {
	h := CORSWithConfig(DefaultCORSConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://anywhere.com")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" { t.Error("not wildcard") }
}
func TestCORS_ExplicitOrigin_C20(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://trusted.com"}, AllowCredentials: true}
	h := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://trusted.com")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://trusted.com" { t.Error("wrong origin") }
	if rr.Header().Get("Access-Control-Allow-Credentials") != "true" { t.Error("no creds") }
	if rr.Header().Get("Vary") != "Origin" { t.Error("no vary") }
}
func TestCORS_DisallowedOrigin_C20(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://trusted.com"}}
	h := CORSWithConfig(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") == "https://evil.com" { t.Error("should not echo") }
}
func TestPerTenantCORS_Preflight_C20(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"*"}})
	h := PerTenantCORS(store, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("should not call") }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://x.com")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent { t.Errorf("code=%d", rr.Code) }
}
func TestPerTenantCORS_Disallowed_C20(t *testing.T) {
	store := NewTenantCORSStore(CORSConfig{AllowedOrigins: []string{"https://allowed.com"}})
	h := PerTenantCORS(store, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden { t.Errorf("code=%d want 403", rr.Code) }
}
func TestContainsWildcard_C20(t *testing.T) {
	if !containsWildcard([]string{"*"}) { t.Error() }
	if containsWildcard([]string{"https://x.com"}) { t.Error() }
}

// Task 5: Retry
func TestBackoffWithJitter_C20(t *testing.T) {
	d0 := backoffWithJitter(0, 100*time.Millisecond, 2*time.Second)
	if d0 < 50*time.Millisecond || d0 > 150*time.Millisecond { t.Errorf("d0=%v", d0) }
	d1 := backoffWithJitter(1, 100*time.Millisecond, 2*time.Second)
	if d1 < 100*time.Millisecond || d1 > 200*time.Millisecond { t.Errorf("d1=%v", d1) }
}
func TestBackoffMaxCap_C20(t *testing.T) {
	d := backoffWithJitter(20, 100*time.Millisecond, 500*time.Millisecond)
	if d > 500*time.Millisecond { t.Errorf("should cap: %v", d) }
}
func TestRetry_503Retryable_C20(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond, RetryableStatus: []int{503}}
	cc := 0
	h := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cc++
		if cc < 3 { w.WriteHeader(503); return }
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if cc != 3 || rr.Code != 200 { t.Errorf("cc=%d code=%d", cc, rr.Code) }
}
func TestRetry_502Retryable_C20(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 2, InitialDelay: 1 * time.Millisecond, RetryableStatus: []int{502}}
	cc := 0
	h := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cc++
		if cc == 1 { w.WriteHeader(502); return }
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if cc != 2 { t.Errorf("cc=%d", cc) }
}
func TestRetry_504Retryable_C20(t *testing.T) {
	cfg := &RetryConfig{MaxAttempts: 2, InitialDelay: 1 * time.Millisecond, RetryableStatus: []int{504}}
	cc := 0
	h := RetryMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cc++
		if cc == 1 { w.WriteHeader(504); return }
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if cc != 2 { t.Errorf("cc=%d", cc) }
}
func TestRetry_200NotRetried_C20(t *testing.T) {
	cc := 0
	h := RetryMiddleware(DefaultRetryConfig())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cc++; w.WriteHeader(200)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if cc != 1 { t.Errorf("cc=%d", cc) }
}
func TestRetry_DefaultConfig_C20(t *testing.T) {
	h := RetryMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 200 { t.Errorf("code=%d", rr.Code) }
}
func TestIsRetryableMethod_C20(t *testing.T) {
	for _, m := range []string{"GET", "HEAD", "OPTIONS"} {
		if !isRetryableMethod(m) { t.Errorf("%s should be retryable", m) }
	}
	for _, m := range []string{"POST", "DELETE", "PUT"} {
		if isRetryableMethod(m) { t.Errorf("%s should not be retryable", m) }
	}
}
