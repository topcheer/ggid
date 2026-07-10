# Session Management Design

> Session architecture, token lifecycle, and revocation strategies for GGID.

## 1. Session Architecture Options

### Opaque Session Tokens

Server stores session in Redis/DB; token is a random ID.

- **Pros**: instantly revocable, no payload leakage, small token
- **Cons**: DB/Redis lookup per request, adds latency at scale

### JWT Sessions

Stateless: all claims in token, verified by signature.

- **Pros**: zero DB lookups, horizontally scalable, self-describing
- **Cons**: cannot revoke before `exp`, larger payload (~1-2 KB)

### Hybrid (Current GGID Approach)

GGID already follows this pattern: short-lived JWT access tokens +
opaque refresh tokens persisted in DB and Redis.

```
Client ── Access JWT (15min) ──▶ Gateway  (RS256 signature verify)
Client ◀─ Refresh Token (30d) ── Auth     (opaque, Redis-validated)
```

- Access token: RS256 JWT, 15-min TTL, no DB lookup at gateway.
- Refresh token: 32-byte opaque, SHA-256 hashed, DB + Redis.
- Gap: JWTs cannot be revoked before expiry (see Section 6).

---

## 2. Refresh Token Rotation

### How It Works

Each use of a refresh token issues a new access token AND a new refresh token.
The old refresh token is immediately invalidated.

```
Login → RT₁ (active)
  Refresh(RT₁) → RT₁ (used) → RT₂ (active)
    Refresh(RT₂) → RT₂ (used) → RT₃ (active)

Attack scenario:
  Attacker steals RT₂, uses it → RT₂ used, gets RT₄
  Legitimate user tries RT₂ → ALREADY USED → revoke entire chain
```

### Current GGID Implementation

Rotation with reuse detection is **already implemented** (`token_service.go`):

```go
func (ts *TokenService) RotateRefreshToken(ctx context.Context, plaintext string) (...) {
    tokenHash := hashToken(plaintext)
    ts.rdb.Del(ctx, refreshTokenKey(tokenHash)) // Redis fast-path

    oldToken, _ := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if !oldToken.IsActive() {
        // REUSE DETECTED → revoke all session tokens
        ts.refreshRepo.RevokeAllForSession(ctx, oldToken.SessionID)
        return "", nil, fmt.Errorf("refresh token replay detected")
    }

    ts.refreshRepo.Revoke(ctx, oldToken.ID) // revoke old
    newToken := &domain.RefreshToken{RotatedFrom: &oldToken.ID, ...}
    ts.refreshRepo.Create(ctx, newToken) // issue new
    return ...
}
```

### Gap: Token Family vs Session-Level Revocation

Currently, reuse detection revokes **all tokens for the session**
(`RevokeAllForSession`). The recommended approach is **token family**
revocation, where tokens descended from the same initial authentication share
a `family_id`. This prevents a compromised sibling session from being
collateral-damaged.

| Current | Recommended |
|---------|-------------|
| `RefreshToken.SessionID` links tokens | Add `RefreshToken.FamilyID` field |
| Reuse → revoke by session | Reuse → revoke by family |
| No parent chain traversal | `RotatedFrom` exists but unused for traversal |

```go
// Recommended addition to domain.RefreshToken
type RefreshToken struct {
    // ...existing fields...
    FamilyID uuid.UUID // all tokens from one login share this
}

// On reuse detection:
ts.refreshRepo.RevokeAllForFamily(ctx, oldToken.FamilyID) // not session
```

### Security Analysis

| Scenario | Without Rotation | With Rotation (current) | With Family Revocation |
|----------|-----------------|------------------------|----------------------|
| Attacker steals RT | Persistent access until expiry | One-time use, then detected | Entire family killed |
| Legitimate user | Unaffected | Also kicked out (session-wide) | Only that family killed |
| Detection window | None | Immediate on next refresh | Immediate on next refresh |

---

## 3. Session Fixation Prevention

**Attack**: An attacker sets a known session ID on the victim's browser
(via URL or cookie injection) before the victim logs in. After login, the
attacker reuses the same session ID.

### Prevention

1. **Always generate a fresh session on login.** GGID's `SessionService.Create()`
   generates a new `uuid.New()` + random token — this is implicitly safe.
2. **Clear pre-auth cookies.** The auth handler should invalidate any session
   cookie presented before authentication succeeds.
3. **Bind session to device fingerprint.** GGID has `BindDeviceFingerprint()`
   and `VerifySessionFingerprint()` in `session_management.go`, storing a
   hash of `userAgent:ip` in Redis.

```go
// session_management.go — exists but NOT enforced in gateway
func GenerateDeviceFingerprint(userAgent, ip string) string {
    return hashToken(fmt.Sprintf("%s:%s", userAgent, ip))
}
func (s *AuthService) VerifySessionFingerprint(ctx context.Context, sessionID uuid.UUID, fp string) bool {
    stored, _ := s.rateLimiter.rdb.Get(ctx, "ggid:session_fp:"+sessionID.String()).Result()
    return stored == fp // or true if unset
}
```

### Gap

Fingerprint verification exists but is **not wired into the gateway middleware
pipeline**. The gateway validates JWT signature + expiry but does not check
device fingerprint. An attacker who steals the JWT can use it from any device.

---

## 4. Concurrent Session Limits

GGID implements `EnforceSessionLimit()` in both `session_management.go` and
`auth_service.go`:

```go
func (s *AuthService) EnforceSessionLimit(ctx context.Context, tenantID, userID uuid.UUID) error {
    if s.cfg.SessionTimeout.MaxSessions <= 0 { return nil } // off by default
    // list active sessions, revoke oldest if over limit
}
```

### Gap

- `MaxSessions` defaults to **0** (disabled). Set to 5 for production.
- Gateway `SessionListHandler` uses Redis sets, not DB-backed `SessionRepo` — potential inconsistency.

---

## 5. Idle vs Absolute Timeout

### Configuration (conf.go)

```go
type SessionTimeoutConfig struct {
    AbsoluteTimeout time.Duration // max session lifetime (default: 8h)
    IdleTimeout     time.Duration // inactivity timeout (default: 30m)
    MaxSessions     int           // concurrent sessions (default: 0)
}
```

### How Each Works

| Timeout Type | Mechanism | Default | Enforced Where |
|-------------|-----------|---------|---------------|
| **Access token absolute** | JWT `exp` claim | 15 min | Gateway (signature validation) |
| **Refresh token absolute** | DB `expires_at` column | 30 days | Auth service (DB check on rotate) |
| **Session idle** | Redis TTL reset on activity | 30 min | `touchSessionTTL()` exists but **not called in middleware** |
| **Session absolute** | DB `expires_at` column | 8 hours | Auth service (session lookup) |

### Gap

`touchSessionTTL()` in gateway `session.go` can reset Redis TTL but is **never
invoked** in the request pipeline. Sliding/idle expiration is configured but
not enforced. Active users' sessions expire at the absolute timeout regardless
of activity.

```go
// EXISTS but never called in the request pipeline
func (sm *SessionManager) touchSessionTTL(ctx context.Context, sessionID string, ttl time.Duration) {
    sm.rdb.Expire(ctx, sessionKey(sessionID), ttl)
}
```

### Recommendation

| Token | Absolute | Idle | Notes |
|-------|----------|------|-------|
| Access JWT | 15 min | N/A | Stateless |
| Refresh | 7 days | 24 h | Reduce from 30 days |
| Session | 8 hours | 30 min | Wire `touchSessionTTL` |

---

## 6. Distributed Session Revocation

### Current State

| Mechanism | Status | Detail |
|-----------|--------|--------|
| Refresh token revoke | Implemented | `RevokeRefreshToken()` deletes Redis + marks DB |
| Session revoke | Implemented | `SessionService.Revoke()` sets `RevokedAt` |
| Global logout | Implemented | `LogoutAll()` revokes all sessions + refresh tokens |
| JWT jti blacklist | **NOT implemented** | JWT `jti` claim is generated but never checked |
| NATS revocation event | **NOT implemented** | No event published on revoke |
| CAEP session-revoked | **NOT implemented** | No CAEP integration |

### Redis-Based JWT Revocation (Recommended)

The critical gap: **JWTs cannot be revoked before expiry**. The gateway
verifies signature + `exp` but has no blacklist check.

```
┌──────────┐                        ┌─────────┐
│  Auth    │── Revokes jti ──────▶ │  Redis  │  SET jti:{token_jti} TTL={remaining}
│ Service  │                        │         │
└──────────┘                        └────┬────┘
                                         │ SISMEMBER
                                    ┌────▼────┐
                                    │ Gateway │  check before honoring JWT
                                    └─────────┘
```

```go
// Proposed gateway middleware
func CheckRevocation(rdb *redis.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            jti, _ := r.Context().Value(JTIKey).(string)
            if jti != "" && rdb.Exists(r.Context(), "ggid:jti:"+jti).Val() > 0 {
                writeUnauthorized(w, "token revoked")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

**Performance**: `EXISTS` is O(1), ~0.1 ms. TTL auto-cleans after token expiry.

### NATS Event Propagation

Publish revocation events for multi-instance gateway local cache updates:

```go
nats.Publish("session.revoked", &RevocationEvent{JTI: jti, UserID: uid, Reason: "logout"})
```

Redis is source of truth; NATS reduces per-request Redis lookups.

### CAEP Integration

Publish CAEP `session-revoked` SET events for external RPs:

```json
{"events":{"https://schemas.openid.net/secevent/caep/event-type/session-revoked":
  {"subject":{"iss":"https://iam.ggid.dev","sub":"user-uuid"},"reason":"user_request"}}}
```

---

## 7. GGID Current Session Architecture (Code Audit)

| Feature | Current | Recommended | Gap? |
|---------|---------|-------------|------|
| Access token format | RS256 JWT, 15-min TTL | Same | None |
| JWT claims | `iss`, `sub`, `aud`, `iat`, `exp`, `jti`, `tenant_id` | Add `scope`, `amr` | Minor |
| Refresh token | Opaque, SHA-256 hashed, DB + Redis | Same | None |
| Refresh TTL | 30 days (hardcoded) | 7 days | Reduce |
| Refresh rotation | Implemented | Same | None |
| Reuse detection | Revoke by session ID | Revoke by family ID | Add `FamilyID` |
| Session storage | DB (`SessionRepo`) + Redis sets in gateway | DB authoritative, Redis cache | Sync mechanisms |
| Session revocation | `LogoutAll()` + `ForceLogout()` | Same | None |
| JWT jti blacklist | **Not implemented** | Redis SET with TTL | **P0 gap** |
| Concurrent limit | Code exists, defaults to 0 (off) | Default 5 | Enable |
| Idle timeout | Configured (30 min), `touchSessionTTL` exists | Wire into middleware | **P1 gap** |
| Device fingerprint | `BindDeviceFingerprint` + `VerifySessionFingerprint` | Check in gateway | **P1 gap** |
| Session fixation | Fresh session on login (implicit) | Explicit regeneration | Minor |
| NATS revocation event | Not published | Publish on revoke | **P1 gap** |
| CAEP integration | Not implemented | Publish `session-revoked` | P2 |

---

## 8. Implementation Roadmap

### Phase 1: JWT jti Revocation Set (P0)

- Redis `SET ggid:jti:{jti} 1 EX {remaining}` on logout.
- Gateway checks `EXISTS` before honoring JWT.
- **Effort**: 2-3 days. `gateway/middleware/middleware.go`, `auth/service/logout_all.go`.

### Phase 2: Token Family Revocation (P0)

- Add `FamilyID` to `domain.RefreshToken` + migration.
- Change `RevokeAllForSession` → `RevokeAllForFamily`.
- **Effort**: 2 days. `domain/token.go`, `repo/refresh_token_repo.go`, `service/token_service.go`.

### Phase 3: Concurrent Limits + Idle Timeout (P1)

- Set `MaxSessions` default to 5.
- Wire `touchSessionTTL()` into gateway middleware.
- Reduce refresh TTL 30d → 7d.
- **Effort**: 1 day.

### Phase 4: Device Fingerprint Enforcement (P1)

- Call `VerifySessionFingerprint()` after JWT validation in gateway.
- **Effort**: 1 day.

### Phase 5: NATS Revocation + CAEP (P2)

- Publish `session.revoked` NATS events.
- Implement CAEP `session-revoked` SET for external RPs.
- **Effort**: 3-4 days.

---

### Summary

GGID has a solid foundation: hybrid token architecture, refresh rotation with
reuse detection, and concurrent session management code. The two critical gaps
are: (1) JWTs are not revocable before expiry (no jti blacklist), and (2) reuse
detection operates at session level instead of family level. Addressing these
closes the most significant security exposure.
