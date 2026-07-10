# GGID Plugin API Reference

Complete interface definitions for all auth hooks, plugin manifest format,
and lifecycle events.

---

## Table of Contents

- [Hook Interface](#hook-interface)
- [Hook Events](#hook-events)
- [Webhook Request Format](#webhook-request-format)
- [Webhook Response Format](#webhook-response-format)
- [Plugin Manifest](#plugin-manifest)
- [Lifecycle Events](#lifecycle-events)
- [HMAC Signature Verification](#hmac-signature-verification)

---

## Hook Interface

All hooks are HTTP webhook callbacks. When a registered event occurs, GGID
sends an HTTP POST to the configured URL.

### Registration

```bash
POST /api/v1/auth/hooks
Authorization: Bearer <admin-token>
X-Tenant-ID: <tenant-uuid>
Content-Type: application/json

{
  "event": "pre-registration",
  "url": "https://hooks.yourapp.com/ggid/email-check",
  "method": "POST",
  "secret": "hmac-shared-secret",
  "timeout_seconds": 3,
  "enabled": true
}
```

### Registration Response

```json
{
  "id": "hook_abc123",
  "event": "pre-registration",
  "url": "https://hooks.yourapp.com/ggid/email-check",
  "enabled": true,
  "created_at": "2024-07-10T12:00:00Z"
}
```

---

## Hook Events

### Complete Event List

| Event | Can Modify Data | Can Abort Flow | Data Fields |
|-------|:---:|:---:|-------------|
| `pre-registration` | Yes | Yes | username, email, ip_address, user_agent |
| `post-registration` | No | No | user_id, username, email, tenant_id |
| `post-login` | Yes | No | user_id, username, ip_address, user_agent, method |
| `pre-token-issue` | Yes | Yes | user_id, username, roles, scopes, claims |
| `post-token-refresh` | No | No | user_id, token_id |
| `pre-password-reset` | Yes | Yes | email, ip_address |
| `post-password-change` | No | No | user_id, ip_address |
| `pre-mfa-verify` | No | Yes | user_id, method, ip_address |
| `post-user-lock` | No | No | user_id, reason, locked_by |
| `post-user-unlock` | No | No | user_id, unlocked_by |

---

## Webhook Request Format

### Headers

```
POST /your/hook-url HTTP/1.1
Host: hooks.yourapp.com
Content-Type: application/json
User-Agent: GGID-Webhook/1.0
X-GGID-Event: pre-registration
X-GGID-Signature: sha256=abc123...
X-GGID-Delivery: dlq_xyz789
X-GGID-Timestamp: 1720612860
```

| Header | Description |
|--------|-------------|
| `X-GGID-Event` | Event type that triggered this webhook |
| `X-GGID-Signature` | HMAC-SHA256 signature of the body (if secret is set) |
| `X-GGID-Delivery` | Unique delivery ID for idempotency / deduplication |
| `X-GGID-Timestamp` | Unix timestamp when the webhook was sent |

### Body

```json
{
  "event": "pre-registration",
  "timestamp": "2024-07-10T12:00:00.000Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "request_id": "req-abc123",
  "delivery_id": "dlq_xyz789",
  "data": {
    "username": "john.doe",
    "email": "john@example.com",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
  }
}
```

---

## Webhook Response Format

### Allow (Pass-through)

```json
{
  "action": "allow"
}
```

Or simply HTTP 200 with empty body.

### Allow with Modifications

```json
{
  "action": "allow",
  "modify": {
    "metadata": {
      "department": "engineering",
      "source": "import-batch-7"
    },
    "claims": {
      "custom_role": "contractor",
      "clearance_level": 3
    }
  }
}
```

| `modify` Field | Events | Description |
|----------------|--------|-------------|
| `metadata` | pre-registration, post-login | Adds key-values to user metadata |
| `claims` | pre-token-issue, post-login | Adds custom JWT claims |
| `force_mfa` | post-login | Forces MFA challenge before token issuance |
| `roles` | post-login | Overrides or adds roles for this session |

### Deny (Abort)

```json
{
  "action": "deny",
  "reason": "Email domain not allowed"
}
```

The `reason` string is returned to the client in the error response:
- Registration: `403 Forbidden {"error": "Email domain not allowed"}`
- Login: `401 Unauthorized {"error": "Account suspended"}`
- Token issue: `403 Forbidden {"error": "Token issuance blocked"}`

---

## Plugin Manifest

For compiled Go plugins (future Phase 12), a manifest declares plugin metadata:

```json
{
  "name": "email-denylist",
  "version": "1.0.0",
  "description": "Blocks registration from disposable email domains",
  "author": "GGID Team",
  "license": "Apache-2.0",
  "hooks": [
    {
      "event": "pre-registration",
      "handler": "CheckEmailDomain",
      "priority": 10
    },
    {
      "event": "post-registration",
      "handler": "SyncToCRM",
      "priority": 50
    }
  ],
  "config_schema": {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "properties": {
      "blocked_domains": {
        "type": "array",
        "items": {"type": "string"},
        "default": ["mailinator.com", "tempmail.com"]
      }
    }
  }
}
```

| Field | Required | Description |
|-------|:--------:|-------------|
| `name` | Yes | Plugin identifier (unique within tenant) |
| `version` | Yes | Semantic version |
| `hooks` | Yes | Array of hook registrations |
| `hooks[].event` | Yes | Event name (see [Hook Events](#hook-events)) |
| `hooks[].handler` | Yes | Function name in the plugin binary |
| `hooks[].priority` | No | Execution order (lower = earlier, default 50) |
| `config_schema` | No | JSON Schema for plugin-specific configuration |

---

## Lifecycle Events

```
Plugin Lifecycle
      │
      ├── Install ──── Plugin manifest validated, binary stored
      ├── Enable ───── Hooks registered, starts receiving events
      ├── Disable ──── Hooks unregistered, stops receiving events
      ├── Configure ── Config updated (hot-reload if supported)
      ├── Upgrade ──── New version installed, hooks re-registered
      └── Uninstall ── Binary removed, all hooks deleted
```

### Hook Execution Order

When multiple hooks are registered for the same event:

```
1. Sort by priority (ascending: 1, 10, 50, 100)
2. Execute sequentially
3. If any hook returns "deny" → abort immediately
4. Modifications from each hook are merged (later hooks can override)
5. After all hooks pass → continue the original flow
```

### Fail-Open Behavior

| Scenario | Behavior |
|----------|----------|
| Hook URL unreachable | Log error, continue (fail-open) |
| Hook times out (>3s) | Log error, continue (fail-open) |
| Hook returns 500 | Log error, continue (fail-open) |
| Hook returns invalid JSON | Log warning, continue (fail-open) |
| Hook returns `deny` | Abort flow immediately |

> **Critical hooks:** If you need strict enforcement (no fail-open), run your
> hook service with high availability and health checks. Consider using
> compiled Go plugins (Phase 12) instead of webhooks for zero-latency hooks.

---

## HMAC Signature Verification

If a `secret` is set during hook registration, every webhook request is signed:

### Header

```
X-GGID-Signature: sha256=<hex-hmac>
```

### Verification (Python)

```python
import hmac, hashlib

def verify_signature(body: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        body,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)

# In Flask handler
sig = request.headers.get("X-GGID-Signature", "")
if not verify_signature(request.data, sig, "your-secret"):
    return jsonify({"error": "invalid signature"}), 401
```

### Verification (Node.js)

```typescript
import crypto from 'crypto';

function verifySignature(body: string, signature: string, secret: string): boolean {
  const expected = crypto.createHmac('sha256', secret).update(body).digest('hex');
  try {
    return crypto.timingSafeEqual(
      Buffer.from(`sha256=${expected}`),
      Buffer.from(signature),
    );
  } catch {
    return false;
  }
}

// In Express handler
const sig = req.headers['x-ggid-signature'] as string;
if (!verifySignature(req.rawBody, sig, process.env.GGID_HOOK_SECRET!)) {
  return res.status(401).json({ error: 'invalid signature' });
}
```

### Verification (Go)

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifySignature(body []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

---

## Management API

### List Hooks

```bash
GET /api/v1/auth/hooks
Authorization: Bearer <admin-token>
X-Tenant-ID: <tenant-uuid>
```

Response:
```json
{
  "hooks": [
    {
      "id": "hook_abc123",
      "event": "pre-registration",
      "url": "https://hooks.yourapp.com/email-check",
      "enabled": true,
      "timeout_seconds": 3,
      "created_at": "2024-07-10T12:00:00Z"
    }
  ]
}
```

### Update Hook

```bash
PATCH /api/v1/auth/hooks/hook_abc123
{
  "url": "https://new-url.com/hook",
  "enabled": false
}
```

### Delete Hook

```bash
DELETE /api/v1/auth/hooks/hook_abc123
```

### Send Test Event

```bash
POST /api/v1/auth/hooks/hook_abc123/test
```

Sends a test webhook with dummy data and returns the delivery result:

```json
{
  "status_code": 200,
  "duration_ms": 45,
  "response_body": "{\"action\":\"allow\"}"
}
```
