# Design: Gateway Architecture

> **Status:** Implemented

Complete architecture of the GGID API Gateway — middleware chain, routing,
load balancing, circuit breaking, rate limiting, caching, and observability.

---

## Overview

```
                           Client Request
                                │
                                ▼
┌───────────────────────────────────────────────────────────┐
│                        API Gateway                        │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                  Middleware Chain                   │ │
│  │                                                     │ │
│  │  Request ──► Recovery ──► RequestID ──► Logging    │ │
│  │       ──► CORS ──► RateLimit ──► BodyLimit         │ │
│  │       ──► Compress ──► JWT Auth ──► TenantInject   │ │
│  │       ──► OTel Trace ──► BotDetect ──► IPAllow     │ │
│  │       ──► [Slow Detect]                            │ │
│  └─────────────────────────────────────────────────────┘ │
│                          │                                │
│                          ▼                                │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                    Router                           │ │
│  │                                                     │ │
│  │  Match path prefix ──► Select backend              │ │
│  │  Coalesce (singleflight) ──► Shadow traffic?       │ │
│  │  Canary routing? ──► Health-weighted LB            │ │
│  └─────────────────────────────────────────────────────┘ │
│                          │                                │
│                          ▼                                │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                 Reverse Proxy                       │ │
│  │                                                     │ │
│  │  Circuit Breaker ──► HTTP Transport (keep-alive)   │ │
│  │  ──► Forward to Backend ──► Response               │ │
│  └─────────────────────────────────────────────────────┘ │
│                          │                                │
│                          ▼                                │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                  Post-Process                       │ │
│  │                                                     │ │
│  │  Record Health ──► Metrics ──► Audit ──► Response  │ │
│  └─────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────┘
                                │
                                ▼
                        Backend Service
```

---

## Middleware Chain

Ordered execution (each middleware wraps the next):

| # | Middleware | Purpose | Bypass for Public |
|---|-----------|---------|:-:|
| 1 | **Panic Recovery** | Catches panics, returns 500 with request_id | No |
| 2 | **Request ID** | Injects `X-Request-ID` header (or accepts inbound) | No |
| 3 | **Structured Logging** | JSON log: method, path, status, duration, IP | No |
| 4 | **CORS** | Handles preflight OPTIONS, adds headers | Yes (OPTIONS) |
| 5 | **Rate Limiter** | Fixed-window per-IP rate limiting | No |
| 6 | **Body Size Limit** | Rejects oversized request bodies | No |
| 7 | **Compression** | gzip response if `Accept-Encoding: gzip` | No |
| 8 | **JWT Authentication** | Verifies RS256 signature via JWKS | Yes (public paths) |
| 9 | **Tenant Injection** | Extracts `X-Tenant-ID`, injects query/body | Yes (no-auth) |
| 10 | **OTel Tracing** | Creates W3C traceparent span | No |
| 11 | **Bot Detection** | Blocks known bots (configurable UA denylist) | No |
| 12 | **IP Allowlist** | Restricts access to CIDR ranges | No |
| 13 | **Slow Request Detector** | Logs requests exceeding threshold | No |

### Public Paths (JWT Bypassed)

```
/healthz
/healthz/*
/.well-known/*
/api/v1/auth/login
/api/v1/auth/register
/api/v1/auth/refresh
/api/v1/auth/password/forgot
/api/v1/auth/password/reset
/api/v1/auth/magic-link
/api/v1/auth/magic-link/verify
/api/v1/auth/email/verify
/api/v1/auth/social/*
/oauth/authorize
/oauth/token
/saml/*
/login
/register
/forgot-password
```

---

## Routing Strategy

### Route Table

The Gateway matches request paths by prefix to backend services:

| Path Prefix | Backend | Port |
|-------------|---------|------|
| `/api/v1/auth` | auth | 9001 |
| `/api/v1/users` | identity | 8080 |
| `/api/v1/roles` | policy | 8070 |
| `/api/v1/permissions` | policy | 8070 |
| `/api/v1/policies` | policy | 8070 |
| `/api/v1/orgs` | org | 8071 |
| `/api/v1/audit` | audit | 8072 |
| `/oauth/*` | oauth | 9005 |
| `/saml/*` | oauth | 9005 |
| `/scim/v2/*` | identity | 8080 |
| `/api/v1/idp/*` | auth | 9001 |

### Route Resolution

```
1. Match longest prefix
2. Check if path matches a public route (skip JWT)
3. Select backend(s) for the matched prefix
4. Apply coalescing (if GET and singleflight key matches)
5. Apply shadow traffic (if X-Shadow-Backend header present)
6. Apply canary routing (if X-Canary-Percent configured)
7. Health-weighted selection among healthy backends
8. Forward via reverse proxy
```

### Request Coalescing

For identical concurrent GET requests, the Gateway collapses them into one
backend call using `golang.org/x/sync/singleflight`:

```go
key := r.Method + ":" + r.URL.Path
result := sf.Do(key, func() (interface{}, error) {
    return proxy(r)  // only first caller hits backend
})
// All callers receive the same response
```

---

## Load Balancing

### Health-Weighted Selection

When multiple backend instances exist for a service:

```
Backend A: score=90, weight=0.90
Backend B: score=80, weight=0.80
Backend C: score=0  (unhealthy), weight=0.10 (minimum)

Traffic distribution: A=~53%, B=~47%, C=0% (below threshold)
```

Score is computed from:
- **Success rate** (0-70 points)
- **Average latency** (0-30 points)
- **Decay factor** (multiplied by 0.95)

### Health Scoring

```go
type HealthScore struct {
    backends map[string]*backendHealth
    // Each backend tracks:
    // - totalReqs, successReqs, errorReqs
    // - totalLatency, lastUpdate
    // - score (0-100), weight (0.1-1.0)
}
```

- `RecordSuccess(backend, latency)` — called on 2xx/3xx
- `RecordError(backend)` — called on 5xx or timeout
- `Score(backend)` — returns current health score
- `IsHealthy(backend, threshold)` — true if score >= threshold (default 50)

---

## Circuit Breaker

Per-backend circuit breaker prevents cascading failures:

```
States:
  CLOSED    ──► errors exceed threshold ──► OPEN
  OPEN      ──► cooldown elapsed ──► HALF_OPEN
  HALF_OPEN ──► probe succeeds ──► CLOSED
  HALF_OPEN ──► probe fails ──► OPEN
```

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MaxRequests` | 32 | Max requests allowed in HALF_OPEN |
| `Interval` | 10s | Period in CLOSED before clearing counts |
| `Timeout` | 30s | Duration of OPEN before HALF_OPEN |
| `TripThreshold` | 0.6 | Error rate to trip (60%) |
| `MinRequests` | 10 | Minimum requests before evaluation |

When a circuit is OPEN:
- Requests are not forwarded to that backend
- Returns `503 Service Unavailable` with `Retry-After` header
- Other healthy backends are preferred

---

## Rate Limiting

### Configuration

| Scope | Default | Environment |
|-------|---------|-------------|
| Login (per IP) | 5 req/min | Fixed in middleware |
| Register (per IP) | 3 req/min | Fixed in middleware |
| General API (per IP) | 100 req/min | Configurable |

### Implementation

Fixed-window counter (in-memory per Gateway instance):

```go
type RateLimiter struct {
    mu       sync.Mutex
    buckets  map[string]*bucket  // key: IP + endpoint pattern
}

type bucket struct {
    count    int
    window   time.Time  // reset time
}
```

> For multi-instance production deployments, use Redis-backed rate limiting
> to share state across Gateway replicas.

---

## Response Caching

The Gateway does not cache application data (to avoid stale results).
However, it optimizes:

1. **JWKS caching** — Public keys cached for 15 minutes (configurable)
2. **Connection reuse** — TCP keep-alive to backends (up to 20 idle conns/host)
3. **HTTP/2** — Automatic upgrade to backends when supported

---

## Observability

### OpenTelemetry Tracing

```
Client Request
  │
  ├─► Gateway Span (root)
  │     ├─► JWT Verification Span
  │     ├─► Backend Proxy Span
  │     │     ├─► Auth Service Span (if propagated)
  │     │     └─► Database Span (if propagated)
  │     └─► Response Span
```

- **Propagation:** W3C `traceparent` header
- **Exporter:** OTLP HTTP to `OTEL_EXPORTER_OTLP_ENDPOINT`
- **Auto-instrumented:** HTTP method, path, status, duration

### Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `ggid_http_requests_total` | Counter | method, path, status |
| `ggid_http_request_duration_seconds` | Histogram | method, path |
| `ggid_jwt_verifications_total` | Counter | result |
| `ggid_backend_healthy` | Gauge | backend |
| `ggid_rate_limit_hits_total` | Counter | path |
| `ggid_circuit_breaker_state` | Gauge | backend |

### Structured Logging

```json
{
  "timestamp": "2024-07-10T12:00:00Z",
  "level": "info",
  "service": "gateway",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "duration": "8.2ms",
  "remote_addr": "192.168.1.100",
  "request_id": "req-abc123"
}
```

### Custom Error Pages

For backend failures (502/503/504), the Gateway returns a consistent JSON error:

```json
{
  "error": "Service temporarily unavailable",
  "request_id": "req-abc123",
  "retry_after": 30
}
```

---

## Graceful Shutdown

```
1. Receive SIGTERM
2. Stop accepting new connections
3. Wait up to 30s for in-flight requests
4. Close idle connections
5. Close backend connections
6. Exit
```

Health check endpoint returns `503` during shutdown so the load balancer
stops sending traffic.
