# Session Management Guide

This guide covers configuring session timeouts, concurrent session limits, session revocation, and session monitoring in GGID.

## Overview

GGID uses stateless JWTs for authentication with Redis-backed session tracking for revocation and monitoring.

## Session Lifecycle

```
Login → JWT issued (RS256, 15min) + Refresh token (7 days)
  ↓
Each request → Gateway validates JWT signature + checks jti in Redis
  ↓
Token expires (15min) → Client uses refresh token → New JWT issued
  ↓
Idle timeout (30min) → Session marked idle → Next request: 401
  ↓
Absolute timeout (8h) → Session revoked → Must re-authenticate
  ↓
Logout → jti blacklisted in Redis → Token immediately invalid
```

## Configuration

### Timeout Settings

```bash
# Environment variables
SESSION_ABSOLUTE_TIMEOUT=8h      # Force re-auth after 8 hours
SESSION_IDLE_TIMEOUT=30m          # Expire after 30min inactivity
ACCESS_TOKEN_EXPIRY=15m           # JWT access token TTL
REFRESH_TOKEN_EXPIRY=168h         # Refresh token TTL (7 days)
```
### Via API

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "absolute_timeout_minutes": 480,
    "idle_timeout_minutes": 30,
    "access_token_expiry_minutes": 15,
    "refresh_token_expiry_hours": 168,
    "max_concurrent_sessions": 5
  }'
```

### Timeout Recommendations

| Deployment Type | Absolute | Idle | Access Token | Use Case |
|----------------|----------|------|-------------|----------|
| High-security | 2h | 15min | 5min | Banking, healthcare |
| Enterprise | 8h | 30min | 15min | Standard SaaS |
| Consumer | 24h | 7d | 60min | Social platforms |
| Admin console | 4h | 15min | 10min | Admin operations |

## Concurrent Session Control

### Per-User Session Limit

```yaml
MAX_CONCURRENT_SESSIONS: 5  # Max active sessions per user
```

When a user exceeds the limit:

| Strategy | Behavior |
|----------|---------|
| `deny_new` | Block new login, return 429 |
| `revoke_oldest` | Kill oldest session, allow new |

Configure via API:
```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "max_concurrent_sessions": 3,
    "session_overflow_strategy": "revoke_oldest"
  }'
```

## Session Revocation

### Revoke Single Session

```bash
curl -X DELETE https://api.ggid.example.com/api/v1/sessions/$SESSION_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Revoke All User Sessions

```bash
curl -X DELETE "https://api.ggid.example.com/api/v1/sessions?user_id=$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

This adds the user's jti to the Redis blacklist. Subsequent requests with their JWT return 401.

### Auto-Revoke Triggers

Sessions are automatically revoked when:

| Event | Action |
|-------|--------|
| Password changed | All sessions revoked |
| MFA enrolled/removed | All sessions revoked |
| Account locked | All sessions revoked |
| Account deleted | All sessions revoked |
| Admin forces logout | Target user sessions revoked |
| Refresh token reused | Old token's session revoked |

## List Active Sessions

### All Sessions (Admin)

```bash
curl https://api.ggid.example.com/api/v1/sessions \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response**:
```json
{
  "sessions": [
    {
      "id": "session-uuid",
      "user_id": "user-uuid",
      "ip": "192.168.1.50",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2025-01-24T10:00:00Z",
      "last_activity": "2025-01-24T14:25:00Z",
      "expires_at": "2025-01-24T18:00:00Z"
    }
  ]
}
```

### User's Own Sessions

```bash
curl https://api.ggid.example.com/api/v1/users/me/sessions \
  -H "Authorization: Bearer $TOKEN"
```

## JWT Anti-Replay (jti)

Every JWT contains a unique `jti` (JWT ID) claim. GGID tracks these in Redis:

```
Login → JWT jti=abc123
  ↓ Redis SETNX jti:{tenant}:abc123 (TTL: token_expiry)

Request → Gateway checks jti in Redis → Found → VALID

Revoke → DELETE jti:{tenant}:abc123 → Next request → 401

Logout → jti deleted immediately
```

## Refresh Token Rotation

```
Access Token (15min) + Refresh Token (single-use)
                              ↓ used
                    New Access Token + New Refresh Token
                                        ↓
                              Old refresh token INVALIDATED
```

If a **used refresh token is reused** (possible theft):
- Request is rejected (400 invalid_grant)
- Audit event logged: `security.suspicious_activity`
- Optional: All user sessions revoked

## Session Storage

### Redis Key Structure

```
session:{tenant}:{session_id}     → {user_id, ip, ua, created, expires}  TTL: absolute_timeout
sessions:{tenant}:{user_id}        → SET of session_ids
jti:{tenant}:{jti}                 → 1  TTL: token_expiry
```

### Multi-Instance

Redis-backed sessions work across multiple gateway instances. All instances share the same session state via Redis.

## Monitoring

### Session Metrics

| Metric | Description |
|--------|-------------|
| `auth_active_sessions` | Current active session count |
| `auth_session_revoke_total` | Sessions revoked (by reason) |
| `auth_token_refresh_total` | Token refresh operations |
| `auth_session_create_total` | New sessions created |
| `auth_session_expired_total` | Sessions expired naturally |

### Alerts

| Alert | Condition |
|-------|-----------|
| Session spike | Active sessions > 2x average |
| Mass revocation | > 100 revocations in 5m |
| Refresh anomaly | Refresh token reuse detected |

## Security Best Practices

- [ ] Access token TTL <= 15 min
- [ ] Refresh token rotation (single-use)
- [ ] jti anti-replay enabled
- [ ] Absolute timeout <= 8h
- [ ] Idle timeout <= 30min
- [ ] Concurrent session limit for admins
- [ ] Session revocation on password change
- [ ] Session metadata logged (IP, user agent)

## See Also

- [Session Management Research](../research/session-management.md)
- [Password Policy Guide](password-policy-guide.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Performance Tuning](performance-tuning.md)
