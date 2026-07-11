package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// --- GRPCHTTPHandler coverage tests ---

func TestGRPCHTTPHandler_NonGRPCRequest(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	handler := proxy.GRPCHTTPHandler(next)
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected next handler to be called for non-gRPC request")
	}
}

func TestGRPCHTTPHandler_GRPCNoBackend(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())
	handler := proxy.GRPCHTTPHandler(nil)

	req := httptest.NewRequest("POST", "/package.ServiceName/Method", nil)
	req.Header.Set("Content-Type", "application/grpc")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for unknown backend, got %d", rr.Code)
	}
}

// (TestGRPCProxy_Stats, TestGRPCProxy_GetBackend already exist in coverage_boost_test.go)

// --- GRPCStreamInterceptor coverage tests ---

func TestGRPCStreamInterceptor_TenantInjection(t *testing.T) {
	cfg := &GRPCInterceptorConfig{LogRequests: true}
	interceptor := GRPCStreamInterceptor(cfg)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("x-tenant-id", "tenant-123"))

	ss := &mockServerStream{ctx: ctx}
	handlerCalled := false
	handler := func(srv any, stream grpc.ServerStream) error {
		handlerCalled = true
		sctx := stream.Context()
		tenant := sctx.Value(grpcTenantCtxKey)
		if tenant != "tenant-123" {
			t.Errorf("expected tenant-123, got %v", tenant)
		}
		return nil
	}

	err := interceptor(nil, ss, nil, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler not called")
	}
}

func TestGRPCStreamInterceptor_NoMetadata(t *testing.T) {
	cfg := &GRPCInterceptorConfig{}
	interceptor := GRPCStreamInterceptor(cfg)

	ss := &mockServerStream{ctx: context.Background()}
	handler := func(srv any, stream grpc.ServerStream) error {
		return nil
	}

	err := interceptor(nil, ss, nil, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGRPCStreamInterceptor_JWTAuth_Missing(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "test-secret"}
	interceptor := GRPCStreamInterceptor(cfg)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("x-tenant-id", "t1"))
	ss := &mockServerStream{ctx: ctx}
	handler := func(srv any, stream grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	}

	err := interceptor(nil, ss, nil, handler)
	if err == nil {
		t.Fatal("expected error for missing auth")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestGRPCStreamInterceptor_JWTAuth_BadScheme(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "test-secret"}
	interceptor := GRPCStreamInterceptor(cfg)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Basic abc123"))
	ss := &mockServerStream{ctx: ctx}
	handler := func(srv any, stream grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	}

	err := interceptor(nil, ss, nil, handler)
	if err == nil {
		t.Fatal("expected error for bad auth scheme")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", status.Code(err))
	}
}

func TestGRPCStreamInterceptor_JWTAuth_Valid(t *testing.T) {
	cfg := &GRPCInterceptorConfig{JWTSecret: "test-secret"}
	interceptor := GRPCStreamInterceptor(cfg)

	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer valid-token"))
	ss := &mockServerStream{ctx: ctx}
	handlerCalled := false
	handler := func(srv any, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	err := interceptor(nil, ss, nil, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler should be called with valid Bearer token")
	}
}

func TestWrappedServerStream_Context(t *testing.T) {
	ctx := context.WithValue(context.Background(), grpcTenantCtxKey, "test-tenant")
	w := &wrappedServerStream{ctx: ctx}
	got := w.Context()
	if got.Value(grpcTenantCtxKey) != "test-tenant" {
		t.Error("expected tenant in context")
	}
}

// mockServerStream already defined in coverage_sprint18_test.go.
