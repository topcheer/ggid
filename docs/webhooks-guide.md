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
