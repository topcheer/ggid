# GGID API Rate Limits

Complete reference for API rate limiting in GGID.

---

## Overview

GGID Gateway enforces rate limits to protect against brute-force attacks,
credential stuffing, and API abuse. Limits are applied per-IP by default.

---

## Rate Limit Tiers

### Authentication Endpoints (Strict)

| Endpoint | Limit | Window | Scope | Enforcement |
|----------|-------|--------|-------|-------------|
| `POST /api/v1/auth/login` | 5 | per minute | per IP | Fixed-window, in-memory |
| `POST /api/v1/auth/register` | 3 | per minute | per IP | Fixed-window, in-memory |
| `POST /api/v1/auth/password/forgot` | 3 | per minute | per IP | Fixed-window, in-memory |
| `POST /api/v1/auth/magic-link` | 3 | per minute | per IP | Fixed-window, in-memory |
| `POST /api/v1/auth/password/reset` | 5 | per minute | per IP | Fixed-window, in-memory |
| `POST /api/v1/auth/mfa/login` | 5 | per minute | per IP | Fixed-window, in-memory |

### General API Endpoints

| Endpoint Pattern | Limit | Window | Scope |
|------------------|-------|--------|-------|
| `GET /api/v1/*` | 100 | per minute | per IP |
| `POST /api/v1/*` | 60 | per minute | per IP |
| `PUT/PATCH/DELETE /api/v1/*` | 30 | per minute | per IP |

### OAuth Endpoints

| Endpoint | Limit | Window | Scope |
|----------|-------|--------|-------|
| `GET /oauth/authorize` | 30 | per minute | per IP |
| `POST /oauth/token` | 30 | per minute | per IP |

### SCIM Endpoints

| Endpoint | Limit | Window | Scope |
|----------|-------|--------|-------|
| `GET /scim/v2/*` | 100 | per minute | per IP |
| `POST/PUT/DELETE /scim/v2/*` | 30 | per minute | per IP |

---

## 429 Response Format

When rate limited, the API returns HTTP 429:

```json
{
  "error": "rate limit exceeded",
  "code": "RATE_LIMITED",
  "retry_after": 60
}
```

### Response Headers

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 60
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1720612860
```

| Header | Description |
|--------|-------------|
| `Retry-After` | Seconds until the rate limit window resets. Wait this long before retrying. |
| `X-RateLimit-Limit` | Maximum requests allowed in the current window. |
| `X-RateLimit-Remaining` | Requests remaining in the current window (0 when rate limited). |
| `X-RateLimit-Reset` | Unix timestamp when the window resets. |

---

## Per-Tenant Rate Limiting

For multi-instance deployments, per-IP limiting is insufficient. GGID supports
per-tenant rate limiting via Redis:

```
Per-tenant limit: configurable via REST API
Default: 1000 requests/minute per tenant
```

Configure via API:

```bash
curl -X PUT "$GW/api/v1/tenants/$TENANT_ID/rate-limit" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"requests_per_minute": 2000}'
```

---

## Handling 429 in Your Application

### Retry Strategy (Exponential Backoff)

```python
import time
import random

def api_call_with_retry(url, max_retries=5):
    for attempt in range(max_retries):
        response = requests.post(url, ...)
        if response.status_code == 429:
            retry_after = int(response.headers.get("Retry-After", 60))
            # Add jitter to avoid thundering herd
            jitter = random.uniform(0, retry_after * 0.1)
            time.sleep(retry_after + jitter)
            continue
        return response
    raise Exception("Max retries exceeded")
```

### Go

```go
func withRetry(ctx context.Context, fn func() (*http.Response, error)) error {
    for attempt := 0; attempt < 5; attempt++ {
        resp, err := fn()
        if err != nil { return err }
        if resp.StatusCode != 429 { return nil }

        retryAfter := resp.Header.Get("Retry-After")
        wait, _ := strconv.Atoi(retryAfter)
        if wait == 0 { wait = 60 }

        jitter := time.Duration(rand.Intn(wait*100)) * time.Millisecond
        select {
        case <-time.After(time.Duration(wait)*time.Second + jitter):
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return errors.New("max retries exceeded")
}
```

### Node.js

```typescript
async function callWithRetry(fn: () => Promise<Response>, maxRetries = 5): Promise<Response> {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    const res = await fn();
    if (res.status !== 429) return res;

    const retryAfter = parseInt(res.headers.get('Retry-After') || '60', 10);
    const jitter = Math.random() * retryAfter * 0.1;
    await new Promise(r => setTimeout(r, (retryAfter + jitter) * 1000));
  }
  throw new Error('Max retries exceeded');
}
```

---

## Best Practices

1. **Respect `Retry-After`** — Always wait the specified duration before retrying
2. **Add jitter** — Randomize retry timing to avoid thundering herd
3. **Cache responses** — Reduce API calls by caching GET responses client-side
4. **Batch operations** — Use bulk import (`POST /users/import`) instead of individual creates
5. **Use webhooks instead of polling** — Subscribe to events rather than polling the API
6. **Implement circuit breakers** — Stop calling the API after repeated 429s

---

## Requesting Limit Increases

### Self-Service (Console)

1. Navigate to **Settings** > **Security** > **Rate Limits**
2. Adjust per-tenant limits (requires admin role)
3. Changes take effect immediately

### API

```bash
curl -X PUT "$GW/api/v1/tenants/$TENANT_ID/rate-limit" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"requests_per_minute": 5000}'
```

### Enterprise

For limits above 10,000 requests/minute, contact your GGID administrator to
configure Gateway-level limits and add additional Gateway replicas.

---

## Multi-Instance Considerations

The default in-memory rate limiter does not share state across Gateway replicas.
For production with multiple Gateway instances:

1. **Use Redis-backed rate limiting** — All Gateway instances share rate limit counters
2. **Configure `GATEWAY_REDIS_RATELIMIT=true`** — Enables Redis-based counters
3. **Sticky sessions** — If using a load balancer, enable sticky sessions so the
   same IP hits the same Gateway instance (preserves in-memory counter accuracy)

---

## Monitoring Rate Limits

### Prometheus Metrics

| Metric | Description |
|--------|-------------|
| `ggid_rate_limit_hits_total{path}` | Counter of rate-limited requests |
| `ggid_rate_limit_remaining{path}` | Gauge of remaining requests in window |

### Grafana

The GGID dashboard includes a "Rate Limit Hits" panel showing:
- Rate-limited requests per minute
- Top rate-limited paths
- Rate-limited IPs

### Alerting

```yaml
- alert: HighRateLimitHits
  expr: rate(ggid_rate_limit_hits_total[5m]) > 10
  for: 5m
  annotations:
    summary: "High rate limit rejections — possible abuse or misconfigured client"
```
