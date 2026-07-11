# Webhook Setup Guide

> Register webhooks, handle callbacks, verify signatures, and configure retry behavior.

---

## 1. Register a Webhook

```bash
JWT="your-admin-jwt"
TENANT="00000000-0000-0000-0000-000000000001"

curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-app.com/webhook",
    "events": ["user.created", "user.deleted", "auth.login_failed"],
    "description": "My app integration"
  }' | jq .
```

**Response:**
```json
{
  "id": "wh_abc123",
  "url": "https://your-app.com/webhook",
  "secret": "whsec_abc123def456",
  "events": ["user.created", "user.deleted", "auth.login_failed"],
  "active": true
}
```

Save the `secret` for signature verification.

---

## 2. Payload Format

```json
{
  "event_id": "evt_unique_123",
  "event_type": "user.created",
  "timestamp": "2025-07-11T12:00:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "usr_abc",
    "email": "user@example.com"
  }
}
```

### Headers

| Header | Description |
|--------|-------------|
| `X-GGID-Signature` | `sha256=<hex>` HMAC |
| `X-GGID-Event` | Event type |
| `X-GGID-Event-ID` | Unique ID for idempotency |

---

## 3. Signature Verification

### Go

```go
func verifySignature(payload []byte, sigHeader, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sigHeader))
}
```

### Node.js

```javascript
const crypto = require('crypto');

function verifySignature(payload, sigHeader, secret) {
    const expected = 'sha256=' + crypto.createHmac('sha256', secret).update(payload).digest('hex');
    return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(sigHeader));
}
```

### Python

```python
import hmac, hashlib

def verify_signature(payload: bytes, sig_header: str, secret: str) -> bool:
    expected = 'sha256=' + hmac.new(secret.encode(), payload, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, sig_header)
```

---

## 4. Retry Strategy

GGID retries with exponential backoff:

| Attempt | Delay |
|---------|------|
| 1 | Immediate |
| 2 | 30s |
| 3 | 2m |
| 4 | 10m |
| 5 | 1h |
| 6+ | Up to 24h |

After 8 failures → dead letter queue.

**Respond 200 to stop retries.**

---

## 5. Testing

```bash
# Send test event
curl -X POST http://localhost:8080/api/v1/webhooks/{id}/test \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT"

# Check delivery status
curl http://localhost:8080/api/v1/webhooks/{id}/deliveries \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

---

## 6. Idempotency

Always deduplicate using `event_id`:

```go
var processed sync.Map

func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    var event WebhookEvent
    json.Unmarshal(body, &event)

    if _, ok := processed.Load(event.EventID); ok {
        w.WriteHeader(200) // Already processed
        return
    }
    processed.Store(event.EventID, true)
    // Process event...
    w.WriteHeader(200)
}
```

---

*See: [Webhook Tutorial](../tutorials/webhook-integration.md) | [API Reference](../api-reference.md)*

*Last updated: 2025-07-11*