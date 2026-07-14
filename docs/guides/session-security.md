# Session Security Guide

Comprehensive session management and security practices for GGID.

## Session Fixation Prevention

GGID generates a fresh session ID after every authentication event:

```go
// After successful login
oldSessionID := session.ID
session.Regenerate() // New random session ID, old invalidated
session.Set("user_id", userID)
session.Set("auth_time", time.Now())
```

Mitigations:
- New session ID on login, privilege change, and MFA step-up
- Old session ID immediately invalidated in Redis
- `Set-Cookie` with new ID overwrites browser cookie
- No session ID in URL parameters

## CSRF Protection

### Double Submit Cookie

```go
// Generate CSRF token on page load
csrfToken := generateCSRFToken()
setCookie(w, "csrf_token", csrfToken, SameSiteStrict)

// Client sends token in header + cookie
// Server verifies match
if r.Header.Get("X-CSRF-Token") != r.Cookie("csrf_token") {
    return ErrCSRFMismatch
}
```

### SameSite Cookies

```http
Set-Cookie: ggid_session=...; HttpOnly; Secure; SameSite=Lax; Path=/
```

| SameSite | Use Case |
|----------|----------|
| Strict | Admin console, sensitive ops |
| Lax (default) | Most web apps — allows top-level navigations |
| None | Cross-site embeds — requires `Secure` |

## Session Binding

### IP Binding

```go
if session.IP != clientIP(r) {
    // IP changed — require re-authentication
    session.Invalidate()
    return redirect("/login?reason=session_binding")
}
```

### Device Fingerprint

- TLS fingerprint (JA3 hash)
- User-Agent string
- Screen resolution hash
- If fingerprint changes >30% → step-up MFA

## Timeout Policies

| Timeout Type | Duration | Trigger |
|-------------|----------|---------|
| Idle timeout | 30 min | No activity |
| Absolute timeout | 8 hours | Hard limit regardless of activity |
| Admin session | 4 hours absolute | Shorter for privileged access |
| API session (remember-me) | 30 days | Only if remember-me flag set |

```go
// Idle timeout check
if time.Since(session.LastActivity()) > 30*time.Minute {
    session.Invalidate()
}
// Absolute timeout
if time.Since(session.CreatedAt()) > 8*time.Hour {
    session.Invalidate()
}
```

## Concurrent Session Limits

| User Type | Max Concurrent Sessions |
|-----------|----------------------|
| Standard user | 5 |
| Admin | 2 |
| Service account | 1 |

When limit exceeded, oldest session is revoked (FIFO).

```go
func enforceSessionLimit(userID string, maxSessions int) {
    sessions := redis.SessionsForUser(userID)
    if len(sessions) >= maxSessions {
        oldest := findOldest(sessions)
        redis.DeleteSession(oldest.ID)
        audit.Log("session_evicted", userID, oldest.ID)
    }
}
```

## Session Revocation

### User-Initiated

```bash
# Revoke specific session
DELETE /api/v1/auth/sessions/{session_id}

# Revoke all sessions (logout everywhere)
DELETE /api/v1/auth/sessions?all=true
```

### Admin-Initiated

```bash
# Security incident — revoke all sessions for a user
DELETE /api/v1/admin/users/{user_id}/sessions
```

### Token Revocation (RFC 7009)

```bash
POST /api/v1/oauth/revoke
token=...&token_type_hint=refresh_token
```

## Session Storage

| Storage | Purpose | TTL |
|---------|---------|-----|
| Redis | Active sessions | Matches idle timeout |
| PostgreSQL | Session metadata (audit) | 7 years |
| JWT | Stateless claims | Matches token TTL |

Redis session structure:
```
session:{session_id} → {
  user_id, tenant_id, ip, device_fingerprint,
  auth_time, last_activity, scopes, mfa_verified
}
```

## Security Headers

```http
Set-Cookie: ggid_session=...; HttpOnly; Secure; SameSite=Lax
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'
X-Content-Type-Options: nosniff
Referrer-Policy: strict-origin-when-cross-origin
```

## Monitoring & Detection

| Event | Action |
|-------|--------|
| Session from new geography | Require MFA step-up |
| Concurrent sessions > limit | Evict oldest |
| Session token reuse after revocation | Alert — possible token theft |
| Rapid session creation (>10/min) | Rate limit, possible brute force |
| Session with mismatched tenant_id | Deny immediately |

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Token Revocation Strategy](token-revocation-strategy.md)
- [Multi-Factor Step-Up](multi-factor-step-up.md)
- CSRF Protection
