# Session Management Guide

This guide covers session lifecycle, timeout configuration, concurrent sessions, session binding, revocation, failover, and degraded mode.

> **Note**: For a different session management guide that was previously created, see [session-management-guide.md](session-management-guide.md).

## Session Lifecycle

```
Login → JWT (15min) + Refresh Token (7d)
  ↓ Each request: Gateway verifies JWT + checks jti in Redis
  ↓ Token expires → Refresh → New JWT + New Refresh (rotation)
  ↓ Idle timeout (30min) → Session idle → 401
  ↓ Absolute timeout (8h) → Session revoked → Must re-authenticate
  ↓ Logout → jti blacklisted → Immediately invalid
```

## Timeout Configuration

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

| Deployment | Absolute | Idle | Token TTL |
|-----------|----------|------|-----------|
| High-security | 2h | 15min | 5min |
| Enterprise | 8h | 30min | 15min |
| Consumer | 24h | 7d | 60min |
| Admin console | 4h | 15min | 10min |

## Concurrent Sessions

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/sessions \
  -d '{
    "max_concurrent_sessions": 3,
    "session_overflow_strategy": "revoke_oldest"
  }'
```

| Strategy | Behavior |
|----------|---------|
| `deny_new` | Block new login, return 429 |
| `revoke_oldest` | Kill oldest session |

## Session Binding

GGID can bind sessions to additional context for security:

### IP Binding

Sessions can be bound to the client IP. If the IP changes mid-session, the session is revoked:

```yaml
session:
  ip_binding: false  # Disabled by default (mobile users change IPs frequently)
```

### Device Binding

Sessions can be bound to a device fingerprint:

```json
{
  "device_fingerprint": "sha256:abc123...",
  "require_match": true
}
```

## Revocation

### Revoke Single Session
```
DELETE /api/v1/sessions/{id}
```

### Revoke All User Sessions
```
DELETE /api/v1/sessions?user_id={user_id}
```

### Auto-Revoke Triggers

| Event | Action |
|-------|--------|
| Password changed | All sessions revoked |
| MFA enrolled/removed | All sessions revoked |
| Account locked | All sessions revoked |
| Account deleted | All sessions revoked |
| Refresh token reused | Token family revoked |

## Redis Failover

### Redis Unavailable

When Redis is down, GGID enters **degraded mode**:

| Feature | Degraded Behavior |
|---------|------------------|
| JWT verification | Works (stateless signature check) |
| jti anti-replay | Skipped (best-effort) |
| Session list | Empty (can't query) |
| Session revocation | Deferred until Redis recovers |
| Rate limiting | Disabled |

**Security implication**: In degraded mode, revoked tokens remain valid until expiry. Keep token TTL short (<= 15min) to limit exposure.

### Redis Sentinel/Cluster

For HA, deploy Redis Sentinel or Redis Cluster:

```yaml
redis:
  mode: sentinel
  master_name: ggid-redis
  sentinel_addrs:
    - redis-sentinel-1:26379
    - redis-sentinel-2:26379
    - redis-sentinel-3:26379
```

## Session Storage

```
session:{tenant}:{session_id}     → JSON, TTL: absolute_timeout
sessions:{tenant}:{user_id}        → SET of session_ids
jti:{tenant}:{jti}                 → 1, TTL: token_expiry
```

## Monitoring

| Metric | Description |
|--------|-------------|
| `auth_active_sessions` | Current sessions |
| `auth_session_create_total` | New sessions |
| `auth_session_revoke_total` | Revocations (by reason) |
| `auth_token_refresh_total` | Refresh operations |
| `auth_redis_degraded` | Redis availability (0/1) |

## See Also

- [Session Management Research](../research/session-management.md)
- [Session Management Guide (config)](session-management-guide.md)
- [Password Policy](password-policy-guide.md)
