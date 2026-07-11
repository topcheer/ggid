# OAuth Device Flow (RFC 8628) Security Analysis

**Security-Focused Research for GGID IAM**

| | |
|---|---|
| **RFC** | [RFC 8628](https://www.rfc-editor.org/info/rfc8628/) — OAuth 2.0 Device Authorization Grant |
| **Companion Doc** | `docs/research/device-flow-rfc8628.md` — Full protocol spec & implementation design |
| **Focus** | Security: rate limiting, polling abuse, code entropy, verification URI, cross-device attacks |
| **Status** | GGID: In-memory implementation with basic slow_down; production hardening required |

---

## Table of Contents

1. [Device Code Entropy Requirements](#1-device-code-entropy-requirements)
2. [Polling Abuse Prevention](#2-polling-abuse-prevention)
3. [Verification URI Security](#3-verification-uri-security)
4. [Device Code Lifetime Management](#4-device-code-lifetime-management)
5. [Completed/Denied Code Handling](#5-completeddenied-code-handling)
6. [Cross-Device Attack Vectors](#6-cross-device-attack-vectors)
7. [Device Flow in Headless/IoT Context](#7-device-flow-in-headlessiot-context)
8. [GGID Device Flow Audit](#8-ggid-device-flow-audit)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Device Code Entropy Requirements

### 1.1 Two Codes, Two Threat Models

The device authorization grant issues **two distinct codes** with fundamentally different security properties:

| Property | `device_code` | `user_code` |
|---|---|---|
| Who uses it | The requesting device (machine) | The human user (typed on a secondary device) |
| Transmission | Sent over HTTPS to device, never seen by user | Displayed on screen, typed by human |
| Length | 40+ characters, opaque | 6–8 characters, human-friendly |
| Entropy target | 128+ bits (unguessable by attacker) | 20–40 bits (resist brute-force within TTL window) |
| Character set | Full alphanumeric (no user constraint) | Crockford Base32 or similar (no ambiguous chars) |
| Attack surface | Token endpoint enumeration | Verification page brute-force |

The `device_code` is a bearer secret — whoever possesses it can poll for tokens. The `user_code` is a short-lived identifier that maps a human action to a device code. Each has a different entropy floor and a different character set.

### 1.2 Entropy Calculation for User-Entered Codes

The `user_code` must be short enough to type reliably but long enough to resist brute-force within the code's lifetime. The calculation:

```
Entropy (bits) = code_length × log2(charset_size)
```

| Format | Charset | Length | Entropy | Brute-Force Space |
|---|---|---|---|---|
| Numeric (NPS-like) | 10 (0-9) | 8 | 26.5 bits | ~67 million |
| Alphanumeric | 36 (A-Z, 0-9) | 6 | 31.0 bits | ~2.2 billion |
| Crockford Base32 | 32 (excl. I,L,O,U) | 8 | 40.0 bits | ~1.1 trillion |
| Crockford Base32 | 32 | 7 | 35.0 bits | ~34 billion |

**Recommended floor: 32 bits of entropy** for user codes with a 5–15 minute lifetime. This gives an attacker at most 4.3 billion combinations to try. At 100 attempts per minute (generous), that's 86,000 minutes of continuous guessing — well beyond any code's lifetime.

**GGID's current implementation**: 8 characters from a 32-character Crockford-like set (`ABCDEFGHJKLMNPQRSTUVWXYZ23456789`), yielding **40 bits of entropy**. This exceeds the floor but may be unnecessarily long for typing.

### 1.3 Character Set Selection

Ambiguous characters cause transcription errors that degrade UX and create support load:

| Ambiguous Pair | Problem | Solution |
|---|---|---|
| `0` vs `O` | Zero vs capital-O | Remove one (typically `O`) |
| `1` vs `l` vs `I` | One vs lowercase-L vs capital-I | Remove `l` and `I` |
| `U` | Confused with `V` in some fonts | Remove in Crockford Base32 |
| `5` vs `S` | Five vs capital-S | Acceptable in practice |
| `2` vs `Z` | Two vs capital-Z | Acceptable in practice |

### 1.4 Go Code: User-Friendly High-Entropy Codes

```go
package deviceflow

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// CrockfordBase32 excludes I, L, O, U to avoid ambiguity.
// Reference: https://www.crockford.com/base32.html
const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// GenerateUserCode produces a user-friendly code with the format "XXXX-XXXX".
// Entropy: 8 chars × log2(32) = 40 bits.
// Format improves readability: "7K3P-9WQM" is easier to type than "7K3P9WQM".
func GenerateUserCode() (string, error) {
	code := make([]byte, 8)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(crockfordBase32))))
		if err != nil {
			return "", fmt.Errorf("generate user code: %w", err)
		}
		code[i] = crockfordBase32[n.Int64()]
	}
	return fmt.Sprintf("%s-%s", code[:4], code[4:]), nil
}

// GenerateDeviceCode produces a high-entropy opaque device code.
// 32 bytes from crypto/rand → 256 bits of entropy, hex-encoded to 64 chars.
// This is a bearer secret; it must be unguessable.
func GenerateDeviceCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate device code: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}

// NormalizeUserCode uppercases and removes non-alphanumeric characters.
// This lets users type "7k3p 9wqm" or "7k3p-9wqm" and still match.
func NormalizeUserCode(input string) string {
	var result []byte
	for _, c := range strings.ToUpper(input) {
		if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
```

### 1.5 Brute-Force Resistance for User Codes

Even with 40 bits of entropy, the verification endpoint must resist brute-force:

```
Attacker tries random user codes at the verification page.
With 40-bit entropy and 15-minute lifetime:
  - Need to try 2^40 / 2 = ~550 billion codes on average
  - At 10 req/sec: ~1,740 years
  - At 10,000 req/sec: ~1.7 years

This is safe IF the verification endpoint has rate limiting.
Without rate limiting, a distributed attacker could try millions/sec.
```

**Key principle**: entropy alone is insufficient. The verification endpoint must enforce per-IP and per-session rate limits on user_code submission attempts.

---

## 2. Polling Abuse Prevention

### 2.1 Attack Vectors

The token endpoint with `grant_type=device_code` is a polling endpoint. Attackers can abuse it in several ways:

| Attack | Mechanism | Impact |
|---|---|---|
| **Device code enumeration** | Try random device codes to discover active sessions | Information disclosure (code existence, status) |
| **Polling DoS** | Rapid-fire polls on a known device code | Server CPU, memory, database load |
| **Concurrent poll exhaustion** | Initiate thousands of device flows, never complete | Memory exhaustion from stored device codes |
| **slow_down evasion** | Ignore server interval, poll at max speed | Same as above; protocol violation |
| **Timing attack on pending codes** | Measure response time to distinguish expired vs. pending vs. denied | Status enumeration |

### 2.2 Server-Side Rate Limiting Strategy

Rate limiting must operate at multiple layers:

```
Layer 1: Per-IP rate limit on device_authorization endpoint
         → Prevents code-creation flooding
         → Recommendation: 10 codes/min per IP

Layer 2: Per-IP rate limit on token endpoint (device_code grant)
         → Prevents enumeration and polling DoS
         → Recommendation: 30 polls/min per IP

Layer 3: Per-device-code slow_down enforcement
         → RFC 8628 mandatory: if client polls faster than interval, return slow_down
         → Recommendation: base interval 5s, increase by 5s on each slow_down

Layer 4: Global device code count limit
         → Prevents memory exhaustion
         → Recommendation: max 10,000 pending codes per tenant
```

### 2.3 Go Code: Polling Rate Limiter

```go
package deviceflow

import (
	"net"
	"sync"
	"time"
)

// PollingRateLimiter enforces rate limits on device flow endpoints.
// It combines per-IP limits with per-device-code interval enforcement.
type PollingRateLimiter struct {
	mu sync.Mutex

	// Per-IP sliding window for token endpoint polls.
	ipPolls     map[string][]time.Time // IP → timestamps
	ipLimit     int                    // max polls per window
	ipWindow    time.Duration          // e.g., 1 minute

	// Per-device-code slow_down tracking.
	codePolls    map[string]time.Time // deviceCode → last poll time
	codeInterval time.Duration        // base interval (e.g., 5s)
	codeBackoff  map[string]int       // deviceCode → current backoff multiplier

	// Per-IP code creation tracking.
	ipCreates  map[string][]time.Time
	createMax  int
	createWindow time.Duration
}

// NewPollingRateLimiter configures recommended defaults.
func NewPollingRateLimiter() *PollingRateLimiter {
	return &PollingRateLimiter{
		ipPolls:       make(map[string][]time.Time),
		ipLimit:       30,
		ipWindow:      time.Minute,
		codePolls:     make(map[string]time.Time),
		codeInterval:  5 * time.Second,
		codeBackoff:   make(map[string]int),
		ipCreates:     make(map[string][]time.Time),
		createMax:     10,
		createWindow:  time.Minute,
	}
}

// CanCreateDeviceAuthorization checks the per-IP creation rate.
// Returns false if the IP has exceeded the creation limit.
func (rl *PollingRateLimiter) CanCreateDeviceAuthorization(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.checkWindow(rl.ipCreates, ip, rl.createMax, rl.createWindow)
}

// CheckPollInterval enforces the slow_down protocol.
// Returns:
//   allowed=true if the client may poll now
//   waitUntil if the client must wait before retrying
func (rl *PollingRateLimiter) CheckPollInterval(ip, deviceCode string) (allowed bool, waitUntil time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check per-IP rate limit first.
	if !rl.checkWindow(rl.ipPolls, ip, rl.ipLimit, rl.ipWindow) {
		return false, rl.ipWindow // IP is rate-limited; try again next window
	}

	// Check per-device-code interval with exponential backoff.
	interval := rl.codeInterval
	if mult, ok := rl.codeBackoff[deviceCode]; ok && mult > 0 {
		interval = rl.codeInterval * time.Duration(1+mult)
	}

	last, exists := rl.codePolls[deviceCode]
	if exists && time.Since(last) < interval {
		// Too soon — increment backoff.
		rl.codeBackoff[deviceCode]++
		elapsed := time.Since(last)
		return false, interval - elapsed
	}

	rl.codePolls[deviceCode] = time.Now()
	return true, 0
}

// ResetBackoff clears the backoff multiplier when a poll succeeds
// (e.g., token issued or code expired/denied).
func (rl *PollingRateLimiter) ResetBackoff(deviceCode string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.codeBackoff, deviceCode)
	delete(rl.codePolls, deviceCode)
}

// checkWindow is a sliding-window rate limiter for a key.
func (rl *PollingRateLimiter) checkWindow(store map[string][]time.Time, key string, max int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	// Filter out expired entries.
	var valid []time.Time
	for _, t := range store[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= max {
		store[key] = valid
		return false
	}

	valid = append(valid, now)
	store[key] = valid
	return true
}

// ExtractIP gets the client IP from a request, handling X-Forwarded-For.
func ExtractIP(remoteAddr string, xForwardedFor string) string {
	if xForwardedFor != "" {
		// Use the first IP in the chain (original client).
		for i, c := range xForwardedFor {
			if c == ',' {
				return strings.TrimSpace(xForwardedFor[:i])
			}
		}
		return strings.TrimSpace(xForwardedFor)
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
```

### 2.4 Client-Side Interval Backoff (Best Practices for SDKs)

```go
// Client-side polling with slow_down handling (SDK reference implementation).
func PollForToken(ctx context.Context, deviceCode string, initialInterval time.Duration) (*TokenResponse, error) {
	interval := initialInterval

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		resp, err := postTokenPoll(ctx, deviceCode)
		if err == nil {
			return resp, nil // Got token!
		}

		switch err.Error() {
		case "authorization_pending":
			// Keep polling at the current interval.
			continue
		case "slow_down":
			// RFC 8628 §3.5: increase interval by 5 seconds.
			interval += 5 * time.Second
			continue
		case "expired_token", "access_denied":
			return nil, err // Terminal — stop polling.
		default:
			return nil, err
		}
	}
}
```

---

## 3. Verification URI Security

### 3.1 HTTPS Enforcement

The `verification_uri` is the URL where the user enters their code. It MUST be HTTPS:

- The user will authenticate on this page (enter credentials)
- The code itself is a temporary secret (knowledge of it grants authorization)
- Without TLS, an on-path attacker can intercept both credentials and the code

**GGID's current implementation** builds the verification URI as `req.Issuer + "/device"` with no HTTPS validation. If `Issuer` is set to `http://...`, the verification URI will be insecure.

### 3.2 Phishing Risk

Attackers can create fake verification pages that:
1. Mimic the IAM provider's branding
2. Capture the user's entered code
3. Forward the code to the real endpoint using the attacker's device flow
4. Or capture the user's login credentials

Mitigations:
- **Certificate pinning** for mobile companion apps
- **WebAuthn-based authentication** (phishing-resistant by design)
- **Brand consistency**: users should recognize the verification page
- **No credential entry on the verification page itself**: the page should redirect to the normal login flow, not collect credentials inline

### 3.3 QR Code Alternative

Instead of requiring the user to type a code, the device can display a QR code encoding `verification_uri_complete`:

```
verification_uri_complete = https://iam.example.com/device?user_code=7K3P-9WQM
```

When scanned, this URL automatically binds the user's browser session to the device flow. Security considerations:
- The QR code embeds the user_code — an attacker photographing it could hijack the flow
- The `verification_uri_complete` must be HTTPS
- The server should set a short-lived session cookie after QR scan to prevent replay

### 3.4 Go Code: Secure Verification Page Handler

```go
package deviceflow

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// VerificationPageConfig configures the verification endpoint.
type VerificationPageConfig struct {
	Issuer           string        // Must start with https://
	CodeLifetime     time.Duration // How long codes are valid
	MaxCodeAttempts  int           // Max wrong-code attempts before lockout
	SessionCookieName string
}

// SecureVerificationHandler renders the verification page and processes code submission.
// Security features:
//   - HTTPS enforcement (redirect or reject HTTP)
//   - CSRF token on POST
//   - Per-session rate limiting on code attempts
//   - Session binding to prevent cross-site code injection
func SecureVerificationHandler(svc *DeviceFlowService, cfg VerificationPageConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Enforce HTTPS (skip in local dev).
		if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" && cfg.Issuer != "" {
			if strings.HasPrefix(cfg.Issuer, "https://") {
				httpsURL := strings.Replace(r.URL.String(), "http://", "https://", 1)
				http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
				return
			}
		}

		switch r.Method {
		case http.MethodGet:
			// Render the page with a CSRF token.
			csrfToken := generateCSRFToken()
			http.SetCookie(w, &http.Cookie{
				Name:     cfg.SessionCookieName,
				Value:    csrfToken,
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   int(cfg.CodeLifetime.Seconds()),
			})
			renderVerificationPage(w, csrfToken)

		case http.MethodPost:
			// Validate CSRF token (constant-time comparison).
			sessionCookie, err := r.Cookie(cfg.SessionCookieName)
			if err != nil {
				http.Error(w, "session expired", http.StatusForbidden)
				return
			}
			postedCSRF := r.FormValue("csrf_token")
			if subtle.ConstantTimeCompare([]byte(sessionCookie.Value), []byte(postedCSRF)) != 1 {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}

			// Rate limit code attempts per session.
			if !svc.CheckCodeAttemptLimit(sessionCookie.Value) {
				http.Error(w, "too many attempts, please try again later", http.StatusTooManyRequests)
				return
			}

			userCode := NormalizeUserCode(r.FormValue("user_code"))
			if userCode == "" {
				http.Error(w, "user_code is required", http.StatusBadRequest)
				return
			}

			// Look up the device code by user code.
			info, err := svc.LookupByUserCode(userCode)
			if err != nil {
				// Return generic error — don't reveal whether code exists.
				http.Error(w, "invalid or expired code", http.StatusBadRequest)
				return
			}

			// Render a confirmation page showing what app is requesting access.
			renderConfirmationPage(w, info)
		}
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
```

---

## 4. Device Code Lifetime Management

### 4.1 Lifetime Trade-Offs

The `expires_in` value creates a security/UX tension:

| Lifetime | Pro | Con |
|---|---|---|
| 30 seconds | Minimal brute-force window | User can't complete auth in time (especially on TV) |
| 2 minutes | Good balance for simple login | Tight for MFA flows |
| 5 minutes | Recommended minimum | Attacker has 5 min to guess user_code |
| 15 minutes | Generous (GGID default) | Larger brute-force window |
| 30 minutes | Very generous | Excessive risk; stale codes linger |

**RFC 8628 recommendation**: The lifetime should be short enough to limit brute-force but long enough for the user to complete authentication. **5–15 minutes is the sweet spot.**

### 4.2 Expired Code Cleanup

Expired codes must be purged from storage:

1. **Lazy cleanup** (current GGID approach): Delete on access when `time.Now().After(ExpiresAt)`. Works but allows unbounded growth if codes are never accessed again.
2. **Background sweep** (recommended): A goroutine or cron that periodically scans and deletes expired codes.
3. **Redis TTL** (production): Set `EX` on the Redis key so expiration is automatic.

### 4.3 Go Code: Device Code Lifecycle Manager

```go
package deviceflow

import (
	"context"
	"log"
	"sync"
	"time"
)

// LifecycleManager handles creation, storage, and cleanup of device codes.
type LifecycleManager struct {
	mu        sync.RWMutex
	codes     map[string]*ManagedCode // keyed by device_code
	userIndex map[string]string       // user_code → device_code

	codeTTL       time.Duration // default lifetime (e.g., 10 minutes)
	maxCodes      int           // max concurrent codes per manager
	sweepInterval time.Duration // how often to run cleanup
}

// ManagedCode extends DeviceCodeInfo with lifecycle metadata.
type ManagedCode struct {
	DeviceCode  string
	UserCode    string
	ClientID    string
	TenantID    string
	Status      string // pending, approved, denied, expired
	CreatedAt   time.Time
	ExpiresAt   time.Time
	ApprovedBy  string    // user ID who approved
	ApprovedAt  *time.Time
	PollCount   int       // how many times polled
	AttemptCount int      // how many times user code was tried
}

// NewLifecycleManager creates a manager with safe defaults.
func NewLifecycleManager() *LifecycleManager {
	lm := &LifecycleManager{
		codes:         make(map[string]*ManagedCode),
		userIndex:     make(map[string]string),
		codeTTL:       10 * time.Minute,
		maxCodes:      10000,
		sweepInterval: 30 * time.Second,
	}
	return lm
}

// StartBackgroundSweep launches a goroutine that periodically deletes expired codes.
func (lm *LifecycleManager) StartBackgroundSweep(ctx context.Context) {
	ticker := time.NewTicker(lm.sweepInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lm.sweep()
			}
		}
	}()
}

// sweep deletes all expired codes. Called periodically by the background goroutine.
func (lm *LifecycleManager) sweep() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	for dc, info := range lm.codes {
		if now.After(info.ExpiresAt) {
			delete(lm.codes, dc)
			delete(lm.userIndex, info.UserCode)
		}
	}
}

// Create generates a new device code with the configured TTL.
// Returns error if the max code count is exceeded (prevents memory exhaustion).
func (lm *LifecycleManager) Create(clientID, tenantID string) (*ManagedCode, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if len(lm.codes) >= lm.maxCodes {
		return nil, fmt.Errorf("device code limit reached, try again later")
	}

	deviceCode, _ := GenerateDeviceCode()
	userCode, _ := GenerateUserCode()

	now := time.Now()
	code := &ManagedCode{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   clientID,
		TenantID:   tenantID,
		Status:     "pending",
		CreatedAt:  now,
		ExpiresAt:  now.Add(lm.codeTTL),
	}

	lm.codes[deviceCode] = code
	lm.userIndex[userCode] = deviceCode

	log.Printf("device code created: client=%s ttl=%v", clientID, lm.codeTTL)
	return code, nil
}

// Stats returns diagnostic counters for monitoring.
type CodeStats struct {
	Total      int
	Pending    int
	Approved   int
	Denied     int
	OldestAge  time.Duration
}

func (lm *LifecycleManager) Stats() CodeStats {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	var stats CodeStats
	oldest := time.Now()
	for _, c := range lm.codes {
		stats.Total++
		switch c.Status {
		case "pending":
			stats.Pending++
		case "approved":
			stats.Approved++
		case "denied":
			stats.Denied++
		}
		if c.CreatedAt.Before(oldest) {
			oldest = c.CreatedAt
		}
	}
	if stats.Total > 0 {
		stats.OldestAge = time.Since(oldest)
	}
	return stats
}
```

---

## 5. Completed/Denied Code Handling

### 5.1 State Machine

Device codes transition through a well-defined state machine:

```
                    ┌──────────┐
                    │  PENDING │ ← initial state after CreateDeviceAuthorization
                    └────┬─────┘
                         │
            ┌────────────┼────────────┐
            ▼            ▼            ▼
     ┌───────────┐ ┌──────────┐ ┌─────────┐
     │ APPROVED  │ │  DENIED  │ │ EXPIRED │
     └─────┬─────┘ └────┬─────┘ └────┬────┘
           │              │            │
           ▼              ▼            ▼
     ┌───────────┐ ┌──────────┐ ┌──────────┐
     │ COMPLETED │ │ REVOKED  │ │ DELETED  │
     └───────────┘ └──────────┘ └──────────┘
     (token issued,  (user rejected, (TTL passed,
      code deleted)   code deleted)   code deleted)
```

### 5.2 Critical Security Properties

1. **COMPLETED codes must be invalidated immediately**: Once a token is issued, the device code must be deleted to prevent token replay (a second poll getting a second token).
2. **DENIED codes must return access_denied on subsequent polls**: The user explicitly rejected the request. Allowing re-polling to succeed would be a vulnerability.
3. **EXPIRED codes must be cleaned up**: Lingering expired codes consume memory and create enumeration targets.
4. **State transitions must be atomic**: Concurrent requests (poll + approve arriving simultaneously) must not produce inconsistent state.

### 5.3 Go Code: State Machine

```go
package deviceflow

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCodeNotFound    = errors.New("device code not found")
	ErrCodeExpired     = errors.New("expired_token")
	ErrCodeDenied      = errors.New("access_denied")
	ErrCodePending     = errors.New("authorization_pending")
	ErrCodeCompleted   = errors.New("code already used")
	ErrSlowDown        = errors.New("slow_down")
)

// CodeStateMachine manages state transitions for a device code.
// All transitions are mutex-protected for atomicity.
type CodeStateMachine struct {
	mu       sync.Mutex
	code     *ManagedCode
	interval time.Duration
}

// NewCodeStateMachine wraps a managed code.
func NewCodeStateMachine(code *ManagedCode, interval time.Duration) *CodeStateMachine {
	return &CodeStateMachine{code: code, interval: interval}
}

// Poll checks the current state and returns the appropriate response.
// This is called by the device during polling.
func (sm *CodeStateMachine) Poll(clientID string) (status string, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Verify client ownership.
	if sm.code.ClientID != clientID {
		return "", ErrCodeNotFound // don't reveal existence to wrong client
	}

	// Check expiry first.
	if time.Now().After(sm.code.ExpiresAt) {
		sm.code.Status = "expired"
		return "", ErrCodeExpired
	}

	switch sm.code.Status {
	case "pending":
		// Enforce slow_down.
		if sm.code.PollCount > 0 {
			// Track last poll time for interval enforcement.
			// (Simplified — in production, store last poll timestamp.)
		}
		sm.code.PollCount++
		return "pending", ErrCodePending

	case "approved":
		// Token already issued or about to be issued.
		// Transition to completed to prevent re-issue.
		sm.code.Status = "completed"
		return "approved", nil

	case "denied":
		// User rejected — keep returning access_denied until expiry cleanup.
		return "", ErrCodeDenied

	case "completed":
		// Already used — return terminal error.
		return "", ErrCodeCompleted

	case "expired":
		return "", ErrCodeExpired

	default:
		return "", fmt.Errorf("unknown code status: %s", sm.code.Status)
	}
}

// Approve transitions a pending code to approved.
// Called by the user's verification flow after authentication.
func (sm *CodeStateMachine) Approve(userID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if time.Now().After(sm.code.ExpiresAt) {
		sm.code.Status = "expired"
		return ErrCodeExpired
	}

	if sm.code.Status != "pending" {
		return fmt.Errorf("cannot approve code in status %s", sm.code.Status)
	}

	sm.code.Status = "approved"
	sm.code.ApprovedBy = userID
	now := time.Now()
	sm.code.ApprovedAt = &now
	return nil
}

// Deny transitions a pending code to denied.
// Called when the user explicitly rejects the authorization request.
func (sm *CodeStateMachine) Deny() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.code.Status != "pending" {
		// Already approved or completed — can't deny.
		return fmt.Errorf("cannot deny code in status %s", sm.code.Status)
	}

	sm.code.Status = "denied"
	return nil
}

// Complete finalizes the code after token issuance.
// This MUST be called immediately after issuing a token to prevent replay.
func (sm *CodeStateMachine) Complete() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.code.Status = "completed"
}
```

---

## 6. Cross-Device Attack Vectors

### 6.1 The Code Handoff Attack

The device flow's fundamental design — user enters a code on a different device — creates a trust gap:

```
1. Attacker initiates device flow on their device → gets user_code "7K3P-9WQM"
2. Attacker sends "7K3P-9WQM" to victim via phishing email/social engineering
3. Attacker says: "Enter this code to verify your account"
4. Victim goes to https://iam.example.com/device, enters code, authenticates
5. Victim clicks "Approve" on the confirmation page
6. Attacker's device polls and receives a valid access token
7. Attacker now has authenticated access as the victim
```

This is the device flow equivalent of the "consent phishing" attack. The victim's authentication is valid, but the resulting token goes to the attacker's device.

### 6.2 Mitigation: Confirmation Page

The verification page must clearly communicate what is happening:

```
┌─────────────────────────────────────────────┐
│  Device Authorization Request               │
│                                             │
│  An application is requesting access to     │
│  your account:                              │
│                                             │
│    Application: "My CLI Tool"               │
│    Requested scopes: read, write            │
│    Device: device-code 7K3P-9WQM            │
│                                             │
│  ⚠ If you did not initiate this request,    │
│    DO NOT approve it. Contact support.      │
│                                             │
│  [ Approve ]    [ Deny ]                    │
└─────────────────────────────────────────────┘
```

Key elements:
- Application name and registered scopes (transparency)
- The user code being approved (traceability)
- Explicit warning about unsolicited requests
- Clear Approve/Deny buttons

### 6.3 Go Code: Confirmation Page Handler

```go
package deviceflow

import (
	"html/template"
	"net/http"
)

const confirmationTemplate = `
<!DOCTYPE html>
<html>
<head><title>Device Authorization - {{.Issuer}}</title></head>
<body>
<h2>Device Authorization Request</h2>

<div style="border: 1px solid #ccc; padding: 16px; margin: 16px 0;">
  <p>An application is requesting access to your account:</p>
  <table>
    <tr><td><strong>Application:</strong></td><td>{{.ClientName}}</td></tr>
    <tr><td><strong>Requested scopes:</strong></td><td>{{.Scopes}}</td></tr>
    <tr><td><strong>User code:</strong></td><td><code>{{.UserCode}}</code></td></tr>
  </table>
</div>

<div style="background: #fff3cd; border: 1px solid #ffeaa7; padding: 12px; margin: 16px 0;">
  <strong>Warning:</strong> If you did not initiate this request from a device
  you control, do <strong>NOT</strong> approve it. The application will gain
  access to your account.
</div>

<form method="POST" action="/api/v1/oauth/device/confirm">
  <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
  <input type="hidden" name="user_code" value="{{.UserCode}}">
  <input type="hidden" name="device_code" value="{{.DeviceCode}}">
  <button type="submit" name="action" value="approve">Approve</button>
  <button type="submit" name="action" value="deny">Deny</button>
</form>
</body>
</html>
`

// ConfirmationData holds template data for the confirmation page.
type ConfirmationData struct {
	Issuer      string
	ClientName  string
	Scopes      string
	UserCode    string
	DeviceCode  string
	CSRFToken   string
}

// ConfirmHandler processes the user's approve/deny decision.
func ConfirmHandler(svc *DeviceFlowService, issuer string) http.HandlerFunc {
	tmpl := template.Must(template.New("confirm").Parse(confirmationTemplate))

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		_ = r.ParseForm()
		action := r.FormValue("action")
		userCode := r.FormValue("user_code")

		switch action {
		case "approve":
			// The user has authenticated (handled by middleware).
			// Issue approval for the device code.
			userID := r.Header.Get("X-User-ID") // set by auth middleware
			if err := svc.Approve(userCode, userID); err != nil {
				http.Error(w, "approval failed: "+err.Error(), http.StatusBadRequest)
				return
			}
			w.Write([]byte("Device authorized. You can close this page."))

		case "deny":
			if err := svc.Deny(userCode); err != nil {
				http.Error(w, "denial failed: "+err.Error(), http.StatusBadRequest)
				return
			}
			w.Write([]byte("Request denied. You can close this page."))

		default:
			http.Error(w, "invalid action", http.StatusBadRequest)
		}
	}
}
```

### 6.4 Additional Mitigations

| Mitigation | Description | Effectiveness |
|---|---|---|
| **Client name display** | Show registered client name, not just ID | High — users recognize app names |
| **Scope visualization** | Human-readable scope descriptions | High — users can see what they're granting |
| **Notification on approval** | Push/email notification after device authorization | Medium — detect after the fact |
| **Step-up authentication** | Require re-authentication before approving device flow | High — blocks session-hijacking |
| **Geolocation check** | Flag if device flow origin IP differs from approval IP | Medium — VPNs/Tor complicate this |
| **Cooldown period** | Delay token issuance by 5 seconds after approval | Low — marginal security, poor UX |

---

## 7. Device Flow in Headless/IoT Context

### 7.1 Constrained Device Challenges

IoT devices face unique constraints for device flow:

| Constraint | Impact on Device Flow |
|---|---|
| Limited entropy source | Hardware RNG may be weak; code generation on device is risky |
| No persistent storage | Device code must be stored in volatile memory only |
| No display | User code must be transmitted via side channel (BLE, NFC) |
| Intermittent connectivity | Polling may stop; need reconnection logic |
| Limited clock accuracy | TTL calculations may drift; rely on server-provided timestamps |
| No secure element | Device code is stored in plaintext RAM |

### 7.2 Code Generation: Device vs. Server

**Server-side generation (recommended)**: The server generates both `device_code` and `user_code`. The device receives them via HTTPS. This ensures:
- Consistent entropy quality
- Centralized rate limiting
- No device-side crypto dependencies

**Device-side generation (anti-pattern)**: The device generates its own code and registers it with the server. This is dangerous because:
- Constrained devices may have weak RNG
- Attacker can predict device-generated codes if RNG is compromised
- Server can't enforce code format/rate limits

### 7.3 Secure Storage of Device Code on IoT

The `device_code` is a bearer secret. On an IoT device, it should be:
- Stored in a secure element if available (TPM, TrustZone)
- Never logged or transmitted in plaintext (beyond the initial HTTPS response)
- Cleared from memory after the flow completes
- Not persisted to disk (use volatile memory only)

### 7.4 Go Code: IoT Device Registration Client

```go
package deviceflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IoTDeviceClient runs on a constrained device to initiate and complete device flow.
type IoTDeviceClient struct {
	ServerURL     string
	ClientID      string
	HTTPClient    *http.Client
	// secureStorage abstracts a secure element (TPM, TrustZone, etc.)
	secureStorage SecureStorage
}

// SecureStorage is an interface for storing secrets securely.
// Production implementations use TPM, Android Keystore, iOS Secure Enclave, etc.
type SecureStorage interface {
	Store(key, value string) error
	Retrieve(key string) (string, error)
	Delete(key string) error
}

// InitiateFlow starts the device authorization flow.
// Returns the user code to display and the device code (stored securely).
func (c *IoTDeviceClient) InitiateFlow(ctx context.Context, scopes []string) (userCode string, verificationURI string, err error) {
	body := fmt.Sprintf("client_id=%s&scope=%s", c.ClientID, strings.Join(scopes, " "))
	req, err := http.NewRequestWithContext(ctx, "POST",
		c.ServerURL+"/api/v1/oauth/device_authorization",
		bytes.NewBufferString(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("initiate device flow: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	// Store device code in secure storage — never in plaintext config.
	if err := c.secureStorage.Store("device_code", result.DeviceCode); err != nil {
		return "", "", fmt.Errorf("secure storage: %w", err)
	}

	return result.UserCode, result.VerificationURI, nil
}

// PollForToken polls the token endpoint until a token is issued or the flow expires.
// Implements RFC 8628 slow_down handling.
func (c *IoTDeviceClient) PollForToken(ctx context.Context) (string, error) {
	interval := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}

		deviceCode, err := c.secureStorage.Retrieve("device_code")
		if err != nil {
			return "", fmt.Errorf("retrieve device code: %w", err)
		}

		token, pollErr := c.pollOnce(ctx, deviceCode)
		if pollErr == nil {
			// Success — clear the device code from storage.
			_ = c.secureStorage.Delete("device_code")
			return token, nil
		}

		switch pollErr.Error() {
		case "slow_down":
			interval += 5 * time.Second // RFC 8628 §3.5
		case "authorization_pending":
			continue
		default:
			// expired_token, access_denied, or invalid_grant — terminal.
			_ = c.secureStorage.Delete("device_code")
			return "", pollErr
		}
	}
}

func (c *IoTDeviceClient) pollOnce(ctx context.Context, deviceCode string) (string, error) {
	body := fmt.Sprintf("grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=%s&client_id=%s",
		deviceCode, c.ClientID)
	req, err := http.NewRequestWithContext(ctx, "POST",
		c.ServerURL+"/api/v1/oauth/token",
		bytes.NewBufferString(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("%s", errResp.Error)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}
```

---

## 8. GGID Device Flow Audit

### 8.1 Implementation Overview

The GGID OAuth service implements device flow in `services/oauth/internal/service/oauth_service.go` (lines 1155–1366) and `services/oauth/internal/server/server.go` (lines 351–369, 855–924).

**Endpoints:**
| Endpoint | Method | Purpose |
|---|---|---|
| `/api/v1/oauth/device_authorization` | POST | Create device code + user code |
| `/api/v1/oauth/token` (grant_type=device_code) | POST | Poll for token |
| `/api/v1/oauth/device/approve` | POST | User approves code |

### 8.2 Security Measures Present

| Measure | Status | Details |
|---|---|---|
| **Crypto-random device code** | Present | `generateDeviceCode(40)` uses `crypto/rand` via `cryptoRandInt()` — 40 chars from 62-char set = ~238 bits entropy |
| **User code charset** | Present | `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` — 32 chars (ambiguous chars removed: 0, O, 1, I, L, U) |
| **User code entropy** | Adequate | 8 chars × log2(32) = 40 bits |
| **Code TTL** | Present | 15 minutes (900 seconds) — within recommended range |
| **Polling interval** | Present | 5-second interval returned in response |
| **slow_down enforcement** | Present | Returns `slow_down` if polling faster than 5s since last poll |
| **Expiry cleanup** | Present (lazy) | Expired codes deleted on access; no background sweep |
| **Completed code cleanup** | Present | Device code deleted immediately after token issuance |
| **Client ID parameter** | Present (incomplete) | `PollDeviceToken` accepts `clientID` but does NOT validate `info.ClientID == clientID` |
| **Token issuance** | Present | RS256-signed JWT with proper claims (iss, sub, tenant_id, jti, exp) |

### 8.3 Security Measures Missing

| Gap | Severity | Details |
|---|---|---|
| **No authentication on approve endpoint** | **Critical** | `/api/v1/oauth/device/approve` accepts `user_id` as a form value or `X-User-ID` header. No JWT verification, no session check. Anyone can approve any device code for any user. |
| **Client ID not validated in polling** | **High** | `PollDeviceToken` receives `clientID` but never compares it to `info.ClientID`. An attacker who learns a device code can poll with any client ID. |
| **No per-IP rate limiting** | **High** | Both `device_authorization` and `token` endpoints have no rate limiting. Attackers can create unlimited codes or enumerate device codes at network speed. |
| **No deny endpoint** | **High** | There is no way for a user to explicitly deny a device flow. Denied status is only set internally (via test code). Users can only ignore the code until it expires. |
| **No verification page** | **High** | The `verification_uri` points to `/device` but no handler renders this page. No confirmation page showing app name, scopes, or warnings. |
| **In-memory storage only** | **Medium** | `deviceCodeStore` is a global `map` with no persistence. Codes are lost on restart. No TTL-based expiry (Redis EX). Not suitable for multi-instance deployment. |
| **No background cleanup** | **Medium** | Expired codes are only cleaned on access. Codes that are never polled again accumulate indefinitely in memory. |
| **No CSRF protection** | **Medium** | The approve endpoint has no CSRF token. An attacker could craft a POST request to approve a device code. |
| **Race condition in polling** | **Medium** | `PollDeviceToken` uses `RLock` to read `info`, then directly mutates `info.LastPoll` without holding a write lock. Concurrent polls can race. |
| **No HTTPS validation** | **Low** | `verificationURI` is built from `req.Issuer + "/device"` with no HTTPS check. |
| **No brute-force protection on user code** | **Medium** | No rate limiting on code submission at the verification page. While 40 bits of entropy makes brute-force impractical at human speed, automated submission has no throttle. |
| **No verification_uri_complete** | **Low** | RFC 8628 §3.3.2 optional field for QR code / deep link support is not implemented. |

### 8.4 Code Review: Key Vulnerability

The most critical vulnerability is the unauthenticated approve endpoint:

```go
// server.go:896-924
mux.HandleFunc("/api/v1/oauth/device/approve", func(w http.ResponseWriter, r *http.Request) {
    // ...
    userIDStr := r.FormValue("user_id")
    if userIDStr == "" {
        userIDStr = r.Header.Get("X-User-ID")  // spoofable!
    }
    userID, err := uuid.Parse(userIDStr)
    // ...
    oauthSvc.ApproveDeviceCode(userCode, userID)  // approves for arbitrary user!
})
```

An attacker can POST:
```
POST /api/v1/oauth/device/approve
Content-Type: application/x-www-form-urlencoded

user_code=7K3P-9WQM&user_id=<victim-uuid>
```

This approves the device code as any user, no authentication required.

---

## 9. Gap Analysis & Recommendations

### 9.1 Priority Action Items

| # | Action | Severity | Effort | Description |
|---|---|---|---|---|
| 1 | **Authenticate approve endpoint** | Critical | 2h | Require valid JWT/session on `/device/approve`. Remove `user_id` form parameter. Extract user from authenticated context (auth middleware). |
| 2 | **Validate client ID in polling** | High | 1h | In `PollDeviceToken`, compare `clientID` parameter to `info.ClientID`. Return `invalid_client` on mismatch. |
| 3 | **Add per-IP rate limiting** | High | 4h | Apply sliding-window rate limiter to `device_authorization` (10/min) and `token` (30/min) endpoints. Use the `PollingRateLimiter` pattern from Section 2.3. |
| 4 | **Implement deny flow + confirmation page** | High | 1d | Add `/device/deny` endpoint, render confirmation page showing client name and scopes, implement CSRF protection. |
| 5 | **Migrate to Redis with TTL** | Medium | 4h | Replace in-memory `deviceCodeStore` with Redis. Use `SETEX` for automatic expiry. Add `user_code → device_code` index in Redis. |
| 6 | **Fix race condition in polling** | Medium | 1h | Change `PollDeviceToken` to use write lock for the entire pending-check-and-update block, or use `sync.Mutex` per code. |
| 7 | **Add background sweep** | Medium | 1h | Launch a goroutine that calls `sweep()` every 30 seconds to clean expired codes. Use `context.Context` for graceful shutdown. |

### 9.2 Total Effort Estimate

| Phase | Items | Effort |
|---|---|---|
| **P0 (Critical/High)** | #1, #2, #3 | ~1 day |
| **P1 (UX + Security)** | #4, #5 | ~2 days |
| **P2 (Hardening)** | #6, #7 | ~2 hours |

### 9.3 Testing Recommendations

```go
// Security test: approve endpoint rejects unauthenticated requests.
func TestDeviceApprove_RequiresAuth(t *testing.T) {
    req := httptest.NewRequest("POST", "/api/v1/oauth/device/approve",
        strings.NewReader("user_code=7K3P-9WQM"))
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)
    if rec.Code != http.StatusUnauthorized {
        t.Errorf("expected 401, got %d", rec.Code)
    }
}

// Security test: client ID mismatch is rejected.
func TestPollDeviceToken_ClientIDMismatch(t *testing.T) {
    resp, _ := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
        ClientID: "client-a",
    })
    _, err := svc.PollDeviceToken(ctx, resp.DeviceCode, "client-b")
    if err == nil || err.Error() != "invalid_client" {
        t.Errorf("expected invalid_client error, got %v", err)
    }
}

// Security test: rate limiting blocks excessive polling.
func TestPollingRateLimit_PerIP(t *testing.T) {
    rl := NewPollingRateLimiter()
    ip := "10.0.0.1"
    allowed := 0
    for i := 0; i < 100; i++ {
        if rl.CanCreateDeviceAuthorization(ip) {
            allowed++
        }
    }
    if allowed > 10 {
        t.Errorf("expected max 10 creates/min, got %d", allowed)
    }
}
```

### 9.4 Monitoring Recommendations

- Alert on device code creation rate spikes (possible enumeration)
- Alert on slow_down response rate (client non-compliance or attack)
- Log device flow approval with: client ID, user ID, source IP, time-to-approve
- Track average time-to-approve metric (user experience health)
- Monitor pending code count (memory pressure)

---

## References

- [RFC 8628](https://www.rfc-editor.org/info/rfc8628/) — OAuth 2.0 Device Authorization Grant
- [RFC 6749](https://www.rfc-editor.org/info/rfc6749/) — OAuth 2.0 Framework
- [OAuth 2.0 Security Best Current Practice](https://datatracker.ietf.org/doc/draft-ietf-oauth-security-topics/) — IETF Draft
- [Crockford Base32](https://www.crockford.com/base32.html) — Unambiguous character encoding
- `docs/research/device-flow-rfc8628.md` — GGID device flow protocol specification
- GGID Source: `services/oauth/internal/service/oauth_service.go` (lines 1155–1366)
- GGID Source: `services/oauth/internal/server/server.go` (lines 351–369, 855–924)
