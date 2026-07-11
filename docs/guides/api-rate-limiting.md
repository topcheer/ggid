# API Rate Limiting

> Configure rate limiting to protect GGID APIs from abuse, DDoS, and credential stuffing.

---

## Built-in Rate Limits

GGID Gateway enforces rate limits at multiple layers:

| Scope | Limit | Algorithm | Key |
|-------|-------|-----------|-----|
| Per-IP (global) | 100 req/s | Token bucket | Client IP |
| Login attempts | 5 per 60s | Sliding window | IP + username |
| Password reset | 3 per hour | Sliding window | IP + email |
| API key | 1000 req/min | Token bucket | API key ID |
| OAuth authorize | 10 per 10s | Sliding window | IP + client_id |

Rate limiter runs **before** auth middleware in the handler chain.

---

## 429 Response

When rate limited, GGID returns:

```json
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 30

{
  "error": "rate_limited",
  "message": "Too many requests. Try again in 30 seconds.",
  "retry_after": 30
}
```

### Client Handling

```javascript
// Retry with exponential backoff
async function apiCall(url, retries = 3) {
  for (let i = 0; i < retries; i++) {
    const resp = await fetch(url);
    if (resp.status === 429) {
      const retryAfter = parseInt(resp.headers.get('Retry-After') || '30');
      await new Promise(r => setTimeout(r, retryAfter * 1000));
      continue;
    }
    return resp;
  }
  throw new Error('Max retries exceeded');
}
```

---

## Adaptive Rate Limiting

GGID includes an adaptive rate limiter that adjusts QPS based on backend response latency:

- **Normal latency (< 100ms):** Full rate limit allowed
- **Degraded latency (100-500ms):** Limit reduced to 50%
- **Critical latency (> 500ms):** Limit reduced to 10% (circuit-breaker mode)

This prevents cascading failures when backend services are slow.

---

## Configuration

### Environment Variables

```bash
# Global rate limit (requests per second per IP)
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200

# Login rate limit (attempts per window)
LOGIN_RATE_LIMIT=5
LOGIN_RATE_WINDOW=60s

# API key rate limit (requests per minute)
API_KEY_RPM=1000
```

### Helm Values

```yaml
gateway:
  rateLimiting:
    enabled: true
    rps: 100
    burst: 200
    loginAttempts: 5
    loginWindow: 60s
```

### Docker Compose

```yaml
gateway:
  environment:
    RATE_LIMIT_RPS: 100
    RATE_LIMIT_BURST: 200
    LOGIN_RATE_LIMIT: 5
```

---

## Custom Strategies

### Per-Tenant Limits

Configure different rate limits per tenant via Policy Service:

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Enterprise tenant rate limit",
    "effect": "allow",
    "actions": ["*"],
    "resources": ["rate_limit:enterprise"]
  }'
```

### Exempting Trusted IPs

```bash
# Whitelist internal monitoring systems
TRUSTED_IPS=10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
```

Trusted IPs bypass rate limiting entirely.

---

## Monitoring

Rate limit metrics exposed at `/metrics` (Prometheus format):

```
ggid_rate_limit_total{scope="ip",result="allowed"} 15423
ggid_rate_limit_total{scope="ip",result="denied"} 42
ggid_rate_limit_total{scope="login",result="denied"} 8
```

### Grafana Dashboard Query

```promql
rate(ggid_rate_limit_total{result="denied"}[5m])
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Getting 429 frequently | Rate limit too low for usage | Increase `RATE_LIMIT_RPS` |
| Login always fails after 5 tries | Auth rate limit too aggressive | Restart auth container or increase `LOGIN_RATE_LIMIT` |
| Monitoring IP rate limited | Not in trusted IPs | Add to `TRUSTED_IPS` |

---

*See: [Performance Tuning](../performance-tuning.md) | [Troubleshooting](troubleshooting.md) | [Security Hardening](security-hardening.md)*

*Last updated: 2025-07-11*
