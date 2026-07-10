package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- wasm_plugin.go ---

func TestFlattenHeaders_V2(t *testing.T) {
	h := http.Header{}
	h.Set("X-Custom", "val1")
	h.Add("X-Multi", "first")
	h.Add("X-Multi", "second")
	result := flattenHeaders(h)
	if result["X-Custom"] != "val1" {
		t.Errorf("expected val1, got %s", result["X-Custom"])
	}
	if result["X-Multi"] != "first" {
		t.Errorf("expected first, got %s", result["X-Multi"])
	}
	empty := flattenHeaders(http.Header{})
	if len(empty) != 0 {
		t.Errorf("expected empty map, got %d items", len(empty))
	}
}

func TestWasmMiddleware_WithHost_EmptyPlugins_V2(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())
	mw := WasmMiddleware(host, []string{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWasmMiddleware_WithHost_NonExistentPlugin_V2(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())
	mw := WasmMiddleware(host, []string{"ghost"})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWasmPluginHost_Close_DoubleClose_V2(t *testing.T) {
	host := NewWasmPluginHost()
	ctx := context.Background()
	_ = host.Close(ctx)
	_ = host.Close(ctx)
}

func TestWasmPluginHost_Execute_EmptyContext_V2(t *testing.T) {
	host := NewWasmPluginHost()
	defer host.Close(context.Background())
	_, err := host.Execute(context.Background(), "missing", PhaseRequest, PluginContext{
		Method: "POST",
		Path:   "/api/v1/data",
	})
	if err == nil {
		t.Error("expected error for missing plugin")
	}
}

// --- grpc.go ---

func TestGRPCProxy_AddAndGetBackend_V2(t *testing.T) {
	p := NewGRPCProxy(GRPCProxyConfig{})
	p.AddBackend("svc1", "localhost:9090")
	bp := p.GetBackend("svc1")
	if bp != "localhost:9090" {
		t.Errorf("expected localhost:9090, got %s", bp)
	}
}

func TestGRPCProxy_GetBackend_NotFound_V2(t *testing.T) {
	p := NewGRPCProxy(GRPCProxyConfig{})
	bp := p.GetBackend("missing")
	if bp != "" {
		t.Errorf("expected empty string, got %s", bp)
	}
}

func TestGRPCProxy_AddBackend_Overwrite_V2(t *testing.T) {
	p := NewGRPCProxy(GRPCProxyConfig{})
	p.AddBackend("svc1", "localhost:9090")
	p.AddBackend("svc1", "localhost:9091")
	bp := p.GetBackend("svc1")
	if bp != "localhost:9091" {
		t.Errorf("expected overwritten, got %s", bp)
	}
}

// --- compress.go ---

func TestCompressWriter_Write_AfterHeader_V2(t *testing.T) {
	w := httptest.NewRecorder()
	cw := &compressWriter{
		ResponseWriter: w,
		wroteHeader:     true,
		skip:            true,
	}
	cw.WriteHeader(http.StatusOK)
	n, err := cw.Write([]byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if n != len(`{"ok":true}`) {
		t.Errorf("expected %d bytes, got %d", len(`{"ok":true}`), n)
	}
}

// --- coalesce.go ---

func TestCoalesceRecorder_Header_WithResponseWriter_V2(t *testing.T) {
	rw := httptest.NewRecorder()
	rw.Header().Set("X-Inherited", "yes")
	r := &coalesceRecorder{
		ResponseWriter: rw,
		status:         200,
	}
	h := r.Header()
	if h.Get("X-Inherited") != "yes" {
		t.Error("expected inherited header")
	}
}

// --- graphql.go ---

func TestSubstituteVariables_WithVars_V2(t *testing.T) {
	field := graphqlField{Name: "user", Type: "user", Path: "/api/v1/users/$id"}
	result := substituteVariables(field, map[string]any{"id": "123"})
	if result.Path != "/api/v1/users/123" {
		t.Errorf("expected substituted path, got %s", result.Path)
	}
}

func TestSubstituteVariables_NoVars_V2(t *testing.T) {
	field := graphqlField{Name: "user", Type: "user", Path: "/api/v1/users"}
	result := substituteVariables(field, nil)
	if result.Path != "/api/v1/users" {
		t.Errorf("expected same path, got %s", result.Path)
	}
}

// --- ratelimit.go ---

func TestRateLimiter_Middleware_AllowUnderLimit_V2(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	rl := NewRateLimiter(cfg)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 under limit, got %d", w.Code)
	}
}

// --- session.go ---

func TestSessionManager_IsSessionRevoked_NilRedis_V3(t *testing.T) {
	sm := NewSessionManager(nil)
	revoked := sm.IsSessionRevoked(context.Background(), "test-session")
	if revoked {
		t.Error("expected false for nil redis")
	}
}

func TestSessionManager_MarkSessionRevoked_NilRedis_V3(t *testing.T) {
	sm := NewSessionManager(nil)
	_ = sm.MarkSessionRevoked(context.Background(), "test-session")
}

func TestSessionManager_Middleware_NilRedis_V2(t *testing.T) {
	sm := NewSessionManager(nil)
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- health_score.go ---

func TestHealthScore_IsHealthy_NewBackend_V2(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)
	if !hs.IsHealthy("svc1", 0.95) {
		t.Error("expected new backend to be healthy")
	}
}

// --- shadow.go ---

func TestShadowTrafficMirror_SetPercentage100_V2(t *testing.T) {
	stm := NewShadowTrafficMirror(ShadowTrafficConfig{
		ShadowBackend: "http://shadow:8080",
		Percentage:    50,
	})
	stm.SetPercentage(100)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if !stm.shouldMirror(req) {
		t.Error("expected shouldMirror=true at 100%")
	}
}
