# Rate Limiting Strategy

Gateway and Auth service rate limiting policies, configuration, and best practices.

---

## Overview

GGID enforces rate limiting at multiple layers:

| Layer | Scope | Purpose |
|-------|-------|---------|
| Gateway (IP) | Per client IP | Protect against brute-force, credential stuffing |
| Gateway (Tenant) | Per tenant ID | Prevent noisy-neighbor problems |
| Auth Service | Per username/IP | Login attempt throttling |

---

## Gateway Rate Limits

### Default Limits

| Endpoint Pattern | Limit | Window | Scope |
|------------------|-------|--------|-------|
| `POST /auth/login` | 5 | minute | per IP |
| `POST /auth/register` | 3 | minute | per IP |
| `POST /auth/password/forgot` | 3 | minute | per IP |
| `POST /auth/magic-link` | 3 | minute | per IP |
| `POST /auth/password/reset` | 5 | minute | per IP |
| `POST /auth/mfa/login` | 5 | minute | per IP |
| `GET /api/v1/*` | 100 | minute | per IP |
| `POST /api/v1/*` | 60 | minute | per IP |
| `PUT/PATCH/DELETE /api/v1/*` | 30 | minute | per IP |
| `GET /oauth/authorize` | 30 | minute | per IP |
| `POST /oauth/token` | 30 | minute | per IP |
| `GET /scim/v2/*` | 100 | minute | per IP |
| `POST/PUT/DELETE /scim/v2/*` | 30 | minute | per IP |

### Algorithm: Fixed Window

```go
// Gateway uses a fixed-window counter per (IP, endpoint_pattern)
key := fmt.Sprintf("ratelimit:%s:%s", ip, pattern)
count := redis.Incr(key)
if count == 1 { redis.Expire(key, 60*time.Second) }
if count > limit { return 429 }
```

### 429 Response

```json
{
  "error": "rate limit exceeded",
  "code": "RATE_LIMITED",
  "retry_after": 60
}
```

**Headers:**
```
Retry-After: 60
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1720612860
```

---

## Auth Service: Login Failure Throttling

### Per-Username Locking

After consecutive failed login attempts for a specific username:

| Failed Attempts | Action |
|:--------------:|--------|
| 1-4 | Normal rejection (401) |
| 5 | Account temporarily locked (5 min cooldown) |
| 10 | Account locked (requires admin unlock) |

### Per-IP Throttling

The Gateway limits login attempts per IP at 5/minute, independent of username. This prevents trying many usernames from one IP (credential stuffing).

### Redis Key Structure

```
auth:fail:{tenant_id}:{username}     → failure count (TTL: 15 min)
auth:lock:{tenant_id}:{username}     → lock flag (TTL: 5 min or permanent)
auth:cooldown:{tenant_id}:{username} → cooldown timestamp
```

---

## Per-Tenant Rate Limiting

### Configure Tenant Limits

```bash
PUT /api/v1/tenants/{tenant_id}/rate-limit
{
  "requests_per_minute": 2000,
  "login_attempts_per_minute": 20
}
```

### Default Tenant Limits

| Plan | RPS | Login/min |
|------|:---:|:---------:|
| Free | 100 | 5 |
| Pro | 500 | 20 |
| Enterprise | 5000 | 100 |

---

## Multi-Instance Configuration

### Problem

In-memory rate limiting doesn't work across multiple Gateway replicas — each instance has its own counter.

### Solution: Redis-Backed Rate Limiting

```bash
# Enable shared rate limiting
GATEWAY_REDIS_RATELIMIT=true
REDIS_ADDR=redis:6379
```

All Gateway instances share rate limit counters via Redis `INCR` + `EXPIRE`.

### Alternative: Sticky Sessions

If Redis-backed limiting is unavailable, configure the load balancer with sticky sessions (by client IP) so the same IP always hits the same Gateway instance.

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_RATE_LIMIT_ENABLED` | `true` | Enable/disable rate limiting |
| `GATEWAY_RATE_LIMIT_LOGIN` | `5` | Login attempts per minute |
| `GATEWAY_RATE_LIMIT_REGISTER` | `3` | Registrations per minute |
| `GATEWAY_RATE_LIMIT_API` | `100` | General API requests per minute |
| `GATEWAY_RATE_LIMIT_WINDOW` | `60` | Window size in seconds |
| `GATEWAY_REDIS_RATELIMIT` | `false` | Use Redis for shared counters |

### Programmatic Configuration

```go
config := gateway.RateLimitConfig{
    Enabled:          true,
    LoginLimit:       5,
    RegisterLimit:    3,
    APILimit:         100,
    WindowSeconds:    60,
    UseRedis:         true,
    RedisAddr:        "redis:6379",
}
```

---

## Best Practices

1. **Respect `Retry-After`** — Wait the specified duration before retrying
2. **Exponential backoff + jitter** — Don't retry all clients simultaneously
3. **Use webhooks instead of polling** — Reduce API calls
4. **Batch operations** — Use bulk import instead of individual creates
5. **Cache GET responses** — Reduce repeated identical queries
6. **Monitor rate limit hits** — Alert if hitting limits frequently (misconfigured client)
7. **Separate rate limits for SCIM** — Provisioning bursts shouldn't affect user-facing limits

---

## Monitoring

### Prometheus Metrics

| Metric | Description |
|--------|-------------|
| `ggid_rate_limit_hits_total{path}` | Counter of rate-limited requests |
| `ggid_rate_limit_remaining{path}` | Remaining requests in current window |
| `ggid_auth_failures_total{username}` | Login failure count |

### Alerting

```yaml
- alert: HighRateLimitHits
  expr: rate(ggid_rate_limit_hits_total[5m]) > 10
  for: 5m
  annotations:
    summary: "Rate limit rejections high — possible abuse or misconfigured client"

- alert: BruteForceDetected
  expr: rate(ggid_auth_failures_total[1m]) > 20
  for: 2m
  annotations:
    summary: "Possible brute-force attack — many login failures"
```
