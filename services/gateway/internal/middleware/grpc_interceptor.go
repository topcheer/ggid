package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCInterceptorConfig configures the gRPC server interceptors.
type GRPCInterceptorConfig struct {
	// JWTSecret is the HMAC secret for JWT validation. If empty, auth is skipped.
	JWTSecret string
	// RSAPublicKey is an optional RSA public key (PEM) for RS256 token validation.
	// If set, RS256 tokens are validated against this key in addition to HMAC.
	RSAPublicKey string
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

// authRequired returns true if any JWT validation key is configured.
func (cfg *GRPCInterceptorConfig) authRequired() bool {
	return cfg.JWTSecret != "" || cfg.RSAPublicKey != ""
}

// validSigningMethods returns the allowed JWT signing methods based on config.
func (cfg *GRPCInterceptorConfig) validSigningMethods() []string {
	methods := []string{}
	if cfg.JWTSecret != "" {
		methods = append(methods, "HS256")
	}
	if cfg.RSAPublicKey != "" {
		methods = append(methods, "RS256")
	}
	return methods
}

// jwtKeyFunc returns the verification key for a parsed JWT token.
func (cfg *GRPCInterceptorConfig) jwtKeyFunc(t *jwt.Token) (any, error) {
	if _, ok := t.Method.(*jwt.SigningMethodRSA); ok {
		if cfg.RSAPublicKey == "" {
			return nil, status.Error(codes.Unauthenticated, "RSA public key not configured")
		}
		block, _ := pem.Decode([]byte(cfg.RSAPublicKey))
		if block == nil {
			return nil, status.Error(codes.Unauthenticated, "invalid RSA public key PEM")
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "cannot parse RSA public key")
		}
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "RSA public key is not RSA")
		}
		return rsaPub, nil
	}
	if _, ok := t.Method.(*jwt.SigningMethodHMAC); ok {
		if cfg.JWTSecret == "" {
			return nil, status.Error(codes.Unauthenticated, "HMAC secret not configured")
		}
		return []byte(cfg.JWTSecret), nil
	}
	return nil, status.Error(codes.Unauthenticated, "unsupported signing method")
}

// validateGRPCJWT parses and validates the bearer token, returning claims on success.
func validateGRPCJWT(cfg *GRPCInterceptorConfig, token string) (jwt.MapClaims, error) {
	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods(cfg.validSigningMethods()),
		jwt.WithExpirationRequired(),
	}
	if cfg.JWTIssuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(cfg.JWTIssuer))
	}
	claims := jwt.MapClaims{}
	_, err := jwt.NewParser(parserOpts...).ParseWithClaims(token, claims, cfg.jwtKeyFunc)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("invalid token: %v", err))
	}
	return claims, nil
}

// extractBearerToken extracts the token from the "authorization" metadata entry.
func extractBearerToken(md metadata.MD) (string, error) {
	authVals := md.Get("authorization")
	if len(authVals) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization")
	}
	token := strings.TrimPrefix(authVals[0], "Bearer ")
	if token == authVals[0] {
		return "", status.Error(codes.Unauthenticated, "invalid authorization scheme")
	}
	return token, nil
}

// injectClaimsIntoContext sets user and tenant context values from JWT claims.
func injectClaimsIntoContext(ctx context.Context, claims jwt.MapClaims) context.Context {
	if sub, ok := claims["sub"].(string); ok {
		ctx = context.WithValue(ctx, grpcUserCtxKey, sub)
	}
	if tid, ok := claims["tenant_id"].(string); ok && tid != "" {
		ctx = context.WithValue(ctx, grpcTenantCtxKey, tid)
	}
	return ctx
}

// GRPCUnaryInterceptor returns a gRPC server unary interceptor that:
// 1. Validates JWT from metadata (authorization bearer token).
// 2. Injects tenant + user ID into context.
// 3. Logs request duration and status.
func GRPCUnaryInterceptor(cfg *GRPCInterceptorConfig) grpc.UnaryServerInterceptor {
	if cfg == nil {
		cfg = &GRPCInterceptorConfig{}
	}
	// P0 Security: If RequireAuth is true but no JWT validation key is set, fail hard.
	if cfg.RequireAuth && !cfg.authRequired() {
		slog.Error("GRPCUnaryInterceptor: RequireAuth=true but no JWT validation key (JWTSecret or RSAPublicKey) is set")
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

		md, ok := metadata.FromIncomingContext(ctx)
		if cfg.authRequired() {
			if !ok {
				return nil, status.Error(codes.Unauthenticated, "missing metadata")
			}
			token, err := extractBearerToken(md)
			if err != nil {
				return nil, err
			}
			claims, err := validateGRPCJWT(cfg, token)
			if err != nil {
				return nil, err
			}
			ctx = injectClaimsIntoContext(ctx, claims)
		}
		if ok && ctx.Value(grpcTenantCtxKey) == nil {
			if vals := md.Get(tenantHeader); len(vals) > 0 {
				ctx = context.WithValue(ctx, grpcTenantCtxKey, vals[0])
			}
		}

		resp, err := handler(ctx, req)

		if cfg.LogRequests {
			slog.Info("grpc request",
				"method", info.FullMethod,
				"duration", time.Since(start).String(),
				"code", status.Code(err).String(),
			)
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
	// P0 Security: Same guard as unary interceptor.
	if cfg.RequireAuth && !cfg.authRequired() {
		slog.Error("GRPCStreamInterceptor: RequireAuth=true but no JWT validation key is set")
		os.Exit(1)
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
		if cfg.authRequired() {
			if !ok {
				return status.Error(codes.Unauthenticated, "missing metadata")
			}
			token, err := extractBearerToken(md)
			if err != nil {
				return err
			}
			claims, err := validateGRPCJWT(cfg, token)
			if err != nil {
				return err
			}
			ctx = injectClaimsIntoContext(ctx, claims)
		}
		if ok && ctx.Value(grpcTenantCtxKey) == nil {
			if vals := md.Get(tenantHeader); len(vals) > 0 {
				ctx = context.WithValue(ctx, grpcTenantCtxKey, vals[0])
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
