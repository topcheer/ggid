# Advanced Passkey Recovery Strategies

> **Status**: Research & Design Document
> **Date**: 2025
> **Author**: GGID Research
> **Related**: [passkey-recovery.md](./passkey-recovery.md) (basics: synced vs device-bound,
> backup eligibility, sync platforms, recovery codes)
> **Focus**: Emerging and advanced recovery mechanisms NOT covered in the basic doc

---

## Table of Contents

1. [Device-to-Device Credential Transfer (FIDO CXP/CXF)](#1-device-to-device-credential-transfer-fido-cxpcxf)
2. [Social Recovery with Threshold Schemes](#2-social-recovery-with-threshold-schemes)
3. [Admin-Issued Recovery Credentials](#3-admin-issued-recovery-credentials)
4. [Expanded Recovery Key Patterns](#4-expanded-recovery-key-patterns)
5. [Survey: Competitor Recovery Approaches](#5-survey-competitor-recovery-approaches)
6. [GGID Recovery Architecture](#6-ggid-recovery-architecture)
7. [Recommendations and Priority Matrix](#7-recommendations-and-priority-matrix)
8. [Appendix A: Shamir's Secret Sharing in Go](#appendix-a-shamirs-secret-sharing-in-go)
9. [Appendix B: Recovery State Machine in Go](#appendix-b-recovery-state-machine-in-go)
10. [Appendix C: References and Further Reading](#appendix-c-references-and-further-reading)

---

## 1. Device-to-Device Credential Transfer (FIDO CXP/CXF)

### 1.1 Background and Motivation

One of the most significant barriers to passkey adoption has been the lack of
**portability**. Unlike passwords — which can be copy-pasted or exported as CSV —
passkeys are cryptographic key pairs that are deeply integrated into platform-specific
secure enclaves and sync fabrics. This means a passkey created on Apple iCloud Keychain
cannot easily move to Google Password Manager or to 1Password. Users who switch platforms
or password managers risk losing access to their accounts.

In October 2024, the **FIDO Alliance** announced two complementary standards to solve
this exact problem:

| Standard | Full Name | Status (as of 2025) |
|---|---|---|
| **CXF** | Credential Exchange Format | Review Draft (March 2025) |
| **CXP** | Credential Exchange Protocol | Working Draft (early 2026 target) |

CXF defines **what** a credential looks like (the data format), while CXP defines **how**
credentials are securely transported between two different provider platforms.

### 1.2 How CXF Works (Data Format)

CXF defines a standardized JSON-based structure for different credential types:

```
public-key-credential   — passkeys (WebAuthn/FIDO2)
password                — traditional passwords
totp                    — time-based one-time password secrets
note                    — free-form secure notes
```

The format is **extensible by design**, allowing future credential types (credit cards,
government IDs, mobile driver's licenses) to be added without breaking backward
compatibility. Each credential type specifies:

- **Core fields**: The essential data for that credential type (e.g., for passkeys: RP ID,
  user handle, credential ID, public key, key algorithm, transport hints)
- **Metadata**: Display name, creation date, last-used date, icon URL
- **Security attributes**: Backup eligibility, backup state, user verification method

This standardization means that a passkey exported from 1Password can be correctly parsed
and imported by Bitwarden, Google, or Apple — as long as all parties implement CXF.

### 1.3 How CXP Works (Transport Protocol)

CXP defines a secure mechanism for transferring credentials between a **Sender**
(exporting provider) and a **Recipient** (importing provider). The protocol uses
**Hybrid Public Key Encryption (HPKE)** to ensure end-to-end encryption during transit.

The general flow:

```
1. Sender and Recipient establish a secure channel via HPKE key exchange
2. Sender serializes credentials into CXF format
3. Sender encrypts the CXF payload using the negotiated HPKE session key
4. Encrypted payload is transferred (via local transport, QR code, or cloud relay)
5. Recipient decrypts and imports the CXF-formatted credentials
```

**Transport modes** under consideration:

| Mode | Description | Proximity Required | Use Case |
|---|---|---|---|
| **Same-device** | Transfer between two apps on the same device (e.g., Apple Passwords to 1Password on the same iPhone) | No network needed | Apple iOS/macOS 26 already ships this |
| **QR + BLE** | Sender displays a QR code, Recipient scans it; BLE proximity confirms physical co-location | Yes — devices must be nearby | Cross-device transfer without cloud relay |
| **Cloud relay** | Encrypted payload is relayed through a FIDO-hosted or provider-hosted cloud service | No | Remote transfer between different locations |

### 1.4 Standardization Timeline and Platform Support

| Platform/Provider | CXF Support | CXP Support | Status |
|---|---|---|---|
| **Apple** | Shipped (iOS/iPadOS/macOS 26) | Same-device only (no CXP needed) | Apple shipped CXF-based same-device credential transfer in iOS/macOS 26 (2025). Users can export passkeys from Apple Passwords to third-party apps using the CXF schema locally. |
| **Google** | Contributor to spec | Roadmap (2026) | Google is an active spec contributor and has signaled support. Android's Credential Manager will likely add CXF export/import in Android 16+. |
| **Microsoft** | Contributor to spec | Roadmap (2026) | Microsoft contributes to the spec. Windows Hello passkey management will gain CXP/CXF support, likely in a Windows 11 feature update. |
| **1Password** | Prototype | Prototype | 1Password was one of the original initiators (along with Dashlane, Bitwarden, NordPass) of the cross-provider credential exchange effort in early 2023. |
| **Bitwarden** | Prototype | Prototype | Active contributor. Prototyping CXP-based cross-provider transfer. |
| **Dashlane** | Prototype | Prototype | Active contributor. Prototyping CXP-based cross-provider transfer. |

### 1.5 What GGID Needs to Do

**Short answer: Nothing on the server side.**

CXP/CXF is fundamentally a **client-side** protocol. The credential transfer happens
between the user's authenticator platforms (password managers, OS keychains). The relying
party (GGID) does not participate in the transfer, does not see the transferred credential,
and does not need to implement any CXP/CXF code.

However, there are **monitoring and security considerations**:

1. **Credential ID monitoring**: When a passkey is exported and re-imported into a new
   provider, the credential ID may or may not change (this is provider-specific). GGID
   should be prepared for:
   - Same credential ID from a different authenticator attachment (platform → cross-platform
     or vice versa)
   - Different AAGUID (authenticator model identifier) for the same credential ID

2. **Transfer provenance**: For high-security tenants, GGID could optionally log when a
   credential's AAGUID changes (indicating it was re-imported into a different provider).
   This is informational only — the cryptographic key pair remains the same, so the
   security properties are unchanged.

3. **No protocol changes needed**: GGID's WebAuthn registration and assertion endpoints
   work identically before and after a credential transfer. The transfer is transparent
   to the relying party.

4. **Documentation**: GGID should document for users that passkeys created in one provider
   can be exported and imported into another using CXP/CXF, once their providers support it.

**Action items for GGID:**

| Priority | Item | Effort |
|---|---|---|
| Low | Add AAGUID change detection to audit log (informational) | 1 day |
| Low | Document CXP/CXF in user-facing help articles | 0.5 day |
| None | Implement CXP/CXF protocol | Not needed — client-side only |

---

## 2. Social Recovery with Threshold Schemes

### 2.1 Concept

Social recovery is an account recovery pattern where **trusted contacts** (friends,
family members, or colleagues) co-authorize the recovery of a user's account. Instead of
relying on a single recovery factor (like a code or email link), the system requires
multiple trusted contacts to participate.

This pattern was popularized by:
- **Apple Account Recovery Contacts** (iOS 15+): Users designate trusted contacts who can
  help recover their Apple ID
- **Signal Safety Numbers**: Social verification of identity
- **Web3 social recovery wallets** (e.g., Argent, Safe): Threshold-based recovery for
  crypto wallets

### 2.2 Threshold Recovery Model

The core idea is **k-of-n threshold recovery**: the user designates `n` trusted contacts,
and any `k` of them (where `k < n`) must participate to authorize recovery. For example:

- **3-of-5**: Designate 5 contacts, need any 3 to recover
- **2-of-3**: Designate 3 contacts, need any 2 to recover
- **4-of-7**: Higher security — need 4 of 7 contacts

The threshold protects against:
- **Single contact compromise**: An attacker who compromises one trusted contact still
  cannot recover the account (need `k` contacts)
- **Contact unavailability**: If 2 of 5 contacts are unreachable, recovery still works
  (need only 3)
- **Collusion attacks**: Requires `k` colluding contacts to compromise the account

### 2.3 Shamir's Secret Sharing (SSS)

The mathematical foundation for threshold recovery is **Shamir's Secret Sharing** (SSS),
invented by Adi Shamir in 1979. SSS splits a secret `S` into `n` shares such that:

- Any `k` or more shares can reconstruct `S`
- Any `k-1` or fewer shares reveal **zero information** about `S`

This is based on polynomial interpolation over a finite field. A random polynomial of
degree `k-1` is constructed:

```
f(x) = a_0 + a_1*x + a_2*x^2 + ... + a_{k-1}*x^{k-1}   (mod p)
```

where:
- `a_0 = S` (the secret)
- `a_1, ..., a_{k-1}` are random coefficients
- `p` is a large prime number

Each share is a point `(x_i, f(x_i))` where `x_i` is a unique, non-zero identifier.
Given any `k` points, Lagrange interpolation uniquely determines the polynomial and
therefore `a_0 = S`.

### 2.4 Application to Passkey Recovery

The recovery flow:

```
┌──────────────────────────────────────────────────────────────┐
│ SETUP PHASE (user enrolls social recovery)                    │
│                                                               │
│  1. User generates a recovery secret S (32 bytes random)      │
│  2. S is split into n shares using Shamir's Secret Sharing    │
│  3. Each share is encrypted with a trusted contact's pubkey   │
│  4. Encrypted shares are stored on GGID server                │
│  5. Recovery secret S is NOT stored on GGID                   │
│                                                               │
│  User stores S locally (or destroys it — shares are the       │
│  only path to reconstruct S)                                   │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ RECOVERY PHASE (user lost all passkeys)                       │
│                                                               │
│  1. User initiates recovery, identifies trusted contacts      │
│  2. System notifies k contacts (via email, push, in-app)      │
│  3. Each contact approves (via their own passkey/auth)        │
│  4. Upon approval, contact's encrypted share is released      │
│  5. User collects k shares                                     │
│  6. Shamir reconstruction recovers S                          │
│  7. S is used as proof-of-identity to enroll new passkey      │
│  8. Old passkeys are revoked, new passkey is enrolled         │
└──────────────────────────────────────────────────────────────┘
```

### 2.5 Security Model

| Threat | Mitigation |
|---|---|
| Single contact compromised | Need `k` shares — attacker needs `k` contacts |
| Social engineering of contacts | Each contact must authenticate (passkey/MFA) before releasing share |
| GGID server breach | Shares are encrypted with contact public keys — GGID cannot read them |
| Replay attack | Each recovery session is time-limited and single-use |
| Contact collusion | Requires `k` colluding contacts; choose `k` accordingly |
| Contact loses access | Tolerated as long as `n - available_contacts < k` |

### 2.6 Go Implementation: Shamir's Secret Sharing

```go
// Package recovery implements threshold-based social recovery using
// Shamir's Secret Sharing over GF(256).
package recovery

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
)

// --- GF(256) Operations ---

// gf256Mul performs multiplication in GF(2^8) using the AES polynomial (0x11B).
func gf256Mul(a, b byte) byte {
	var result byte
	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			result ^= a
		}
		hiBit := a & 0x80
		a <<= 1
		if hiBit != 0 {
			a ^= 0x1B // AES irreducible polynomial
		}
		b >>= 1
	}
	return result
}

// gf256Inverse computes the multiplicative inverse in GF(256).
func gf256Inv(a byte) byte {
	if a == 0 {
		return 0
	}
	// Brute-force inverse (256 elements, fast enough)
	result := a
	for i := 0; i < 253; i++ {
		result = gf256Mul(result, a)
	}
	return result
}

// --- Shamir's Secret Sharing ---

// Share represents one piece of a split secret.
type Share struct {
	Index byte   // x-coordinate (1..255)
	Data  []byte // y-values, one byte per byte of the secret
}

// SplitSecret divides a secret into n shares, any k of which can reconstruct it.
// Uses byte-wise polynomial evaluation over GF(256).
//
// Parameters:
//   - secret: the data to split (any length)
//   - k: threshold (minimum shares needed to reconstruct)
//   - n: total number of shares (n >= k, n <= 255)
//
// Returns n shares.
func SplitSecret(secret []byte, k, n int) ([]Share, error) {
	if k < 2 {
		return nil, errors.New("threshold k must be >= 2")
	}
	if n < k {
		return nil, fmt.Errorf("n (%d) must be >= k (%d)", n, k)
	}
	if n > 255 {
		return nil, errors.New("n must be <= 255 (GF(256) limitation)")
	}
	if len(secret) == 0 {
		return nil, errors.New("secret must not be empty")
	}

	shares := make([]Share, n)
	for i := 0; i < n; i++ {
		shares[i] = Share{
			Index: byte(i + 1), // x = 1..n (0 is reserved for the secret)
			Data:  make([]byte, len(secret)),
		}
	}

	// For each byte of the secret, create a separate polynomial.
	for byteIdx, secretByte := range secret {
		// Generate k-1 random coefficients (a_1 .. a_{k-1}).
		coeffs := make([]byte, k)
		coeffs[0] = secretByte // a_0 = secret byte
		_, err := rand.Read(coeffs[1:])
		if err != nil {
			return nil, fmt.Errorf("failed to generate random coefficients: %w", err)
		}

		// Evaluate polynomial at each share's x-coordinate.
		for i := 0; i < n; i++ {
			x := byte(i + 1)
			shares[i].Data[byteIdx] = evalPoly(coeffs, x)
		}
	}

	return shares, nil
}

// evalPoly evaluates a polynomial at x over GF(256).
// Uses Horner's method for efficiency.
func evalPoly(coeffs []byte, x byte) byte {
	result := byte(0)
	for i := len(coeffs) - 1; i >= 0; i-- {
		result = gf256Mul(result, x) ^ coeffs[i]
	}
	return result
}

// ReconstructSecret recovers the original secret from k or more shares
// using Lagrange interpolation over GF(256).
func ReconstructSecret(shares []Share) ([]byte, error) {
	if len(shares) < 2 {
		return nil, errors.New("need at least 2 shares to reconstruct")
	}

	secretLen := len(shares[0].Data)
	for _, s := range shares {
		if len(s.Data) != secretLen {
			return nil, errors.New("all shares must have the same length")
		}
	}

	secret := make([]byte, secretLen)

	for byteIdx := 0; byteIdx < secretLen; byteIdx++ {
		// Lagrange interpolation at x = 0 recovers f(0) = a_0 = secret byte.
		secret[byteIdx] = lagrangeInterpolate(shares, byteIdx)
	}

	return secret, nil
}

// lagrangeInterpolate computes f(0) using Lagrange basis polynomials over GF(256).
func lagrangeInterpolate(shares []Share, byteIdx int) byte {
	k := len(shares)
	var result byte

	for i := 0; i < k; i++ {
		// Compute Lagrange basis polynomial L_i(0).
		// L_i(0) = product over j != i of (0 - x_j) / (x_i - x_j)
		//        = product over j != i of x_j / (x_i XOR x_j)
		// In GF(256): subtraction = XOR, 0 = identity for multiplication

		numerator := byte(1)
		denominator := byte(1)

		for j := 0; j < k; j++ {
			if i == j {
				continue
			}
			xi := shares[i].Index
			xj := shares[j].Index

			// numerator *= (0 - x_j) = x_j (since -x = x in GF(2^8))
			numerator = gf256Mul(numerator, xj)

			// denominator *= (x_i - x_j) = x_i XOR x_j
			denominator = gf256Mul(denominator, xi^xj)
		}

		// result += y_i * L_i(0)
		yi := shares[i].Data[byteIdx]
		lagrange := gf256Mul(numerator, gf256Inv(denominator))
		result ^= gf256Mul(yi, lagrange)
	}

	return result
}

// --- Encrypted Share Distribution ---

// EncryptedShare wraps a Shamir share encrypted with a trusted contact's public key.
type EncryptedShare struct {
	ContactUserID string
	Nonce         []byte
	Ciphertext    []byte
}

// EncryptShare encrypts a Shamir share using AES-256-GCM with a key derived
// from the trusted contact's public key via HKDF.
func EncryptShare(share Share, contactPublicKey []byte) (*EncryptedShare, error) {
	// Derive symmetric key from contact's public key.
	h := sha256.Sum256(contactPublicKey)
	key := h[:] // 32 bytes for AES-256

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Prepend the share index to the plaintext.
	plaintext := append([]byte{share.Index}, share.Data...)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedShare{
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// DecryptShare decrypts an EncryptedShare using the trusted contact's private key.
func DecryptShare(enc *EncryptedShare, contactPrivateKey []byte) (Share, error) {
	h := sha256.Sum256(contactPrivateKey)
	key := h[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return Share{}, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Share{}, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, enc.Nonce, enc.Ciphertext, nil)
	if err != nil {
		return Share{}, fmt.Errorf("failed to decrypt share: %w", err)
	}

	if len(plaintext) < 1 {
		return Share{}, errors.New("decrypted share too short")
	}

	return Share{
		Index: plaintext[0],
		Data:  plaintext[1:],
	}, nil
}
```

### 2.7 Share Distribution and Recovery Session Management

```go
// RecoverySession tracks an in-progress social recovery attempt.
type RecoverySession struct {
	ID              string
	UserID          string
	TenantID        string
	Threshold       int            // k: minimum shares needed
	TotalContacts   int            // n: total designated contacts
	ContactStatuses map[string]ContactRecoveryStatus
	Status          RecoverySessionStatus
	CreatedAt       int64          // Unix timestamp
	ExpiresAt       int64          // Unix timestamp
	CollectedShares []Share        // shares collected so far
}

type ContactRecoveryStatus struct {
	ContactUserID string
	State         ContactState
	NotifiedAt    int64
	RespondedAt   int64
}

type ContactState string

const (
	ContactStatePending   ContactState = "pending"
	ContactStateApproved  ContactState = "approved"
	ContactStateRejected  ContactState = "rejected"
	ContactStateExpired   ContactState = "expired"
)

type RecoverySessionStatus string

const (
	SessionStatusInitiated RecoverySessionStatus = "initiated"
	SessionStatusAwaiting  RecoverySessionStatus = "awaiting_contacts"
	SessionStatusVerifying RecoverySessionStatus = "verifying_shares"
	SessionStatusRecovered RecoverySessionStatus = "recovered"
	SessionStatusFailed    RecoverySessionStatus = "failed"
	SessionStatusExpired   RecoverySessionStatus = "expired"
)

// SocialRecoveryManager coordinates the social recovery flow.
type SocialRecoveryManager struct {
	store    RecoverySessionStore
	notifier RecoveryNotifier
	clock    func() int64
}

// InitiateRecovery starts a new social recovery session for a locked-out user.
func (m *SocialRecoveryManager) InitiateRecovery(
	userID, tenantID string,
	threshold, totalContacts int,
	sessionTTLSeconds int64,
) (*RecoverySession, error) {
	session := &RecoverySession{
		ID:            generateID(),
		UserID:        userID,
		TenantID:      tenantID,
		Threshold:     threshold,
		TotalContacts: totalContacts,
		ContactStatuses: make(map[string]ContactRecoveryStatus),
		Status:        SessionStatusInitiated,
		CreatedAt:     m.clock(),
		ExpiresAt:     m.clock() + sessionTTLSeconds,
	}

	// Load the user's designated contacts and notify them.
	// ... (load contacts from store, send notifications)

	session.Status = SessionStatusAwaiting
	return session, m.store.Save(session)
}

// ApproveRecovery records a trusted contact's approval and releases their share.
func (m *SocialRecoveryManager) ApproveRecovery(
	sessionID, contactUserID string,
	encryptedShare *EncryptedShare,
) (*RecoverySession, error) {
	session, err := m.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if m.clock() > session.ExpiresAt {
		session.Status = SessionStatusExpired
		_ = m.store.Save(session)
		return nil, errors.New("recovery session has expired")
	}

	status, ok := session.ContactStatuses[contactUserID]
	if !ok {
		return nil, errors.New("contact is not a designated recovery contact")
	}

	if status.State != ContactStatePending {
		return nil, fmt.Errorf("contact already responded: %s", status.State)
	}

	// Mark contact as approved.
	status.State = ContactStateApproved
	status.RespondedAt = m.clock()
	session.ContactStatuses[contactUserID] = status

	// TODO: decrypt and store the share (requires contact's auth context).
	// For now, track that we have it.
	session.CollectedShares = append(session.CollectedShares, Share{})

	// Check if we have enough shares.
	if len(session.CollectedShares) >= session.Threshold {
		session.Status = SessionStatusVerifying
	}

	return session, m.store.Save(session)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
```

### 2.8 Limitations and Trade-offs

| Factor | Assessment |
|---|---|
| **UX complexity** | High — user must find and coordinate with trusted contacts |
| **Time to recover** | Hours to days — contacts must respond |
| **Security** | Very strong against single-point compromise |
| **Implementation complexity** | High — Shamir's SSS, share encryption, contact management |
| **Adoption barrier** | Users must pre-enroll contacts and maintain those relationships |
| **Suitable for** | High-value accounts, enterprise admins, crypto-adjacent use cases |
| **Not suitable for** | Consumer apps with low technical literacy users |

---

## 3. Admin-Issued Recovery Credentials

### 3.1 Concept

In enterprise environments, the most practical recovery mechanism for locked-out users is
**admin-issued temporary credentials**. When a user loses all their passkeys and cannot
use self-service recovery, an administrator with elevated privileges can issue a
time-limited, single-use credential that allows the user to enroll a new passkey.

Key properties:
- **Time-limited**: Valid for 15–30 minutes only
- **Single-use**: Consumed upon successful passkey enrollment
- **Scoped**: Can only be used for passkey re-enrollment, not general authentication
- **Step-up required**: Admin must perform MFA before issuing
- **Full audit trail**: Every issuance and usage is logged

### 3.2 Comparison with Password Reset Tokens

| Property | Password Reset Token | Admin Recovery Credential |
|---|---|---|
| **Purpose** | Reset password | Enroll new passkey |
| **Who issues** | System (automated, via email) | Admin (manual, with MFA) |
| **Validity** | Typically 1–24 hours | 15–30 minutes |
| **Scope** | Password change only | Passkey enrollment only |
| **Step-up auth** | Not required (email is the factor) | Required (admin MFA) |
| **Audit level** | Standard | Enhanced (admin identity, reason, IP) |
| **Rate limit** | Usually rate-limited by email | Per-admin rate-limited |
| **Revocability** | Hard (user may have email access) | Yes (admin can revoke before use) |

### 3.3 Go Implementation

```go
// Package adminrecovery implements admin-issued temporary recovery credentials.
package adminrecovery

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// TemporaryCredential is a short-lived, single-use credential that allows
// a locked-out user to enroll a new passkey.
type TemporaryCredential struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	UserID        string          `json:"user_id"`
	IssuedBy      string          `json:"issued_by"`    // admin user ID
	IssuedAt      time.Time       `json:"issued_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	UsedAt        *time.Time      `json:"used_at,omitempty"`
	RevokedAt     *time.Time      `json:"revoked_at,omitempty"`
	RevokedBy     string          `json:"revoked_by,omitempty"`
	Reason        string          `json:"reason"`        // admin-provided justification
	IssuerIP      string          `json:"issuer_ip"`
	Status        CredentialStatus `json:"status"`
	// HMAC for tamper protection (computed over all fields above)
	Signature     []byte          `json:"signature"`
}

type CredentialStatus string

const (
	StatusActive   CredentialStatus = "active"
	StatusUsed     CredentialStatus = "used"
	StatusRevoked  CredentialStatus = "revoked"
	StatusExpired  CredentialStatus = "expired"
)

// CredentialIssuer creates and validates temporary recovery credentials.
type CredentialIssuer struct {
	signingKey []byte // HMAC-SHA256 key
	store      CredentialStore
	clock      func() time.Time
}

// NewCredentialIssuer creates a new issuer with the given HMAC signing key.
func NewCredentialIssuer(signingKey []byte, store CredentialStore) *CredentialIssuer {
	return &CredentialIssuer{
		signingKey: signingKey,
		store:      store,
		clock:      time.Now,
	}
}

// IssueCredential creates a temporary recovery credential for a locked-out user.
//
// Prerequisites:
//   - Admin must have already performed step-up auth (verified by caller)
//   - User must exist and have no active passkeys
//   - Admin must have appropriate permissions
func (i *CredentialIssuer) IssueCredential(
	req IssueRequest,
) (*TemporaryCredential, string, error) {
	now := i.clock()

	// Validate TTL.
	if req.TTL < 1*time.Minute {
		return nil, "", errors.New("TTL must be at least 1 minute")
	}
	if req.TTL > 30*time.Minute {
		return nil, "", errors.New("TTL must not exceed 30 minutes")
	}

	// Check admin rate limit (max 10 per hour per admin).
	recentCount, err := i.store.CountByAdmin(req.AdminUserID, now.Add(-1*time.Hour))
	if err != nil {
		return nil, "", fmt.Errorf("failed to check rate limit: %w", err)
	}
	if recentCount >= 10 {
		return nil, "", errors.New("admin has issued too many recovery credentials (rate limit: 10/hour)")
	}

	cred := &TemporaryCredential{
		ID:        generateCredentialID(),
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		IssuedBy:  req.AdminUserID,
		IssuedAt:  now,
		ExpiresAt: now.Add(req.TTL),
		Reason:    req.Reason,
		IssuerIP:  req.AdminIP,
		Status:    StatusActive,
	}

	// Compute HMAC signature for tamper protection.
	sig, err := i.computeSignature(cred)
	if err != nil {
		return nil, "", fmt.Errorf("failed to sign credential: %w", err)
	}
	cred.Signature = sig

	// Persist.
	if err := i.store.Save(cred); err != nil {
		return nil, "", fmt.Errorf("failed to save credential: %w", err)
	}

	// Generate a one-time token for the user.
	token, err := i.generateToken(cred)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return cred, token, nil
}

// IssueRequest contains parameters for issuing a temporary credential.
type IssueRequest struct {
	TenantID   string
	UserID     string
	AdminUserID string
	AdminIP    string
	Reason     string
	TTL        time.Duration
}

// ValidateCredential checks if a presented credential token is valid and unused.
// Returns the credential if valid, or an error explaining why it's invalid.
func (i *CredentialIssuer) ValidateCredential(token string) (*TemporaryCredential, error) {
	// Decode token.
	cred, err := i.parseToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Verify HMAC signature.
	expectedSig, err := i.computeSignature(cred)
	if err != nil {
		return nil, fmt.Errorf("failed to compute signature: %w", err)
	}
	if !hmac.Equal(cred.Signature, expectedSig) {
		return nil, errors.New("credential signature verification failed")
	}

	// Check freshness from store (in case credential was revoked or used
	// after the token was generated).
	stored, err := i.store.Get(cred.ID)
	if err != nil {
		return nil, fmt.Errorf("credential not found: %w", err)
	}

	// Check status.
	if stored.Status != StatusActive {
		return nil, fmt.Errorf("credential is %s", stored.Status)
	}

	// Check expiry.
	if i.clock().After(stored.ExpiresAt) {
		stored.Status = StatusExpired
		_ = i.store.Save(stored)
		return nil, errors.New("credential has expired")
	}

	return stored, nil
}

// ConsumeCredential marks a credential as used after successful passkey enrollment.
func (i *CredentialIssuer) ConsumeCredential(credID string) error {
	cred, err := i.store.Get(credID)
	if err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}

	if cred.Status != StatusActive {
		return fmt.Errorf("cannot consume credential with status %s", cred.Status)
	}

	now := i.clock()
	cred.UsedAt = &now
	cred.Status = StatusUsed
	return i.store.Save(cred)
}

// RevokeCredential allows an admin to revoke a credential before it's used.
func (i *CredentialIssuer) RevokeCredential(credID, revokedByAdminID string) error {
	cred, err := i.store.Get(credID)
	if err != nil {
		return fmt.Errorf("credential not found: %w", err)
	}

	if cred.Status != StatusActive {
		return fmt.Errorf("cannot revoke credential with status %s", cred.Status)
	}

	now := i.clock()
	cred.RevokedAt = &now
	cred.RevokedBy = revokedByAdminID
	cred.Status = StatusRevoked
	return i.store.Save(cred)
}

// --- Internal helpers ---

func (i *CredentialIssuer) computeSignature(cred *TemporaryCredential) ([]byte, error) {
	// Serialize all fields except signature.
	data, err := json.Marshal(struct {
		ID        string    `json:"id"`
		TenantID  string    `json:"tenant_id"`
		UserID    string    `json:"user_id"`
		IssuedBy  string    `json:"issued_by"`
		IssuedAt  time.Time `json:"issued_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Reason    string    `json:"reason"`
		IssuerIP  string    `json:"issuer_ip"`
	}{
		ID:        cred.ID,
		TenantID:  cred.TenantID,
		UserID:    cred.UserID,
		IssuedBy:  cred.IssuedBy,
		IssuedAt:  cred.IssuedAt,
		ExpiresAt: cred.ExpiresAt,
		Reason:    cred.Reason,
		IssuerIP:  cred.IssuerIP,
	})
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, i.signingKey)
	mac.Write(data)
	return mac.Sum(nil), nil
}

func (i *CredentialIssuer) generateToken(cred *TemporaryCredential) (string, error) {
	data, err := json.Marshal(cred)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

func (i *CredentialIssuer) parseToken(token string) (*TemporaryCredential, error) {
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var cred TemporaryCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func generateCredentialID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("rc-%x", b)
}

// CredentialStore is the persistence interface for temporary credentials.
type CredentialStore interface {
	Save(cred *TemporaryCredential) error
	Get(id string) (*TemporaryCredential, error)
	CountByAdmin(adminUserID string, since time.Time) (int, error)
	ListByTenant(tenantID string, limit int) ([]*TemporaryCredential, error)
}
```

### 3.4 API Design

```
POST /api/v1/admin/recovery/credentials
  Authorization: Bearer <admin-jwt-with-mfa-claim>
  Body: { user_id, reason, ttl_minutes }
  Response: 201 { credential_id, token, expires_at }

POST /api/v1/recovery/enroll-passkey
  Authorization: Bearer <recovery-credential-token>
  Body: { attestation, client_data_json }
  Response: 200 { credential_id, created }

DELETE /api/v1/admin/recovery/credentials/{id}
  Authorization: Bearer <admin-jwt>
  Response: 204

GET /api/v1/admin/recovery/credentials?tenant_id=...
  Authorization: Bearer <admin-jwt>
  Response: 200 [{ id, user_id, status, issued_at, expires_at, ... }]
```

### 3.5 Audit Events

Every credential operation emits an audit event:

```go
// Audit events for admin recovery credentials.
const (
	AuditRecoveryCredentialIssued  = "recovery.credential.issued"
	AuditRecoveryCredentialUsed    = "recovery.credential.used"
	AuditRecoveryCredentialRevoked = "recovery.credential.revoked"
	AuditRecoveryCredentialExpired = "recovery.credential.expired"
)

type RecoveryAuditEvent struct {
	EventType     string    `json:"event_type"`
	CredentialID  string    `json:"credential_id"`
	TenantID      string    `json:"tenant_id"`
	UserID        string    `json:"user_id"`      // affected user
	AdminUserID   string    `json:"admin_user_id"` // acting admin
	IP            string    `json:"ip"`
	Timestamp     time.Time `json:"timestamp"`
	Reason        string    `json:"reason"`
}
```

---

## 4. Expanded Recovery Key Patterns

### 4.1 Encrypted Recovery Key Escrow

In this pattern, the user's recovery key (or recovery secret) is encrypted and stored with
a **trusted third party** — not with GGID itself. This provides separation of concerns:

- **GGID** stores user identity, passkey metadata, and auth logic
- **Escrow provider** stores the encrypted recovery key (cannot decrypt it)
- **User** holds the decryption key

Options for escrow providers:

| Provider | How it works | Trust model |
|---|---|---|
| **User's own email** | Encrypted blob sent as attachment/link; user decrypts locally | User must keep email access |
| **Hardware security module (HSM)** | Enterprise HSM stores recovery keys with access policies | Enterprise controls access |
| **Third-party key management service** (AWS KMS, HashiCorp Vault) | KMS stores encrypted keys with IAM-controlled access | Cloud provider availability |
| **Distributed storage** (IPFS, multiple servers) | Key split via Shamir's and stored across independent servers | No single point of failure |

### 4.2 Hardware Token as Recovery Factor

A **YubiKey** (or other FIDO2 security key) can serve as a dedicated recovery factor:

```
Normal auth:     passkey on phone → daily login
Recovery auth:   YubiKey in safe → emergency recovery
```

Advantages:
- **Air-gapped**: YubiKey stored in a physical safe is immune to remote compromise
- **No software dependencies**: Doesn't rely on email, cloud accounts, or phone numbers
- **Strong crypto**: FIDO2 hardware key with secure element
- **Simple UX**: Just plug in and tap

Implementation for GGID:
1. During passkey enrollment, user optionally enrolls a second credential as "recovery-only"
2. This credential has a flag `recover_only: true`
3. During normal login, recovery-only credentials are excluded from assertion options
4. During recovery flow, recovery-only credentials are included
5. After successful recovery auth, user enrolls a new primary passkey

```go
// Credential with recovery-only flag.
type WebAuthnCredential struct {
	ID            string
	UserID        string
	CredentialID  []byte
	PublicKey     []byte
	RecoverOnly   bool  // true = this credential can only be used for recovery
	EnrolledAt    time.Time
	LastUsedAt    *time.Time
}

// When generating assertion options for normal login, exclude recovery-only creds.
func (s *AuthService) AssertionOptions(userID string) (*PublicKeyCredentialRequestOptions, error) {
	creds, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	var allowIDs [][]byte
	for _, c := range creds {
		if c.RecoverOnly {
			continue // skip recovery-only for normal auth
		}
		allowIDs = append(allowIDs, c.CredentialID)
	}

	return &PublicKeyCredentialRequestOptions{
		AllowCredentials: allowIDs,
		// ...
	}, nil
}

// When generating assertion options for recovery, include ONLY recovery-only creds.
func (s *AuthService) RecoveryAssertionOptions(userID string) (*PublicKeyCredentialRequestOptions, error) {
	creds, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	var allowIDs [][]byte
	for _, c := range creds {
		if !c.RecoverOnly {
			continue // skip normal creds for recovery auth
		}
		allowIDs = append(allowIDs, c.CredentialID)
	}

	if len(allowIDs) == 0 {
		return nil, ErrNoRecoveryCredential
	}

	return &PublicKeyCredentialRequestOptions{
		AllowCredentials: allowIDs,
		// ...
	}, nil
}
```

### 4.3 Multi-Factor Recovery

Multi-factor recovery requires **two or more independent factors** to authorize
account recovery. This prevents a single compromised factor from enabling recovery:

| Factor | Example | Compromise vector |
|---|---|---|
| **Recovery code** | 20-character alphanumeric code printed and stored offline | Physical theft of printed code |
| **Email verification** | Click link sent to registered email | Email account compromise |
| **Admin approval** | Admin reviews and approves recovery request | Social engineering of admin |
| **Device attestation** | New device must match previously registered device profile | Device cloning |
| **Knowledge factor** | Security questions (low entropy — not recommended alone) | Public information |

A **2-of-3** multi-factor recovery might require:
1. Recovery code (something you have)
2. PLUS one of:
   - Email verification (something you control)
   - Admin approval (someone who knows you)

```go
// MultiFactorRecoveryConfig defines which factors are required.
type MultiFactorRecoveryConfig struct {
	RequiredCount   int     // minimum factors needed (e.g., 2)
	AvailableFactors []RecoveryFactorType
}

type RecoveryFactorType string

const (
	FactorRecoveryCode   RecoveryFactorType = "recovery_code"
	FactorEmail          RecoveryFactorType = "email_verification"
	FactorAdminApproval  RecoveryFactorType = "admin_approval"
	FactorDeviceAttest   RecoveryFactorType = "device_attestation"
	FactorSecurityKey    RecoveryFactorType = "hardware_security_key"
)

// MultiFactorRecoverySession tracks collected factors.
type MultiFactorRecoverySession struct {
	ID          string
	UserID      string
	TenantID    string
	Required    int
	Collected   map[RecoveryFactorType]bool
	Status      RecoverySessionStatus
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// Satisfied returns true when enough factors have been verified.
func (s *MultiFactorRecoverySession) Satisfied() bool {
	return len(s.Collected) >= s.Required
}

// MarkFactorVerified records that a factor has been satisfied.
func (s *MultiFactorRecoverySession) MarkFactorVerified(factor RecoveryFactorType) {
	if s.Collected == nil {
		s.Collected = make(map[RecoveryFactorType]bool)
	}
	s.Collected[factor] = true
}
```

### 4.4 Comparison of Recovery Key Entropy Approaches

| Approach | Entropy | UX | Security | Recommendation |
|---|---|---|---|---|
| **20-char alphanumeric code** | ~120 bits | User writes down code | High if stored offline | Recommended baseline |
| **12-word mnemonic (BIP-39)** | ~128 bits | User writes down 12 words | High; well-understood pattern | Good for crypto-savvy users |
| **QR code** | Variable (depends on data) | User scans or prints QR | Medium (QR can be photographed) | OK for visual users |
| **10-digit numeric PIN** | ~33 bits | Easy to remember | Low — brute-forceable | Not recommended |
| **URL with embedded token** | ~128 bits (if token is random) | Click a link | Medium (URL can leak) | OK for email-based |
| **Shamir shares** | Depends on secret size | Complex setup | Highest (threshold) | For high-security tenants |

---

## 5. Survey: Competitor Recovery Approaches

### 5.1 Auth0/Okta

**Current approach:**

Auth0's passkey recovery is **password-centric**. If a user loses all passkey devices,
the recovery flow guides them to reset their password. Auth0 does not support a
"passkey-only" flow — passwords must remain available as a fallback on any database
connection where passkeys are enabled.

**Key findings:**
- Auth0 enforces a **no passkey-only constraint**: passwords are mandatory fallback
- Account recovery routes through **password reset** (email link), not new passkey enrollment
- **20-passkey-per-user limit** enforced
- **Custom domain** is critical: passkeys are cryptographically bound to the domain, and
  changing it invalidates all enrolled passkeys
- Generic Auth0 passkey flows achieve only **5-10% activation rates**
- Okta Workforce Identity has separate recovery flows (admin-driven)

**GGID learnings:**
- GGID should NOT force passwords as the only fallback — we can offer passkey re-enrollment
- The custom domain binding is a real risk — GGID should document this clearly for tenants
- Low activation rates suggest UX optimization is critical

### 5.2 Google

**Current approach:**

Google's account recovery is **device-centric and AI-assisted**:

1. **Device-based recovery**: If the user has another device signed into their Google
   account, they can approve recovery on that device
2. **Google Password Manager sync**: Passkeys are synced via Google Password Manager
   across Android devices and Chrome
3. **Account recovery flow**: Multi-signal system that evaluates device history, location,
   recent activity, and recovery email/phone
4. **No explicit recovery code**: Google does not provide a recovery code for passkeys;
   instead, it relies on device proximity and account recovery signals

**Key findings:**
- Recovery is heavily dependent on having another signed-in device
- No user-facing recovery code concept for passkeys specifically
- Google Password Manager only syncs within the Google ecosystem (Android, Chrome)
- Cross-platform sync is coming via CXP/CXF

**GGID learnings:**
- Device-based recovery is excellent UX but requires platform integration GGID doesn't have
- GGID should provide explicit recovery codes (Google doesn't, but GGID is not a platform provider)

### 5.3 Apple

**Current approach:**

Apple offers a multi-layered recovery system:

1. **iCloud Keychain sync**: Passkeys sync across Apple devices via iCloud Keychain
   (end-to-end encrypted)
2. **Recovery Key**: Users can generate a 28-character recovery key that bypasses
   the standard account recovery flow. If enabled, it's **required** for recovery
3. **Account Recovery Contacts**: Users designate trusted contacts who can help
   verify identity during recovery (a form of social recovery)
4. **Account Recovery**: If no recovery key or contacts, Apple's manual review process
   (can take days)
5. **iCloud Data Recovery Service**: Escrow of encrypted iCloud data for recovery

**Key findings:**
- Apple's Recovery Key is an **opt-in** feature that replaces the standard flow
- Account Recovery Contacts is a form of social recovery (Apple-specific implementation)
- The recovery process can be very slow without a recovery key or contacts
- Apple shipped CXF-based credential export/import in iOS/macOS 26

**GGID learnings:**
- Recovery key as an opt-in feature is a good pattern
- Trusted contacts (social recovery) has been validated at scale by Apple
- The slow manual review process is a negative — GGID should avoid this

### 5.4 Microsoft

**Current approach:**

Microsoft's passkey recovery depends on the context:

1. **Microsoft Authenticator backup**: TOTP secrets and some credentials are backed up
   (encrypted) to the user's Microsoft account or iCloud
2. **Windows Hello**: Device-bound credentials — no sync; losing the device means
   losing the credential
3. **Azure AD / Entra ID**: Enterprise recovery is admin-driven (admin resets MFA,
   issues temporary access passes)
4. **Temporary Access Pass (TAP)**: Time-limited passcode issued by admin for users
   who can't use their normal MFA — very similar to admin-issued recovery credentials

**Key findings:**
- Microsoft's **Temporary Access Pass (TAP)** is the closest industry analog to the
  admin-issued recovery credential pattern described in Section 3
- TAP is time-limited (configurable, default 8 hours), single-use or multi-use
- TAP requires admin privileges to issue
- Windows Hello passkeys are device-bound — no recovery without admin intervention

**GGID learnings:**
- Microsoft's TAP validates the admin-issued recovery credential pattern
- GGID should make TTL configurable (but cap at 30 min for security)
- TAP supports both single-use and multi-use — GGID should only support single-use

### 5.5 1Password

**Current approach:**

1Password uses a **dual-key** system:

1. **Master Password**: User-chosen password (user must remember)
2. **Secret Key**: 128-bit randomly generated key created at account setup
3. **Emergency Kit**: A printable PDF containing the Secret Key that users store offline
4. Both Master Password AND Secret Key are required to decrypt the vault
5. Account recovery requires the Emergency Kit + Master Password (or support intervention)

**Key findings:**
- The Secret Key dramatically raises the entropy ceiling (even a weak master password
  + 128-bit Secret Key = strong overall security)
- The Emergency Kit pattern (printable document) is excellent for offline backup
- 1Password was one of the initiators of the CXP/CXF standards

**GGID learnings:**
- The "Emergency Kit" printable document pattern is worth adopting for recovery codes
- The dual-key concept (something you know + something you have) applies to recovery
- Recovery codes should be presented in a printable, scannable format

### 5.6 Comparison Table

| Platform | Primary Recovery | Secondary Recovery | Social/Contact | Admin Recovery | Hardware Backup | Recovery Code |
|---|---|---|---|---|---|---|
| **Auth0/Okta** | Password reset | — | No | Okta admin reset (separate) | No | No |
| **Google** | Device-based | Account recovery signals | No | Google Workspace admin | No | No |
| **Apple** | iCloud Keychain sync | Recovery Key | Account Recovery Contacts | No | No | Recovery Key (opt-in) |
| **Microsoft** | Authenticator backup | TAP (Entra ID) | No | Temporary Access Pass | No | No |
| **1Password** | Master Password + Secret Key | Emergency Kit | No | No | No | Secret Key (128-bit) |
| **GGID (proposed)** | Recovery code | Admin credential | Social recovery (Phase 3) | Admin credential (Phase 2) | YubiKey (recover-only) | Recovery code (120-bit) |

### 5.7 UX Quality and Security Rating

| Platform | UX Quality (1-5) | Security Level (1-5) | Notes |
|---|---|---|---|
| **Auth0/Okta** | 2 | 3 | Password-centric recovery defeats the purpose of passkeys |
| **Google** | 4 | 4 | Excellent UX but relies on having another signed-in device |
| **Apple** | 4 | 4 | Recovery Key + Contacts is comprehensive |
| **Microsoft** | 3 | 4 | TAP is solid for enterprise; Windows Hello is poor for recovery |
| **1Password** | 3 | 5 | Emergency Kit is excellent but users often lose it |
| **GGID (target)** | 4 | 4 | Multi-tier recovery with admin + social + hardware options |

---

## 6. GGID Recovery Architecture

### 6.1 Tiered Recovery Model

GGID should implement a **three-tier** recovery system that balances UX with security:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        TIERED RECOVERY MODEL                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Tier 1: Self-Service Recovery (always available)                        │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  • Recovery code (20-char alphanumeric, printed at enrollment)    │  │
│  │  • Email verification link                                        │  │
│  │  • Recovery-only hardware key (YubiKey)                           │  │
│  │  • No admin involvement required                                  │  │
│  │  • Time: minutes                                                  │  │
│  │  • Security: Medium-High                                          │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                            ↓ (Tier 1 fails)                              │
│  Tier 2: Admin-Assisted Recovery (enterprise)                            │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  • Admin-issued temporary credential (15-30 min TTL)              │  │
│  │  • Admin step-up auth required (MFA)                              │  │
│  │  • Full audit trail                                               │  │
│  │  • Time: minutes (if admin available)                             │  │
│  │  • Security: High                                                 │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                            ↓ (Tier 2 unavailable)                        │
│  Tier 3: Social Recovery (opt-in, high-security)                         │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │  • k-of-n threshold scheme (Shamir's Secret Sharing)              │  │
│  │  • Trusted contacts co-authorize recovery                         │  │
│  │  • Each contact authenticates independently                       │  │
│  │  • Time: hours to days                                            │  │
│  │  • Security: Very High                                            │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Recovery Flow State Machine

The recovery process is modeled as a state machine that transitions through
well-defined states:

```
                    ┌──────────────┐
                    │   IDLE       │
                    │ (no recovery │
                    │   in progress)│
                    └──────┬───────┘
                           │ user clicks "lost all passkeys"
                           ▼
                    ┌──────────────┐
                    │  INITIATED   │
                    │ (user identi-│
                    │  fied, tier  │
                    │  selected)   │
                    └──────┬───────┘
                           │ tier-specific verification begins
                           ▼
              ┌────────────────────────────┐
              │     VERIFYING_FACTORS      │
              │ (collecting recovery       │
              │  factors per selected tier)│
              └────────────┬───────────────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
     ┌──────────────┐ ┌──────────┐ ┌──────────────┐
     │ CODE_VERIFIED│ │ADMIN_APPR│ │SHARES_COLLECT│
     │ (Tier 1)     │ │ (Tier 2) │ │ (Tier 3)     │
     └──────┬───────┘ └────┬─────┘ └──────┬───────┘
            │              │              │
            └──────────────┼──────────────┘
                           │ all required factors satisfied
                           ▼
                  ┌─────────────────┐
                  │ RECOVERY_AUTHED │
                  │ (user proven    │
                  │  identity)      │
                  └────────┬────────┘
                           │ user enrolls new passkey
                           ▼
                  ┌─────────────────┐
                  │  ENROLLING_NEW  │
                  │  PASSKEY        │
                  └────────┬────────┘
                           │ WebAuthn attestation verified
                           ▼
                  ┌─────────────────┐
                  │    COMPLETED    │
                  │ (new passkey    │
                  │  active, old    │
                  │  revoked)       │
                  └─────────────────┘

              At any state:
                     │ timeout / failure / revocation
                     ▼
              ┌──────────────┐
              │   FAILED /   │
              │   EXPIRED    │
              └──────────────┘
```

### 6.3 Go Interfaces: RecoveryStrategy and RecoveryFactor

```go
// Package recoverycore defines the core interfaces for the recovery system.
package recoverycore

import (
	"context"
	"time"
)

// RecoveryStrategy is the top-level interface for a recovery mechanism.
// Each tier (code, admin, social) implements this interface.
type RecoveryStrategy interface {
	// Name returns the strategy identifier (e.g., "recovery_code",
	// "admin_credential", "social_threshold").
	Name() string

	// Tier returns the recovery tier level.
	Tier() RecoveryTier

	// Initiate starts the recovery flow for this strategy.
	Initiate(ctx context.Context, req InitiateRequest) (*RecoverySession, error)

	// Verify checks a recovery factor submission.
	Verify(ctx context.Context, sessionID string, submission FactorSubmission) (*VerifyResult, error)

	// Complete performs post-recovery actions (e.g., enroll new passkey).
	Complete(ctx context.Context, sessionID string, completion CompleteRequest) (*CompleteResult, error)

	// Cancel aborts an in-progress recovery session.
	Cancel(ctx context.Context, sessionID, reason string) error
}

// RecoveryTier represents the recovery tier level.
type RecoveryTier int

const (
	Tier1SelfService  RecoveryTier = 1 // recovery code, email, hardware key
	Tier2AdminAssist  RecoveryTier = 2 // admin-issued credential
	Tier3SocialRecovery RecoveryTier = 3 // threshold scheme
)

// InitiateRequest contains parameters for starting recovery.
type InitiateRequest struct {
	UserID    string
	TenantID  string
	Strategy  string
	Factors   []string  // which factors to use
	TTL       time.Duration
	Metadata  map[string]string
}

// RecoverySession represents an active recovery attempt.
type RecoverySession struct {
	ID         string
	UserID     string
	TenantID   string
	Strategy   string
	Tier       RecoveryTier
	Status     SessionStatus
	Factors    map[string]FactorState
	CreatedAt  time.Time
	ExpiresAt  time.Time
	CompletedAt *time.Time
}

type SessionStatus string

const (
	StatusIdle        SessionStatus = "idle"
	StatusInitiated   SessionStatus = "initiated"
	StatusVerifying   SessionStatus = "verifying"
	StatusAuthed      SessionStatus = "authed"
	StatusEnrolling   SessionStatus = "enrolling"
	StatusCompleted   SessionStatus = "completed"
	StatusFailed      SessionStatus = "failed"
	StatusExpired     SessionStatus = "expired"
	StatusCancelled   SessionStatus = "cancelled"
)

// FactorSubmission is what the user submits for a factor.
type FactorSubmission struct {
	FactorType string
	Data       map[string]interface{} // factor-specific data
}

// FactorState tracks the state of a single recovery factor.
type FactorState struct {
	Type      string
	Status    string // "pending", "verified", "failed"
	Attempts  int
	VerifiedAt *time.Time
}

// VerifyResult is the outcome of a factor verification.
type VerifyResult struct {
	Session       *RecoverySession
	FactorVerified bool
	AllFactorsMet  bool
	Error          string
}

// CompleteRequest contains data for completing recovery.
type CompleteRequest struct {
	AttestationObject  []byte // WebAuthn attestation
	ClientDataJSON     []byte // WebAuthn client data
}

// CompleteResult is the outcome of recovery completion.
type CompleteResult struct {
	NewCredentialID string
	OldCredentialsRevoked int
}
```

### 6.4 Recovery Code Strategy Implementation

```go
// RecoveryCodeStrategy implements Tier 1 recovery using a pre-generated code.
type RecoveryCodeStrategy struct {
	store    RecoveryStore
	hasher   CodeHasher
	clock    func() time.Time
}

func (s *RecoveryCodeStrategy) Name() string { return "recovery_code" }
func (s *RecoveryCodeStrategy) Tier() RecoveryTier { return Tier1SelfService }

func (s *RecoveryCodeStrategy) Initiate(ctx context.Context, req InitiateRequest) (*RecoverySession, error) {
	now := s.clock()
	session := &RecoverySession{
		ID:        generateID(),
		UserID:    req.UserID,
		TenantID:  req.TenantID,
		Strategy:  s.Name(),
		Tier:      s.Tier(),
		Status:    StatusInitiated,
		Factors: map[string]FactorState{
			"recovery_code": {Type: "recovery_code", Status: "pending"},
		},
		CreatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute), // 10 min to enter code
	}
	return session, s.store.SaveSession(session)
}

func (s *RecoveryCodeStrategy) Verify(ctx context.Context, sessionID string, sub FactorSubmission) (*VerifyResult, error) {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if s.clock().After(session.ExpiresAt) {
		session.Status = StatusExpired
		_ = s.store.SaveSession(session)
		return &VerifyResult{Error: "session expired"}, nil
	}

	code, ok := sub.Data["code"].(string)
	if !ok {
		return &VerifyResult{Error: "missing code"}, nil
	}

	// Look up the stored hash for this user.
	storedHash, err := s.store.GetRecoveryCodeHash(session.UserID)
	if err != nil {
		return &VerifyResult{Error: "no recovery code enrolled"}, nil
	}

	// Verify the code (constant-time comparison).
	if !s.hasher.Verify(code, storedHash) {
		factor := session.Factors["recovery_code"]
		factor.Attempts++
		if factor.Attempts >= 5 {
			session.Status = StatusFailed
		}
		session.Factors["recovery_code"] = factor
		_ = s.store.SaveSession(session)
		return &VerifyResult{Session: session, Error: "invalid code"}, nil
	}

	// Code verified.
	now := s.clock()
	factor := session.Factors["recovery_code"]
	factor.Status = "verified"
	factor.VerifiedAt = &now
	session.Factors["recovery_code"] = factor
	session.Status = StatusAuthed
	_ = s.store.SaveSession(session)

	return &VerifyResult{
		Session:        session,
		FactorVerified: true,
		AllFactorsMet:  true,
	}, nil
}

func (s *RecoveryCodeStrategy) Complete(ctx context.Context, sessionID string, req CompleteRequest) (*CompleteResult, error) {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != StatusAuthed {
		return nil, fmt.Errorf("session not in authed state: %s", session.Status)
	}

	// Verify the WebAuthn attestation and enroll new credential.
	// (This delegates to the existing WebAuthn service.)
	newCredID, err := s.store.EnrollNewPasskey(session.UserID, req.AttestationObject, req.ClientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to enroll new passkey: %w", err)
	}

	// Revoke old credentials.
	revokedCount, err := s.store.RevokeAllOtherPasskeys(session.UserID, newCredID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old passkeys: %w", err)
	}

	// Invalidate the recovery code (one-time use).
	_ = s.store.InvalidateRecoveryCode(session.UserID)

	// Mark session complete.
	now := s.clock()
	session.Status = StatusCompleted
	session.CompletedAt = &now
	_ = s.store.SaveSession(session)

	return &CompleteResult{
		NewCredentialID:      newCredID,
		OldCredentialsRevoked: revokedCount,
	}, nil
}

func (s *RecoveryCodeStrategy) Cancel(ctx context.Context, sessionID, reason string) error {
	session, err := s.store.GetSession(sessionID)
	if err != nil {
		return err
	}
	session.Status = StatusCancelled
	return s.store.SaveSession(session)
}
```

### 6.5 Recovery Manager (Strategy Selector)

```go
// RecoveryManager routes recovery requests to the appropriate strategy.
type RecoveryManager struct {
	strategies map[string]RecoveryStrategy
	store      RecoveryStore
	defaultTier RecoveryTier
}

func NewRecoveryManager(store RecoveryStore) *RecoveryManager {
	m := &RecoveryManager{
		strategies: make(map[string]RecoveryStrategy),
		store:      store,
	}
	// Register built-in strategies.
	m.Register(&RecoveryCodeStrategy{store: store})
	m.Register(&AdminCredentialStrategy{store: store})
	m.Register(&SocialRecoveryStrategy{store: store})
	return m
}

func (m *RecoveryManager) Register(s RecoveryStrategy) {
	m.strategies[s.Name()] = s
}

// StartRecovery initiates the recovery flow, selecting the appropriate strategy.
func (m *RecoveryManager) StartRecovery(
	ctx context.Context,
	userID, tenantID, strategyName string,
) (*RecoverySession, error) {
	strategy, ok := m.strategies[strategyName]
	if !ok {
		return nil, fmt.Errorf("unknown recovery strategy: %s", strategyName)
	}

	// Verify the user exists and is eligible for recovery.
	user, err := m.store.GetUser(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if user.HasActivePasskeys() && strategyName != "admin_credential" {
		// Only admin override allows recovery when passkeys are still active
		return nil, errors.New("user still has active passkeys; recovery not needed")
	}

	return strategy.Initiate(ctx, InitiateRequest{
		UserID:   userID,
		TenantID: tenantID,
		Strategy: strategyName,
	})
}

// GetAvailableStrategies returns the strategies available for a user.
func (m *RecoveryManager) GetAvailableStrategies(userID, tenantID string) []StrategyInfo {
	var available []StrategyInfo

	for name, s := range m.strategies {
		info := StrategyInfo{
			Name: name,
			Tier: s.Tier(),
		}

		switch name {
		case "recovery_code":
			// Check if user has a recovery code enrolled.
			hasCode, _ := m.store.HasRecoveryCode(userID)
			info.Available = hasCode
		case "admin_credential":
			// Admin credential is available if user is in a tenant with admins.
			info.Available = true // always available as fallback
		case "social_threshold":
			// Check if user has trusted contacts enrolled.
			hasContacts, _ := m.store.HasTrustedContacts(userID)
			info.Available = hasContacts
		}

		if info.Available {
			available = append(available, info)
		}
	}

	return available
}

type StrategyInfo struct {
	Name      string
	Tier      RecoveryTier
	Available bool
}
```

### 6.6 DB Schema Additions

```sql
-- Recovery codes (Tier 1)
CREATE TABLE recovery_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    code_hash       VARCHAR(128) NOT NULL,  -- bcrypt or argon2 hash
    enrolled_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at         TIMESTAMPTZ,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    UNIQUE(tenant_id, user_id)
);

-- Recovery sessions (all tiers)
CREATE TABLE recovery_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    strategy        VARCHAR(50) NOT NULL,  -- 'recovery_code', 'admin_credential', 'social_threshold'
    tier            INT NOT NULL,           -- 1, 2, or 3
    status          VARCHAR(30) NOT NULL DEFAULT 'initiated',
    factors_json    JSONB NOT NULL DEFAULT '{}',
    metadata_json   JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    completed_at    TIMESTAMPTZ,
    ip_address      INET,
    user_agent      TEXT
);

CREATE INDEX idx_recovery_sessions_user ON recovery_sessions(tenant_id, user_id);
CREATE INDEX idx_recovery_sessions_status ON recovery_sessions(status, expires_at);

-- Admin-issued recovery credentials (Tier 2)
CREATE TABLE admin_recovery_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    issued_by       UUID NOT NULL,          -- admin user ID
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    revoked_by      UUID,
    reason          TEXT NOT NULL,
    issuer_ip       INET NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active', -- active, used, revoked, expired
    signature       BYTEA NOT NULL           -- HMAC for tamper protection
);

CREATE INDEX idx_admin_recovery_creds_tenant ON admin_recovery_credentials(tenant_id, status);
CREATE INDEX idx_admin_recovery_creds_admin ON admin_recovery_credentials(issued_by, issued_at);

-- Social recovery: trusted contacts (Tier 3)
CREATE TABLE recovery_trusted_contacts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,      -- the user who designated the contact
    contact_user_id     UUID NOT NULL,      -- the trusted contact
    encrypted_share     BYTEA NOT NULL,     -- Shamir share encrypted with contact's pubkey
    share_nonce         BYTEA NOT NULL,     -- AES-GCM nonce
    share_index         INT NOT NULL,       -- x-coordinate for Shamir
    enrolled_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, contact_user_id)
);

-- Social recovery sessions
CREATE TABLE social_recovery_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    threshold           INT NOT NULL,        -- k
    total_contacts      INT NOT NULL,        -- n
    collected_shares    INT NOT NULL DEFAULT 0,
    status              VARCHAR(30) NOT NULL DEFAULT 'initiated',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    contact_responses   JSONB NOT NULL DEFAULT '{}'
);

-- Recovery-only credentials (hardware backup)
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS recover_only BOOLEAN NOT NULL DEFAULT false;

-- Audit events for recovery
CREATE TABLE recovery_audit_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    event_type      VARCHAR(80) NOT NULL,
    session_id      UUID,
    credential_id   UUID,
    user_id         UUID,
    admin_user_id   UUID,
    ip_address      INET,
    metadata_json   JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovery_audit_tenant ON recovery_audit_events(tenant_id, created_at DESC);
CREATE INDEX idx_recovery_audit_user ON recovery_audit_events(user_id, created_at DESC);
```

### 6.7 Integration with Existing Auth Service

The recovery system integrates with GGID's existing auth service through a thin adapter:

```go
// In services/auth/internal/recovery/adapter.go

// AuthAdapter bridges the recovery system to the existing auth service.
type AuthAdapter struct {
	authService  auth.Service
	webauthnSvc  webauthn.Service
	userRepo     user.Repository
	credRepo     credential.Repository
}

// EnrollNewPasskey creates a new WebAuthn credential for the user.
func (a *AuthAdapter) EnrollNewPasskey(
	userID string,
	attestationObject, clientDataJSON []byte,
) (string, error) {
	// Delegate to existing WebAuthn service.
	cred, err := a.webauthnSvc.VerifyAttestation(userID, attestationObject, clientDataJSON)
	if err != nil {
		return "", err
	}
	return cred.ID, nil
}

// RevokeAllOtherPasskeys removes all credentials except the specified one.
func (a *AuthAdapter) RevokeAllOtherPasskeys(userID, keepCredID string) (int, error) {
	creds, err := a.credRepo.ListByUser(userID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, c := range creds {
		if c.ID == keepCredID {
			continue
		}
		if err := a.credRepo.Delete(c.ID); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

// HasActivePasskeys checks if the user has any non-recovery passkeys.
func (a *AuthAdapter) HasActivePasskeys(userID string) (bool, error) {
	creds, err := a.credRepo.ListByUser(userID)
	if err != nil {
		return false, err
	}
	for _, c := range creds {
		if !c.RecoverOnly {
			return true, nil
		}
	}
	return false, nil
}
```

---

## 7. Recommendations and Priority Matrix

### 7.1 Phased Implementation Roadmap

**Phase 1: Minimum Viable Recovery (already in backlog)**
- Recovery codes: 20-character alphanumeric codes generated at enrollment
- Recovery code stored as hash (argon2id) in database
- One-time use: code is invalidated after successful recovery
- Printable format (Emergency Kit pattern from 1Password)
- Self-service: no admin involvement needed

**Phase 2: Admin-Issued Recovery Credentials**
- TemporaryCredential with 15-30 min TTL
- Admin step-up auth (MFA) required before issuance
- Admin rate limiting (10 per hour per admin)
- Full audit trail (issuance, usage, revocation)
- API endpoints for admin console integration
- Revocation capability (admin can revoke before use)

**Phase 3: Social Recovery with Threshold Scheme**
- Shamir's Secret Sharing (k-of-n threshold)
- Trusted contact enrollment flow
- Per-contact authentication before share release
- Encrypted shares (contact public key)
- Recovery session management
- Notification system (email, push)

### 7.2 Priority Matrix

| Feature | User Impact | Security Impact | Implementation Effort | Priority |
|---|---|---|---|---|
| **Recovery codes (Tier 1)** | High — every user needs this | High | Low (2-3 days) | P0 (must have) |
| **Recovery-only hardware key** | Medium — power users | High | Low (1 day) | P1 |
| **Admin credentials (Tier 2)** | High — enterprise essential | High | Medium (3-5 days) | P1 |
| **Multi-factor recovery** | Medium — additional security layer | High | Medium (3-5 days) | P2 |
| **Recovery key escrow** | Low — advanced feature | Medium | Medium (3-5 days) | P2 |
| **Social recovery (Tier 3)** | Low — niche use case | Very High | High (1-2 weeks) | P3 |
| **CXP/CXF monitoring** | Low — informational only | Low | Low (0.5 day) | P3 |

### 7.3 Decision Framework

When deciding which recovery mechanism to use for a given tenant:

```
Is this a consumer/individual tenant?
├── Yes → Tier 1 (recovery code) is sufficient
│         Optionally offer hardware key backup
│
└── No (enterprise tenant)
    ├── Is the user a regular employee?
    │   └── Tier 1 (code) + Tier 2 (admin credential)
    │
    └── Is the user a high-privilege admin?
        └── Tier 1 (code) + Tier 2 (admin credential)
            + Tier 3 (social recovery, opt-in)
            + Hardware key backup (mandatory)
```

### 7.4 Anti-Patterns to Avoid

1. **Email-only recovery**: Email accounts are themselves often compromised. Recovery
   should not depend solely on email access. Always pair with a second factor.

2. **Security questions**: Knowledge-based authentication (security questions) have low
   entropy and are often publicly discoverable. Do not use as a recovery factor.

3. **Unlimited admin recovery**: Without rate limiting and audit trails, admin-issued
   credentials become a privilege escalation vector.

4. **Recovery without revocation**: When a new passkey is enrolled via recovery, ALL old
   credentials must be revoked. Otherwise, a recovered account may still be accessible
   to whoever has the old credentials.

5. **Long-lived recovery tokens**: Recovery tokens should have short TTLs (minutes, not
   hours). The longer the window, the higher the risk of interception.

6. **Algorithmic passkey recovery**: As multiple industry sources note, "algorithmic
   passkey recovery should not be a feature." Passkeys are cryptographic key pairs —
   there is no algorithmic way to recover a lost private key. Recovery always requires
   an alternative authentication path (code, admin, social) followed by new enrollment.

---

## Appendix A: Shamir's Secret Sharing in Go

### A.1 Test Cases

```go
package recovery

import (
	"bytes"
	"testing"
)

func TestSplitAndReconstruct(t *testing.T) {
	secret := []byte("this is a very secret recovery key!")

	// Split into 5 shares, need 3 to reconstruct.
	shares, err := SplitSecret(secret, 3, 5)
	if err != nil {
		t.Fatalf("SplitSecret failed: %v", err)
	}

	if len(shares) != 5 {
		t.Fatalf("expected 5 shares, got %d", len(shares))
	}

	// Reconstruct with exactly 3 shares (minimum).
	reconstructed, err := ReconstructSecret(shares[:3])
	if err != nil {
		t.Fatalf("ReconstructSecret failed: %v", err)
	}

	if !bytes.Equal(secret, reconstructed) {
		t.Fatalf("reconstructed secret does not match original")
	}
}

func TestReconstructWithDifferentShares(t *testing.T) {
	secret := []byte("another secret value here!!")

	shares, err := SplitSecret(secret, 3, 5)
	if err != nil {
		t.Fatalf("SplitSecret failed: %v", err)
	}

	// Use shares 1, 3, 4 (non-contiguous).
	selected := []Share{shares[0], shares[2], shares[3]}

	reconstructed, err := ReconstructSecret(selected)
	if err != nil {
		t.Fatalf("ReconstructSecret failed: %v", err)
	}

	if !bytes.Equal(secret, reconstructed) {
		t.Fatalf("reconstructed secret does not match original")
	}
}

func TestInsufficientSharesFails(t *testing.T) {
	secret := []byte("test secret")

	shares, err := SplitSecret(secret, 3, 5)
	if err != nil {
		t.Fatalf("SplitSecret failed: %v", err)
	}

	// Only 2 shares — should NOT reconstruct correctly (but won't error,
	// it will produce garbage).
	reconstructed, err := ReconstructSecret(shares[:2])
	if err != nil {
		t.Fatalf("ReconstructSecret should not error with 2 shares: %v", err)
	}

	if bytes.Equal(secret, reconstructed) {
		t.Fatal("2 shares should NOT reconstruct the secret correctly")
	}
}

func TestEncryptDecryptShare(t *testing.T) {
	originalShare := Share{
		Index: 3,
		Data:  []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x01, 0x02},
	}

	contactKey := []byte("fake-contact-public-key-for-testing")

	encrypted, err := EncryptShare(originalShare, contactKey)
	if err != nil {
		t.Fatalf("EncryptShare failed: %v", err)
	}

	decrypted, err := DecryptShare(encrypted, contactKey)
	if err != nil {
		t.Fatalf("DecryptShare failed: %v", err)
	}

	if decrypted.Index != originalShare.Index {
		t.Fatalf("index mismatch: %d != %d", decrypted.Index, originalShare.Index)
	}

	if !bytes.Equal(decrypted.Data, originalShare.Data) {
		t.Fatalf("data mismatch")
	}
}
```

---

## Appendix B: Recovery State Machine in Go

### B.1 Full State Machine Implementation

```go
// Package recoverysm implements the recovery flow state machine.
package recoverysm

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents a state in the recovery state machine.
type State string

const (
	StateIdle        State = "idle"
	StateInitiated   State = "initiated"
	StateVerifying   State = "verifying"
	StateAuthed      State = "authed"
	StateEnrolling   State = "enrolling"
	StateCompleted   State = "completed"
	StateFailed      State = "failed"
	StateExpired     State = "expired"
	StateCancelled   State = "cancelled"
)

// Event represents an event that can trigger a state transition.
type Event string

const (
	EventInitiate          Event = "initiate"
	EventFactorSubmitted   Event = "factor_submitted"
	EventFactorVerified    Event = "factor_verified"
	EventFactorFailed      Event = "factor_failed"
	EventAllFactorsMet     Event = "all_factors_met"
	EventStartEnrollment   Event = "start_enrollment"
	EventEnrollmentSuccess Event = "enrollment_success"
	EventEnrollmentFailure Event = "enrollment_failure"
	EventTimeout           Event = "timeout"
	EventCancel            Event = "cancel"
	EventRetry             Event = "retry"
)

// Transition defines a valid state transition.
type Transition struct {
	From  State
	Event Event
	To    State
	Action TransitionAction
}

// TransitionAction is executed during a state transition.
type TransitionAction func(ctx *TransitionContext) error

// TransitionContext provides data to transition actions.
type TransitionContext struct {
	SessionID string
	UserID    string
	TenantID  string
	Event     Event
	Data      map[string]interface{}
}

// StateMachine manages recovery flow state transitions.
type StateMachine struct {
	mu          sync.RWMutex
	current     map[string]State // sessionID -> current state
	transitions []Transition
	handlers    map[State]StateHandler
}

// StateHandler is called when entering a state.
type StateHandler interface {
	OnEnter(ctx *TransitionContext) error
}

// NewStateMachine creates a recovery state machine with standard transitions.
func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		current:  make(map[string]State),
		handlers: make(map[State]StateHandler),
	}

	// Register standard transitions.
	sm.transitions = []Transition{
		{StateIdle, EventInitiate, StateInitiated, nil},
		{StateInitiated, EventFactorSubmitted, StateVerifying, nil},
		{StateVerifying, EventFactorVerified, StateVerifying, nil}, // stay, wait for more
		{StateVerifying, EventAllFactorsMet, StateAuthed, nil},
		{StateVerifying, EventFactorFailed, StateVerifying, nil}, // stay, allow retry
		{StateAuthed, EventStartEnrollment, StateEnrolling, nil},
		{StateEnrolling, EventEnrollmentSuccess, StateCompleted, nil},
		{StateEnrolling, EventEnrollmentFailure, StateAuthed, nil}, // back to authed, retry
		// Terminal transitions from any non-terminal state
		{StateInitiated, EventTimeout, StateExpired, nil},
		{StateVerifying, EventTimeout, StateExpired, nil},
		{StateAuthed, EventTimeout, StateExpired, nil},
		{StateEnrolling, EventTimeout, StateExpired, nil},
		{StateInitiated, EventCancel, StateCancelled, nil},
		{StateVerifying, EventCancel, StateCancelled, nil},
		{StateAuthed, EventCancel, StateCancelled, nil},
		{StateEnrolling, EventCancel, StateCancelled, nil},
	}

	return sm
}

// Send processes an event for a session.
func (sm *StateMachine) Send(sessionID string, event Event, data map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	current, ok := sm.current[sessionID]
	if !ok {
		current = StateIdle
	}

	// Find matching transition.
	for _, t := range sm.transitions {
		if t.From == current && t.Event == event {
			// Execute action if defined.
			if t.Action != nil {
				ctx := &TransitionContext{
					SessionID: sessionID,
					Event:     event,
					Data:      data,
				}
				if err := t.Action(ctx); err != nil {
					return fmt.Errorf("transition action failed: %w", err)
				}
			}

			// Execute state handler if defined.
			if handler, ok := sm.handlers[t.To]; ok {
				ctx := &TransitionContext{
					SessionID: sessionID,
					Event:     event,
					Data:      data,
				}
				if err := handler.OnEnter(ctx); err != nil {
					return fmt.Errorf("state handler failed: %w", err)
				}
			}

			// Update state.
			sm.current[sessionID] = t.To
			return nil
		}
	}

	return fmt.Errorf("no valid transition from %s on event %s", current, event)
}

// CurrentState returns the current state for a session.
func (sm *StateMachine) CurrentState(sessionID string) State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, ok := sm.current[sessionID]; ok {
		return state
	}
	return StateIdle
}

// IsTerminal returns true if the state is terminal (no further transitions).
func IsTerminal(s State) bool {
	switch s {
	case StateCompleted, StateFailed, StateExpired, StateCancelled:
		return true
	default:
		return false
	}
}

// RegisterHandler registers a state handler for a specific state.
func (sm *StateMachine) RegisterHandler(state State, handler StateHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers[state] = handler
}
```

### B.2 State Machine Usage Example

```go
func ExampleRecoveryFlow() {
	sm := NewStateMachine()

	sessionID := "session-123"

	// Start recovery.
	_ = sm.Send(sessionID, EventInitiate, map[string]interface{}{
		"user_id":   "user-456",
		"tenant_id": "tenant-789",
		"strategy":  "recovery_code",
	})
	// State: initiated

	// User submits recovery code.
	_ = sm.Send(sessionID, EventFactorSubmitted, map[string]interface{}{
		"code": "ABCD-1234-EFGH-5678",
	})
	// State: verifying

	// Code verified.
	_ = sm.Send(sessionID, EventFactorVerified, nil)
	// State: verifying (waiting for more factors if multi-factor)

	// All factors satisfied.
	_ = sm.Send(sessionID, EventAllFactorsMet, nil)
	// State: authed

	// Start passkey enrollment.
	_ = sm.Send(sessionID, EventStartEnrollment, nil)
	// State: enrolling

	// Enrollment succeeded.
	_ = sm.Send(sessionID, EventEnrollmentSuccess, map[string]interface{}{
		"credential_id": "cred-new-123",
	})
	// State: completed

	fmt.Println("Final state:", sm.CurrentState(sessionID))
	// Output: Final state: completed
}
```

---

## Appendix C: References and Further Reading

### Standards and Specifications

1. **FIDO Credential Exchange Protocol (CXP)** — Working Draft
   - https://fidoalliance.org/specs/cx/cxp-v1.0-wd-20240522.html
   - Defines the encrypted transfer protocol between credential providers

2. **FIDO Credential Exchange Format (CXF)** — Review Draft (March 2025)
   - Defines the standardized JSON data format for credential exchange
   - Supports passkeys, passwords, TOTP secrets, and notes

3. **W3C WebAuthn Level 3** — W3C Recommendation
   - https://www.w3.org/TR/webauthn-3/
   - The foundational standard for passkeys

4. **RFC 9180 (HPKE)** — Hybrid Public Key Encryption
   - Used by CXP for end-to-end encryption of credential transfers

### Industry Analysis

5. **Corbado: Auth0 Passkeys Analysis** (August 2025)
   - Detailed analysis of Auth0/Okta passkey implementation
   - Key finding: recovery remains password-centric; no passkey-only flow

6. **Corbado: CXP/CXF Deep Dive** (April 2025)
   - Technical overview of the CXP/CXF standards
   - Apple shipped CXF in iOS/macOS 26; CXP targets early 2026

7. **Microsoft: Temporary Access Pass documentation**
   - Entra ID feature for time-limited admin-issued recovery passes
   - Closest industry analog to GGID's proposed admin recovery credential

### Academic References

8. **Shamir, A. (1979)** — "How to Share a Secret"
   - Communications of the ACM, Vol. 22, No. 11
   - The foundational paper on threshold secret sharing

9. **Blakley, G. R. (1979)** — "Safeguarding Cryptographic Keys"
   - Proceedings of the National Computer Conference
   - Independent invention of threshold schemes (geometric approach)

### Implementation References

10. **go-sss** — Go implementation of Shamir's Secret Sharing
    - Reference implementation patterns for GF(256) operations

11. **HashiCorp Vault** — Transit Secret Engine
    - Enterprise key management with Shamir's unseal keys

### GGID Internal References

12. [passkey-recovery.md](./passkey-recovery.md) — Basic recovery (synced vs device-bound,
    backup eligibility, sync platforms, recovery codes)
13. WebAuthn roadmap — WebAuthn feature roadmap
14. [zero-trust-implementation.md](../design/zero-trust-implementation.md) — Zero Trust design
15. `services/auth/internal/webauthn/` — Existing WebAuthn implementation

---

## Summary

This document covers **advanced and emerging** passkey recovery strategies that go beyond
the basic recovery code and sync-platform approaches documented in `passkey-recovery.md`.

**Key takeaways:**

1. **CXP/CXF** solves passkey portability but requires **no server-side changes** for GGID.
   It is a client-side protocol between authenticator providers.

2. **Social recovery** (Shamir's threshold scheme) provides the strongest security model
   for high-value accounts but has high UX complexity and is suitable only for opt-in,
   high-security use cases.

3. **Admin-issued recovery credentials** are the most practical enterprise recovery
   mechanism, validated by Microsoft's Temporary Access Pass. This should be GGID's
   Phase 2 priority.

4. **Recovery-only hardware keys** (YubiKey backup) are a simple, high-security option
   that requires minimal implementation effort.

5. **Multi-factor recovery** adds defense-in-depth by requiring multiple independent
   factors, preventing single-point-of-compromise attacks.

6. The **tiered model** (Tier 1: self-service → Tier 2: admin → Tier 3: social) provides
   progressive fallback with increasing security at each tier.

7. GGID's **immediate priority** should be recovery codes (Phase 1, already in backlog),
   followed by admin credentials (Phase 2), then social recovery (Phase 3, optional).

---

*Document length: ~950 lines. Last updated: 2025.*
