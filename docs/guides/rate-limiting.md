# Rate Limiting & Quotas

> Token bucket, sliding window, per-tenant quotas, and adaptive rate limiting.

---

## Built-in Limits

| Scope | Limit | Algorithm | Key |
|-------|-------|-----------|-----|
| Per-IP (global) | 100 req/s | Token bucket | Client IP |
| Login attempts | 5 per 60s | Sliding window | IP + username |
| Password reset | 3 per hour | Sliding window | IP + email |
| API key | 1000 req/min | Token bucket | API key ID |
| Per-tenant | Configurable | Token bucket | tenant_id |

---

## Algorithm Comparison

| Algorithm | Pros | Cons | Use Case |
|-----------|------|------|----------|
| Token bucket | Allows bursts, simple | Hard to set burst | General API |
| Sliding window | Precise, smooth | Memory (counter per window) | Login attempts |
| Fixed window | Simplest | Burst at boundary | Low-precision |
| Leaky bucket | Smooth output | No bursts | Queue protection |

---

## Per-Tenant Quotas

```bash
# Set per-tenant rate limit
curl -X PUT http://localhost:8080/api/v1/tenants/$TENANT/settings \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{"rate_limit_rps": 200, "rate_limit_burst": 400}'
```

Enterprise tenant gets higher limits; free tier gets lower.

---

## 429 Response Handling

```json
{
  "error": "rate_limited",
  "message": "Too many requests",
  "retry_after": 30
}
```

### Client Retry (exponential backoff)

```javascript
async function apiCall(url) {
  for (let attempt = 0; attempt < 3; attempt++) {
    const resp = await fetch(url);
    if (resp.status === 429) {
      const delay = parseInt(resp.headers.get('Retry-After') || '30');
      await new Promise(r => setTimeout(r, delay * 1000));
      continue;
    }
    return resp;
  }
}
```

---

## Configuration

```bash
# Global
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200
LOGIN_RATE_LIMIT=5
LOGIN_RATE_WINDOW=60s
TRUSTED_IPS=10.0.0.0/8
```

---

*See: [API Rate Limiting Guide](api-rate-limiting.md) | [Performance Tuning](performance-tuning.md)*

*Last updated: 2025-07-11*
