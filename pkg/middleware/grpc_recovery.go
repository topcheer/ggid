package middleware

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
// IMPORTANT: Uses ChainUnaryInterceptor/ChainStreamInterceptor (not UnaryInterceptor/
// StreamInterceptor) so that additional interceptors appended via SecureGRPCOptsWithPrev
// are composed, not overridden. In grpc-go, grpc.UnaryInterceptor and
// grpc.ChainUnaryInterceptor are mutually exclusive server options — the last one
// in the variadic list wins, silently dropping the other.
func GRPCRecoveryOpts() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(GRPCUnaryRecovery),
		grpc.ChainStreamInterceptor(GRPCStreamRecovery),
	}
}

// GRPCInternalAuthStream returns a stream interceptor that enforces internal
// auth (HMAC signature in metadata) on every streaming gRPC call.
func GRPCInternalAuthStream(cfg InternalAuthConfig) grpc.StreamServerInterceptor {
	if cfg.ReplayWindow <= 0 {
		cfg.ReplayWindow = defaultReplayWindow
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if len(cfg.Secret) == 0 {
			return handler(srv, ss)
		}
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Errorf(codes.PermissionDenied, "missing internal auth metadata")
		}
		svc := firstMeta(md, InternalAuthHeaderService)
		tsStr := firstMeta(md, InternalAuthHeaderTimestamp)
		sigHex := firstMeta(md, InternalAuthHeaderSignature)
		reqID := firstMeta(md, "x-request-id")
		if svc == "" || tsStr == "" || sigHex == "" {
			return status.Errorf(codes.PermissionDenied, "missing internal auth metadata")
		}
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return status.Errorf(codes.PermissionDenied, "invalid internal auth timestamp")
		}
		if math.Abs(float64(time.Now().Unix()-ts)) > float64(cfg.ReplayWindow) {
			return status.Errorf(codes.PermissionDenied, "internal auth timestamp outside replay window")
		}
		payload := svc + "|" + tsStr + "|" + reqID + "|" + info.FullMethod
		if !verifyHMAC(cfg.Secret, payload, sigHex) {
			prevOK := len(cfg.PrevSecret) > 0 && verifyHMAC(cfg.PrevSecret, payload, sigHex)
			if !prevOK {
				return status.Errorf(codes.PermissionDenied, "invalid internal auth signature")
			}
		}
		return handler(srv, ss)
	}
}

// GRPCInternalAuthUnary returns a unary interceptor that enforces internal
// auth (HMAC signature in metadata) on every gRPC call. Same scheme as the
// HTTP InternalAuth middleware: x-internal-service | x-internal-timestamp |
// x-request-id signed with the shared GGID_INTERNAL_SECRET. Fail-closed:
// missing/invalid signature → PermissionDenied.
func GRPCInternalAuthUnary(cfg InternalAuthConfig) grpc.UnaryServerInterceptor {
	if cfg.ReplayWindow <= 0 {
		cfg.ReplayWindow = defaultReplayWindow
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if len(cfg.Secret) == 0 {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "missing internal auth metadata")
		}
		svc := firstMeta(md, InternalAuthHeaderService)
		tsStr := firstMeta(md, InternalAuthHeaderTimestamp)
		sigHex := firstMeta(md, InternalAuthHeaderSignature)
		reqID := firstMeta(md, "x-request-id")
		if svc == "" || tsStr == "" || sigHex == "" {
			return nil, status.Errorf(codes.PermissionDenied, "missing internal auth metadata")
		}
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "invalid internal auth timestamp")
		}
		if math.Abs(float64(time.Now().Unix()-ts)) > float64(cfg.ReplayWindow) {
			return nil, status.Errorf(codes.PermissionDenied, "internal auth timestamp outside replay window")
		}
		payload := svc + "|" + tsStr + "|" + reqID + "|" + info.FullMethod
		if !verifyHMAC(cfg.Secret, payload, sigHex) {
			prevOK := len(cfg.PrevSecret) > 0 && verifyHMAC(cfg.PrevSecret, payload, sigHex)
			if !prevOK {
				return nil, status.Errorf(codes.PermissionDenied, "invalid internal auth signature")
			}
		}
		return handler(ctx, req)
	}
}

func firstMeta(md metadata.MD, key string) string {
	for _, k := range md.Get(key) {
		if k != "" {
			return k
		}
	}
	return ""
}

// SecureGRPCOpts combines panic recovery with optional internal auth.
// authSecret may be nil to keep recovery-only (dev/test).
// authPrevSecret enables key-rotation: signatures valid under either
// current or previous secret are accepted.
func SecureGRPCOpts(authSecret []byte) []grpc.ServerOption {
	return SecureGRPCOptsWithPrev(authSecret, nil)
}

// SecureGRPCOptsWithPrev is SecureGRPCOpts with prev-secret rotation support.
// Appends internal auth as an additional ChainUnaryInterceptor, composing with
// the recovery interceptor from GRPCRecoveryOpts(). Also adds stream
// interceptor for internal auth on streaming RPCs.
func SecureGRPCOptsWithPrev(authSecret, authPrevSecret []byte) []grpc.ServerOption {
	opts := GRPCRecoveryOpts()
	if len(authSecret) > 0 {
		authCfg := InternalAuthConfig{Secret: authSecret, PrevSecret: authPrevSecret}
		opts = append(opts,
			grpc.ChainUnaryInterceptor(GRPCInternalAuthUnary(authCfg)),
			grpc.ChainStreamInterceptor(GRPCInternalAuthStream(authCfg)),
		)
	}
	return opts
}
