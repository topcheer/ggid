# Privacy-Enhancing Technologies for IAM

## 1. Overview

Privacy-Enhancing Technologies (PETs) minimize data exposure while preserving
functional value. In Identity and Access Management (IAM), PETs are critical:
identity systems **inherently** process PII — names, emails, phone numbers, IP
addresses, biometric templates, behavioral patterns.

The core tension: IAM must verify identity (requiring data) while privacy
regulation demands data minimization. Key regulatory drivers:

- **GDPR** (EU): Art. 5(1)(c) data minimization, Art. 25 privacy by design, Art. 32 security of processing.
- **CCPA/CPRA** (California): right to know, right to delete, right to opt out.
- **PIPL** (China): explicit consent for sensitive PII, cross-border transfer restrictions.

**Goal:** deliver authentication/authorization with minimal PII exposure at every
layer — in transit, at rest, in logs, in tokens, and in analytics.

---

## 2. Data Minimization in IAM

**Principle:** Collect only the data strictly necessary for the specific purpose,
at the specific time it is needed.

### Progressive Profiling

Collect data over time as capabilities are unlocked (e.g., phone number only
when SMS MFA is enabled). Reduces attack surface, aligns with GDPR Art. 5(1)(c).

### Scope-Limited Claims

OIDC scopes define which user claims a client receives. GGID should enforce
strict scope-to-claim mapping:

```
openid     → sub only
profile    → name, family_name, given_name, preferred_username
email      → email, email_verified
phone      → phone_number, phone_number_verified
```

### Token Minimization

JWTs should contain only the claims needed for authorization, not the full user
profile. GGID's current `AccessTokenClaims` is already minimal:

```go
// services/auth/internal/service/token_service.go
type AccessTokenClaims struct {
    TenantID string `json:"tenant_id"`
    jwt.RegisteredClaims  // sub, iss, aud, iat, exp, jti
}
```

This is good — only `tenant_id` and standard registered claims are included.
**No email, name, or phone in the access token.**

### Audit: Operation → Minimum Data Required

| IAM Operation | Minimum Data | GGID Current | Gap |
|---|---|---|---|
| Login (local) | username, password hash | username, bcrypt hash | None |
| JWT validation | sub, iss, aud, exp | sub, iss, aud, exp, tenant_id | None |
| User profile read | fields requested by scope | full user record | Filter by scope |
| Audit log write | action, resource_id, timestamp | includes IP, email | Mask IP/email |
| MFA enrollment | user_id, secret | user_id, TOTP secret | None |
| Account deletion | user_id | user_id, all related records | Cascade delete |

---

## 3. PII Pseudonymization Patterns

### What is Pseudonymization

Pseudonymization replaces direct identifiers with pseudonyms — tokens, hashes, or
encrypted values — while retaining a (protected) mapping to the original.
Critically, **pseudonymized data is still personal data** under GDPR (Recital 26),
but the risk is significantly reduced: a breach of pseudonymized records alone
cannot identify individuals without the separate mapping key.

### Techniques

| Technique | Reversible | Searchable | Example |
|---|---|---|---|
| Tokenization (vault) | Yes (vault lookup) | No (must query vault) | `email_a1b2c3` |
| Deterministic hash | No (one-way) | Yes (hash same) | `SHA-256(email + pepper)` |
| Format-preserving encryption | Yes (key holder) | No | Encrypted email looks like email |
| Blind index | No (one-way) | Yes | Separate `email_hash` column |

### GGID's Current State

GGID's `pkg/pii/` package provides **log obfuscation only** — it masks PII for
display but does not encrypt or tokenize at rest:

```go
// pkg/pii/pii.go — current capabilities
func MaskEmail(email string) string   // "user@example.com" → "u***@e***.com"
func MaskPhone(phone string) string   // "+1-234-567-8901" → "*******8901"
func MaskIP(ip string) string         // "192.168.1.100" → "192.168.x.x"
func MaskUUID(id string) string       // "550e8400-e29b-..." → "550e8400-****-..."
func Obfuscate(s string) string       // regex-based masking for log streams
```

This covers SSNs, credit card numbers, UUIDs, emails, phones, and IPs in log
output. **However, PII stored in PostgreSQL is plaintext** — no column-level
encryption, no tokenization, no blind index.

### Proposed: PII Vault Service

```go
// pkg/pii/vault.go — proposed
package pii

type Pseudonymizer interface {
    Tokenize(field, plaintext string) (token string, err error)
    Detokenize(token string) (plaintext string, err error)
    BlindIndex(field, plaintext string) (index string, err error)
}

type AESVault struct {
    key   []byte  // 256-bit key from KMS
    audit func(action, field string)
}

// Tokenize encrypts plaintext into an opaque token.
func (v *AESVault) Tokenize(field, plaintext string) (string, error) {
    ciphertext, err := aesGCMEncrypt(v.key, []byte(plaintext))
    if err != nil {
        return "", err
    }
    token := base64URL(ciphertext)
    v.audit("tokenize", field)
    return token, nil
}

// BlindIndex creates a deterministic hash for equality search.
func (v *AESVault) BlindIndex(field, plaintext string) (string, error) {
    h := hmac.New(sha256.New, v.blindKey)
    h.Write([]byte(field))
    h.Write([]byte(plaintext))
    return base64URL(h.Sum(nil)), nil
}
```

**Database schema change:**
```sql
-- Before: plaintext PII
CREATE TABLE users (
    email VARCHAR(255),
    phone VARCHAR(20)
);

-- After: encrypted PII + blind index
CREATE TABLE users (
    email_encrypted  BYTEA,           -- AES-GCM ciphertext
    email_blind_idx  VARCHAR(64),     -- HMAC-SHA256 for equality search
    phone_encrypted  BYTEA,
    phone_blind_idx  VARCHAR(64)
);
CREATE UNIQUE INDEX idx_users_email_blind ON users(email_blind_idx);
```

---

## 4. Differential Privacy for Analytics

### What it Does

Differential privacy (DP) adds calibrated noise to aggregate query results so
that individual records cannot be inferred. The privacy budget (epsilon, epsilon)
controls the privacy-utility tradeoff: smaller epsilon = more privacy = more noise.

**Example:** "How many users in tenant X have MFA enabled?" → exact count 1,247
becomes 1,247 +/- Laplace(1/epsilon). For trend analysis (is adoption
increasing?), the noise is negligible. For exact enumeration, it defeats the
query.

### Application in IAM

- **Login analytics:** success/failure rates per tenant, without revealing
  individual login attempts.
- **MFA adoption metrics:** adoption rate over time without exposing which users
  have/don't have MFA.
- **Session duration statistics:** average session length without revealing
  individual session patterns.
- **Cross-tenant benchmarks:** "your tenant is in the 75th percentile for MFA
  adoption" — no exact counts shared.

### Implementation Sketch

```go
// pkg/analytics/dp.go — proposed
func DPCount(trueCount int64, epsilon float64) int64 {
    if epsilon <= 0 { return trueCount } // no privacy (debug only)
    return trueCount + int64(laplaceNoise(1.0 / epsilon))
}

func laplaceNoise(b float64) float64 {
    u := rand.Float64() - 0.5
    return -b * sign(u) * math.Log(1-2*math.Abs(u))
}
```

### Practical Constraints

DP is **not suitable for real-time auth decisions** — the noise makes exact
answers impossible. Use only for batch reporting, analytics dashboards, and
privacy-preserving cross-tenant benchmarks.

---

## 5. Homomorphic Encryption

### What it Does

Homomorphic Encryption (HE) enables computation on encrypted data without
decryption. Three categories:

- **Partially Homomorphic (PHE):** one operation (e.g., Paillier = addition).
- **Somewhat Homomorphic (SHE):** limited depth of additions and multiplications.
- **Fully Homomorphic (FHE):** arbitrary computation (Gentry's scheme, 2009).

### Application in IAM

- **Password verification on encrypted hash:** verify credentials without the
  server ever seeing the plaintext password or the hash.
- **Attribute verification across orgs:** verify "user is admin" without
  revealing the user's identity to the verifying org.
- **Threshold authentication:** multiple parties jointly authenticate a user
  without any single party seeing the full credential.

### Practical Status (2025)

FHE remains 1000x-10000x slower than plaintext. Recent benchmarks (Nature, 2025)
show it is practical for offline analytics and privacy-preserving ML inference,
but **not for real-time auth** (sub-100ms needed). FHE hardware acceleration
(GPU/ASIC) shows 10-100x speedup but is still years from production IAM.

### Recommendation

- **Monitor:** FHE performance improving — timeline 5-10 years for production IAM.
- **Alternative now:** Zero-knowledge proofs for narrow use cases (age
  verification, membership proof) are viable today.

---

## 6. Zero-Knowledge Proofs for Identity

### Use Cases

- Prove **age >= 18** without revealing birthdate
- Prove **membership in a group** without revealing which group or which member
- Prove **credential validity** without revealing credential contents
- Prove **email ownership** without revealing the email address

### Implementation: zk-SNARKs

zk-SNARKs let a prover create a succinct proof that a statement is true,
verifiable in milliseconds, with the verifier learning nothing beyond validity.

### Integration with Verifiable Credentials

**BBS+ signatures** embed ZKP directly into the credential format:

- Issuer signs a credential with BBS+ (e.g., "date_of_birth: 1990-01-15").
- Holder creates a ZKP proving a derived claim (e.g., "age >= 18") **without
  revealing the birthdate**.
- Verifier checks the ZKP against the issuer's public key — no contact with
  issuer needed, and the proof is unlinkable across verifications.

**Standards status (2025):** BBS+ is in IETF/CFRG draft ("The BBS Signature
Scheme"). DIF has reference implementations. NIST has presented on BBS+
standardization. Not yet RFC, but active pilot deployments.

### GGID Consideration

- **Not P0/P1** — standards not finalized, ecosystem immature.
- **Monitor:** When BBS+ reaches RFC status (expected 2026-2027), integrate
  with a Verifiable Credentials system.
- **Use case:** Privacy-preserving attribute verification for regulated
  industries (healthcare, finance) — prove eligibility without disclosing PII.

---

## 7. Secure Multi-Party Computation (SMPC)

### Cross-Org Identity Verification

Two organizations want to check if the same user exists in both systems (for
deduplication or fraud detection) **without revealing their full user lists** to
each other.

### Private Set Intersection (PSI)

PSI is a specific SMPC protocol: both parties hash their user identifiers, exchange
hashed sets, and compute intersection locally — each party learns only shared
users, not the full set.

**IAM use cases:** fraud detection (user in multiple deny-lists), account
deduplication during mergers, compliance checks.

### Practical Status

PSI is viable for moderate-size sets (thousands to millions of entries) with batch
processing. Too slow for real-time auth. Typical use: nightly fraud-detection runs.

---

## 8. GDPR/CCPA Compliance Through PETs

### GDPR Principles Mapped to PETs

| GDPR Article | Principle | PET Technique | GGID Status |
|---|---|---|---|
| Art. 5(1)(c) | Data minimization | Scope-limited claims, progressive profiling | JWT minimal; profile read needs scope filter |
| Art. 5(1)(e) | Storage limitation | Token expiry, automatic data deletion | Token TTL implemented; data retention not |
| Art. 25 | Privacy by design | Pseudonymization by default | Not implemented — PII stored plaintext |
| Art. 32 | Security of processing | Encryption at rest, tokenization | **Gap: no column-level encryption** |
| Art. 33 | Breach notification | Pseudonymization reduces impact | N/A until encryption implemented |
| Art. 35 | DPIA | Risk assessment for high-risk processing | Not documented |

### CCPA Considerations

- **Right to know:** Tokenization vault must support full data export for DSARs.
- **Right to delete:** Vault must cascade-delete plaintext + blind indices.
- **Opt-out of sale:** Pseudonymized tokens have no market value.

### GGID Compliance Gaps

1. **PII at rest is plaintext** — PostgreSQL columns store email, phone, name in
   cleartext. No AES-GCM, no field-level encryption.
2. **Audit logs contain PII** — `pkg/pii.Obfuscate()` exists for masking but is
   only applied in log output, not consistently across all audit event fields.
3. **No data retention policy** — user records have no automatic expiry or TTL.
4. **No DP for analytics** — tenant-level queries return exact counts.

---

## 9. GGID PET Implementation Roadmap

| Phase | Task | Priority | Effort | Impact |
|---|---|---|---|---|
| 1 | PII encryption at rest (AES-GCM for email/phone columns) | P0 | 2-3 sprints | GDPR Art. 32 compliance |
| 2 | Pseudonymization vault (tokenize/detokenize, audit log) | P1 | 2 sprints | Breach impact reduction |
| 3 | Blind index search (HMAC column for equality queries) | P1 | 1 sprint | Searchable encrypted PII |
| 4 | Scope-limited JWT claims enforcement (filter profile by OIDC scope) | P1 | 1 sprint | GDPR Art. 5(1)(c) |
| 5 | Audit log PII scrubbing (apply Obfuscate to all audit fields) | P1 | 1 sprint | Log breach containment |
| 6 | Differential privacy for analytics (DPCount for dashboards) | P2 | 1 sprint | Privacy-preserving metrics |
| 7 | Data retention policies (automatic TTL, cascade delete) | P2 | 2 sprints | GDPR Art. 5(1)(e) |
| 8 | ZKP/BBS+ for attribute proofs (when RFC published) | P3 | 3-4 sprints | Advanced privacy features |
| 9 | SMPC/PSI for cross-tenant fraud detection | P3 | 3 sprints | Future capability |

### Phase 1 Detail: PII Encryption at Rest

This is the highest-impact, lowest-risk enhancement:

```go
// pkg/pii/encrypt.go — proposed Phase 1
func Encrypt(key, plaintext []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    rand.Read(nonce)
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(key, ct []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    ns := gcm.NonceSize()
    return gcm.Open(nil, ct[:ns], ct[ns:], nil)
}
```

Migration: dual-write (plaintext + encrypted), backfill existing rows, then drop
plaintext columns. Key management via environment key (now) -> KMS (later).

### Summary

GGID's current PII posture is **masking for logs** and **minimal JWT claims** —
both good. The primary gap is **no encryption at rest**. Phases 1-5 deliver
immediate compliance value. Phases 8-9 (ZKP, SMPC) are research-stage.
