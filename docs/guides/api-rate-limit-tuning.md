# API Rate Limit Tuning

Per-endpoint vs per-user vs per-IP vs per-tenant, token bucket sizing, burst handling, 429 response design, and client retry guidance.

## Rate Limit Dimensions

| Dimension | Scope | Use Case |
|-----------|-------|---------|
| Per-IP | All requests from one IP | Brute force, DDoS |
| Per-user | Requests from authenticated user | API abuse, fair usage |
| Per-tenant | All requests for a tenant | Tenant quotas |
| Per-endpoint | Specific endpoint (e.g., login) | Protect expensive ops |
| Per-client | OAuth client | App-level quotas |

## Token Bucket

```go
type TokenBucket struct {
    rate       float64       // Tokens per second
    burst      int           // Max tokens (bucket size)
    tokens     float64
    lastRefill time.Time
    mu         sync.Mutex
}

func (b *TokenBucket) Allow() bool {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(b.lastRefill).Seconds()
    b.tokens = math.Min(b.burst, b.tokens + elapsed*b.rate)
    b.lastRefill = now
    
    if b.tokens >= 1 {
        b.tokens--
        return true
    }
    return false
}
```

## Default Limits

| Endpoint | Dimension | Rate | Burst |
|----------|-----------|------|-------|
| POST /auth/login | Per-IP | 10/min | 20 |
| POST /auth/login | Per-user | 5/min | 10 |
| POST /auth/register | Per-IP | 5/min | 10 |
| GET /users (list) | Per-tenant | 100/min | 200 |
| POST /users (create) | Per-tenant | 30/min | 60 |
| GET /audit/events | Per-tenant | 60/min | 120 |
| POST /oauth/token | Per-client | 300/min | 600 |
| GET /healthz | Per-IP | 1000/min | 2000 |

## 429 Response Design

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 30
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1700000600

{
  "error": "too_many_requests",
  "error_description": "Rate limit exceeded. Retry after 30 seconds.",
  "retry_after": 30
}
```

## Client Retry Guidance

```yaml
retry_strategy:
  on_429:
    retry: true
    read_retry_after_header: true
    fallback_delay: 30s
    backoff: exponential
    max_retries: 3
    
  headers_to_read:
    - Retry-After        # Server-recommended wait
    - X-RateLimit-Reset  # When limit resets (unix timestamp)
```

## Burst Handling

```go
// Sliding window + token bucket hybrid
type HybridLimiter struct {
    window    *SlidingWindow  // Long-term rate
    bucket    *TokenBucket    // Short-term burst
}

func (h *HybridLimiter) Allow() bool {
    return h.window.Allow() && h.bucket.Allow()
}
```

## Redis-Based Distributed Limiting

```go
// Lua script for atomic token consumption
const luaScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local tokens = tonumber(redis.call("get", key) or burst)
tokens = math.min(burst, tokens + rate * tonumber(ARGV[3]))
if tokens >= 1 then
    tokens = tokens - 1
    redis.call("setex", key, 60, tokens)
    return 1
else
    return 0
end
`
```

## Monitoring

| Metric | Alert |
|--------|-------|
| 429 rate | >5% → tune limits or scale |
| Per-endpoint limit hits | Spike → possible abuse |
| Per-IP 429 concentration | Single IP → block |
| Retry-After compliance | Clients not retrying correctly |

## See Also

- [Rate Limiting Strategy](rate-limiting-strategy.md)
- [OAuth Backpressure](oauth-backpressure.md)
- [Gateway Architecture](gateway-architecture.md)
- [OAuth Error Catalog](oauth-error-catalog.md)
