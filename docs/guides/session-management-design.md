# Session Management Design

Stateless JWT vs stateful sessions, refresh token rotation, session fixation prevention, concurrent limits, device-aware sessions, timeout policies, and revocation propagation.

## Stateless vs Stateful

| Aspect | Stateless JWT | Stateful (Redis) |
|--------|-------------|-----------------|
| Storage | Client-side (cookie) | Server-side (Redis) |
| Revocation | Hard (need blacklist) | Easy (delete from Redis) |
| Scalability | Excellent | Good (Redis cluster) |
| Latency | 0ms (no lookup) | 0.2ms (Redis GET) |
| Size | ~1KB per token | Session ID only |
| Use case | API access, services | Web apps, admin console |

GGID uses **hybrid**: JWT for API access + Redis for session metadata and revocation.

## Session Lifecycle

```
Login → Create session (Redis + JWT)
  ↓
Activity → Touch session (update last_activity)
  ↓
Idle timeout (30 min) → Expire session
  ↓
Absolute timeout (8h) → Expire regardless
  ↓
Logout → Delete from Redis + blacklist JWT jti
  ↓
Revoked → Deleted from Redis + jti in blacklist
```

## Refresh Token Rotation

See [OAuth Refresh Token Rotation](oauth-refresh-token-rotation.md) for full details.

```
Refresh A → New access + Refresh B (A invalidated)
If A reused → Family revoked (theft detected)
Grace period: 30s for race conditions
```

## Session Fixation Prevention

```go
func afterLogin(session *Session) {
    // Regenerate session ID (invalidate old)
    oldID := session.ID
    session.ID = generateSessionID()

    // Delete old session from Redis
    redis.Del("session:" + oldID)

    // Store new session
    redis.Set("session:"+session.ID, session.Serialize(), sessionTTL)

    // Set new cookie
    setCookie(w, "ggid_session", session.ID, Secure|HttpOnly|SameSiteLax)
}
```

Triggers for regeneration: login, MFA step-up, privilege change, device change.

## Concurrent Session Limits

```go
func enforceSessionLimit(userID string, maxSessions int) {
    sessions := redis.SMembers("user:" + userID + ":sessions")

    if len(sessions) >= maxSessions {
        // Evict oldest (FIFO)
        oldest := findOldest(sessions)
        redis.Del("session:" + oldest)
        redis.SRem("user:"+userID+":sessions", oldest)
        audit.Log("session_evicted", map[string]interface{}{
            "user": userID, "reason": "concurrent_limit",
        })
    }
}
```

| User Type | Max Sessions |
|-----------|-------------|
| Standard | 5 |
| Admin | 2 |
| Service account | 1 |

## Device-Aware Sessions

```go
type Session struct {
    ID                string
    UserID            string
    DeviceFingerprint string  // JA3 + UA + screen hash
    DeviceName        string  // "Jane's iPhone"
    IP                string
    CreatedAt         time.Time
    LastActivity      time.Time
    ExpiresAt         time.Time
    TrustLevel        int     // 1=base, 2=step-up, 3=admin
}
```

### Device Change Detection

```go
func (s *Session) ValidateDevice(r *http.Request) error {
    currentFP := hashDeviceFingerprint(r)
    if s.DeviceFingerprint != currentFP {
        // Fingerprint changed — possible session theft
        if sameSubnet(s.IP, clientIP(r)) {
            return ErrStepUpRequired  // Mild change (new browser version)
        }
        return ErrDeviceMismatch  // Major change — revoke
    }
    return nil
}
```

## Timeout Policies

| Session Type | Idle Timeout | Absolute Timeout |
|-------------|-------------|-----------------|
| Standard user | 30 min | 8 hours |
| Admin | 15 min | 4 hours |
| Remember-me | 30 days | 30 days (no idle) |
| API (JWT only) | N/A (token TTL) | Token TTL (15 min) |
| Step-up elevation | 10 min | 10 min (absolute) |

```go
func checkTimeout(session *Session) error {
    // Idle timeout
    if time.Since(session.LastActivity) > idleTimeout {
        return ErrSessionIdleExpired
    }
    // Absolute timeout
    if time.Since(session.CreatedAt) > absoluteTimeout {
        return ErrSessionAbsoluteExpired
    }
    return nil
}
```

## Revocation Propagation

### User-Initiated

```bash
# Revoke specific session
DELETE /api/v1/auth/sessions/{session_id}

# Revoke all sessions (logout everywhere)
DELETE /api/v1/auth/sessions?all=true
```

### Admin-Initiated

```bash
# Security incident — revoke all sessions for user
DELETE /api/v1/admin/users/{user_id}/sessions
```

### Propagation Mechanism

```go
func revokeSession(sessionID string) {
    // 1. Delete from Redis (immediate)
    redis.Del("session:" + sessionID)

    // 2. Add JWT jti to blacklist (for JWT-based access)
    jti := getJTIForSession(sessionID)
    redis.Set("jwt:blacklist:"+jti, "1", 15*time.Minute) // Until JWT expires

    // 3. Publish NATS event (other services update local cache)
    nats.Publish("session.revoked", sessionID)

    // 4. Audit
    audit.Log("session.revoked", map[string]interface{}{
        "session_id": sessionID,
    })
}
```

## Security Headers

```http
Set-Cookie: ggid_session=...; HttpOnly; Secure; SameSite=Lax; Path=/
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Session creation rate | Spike → possible bot/attack |
| Concurrent session evictions | High → users sharing accounts |
| Device mismatch detections | Any → investigate |
| Idle timeout evictions | Track baseline |

## See Also

- [Session Security](session-security.md)
- [Session Clustering](session-clustering.md)
- [OAuth Refresh Token Rotation](oauth-refresh-token-rotation.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
