# Password Policy Best Practices — NIST SP 800-63B & GGID Audit

> Research document analyzing modern password policy guidelines and auditing GGID's
> current implementation against NIST SP 800-63B (Revision 3, superseded by
> SP 800-63-4 as of August 2025) recommendations.

---

## 1. NIST SP 800-63B Password Guidelines

### What NIST REQUIRES (Revision 3 / carried into Revision 4)

The guidelines fundamentally changed the approach to password policy — shifting
from complexity rules to length, uniqueness, and breached-password checks.

| Requirement | Detail |
|---|---|
| **Minimum length** | **8 characters** (AAL2+). Not 12+, not 16+. Length is the primary strength factor. |
| **Maximum length** | Must accept at least **64 characters**. Allows long passphrases. |
| **No composition rules** | Remove "must contain uppercase, number, special character." Proven ineffective. |
| **No mandatory rotation** | Only rotate on **suspected or confirmed compromise**. Periodic rotation produces weaker passwords. |
| **Character set** | Accept **all printable ASCII** (0x20–0x7E) + **spaces** + **Unicode**. Do not strip or sanitize. |
| **Breached password check** | **MANDATORY.** Check new passwords against known breached password corpora. |
| **Rate limiting** | Throttle authentication attempts. Lockout or exponential backoff after failures. |
| **Salt + memory-hard hashing** | Use Argon2id, bcrypt, scrypt, or PBKDF2 with high iteration counts. |

### What NIST REJECTs

| Rejected Practice | Reason |
|---|---|
| **Composition rules** | Users predictably add "1!" — negligible entropy gain, increases frustration. |
| **Mandatory periodic rotation** | Users choose weaker passwords ("summer2024", "summer2025"). Only rotate on compromise. |
| **Knowledge-based verification** | Security questions (mother's maiden name) are easily discovered via OSINT. |
| **SMS for high assurance** | SIM-swapping risk. Use TOTP, WebAuthn, or push notifications for MFA. |
| **Password hints** | Trivially guessable. Eliminate entirely. |

> **Key insight:** The single most effective control is **breached-password
> checking** combined with **minimum length 8** and **memory-hard hashing**.

---

## 2. Breached Password Checking

### k-Anonymity Approach (HIBP)

The k-anonymity protocol, popularized by Have I Been Pwned (HIBP), allows
breach checking without exposing the full password hash:

```
1. Client computes SHA-1(password)
2. Client sends FIRST 5 HEX CHARS of hash to breach API
   e.g. GET https://api.pwnedpasswords.com/range/21BD1
3. Server returns ALL hash suffixes starting with that prefix (~500 entries)
4. Client checks locally whether the FULL hash is in the returned list
```

**Privacy guarantee:** The server never sees more than 5 hex characters —
approximately 1 million possible prefixes, so the set is large enough that
individual passwords cannot be identified.

### Implementation Considerations

- **When to check:** On password creation and password change (not on login).
- **Action on match:** Reject with clear message: "This password has appeared in a known data breach. Please choose a different password."
- **Rate limiting:** Throttle breach-check requests to prevent enumeration.
- **Offline alternative:** Download full SHA-1 list (~30 GB compressed) and check locally (air-gapped environments).
- **Caching:** Cache API responses (per-prefix) in Redis with 1h TTL.

### Go Interface Sketch

```go
type BreachChecker interface {
    IsBreached(ctx context.Context, password string) (bool, error)
}

// HIBPClient checks via the HIBP k-anonymity API.
type HIBPClient struct {
    baseURL string
    rdb     *redis.Client  // prefix-level cache
}

func (c *HIBPClient) IsBreached(ctx context.Context, password string) (bool, error) {
    sum := sha1.Sum([]byte(password))
    hexHash := strings.ToUpper(fmt.Sprintf("%x", sum))
    prefix, suffix := hexHash[:5], hexHash[5:]

    // Check Redis cache first
    cached, err := c.rdb.Get(ctx, "hibp:"+prefix).Result()
    if err == nil {
        return strings.Contains(cached, suffix), nil
    }

    // Fetch from HIBP API
    resp, err := c.fetchRange(ctx, prefix)
    if err != nil {
        return false, err // fail-open: don't block on API errors
    }

    c.rdb.Set(ctx, "hibp:"+prefix, resp, time.Hour)
    return strings.Contains(resp, suffix), nil
}

// LocalChecker loads a downloaded breach list for offline environments.
type LocalChecker struct {
    hashes map[string]struct{} // set of full SHA-1 hashes
}
```

---

## 3. Password Hashing

### Algorithm Comparison

| Algorithm | NIST Status | Pros | Cons | Parameters |
|---|---|---|---|---|
| **Argon2id** | Recommended | Memory-hard, GPU/ASIC-resistant, won PHC | Higher CPU/memory cost | memory=64 MB, iterations=3, parallelism=2 |
| **bcrypt** | Acceptable | Battle-tested, widely supported | 72-byte password limit | cost=12+ |
| **scrypt** | Acceptable | Memory-hard | Less widely deployed | N=2^17, r=8, p=1 |
| **PBKDF2** | Acceptable | FIPS-compliant | Not memory-hard, GPU-vulnerable | 600,000+ iterations (SHA-256) |
| **MD5/SHA-1/SHA-256** | **NEVER** | — | Fast, GPU-crackable, no salt by default | — |
| **Plain text** | **NEVER** | — | No protection at all | — |

### GGID Current State

GGID uses **Argon2id** via `golang.org/x/crypto/argon2` — the recommended algorithm:

```go
// pkg/crypto/crypto.go
const (
    argonMemory      = 64 * 1024 // 64 MB
    argonIterations  = 3
    argonParallelism = 2
    argonKeyLength   = 32
    argonSaltLength  = 16
)
```

**Assessment:**
- memory=64 MB: matches OWASP recommendation (19,456 KB minimum, 65,536 KB preferred)
- iterations=3: matches OWASP recommendation (2+)
- parallelism=2: reasonable for server-side
- keyLength=32 bytes (256 bits): strong
- saltLength=16 bytes (128 bits): strong
- Random salt from `crypto/rand`: correct
- Constant-time comparison: implemented via `constantTimeCompare()`
- Hash format includes algorithm + parameters: supports future migration

**Verdict: Argon2id parameters are NIST-compliant.** The hashing layer needs no changes.

---

## 4. Password Manager Friendliness

Modern password managers (1Password, Bitwarden, KeePassXC, browser built-ins)
generate and autofill long, random passwords. Systems should not interfere:

| Practice | Status | Notes |
|---||---|
| Allow paste | Required | Never `onpaste="return false"` or `autocomplete="off"` on password fields |
| Max length >= 64 | Required | GGID has no `MaxLength` config — acceptable (effectively unlimited) |
| Accept all characters | Required | Unicode, spaces, emoji — do not strip or reject |
| Single field (no confirm) | Preferred | Or make "confirm password" optional for paste workflows |
| `autocomplete` attributes | Recommended | `new-password` for registration/change, `current-password` for login |
| No custom input widgets | Required | Don't force on-screen keyboards or masked custom controls |

**GGID status:** The password validation logic in `password_service.go` does
not restrict character types or paste behavior. The frontend console would
need an audit to verify `autocomplete` attributes and paste handling.

---

## 5. Rate Limiting

GGID implements rate limiting at multiple layers:

### Current Implementation

| Layer | Config | Default | Code Location |
|---|---|---|---|
| Per-account lockout | `PasswordPolicy.MaxAttempts` | 5 attempts | `password_service.go` + auth service lockout logic |
| Lockout duration | `PasswordPolicy.LockDuration` | 30 min | Redis-based lock with TTL |
| Per-IP/minute | `RateLimitConfig.LoginPerMinute` | 5/min | Gateway or middleware Redis rate limiter |

### NIST-Recommended Enhancements

| Enhancement | Purpose |
|---|---|
| **Exponential backoff** | 1st fail: no delay. 2nd: 1s. 3rd: 2s. 4th: 4s. Thwarts online brute-force. |
| **Artificial response delay** | Add 200-500ms to every auth response to mask timing differences between valid/invalid usernames. |
| **CAPTCHA after N failures** | Optional: present CAPTCHA after 3 failures to slow automated attacks. |
| **Progressive lockout** | 5 min → 15 min → 1 hour → permanent (admin unlock). |
| **Per-tenant throttling** | Configurable rate limits per tenant for multi-tenant fairness. |

```go
// Exponential backoff example
func authDelay(failedAttempts int) time.Duration {
    if failedAttempts <= 0 {
        return 0
    }
    delay := time.Duration(1<<uint(failedAttempts-1)) * time.Second
    if delay > 30*time.Second {
        return 30 * time.Second // cap
    }
    return delay
}
```

---

## 6. Password Strength Estimation

### zxcvbn (Dropbox)

Rather than simplistic "8 chars + uppercase" rules, zxcvbn estimates real-world
password strength using:

- Dictionary word matching (common passwords, English words, names)
- Keyboard pattern detection (qwerty, asdfgh)
- L33t speak substitution (p@ssword → password)
- Repeat detection (aaaa, abcabc)
- Sequence detection (1234, abcd)
- Date matching

**Score range:** 0 (very weak, < 10^3 guesses) → 4 (very strong, > 10^12 guesses).

### Usage Recommendation

- **Informational only** — show strength meter in UI, do not block on score
- Show suggestions ("avoid dictionary words", "add more length")
- Minimum threshold (if any): score >= 2 (still weaker than recommended, but not trivially crackable)
- Go library: `github.com/wagslane/go-password-validator` or `github.com/nbutton23/zxcvbn-go`

```go
import "github.com/nbutton23/zxcvbn-go"

score := zxcvbn.PasswordStrength(password, nil).Score
// 0=very weak, 1=weak, 2=fair, 3=strong, 4=very strong
```

---

## 7. GGID Password Policy Audit

### Current Configuration (`conf.go` defaults)

```go
PasswordPolicy{
    MinLength:      12,     // NIST says 8 — stricter than required, acceptable
    RequireUpper:   true,   // NIST says NO composition rules — NON-COMPLIANT
    RequireLower:   true,   // NON-COMPLIANT
    RequireDigit:   true,   // NON-COMPLIANT
    RequireSpecial: false,  // Compliant (not required)
    Blacklist:      nil,    // Partial — should use breach database
    HistoryCount:   5,      // Acceptable (prevents reuse, not periodic rotation)
    MaxAttempts:    5,      // Compliant
    LockDuration:   30*time.Minute, // Compliant
    MaxAgeDays:     0,      // Compliant (no forced rotation)
}
```

### Compliance Audit Table

| NIST Requirement | GGID Current | Compliant? | Action Needed |
|---|---|---|---|
| Min 8 characters | `MinLength: 12` | Yes (stricter) | Consider lowering to 8 for UX |
| Max length >= 64 | No MaxLength field | Yes (unlimited) | Add explicit MaxLength=128 for safety |
| No composition rules | `RequireUpper/Lower/Digit: true` | **No** | Set all to `false` by default |
| No mandatory rotation | `MaxAgeDays: 0` | Yes | Keep default |
| Breached password check | Not implemented | **No** | Implement HIBP k-anonymity check |
| Argon2id hashing | Argon2id, 64 MB/3 iter/2 par | Yes | No change needed |
| Rate limiting | MaxAttempts=5, LoginPerMinute=5 | Yes | Add exponential backoff |
| Salt (random, per-password) | 16-byte random salt | Yes | No change needed |
| Password history | HistoryCount=5 | Acceptable | Optional: align with breach check |
| Accept Unicode/spaces | Validation uses `unicode` package, no rejection | Yes | Verify frontend allows paste |
| Knowledge-based auth | Not implemented | Yes (N/A) | Keep as-is |

---

## 8. Implementation Roadmap

| Phase | Task | Effort | Priority |
|---|---|---|---|
| **1** | Remove composition rules: set `RequireUpper/Lower/Digit` defaults to `false`. Add optional `MaxLength` field (default 128). | S (2h) | High — immediate NIST compliance |
| **2** | Implement `BreachChecker` interface with HIBP k-anonymity client + Redis cache. Call during `SetPassword()` / registration. | M (1-2d) | High — mandatory per NIST |
| **3** | Add zxcvbn-go strength estimation for UI feedback (score + suggestions). Frontend strength meter. | S (4h) | Medium — UX improvement |
| **4** | Password manager friendliness audit: verify `autocomplete` attributes, no paste-blocking, in console frontend. | S (2h) | Medium |
| **5** | Exponential backoff on auth failures + timing-safe response delay. | S (4h) | Medium — defense in depth |

**Total estimated effort: ~1 week (1 developer).**

Phase 1 is a config-only change and can ship immediately. Phase 2 is the
critical gap — breached password checking is the one remaining NIST mandatory
requirement GGID does not yet implement.

---

## References

- NIST SP 800-63B Revision 3: https://pages.nist.gov/800-63-3/sp800-63b.html
- NIST SP 800-63-4 (current, August 2025): https://csrc.nist.gov/pubs/sp/800/63/4/final
- HIBP Pwned Passwords API: https://haveibeenpwned.com/API/v3
- OWASP Password Storage Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- Argon2 RFC 9106: https://datatracker.ietf.org/doc/rfc9106/
- zxcvbn: https://github.com/dropbox/zxcvbn
- zxcvbn-go: https://github.com/nbutton23/zxcvbn-go
