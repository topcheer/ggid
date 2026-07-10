package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ==================== Task 1: Coverage Boost ====================

// --- retry.go: retryResponseWriter ---

func TestRetryResponseWriter_Header_C18(t *testing.T) {
	w := newRetryResponseWriter()
	h := w.Header()
	if h == nil {
		t.Fatal("Header() returned nil")
	}
	h.Set("X-Test", "value")
	if w.header.Get("X-Test") != "value" {
		t.Error("Header() should return internal header map")
	}
}

func TestRetryResponseWriter_Write_C18(t *testing.T) {
	w := newRetryResponseWriter()
	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("n = %d, want 5", n)
	}
	if string(w.body) != "hello" {
		t.Errorf("body = %q", w.body)
	}
}

func TestRetryResponseWriter_WriteHeader_C18(t *testing.T) {
	w := newRetryResponseWriter()
	w.WriteHeader(http.StatusBadGateway)
	if w.status != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", w.status, http.StatusBadGateway)
	}
}

// --- request_logging.go: JSONLogger methods ---

func TestJSONLogger_Info_C18(t *testing.T) {
	var captured string
	logger := &JSONLogger{writer: func(s string) { captured = s }}
	logger.Info(LogEntry{Method: "GET", Path: "/test"})
	if !strings.Contains(captured, "info") {
		t.Errorf("expected 'info' in output: %s", captured)
	}
}

func TestJSONLogger_Warn_C18(t *testing.T) {
	var captured string
	logger := &JSONLogger{writer: func(s string) { captured = s }}
	logger.Warn(LogEntry{Method: "POST", Path: "/warn"})
	if !strings.Contains(captured, "warn") {
		t.Errorf("expected 'warn' in output: %s", captured)
	}
}

func TestJSONLogger_Error_C18(t *testing.T) {
	var captured string
	logger := &JSONLogger{writer: func(s string) { captured = s }}
	logger.Error(LogEntry{Method: "PUT", Path: "/error"})
	if !strings.Contains(captured, "error") {
		t.Errorf("expected 'error' in output: %s", captured)
	}
}

func TestJSONLogger_NilWriter_C18(t *testing.T) {
	logger := &JSONLogger{writer: nil}
	// Should not panic
	logger.Info(LogEntry{Method: "GET", Path: "/"})
	logger.Warn(LogEntry{Method: "GET", Path: "/"})
	logger.Error(LogEntry{Method: "GET", Path: "/"})
}

func TestStatusLogLevel_C18(t *testing.T) {
	if level := statusLogLevel(500); level != LogLevelError {
		t.Errorf("500 -> %v, want Error", level)
	}
	if level := statusLogLevel(404); level != LogLevelWarn {
		t.Errorf("404 -> %v, want Warn", level)
	}
	if level := statusLogLevel(200); level != LogLevelInfo {
		t.Errorf("200 -> %v, want Info", level)
	}
}

// --- metrics.go: MetricsHandler ---

func TestMetricsHandler_C18(t *testing.T) {
	h := MetricsHandler()
	if h == nil {
		t.Fatal("MetricsHandler returned nil")
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// --- gzip.go: WriteHeader ---

func TestGzipResponseWriter_WriteHeader_TextContent_C18(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusCreated)
	}
}

func TestGzipResponseWriter_WriteHeader_BinaryContent_C18(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress binary content")
	}
}

// --- timeout.go: TimeoutMiddleware tests ---

func TestTimeoutMiddleware_Default_C18(t *testing.T) {
	called := false
	h := TimeoutMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("ok"))
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/data", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("handler should be called")
	}
}

func TestTimeoutMiddleware_HealthzSkipped_C18(t *testing.T) {
	h := TimeoutMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d", rr.Code)
	}
}

func TestTimeoutMiddleware_WebSocketSkipped_C18(t *testing.T) {
	h := TimeoutMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusSwitchingProtocols {
		t.Errorf("status = %d", rr.Code)
	}
}

func TestTimeoutMiddleware_Timeout_C18(t *testing.T) {
	cfg := &TimeoutConfig{
		Default:      50 * time.Millisecond,
		RouteConfigs: make(map[string]time.Duration),
	}
	h := TimeoutMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.Write([]byte("should not reach"))
		}
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/slow", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusGatewayTimeout)
	}
}

func TestTimeoutMiddleware_GetTimeoutForRoute_C18(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	if d := cfg.GetTimeoutForRoute("/api/v1/auth/login"); d != 10*time.Second {
		t.Errorf("login timeout = %v, want 10s", d)
	}
	if d := cfg.GetTimeoutForRoute("/api/v1/unknown"); d != 30*time.Second {
		t.Errorf("unknown timeout = %v, want 30s", d)
	}
}

func TestTimeoutMiddleware_RouteConfig_C18(t *testing.T) {
	cfg := &TimeoutConfig{
		Default: 30 * time.Second,
		RouteConfigs: map[string]time.Duration{
			"/api/v1/fast": 100 * time.Millisecond,
		},
	}
	h := TimeoutMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.Write([]byte("ok"))
		}
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/fast", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want %d (504)", rr.Code, http.StatusGatewayTimeout)
	}
}

// ==================== Task 2: Request Timeout (done above) ====================

// ==================== Task 3: Request ID Propagation ====================

func TestRequestIDMiddleware_Generate_C18(t *testing.T) {
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("request ID should be in context")
		}
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	// No X-Request-ID header → should auto-generate
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("response should have X-Request-ID header")
	}
}

func TestRequestIDMiddleware_PreserveExisting_C18(t *testing.T) {
	existing := "test-request-id-12345"
	h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id != existing {
			t.Errorf("context ID = %q, want %q", id, existing)
		}
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", existing)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") != existing {
		t.Errorf("response ID = %q, want %q", rr.Header().Get("X-Request-ID"), existing)
	}
}

func TestWithRequestID_C18(t *testing.T) {
	ctx := WithRequestID(context.Background(), "my-id")
	if id := GetRequestID(ctx); id != "my-id" {
		t.Errorf("GetRequestID = %q", id)
	}
}

func TestGetRequestID_Empty_C18(t *testing.T) {
	if id := GetRequestID(context.Background()); id != "" {
		t.Errorf("empty context should return empty ID, got %q", id)
	}
}

// --- requestid_propagation.go ---

func TestPropagateRequestID_Generate_C18(t *testing.T) {
	called := false
	h := PropagateRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Error("request ID should be in context")
		}
		called = true
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("handler should be called")
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("response should have X-Request-ID")
	}
}

func TestPropagateRequestID_Preserve_C18(t *testing.T) {
	id := "existing-id-abc"
	h := PropagateRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxID := RequestIDFromContext(r.Context())
		if ctxID != id {
			t.Errorf("context ID = %q, want %q", ctxID, id)
		}
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", id)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") != id {
		t.Errorf("response ID = %q, want %q", rr.Header().Get("X-Request-ID"), id)
	}
}

func TestInjectRequestIDHeader_C18(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "ctx-id")
	req := httptest.NewRequest("GET", "/", nil)
	InjectRequestIDHeader(ctx, req)
	if req.Header.Get("X-Request-ID") != "ctx-id" {
		t.Errorf("header = %q", req.Header.Get("X-Request-ID"))
	}
}

func TestInjectRequestIDHeader_NoCtxID_C18(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	InjectRequestIDHeader(context.Background(), req)
	if req.Header.Get("X-Request-ID") == "" {
		t.Error("should auto-generate ID when context has none")
	}
}

func TestInjectRequestIDHeader_PreserveExisting_C18(t *testing.T) {
	ctx := WithRequestID(context.Background(), "ctx-id")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "existing")
	InjectRequestIDHeader(ctx, req)
	if req.Header.Get("X-Request-ID") != "existing" {
		t.Error("should not overwrite existing header")
	}
}

func TestRequestIDFromIncomingMetadata_C18(t *testing.T) {
	// Tested via import of metadata package in requestid_propagation.go
	// Direct function test
	md := newTestMD("x-request-id", "test-id-123")
	if id := RequestIDFromIncomingMetadata(md); id != "test-id-123" {
		t.Errorf("got %q", id)
	}
	md2 := newTestMD("x-request-id")
	if id := RequestIDFromIncomingMetadata(md2); id != "" {
		t.Errorf("empty should return empty, got %q", id)
	}
}

func TestRequestIDToOutgoingContext_C18(t *testing.T) {
	ctx := WithRequestID(context.Background(), "outgoing-id")
	newCtx := RequestIDToOutgoingContext(ctx)
	_ = newCtx // Should not panic
}

func TestRequestIDToOutgoingContext_NoID_C18(t *testing.T) {
	// Should not panic and should return context unchanged
	ctx := context.Background()
	newCtx := RequestIDToOutgoingContext(ctx)
	if newCtx != ctx {
		t.Error("should return same context when no request ID")
	}
}

// ==================== Task 4: Security Headers ====================

func TestSecurityHeadersConfigurable_Default_C18(t *testing.T) {
	h := SecurityHeadersConfigurable(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options: nosniff")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options: DENY")
	}
	if rr.Header().Get("Strict-Transport-Security") == "" {
		t.Error("missing Strict-Transport-Security")
	}
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Error("missing Content-Security-Policy")
	}
	if rr.Header().Get("Referrer-Policy") == "" {
		t.Error("missing Referrer-Policy")
	}
}

func TestSecurityHeadersConfigurable_Disabled_C18(t *testing.T) {
	cfg := &SecurityHeadersConfig{Enabled: false}
	h := SecurityHeadersConfigurable(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("X-Content-Type-Options") != "" {
		t.Error("should not set headers when disabled")
	}
}

func TestSecurityHeadersConfigurable_FrameAllowFrom_C18(t *testing.T) {
	cfg := &SecurityHeadersConfig{
		Enabled:        true,
		FrameDeny:      false,
		FrameAllowFrom: "https://trusted.example.com",
	}
	h := SecurityHeadersConfigurable(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if got := rr.Header().Get("X-Frame-Options"); !strings.Contains(got, "ALLOW-FROM") {
		t.Errorf("X-Frame-Options = %q", got)
	}
}

func TestSecurityHeadersConfigurable_PerTenantOverride_C18(t *testing.T) {
	base := DefaultSecurityHeadersConfig()
	base.PerTenantOverrides = map[string]*SecurityHeadersConfig{
		"tenant-special": {
			Enabled: true,
			CSP:     "default-src 'none'",
		},
	}
	h := SecurityHeadersConfigurable(base)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-special")
	h.ServeHTTP(rr, req)
	if got := rr.Header().Get("Content-Security-Policy"); got != "default-src 'none'" {
		t.Errorf("CSP = %q, want tenant override", got)
	}
}

func TestSecurityHeadersConfigurable_NoHSTS_C18(t *testing.T) {
	cfg := &SecurityHeadersConfig{
		Enabled:            true,
		ContentTypeNosniff: true,
		HSTSMaxAge:         0, // No HSTS
	}
	h := SecurityHeadersConfigurable(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Error("should not set HSTS when maxAge=0")
	}
}

func TestMergeSecurityHeaders_C18(t *testing.T) {
	base := DefaultSecurityHeadersConfig()
	override := &SecurityHeadersConfig{CSP: "custom-csp"}
	merged := mergeSecurityHeaders(base, override)
	if merged.CSP != "custom-csp" {
		t.Errorf("CSP = %q", merged.CSP)
	}
	if !merged.ContentTypeNosniff {
		t.Error("should preserve base nosniff")
	}
}

func TestMergeSecurityHeaders_NilOverride_C18(t *testing.T) {
	base := DefaultSecurityHeadersConfig()
	merged := mergeSecurityHeaders(base, nil)
	if merged != base {
		t.Error("nil override should return base")
	}
}

func TestMergeSecurityHeaders_NilBase_C18(t *testing.T) {
	override := &SecurityHeadersConfig{CSP: "test"}
	merged := mergeSecurityHeaders(nil, override)
	if merged != override {
		t.Error("nil base should return override")
	}
}

// ==================== Task 5: gRPC Interceptor ====================

func TestGRPCUnaryInterceptor_Basic_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{LogRequests: true}
	interceptor := GRPCUnaryInterceptor(cfg)
	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	_, err := interceptor(context.Background(), nil, info, handler)
	if !called {
		t.Error("handler should be called")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCUnaryInterceptor_TenantInjection_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{}
	interceptor := GRPCUnaryInterceptor(cfg)
	md := newTestMD("x-tenant-id", "tenant-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	handler := func(ctx context.Context, req any) (any, error) {
		tenantID := TenantFromGRPCContext(ctx)
		if tenantID != "tenant-123" {
			t.Errorf("tenant = %q, want 'tenant-123'", tenantID)
		}
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	interceptor(ctx, nil, info, handler)
}

func TestGRPCUnaryInterceptor_AuthRequired_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "secret"}
	interceptor := GRPCUnaryInterceptor(cfg)
	// Use empty incoming metadata (present but no authorization)
	md := metadata.MD{}
	ctx := metadata.NewIncomingContext(context.Background(), md)
	handler := func(ctx context.Context, req any) (any, error) {
		t.Error("handler should not be called without auth")
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	_, err := interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Error("expected error for missing auth")
	}
}

func TestGRPCUnaryInterceptor_AuthInvalidScheme_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "secret"}
	interceptor := GRPCUnaryInterceptor(cfg)
	md := newTestMD("authorization", "Basic abc123")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	handler := func(ctx context.Context, req any) (any, error) {
		t.Error("handler should not be called with invalid auth scheme")
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	_, err := interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Error("expected error for invalid auth scheme")
	}
}

func TestGRPCUnaryInterceptor_AuthValid_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "secret"}
	interceptor := GRPCUnaryInterceptor(cfg)
	md := newTestMD("authorization", "Bearer valid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		userID := UserFromGRPCContext(ctx)
		if userID != "valid-token" {
			t.Errorf("user = %q", userID)
		}
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	interceptor(ctx, nil, info, handler)
	if !called {
		t.Error("handler should be called with valid auth")
	}
}

func TestGRPCStreamInterceptor_Basic_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{LogRequests: true}
	interceptor := GRPCStreamInterceptor(cfg)
	called := false
	handler := func(srv any, stream grpc.ServerStream) error {
		called = true
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}
	err := interceptor(nil, &mockServerStream{ctx: context.Background()}, info, handler)
	if !called {
		t.Error("handler should be called")
	}
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGRPCStreamInterceptor_AuthRequired_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "secret"}
	interceptor := GRPCStreamInterceptor(cfg)
	md := metadata.MD{}
	ctx := metadata.NewIncomingContext(context.Background(), md)
	handler := func(srv any, stream grpc.ServerStream) error {
		t.Error("handler should not be called")
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}
	err := interceptor(nil, &mockServerStream{ctx: ctx}, info, handler)
	if err == nil {
		t.Error("expected error for missing auth")
	}
}

func TestGRPCStreamInterceptor_TenantInjection_C18(t *testing.T) {
	cfg := &GRPCInterceptorConfig{}
	interceptor := GRPCStreamInterceptor(cfg)
	md := newTestMD("x-tenant-id", "stream-tenant")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	handler := func(srv any, stream grpc.ServerStream) error {
		tid := TenantFromGRPCContext(stream.Context())
		if tid != "stream-tenant" {
			t.Errorf("tenant = %q", tid)
		}
		return nil
	}
	info := &grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"}
	interceptor(nil, &mockServerStream{ctx: ctx}, info, handler)
}

func TestTenantFromGRPCContext_Empty_C18(t *testing.T) {
	if tid := TenantFromGRPCContext(context.Background()); tid != "" {
		t.Errorf("expected empty, got %q", tid)
	}
}

func TestUserFromGRPCContext_Empty_C18(t *testing.T) {
	if uid := UserFromGRPCContext(context.Background()); uid != "" {
		t.Errorf("expected empty, got %q", uid)
	}
}

// --- helpers ---

// newTestMD creates a gRPC metadata.MD for testing.
func newTestMD(key string, values ...string) metadata.MD {
	if len(values) == 0 {
		return metadata.MD{key: []string{}}
	}
	return metadata.Pairs(key, values[0])
}

// mockServerStream implements grpc.ServerStream for testing.
type mockServerStream struct {
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context              { return m.ctx }
func (m *mockServerStream) SetHeader(metadata.MD) error           { return nil }
func (m *mockServerStream) SendHeader(metadata.MD) error          { return nil }
func (m *mockServerStream) SetTrailer(metadata.MD)                {}
func (m *mockServerStream) SendMsg(any) error                     { return nil }
func (m *mockServerStream) RecvMsg(any) error                     { return nil }
