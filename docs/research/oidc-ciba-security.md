# OIDC Client-Initiated Backchannel Authentication (CIBA) — Security Research

> **RFC**: [9101](https://datatracker.ietf.org/doc/html/rfc9101) — OAuth 2.0 Pushed Authorization Requests
> companion for CIBA (core: [RFC 9101](https://datatracker.ietf.org/doc/html/rfc9101),
> delivery modes: OpenID Connect CIBA)
> **Scope**: GGID IAM system, OAuth/OIDC service (`services/oauth/`)
> **Date**: 2025

---

## Table of Contents

1. [CIBA Flow Overview](#1-ciba-flow-overview)
2. [Authentication Request ID (auth_req_id) Lifecycle](#2-authentication-request-id-auth_req_id-lifecycle)
3. [Polling Modes — poll, ping, push](#3-polling-modes--poll-ping-push)
4. [Binding Message](#4-binding-message)
5. [User Identification and Consent](#5-user-identification-and-consent)
6. [Open Banking / PSD2 Context (Modena)](#6-open-banking--psd2-context-modena)
7. [Security Threats Specific to CIBA](#7-security-threats-specific-to-ciba)
8. [CIBA vs Device Flow Comparison](#8-ciba-vs-device-flow-comparison)
9. [GGID CIBA Gap Analysis](#9-ggid-ciba-gap-analysis)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. CIBA Flow Overview

CIBA (Client-Initiated Backchannel Authentication) is a **decoupled** authentication flow
where the consuming application (client) initiates authentication on the backchannel,
without requiring the user to interact with a browser on the same device that runs the
client. The user authenticates on a **separate consumption device** (typically a mobile
banking app or authenticator app).

### Why CIBA Exists

| Driver | Use Case |
|---|---|
| **PSD2 / Open Banking** | Payment Service Providers initiate SCA (Strong Customer Authentication) from a server; the user approves on their banking app. |
| **IoT / Connected Devices** | Smart TV, car infotainment — device has no browser input but user has a phone with an authenticator. |
| **Call Center / Kiosk** | Agent initiates authentication for a customer; customer approves on their own phone. |
| **FIDO2 bridging** | Leverage biometric authenticators on a phone without WebAuthn browser support on the consuming device. |

### Comparison with authorization_code Flow

```
                     authorization_code               CIBA
  ─────────────────────────────────────────────────────────────
  User interaction    Front-channel (browser redirect)  Backchannel (separate device)
  Device coupling     User + client = same browser      User device ≠ client device
  Redirect URI        Required                           Not required
  Polling             No                                 Yes (poll mode) or callback (ping/push)
  Token endpoint      Exchanges code for tokens          Polls with auth_req_id
  SCA support         Awkward (browser pop-ups)          Native (authenticator app)
  Browser dependency  Hard                                None
```

### ASCII Sequence Diagram

```
  Client (App)              Authorization Server          Consumption Device (Phone)
  ─────────────             ────────────────────          ──────────────────────────
        │                           │                               │
        │  POST /backchannel/auth   │                               │
        │  (login_hint, scope,      │                               │
        │   binding_message)        │                               │
        │ ─────────────────────────>│                               │
        │                           │                               │
        │   200 auth_req_id         │                               │
        │   expires_in, interval    │   Push notification           │
        │ <─────────────────────────│  "Confirm login: code 8492"   │
        │                           │ ──────────────────────────────>│
        │                           │                               │
        │                           │                    User approves/denies
        │                           │                               │
        │   POST /token             │   POST /approve  (from phone)  │
        │   (auth_req_id)           │<──────────────────────────────│
        │ ─────────────────────────>│                               │
        │                           │                               │
        │   (authorization_pending) │                               │
        │ <─────────────────────────│                               │
        │                           │                               │
        │   ... interval passes ... │                               │
        │                           │                               │
        │   POST /token             │                               │
        │   (auth_req_id)           │                               │
        │ ─────────────────────────>│                               │
        │                           │                               │
        │   200 access_token        │                               │
        │   id_token                │                               │
        │ <─────────────────────────│                               │
        │                           │                               │
```

---

## 2. Authentication Request ID (auth_req_id) Lifecycle

The `auth_req_id` is the central artifact of CIBA. It is issued by the authorization
server upon a backchannel authentication request and is later used by the client to poll
the token endpoint. Its security properties are critical.

### Lifecycle

```
  Client ──POST /bc-authorize──> AS issues auth_req_id
                                      │
                                      ├── Stored with: client_id binding, expiry,
                                      │   user identity, scope, binding_message
                                      │
  Client ──POST /token (auth_req_id)──> AS checks:
                                      │   1. Exists? (not fabricated)
                                      │   2. Same client? (not stolen by another client)
                                      │   3. Not expired?
                                      │   4. Polling interval respected?
                                      │   5. Status: pending / approved / denied / expired
                                      │
                                      └── On approval: issue tokens, invalidate auth_req_id
```

### Security Requirements

| Requirement | Rationale |
|---|---|
| **128+ bits entropy** | Prevents brute-force guessing of valid IDs. |
| **Bound to client_id** | Prevents a malicious client from polling another client's auth_req_id. |
| **Short-lived** (default 300s, max 900s) | Limits window for replay and theft. |
| **Single-use on approval** | Prevents replay after token issuance. |
| **Deleted on expiry** | Prevents zombie requests. |
| **Not enumerable** | UUID v4 + random suffix prevents sequential enumeration. |

### Go Code: auth_req_id Generation and Validation

```go
package ciba

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"
)

// authReqIDEntropy is the number of random bytes used for the auth_req_id.
// 32 bytes = 256 bits of entropy, well above the 128-bit minimum.
const authReqIDEntropy = 32

// AuthReqID represents a decoded auth_req_id for internal use.
type AuthReqID struct {
	Raw       string
	IssuedAt  time.Time
	ExpiresAt time.Time
	ClientID  string
}

// GenerateAuthReqID creates a cryptographically random auth_req_id.
// Format: base64url(32 random bytes) — 43 characters, 256 bits entropy.
func GenerateAuthReqID() (string, error) {
	b := make([]byte, authReqIDEntropy)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate auth_req_id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateAuthReqID checks that the auth_req_id has the correct format
// and minimum entropy characteristics.
func ValidateAuthReqID(id string) error {
	if len(id) < 32 {
		return fmt.Errorf("auth_req_id too short: %d chars (min 32)", len(id))
	}
	if len(id) > 128 {
		return fmt.Errorf("auth_req_id too long: %d chars (max 128)", len(id))
	}
	// Verify it's valid base64url
	raw, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return fmt.Errorf("auth_req_id not valid base64url: %w", err)
	}
	if len(raw) < 16 {
		return fmt.Errorf("auth_req_id entropy below 128 bits: %d bytes", len(raw))
	}
	return nil
}

// ConstantTimeCompare safely compares two auth_req_id values without
// leaking length or prefix information through timing side-channels.
func ConstantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

### Anti-Replay: Client Binding Check

```go
// validateClientBinding ensures the polling client matches the one that
// initiated the backchannel authentication request. This prevents a
// stolen auth_req_id from being redeemed by a different client.
func (s *CIBAService) validateClientBinding(authReqID, clientID string) error {
	entry, ok := s.store.Get(authReqID)
	if !ok {
		return ErrUnknownAuthReqID
	}
	if !ConstantTimeCompare(entry.ClientID, clientID) {
		// Do NOT reveal that the auth_req_id exists — return the same
		// error as "unknown" to avoid information leakage.
		return ErrUnknownAuthReqID
	}
	return nil
}
```

---

## 3. Polling Modes — poll, ping, push

CIBA defines three delivery modes, negotiated per-client during registration:

### Mode Comparison

```
  Mode    Direction               Mechanism                         Client Complexity
  ────    ────────                ─────────                         ────────────────
  poll    Client → AS             Client polls /token repeatedly    Low (simple loop)
  ping    AS → Client callback    AS sends notification, client     Medium (needs callback endpoint)
                                  then polls /token
  push    AS → Client             AS pushes tokens directly to      High (callback security)
                                  client callback (no polling)
```

### Security Implications

| Mode | Threat | Mitigation |
|---|---|---|
| **poll** | DoS via rapid polling | `slow_down` error + interval enforcement + rate limit |
| **ping** | Callback URL spoofing | Client callback must be TLS + registered + mTLS |
| **push** | Token interception at callback | Client callback must verify AS signature (mTLS or signed JWT) |

### slow_down Error Response

RFC 9101 mandates that when a client polls faster than the declared interval, the AS
responds with HTTP 400 and:

```json
{
  "error": "slow_down",
  "error_description": "The polling interval is too short. Wait at least 5 seconds."
}
```

The client MUST increase its interval by at least 5 seconds after receiving `slow_down`.

### Go Code: Poll Endpoint with Interval Enforcement

```go
package ciba

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// CIBAPollConfig controls polling behavior.
type CIBAPollConfig struct {
	DefaultInterval  time.Duration // default minimum between polls (e.g., 5s)
	MaxInterval      time.Duration // upper bound after slow_down escalations
	SlowDownPenalty  time.Duration // added interval on each slow_down (e.g., 5s)
	MaxPollsPerWindow int          // rate limit: max polls per rate window
	RateWindow        time.Duration
}

var DefaultPollConfig = CIBAPollConfig{
	DefaultInterval:   5 * time.Second,
	MaxInterval:       60 * time.Second,
	SlowDownPenalty:   5 * time.Second,
	MaxPollsPerWindow: 20,
	RateWindow:        time.Minute,
}

// pollState tracks per-auth_req_id polling metadata.
type pollState struct {
	mu            sync.Mutex
	lastPoll      time.Time
	currentInterval time.Duration
	pollCount     int
	windowStart   time.Time
}

// CheckPollAllowed verifies that the client is respecting the polling
// interval. Returns nil if allowed, or a *CIBAError to send to the client.
func (ps *pollState) CheckPollAllowed(cfg CIBAPollConfig) *CIBAError {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()

	// Rate limit: too many polls in the window.
	if now.Sub(ps.windowStart) > cfg.RateWindow {
		ps.windowStart = now
		ps.pollCount = 0
	}
	ps.pollCount++
	if ps.pollCount > cfg.MaxPollsPerWindow {
		return &CIBAError{
			Err:  "slow_down",
			Desc: "rate limit exceeded; increase polling interval significantly",
		}
	}

	// Interval enforcement.
	minInterval := ps.currentInterval
	if minInterval == 0 {
		minInterval = cfg.DefaultInterval
	}

	if !ps.lastPoll.IsZero() && now.Sub(ps.lastPoll) < minInterval {
		// Escalate: each violation increases the required interval.
		ps.currentInterval += cfg.SlowDownPenalty
		if ps.currentInterval > cfg.MaxInterval {
			ps.currentInterval = cfg.MaxInterval
		}
		return &CIBAError{
			Err:  "slow_down",
			Desc: "polling too fast; wait at least " + ps.currentInterval.String(),
		}
	}

	ps.lastPoll = now
	return nil
}

// HandlePollToken is the HTTP handler for the token endpoint when
// grant_type=urn:openid:params:grant-type:ciba.
func (h *CIBAHandler) HandlePollToken(w http.ResponseWriter, r *http.Request) {
	authReqID := r.PostFormValue("auth_req_id")
	clientID := r.PostFormValue("client_id")

	// 1. Validate auth_req_id format.
	if err := ValidateAuthReqID(authReqID); err != nil {
		writeCIBAError(w, http.StatusBadRequest, "invalid_grant", "malformed auth_req_id")
		return
	}

	// 2. Validate client binding.
	entry, err := h.svc.GetCIBAEntry(authReqID, clientID)
	if err != nil {
		writeCIBAError(w, http.StatusBadRequest, "invalid_grant", "unknown auth_req_id")
		return
	}

	// 3. Check polling interval.
	if cerr := entry.PollState.CheckPollAllowed(h.cfg); cerr != nil {
		writeCIBAError(w, http.StatusBadRequest, cerr.Err, cerr.Desc)
		return
	}

	// 4. Check status.
	switch entry.Status {
	case CIBAStatusPending:
		writeCIBAError(w, http.StatusBadRequest, "authorization_pending",
			"user has not yet responded")
	case CIBAStatusApproved:
		tokens, err := h.svc.IssueCIBATokens(entry)
		if err != nil {
			writeCIBAError(w, http.StatusInternalServerError, "server_error",
				"failed to issue tokens")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(tokens)
	case CIBAStatusDenied:
		writeCIBAError(w, http.StatusBadRequest, "access_denied",
			"user denied the request")
	case CIBAStatusExpired:
		writeCIBAError(w, http.StatusBadRequest, "expired_token",
			"auth_req_id has expired")
	}
}

func writeCIBAError(w http.ResponseWriter, status int, errCode, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": desc,
	})
}
```

---

## 4. Binding Message

The **binding message** is a short, human-readable string that the consumption device
displays to the user when asking for approval. It binds the backchannel request to a
specific, recognizable context so the user can confirm they are approving the right thing.

### How It Prevents MITM

In a CIBA flow, the user never sees the client application directly — they only see the
authenticator app. Without a binding message, a phishing attack could work as follows:

1. Attacker initiates CIBA auth with the user's login_hint.
2. The user's phone receives a generic "Confirm login?" prompt.
3. User approves, thinking it's their own login.
4. Attacker receives the token.

The binding message breaks this attack:

1. Attacker initiates CIBA auth.
2. The user's phone shows: **"Confirm login for Online Banking — Reference: 8492"**.
3. The user is also seeing **"8492"** on their banking website (the client displays it).
4. The user can cross-check the codes match — if they don't, the user denies.

This creates a **transaction binding** similar to FIDO transaction confirmation.

### Binding Message Entropy

The binding message should contain at least **20 bits of entropy** (4-6 alphanumeric
characters) to prevent guessing attacks where an attacker tries to generate a CIBA
request with the same binding message.

### Go Code: Binding Message Generation and Verification

```go
package ciba

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"strings"
)

// BindingMessage represents a CIBA binding message.
type BindingMessage struct {
	Category string // e.g., "Login", "Payment", "Transfer"
	Entity   string // e.g., "ACME Bank"
	Code     string // random entropy code, e.g., "8492"
}

// GenerateBindingMessage creates a binding message with a random code
// of the specified number of digits.
func GenerateBindingMessage(category, entity string, codeLen int) (*BindingMessage, error) {
	if codeLen < 4 {
		codeLen = 4
	}
	code, err := generateNumericCode(codeLen)
	if err != nil {
		return nil, err
	}
	return &BindingMessage{
		Category: category,
		Entity:   entity,
		Code:     code,
	}, nil
}

// generateNumericCode produces a random numeric string of the given length.
func generateNumericCode(length int) (string, error) {
	digits := make([]byte, length)
	if _, err := rand.Read(digits); err != nil {
		return "", err
	}
	for i := range digits {
		digits[i] = '0' + (digits[i] % 10)
	}
	return string(digits), nil
}

// String returns the human-readable binding message.
func (bm *BindingMessage) String() string {
	return fmt.Sprintf("%s for %s — Code: %s",
		bm.Category, bm.Entity, bm.Code)
}

// VerifyBindingMessage constant-time compares the expected binding message
// code with what the user entered or confirmed on the consumption device.
func VerifyBindingMessage(expected, actual string) bool {
	// Normalize to uppercase, trim whitespace.
	e := strings.ToUpper(strings.TrimSpace(expected))
	a := strings.ToUpper(strings.TrimSpace(actual))
	if len(e) != len(a) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(e), []byte(a)) == 1
}

// ValidateBindingMessage checks the binding message for security properties:
// - Length between 4 and 128 characters (RFC 9101 allows 1-128 but we enforce minimum)
// - Contains at least one digit or alphanumeric code
// - No control characters
func ValidateBindingMessage(msg string) error {
	if len(msg) < 4 || len(msg) > 128 {
		return fmt.Errorf("binding_message length %d out of range [4, 128]", len(msg))
	}
	hasAlnum := false
	for _, r := range msg {
		if r < 0x20 || r > 0x7E {
			return fmt.Errorf("binding_message contains non-printable character")
		}
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasAlnum = true
		}
	}
	if !hasAlnum {
		return fmt.Errorf("binding_message must contain at least one alphanumeric character")
	}
	return nil
}
```

---

## 5. User Identification and Consent

CIBA decouples the client from the user interaction. The authorization server must
identify the target user from one of three hints:

| Hint | Description | Privacy Risk |
|---|---|---|
| `login_hint` | Plaintext identifier (email, phone, username) | **High** — visible in request logs |
| `login_hint_token` | Opaque or JWT token referencing the user | **Low** — token can be opaque |
| `id_token_hint` | Previously issued ID token for the user | **Low** — encrypted JWT preferred |

### Privacy: login_hint Must Not Leak PII

The `login_hint` field is transmitted in the HTTP request body. If it contains a raw
email address or phone number, it will appear in:

- Load balancer / WAF access logs
- Monitoring/observability pipelines
- Potential MITM if TLS is misconfigured

**Best practice**: Use `login_hint_token` (an opaque server-side reference) whenever
possible. If `login_hint` must be used, hash or alias it.

### Go Code: User Identification

```go
package ciba

import (
	"context"
	"fmt"
	"strings"

	"github.com/ggid/ggid/pkg/crypto"
)

// UserIdentifier resolves a user from CIBA hints.
type UserIdentifier struct {
	userRepo UserRepo
	hmacKey  []byte // for login_hint hashing
}

// IdentifyUser resolves the target user from the provided hints.
// Priority: id_token_hint > login_hint_token > login_hint.
func (ui *UserIdentifier) IdentifyUser(ctx context.Context,
	idTokenHint, loginHintToken, loginHint string,
) (string, error) {
	// 1. id_token_hint: validate the JWT and extract subject.
	if idTokenHint != "" {
		userID, err := ui.resolveFromIDToken(ctx, idTokenHint)
		if err != nil {
			return "", fmt.Errorf("id_token_hint resolution failed: %w", err)
		}
		return userID, nil
	}

	// 2. login_hint_token: lookup opaque token.
	if loginHintToken != "" {
		userID, err := ui.userRepo.LookupHintToken(ctx, loginHintToken)
		if err != nil {
			return "", fmt.Errorf("login_hint_token resolution failed: %w", err)
		}
		return userID, nil
	}

	// 3. login_hint: hash the hint before any lookup or logging.
	if loginHint != "" {
		hashed := ui.hashHint(loginHint)
		userID, err := ui.userRepo.LookupByHashedHint(ctx, hashed)
		if err != nil {
			return "", fmt.Errorf("login_hint resolution failed: %w", err)
		}
		return userID, nil
	}

	return "", fmt.Errorf("no user identification hint provided")
}

// hashHint creates a keyed HMAC hash of the login_hint to avoid
// storing or logging the raw PII value.
func (ui *UserIdentifier) hashHint(hint string) string {
	return crypto.HMACSHA256(ui.hmacKey, []byte(strings.ToLower(strings.TrimSpace(hint))))
}

// resolveFromIDToken validates the id_token_hint JWT and extracts
// the subject (user ID).
func (ui *UserIdentifier) resolveFromIDToken(ctx context.Context, token string) (string, error) {
	claims, err := ui.validateIDToken(ctx, token)
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", fmt.Errorf("id_token_hint has no subject")
	}
	return claims.Subject, nil
}
```

### Consent Handling

In the decoupled CIBA flow, consent is obtained on the consumption device, not in a
browser redirect. The AS must:

1. Display the requested scopes and binding message on the authenticator app.
2. Require explicit user action (approve/deny) — no implicit consent.
3. Record the consent with timestamp, device ID, and authenticator method.
4. Allow the user to revoke consent later.

```go
// ConsentRecord stores CIBA consent for audit.
type ConsentRecord struct {
	UserID          string    `json:"user_id"`
	ClientID        string    `json:"client_id"`
	Scopes          []string  `json:"scopes"`
	AuthReqID       string    `json:"auth_req_id"`
	BindingMessage  string    `json:"binding_message"`
	Approved        bool      `json:"approved"`
	DeviceID        string    `json:"device_id"` // consumption device
	AuthenticatorMethod string `json:"authenticator_method"` // e.g., "biometric", "pin"
	Timestamp       time.Time `json:"timestamp"`
	IPAddress       string    `json:"ip_address"`
}
```

---

## 6. Open Banking / PSD2 Context (Modena)

CIBA was designed primarily for the Open Banking / PSD2 ecosystem. The **Modena**
consortium (now part of OpenID Foundation) contributed to the CIBA specification to
meet PSD2 Strong Customer Authentication (SCA) requirements.

### PSD2 SCA Requirements

| Requirement | How CIBA Satisfies It |
|---|---|
| **Two-factor authentication** | Consumption device provides possession factor; PIN/biometric provides knowledge/inherence |
| **Dynamic linking** (transaction binding) | `binding_message` ties auth to a specific transaction amount/payee |
| **Non-repudiation** | Authenticator app can produce a signed assertion |
| **No browser dependency** | Entirely backchannel — no user agent required on the consuming device |

### Transaction Signing via CIBA

In banking, CIBA can be used for payment confirmation:

1. Bank backend initiates CIBA with `binding_message` = "Pay €500 to IBAN DE89... code: 8492".
2. User's authenticator app displays the payment details + code.
3. User confirms with biometric.
4. AS issues a signed authorization that the bank backend uses to execute the payment.

### Go Code: Transaction-Signing CIBA Flow

```go
package ciba

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// TransactionCIBARequest extends a standard CIBA request with transaction
// details for PSD2 payment confirmation.
type TransactionCIBARequest struct {
	BackchannelAuthRequest            // embedded standard CIBA params
	PaymentID              string     // unique payment identifier
	PayeeAccount           string     // IBAN or account number
	PayeeName              string     // recipient name
	Amount                 float64    // payment amount
	Currency               string     // ISO 4217 currency code
	ExecutionDate          time.Time  // scheduled execution
}

// TransactionDataForBinding creates the binding message content that
// the user will see on their authenticator app. This creates the
// "dynamic linking" required by PSD2 SCA.
func (r *TransactionCIBARequest) TransactionDataForBinding() string {
	return fmt.Sprintf("Pay %s %.2f to %s (%s) — Ref: %s",
		r.Currency, r.Amount, r.PayeeName, maskAccount(r.PayeeAccount), r.PaymentID)
}

// TransactionHash produces a SHA-256 hash of the transaction data
// that gets signed by the authenticator device. This hash is embedded
// in the issued tokens for non-repudiation.
func (r *TransactionCIBARequest) TransactionHash() string {
	data := fmt.Sprintf("%s|%s|%s|%.2f|%s|%s",
		r.PaymentID, r.PayeeAccount, r.PayeeName,
		r.Amount, r.Currency, r.ExecutionDate.Format(time.RFC3339))
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// maskAccount partially obscures the account number for display.
func maskAccount(account string) string {
	if len(account) <= 4 {
		return "****"
	}
	return account[:2] + "..." + account[len(account)-4:]
}

// InitiatePaymentAuth starts a CIBA flow for a payment confirmation.
func (s *CIBAService) InitiatePaymentAuth(ctx context.Context,
	req *TransactionCIBARequest,
) (*BackchannelAuthResponse, error) {
	// 1. Generate binding message from transaction data.
	bindingData := req.TransactionDataForBinding()
	bm, err := GenerateBindingMessage("Confirm Payment", req.PayeeName, 6)
	if err != nil {
		return nil, fmt.Errorf("generate binding message: %w", err)
	}
	req.BindingMessage = bindingData + " — Code: " + bm.Code

	// 2. Verify client is authorized for payment scope.
	if !s.clientSupportsScope(ctx, req.ClientID, "payments") {
		return nil, fmt.Errorf("client not authorized for payment confirmation")
	}

	// 3. Store transaction hash for non-repudiation.
	txHash := req.TransactionHash()

	// 4. Initiate standard CIBA flow.
	resp, err := s.BackchannelAuthentication(ctx, &req.BackchannelAuthRequest)
	if err != nil {
		return nil, err
	}

	// 5. Store transaction metadata alongside the CIBA entry.
	s.txStore.Store(resp.AuthReqID, &TransactionConsent{
		PaymentID:      req.PaymentID,
		TransactionHash: txHash,
		Amount:         req.Amount,
		Currency:       req.Currency,
		PayeeAccount:   req.PayeeAccount,
		BindingMessage: req.BindingMessage,
	})

	return resp, nil
}

// TransactionConsent stores the payment confirmation context.
type TransactionConsent struct {
	PaymentID       string
	TransactionHash string
	Amount          float64
	Currency        string
	PayeeAccount    string
	BindingMessage  string
	SignedBy        string    // authenticator device ID
	SignedAt        time.Time
}
```

---

## 7. Security Threats Specific to CIBA

CIBA introduces attack surfaces that don't exist in the standard authorization_code flow.

### 7.1 auth_req_id Replay

**Threat**: An attacker steals a valid `auth_req_id` (e.g., from logs, network sniffing)
and polls the token endpoint to steal the resulting tokens.

**Mitigation**: Bind auth_req_id to the originating client_id via constant-time comparison.

```go
func (s *CIBAService) pollWithBindingCheck(authReqID, clientID string) (*TokenResponse, error) {
	entry, ok := s.store.Get(authReqID)
	if !ok {
		return nil, ErrUnknownAuthReqID
	}
	// CRITICAL: constant-time comparison prevents timing oracle.
	if subtle.ConstantTimeCompare([]byte(entry.ClientID), []byte(clientID)) != 1 {
		// Return same error to avoid leaking that the ID exists.
		return nil, ErrUnknownAuthReqID
	}
	// ... proceed with normal poll logic
}
```

### 7.2 Binding Message Bypass

**Threat**: A client omits the `binding_message` or sets it to a generic value, removing
the MITM protection.

**Mitigation**: The AS should REJECT requests with empty or too-short binding messages
for sensitive scopes.

```go
func (s *CIBAService) enforceBindingMessage(scope, bindingMsg string) error {
	scopes := strings.Fields(scope)
	for _, sc := range scopes {
		if isSensitiveScope(sc) && len(bindingMsg) < 6 {
			return fmt.Errorf("scope %s requires a binding_message of at least 6 chars", sc)
		}
	}
	return nil
}

func isSensitiveScope(scope string) bool {
	sensitive := map[string]bool{
		"payments":         true,
		"transfer":         true,
		"admin":            true,
		"accounts:write":   true,
	}
	return sensitive[scope]
}
```

### 7.3 User Confusion (Approving Wrong Request)

**Threat**: The user has multiple pending CIBA requests and approves the wrong one.

**Mitigation**: Display enough context (binding message, requesting client name, timestamp)
and require the user to type the binding code, not just tap "Approve".

```go
// RequireUserCodeEntry forces the consumption device to collect the
// binding message code from the user, preventing tap-to-approve attacks.
type ApprovalRequest struct {
	AuthReqID      string
	ExpectedCode   string // the binding message code
	ClientName     string
	RequestedAt    time.Time
}

// VerifyApproval checks the user-typed code against the expected code.
func VerifyApproval(req ApprovalRequest, userTypedCode string) error {
	if !VerifyBindingMessage(req.ExpectedCode, userTypedCode) {
		// Log suspicious activity — possible blind approval attempt.
		return fmt.Errorf("binding code mismatch")
	}
	return nil
}
```

### 7.4 Authenticator App Compromise

**Threat**: Malware on the user's phone intercepts or auto-approves CIBA requests.

**Mitigation**: Require step-up authentication (biometric) on the consumption device
and detect anomalous approval patterns (e.g., approving within 1 second of request).

```go
// DetectFastApproval flags approvals that happen suspiciously quickly,
// indicating possible automated approval by malware.
func (s *CIBAService) DetectFastApproval(entry *cibaEntry, approvedAt time.Time) bool {
	elapsed := approvedAt.Sub(entry.CreatedAt)
	// Less than 2 seconds is suspiciously fast for a human to read
	// a binding message and approve.
	return elapsed < 2*time.Second
}
```

### 7.5 Polling DoS

**Threat**: A malicious client (or many distributed clients) flood the token endpoint
with rapid polls, exhausting server resources.

**Mitigation**: Per-auth_req_id interval enforcement + per-IP rate limiting + circuit
breaker for the token endpoint.

```go
// PollRateLimiter combines per-auth_req_id and per-IP rate limiting.
type PollRateLimiter struct {
	perID   *TokenBucketLimiter // keyed by auth_req_id
	perIP   *TokenBucketLimiter // keyed by client IP
}

func (rl *PollRateLimiter) Allow(authReqID, clientIP string) bool {
	if !rl.perID.Allow(authReqID, 12, time.Minute) { // 12 polls/min per ID
		return false
	}
	if !rl.perIP.Allow(clientIP, 60, time.Minute) { // 60 polls/min per IP
		return false
	}
	return true
}
```

---

## 8. CIBA vs Device Flow Comparison

Both CIBA (RFC 9101) and Device Authorization Grant (RFC 8628) are **decoupled**
flows, but they serve fundamentally different scenarios.

```
  Dimension              CIBA (RFC 9101)                 Device Flow (RFC 8628)
  ─────────              ─────────────────                ──────────────────────
  User has authenticator Yes (separate device)           No (device has no browser/input)
  User input device      Consumption device (phone)      Any browser (second device)
  Who initiates          Client (backchannel POST)       Client (device_authorization POST)
  User identification    login_hint / id_token_hint      User enters code on browser
  Binding message        Yes (anti-MITM)                 No (user_code is the binding)
  Polling mechanism      Poll / ping / push              Poll only
  Expiry default         300 seconds                     ~1800 seconds (device_code)
  Typical use case       PSD2 banking, IoT               Smart TV, CLI tools, game consoles
  Consent device         Authenticator app               Browser on second device
  SCA compliance         Yes (designed for PSD2)         No (not designed for SCA)
  User experience        Transparent (push notification) Requires code entry on browser
```

### When to Use Each

| Scenario | Recommended Flow | Why |
|---|---|---|
| Banking payment confirmation | **CIBA** | PSD2 SCA, transaction binding |
| Smart TV login | **Device Flow** | TV has no authenticator, user enters code on phone |
| IoT device authentication | **CIBA** | Device has paired authenticator app |
| CLI tool authentication | **Device Flow** | No authenticator, user opens browser |
| Call center identity verification | **CIBA** | Agent triggers auth, customer approves on phone |
| Game console login | **Device Flow** | Console displays code, user enters on phone browser |

### Key Security Difference

```
  CIBA:  auth_req_id is BOUND to client_id + user → replay-resistant
  Device: device_code is BOUND to client_id → user enters user_code separately

  CIBA:  binding_message provides transaction-level MITM protection
  Device: user_code provides weak binding (guessable, though rate-limited)

  CIBA:  Three delivery modes (poll/ping/push) → push is highest risk
  Device: Only poll mode → simpler attack surface
```

---

## 9. GGID CIBA Gap Analysis

### What Exists

The GGID OAuth service (`services/oauth/internal/service/ciba.go`) already contains a
**substantial CIBA implementation** at the service layer:

| Component | Status | Location |
|---|---|---|
| `BackchannelAuthRequest` struct | **Exists** | `ciba.go:17` |
| `BackchannelAuthResponse` struct | **Exists** | `ciba.go:33` |
| `CIBAStatus` enum (pending/approved/denied/expired) | **Exists** | `ciba.go:42` |
| `cibaEntry` internal storage struct | **Exists** | `ciba.go:50` |
| `BackchannelAuthentication()` — initiates CIBA | **Exists** | `ciba.go:73` |
| `PollCIBAToken()` — polls for token | **Exists** | `ciba.go:141` |
| `ApproveCIBAAuth()` — approval from authenticator | **Exists** | `ciba.go:195` |
| `DenyCIBAAuth()` — denial from authenticator | **Exists** | `ciba.go:213` |
| `CIBAError` type | **Exists** | `ciba.go:226` |
| `generateAuthReqID()` | **Exists** | `ciba.go:235` |
| Client validation + grant-type check | **Exists** | `ciba.go:74-91` |
| Expiry enforcement | **Exists** | `ciba.go:98-102` |
| Polling interval enforcement (`slow_down`) | **Exists** | `ciba.go:156-158` |
| User resolution from `login_hint` | **Exists** | `ciba.go:109-118` |
| CIBA test coverage | **Exists** | `par_logout_ciba_test.go` |

### What's Missing

| Gap | Severity | Details |
|---|---|---|
| **No HTTP endpoint** | **Critical** | `server.go` has no `/backchannel/authenticate` or `/bc-authorize` route. The service-layer code is unreachable from HTTP. |
| **No discovery metadata** | **High** | `models.go` has `backchannel_logout_supported` but NO `backchannel_authentication_endpoint`, `backchannel_token_delivery_modes_supported`, `backchannel_authentication_request_signing_alg_values_supported` fields. |
| **In-memory storage** | **Medium** | Uses `sync.Map` (global var). Not distributed, not persistent across restarts. Should use Redis. |
| **No ping/push delivery modes** | **Medium** | Only `poll` mode is implemented. `ping` (callback notification) and `push` (direct token push) are missing. |
| **No binding message validation** | **Medium** | Binding message is stored but not validated for length/entropy. Empty binding messages are accepted. |
| **No client binding on poll** | **High** | `PollCIBAToken` does NOT verify that the polling `client_id` matches the one that initiated the request. Replay vulnerability. |
| **No `login_hint_token` support** | **Low** | Only `login_hint` (plaintext) is processed. `login_hint_token` and `id_token_hint` are accepted as parameters but not resolved. |
| **No user_code support** | **Low** | `user_code` parameter is stored but never validated during approval. |
| **No request JWT (JAR) support** | **Low** | CIBA supports signed request objects (RFC 9101). Not implemented. |
| **No context parameter processing** | **Low** | `Context` field is in the struct but unused. |
| **No audit logging** | **Medium** | CIBA events (initiation, approval, denial, expiry) are not published to the audit service. |
| **No rate limiting on /bc-authorize** | **Medium** | An attacker could flood the backchannel endpoint. |

### Architecture for Adding CIBA HTTP Endpoints

```
  ┌──────────────────────────────────────────────────────┐
  │                  OAuth Server (server.go)             │
  │                                                       │
  │  POST /bc-authorize ──> BackchannelAuthentication()   │
  │  POST /token          ─> PollCIBAToken()              │
  │    (grant_type=ciba)                                  │
  │  POST /bc-approve     ─> ApproveCIBAAuth()            │
  │  POST /bc-deny        ─> DenyCIBAAuth()               │
  │                                                       │
  │  Discovery:                                           │
  │    backchannel_authentication_endpoint                │
  │    backchannel_token_delivery_modes_supported         │
  │    backchannel_authentication_request_signing_alg_*   │
  └────────────────────┬─────────────────────────────────┘
                       │
  ┌────────────────────▼─────────────────────────────────┐
  │             CIBA Service (ciba.go)                    │
  │  (existing logic + client binding fix)                │
  └────────────────────┬─────────────────────────────────┘
                       │
  ┌────────────────────▼─────────────────────────────────┐
  │           Redis-backed CIBA Store                     │
  │  (replaces sync.Map for distributed deployment)       │
  └──────────────────────────────────────────────────────┘
```

---

## 10. Gap Analysis & Recommendations

### Priority Action Items

| # | Action | Effort | Impact |
|---|---|---|---|
| 1 | **Wire HTTP endpoints**: Add `/bc-authorize`, `/bc-approve`, `/bc-deny` routes to `server.go`. Add CIBA grant-type handling in the existing `/token` endpoint. | **M** (1-2 days) | Unlocks the entire CIBA flow — service-layer code is already complete. |
| 2 | **Fix client binding vulnerability**: In `PollCIBAToken()`, verify that `client_id` from the poll request matches the `client_id` stored in the `cibaEntry`. Use constant-time comparison. | **S** (2-4 hours) | Closes a replay/impersonation vulnerability where a stolen `auth_req_id` can be redeemed by any client. |
| 3 | **Add CIBA discovery metadata**: Add `backchannel_authentication_endpoint`, `backchannel_token_delivery_modes_supported`, `backchannel_authentication_request_signing_alg_values_supported` to the OIDC discovery response in `models.go` and the discovery handler in `server.go`. | **S** (2-4 hours) | Standards compliance — clients can discover CIBA support. |
| 4 | **Migrate to Redis-backed storage**: Replace `sync.Map` with a Redis-backed store with TTL-based expiry. This enables multi-instance deployment and automatic expiry cleanup. | **M** (1-2 days) | Production readiness — the current in-memory store doesn't survive restarts and doesn't work in multi-replica deployments. |
| 5 | **Add binding message enforcement**: Validate binding message length (min 4 chars) and require it for sensitive scopes (payments, admin). Reject requests with empty binding messages for SCA-required flows. | **S** (2-4 hours) | PSD2 SCA compliance — the binding message is the primary MITM defense in CIBA. |

### Implementation Notes

- **Effort scale**: S = small (half-day), M = medium (1-2 days), L = large (3-5 days)
- **Item 1** is the highest-ROI: the service-layer code in `ciba.go` is well-structured
  and tested. Adding HTTP routes is straightforward — follow the pattern of the existing
  `/api/v1/oauth/device_authorization` endpoint.
- **Item 2** is a security-critical fix. The current `PollCIBAToken` signature accepts
  `clientID` but the function body never compares it to `entry.ClientID`.
- **Item 3** enables OIDC-compliant client auto-configuration. Without discovery metadata,
  clients won't know CIBA is available.
- **Item 4** follows the same pattern as other Redis-backed features in GGID (session
  store, rate limiter).
- **Item 5** can use the `isSensitiveScope()` function pattern shown in section 7.2.

### Future Enhancements (Lower Priority)

- Ping and push delivery modes (requires callback URL management and notification dispatch).
- Signed request objects (JAR) for CIBA — enables client-signed authentication requests.
- Integration with the GGID WebAuthn service for biometric step-up on the consumption device.
- Transaction signing (section 6) for full PSD2 payment confirmation support.
- Audit event publishing for all CIBA state transitions.

---

## References

- [RFC 9101](https://datatracker.ietf.org/doc/html/rfc9101) — OpenID Connect Client-Initiated Backchannel Authentication (CIBA)
- [OpenID CIBA Core 1.0](https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html)
- [RFC 8628](https://datatracker.ietf.org/doc/html/rfc8628) — OAuth 2.0 Device Authorization Grant
- [PSD2 RTS](https://www.eba.europa.eu/regulation-and-policy/payment-services-and-emoney/regulatory-technical-standards-on-strong-customer-authentication-and-secure-communication) — EBA Regulatory Technical Standards on SCA
- [FIDO Alliance CIBA Best Practices](https://fidoalliance.org/)

---

*This document is part of the GGID IAM security research series.*
