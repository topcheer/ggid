package middleware

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCUnaryRecovery is a gRPC unary server interceptor that recovers from panics,
// preventing a single bad request from crashing the entire gRPC server.
func GRPCUnaryRecovery(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("gRPC panic recovered", "method", info.FullMethod, "panic", r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

// GRPCStreamRecovery is a gRPC stream server interceptor that recovers from panics.
func GRPCStreamRecovery(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("gRPC stream panic recovered", "method", info.FullMethod, "panic", r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(srv, ss)
}

// GRPCRecoveryOpts returns grpc.ServerOption interceptors for panic recovery.
// Usage: grpc.NewServer(middleware.GRPCRecoveryOpts()...)
func GRPCRecoveryOpts() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.UnaryInterceptor(GRPCUnaryRecovery),
		grpc.StreamInterceptor(GRPCStreamRecovery),
	}
}
