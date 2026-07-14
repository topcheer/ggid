# GGID Webhook Events Guide

This guide covers webhook event types, payload schemas, HMAC signature verification, retry policies, and delivery configuration.

> **Related**: [Webhook Integration Guide](webhook-integration-guide.md), Webhook Events Reference

## Overview

GGID delivers real-time event notifications to registered webhook endpoints via HTTP POST with HMAC-SHA256 signatures. Webhooks are managed by the gateway's webhook subsystem (`services/gateway/internal/webhooks/`).

## Event Types

### Authentication Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `user.login` | Successful login | user_id, ip, user_agent, method |
| `user.login_failed` | Failed login attempt | username, ip, reason |
| `user.logout` | User logout | user_id, session_id |
| `user.register` | New user registration | user_id, email, username |
| `user.locked` | Account locked (too many failures) | user_id, reason, attempts |
| `user.unlocked` | Account unlocked by admin | user_id, admin_id |
| `user.password_reset` | Password reset completed | user_id |
| `auth.mfa_enrolled` | MFA device enrolled | user_id, device_type (totp/webauthn) |
| `auth.mfa_removed` | MFA device removed | user_id, device_type |
| `auth.mfa_verify` | MFA verification | user_id, result (success/fail) |

### User Management Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `user.created` | User created (direct or SCIM) | user_id, email, username, source |
| `user.updated` | User profile updated | user_id, fields_changed[] |
| `user.deleted` | User deleted | user_id |
| `user.suspended` | Account suspended | user_id, reason |

### Role & Access Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `role.created` | New role created | role_id, key, name |
| `role.assigned` | Role assigned to user | user_id, role_id |
| `role.revoked` | Role revoked from user | user_id, role_id |
| `permission.granted` | Permission granted | user_id, resource, action |

### Organization Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `org.created` | Organization created | org_id, name, parent_id |
| `org.deleted` | Organization deleted | org_id |
| `org.member_added` | User added to org | org_id, user_id, role |
| `org.member_removed` | User removed from org | org_id, user_id |

### OAuth Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `oauth.consent_granted` | User granted OAuth consent | user_id, client_id, scopes[] |
| `oauth.consent_revoked` | User revoked consent | user_id, client_id |
| `oauth.token_issued` | Token issued | client_id, grant_type, scopes[] |
| `oauth.token_revoked` | Token revoked | client_id, reason |

### Agent Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `agent.registered` | AI agent registered | agent_id, name, type |
| `agent.suspended` | Agent suspended | agent_id, reason |
| `agent.token_exchanged` | Agent token delegated | agent_id, scope, delegation_depth |

### Security Events

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `security.rate_limited` | Rate limit triggered | ip, endpoint, limit |
| `security.circuit_open` | Circuit breaker opened | upstream, failure_count |
| `security.suspicious_activity` | Anomalous behavior detected | user_id, ip, signal_type |

## Payload Schema

### Common Envelope

```json
{
  "event_id": "uuid-v4",
  "event_type": "user.login",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "timestamp": "2025-01-24T14:30:00.123Z",
  "actor": {
    "user_id": "uuid",
    "ip": "192.168.1.50",
    "user_agent": "Mozilla/5.0..."
  },
  "data": { ... }
}
```

### Example: `user.login`

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "user.login",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "timestamp": "2025-01-24T14:30:00.123Z",
  "actor": {
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "ip": "192.168.1.50",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
  },
  "data": {
    "method": "password",
    "mfa_used": true,
    "mfa_method": "totp",
    "session_id": "sess_abc123"
  }
}
```

### Example: `role.assigned`

```json
{
  "event_id": "uuid-v4",
  "event_type": "role.assigned",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "timestamp": "2025-01-24T14:35:00Z",
  "actor": {
    "user_id": "admin-uuid",
    "ip": "10.0.0.5"
  },
  "data": {
    "user_id": "target-user-uuid",
    "role_id": "role-uuid",
    "role_key": "admin",
    "role_name": "Administrator"
  }
}
```

## HMAC Signature Verification

Every webhook delivery includes an HMAC-SHA256 signature header:

### Header Format

```
X-GGID-Signature: sha256=<hex-encoded-hmac>
X-GGID-Event: user.login
X-GGID-Delivery: <delivery-uuid>
X-GGID-Timestamp: 1706104200
```

### Verification (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhook(rawBody, signature, secret) {
    const expected = crypto
        .createHmac('sha256', secret)
        .update(rawBody)
        .digest('hex');

    // Use timing-safe comparison
    const a = Buffer.from(signature.replace('sha256=', ''));
    const b = Buffer.from(expected);

    if (a.length !== b.length) return false;
    return crypto.timingSafeEqual(a, b);
}

// Express middleware
app.post('/webhooks/ggid', (req, res) => {
    const sig = req.headers['x-ggid-signature'];
    const secret = process.env.GGID_WEBHOOK_SECRET;

    if (!verifyWebhook(req.rawBody, sig, secret)) {
        return res.status(401).send('Invalid signature');
    }

    const event = JSON.parse(req.rawBody);
    handleEvent(event);
    res.status(200).send('OK');
});
```

### Verification (Go)

```go
import "crypto/hmac"
import "crypto/sha256"
import "encoding/hex"

func VerifyWebhook(body []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := hex.EncodeToString(mac.Sum(nil))

    received := strings.TrimPrefix(signature, "sha256=")
    return hmac.Equal([]byte(expected), []byte(received))
}
```

### Verification (Python)

```python
import hmac
import hashlib

def verify_webhook(raw_body: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        raw_body,
        hashlib.sha256
    ).hexdigest()
    received = signature.replace('sha256=', '')
    return hmac.compare_digest(expected, received)
```

## Retry Policy

GGID uses exponential backoff for failed deliveries:

| Attempt | Delay | Total Elapsed |
|---------|-------|---------------|
| 1 | Immediate | 0s |
| 2 | 30s | 30s |
| 3 | 2m | 2m30s |
| 4 | 10m | 12m30s |
| 5 | 1h | 1h12m |
| 6 | 6h | 7h12m |
| 7 | 24h | 31h12m |

**Abandonment**: After 7 attempts (~31 hours), the delivery is permanently failed and logged.

### Response Requirements

Your endpoint MUST:
1. Return `200`-`299` status code to acknowledge
2. Respond within **10 seconds**
3. Process asynchronously if needed (return 200 immediately, queue for processing)

Non-2xx responses or timeouts trigger retry.

### Idempotency

GGID includes `event_id` in every delivery. Your handler should be idempotent:

```javascript
const processed = new Set();

function handleEvent(event) {
    if (processed.has(event.event_id)) return; // Already processed
    processed.add(event.event_id);

    // Process event...
}
```

## SSRF Protection

Webhook URLs are validated before delivery:

1. **Protocol**: Only `http://` and `https://` allowed
2. **Private IP blocking**: RFC 1918, loopback, link-local, metadata endpoints
3. **DNS check**: Resolved IP must not be private

Blocked ranges:
- `127.0.0.0/8` (loopback)
- `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` (private)
- `169.254.0.0/16` (link-local, AWS metadata)
- `::1/128`, `fc00::/7`, `fe80::/10` (IPv6)

## Registering a Webhook

```bash
curl -X POST "https://api.ggid.example.com/api/v1/webhooks" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "url": "https://your-app.example.com/webhooks/ggid",
    "secret": "your-webhook-secret",
    "events": ["user.login", "user.login_failed", "role.assigned", "security.suspicious_activity"]
  }'
```

### Listing Webhooks

```bash
curl -X GET "https://api.ggid.example.com/api/v1/webhooks" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Deleting a Webhook

```bash
curl -X DELETE "https://api.ggid.example.com/api/v1/webhooks/$WEBHOOK_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Delivery Log

```bash
curl -X GET "https://api.ggid.example.com/api/v1/webhooks/$WEBHOOK_ID/deliveries?status=failed" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response
{
  "deliveries": [
    {
      "id": "delivery-uuid",
      "event_id": "event-uuid",
      "event_type": "user.login",
      "status": "failed",
      "status_code": 500,
      "attempts": 3,
      "last_attempt": "2025-01-24T14:35:00Z",
      "next_retry": "2025-01-24T14:37:00Z"
    }
  ]
}
```

## See Also

- [Webhook Integration Guide](webhook-integration-guide.md)
- [Audit & SIEM Guide](audit-siem-guide.md)
- [Security Audit Checklist](security-audit-checklist.md)
