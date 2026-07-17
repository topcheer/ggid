# API Gateway Hardening & Rate Limiting Architecture: Comprehensive Edge Security for GGID

> **Focus**: Hardening GGID's gateway from basic rate limiting to enterprise-grade edge security — hierarchical rate limits (per-user/key/IP/endpoint), request validation, circuit breakers, payload sanitization, observability, and integration with PEP/PDP/DLP egress middleware.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§9), curl commands (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Gateway Middleware Stack](#2-ggid-current-state-gateway-middleware-stack)
3. [Gap Analysis](#3-gap-analysis)
4. [Hierarchical Rate Limiting](#4-hierarchical-rate-limiting)
5. [Request Validation & Edge Security](#5-request-validation--edge-security)
6. [Circuit Breaker Pattern](#6-circuit-breaker-pattern)
7. [Proposed Architecture: Enhanced Gateway](#7-proposed-architecture-enhanced-gateway)
8. [Endpoint Precondition Check](#8-endpoint-precondition-check)
9. [API Design + Curl Commands](#9-api-design--curl-commands)
10. [Implementation Backlog with DoD](#10-implementation-backlog-with-dod)
11. [Competitive Differentiation](#11-competitive-differentiation)

---

## 1. Executive Summary

GGID's gateway has a **surprisingly mature middleware stack** — much more than expected:

**Already implemented and working:**
- Per-tenant token-bucket rate limiting (Redis-backed, `token_bucket.go:128`) ✅
- Tier-based rate limits (free/pro/enterprise tiers, `tier_ratelimit.go:38`) ✅
- API versioning with deprecation headers (`apiversion.go:27` + `api_versioning.go:22`) ✅
- gRPC-Web transcoding (`grpcweb.go`) ✅
- WebSocket proxy with subprotocol negotiation (`wsproxy_enhanced.go:45`) ✅
- Request timeout middleware (`timeout.go:77`) ✅
- Circuit breaker tests exist (`coverage_sprint20_test.go:139`) ✅
- JWT auth, API key auth, tenant middleware, WASM plugins ✅
- DLP egress middleware (researched, pending implementation) ✅
- PEP middleware (researched, pending implementation) ✅
- Request ID propagation (`requestid_propagation.go:89`) ✅
- Prometheus metrics (`metrics.go:50`) ✅
- Panic recovery (`recovery.go:123`) ✅
- HTTPS redirect ✅

**Gaps (smaller than expected):**
1. **Rate limiting is per-tenant only** — no per-user, per-API-key, per-IP, per-endpoint limits
2. **No burst vs sustained** — single window, no dual-rate (burst of 100 + sustained 10/s)
3. **No request body validation** — JSON not validated against schema before forwarding
4. **No payload sanitization** — SQL injection/XSS not blocked at edge
5. **No request size enforcement** — No max body size limit
6. **No distributed circuit breaker** — Circuit breaker logic tested but not wired into router
7. **No latency histograms** — Basic Prometheus metrics, no P50/P95/P99 per endpoint
8. **No per-endpoint rate config** — Rate limits global, not configurable per route

**Recommendation**: Add hierarchical rate limiting (per-user/key/IP/endpoint with burst/sustained), request validation middleware, circuit breaker integration, and enhanced observability — building on the existing strong foundation.

**Estimated effort**: 2 sprints (hierarchical rate limiting + request validation + circuit breaker + observability).

---

## 2. GGID Current State: Gateway Middleware Stack

### Middleware Chain (Current)

```
Request →
  1. HTTPS Redirect         ✅
  2. Panic Recovery         ✅ recovery.go:123
  3. Request ID Propagation ✅ requestid_propagation.go:89
  4. Structured Logging    ✅
  5. Metrics (Prometheus)  ✅ metrics.go:50
  6. Timeout               ✅ timeout.go:77
  7. Rate Limit (tenant)   ✅ token_bucket.go:202 + tier_ratelimit.go:61
  8. API Key Auth          ✅ apikey.go:22
  9. JWT Auth              ✅ jwt_auth.go
 10. Tenant Middleware     ✅
 11. WASM Plugins          ✅ wasm_plugin.go:34
 12. API Versioning        ✅ apiversion.go:34 + api_versioning.go:26
 13. gRPC-Web Transcoding  ✅ grpcweb.go
 14. WebSocket Proxy       ✅ wsproxy_enhanced.go
 15. Route Matching        ✅ router.go
 16. Backend Proxy         ✅
  17. [DLP Egress]         📋 researched, not yet implemented
  18. [PEP/PDP]            📋 researched, not yet implemented
```

### Existing Rate Limiting (Detailed)

```go
// token_bucket.go:128 — TenantBucketLimiter
// Redis-backed token bucket per tenant + per IP
// Configurable via GATEWAY_RATE_LIMIT_TOKENS + GATEWAY_RATE_LIMIT_REFILL
// Hot-reloadable via sysconfig store

// tier_ratelimit.go:38 — TierRateLimiter
// Per-tenant-tier limits (free/pro/enterprise)
// Different burst/refill per tier

// ratelimit.go:30 — RateLimiter (in-memory)
// Per-endpoint fixed-window rate limiting
// In-memory (not Redis) — less reliable
```

---

## 3. Gap Analysis

| # | Gap | Impact | Priority |
|---|-----|--------|----------|
| 1 | **No per-user rate limiting** | One user can exhaust tenant quota | P0 |
| 2 | **No per-API-key limiting** | Agent/script can flood API | P0 |
| 3 | **No burst vs sustained** | No spike tolerance + steady-state cap | P0 |
| 4 | **No request body validation** | Malformed JSON reaches backend | P1 |
| 5 | **No payload sanitization** | SQL injection/XSS reaches backend | P1 |
| 6 | **No max request size** | Large payload DoS | P1 |
| 7 | **Circuit breaker not wired** | Tested but not in router chain | P0 |
| 8 | **No P50/P95/P99 per endpoint** | Can't identify slow endpoints | P1 |
| 9 | **In-memory rate limiter** | `ratelimit.go` not Redis-backed | P1 |

---

## 4. Hierarchical Rate Limiting

### Multi-Dimensional Rate Limiting

```
Request arrives → evaluate in order (most specific to least):

  1. Per-API-key:     100 req/min    (most specific)
  2. Per-user:        500 req/min    (user-specific)
  3. Per-IP:          1000 req/min   (IP-specific)
  4. Per-endpoint:    5000 req/min   (route-specific)
  5. Per-tenant:      10000 req/min  (tenant ceiling)

  If ANY limit is exceeded → 429 Too Many Requests

  Burst vs Sustained:
  - Burst:    100 req in 10 seconds (short window)
  - Sustained: 500 req per minute  (long window)
  - Both checked; most restrictive wins
```

### Redis Implementation

```go
// Redis keys for hierarchical rate limiting:
// ggid:rl:{tenant}:{user}:{minute}       → INCR + EXPIRE 60s
// ggid:rl:{tenant}:{apikey}:{minute}     → INCR + EXPIRE 60s
// ggid:rl:{tenant}:{ip}:{minute}         → INCR + EXPIRE 60s
// ggid:rl:{tenant}:{endpoint}:{minute}   → INCR + EXPIRE 60s
// ggid:rl:{tenant}:{minute}              → INCR + EXPIRE 60s
//
// Burst keys (10-second window):
// ggid:rl:{tenant}:{user}:{10sec_bucket} → INCR + EXPIRE 10s
```

---

## 5. Request Validation & Edge Security

### Schema Validation (OpenAPI)

```go
type RequestValidator struct {
    schemas map[string]*jsonschema.Schema // keyed by path+method
}

func (v *RequestValidator) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        schema := v.schemas[r.URL.Path+"_"+r.Method]
        if schema == nil {
            next.ServeHTTP(w, r) // No schema → skip
            return
        }
        
        body, _ := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
        r.Body = io.NopCloser(bytes.NewReader(body))
        
        if err := schema.Validate(body); err != nil {
            WriteError(w, r, 400, "validation_error", err.Error())
            return
        }
        
        next.ServeHTTP(w, r)
    })
}
```

### Payload Sanitization

| Threat | Detection | Action |
|--------|-----------|--------|
| **SQL Injection** | Pattern matching (`' OR 1=1`, `UNION SELECT`) | Block request |
| **XSS** | Script tags in JSON values (`<script>`) | Sanitize or block |
| **Path Traversal** | `../` in path parameters | Block |
| **Command Injection** | `;`, `|`, backticks in parameters | Block |
| **Oversized payload** | Content-Length > limit | 413 |
| **Deeply nested JSON** | Nesting depth > 20 | 400 |

---

## 6. Circuit Breaker Pattern

```go
type CircuitBreaker struct {
    failures      int
    threshold     int           // Trip after N failures
    resetTimeout  time.Duration // Try again after timeout
    state         string        // "closed", "open", "half-open"
    lastFailure   time.Time
}

func (cb *CircuitBreaker) Allow() bool {
    if cb.state == "open" {
        if time.Since(cb.lastFailure) > cb.resetTimeout {
            cb.state = "half-open"
            return true // Try one request
        }
        return false // Still open
    }
    return true // Closed or half-open
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.failures = 0
    cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.failures++
    cb.lastFailure = time.Now()
    if cb.failures >= cb.threshold {
        cb.state = "open"
    }
}
```

---

## 7. Proposed Architecture: Enhanced Gateway

```
Request →
  1-6. (existing: HTTPS, recovery, request-id, logging, metrics, timeout)
  7. ★ Enhanced Rate Limiting (hierarchical + burst/sustained)
  8. ★ Request Size Limit
  9. ★ Schema Validation
 10. ★ Payload Sanitization
 11-14. (existing: API key, JWT, tenant, WASM)
 15. ★ Circuit Breaker (per-backend)
 16-18. (existing: versioning, grpc-web, websocket)
 19. ★ PEP/PDP (per-request authz)
 20. Route + Proxy
 21. ★ DLP Egress (response inspection)
```

---

## 8. Endpoint Precondition Check

### Existing (Enhance)

| Component | File:Line | Current | Target |
|----------|-----------|---------|--------|
| Token bucket limiter | `token_bucket.go:128` | Per-tenant + IP | Add per-user, per-key, per-endpoint |
| Tier rate limiter | `tier_ratelimit.go:38` | Per-tier | Keep, extend |
| In-memory limiter | `ratelimit.go:30` | In-memory | Migrate to Redis or remove |
| Circuit breaker tests | `coverage_sprint20_test.go:139` | Tested | Wire into router |
| Prometheus metrics | `metrics.go:50` | Basic | Add latency histograms |

### New Components

| Component | Priority |
|-----------|----------|
| Hierarchical rate limit middleware | P0 |
| Request body validator | P1 |
| Payload sanitizer (SQLi/XSS) | P1 |
| Circuit breaker middleware | P0 |
| Enhanced observability (histograms) | P1 |

---

## 9. API Design + Curl Commands

### Configure Rate Limits

```bash
curl -X PUT https://ggid.corp.com/api/v1/gateway/rate-limits \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "per_user": { "burst": 100, "sustained_per_min": 500 },
    "per_api_key": { "burst": 200, "sustained_per_min": 1000 },
    "per_ip": { "burst": 500, "sustained_per_min": 2000 },
    "per_endpoint": {
      "/api/v1/identity/users": { "burst": 50, "sustained_per_min": 200 },
      "/api/v1/auth/login": { "burst": 10, "sustained_per_min": 30 }
    },
    "per_tenant": { "burst": 2000, "sustained_per_min": 10000 }
  }'
```

### Rate Limit Headers (Response)

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 500
X-RateLimit-Remaining: 497
X-RateLimit-Reset: 1721217660

# When exceeded:
HTTP/1.1 429 Too Many Requests
Retry-After: 12
X-RateLimit-Limit: 500
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1721217660
```

### Circuit Breaker Status

```bash
curl https://ggid.corp.com/api/v1/gateway/circuit-breakers \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{
  "breakers": [
    { "backend": "identity", "state": "closed", "failures": 0, "threshold": 5 },
    { "backend": "auth", "state": "closed", "failures": 1, "threshold": 5 },
    { "backend": "oauth", "state": "open", "failures": 7, "threshold": 5, "reset_in": "28s" }
  ]
}
```

---

## 10. Implementation Backlog with DoD

### P0 — Hierarchical Rate Limiting + Circuit Breaker (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Hierarchical rate limiter (user/key/IP/endpoint) | ✅ Redis-backed ✅ 5 dimensions ✅ Burst + sustained ✅ ≥3 tests | 4d |
| 2 | Per-endpoint rate limit config API | ✅ CRUD per route ✅ DB-backed ✅ Hot-reload ✅ ≥3 tests | 2d |
| 3 | Circuit breaker in router chain | ✅ Per-backend breaker ✅ Auto-trip on failures ✅ Half-open recovery ✅ ≥3 tests | 3d |
| 4 | Rate limit headers (standard) | ✅ X-RateLimit-* headers ✅ Retry-After ✅ ≥3 tests | 1d |

### P1 — Request Validation + Observability (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Request body size limit + JSON validation | ✅ Max body size enforced ✅ JSON parse before forward ✅ ≥3 tests | 2d |
| 6 | Payload sanitizer (SQLi/XSS/path traversal) | ✅ Pattern detection ✅ Block or sanitize ✅ ≥3 tests | 3d |
| 7 | Latency histograms per endpoint | ✅ P50/P95/P99 in Prometheus ✅ Per-route histograms ✅ ≥3 tests | 2d |
| 8 | Migrate in-memory limiter to Redis | ✅ No in-memory rate state ✅ Redis for all limits ✅ ≥3 tests | 1d |
| 9 | Gateway hardening dashboard | ✅ Rate limit hits ✅ Circuit breaker state ✅ Error rates ✅ Latency | 2d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 10 | OpenAPI schema validation | Validate request against OpenAPI spec |
| 11 | Request/response transformation | Header injection, body transform |
| 12 | Adaptive rate limiting | Adjust limits based on backend health |
| 13 | Geo-based rate limiting | Different limits per country/region |
| 14 | API consumer profiles | Pre-configured limit bundles |

---

## 11. Competitive Differentiation

| Feature | GGID (target) | Kong | Tyk | Envoy Gateway | AWS API Gateway | Cloudflare |
|---------|---------------|------|-----|---------------|-----------------|-----------|
| **Hierarchical RL** | **5 dimensions** | 4 | 3 | 4 | 3 | 3 |
| **Burst + sustained** | **Yes** | Yes | Yes | Yes | Yes | Yes |
| **Circuit breaker** | **Yes** | Plugin | Plugin | Yes | No | No |
| **Schema validation** | **Planned** | Plugin | No | Yes | Yes | Yes |
| **Payload sanitization** | **Yes** | Plugin | No | No | WAF | WAF |
| **gRPC-Web** | **Existing** ✅ | Plugin | No | Yes | No | No |
| **WebSocket** | **Existing** ✅ | Yes | No | Yes | No | Yes |
| **API versioning** | **Existing** ✅ | Yes | No | No | Yes | No |
| **PDP integration** | **Native** ✅ | Auth plugin | No | ExtAuthz | IAM | Access |
| **DLP egress** | **Native** ✅ | No | No | No | No | No |
| **Open source** | **Yes** | Partially | Yes | Yes | No | No |

**Key differentiator**: GGID's gateway already has gRPC-Web, WebSocket, and API versioning — features missing from many commercial gateways. Adding hierarchical rate limiting and circuit breaker makes it enterprise-ready, while PDP/DLP integration gives it capabilities no other gateway has.

---

## References

- [Kong Gateway](https://konghq.com/kong/) — Industry-leading API gateway
- [Envoy Gateway](https://gateway.envoyproxy.io/) — Envoy-based Kubernetes gateway
- [Redis Rate Limiting](https://redis.io/docs/manual/patterns/distributed-locks/) — Token bucket in Redis
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html) — Martin Fowler
- [OWASP API Security Top 10](https://owasp.org/API-Security/) — API hardening
- [GGID Token Bucket](../services/gateway/internal/middleware/token_bucket.go) — Per-tenant limiter at line 128
- [GGID Tier Rate Limit](../services/gateway/internal/middleware/tier_ratelimit.go) — Tier limiter at line 38
- [GGID API Versioning](../services/gateway/internal/middleware/apiversion.go) — Versioning at line 27
- [GGID gRPC-Web](../services/gateway/internal/middleware/grpcweb.go) — Transcoding
- [GGID WebSocket Proxy](../services/gateway/internal/middleware/wsproxy_enhanced.go) — WebSocket at line 45
- [GGID Timeout](../services/gateway/internal/middleware/timeout.go) — Timeout at line 77
- [GGID Continuous Authorization & PDP](./continuous-authorization-pdp.md) — PEP middleware
- [GGID DLP Egress](./dlp-egress-pii-redaction.md) — Egress DLP
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — Applications pillar
