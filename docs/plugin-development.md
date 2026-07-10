# GGID Plugin Development Guide

How to write, register, and deploy plugins (hooks) for the GGID Auth Service.

---

## Table of Contents

- [Overview](#overview)
- [Hook Types](#hook-types)
- [Plugin Interface](#plugin-interface)
- [Plugin Lifecycle](#plugin-lifecycle)
- [Error Handling](#error-handling)
- [Example Plugins](#example-plugins)
- [Registering Plugins](#registering-plugins)
- [Testing Plugins](#testing-plugins)
- [Best Practices](#best-practices)

---

## Overview

GGID supports an extensible **auth hooks engine** that allows you to intercept
and modify authentication flows at key points. Hooks are HTTP webhook callbacks
— when a configured event occurs, GGID sends a POST request to your hook URL.

This enables:
- Custom validation (e.g., check an external denylist before registration)
- Side effects (e.g., sync to a CRM after user creation)
- Notification (e.g., alert Slack on suspicious login)
- Token customization (e.g., inject custom claims)

---

## Hook Types

| Hook | Triggered When | Can Modify? | Abort Flow? |
|------|---------------|-------------|-------------|
| `pre-registration` | Before user registration | Yes (add metadata) | Yes (block registration) |
| `post-registration` | After user is created | No | No |
| `post-login` | After successful login | Yes (add custom claims) | No |
| `pre-token-issue` | Before JWT is issued | Yes (add/modify claims) | Yes (block token) |
| `post-token-refresh` | After token refresh | No | No |
| `pre-password-reset` | Before password reset email sent | Yes (override recipient) | Yes (block reset) |
| `post-password-change` | After password is changed | No | No |
| `pre-mfa-verify` | Before MFA code verification | No | Yes (force MFA) |
| `post-user-lock` | After account is locked | No | No |

### Hook Execution Order

```
User submits registration
  │
  ├─► pre-registration hook (can block)
  │     └─► User created in DB
  │           └─► post-registration hook (notification)
  │
User submits login
  │
  ├─► Password verified
  ├─► post-login hook (can add claims)
  ├─► pre-token-issue hook (can modify token, can block)
  │     └─► JWT issued
  └─► Response returned
```

---

## Plugin Interface

### Webhook Payload

All hooks receive an HTTP POST with this JSON structure:

```json
{
  "event": "pre-registration",
  "timestamp": "2024-07-10T12:00:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "request_id": "req-abc123",
  "data": {
    "username": "john.doe",
    "email": "john@example.com",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0..."
  }
}
```

### Webhook Response

Your hook must respond within **3 seconds** with HTTP 200 and optional JSON body:

#### Blocking Response (abort the flow)

```json
{
  "action": "deny",
  "reason": "Email domain not allowed"
}
```

When `action` is `deny`, GGID returns the error to the client:
- Registration: `403 Forbidden` with `{"error": "Email domain not allowed"}`
- Login: `401 Unauthorized`

#### Modification Response (add data)

```json
{
  "action": "allow",
  "modify": {
    "metadata": {
      "department": "engineering",
      "manager": "jane@example.com"
    },
    "claims": {
      "custom_role": "contractor"
    }
  }
}
```

#### Pass-Through Response (no modification)

```json
{
  "action": "allow"
}
```

Or simply HTTP 200 with empty body.

---

## Plugin Lifecycle

```
1. Event occurs (e.g., user submits registration)
      │
2. GGID checks if any hooks are registered for this event
      │
      ├─ No hooks ──► Proceed normally
      │
      ├─ Hook(s) found
      │     │
      │     ├─► POST to hook URL (with 3s timeout)
      │     │     │
      │     │     ├─► 200 + allow ──► Apply modifications, proceed
      │     │     ├─► 200 + deny  ──► Abort with error message
      │     │     ├─► Timeout (3s) ──► Fail-open (proceed, log warning)
      │     │     └─► Non-200 / error ──► Fail-open (proceed, log error)
      │     │
      │     └─► Multiple hooks run sequentially
      │           If any hook denies, flow aborts immediately
      │
3. Flow completes (or aborted with error)
```

### Fail-Open Behavior

If a hook URL is unreachable or times out, GGID **fails open** (proceeds with
the flow). This ensures hooks don't cause availability issues. Failed hook
invocations are logged and visible in the audit trail.

> For critical blocking hooks (e.g., denylist enforcement), ensure your hook
> service is highly available.

---

## Error Handling

### Hook Response Codes

| Response | GGID Behavior |
|----------|---------------|
| 200 + `action: allow` | Proceed, apply modifications |
| 200 + `action: deny` | Abort flow, return reason to client |
| 200 (no body) | Proceed normally |
| 200 (invalid JSON) | Proceed normally, log warning |
| 408 / timeout (>3s) | Proceed normally (fail-open), log error |
| 500 / 502 / 503 | Proceed normally (fail-open), log error |
| Connection refused | Proceed normally (fail-open), log error |

### Idempotency

Hooks may be called multiple times for the same event (e.g., during retries).
Your hook should be idempotent — processing the same event twice should not
cause side effects.

Use `request_id` to deduplicate:

```python
if request_id in processed_set:
    return {"action": "allow"}
processed_set.add(request_id)
```

---

## Example Plugins

### Example 1: Block Disposable Email Registration (pre-registration)

```python
# hook_server.py
from flask import Flask, request, jsonify

app = Flask(__name__)

DISPOSABLE_DOMAINS = {"mailinator.com", "tempmail.com", "guerrillamail.com"}

@app.route("/hooks/email-check", methods=["POST"])
def email_check():
    event = request.json
    email = event["data"]["email"]

    domain = email.split("@")[1].lower()
    if domain in DISPOSABLE_DOMAINS:
        return jsonify({
            "action": "deny",
            "reason": f"Email domain '{domain}' is not allowed"
        }), 200

    return jsonify({"action": "allow"}), 200
```

### Example 2: Slack Notification on Login (post-login)

```python
import requests

@app.route("/hooks/slack-notify", methods=["POST"])
def slack_notify():
    event = request.json
    username = event["data"]["username"]
    ip = event["data"]["ip_address"]

    requests.post("https://hooks.slack.com/services/...", json={
        "text": f":white_check_mark: User *{username}* logged in from {ip}"
    })

    return jsonify({"action": "allow"}), 200
```

### Example 3: Inject Custom JWT Claims (pre-token-issue)

```python
@app.route("/hooks/custom-claims", methods=["POST"])
def custom_claims():
    event = request.json
    user_id = event["data"]["user_id"]

    # Fetch department from your internal system
    department = get_department(user_id)
    clearance = get_clearance_level(user_id)

    return jsonify({
        "action": "allow",
        "modify": {
            "claims": {
                "department": department,
                "clearance_level": clearance
            }
        }
    }), 200
```

### Example 4: Force MFA for High-Risk Logins (post-login)

```python
@app.route("/hooks/risk-check", methods=["POST"])
def risk_check():
    event = request.json
    ip = event["data"]["ip_address"]
    user_agent = event["data"]["user_agent"]

    # Check if login is from a new location or suspicious UA
    risk_score = calculate_risk(ip, user_agent)

    if risk_score > 0.7:
        return jsonify({
            "action": "allow",
            "modify": {
                "force_mfa": True,
                "metadata": {"risk_score": risk_score}
            }
        }), 200

    return jsonify({"action": "allow"}), 200
```

### Example 5: Go Hook Server

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type HookRequest struct {
    Event     string                 `json:"event"`
    Timestamp string                 `json:"timestamp"`
    TenantID  string                 `json:"tenant_id"`
    Data      map[string]interface{} `json:"data"`
}

type HookResponse struct {
    Action string                 `json:"action"`
    Reason string                 `json:"reason,omitempty"`
    Modify map[string]interface{} `json:"modify,omitempty"`
}

func auditHookHandler(w http.ResponseWriter, r *http.Request) {
    var req HookRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Log the event to your SIEM
    fmt.Printf("[HOOK] %s: %+v\n", req.Event, req.Data)

    // Check denylist
    email, _ := req.Data["email"].(string)
    if isDenylisted(email) {
        json.NewEncoder(w).Encode(HookResponse{
            Action: "deny",
            Reason: "Account is suspended",
        })
        return
    }

    json.NewEncoder(w).Encode(HookResponse{Action: "allow"})
}

func main() {
    http.HandleFunc("/hooks/audit", auditHookHandler)
    http.ListenAndServe(":9100", nil)
}
```

---

## Registering Plugins

### Via REST API

```bash
# Register a webhook hook
curl -X POST "$GW/api/v1/auth/hooks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "event": "pre-registration",
    "url": "https://hooks.yourapp.com/email-check",
    "method": "POST",
    "secret": "hmac-secret-for-signing"
  }'
```

### Via Admin Console

1. Navigate to **Settings** > **Webhooks**
2. Click **"Add Hook"**
3. Select the event type
4. Enter your hook URL
5. Optionally set an HMAC secret (sent as `X-GGID-Signature` header)

### Listing Hooks

```bash
curl -s "$GW/api/v1/auth/hooks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

### Deleting a Hook

```bash
curl -X DELETE "$GW/api/v1/auth/hooks/{hook_id}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

---

## Testing Plugins

### Local Testing with ngrok

```bash
# Start your hook server locally
python hook_server.py  # runs on :5000

# Expose it via ngrok
ngrok http 5000

# Register the ngrok URL as a hook
curl -X POST "$GW/api/v1/auth/hooks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "event": "pre-registration",
    "url": "https://abc123.ngrok.io/hooks/email-check"
  }'
```

### Simulating Hook Events

Send a test event directly to your hook:

```bash
curl -X POST https://your-hook-url.com/hooks/email-check \
  -H "Content-Type: application/json" \
  -d '{
    "event": "pre-registration",
    "timestamp": "2024-07-10T12:00:00Z",
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "request_id": "test-req-001",
    "data": {
      "username": "testuser",
      "email": "test@mailinator.com",
      "ip_address": "127.0.0.1"
    }
  }'
```

### HMAC Signature Verification

If you set a `secret`, GGID signs each request with HMAC-SHA256:

```python
import hmac, hashlib

def verify_signature(request_body, signature_header, secret):
    expected = hmac.new(
        secret.encode(),
        request_body,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature_header)

# In your handler
signature = request.headers.get("X-GGID-Signature", "")
if not verify_signature(request.data, signature, "your-hmac-secret"):
    return jsonify({"error": "invalid signature"}), 401
```

---

## Best Practices

1. **Respond within 1 second** — the hard timeout is 3s, but faster is better
2. **Be idempotent** — process each `request_id` only once
3. **Fail gracefully** — if your service is down, GGID fails open (allows the flow)
4. **Use HTTPS** — hook URLs must be HTTPS in production
5. **Verify HMAC signatures** — ensures requests are genuinely from GGID
6. **Log all events** — keep an audit trail of hook invocations
7. **Return meaningful deny reasons** — users see the `reason` field on denial
8. **Test edge cases** — empty fields, Unicode, very long strings
9. **Monitor hook latency** — add the `X-GGID-Signature` timestamp to measure
10. **Don't block critical paths** — use `post-*` hooks for non-critical side effects
