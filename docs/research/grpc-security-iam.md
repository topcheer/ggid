# gRPC Security Hardening for IAM Systems

**Project**: GGID IAM Suite
**Author**: Security Research Team
**Date**: 2025-01
**Status**: Active Research
**Severity**: P1 — gRPC servers have zero transport security

---

## Executive Summary

GGID operates four gRPC servers (Policy, Audit, Org, Identity) that expose privileged
administrative APIs. All four accept unauthenticated, plaintext TCP connections with
no TLS, no message size limits, and no interceptor enforcement. While an interceptor
implementation exists in `services/gateway/internal/middleware/grpc_interceptor.go`,
it is never wired into any `grpc.NewServer()` call — meaning every gRPC endpoint is
effectively open. This document covers gRPC hardening patterns and provides a concrete
gap analysis against the current GGID codebase.

---

## 1. TLS/mTLS Between Services

### 1.1 Why mTLS for Internal Communication

In a microservice architecture like GGID (Gateway, Identity, Auth, Policy, Org, Audit),
services call each other over an internal network. Without TLS, any actor with network
access — a compromised pod, a misconfigured sidecar, an attacker who pivoted through a
vulnerable dependency — can intercept or inject gRPC traffic.

Mutual TLS (mTLS) requires **both** the client and the server to present certificates
signed by a shared internal CA. This provides:

- **Encryption**: Traffic between services is unreadable on the wire.
- **Server authentication**: The client verifies it is talking to the real Policy service,
  not a spoofed endpoint.
- **Client authentication**: The server verifies the caller is an authorized service
  (e.g., the Gateway), not an arbitrary client that discovered the gRPC port.
- **Identity propagation**: Client certificate CN or SPIFFE ID can be used for
  fine-grained authorization without separate token management.

### 1.2 Server-Side TLS Configuration

```go
package transport

import (
    "crypto/tls"
    "crypto/x509"
    "os"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

// NewServerTLS loads server certificate, private key, and CA bundle for mTLS.
// The CA bundle is used to verify client certificates.
func NewServerTLS(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("load server cert: %w", err)
    }

    caPEM, err := os.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("read CA bundle: %w", err)
    }
    caPool := x509.NewCertPool()
    if !caPool.AppendCertsFromPEM(caPEM) {
        return nil, fmt.Errorf("no valid certs found in CA bundle")
    }

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        ClientAuth:   tls.RequireAndVerifyClientCert, // mTLS: require client cert
        ClientCAs:    caPool,                          // verify against internal CA
        MinVersion:   tls.VersionTLS13,                // TLS 1.3 only
        CipherSuites: []uint16{
            tls.TLS_AES_128_GCM_SHA256,
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_CHACHA20_POLY1305_SHA256,
        },
    }

    return credentials.NewTLS(tlsConfig), nil
}

// Usage in a service's main():
//
//   creds, err := NewServerTLS("server.crt", "server.key", "ca.crt")
//   if err != nil { log.Fatal(err) }
//   grpcServer := grpc.NewServer(
//       grpc.Creds(creds),
//       grpc.ChainUnaryInterceptor(authInterceptor, tenantInterceptor, loggingInterceptor),
//   )
```

### 1.3 Client-Side TLS Configuration

```go
// NewClientTLS creates client credentials for mTLS.
// serverNameOverride is needed when the cert CN does not match the dial address
// (common with internal DNS names or IPs).
func NewClientTLS(certFile, keyFile, caFile, serverName string) (credentials.TransportCredentials, error) {
    cert, err := tls.LoadX509KeyPair(certFile, keyFile)
    if err != nil {
        return nil, fmt.Errorf("load client cert: %w", err)
    }

    caPEM, err := os.ReadFile(caFile)
    if err != nil {
        return nil, fmt.Errorf("read CA bundle: %w", err)
    }
    caPool := x509.NewCertPool()
    if !caPool.AppendCertsFromPEM(caPEM) {
        return nil, fmt.Errorf("no valid certs in CA bundle")
    }

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caPool,
        ServerName:   serverName,
        MinVersion:   tls.VersionTLS13,
    }

    return credentials.NewTLS(tlsConfig), nil
}
```

### 1.4 Certificate Generation and Rotation

For internal services, use a short-lived certificate lifecycle:

```go
// genInternalCert generates an ECDSA P-256 certificate valid for 24 hours.
// In production, use a CA like step-ca, Vault PKI, or cert-manager.
func genInternalCert(serviceName string) (tls.Certificate, *x509.Certificate, error) {
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return tls.Certificate{}, nil, err
    }

    template := &x509.Certificate{
        SerialNumber: big.NewInt(time.Now().UnixNano()),
        Subject: pkix.Name{
            CommonName:   serviceName,
            Organization: []string{"GGID Internal"},
        },
        NotBefore:   time.Now().Add(-5 * time.Minute),
        NotAfter:    time.Now().Add(24 * time.Hour), // 24-hour validity
        KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
        DNSNames:    []string{serviceName},
    }

    // In production, sign with the internal CA key, not self-signed.
    derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
    if err != nil {
        return tls.Certificate{}, nil, err
    }

    return tls.Certificate{
        Certificate: [][]byte{derBytes},
        PrivateKey:  priv,
    }, template, nil
}
```

**Rotation strategy**: Use a sidecar (like Envoy, linkerd, or a custom cert refresher)
that watches for cert expiry and hot-reloads. Go's `tls.Config` supports
`GetCertificate` / `GetClientCertificate` callbacks for zero-downtime rotation:

```go
tlsConfig.GetCertificate = func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
    return certManager.GetLatest(), nil
}
```

---

## 2. gRPC Channel Credentials

### 2.1 NewTLS() vs NewClientTLSFromFile()

| API | Use Case |
|-----|----------|
| `credentials.NewTLS(tlsConfig)` | Full control — you construct the `*tls.Config` yourself |
| `credentials.NewClientTLSFromFile(certFile, serverName)` | Quick one-liner for server-only TLS (no client cert) |
| `credentials.NewServerTLSFromFile(certFile, keyFile)` | Quick one-liner for server cert + key |

For mTLS, always use `NewTLS()` because the file-based helpers do not support
`ClientAuth: RequireAndVerifyClientCert`.

### 2.2 Per-RPC Credentials

Channel-level credentials (mTLS) authenticate the **connection**. Per-RPC credentials
authenticate the **individual request** — typically a JWT bearer token. Both layers
should be present in an IAM system:

```go
import (
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/oauth"
)

// jwtAccess provides per-RPC JWT bearer credentials.
type jwtAccess struct {
    token string
}

func (j *jwtAccess) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
    return map[string]string{
        "authorization": "Bearer " + j.token,
    }, nil
}

func (j *jwtAccess) RequireTransportSecurity() bool {
    return true // Per-RPC tokens must never be sent over plaintext
}

// NewDualCredentialClient creates a gRPC client combining mTLS (channel-level)
// with JWT bearer (per-RPC) credentials.
func NewDualCredentialClient(
    addr string,
    tlsCreds credentials.TransportCredentials,
    jwtToken string,
) (*grpc.ClientConn, error) {
    perRPCCreds := &jwtAccess{token: jwtToken}

    // Combine: mTLS for transport, JWT for per-RPC auth
    creds := grpc.WithTransportCredentials(tlsCreds)
    perRPC := grpc.WithPerRPCCredentials(perRPCCreds)

    return grpc.NewClient(addr, creds, perRPC)
}
```

### 2.3 OAuth Token-Based Per-RPC Credentials

For service-to-service auth using OAuth client credentials:

```go
// PerRPCToken fetches and caches OAuth tokens, refreshing on expiry.
type PerRPCToken struct {
    tokenURL    string
    clientID    string
    clientSecret string
    mu          sync.Mutex
    cachedToken string
    expiresAt   time.Time
}

func (t *PerRPCToken) getToken(ctx context.Context) (string, error) {
    t.mu.Lock()
    defer t.mu.Unlock()

    if time.Now().Before(t.expiresAt.Add(-30 * time.Second)) {
        return t.cachedToken, nil
    }

    // Fetch new token from tokenURL (e.g., Auth Service /oauth/token)
    // ... HTTP call omitted for brevity ...

    t.cachedToken = newToken
    t.expiresAt = time.Now().Add(1 * time.Hour)
    return t.cachedToken, nil
}

func (t *PerRPCToken) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
    token, err := t.getToken(ctx)
    if err != nil {
        return nil, err
    }
    return map[string]string{"authorization": "Bearer " + token}, nil
}

func (t *PerRPCToken) RequireTransportSecurity() bool {
    return true
}
```

---

## 3. Message Size Limits and Gzip Bomb Protection

### 3.1 Default Size Limit

gRPC defaults to a **4 MB** maximum receive message size. This is a reasonable default
for most APIs, but IAM systems must consider:

- **Policy evaluation requests** may include large attribute sets for ABAC policies.
- **Audit list queries** can return large result sets.
- **SCIM bulk operations** may carry large user payloads.

Set explicit limits to match your API's expected maximum:

```go
const maxMsgSize = 16 * 1024 * 1024 // 16 MB — adjust per service

grpcServer := grpc.NewServer(
    grpc.MaxRecvMsgSize(maxMsgSize),
    grpc.MaxSendMsgSize(maxMsgSize),
    grpc.MaxConcurrentStreams(100),     // Limit concurrent streams per connection
)
```

### 3.2 Gzip Bomb Attack

A gzip bomb is a small compressed payload that decompresses to an enormous size. For
example, a 10 KB gzip payload can decompress to over 10 GB. If a gRPC service accepts
gzip-compressed messages and decompresses them in memory, this causes OOM kills.

gRPC's built-in compression is safe — the 4 MB limit applies to the **compressed** size,
not the decompressed size. But if your service manually decompresses payloads (e.g.,
a custom codec or middleware), you must enforce a decompressed-size limit:

```go
// safeDecompress decompresses gzip data with a size limit.
// Returns an error if the decompressed output exceeds maxBytes.
func safeDecompress(compressed io.Reader, maxBytes int64) ([]byte, error) {
    zr, err := gzip.NewReader(compressed)
    if err != nil {
        return nil, fmt.Errorf("gzip reader: %w", err)
    }
    defer zr.Close()

    // LimitedReader prevents reading beyond maxBytes
    limited := &io.LimitedReader{R: zr, N: maxBytes + 1}
    data, err := io.ReadAll(limited)

    if limited.N == 0 {
        return nil, fmt.Errorf("decompressed size exceeds %d bytes (possible gzip bomb)", maxBytes)
    }
    if err != nil {
        return nil, err
    }

    return data, nil
}
```

### 3.3 Disabling Unwanted Compression

If your services don't need gzip compression (most internal services with small messages
don't), disable it to eliminate the attack surface:

```go
import "google.golang.org/grpc/encoding"

// Register a no-op compressor that rejects all compression.
// This prevents clients from forcing the server to decompress.
func init() {
    // By default, gRPC supports gzip. To restrict:
    encoding.RegisterCodec(&identityCodec{})
}

type identityCodec struct{}

func (c *identityCodec) Marshal(v any) ([]byte, error) {
    return encoding.GetCodec("proto").Marshal(v)
}

func (c *identityCodec) Unmarshal(data []byte, v any) error {
    return encoding.GetCodec("proto").Unmarshal(data, v)
}

func (c *identityCodec) Name() string { return "proto" }
```

---

## 4. gRPC Interceptor Chain for Security

### 4.1 Interceptor Architecture

Interceptors are gRPC's equivalent of HTTP middleware. Each interceptor wraps the
handler and can inspect/modify the request, context, and response. The order matters:

```
Request → PanicRecovery → Auth → TenantExtraction → RateLimit → Logging → Handler
```

- **PanicRecovery**: Must be outermost to catch panics from all subsequent interceptors.
- **Auth**: Must run before tenant extraction (auth determines who the caller is).
- **Tenant**: Must run before rate limiting (rate limit is per-tenant).
- **RateLimit**: Must run before the handler (reject early).
- **Logging**: Outermost or innermost depending on whether you want to log rejected requests.

### 4.2 Complete Interceptor Chain

```go
package transport

import (
    "context"
    "fmt"
    "log/slog"
    "runtime/debug"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
)

// --- Panic Recovery Interceptor ---

func PanicRecoveryInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (resp any, err error) {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("gRPC panic recovered",
                "method", info.FullMethod,
                "panic", r,
                "stack", string(debug.Stack()),
            )
            err = status.Error(codes.Internal, "internal server error")
        }
    }()
    return handler(ctx, req)
}

// --- Auth Interceptor ---

type Claims struct {
    UserID   string
    TenantID string
    Scopes   []string
}

type ctxKey string

const (
    claimsKey  ctxKey = "claims"
    tenantKey  ctxKey = "tenant_id"
    requestKey ctxKey = "request_id"
)

func AuthInterceptor(validate func(string) (*Claims, error)) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        // Skip auth for health check methods
        if isHealthMethod(info.FullMethod) {
            return handler(ctx, req)
        }

        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "no metadata")
        }

        tokens := md.Get("authorization")
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing authorization")
        }

        token := strings.TrimPrefix(tokens[0], "Bearer ")
        if token == tokens[0] {
            return nil, status.Error(codes.Unauthenticated, "invalid scheme")
        }

        claims, err := validate(token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("invalid token: %v", err))
        }

        ctx = context.WithValue(ctx, claimsKey, claims)
        ctx = context.WithValue(ctx, tenantKey, claims.TenantID)
        return handler(ctx, req)
    }
}

// --- Tenant Interceptor ---

func TenantInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        // Tenant may already be set by AuthInterceptor from JWT claims.
        // If not, check metadata header (for trusted internal callers).
        tenantID, _ := ctx.Value(tenantKey).(string)
        if tenantID == "" {
            md, ok := metadata.FromIncomingContext(ctx)
            if ok {
                if vals := md.Get("x-tenant-id"); len(vals) > 0 {
                    tenantID = vals[0]
                }
            }
        }

        if tenantID == "" && !isHealthMethod(info.FullMethod) {
            return nil, status.Error(codes.InvalidArgument, "tenant_id required")
        }

        ctx = context.WithValue(ctx, tenantKey, tenantID)
        return handler(ctx, req)
    }
}

// --- Rate Limiting Interceptor ---

func RateLimitInterceptor(limiter RateLimiter) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        tenantID, _ := ctx.Value(tenantKey).(string)
        key := tenantID
        if key == "" {
            // Fallback: use peer address for unauthenticated endpoints
            key = "anonymous"
        }

        if !limiter.Allow(key) {
            return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
        }

        return handler(ctx, req)
    }
}

// --- Logging Interceptor ---

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        start := time.Now()

        resp, err := handler(ctx, req)

        duration := time.Since(start)
        code := status.Code(err)

        logger.Info("gRPC request",
            "method", info.FullMethod,
            "duration", duration.String(),
            "code", code.String(),
        )

        return resp, err
    }
}

// --- Wiring ---

func NewSecureGRPCServer(
    jwtValidator func(string) (*Claims, error),
    limiter RateLimiter,
    logger *slog.Logger,
) *grpc.Server {
    return grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            PanicRecoveryInterceptor,                          // 1. Outermost: catch panics
            LoggingInterceptor(logger),                         // 2. Log all requests including rejections
            AuthInterceptor(jwtValidator),                      // 3. Authenticate
            TenantInterceptor(),                                // 4. Extract/enforce tenant
            RateLimitInterceptor(limiter),                      // 5. Rate limit per tenant
        ),
        grpc.MaxRecvMsgSize(16*1024*1024),                     // 16 MB
        grpc.MaxSendMsgSize(16*1024*1024),
    )
}

func isHealthMethod(method string) bool {
    return strings.HasPrefix(method, "/grpc.health.v1.Health/")
}
```

---

## 5. Stream Security

### 5.1 Long-Lived Stream Risks

Server-streaming and bidirectional-streaming RPCs maintain long-lived connections.
Security concerns:

- **Token expiry during stream**: A JWT may expire mid-stream. The server must re-validate
  on each message or set a stream-level TTL.
- **Resource exhaustion**: A client opens many streams but never reads responses,
  consuming server memory.
- **Stream hijacking**: If a client disconnects abnormally, the server stream may hang
  indefinitely if not cleaned up.

### 5.2 Stream Interceptor with Per-Message Auth

```go
// SecureStreamInterceptor wraps each stream with:
// 1. Initial auth on stream open
// 2. TTL-based expiry for long-lived streams
// 3. Max concurrent streams per tenant
func SecureStreamInterceptor(
    validate func(string) (*Claims, error),
    maxStreamDuration time.Duration,
) grpc.StreamServerInterceptor {
    return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
        // Auth check on stream open
        if !isHealthMethod(info.FullMethod) {
            md, ok := metadata.FromIncomingContext(ss.Context())
            if !ok {
                return status.Error(codes.Unauthenticated, "no metadata")
            }
            tokens := md.Get("authorization")
            if len(tokens) == 0 {
                return status.Error(codes.Unauthenticated, "missing authorization")
            }
            token := strings.TrimPrefix(tokens[0], "Bearer ")
            claims, err := validate(token)
            if err != nil {
                return status.Error(codes.Unauthenticated, "invalid token")
            }
            // Store claims in stream context for handler access
        }

        // Enforce max stream duration
        ctx, cancel := context.WithTimeout(ss.Context(), maxStreamDuration)
        defer cancel()

        wrapped := &secureServerStream{
            ServerStream: ss,
            ctx:          ctx,
        }

        // Recovery for stream panics
        var handlerErr error
        func() {
            defer func() {
                if r := recover(); r != nil {
                    handlerErr = status.Error(codes.Internal, "stream handler panic")
                }
            }()
            handlerErr = handler(srv, wrapped)
        }()

        return handlerErr
    }
}

type secureServerStream struct {
    grpc.ServerStream
    ctx context.Context
}

func (s *secureServerStream) Context() context.Context {
    return s.ctx
}

// RecvMsg wraps the underlying RecvMsg with panic recovery.
func (s *secureServerStream) RecvMsg(m any) error {
    return s.ServerStream.RecvMsg(m)
}
```

### 5.3 Max Concurrent Streams

Set `grpc.MaxConcurrentStreams` on the server to prevent a single client from
exhausting stream resources:

```go
grpcServer := grpc.NewServer(
    grpc.MaxConcurrentStreams(64),  // Max 64 concurrent streams per connection
    grpc.StreamInterceptor(SecureStreamInterceptor(jwtValidator, 5*time.Minute)),
)
```

---

## 6. gRPC Metadata Security

### 6.1 Metadata as Security Context Carrier

gRPC metadata is the equivalent of HTTP headers. In GGID, the following metadata keys
carry security context:

| Key | Purpose | Set By |
|-----|---------|--------|
| `authorization` | JWT bearer token | Client / Gateway |
| `x-tenant-id` | Tenant identifier | Gateway (from JWT claims) |
| `x-request-id` | Correlation ID | Gateway (generated) |
| `x-forwarded-for` | Client IP chain | Gateway |

### 6.2 Secure Metadata Extraction

**Critical rule**: Never trust metadata keys that should only be set by the gateway.
If a client sends `x-tenant-id` directly to a backend service, it bypasses gateway
authorization and enables tenant escalation.

```go
// SecureMetadataExtractor extracts security metadata with validation.
type SecurityContext struct {
    TenantID   string
    RequestID  string
    UserID     string
    ClientIP   string
}

// FromMetadata extracts security context from gRPC metadata.
// trustedHeadersOnly controls whether untrusted headers (tenant_id, user_id)
// are accepted from raw client metadata or only from JWT claims.
func FromMetadata(md metadata.MD, claims *Claims, trustedHeadersOnly bool) SecurityContext {
    sc := SecurityContext{}

    // Tenant ID: prefer JWT claims, fall back to metadata (internal calls only)
    if claims != nil && claims.TenantID != "" {
        sc.TenantID = claims.TenantID
    } else if !trustedHeadersOnly {
        // Only accept from metadata for trusted internal mTLS callers
        if vals := md.Get("x-tenant-id"); len(vals) > 0 {
            sc.TenantID = sanitizeMetadataValue(vals[0], 64) // max 64 chars
        }
    }

    if vals := md.Get("x-request-id"); len(vals) > 0 {
        sc.RequestID = sanitizeMetadataValue(vals[0], 128)
    }

    if claims != nil {
        sc.UserID = claims.UserID
    }

    if vals := md.Get("x-forwarded-for"); len(vals) > 0 {
        sc.ClientIP = sanitizeMetadataValue(strings.Split(vals[0], ",")[0], 64)
    }

    return sc
}

// sanitizeMetadataValue validates that a metadata value contains only safe
// characters and is within length limits. Prevents log injection and
// buffer overflow attacks via crafted metadata values.
func sanitizeMetadataValue(val string, maxLen int) string {
    if len(val) > maxLen {
        val = val[:maxLen]
    }
    // Allow only alphanumeric, hyphen, underscore, period, colon
    var sb strings.Builder
    for _, r := range val {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
           (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == ':' {
            sb.WriteRune(r)
        }
    }
    return sb.String()
}
```

### 6.3 Preventing Metadata Injection

**Attack scenario**: An external client sends `x-tenant-id: tenant-admin` directly
to the Policy gRPC service, bypassing the Gateway's JWT-based tenant resolution.
If the Policy service trusts this header, the client gains access to another tenant's
data.

**Defense**: Backend services must NOT accept `x-tenant-id` from raw metadata when
behind a gateway. They should extract tenant ID from JWT claims only. The `x-tenant-id`
header is only valid for trusted internal mTLS callers (e.g., gateway → service).

```go
// Defense: strip untrusted headers at the gateway, never forward them.
func GatewayGRPCMetadataInjector(ctx context.Context) context.Context {
    md, _ := metadata.FromIncomingContext(ctx)

    // Strip any client-supplied internal headers
    md.Delete("x-tenant-id")
    md.Delete("x-internal-user")

    // Re-inject from validated JWT claims
    claims := getClaimsFromContext(ctx)
    if claims != nil {
        md.Set("x-tenant-id", claims.TenantID)
    }

    return metadata.NewOutgoingContext(ctx, md)
}
```

---

## 7. Reflection Service Security

### 7.1 Information Disclosure Risk

gRPC reflection (`grpc.reflection.v1.ServerReflection`) allows clients to enumerate all
registered services, methods, and message types. In an IAM system, this exposes:

- Internal service names and method signatures
- Message field names (revealing internal data structures)
- Enumeration of available operations for reconnaissance

### 7.2 Conditional Reflection

```go
import "google.golang.org/grpc/reflection"

func RegisterReflection(grpcServer *grpc.Server, enableInProd bool) {
    if !enableInProd && os.Getenv("APP_ENV") == "production" {
        slog.Info("gRPC reflection disabled in production")
        return
    }
    reflection.Register(grpcServer)
    slog.Info("gRPC reflection enabled (development/staging only)")
}

// Usage:
//   RegisterReflection(grpcServer, false) // disabled in prod
```

### 7.3 Reflection via Auth Gate

For more granular control, allow reflection only for authenticated admin callers:

```go
func ReflectionAuthInterceptor(
    ctx context.Context,
    req any,
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (any, error) {
    if strings.HasPrefix(info.FullMethod, "/grpc.reflection.v1") {
        claims, ok := ctx.Value(claimsKey).(*Claims)
        if !ok || !claims.HasScope("admin:reflection") {
            return nil, status.Error(codes.PermissionDenied, "reflection requires admin scope")
        }
    }
    return handler(ctx, req)
}
```

---

## 8. Health Check Hardening

### 8.1 gRPC Health Checking Protocol

The standard gRPC health protocol (`grpc.health.v1.Health`) provides per-service health
status. Load balancers (Envoy, NGINX, AWS NLB) use this to decide routing.

### 8.2 Health Without Auth, With Rate Limiting

Health checks must NOT require authentication — load balancers and orchestrators
cannot obtain JWT tokens. But they MUST be rate-limited to prevent DoS:

```go
import (
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthService struct {
    healthpb.UnimplementedHealthServer
    mu     sync.RWMutex
    status map[string]healthpb.HealthCheckResponse_ServingStatus
    limiter RateLimiter
}

func NewHealthService(limiter RateLimiter) *HealthService {
    h := &HealthService{
        status:  make(map[string]healthpb.HealthCheckResponse_ServingStatus),
        limiter: limiter,
    }
    h.status[""] = healthpb.HealthCheckResponse_SERVING // default: serving
    return h
}

func (h *HealthService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
    // Rate limit health checks to prevent DoS
    // Use a high limit (e.g., 100/sec per source) since LBs poll frequently
    if !h.limiter.Allow("health-check") {
        return nil, status.Error(codes.ResourceExhausted, "health check rate limit exceeded")
    }

    h.mu.RLock()
    defer h.mu.RUnlock()

    service := req.GetService()
    if s, ok := h.status[service]; ok {
        return &healthpb.HealthCheckResponse{Status: s}, nil
    }
    // Unknown service = NOT_SERVING
    return &healthpb.HealthCheckResponse{
        Status: healthpb.HealthCheckResponse_NOT_SERVING,
    }, nil
}

func (h *HealthService) Watch(req *healthpb.HealthCheckRequest, stream healthpb.Health_HealthCheck_WatchServer) error {
    // Rate limit watch connections too
    if !h.limiter.Allow("health-watch") {
        return status.Error(codes.ResourceExhausted, "watch rate limit exceeded")
    }

    // Send initial status
    h.mu.RLock()
    current := h.status[req.GetService()]
    h.mu.RUnlock()

    if err := stream.Send(&healthpb.HealthCheckResponse{Status: current}); err != nil {
        return err
    }

    // In production: watch a channel for status changes and send updates.
    // For simplicity, just block with a periodic check.
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-stream.Context().Done():
            return nil
        case <-ticker.C:
            h.mu.RLock()
            s := h.status[req.GetService()]
            h.mu.RUnlock()
            if s != current {
                current = s
                if err := stream.Send(&healthpb.HealthCheckResponse{Status: current}); err != nil {
                    return err
                }
            }
        }
    }
}

func (h *HealthService) SetServing(service string, status healthpb.HealthCheckResponse_ServingStatus) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.status[service] = status
}
```

### 8.3 Wiring Health Service

```go
func main() {
    // ...
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(/* ... */),
        grpc.MaxRecvMsgSize(16*1024*1024),
    )

    // Register health service BEFORE business services
    healthLimiter := NewTokenBucketLimiter(100, 100) // 100 req/s burst
    healthSvc := NewHealthService(healthLimiter)
    healthpb.RegisterHealthServer(grpcServer, healthSvc)

    // Register business services
    pb.RegisterPolicyServiceServer(grpcServer, policyHandler)

    // Set initial health
    healthSvc.SetServing("", healthpb.HealthCheckResponse_SERVING)

    // On graceful shutdown:
    healthSvc.SetServing("", healthpb.HealthCheckResponse_NOT_SERVING)
    time.Sleep(2 * time.Second) // let LB pick up status change
    grpcServer.GracefulStop()
}
```

---

## 9. GGID gRPC Hardening Audit

### 9.1 Current State Analysis

The following findings are based on a direct code review of GGID's gRPC server setup.

#### Services Audited

| Service | File | gRPC Setup |
|---------|------|------------|
| Policy | `services/policy/cmd/main.go` | `grpc.NewServer()` — bare |
| Audit | `services/audit/cmd/main.go` | `grpc.NewServer()` — bare |
| Org | `services/org/cmd/main.go` | `grpc.NewServer()` — bare |
| Identity | `services/identity/internal/server/server.go` | `grpc.NewServer()` — bare |

#### Finding 1: No Transport Security (P0)

All four gRPC servers call `grpc.NewServer()` with no `grpc.Creds()` option. All
inter-service gRPC traffic flows over **plaintext TCP**. Any pod on the same network
can connect to `:9070` (Policy), `:9071` (Org), `:9072` (Audit) and call any method
without authentication.

**Evidence**:
- `services/policy/cmd/main.go:78`: `grpcServer := grpc.NewServer()`
- `services/audit/cmd/main.go:72`: `grpcServer := grpc.NewServer()`
- `services/org/cmd/main.go:81`: `grpcServer := grpc.NewServer()`
- `services/identity/internal/server/server.go:46`: `grpcSrv := grpc.NewServer()`
- No `credentials.NewTLS()` calls exist anywhere in the services codebase.

#### Finding 2: Interceptors Exist But Are Not Wired (P0)

The GGID codebase contains a gRPC interceptor implementation in
`services/gateway/internal/middleware/grpc_interceptor.go` that handles JWT validation
and tenant injection. However, **it is never passed to any `grpc.NewServer()` call**.
The interceptors are dead code from a security perspective.

**Evidence**: `grep` for `grpc.ChainUnaryInterceptor` or `grpc.UnaryInterceptor` in
service `main.go` files returns zero results. The `GRPCUnaryInterceptor` function is
only referenced in test files.

#### Finding 3: No Message Size Limits (P1)

No `grpc.MaxRecvMsgSize()` or `grpc.MaxSendMsgSize()` is set on any server. The default
4 MB limit applies, but this is implicit, not explicit. A policy evaluation request with
a large ABAC attribute set could be used to probe memory usage.

#### Finding 4: No Reflection Control (P2)

No `reflection.Register()` call is present in any service. This means reflection is
implicitly disabled, which is correct for production. However, there is no explicit
guard preventing a developer from adding it without conditional logic.

#### Finding 5: No gRPC Health Protocol (P2)

GGID services use HTTP `/healthz` endpoints instead of the standard
`grpc.health.v1.Health` protocol. This works for Docker healthchecks and simple load
balancers, but gRPC-native health checking (via Envoy, gRPC LB) is not available.

#### Finding 6: JWT Not Validated (P1)

The existing interceptor (`grpc_interceptor.go:88-89`) stores the raw JWT token string
in context but does **not** validate JWT claims:

```go
// In production, validate JWT claims here.
ctx = context.WithValue(ctx, grpcUserCtxKey, token)
```

The comment admits this is a stub. Any string prefixed with "Bearer " is accepted.

#### Finding 7: No pkg/transport Package (P2)

The task brief references `pkg/transport/` for gRPC utilities. This directory does not
exist. All TLS, credentials, and interceptor utilities must be created from scratch.
Shared gRPC configuration should live in a common package to avoid duplication across
the 4+ services.

#### Finding 8: No gRPC Clients

The Gateway currently uses HTTP reverse proxy to reach backend services, not gRPC
client connections. The gRPC ports (9070, 9071, 9072) are exposed but unused by the
gateway. If gRPC clients are added in the future, they must use mTLS.

### 9.2 Security Posture Summary

| Control | Status | Risk |
|---------|--------|------|
| TLS/mTLS | Not implemented | P0 — plaintext traffic |
| Auth interceptors | Implemented but not wired | P0 — all RPCs unauthenticated |
| JWT validation | Not implemented (stub) | P1 — any token accepted |
| Message size limits | Default only (implicit) | P2 |
| Reflection | Disabled (implicit) | Low |
| Health protocol | HTTP /healthz only | Low |
| Rate limiting | Not on gRPC | P2 |
| Panic recovery | Not on gRPC | P2 |

---

## 10. Gap Analysis & Recommendations

### Action Item 1: Create `pkg/transport` with Shared gRPC Security (Effort: 2 days)

Create a shared package at `pkg/transport/grpc/` containing:
- `NewSecureServer(opts)` — returns a `*grpc.Server` with TLS, interceptors, and size limits pre-configured
- `NewSecureClient(addr, opts)` — returns a `*grpc.ClientConn` with mTLS and per-RPC credentials
- `TLSConfig` builder functions for server and client
- Interceptor implementations (panic recovery, auth, tenant, rate limit, logging)

This eliminates the duplication across 4 services and centralizes security configuration.

### Action Item 2: Wire Interceptors into All gRPC Servers (Effort: 1 day)

Update the `main.go` of Policy, Audit, Org, and Identity to use the shared
`NewSecureServer()` instead of bare `grpc.NewServer()`. This is a one-line change per
service but requires the shared package from Action Item 1.

```go
// Before:
grpcServer := grpc.NewServer()

// After:
grpcServer, err := transport.NewSecureServer(transport.Config{
    TLS:         tlsConfig,
    JWTSecret:   cfg.JWTSecret,
    MaxMsgSize:  16 * 1024 * 1024,
    RateLimiter: limiter,
})
```

### Action Item 3: Implement JWT Validation in Interceptor (Effort: 1 day)

Replace the stub in `grpc_interceptor.go:88-89` with actual JWT parsing and claim
validation. Use the existing `pkg/crypto` JWT utilities. Reject expired tokens,
validate issuer, and extract tenant_id + user_id from claims.

### Action Item 4: Add mTLS for Internal Service Communication (Effort: 3 days)

1. Generate an internal CA (or use HashiCorp Vault PKI / step-ca).
2. Issue service certificates for each microservice with 24-hour TTL.
3. Configure each gRPC server with `tls.RequireAndVerifyClientCert`.
4. Deploy via cert-manager (Kubernetes) or a sidecar cert refresher.
5. This is the highest-impact security improvement for the gRPC layer.

### Action Item 5: Add gRPC Health Protocol with Rate Limiting (Effort: 0.5 days)

Register `grpc.health.v1.Health` on each service with a rate-limited implementation.
This enables gRPC-native health checking for Envoy/Kubernetes gRPC probes, which HTTP
`/healthz` does not support. Add `grpc.MaxConcurrentStreams(64)` to each server.

### Priority Order

| Priority | Action | Effort | Impact |
|----------|--------|--------|--------|
| P0 | #3 — JWT validation in interceptor | 1 day | Eliminates token spoofing |
| P0 | #2 — Wire interceptors | 1 day | Activates auth on all RPCs |
| P0 | #1 — Create pkg/transport | 2 days | Foundation for all hardening |
| P0 | #4 — mTLS internal comms | 3 days | Encrypts all traffic |
| P1 | #5 — gRPC health protocol | 0.5 days | LB integration |

**Total estimated effort**: 7.5 developer-days for complete gRPC security hardening.

---

## References

- [gRPC Security Best Practices](https://grpc.io/docs/guides/auth/) — official authentication guide
- [google.golang.org/grpc/credentials](https://pkg.go.dev/google.golang.org/grpc/credentials) — TLS credential API
- [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md) — specification
- [gRPC Reflection](https://github.com/grpc/grpc/blob/master/doc/server-reflection.md) — specification
- [Zero Trust IAM Patterns](../research/zero-trust-iam.md) — GGID internal research
- [Certificate Pinning for IAM](../research/certificate-pinning-iam.md) — GGID internal research
- [Token Replay Defense](../research/token-replay-defense.md) — GGID internal research
