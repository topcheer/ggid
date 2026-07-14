# OAuth Refresh Token Rotation

Detection + reuse handling, token family revocation, rotation vs non-rotation tradeoffs, backward compat, and migration.

## Overview

Refresh token rotation issues a new refresh token with every access token refresh. The old refresh token is immediately invalidated. If a used (already-rotated) token appears again, GGID detects theft and revokes the entire token family.

## Rotation Flow

```
Login → access_token (15min) + refresh_token_A (7d)

Refresh with A → access_token (15min) + refresh_token_B (7d)
                  A is now INVALID

Refresh with B → access_token (15min) + refresh_token_C (7d)
                  B is now INVALID

If A is used again (after B was issued):
  → A is already-rotated → THEFT DETECTED
  → Revoke entire family (A, B, C + all access tokens)
```

## Implementation

### Token Family

Each login creates a new token family. All refresh tokens in a family share a `family_id`:

```go
type RefreshToken struct {
    ID         string    // Unique per token
    FamilyID   string    // Shared across all rotations from same login
    UserID     string
    TenantID   string
    Scopes     []string
    CreatedAt  time.Time
    ExpiresAt  time.Time
    UsedAt     *time.Time  // Set when rotated
    RevokedAt  *time.Time
}
```

### Rotation Logic

```go
func RotateRefreshToken(ctx context.Context, oldTokenID string) (*TokenPair, error) {
    // 1. Look up the presented refresh token
    oldToken, err := store.Get(oldTokenID)
    if err != nil { return nil, ErrInvalidToken }

    // 2. Check expiry
    if time.Now().After(oldToken.ExpiresAt) {
        return nil, ErrTokenExpired
    }

    // 3. Check if already used (REUSE DETECTION)
    if oldToken.UsedAt != nil {
        // TOKEN THEFT! Attacker is replaying an old token
        revokeEntireFamily(ctx, oldToken.FamilyID)
        audit.Log("refresh_token_reuse", oldToken)
        return nil, ErrTokenReuseDetected
    }

    // 4. Mark old token as used
    now := time.Now()
    oldToken.UsedAt = &now
    store.Update(oldToken)

    // 5. Issue new tokens (same family)
    newRefresh := &RefreshToken{
        ID:       uuid.New(),
        FamilyID: oldToken.FamilyID,  // Same family
        UserID:   oldToken.UserID,
        Scopes:   oldToken.Scopes,
        ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
    }

    newAccess := issueAccessToken(oldToken.UserID, oldToken.Scopes)

    store.Create(newRefresh)
    return &TokenPair{Access: newAccess, Refresh: newRefresh.ID}, nil
}
```

### Family Revocation

```go
func revokeEntireFamily(ctx context.Context, familyID string) error {
    tokens, err := store.GetByFamily(familyID)
    if err != nil { return err }

    now := time.Now()
    for _, t := range tokens {
        t.RevokedAt = &now
        store.Update(t)
    }

    // Also revoke all access tokens from this family
    jtiList := store.GetJTIsByFamily(familyID)
    for _, jti := range jtiList {
        redis.Set("jwt:blacklist:"+jti, "1", 15*time.Minute)
    }

    // Security alert
    alert.Send("refresh_token_theft_detected", map[string]interface{}{
        "family_id": familyID,
        "action": "all_tokens_revoked",
    })

    return nil
}
```

## Reuse Detection Scenarios

### Scenario 1: Legitimate Client Race

```
Client sends two refresh requests simultaneously (race)
  → Request 1: A → B (A marked used)
  → Request 2: A → REUSE DETECTED → family revoked
```

Mitigation: Clients must serialize refresh calls. Use a mutex or single-flight.

### Scenario 2: Token Theft

```
Attacker copies refresh_token_A from compromised client
  → Legitimate client refreshes: A → B
  → Attacker tries A later → REUSE DETECTED → family revoked
  → OR: Attacker refreshes first: A → B'
  → Legitimate client tries A → REUSE DETECTED → family revoked
```

Both cases correctly detect theft and revoke.

### Scenario 3: Offline Client

```
Client goes offline, comes back with old refresh_token_A
  → A was rotated to B before going offline
  → Client presents A → REUSE DETECTED → family revoked
```

Mitigation: Grace period (below) or client retries with latest token.

## Grace Period

To handle race conditions and network issues:

```go
const GracePeriod = 30 * time.Second

func RotateRefreshToken(ctx context.Context, oldTokenID string) (*TokenPair, error) {
    oldToken := store.Get(oldTokenID)

    if oldToken.UsedAt != nil {
        // Within grace period → allow (likely race condition)
        if time.Since(*oldToken.UsedAt) < GracePeriod {
            // Return the already-issued token pair (idempotent)
            return store.GetLatestPair(oldToken.FamilyID)
        }
        // Outside grace period → theft
        revokeEntireFamily(ctx, oldToken.FamilyID)
        return nil, ErrTokenReuseDetected
    }
    // ... normal rotation
}
```

## Rotation vs Non-Rotation

| Aspect | Rotating (Recommended) | Non-Rotating |
|--------|----------------------|--------------|
| Theft detection | ✅ Detected on reuse | ❌ Undetectable |
| Token lifetime | Bounded (single use) | Long-lived |
| Client complexity | Must update stored token | Store once |
| Race conditions | Possible (mitigated by grace) | None |
| Backward compat | Needs migration | Legacy default |
| Security | High | Low |

## Backward Compatibility

```go
// Detect token type at refresh
func Refresh(ctx context.Context, tokenID string) (*TokenPair, error) {
    token := store.Get(tokenID)

    if token.FamilyID != "" {
        // New rotating token
        return RotateRefreshToken(ctx, tokenID)
    }

    // Legacy non-rotating token — still valid but no reuse detection
    if token.RotatingEnabled {
        return RotateRefreshToken(ctx, tokenID)
    }
    return legacyRefresh(ctx, tokenID)  // Issue new access, keep same refresh
}
```

## Migration Strategy

```
Phase 1 (Current): Rotation enabled for new tokens, legacy tokens still work
Phase 2 (3 months): Console warning for legacy token usage
Phase 3 (6 months): Legacy tokens forced to rotate on next refresh
Phase 4 (12 months): Legacy non-rotating tokens rejected entirely
```

## Token Storage

### Redis (Active tokens)

```
refresh:{token_id} → {family_id, user_id, scopes, used_at, expires_at}
TTL: 7 days (matches token expiry)
```

### PostgreSQL (Audit trail)

```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    family_id UUID NOT NULL,
    user_id UUID NOT NULL,
    scopes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    INDEX idx_family (family_id),
    INDEX idx_user (user_id)
);
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Reuse detection events | Any → security investigation |
| Family revocations | Any → page security |
| Rotation failures | >1% → investigate |
| Grace period hits | >5% → client concurrency issue |
| Legacy token usage | Track for migration |

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [JWT Claim Validation](jwt-claim-validation.md)
- [Session Security](session-security.md)
- [Token Exchange Patterns](token-exchange-patterns.md)
