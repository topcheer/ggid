# Refresh Token Security: Rotation, Detection & Binding

> **Scope**: Comprehensive refresh token security design — rotation strategies, family
> detection, cryptographic binding, storage patterns, and concurrency. This document
> does NOT repeat session timeout or access token design (see
> `session-management-design.md` and `token-lifecycle-security.md`).

---

## 1. Overview

Refresh tokens are long-lived credentials that allow clients to obtain new access
tokens without re-authenticating the user. Because they outlive access tokens by
orders of magnitude (days/weeks vs minutes), a stolen refresh token grants
persistent access until expiry or revocation.

**Threat model**: refresh tokens are prime targets for:
- Network interception (insecure channels, XSS exfiltration)
- Database compromise (plaintext token storage)
- Process memory dumps (bearer tokens have no proof-of-possession)
- Token replay (attacker reuses a stolen token in parallel with the legitimate user)

**OAuth 2.0 Security BCP (RFC 9700)** mandates two controls for refresh tokens:
1. **Rotation** — issue a new refresh token on every use, invalidate the old one.
2. **Reuse detection** — if a used/revoked token is presented again, revoke the
   entire token family.

This document covers: rotation strategies (Section 2), family detection (Section 3),
cryptographic binding — DPoP, mTLS, device-bound (Section 4), offline_access scope
(Section 5), storage patterns (Section 6), a full GGID code audit (Section 7),
race condition handling (Section 8), and a phased roadmap (Section 9).

---

## 2. Rotation Strategies

### Strategy A: Non-Rotating (Fixed Refresh Token)

Same refresh token used for its entire lifetime (e.g., 30 days). One row per
user+client, no state transitions.

- **Security**: Low. Stolen token = persistent access until expiry.
- **When appropriate**: Low-risk internal services, read-only APIs, short TTLs (< 1h).
- **GGID status**: NOT used. GGID implements Strategy B.

### Strategy B: Rotating (New Token Each Use)

Each refresh issues a new access token AND a new refresh token. The old token is
immediately invalidated.

- **Security**: High. Stolen token has a narrow window — if the legitimate user
  refreshes first, the stolen token becomes invalid.
- **Window of opportunity**: One access-token TTL (~15 min) if user is active.
- **RFC 9700**: Recommended.
- **GGID status**: IMPLEMENTED (`oauth_service.go` lines 730–752).

### Strategy C: Detection-Based (Risk-Triggered) Rotation

Rotate only when risk indicators change — new IP, new device fingerprint, anomalous
behaviour, or every Nth refresh (e.g., every 10th). Reduces DB writes at scale.

- **Security**: Medium-High. Fewer rotations = larger exposure window.
- **Complexity**: Highest — requires risk-scoring pipeline, device tracking.
- **Best for**: High-traffic platforms where per-refresh rotation is cost-prohibitive.

### Comparison

| Strategy      | Security | DB Writes | Complexity | RFC 9700? |
|---------------|----------|-----------|------------|-----------|
| A: Non-rotating | Low    | 1 (store) | Low        | No        |
| B: Rotating     | High   | 2/read    | Medium     | Yes       |
| C: Risk-triggered | Med-High | Variable | High     | Partial   |

### GGID Go Implementation (Current — Strategy B)

The `RefreshToken()` method (lines 690–761) performs: SHA-256 lookup → reuse check
(`Used || Revoked` triggers `RevokeAllRefreshTokens`) → expiry check → revoke old →
issue new 256-bit token with 30-day TTL. Full code analysis in Section 7.

---

## 3. Family Detection

### Token Family

A **family** is the set of all refresh tokens descended from a single initial
authentication event. Each rotation produces a new token in the same family. The
`family_id` is shared across the entire chain; each token has its own `token_hash`
and optionally a `parent_hash` linking it to its predecessor.

```
Login → [token_A, family=F1, parent=null]
  Refresh → [token_B, family=F1, parent=token_A]  (A marked used)
    Refresh → [token_C, family=F1, parent=token_B]  (B marked used)
```

### Reuse Detection Flow

1. **Legitimate refresh**: User presents `token_A` (active) → server issues
   `token_B`, marks `token_A` as `used`.
2. **Attacker replays**: Attacker presents `token_A` → server sees `token_A` is
   already `used` → **REUSE DETECTED**.
3. **Response**: Revoke the ENTIRE family — `token_B` is invalidated too.
4. Both parties are kicked out. The user must re-authenticate.
5. **Rationale**: if reuse is detected, one party holds a stolen token. The only
   safe action is to invalidate everything and force re-authentication.

### GGID Gap: Family ID vs Client ID

GGID's current reuse detection calls `RevokeAllRefreshTokens(ctx, tenantID, clientID)`
which revokes **all tokens for the entire client**, not just the family. If multiple
sessions/devices use the same OAuth client, one stolen token kicks out all of them.
Adding a `family_id` column would scope revocation to only the compromised chain.

### Recommended Data Model

```sql
CREATE TABLE oidc_refresh_tokens (
    token_hash   VARCHAR(64) PRIMARY KEY,  -- SHA-256 of plaintext token
    family_id    UUID NOT NULL,             -- groups tokens from one login
    tenant_id    UUID NOT NULL,
    user_id      UUID NOT NULL,
    client_id    UUID NOT NULL,
    scope        TEXT[],
    status       VARCHAR(20) DEFAULT 'active', -- active, used, revoked
    parent_hash  VARCHAR(64),               -- links to predecessor
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    used_at      TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_rt_family  ON oidc_refresh_tokens(family_id);
CREATE INDEX idx_rt_user    ON oidc_refresh_tokens(user_id, tenant_id);
```

### Family Revocation Query (PostgreSQL)

```sql
UPDATE oidc_refresh_tokens
   SET status = 'revoked', revoked = true
 WHERE tenant_id = $1 AND family_id = $2;
```

---

## 4. Token Binding

Binding a refresh token to cryptographic proof-of-possession prevents an attacker
from using a stolen token without the corresponding key material.

### DPoP-Bound Refresh Tokens (RFC 9449)

- Token stores `cnf.jkt` (thumbprint of the client's DPoP public key).
- Each refresh request must include a fresh DPoP proof JWT signed by the private key.
- Stolen refresh token is useless without the DPoP private key.
- Works in browsers (Web Crypto), mobile, and backend services.
- See `dpop-rfc9449.md` for full DPoP implementation details.

### mTLS-Bound Refresh Tokens (RFC 8705)

- Token stores `cnf.x5t#S256` (thumbprint of the client's TLS certificate).
- Refresh request must arrive over mutual TLS with the same certificate.
- Strongest binding: hardware-backed certs (TPM/Secure Enclave) can't be copied.
- Poor browser UX (certificate prompts); ideal for service-to-service and enterprise.

### Device-Bound Refresh Tokens

- Token stores a `device_id` (fingerprint or registered device identifier).
- Gateway validates `device_id` header matches the stored value before forwarding.
- Less formal than DPoP/mTLS but adds friction for token replay across devices.
- Implementation: add `device_id` column to `oidc_refresh_tokens`, validate on refresh.

### Comparison

| Binding     | Key Material   | Browser | Mobile | S2S   | Theft Resistance |
|-------------|----------------|---------|--------|-------|------------------|
| None (bearer) | —           | Yes     | Yes    | Yes   | Low              |
| DPoP        | EC key pair   | Yes     | Yes    | Limited | High          |
| mTLS        | X.509 cert    | Poor UX | Yes    | Yes   | Highest          |
| Device      | Fingerprint   | Partial | Yes    | No    | Medium           |

---

## 5. offline_access Scope

OIDC defines `offline_access` as the standard scope for requesting refresh tokens.
Per OIDC Core (Section 11):

- Client **MUST** request `offline_access` to receive a refresh token.
- User **MUST** give explicit consent (cannot be silent/skipped).
- If `offline_access` is not granted, no refresh token is issued.

### GGID Status

GGID lists `offline_access` in `ScopesSupported` (discovery metadata) and in the
basic scope whitelist (`server.go` line 235), but **does not gate refresh token
issuance on it**. `ExchangeAuthorizationCode()` issues tokens without checking
whether `offline_access` was requested or consented. Currently, no refresh token
is issued during code exchange at all — the initial refresh token must come from
another grant flow.

**Recommendation**: add an `offline_access` check in `ExchangeAuthorizationCode`
and any grant that issues refresh tokens:

```go
if contains(code.Scope, "offline_access") {
    refreshToken, _ := crypto.GenerateRandomToken(32)
    s.tokenRepo.StoreRefreshToken(ctx, &domain.RefreshTokenRecord{
        TokenHash: hashTokenSHA256(refreshToken),
        Scope:     code.Scope,
        ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
    })
    resp.RefreshToken = refreshToken
}
```

---

## 6. Storage Patterns

### GGID Current: PostgreSQL

Tokens are stored in the `oidc_refresh_tokens` table with columns:
`id, tenant_id, user_id, client_id, token_hash, scope, expires_at, revoked, used, created_at`.
Token values are **never stored in plaintext** — only the SHA-256 hash. This is
correct and important: even a full DB dump cannot recover usable tokens.

### Redis Hash Storage (Recommended for Hot Path)

```
Key:    rt:{token_hash}        TTL: expires_at - now()
Fields: user_id, family_id, client_id, status, device_id, scope, expires_at
```

- **O(1)** lookup via `HGETALL`
- **Atomic** rotation via Lua script (see Section 8)
- **Automatic cleanup**: Redis TTL evicts expired keys without a sweeper job
- Use PostgreSQL as durable audit trail (async write-behind)

### Token Hashing

- **Always** store `SHA-256(token)` — never the plaintext token.
- GGID already does this via `hashTokenSHA256()` (`oauth_service.go` line 670).
- Lookup: `WHERE token_hash = SHA-256(submitted_token)`.
- The plaintext token is only ever in transit (HTTPS) and in the client's memory.

### Storage Audit

| Aspect               | GGID Current        | Recommended               | Gap? |
|----------------------|---------------------|---------------------------|------|
| Hashing              | SHA-256             | SHA-256                   | No   |
| Backend              | PostgreSQL          | Redis hot-path + PG audit | Yes  |
| Atomicity            | Read-then-write     | Lua/SELECT FOR UPDATE     | Yes  |
| Auto-cleanup         | Manual (no sweeper) | Redis TTL                 | Yes  |
| family_id column     | Missing             | Required for family scope | Yes  |
| device_id column     | Missing             | Required for device bind  | Yes  |

---

## 7. GGID Refresh Token Audit

### Feature Matrix

| Feature                      | GGID Status | RFC 9700 Compliant? | Priority |
|------------------------------|-------------|---------------------|----------|
| Rotation (new token/use)     | YES         | Yes                 | Done     |
| Reuse detection              | Partial     | Partial (revokes client-wide, not family) | P0 |
| Family tracking (family_id)  | NO          | No (missing column) | P0       |
| Hashed storage (SHA-256)     | YES         | Yes                 | Done     |
| Token binding (DPoP/mTLS)    | NO          | Recommended         | P2       |
| Device binding               | NO          | Recommended         | P2       |
| offline_access enforcement   | NO          | No (not gated)      | P1       |
| TTL / expiry                 | 30 days     | Yes                 | Done     |
| Concurrent refresh handling  | NO          | No (race condition) | P1       |
| Initial refresh in code flow | NO          | Gap (not issued)    | P0       |

### Code Analysis

**`RefreshToken()` (oauth_service.go:690–761)** — Strengths:
- SHA-256 hashed lookup — tokens never stored plaintext.
- Rotation: old token revoked, new 256-bit token issued on every use.
- Reuse detection: `record.Used || record.Revoked` triggers `RevokeAllRefreshTokens`.
- Expiry check before issuance.

**Weaknesses**:
1. **No family_id** — reuse revokes ALL tokens for the client, affecting
   legitimate sessions on other devices.
2. **Not atomic** — `GetRefreshToken()` + `RevokeRefreshToken()` are separate
   queries. Two concurrent requests with the same token both pass the "not used"
   check before either marks it used → race condition (see Section 8).
3. **No offline_access gating** — refresh tokens can be issued to clients that
   never requested `offline_access`.
4. **No initial refresh token** — `ExchangeAuthorizationCode()` (lines 290–352)
   never issues a refresh token, only access + ID tokens. The first refresh token
   must originate from a different grant or is simply missing.
5. **Error suppression** — `_ = s.tokenRepo.RevokeRefreshToken(...)` silently
   discards revocation errors. A failed revocation leaves the token active.

---

## 8. Race Condition Handling

### The Problem

Two concurrent requests carrying the same refresh token:

```
Request 1: GetRefreshToken(token_A) → active ✓
Request 2: GetRefreshToken(token_A) → active ✓   (same row, not yet updated)
Request 1: RevokeRefreshToken(token_A)            (marks used)
Request 2: RevokeRefreshToken(token_A)            (already used — no-op)
Request 1: StoreRefreshToken(token_B)
Request 2: StoreRefreshToken(token_C)
Result: TWO new tokens issued from one refresh — rotation invariant broken.
```

### Solution A: PostgreSQL SELECT FOR UPDATE

```go
// Start transaction
tx, _ := r.pool.Begin(ctx)
defer tx.Rollback(ctx)

var status string
err := tx.QueryRow(ctx,
    `SELECT status FROM oidc_refresh_tokens
      WHERE token_hash = $1 FOR UPDATE`, tokenHash).Scan(&status)

if status != "active" {
    return errors.New("reuse detected")
}

tx.Exec(ctx,
    `UPDATE oidc_refresh_tokens SET status = 'used', used = true
      WHERE token_hash = $1`, tokenHash)
tx.Commit(ctx)
```

Row-level lock ensures only one request sees `active`; the second blocks until
the first commits, then sees `used`.

### Solution B: Redis Lua Script (Atomic)

```lua
local current = redis.call("HGET", KEYS[1], "status")
if current == "used" or current == "revoked" then
    return "REUSE"
end
redis.call("HSET", KEYS[1], "status", "used")
return "OK"
```

The entire check-and-set runs as a single atomic operation. No lock contention,
no window for concurrent use.

---

## 9. Roadmap

| Phase | Task                                     | Priority | Effort  |
|-------|------------------------------------------|----------|---------|
| 1     | Issue initial refresh token in code flow | P0       | ~1 day  |
| 2     | Add `family_id` column + family-scoped revocation | P0 | ~2 days |
| 3     | Fix race condition (SELECT FOR UPDATE)   | P0       | ~1 day  |
| 4     | offline_access scope enforcement         | P1       | ~0.5 day|
| 5     | Redis hot-path storage (Lua rotation)    | P1       | ~2 days |
| 6     | DPoP binding for refresh tokens          | P2       | ~3 days |
| 7     | Device-bound refresh tokens              | P2       | ~2 days |

**Total estimated effort**: ~11–12 engineering days for full RFC 9700 compliance.

### Quick Wins (P0, ~4 days)

1. Add `family_id UUID NOT NULL` + `parent_hash VARCHAR(64)` to
   `oidc_refresh_tokens`. Backfill existing rows with per-token families.
2. Change reuse response from `RevokeAllRefreshTokens(clientID)` to
   `RevokeFamily(familyID)`.
3. Wrap the read-mark-write sequence in a PostgreSQL transaction with
   `SELECT ... FOR UPDATE`.
4. Issue the initial refresh token in `ExchangeAuthorizationCode` (gated on
   `offline_access`).

### Medium-Term (P1, ~2.5 days)

5. Gate all refresh token issuance behind `offline_access` scope + consent check.
6. Add Redis as hot-path cache for token lookups, with Lua atomic rotation.

### Long-Term (P2, ~5 days)

7. Integrate DPoP proof validation on the refresh endpoint (validate `cnf.jkt`).
8. Add `device_id` column and gateway-level device fingerprint validation.

---

*Last updated: GGID OAuth service audit. Refer to `token-lifecycle-security.md` for
access token design and `session-management-design.md` for session timeout policies.*
