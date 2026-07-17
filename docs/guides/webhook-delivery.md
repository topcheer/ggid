# Webhook Delivery — Technical Guide

> Feature: Audit Webhook Delivery with HMAC Signing + Retry
> Location: `services/audit/internal/server/webhook_delivery_handler.go`
> Endpoints: `/api/v1/audit/webhooks/*`

## What It Does

GGID's webhook system delivers real-time audit events to external systems via HTTP POST. Each delivery is HMAC-signed for integrity, tracked with retry logic, and supports dead-letter queues and replay for failed deliveries.

## Architecture

```
Audit Event Created
         ↓
   Match Event Subscriptions
         ↓
   For each subscriber:
   ┌───────────────────────────┐
   │ 1. Build payload + HMAC   │
   │ 2. POST to webhook URL    │
   │ 3. Record delivery result │
   │ 4. If failed: schedule    │
   │    retry with backoff     │
   │ 5. If exhausted: dead     │
   │    letter queue           │
   └───────────────────────────┘
```

## Event Subscriptions

Subscribers register for specific event types:

| Event Type | Description |
|------------|-------------|
| `user.created` | New user registered |
| `user.deleted` | User account deleted |
| `session.revoked` | Session terminated |
| `role.assigned` | Role granted to user |
| `policy.violation` | Authorization denied |
| `itdr.detection` | Threat detection fired |
| `audit.integrity_failed` | Hash chain broken |

Wildcard `*` subscribes to all events.

## HMAC Signing

Every webhook delivery includes an HMAC-SHA256 signature header:

```
X-GGID-Signature: sha256=<hex_hmac>
X-GGID-Event: user.created
X-GGID-Delivery: <delivery_uuid>
```

**Verification (receiver side):**
```python
import hmac, hashlib
expected = hmac.new(secret, payload, hashlib.sha256).hexdigest()
if hmac.compare_digest(request.headers['X-GGID-Signature'], f'sha256={expected}'):
    # Valid delivery
```

## Retry & Dead Letter

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 30 seconds |
| 3 | 2 minutes |
| 4 | 10 minutes |
| 5 | 1 hour |
| 6 | 6 hours (final) |

After 6 failed attempts, the delivery moves to dead-letter status. Administrators can replay dead-lettered deliveries.

## WebhookDelivery Model

```go
type WebhookDelivery struct {
    ID           string     `json:"id"`
    WebhookID    string     `json:"webhook_id"`
    EventID      string     `json:"event_id"`
    EventType    string     `json:"event_type"`
    Status       string     `json:"status"`       // delivered, failed, pending, dead_letter
    ResponseCode int        `json:"response_code"`
    Attempts     int        `json:"attempts"`
    NextRetryAt  *time.Time `json:"next_retry_at"`
    CreatedAt    string     `json:"created_at"`
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/audit/webhooks` | GET/POST | List/create webhook subscriptions |
| `/api/v1/audit/webhooks/:id/delivery-status` | GET | Check delivery status |
| `/api/v1/audit/webhooks/:id/retry` | POST | Replay a failed delivery |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List webhook subscriptions
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/webhooks" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Create webhook subscription
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/audit/webhooks" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"url":"https://hooks.example.com/ggid","events":["user.created","itdr.detection"],"secret":"my-webhook-secret"}'

# Check delivery status
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/webhooks/wh-123/delivery-status" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Replay failed delivery
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/audit/webhooks/wh-123/retry" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Deliveries always fail | Target URL returns non-2xx | Check receiver endpoint health |
| HMAC verification fails | Wrong secret configured | Update secret in both GGID and receiver |
| Too many dead letters | Receiver consistently down | Fix receiver; replay after recovery |
| Missing events | Event filter too narrow | Add event types to subscription |

## Best Practices

- **Always verify HMAC**: Never process webhooks without signature verification.
- **Return 200 fast**: Acknowledge receipt immediately, process asynchronously.
- **Use idempotency**: Handle duplicate deliveries using the delivery UUID.
- **Monitor dead letters**: Set up alerts for dead-letter queue growth.
- **Rotate secrets**: Periodically rotate webhook signing secrets.
