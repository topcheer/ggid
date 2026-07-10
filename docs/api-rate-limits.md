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

---

## Per-Endpoint Rate Limits

Each endpoint has specific rate limits tuned for its expected usage pattern:

| Endpoint | Method | Tier Limit | Burst | Window | Notes |
|----------|--------|-----------|-------|--------|-------|
| `/api/v1/auth/login` | POST | 10 | 5 | 1 min | Brute force protection |
| `/api/v1/auth/register` | POST | 5 | 3 | 1 min | Spam prevention |
| `/api/v1/auth/refresh` | POST | 30 | 10 | 1 min | Token refresh |
| `/api/v1/auth/password-reset` | POST | 3 | 2 | 1 min | Reset spam prevention |
| `/api/v1/auth/mfa/verify` | POST | 5 | 3 | 1 min | MFA brute force |
| `/api/v1/users` | GET | 60 | 20 | 1 min | List queries |
| `/api/v1/users` | POST | 20 | 10 | 1 min | User creation |
| `/api/v1/users/{id}` | GET | 120 | 40 | 1 min | Read operations |
| `/api/v1/users/{id}` | PUT | 30 | 10 | 1 min | Updates |
| `/api/v1/users/{id}` | DELETE | 10 | 5 | 1 min | Deletions |
| `/api/v1/roles` | GET | 60 | 20 | 1 min | Role queries |
| `/api/v1/roles` | POST | 20 | 10 | 1 min | Role creation |
| `/api/v1/orgs` | GET | 60 | 20 | 1 min | Org queries |
| `/api/v1/audit/events` | GET | 30 | 10 | 1 min | Audit queries (heavy) |
| `/api/v1/audit/stream` | GET (SSE) | 5 | 2 | 1 min | SSE connections |
| `/oauth/token` | POST | 30 | 10 | 1 min | Token endpoint |
| `/oauth/authorize` | GET | 60 | 20 | 1 min | Authorization |
| `/scim/v2/Users` | GET | 100 | 50 | 1 min | SCIM provisioning |
| `/scim/v2/Users` | POST | 20 | 10 | 1 min | SCIM user creation |
| `/saml/sso` | GET/POST | 30 | 10 | 1 min | SAML SSO |

---

## 429 Response Format

When rate limited, GGID returns HTTP 429 with standardized headers:

### Response Headers

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1721034600
Retry-After: 42
```

### Response Body

```json
{
  "error": "rate_limit_exceeded",
  "error_description": "Rate limit of 10 requests per minute exceeded for this endpoint.",
  "code": "RATE_LIMITED",
  "retry_after_seconds": 42,
  "limit": 10,
  "window_seconds": 60,
  "reset_at": "2024-07-15T10:31:00Z"
}
```

### Header Reference

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests allowed in the window |
| `X-RateLimit-Remaining` | Requests remaining in current window |
| `X-RateLimit-Reset` | Unix timestamp when the window resets |
| `Retry-After` | Seconds to wait before retrying |

---

## Client-Side Handling

### Axios Interceptor (JavaScript)

```typescript
axios.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 429) {
      const retryAfter = error.response.headers['retry-after'];
      const retryAfterSeconds = parseInt(retryAfter) || 60;

      console.warn(`Rate limited. Retrying in ${retryAfterSeconds}s...`);

      // Wait and retry
      await new Promise(resolve => setTimeout(resolve, retryAfterSeconds * 1000));
      return axios.request(error.config);
    }
    return Promise.reject(error);
  }
);
```

### Go HTTP Client

```go
func doWithRetry(client *http.Client, req *http.Request, maxRetries int) (*http.Response, error) {
    for i := 0; i < maxRetries; i++ {
        resp, err := client.Do(req)
        if err != nil {
            return nil, err
        }

        if resp.StatusCode == http.StatusTooManyRequests {
            retryAfter := resp.Header.Get("Retry-After")
            seconds, _ := strconv.Atoi(retryAfter)
            if seconds == 0 {
                seconds = 60
            }
            resp.Body.Close()

            log.Printf("Rate limited, waiting %ds (attempt %d/%d)", seconds, i+1, maxRetries)
            time.Sleep(time.Duration(seconds) * time.Second)
            continue
        }

        return resp, nil
    }
    return nil, fmt.Errorf("max retries exceeded")
}
```

---

## Custom Rate Limit Overrides

Administrators can override default limits per tenant:

```bash
# Set custom limits for a tenant
curl -X PUT $API/api/v1/settings/rate-limits \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "overrides": {
            "/api/v1/auth/login": { "limit": 20, "window": 60 },
            "/api/v1/users": { "limit": 200, "window": 60 },
            "/api/v1/audit/events": { "limit": 100, "window": 60 }
        }
    }'
```

### Per-IP-Range Overrides

```bash
# Unlimited API access for internal IPs
curl -X PUT $API/api/v1/settings/rate-limits/ip-range \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "cidr": "10.0.0.0/8",
        "multiplier": 10
    }'
```

### Bypass for Trusted Clients

```bash
# Create an unlimited API key for trusted internal services
curl -X POST $API/api/v1/api-keys \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "name": "internal-service",
        "rate_limit_bypass": true,
        "scopes": ["users:read", "users:write"]
    }'
```

---

## Redis Implementation

GGID uses a sliding window rate limiter implemented as a Redis Lua script
for atomicity:

```lua
-- ratelimit.lua
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove entries outside the window
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current entries
local count = redis.call('ZCARD', key)

if count < limit then
    -- Allow: add current request
    redis.call('ZADD', key, now, now .. '-' .. math.random())
    redis.call('EXPIRE', key, window)
    return {1, limit - count - 1, now + window}
else
    -- Deny
    return {0, 0, now + window}
end
```

### Key Structure

```
tid:{tenant_id}:rl:{endpoint}:{ip_address}
```

This ensures rate limits are scoped per-tenant, per-endpoint, per-IP.

### Fail-Open vs Fail-Closed

When Redis is unavailable:

| Mode | Behavior | Config |
|------|----------|--------|
| Fail-open (default) | Allow all requests | `RATELIMIT_FAIL_MODE=open` |
| Fail-closed | Deny all requests | `RATELIMIT_FAIL_MODE=closed` |

```bash
# Configure fail mode
RATELIMIT_FAIL_MODE=open
```

---

## Monitoring Rate Limits

### Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ggid_ratelimit_checks_total` | Counter | `endpoint`, `result` | Total rate limit checks |
| `ggid_ratelimit_denied_total` | Counter | `endpoint`, `tenant_id` | Denied requests |
| `ggid_ratelimit_remaining` | Gauge | `endpoint` | Remaining quota |
| `ggid_ratelimit_window_seconds` | Gauge | `endpoint` | Window size |

### Key Queries

```promql
# Denial rate by endpoint
sum(rate(ggid_ratelimit_denied_total[5m])) by (endpoint)

# Top rate-limited tenants
topk(10, sum(rate(ggid_ratelimit_denied_total[5m])) by (tenant_id))
