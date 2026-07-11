# Rate Limiting Guide

This guide covers GGID's token bucket rate limiter — configuration tiers, per-user/IP/tenant tracking, Redis backend, and middleware integration.

> **Related**: [Rate Limiting](rate-limiting.md) (86 lines, brief overview), [Gateway Configuration](gateway-config.md)

## Overview

GGID implements rate limiting in `services/gateway/internal/middleware/token_bucket.go` using the classic token bucket algorithm, wired into the production handler chain.

## Token Bucket Algorithm

```
Bucket capacity: 20 tokens (burst)
Refill rate:      10 tokens/second (sustained)

Each request consumes 1 token:
  Request → bucket.Allow() → true (token consumed)
  Request → bucket.Allow() → true
  ... bucket drains ...
  Request → bucket.Allow() → false → 429 Too Many Requests
  
Bucket refills continuously:
  Idle 2 seconds → bucket refills by 20 tokens
```

### Properties

| Property | Description |
|----------|-------------|
| **Burst capacity** | Maximum tokens that can accumulate (allows short bursts) |
| **Sustained rate** | Long-term average request rate |
| **Self-healing** | Bucket refills after idle period — no permanent block |

## Configuration Tiers

GGID supports per-tenant rate limit tiers via `BucketRateLimitConfig`:

```go
type BucketRateLimitConfig struct {
    Default BucketTierConfig
    Basic   BucketTierConfig
    Premium BucketTierConfig
    Enterprise BucketTierConfig
}

type BucketTierConfig struct {
    MaxTokens  float64  // Burst capacity
    RefillPerSec float64  // Sustained rate (tokens/sec)
}
```

### Default Tier Configuration

| Tier | Burst (maxTokens) | RPS (refillPerSec) | Use Case |
|------|-------------------|-------------------|----------|
| Default | 20 | 10 | Standard API access |
| Basic | 50 | 25 | Paid tier |
| Premium | 200 | 100 | Enterprise tier |
| Enterprise | 1000 | 500 | Custom SLA |

### Endpoint-Specific Limits

| Endpoint Pattern | Burst | RPS | Rationale |
|-----------------|-------|-----|-----------|
| `/api/v1/auth/login` | 5 | 3 | Prevent brute-force |
| `/api/v1/auth/register` | 3 | 1 | Prevent spam |
| `/api/v1/users` (GET) | 100 | 50 | Read-heavy |
| `/api/v1/audit` (GET) | 100 | 50 | Analytics queries |
| `/api/v1/admin/*` | 10 | 5 | Admin operations |

## Rate Limit Key Strategy

### Per-IP

```
Key: ratelimit:{tenant_id}:{client_ip}
```

Extracts client IP using `ClientIP()` method, which strips the port from `RemoteAddr` via `net.SplitHostPort`:

```go
func ClientIP(r *http.Request) string {
    // Check X-Forwarded-For (trusted proxy)
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return strings.Split(xff, ",")[0]
    }
    // Check X-Real-IP
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    // Fall back to RemoteAddr with port stripped
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}
```

### Per-User (Authenticated)

```
Key: ratelimit:{tenant_id}:{user_id}
```

After JWT validation, the rate limiter can switch to per-user tracking:

```go
// If authenticated, rate limit by user ID
if userID := ctx.Value("user_id"); userID != nil {
    key = fmt.Sprintf("ratelimit:%s:%s", tenantID, userID)
} else {
    key = fmt.Sprintf("ratelimit:%s:%s", tenantID, clientIP)
}
```

### Per-Tenant

```
Key: ratelimit:{tenant_id}
```

Used for tenant-level throttling (total requests per tenant regardless of user).

## Redis Backend

For multi-instance deployments, rate limit state must be shared. GGID supports Redis-backed rate limiting:

```go
// Redis-based token bucket
// Atomic Lua script for token consumption
const luaScript = `
local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local bucket = redis.call('HMGET', key, 'tokens', 'timestamp')
local tokens = tonumber(bucket[1]) or max_tokens
local last_refill = tonumber(bucket[2]) or now

-- Refill based on elapsed time
local elapsed = math.max(0, now - last_refill)
tokens = math.min(max_tokens, tokens + elapsed * refill_rate)

if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'timestamp', now)
    redis.call('EXPIRE', key, ttl)
    return {1, math.floor(tokens)}
else
    redis.call('HMSET', key, 'tokens', tokens, 'timestamp', now)
    redis.call('EXPIRE', key, ttl)
    return {0, 0}
end
`
```

**Configuration**:
```yaml
RATE_LIMIT_STORAGE: redis          # redis | memory
RATE_LIMIT_REDIS_ADDR: redis:6379
RATE_LIMIT_REDIS_PASSWORD: ${REDIS_PASSWORD}
```

## 429 Response Format

```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Too many requests. Please retry later.",
    "retry_after": 60
  }
}
```

**Headers**:
```
HTTP/1.1 429 Too Many Requests
Retry-After: 60
X-RateLimit-Limit: 20
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1706104260
```

## Middleware Integration

Rate limiting is wired into the gateway's middleware chain:

```
1. Host Validation
2. CORS
3. Body Size Limit
4. Rate Limiting          ← Here
5. JWT Validation
6. Scope Enforcement
7. Security Headers
8. Compression
9. Circuit Breaker
10. Proxy
```

### TenantBucketLimiter

```go
limiter := NewTenantBucketLimiter(DefaultBucketRateLimitConfig())

// Per request
handler := func(w http.ResponseWriter, r *http.Request) {
    tenantID := getTenantID(r)
    clientIP := ClientIP(r)
    key := tenantID + ":" + clientIP

    if !limiter.Allow(key) {
        w.Header().Set("Retry-After", strconv.Itoa(limiter.RetryAfter(key)))
        writeError(w, 429, "rate_limit_exceeded")
        return
    }
    next.ServeHTTP(w, r)
}
```

## Monitoring

### Metrics

| Metric | Description |
|--------|-------------|
| `gateway_rate_limit_hits_total` | Total 429 responses by tier |
| `gateway_rate_limit_tokens` | Current token count per bucket |
| `gateway_rate_limit_bucket_count` | Active bucket count |

### Alerting

| Alert | Condition | Severity |
|-------|-----------|----------|
| Rate limit spike | 429s > 5% of total requests in 5m | Warning |
| Persistent throttle | Same IP rate-limited for > 10 min | Critical |
| Auth brute force | > 100 login 429s in 1m from same IP | Critical |

## Best Practices

1. **Set burst > sustained rate**: Allows legitimate bursts without throttling
2. **Per-IP for unauthenticated**: Prevents anonymous abuse
3. **Per-user for authenticated**: More granular, fair limits
4. **Redis for multi-instance**: In-memory limits don't share state across instances
5. **Monitor 429 rates**: High 429 rates may indicate limits too low
6. **Use `Retry-After` header**: Tells clients when to retry
7. **Exponential client backoff**: Clients should back off on 429, not retry immediately

## See Also

- [Rate Limiting](rate-limiting.md)
- [Gateway Configuration](gateway-config.md)
- [Performance Tuning](performance-tuning.md)
- [Security Audit Checklist](security-audit-checklist.md)
