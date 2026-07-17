# Privacy-Enhancing Technologies (PETs) for Identity Systems: Privacy-by-Design for GGID

> **Focus**: Applying privacy-enhancing technologies to GGID — zero-knowledge proofs, selective disclosure (BBS+/SD-JWT), differential privacy for analytics, pseudonymization, data minimization, and right-to-be-forgotten implementation. This document builds on the theoretical analysis in `privacy-enhancing-technologies.md` (384 lines) with production implementation specifics.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§11), DoD per backlog item (§12), curl commands where applicable.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Regulatory Drivers](#2-regulatory-drivers)
3. [GGID Current State: Privacy Infrastructure](#3-ggid-current-state-privacy-infrastructure)
4. [Gap Analysis](#4-gap-analysis)
5. [Zero-Knowledge Proofs for Authentication](#5-zero-knowledge-proofs-for-authentication)
6. [Selective Disclosure: BBS+ and SD-JWT](#6-selective-disclosure-bbs-and-sd-jwt)
7. [Differential Privacy for Identity Analytics](#7-differential-privacy-for-identity-analytics)
8. [Pseudonymization and Tokenization](#8-pseudonymization-and-tokenization)
9. [Right to Be Forgotten: Technical Implementation](#9-right-to-be-forgotten-technical-implementation)
10. [Proposed Architecture](#10-proposed-architecture)
11. [Endpoint Precondition Check](#11-endpoint-precondition-check)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Competitive Differentiation](#13-competitive-differentiation)

---

## 1. Executive Summary

Privacy is a legal mandate. GDPR Art. 25 requires "data protection by design and by default." GGID has foundational privacy infrastructure (SD-JWT, DLP, PII log masking, data classification) but lacks BBS+ selective disclosure, differential privacy, automated erasure pipeline, and pseudonymization vault.

**Recommendation**: Implement PETs in 4 layers: cryptographic (BBS+/asymmetric SD-JWT), analytics (differential privacy), operational (pseudonymization/erasure/crypto-shredding), and governance (minimization/consent).

**Estimated effort**: 4 sprints for MVP.

---

## 2. Regulatory Drivers

| Regulation | Article | Requirement |
|-----------|---------|-------------|
| **GDPR** | Art. 25 | Data protection by design and by default |
| **GDPR** | Art. 17 | Right to erasure |
| **GDPR** | Art. 20 | Data portability |
| **CCPA** | §1798.105 | Right to delete |
| **PIPL** | Art. 6 | Minimum necessary principle |
| **LGPD** | Art. 16 | Data minimization |

---

## 3. GGID Current State: Privacy Infrastructure

| Component | File:Line | Status | Privacy Capability |
|-----------|-----------|--------|-------------------|
| SD-JWT issue | `identity/server/sdjwt_handler.go:86` | ⚠️ HMAC | Selective disclosure (limited) |
| SD-JWT verify | `sdjwt_handler.go:174` | ⚠️ HMAC | Third-party can't verify |
| DLP EvaluateDLP | `dlp_handler.go:157` | ✅ | Data classification enforcement |
| Data classification | `data_gov_repo.go:14` | ✅ | Resource labeling |
| PII log masking (auth) | `pii_logging.go:7` | ✅ | PII obfuscated in logs |
| PII log masking (oauth) | `pii_logging.go:7` | ✅ | PII obfuscated in logs |
| Audit PII masking | `audit_service.go:79` | ✅ | Event fields masked |

---

## 4. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | SD-JWT uses HMAC | Third parties can't verify disclosed claims |
| 2 | No BBS+ signatures | No unlinkable multi-claim selective disclosure |
| 3 | No ZKP authentication | Can't prove attributes without revealing |
| 4 | No differential privacy | Analytics expose individual patterns |
| 5 | No pseudonymization vault | PII stored in plaintext |
| 6 | No automated erasure | Manual deletion only |
| 7 | No data minimization rules | No schema-level constraints |
| 8 | No backup purge for erasure | Deleted from DB, remains in backups |

---

## 5. Zero-Knowledge Proofs for Authentication

### Problem ZKP Solves

```
Traditional: "I was born on 1990-03-15" → reveals exact birth date
ZKP: "I can prove I'm ≥18 without revealing my birth date"
```

### Implementation: zk-SNARKs (P3/Future)

Using gnark (Go-native ZKP library). Most enterprise use cases served by BBS+ (P1) until ZKP tooling matures.

---

## 6. Selective Disclosure: BBS+ and SD-JWT

### BBS+ vs SD-JWT Comparison

| Property | BBS+ | SD-JWT |
|----------|------|--------|
| Unlinkability | ✅ Yes | ❌ No |
| Crypto basis | BLS12-381 pairings | HMAC/EdDSA |
| W3C VC compatible | ✅ | ✅ |
| Performance | ~10ms sign | ~1ms sign |
| Standardization | W3C + IETF draft | IETF RFC 9496 |

### BBS+ Flow

```
1. Issuer signs all claims with BBS+
2. Holder derives proof for subset only (e.g., name + age, not salary)
3. Verifier validates: signed by trusted issuer, contains only disclosed claims
4. Unlinkable: multiple presentations can't be correlated
```

### SD-JWT Asymmetric Upgrade

```go
// Current (HMAC): header: { "alg": "HS256", "typ": "sd-jwt" }
// Target (EdDSA): header: { "alg": "EdDSA", "typ": "sd-jwt", "kid": "did:web:corp.com#key-1" }
```

---

## 7. Differential Privacy for Identity Analytics

### Laplace Mechanism

```go
func AddNoise(actual int, epsilon float64) int {
    scale := 1.0 / epsilon  // sensitivity = 1 for count queries
    noise := laplaceSample(0, scale)
    return actual + int(math.Round(noise))
}
// epsilon=1.0: reasonable privacy budget for identity analytics
```

### Privacy Budget

| Setting | Epsilon | Use Case |
|---------|---------|----------|
| Strict | 0.1 | Health, minors |
| Standard | 1.0 | Default identity analytics |
| Relaxed | 10.0 | Total user count |

---

## 8. Pseudonymization and Tokenization

```
Application Layer:  sees pseud_abc123 (never real email)
Pseudonymization Vault:  alice@corp.com ↔ pseud_abc123 (CMK-encrypted)
Database Layer:  stores pseud_abc123 (DB compromise = no PII)
```

---

## 9. Right to Be Forgotten: Technical Implementation

### Erasure Pipeline

```
1. Identity verification (email OTP + admin approval for high-risk)
2. Cascade deletion:
   ├── users, user_credentials, user_sessions → DELETE
   ├── oauth_tokens, api_keys, device_posture → DELETE
   ├── audit_events → ANONYMIZE (replace user_id with hash for compliance)
   ├── consent_records, analytics_events → DELETE or anonymize
   └── plugin_kv_store → DELETE
3. Cache invalidation: Redis flush + PDP cache
4. Crypto-shredding: delete user's DEK → encrypted backups unreadable
5. Third-party notification: SCIM, SAML SPs
6. Audit: log erasure completion (user_id → erasure_id)
```

### Crypto-Shredding

```
Each user's data encrypted with their own DEK.
Erasure = delete the DEK → all encrypted data cryptographically unrecoverable.
No need to find and purge every backup.
```

---

## 10. Proposed Architecture

```
┌──────────────────────────────────────────────┐
│         Privacy Enhancement Layer             │
│                                              │
│  Cryptographic: BBS+ / Asymmetric SD-JWT / ZKP│
│  Analytics: Differential Privacy / Budget     │
│  Operational: Pseudonymization / Erasure /    │
│               Crypto-Shredding                │
│  Governance: Consent-driven / Purpose limit / │
│              Retention auto-purge             │
└──────────────────────────────────────────────┘
```

---

## 11. Endpoint Precondition Check

### Existing (Upgrade)

| Component | File:Line | Current | Target |
|----------|-----------|---------|--------|
| SD-JWT issue | `sdjwt_handler.go:86` | HMAC | EdDSA |
| SD-JWT verify | `sdjwt_handler.go:174` | HMAC | EdDSA |
| PII log masking | `pii_logging.go:7` | ✅ Static | Configurable |
| DLP engine | `dlp_handler.go:157` | ✅ | + pseudonymization |

### New Components

| Component | Priority |
|-----------|----------|
| Asymmetric SD-JWT (EdDSA) | P0 |
| Erasure pipeline | P0 |
| BBS+ signatures | P1 |
| Differential privacy | P1 |
| Crypto-shredding | P1 |
| Pseudonymization vault | P2 |

---

## 12. Implementation Backlog with DoD

### P0 — Asymmetric SD-JWT + Erasure (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Upgrade SD-JWT to EdDSA | ✅ Third-party verifiable ✅ Replace HMAC ✅ ≥3 tests | 2d |
| 2 | Erasure pipeline | ✅ Cascade DELETE all tables ✅ Cache flush ✅ ≥3 tests | 4d |
| 3 | Erasure API | ✅ POST /privacy/erase ✅ Identity verification ✅ Audit trail ✅ ≥3 tests | 2d |
| 4 | Audit anonymization | ✅ Retain audit with user_id → hash ✅ ≥3 tests | 2d |

### P1 — BBS+ + Differential Privacy (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | BBS+ signatures | ✅ Issue + derive + verify ✅ Selective disclosure ✅ ≥3 tests | 4d |
| 6 | Differential privacy | ✅ Laplace noise ✅ Privacy budget tracking ✅ ≥3 tests | 3d |
| 7 | Privacy audit API | ✅ Track data access per user ✅ ≥3 tests | 2d |
| 8 | Crypto-shredding | ✅ Per-user DEK deletion ✅ ≥3 tests | 3d |

### P2 — Pseudonymization + Minimization (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 9 | Pseudonymization vault | ✅ PII → token ✅ CMK-backed ✅ Reversible by auth only ✅ ≥3 tests | 4d |
| 10 | Data minimization rules | ✅ Schema field constraints ✅ ≥3 tests | 2d |
| 11 | Retention auto-purge | ✅ Auto-delete after period ✅ Per-type policy ✅ ≥3 tests | 3d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 12 | ZKP authentication (zk-SNARK) | Prove attributes without revealing |
| 13 | Homomorphic encryption | Compute on encrypted data |
| 14 | Secure multi-party computation | Cross-org attribute verification |
| 15 | Blind signatures | Server signs without seeing content |
| 16 | Privacy Pass integration | Anonymous auth tokens |

---

## 13. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | Auth0 | Apple |
|---------|---------------|------|-----------------|-------|-------|
| **Asymmetric SD-JWT** | **EdDSA** | No | Partial | No | Private |
| **BBS+ signatures** | **Yes** | No | No | No | Private |
| **Differential privacy** | **Laplace** | No | No | No | Private |
| **Erasure pipeline** | **Cascade + crypto-shred** | Manual | Partial | Manual | N/A |
| **Pseudonymization** | **CMK vault** | No | No | No | Private |
| **ZKP** | **Planned** | No | No | No | Private |
| **Open source** | **Yes** | No | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM with BBS+ selective disclosure, differential privacy analytics, and automated crypto-shredding erasure.

---

## References

- [GDPR Art. 25 (Data Protection by Design)](https://gdpr-info.eu/art-25-gdpr/)
- [GDPR Art. 17 (Right to Erasure)](https://gdpr-info.eu/art-17-gdpr/)
- [IETF RFC 9496 (SD-JWT)](https://datatracker.ietf.org/doc/html/rfc9496)
- [BBS+ Signature (IETF Draft)](https://datatracker.ietf.org/doc/draft-irtf-cfrg-bbs-signature/)
- [Differential Privacy Guide](https://desfontain.es/privacy/)
- [Microsoft Presidio](https://microsoft.github.io/presidio/)
- [gnark (Go ZKP)](https://github.com/consensys/gnark)
- [GGID SD-JWT Handler](../services/identity/internal/server/sdjwt_handler.go) — HMAC at line 86
- [GGID PII Logging](../services/auth/internal/service/pii_logging.go) — Log masking at line 7
- [GGID DLP Engine](../services/identity/internal/server/dlp_handler.go) — EvaluateDLP at line 157
- [GGID Privacy Technologies (existing)](./privacy-enhancing-technologies.md) — Previous research (384 lines)
- [GGID Consent Management](./consent-management-platform.md) — Consent platform
- [GGID DLP Egress](./dlp-egress-pii-redaction.md) — PII redaction
- [GGID CMK/KMS](./customer-managed-keys-kms.md) — Crypto-shredding foundation
