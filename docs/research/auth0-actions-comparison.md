# Auth0 Actions vs GGID Webhooks

> How Auth0 Actions compare to GGID's auth hooks and webhook system.

---

## Auth0 Actions

Auth0 Actions are serverless functions that run at specific points in the authentication pipeline:

| Trigger | When | GGID Equivalent |
|---------|------|-----------------|
| `post-login` | After successful auth | Post-login webhook |
| `pre-user-registration` | Before user creation | Pre-registration auth hook |
| `post-user-registration` | After user creation | Post-registration webhook |
| `post-change-password` | After password change | Webhook (custom) |
| `send-phone-message` | MFA SMS | Notification service |

### Auth0 Action Example (Node.js)

```javascript
exports.onExecutePostLogin = async (event, api) => {
  // Add custom claim to JWT
  api.idToken.setCustomClaim('https://myapp/role', event.user.app_metadata.role);
  
  // Deny access based on condition
  if (event.user.blocked) {
    api.access.deny('User is blocked');
  }
};
```

### Limitations
- Runs on Auth0's infrastructure (no self-hosting)
- 20-second timeout
- Node.js only (no Go/Python)
- Cold start latency
- Pricing per action execution

---

## GGID Approach: Auth Hooks + Webhooks

GGID uses **webhooks** for the same extensibility:

```bash
# Register a post-login webhook
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "event": "user.login",
    "url": "https://myapp.com/hooks/post-login",
    "secret": "whsec_abc123"
  }'
```

### Webhook Payload

```json
{
  "event_id": "evt_001",
  "event": "user.login",
  "user_id": "usr_abc123",
  "tenant_id": "00000000-...",
  "timestamp": "2025-07-11T12:00:00Z",
  "metadata": {"ip": "192.168.1.1", "success": true},
  "signature": "sha256=abc..."
}
```

### Custom Claims via Auth Hooks

GGID supports pre-token-issue hooks that can inject custom JWT claims:

```go
// Register auth hook
hook := &AuthHook{
    Event:   "pre-token-issue",
    Handler: func(ctx context.Context, user *User, claims *jwt.Claims) error {
        claims.Add("role", user.Roles[0])
        claims.Add("department", user.Metadata["department"])
        return nil
    },
}
```

---

## Comparison Matrix

| Feature | Auth0 Actions | GGID Webhooks |
|---------|--------------|---------------|
| Language | Node.js only | Any (HTTP endpoint) |
| Execution | Auth0-hosted | Your infrastructure |
| Timeout | 20 seconds | No limit |
| Cold start | Yes (serverless) | No |
| Custom claims | Yes | Yes (pre-token-issue hook) |
| Access deny | Yes | Yes (return 403) |
| Self-hosted | No | Yes |
| Cost per execution | Yes | Free |
| Debugging | Auth0 dashboard | Your logs |
| Marketplace | Yes (pre-built) | Template gallery (planned) |

---

## GGID Advantages

1. **Language-agnostic**: Any HTTP endpoint — Go, Python, Rust, anything
2. **No cold start**: Your server is always running
3. **Self-hosted**: Full control over execution environment
4. **No per-execution cost**: Run unlimited webhooks
5. **HMAC-signed**: Every payload signed for verification

## GGID Gaps vs Auth0 Actions

1. **No marketplace**: Auth0 has pre-built actions for common integrations
2. **No inline editor**: Auth0 lets you edit code in the dashboard
3. **No sandboxing**: GGID webhooks run on your server (security boundary needed)

---

## Recommendation

For parity with Auth0 Actions, GGID should:
1. Build a **webhook template gallery** (e.g., "Add role to JWT", "Sync to HR system")
2. Add **inline webhook testing** in Admin Console
3. Document **pre-token-issue hook** more prominently

Priority: P2 (functional parity exists, UX gap only).

---

*See: Webhook Setup | Custom Claims | [Gap Closure Report](gap-closure-report.md)*

*Last updated: 2025-07-11*
