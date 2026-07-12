# Rate Limiting Design

This guide covers GGID's rate limiting architecture, algorithm selection, configuration, and implementation details.

## Overview

Rate limiting protects GGID from brute-force attacks, credential stuffing, API abuse, and denial-of-service attempts while maintaining good user experience for legitimate traffic.

## Algorithm Comparison

| Algorithm | Burst | Smoothness | Memory | Use Case |
|---|---|---|---|---|
| Token Bucket | Yes | Smooth | O(1) | General purpose, APIs |
| Leaky Bucket | Limited | Very smooth | O(1) | Traffic shaping |
| Sliding Window | Yes | Smooth | O(N) | Precise limits |
| Fixed Window | No | Bursty at boundary | O(1) | Simple limits |

### Token Bucket

```
Bucket capacity: 100 tokens
Refill rate: 10 tokens/second

Each request consumes 1 token.
If bucket is empty → 429 Too Many Requests.
Tokens refill continuously.
```

**Pros**: Allows bursts up to bucket capacity, smooth average rate.
**Cons**: Large bursts can temporarily exceed target rate.

### Leaky Bucket

```
Bucket capacity: 100
Leak rate: 10 requests/second

Requests enter bucket, leak out at constant rate.
If bucket overflows → 429.
```

**Pros**: Forces constant output rate.
**Cons**: No burst capability, higher latency.

### Sliding Window

```
Window: 60 seconds
Limit: 100 requests

Count requests in the last 60 seconds.
If count >= limit → 429.
```

**Pros**: Most accurate, no boundary bursts.
**Cons**: Higher memory (stores timestamps).

### Fixed Window

```
Window: 60 seconds (aligned to clock)
Limit: 100 requests

Count resets at window boundary.
```

**Pros**: Simple, low memory.
**Cons**: 2x burst at window boundary (100 at end + 100 at start).

## GGID Implementation

GGID uses **token bucket** as the default algorithm:

```go
type TokenBucket struct {
    capacity   int64
    refillRate int64  // tokens per second
    tokens     int64
    lastRefill time.Time
    mu         sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    tb.refill()
    if tb.tokens > 0 {
        tb.tokens--
        return true
    }
    return false
}
```

## Limit Scopes

### Per-User

```yaml
rate_limit:
  per_user:
    login: 5/minute
    password_reset: 3/hour
    mfa_verify: 10/minute
```

Identified by JWT subject or session user ID.

### Per-IP

```yaml
rate_limit:
  per_ip:
    register: 10/hour
    api_general: 100/minute
```

Identified by client IP (with X-Forwarded-For handling).

### Per-Tenant

```yaml
rate_limit:
  per_tenant:
    api_total: 10000/minute
    admin_operations: 100/minute
```

Identified by tenant ID from JWT claim or header.

### Combined Limits

```yaml
rate_limit:
  strategy: "strictest"  # or "first_match"
  per_user:
    login: 5/minute
  per_ip:
    login: 50/minute  # Max 50 logins/min from one IP across all users
```

## Distributed Rate Limiting

### Redis-Based

For multi-instance deployments, GGID uses Redis for shared rate limit state:

```go
type RedisLimiter struct {
    client *redis.Client
    key    string
    rate   int64
    burst  int64
}

func (rl *RedisLimiter) Allow(ctx context.Context) (bool, error) {
    // Atomic token bucket using Redis Lua script
    script := `
        local tokens = tonumber(redis.call('get', KEYS[1]) or ARGV[1])
        local now = tonumber(ARGV[2])
        local last = tonumber(redis.call('get', KEYS[2]) or now)
        local refill = (now - last) * tonumber(ARGV[3]) / 1000
        tokens = math.min(tonumber(ARGV[1]), tokens + refill)
        if tokens >= 1 then
            tokens = tokens - 1
            redis.call('setex', KEYS[1], 3600, tokens)
            redis.call('setex', KEYS[2], 3600, now)
            return 1
        else
            return 0
        end
    `
    result, err := rl.client.Eval(ctx, script, []string{rl.key + ":tokens", rl.key + ":last"}, rl.burst, now, rl.rate).Int()
    return result == 1, err
}
```

### Configuration

```yaml
rate_limit:
  store: "redis"  # or "memory" for single-instance
  redis:
    url: "redis://redis:6379"
    key_prefix: "ggid:rl:"
    ttl: 3600
```

## Rate Limit Headers

GGID returns standard rate limit headers on every API response:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1700000000
Retry-After: 12  # Only on 429 responses
```

| Header | Description |
|---|---|
| `X-RateLimit-Limit` | Maximum requests in current window |
| `X-RateLimit-Remaining` | Remaining requests in current window |
| `X-RateLimit-Reset` | Unix timestamp when window resets |
| `Retry-After` | Seconds to wait before retrying (429 only) |

## 429 Response

```json
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 30

{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please retry after 30 seconds.",
  "retry_after": 30
}
```

## Graceful Degradation

When the rate limit store (Redis) is unavailable:

```yaml
rate_limit:
  on_store_failure: "allow"  # or "deny" or "local_only"
```

| Strategy | Behavior |
|---|---|
| `allow` | Allow all requests (fail open) |
| `deny` | Deny all requests (fail closed) |
| `local_only` | Fall back to in-memory per-instance limits |

**Recommendation**: Use `allow` for authentication endpoints (availability > strictness) and `deny` for admin endpoints (security > availability).

## Exempt Paths

```yaml
rate_limit:
  exempt_paths:
    - /healthz
    - /readyz
    - /metrics
    - /api/v1/.well-known/*
    - /api/v1/oauth/.well-known/*
```

Health checks and discovery endpoints are never rate-limited.

## Burst Handling

```yaml
rate_limit:
  burst_multiplier: 2  # Bucket capacity = 2x rate
  burst_duration: 10s  # Burst window
```

This allows short bursts (e.g., 200 requests in 10 seconds at 100/min rate) while maintaining the average rate over time.

## GGID Rate Limiter Implementation

### Middleware

```go
func RateLimitMiddleware(limiter Limiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := rateLimitKey(r)  // user ID, IP, or tenant ID
            allowed, headers := limiter.Allow(key)
            for k, v := range headers {
                w.Header().Set(k, v)
            }
            if !allowed {
                w.Header().Set("Retry-After", strconv.Itoa(headers["Retry-After"]))
                writeError(w, 429, "rate_limit_exceeded")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Key Extraction

```go
func rateLimitKey(r *http.Request) string {
    // Priority: JWT subject > tenant ID > client IP
    if claims := getJWTClaims(r); claims != nil {
        return "user:" + claims.Subject
    }
    if tenant := r.Header.Get("X-Tenant-ID"); tenant != "" {
        return "tenant:" + tenant
    }
    return "ip:" + clientIP(r)
}

func clientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        // Use leftmost IP (closest to client)
        return strings.Split(xff, ",")[0]
    }
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}
```

### Default Limits

| Endpoint | Limit | Scope |
|---|---|---|
| `/api/v1/auth/login` | 5/min | per-user |
| `/api/v1/auth/login` | 50/min | per-IP |
| `/api/v1/auth/register` | 10/hour | per-IP |
| `/api/v1/auth/mfa/verify` | 10/min | per-user |
| `/api/v1/auth/password/reset` | 3/hour | per-user |
| `/api/v1/oauth/token` | 30/min | per-client |
| `/api/v1/users` (GET) | 100/min | per-user |
| `/api/v1/users` (POST) | 20/min | per-user |
| `/api/v1/admin/*` | 50/min | per-tenant |

## Monitoring

```yaml
rate_limit:
  metrics: true
  prometheus:
    namespace: "ggid"
    subsystem: "rate_limit"
```

Exposed metrics:
- `ggid_rate_limit_allowed_total{scope,endpoint}`
- `ggid_rate_limit_denied_total{scope,endpoint}`
- `ggid_rate_limit_current_tokens{scope,key}`

## Best Practices

1. **Use per-user limits for authenticated endpoints** — prevents one compromised token from affecting others
2. **Use per-IP limits for unauthenticated endpoints** — prevents distributed attacks
3. **Set Retry-After on all 429 responses** — improves client experience
4. **Monitor denied rates** — sudden spikes indicate attacks
5. **Use Redis for distributed deployments** — ensures consistent limits across instances
6. **Exempt health and discovery endpoints** — prevents monitoring tools from being blocked
7. **Log rate limit events** — audit trail for security analysis
8. **Tune burst multiplier per endpoint** — login needs strict limits, read APIs can be more lenient