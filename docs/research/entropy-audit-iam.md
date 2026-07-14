# Entropy Audit: Random Number Generation Security for IAM Systems

**Audit Date:** 2025-07-11
**Scope:** GGID IAM Suite — all Go source files across `pkg/`, `services/`, `sdk/`
**Classification:** CRITICAL — Security Audit
**Auditor:** Security Research Team

---

## Executive Summary

This audit examines every random number generation path in the GGID codebase to verify
that security-sensitive values (tokens, session IDs, salts, challenges, OAuth codes) are
generated using cryptographically secure random number generators (CSPRNG). The audit
identifies **3 insecure entropy sources**, **2 acceptable-but-improvable patterns**,
and confirms that **24+ security-critical values use proper `crypto/rand` entropy**.

### Key Findings

| Finding | Severity | Status |
|---------|----------|--------|
| Login attempt IDs use `time.Now().UnixNano()` | **LOW** | Non-security value |
| SAML `generateID()` falls back to timestamp on CSPRNG failure | **MEDIUM** | Defense-in-depth gap |
| `math/rand` in retry/canary/shadow middleware | **NONE** | Non-security (traffic routing) |
| CSRF token: previously time-based, now fixed with `crypto/rand` | **RESOLVED** | Fixed in commit 29b51c1 |
| All token generation uses `crypto.GenerateRandomToken(32)` | **PASS** | 256-bit entropy |
| Password salts use `crypto/rand` (16 bytes) | **PASS** | 128-bit entropy |
| Phone OTP uses `crypto/rand.Int` | **PASS** | Uniform distribution |
| Device/user codes use `cryptoRandInt` (wraps `crypto/rand`) | **PASS** | CSPRNG-based |

---

## 1. crypto/rand vs math/rand

### Why crypto/rand Is Mandatory for Security-Sensitive Values

In an Identity and Access Management (IAM) system, the unpredictability of tokens,
session IDs, salts, and challenges is a foundational security property. If an attacker
can predict these values, they can forge sessions, hijack accounts, bypass MFA, and
compromise the entire authentication pipeline.

**`crypto/rand`** uses the operating system's CSPRNG:
- **Linux:** `/dev/urandom` (or `getrandom(2)` syscall)
- **macOS:** `SecRandomCopyBytes`
- **Windows:** `RtlGenRandom` / `ProcessPrng`

These sources draw entropy from hardware events (disk seek times, interrupt timing,
keyboard/mouse input, thermal noise) and feed it through a DRBG (e.g., ChaCha20 or HMAC-DRBG).
The output is **computationally indistinguishable from true randomness**.

**`math/rand`** is a PRNG based on a linear-feedback algorithm:
- **Deterministic:** Same seed → identical sequence
- **Predictable:** Once the internal state is known (624 32-bit words for MT), all future
  outputs are predictable
- **Not seeded by default in Go < 1.20:** The default seed is `1`, making the output
  sequence identical across process restarts — a catastrophic weakness

### Common Go Mistakes

```go
// MISTAKE 1: Using math/rand.Read for tokens
import "math/rand"
func generateToken() string {
    b := make([]byte, 32)
    rand.Read(b)  // PREDICTABLE — not cryptographically secure
    return hex.EncodeToString(b)
}

// MISTAKE 2: Seeding math/rand with time
import (
    "math/rand"
    "time"
)
func init() {
    rand.Seed(time.Now().UnixNano()) // Seeds to ~2^30 distinct values per second
}
// Problem: An attacker who knows the approximate seed time can brute-force
// the seed and predict all generated values.

// MISTAKE 3: Using timestamp directly as a token
func generateID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
// Problem: Only ~30 bits of entropy per second — trivially predictable.
```

### Correct Pattern (used throughout GGID)

```go
import (
    "crypto/rand"
    "encoding/base64"
    "io"
)

func GenerateRandomToken(byteLen int) (string, error) {
    b := make([]byte, byteLen)
    if _, err := io.ReadFull(rand.Reader, b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}
```

This pattern is used by `pkg/crypto/crypto.go` and is the central token generation
function for the entire GGID system. All service-level code that needs random tokens
calls `crypto.GenerateRandomToken(N)`.

---

## 2. Token Generation Audit

### Central Token Generation: `pkg/crypto/crypto.go`

**File:** `pkg/crypto/crypto.go:157-164`

```go
func GenerateRandomToken(byteLen int) (string, error) {
    b := make([]byte, byteLen)
    if _, err := io.ReadFull(rand.Reader, b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}
```

**Verdict: PASS.** Uses `crypto/rand.Reader` via `io.ReadFull`, which guarantees the
full byte slice is filled. Output is base64url-encoded for URL safety.

### Callers of GenerateRandomToken

| Caller | File | Size | Entropy | Status |
|--------|------|------|---------|--------|
| Auth token service (encrypt/decrypt) | `auth/internal/service/token_service.go:103` | 32 bytes | 256 bits | PASS |
| Auth token rotation | `auth/internal/service/token_service.go:169` | 32 bytes | 256 bits | PASS |
| Email lockout token | `auth/internal/service/email_lockout.go:29` | 32 bytes | 256 bits | PASS |
| Identity API token | `identity/internal/service/identity_service.go:176` | 32 bytes | 256 bits | PASS |
| Session token | `auth/internal/service/session_service.go:36` | 32 bytes | 256 bits | PASS |
| Step-up challenge | `auth/internal/service/stepup.go:39` | 32 bytes | 256 bits | PASS |
| Step-up token | `auth/internal/service/stepup.go:121` | 32 bytes | 256 bits | PASS |
| Password reset token | `auth/internal/service/password_service.go:139` | 32 bytes | 256 bits | PASS |
| MFA challenge | `auth/internal/service/auth_service.go:121` | 32 bytes | 256 bits | PASS |
| Magic link token | `auth/internal/service/auth_service.go:512` | 32 bytes | 256 bits | PASS |
| WebAuthn challenge | `auth/internal/service/auth_service.go:797` | 32 bytes | 256 bits | PASS |
| OAuth auth code | `oauth/internal/service/oauth_service.go:243` | 32 bytes | 256 bits | PASS |
| OAuth refresh token | `oauth/internal/service/oauth_service.go:795` | 32 bytes | 256 bits | PASS |
| OAuth client ID | `oauth/internal/service/oauth_service.go:908` | 16 bytes | 128 bits | PASS |
| OAuth client secret | `oauth/internal/service/oauth_service.go:913` | 32 bytes | 256 bits | PASS |
| Email change tokens | `auth/internal/service/email_change.go:38,42` | 32 bytes | 256 bits | PASS |
| Temp password | `auth/internal/server/http.go:1654` | 32 bytes | 256 bits | PASS |

**All 17 callers use `crypto/rand` via `GenerateRandomToken`.** No instances of
`math/rand` are used for security-sensitive token generation.

### UUID Generation

GGID uses `github.com/google/uuid` throughout for UUID v4 generation (`uuid.New()`).
The Google UUID library generates UUID v4 using `crypto/rand`:

```go
// google/uuid/marshal.go (internal)
func New() UUID {
    var uuid UUID
    _, _ = rand.Read(uuid[:])  // crypto/rand
    uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
    uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 10
    return uuid
}
```

**Verdict: PASS.** UUID v4 uses 122 bits of CSPRNG entropy.

---

## 3. Session ID / CSRF Token Generation

### CSRF Token — `services/gateway/internal/middleware/middleware.go`

**File:** `services/gateway/internal/middleware/middleware.go:204-213`

```go
func generateCSRFToken() string {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        panic("crypto/rand failed: " + err.Error())
    }
    hash := sha256.Sum256(b)
    return base64.RawURLEncoding.EncodeToString(hash[:])
}
```

**Verdict: PASS (previously RESOLVED).** This was previously a P0 vulnerability
using `time.Now().UnixNano()` as entropy. Fixed in commit `29b51c1` to use
`crypto/rand` with a fail-closed panic. 256 bits of raw entropy, hashed to
SHA-256 (256-bit output).

**Entropy:** 256 bits (32 bytes from `crypto/rand`).

### Session ID

Session IDs are generated via `crypto.GenerateRandomToken(32)` in
`auth/internal/service/session_service.go:36`.

**Entropy:** 256 bits.

### OAuth State Parameter

The OAuth `state` parameter is generated on the client side and validated server-side.
GGID's authorization flow does NOT generate the state parameter server-side; it
validates it against a Redis-stored value during the callback. The CSRF protection
relies on the state matching, and the state value's entropy depends on the client.

**Recommendation:** Server-generated state parameters should use
`crypto.GenerateRandomToken(32)` (256 bits) as documented in
`docs/research/oauth-state-csrf.md`.

### Nonce Generation

Nonce usage is seen in:
- AES-GCM encryption nonces: 12 bytes from `crypto/rand` in `crypto.go:122-123`
- PKCE/OIDC nonce parameter: stored in authorization codes (`domain/models.go:110`)
- RFC 7523 back-channel logout: `jti` field for replay prevention

**Verdict: PASS.** AES-GCM nonces use 96-bit CSPRNG entropy (standard for GCM).

---

## 4. Password Salt Generation

**File:** `pkg/crypto/crypto.go:57-63`

```go
func HashPassword(password string) (string, error) {
    salt := make([]byte, argonSaltLength) // argonSaltLength = 16
    if _, err := io.ReadFull(rand.Reader, salt); err != nil {
        return "", fmt.Errorf("failed to generate salt: %w", err)
    }
    hash := argon2.IDKey(applyPepper(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
    // ...
}
```

**Verdict: PASS.** Analysis:

1. **Salt length:** 16 bytes (128 bits) — meets the OWASP minimum of 16 bytes for Argon2id.
2. **Entropy source:** `crypto/rand.Reader` via `io.ReadFull`.
3. **Uniqueness:** A fresh salt is generated on every `HashPassword` call. The test
   `TestHashPassword_DifferentSalats` verifies that the same password produces different
   hashes across calls.
4. **Per-user:** Every password (including password changes) gets a new random salt.
5. **Pepper:** Optional HMAC-SHA256 pepper can be set via `SetPepper()`, adding
   server-side secret protection. Even if the database is leaked, the attacker cannot
   verify password guesses without the pepper.

**Entropy:** 128 bits per salt.

---

## 5. JWT ID (jti) Generation

The `jti` claim in JWTs is used for replay prevention. GGID's JWT implementation
generates `jti` claims in two ways:

### Auth Service JWTs

Auth service JWTs include a UUID-based `jti`:
```go
jti := uuid.New().String()  // crypto/rand-based UUID v4
```

**Verdict: PASS.** 122 bits of CSPRNG entropy.

### RFC 7523 Back-Channel Logout

**File:** `services/oauth/internal/service/rfc7523.go:83-84`

```go
jti, _ := claims["jti"].(string)
```

The `jti` is extracted from the logout token's claims. It is used for replay
prevention via a Redis-backed deduplication set.

**Verdict: PASS.** The `jti` value is provided by the issuer and stored for
replay detection. GGID correctly checks for `jti` presence and tracks it.

### Anti-Replay (JWT jti tracking)

JWT `jti` values are tracked in Redis using `SETNX` to prevent token replay attacks.
This was fixed in commit `72edaa5`.

---

## 6. OAuth Code and Token Generation

### Authorization Code

**File:** `services/oauth/internal/service/oauth_service.go:243`

```go
plaintextCode, err := crypto.GenerateRandomToken(32)
```

**Entropy:** 256 bits (32 bytes from `crypto/rand`). Exceeds the 256-bit minimum
for OAuth authorization codes.

### Refresh Token

**File:** `services/oauth/internal/service/oauth_service.go:795`

```go
newRefreshToken, err := crypto.GenerateRandomToken(32)
```

**Entropy:** 256 bits. Token rotation is implemented — old tokens are revoked on use.

### Access Token

Access tokens are JWTs signed with the service's signing key. The entropy of the token
comes from the `jti` (UUID v4, 122 bits) and the cryptographic signature.

### Client ID and Secret

**File:** `services/oauth/internal/service/oauth_service.go:906-915`

```go
func generateClientID() string {
    id, _ := crypto.GenerateRandomToken(16)
    return "gcid_" + id
}

func generateClientSecret() string {
    secret, _ := crypto.GenerateRandomToken(32)
    return "gcs_" + secret
}
```

**Client ID:** 128 bits. Public identifier — entropy requirement is lower.
**Client Secret:** 256 bits. Secret credential — meets 256-bit minimum.

### Device Code and User Code

**File:** `services/oauth/internal/service/oauth_service.go:1344-1366`

```go
func generateDeviceCode(length int) string {
    const charset = "ABCDEFGH...xyz0123456789" // 62 chars
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[cryptoRandInt(len(charset))]
    }
    return string(b)
}

func generateUserCode() string {
    const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 32 chars, no confusing chars
    // ... uses cryptoRandInt(32) per character
}
```

Where `cryptoRandInt` wraps `crypto/rand.Int`:

```go
func cryptoRandInt(max int) int {
    bigN, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
    // ...
}
```

**Device Code (20 chars, 62-char alphabet):** ~119 bits. **PASS.**
**User Code (8 chars, 32-char alphabet):** ~40 bits. **Acceptable** — user codes
are short-lived (5 minutes), rate-limited, and entered manually by humans.

### Phone OTP

**File:** `services/auth/internal/service/phone_otp.go:116-123`

```go
func generateNumericOTP(n int) (string, error) {
    max := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
    num, err := rand.Int(rand.Reader, max) // crypto/rand
    return fmt.Sprintf("%0*d", n, num), nil
}
```

**Verdict: PASS.** Uses `crypto/rand.Int` for uniform distribution. 6-digit OTP
= ~20 bits of entropy. Acceptable for a rate-limited, 5-minute TTL OTP.

---

## 7. SAML ID Generation

**File:** `pkg/saml/sp.go:123-131`

```go
func generateID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Fallback to timestamp-based ID if crypto/rand fails.
        return fmt.Sprintf("_%d", time.Now().UnixNano())
    }
    return fmt.Sprintf("_%x", b)
}
```

**Verdict: MEDIUM RISK (defense-in-depth gap).**

The primary path uses `crypto/rand` (16 bytes = 128 bits), which is correct.
However, the fallback path degrades to `time.Now().UnixNano()`, which provides only
~30 bits of entropy per second. While `crypto/rand` failure is extremely unlikely on
modern operating systems, the fallback should **fail closed** (panic or return an error)
rather than silently degrading to predictable values.

**Remediation:**

```go
func generateID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        // Fail closed — never use weak entropy for SAML IDs.
        panic("crypto/rand failed: " + err.Error())
    }
    return fmt.Sprintf("_%x", b)
}
```

**Entropy (primary path):** 128 bits.

---

## 8. WebAuthn Challenge Generation

**File:** `services/auth/internal/service/auth_service.go:795-798`

```go
func (s *AuthService) GenerateWebAuthnChallenge(ctx context.Context) (string, error) {
    return crypto.GenerateRandomToken(32)
}
```

**Verdict: PASS.** 256 bits of CSPRNG entropy. Exceeds the WebAuthn spec minimum
of 128 bits (16 bytes).

### WebAuthn Fallback Concern

**File:** `services/auth/internal/server/http.go:777-780`

```go
challenge, err := h.authSvc.GenerateWebAuthnChallenge(r.Context())
if err != nil {
    // Fall back to a simple random challenge.
    challenge = uuid.New().String()
}
```

**Verdict: ACCEPTABLE.** The fallback uses UUID v4 (122 bits CSPRNG), which is
still cryptographically secure. However, if the primary path failed, it is better
to fail the request than to silently fall back. The likelihood of `crypto/rand`
failure is near zero on modern systems.

---

## 9. Entropy Quantification

### Comprehensive Entropy Source Table

| # | Component | File | Generator | Bytes | Entropy | Min Required | Status |
|---|-----------|------|-----------|-------|---------|-------------|--------|
| 1 | Auth token | `token_service.go:103` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 2 | Auth token rotation | `token_service.go:169` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 3 | Email token | `email_lockout.go:29` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 4 | API token | `identity_service.go:176` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 5 | Session token | `session_service.go:36` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 6 | Step-up challenge | `stepup.go:39` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 7 | Step-up token | `stepup.go:121` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 8 | Password reset | `password_service.go:139` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 9 | MFA challenge | `auth_service.go:121` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 10 | Magic link token | `auth_service.go:512` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 11 | WebAuthn challenge | `auth_service.go:797` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 12 | Password salt | `crypto.go:58` | `crypto/rand` | 16 | 128 bits | 128 bits | PASS |
| 13 | AES-GCM nonce | `crypto.go:122` | `crypto/rand` | 12 | 96 bits | 96 bits | PASS |
| 14 | CSRF token | `middleware.go:204` | `crypto/rand` | 32+SHA256 | 256 bits | 128 bits | PASS |
| 15 | Request ID | `middleware.go:46` | UUID v4 | 16 | 122 bits | N/A (tracing) | PASS |
| 16 | Audit event ID | `audit/publisher.go:95` | UUID v4 | 16 | 122 bits | N/A | PASS |
| 17 | OAuth auth code | `oauth_service.go:243` | `crypto/rand` | 32 | 256 bits | 256 bits | PASS |
| 18 | OAuth refresh token | `oauth_service.go:795` | `crypto/rand` | 32 | 256 bits | 256 bits | PASS |
| 19 | OAuth client ID | `oauth_service.go:908` | `crypto/rand` | 16 | 128 bits | 128 bits | PASS |
| 20 | OAuth client secret | `oauth_service.go:913` | `crypto/rand` | 32 | 256 bits | 256 bits | PASS |
| 21 | Device code | `oauth_service.go:1345` | `crypto/rand.Int` | 20 chars | 119 bits | 128 bits | PASS* |
| 22 | User code | `oauth_service.go:1355` | `crypto/rand.Int` | 8 chars | 40 bits | N/A (short-lived) | PASS |
| 23 | Phone OTP | `phone_otp.go:116` | `crypto/rand.Int` | 6 digits | 20 bits | N/A (rate-limited) | PASS |
| 24 | SAML ID (primary) | `saml/sp.go:125` | `crypto/rand` | 16 | 128 bits | 128 bits | PASS |
| 25 | SAML ID (fallback) | `saml/sp.go:128` | `time.Now()` | ~8 | ~30 bits | 128 bits | **FLAG** |
| 26 | Login attempt ID | `login_attempt.go:34` | `time.Now()` | ~8 | ~30 bits | N/A (non-security) | LOW |
| 27 | WebAuthn fallback | `http.go:780` | UUID v4 | 16 | 122 bits | 128 bits | PASS |
| 28 | Email change tokens | `email_change.go:38,42` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 29 | Temp password | `http.go:1654` | `crypto/rand` | 32 | 256 bits | 128 bits | PASS |
| 30 | PAR request_uri | `par.go:89` | UUID v4 | 16 | 122 bits | 128 bits | PASS |
| 31 | Webhook ID | `webhooks.go:202` | UUID v4 | 16 | 122 bits | N/A | PASS |
| 32 | Consent record ID | `consent.go:66` | UUID v4 | 16 | 122 bits | N/A | PASS |

*Device code at 119 bits slightly below 128-bit target but uses a 62-character alphabet
over 20 positions and is short-lived with rate limiting. Practically sufficient.

---

## 10. Comprehensive GGID Entropy Audit

### math/rand Usage in Source Code

**Files using `math/rand` in non-test, non-doc source:**

| File | Purpose | Security Impact |
|------|---------|-----------------|
| `services/gateway/internal/middleware/retry.go:4` | Exponential backoff jitter | **NONE** — traffic routing, not security |
| `services/gateway/internal/middleware/canary.go:5` | Canary traffic percentage | **NONE** — deployment routing |
| `services/gateway/internal/middleware/shadow_mirror.go:6` | Shadow traffic mirror | **NONE** — deployment routing |

All three uses of `math/rand` are for **non-security traffic routing decisions**
(backoff delays, canary percentages, shadow traffic sampling). These are not
security-sensitive values, and using `math/rand` here is the correct engineering
choice (lower overhead, no entropy pool consumption).

### crypto/rand Usage in Source Code

**All security-sensitive files correctly import `crypto/rand`:**

| File | Purpose |
|------|---------|
| `pkg/crypto/crypto.go:9` | Token generation, salt, AES-GCM nonce |
| `pkg/saml/sp.go:4` | SAML ID generation |
| `services/auth/internal/service/token_service.go:5` | Auth token encryption |
| `services/auth/internal/service/phone_otp.go:5` | OTP generation |
| `services/auth/internal/service/auth_service.go` | MFA challenge, WebAuthn |
| `services/oauth/internal/service/oauth_service.go:6` | Auth codes, refresh tokens, client secrets |
| `services/oauth/internal/server/server.go:6` | OAuth server operations |
| `services/gateway/internal/middleware/middleware.go:6` | CSRF token, request ID |
| `services/gateway/internal/middleware/otel.go:7` | Sampling |

### time.Now().UnixNano as Entropy Source

**Non-test source files using `time.Now().UnixNano()` or `.Unix()` for IDs/values:**

| File | Line | Usage | Security Impact |
|------|------|-------|-----------------|
| `services/auth/internal/service/login_attempt.go:34` | `ID: fmt.Sprintf("%d", time.Now().UnixNano())` | **LOW** — internal attempt tracking, not a credential |
| `services/auth/internal/service/login_attempt.go:48` | `score := float64(time.Now().UnixNano())` | **NONE** — Redis sorted set score |
| `services/auth/internal/service/auth_service.go:755` | `val := fmt.Sprintf("%s:%d", deviceName, time.Now().Unix())` | **NONE** — trusted device cache key |
| `pkg/saml/sp.go:128` | `return fmt.Sprintf("_%d", time.Now().UnixNano())` | **MEDIUM** — SAML ID fallback (see Section 7) |

**Login attempt IDs** are used as Redis keys for rate-limiting bookkeeping. They are
not security credentials and their predictability has no security impact. However, for
consistency and to avoid collisions under concurrent attempts, using UUID would be better.

---

## 11. Gap Analysis & Recommendations

### Finding 1: SAML generateID Fallback to Timestamp (MEDIUM)

**Location:** `pkg/saml/sp.go:128`

**Problem:** When `crypto/rand.Read` fails (extremely unlikely), the function falls
back to `time.Now().UnixNano()`, producing a SAML ID with only ~30 bits of entropy.
This creates a predictable SAML response/request ID.

**Risk:** A predictable SAML ID could theoretically be exploited in replay or
correlation attacks. However, the probability of `crypto/rand` failure on a modern
operating system is effectively zero.

**Priority:** MEDIUM (defense-in-depth)

**Remediation:**

```go
func generateID() string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        panic("crypto/rand failed: " + err.Error())
    }
    return fmt.Sprintf("_%x", b)
}
```

### Finding 2: Login Attempt ID Uses Timestamp (LOW)

**Location:** `services/auth/internal/service/login_attempt.go:34`

**Problem:** Login attempt IDs use `time.Now().UnixNano()` which provides only
nanosecond-resolution uniqueness. Under concurrent login attempts, this can produce
duplicate IDs, causing one attempt to overwrite another in Redis.

**Risk:** Low. Not a security credential. Only affects audit accuracy under
extreme concurrency.

**Priority:** LOW (code quality)

**Remediation:**

```go
attempt := LoginAttempt{
    ID:            uuid.New().String(),
    // ...
}
```

### Finding 3: WebAuthn Challenge Fallback to UUID (LOW)

**Location:** `services/auth/internal/server/http.go:780`

**Problem:** If `GenerateWebAuthnChallenge` fails, the handler falls back to
`uuid.New().String()`. While UUID v4 is cryptographically secure (122 bits), it is
shorter than the primary path's 256-bit challenge. Mixing generators can also cause
encoding inconsistencies (UUID hex vs base64url).

**Risk:** Very low. UUID v4 provides sufficient entropy. The concern is consistency.

**Priority:** LOW (code quality)

**Remediation:** Return an error instead of silently falling back:

```go
challenge, err := h.authSvc.GenerateWebAuthnChallenge(r.Context())
if err != nil {
    writeError(w, http.StatusInternalServerError, "challenge generation failed")
    return
}
```

### Finding 4: OAuth State Parameter Not Server-Generated (INFORMATIONAL)

**Observation:** GGID does not generate OAuth `state` parameters server-side. The
state is passed from the client and validated against a stored value. While this
works if the client generates a sufficiently random state, it places entropy
responsibility on the client.

**Priority:** INFORMATIONAL

**Recommendation:** For server-side authorization flows, generate state server-side
using `crypto.GenerateRandomToken(32)` and store it in Redis with a short TTL.

### Positive Findings

1. **No `math/rand` in any security-sensitive code path.** All token, salt,
   challenge, and code generation uses `crypto/rand`.
2. **Centralized token generation** via `pkg/crypto.GenerateRandomToken` ensures
   consistency across all services.
3. **CSRF token vulnerability was already fixed** (commit 29b51c1).
4. **Password hashing uses proper CSPRNG salts** (16 bytes, per-password).
5. **OAuth authorization codes and refresh tokens exceed the 256-bit minimum.**
6. **Phone OTP uses `crypto/rand.Int` for uniform distribution** (no modulo bias).
7. **Device/user codes use `crypto/rand.Int` via `cryptoRandInt`** wrapper.
8. **UUID generation uses `google/uuid` library** which internally uses `crypto/rand`.

### Remediation Priority Matrix

| Finding | Priority | Effort | Risk if Unfixed |
|---------|----------|--------|-----------------|
| SAML ID fallback | MEDIUM | 1 line change | Predictable SAML IDs on CSPRNG failure |
| Login attempt ID | LOW | 1 line change | Audit duplicates under concurrency |
| WebAuthn challenge fallback | LOW | 2 lines | Inconsistent challenge format |
| Server-side OAuth state | INFORMATIONAL | Design change | Client entropy responsibility |

### Code Review Checklist for Future Development

- [ ] All token/secret values use `crypto/rand` (via `crypto.GenerateRandomToken`)
- [ ] No `math/rand` usage in security-sensitive code paths
- [ ] No `time.Now()` used as an entropy source for credentials
- [ ] CSPRNG failure paths fail closed (panic or error), not fall back to weak entropy
- [ ] Tokens are at least 256 bits (32 bytes) for long-lived credentials
- [ ] Salts are at least 128 bits (16 bytes) and unique per credential
- [ ] Challenges are at least 128 bits for WebAuthn/OAuth
- [ ] OTP codes use `crypto/rand.Int` to avoid modulo bias
- [ ] UUID v4 (`google/uuid`) is used for UUID generation

---

## Appendix A: GGID `math/rand` Full Inventory (Source Files Only)

| File | Line | Import | Purpose | Security |
|------|------|--------|---------|----------|
| `gateway/middleware/retry.go` | 4 | `math/rand` | Backoff jitter | Safe |
| `gateway/middleware/canary.go` | 5 | `math/rand` | Canary % routing | Safe |
| `gateway/middleware/shadow_mirror.go` | 6 | `math/rand` | Shadow traffic % | Safe |

**Conclusion:** Zero `math/rand` usage in security-sensitive code. All three usages
are in non-security traffic routing middleware where `math/rand` is the correct choice.

## Appendix B: GGID `crypto/rand` Full Inventory (Source Files Only)

| File | Line | Import | Purpose |
|------|------|--------|---------|
| `pkg/crypto/crypto.go` | 9 | `crypto/rand` | Tokens, salts, nonces |
| `pkg/saml/sp.go` | 4 | `crypto/rand` | SAML ID generation |
| `auth/service/token_service.go` | 5 | `crypto/rand` | Auth token encryption |
| `auth/service/phone_otp.go` | 5 | `crypto/rand` | Phone OTP generation |
| `oauth/service/oauth_service.go` | 6 | `crand "crypto/rand"` | Codes, tokens, secrets |
| `oauth/server/server.go` | 6 | `crypto/rand` | OAuth server |
| `gateway/middleware/middleware.go` | 6 | `crypto/rand` | CSRF, request ID |
| `gateway/middleware/otel.go` | 7 | `crypto/rand` | Sampling |

---

## References

- NIST SP 800-90A: Recommendation for Random Number Generation Using Deterministic RBGs
- OWASP Cryptographic Storage Cheat Sheet
- RFC 6749 Section 10.10: OAuth 2.0 Credentials Entropy
- WebAuthn Level 3 Section 7.2: Cryptographic Challenges
- Go Documentation: `crypto/rand` package
- GGID commit 29b51c1: CSRF predictable entropy fix
- GGID commit 72edaa5: JWT anti-replay (jti tracking)

---

*This document was generated as part of the GGID Security Research initiative. All
findings are based on source code analysis of the GGID monorepo as of 2025-07-11.*
