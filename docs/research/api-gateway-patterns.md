# API Gateway Patterns in GGID

## 1. Overview

The API Gateway is the single entry point for all client requests in the GGID IAM
suite. It acts as a reverse proxy with a middleware chain that handles JWT
validation, tenant resolution, rate limiting, circuit breaking, and more before
forwarding requests to downstream microservices (auth, oauth, identity, policy,
org, audit).

GGID's gateway (`services/gateway/internal/router/router.go`) is a Go-native
reverse proxy built on `net/http/httputil.ReverseProxy`. Requests flow through
this ordered middleware chain:

```
Client → PanicRecovery → CORS → RequestID → RequestLogger
       → TenantResolver → JWTAuth → [per-route middleware] → Backend
```

This document analyzes six advanced gateway patterns and maps each to GGID's
current implementation, gaps, and a phased roadmap:

| Pattern | Coverage in this doc |
|---|---|
| Per-tenant rate limiting | Section 2 |
| Circuit breaker | Section 3 |
| Canary deployment | Section 4 |
| Blue-green deployment | Section 5 |
| Shadow traffic | Section 6 |
| WASM plugins | Section 7 |

---

## 2. Per-Tenant Rate Limiting

### Why Per-Tenant

In a multi-tenant IAM system, all tenants share the same infrastructure. Without
per-tenant rate limiting, one noisy tenant's traffic spike could exhaust
resources and cause global throttling. Fair usage policies enforce plan-based
limits: free tenants get 100 req/min, pro tenants get 1000 req/min, and
enterprise tenants are unlimited.

### Implementation

GGID implements per-tenant rate limiting across three complementary middleware
files, each using a different algorithm:

**Token bucket** (`token_bucket.go`) — per tenant+IP, burst-capable. Keys
buckets by `tenantID:clientIP` with tier-based capacity overrides:

| Tier | MaxTokens (burst) | RefillPerSec (sustained) | Effective limit |
|---|---|---|---|
| free | 20 | 2 | 120/min |
| pro | 100 | 10 | 600/min |
| enterprise | 1000 | 100 | 6000/min |

**Tier-based fixed window** (`tier_ratelimit.go`) — per tenant, per tier. Simple
counter per window: free=100/min, pro=1000/min, enterprise=0 (unlimited).

**Config store** (`tenant_ratelimit.go`) — per-tenant custom limits with a REST
management API (`GET/PUT/DELETE /api/v1/gateway/ratelimits/{tenant_id}`).

### Current Gaps

| Aspect | State | Gap |
|---|---|---|
| Storage | In-memory maps | Not distributed — lost on restart, not shared across replicas |
| Backend | `sync.Mutex` + maps | No Redis Lua atomic check+increment |
| Global limit | None | No aggregate rate cap across all tenants |

### Redis Lua Script (Recommended)

For distributed deployments, a Redis sorted-set or Lua script provides atomic
check+increment:

```lua
-- KEYS[1] = rate:{tenant_id}:{window}
-- ARGV[1] = limit, ARGV[2] = current timestamp, ARGV[3] = window size
local count = redis.call('ZADD', KEYS[1], ARGV[2], ARGV[2])
redis.call('EXPIRE', KEYS[1], ARGV[3])
if count > tonumber(ARGV[1]) then
    return 0  -- denied
end
return 1     -- allowed
```

---

## 3. Circuit Breaker

### Purpose

When a downstream service (e.g., auth) starts failing, continuing to send
requests wastes resources and cascades failures. The circuit breaker pattern
monitors success/failure rates per backend and trips to fail-fast when error
rates exceed a threshold.

### Implementation

GGID implements a **custom** circuit breaker (not Sony's `gobreaker`) in
`circuitbreaker.go` with the three classic states:

```
          failures >= MaxFailures
     CLOSED ───────────────────────► OPEN
       ▲                                │
       │ successes >= HalfOpenSuccess   │ cooldown elapsed (Timeout)
       │                                ▼
     HALF-OPEN ◄──────────────────── HALF-OPEN
       │  probe fails → re-open
       ▼
```

The `CircuitConfig` defaults: `MaxFailures=5`, `Timeout=30s`,
`HalfOpenMax=3` trial requests, `HalfOpenSuccess=2` to close.

The `CircuitRegistry` manages one breaker per backend prefix (`map[string]*CircuitBreaker`).
`CircuitMiddleware` wraps each downstream route. When open, it returns `503 Service
Unavailable` with `X-Circuit-State: open`.

### Per-Service Breakers

Each downstream service (auth, oauth, identity, policy, org, audit) gets its own
breaker via `registry.Get(prefix)`. This isolation ensures that one failing
service doesn't trip all breakers. The `AllStats()` method returns a snapshot of
all breaker states for ops dashboards.

### Current Gaps

| Aspect | State | Gap |
|---|---|---|
| Integration | Middleware defined | Not wired into the active middleware chain in `Handler()` |
| Failure detection | HTTP 5xx only | No timeout/network-error counting |
| Metrics | `CircuitStats` available | Not exported to Prometheus |

---

## 4. Canary Deployment

### Pattern

Route a small percentage of traffic to a new version (v2) of a backend service.
Gradually increase from 1% to 10% to 50% to 100% if healthy. Roll back to 0%
immediately if error rates spike.

### Implementation

GGID implements canary routing in `canary.go`:

```go
type CanaryConfig struct {
    StableURL  string // primary backend
    CanaryURL  string // new version backend
    Percentage int    // 0–100
    Header     string // force canary via header (e.g., "X-Canary")
    CookieName string // sticky canary cookie
}

func (cr *CanaryRouter) ShouldRouteCanary(cfg *CanaryConfig, r *http.Request) bool {
    // 1. Header override — X-Canary: true forces canary
    if cfg.Header != "" && r.Header.Get(cfg.Header) == "true" {
        return true
    }
    // 2. Sticky cookie — "canary" value pins to canary
    if cfg.CookieName != "" {
        if c, _ := r.Cookie(cfg.CookieName); c != nil && c.Value == "canary" {
            return true
        }
    }
    // 3. Percentage-based — deterministic counter
    n := cr.counter.Add(1)
    return int(n%100) < cfg.Percentage
}
```

The routing decision is deterministic per-request using an atomic counter
(`n%100 < percentage`), avoiding the non-uniformity of `math/rand`. Sticky
cookies ensure the same client always sees the same version.

### Current Gaps

| Aspect | State | Gap |
|---|---|---|
| Routing logic | Implemented | Not integrated into proxy director |
| Health monitoring | None | No automatic rollback on canary error spike |
| Weighted routing | Percentage only | No header/tenant-based canary selection |

---

## 5. Blue-Green Deployment

### Pattern

Maintain two identical environments: blue (current production) and green (new
release). Switch all traffic from blue to green instantly. If problems appear,
switch back to blue for instant rollback.

### Implementation

GGID does not implement blue-green as a named feature, but the infrastructure
supports it via the **hot reload** API:

```
POST /api/v1/gateway/routes/reload → rebuilds all proxies from config
POST /api/v1/admin/routes/{prefix}/toggle → enable/disable individual routes
```

A blue-green workflow:
1. Deploy green instances alongside blue
2. Update config: change `routes.auth` from `http://blue:9001` to `http://green:9001`
3. Call `POST /api/v1/gateway/routes/reload` (zero downtime, no process restart)
4. If problems: revert config + reload again

### Comparison: Blue-Green vs Canary

| Aspect | Blue-Green | Canary |
|---|---|---|
| Traffic shift | All at once | Gradual (1%→100%) |
| Rollback speed | Instant (config flip) | Gradual (reduce %) |
| Risk | Higher (all users affected) | Lower (subset exposed) |
| Complexity | Low | Medium |
| Best for | Small deployments, stateless services | Large user bases, risk-averse orgs |

---

## 6. Shadow Traffic

### Pattern

Mirror production traffic to a shadow backend that processes requests but
discards responses. This tests new versions under real load without affecting
users. Discrepancies between production and shadow responses indicate bugs.

### Implementation

GGID implements shadow traffic mirroring in `shadow.go`. The
`ShadowTrafficConfig` specifies `ShadowBackend`, `Percentage` (0-100), optional
method filter, and timeout. The middleware is fire-and-forget:

```go
func ShadowMiddleware(mirror *ShadowTrafficMirror) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if mirror.shouldMirror(r) {
                go mirror.sendShadow(r)  // async, response discarded
            }
            next.ServeHTTP(w, r)  // original request unaffected
        })
    }
}
```

Key design decisions:
- **Asynchronous**: Shadow requests run in a goroutine — the original request
  is never delayed
- **Fire-and-forget**: Shadow responses are always discarded
- **Per-request override**: `X-Shadow-Backend` header redirects to a custom
  shadow URL
- **Stats tracking**: `ShadowStats` records total mirrored, errors, and latency

### Use Cases

- Pre-production validation with real production data
- Performance testing: can the new version handle prod load?
- Schema migration validation: does a new DB schema break existing queries?

---

## 7. WASM Plugins

### Pattern

Gateway middleware as WebAssembly modules — write once in any language (Rust,
Go, AssemblyScript), run in a sandboxed VM with no filesystem or network access
beyond what the host explicitly provides.

### Implementation

GGID uses **wazero** — a pure-Go WASM runtime with zero CGO dependencies (ideal
for GGID's Go-native architecture). Implementation lives in `wasm_plugin.go`.

The `WasmPluginHost` manages the wazero runtime and a `map[string]*loadedPlugin`.
Each request, the middleware serializes a `PluginContext` (method, path, headers,
body, tenantID, userID) to JSON, writes it into the WASM module's memory, and
calls the exported `on_request` function. The plugin returns a `PluginResult`
with `ShouldBlock`, `StatusCode`, and `BlockReason` fields.

Plugin lifecycle:
1. `LoadPlugin()` compiles the `.wasm` file and extracts metadata via `get_metadata`
2. `Execute()` instantiates the module, writes JSON context to plugin memory,
   calls `on_request`/`on_response`, reads the result back
3. `WasmMiddleware` runs request-phase plugins in sequence; if any returns
   `ShouldBlock`, the request is rejected

### Comparison: WASM Approaches

| Aspect | wazero (GGID) | Envoy WASM | wasmer-go |
|---|---|---|---|
| Language | Pure Go | C++ | C/C via CGO |
| CGO required | No | N/A (separate process) | Yes |
| Performance | ~0.5-2ms overhead | Lowest | Low |
| WASI support | Yes | Yes | Yes |
| Hot reload | Manual (re-LoadPlugin) | Via xDS | Manual |

### GGID Considerations

- Per-tenant plugins: load a tenant-specific WASM module dynamically based on
  the `TenantID` in `PluginContext`
- Security: WASM sandbox guarantees plugins cannot access the host filesystem
  or network
- The `alloc` export function is required for memory management between host
  and guest

---

## 8. GGID Current Gateway Analysis

| Pattern | File(s) | Implemented? | Maturity | Gap |
|---|---|---|---|---|
| Per-tenant rate limiting | `token_bucket.go`, `tier_ratelimit.go`, `tenant_ratelimit.go` | Yes | High | In-memory only; no Redis backend for distributed enforcement |
| Circuit breaker | `circuitbreaker.go` | Yes | Medium | Custom impl (not gobreaker); not wired into active middleware chain; 5xx-only failure detection |
| Canary routing | `canary.go` | Yes | Medium | Routing logic exists but not integrated into proxy director; no auto-rollback |
| Blue-green | `router.go` (hot reload) | Partial | Low | Achievable via config reload but no first-class blue-green abstraction |
| Shadow traffic | `shadow.go` | Yes | Medium | Async mirroring works; no response comparison/diffing |
| WASM plugins | `wasm_plugin.go` | Yes | Medium | wazero integration complete; no hot-reload on file change; not in default chain |

### Gateway Middleware Chain (current active chain)

```
PanicRecovery → CORS → RequestID → RequestLogger → TenantResolver
→ JWTAuth(required?) → ReverseProxy(matchBackend by longest prefix)
```

Not yet in the active chain (available but not wired in `Handler()`):
- `TenantBucketLimiter.Middleware`
- `TierRateLimiter.Middleware`
- `CircuitMiddleware`
- `CanaryRouter` (not applied in director)
- `ShadowMiddleware`
- `WasmMiddleware`

---

## 9. Roadmap

| Phase | Task | Priority | Effort | Dependency |
|---|---|---|---|---|
| 1 | Wire `TenantBucketLimiter` into the active middleware chain | P0 | 1 day | None |
| 2 | Add Redis backend for distributed rate limiting | P0 | 3 days | Redis infrastructure |
| 3 | Wire `CircuitMiddleware` per-backend and export Prometheus metrics | P1 | 2 days | Phase 1 |
| 4 | Integrate `CanaryRouter` into the reverse proxy director | P2 | 3 days | None |
| 5 | Add auto-rollback for canary based on error-rate threshold | P2 | 2 days | Phase 4 |
| 6 | Implement response diffing for shadow traffic | P3 | 3 days | None |
| 7 | WASM hot-reload on file change + admin API for plugin management | P3 | 3 days | None |

**Effort summary**: Phases 1-3 are production-readiness items (~6 days). Phases
4-7 are advanced deployment capabilities (~11 days). Total: ~17 engineering days.

### Architecture Diagram: Target State

```
                      ┌──────────────────────────────────────────┐
                      │              API Gateway                  │
                      │                                          │
  Client ───────────► │  PanicRecovery → CORS → RequestID        │
                      │     → RequestLogger → TenantResolver     │
                      │     → JWTAuth                            │
                      │     → WASM Plugins (request phase)       │
                      │     → Per-Tenant Rate Limit (Redis)      │
                      │     → Circuit Breaker (per backend)      │
                      │     → Shadow Traffic (async mirror)      │
                      │     → Canary Router                      │
                      │     → ReverseProxy → Backend             │
                      │                                          │
                      │  Admin API: /routes, /stats, /reload     │
                      │  Metrics:  /metrics (Prometheus)         │
                      └──────────────────────────────────────────┘
                           │          │          │
                      ┌────▼───┐ ┌────▼───┐ ┌────▼───┐
                      │  Auth  │ │  OAuth │ │ Policy │  ... (6 backends)
                      └────────┘ └────────┘ └────────┘
```
