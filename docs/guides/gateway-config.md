# API Gateway Configuration Guide

This guide covers the GGID API Gateway configuration: routing, JWT validation, rate limiting, CORS, SSRF protection, host validation, and circuit breaker settings.

## Overview

The GGID API Gateway is the single entry point for all client requests. It handles:

- Request routing to 7 backend microservices
- JWT authentication and authorization
- Rate limiting (token bucket per IP/user)
- CORS policy enforcement
- SSRF protection for webhook/webhook-like traffic
- Host header validation (DNS rebinding defense)
- Circuit breaker for service resilience
- API key authentication for service-to-service calls
- Body size limiting
- Request/response compression

## Route Configuration

### Service Routes

Routes are defined in `services/gateway/internal/config/config.go`:

```go
var Routes = []Route{
    // Identity Service
    {Prefix: "/api/v1/users",     Upstream: "identity:8080"},
    {Prefix: "/api/v1/groups",    Upstream: "identity:8080"},
    {Prefix: "/api/v1/scim",      Upstream: "identity:8080"},

    // Auth Service
    {Prefix: "/api/v1/auth",      Upstream: "auth:9001"},
    {Prefix: "/api/v1/mfa",       Upstream: "auth:9001"},
    {Prefix: "/api/v1/webauthn",  Upstream: "auth:9001"},

    // OAuth Service
    {Prefix: "/api/v1/oauth",     Upstream: "oauth:9005"},
    {Prefix: "/api/v1/agents",    Upstream: "oauth:9005"},
    {Prefix: "/.well-known",      Upstream: "oauth:9005"},

    // Policy Service
    {Prefix: "/api/v1/roles",     Upstream: "policy:8070"},
    {Prefix: "/api/v1/policies",  Upstream: "policy:8070"},
    {Prefix: "/api/v1/access-requests", Upstream: "policy:8070"},

    // Org Service
    {Prefix: "/api/v1/organizations", Upstream: "org:8071"},

    // Audit Service
    {Prefix: "/api/v1/audit",     Upstream: "audit:8072"},
}
```

### Health Endpoints

Public health endpoints bypass JWT validation:

```
GET /healthz         → Gateway health
GET /api/v1/health   → Aggregated service health
```

### Adding a New Route

1. Add route definition to `config.go`
2. Restart gateway service
3. Test with healthcheck

## JWT Validation

### Validation Pipeline

```
Request → Extract JWT from Authorization header
         ↓
    Verify signature (RS256/ES256)
         ↓
    Check expiry (exp claim)
         ↓
    Check issuer (iss claim) matches JWT_ISSUER
         ↓
    Check tenant_id claim → override X-Tenant-ID header
         ↓
    Extract scopes → HasScope() enforcement
         ↓
    Forward to upstream with X-Tenant-ID + X-User-ID headers
```

### Key Configuration

```yaml
JWT:
  issuer: "ggid"
  algorithm: "RS256"          # or ES256
  key_file: "/keys/jwt.pub"   # Public key (verification)
  # Private key only on Auth service
```

### Scope Enforcement

```go
// Admin endpoints require admin scope
if !hasAdminScope(claims) {
    return Error(403, "insufficient_scope")
}

// Resource endpoints check specific scopes
if !HasScope(claims, "users:read") {
    return Error(403, "insufficient_scope")
}
```

### Tenant Resolution

Priority order (highest wins):
1. JWT `tenant_id` claim (trusted, from auth service)
2. `X-Tenant-ID` header (untrusted, may be spoofed)

The JWT claim always takes priority to prevent tenant spoofing.

## Rate Limiting

### Token Bucket Configuration

GGID uses a token bucket rate limiter (`ratelimit.go`) wired into the production handler chain:

```yaml
RateLimit:
  enabled: true
  requests_per_second: 10
  burst: 20
  # Per-IP tracking
  key_func: "client_ip"
  # Storage
  storage: "redis"            # or "memory"
```

### Rate Limit Tiers

| Tier          | RPS   | Burst | Applies To                  |
|---------------|-------|-------|-----------------------------|
| Default       | 10    | 20    | Standard API endpoints      |
| Auth          | 3     | 5     | /api/v1/auth/* (login, register) |
| Read          | 50    | 100   | /api/v1/users (GET), /api/v1/audit |
| Admin         | 5     | 10    | /api/v1/admin/*             |

### Client IP Extraction

```go
func ClientIP(r *http.Request) string {
    // 1. Check X-Forwarded-For (trusted proxy only)
    // 2. Check X-Real-IP
    // 3. Fall back to RemoteAddr (with port stripped via net.SplitHostPort)
}
```

> The `RemoteAddr` is stripped of the port number to ensure CIDR matching works correctly.

### 429 Response Format

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Try again in 60 seconds.",
  "retry_after": 60
}
```

## CORS Configuration

### Policy

```yaml
CORS:
  allowed_origins:
    - "https://console.ggid.example.com"
    - "https://admin.ggid.example.com"
  allowed_methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]
  allowed_headers: ["Authorization", "Content-Type", "X-Tenant-ID", "X-Request-ID"]
  exposed_headers: ["X-Request-ID", "X-RateLimit-Remaining"]
  allow_credentials: true
  max_age: 3600                # Preflight cache (seconds)
```

### Preflight Handling

OPTIONS requests are handled by the CORS middleware before JWT validation:

```go
if r.Method == "OPTIONS" {
    handlePreflight(w, r)
    return
}
```

> Do NOT use `*` for `allowed_origins` when `allow_credentials` is true — browsers will reject it.

## SSRF Protection

### Webhook URL Validation

The webhook deliverer validates outbound URLs against:

1. **Private IP range blocking**: RFC 1918, link-local, loopback, metadata endpoints
2. **Protocol restriction**: Only `http` and `https` allowed
3. **DNS resolution check**: Resolved IP must not be private

```go
func isPrivateIP(ip net.IP) bool {
    return ip.IsLoopback() ||
           ip.IsPrivate() ||       // 10.x, 172.16-31.x, 192.168.x
           ip.IsLinkLocalUnicast() || // 169.254.x (AWS metadata!)
           ip.IsUnspecified()
}
```

### Blocked Endpoints

| Endpoint                    | Reason                          |
|-----------------------------|---------------------------------|
| 127.0.0.0/8                 | Loopback                        |
| 10.0.0.0/8                  | Private (RFC 1918)              |
| 172.16.0.0/12               | Private (RFC 1918)              |
| 192.168.0.0/16              | Private (RFC 1918)              |
| 169.254.169.254             | Cloud metadata service          |
| ::1/128                     | IPv6 loopback                   |
| fc00::/7                    | IPv6 unique local               |
| fe80::/10                   | IPv6 link-local                 |

## Host Header Validation

### DNS Rebinding Defense

The `host_validation.go` middleware prevents DNS rebinding attacks by validating the `Host` header against an allowlist:

```yaml
HostValidation:
  enabled: true
  allowed_hosts:
    - "api.ggid.example.com"
    - "ggid.example.com"
    - "localhost"               # Development only
```

Requests with a `Host` header not in the allowlist receive a `403 Forbidden`.

### How DNS Rebinding Works

1. Attacker controls DNS for `evil.com`
2. First DNS query resolves to attacker's server (serves malicious JS)
3. JS makes request to `evil.com` but DNS now resolves to `127.0.0.1`
4. Without host validation, the gateway processes the request as if from localhost

Host validation breaks this attack by rejecting any `Host` not in the allowlist.

## Circuit Breaker

### Configuration

```yaml
CircuitBreaker:
  enabled: true
  max_requests: 5              # Max requests allowed when half-open
  interval: 60s                # Cyclical period in closed state
  timeout: 30s                 # Open state duration before half-open
  failure_threshold: 5         # Consecutive failures before opening
}
```

### States

```
CLOSED → (N failures) → OPEN → (timeout) → HALF-OPEN → (success) → CLOSED
                                         ↘ (failure) → OPEN
```

| State      | Behavior                                         |
|------------|--------------------------------------------------|
| CLOSED     | All requests forwarded to upstream               |
| OPEN       | All requests return 503 immediately              |
| HALF-OPEN  | Limited requests (max_requests) test if upstream recovered |

### Per-Upstream Tracking

Each upstream service has its own circuit breaker:

```go
breakers := map[string]*CircuitBreaker{
    "identity:8080": NewCircuitBreaker(config),
    "auth:9001":     NewCircuitBreaker(config),
    "policy:8070":   NewCircuitBreaker(config),
    // ...
}
```

## Body Size Limiting

```yaml
BodySize:
  max_bytes: 1048576           # 1 MB default
  endpoints:
    "/api/v1/users/import":
      max_bytes: 10485760      # 10 MB for bulk import
    "/api/v1/branding/logo":
      max_bytes: 5242880       # 5 MB for logo upload
```

Requests exceeding the limit receive `413 Request Entity Too Large`.

## Compression

```yaml
Compression:
  enabled: true
  algorithms: ["gzip", "brotli"]
  min_size: 1024               # Don't compress responses < 1 KB
  skip_types:                  # Already compressed types
    - "image/png"
    - "image/jpeg"
    - "video/mp4"
    - "application/zip"
```

## Security Headers

The gateway adds security headers to all responses:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

## Middleware Chain Order

```
1. Host Validation        → DNS rebinding defense
2. CORS                   → Preflight handling
3. Body Size Limit        → 413 on oversized requests
4. Rate Limiting          → 429 on threshold exceeded
5. JWT Validation         → 401 on invalid/expired token
6. Scope Enforcement      → 403 on insufficient scope
7. Security Headers       → Add response headers
8. Compression            → Gzip/Brotli response compression
9. Audit Logging          → Record request metadata
10. Circuit Breaker       → 503 when upstream is down
11. Proxy                 → Forward to upstream service
```

## Monitoring

### Metrics Endpoints

```
GET /metrics    → Prometheus metrics (rate limit hits, circuit breaker state, latency)
GET /healthz    → Gateway health
GET /api/v1/health → Aggregated backend health
```

### Key Metrics

| Metric                          | Description                            |
|---------------------------------|----------------------------------------|
| `gateway_requests_total`        | Total requests by method/path/status   |
| `gateway_rate_limit_hits`       | Rate limit 429 responses               |
| `gateway_circuit_breaker_state` | Circuit breaker state per upstream     |
| `gateway_jwt_validation_errors` | JWT validation failures                |
| `gateway_upstream_latency`      | Latency histogram per upstream         |

## See Also

- [Rate Limiting Guide](rate-limiting.md)
- [Security Overview](../architecture/security-overview.md)
- [STRIDE Threat Analysis](stride-analysis.md)
- [Security Audit Checklist](security-audit-checklist.md)
