# Automated Key Rotation & Certificate Lifecycle Management for GGID

> **Focus**: Comprehensive key and certificate rotation strategy — JWT signing keys, OAuth client secrets, device certs (SCEP), TLS certs (cert-manager), API keys, CMK/DEK rotation, and webhook signing secrets — with zero-downtime dual-key pattern.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: DoD per backlog item (§9).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Key Infrastructure](#2-ggid-current-state-key-infrastructure)
3. [Gap Analysis](#3-gap-analysis)
4. [Dual-Key Rotation Pattern](#4-dual-key-rotation-pattern)
5. [JWT Signing Key Rotation](#5-jwt-signing-key-rotation)
6. [OAuth Client Secret Rotation](#6-oauth-client-secret-rotation)
7. [Device Certificate Lifecycle (SCEP)](#7-device-certificate-lifecycle-scep)
8. [TLS Certificate Management](#8-tls-certificate-management)
9. [API Key Rotation](#9-api-key-rotation)
10. [CMK/KMS Key Rotation](#10-cmkkms-key-rotation)
11. [Webhook Signing Secrets](#11-webhook-signing-secrets)
12. [Rotation Audit](#12-rotation-audit)
13. [Implementation Backlog with DoD](#13-implementation-backlog-with-dod)
14. [Competitive Differentiation](#14-competitive-differentiation)

---

## 1. Executive Summary

Key rotation is a fundamental security practice — rotating keys limits the blast radius of key compromise. GGID has a mature KeyProvider abstraction (`pkg/crypto/key_provider.go:39`, 7 providers) but **no automated rotation** for any key type.

**Current state:**
- KeyProvider supports local/PKCS11/AWS/GCP/Azure/Vault/SM2 ✅
- JWT signing keys loaded at startup, never rotated ❌
- OAuth client secrets manually created, never rotated ❌
- Device certs issued via SCEP, no auto-renewal ❌
- TLS certs manually managed ❌
- API keys manually created, no scheduled rotation ❌
- CMK rotation designed but not implemented ❌

**Recommendation**: Implement a **Rotation Engine** — a cron-based service that manages all key types with the dual-key pattern (old key remains valid during grace period), audit trail, and zero-downtime transitions.

---

## 2. GGID Current State

| Key Type | Provider | Rotation | Status |
|----------|----------|----------|--------|
| JWT signing | `key_provider.go:39` | Manual only | ❌ No auto-rotation |
| OAuth client secret | OAuth service | Manual only | ❌ |
| Device certs (SCEP) | `pkg/scep/` | None | ❌ No auto-renewal |
| TLS certs | Gateway | Manual | ❌ No cert-manager |
| API keys | Gateway middleware | Manual | ❌ No scheduled rotation |
| CMK/DEK | `key_provider.go` (researched) | Designed | ❌ Not implemented |
| Webhook secrets | Audit webhook engine | Manual | ❌ |
| Hash chain HMAC | `hash_chain.go:13` | Manual | ❌ |

---

## 3. Gap Analysis

| # | Gap | Risk |
|---|-----|------|
| 1 | No JWT signing key rotation | Key compromise = all tokens forgeable |
| 2 | No client secret rotation | Compromised secret = impersonation |
| 3 | No device cert renewal | Expired certs = devices locked out |
| 4 | No TLS automation | Expired TLS = service down |
| 5 | No API key rotation | Compromised key = persistent access |
| 6 | No CMK rotation | KMS key compromise = all data exposed |
| 7 | No webhook secret rotation | Replay attacks on webhooks |

---

## 4. Dual-Key Rotation Pattern

### Universal Pattern (applies to all key types)

```
Time:      T-0           T+grace         T+grace+cleanup
           │             │                │
Key A:  ────████████████████████████────── (old, expires after grace)
Key B:       ───████████████████████████── (new, becomes primary)

Phase 1 (T-0): Generate Key B alongside Key A
Phase 2 (T-0 to T+grace): Both keys valid (signing=B, verification=A+B)
Phase 3 (T+grace): Key A expires, only Key B valid
Phase 4 (T+cleanup): Key A deleted from active store
```

### Verification During Grace Period

```go
// During grace period, verifier accepts both old and new keys
func VerifyToken(token string) error {
    for _, key := range activeKeys {
        if err := verifyWithKey(token, key); err == nil {
            return nil  // Verified by either old or new key
        }
    }
    return ErrInvalidToken
}
```

---

## 5. JWT Signing Key Rotation

### Rotation Schedule

| Parameter | Default | Configurable |
|-----------|---------|-------------|
| Rotation interval | 30 days | Per-tenant |
| Grace period | 7 days | Per-tenant |
| Key algorithm | Ed25519 | RS256/ES256/EdDSA/SM2SM3 |
| JWKS update | Immediate | On new key activation |

### Flow

```
1. Cron triggers rotation (every 30 days)
2. Generate new Ed25519 key pair
3. Store new key in KMS (key_id = "jwt-signing-v2")
4. Add to active key set (signing switches to new key)
5. JWKS endpoint publishes both old + new public keys
6. Old key remains in verify-only mode for 7 days
7. After 7 days: remove old key, update JWKS
```

### JWKS Endpoint

```bash
GET /.well-known/jwks.json

{
  "keys": [
    { "kid": "jwt-signing-v1", "kty": "OKP", "crv": "Ed25519", "x": "...", "exp": 1721217600 },
    { "kid": "jwt-signing-v2", "kty": "OKP", "crv": "Ed25519", "x": "..." }
  ]
}
```

---

## 6. OAuth Client Secret Rotation

### Rotation Flow

```bash
# Admin triggers rotation (or cron)
POST /api/v1/oauth/clients/{id}/rotate-secret

# Response (grace period):
{
  "new_secret": "ggid_new_5f8a3b2c...",
  "old_secret_expires_at": "2026-07-24T00:00:00Z",
  "grace_period_days": 7
}
```

### Grace Period Logic

```go
func ValidateClientSecret(clientID, secret string) bool {
    client := getClient(clientID)
    
    // Check new secret first
    if bcrypt.CompareHashAndPassword(client.SecretHash, []byte(secret)) == nil {
        return true
    }
    
    // During grace period, check old secret too
    if client.OldSecretHash != nil && time.Now().Before(client.OldSecretExpiresAt) {
        if bcrypt.CompareHashAndPassword(client.OldSecretHash, []byte(secret)) == nil {
            return true  // Old secret still valid during grace
        }
    }
    
    return false
}
```

---

## 7. Device Certificate Lifecycle (SCEP)

### Auto-Renewal Flow

```
Cron (daily at 03:00):
  1. Query: device_certificates WHERE expires_at < NOW() + 7 days AND revoked_at IS NULL
  2. For each expiring cert:
     a. Generate new CSR for device
     b. Sign with internal CA
     c. Push new cert to device (via MDM or SCEP renewal)
     d. Mark old cert as "superseded" (not revoked — still valid until expiry)
     e. Log rotation
```

### Auto-Revoke on Unenroll

```
MDM webhook: deviceunenrolled
  → GGID receives webhook
  → Revoke device cert (CRL update)
  → Revoke all sessions for device
  → Revoke all tokens bound to device
```

---

## 8. TLS Certificate Management

### cert-manager Integration (K8s)

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ggid-gateway-tls
spec:
  secretName: ggid-gateway-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - ggid.corp.com
  duration: 2160h    # 90 days
  renewBefore: 360h  # Renew 15 days before expiry
```

### Benefits

| Feature | Manual | cert-manager |
|---------|--------|-------------|
| Renewal | Manual (human remembers) | Automatic |
| Before expiry | Unknown | 15 days |
| Multi-domain | SAN manually added | Wildcard or per-domain |
| Let's Encrypt | Manual ACME | Automatic ACME |
| Failure alerting | None | cert-manager status |

---

## 9. API Key Rotation

### Rotation Flow

```bash
# User rotates API key
POST /api/v1/auth/api-keys/{id}/rotate

# Response:
{
  "new_key": "ggid_ak_new_abc123...",
  "old_key_expires_at": "2026-07-24T00:00:00Z",
  "grace_period_days": 7
}
```

### Scheduled Rotation Enforcement

```sql
-- API keys older than max_age must be rotated
SELECT id, key_prefix, created_at, last_rotated_at
FROM api_keys
WHERE last_rotated_at < NOW() - INTERVAL '90 days'
  AND status = 'active';

-- Result: list of keys needing rotation → notify user
```

---

## 10. CMK/KMS Key Rotation

### When CMK Rotates

```
1. New CMK created (CMK-v2)
2. For each existing DEK:
   a. Decrypt DEK with old CMK (CMK-v1)
   b. Re-encrypt DEK with new CMK (CMK-v2)
   c. Update encrypted_dek in database
3. Old CMK marked as "retired" (still can decrypt old DEKs if any missed)
4. After verification: delete CMK-v1
```

---

## 11. Webhook Signing Secrets

```go
// Rotation: new HMAC secret generated, old secret kept for grace period
type WebhookConfig struct {
    CurrentSecret  string    // Used for signing new webhooks
    PreviousSecret string    // Accepted for verification during grace
    SecretChangedAt time.Time
    GraceEndsAt    time.Time
}

// Verification: try current first, then previous during grace
func VerifyWebhook(payload, signature string, cfg *WebhookConfig) bool {
    if hmac.Equal(signature, computeHMAC(payload, cfg.CurrentSecret)) {
        return true
    }
    if time.Now().Before(cfg.GraceEndsAt) {
        return hmac.Equal(signature, computeHMAC(payload, cfg.PreviousSecret))
    }
    return false
}
```

---

## 12. Rotation Audit

### Database Schema

```sql
CREATE TABLE key_rotation_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    key_type        VARCHAR(32) NOT NULL,     -- 'jwt_signing', 'client_secret', 'device_cert', 'tls', 'api_key', 'cmk', 'webhook'
    key_id          VARCHAR(128) NOT NULL,    -- Resource being rotated
    
    old_key_id      VARCHAR(128),             -- Previous key identifier
    new_key_id      VARCHAR(128) NOT NULL,    -- New key identifier
    triggered_by    VARCHAR(32) NOT NULL,     -- 'scheduled', 'manual', 'auto_renew'
    triggered_by_user UUID,                   -- If manual
    
    grace_ends_at   TIMESTAMPTZ,              -- When old key becomes invalid
    status          VARCHAR(16) DEFAULT 'active', -- 'active', 'completed', 'failed'
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_rotation_tenant_type ON key_rotation_log (tenant_id, key_type, created_at DESC);
CREATE INDEX idx_rotation_pending ON key_rotation_log (status) WHERE status = 'active';
```

---

## 13. Implementation Backlog with DoD

### P0 — Rotation Engine + JWT Keys (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Rotation engine (cron + dual-key pattern) | ✅ Supports all key types ✅ DB-backed audit ✅ Grace period ✅ ≥3 tests | 4d |
| 2 | JWT signing key rotation (30d + 7d grace) | ✅ Auto-generate new key ✅ JWKS publishes both ✅ Zero-downtime ✅ ≥3 tests | 3d |
| 3 | Rotation audit log | ✅ Every rotation logged ✅ Key IDs tracked ✅ ≥3 tests | 2d |
| 4 | cert-manager integration for TLS | ✅ Auto-renew TLS certs ✅ Let's Encrypt ACME ✅ ≥3 tests | 2d |

### P1 — Client Secrets + Device Certs + API Keys (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | OAuth client secret rotation | ✅ Grace period logic ✅ API endpoint ✅ ≥3 tests | 2d |
| 6 | Device cert auto-renewal (SCEP) | ✅ 7-day window ✅ Auto-renew ✅ ≥3 tests | 3d |
| 7 | API key rotation + scheduled enforcement | ✅ User-initiated + scheduled ✅ Grace period ✅ ≥3 tests | 2d |
| 8 | Webhook signing secret rotation | ✅ Periodic rotation ✅ Grace verification ✅ ≥3 tests | 1d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 9 | CMK rotation with DEK re-encryption | Full KMS key lifecycle |
| 10 | Rotation dashboard (Console) | Visual rotation schedule + alerts |
| 11 | Emergency key revocation | One-click revoke all keys |
| 12 | Key escrow (optional) | Backup keys to escrow service |

---

## 14. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak | AWS Cognito |
|---------|---------------|------|-------|----------|-------------|
| **JWT key rotation** | 30d + dual-key | Yes | Yes | Manual | Yes |
| **Client secret rotation** | Scheduled + grace | Yes | Custom | Manual | Yes |
| **Device cert auto-renew** | SCEP 7d window | Via Intune | No | No | No |
| **TLS automation** | cert-manager + ACME | Managed | Managed | Manual | Managed |
| **API key rotation** | Scheduled + grace | Yes | Custom | No | Yes |
| **CMK rotation** | DEK re-encryption | Managed | Managed | No | Managed |
| **Webhook secret rotation** | Periodic + grace | Yes | No | No | No |
| **Rotation audit** | DB-backed trail | Internal | Internal | No | CloudTrail |
| **Open source** | Yes | No | No | Yes | No |

---

## References

- [NIST SP 800-57: Key Management](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-57pt1r5.pdf) — Key rotation guidance
- [cert-manager](https://cert-manager.io/) — K8s TLS automation
- [Let's Encrypt ACME](https://letsencrypt.org/docs/) — Free TLS certs
- [JWKS (RFC 7517)](https://datatracker.ietf.org/doc/html/rfc7517) — JSON Web Key Set
- [RFC 8594: Deprecation](https://datatracker.ietf.org/doc/html/rfc8594) — Sunset headers
- [GGID KeyProvider](../pkg/crypto/key_provider.go) — 7 providers at line 39
- [GGID SCEP Package](../pkg/scep/) — Device certificate infrastructure
- [GGID Hash Chain](../services/audit/internal/domain/hash_chain.go) — HMAC at line 13
- [GGID CMK/KMS Research](./customer-managed-keys-kms.md) — CMK rotation design
- [GGID MDM Research](./mdm-integration.md) — Device cert lifecycle
- [GGID Production Hardening](./production-hardening-checklist.md) — cert-manager flagged
