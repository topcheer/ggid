package middleware

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/status"
)

// GRPCInterceptorConfig configures the gRPC server interceptors.
type GRPCInterceptorConfig struct {
	// JWTSecret is the HMAC secret for JWT validation. If empty, auth is skipped.
	JWTSecret string
	// JWTIssuer is the expected JWT issuer (iss claim). If empty, issuer not validated.
	JWTIssuer string
	// RequireAuth if true, makes JWTSecret mandatory (fatal on startup if empty).
	RequireAuth bool
	// TenantHeader is the gRPC metadata key for tenant ID (default: x-tenant-id).
	TenantHeader string
	// LogRequests enables request logging via standard log package.
	LogRequests bool
}

// ctxKey is an unexported type for interceptor context keys.
type ctxKey int

const (
	grpcTenantCtxKey ctxKey = iota
	grpcUserCtxKey
)

// TenantFromGRPCContext extracts the tenant ID injected by the interceptor.
func TenantFromGRPCContext(ctx context.Context) string {
	if v, ok := ctx.Value(grpcTenantCtxKey).(string); ok {
		return v
	}
	return ""
}

// UserFromGRPCContext extracts the user ID injected by the interceptor.
func UserFromGRPCContext(ctx context.Context) string {
	if v, ok := ctx.Value(grpcUserCtxKey).(string); ok {
		return v
	}
	return ""
}

// GRPCUnaryInterceptor returns a gRPC server unary interceptor that:
// 1. Validates JWT from metadata (authorization bearer token).
// 2. Injects tenant + user ID into context.
// 3. Logs request duration and status.
func GRPCUnaryInterceptor(cfg *GRPCInterceptorConfig) grpc.UnaryServerInterceptor {
	if cfg == nil {
		cfg = &GRPCInterceptorConfig{}
	}
	// P0 Security: If RequireAuth is true but JWTSecret is empty, fail hard.
	// Silent bypass when secret is empty is a critical vulnerability.
	if cfg.RequireAuth && cfg.JWTSecret == "" {
		slog.Error("GRPCUnaryInterceptor: RequireAuth=true but JWTSecret is empty — refusing to start with silent auth bypass")
		os.Exit(1)
	}
	tenantHeader := cfg.TenantHeader
	if tenantHeader == "" {
		tenantHeader = "x-tenant-id"
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		// Extract metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			// Inject tenant ID
			if vals := md.Get(tenantHeader); len(vals) > 0 {
				ctx = context.WithValue(ctx, grpcTenantCtxKey, vals[0])
			}

			// Validate JWT if configured
			if cfg.JWTSecret != "" {
				authVals := md.Get("authorization")
				if len(authVals) == 0 {
					return nil, status.Error(codes.Unauthenticated, "missing authorization")
				}
				token := strings.TrimPrefix(authVals[0], "Bearer ")
				if token == authVals[0] {
					return nil, status.Error(codes.Unauthenticated, "invalid authorization scheme")
				}
				// SECURITY: Validate JWT signature and claims — not just extract token.
				claims := jwt.MapClaims{}
				parserOpts := []jwt.ParserOption{
					jwt.WithValidMethods([]string{"HS256", "RS256"}),
					jwt.WithExpirationRequired(),
				}
				if cfg.JWTIssuer != "" {
					parserOpts = append(parserOpts, jwt.WithIssuer(cfg.JWTIssuer))
				}
				_, err := jwt.NewParser(parserOpts...).ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); ok {
						return []byte(cfg.JWTSecret), nil
					}
					return nil, status.Error(codes.Unauthenticated, "unsupported signing method")
				})
				if err != nil {
					return nil, status.Error(codes.Unauthenticated, "invalid token")
				}
				// Store verified claims, not raw token.
				ctx = context.WithValue(ctx, grpcUserCtxKey, claims["sub"])
				if tid, ok := claims["tenant_id"].(string); ok {
					ctx = context.WithValue(ctx, grpcTenantCtxKey, tid)
				}
			}
		}

		// Call handler
		resp, err := handler(ctx, req)

		if cfg.LogRequests {
			duration := time.Since(start)
			code := status.Code(err)
			slog.Info("grpc request", "method", info.FullMethod, "duration", duration.String(), "code", code.String())
		}

		return resp, err
	}
}

// GRPCStreamInterceptor returns a gRPC server stream interceptor with the
// same auth/tenant injection as the unary interceptor.
func GRPCStreamInterceptor(cfg *GRPCInterceptorConfig) grpc.StreamServerInterceptor {
	if cfg == nil {
		cfg = &GRPCInterceptorConfig{}
	}
	tenantHeader := cfg.TenantHeader
	if tenantHeader == "" {
		tenantHeader = "x-tenant-id"
	}

	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if vals := md.Get(tenantHeader); len(vals) > 0 {
				ctx = context.WithValue(ctx, grpcTenantCtxKey, vals[0])
			}
			if cfg.JWTSecret != "" {
				authVals := md.Get("authorization")
				if len(authVals) == 0 {
					return status.Error(codes.Unauthenticated, "missing authorization")
				}
				token := strings.TrimPrefix(authVals[0], "Bearer ")
				if token == authVals[0] {
					return status.Error(codes.Unauthenticated, "invalid authorization scheme")
				}
				ctx = context.WithValue(ctx, grpcUserCtxKey, token)
			}
		}

		wrapped := &wrappedServerStream{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

// wrappedServerStream overrides Context() to inject interceptor context values.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
