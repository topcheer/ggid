# Webhook Events Catalog

Complete reference for every webhook event type in GGID. Includes payload
schema for each event, retry policy details, HMAC signature verification,
and ordering guarantees.

> For webhook registration and management, see
> [Webhooks Guide](webhooks-guide.md).

---

## Table of Contents

- [Event Delivery Model](#event-delivery-model)
- [HMAC Signature Verification](#hmac-signature-verification)
- [Retry Policy](#retry-policy)
- [Ordering Guarantees](#ordering-guarantees)
- [User Events](#user-events)
- [Authentication Events](#authentication-events)
- [Role Events](#role-events)
- [Organization Events](#organization-events)
- [Session Events](#session-events)
- [OAuth Events](#oauth-events)
- [Policy Events](#policy-events)
- [Dead-Letter Queue](#dead-letter-queue)

---

## Event Delivery Model

```
GGID Service → NATS JetStream → Webhook Dispatcher → HTTP POST → Your Endpoint
                                      │
                                      ├─ HMAC-SHA256 signature
                                      ├─ Exponential backoff retry
                                      └─ Dead-letter on permanent failure
```

### Delivery Headers

Every webhook delivery includes these headers:

| Header | Description |
|--------|-------------|
| `X-GGID-Signature` | `sha256=<hex-hmac>` — HMAC-SHA256 of the raw body |
| `X-GGID-Event` | Event type (e.g., `user.created`) |
| `X-GGID-Delivery` | Unique delivery ID (UUID) |
| `X-GGID-Timestamp` | Event timestamp (RFC 3339) |
| `X-GGID-Tenant` | Tenant UUID |
| `Content-Type` | `application/json` |

### Envelope Format

```json
{
  "event_id": "evt-550e8400-e29b-41d4-a716-446655440000",
  "event_type": "user.created",
  "timestamp": "2024-01-15T10:30:00.123Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "delivery_id": "dlv-abc123",
  "attempt": 1,
  "data": { ... }
}
```

---

## HMAC Signature Verification

Every webhook payload is signed with HMAC-SHA256 using the secret configured
at registration time.

### Verification (Python)

```python
import hmac, hashlib

def verify_webhook(body: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        body,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)

# Usage (use raw body, not parsed JSON)
sig = request.headers.get("X-GGID-Signature", "")
if not verify_webhook(request.data, sig, WEBHOOK_SECRET):
    abort(401, "Invalid signature")
```

### Verification (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhook(body, signature, secret) {
    const expected = crypto
        .createHmac('sha256', secret)
        .update(body)  // raw buffer, not string
        .digest('hex');
    return `sha256=${expected}` === signature;
}

// Express middleware
app.use('/webhooks/ggid', express.raw({ type: 'application/json' }), (req, res) => {
    const sig = req.headers['x-ggid-signature'];
    if (!verifyWebhook(req.body, sig, WEBHOOK_SECRET)) {
        return res.status(401).send('Invalid signature');
    }
    const event = JSON.parse(req.body);
    // Process event...
    res.status(200).send('OK');
});
```

### Verification (Go)

```go
func verifyWebhook(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := fmt.Sprintf("sha256=%x", mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

> **Important**: Always verify signatures using the raw request body before
> parsing JSON. Parsing then re-serializing can change whitespace and break
> the signature.

---

## Retry Policy

### Standard Retry Schedule

| Attempt | Delay | Total Elapsed |
|---------|-------|---------------|
| 1 (initial) | Immediate | 0s |
| 2 | 30 seconds | 30s |
| 3 | 2 minutes | 2m 30s |
| 4 | 10 minutes | 12m 30s |
| 5 | 1 hour | 1h 12m |
| 6 | 6 hours | 7h 12m |
| 7 | 24 hours | 1d 7h (final attempt) |

### Success Criteria

A delivery is considered successful when:
- HTTP status code is `200`–`299`
- Response is received within 10 seconds (timeout)

### Failure Handling

| HTTP Status | Action |
|-------------|--------|
| 200–299 | Success — stop retrying |
| 301–308 | Follow redirect (max 3), then retry |
| 400–499 | Client error — retry with backoff |
| 500–599 | Server error — retry with backoff |
| Timeout | Retry with backoff |
| Connection refused | Retry with backoff |

### Enterprise Extended Retry

```json
{
  "retry_config": {
    "max_attempts": 10,
    "initial_delay": "30s",
    "max_delay": "24h",
    "backoff_multiplier": 2.5,
    "retry_on_status": [408, 429, 500, 502, 503, 504]
  }
}
```

---

## Ordering Guarantees

| Guarantee | Behavior |
|-----------|----------|
| **Per-event-type ordering** | Events of the same type for the same tenant are delivered in order |
| **Per-user ordering** | Events for the same user_id are delivered in order |
| **Cross-user ordering** | Not guaranteed — events for different users may arrive out of order |
| **Exactly-once** | At-least-once delivery. Deduplicate using `event_id` |
| **Retry ordering** | Retries do not block subsequent events |

### Deduplication

```python
# Track processed event IDs to handle duplicates
processed = set()

def handle_event(event):
    if event['event_id'] in processed:
        return  # Already processed
    processed.add(event['event_id'])
    # Process event...
```

---

## User Events

### user.created

Triggered when a new user is registered (via API, SCIM, or LDAP auto-provision).

```json
{
  "event_id": "evt-uuid",
  "event_type": "user.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "display_name": "Jane Doe",
    "status": "active",
    "source": "api",
    "groups": [],
    "created_by": "admin-user-id"
  }
}
```

### user.updated

Triggered when user profile fields are modified.

```json
{
  "event_type": "user.updated",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "changes": {
      "display_name": { "old": "Jane Doe", "new": "Jane Smith" },
      "department": { "old": null, "new": "Engineering" }
    },
    "updated_by": "admin-user-id"
  }
}
```

### user.deleted

```json
{
  "event_type": "user.deleted",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "hard_delete": false,
    "deleted_by": "admin-user-id",
    "reason": "offboarding"
  }
}
```

### user.activated / user.deactivated

```json
{
  "event_type": "user.deactivated",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "reason": "security_policy_violation",
    "sessions_revoked": 3,
    "deactivated_by": "security_admin-id"
  }
}
```

### user.locked

```json
{
  "event_type": "user.locked",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "failed_attempts": 5,
    "source_ip": "192.168.1.50",
    "user_agent": "Mozilla/5.0...",
    "locked_at": "2024-01-15T10:30:00Z",
    "auto_unlock_at": "2024-01-15T11:00:00Z"
  }
}
```

---

## Authentication Events

### user.login

```json
{
  "event_type": "user.login",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "source_ip": "192.168.1.50",
    "user_agent": "Mozilla/5.0...",
    "method": "password",
    "mfa_used": true,
    "mfa_method": "totp",
    "session_id": "sess-uuid",
    "device_type": "desktop"
  }
}
```

### user.login.failed

```json
{
  "event_type": "user.login.failed",
  "data": {
    "username": "jane.doe",
    "source_ip": "10.0.0.15",
    "reason": "invalid_password",
    "attempt_number": 3,
    "user_agent": "Mozilla/5.0..."
  }
}
```

### auth.mfa_triggered

```json
{
  "event_type": "auth.mfa_triggered",
  "data": {
    "user_id": "550e8400-...",
    "method": "totp",
    "source_ip": "192.168.1.50",
    "challenge_id": "mfa-challenge-uuid"
  }
}
```

---

## Role Events

### role.assigned

```json
{
  "event_type": "role.assigned",
  "data": {
    "user_id": "550e8400-...",
    "role_id": "role-uuid",
    "role_name": "admin",
    "scope": "tenant",
    "assigned_by": "admin-user-id",
    "expires_at": "2025-12-31T23:59:59Z"
  }
}
```

### role.revoked

```json
{
  "event_type": "role.revoked",
  "data": {
    "user_id": "550e8400-...",
    "role_id": "role-uuid",
    "role_name": "admin",
    "scope": "tenant",
    "revoked_by": "admin-user-id",
    "reason": "role_review"
  }
}
```

---

## Organization Events

### org.member.added

```json
{
  "event_type": "org.member.added",
  "data": {
    "org_id": "org-uuid",
    "org_name": "Engineering",
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "added_by": "admin-user-id"
  }
}
```

### org.member.removed

```json
{
  "event_type": "org.member.removed",
  "data": {
    "org_id": "org-uuid",
    "org_name": "Engineering",
    "user_id": "550e8400-...",
    "removed_by": "admin-user-id"
  }
}
```

---

## Session Events

### session.revoked

```json
{
  "event_type": "session.revoked",
  "data": {
    "session_id": "sess-uuid",
    "user_id": "550e8400-...",
    "revoked_by": "admin-user-id",
    "reason": "admin_revocation",
    "ip_address": "192.168.1.50"
  }
}
```

### session.expired

```json
{
  "event_type": "session.expired",
  "data": {
    "session_id": "sess-uuid",
    "user_id": "550e8400-...",
    "expired_at": "2024-01-15T18:00:00Z"
  }
}
```

---

## OAuth Events

### oauth.token.issued

```json
{
  "event_type": "oauth.token.issued",
  "data": {
    "client_id": "web-app",
    "user_id": "550e8400-...",
    "grant_type": "authorization_code",
    "scopes": ["openid", "profile", "email"],
    "expires_in": 900,
    "source_ip": "192.168.1.50"
  }
}
```

### oauth.consent.granted

```json
{
  "event_type": "oauth.consent.granted",
  "data": {
    "user_id": "550e8400-...",
    "client_id": "third-party-app",
    "scopes": ["openid", "profile"],
    "granted_at": "2024-01-15T10:30:00Z"
  }
}
```

---

## Policy Events

### policy.evaluated

```json
{
  "event_type": "policy.evaluated",
  "data": {
    "user_id": "550e8400-...",
    "resource": "api:/v1/users",
    "action": "read",
    "decision": "allow",
    "policy_id": "policy-uuid",
    "policy_name": "allow-admin-read",
    "evaluation_time_ms": 2,
    "attributes": {
      "role": "admin",
      "department": "engineering"
    }
  }
}
```

### policy.evaluated (denied)

```json
{
  "event_type": "policy.evaluated",
  "data": {
    "user_id": "550e8400-...",
    "resource": "api:/v1/admin/users",
    "action": "delete",
    "decision": "deny",
    "policy_id": "deny-viewer-delete",
    "reason": "insufficient_role",
    "evaluation_time_ms": 1
  }
}
```

---

## Dead-Letter Queue

Events that fail all retry attempts are moved to the dead-letter queue (DLQ)
for manual inspection and replay.

### Query Dead-Letter Events

```bash
curl "https://iam.example.com/api/v1/auth/hooks/{id}/dead-letter" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
[
  {
    "delivery_id": "dlv-abc123",
    "event_id": "evt-uuid",
    "event_type": "user.created",
    "payload": { ... },
    "last_status": 500,
    "last_error": "Internal Server Error",
    "attempts": 7,
    "first_attempt": "2024-01-15T10:30:00Z",
    "last_attempt": "2024-01-16T17:30:00Z"
  }
]
```

### Replay Dead-Letter Event

```bash
curl -X POST "https://iam.example.com/api/v1/auth/hooks/{id}/dead-letter/{delivery_id}/replay" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Purge Dead-Letter

```bash
curl -X DELETE "https://iam.example.com/api/v1/auth/hooks/{id}/dead-letter" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```
