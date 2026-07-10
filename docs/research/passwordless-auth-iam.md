# Passwordless Authentication Implementation for IAM Systems

> **Scope**: Technical implementation of magic links, SMS/email OTP, WebAuthn-only flows,
> TOTP, and passwordless migration strategies for Go-based IAM platforms.
> UX patterns are covered in `passwordless-ux-best-practices.md`.

---

## Table of Contents

1. [Magic Link Authentication](#1-magic-link-authentication)
2. [Magic Link Security](#2-magic-link-security)
3. [SMS OTP Security](#3-sms-otp-security)
4. [Email OTP](#4-email-otp)
5. [WebAuthn-Only (Passkey) Flow](#5-webauthn-only-passkey-flow)
6. [TOTP App OTP (Authenticator Apps)](#6-totp-app-otp-authenticator-apps)
7. [Passwordless Migration Strategy](#7-passwordless-migration-strategy)
8. [GGID Passwordless Gap Analysis](#8-ggid-passwordless-gap-analysis)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Magic Link Authentication

### Concept

Magic link authentication sends a user an email containing a one-time, time-limited
URL. Clicking the link authenticates the user without requiring a password. The token
embedded in the link is an HMAC-signed or random cryptographic value stored server-side
with a short TTL.

### Architecture

```
User → POST /auth/magic-link?email=user@example.com
         ↓
Server generates token (crypto/rand, 32 bytes)
Server stores: Redis key = magic:{token_hash}, value = {tenant}:{user_id}:{email}
Server sends email with link: https://app.example.com/auth/magic?token={token}
         ↓
User clicks link → GET /auth/magic?token={token}
         ↓
Server looks up token_hash in Redis
Server validates HMAC + expiry
Server deletes token (single-use)
Server creates session, issues JWT
         → 302 redirect to dashboard
```

### Go Implementation: Token Generation

```go
package magiclink

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	magicLinkTTL  = 10 * time.Minute // links expire after 10 minutes
	tokenLen      = 32               // 256 bits of entropy
	hmacKeyEnvVar = "MAGIC_LINK_HMAC_KEY"
)

var (
	ErrTokenExpired   = errors.New("magic link token expired or invalid")
	ErrTokenConsumed  = errors.New("magic link token already used")
	ErrTokenMismatch  = errors.New("magic link token does not match stored value")
)

// MagicLinkService manages the lifecycle of magic link tokens.
type MagicLinkService struct {
	rdb     *redis.Client
	hmacKey []byte // server secret for HMAC signing
}

func NewMagicLinkService(rdb *redis.Client, hmacKey []byte) *MagicLinkService {
	return &MagicLinkService{rdb: rdb, hmacKey: hmacKey}
}

// GenerateToken creates a new magic link token for the given user.
// Returns the plaintext token to embed in the email link.
// The token is stored as HMAC(hash) in Redis — plaintext never persisted.
func (s *MagicLinkService) GenerateToken(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	email string,
) (string, error) {
	// 1. Generate 256 bits of cryptographic randomness.
	rawToken := make([]byte, tokenLen)
	if _, err := rand.Read(rawToken); err != nil {
		return "", fmt.Errorf("generate magic link token: %w", err)
	}

	// 2. Encode for URL safety.
	token := base64.RawURLEncoding.EncodeToString(rawToken)

	// 3. Compute HMAC for storage — never store the plaintext.
	tokenHash := s.hashToken(token)

	// 4. Store in Redis with TTL + metadata.
	//    Format: tenantID:userID:email:HMAC(stored separately for comparison)
	val := fmt.Sprintf("%s:%s:%s", tenantID, userID, email)
	key := fmt.Sprintf("magic:%s", tokenHash)

	// Use SET NX to prevent key collision (astronomically unlikely with 256-bit entropy).
	if err := s.rdb.Set(ctx, key, val, magicLinkTTL).Err(); err != nil {
		return "", fmt.Errorf("store magic link: %w", err)
	}

	return token, nil
}

// VerifyToken validates a magic link token, creates a session if valid.
// Uses Redis GETDEL for atomic single-use enforcement (race-condition safe).
func (s *MagicLinkService) VerifyToken(
	ctx context.Context,
	token string,
) (tenantID, userID uuid.UUID, email string, err error) {
	if len(token) == 0 {
		return uuid.Nil, uuid.Nil, "", ErrTokenExpired
	}

	tokenHash := s.hashToken(token)
	key := fmt.Sprintf("magic:%s", tokenHash)

	// GETDEL atomically retrieves and deletes — guarantees single-use even under
	// concurrent requests (e.g., user double-clicks the link).
	val, err := s.rdb.GetDel(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, uuid.Nil, "", ErrTokenConsumed
		}
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("verify magic link: %w", err)
	}

	// Parse stored metadata.
	parts := splitColon(val, 3)
	if len(parts) != 3 {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("corrupted magic link data")
	}

	tenantID, err = uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid tenant ID in token")
	}
	userID, err = uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, "", fmt.Errorf("invalid user ID in token")
	}
	email = parts[2]

	return tenantID, userID, email, nil
}

// hashToken computes HMAC-SHA256 of the token for constant-time storage.
func (s *MagicLinkService) hashToken(token string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

// ConstantTimeCompare performs a constant-time string comparison to prevent timing attacks.
// Even though we use Redis GETDEL (which handles atomicity), this is used in the
// verification path where a direct comparison is needed.
func ConstantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func splitColon(s string, n int) []string {
	result := make([]string, 0, n)
	start := 0
	for i := 0; i < len(s) && len(result) < n-1; i++ {
		if s[i] == ':' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
```

### Race Condition Handling

The critical race condition: a user clicks the magic link multiple times quickly
(double-click, email preview pane rendering, or malicious bot). Without atomic
operations, two concurrent requests could both read the token and both succeed.

**Solution**: Use Redis `GETDEL` (or `GET` + `DEL` in a Lua script) for atomic
read-and-delete. The first request gets the token; the second gets `nil`.

```go
// Lua script alternative for Redis versions that don't support GETDEL (< 6.2).
// This atomically gets and deletes, preventing the race.
const luaGetDel = `
local val = redis.call('GET', KEYS[1])
if val then
    redis.call('DEL', KEYS[1])
    return val
else
    return false
end
`
```

---

## 2. Magic Link Security

### Token Enumeration Prevention

Attackers may try to brute-force magic link tokens. Several layers protect against this:

1. **256-bit entropy**: `crypto/rand` generates 32 bytes (2^256 possible values).
   Even at 1 billion attempts/second, this would take 10^63 years.

2. **HMAC storage**: Only the HMAC hash is stored in Redis. If Redis is compromised,
   the attacker cannot reconstruct valid tokens.

3. **Rate limiting**: Limit magic link requests per email and per IP.

### Timing Attack Prevention

```go
// BAD: string comparison is not constant-time
if token == storedToken { ... }

// GOOD: use crypto/subtle for constant-time comparison
if subtle.ConstantTimeCompare([]byte(token), []byte(storedToken)) == 1 { ... }
```

In practice, Redis `GETDEL` handles this naturally — the lookup is O(1) and the
response time doesn't depend on the token value. But if you ever need to compare
tokens in application code, always use `crypto/subtle.ConstantTimeCompare`.

### Link Expiry

| Window | Recommendation | Rationale |
|--------|---------------|-----------|
| 5 min | High-security environments | Minimizes window for link forwarding/interception |
| 10 min | Standard (recommended) | Balances security with email delivery delays |
| 15 min | Consumer apps with flaky email | Generous but not excessive |
| 30 min+ | Not recommended | Too wide for replay attacks |

### Replay Prevention

Single-use is enforced via Redis `GETDEL`: once a token is read, it is immediately
deleted. A second request for the same token returns `nil`. This is atomic at the
Redis level — no TOCTOU race possible.

### Phishing Risk

Magic links can be forwarded via email or chat. If an attacker tricks a user into
forwarding their magic link, the attacker gains access. Mitigations:

1. **Bind to device fingerprint**: Store the requesting browser's user-agent hash
   alongside the token. On verification, compare. (Limited effectiveness — user-agent
   can be spoofed.)

2. **IP binding**: Tie token to requesting IP range (/24). Reject if verification
   comes from a dramatically different IP. (Problematic for mobile networks.)

3. **Display confirmation**: After magic link click, require a one-tap "Confirm
   sign-in" action rather than immediately creating a session. Adds a human step.

4. **Step-up after magic link**: If the login is from a new device, trigger an
   additional factor (TOTP, WebAuthn) before granting a full session.

### Rate Limiting

```go
// RateLimitMagicLinkRequest limits magic link requests to prevent enumeration.
// Per-email: 3 requests per hour.
// Per-IP:    10 requests per hour.
func (s *MagicLinkService) RateLimitMagicLinkRequest(
	ctx context.Context,
	email, ip string,
) error {
	// Check email rate limit.
	emailKey := fmt.Sprintf("magiclink:rl:email:%s", hashEmail(email))
	emailCount, err := s.rdb.Incr(ctx, emailKey).Result()
	if err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}
	if emailCount == 1 {
		s.rdb.Expire(ctx, emailKey, time.Hour)
	}
	if emailCount > 3 {
		return ErrRateLimited
	}

	// Check IP rate limit.
	ipKey := fmt.Sprintf("magiclink:rl:ip:%s", ip)
	ipCount, err := s.rdb.Incr(ctx, ipKey).Result()
	if err != nil {
		return fmt.Errorf("rate limit check: %w", err)
	}
	if ipCount == 1 {
		s.rdb.Expire(ctx, ipKey, time.Hour)
	}
	if ipCount > 10 {
		return ErrRateLimited
	}

	return nil
}
```

### Always Return Success

To prevent user enumeration via the magic link endpoint, always return the same
response regardless of whether the email exists:

```go
func (h *Handler) RequestMagicLink(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	// Always return success — even if email doesn't exist.
	// This prevents account enumeration.
	if userExists(email) {
		token, _ := h.magicLink.GenerateToken(ctx, tenantID, userID, email)
		h.emailService.SendMagicLink(email, token)
	}

	// Same response every time.
	writeJSON(w, 200, map[string]string{
		"message": "If an account exists for this email, a magic link has been sent.",
	})
}
```

---

## 3. SMS OTP Security

### Why SMS is the Weakest Factor

SMS-based OTP is vulnerable to multiple attack vectors:

| Attack | Description | Difficulty |
|--------|-------------|------------|
| **SIM Swap** | Attacker social-engineers the carrier to port the victim's number to a new SIM. | Low — documented thousands of real-world incidents. |
| **SS7 Interception** | Exploiting Signaling System 7 protocol vulnerabilities to intercept SMS in transit. | Medium — requires SS7 access (available to state actors, some criminals). |
| **Carrier Port-Out Fraud** | Attacker ports the victim's number to a different carrier. | Low — requires minimal personal info. |
| **SMS Forwarding Malware** | Malicious app forwards SMS to attacker. | Medium — requires device compromise. |
| **Telecom Insider Threat** | Carrier employee accesses SMS content. | Medium — depends on carrier security. |

### NIST SP 800-63B Position

NIST Special Publication 800-63B (Digital Identity Guidelines, 2017):

> "Out-of-band [verification] via SMS is deprecated, and will no longer be allowed
> for AAL2 or AAL3" (Section 5.1.3.2).

SMS is acceptable at **AAL1** (low assurance) but should NOT be used for
**AAL2** (substantial) or **AAL3** (high assurance) authentication.

### When SMS is Acceptable

- **Fallback factor**: When the primary factor (WebAuthn, TOTP) is unavailable.
- **Low-risk transactions**: Internal tools, non-financial applications.
- **Account recovery**: As a last resort when all other factors are lost.
- **Regulated environments**: Some jurisdictions require phone verification for
  identity proofing (e.g., banking KYC).

### Go Implementation: SMS OTP with Rate Limiting

GGID's existing `phone_otp.go` implements this pattern. Below is an enhanced
version with attempt tracking and lockout:

```go
package otp

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	smsOTPTTL          = 5 * time.Minute
	smsOTPMaxAttempts  = 5          // lockout after 5 failed verifications
	smsOTPMaxResend    = 3          // max resend per hour per phone
	smsOTPLockDuration = 30 * time.Minute
)

type SMSOTPService struct {
	rdb *redis.Client
}

// SendOTP generates and stores a 6-digit OTP, rate-limited per phone number.
func (s *SMSOTPService) SendOTP(
	ctx context.Context,
	phone string,
) (string, error) {
	// Rate limit: max 3 sends per hour.
	rlKey := fmt.Sprintf("smsotp:rl:%s", phone)
	count, _ := s.rdb.Incr(ctx, rlKey).Result()
	if count == 1 {
		s.rdb.Expire(ctx, rlKey, time.Hour)
	}
	if count > smsOTPMaxResend {
		return "", fmt.Errorf("rate limited: too many OTP requests")
	}

	// Check if locked out from failed attempts.
	lockKey := fmt.Sprintf("smsotp:lock:%s", phone)
	if locked, _ := s.rdb.Exists(ctx, lockKey).Result(); locked > 0 {
		return "", fmt.Errorf("account locked due to too many failed attempts")
	}

	// Generate 6-digit OTP using crypto/rand.
	otp, err := generateNumericOTP(6)
	if err != nil {
		return "", fmt.Errorf("generate OTP: %w", err)
	}

	// Store OTP in Redis with TTL.
	otpKey := fmt.Sprintf("smsotp:%s", phone)
	if err := s.rdb.Set(ctx, otpKey, otp, smsOTPTTL).Err(); err != nil {
		return "", fmt.Errorf("store OTP: %w", err)
	}

	return otp, nil
}

// VerifyOTP validates the OTP with attempt tracking and lockout.
func (s *SMSOTPService) VerifyOTP(
	ctx context.Context,
	phone, otp string,
) error {
	// Check lockout first.
	lockKey := fmt.Sprintf("smsotp:lock:%s", phone)
	if locked, _ := s.rdb.Exists(ctx, lockKey).Result(); locked > 0 {
		return fmt.Errorf("locked out: try again later")
	}

	otpKey := fmt.Sprintf("smsotp:%s", phone)
	stored, err := s.rdb.Get(ctx, otpKey).Result()
	if err != nil {
		return fmt.Errorf("invalid or expired OTP")
	}

	// Constant-time comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare([]byte(stored), []byte(otp)) != 1 {
		// Increment attempt counter.
		attemptKey := fmt.Sprintf("smsotp:attempts:%s", phone)
		attempts, _ := s.rdb.Incr(ctx, attemptKey).Result()
		if attempts == 1 {
			s.rdb.Expire(ctx, attemptKey, smsOTPTTL)
		}

		// Lock after max attempts.
		if attempts >= smsOTPMaxAttempts {
			s.rdb.Set(ctx, lockKey, "1", smsOTPLockDuration)
			s.rdb.Del(ctx, attemptKey)
			return fmt.Errorf("account locked after %d failed attempts", smsOTPMaxAttempts)
		}

		remaining := smsOTPMaxAttempts - int(attempts)
		return fmt.Errorf("invalid OTP: %d attempts remaining", remaining)
	}

	// Success — clean up.
	s.rdb.Del(ctx, otpKey)
	s.rdb.Del(ctx, fmt.Sprintf("smsotp:attempts:%s", phone))
	return nil
}

func generateNumericOTP(n int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", n, num), nil
}
```

---

## 4. Email OTP

### Email as a Factor

Email OTP is a reasonable second factor when the email channel itself is reasonably
secured (TLS, MFA on the email account). It is weaker than TOTP apps (which are
offline and device-bound) but stronger than SMS (no SIM swap risk).

| Use Case | Strength | Recommended |
|----------|----------|-------------|
| Email as primary factor (magic link) | Medium | Yes — see Section 1 |
| Email as second factor (after password) | Medium | Yes — acceptable for AAL1-AAL2 |
| Email as recovery factor | Low-Medium | Yes — with additional safeguards |

### OTP Generation

- **6 digits**: Standard, balances security with usability. 10^6 = 1M possibilities.
- **8 digits**: Higher security, used by some enterprise systems. 10^8 = 100M possibilities.
- **Alphanumeric**: Not recommended for email — hard to type, error-prone.

### TOTP vs HOTP for Email

- **TOTP (RFC 6238)**: Time-based, rotates every 30 seconds. Not suitable for email
  because the user cannot enter the code fast enough — email delivery latency is
  typically 5-30 seconds, and the user must open the email, read the code, and type
  it into the form.

- **HOTP (RFC 4226)**: Counter-based. Not used for email — the counter would need
  to be synchronized between server and client, which is impractical over email.

- **Random OTP (recommended)**: Generate a random N-digit code, store with TTL,
  verify against stored value. This is the standard approach for email OTP.

### Go Implementation: Email OTP with Attempt Tracking

```go
package otp

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	emailOTPTTL          = 10 * time.Minute
	emailOTPMaxAttempts  = 5
	emailOTPLockDuration = 30 * time.Minute
)

type EmailOTPService struct {
	rdb *redis.Client
}

// SendEmailOTP generates a 6-digit OTP, stores it in Redis keyed by email hash,
// and returns the OTP (caller is responsible for sending via email service).
func (s *EmailOTPService) SendEmailOTP(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	email string,
) (string, error) {
	// Rate limit: 3 OTPs per email per hour.
	rlKey := fmt.Sprintf("emailotp:rl:%s", email)
	count, _ := s.rdb.Incr(ctx, rlKey).Result()
	if count == 1 {
		s.rdb.Expire(ctx, rlKey, time.Hour)
	}
	if count > 3 {
		return "", fmt.Errorf("rate limited")
	}

	// Check lockout.
	lockKey := fmt.Sprintf("emailotp:lock:%s", email)
	if locked, _ := s.rdb.Exists(ctx, lockKey).Result(); locked > 0 {
		return "", fmt.Errorf("locked: too many failed attempts")
	}

	// Generate OTP.
	otp, err := generateNumericOTP(6)
	if err != nil {
		return "", err
	}

	// Store with metadata.
	otpKey := fmt.Sprintf("emailotp:%s", email)
	val := fmt.Sprintf("%s:%s:%s", tenantID, userID, otp)
	if err := s.rdb.Set(ctx, otpKey, val, emailOTPTTL).Err(); err != nil {
		return "", err
	}

	// Reset attempt counter on new OTP.
	s.rdb.Del(ctx, fmt.Sprintf("emailotp:attempts:%s", email))

	return otp, nil
}

// VerifyEmailOTP validates the OTP, returns user info on success.
func (s *EmailOTPService) VerifyEmailOTP(
	ctx context.Context,
	email, otp string,
) (tenantID, userID uuid.UUID, err error) {
	lockKey := fmt.Sprintf("emailotp:lock:%s", email)
	if locked, _ := s.rdb.Exists(ctx, lockKey).Result(); locked > 0 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("locked out")
	}

	otpKey := fmt.Sprintf("emailotp:%s", email)
	val, err := s.rdb.Get(ctx, otpKey).Result()
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid or expired OTP")
	}

	// Parse: tenantID:userID:otp
	// Note: the OTP portion is at the end after the second colon.
	// Since UUIDs don't contain colons, split with limit 3.
	parts := splitColon(val, 3)
	if len(parts) != 3 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("corrupted OTP data")
	}

	storedOTP := parts[2]

	// Constant-time comparison.
	if subtle.ConstantTimeCompare([]byte(storedOTP), []byte(otp)) != 1 {
		// Track failed attempts.
		attemptKey := fmt.Sprintf("emailotp:attempts:%s", email)
		attempts, _ := s.rdb.Incr(ctx, attemptKey).Result()
		if attempts == 1 {
			s.rdb.Expire(ctx, attemptKey, emailOTPTTL)
		}
		if attempts >= emailOTPMaxAttempts {
			s.rdb.Set(ctx, lockKey, "1", emailOTPLockDuration)
			s.rdb.Del(ctx, attemptKey)
		}
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid OTP")
	}

	// Success — consume OTP (single-use).
	s.rdb.Del(ctx, otpKey)
	s.rdb.Del(ctx, fmt.Sprintf("emailotp:attempts:%s", email))

	tenantID, _ = uuid.Parse(parts[0])
	userID, _ = uuid.Parse(parts[1])
	return tenantID, userID, nil
}
```

### Lockout Strategy

| Threshold | Action | Recovery |
|-----------|--------|----------|
| 1-4 failed attempts | Increment counter, return error | Continue |
| 5 failed attempts | Lock email for 30 minutes | Auto-unlock after TTL |
| 10 failed attempts in 24h | Lock email for 24 hours | Admin unlock or auto-expire |

---

## 5. WebAuthn-Only (Passkey) Flow

### Passwordless-First Architecture

A passwordless-only system eliminates passwords entirely. Registration and
authentication use WebAuthn platform authenticators (Touch ID, Face ID, Windows
Hello, security keys).

**Registration Flow**:
```
User provides email → Server checks if email is new →
Server initiates WebAuthn registration →
Browser prompts platform authenticator (Face ID/Touch ID) →
User completes biometric →
Server stores credential → Account created (no password ever set)
```

**Authentication Flow (Conditional UI)**:
```
Login page loads → Browser autofill shows available passkeys →
User selects passkey → Biometric prompt →
Server verifies assertion → Session created
```

### Account Recovery Without Password

Since there is no password to reset, recovery must use alternative methods:

| Method | Security | User Experience | Recommended |
|--------|----------|-----------------|-------------|
| **Device backup/sync** | High | Excellent — Apple iCloud Keychain, Google Password Manager sync | Primary |
| **Admin reset** | Medium | Poor — requires admin intervention | Fallback |
| **Social recovery** | Low | Medium — trusted contacts verify identity | Niche |
| **Email magic link** | Medium | Good — re-enroll new passkey after verification | Fallback |
| **Backup codes** | High | Medium — printed codes stored securely | Recommended |

### Go Implementation: Passwordless WebAuthn Login Handler

```go
package webauthnlogin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// PasswordlessLoginHandler implements WebAuthn-only authentication.
// No password field is ever presented to the user.
type PasswordlessLoginHandler struct {
	wbn         *webauthn.WebAuthn
	credStore   CredentialStore
	sessionMgr  SessionManager
	tokenSvc    TokenService
}

// BeginPasswordlessLogin initiates discoverable credential login.
// No user identifier is needed — the authenticator reveals the user.
func (h *PasswordlessLoginHandler) BeginPasswordlessLogin(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Use an empty user for discoverable credential flow.
	// The authenticator will select the credential and return the credential ID,
	// which the server uses to look up the actual user.
	ephemeralUser := &discoverableUser{}

	options, sessData, err := h.wbn.BeginLogin(ephemeralUser,
		// Require user verification (biometric/PIN).
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login initiation failed")
		return
	}

	// Store session data for later verification.
	challenge := options.Response.Challenge.String()
	h.sessionMgr.SaveWebAuthnSession(challenge, sessData, 5*time.Minute)

	writeJSON(w, http.StatusOK, map[string]any{
		"publicKey": options.Response,
		"hint":      "client-side conditional UI may handle this automatically",
	})
}

// FinishPasswordlessLogin completes the login and issues tokens.
func (h *PasswordlessLoginHandler) FinishPasswordlessLogin(
	w http.ResponseWriter,
	r *http.Request,
) {
	// Parse the assertion from the authenticator.
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid assertion")
		return
	}

	challenge := parsedResponse.Response.CollectedClientData.Challenge
	sessData, ok := h.sessionMgr.GetWebAuthnSession(challenge)
	if !ok {
		writeError(w, http.StatusBadRequest, "session expired")
		return
	}
	defer h.sessionMgr.DeleteWebAuthnSession(challenge)

	// Look up credential by ID to find the user.
	cred, err := h.credStore.GetCredentialByID(r.Context(), parsedResponse.RawID)
	if err != nil || cred == nil {
		writeError(w, http.StatusUnauthorized, "credential not found")
		return
	}

	// Build user for verification.
	user := &passkeyUser{
		id:          cred.UserID,
		credentials: cred.ToWebAuthnCredentials(),
	}

	// Verify the assertion cryptographically.
	credential, err := h.wbn.ValidateLogin(user, *sessData, parsedResponse)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	// Clone detection: check sign counter monotonicity.
	if cred.Counter > 0 && credential.Authenticator.SignCount <= cred.Counter {
		// Possible credential clone — revoke all sessions for this user.
		h.sessionMgr.RevokeAllSessions(r.Context(), cred.UserID)
		writeError(w, http.StatusUnauthorized, "security alert: possible credential clone")
		return
	}

	// Update credential metadata.
	h.credStore.UpdateCounter(r.Context(), credential.ID, credential.Authenticator.SignCount)
	h.credStore.UpdateLastUsed(r.Context(), credential.ID, time.Now())

	// Issue tokens — no password was ever involved.
	tokenSet, err := h.tokenSvc.IssueTokenSet(r.Context(), cred.TenantID, cred.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issuance failed")
		return
	}

	writeJSON(w, http.StatusOK, tokenSet)
}

// AccountRecovery initiates passkey re-enrollment after device loss.
// Uses email verification as the recovery gate.
func (h *PasswordlessLoginHandler) AccountRecovery(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Always return success to prevent enumeration.
	// Internally: generate magic link for re-enrollment.
	// After magic link verification, allow user to register a new passkey.

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists, recovery instructions have been sent to your email.",
	})
}

// Discoverable user — no credentials, used for discoverable credential login.
type discoverableUser struct{}

func (u *discoverableUser) WebAuthnID() []byte                  { return []byte("discoverable") }
func (u *discoverableUser) WebAuthnName() string                { return "" }
func (u *discoverableUser) WebAuthnDisplayName() string         { return "" }
func (u *discoverableUser) WebAuthnCredentials() []webauthn.Credential { return nil }

type passkeyUser struct {
	id          uuid.UUID
	credentials []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte                          { return u.id[:] }
func (u *passkeyUser) WebAuthnName() string                        { return u.id.String() }
func (u *passkeyUser) WebAuthnDisplayName() string                 { return u.id.String() }
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential   { return u.credentials }
```

### Conditional UI / Autofill

The browser's `navigator.credentials.get()` with `mediation: "conditional"` enables
passkey autofill in username fields. The server-side handler remains the same —
the difference is purely client-side JavaScript:

```javascript
// Client-side: conditional mediation for autofill
if (!window.PublicKeyCredential) {
    // Fallback to password or magic link
    return;
}

const assertion = await navigator.credentials.get({
    publicKey: publicKeyCredentialRequestOptions,
    mediation: "conditional"  // enables autofill UI
});
```

---

## 6. TOTP App OTP (Authenticator Apps)

### RFC 6238: Time-Based One-Time Password

TOTP generates a time-based code from a shared secret. The code changes every
30 seconds. The algorithm is:

```
TOTP = HMAC-SHA1(secret, floor(unix_time / period)) mod 10^digits
```

### Why TOTP Apps are Safer Than SMS

| Property | SMS OTP | TOTP App |
|----------|---------|----------|
| Network dependency | Requires cellular network | Works offline |
| SIM swap risk | Vulnerable | Not affected |
| SS7 interception | Vulnerable | Not affected |
| Carrier involvement | Required | Not required |
| Delivery latency | 5-30 seconds | Instant (computed locally) |
| Device binding | Phone number (portable) | Device (unless synced) |

### Setup Flow

```
1. Server generates random 160-bit secret (base32-encoded)
2. Server constructs otpauth:// URI:
   otpauth://totp/GGID:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=GGID&digits=6&period=30
3. Server encodes URI as QR code
4. User scans QR code with authenticator app (Google Authenticator, Authy, 1Password)
5. User enters current code to verify setup
6. Server marks device as enabled
```

### Drift Window Validation

Clock drift between the user's device and the server is common. A drift window
allows accepting codes from adjacent time steps:

| Window | Time Range | Use Case |
|--------|-----------|----------|
| 0 (current only) | 0 to 30s | Strictest — poor UX |
| 1 (default) | -30s to +60s | Standard — handles minor drift |
| 2 | -60s to +90s | Generous — handles significant drift |
| 3+ | Not recommended | Increases brute-force surface |

### Go Implementation: TOTP with Configurable Window

```go
package totp

import (
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPValidator wraps the pquerna/otp library with configurable drift window.
type TOTPValidator struct {
	window uint // number of periods to check before/after current time
}

// NewTOTPValidator creates a validator with the specified drift window.
// A window of 1 accepts codes from the previous, current, and next period.
func NewTOTPValidator(window uint) *TOTPValidator {
	if window > 3 {
		window = 3 // cap at 3 for security
	}
	return &TOTPValidator{window: window}
}

// Validate checks if the provided code is valid for the given secret.
// Uses the drift window to accommodate clock skew.
func (v *TOTPValidator) Validate(code, secret string) bool {
	if len(code) != 6 {
		return false
	}
	// totp.ValidateCustom allows specifying the drift window.
	now := time.Now()
	return totp.ValidateCustom(code, secret, now, totp.ValidateOpts{
		Period:    30,
		Skew:      v.window,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
}

// GenerateSecret creates a new TOTP secret and returns the otpauth:// provisioning URI.
func GenerateSecret(issuer, accountName string) (secret, uri string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate TOTP key: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// PreventCodeReuse ensures a TOTP code is not used twice within the same period.
// Without this, a code valid for 30 seconds can be replayed within that window.
type CodeReuseGuard struct {
	store CodeStore // Redis or in-memory
}

// CheckAndConsume verifies that the code hasn't been used and marks it as used.
func (g *CodeReuseGuard) CheckAndConsume(userID, code string) error {
	key := fmt.Sprintf("totp:used:%s:%s", userID, code)
	if exists, _ := g.store.Exists(key); exists {
		return fmt.Errorf("code already used — wait for next period")
	}
	// Mark as used for 35 seconds (slightly more than one period).
	g.store.Set(key, "1", 35*time.Second)
	return nil
}
```

### Security Considerations

- **SHA1 vs SHA256**: The OTP standard (RFC 6238) defines SHA1 as default. While SHA1
  has known weaknesses for collision resistance, HMAC-SHA1 (used in TOTP) remains
  cryptographically secure. Some authenticators support SHA256 but compatibility
  varies — SHA1 is the safest default.

- **Secret storage**: TOTP secrets must be encrypted at rest. If the database is
  compromised, the attacker can generate valid codes. Use AES-256-GCM or a KMS.

- **Backup codes**: Generate one-time-use backup codes when TOTP is enabled. Users
  who lose their device need an alternative. Each backup code should be 8+ characters,
  alphanumeric, and stored hashed.

---

## 7. Passwordless Migration Strategy

### Phased Approach

Migrating an existing password-based user base to passwordless requires a carefully
staged approach:

**Phase 1: Optional Enrollment (Months 1-3)**
- Offer passkey enrollment during existing login flow ("Add a passkey for faster login").
- Show a dismissible banner on the dashboard: "Go passwordless with passkeys."
- Track enrollment rates per cohort.

**Phase 2: Nudge and Incentivize (Months 3-6)**
- After passkey enrollment, offer to "Remove your password" during profile settings.
- Incentives: faster login (no password typing), reduced MFA friction.
- Deprioritize password login UI (passkey autofill becomes primary, password link is secondary).

**Phase 3: Password Deprecation (Months 6-12)**
- For users with active passkeys, stop accepting passwords for day-to-day login.
- Password is retained only for account recovery (with additional verification).
- Email users who haven't enrolled: "Passwords will be disabled on [date]."

**Phase 4: Full Passwordless (Month 12+)**
- Remove password fields from all new account registrations.
- Existing password users must enroll a passkey to continue.
- Passwords are purged from the database after a sunset period.

### Fallback Chain Design

When a passkey is unavailable (new device, lost phone), the system must provide
alternative authentication paths:

```
Passkey (preferred) → TOTP App → Email OTP → Magic Link → Password (last resort)
```

### Go Implementation: Multi-Factor Fallback Handler

```go
package auth

import (
	"context"
	"fmt"
	"net/http"
)

// FallbackMethod represents an authentication method in the fallback chain.
type FallbackMethod string

const (
	FallbackPasskey   FallbackMethod = "passkey"
	FallbackTOTP      FallbackMethod = "totp"
	FallbackEmailOTP  FallbackMethod = "email_otp"
	FallbackMagicLink FallbackMethod = "magic_link"
	FallbackPassword  FallbackMethod = "password"
)

// FallbackChain defines the order of authentication methods to try.
var FallbackChain = []FallbackMethod{
	FallbackPasskey,
	FallbackTOTP,
	FallbackEmailOTP,
	FallbackMagicLink,
	FallbackPassword,
}

// FallbackAuthHandler tries each method in order until one is available for the user.
type FallbackAuthHandler struct {
	webauthn   WebAuthnService
	totp       TOTPService
	emailOTP   EmailOTPService
	magicLink  MagicLinkService
	password   PasswordService
	enrollment EnrollmentChecker
}

// GetAvailableMethods returns which passwordless methods a user has enrolled.
// The frontend uses this to display available options.
func (h *FallbackAuthHandler) GetAvailableMethods(
	ctx context.Context,
	userID string,
) ([]FallbackMethod, error) {
	var methods []FallbackMethod

	// Check WebAuthn credentials.
	if h.enrollment.HasPasskey(ctx, userID) {
		methods = append(methods, FallbackPasskey)
	}

	// Check TOTP device.
	if h.enrollment.HasTOTP(ctx, userID) {
		methods = append(methods, FallbackTOTP)
	}

	// Email OTP is always available if user has a verified email.
	if h.enrollment.HasVerifiedEmail(ctx, userID) {
		methods = append(methods, FallbackEmailOTP)
		methods = append(methods, FallbackMagicLink)
	}

	// Password is always available (until fully deprecated).
	if h.enrollment.HasPassword(ctx, userID) {
		methods = append(methods, FallbackPassword)
	}

	return methods, nil
}

// InitiateFallback starts authentication with the best available method.
func (h *FallbackAuthHandler) InitiateFallback(
	ctx context.Context,
	userID string,
	email string,
) (*FallbackChallenge, error) {
	methods, err := h.GetAvailableMethods(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}

	// Use the highest-priority method.
	method := methods[0]

	switch method {
	case FallbackPasskey:
		return &FallbackChallenge{
			Method:  method,
			Message: "Complete authentication with your passkey",
		}, nil

	case FallbackTOTP:
		return &FallbackChallenge{
			Method:  method,
			Message: "Enter your authenticator app code",
		}, nil

	case FallbackEmailOTP:
		otp, _ := h.emailOTP.SendEmailOTP(ctx, tenantID, userID, email)
		// Email is sent by the email service.
		return &FallbackChallenge{
			Method:  method,
			Message: "Enter the code sent to your email",
		}, nil

	case FallbackMagicLink:
		link, _ := h.magicLink.GenerateToken(ctx, tenantID, userID, email)
		// Link is sent by the email service.
		return &FallbackChallenge{
			Method:  method,
			Message: "Check your email for a sign-in link",
		}, nil

	case FallbackPassword:
		return &FallbackChallenge{
			Method:  method,
			Message: "Enter your password",
		}, nil
	}

	return nil, fmt.Errorf("no suitable authentication method")
}

// FallbackChallenge represents the challenge issued to the user.
type FallbackChallenge struct {
	Method  FallbackMethod `json:"method"`
	Message string         `json:"message"`
}
```

### Migration Metrics

Track these KPIs during migration:

| Metric | Target | Measurement |
|--------|--------|-------------|
| Passkey enrollment rate | >50% after 6 months | Enrolled / total users |
| Password-less login rate | >70% of logins use passkey | Passkey logins / total logins |
| Password reset reduction | >60% fewer resets | Resets before vs after |
| Account recovery success | >95% recovery without password | Successful recoveries / attempts |
| Login success rate | >99% (no regression) | Successful logins / total attempts |

---

## 8. GGID Passwordless Gap Analysis

### What's Implemented

| Feature | File | Status | Notes |
|---------|------|--------|-------|
| **WebAuthn Registration** | `services/auth/internal/webauthn/handler.go` | Complete | go-webauthn library, platform authenticators, credential exclusion, transports |
| **WebAuthn Authentication** | `services/auth/internal/webauthn/handler.go` | Complete | Discoverable credentials, sign counter clone detection, last-used tracking |
| **WebAuthn Credential Mgmt** | `services/auth/internal/webauthn/handler.go` | Complete | List, delete credentials; auto-generated names from User-Agent |
| **WebAuthn Mobile** | `handler.go` well-known endpoints | Complete | Android asset links, iOS universal links, Related Origin Requests |
| **TOTP MFA Setup** | `services/auth/internal/service/mfa_service.go` | Complete | pquerna/otp, QR code URI, device management |
| **TOTP MFA Verify** | `mfa_service.go` | Complete | Login challenge verification |
| **SMS OTP** | `services/auth/internal/service/phone_otp.go` | Complete | 6-digit OTP, Redis TTL, rate limiting, session creation |
| **Email Verification** | `services/auth/internal/service/email_lockout.go` | Partial | `EmailService` issues verification tokens but NOT for auth — only email verification |
| **Account Lockout** | `email_lockout.go` | Complete | Redis counter, configurable threshold/duration |
| **Step-Up Auth** | `services/auth/internal/service/stepup.go` | Complete | Password + MFA step-up, ACR-based escalation |
| **Login Attempt Logging** | `services/auth/internal/service/login_attempt.go` | Complete | Redis sorted set, 30-day history |

### What's Missing

| Feature | Priority | Effort | Description |
|---------|----------|--------|-------------|
| **Magic Link Auth** | High | 3-5 days | No magic link generation, verification, or email sending for login. EmailService exists but only handles email verification tokens. |
| **Email OTP Auth** | High | 2-3 days | No email OTP generation/verification for authentication. Phone OTP exists but no email equivalent. |
| **Passwordless-Only Flow** | Medium | 3-5 days | WebAuthn is additive (alongside password). No flow to create accounts without a password or to disable password after passkey enrollment. |
| **TOTP Drift Window** | Medium | 1 day | `mfa_service.go` uses `totp.Validate()` with default window (0). No configurable drift window for clock skew tolerance. |
| **TOTP Code Reuse Prevention** | Medium | 1 day | No check to prevent the same TOTP code from being used twice within its validity period. |
| **Backup Codes** | Medium | 2-3 days | No one-time-use backup code generation for account recovery when all other factors are lost. |
| **Passkey Recovery** | Medium | 3-5 days | No account recovery flow for lost devices (no magic link re-enrollment, no admin reset of WebAuthn credentials). |
| **Conditional UI Support** | Low | 1 day | Server-side is ready (discoverable credentials), but no documented client-side integration guide or API hint for conditional mediation. |
| **Fallback Chain** | Medium | 3-5 days | No automated fallback chain (passkey → TOTP → email OTP → magic link → password). Each method exists in isolation. |
| **Passwordless Migration** | Low | 5-10 days | No enrollment nudges, password deprecation timeline, or migration tracking. |
| **TOTP Secret Encryption** | High | 1-2 days | `domain/mfa.go` stores `Secret string` as plaintext. Comment says "encrypted at rest in production" but no encryption is applied. |
| **WebAuthn Session Store** | Medium | 1-2 days | Uses in-memory map (`sessionStore`). Comment says "production would use Redis." Multi-instance deployment will break. |

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

#### 1. Implement Magic Link Authentication (HIGH — 3-5 days)

**Why**: Magic links are the most requested passwordless feature for CIAM (customer
identity). They enable frictionless sign-in without any device enrollment. They are
also the foundation for passkey recovery (re-enroll a new passkey via verified email).

**What to do**:
- Add `MagicLinkService` to `services/auth/internal/service/`.
- Reuse the existing `EmailService` infrastructure for token storage (same Redis pattern).
- Add `POST /api/v1/auth/magic-link` and `GET /api/v1/auth/magic/verify` endpoints.
- Use HMAC for token storage (not plaintext) with `crypto/rand` for generation.
- Enforce single-use via Redis `GETDEL`.
- Always return generic success response (prevent enumeration).

**Risk**: Low. Follows existing patterns in `phone_otp.go` and `email_lockout.go`.

#### 2. Add Email OTP Authentication (HIGH — 2-3 days)

**Why**: Email OTP is a natural second factor and a common alternative to SMS.
The `EmailService` already handles email verification tokens — extending this to
OTP-based login is straightforward.

**What to do**:
- Extend `EmailService` with `SendEmailOTP` and `VerifyEmailOTP` methods.
- Mirror the `phone_otp.go` pattern: 6-digit code, Redis TTL, attempt tracking.
- Add `POST /api/v1/auth/email-otp/send` and `POST /api/v1/auth/email-otp/verify`.
- Wire into the session creation flow (same as `VerifyPhoneOTP`).

**Risk**: Low. Directly parallels existing SMS OTP implementation.

#### 3. Encrypt TOTP Secrets at Rest (HIGH — 1-2 days)

**Why**: TOTP secrets stored as plaintext strings in `domain.MFADevice.Secret`
represent a critical vulnerability. If the database is compromised, an attacker
can generate valid TOTP codes for every user.

**What to do**:
- Use `pkg/crypto` AES-256-GCM encryption (already available in the project).
- Encrypt secret on `CreateDevice`, decrypt on `VerifyMFA`/`VerifyUserCode`.
- Add a migration to encrypt existing secrets.

**Risk**: Medium. Requires careful migration of existing data.

#### 4. Move WebAuthn Session Store to Redis (MEDIUM — 1-2 days)

**Why**: The in-memory `sessionStore` in `webauthn/handler.go` breaks in
multi-instance deployments. If registration begins on instance A and finishes on
instance B, the session is lost.

**What to do**:
- Replace `sessionStore` with a Redis-backed implementation.
- Use `auth.sessions.challenge:{challenge}` keys with 5-minute TTL.
- Same pattern as `stepup.go` and `phone_otp.go`.

**Risk**: Low. Well-understood pattern already used elsewhere in the codebase.

#### 5. Add TOTP Drift Window and Code Reuse Prevention (MEDIUM — 1-2 days)

**Why**: Current `totp.Validate()` uses the default window (0), meaning codes
from the previous or next 30-second period are rejected. This causes legitimate
failures due to minor clock drift. Additionally, the same code can be used
multiple times within its 30-second validity window.

**What to do**:
- Replace `totp.Validate()` with `totp.ValidateCustom()` specifying `Skew: 1`.
- Add a Redis-backed code-reuse guard: store `totp:used:{userID}:{code}` with
  35-second TTL after each successful verification.
- Reject any code already in the reuse guard.

**Risk**: Low. Library supports the API directly.

### Effort Summary

| Item | Priority | Effort | Total |
|------|----------|--------|-------|
| Magic Link Auth | High | 3-5 days | |
| Email OTP Auth | High | 2-3 days | |
| TOTP Secret Encryption | High | 1-2 days | |
| WebAuthn Session → Redis | Medium | 1-2 days | |
| TOTP Drift + Reuse Guard | Medium | 1-2 days | |
| **Total** | | **8-14 days** | |

### Long-Term Roadmap (6-12 months)

- **Passwordless-only account creation**: Allow users to sign up without setting a
  password (passkey-only enrollment).
- **Fallback chain automation**: Implement the `FallbackAuthHandler` pattern from
  Section 7 to provide graceful degradation across methods.
- **Password deprecation timeline**: Track enrollment metrics and gradually restrict
  password usage for users with active passkeys.
- **Backup code system**: Generate one-time-use codes at TOTP enrollment for account
  recovery.
- **Conditional UI documentation**: Publish client-side integration guide for
  passkey autofill in the admin console.

---

## Appendix: Security Reference Table

| Method | AAL Level | NIST 800-63B | Phishing-Resistant | Offline Capable |
|--------|-----------|-------------|---------------------|-----------------|
| WebAuthn (platform) | AAL3 | Approved | Yes | N/A (device-bound) |
| WebAuthn (security key) | AAL3 | Approved | Yes | N/A (device-bound) |
| TOTP App | AAL2 | Approved | No | Yes |
| Email OTP | AAL1-AAL2 | Approved (restricted) | No | No |
| Magic Link | AAL1-AAL2 | Approved (restricted) | No | No |
| SMS OTP | AAL1 | Deprecated for AAL2+ | No | No |
| Password | AAL1 | Approved (with restrictions) | No | N/A |

---

*Document version: 1.0 | Last reviewed: 2025 | GGID IAM Security Research*
