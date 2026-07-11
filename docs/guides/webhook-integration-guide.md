# Webhook Integration Guide

> Register webhooks, verify signatures, handle retries, implement idempotency.

---

## Register a Webhook

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "events": ["user.login", "user.register", "role.assign"],
    "url": "https://myapp.com/hooks/ggid",
    "secret": "whsec_my_secret_key"
  }'
```

---

## Signature Verification

### Go

```go
func verifyWebhook(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### Python

```python
import hmac, hashlib

def verify_webhook(payload: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(), payload, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

### Node.js

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expected = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}
```

---

## Idempotency

Each delivery includes unique `event_id`. Store processed IDs:

```sql
CREATE TABLE processed_events (
    event_id VARCHAR PRIMARY KEY,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
```

```go
func handleWebhook(event Event) error {
    _, err := db.Exec("INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING", event.ID)
    if err != nil {
        return nil // Already processed
    }
    // Process event...
    return nil
}
```

---

## Retry Logic

GGID retries failed deliveries:

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 5 seconds |
| 3 | 30 seconds |
| 4 | 2 minutes (final) |

Your endpoint must return HTTP 200 within 10 seconds. Any other status triggers retry.

---

## Testing Webhooks

```bash
# Send test event
curl -X POST http://localhost:8080/api/v1/webhooks/test \
  -H "Authorization: Bearer $JWT" \
  -d '{"url": "https://myapp.com/hooks/ggid"}'

# Check delivery status
curl http://localhost:8080/api/v1/webhooks/deliveries?status=failed \
  -H "Authorization: Bearer $JWT"
```

---

*See: [Webhook Events Reference](webhook-events-reference.md) | [Webhook Setup](webhook-setup.md) | [Event-Driven Architecture](../architecture/event-driven.md)*

*Last updated: 2025-07-11*
