# Rate Limiting — Technical Guide

> Feature: Multi-Dimensional Rate Limiting
> Location: `services/gateway/internal/middleware/ratelimit.go`

## What It Does

GGID's gateway implements multi-dimensional rate limiting to protect APIs from abuse, DDoS, and excessive consumption. Each request is evaluated against limits before being forwarded to backend services.

## Limiting Dimensions

Rate limits are evaluated across multiple dimensions:

| Dimension | Key | Description |
|-----------|-----|-------------|
| **Global** | Total requests to gateway | Protects overall system capacity |
| **Per-IP** | Client IP address | Prevents single-source abuse |
| **Per-User** | Authenticated user ID | Limits per-account consumption |
| **Per-Tenant** | Tenant ID (X-Tenant-ID) | Isolates tenant traffic |
| **Per-Endpoint** | API path pattern | Protects expensive endpoints |

A request is denied if ANY dimension exceeds its limit.

## Rate Limit Tiers

Different tenant tiers have different rate limits:

| Tier | Per-IP (req/min) | Per-User (req/min) | Per-Tenant (req/min) |
|------|------------------|--------------------|---------------------|
| **Free** | 60 | 120 | 1,000 |
| **Pro** | 200 | 600 | 10,000 |
| **Enterprise** | 1,000 | 3,000 | 50,000 |

## Burst vs Sustained

Rate limiting uses a **fixed-window** algorithm with burst capacity:

- **Sustained rate**: Long-term average request rate (e.g., 100 req/min).
- **Burst capacity**: Short-term spike tolerance (e.g., 200 req burst in first 5 seconds).

The window resets at the start of each time period (minute/hour).

## Response Headers

Every API response includes rate limit headers:

```
X-RateLimit-Limit: 120
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1721289600
```

When rate limited, the response is:

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 30

{"error": "rate_limit_exceeded", "message": "Rate limit exceeded. Try again in 30 seconds."}
```

## Configuration

```go
type RateLimitConfig struct {
    GlobalLimit    int            // Total requests per window
    PerIPLimit     int            // Per IP per window
    PerUserLimit   int            // Per authenticated user per window
    PerTenantLimit int            // Per tenant per window
    WindowSeconds  int            // Time window (default: 60)
    TierOverrides  map[string]TierLimits // Per-tier overrides
}
```

## API Endpoints

There is no dedicated rate limit API. Limits are enforced transparently by the gateway middleware on all `/api/v1/*` endpoints.

### Verify Rate Limits

```bash
# Check response headers for rate limit info
curl -sk -D- -o /dev/null \
  -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/users" \
  -H "Authorization: Bearer $TOKEN" \
  2>&1 | grep -i ratelimit

# Output:
# X-RateLimit-Limit: 120
# X-RateLimit-Remaining: 119
# X-RateLimit-Reset: 1721289600

# Test rate limiting (burst test)
for i in $(seq 1 150); do
  curl -sk -o /dev/null -w '%{http_code} ' \
    "https://ggid.iot2.win/api/v1/users" \
    -H "Authorization: Bearer $TOKEN"
done
# Expected: 200 200 200 ... 429 429 429 (after limit exceeded)
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Frequent 429 errors | Rate limit too low for usage pattern | Upgrade tenant tier or contact admin to increase limits |
| Rate limit not working | Redis unavailable (production) or limiter not configured | Check RateLimiter middleware is in gateway chain |
| Inconsistent limits | Multiple gateway instances with in-memory limiter | Use Redis-backed limiter for distributed consistency |
| X-RateLimit headers missing | Middleware order wrong or endpoint excluded | Verify rate limit middleware runs before response is written |

## Best Practices

- **Check headers**: Always inspect `X-RateLimit-Remaining` to avoid hitting limits.
- **Implement backoff**: When receiving 429, honor `Retry-After` header before retrying.
- **Batch requests**: Reduce request count by using bulk endpoints (e.g., `/users/import`).
- **Cache locally**: Cache API responses client-side to reduce redundant calls.
- **Monitor usage**: Track rate limit headers to forecast capacity needs.
