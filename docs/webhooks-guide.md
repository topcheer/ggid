# GGID Webhooks Guide

Complete guide to receiving real-time event notifications from GGID via webhooks.

---

## Overview

GGID webhooks deliver event notifications to your application via HTTP POST.
When an event occurs (e.g., user login, role change), GGID sends a signed
request to your configured URL.

---

## Registration

### Create a Webhook

```bash
POST /api/v1/auth/hooks
Authorization: Bearer <admin-token>
X-Tenant-ID: <tenant-uuid>
Content-Type: application/json

{
  "event": "post-login",
  "url": "https://yourapp.com/hooks/ggid",
  "method": "POST",
  "secret": "your-hmac-secret-key",
  "timeout_seconds": 3,
  "enabled": true
}
```

### Response

```json
{
  "id": "hook_abc123",
  "event": "post-login",
  "url": "https://yourapp.com/hooks/ggid",
  "enabled": true,
  "created_at": "2024-07-10T12:00:00Z"
}
```

### List Webhooks

```bash
GET /api/v1/auth/hooks
```

### Update / Delete

```bash
PATCH /api/v1/auth/hooks/hook_abc123
{"enabled": false}

DELETE /api/v1/auth/hooks/hook_abc123
```

---

## Event Types

| Event | Trigger | Can Modify | Can Abort |
|-------|---------|:----------:|:---------:|
| `pre-registration` | Before user registration | Yes | Yes |
| `post-registration` | After user created | No | No |
| `post-login` | After successful login | Yes | No |
| `pre-token-issue` | Before JWT issuance | Yes | Yes |
| `post-token-refresh` | After token refresh | No | No |
| `pre-password-reset` | Before password reset email | Yes | Yes |
| `post-password-change` | After password changed | No | No |
| `pre-mfa-verify` | Before MFA code verification | No | Yes |
| `post-user-lock` | After account locked | No | No |
| `post-user-unlock` | After account unlocked | No | No |

---

## Request Format

### Headers

```
POST /your/hook-url HTTP/1.1
Host: yourapp.com
Content-Type: application/json
User-Agent: GGID-Webhook/1.0
X-GGID-Event: post-login
X-GGID-Signature: sha256=a1b2c3d4...
X-GGID-Delivery: dlq_xyz789
X-GGID-Timestamp: 1720612860
```

### Body

```json
{
  "event": "post-login",
  "timestamp": "2024-07-10T12:00:00.000Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "request_id": "req-abc123",
  "delivery_id": "dlq_xyz789",
  "data": {
    "user_id": "a1b2c3d4-...",
    "username": "john.doe",
    "email": "john@example.com",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "method": "password"
  }
}
```

---

## Response Format

### Allow (default)

HTTP 200 with empty body or:

```json
{"action": "allow"}
```

### Allow with Modifications

```json
{
  "action": "allow",
  "modify": {
    "claims": {"custom_role": "contractor"},
    "metadata": {"department": "engineering"}
  }
}
```

### Deny (abort the flow)

```json
{
  "action": "deny",
  "reason": "Access blocked by security policy"
}
```

---

## Signature Verification

Every webhook is signed with HMAC-SHA256 using your configured secret.

### Python

```python
import hmac, hashlib

def verify(body: bytes, sig: str, secret: str) -> bool:
    expected = hmac.new(secret.encode(), body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", sig)

# Flask
sig = request.headers.get("X-GGID-Signature", "")
if not verify(request.data, sig, "your-hmac-secret"):
    return "invalid", 401
```

### Node.js

```typescript
import crypto from 'crypto';

function verify(body: string, sig: string, secret: string): boolean {
  const expected = crypto.createHmac('sha256', secret).update(body).digest('hex');
  try {
    return crypto.timingSafeEqual(Buffer.from(`sha256=${expected}`), Buffer.from(sig));
  } catch { return false; }
}
```

### Go

```go
func verify(body []byte, sig, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(sig))
}
```

---

## Retry Strategy

| Attempt | Delay | Behavior |
|:-------:|:-----:|----------|
| 1 | Immediate | Initial delivery |
| 2 | 30s | First retry (if non-200 response) |
| 3 | 2 min | Second retry |
| Max | 3 | After 3 failures, event is dropped |

### Extended Retry with Exponential Backoff (Enterprise)

Enterprise tenants can configure extended retry with exponential backoff:

```bash
curl -X PUT $API/api/v1/webhooks/$WEBHOOK_ID \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "retry_config": {
      "strategy": "exponential",
      "max_attempts": 7,
      "initial_delay_seconds": 30,
      "max_delay_seconds": 86400,
      "backoff_multiplier": 2.5,
      "jitter": true
    }
  }'
```

| Attempt | Delay | Cumulative |
|---------|-------|------------|
| 1 | 0s | 0s |
| 2 | 30s | 30s |
| 3 | 75s | 1m45s |
| 4 | 3m8s | 4m53s |
| 5 | 7m51s | 12m44s |
| 6 | 19m38s | 32m22s |
| 7 | 49m14s | 1h21m |

### Dead-Letter Queue

After max retries, failed deliveries are stored for 7 days:

```bash
# View failed deliveries
curl $API/api/v1/webhooks/$WEBHOOK_ID/dlq \
  -H "Authorization: Bearer $TOKEN"

# Replay a failed delivery
curl -X POST $API/api/v1/webhooks/$WEBHOOK_ID/dlq/$DELIVERY_ID/replay \
  -H "Authorization: Bearer $TOKEN"
```

### Fail-Open Behavior

If your webhook endpoint is unreachable or times out, GGID continues the
original flow (fail-open). The event is logged but the user is not blocked.

**Exception:** If you need strict enforcement, ensure high availability for
your webhook service.

---

## Idempotency

The `X-GGID-Delivery` header contains a unique delivery ID. Your handler
should deduplicate based on this ID:

```python
# Flask handler with deduplication
from flask import Flask, request, jsonify

DELIVERED = set()  # use Redis in production

@app.route("/hooks/ggid", methods=["POST"])
def handle():
    delivery_id = request.headers.get("X-GGID-Delivery")
    if delivery_id in DELIVERED:
        return jsonify({"action": "allow"})  # already processed

    DELIVERED.add(delivery_id)

    event = request.json
    if event["event"] == "post-login":
        sync_to_crm(event["data"])

    return jsonify({"action": "allow"})
```

---

## Example: Slack Notification on Login

```python
import requests

SLACK_WEBHOOK = "https://hooks.slack.com/services/..."

@app.route("/hooks/ggid", methods=["POST"])
def handler():
    # Verify signature
    sig = request.headers.get("X-GGID-Signature", "")
    if not verify(request.data, sig, WEBHOOK_SECRET):
        return "", 401

    data = request.json["data"]

    # Send Slack notification
    requests.post(SLACK_WEBHOOK, json={
        "text": f"User {data['username']} logged in from {data['ip_address']}"
    })

    return jsonify({"action": "allow"})
```

---

## Testing

### Send Test Event

```bash
POST /api/v1/auth/hooks/hook_abc123/test
```

Response shows delivery result:

```json
{
  "status_code": 200,
  "duration_ms": 45,
  "response_body": "{\"action\":\"allow\"}"
}
```

### Test Locally with ngrok

```bash
# Expose local server
ngrok http 5000

# Register webhook with ngrok URL
curl -X POST "$GW/api/v1/auth/hooks" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"event": "post-login", "url": "https://abc123.ngrok.io/hooks/ggid", "secret": "test-secret"}'
```

---

## Event Payload Catalog

Complete reference for every webhook event payload.

### user.created

```json
{
  "event": "user.created",
  "delivery_id": "dlv-abc123",
  "timestamp": "2024-01-15T10:30:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice",
    "email": "alice@test.com",
    "created_by": "admin@test.com"
  }
}
```

### user.updated

```json
{
  "event": "user.updated",
  "delivery_id": "dlv-def456",
  "timestamp": "2024-01-15T10:31:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "changes": {
      "email": { "old": "alice@old.com", "new": "alice@new.com" }
    }
  }
}
```

### user.deleted

```json
{
  "event": "user.deleted",
  "delivery_id": "dlv-ghi789",
  "timestamp": "2024-01-15T10:32:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice",
    "deleted_by": "admin@test.com",
    "soft_delete": true
  }
}
```

### role.assigned

```json
{
  "event": "role.assigned",
  "delivery_id": "dlv-jkl012",
  "timestamp": "2024-01-15T10:33:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role_key": "admin",
    "role_id": "660e8400-e29b-41d4-a716-446655440001",
    "assigned_by": "superadmin@test.com"
  }
}
```

### role.revoked

```json
{
  "event": "role.revoked",
  "delivery_id": "dlv-mno345",
  "timestamp": "2024-01-15T10:34:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role_key": "admin",
    "revoked_by": "superadmin@test.com",
    "reason": "policy_violation"
  }
}
```

### policy.changed

```json
{
  "event": "policy.changed",
  "delivery_id": "dlv-pqr678",
  "timestamp": "2024-01-15T10:35:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "policy_id": "770e8400-e29b-41d4-a716-446655440002",
    "action": "updated",
    "changes": ["effect", "conditions"],
    "changed_by": "admin@test.com"
  }
}
```

### session.expired

```json
{
  "event": "session.expired",
  "delivery_id": "dlv-stu901",
  "timestamp": "2024-01-15T10:36:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "session_id": "sess-xyz789",
    "reason": "timeout"
  }
}
```

---

## Webhook Debugging

### Check Delivery Status

```bash
# View delivery history
curl $API/api/v1/webhooks/$WEBHOOK_ID/deliveries \
  -H "Authorization: Bearer $TOKEN" | jq .

# Response
[
  {
    "delivery_id": "dlv-abc123",
    "event": "user.created",
    "status": "delivered",
    "response_code": 200,
    "delivered_at": "2024-01-15T10:30:01Z"
  },
  {
    "delivery_id": "dlv-def456",
    "event": "user.updated",
    "status": "failed",
    "response_code": 500,
    "attempts": 3,
    "last_attempt": "2024-01-15T10:33:00Z"
  }
]
```

---

## Event Type Categories

### User Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `user.created` | New user registered | `user_id`, `username`, `email`, `source` |
| `user.updated` | Profile modified | `user_id`, `changes`, `updated_by` |
| `user.deleted` | User account deleted | `user_id`, `deleted_by`, `hard_delete` |
| `user.activated` | Account activated | `user_id`, `activated_by` |
| `user.deactivated` | Account deactivated | `user_id`, `reason`, `deactivated_by` |
| `user.locked` | Account locked (failed attempts) | `user_id`, `source_ip`, `failed_attempts` |
| `user.unlocked` | Account unlocked | `user_id`, `unlocked_by` |

### Authentication Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `user.login` | Successful login | `user_id`, `source_ip`, `method`, `mfa_used` |
| `user.login.failed` | Failed login attempt | `username`, `source_ip`, `reason` |
| `user.logout` | User logged out | `user_id`, `session_id` |
| `user.password.changed` | Password changed | `user_id`, `changed_by`, `mfa_verified` |
| `user.password.reset` | Password reset | `user_id`, `reset_method` |
| `user.mfa.enabled` | MFA factor enrolled | `user_id`, `method` (totp/sms/webauthn) |
| `user.mfa.disabled` | MFA factor removed | `user_id`, `method`, `disabled_by` |

### OAuth Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `oauth.token.issued` | Access token issued | `client_id`, `user_id`, `grant_type`, `scopes` |
| `oauth.token.refreshed` | Refresh token rotated | `client_id`, `user_id`, `family_id` |
| `oauth.token.revoked` | Token revoked | `client_id`, `user_id`, `reason` |
| `oauth.token.reuse_detected` | Refresh token reuse | `client_id`, `user_id`, `family_id` |
| `oauth.consent.granted` | User granted consent | `user_id`, `client_id`, `scopes` |
| `oauth.consent.revoked` | User revoked consent | `user_id`, `client_id` |
| `oauth.client.created` | New OAuth client registered | `client_id`, `name`, `created_by` |

### Audit Events

| Event | Trigger | Payload Fields |
|-------|---------|----------------|
| `audit.query` | Audit log queried | `queried_by`, `filters`, `result_count` |
| `audit.export` | Audit log exported | `exported_by`, `format`, `date_range` |
| `admin.config.changed` | Tenant config modified | `changed_by`, `section`, `changes` |
| `admin.impersonation.started` | Admin started impersonation | `admin_id`, `target_user_id`, `reason` |
| `admin.impersonation.ended` | Admin stopped impersonation | `admin_id`, `target_user_id`, `duration` |

### Event Payload Example (OAuth)

```json
{
  "event": "oauth.token.reuse_detected",
  "timestamp": "2024-01-15T10:30:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "client_id": "web-app",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "family_id": "fam-abc-123",
    "revoked_token_count": 3,
    "source_ip": "192.168.1.50",
    "severity": "critical"
  }
}
```

### Filtering Events

Webhook registrations can filter by event category:

```bash
curl -X PATCH https://iam.example.com/api/v1/auth/hooks/{id} \
  -d '{
    "events": [
      "user.login",
      "user.login.failed",
      "oauth.token.reuse_detected",
      "admin.impersonation.*"
    ]
  }'
```

Wildcards (`*`) match all events in a category.
