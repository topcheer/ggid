# API Gateway Guide

> Complete guide to the GGID API Gateway: reverse proxy configuration, route table, middleware chain, JWT verification flow, tenant header injection, rate limiting, and circuit breaker.

---

## Table of Contents

1. [Overview](#overview)
2. [Gateway Architecture](#gateway-architecture)
3. [Route Table](#route-table)
4. [Middleware Chain Order](#middleware-chain-order)
5. [JWT Verification Flow](#jwt-verification-flow)
6. [Tenant Header Injection](#tenant-header-injection)
7. [Rate Limiting](#rate-limiting)
8. [Circuit Breaker](#circuit-breaker)
9. [Compression](#compression)
11. [Distributed Tracing (OpenTelemetry)](#distributed-tracing-opentelemetry)
12. [Graceful Shutdown](#graceful-shutdown)
13. [Configuration Reference](#configuration-reference)

---

## Overview

The API Gateway is the single entry point for all external API requests. It handles:

- **JWT verification** — validates tokens before forwarding to backends
- **Routing** — maps URL paths to backend microservices
- **Rate limiting** — protects backends from abuse
- **Tenant context** — extracts and injects tenant identity
- **Security headers** — CORS, HSTS, X-Frame-Options
- **Compression** — gzip/brotli for response payloads
- **Tracing** — OpenTelemetry span propagation
- **Health checking** — load balancer health endpoint
- **Circuit breaking** — fail-fast on backend failures

```
                    ┌──────────────────────────┐
                    │      API Gateway (:8080) │
                    │                          │
  Client ─────────▶│  CORS → Rate Limit →     │──── Identity (:8081)
   HTTP            │  Security Hdrs → JWT →   │──── Auth (:9001)
   JSON            │  Tenant Ctx → Audit →    │──── OAuth (:9005)
                   │  Proxy                   │──── Policy (:8070)
                    │                          │──── Org (:8071)
                    └──────────────────────────┘──── Audit (:8072)
```

---

## Gateway Architecture

### Design Principles

1. **Stateless**: No session storage in gateway — all state in JWT or Redis
2. **Fail-closed**: If JWT verification fails, request is rejected (not forwarded)
3. **Defense in depth**: Gateway validates auth, backend re-validates
4. **Hot-reloadable**: Configuration changes without restart (where possible)

### Port Bindings

| Protocol | Port | Purpose |
|----------|------|---------|
| HTTP/1.1 | 8080 | Primary REST API |
| HTTP/2 (h2c) | 8080 | Internal gRPC proxying |
| HTTP/3 (QUIC) | 8080 (alt) | Future — UDP transport |

---

## Route Table

The gateway routes requests based on URL path prefix:

| Path Prefix | Backend Service | Backend Port | Methods | Auth Required |
|-------------|----------------|-------------|---------|---------------|
| `/healthz` | Gateway (local) | — | GET | No |
| `/metrics` | Gateway (local) | — | GET | No |
| `/api/v1/users` | Identity | 8081 | GET, POST, PUT, DELETE | Yes |
| `/api/v1/users/*` | Identity | 8081 | GET, PUT, DELETE | Yes |
| `/api/v1/scim/v2/*` | Identity | 8081 | GET, POST, PUT, PATCH, DELETE | Bearer (SCIM) |
| `/api/v1/auth/register` | Auth | 9001 | POST | No |
| `/api/v1/auth/login` | Auth | 9001 | POST | No |
| `/api/v1/auth/refresh` | Auth | 9001 | POST | Refresh token |
| `/api/v1/auth/logout` | Auth | 9001 | POST | Yes |
| `/api/v1/auth/mfa/*` | Auth | 9001 | GET, POST | Yes |
| `/api/v1/auth/webauthn/*` | Auth | 9001 | GET, POST | Yes |
| `/api/v1/oauth/*` | OAuth | 9005 | GET, POST | Varies |
| `/api/v1/oauth/authorize` | OAuth | 9005 | GET, POST | Session |
| `/api/v1/oauth/token` | OAuth | 9005 | POST | Client creds |
| `/api/v1/oauth/introspect` | OAuth | 9005 | POST | Bearer |
| `/api/v1/oauth/revoke` | OAuth | 9005 | POST | Bearer |
| `/api/v1/roles` | Policy | 8070 | GET, POST, PUT, DELETE | Yes |
| `/api/v1/policies/*` | Policy | 8070 | GET, POST | Yes |
| `/api/v1/orgs` | Org | 8071 | GET, POST, PUT, DELETE | Yes |
| `/api/v1/orgs/*` | Org | 8071 | GET, PUT, DELETE | Yes |
| `/api/v1/audit` | Audit | 8072 | GET | Yes |
| `/api/v1/audit/events` | Audit | 8072 | GET | Yes |
| `/api/v1/webhooks` | Gateway (local) | — | GET, POST, PUT, DELETE | Yes |
| `/api/v1/admin/*` | Gateway (local) | — | GET, POST | Admin scope |
| `/api/v1/tenants` | Gateway (local) | — | GET, POST, PUT, DELETE | Super-admin |

### Route Priority

Routes are matched by **longest prefix**. More specific routes take priority:

```
/api/v1/users/me      → matches before /api/v1/users/*
/api/v1/audit/events  → matches before /api/v1/audit
```

---

## Middleware Chain Order

Middleware executes in a strict order. Each layer can short-circuit (return error) or pass to the next:

```
Request arrives
    │
    ▼
┌─────────────────────────────────────┐
│ 1. CORS Middleware                   │  ← Handles preflight OPTIONS
│    • Allowed origins                 │
│    • Allowed methods                 │
│    • Allowed headers                 │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 2. Rate Limiter (Token Bucket)      │  ← Per-IP and per-user
│    • Check Redis for token count    │
│    • 429 if exceeded                │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 3. Security Headers                 │  ← Adds response headers
│    • X-Content-Type-Options: nosniff│
│    • X-Frame-Options: DENY          │
│    • Strict-Transport-Security      │
│    • X-XSS-Protection               │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 4. Request Body Size Limit (10MB)   │  ← Rejects oversized bodies
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 5. Request Logging (Capture)        │  ← Start timer
│    • Method, path, IP, UA           │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 6. JWT Verification (if required)   │  ← Core auth check
│    • Parse Authorization header     │
│    • Verify HMAC-SHA256 signature   │
│    • Check exp, iat, iss            │
│    • Check Redis for revocation     │
│    • Check Redis for JTI replay     │
│    • 401 if invalid                 │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 7. Tenant Context Extraction        │  ← Get tenant_id
│    • From JWT claim (AUTHORITATIVE) │
│    • Fallback: X-Tenant-ID header   │
│    • Inject into request context    │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 8. Scope Check                      │  ← Authorization
│    • Check required scope for route │
│    • Admin scope for /admin/*       │
│    • 403 if insufficient scope      │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 9. Audit Log Generation              │  ← Creates AuditEvent
│    • Method, path, status, tenant   │
│    • Publishes to NATS async        │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 10. Circuit Breaker                 │  ← Backend health check
│     • Track backend failure rate    │
│     • Open circuit on threshold     │
│     • 503 if circuit open           │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 11. Reverse Proxy                   │  ← Forward to backend
│     • Inject tenant_id (query/body) │
│     • Set X-Forwarded-* headers     │
│     • Forward to backend service    │
│     • Capture response              │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 12. Compression (gzip/brotli)       │  ← Response compression
│     • Based on Accept-Encoding      │
│     • Skip for images, SSE          │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│ 13. Request Logging (Complete)      │  ← Finish timer
│     • Duration, status code         │
│     • Log entry                     │
└─────────────────────────────────────┘
                  │
                  ▼
              Response sent
```

---

## JWT Verification Flow

### Step-by-Step

```
1. Extract Authorization: Bearer <token> header
   ↓ (missing? → 401 Unauthorized)

2. Parse JWT header + payload (base64 decode)
   ↓ (malformed? → 401)

3. Verify signature (HMAC-SHA256)
   ↓ (invalid signature? → 401)

4. Check standard claims:
   • exp (expiration) — not expired?
   • iat (issued at) — in the past?
   • iss (issuer) — matches config?
   ↓ (any fail? → 401)

5. Check Redis:
   • session:<session_id> — session revoked?
   • jti:<token_jti> — token replayed?
   ↓ (revoked or replayed? → 401)

6. Extract custom claims:
   • tenant_id → request context
   • scope → for scope check
   • roles → for RBAC check
   • sub (user_id) → for audit
```

### Token Types

| Type | Purpose | Lifetime | Verification |
|------|---------|----------|-------------|
| Access Token | API requests | 15 minutes | Full JWT verification |
| Refresh Token | Get new access token | 7 days | Redis lookup + JWT |
| Service Token | Machine-to-machine | Configurable | JWT + client_id check |

### JTI Anti-Replay

For sensitive operations (admin, delete), the gateway checks JTI uniqueness:

```go
// Redis SETNX — returns true if key was set (first use)
key := "jti:" + claims.JTI
set, err := redis.SetNX(ctx, key, "1", ttl).Result()
if !set {
    // Token already used — replay attack detected
    return 401
}
```

---

## Tenant Header Injection

### The Problem

Backend services (Policy, Org, Audit) expect `tenant_id` as a **query parameter** or **JSON body field**, not just an HTTP header. The gateway must inject it.

### Solution: Two Injection Points

#### 1. Query Parameter (GET/DELETE requests)

```go
// In proxy.Director for GET requests
if tenantID := getTenantFromContext(r.Context()); tenantID != "" {
    q := r.URL.Query()
    q.Set("tenant_id", tenantID)
    r.URL.RawQuery = q.Encode()
}
```

#### 2. JSON Body Field (POST/PUT/PATCH requests)

```go
// In proxy.Director for POST/PUT/PATCH
if r.Body != nil && isJSONContent(r.Header) {
    body, _ := io.ReadAll(r.Body)
    var data map[string]any
    json.Unmarshal(body, &data)
    data["tenant_id"] = tenantID
    newBody, _ := json.Marshal(data)
    r.Body = io.NopCloser(bytes.NewReader(newBody))
    r.ContentLength = int64(len(newBody))
}
```

### Tenant Priority

```
JWT claim "tenant_id"  ──▶  AUTHORITATIVE (always wins)
X-Tenant-ID header     ──▶  Used only when no JWT (service-to-service)
```

This prevents tenant spoofing: a user cannot change tenants by modifying the header.

---

## Rate Limiting

### Algorithm: Token Bucket

```
Bucket per key = rate_limit:{tenant_id}:{ip}
Capacity:     100 tokens (default)
Refill rate:  100 tokens / minute (default)
```

### Rate Limit Tiers

| Tier | Requests/min | Burst | Applies to |
|------|-------------|-------|------------|
| Unauthenticated | 10 | 20 | No JWT (login, register) |
| Authenticated | 1000 | 200 | Valid JWT |
| Admin | 5000 | 1000 | Admin scope |
| Service Account | 10000 | 2000 | Machine-to-machine |

### Rate Limit Headers

Responses include rate limit information:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests per window |
| `X-RateLimit-Remaining` | Remaining requests in current window |
| `X-RateLimit-Reset` | Unix timestamp when window resets |

### 429 Response

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please retry after 60 seconds.",
  "retry_after": 60
}
```

### Client IP Extraction

```go
func ClientIP(r *http.Request) string {
    // Check X-Forwarded-For first (from load balancer)
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        // First IP in the list is the original client
        if idx := strings.Index(xff, ","); idx > 0 {
            return strings.TrimSpace(xff[:idx])
        }
        return xff
    }
    // Fall back to RemoteAddr (strip port)
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}
```

**Important**: `net.SplitHostPort` strips the port number. Without it, CIDR matching fails (e.g., "192.168.1.50:12345" doesn't match CIDR "192.168.1.0/24").

---

## Circuit Breaker

### Purpose

When a backend service is failing (timeout, 500s), the circuit breaker stops sending requests to it — fail-fast instead of queuing.

### States

```
         failures < threshold
    ┌───────────────────────────┐
    │                           ▼
┌───┴───┐                 ┌──────────┐
│ CLOSED │                 │   OPEN   │
│ (pass) │                 │ (block)  │
└───┬───┘                 └─────┬────┘
    │                           │
    │ failures >= threshold     │ timeout expires
    │                           ▼
    │                     ┌──────────┐
    └─────────────────────│HALF-OPEN │
                          │ (test 1) │
                          └──────────┘
```

| State | Behavior | Transition |
|-------|----------|------------|
| **CLOSED** | Requests pass through to backend | After 5 failures in 10s → OPEN |
| **OPEN** | Requests immediately return 503 | After 30s cooldown → HALF-OPEN |
| **HALF-OPEN** | One test request allowed | Success → CLOSED; Failure → OPEN |

### Configuration

```yaml
circuit_breaker:
  failure_threshold: 5      # failures before opening
  failure_window: 10s       # rolling window for counting
  open_timeout: 30s         # cooldown before half-open
  half_open_max_calls: 1    # test requests in half-open
```

### Per-Backend Circuits

Each backend service has its own circuit breaker:

```
gateway → identity:  CLOSED (healthy)
gateway → auth:      OPEN   (5 failures in 10s)
gateway → oauth:     CLOSED (healthy)
gateway → policy:    CLOSED (healthy)
```

When auth's circuit is open, `/api/v1/auth/login` returns 503 immediately instead of waiting for timeout.

---

## Compression

### Supported Encodings

| Encoding | Priority | Use Case |
|----------|----------|----------|
| Brotli | 1 | Best compression (modern browsers) |
| Gzip | 2 | Universal support |

### Compression Decisions

```
Accept-Encoding: br, gzip
   ↓
1. Is response compressible?
   • Content-Type: text/html, application/json, text/css, application/javascript → YES
   • Content-Type: image/png, video/mp4 → NO (already compressed)
   • SSE streams → NO (streaming, not compressible)
   ↓
2. Response size > 1KB?
   • Small responses not worth compressing
   ↓
3. Apply compression based on client preference
```

---

## Distributed Tracing (OpenTelemetry)

The gateway creates and propagates OpenTelemetry spans:

```
Gateway span: "HTTP GET /api/v1/users"
  ├── Middleware: CORS
  ├── Middleware: Rate Limiter
  ├── Middleware: JWT Verification
  ├── Middleware: Audit Log
  └── Proxy: Identity Service
        └── Identity span: "ListUsers"
```

### Trace Propagation

```
Client → Gateway: traceparent header
Gateway → Backend: traceparent header (propagated)
```

### OTLP Export

```yaml
tracing:
  otlp_endpoint: "http://otel-collector:4318"
  service_name: "ggid-gateway"
  sample_rate: 1.0  # 100% sampling in dev, lower in prod
```

---

## Graceful Shutdown

The gateway supports graceful shutdown to avoid dropping in-flight requests:

```
1. Receive SIGTERM (from Kubernetes/container orchestrator)
2. Stop accepting new connections
3. Wait for in-flight requests to complete (up to 30s)
4. Close idle connections
5. Close database/Redis/NATS connections
6. Exit cleanly (exit code 0)
```

### Health Check During Shutdown

```
normal:   /healthz → 200 OK
draining: /healthz → 503 Service Unavailable (tells LB to stop sending)
```

---

## Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_LISTEN` | `:8080` | Listen address |
| `JWT_SECRET` | (required) | HMAC signing key |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `NATS_URL` | `nats://localhost:4222` | NATS connection |
| `IDENTITY_URL` | `http://localhost:8081` | Identity service |
| `AUTH_URL` | `http://localhost:9001` | Auth service |
| `OAUTH_URL` | `http://localhost:9005` | OAuth service |
| `POLICY_URL` | `http://localhost:8070` | Policy service |
| `ORG_URL` | `http://localhost:8071` | Org service |
| `AUDIT_URL` | `http://localhost:8072` | Audit service |
| `RATE_LIMIT_RPM` | `1000` | Rate limit per minute |
| `RATE_LIMIT_BURST` | `200` | Rate limit burst |
| `CIRCUIT_BREAKER_THRESHOLD` | `5` | Failures before open |
| `CIRCUIT_BREAKER_TIMEOUT` | `30s` | Open circuit cooldown |
| `OTLP_ENDPOINT` | (empty) | OTLP collector URL |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated allowed origins |
| `MAX_BODY_SIZE` | `10485760` | Max request body (10MB) |

---

*Last updated: 2025-07-11*
