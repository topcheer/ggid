package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// gRPC metadata key for request ID propagation.
const (
	GRPCRequestIDKey  = "x-request-id"
	GRPCTraceParentKey = "traceparent"
)

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

// ContextWithRequestID returns a new context with the request ID set.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// --- gRPC client interceptor (outgoing propagation) ---

// NewGRPCRequestIDInterceptor returns a grpc.UnaryClientInterceptor that
// injects the X-Request-ID from the context into outgoing gRPC metadata.
//
// Usage:
//
//	conn, _ := grpc.Dial(addr, grpc.WithUnaryInterceptor(
//	    middleware.NewGRPCRequestIDInterceptor(),
//	))
func NewGRPCRequestIDInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string, req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		requestID := RequestIDFromContext(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx = metadata.AppendToOutgoingContext(ctx, GRPCRequestIDKey, requestID)

		// Propagate W3C traceparent from context to downstream gRPC metadata.
		if tc, ok := ctx.Value(traceContextKey{}).(*TraceContext); ok && tc != nil {
			tp := formatTraceparent(tc.TraceID, tc.SpanID, true)
			ctx = metadata.AppendToOutgoingContext(ctx, GRPCTraceParentKey, tp)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// RequestIDToOutgoingContext injects the request ID from an HTTP request's
// context into a gRPC-compatible context (with metadata) for making downstream
// gRPC calls.
func RequestIDToOutgoingContext(ctx context.Context) context.Context {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, GRPCRequestIDKey, requestID)
}

// RequestIDFromIncomingMetadata extracts the request ID from incoming gRPC
// metadata (server side). Returns empty string if not present.
func RequestIDFromIncomingMetadata(md metadata.MD) string {
	if vals := md.Get(GRPCRequestIDKey); len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// --- HTTP header propagation (for reverse proxy) ---

// PropagateRequestID wraps an http.Handler to ensure the X-Request-ID header
// is always set: either from the incoming request header, or auto-generated.
// It stores the ID in context for downstream gRPC/HTTP propagation.
func PropagateRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// InjectRequestIDHeader sets the X-Request-ID header on an outgoing HTTP
// request based on the current context's request ID. If no request ID exists
// in context, a new one is generated.
func InjectRequestIDHeader(ctx context.Context, req *http.Request) {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = uuid.New().String()
	}
	if req.Header.Get("X-Request-ID") == "" {
		req.Header.Set("X-Request-ID", requestID)
	}
}
