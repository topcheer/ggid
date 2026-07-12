# Rate Limiting Strategy Guide

Rate limiting strategies — per-endpoint vs per-user vs per-IP, token bucket vs sliding window, circuit breaker integration, graceful degradation, monitoring.

## Dimension Comparison

| Dimension | Granularity | Bypass Risk | False Positives |
|-----------|-----------|-------------|----------------|
| Per-IP | Coarse | NAT/VPN shared IP | High (shared IPs) |
| Per-user | Fine | Need user account | Low |
| Per-tenant | Very coarse | Whole tenant affected | Medium |
| Per-endpoint | Fine | N/A | Low |
| Per-token | Fine | Token theft → unlimited | Low |

**Recommendation**: Per-user + per-IP (defense in depth).

## Algorithm Comparison

| Algorithm | Memory | Burst | Fairness | GGID |
|-----------|--------|-------|----------|------|
| Token bucket | Low | Yes (burst=size) | Medium | Implemented |
| Sliding window | High | No | High | — |
| Fixed window | Low | Yes (at boundary) | Low | — |
| Leaky bucket | Low | No | Medium | — |

## GGID Token Bucket Config

```yaml
gateway:
  rate_limit:
    enabled: true
    per_ip:
      requests_per_minute: 100
      burst: 20
    per_user:
      requests_per_minute: 500
      burst: 50
    per_endpoint:
      "/api/v1/auth/login":
        requests_per_minute: 10
        burst: 5
      "/api/v1/auth/register":
        requests_per_minute: 5
        burst: 2
```

## Circuit Breaker Integration

```
Request → Rate limit check
  ↓ Allow → Circuit breaker check
  ↓         ↓ Closed → Forward to service
  ↓         ↓ Open → Return 503 (fallback)
  ↓ Block → Return 429 + Retry-After header
```

## Graceful Degradation

When rate limited (429):

```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Too many requests",
    "retry_after": 30
  }
}
```

Response headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1706104230
Retry-After: 30
```

## Monitoring

| Metric | Alert Threshold |
|--------|----------------|
| 429 rate | > 5% of requests |
| Rate limit resets | Spike = bot attack |
| Per-user 429s | > 100/user/hour |
| Circuit opens | Any occurrence |

## See Also

- [Rate Limiting Guide](rate-limiting-guide.md)
- [Performance Tuning](performance-tuning.md)
- [Security Audit Checklist](security-audit-checklist.md)
