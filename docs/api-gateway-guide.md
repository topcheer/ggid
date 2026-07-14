# API Gateway Guide

> **Deprecated:** This document is superseded by docs/api-gateway.md. See the updated version for the latest information.


Gateway configuration: route table, middleware chain, JWT verification, rate
limiting, tenant resolution, health checks, graceful shutdown, and proxy
timeout configuration.

---

## Table of Contents

- [Architecture](#architecture)
- [Route Table](#route-table)
- [Middleware Chain](#middleware-chain)
- [JWT Verification](#jwt-verification)
- [Tenant Resolution](#tenant-resolution)
- [Rate Limiting](#rate-limiting)
- [Health Checks](#health-checks)
- [Proxy Configuration](#proxy-configuration)
- [Graceful Shutdown](#graceful-shutdown)

---

## Architecture

```
Client                    Gateway                      Backend Services
  │                          │                           ┌───────────┐
  │ HTTPS request            │                           │ Identity  │
  ├─────────────────────────►│ ──► Auth (verify JWT) ──► │ :8081     │
  │                          │ ──► Tenant (resolve) ──►  │ Auth      │
  │                          │ ──► RateLimit (check) ──► │ :9001     │
  │                          │ ──► Proxy (forward) ─────►│ OAuth     │
  │                          │                           │ :9005     │
  │                          │                           │ Policy    │
  │                          │                           │ :8070     │
  │                          │                           │ Org       │
  │                          │                           │ :8071     │
  │                          │                           │ Audit     │
  │                          │                           │ :8072     │
  │                          │                           └───────────┘
  │◄─────────────────────────┤
  │ HTTPS response           │
```

---

## Route Table

| Path Prefix | Method | Service | Port | Auth Required |
|-------------|--------|---------|------|:-------------:|
| `/api/v1/auth/*` | ALL | Auth | 9001 | No |
| `/api/v1/users/*` | ALL | Identity | 8081 | Yes |
| `/api/v1/roles/*` | ALL | Policy | 8070 | Yes |
| `/api/v1/orgs/*` | ALL | Org | 8071 | Yes |
| `/api/v1/audit/*` | ALL | Audit | 8072 | Yes |
| `/oauth/*` | ALL | OAuth | 9005 | Varies |
| `/scim/v2/*` | ALL | Identity | 8081 | Bearer |
| `/api/v1/admin/*` | ALL | Various | - | Yes + Admin |
| `/healthz` | GET | Gateway | - | No |
| `/metrics` | GET | Gateway | - | No |

---

## Middleware Chain

```
Request →
  1. RequestID (generate X-Request-ID)
  2. Recovery (panic handler, log stack trace)
  3. Logger (structured JSON, request duration)
  4. TenantResolver (extract X-Tenant-ID, validate against JWT)
  5. JWTAuth (verify Bearer token, extract claims)
  6. RateLimiter (per-IP + per-user token bucket)
  7. CORS (preflight handling)
  8. SecurityHeaders (HSTS, CSP, X-Frame-Options)
  9. Proxy (route to backend)
→ Response
```

### Configuration

```yaml
gateway:
  middleware:
    request_id: true
    recovery: true
    logger:
      format: json
      level: info
    tenant_resolver:
      header: X-Tenant-ID
      jwt_claim: tenant_id
      reject_mismatch: true
    jwt_auth:
      jwks_url: https://auth:9001/.well-known/jwks.json
      cache_ttl: 300s
      skip_paths: [/api/v1/auth/login, /api/v1/auth/register, /healthz]
    rate_limiter:
      enabled: true
      default: 100/min
      auth_endpoints: 10/min
    cors:
      allowed_origins: ["https://console.example.com"]
      allow_credentials: true
    security_headers:
      hsts: true
      csp: "default-src 'self'"
```

---

## JWT Verification

### Flow

```
1. Extract Bearer token from Authorization header
2. Parse JWT header → extract kid
3. Look up public key in JWKS cache (refreshed every 5 min)
4. Verify signature (RS256/ES256)
5. Check exp (not expired)
6. Check iss (matches expected issuer)
7. Check aud (matches this service)
8. Extract claims: sub, tenant_id, scope, amr
9. Inject claims into request context
```

### JWKS Caching

```yaml
jwt:
  jwks:
    url: https://auth:9001/.well-known/jwks.json
    refresh_interval: 300s       # Refresh every 5 min
    cache_ttl: 3600s             # Cache keys for 1 hour
    fallback_static_key: true    # Use static key if JWKS unavailable
```

---

## Tenant Resolution

### Priority Order

1. **JWT claim** (highest priority) — `tenant_id` in token payload
2. **X-Tenant-ID header** — Must match JWT claim if both present
3. **Domain** — Extract tenant from subdomain (e.g., `acme.iam.example.com`)

### Security: Prevent Tenant Spoofing

```go
// If JWT has tenant_id, it takes priority over header
jwtTenant := claims.TenantID
headerTenant := r.Header.Get("X-Tenant-ID")

if jwtTenant != "" && jwtTenant != headerTenant {
    // REJECT: header doesn't match JWT
    respondError(w, 403, "JWT tenant does not match X-Tenant-ID")
    return
}
```

---

## Rate Limiting

```yaml
gateway:
  rate_limit:
    # Per-IP (applies to all requests)
    ip:
      requests_per_minute: 100
      burst: 200

    # Per-user (applies to authenticated requests)
    user:
      requests_per_minute: 1000
      burst: 500

    # Per-endpoint overrides
    endpoints:
      /api/v1/auth/login:
        ip: 10/min
      /api/v1/auth/register:
        ip: 5/min
      /api/v1/auth/password/reset-request:
        ip: 3/min
```

---

## Health Checks

### Gateway Health

```
GET /healthz

Response 200:
{
  "status": "healthy",
  "services": {
    "identity": "healthy",
    "auth": "healthy",
    "oauth": "healthy",
    "policy": "healthy",
    "org": "healthy",
    "audit": "healthy"
  }
}
```

### Probe Types

| Probe | Endpoint | Check |
|-------|----------|-------|
| Liveness | `/healthz` | Gateway process is running |
| Readiness | `/readyz` | All backends reachable |
| Startup | `/healthz` | Initial health check |

---

## Proxy Configuration

```yaml
gateway:
  proxy:
    timeout: 30s                # Overall request timeout
    dial_timeout: 5s            # Connection establishment
    response_header_timeout: 10s
    idle_conn_timeout: 90s
    max_idle_conns: 100
    max_idle_conns_per_host: 10
    keep_alive: 30s

    # Per-route overrides
    route_overrides:
      /api/v1/auth/login:
        timeout: 10s
      /api/v1/audit/events/export:
        timeout: 120s           # Long-running export
      /scim/v2/Bulk:
        timeout: 60s            # Bulk SCIM operations
```

---

## Graceful Shutdown

```yaml
gateway:
  shutdown:
    timeout: 30s               # Max time to finish in-flight requests
    drain_connections: true    # Stop accepting new connections immediately
    health_check_fail: true    # /healthz returns 503 immediately
```

### Shutdown Sequence

```
1. Receive SIGTERM
2. Stop accepting new connections
3. /healthz returns 503 (load balancer removes from pool)
4. Wait for in-flight requests to complete (up to 30s)
5. Close idle connections
6. Close database/Redis/NATS connections
7. Exit process
```

This ensures zero-downtime during rolling updates.
