# API Rate Limiting Strategy

## Overview

Rate limiting is a critical security and reliability control for API platforms. This guide covers rate limit dimensions, algorithm selection, distributed enforcement, graduated responses, exemption strategies, observability, and how GGID implements rate limiting.

## Rate Limit Dimensions

### User-Based Limiting

Rate limits applied per authenticated user identity.

- **Scope**: Per user ID across all IP addresses and clients
- **Use case**: Prevent a single user from overwhelming the API regardless of device
- **Configuration**: `user_id` -> `{ requests, window, burst }`
- **Example**: 1000 requests/minute per user, burst of 50

### IP-Based Limiting

Rate limits applied per client IP address.

- **Scope**: Per IP address across all users (for unauthenticated endpoints) or per IP+user combination
- **Use case**: Prevent DDoS, brute force from single IPs, credential stuffing
- **Configuration**: `ip` -> `{ requests, window, burst }`
- **Example**: 100 requests/minute per IP for unauthenticated endpoints
- **Considerations**: NAT/proxy aggregation, IPv6 subnet throttling, X-Forwarded-For trust

### Tenant-Based Limiting

Rate limits applied per tenant (organization) in multi-tenant systems.

- **Scope**: Per tenant ID across all users and IPs in that tenant
- **Use case**: Enforce tenant quotas, prevent one tenant from impacting others
- **Configuration**: `tenant_id` -> `{ requests, window, burst }`
- **Example**: 10000 requests/minute per tenant
- **Tiered plans**: Different limits per tenant tier (free, pro, enterprise)

### Client-Based Limiting

Rate limits applied per OAuth client or API key.

- **Scope**: Per client_id or API key across all users
- **Use case**: Prevent a single integration from consuming all capacity
- **Configuration**: `client_id` -> `{ requests, window, burst }`
- **Example**: 5000 requests/minute per OAuth client

### Key-Based Limiting

Rate limits applied per API key identifier.

- **Scope**: Per API key, optionally scoped to tenant + key combination
- **Use case**: Per-key quotas for API consumers
- **Configuration**: `api_key` -> `{ requests, window, burst, daily_quota }`
- **Example**: 500 requests/minute, 100K requests/day per key

### Endpoint-Specific Limiting

Rate limits applied to specific high-cost or sensitive endpoints.

| Endpoint Category | Limit | Reason |
|-------------------|-------|--------|
| Authentication (login/register) | 10/min per IP | Brute force prevention |
| Password reset | 3/min per user | Account takeover prevention |
| MFA verification | 5/min per user | MFA fatigue prevention |
| Token revocation | 10/min per client | Abuse prevention |
| Bulk operations | 5/min per user | Data exfiltration prevention |
| Export endpoints | 2/min per user | Data exfiltration prevention |
| Admin operations | 20/min per admin | Privilege abuse prevention |
| Health check | Unlimited | Monitoring |

### Composite Limiting

Multiple dimensions can be combined for layered protection:

```
Request -> Check IP limit -> Check User limit -> Check Tenant limit -> Check Endpoint limit -> Allow/Deny
```

The most restrictive applicable limit wins. A request must pass all applicable limit checks.

## Algorithm Selection

### Token Bucket

Best for most API rate limiting scenarios.

- **How it works**: Bucket holds N tokens, refilled at R tokens/second. Each request consumes 1 token. If no tokens, request is denied.
- **Pros**: Allows burst traffic up to bucket size, smooth average rate
- **Cons**: Slightly more complex than fixed window
- **Best for**: APIs with bursty traffic patterns
- **Parameters**: `capacity` (burst), `refill_rate` (sustained rate)

```
Bucket capacity: 100 tokens
Refill rate: 10 tokens/second
-> Allows 100 instant requests, then sustains 10/sec
```

### Sliding Window

Best for strict rate enforcement without burst tolerance.

- **How it works**: Track timestamps of requests in a sliding window. Count requests in the last N seconds.
- **Pros**: Precise rate enforcement, no burst at window boundaries
- **Cons**: Higher memory (store timestamps), more computation
- **Best for**: Security-sensitive endpoints (auth, password reset)
- **Parameters**: `window_size`, `max_requests`

```
Window: 60 seconds, max: 100 requests
-> At any point, only 100 requests in the last 60 seconds are allowed
```

### Fixed Window

Simplest algorithm, acceptable for coarse limits.

- **How it works**: Count requests in fixed time windows (e.g., each minute). Reset at window boundary.
- **Pros**: Simple, low memory, fast
- **Cons**: Burst at window boundaries (2x limit at boundary)
- **Best for**: Non-critical endpoints, daily quotas
- **Parameters**: `window_size`, `max_requests`

```
Window: 1 minute (00:00-00:59), max: 100
-> 100 requests at 00:59 + 100 at 01:00 = 200 in 1 second
```

### Leaky Bucket

Best for traffic shaping and queue management.

- **How it works**: Requests enter a queue, processed at fixed rate. Queue overflow = rejection.
- **Pros**: Smooths traffic to a constant rate, queues rather than rejects
- **Cons**: Adds latency, requires queue management
- **Best for**: Downstream service protection, webhook delivery
- **Parameters**: `queue_size`, `outflow_rate`

### Algorithm Comparison

| Algorithm | Burst | Memory | Precision | Use Case |
|-----------|-------|--------|-----------|----------|
| Token bucket | Yes (bounded) | O(1) | Good | General API |
| Sliding window | No | O(N) | Exact | Security-sensitive |
| Fixed window | Yes (2x at boundary) | O(1) | Coarse | Quotas |
| Leaky bucket | No (queues) | O(queue) | Smooth | Traffic shaping |

## Distributed Enforcement

### Redis-Based Rate Limiting

For distributed deployments, rate limit state must be shared across all gateway instances.

#### Token Bucket with Redis

```lua
-- Lua script for atomic token bucket in Redis
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

-- Refill tokens
local elapsed = math.max(0, now - last_refill)
local refilled = math.min(capacity, tokens + elapsed * refill_rate)

if refilled < 1 then
    -- Not enough tokens, deny
    redis.call('HMSET', key, 'tokens', refilled, 'last_refill', now)
    redis.call('EXPIRE', key, ttl)
    return {0, refilled}  -- denied, remaining tokens
else
    -- Consume token, allow
    local remaining = refilled - 1
    redis.call('HMSET', key, 'tokens', remaining, 'last_refill', now)
    redis.call('EXPIRE', key, ttl)
    return {1, remaining}  -- allowed, remaining tokens
end
```

#### Sliding Window with Redis

```lua
-- Lua script for atomic sliding window in Redis
local key = KEYS[1]
local max_requests = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove expired entries
local cutoff = now - window
redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)

-- Count current entries
local count = redis.call('ZCARD', key)

if count < max_requests then
    -- Allow: add current request
    redis.call('ZADD', key, now, now .. '-' .. math.random(1000000))
    redis.call('EXPIRE', key, window + 1)
    return {1, max_requests - count - 1}  -- allowed, remaining
else
    return {0, 0}  -- denied
end
```

### Considerations for Distributed Rate Limiting

- **Clock skew**: Use Redis server time or NTP-synced clocks for accurate window calculations
- **Redis latency**: Each rate limit check adds a Redis round trip (~1ms). Use connection pooling.
- **Race conditions**: Lua scripts ensure atomicity in Redis
- **Redis failure**: Define fallback behavior (fail open for non-critical, fail closed for security)
- **Multi-region**: Consider per-region limits with a global aggregator for strict enforcement
- **Memory management**: Use TTLs to expire stale rate limit keys automatically

## Graduated Response

Instead of a binary allow/deny, graduated responses provide escalating enforcement.

### Response Levels

| Level | Trigger | Response |
|-------|---------|----------|
| Normal | Under 80% of limit | 200 OK, headers with current usage |
| Warning | 80-99% of limit | 200 OK, headers with warning, log alert |
| Throttle | At 100% of limit | 429 Too Many Requests, Retry-After header |
| Block | Sustained overage or abuse | 429 + extended cooldown (exponential backoff) |
| Ban | Malicious behavior | 403 Forbidden + temporary IP/user ban |

### Implementation

```
Request -> Rate check -> Under 80%? -> 200 OK + headers
                     -> 80-99%?   -> 200 OK + headers + warning log
                     -> 100%?     -> 429 + Retry-After
                     -> Over 120%?-> 429 + extended cooldown
                     -> Banned?   -> 403 Forbidden
```

### Exponential Backoff for Abuse

For sustained rate limit violations, increase the cooldown period exponentially:

```
Violation 1: 429 + Retry-After: 60s
Violation 2: 429 + Retry-After: 120s
Violation 3: 429 + Retry-After: 300s
Violation 4: 429 + Retry-After: 600s
Violation 5+: 403 + Ban: 3600s
```

Track violation count in Redis with a sliding window to reset after good behavior.

## Exempt Paths

Certain endpoints should be exempt from or have relaxed rate limits.

| Path | Exemption | Reason |
|------|-----------|--------|
| /healthz | Fully exempt | Health check probe |
| /readyz | Fully exempt | Readiness probe |
| /metrics | Fully exempt (internal only) | Monitoring scrape |
| /.well-known/* | Relaxed (higher limit) | Discovery endpoints |
| /api/v1/audit/events (GET) | Relaxed for read | Compliance queries |
| Admin API | Stricter (not exempt) | Privilege abuse prevention |

**Exemption implementation**: Configure exempt paths in rate limit middleware with a path matcher.

## Rate Limit Headers

Following the IETF draft for rate limit headers (`draft-ietf-httpapi-ratelimit-headers`):

### Response Headers

```
RateLimit-Limit: 1000
RateLimit-Remaining: 999
RateLimit-Reset: 30
```

- `RateLimit-Limit`: Maximum requests in the current window
- `RateLimit-Remaining`: Requests remaining in current window
- `RateLimit-Reset`: Seconds until the window resets

### 429 Response Headers

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 30
RateLimit-Limit: 1000
RateLimit-Remaining: 0
RateLimit-Reset: 30

{
  "error": "rate_limited",
  "message": "Rate limit exceeded. Try again in 30 seconds.",
  "retry_after": 30
}
```

### Policy Headers (Optional)

```
RateLimit-Policy: 1000;w=60
```

- Format: `limit;w=window_seconds` or `limit;w=window;burst=burst_capacity`

## Observability

### Metrics

| Metric | Type | Description |
|--------|------|-------------|
| rate_limit_allowed_total | Counter | Requests allowed by rate limiter |
| rate_limit_denied_total | Counter | Requests denied by rate limiter |
| rate_limit_warned_total | Counter | Requests in warning zone (80-99%) |
| rate_limit_banned_total | Counter | Requests from banned IPs/users |
| rate_limit_tokens_remaining | Gauge | Tokens remaining in bucket (per dimension) |
| rate_limit_check_duration_seconds | Histogram | Time spent in rate limit check |

### Logging

```json
{
  "event": "rate_limit_denied",
  "timestamp": "2026-01-15T10:30:00Z",
  "ip": "192.168.1.50",
  "user_id": "usr_abc123",
  "tenant_id": "tenant_001",
  "client_id": "cli_xyz",
  "endpoint": "/api/v1/users",
  "limit_dimension": "user",
  "limit_value": 1000,
  "window": 60,
  "violation_count": 3,
  "cooldown_seconds": 120
}
```

### Alerting

- **High denial rate**: Alert when denial rate > 5% of total requests
- **Sustained abuse**: Alert when a single IP/user is denied > 50 times in 5 minutes
- **Banned entities**: Alert when new IP/user bans are created
- **Redis latency**: Alert when rate limit check takes > 10ms
- **Config drift**: Alert when rate limit config differs across instances

## GGID Rate Limiting

### Implementation

GGID implements rate limiting in the API gateway middleware layer:

- **Middleware**: `services/gateway/internal/middleware/token_bucket.go`
- **Algorithm**: Token bucket with Redis backend for distributed enforcement
- **Dimensions**: IP, user, tenant, client, endpoint-specific
- **Configuration**: Via gateway configuration, hot-reloadable

### Configuration Example

```yaml
gateway:
  rate_limiting:
    enabled: true
    redis:
      addr: "redis:6379"
      pool_size: 20
    default:
      requests: 1000
      window: 60
      burst: 50
    dimensions:
      ip:
        requests: 100
        window: 60
        burst: 20
      user:
        requests: 500
        window: 60
        burst: 50
      tenant:
        requests: 5000
        window: 60
        burst: 200
    endpoints:
      - path: "/api/v1/auth/login"
        method: "POST"
        ip:
          requests: 10
          window: 60
      - path: "/api/v1/auth/register"
        method: "POST"
        ip:
          requests: 5
          window: 60
      - path: "/api/v1/auth/mfa/verify"
        method: "POST"
        user:
          requests: 5
          window: 60
    exempt:
      - "/healthz"
      - "/readyz"
      - "/metrics"
    graduated:
      warning_threshold: 0.8
      block_threshold: 1.2
      ban_threshold: 5
      ban_duration: 3600
```

### Client IP Detection

GGID correctly extracts client IP considering proxy chains:

1. `X-Forwarded-For` (first non-trusted proxy IP, if trusted proxy config is set)
2. `X-Real-IP` (if present and from trusted proxy)
3. `RemoteAddr` (fallback)

**Security**: Only trust `X-Forwarded-For` when the gateway is behind a known reverse proxy. Untrusted `X-Forwarded-For` headers are ignored.

### Rate Limit Response

GGID returns structured error responses on rate limit violations:

```json
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 60
RateLimit-Limit: 100
RateLimit-Remaining: 0
RateLimit-Reset: 60

{
  "error": {
    "code": "RATE_LIMITED",
    "message": "Rate limit exceeded for this endpoint",
    "retry_after": 60,
    "limit": 100,
    "window": 60
  }
}
```

## Best Practices

1. **Layer your limits**: Apply multiple dimensions (IP + user + tenant) for defense in depth
2. **Start conservative**: Begin with lower limits and increase based on actual usage data
3. **Monitor and tune**: Track denial rates and adjust limits to minimize false positives
4. **Use graduated responses**: Binary allow/deny is too harsh - warn before blocking
5. **Provide clear feedback**: Use standard headers and error messages so clients can adapt
6. **Handle Redis failures gracefully**: Fail open for non-critical, fail closed for security endpoints
7. **Document your limits**: Publish rate limit documentation for API consumers
8. **Test under load**: Verify rate limiting works correctly under high concurrency
9. **Consider cost-based limits**: Weight expensive operations (e.g., export) more heavily
10. **Coordinate with caching**: Rate limits and caching work together - cache to reduce request volume

## See Also

- [Gateway Architecture](./gateway-architecture.md)
- API Security Guide
- Security Headers Guide
- Token Bucket Rate Limiting
- DDoS Protection Guide