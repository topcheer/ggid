# JWKS Key Rotation: Design and Implementation Guide

> Research document for GGID's JWT signing key lifecycle management.
> Covers dual-key strategy, zero-downtime rotation, and analysis of the current codebase.

---

## 1. Overview

A **JSON Web Key Set (JWKS)** is a standardized endpoint (`application/json`)
that publishes public cryptographic keys used to verify JWT signatures.
Clients fetch the JWKS, extract the key matching the `kid` header in a JWT,
and verify the signature cryptographically — without sharing any secrets.

**Key rotation** is the practice of periodically replacing the signing key
pair used to mint new JWTs. The old key is retired (kept in the JWKS for a
limited overlap window) then eventually removed.

### Why Rotate?

| Driver | Detail |
|---|---|
| **Security** | Limits blast radius: a compromised key only forges tokens until rotation. |
| **Compliance** | PCI DSS requires annual key rotation (3.5.2); SOC 2 CC6.1; FedRAMP AC-2. |
| **Incident response** | Emergency rotation on suspected key compromise. |
| **Forward secrecy** | Shortens the validity period of any single cryptographic credential. |

### Recommended Frequency

- **Routine:** every 90 days (compliance-driven).
- **Emergency:** immediately on compromise — no overlap window.
- **Industry examples:** Auth0 rotates monthly; Google rotates some keys hourly.

---

## 2. Dual Key Strategy

### Active + Retired Keys

At any point in time the JWKS serves one or more public keys:

- **One ACTIVE key** — signs all newly minted tokens.
- **Zero or more RETIRED keys** — no longer signs, but stays in the JWKS so
  outstanding tokens can still be verified.
- **Zero or one NEXT key** — pre-generated, waiting in the JWKS so verifiers
  cache it before it becomes active.

### Overlap Window

The retired key must remain in the JWKS until **every token** signed with it
has expired. The overlap duration equals the longest token TTL:

```
Access token TTL:  15 min  → overlap window: 15 min after retirement
Refresh token TTL: 7 days  → overlap window: 7 days after retirement
```

### Key States

```
                    generate
                       │
                       ▼
                   ┌───────┐   promote    ┌────────┐   expire overlap  ┌─────────┐
                   │  next │ ──────────►  │ active │ ────────────────► │ retired │
                   └───────┘              └────────┘                   └─────────┘
                                              │                            │
                                              │ (emergency revoke)         │ remove from JWKS
                                              ▼                            ▼
                                          ┌─────────┐                  ┌─────────┐
                                          │ revoked │◄─────────────────│ revoked │
                                          └─────────┘                  └─────────┘
```

| State | Signs new tokens? | In JWKS? | Verifies old tokens? |
|---|---|---|---|
| `next` | No | Yes (pre-published) | Yes |
| `active` | Yes | Yes | Yes |
| `retired` | No | Yes (overlap) | Yes |
| `revoked` | No | No | No — all matching tokens fail |

### Rotation Timeline

1. Generate new key pair → status: `next`, publish to JWKS.
2. Wait one cache-TTL cycle so all verifiers have cached the new public key.
3. Promote `next` → `active`; demote old `active` → `retired`.
4. Wait overlap window (max token TTL).
5. Remove retired key from JWKS → status: `revoked`.

---

## 3. kid Propagation

The **kid** (Key ID) field in the JWT header identifies which key signed the
token. The JWKS endpoint serves multiple keys, each with its own `kid`. The
verifier:

1. Extracts `kid` from the JWT header.
2. Looks up the matching public key in the JWKS cache.
3. Verifies the signature with that key.

### GGID's kid Generation

GGID computes the kid as a truncated SHA-256 fingerprint of the DER-encoded
public key:

```go
// services/oauth/internal/server/server.go
func computeKID(pub *rsa.PublicKey) string {
    data, _ := x509.MarshalPKIXPublicKey(pub)
    h := sha256.Sum256(data)
    return fmt.Sprintf("%x", h[:8])  // 8 bytes = 16 hex chars
}
```

The gateway uses an identical algorithm (`keyFingerprint`). This ensures both
signer and verifier agree on the kid without out-of-band coordination.

### Cache TTL Strategy

The gateway's `JWKSClient` fetches and caches keys in memory with a periodic
background refresh:

```go
// services/gateway/internal/middleware/middleware.go
func (c *JWKSClient) StartRefresh(ctx context.Context, interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        for {
            select {
            case <-ctx.Done(): return
            case <-ticker.C: c.refreshJWKS()
            }
        }
    }()
}
```

**Problem:** If a new key is activated but the gateway hasn't refreshed yet,
tokens with the new `kid` will fail with "key not found."

**Solution — cache-miss fallback:** When `GetKey(kid)` fails, force an
immediate JWKS refresh before rejecting:

```go
func (c *JWKSClient) GetKeyWithFallback(kid string) (*rsa.PublicKey, error) {
    if key, ok := c.getKeyFast(kid); ok {
        return key, nil
    }
    // Cache miss — force refresh
    c.refreshJWKS()
    return c.getKeyFast(kid) // retry after refresh
}
```

Recommended cache TTL: **5–15 minutes**. Combined with cache-miss refresh,
this ensures at most 1 failed request per rotation event.

---

## 4. Zero-Downtime Rotation Procedure

### Pre-Rotation (T-5min to T-0)

```
┌──────────────┐         ┌──────────┐         ┌──────────────┐
│  KeyManager  │         │  JWKS EP │         │   Gateway    │
└──────┬───────┘         └────┬─────┘         └──────┬───────┘
       │ Generate key B       │                      │
       │ kid = fingerprint(B) │                      │
       ├─────────────────────►│ Publish {A, B}      │
       │                      │                      │
       │                      │  ←— periodic refresh │
       │                      │                      │ keys = {A, B}
       │                      │                      │
       │    Wait 1 cache TTL (5–15 min)             │
       │    All verifiers now have both keys cached  │
```

1. Generate new RSA 2048 key pair (key B).
2. Compute `kid`: `computeKID(B.PublicKey)`.
3. Add key B to JWKS endpoint → now serving `{A, B}`.
4. Wait for all gateway instances to refresh cache (1 TTL cycle).

### Active Rotation (T-0)

5. Switch active signer: key B becomes `active`.
6. Old key A becomes `retired` (remains in JWKS).
7. New tokens carry `kid: <B>` in header.

### Post-Rotation (T-overlap)

8. Wait overlap window (max token TTL — e.g., 15 min access, 7 day refresh).
9. Remove key A from JWKS → status `revoked`.
10. Archive old private key (encrypted, for forensic auditing).
11. Audit log: `key_rotated { old_kid, new_kid, timestamp }`.

### Emergency Rotation (Compromise)

```
T-0:  Suspected key compromise detected
  ├── 1. Immediately generate new key C
  ├── 2. Add C to JWKS → serving {A, C}
  ├── 3. Switch active signer to C
  ├── 4. Remove compromised key A from JWKS IMMEDIATELY (no overlap)
  ├── 5. All outstanding tokens with kid=A become invalid
  └── 6. Users forced to re-authenticate
```

**Critical:** Emergency rotation does NOT wait for cache propagation or
overlap. Some requests will fail until clients re-authenticate — this is
acceptable and desirable after compromise.

---

## 5. GGID Implementation Analysis

### Current State

The codebase already has substantial JWKS infrastructure:

**OAuth Service (signer):**
- `domain.KeyProvider` interface: `PublicKey()`, `PrivateKey()`, `KeyID()`.
- `loadOrCreateKeyProvider()` loads RSA keys from PEM files on disk
  (`configs/rsa_private.pem`, `configs/rsa_public.pem`).
- Auto-generates 2048-bit RSA key if files missing.
- `GetJWKS()` returns a single-key JWKS response.
- All token signing methods set `token.Header["kid"] = keyProvider.KeyID()`.
- `computeKID()`: `SHA256(DER_PublicKey)[:8]` → hex string.
- JWKS endpoint served at `/oauth/jwks`.
- Discovery doc includes `jwks_uri` field.

**Gateway (verifier):**
- `JWKSClient` struct with in-memory `map[kid]*rsa.PublicKey`.
- `NewJWKSClient(jwksURL, publicKeyPath)` — fetches from URL or loads PEM.
- `StartRefresh()` — background goroutine with configurable interval.
- `refreshJWKS()` — HTTP GET, parses keys, replaces entire key map.
- `GetKey(kid)` — RLock lookup, falls back to static `publicKey`.
- `JWTAuth()` middleware — extracts `kid` from header, calls `GetKey()`.
- `JWKSHandler()` — serves JWKS at `/.well-known/jwks.json`.
- Also serves `keyFingerprint()` matching OAuth's `computeKID()`.

### Gap Analysis

| Feature | Current State | Gap |
|---|---|---|
| Key storage | PEM files on disk | No DB-backed key store; no encryption at rest |
| JWKS endpoint | `/oauth/jwks` (OAuth service) | Serves only 1 key (no multi-key support) |
| kid in JWT header | Present (RS256) | Working correctly |
| Gateway JWKS cache | In-memory, periodic refresh | No cache-miss fallback refresh |
| kid computation | SHA256(DER)[:8] hex | Consistent between signer and verifier |
| Rotation automation | None | Manual: replace PEM files + restart services |
| Overlap window | Not implemented | Single key only — no retired-key retention |
| Key states | N/A (single static key) | No active/retired/revoked lifecycle |
| Multi-tenant keys | Shared single key | No per-tenant isolation |
| Emergency rotation | Not implemented | No admin API to trigger rotation |

---

## 6. Implementation Design

### Data Model

```sql
CREATE TABLE signing_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    kid         VARCHAR(64) NOT NULL UNIQUE,
    algorithm   VARCHAR(16) NOT NULL DEFAULT 'RS256',
    public_jwk  JSONB NOT NULL,           -- JWK (public only)
    private_key BYTEA NOT NULL,            -- AES-256-GCM encrypted
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
                -- active | retired | revoked | next
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at  TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ                -- when safe to remove from JWKS
);
```

### KeyManager

```go
type KeyManager struct {
    db       *pgxpool.Pool
    encKey   []byte // master encryption key for private keys
}

func (km *KeyManager) GetActiveKey(tenantID uuid.UUID) (*SigningKey, error)
func (km *KeyManager) GetKey(kid string) (*SigningKey, error)
func (km *KeyManager) GenerateKey(tenantID uuid.UUID) (*SigningKey, error)
func (km *KeyManager) RotateKey(tenantID uuid.UUID) error {
    // 1. Generate new key → status: next
    next, _ := km.GenerateKey(tenantID)

    // 2. Publish to JWKS (automatic via DB-backed provider)

    // 3. Wait 1 cache TTL for propagation (done by scheduler)

    // 4. Promote next → active, demote old → retired
    km.promote(next.ID, tenantID)

    // 5. Schedule removal after overlap window
    next.ExpiresAt = time.Now().Add(maxTokenTTL)
}
```

### DB-Backed JWKS Provider

Replace the static `GetJWKS()` with a DB-backed version:

```go
func (s *OAuthService) GetJWKS() *domain.JWKSResponse {
    // Fetch all active + retired keys (not revoked)
    keys, _ := s.keyRepo.GetKeysForJWKS(tenantID)
    var jwksKeys []domain.JWKSKey
    for _, k := range keys {
        jwksKeys = append(jwksKeys, k.ToJWK())
    }
    return &domain.JWKSResponse{Keys: jwksKeys}
}
```

### Gateway: Cache-Miss Fallback

```go
func (c *JWKSClient) GetKey(keyID string) (*rsa.PublicKey, error) {
    c.mu.RLock()
    if key, ok := c.keys[keyID]; ok {
        c.mu.RUnlock()
        return key, nil
    }
    c.mu.RUnlock()

    // Cache miss — force refresh before rejecting
    if c.jwksURL != "" {
        _ = c.refreshJWKS()
        c.mu.RLock()
        defer c.mu.RUnlock()
        if key, ok := c.keys[keyID]; ok {
            return key, nil
        }
    }
    // Final fallback to static key
    if c.publicKey != nil {
        return c.publicKey, nil
    }
    return nil, fmt.Errorf("key not found for kid: %s", keyID)
}
```

### Scheduler (Cron-Based Rotation)

```go
func (s *Scheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(24 * time.Hour) // daily check
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            tenants, _ := s.tenantRepo.ListAll()
            for _, t := range tenants {
                active, _ := s.keyMgr.GetActiveKey(t.ID)
                if time.Since(active.CreatedAt) > 90*24*time.Hour {
                    s.keyMgr.RotateKey(t.ID)
                }
            }
        }
    }
}
```

### Multi-Tenant Considerations

- **Per-tenant keys:** Each tenant gets its own RSA key pair.
  JWKS endpoint: `/oauth/{tenant_id}/jwks`. Full cryptographic isolation.
- **Shared keys with tenant claim:** Simpler — one key pair, `tenant_id` in
  JWT claims provides logical isolation. Lower key management overhead.
- **Recommendation:** Start with shared keys (Phase 1–4), add per-tenant
  keys in Phase 5 when enterprise tenants require cryptographic separation.

---

## 7. Comparison with Other Systems

| System | Rotation Frequency | Key Scope | Automation |
|---|---|---|---|
| **Auth0** | ~30 days (automatic) | Per-tenant | Automatic; manual trigger available |
| **Keycloak** | Manual via admin console | Per-realm | Manual; rotation API available |
| **AWS Cognito** | Automatic (opaque schedule) | Per user pool | Fully managed |
| **Google Identity** | Hourly for some services | Per-service account | Fully automated via discovery |
| **Okta** | On-demand via admin API | Per-authorization server | Manual or API-triggered |
| **GGID (current)** | None (static key) | Global single key | Manual PEM file swap + restart |

**Key insight:** All major IAM platforms support multi-key JWKS with
automatic rotation. GGID's single-key static model is a known gap for
production readiness.

---

## 8. Roadmap

| Phase | Description | Effort |
|---|---|---|
| **1** | Multi-key JWKS: `GetJWKS()` serves all `active + retired` keys from DB | 2–3 days |
| **2** | Dual-key lifecycle: `next → active → retired → revoked` state machine with overlap window | 3–4 days |
| **3** | Automated rotation: cron scheduler checks key age, rotates every 90 days | 2–3 days |
| **4** | Emergency rotation API: admin endpoint to trigger immediate key replacement | 1–2 days |
| **5** | Per-tenant signing keys with tenant-scoped JWKS endpoints | 3–5 days |

**Total estimated effort:** Phase 1–4 ≈ 1.5 weeks; Phase 5 ≈ 1 week.

### Priority Order

1. **Phase 1** (multi-key JWKS) — unblocks all subsequent phases.
2. **Phase 4** (emergency rotation) — highest security value, lowest effort.
3. **Phase 2** (dual-key lifecycle) — enables safe routine rotation.
4. **Phase 3** (automated scheduler) — removes manual toil.
5. **Phase 5** (multi-tenant) — enterprise feature, defer until needed.

### Acceptance Criteria

- [ ] JWKS endpoint returns multiple keys when rotation is in progress.
- [ ] Gateway verifies tokens signed with both old and new keys during overlap.
- [ ] Cache-miss fallback refresh implemented in `GetKey()`.
- [ ] Admin API can trigger emergency rotation.
- [ ] Audit log records all rotation events with timestamps and key IDs.
- [ ] No token validation failures during planned rotation (zero-downtime).

---

## References

- [RFC 7517 — JSON Web Key (JWK)](https://datatracker.ietf.org/doc/html/rfc7517)
- [RFC 7519 — JSON Web Token (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [OIDC Discovery — `jwks_uri`](https://openid.net/specs/openid-connect-discovery-1_0.html)
- [Auth0 — Rotate Signing Keys](https://auth0.com/docs/get-started/tenant-settings/signing-keys/rotate-signing-keys)
- [AWS Cognito — Verifying JWTs](https://docs.aws.amazon.com/cognito/latest/developerguide/amazon-cognito-user-pools-using-tokens-verifying-a-jwt.html)
- PCI DSS v4.0 §3.5.2 (cryptographic key rotation)
