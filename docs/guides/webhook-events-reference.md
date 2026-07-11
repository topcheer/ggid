# Webhook Events Reference

> Complete event type catalog, payload formats, signature verification, and retry behavior.

---

## Event Categories

### Authentication Events

| Event | Trigger | Payload Fields |
|-------|---------|---------------|
| `user.register` | New user created | user_id, username, email |
| `user.login` | Successful login | user_id, ip, method |
| `user.login_failed` | Failed login | username, ip, reason |
| `user.logout` | User logged out | user_id, session_id |
| `user.mfa_verify` | MFA verified | user_id, method (totp/webauthn) |

### User Management Events

| Event | Trigger |
|-------|---------|
| `user.update` | Profile updated |
| `user.delete` | User deleted |
| `user.role_assign` | Role assigned |
| `user.role_revoke` | Role revoked |
| `user.suspend` | Account suspended |

### Authorization Events

| Event | Trigger |
|-------|---------|
| `role.create` | Role created |
| `role.update` | Role modified |
| `role.delete` | Role deleted |
| `policy.check` | Permission check (if logged) |

### Organization Events

| Event | Trigger |
|-------|---------|
| `org.create` | Organization created |
| `org.update` | Org details changed |
| `org.delete` | Organization deleted |

### SCIM Events

| Event | Trigger |
|-------|---------|
| `scim.user_create` | SCIM user provisioning |
| `scim.user_patch` | SCIM PATCH operation |
| `scim.group_update` | SCIM group change |

### Security Events

| Event | Trigger |
|-------|---------|
| `security.rate_limit` | Rate limit triggered |
| `security.ip_block` | IP blocked |
| `security.hash_chain_tamper` | Audit chain broken |

---

## Payload Format

```json
{
  "event_id": "evt_abc123",
  "event": "user.login",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "actor_type": "user",
  "actor_id": "usr_abc123",
  "resource_type": "auth",
  "resource_id": "",
  "timestamp": "2025-07-11T12:00:00Z",
  "metadata": {
    "ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0",
    "method": "password",
    "success": true
  }
}
```

---

## HMAC Signature Verification

Every webhook delivery includes `X-GGID-Signature` header:

```python
import hmac, hashlib

def verify_signature(payload: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

---

## Retry Strategy

| Attempt | Delay | Action on failure |
|---------|-------|-------------------|
| 1 | Immediate | — |
| 2 | 5 seconds | — |
| 3 | 30 seconds | — |
| 4 | 2 minutes | Mark as failed, log |

Endpoint must return HTTP 2xx for success. Any other status triggers retry.

---

## Idempotency

Each delivery includes unique `event_id`. Receivers should deduplicate:

```sql
CREATE TABLE processed_events (
    event_id VARCHAR PRIMARY KEY,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

*See: [Webhook Setup](webhook-setup.md) | [Event-Driven Architecture](../architecture/event-driven.md)*

*Last updated: 2025-07-11*
