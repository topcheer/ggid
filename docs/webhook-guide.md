# Webhook Guide

> Complete guide to GGID's webhook system: registration, event types, HMAC signature verification, retry policy, SSRF protection, and integration examples.

---

## Table of Contents

1. [Overview](#overview)
2. [Webhook Registration](#webhook-registration)
3. [Event Types](#event-types)
4. [Payload Format](#payload-format)
5. [HMAC Signature Verification](#hmac-signature-verification)
6. [Retry Policy](#retry-policy)
7. [SSRF Protection](#ssrf-protection)
8. [Delivery Pipeline](#delivery-pipeline)
9. [Integration Examples](#integration-examples)
10. [Troubleshooting](#troubleshooting)

---

## Overview

GGID webhooks deliver real-time event notifications to external systems via HTTP POST. Events flow through a reliable pipeline: service generates event → NATS JetStream → webhook delivery service → external endpoint.

```
┌──────────┐    ┌───────────┐    ┌──────────────┐    ┌─────────────┐
│  GGID    │───▶│   NATS    │───▶│   Webhook    │───▶│  External   │
│ Service  │    │ JetStream │    │   Delivery   │    │  Endpoint   │
│          │    │ (durable) │    │              │    │             │
└──────────┘    └───────────┘    └──────────────┘    └─────────────┘
                                       │
                                       │ On failure
                                       ▼
                               ┌──────────────┐
                               │  Retry Queue │
                               │  (exponential │
                               │   backoff)   │
                               └──────────────┘
```

---

## Webhook Registration

### Register a Webhook Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <uuid>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/webhooks/ggid",
    "events": [
      "user.created",
      "user.updated",
      "user.deleted",
      "auth.login",
      "auth.login_failed"
    ],
    "description": "SIEM integration for user events",
    "active": true
  }'
```

Response: `201 Created`
```json
{
  "id": "wh_a1b2c3d4",
  "url": "https://example.com/webhooks/ggid",
  "events": ["user.created", "user.updated", "user.deleted", "auth.login", "auth.login_failed"],
  "secret": "whsec_abc123def456",
  "description": "SIEM integration for user events",
  "active": true,
  "created_at": "2025-07-11T12:00:00Z"
}
```

**Important**: Save the `secret` value. It is used for HMAC signature verification. It is only shown once during creation.

### List Webhooks

```bash
curl http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <uuid>"
```

Response: `200 OK`
```json
{
  "webhooks": [
    {
      "id": "wh_a1b2c3d4",
      "url": "https://example.com/webhooks/ggid",
      "events": ["user.created", "user.updated"],
      "active": true,
      "created_at": "2025-07-11T12:00:00Z"
    }
  ],
  "total": 1
}
```

### Update Webhook

```bash
curl -X PUT http://localhost:8080/api/v1/webhooks/wh_a1b2c3d4 \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <uuid>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/webhooks/ggid-v2",
    "events": ["user.created", "user.deleted"],
    "active": true
  }'
```

### Delete Webhook

```bash
curl -X DELETE http://localhost:8080/api/v1/webhooks/wh_a1b2c3d4 \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <uuid>"
```

### Test Webhook

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/wh_a1b2c3d4/test \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: <uuid>"
```

Response: `200 OK`
```json
{
  "status": "delivered",
  "response_code": 200,
  "latency_ms": 45,
  "timestamp": "2025-07-11T12:01:00Z"
}
```

---

## Event Types

### User Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `user.created` | New user registered | `user_id`, `email`, `username`, `tenant_id` |
| `user.updated` | User profile modified | `user_id`, `changes`, `tenant_id` |
| `user.deleted` | User account deleted | `user_id`, `deleted_by`, `tenant_id` |
| `user.suspended` | Account temporarily disabled | `user_id`, `reason`, `tenant_id` |
| `user.activated` | Account re-enabled | `user_id`, `tenant_id` |
| `user.role_assigned` | Role given to user | `user_id`, `role_id`, `role_key`, `tenant_id` |
| `user.role_removed` | Role taken from user | `user_id`, `role_id`, `tenant_id` |

### Authentication Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `auth.login` | Successful login | `user_id`, `method`, `client_ip`, `user_agent`, `tenant_id` |
| `auth.login_failed` | Failed login attempt | `username`, `reason`, `client_ip`, `tenant_id` |
| `auth.logout` | User logged out | `user_id`, `session_id`, `tenant_id` |
| `auth.token_refreshed` | JWT refreshed | `user_id`, `session_id`, `tenant_id` |
| `auth.mfa_enrolled` | MFA setup completed | `user_id`, `method` (totp/webauthn), `tenant_id` |
| `auth.mfa_verified` | MFA challenge passed | `user_id`, `method`, `tenant_id` |
| `auth.password_changed` | Password updated | `user_id`, `tenant_id` |
| `auth.account_locked` | Account locked (brute force) | `user_id`, `reason`, `tenant_id` |

### Organization Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `org.created` | New organization | `org_id`, `name`, `parent_org_id`, `tenant_id` |
| `org.updated` | Organization modified | `org_id`, `changes`, `tenant_id` |
| `org.deleted` | Organization removed | `org_id`, `tenant_id` |
| `org.member_added` | User joined org | `org_id`, `user_id`, `role`, `tenant_id` |
| `org.member_removed` | User left org | `org_id`, `user_id`, `tenant_id` |

### System Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `system.config_changed` | System configuration updated | `key`, `changed_by`, `tenant_id` |
| `system.webhook_test` | Webhook test event | `webhook_id`, `timestamp` |

---

## Payload Format

### Standard Envelope

All webhook deliveries use this envelope:

```json
{
  "event_id": "evt_unique_id_123",
  "event_type": "user.created",
  "timestamp": "2025-07-11T12:00:00.123Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "usr_abc123",
    "email": "john@example.com",
    "username": "johndoe",
    "created_at": "2025-07-11T12:00:00Z"
  },
  "metadata": {
    "delivery_attempt": 1,
    "source": "auth-service"
  }
}
```

### HTTP Headers

Each delivery includes:

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `X-GGID-Event` | Event type (e.g., `user.created`) |
| `X-GGID-Event-ID` | Unique event ID for idempotency |
| `X-GGID-Signature` | HMAC-SHA256 signature |
| `X-GGID-Timestamp` | Delivery timestamp |
| `User-Agent` | `GGID-Webhook/1.0` |

---

## HMAC Signature Verification

### How It Works

1. GGID computes `HMAC-SHA256(secret, payload)` using the webhook secret
2. Sends the result in the `X-GGID-Signature` header as `sha256=<hex>`
3. Receiver recomputes the HMAC and compares

### Verification (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhookSignature(payload, signature, secret) {
    const expected = 'sha256=' + crypto
        .createHmac('sha256', secret)
        .update(payload)
        .digest('hex');

    // Use timing-safe comparison
    if (expected.length !== signature.length) {
        return false;
    }
    return crypto.timingSafeEqual(
        Buffer.from(expected),
        Buffer.from(signature)
    );
}

// Express middleware
app.post('/webhooks/ggid', (req, res) => {
    const signature = req.headers['x-ggid-signature'];
    const secret = process.env.GGID_WEBHOOK_SECRET;

    if (!verifyWebhookSignature(req.rawBody, signature, secret)) {
        return res.status(401).json({ error: 'Invalid signature' });
    }

    const event = JSON.parse(req.rawBody);
    handleEvent(event);
    res.status(200).json({ received: true });
});
```

### Verification (Python)

```python
import hmac
import hashlib

def verify_webhook_signature(payload: bytes, signature: str, secret: str) -> bool:
    expected = "sha256=" + hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(expected, signature)

# Flask example
@app.route('/webhooks/ggid', methods=['POST'])
def handle_webhook():
    signature = request.headers.get('X-GGID-Signature', '')
    secret = os.environ['GGID_WEBHOOK_SECRET']

    if not verify_webhook_signature(request.data, signature, secret):
        return jsonify({'error': 'Invalid signature'}), 401

    event = request.get_json()
    process_event(event)
    return jsonify({'received': True}), 200
```

### Verification (Go)

```go
func verifyWebhookSignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

### Important Notes

- Always use **timing-safe comparison** to prevent timing attacks
- Compare against the **raw request body**, not a re-serialized version
- The secret is shown **only once** during webhook creation
- If you lose the secret, you must delete and recreate the webhook

---

## Retry Policy

### Retry Schedule

When an endpoint returns a non-2xx status code or doesn't respond, GGID retries with **exponential backoff**:

| Attempt | Delay After Failure | Total Time |
|---------|---------------------|------------|
| 1 (initial) | — | 0s |
| 2 | 30 seconds | 30s |
| 3 | 2 minutes | 2m30s |
| 4 | 10 minutes | 12m30s |
| 5 | 1 hour | 1h12m |
| 6 | 6 hours | 7h12m |
| 7 | 24 hours | 31h12m |
| 8 (final) | — | 55h12m |

### Timeout

- **Connection timeout**: 10 seconds
- **Response timeout**: 30 seconds
- If either timeout is exceeded, the delivery is treated as a failure

### Success Criteria

- HTTP status code `2xx` (200-299) = success, no retry
- HTTP status code `3xx` = success (followed redirect)
- HTTP status code `4xx` = failure, will retry
- HTTP status code `5xx` = failure, will retry
- Network error = failure, will retry

### Dead Letter Queue

After all 8 attempts fail, the event is placed in a **dead letter queue** for manual inspection:

```bash
# View failed deliveries
curl http://localhost:8080/api/v1/webhooks/wh_a1b2c3d4/failures \
  -H "Authorization: Bearer <JWT>"

# Replay a failed delivery
curl -X POST http://localhost:8080/api/v1/webhooks/wh_a1b2c3d4/failures/evt_123/replay \
  -H "Authorization: Bearer <JWT>"
```

---

## SSRF Protection

GGID includes protections against Server-Side Request Forgery (SSRF) attacks via webhook URLs.

### Blocked IP Ranges

Webhook URLs resolving to these ranges are rejected:

| Range | Description |
|-------|-------------|
| `127.0.0.0/8` | Loopback (localhost) |
| `10.0.0.0/8` | Private network (RFC 1918) |
| `172.16.0.0/12` | Private network (RFC 1918) |
| `192.168.0.0/16` | Private network (RFC 1918) |
| `169.254.0.0/16` | Link-local |
| `::1/128` | IPv6 loopback |
| `fc00::/7` | IPv6 unique local |
| `fe80::/10` | IPv6 link-local |
| `0.0.0.0/8` | Unspecified |

### DNS Resolution Check

1. Domain is resolved to IP addresses
2. Each IP is checked against blocked ranges
3. If any IP is in a blocked range → delivery rejected
4. Delivery uses the resolved IP (prevents DNS rebinding)

### Domain Allowlist (Optional)

Configure an allowlist of permitted webhook destination domains:

```bash
# In environment configuration
WEBHOOK_ALLOWED_DOMAINS=*.example.com,hooks.slack.com,hooks.zapier.com
```

---

## Delivery Pipeline

### Architecture

```
1. Service generates event
   ↓
2. Publishes to NATS JetStream subject "audit.events.>"
   ↓
3. Webhook delivery service consumes events
   ↓
4. For each webhook subscribed to this event type:
   a. Check SSRF protection on webhook URL
   b. Compute HMAC signature
   c. Send HTTP POST with payload + signature
   d. Record delivery result (success/failure)
   ↓
5. On failure: schedule retry with exponential backoff
   ↓
6. After max retries: dead letter queue
```

### Idempotency

- Each event has a unique `event_id`
- The same event may be delivered **more than once** (at-least-once delivery)
- Receivers should implement idempotency by tracking `event_id`
- Example: Store processed event IDs in a database with UNIQUE constraint

---

## Integration Examples

### Slack Integration

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://hooks.slack.com/services/T000/B000/XXX",
    "events": ["auth.login_failed", "auth.account_locked"],
    "description": "Slack alerts for security events"
  }'
```

### SIEM Integration (Splunk)

```python
@app.route('/webhooks/ggid', methods=['POST'])
def splunk_webhook():
    # Verify signature
    if not verify_signature(request):
        return 401

    event = request.get_json()

    # Forward to Splunk HEC
    requests.post(
        f"{SPLUNK_URL}/services/collector",
        headers={"Authorization": f"Splunk {SPLUNK_TOKEN}"},
        json={"event": event}
    )

    return 200
```

### Auto-Provisioning (Okta SCIM)

```javascript
// When user.created event arrives, provision in Okta
app.post('/webhooks/ggid', async (req, res) => {
    const event = req.body;

    if (event.event_type === 'user.created') {
        await okta.createUser({
            profile: {
                firstName: event.data.first_name,
                lastName: event.data.last_name,
                email: event.data.email,
                login: event.data.email
            }
        });
    }

    if (event.event_type === 'user.deleted') {
        await okta.deactivateUser(event.data.user_id);
    }

    res.status(200).json({ processed: true });
});
```

### Discord Bot Notification

```python
@app.route('/webhooks/ggid', methods=['POST'])
def discord_notify():
    event = request.get_json()

    if event['event_type'] in ['auth.account_locked', 'user.role_assigned']:
        import discord
        channel = discord_client.get_channel(SECURITY_CHANNEL_ID)
        await channel.send(f"Security Event: {event['event_type']}\n"
                          f"User: {event['data'].get('user_id', 'N/A')}\n"
                          f"Time: {event['timestamp']}")

    return jsonify({'ok': True})
```

---

## Troubleshooting

### Webhook Not Receiving Events

1. **Check webhook is active**: `GET /api/v1/webhooks` → `active: true`
2. **Check event subscription**: Verify event type is in the `events` list
3. **Check endpoint is reachable**: Use the test endpoint: `POST /webhooks/{id}/test`
4. **Check for SSRF block**: Endpoint must not resolve to private IP
5. **Check NATS is running**: Events flow through NATS JetStream

### Signature Verification Failing

1. Use the **raw request body** (not re-parsed JSON)
2. Ensure the secret matches the one from webhook creation
3. Compare full signature including `sha256=` prefix
4. Use timing-safe comparison function

### Duplicate Events

- GGID uses **at-least-once delivery**: duplicates are expected
- Implement idempotency by tracking `event_id`
- Example database table:
  ```sql
  CREATE TABLE processed_events (
      event_id VARCHAR(255) PRIMARY KEY,
      processed_at TIMESTAMP DEFAULT NOW()
  );
  ```

### Events Arriving Late

- Events are delivered **asynchronously** via NATS
- Normal latency: <1 second
- High latency: NATS backlog or slow endpoint (check delivery latency in API)
- If endpoint consistently slow: increase timeout or process async

---

*Last updated: 2025-07-11*
