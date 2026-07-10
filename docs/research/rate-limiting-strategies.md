# Rate Limiting Strategies for IAM Systems

## 1. Overview

Rate limiting controls the rate of incoming requests to protect services from
abuse, credential stuffing, and infrastructure overload. In an Identity and
Access Management (IAM) system, rate limiting is a first-line defense:

- **Brute-force protection**: throttle login attempts to defeat password guessing.
- **Credential stuffing**: limit repeated auth attempts from a single source.
- **Resource protection**: prevent one noisy tenant from starving shared infrastructure.
- **API fairness**: enforce per-tier quotas (free vs. enterprise).

GGID applies rate limiting at the **gateway layer** (`middleware/ratelimit.go`
and `middleware/token_bucket.go`), which is the correct single chokepoint for
all inbound traffic.

Key dimensions to consider:

| Dimension     | Key            | Use Case                          |
|---------------|----------------|-----------------------------------|
| Per-IP        | client IP      | anonymous abuse, bot detection    |
| Per-User      | JWT `sub`      | authenticated user quotas         |
| Per-Tenant    | `X-Tenant-ID`  | plan-level enforcement            |
| Per-Endpoint  | path           | strict limits on auth endpoints   |
| Global        | —              | circuit breaker / capacity cap    |

---

## 2. Algorithm Comparison

### Token Bucket

A bucket holds up to `maxTokens` tokens. Tokens refill at a constant rate
(`refillPerSec`). Each request consumes one token; if the bucket is empty the
request is rejected with HTTP 429.

```go
// Simplified from GGID token_bucket.go
func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    elapsed := time.Since(tb.lastRefill).Seconds()
    tb.tokens = math.Min(tb.tokens+elapsed*tb.refillRate, tb.maxTokens)
    tb.lastRefill = time.Now()
    if tb.tokens >= 1 {
        tb.tokens--
        return true
    }
    return false
}
```

**Pros**: smooth average rate, allows bursts up to `maxTokens`, memory-efficient
(O(1) per bucket).
**Cons**: the initial burst can exceed the steady-state rate; doesn't enforce
a hard ceiling per window.

### Sliding Window (Sorted Set)

Track every request timestamp within a moving window. Count entries; reject if
above the limit.

```go
// RateLimiter interface (conceptual)
type RateLimiter interface {
    Allow(key string, limit int, window time.Duration) (allowed bool, remaining int, resetAt time.Time)
}
```

**Pros**: precise limit enforcement, no burst beyond the configured ceiling.
**Cons**: higher memory (must store per-request entries or approximate with
sub-windows).

### Leaky Bucket

Requests enter a queue and are processed (leaked) at a fixed rate. If the queue
is full, requests are rejected.

**Pros**: perfectly smooth output rate.
**Cons**: adds latency to queued requests; not ideal for real-time REST APIs.

### Fixed Window

Count requests in a discrete time window (e.g., per calendar minute). Reset at
the window boundary.

**Pros**: simplest to implement, lowest memory.
**Cons**: boundary burst — a client can issue 2x the limit by clustering
requests around the window reset.

### Comparison Table

| Algorithm     | Burst Support | Smoothness | Memory | Precision | GGID Implementation        |
|---------------|:------------:|:----------:|:------:|:---------:|:--------------------------:|
| Token Bucket  | High         | Medium     | O(1)   | Medium    | Yes (`token_bucket.go`)    |
| Sliding Window| Low          | High       | O(n)   | High      | Planned                   |
| Leaky Bucket  | None         | Highest    | O(queue)| Highest  | No                        |
| Fixed Window  | Low          | Low        | O(1)   | Low       | Yes (`ratelimit.go`)      |

---

## 3. Dimension Strategy

### Per-IP

- **Default**: 100 req/min per IP for general API endpoints.
- **Challenges**: NAT/proxies cause many users to share one IP. IPv6 gives
  each client a unique address, making per-address limits ineffective.
- **IPv6 strategy**: use the `/64` prefix as the key (covers a typical LAN
  allocation) instead of the full 128-bit address.
- GGID's `ClientIP()` correctly extracts the real IP from `X-Forwarded-For`
  and `X-Real-IP`, falling back to `RemoteAddr` with port stripping.

### Per-User (authenticated)

- **Default**: 200 req/min per user (higher than IP — authenticated users are
  more trusted).
- **Key**: `user_id` claim from the verified JWT.
- **Implementation note**: requires running **after** JWT verification in the
  middleware chain so the subject is available in context.

### Per-Tenant

- **Default**: configurable by plan.
  - Free: 120 req/min (burst 20)
  - Pro: 600 req/min (burst 100)
  - Enterprise: 6000 req/min (burst 1000)
- **Aggregate limit**: sum across all users and IPs within a tenant.
- Prevents one tenant from overwhelming shared infrastructure.
- GGID already implements this via `TenantBucketLimiter` with `TierOverrides`
  and `TierFromContext()`.

### Per-Endpoint

| Endpoint               | Limit   | Rationale                        |
|------------------------|---------|----------------------------------|
| `POST /auth/login`     | 5/min   | Brute-force protection           |
| `POST /auth/register`  | 3/min   | Account-creation abuse           |
| `POST /auth/reset`     | 3/min   | Reset-flooding protection        |
| `POST /oauth/token`    | 30/min  | OAuth flow abuse                 |
| `GET /api/v1/*`        | 200/min | Read-heavy clients               |
| `POST/PUT/DELETE`      | 50/min  | Write throttling                 |

GGID's `ratelimit.go` already maps `getLimit()` by path to `LoginLimit: 5`,
`RegisterLimit: 3`, `APILimit: 100`.

### Multi-Dimensional

Apply multiple limits simultaneously; the **most restrictive** wins:

```
per-IP=100 AND per-user=200 AND per-tenant=600
→ request rejected if ANY dimension exceeds its limit.
```

---

## 4. Redis Distributed Implementation

The current in-memory implementation works for a single gateway instance but
does not share state across replicas. For production multi-instance deployments,
a Redis backend is essential.

### Sliding Window with Redis Sorted Set

```
Key:   rate:{dimension}:{id}    e.g. rate:ip:192.168.1.1
ZADD   rate:ip:192.168.1.1 <timestamp> <request-id>
ZREMRANGEBYSCORE rate:ip:192.168.1.1 0 <now-window>
ZCARD  rate:ip:192.168.1.1
```

If `ZCARD > limit` → reject (429).

### Lua Script (Atomic ZADD + ZCARD)

```lua
-- sliding_window.lua
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

local count = redis.call('ZCARD', key)
if count >= limit then
    return 0  -- rejected
end

redis.call('ZADD', key, now, member)
redis.call('EXPIRE', key, window)
return 1  -- allowed
```

### Token Bucket with Redis (Lua)

```lua
-- token_bucket.lua
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill = tonumber(ARGV[2])  -- tokens per second
local now = tonumber(ARGV[3])

local data = redis.call('HMGET', key, 'tokens', 'last')
local tokens = tonumber(data[1]) or capacity
local last = tonumber(data[2]) or now

local elapsed = math.max(0, now - last)
tokens = math.min(capacity, tokens + elapsed * refill)

if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last', now)
    redis.call('EXPIRE', key, 3600)
    return {1, tokens}
else
    redis.call('HMSET', key, 'tokens', tokens, 'last', now)
    redis.call('EXPIRE', key, 3600)
    return {0, tokens}
end
```

### Go Interface

```go
type DistributedRateLimiter interface {
    Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

type RedisSlidingWindow struct { rdb *redis.Client }

func (r *RedisSlidingWindow) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
    result, err := r.rdb.Eval(ctx, slidingWindowLua, []string{key},
        time.Now().Unix(), window.Seconds(), limit, uuid.NewString()).Int()
    if err != nil {
        return true, err // fail-open on Redis error
    }
    return result == 1, nil
}
```

---

## 5. 429 Response Headers

### Standard Headers (RFC 6585 + draft-ietf-httpapi-ratelimit-headers)

| Header                  | Purpose                                         |
|-------------------------|-------------------------------------------------|
| `HTTP/1.1 429`          | Too Many Requests                               |
| `Retry-After`           | Seconds until the client should retry           |
| `X-RateLimit-Limit`     | Max requests per window                         |
| `X-RateLimit-Remaining` | Requests remaining in current window            |
| `X-RateLimit-Reset`     | Epoch timestamp when the window resets          |

### RateLimit-Policy (draft RFC)

Describes the active policy declaratively so clients can self-throttle:

```
RateLimit-Policy: 100;w=60
```

Applied per route: `RateLimit-Policy: 5;w=60` for login endpoints.

### GGID Current Implementation

GGID's `TenantBucketLimiter.Middleware()` already returns on 429:

```
Retry-After: <seconds>
X-RateLimit-Limit: <maxTokens>
X-RateLimit-Remaining: <current tokens>
X-RateLimit-Tier: <free|pro|enterprise>
Content-Type: application/json
```

```json
{"error":"rate_limited","message":"too many requests","retry_after":3}
```

The `ratelimit.go` fixed-window limiter returns `X-RateLimit-Limit`,
`X-RateLimit-Remaining`, `X-RateLimit-Reset`, and `Retry-After`.

**Missing**: `X-RateLimit-Reset` is set by `ratelimit.go` but NOT by
`token_bucket.go`. The `RateLimit-Policy` header is not implemented in either.

---

## 6. GGID Current Implementation Analysis

Examining `ratelimit.go` and `token_bucket.go`:

| Feature             | Current State                                         | Recommendation                              |
|---------------------|-------------------------------------------------------|---------------------------------------------|
| **Algorithms**      | Fixed window (`ratelimit.go`) + Token bucket (`token_bucket.go`) | Keep token bucket as primary; add sliding window option |
| **Per-IP limiting** | Yes — `ClientIP()` + `path:ip` / `tenantID:ip` key    | Add IPv6 `/64` prefix normalization         |
| **Per-user**        | No — no JWT `sub` extraction                          | Extract `user_id` from verified JWT context |
| **Per-tenant**      | Yes — `TenantBucketLimiter` with tier overrides       | Good; add dynamic config from DB            |
| **Per-endpoint**    | Partial — `ratelimit.go` maps 3 paths (login/register/api) | Expand to cover reset, OAuth token, SCIM   |
| **Redis backend**   | No — in-memory `sync.Mutex` maps                      | Critical for multi-instance: add Redis Lua  |
| **429 headers**     | Partial — `X-RateLimit-*` set, but `token_bucket.go` omits `X-RateLimit-Reset` | Add `X-RateLimit-Reset` + `RateLimit-Policy` |
| **Lua atomicity**   | No — in-memory only                                    | Use Redis Lua scripts for atomic check+decrement |
| **Health check skip**| Yes — `/healthz`, `/docs` skipped                     | Add OPTIONS (CORS preflight) skip           |
| **Cleanup**         | Yes — background goroutine every 5 min                | Good; tune `maxAge` to match window size    |
| **Fail mode**       | In-memory: hard-fail. Redis: should fail-open         | Fail-open (allow) on Redis errors to avoid total outage |

**Key findings**:

1. Two separate limiters exist but are not composed into a unified pipeline.
   A production deployment should chain them: per-IP → per-tenant → per-user,
   with the most restrictive winning.
2. No distributed state — with multiple gateway pods, each pod has independent
   counters, so effective limits are multiplied by pod count.
3. `token_bucket.go` does not set `X-RateLimit-Reset` (only `Remaining`), so
   clients cannot compute when to retry from headers alone.
4. CORS preflight (`OPTIONS`) requests are NOT exempt and will consume tokens.

---

## 7. Best Practices

1. **Always rate-limit auth endpoints** — login, register, password reset, OAuth
   token. GGID does this via `getLimit()` but should add `/oauth/token` and
   `/auth/reset`.
2. **Use 429, not 503** — 429 is semantically correct and tells the client this
   is a rate limit, not a server error. Clients can retry intelligently.
3. **Provide Retry-After** — required by RFC 6585. GGID sets it correctly from
   `bucket.RetryAfter()`.
4. **Exempt internal traffic** — service-to-service calls should use mTLS or a
   trusted-network allowlist, not the public rate limiter.
5. **Fail-open on backend errors** — if Redis is down, allow traffic rather than
   causing a total outage. Log and alert.
6. **Monitor rate-limit events** — sustained 429s indicate either an attack or a
   capacity issue. Export metrics (`rate_limit_denied_total{dimension,tenant}`).
7. **Skip CORS preflight** — `OPTIONS` requests should never consume tokens.
   Add `r.Method == "OPTIONS"` to the skip list.
8. **Graceful degradation** — for non-critical endpoints (e.g., analytics),
   consider queuing or returning cached/stale data instead of a hard 429.
9. **IPv6 prefix aggregation** — use `/64` prefix to avoid trivially bypassing
   per-IP limits in IPv6 networks.

---

## 8. Roadmap

| Phase | Scope                                   | Effort   | Priority |
|-------|-----------------------------------------|----------|----------|
| 1     | Add `X-RateLimit-Reset` to token bucket; add `RateLimit-Policy` header; skip OPTIONS | 1 day    | High     |
| 2     | Per-endpoint expansion: add `/oauth/token`, `/auth/reset`, `/scim/*` limits | 1-2 days | High     |
| 3     | Redis backend with Lua scripts (sliding window + token bucket) | 3-4 days | Critical |
| 4     | Per-user limiting via JWT `sub` extraction (post-auth middleware) | 2 days   | Medium   |
| 5     | Per-tenant dynamic config from DB (hot-reload tier overrides) | 2 days   | Medium   |
| 6     | Monitoring dashboard: 429 rate, top-throttled IPs/tenants, limit hit heatmap | 2-3 days | Low      |

**Phases 1-2** can be done immediately with no new dependencies (~3 days).
**Phase 3** (Redis backend) is the critical path for multi-instance production —
without it, effective limits scale with pod count.
