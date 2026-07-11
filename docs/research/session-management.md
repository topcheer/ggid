# Session Management Best Practices

## Overview

Session management is the backbone of web authentication security. This document analyzes session management strategies — cookie vs token, sliding vs absolute timeout, concurrent session control — and maps them to GGID's session model.

> **Related**: [Session Management Design](session-management-design.md), [Session Management IAM](session-management-iam.md), [Session Fixation Prevention](session-fixation-prevention.md), [OIDC Session Management](openid-connect-session-management.md)

## Cookie-Based vs Token-Based Sessions

### Comparison

| Aspect | Cookie-Based | Token-Based (JWT) |
|--------|-------------|-------------------|
| Storage | Browser cookie (httpOnly) | Client-side (localStorage, memory) |
| Transport | Automatic with every request | Manual (Authorization header) |
| CSRF risk | High (needs CSRF tokens) | Low (no auto-send) |
| XSS risk | Low (httpOnly) | High (JS accessible) |
| Revocation | Immediate (delete from store) | Deferred (must wait for expiry) |
| Scaling | Requires session store (sticky/Redis) | Stateless |
| Mobile friendly | Poor (cookie handling) | Excellent |
| Size | Small (session ID) | Large (full JWT payload) |

### GGID's Hybrid Approach

GGID uses **JWT tokens** for API authentication (stateless, mobile-friendly) with **Redis-backed session tracking** for revocation:

```
Login → Generate JWT (RS256) → Store session metadata in Redis
         ↓                         ↓
    Client holds JWT          Server can:
    (Authorization header)     - Track active sessions
                               - Revoke by jti (anti-replay)
                               - Enforce session timeout
                               - List/revoke all user sessions
```

## Timeout Strategies

### Absolute Timeout

User is forced to re-authenticate after a fixed period, regardless of activity:

```
Login → [active] → ... → [absolute timeout: 8h] → RE-AUTH REQUIRED
```

| Use Case | Recommended |
|----------|-------------|
| High-security (banking, healthcare) | 15-30 min |
| Enterprise SaaS | 8-12 hours |
| Consumer apps | 24 hours |

### Sliding/Idle Timeout

Session extends on each activity, expires after inactivity:

```
Login → [active] → activity → [reset timer] → ... → [idle 30m] → EXPIRE
```

| Use Case | Recommended Idle Timeout |
|----------|------------------------|
| Admin console | 15 min |
| Standard dashboard | 30 min |
| Consumer app | 7 days |

### GGID Implementation

GGID implements both:

```go
// CheckSessionTimeout validates session against absolute and idle timeouts
func CheckSessionTimeout(session *Session, absoluteTimeout, idleTimeout time.Duration) error {
    // Absolute timeout
    if time.Since(session.CreatedAt) > absoluteTimeout {
        return ErrSessionExpired
    }
    // Idle timeout
    if time.Since(session.LastActivity) > idleTimeout {
        return ErrSessionIdle
    }
    return nil
}
```

**Configuration**:
```yaml
SESSION_ABSOLUTE_TIMEOUT: 8h     # Force re-auth after 8 hours
SESSION_IDLE_TIMEOUT: 30m        # Expire after 30 min inactivity
```

## Concurrent Session Control

### Strategies

| Strategy | Description | Use Case |
|----------|-------------|----------|
| **Single session** | Only 1 active session per user | High-security banking |
| **N max sessions** | Limit to N concurrent sessions | Enterprise (max 3 devices) |
| **Newest wins** | New login kills oldest session | Consumer apps |
| **Unlimited** | No limit on sessions | Social platforms |

### GGID Implementation

GGID tracks sessions in Redis with per-user session lists:

```go
// CreateSession stores a new session with TTL
session := &domain.Session{
    UserID:       userID,
    TokenHash:    hashToken(token),
    CreatedAt:    time.Now(),
    LastActivity: time.Now(),
    IP:           clientIP,
    UserAgent:    userAgent,
    ExpiresAt:    time.Now().Add(absoluteTimeout),
}
```

**Session revocation**:
```bash
# Revoke a specific session
DELETE /api/v1/sessions/:session_id

# Revoke all sessions for a user
DELETE /api/v1/sessions?user_id=:user_id
```

## Session Fixation Prevention

### Attack Vector

1. Attacker obtains a session ID (via URL, cookie theft)
2. Victim authenticates with attacker's session ID
3. Attacker uses the authenticated session

### Defense: Session Regeneration on Auth

GGID generates a **new** session token after every successful authentication:

```go
// After successful password verification:
oldToken := session.Token
newToken := generateRandomToken(32)  // crypto/rand
session.Token = newToken
sessionRepo.Update(ctx, session)
// Old token is immediately invalid
```

## JWT Anti-Replay with jti

Each JWT contains a unique `jti` (JWT ID) claim. GGID tracks jti values in Redis:

```
Login → JWT with jti=abc123
         ↓
Redis SETNX jti:{tenant}:abc123  → OK (first use)
         ↓
Request with JWT → Check jti in Redis → Found → VALID

Attack: Reuse same JWT
         ↓
Redis already has jti → Still valid until expiry
         ↓
On revocation: DELETE jti:{tenant}:abc123 → Subsequent requests = 401
```

## Refresh Token Rotation

### Strategy

```
Access Token (15 min)  +  Refresh Token (single-use)
                              ↓ used
                    New Access Token + New Refresh Token
                                        ↓
                              Old refresh token INVALIDATED
```

### Benefits

- Stolen refresh tokens are useless after one use
- Detection of token theft (old token used after rotation = alarm)

## Session Storage Architecture

### Redis Session Key Structure

```
# Individual session
session:{tenant_id}:{session_id} → {user_id, ip, ua, created_at, expires_at}
TTL: absolute_timeout

# User's session list
sessions:{tenant_id}:{user_id} → SET of session_ids

# JWT jti tracking
jti:{tenant_id}:{jti} → 1
TTL: token_expiry

# Rate limiting
ratelimit:{tenant_id}:{ip} → token_bucket
```

### Session Data Model

```go
type Session struct {
    ID           uuid.UUID `json:"id"`
    UserID       uuid.UUID `json:"user_id"`
    TenantID     uuid.UUID `json:"tenant_id"`
    TokenHash    string    `json:"-"`          // SHA-256 of token
    IP           string    `json:"ip"`
    UserAgent    string    `json:"user_agent"`
    CreatedAt    time.Time `json:"created_at"`
    LastActivity time.Time `json:"last_activity"`
    ExpiresAt    time.Time `json:"expires_at"`
    Revoked      bool      `json:"revoked"`
}
```

## Industry Comparison

| Feature | Okta | Auth0 | GGID |
|---------|------|-------|------|
| Session type | Cookie + token | Cookie + token | JWT + Redis |
| Absolute timeout | Configurable | Configurable | 8h default |
| Idle timeout | Configurable | Configurable | 30m default |
| Concurrent session limit | Yes | Add-on | Yes (configurable) |
| Session revocation | Immediate | Immediate | Immediate (Redis) |
| jti anti-replay | Yes | No | Yes |
| Refresh token rotation | Yes | Yes | Yes |
| Session fixation defense | Yes | Yes | Yes |
| Device-based sessions | Yes | No | Roadmap |

## Best Practices Checklist

- [ ] httpOnly + Secure + SameSite cookies (if using cookies)
- [ ] JWT in Authorization header (not URL parameter)
- [ ] Short access token expiry (5-15 min)
- [ ] Single-use refresh tokens with rotation
- [ ] jti tracking for revocation
- [ ] Session regeneration on auth elevation
- [ ] Absolute timeout enforced (8h recommended)
- [ ] Idle timeout enforced (15-30 min recommended)
- [ ] Concurrent session limit for privileged accounts
- [ ] Session metadata logged (IP, user agent)
- [ ] Session revocation on password change
- [ ] Session revocation on MFA enrollment change

## See Also

- [Session Management Design](session-management-design.md)
- [Session Management IAM](session-management-iam.md)
- [Session Fixation Prevention](session-fixation-prevention.md)
- [OIDC Session Management](openid-connect-session-management.md)
