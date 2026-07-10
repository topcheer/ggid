# Session Management Guide

JWT lifecycle, session management, concurrent login limits, and device tracking.

---

## JWT Lifecycle

### Token Pair

| Token | TTL | Purpose | Stored Where |
|-------|-----|---------|--------------|
| Access token | 1 hour | API authentication | Client memory (SPA) or cookie |
| Refresh token | 30 days | Get new access tokens | HttpOnly cookie or secure storage |

### Refresh Token Rotation

Each time a refresh token is used, a **new pair** is issued and the old refresh token is invalidated:

```
Client → POST /auth/refresh {refresh_token: "RT-A"}
Server → validates RT-A
       → issues {access_token: "AT-B", refresh_token: "RT-B"}
       → invalidates RT-A (added to Redis blocklist)
       → RT-A cannot be used again
```

### Token Theft Detection

If RT-A is used twice (attacker stole it):

```
1. Attacker uses RT-A → gets new pair (AT-B, RT-B), RT-A invalidated
2. Legitimate user uses RT-A → REJECTED (already used)
3. Server detects reuse → revokes ALL tokens for this user
4. User must re-authenticate
```

---

## Session Timeout

### Idle Timeout

Sessions expire after a period of inactivity:

```bash
PUT /api/v1/settings/security
{
  "session_idle_timeout_minutes": 30
}
```

If no API call is made within the timeout, the access token is revoked.

### Absolute Timeout

Maximum session lifetime regardless of activity:

```bash
PUT /api/v1/settings/security
{
  "session_absolute_timeout_hours": 8
}
```

After 8 hours, the user must re-authenticate.

### Implementation

| Timeout Type | Mechanism |
|-------------|-----------|
| Access token expiry | JWT `exp` claim (1 hour) |
| Idle timeout | Redis key with sliding TTL |
| Absolute timeout | `iat` claim checked against config |

---

## Concurrent Login Limits

Limit the number of active sessions per user:

```bash
PUT /api/v1/settings/security
{
  "max_concurrent_sessions": 3
}
```

When a user exceeds the limit, the oldest session is revoked (FIFO).

### How It Works

1. User logs in → session added to Redis list: `sessions:{user_id}`
2. On each login, check list length
3. If length >= max, remove oldest session
4. Removed session's JWT is added to blocklist

### Per-Device Sessions

Track sessions by device:

```bash
POST /api/v1/auth/login
{
  "username": "john",
  "password": "...",
  "device_name": "MacBook Pro",
  "device_type": "laptop"
}
```

---

## Session Management API

### List Active Sessions

```bash
GET /api/v1/auth/sessions
Authorization: Bearer <token>
```

Response:
```json
{
  "sessions": [
    {
      "id": "sess-abc123",
      "device_name": "MacBook Pro",
      "device_type": "laptop",
      "ip_address": "192.168.1.100",
      "location": "San Francisco, CA",
      "created_at": "2024-07-10T08:00:00Z",
      "last_active": "2024-07-10T11:30:00Z"
    }
  ]
}
```

### Revoke Session

```bash
DELETE /api/v1/auth/sessions/{session_id}
```

Adds the session's JWT to Redis blocklist. The user is immediately logged out on that device.

### Logout (Current Session)

```bash
POST /api/v1/auth/logout
```

Revokes the current session's access and refresh tokens.

### Logout All (All Sessions)

```bash
POST /api/v1/auth/logout-all
```

Iterates all sessions for the user and revokes each one. Useful for "security incident" response.

---

## Forced Logout (Admin)

```bash
# Revoke all sessions for a specific user
POST /api/v1/users/{user_id}/revoke-sessions
{"reason": "Security incident"}
```

All JWTs for that user are added to the Redis blocklist. User is immediately logged out on all devices.

### Redis Blocklist

```go
// Each revoked JWT's jti is stored in Redis with TTL = remaining token lifetime
key := fmt.Sprintf("blocklist:%s", jti)
redis.Set(ctx, key, "1", timeUntilExpiry)
```

On every request, the Gateway checks:
1. Verify JWT signature (JWKS)
2. Check if `jti` is in blocklist → reject if found

---

## Device Management

### Device Registration

During login, devices are registered:

```json
{
  "device_name": "iPhone 15 Pro",
  "device_type": "mobile",
  "user_agent": "Mozilla/5.0 (iPhone; ...)",
  "push_token": "APNS-token-here"
}
```

### Device Trust

Mark trusted devices to skip MFA:

```bash
POST /api/v1/auth/devices/{device_id}/trust
```

Trusted devices skip MFA challenge (if `skip_mfa_on_trusted_device` policy is enabled).

### Revoke Device

```bash
DELETE /api/v1/auth/devices/{device_id}
```

Revokes all sessions associated with that device.

---

## Best Practices

1. **Short access token TTL** — 15 min for sensitive apps, 1 hour for standard
2. **Always rotate refresh tokens** — Never reuse the same refresh token
3. **Store tokens securely** — Access token in memory, refresh in HttpOnly cookie
4. **Implement idle timeout** — Revoke sessions after inactivity
5. **Monitor session count** — Alert if a user has many concurrent sessions
6. **Log session events** — All login/logout/revoke events should be audited
7. **Use CSRF protection** — If using cookie-based auth, implement CSRF tokens
