package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// === GRPC HTTP Handler tests ===

func TestGRPCHTTPHandler_NonGRPC_V2(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())

	called := false
	handler := proxy.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Non-gRPC request should pass through to next handler")
	}
}

func TestGRPCHTTPHandler_NonGRPC_NilNext_V2(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())

	handler := proxy.GRPCHTTPHandler(nil)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Should not panic, just return
}

func TestGRPCHTTPHandler_GRPC_NoBackend_V2(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())

	handler := proxy.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call next for gRPC")
	}))

	req := httptest.NewRequest("POST", "/myapp.UserService/GetUser", nil)
	req.Header.Set("Content-Type", "application/grpc")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("No backend: want 502, got %d", rr.Code)
	}
}

func TestIsGRPCRequest_V2(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/grpc", true},
		{"application/grpc+proto", true},
		{"application/grpc-web", true},
		{"application/json", false},
		{"", false},
		{"text/plain", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/svc/method", nil)
		if tt.ct != "" {
			req.Header.Set("Content-Type", tt.ct)
		}
		if got := isGRPCRequest(req); got != tt.want {
			t.Errorf("isGRPCRequest(ct=%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}

func TestExtractGRPCService_V2(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/myapp.UserService/GetUser", "myapp.UserService"},
		{"/myapp.UserService/CreateUser", "myapp.UserService"},
		{"/svc/Method", "svc"},
		{"noslash", "noslash"},
		{"/", ""},
	}

	for _, tt := range tests {
		if got := extractGRPCService(tt.path); got != tt.want {
			t.Errorf("extractGRPCService(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestGRPCProxy_GetBackend_V2(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())
	proxy.AddBackend("myapp.UserService", "localhost:9090")

	addr := proxy.GetBackend("myapp.UserService")
	if addr != "localhost:9090" {
		t.Errorf("GetBackend: want 'localhost:9090', got '%s'", addr)
	}

	// Unknown service returns empty
	addr = proxy.GetBackend("unknown.Service")
	if addr != "" {
		t.Errorf("Unknown service: want '', got '%s'", addr)
	}
}

// === gRPC Request ID Interceptor ===

func TestGRPCRequestIDInterceptor_WithID(t *testing.T) {
	interceptor := NewGRPCRequestIDInterceptor()

	ctx := ContextWithRequestID(context.Background(), "intercept-req-id")

	var capturedMD metadata.MD
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMD = md.Copy()
		}
		return nil
	}

	// Call the interceptor with proper types
	// We need a *grpc.ClientConn but we can pass nil since the invoker ignores it
	err := interceptor(ctx, "/pkg.Svc/Method", nil, nil, nil, invoker)
	if err != nil {
		t.Errorf("Interceptor error: %v", err)
	}

	if vals := capturedMD.Get(GRPCRequestIDKey); len(vals) == 0 || vals[0] != "intercept-req-id" {
		t.Errorf("Expected request ID in metadata: got %v", capturedMD.Get(GRPCRequestIDKey))
	}
}

func TestGRPCRequestIDInterceptor_WithoutID(t *testing.T) {
	interceptor := NewGRPCRequestIDInterceptor()

	ctx := context.Background()

	var capturedMD metadata.MD
	invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMD = md.Copy()
		}
		return nil
	}

	err := interceptor(ctx, "/pkg.Svc/Method", nil, nil, nil, invoker)
	if err != nil {
		t.Errorf("Interceptor error: %v", err)
	}

	// Should have auto-generated a request ID
	if vals := capturedMD.Get(GRPCRequestIDKey); len(vals) == 0 {
		t.Error("Should auto-generate request ID")
	}
}
