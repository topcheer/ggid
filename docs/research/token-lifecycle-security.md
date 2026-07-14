# Token Lifecycle Security

> Focus: issuance patterns, phantom tokens, sharing prevention, revocation
> propagation, and cache invalidation for GGID's JWT-based access tokens.
> For session timeout and refresh-rotation algorithm details, see
> [session-management-design.md](./session-management-design.md).
> For DPoP/mTLS binding implementation, see
> token-binding-dpop-mtls.md.

## 1. Overview

Every access token passes through four lifecycle phases:

```
Issue ─▶ Validate ─▶ Refresh ─▶ Revoke
```

Security goals across the lifecycle:

| Goal | Why |
|---|---|
| **Short-lived** | Minimizes the window if a token is stolen |
| **Minimal claims** | JWT payload is visible to the client — no PII |
| **Bound to client** | Token cannot be used by someone else |
| **Revocable** | Admin/user can kill a token before expiry |
| **Non-shareable** | Stolen token is useless without proof-of-possession |

This document covers the patterns and infrastructure *beyond* basic JWT
signature validation: phantom tokens, rotating vs reusing refresh tokens,
token-sharing prevention, revocation propagation, and cache invalidation.

---

## 2. Token Issuance Patterns

### Short-Lived JWT (GGID Current Default)

Both `TokenService.IssueAccessToken` and `OAuthService.issueAccessToken`
sign RS256 JWTs with a 15-minute TTL. Claims: `sub`, `tenant_id`, `iss`,
`aud`, `iat`, `exp`, `jti` (UUID), with `kid` in the header.

- **Pros**: stateless validation, fast (signature check only), no DB lookup.
- **Cons**: cannot revoke before `exp`; token is visible to the client.

### Phantom Token Pattern

Instead of issuing a JWT directly, the auth service issues an opaque random
token (32 bytes). The gateway exchanges it for a JWT internally and caches
the result for the token's TTL.

```go
// Issue phantom token — client never sees the JWT
phantom := crypto.GenerateRandomToken(32)
redis.Set(ctx, "phantom:"+hash(phantom), jwtString, 15*time.Minute)

// Gateway: resolve opaque → JWT (cached after first lookup)
func (gw *Gateway) resolvePhantom(ctx context.Context, opaque string) (string, error) {
    if v, ok := gw.localCache.Get(opaque); ok { return v.(string), nil }
    jwtStr, err := gw.authClient.ExchangeToken(ctx, opaque)
    gw.localCache.Set(opaque, jwtStr, 14*time.Minute)
    return jwtStr, err
}
```

- **Pros**: JWT never leaves the server boundary. Claims never appear in
  `localStorage` or DevTools. Stolen opaque token is useless to clients.
- **Cons**: extra round-trip on first request (mitigated by gateway cache).
- **Use when**: client is a browser SPA where JWT theft from storage is a risk.

### Token Reference Pattern

The JWT remains self-contained but includes a `sid` (session ID) claim.
The gateway validates the signature *and* checks the session in Redis.

```go
func (gw *Gateway) validateWithReference(ctx context.Context, jwtStr string) error {
    claims := parseAndVerify(jwtStr)
    sid := claims["sid"].(string)
    if n, _ := gw.redis.Exists(ctx, "session:"+sid).Result(); n == 0 {
        return errors.New("session revoked")
    }
    return nil
}
```

- **Pros**: true server-side revocation while keeping JWT structure.
- **Cons**: every request hits Redis (~1 ms) — loses pure statelessness.

### Comparison

| Pattern | Client Token | Gateway Validates | Revocation | Stateful? |
|---|---|---|---|---|
| Short-Lived JWT | JWT | Signature only | No (until exp) | No |
| Phantom Token | Opaque (32B) | Exchange + cache | Yes (Redis delete) | Partial |
| Token Reference | JWT + sid | Signature + Redis | Yes (Redis delete) | Yes |

---

## 3. Refresh Token: Rotating vs Reusing

### Current GGID Implementation (Rotating with Reuse Detection)

GGID **already implements rotating refresh tokens** with reuse detection.
Both `TokenService.RotateRefreshToken` (auth) and `OAuthService.RefreshToken`
(oauth) revoke the old token and issue a new one on each refresh. On reuse
detection (presenting an already-used token), the entire token family or
session is revoked:

```go
// Auth service: revokes entire session on replay
if !oldToken.IsActive() {
    ts.refreshRepo.RevokeAllForSession(ctx, oldToken.SessionID)
    return fmt.Errorf("refresh token replay detected — session revoked")
}
```

The auth variant also links tokens via `RotatedFrom` for full chain tracking.

### Strategies Compared

| Strategy | Thief Window | DB Writes | Complexity |
|---|---|---|---|
| **Reuse** (never rotate) | 30 days (full RT lifetime) | None | Lowest |
| **Rotate every use** (GGID default) | 15 min (one AT lifetime) | 1 per refresh | Medium |
| **Hybrid** (rotate every Nth) | N × 15 min | 1 per N refreshes | Medium |
| **Risk-based** (rotate on anomaly) | Variable | On anomaly only | Highest |

GGID's "rotate every use" is the most secure option and aligns with OAuth
2.1 draft. The cost (one DB write per refresh) is negligible since refreshes
happen at most once per 15 minutes per active session.

### Recommended Enhancement: Risk-Based Rotation Trigger

```go
func (s *OAuthService) shouldForceRotate(record *domain.RefreshTokenRecord, reqIP string) bool {
    if record.LastUsedIP != "" && record.LastUsedIP != reqIP {
        return true // IP changed — potential theft
    }
    if time.Since(record.LastUsedAt) < 30*time.Second {
        return true // rapid reuse — automation
    }
    return false
}
```

---

## 4. Token Sharing Prevention

### The Problem

Bearer tokens are proof-of-possession only by transport — anyone who obtains
the token string can use it. Users share tokens via screenshots, copy-paste,
or malicious browser extensions. There is no way to distinguish the legitimate
holder from an attacker.

### Comparison of Binding Methods

| Method | Prevents Sharing | Browser Support | Implementation Effort |
|---|---|---|---|
| **None** (bearer) | No | Universal | None |
| **DPoP** (RFC 9449) | Yes (needs private key) | Web Crypto API | Medium |
| **mTLS** (RFC 8705) | Yes (needs client cert) | Limited (no SPA) | High |
| **IP binding** | Partial (fails on NAT/mobile) | Universal | Low |
| **Device fingerprint** | Partial (fingerprint drift) | Universal | Medium |

### DPoP Binding (Recommended for GGID)

The token includes a `cnf` claim with a JWK thumbprint. The client must sign
each request with the corresponding private key via a DPoP proof JWT header.

```go
// Token issuance — add cnf (confirmation) claim
claims["cnf"] = map[string]any{"jkt": jwkThumbprint(clientPublicKey)}

// Gateway: verify DPoP proof matches token's bound key
func (gw *Gateway) verifyDPoP(r *http.Request, at string) error {
    proof := r.Header.Get("DPoP")
    if proof == "" { return errors.New("DPoP proof required") }
    pc := parseAndVerifyDPoP(proof, r.Method, r.URL.String())
    tc := parseAccessTokenClaims(at)
    if tc["cnf"].(map[string]any)["jkt"] != pc["jkt"] {
        return errors.New("DPoP key mismatch")
    }
    return nil
}
```

See token-binding-dpop-mtls.md for full
implementation details.

### mTLS Binding

Best for service-to-service or enterprise. The `cnf` claim carries the TLS
cert thumbprint:

```go
claims["cnf"] = map[string]any{"x5t#S256": certThumbprint(r.TLS.PeerCertificates[0])}
```

### IP and Device Binding (Supplementary Only)

Cannot be sole enforcement due to NAT (shared IP) and fingerprint drift.
Use as risk signals only:

```go
func riskScore(r *http.Request, claims jwt.MapClaims) int {
    score := 0
    if claims["ip"] != clientIP(r) { score += 30 }
    if claims["fp_hash"] != deviceFingerprint(r) { score += 20 }
    return score // >40 → require step-up auth
}
```

---

## 5. Token Revocation Propagation

### Current Gap: In-Memory Revocation

GGID's `OAuthService` uses an in-memory `sync.Map` for the revocation list.
This means: (1) revocations lost on restart, (2) invisible across instances,
(3) gateway cannot check revocation. This is the most critical gap.

### Local Revocation (Redis — Recommended)

```go
// Revoke by jti — TTL auto-matches token expiry (self-cleaning)
func (rs *RevocationService) RevokeJTI(ctx context.Context, jti string, exp time.Time) error {
    if ttl := time.Until(exp); ttl > 0 {
        return rs.rdb.Set(ctx, "revoked:"+jti, "1", ttl).Err()
    }
    return nil
}
func (rs *RevocationService) IsRevoked(ctx context.Context, jti string) bool {
    n, _ := rs.rdb.Exists(ctx, "revoked:"+jti).Result()
    return n > 0 // < 1ms lookup
}
```

### Distributed Propagation (NATS)

For multi-region deployments with separate Redis clusters, NATS provides
cross-cluster cache invalidation (1–5s propagation). Redis remains the
source of truth; NATS drives invalidation.

```go
// Auth: publish revocation
rs.nats.Publish("token.revoked", []byte(jti))
// Gateway: subscribe and invalidate local caches
gw.nats.Subscribe("token.revoked", func(m *nats.Msg) {
    gw.tokenCache.Delete(string(m.Data))
})
```

### Cross-Domain (CAEP)

For federated scenarios where GGID is the IdP, emit a CAEP session-revoked
event so external RPs revoke their sessions. See
[caep-analysis.md](./caep-analysis.md).

### Revocation Cascade

On user logout or admin disable:

```go
func (rs *RevocationService) CascadeRevoke(ctx context.Context, userID uuid.UUID) error {
    rs.RevokeJTI(ctx, currentJTI, exp)                        // 1. Access token
    rs.refreshRepo.RevokeAllForUser(ctx, tenantID, userID)     // 2. Refresh token
    rs.rdb.Del(ctx, "session:"+sessionID)                     // 3. Session
    rs.nats.Publish("token.revoked", []byte(currentJTI))       // 4. NATS (gateways)
    rs.caepEmitter.SessionRevoked(userID)                      // 5. CAEP (external RPs)
    rs.grantRepo.RevokeAllForUser(ctx, tenantID, userID)       // 6. OAuth grants
    return nil
}
```

---

## 6. Cache Invalidation Strategies

### JWKS Cache

On key rotation, a new `kid` appears in the JWT header. Cache miss triggers
force-refresh. Background refresh every 5 min closes the window where
new-kid tokens fail.

```go
func (gw *Gateway) getKey(kid string) (*rsa.PublicKey, error) {
    if key, ok := gw.jwksCache.Get(kid); ok { return key, nil }
    jwks, err := gw.fetchJWKS() // force refresh on unknown kid
    gw.jwksCache.SetAll(jwks, 15*time.Minute)
    return jwks[kid], nil
}
```

### Token Validation Cache

If the gateway caches "JWT is valid" to skip RSA verification, revocation
must invalidate the entry. NATS `token.revoked` event → `cache.Delete(jti)`.

### Session Cache

Short TTL (60s) session cache limits stale data. On NATS revocation event,
delete the session entry immediately.

---

## 7. Token Security Checklist

| # | Measure | Status in GGID |
|---|---|---|
| 1 | Access tokens < 15 min TTL | Done (hardcoded 15m in oauth; configurable in auth) |
| 2 | Refresh tokens rotated on use | Done (both services) |
| 3 | Refresh reuse detection → revoke family | Done (session-wide revocation) |
| 4 | JWT contains minimal claims (no PII) | Done (sub, tenant_id, jti only) |
| 5 | `kid` in JWT header (key rotation) | Done |
| 6 | `jti` unique per token (for revocation) | Done (`uuid.New()`) |
| 7 | `aud` claim set | Done (per-client) |
| 8 | `iss` claim validated | Done |
| 9 | Algorithm enforcement (reject `none`) | Done (RSA-only in `ParseAccessToken`) |
| 10 | Clock skew tolerance (60s) | **Gap** — no explicit skew config |
| 11 | Revocation: Redis SET for jtis | **Gap** — in-memory `sync.Map` |
| 12 | Revocation survives restart | **Gap** — sync.Map lost on restart |
| 13 | Distributed revocation (NATS) | **Gap** — no NATS propagation |
| 14 | DPoP/mTLS binding | **Gap** — bearer only |
| 15 | No tokens in URL | OK (Authorization header) |
| 16 | No tokens in localStorage | **Gap** — console uses localStorage |

---

## 8. GGID Current State Audit

| Security Measure | Implemented? | Gap | Priority |
|---|---|---|---|
| Short-lived access tokens (15m) | Yes | OAuth service hardcodes 15m (not configurable) | P2 |
| Refresh rotation + reuse detection | Yes | None — fully implemented | — |
| Minimal claims (no PII) | Yes | None | — |
| `kid` header for key rotation | Yes | Single key only (no active rotation pipeline) | P1 |
| `jti` for per-token revocation | Yes | Revocation list is in-memory only | P0 |
| `aud`/`iss` validation | Yes | None | — |
| Algorithm enforcement | Yes | RSA-only check; consider explicit RS256/ES256 allowlist | P2 |
| Revocation mechanism | **Partial** | `sync.Map` — lost on restart, not shared | **P0** |
| Token binding (DPoP/mTLS) | No | Bearer-only | P2 |
| Client storage recommendations | No | Console uses localStorage (XSS-vulnerable) | P1 |

### Critical Gaps

1. **Revocation is in-memory** (`sync.Map`). On restart, all revocations are
   lost. Multiple gateway instances cannot share revocation state. This is
   the single highest-priority fix.

2. **Console stores JWT in localStorage**. If an XSS attack executes in the
   console context, it can exfiltrate the token. Switch to httpOnly cookies
   or implement the phantom token pattern.

3. **No token binding**. Any stolen token is immediately usable. For
   high-security tenants, DPoP binding should be available as an opt-in.

---

## 9. Roadmap

| Phase | Task | Priority | Effort |
|---|---|---|---|
| 1 | **Redis-based jti revocation** — replace `sync.Map` with Redis SET, TTL = token exp | P0 | 2 days |
| 2 | **Revocation cascade** — implement full 6-step cascade on logout/disable | P0 | 1 day |
| 3 | **NATS propagation** — publish revocation events for distributed gateways | P1 | 2 days |
| 4 | **Phantom token pattern** — opaque token to client, JWT stays server-side | P1 | 5 days |
| 5 | **Console storage fix** — httpOnly cookies or in-memory + silent refresh | P1 | 2 days |
| 6 | **JWKS rotation pipeline** — multi-key support, background refresh | P1 | 3 days |
| 7 | **Clock skew config** — configurable tolerance (default 60s) | P2 | 0.5 day |
| 8 | **DPoP token binding** — `cnf` claim + proof JWT validation | P2 | 5 days |
| 9 | **Risk-based rotation trigger** — IP change / rapid reuse detection | P2 | 2 days |

**Total estimated effort**: ~22.5 engineering days.

**Quick wins** (Phase 1–2): Move revocation to Redis and implement the
cascade. This closes the most dangerous gap with minimal effort and no
client-side changes.
