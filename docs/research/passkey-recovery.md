# Synced Passkeys and Account Recovery

> **Status**: Research & Design Document
> **Date**: 2025
> **Author**: GGID Research
> **Related**: [docs/design/zero-trust-implementation.md](../design/zero-trust-implementation.md),
> [services/auth/internal/webauthn/](../../services/auth/internal/webauthn/)

---

## Table of Contents

1. [Synced vs Device-Bound Passkeys](#1-synced-vs-device-bound-passkeys)
2. [Sync Platforms (Detailed)](#2-sync-platforms-detailed)
3. [Backup Eligibility and Backup State](#3-backup-eligibility-and-backup-state)
4. [Account Recovery Scenarios](#4-account-recovery-scenarios)
5. [Recovery Mechanisms for GGID](#5-recovery-mechanisms-for-ggid)
6. [GGID Implementation Design](#6-ggid-implementation-design)
7. [Security Considerations](#7-security-considerations)
8. [UX Best Practices](#8-ux-best-practices)
9. [Industry Approaches](#9-industry-approaches)

---

## 1. Synced vs Device-Bound Passkeys

Passkeys (WebAuthn/FIDO2 credentials) come in two fundamental flavors that
differ in where the private key material lives and how it can be recovered.

### 1.1 Device-Bound Passkeys

A **device-bound** passkey never leaves the authenticator that created it.
The private key is stored in a Secure Enclave, TPM, or on a hardware security
key (YubiKey, Titan, Feitian).

| Attribute | Detail |
|---|---|
| **Key storage** | Hardware-secured on a single device |
| **Sync** | Never synced — key stays on device |
| **Security** | Maximum — extracting the key requires physical compromise |
| **Phishing resistance** | Full — FIDO2 origin binding |
| **Device loss** | Credential is gone permanently; no recovery from sync |
| **Backup Eligible (BE)** | `false` |
| **Use case** | High-security environments, regulated industries, admin accounts |

**Strengths**: Strongest security guarantee. The key cannot be exfiltrated
even if the user's cloud account is compromised. Ideal for privileged
accounts (e.g., GGID tenant administrators).

**Weaknesses**: Poor resilience to device loss. If the user loses or breaks
their only authenticator, they are locked out. This makes fallback recovery
flows essential.

### 1.2 Synced Passkeys

A **synced** passkey is backed up to a cloud sync fabric (Apple iCloud
Keychain, Google Password Manager, etc.) and distributed across the user's
devices. The private key is end-to-end encrypted so that the sync provider
cannot read it.

| Attribute | Detail |
|---|---|
| **Key storage** | Hardware-secured per device + E2E-encrypted cloud backup |
| **Sync** | Synced across same-vendor devices |
| **Security** | Strong — E2E encrypted, biometric/PIN gated |
| **Phishing resistance** | Full — FIDO2 origin binding |
| **Device loss** | Key available on other synced devices |
| **Backup Eligible (BE)** | `true` |
| **Use case** | Consumer apps, BYOD, most enterprise users |

**Strengths**: Excellent UX. When a user gets a new phone, their passkeys
are automatically available. Reduces account lockout risk dramatically.

**Weaknesses**: Depends on the sync provider's security. If the user's
iCloud/Google account is compromised AND the device passcode is known, an
attacker could theoretically access synced passkeys (though this requires
circumventing multiple layers of protection).

### 1.3 Sync Fabric Providers Comparison

| Provider | Platforms | E2E Encrypted | BE Flag | Recovery | Cross-Platform |
|---|---|---|---|---|---|
| **Apple iCloud Keychain** | iOS, iPadOS, macOS, visionOS | Yes | `true` | Apple account recovery | No (Apple ecosystem only) |
| **Google Password Manager** | Android, Chrome OS, Chrome (desktop) | Yes | `true` (Android) | Google account recovery | Partial (Chrome on desktop) |
| **Microsoft Authenticator** | Windows 11, Edge | Yes | `true` | Microsoft account recovery | Partial (Edge on desktop) |
| **1Password** | iOS, Android, macOS, Windows, Linux, Web | Yes (Secret Key) | `true` | Secret Key + Master Password | **Yes** (all platforms) |
| **Bitwarden** | iOS, Android, macOS, Windows, Linux, Web | Yes | `true` | Master Password | **Yes** (all platforms) |
| **Dashlane** | iOS, Android, macOS, Windows, Web | Yes | `true` | Master Password | **Yes** (all platforms) |
| **YubiKey / Hardware** | Platform-independent | N/A | `false` | N/A (device-bound) | Yes (USB/NFC) |

**Key insight**: First-party sync fabrics (Apple, Google, Microsoft) are
ecosystem-locked. A passkey created on iPhone with iCloud Keychain sync
does not appear on the user's Android phone. Third-party password managers
(1Password, Bitwarden) bridge this gap by syncing across all platforms.

---

## 2. Sync Platforms (Detailed)

### 2.1 Apple iCloud Keychain

Apple's passkey sync is built into iCloud Keychain and available on all
Apple devices signed in to the same Apple ID.

**How it works**:
- When a user creates a passkey on iPhone, the WebKit/Authentication
  Services framework stores it in the Secure Enclave and queues it for
  iCloud Keychain sync.
- The key is encrypted with a key derived from the user's device passcode
  and escrowed via Apple's Hardware Security Module (HSM) infrastructure.
- Apple cannot read the passkey — the encryption key is never available
  to Apple in plaintext.

**Backup eligibility**: `true` — all passkeys created through iCloud
Keychain report `BE=true` in the WebAuthn attestation.

**Recovery paths**:
1. **Trusted device**: If the user has another Apple device, passkeys
   appear automatically after iCloud Keychain syncs.
2. **iCloud Keychain Verification Code**: Sent to a trusted phone number.
3. **Apple Account Recovery**: A multi-step process involving trusted
   phone number verification, device verification, and an escrow key
   verification flow. Can take hours to days.

**Security properties**:
- Biometric (Face ID / Touch ID) or device passcode required for every
  passkey use — the user presence (UP) and user verification (UV) flags
  are always set.
- If the user removes a device from their Apple ID, that device's copy of
  synced passkeys is deleted on next check-in.

### 2.2 Google Password Manager

Google's passkey sync is integrated with Google Password Manager, the
credential manager built into Android and Chrome.

**How it works**:
- On Android 9+, passkeys created through the Google Play Services
  credential manager are stored in the device's secure hardware and synced
  via the user's Google Account.
- Chrome on desktop can also sync passkeys via the Google Account,
  enabling cross-device sign-in on Windows/macOS/Linux via a QR-code
  pairing flow (hybrid transport / `cable`).
- The sync uses a Google Account encryption key that is not readable by
  Google (for passkeys on Android).

**Backup eligibility**: `true` on Android.

**Recovery paths**:
1. **Another Android/Chrome device**: Passkeys appear automatically.
2. **Google Account Recovery**: Involves email/SMS verification,
   previously used devices, and security questions. Google's recovery
   process is knowledge-and-device based.
3. **Phone as a security key**: Android phone can authenticate for a
   desktop Chrome session via BLE + QR code (hybrid transport). The phone
   acts as a roaming authenticator for the desktop without the passkey
   leaving the phone.

### 2.3 Microsoft Authenticator

Microsoft offers passkey sync via the Microsoft Authenticator app and
Windows Hello.

**How it works**:
- Windows 11 supports creating synced passkeys stored via Windows Hello
  and backed up to the Microsoft account.
- The Microsoft Authenticator app (iOS/Android) can also store passkeys
  synced through the Microsoft account.
- Sync is E2E encrypted, similar to Apple's model.

**Backup eligibility**: `true` on Windows 11 and Microsoft Authenticator.

**Recovery paths**:
1. **Another Windows 11 device** signed in to the same Microsoft account.
2. **Microsoft Account Recovery**: Email verification, SMS, and
   authenticator app verification.

### 2.4 Third-Party Password Managers

Third-party credential managers provide the critical ability to sync
passkeys across the iOS/Android divide.

#### 1Password

- **Architecture**: Zero-knowledge. The vault is encrypted with a key
  derived from the user's Master Password and a 128-bit Secret Key
  (generated at account creation). 1Password servers never see the
  Secret Key or Master Password.
- **Passkey support**: Since 2023, 1Password can create and store
  passkeys as vault items, sync them across all platforms (iOS, Android,
  macOS, Windows, Linux, Web), and autofill them in any browser.
- **Backup eligibility**: `true` — 1Password reports `BE=true`.
- **Recovery**: If the user loses all devices, they need their Secret Key
  (which they downloaded/saved at signup) plus their Master Password to
  recover on a new device. The Emergency Kit document contains the
  Secret Key.

#### Bitwarden

- **Architecture**: Open-source zero-knowledge vault. Encryption key
  derived from Master Password via PBKDF2 or Argon2id.
- **Passkey support**: Stores passkeys as vault items, syncs across all
  platforms.
- **Backup eligibility**: `true`.
- **Recovery**: Master Password + email. No Secret Key equivalent (relies
  solely on Master Password strength).

#### Dashlane

- **Architecture**: Zero-knowledge, proprietary.
- **Passkey support**: Stores and syncs passkeys across platforms.
- **Recovery**: Master Password or account recovery key.

**Why third-party managers matter for GGID**: In a BYOD or mixed-device
enterprise, users may have an iPhone and a Windows laptop. Apple and
Google sync fabrics won't bridge that gap natively. Recommending or
mandating a third-party password manager (1Password, Bitwarden) ensures
users have their passkeys everywhere.

---

## 3. Backup Eligibility and Backup State

The WebAuthn specification defines two authenticator flags that tell the
Relying Party (RP) about the passkey's sync properties.

### 3.1 Backup Eligible (BE)

- **Set at**: credential creation time (registration ceremony).
- **Meaning**: This passkey **can** be backed up to a sync fabric.
- **Immutable**: Once set at registration, it does not change.
- **Values**:
  - `true` — credential is backed by a sync fabric (e.g., iCloud Keychain,
    Google Password Manager).
  - `false` — credential is device-bound (e.g., YubiKey, some platform
    authenticators without sync).

### 3.2 Backup State (BS)

- **Dynamic**: Can change over the credential's lifetime.
- **Meaning**: This passkey **is currently** backed up / synced.
- **Values**:
  - `true` — the credential has been successfully backed up to the sync
    fabric and is available on other devices.
  - `false` — the credential is eligible for backup but has not yet been
    synced, OR backup is pending.

**Example**: A passkey created on iPhone reports `BE=true`. Immediately
after creation, `BS` might be `false` (sync in progress). After iCloud
Keychain syncs, the next authentication will report `BS=true`.

### 3.3 How to Check in Code

In the WebAuthn registration response (attestation), the authenticator
data contains flags. Using the `go-webauthn` library:

```go
import "github.com/go-webauthn/webauthn/protocol"

// After parsing the registration response:
parsedResponse, err := protocol.ParseCredentialCreationResponse(r)
if err != nil {
    // handle error
}

flags := parsedResponse.Response.AttestationObject.AuthData.Flags
backupEligible := flags.HasBackupEligible()   // BE flag
backupState    := flags.HasBackupState()       // BS flag
```

During authentication (assertion):

```go
parsedResponse, err := protocol.ParseCredentialRequestResponse(r)
if err != nil {
    // handle error
}

flags := parsedResponse.Response.AuthenticatorData.Flags
backupEligible := flags.HasBackupEligible()
backupState    := flags.HasBackupState()
```

### 3.4 Why This Matters for RP Policy

| Policy Decision | Check | Rationale |
|---|---|---|
| Require synced passkeys | `BE == true` | Ensures users can recover from device loss |
| Require device-bound | `BE == false` | For admin/high-privilege accounts requiring hardware keys |
| Monitor sync state | `BS == true` | Alert users whose passkey hasn't synced yet |
| Conditional access | `BS == true` then allow | Only allow login from backed-up credentials |

**GGID recommendation**: Store both `backup_eligible` and `backup_state`
on each credential record. For standard users, recommend `BE=true` (synced).
For tenant admins, optionally enforce `BE=false` (device-bound with YubiKey).

---

## 4. Account Recovery Scenarios

### 4.1 User Loses All Synced Devices

**Scenario**: User's iPhone is lost/stolen and they have no other Apple
devices. Their MacBook was also stolen. All devices with their synced
passkeys are gone.

**Recovery paths by platform**:

| Platform | Recovery Method | Timeframe |
|---|---|---|
| **Apple iCloud** | Account Recovery flow: trusted phone number → verification code → device verification → escrow key | Hours to days |
| **Google** | Account Recovery: email + SMS + previously used devices + security questions | Minutes to hours |
| **Microsoft** | Account Recovery: email + SMS + authenticator push | Minutes to hours |
| **1Password** | Emergency Kit (Secret Key) + Master Password on new device | Immediate |
| **Bitwarden** | Master Password + email on new device | Immediate |
| **Device-bound only** | **Account is locked.** No recovery from sync fabric. Requires RP-side fallback. | N/A |

**Key takeaway**: If a user's only credential is a device-bound passkey
and the device is lost, the RP (GGID) must provide a recovery path.
Synced passkeys significantly reduce this risk but don't eliminate it
entirely (user could lose all synced devices + fail platform recovery).

### 4.2 User Switches Platforms (iOS to Android)

**Scenario**: User has been using an iPhone with iCloud Keychain-synced
passkeys. They switch to an Android phone.

**Current state**:
- Apple's iCloud Keychain passkeys **do not** transfer to Android
  natively. They are locked to the Apple ecosystem.
- The user's synced passkeys remain on their Apple ID but are
  inaccessible from the Android device.

**Solutions**:

1. **Third-party password manager as bridge**: If the user had been using
   1Password or Bitwarden for passkeys, their credentials transfer
   seamlessly to the new platform.

2. **FIDO Credential Exchange Protocol (CXP/CXF)**: In October 2024, the
   FIDO Alliance published working draft specifications for a standard
   credential exchange format. Once finalized and implemented by
   credential providers (Apple, Google, Microsoft, 1Password, Bitwarden,
   Dashlane, and others in the Credential Provider Special Interest
   Group), users will be able to export passkeys from one provider and
   import them into another securely.

   > The specifications define "a standard format for transferring
   > credentials in a credential manager including passwords, passkeys
   > and more to another provider in a manner that ensures transfers are
   > not made in the clear and are secure by default."
   > — FIDO Alliance, October 2024

3. **Apple's QR-code import/export**: Apple announced cross-platform
   passkey import/export features using QR codes, matching FIDO CXP/CXF
   standards. Available in recent iOS/macOS versions.

4. **RP-side approach**: GGID allows the user to authenticate via a
   fallback factor (password, TOTP, recovery code) on the new device,
   then enroll a new passkey.

**GGID recommendation**: During enrollment, offer users a choice:
"Use iCloud Keychain" / "Use Google Password Manager" / "Use 1Password".
Recommend cross-platform managers to users likely to switch platforms.

### 4.3 User's Device Is Stolen

**Scenario**: User's phone is stolen. The thief has physical access to
the device.

**Mitigations**:

1. **Remote wipe**: The user uses Find My iPhone / Find My Device /
   Find My Device (Google) / Intune MDM to remotely erase the device.
   This deletes all local copies of passkeys.

2. **Biometric/PIN protection**: Even with physical access, the thief
   cannot use passkeys because:
   - Apple: Face ID / Touch ID / device passcode is required for each
     passkey use.
   - Android: Biometric or PIN is required.
   - The UV (User Verification) flag guarantees this at the protocol level.

3. **Re-enrollment on new device**:
   - **Synced passkey**: If the passkey is synced, the user signs in to
     their iCloud/Google account on the new device, and the passkey
     becomes available automatically.
   - **Device-bound passkey**: The user must go through GGID's recovery
     flow (recovery code, admin-assisted, etc.) to enroll a new credential.

4. **Revoke from RP side**: GGID should allow the user (or admin) to
   revoke the credential associated with the stolen device. Even though
   remote wipe handles the local copy, revocation ensures the credential
   ID is rejected at the server level.

```json
{
  "action": "revoke_credential",
  "credential_id": "base64url-credential-id",
  "reason": "device_stolen",
  "revoked_by": "user:self",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

---

## 5. Recovery Mechanisms for GGID

### 5.1 Multi-Factor Fallback

When a user's primary authentication is passkey-based and all passkeys
are lost, GGID must offer alternative authentication paths.

**Existing GGID capabilities**:
- Password-based login (credentials table, Argon2id hashed)
- Email verification (magic link via `pkg/email`)
- MFA TOTP (mfa_devices table, RFC 6238)
- LDAP authentication (authprovider chain)
- OAuth/OIDC social login (pkg/social connectors)

**Recommended policy**: When passkey is the primary credential, require
at least one fallback factor to be configured:
- Option A: Password + email verification
- Option B: TOTP (with backup codes)
- Option C: Social login (Google/Apple/Microsoft)
- Option D: Recovery codes (see 5.2)

**Implementation**: The enrollment flow checks for fallback factors and
blocks "passkey-only" enrollment unless the user explicitly acknowledges
the risk or sets up recovery codes.

### 5.2 Account Recovery Keys (Recovery Codes)

Similar to TOTP backup codes, recovery codes are generated at passkey
enrollment time and serve as a last-resort authentication factor.

**Design**:
- Generate **10 single-use codes** at enrollment.
- Each code is **20 bytes** of cryptographically random data, base32
  encoded (32 characters).
- Format: `XXXX-XXXX-XXXX-XXXX-XXXX` (groups of 4 for readability).
- Stored as **bcrypt hashes** in the database (same security level as
  passwords — never store plaintext).
- Each code can be used exactly once. Once used, `used_at` is set and
  the code is rejected.
- Rate limited: maximum 5 verification attempts per user per hour.

**Go implementation sketch**:

```go
package recovery

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RecoveryCode represents a single recovery code for a user.
type RecoveryCode struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    uuid.UUID  `json:"user_id"`
	CodeHash  string     `json:"-"`           // bcrypt hash, never exposed
	UsedAt    *time.Time `json:"used_at"`     // nil if unused
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// GenerateCodes creates n recovery codes for a user.
// Returns the plaintext codes (shown to the user once) and the
// RecoveryCode records (with hashed codes for storage).
func GenerateCodes(tenantID, userID uuid.UUID, n int) ([]string, []*RecoveryCode, error) {
	codes := make([]string, 0, n)
	records := make([]*RecoveryCode, 0, n)

	for i := 0; i < n; i++ {
		plaintext, hash, err := generateOneCode()
		if err != nil {
			return nil, nil, fmt.Errorf("generate recovery code %d: %w", i, err)
		}
		codes = append(codes, plaintext)
		records = append(records, &RecoveryCode{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CodeHash:  hash,
			ExpiresAt: time.Now().Add(365 * 24 * time.Hour), // 1-year expiry
			CreatedAt: time.Now(),
		})
	}

	return codes, records, nil
}

// generateOneCode creates a single recovery code.
// Returns the formatted plaintext code and its bcrypt hash.
func generateOneCode() (string, string, error) {
	raw := make([]byte, 20) // 160 bits of entropy
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}

	encoded := base32.StdEncoding.EncodeToString(raw)
	formatted := formatCode(encoded) // "XXXX-XXXX-XXXX-XXXX-XXXX"

	hash, err := bcrypt.GenerateFromPassword([]byte(formatted), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	return formatted, string(hash), nil
}

// formatCode inserts dashes every 4 characters for readability.
func formatCode(s string) string {
	// Take first 20 characters of base32 output
	s = s[:20]
	var result []byte
	for i, c := range s {
		if i > 0 && i%4 == 0 {
			result = append(result, '-')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// VerifyCode checks a plaintext code against all stored hashes for a user.
// Returns the matching RecoveryCode if found, or nil if no match.
// The caller is responsible for marking the code as used.
func VerifyCode(plaintext string, codes []*RecoveryCode) *RecoveryCode {
	for _, rc := range codes {
		if rc.UsedAt != nil {
			continue // already consumed
		}
		if time.Now().After(rc.ExpiresAt) {
			continue // expired
		}
		if bcrypt.CompareHashAndPassword([]byte(rc.CodeHash), []byte(plaintext)) == nil {
			return rc
		}
	}
	return nil
}
```

### 5.3 Admin-Assisted Recovery

For enterprise tenants, an administrator can assist with recovery when a
user cannot self-recover.

**Flow**:
1. User contacts support: "I lost my device and don't have recovery codes."
2. Admin verifies user identity through out-of-band channels (HR
   verification, manager approval, identity document).
3. Admin authenticates to GGID Console with their own MFA.
4. Admin invokes: `POST /api/v1/admin/users/{id}/passkeys/revoke-all`
5. All passkeys for the user are revoked (marked as disabled).
6. Admin optionally resets the user's password or issues a temporary
   enrollment token.
7. All actions are recorded in the audit log with admin identity,
   timestamp, and reason.

```json
{
  "event": "admin.recovery.revoke_all_passkeys",
  "actor": {"type": "admin", "id": "uuid-admin-001"},
  "target": {"type": "user", "id": "uuid-user-042"},
  "reason": "device_loss_no_recovery_codes",
  "revoked_count": 3,
  "timestamp": "2025-01-15T10:30:00Z",
  "ip": "10.0.1.5"
}
```

**Security requirements**:
- Admin must have completed step-up MFA within the last 5 minutes.
- Action requires `user.recovery.manage` permission.
- All admin recovery actions are audit-logged and cannot be deleted.
- Configurable: require two-admin approval for high-privilege users.

### 5.4 Grace Period Login

To prevent lockout during device transitions (e.g., phone being replaced),
GGID can offer a temporary grace period where alternative authentication
is accepted.

**Design**:
- When a user fails passkey authentication N times, the system offers:
  "Having trouble? Sign in with your password instead."
- The grace period is configurable per tenant (default: 72 hours).
- During grace period, password + TOTP or password + email link are
  accepted.
- After successful grace-period login, the user is prompted to enroll a
  new passkey.

**Implementation**:
```go
// GracePeriodConfig defines tenant-level grace period settings.
type GracePeriodConfig struct {
	Enabled        bool          `json:"enabled"`
	Duration       time.Duration `json:"duration"`        // default: 72h
	MaxRetries     int           `json:"max_retries"`     // default: 3
	CooldownPeriod time.Duration `json:"cooldown_period"` // default: 24h
}
```

---

## 6. GGID Implementation Design

### 6.1 Database Schema Changes

**New table: `recovery_codes`**

```sql
-- Recovery codes for passkey account recovery.
-- Generated at passkey enrollment time, stored as bcrypt hashes.
CREATE TABLE IF NOT EXISTS recovery_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    code_hash   TEXT NOT NULL,                  -- bcrypt hash of recovery code
    used_at     TIMESTAMPTZ,                    -- NULL if unused
    expires_at  TIMESTAMPTZ NOT NULL,            -- default 1 year from creation
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    consumed_at TIMESTAMPTZ,                    -- set when used in recovery flow
    consumed_ip INET,                           -- IP of recovery session
    consumed_ua TEXT                            -- user agent of recovery session
);

-- Index for efficient lookup by user
CREATE INDEX idx_recovery_codes_user ON recovery_codes (tenant_id, user_id)
    WHERE used_at IS NULL;

-- Prevent duplicate code hashes (extremely unlikely but defense in depth)
CREATE UNIQUE INDEX uq_recovery_codes_hash ON recovery_codes (code_hash);

-- Enable RLS for multi-tenant isolation
ALTER TABLE recovery_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE recovery_codes FORCE ROW LEVEL SECURITY;

CREATE POLICY recovery_codes_tenant_isolation ON recovery_codes
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

COMMENT ON TABLE recovery_codes IS 'Single-use recovery codes for passkey account recovery';
COMMENT ON COLUMN recovery_codes.code_hash IS 'bcrypt hash of formatted recovery code (never store plaintext)';
```

**Extend `users` table**:

```sql
-- Add recovery_enabled flag to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS recovery_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS recovery_codes_generated_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS recovery_remind_at TIMESTAMPTZ; -- next reminder date
```

**Extend WebAuthn credential storage**:

The existing `Credential` struct in `services/auth/internal/webauthn/handler.go`
should be extended with backup flags:

```sql
-- If passkeys are stored in a webauthn_credentials table, add backup flags:
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_state BOOLEAN NOT NULL DEFAULT FALSE;
```

### 6.2 API Endpoints

#### POST /api/v1/auth/recovery/generate

Generate new recovery codes. Requires an active authenticated session
with recent MFA verification (step-up auth).

**Request**:
```json
{
  "current_password": "user-current-password",
  "mfa_code": "123456"
}
```

**Response** (200 OK):
```json
{
  "codes": [
    "KQXR-MZLA-7YBP-VD3N-JE2W",
    "PNF4-RAUG-XQKW-TS5Z-HB7L",
    "LJY3-WCVD-NK8E-PXR7-QFM2",
    "GZ4S-TPBQ-MH8V-YK7L-JX3N",
    "WD2P-FLRC-BK7E-XVNS-TQ4M",
    "QR5P-SYJK-VM3N-WLBA-EF2D",
    "BH8X-GN7T-KP2W-QMRC-SDL4",
    "VT3K-RFJ5-WC7P-NEXQ-BMZ2",
    "EN9Q-LWP3-VKS4-JXHR-TDCB",
    "KY2M-FPR7-JWNT-VB5L-GSC3"
  ],
  "generated_at": "2025-01-15T10:30:00Z",
  "expires_at": "2026-01-15T10:30:00Z",
  "warning": "Store these codes securely. They will not be shown again."
}
```

**Errors**:
- `401 Unauthorized` — session invalid
- `403 Forbidden` — MFA verification required or expired
- `429 Too Many Requests` — rate limited (max 1 generation per hour)

#### POST /api/v1/auth/recovery/verify

Verify a recovery code and start a recovery session. This endpoint is
unauthenticated (no existing session required).

**Request**:
```json
{
  "username": "jane.doe@example.com",
  "recovery_code": "KQXR-MZLA-7YBP-VD3N-JE2W"
}
```

**Response** (200 OK):
```json
{
  "recovery_session_token": "rs_tok_abc123...",
  "expires_at": "2025-01-15T10:45:00Z",
  "user": {
    "id": "uuid-user-042",
    "username": "jane.doe@example.com",
    "display_name": "Jane Doe"
  },
  "required_actions": [
    "enroll_new_passkey",
    "set_new_password"
  ],
  "revoked_passkeys": [
    {"id": "uuid-cred-1", "name": "iPhone 15", "revoked_at": "2025-01-15T10:30:00Z"},
    {"id": "uuid-cred-2", "name": "MacBook Pro", "revoked_at": "2025-01-15T10:30:00Z"}
  ]
}
```

**Errors**:
- `400 Bad Request` — invalid or expired code
- `404 Not Found` — user not found
- `429 Too Many Requests` — rate limited (max 5 attempts per hour per user)

#### POST /api/v1/auth/passkeys/{id}/revoke

Revoke a specific passkey credential. Requires authenticated session.

**Request**:
```json
{
  "reason": "device_lost"
}
```

**Response** (200 OK):
```json
{
  "credential_id": "uuid-cred-1",
  "revoked_at": "2025-01-15T10:30:00Z",
  "remaining_credentials": 2
}
```

#### GET /api/v1/auth/recovery/status

Check whether recovery codes are set up for the current user.

**Response** (200 OK):
```json
{
  "recovery_enabled": true,
  "codes_generated_at": "2024-06-01T10:00:00Z",
  "codes_expires_at": "2025-06-01T10:00:00Z",
  "unused_codes_remaining": 7,
  "total_codes": 10,
  "next_reminder_at": "2025-03-01T10:00:00Z"
}
```

### 6.3 Recovery Flow (Sequence Diagram)

```
User                Browser                GGID Gateway          Auth Service
 |                     |                        |                      |
 |  "I lost my phone"  |                        |                      |
 |-------------------->|                        |                      |
 |                     |  GET /login            |                      |
 |                     |----------------------->|                      |
 |                     |  Login page (passkey)  |                      |
 |                     |<-----------------------|                      |
 |                     |                        |                      |
 |  Click "Recover"    |                        |                      |
 |-------------------->|                        |                      |
 |                     |  GET /recover          |                      |
 |                     |----------------------->|                      |
 |                     |  Recovery page         |                      |
 |                     |<-----------------------|                      |
 |                     |                        |                      |
 |  Enter username     |                        |                      |
 |  + recovery code    |                        |                      |
 |-------------------->|                        |                      |
 |                     |  POST /recovery/verify |                      |
 |                     |  {username, code}      |                      |
 |                     |----------------------->|                      |
 |                     |                        |  VerifyCode()        |
 |                     |                        |  - bcrypt compare    |
 |                     |                        |  - check expiry      |
 |                     |                        |  - check used_at     |
 |                     |                        |  - rate limit check  |
 |                     |                        |<-------------------->|
 |                     |                        |                      |
 |                     |                        |  Mark code as used   |
 |                     |                        |  Revoke all passkeys |
 |                     |                        |  Create recovery     |
 |                     |                        |    session (15 min)  |
 |                     |                        |  Emit audit event    |
 |                     |                        |                      |
 |                     |  200 Recovery Session  |                      |
 |                     |  + recovery_token      |                      |
 |                     |<-----------------------|                      |
 |                     |                        |                      |
 |  "Enroll new passkey"                       |                      |
 |-------------------->|                        |                      |
 |                     |  POST /webauthn/       |                      |
 |                     |  register/begin        |                      |
 |                     |  (with recovery_token) |                      |
 |                     |----------------------->|                      |
 |                     |                        |  BeginRegistration   |
 |                     |                        |  (recovery session)  |
 |                     |                        |<-------------------->|
 |                     |  Challenge options     |                      |
 |                     |<-----------------------|                      |
 |                     |                        |                      |
 |  Biometric on new   |                        |                      |
 |  device             |                        |                      |
 |-------------------->|                        |                      |
 |                     |  POST /webauthn/       |                      |
 |                     |  register/finish       |                      |
 |                     |----------------------->|                      |
 |                     |                        |  FinishRegistration  |
 |                     |                        |  - verify attestation|
 |                     |                        |  - store new cred    |
 |                     |                        |  - store BE/BS flags |
 |                     |                        |  - generate new      |
 |                     |                        |    recovery codes    |
 |                     |                        |  - end recovery sess |
 |                     |                        |  - emit audit event  |
 |                     |                        |                      |
 |                     |  201 New credential    |                      |
 |                     |  + new recovery codes  |                      |
 |                     |<-----------------------|                      |
 |                     |                        |                      |
 |  Save new codes     |                        |                      |
 |<--------------------|                        |                      |
 |                     |                        |                      |
```

### 6.4 Go Implementation

#### RecoveryCodeService

```go
package recovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCodeNotFound    = errors.New("recovery code not found or already used")
	ErrCodeExpired     = errors.New("recovery code has expired")
	ErrRateLimited     = errors.New("too many recovery attempts")
	ErrRecoverySession = errors.New("recovery session invalid or expired")
)

// Store defines the persistence interface for recovery codes.
type Store interface {
	SaveCodes(ctx context.Context, codes []*RecoveryCode) error
	GetUnusedCodes(ctx context.Context, tenantID, userID uuid.UUID) ([]*RecoveryCode, error)
	MarkUsed(ctx context.Context, codeID uuid.UUID, ip, userAgent string) error
	DeleteAllCodes(ctx context.Context, tenantID, userID uuid.UUID) error
}

// PasskeyRevokeer defines the interface for revoking passkeys during recovery.
type PasskeyRevokeer interface {
	RevokeAllForUser(ctx context.Context, tenantID, userID uuid.UUID, reason string) (int, error)
}

// RateLimiter checks recovery attempt rate limits.
type RateLimiter interface {
	CheckAndIncrement(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
}

// CodeCount is the default number of recovery codes generated.
const CodeCount = 10

// CodeExpiry is the default validity period for recovery codes.
const CodeExpiry = 365 * 24 * time.Hour

// Service handles recovery code generation, verification, and consumption.
type Service struct {
	store     Store
	passkeys  PasskeyRevokeer
	limiter   RateLimiter
	codeCount int
}

// NewService creates a new recovery code service.
func NewService(store Store, passkeys PasskeyRevokeer, limiter RateLimiter) *Service {
	return &Service{
		store:     store,
		passkeys:  passkeys,
		limiter:   limiter,
		codeCount: CodeCount,
	}
}

// GenerateResult contains the plaintext codes and metadata.
type GenerateResult struct {
	Codes      []string
	ExpiresAt  time.Time
	RecordCount int
}

// Generate creates a new set of recovery codes for a user.
// This invalidates any previously unused codes.
func (s *Service) Generate(ctx context.Context, tenantID, userID uuid.UUID) (*GenerateResult, error) {
	// Delete old unused codes first
	if err := s.store.DeleteAllCodes(ctx, tenantID, userID); err != nil {
		return nil, fmt.Errorf("delete old codes: %w", err)
	}

	codes, records, err := GenerateCodes(tenantID, userID, s.codeCount)
	if err != nil {
		return nil, fmt.Errorf("generate codes: %w", err)
	}

	if err := s.store.SaveCodes(ctx, records); err != nil {
		return nil, fmt.Errorf("save codes: %w", err)
	}

	return &GenerateResult{
		Codes:       codes,
		ExpiresAt:   records[0].ExpiresAt,
		RecordCount: len(records),
	}, nil
}

// VerifyRequest contains parameters for code verification.
type VerifyRequest struct {
	TenantID   uuid.UUID
	UserID     uuid.UUID
	Plaintext  string
	ClientIP   string
	UserAgent  string
}

// VerifyResult contains the outcome of successful verification.
type VerifyResult struct {
	ConsumedCodeID uuid.UUID
	RemainingCodes int
	RevokedPasskeys int
}

// Verify checks a recovery code and, if valid, consumes it and revokes passkeys.
func (s *Service) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	// Rate limit: max 5 attempts per hour per user
	rateKey := fmt.Sprintf("recovery:verify:%s:%s", req.TenantID, req.UserID)
	allowed, err := s.limiter.CheckAndIncrement(ctx, rateKey, 5, time.Hour)
	if err != nil {
		return nil, fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		return nil, ErrRateLimited
	}

	// Fetch unused codes
	codes, err := s.store.GetUnusedCodes(ctx, req.TenantID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get codes: %w", err)
	}
	if len(codes) == 0 {
		return nil, ErrCodeNotFound
	}

	// Verify code
	matching := VerifyCode(req.Plaintext, codes)
	if matching == nil {
		return nil, ErrCodeNotFound
	}

	// Consume the code
	if err := s.store.MarkUsed(ctx, matching.ID, req.ClientIP, req.UserAgent); err != nil {
		return nil, fmt.Errorf("mark used: %w", err)
	}

	// Revoke all passkeys for the user
	revokedCount, err := s.passkeys.RevokeAllForUser(ctx, req.TenantID, req.UserID, "recovery_code_used")
	if err != nil {
		// Log but don't fail — the code is already consumed
		revokedCount = 0
	}

	return &VerifyResult{
		ConsumedCodeID:  matching.ID,
		RemainingCodes:  len(codes) - 1,
		RevokedPasskeys: revokedCount,
	}, nil
}
```

#### Recovery Session

```go
package recovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a temporary recovery session.
type Session struct {
	Token     string
	TenantID  uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SessionDuration is the maximum lifetime of a recovery session.
const SessionDuration = 15 * time.Minute

// SessionStore manages recovery sessions. Production should use Redis.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]*Session)}
}

// Create starts a new recovery session for the given user.
func (s *SessionStore) Create(tenantID, userID uuid.UUID) (*Session, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}

	session := &Session{
		Token:     "rs_" + hex.EncodeToString(raw),
		TenantID:  tenantID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	s.mu.Lock()
	s.sessions[session.Token] = session
	s.mu.Unlock()

	return session, nil
}

// Validate checks if a recovery session token is valid and not expired.
func (s *SessionStore) Validate(token string) (*Session, error) {
	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrRecoverySession
	}
	if time.Now().After(session.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, token)
		s.mu.Unlock()
		return nil, ErrRecoverySession
	}
	return session, nil
}

// Consume invalidates a recovery session (called after successful re-enrollment).
func (s *SessionStore) Consume(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// Cleanup removes expired sessions. Should be called periodically.
func (s *SessionStore) Cleanup(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, token)
		}
	}
}
```

#### Integration with Existing Auth Service

The recovery service integrates with the existing WebAuthn handler and
auth service:

```go
// In services/auth/internal/webauthn/handler.go:

// Extended Credential struct with backup flags.
type Credential struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	UserID         uuid.UUID
	Name           string
	CredentialID   []byte
	PublicKey      []byte
	Transports     []string
	Counter        uint32
	BackupEligible bool      // BE flag from registration
	BackupState    bool      // BS flag from registration/auth
	CreatedAt      time.Time
	LastUsedAt     *time.Time
}

// Extended CredentialStore interface.
type CredentialStore interface {
	SaveCredential(ctx context.Context, cred *Credential) error
	GetCredentialsByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*Credential, error)
	GetCredentialByID(ctx context.Context, tenantID uuid.UUID, credID []byte) (*Credential, error)
	UpdateCounter(ctx context.Context, tenantID uuid.UUID, credID []byte, counter uint32) error
	UpdateBackupState(ctx context.Context, tenantID uuid.UUID, credID []byte, backupState bool) error
	DeleteCredential(ctx context.Context, tenantID uuid.UUID, credID []byte) error
	RevokeAllForUser(ctx context.Context, tenantID, userID uuid.UUID, reason string) (int, error)
}
```

```go
// Handler method to extract backup flags during registration:

func (h *Handler) handleFinishRegistration(w http.ResponseWriter, r *http.Request) {
	// ... existing parsing logic ...

	parsedResponse, err := protocol.ParseCredentialCreationResponse(r)
	if err != nil {
		http.Error(w, "parse error", http.StatusBadRequest)
		return
	}

	// Verify with go-webauthn
	credential, err := h.wbn.CreateCredential(webAuthnUser, *parsedResponse)
	if err != nil {
		http.Error(w, "verification failed", http.StatusBadRequest)
		return
	}

	// Extract backup flags from authenticator data
	flags := parsedResponse.Response.AttestationObject.AuthData.Flags
	backupEligible := flags.HasBackupEligible()
	backupState := flags.HasBackupState()

	// Store credential with backup flags
	cred := &Credential{
		ID:             uuid.New(),
		TenantID:       session.tenantID,
		UserID:         session.userID,
		CredentialID:   credential.ID,
		PublicKey:      credential.PublicKey,
		Transports:     credential.TransportStrings(),
		Counter:        credential.Authenticator.Counter,
		BackupEligible: backupEligible,
		BackupState:    backupState,
		CreatedAt:      time.Now(),
	}

	if err := h.creds.SaveCredential(r.Context(), cred); err != nil {
		http.Error(w, "save credential", http.StatusInternalServerError)
		return
	}

	// After successful registration, generate recovery codes
	// (if this is the user's first passkey and recovery not yet enabled)
	json.NewEncoder(w).Encode(map[string]any{
		"status":          "success",
		"credential_id":   cred.ID,
		"backup_eligible": backupEligible,
		"backup_state":    backupState,
	})
}
```

---

## 7. Security Considerations

### 7.1 Recovery Code Entropy

| Parameter | Value | Rationale |
|---|---|---|
| Random bytes | 20 (160 bits) | Exceeds OWASP minimum of 128 bits for secrets |
| Encoding | Base32 | Human-readable, no ambiguous characters |
| Formatted length | 24 chars (`XXXX-XXXX-XXXX-XXXX-XXXX`) | Easy to read/type |
| Entropy | 160 bits | Brute-force infeasible: 2^160 ≈ 1.46 × 10^48 |
| Collision probability | Negligible | Birthday bound at 2^80 codes |

### 7.2 Rate Limiting

| Limit | Value | Scope |
|---|---|---|
| Verification attempts | 5 per hour | Per user |
| Code generation | 1 per hour | Per user |
| Recovery sessions | 1 active per user | Global |
| IP-based throttle | 20 per hour | Per IP address |

Implementation should use a sliding window counter (Redis or in-memory
with the existing `SlidingRateLimiter` in gateway middleware).

### 7.3 Session Security

- **Recovery session lifetime**: 15 minutes (configurable per tenant).
- **Session scope**: Can only enroll new credentials and set new password.
  Cannot access user data, read messages, or perform sensitive operations.
- **Session binding**: Recovery session token is bound to the IP address
  that initiated it. If the IP changes, the session is invalidated.
- **Single active session**: Only one recovery session per user at a time.
  Starting a new one invalidates any existing one.

### 7.4 Step-Up Authentication

Generating or viewing recovery codes requires step-up authentication:
- The user must have completed MFA (TOTP or WebAuthn) within the last
  5 minutes.
- This prevents an attacker with a stolen session token from generating
  new recovery codes (which would give them a persistent backdoor).

### 7.5 Audit Trail

All recovery-related events are logged to the audit service:

```json
{
  "event_type": "recovery.code_generated",
  "tenant_id": "uuid-tenant",
  "actor": {"type": "user", "id": "uuid-user"},
  "details": {"code_count": 10, "expires_at": "2026-01-15T10:30:00Z"},
  "ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

Event types:
- `recovery.code_generated`
- `recovery.code_verified`
- `recovery.code_consumed`
- `recovery.session_started`
- `recovery.session_expired`
- `recovery.session_consumed`
- `recovery.passkey_revoked`
- `admin.recovery.revoke_all_passkeys`

### 7.6 Hash Storage

Recovery codes are hashed with **bcrypt** (cost 10+), matching the
security level of password storage. Alternative: **argon2id** for
consistent hashing across all secret types.

```go
// Use bcrypt with a reasonable cost factor
hash, err := bcrypt.GenerateFromPassword([]byte(code), 12)
```

**Never** store recovery codes in plaintext, reversible encryption, or
unsalted hashes (MD5, SHA1).

---

## 8. UX Best Practices

### 8.1 Enrollment Integration

**Mandatory, not optional**: When a user enrolls their first passkey,
the system should present recovery codes as part of the same flow. The
user should not be able to skip this step without explicit confirmation:

```
+---------------------------------------------------+
|  Passkey Created Successfully!                    |
|                                                   |
|  Your recovery codes:                             |
|                                                   |
|  KQXR-MZLA-7YBP-VD3N-JE2W                        |
|  PNF4-RAUG-XQKW-TS5Z-HB7L                        |
|  LJY3-WCVD-NK8E-PXR7-QFM2                        |
|  GZ4S-TPBQ-MH8V-YK7L-JX3N                        |
|  WD2P-FLRC-BK7E-XVNS-TQ4M                        |
|  QR5P-SYJK-VM3N-WLBA-EF2D                        |
|  BH8X-GN7T-KP2W-QMRC-SDL4                        |
|  VT3K-RFJ5-WC7P-NEXQ-BMZ2                        |
|  EN9Q-LWP3-VKS4-JXHR-TDCB                        |
|  KY2M-FPR7-JWNT-VB5L-GSC3                        |
|                                                   |
|  [ Download ]  [ Print ]  [ Copy All ]           |
|                                                   |
|  [ ] I have saved these codes securely            |
|                                                   |
|  [ Continue ]                                     |
+---------------------------------------------------+
```

### 8.2 Display Once

- Recovery codes are shown **exactly once** at generation time.
- Subsequent views show only metadata (count remaining, expiry date).
- If the user loses their codes, they must generate a new set (which
  invalidates all old codes).

### 8.3 Periodic Reminders

- Every 90 days (configurable), prompt users to verify they still have
  their recovery codes.
- Prompt: "Do you still have access to your recovery codes?"
  - Yes → reset reminder timer
  - No → offer to regenerate codes

### 8.4 Don't Expose Code Properties

- Never display code entropy, hash, or partial code content in the UI.
- Never include codes in API responses after initial generation.
- Never log codes in plaintext (audit logs record only `code_id`).

### 8.5 Recovery Entry UX

The recovery code entry should:
- Accept paste (most users will paste from a password manager or file).
- Normalize input (remove spaces, dashes, uppercase).
- Show masked input (like password field) with show/hide toggle.
- Clear error messages: "Invalid or expired code" (not "Code KQXR-... is
  invalid" — avoids information leakage).

---

## 9. Industry Approaches

### 9.1 Apple — iCloud Account Recovery

**Model**: Device-based recovery.

- Apple's account recovery is anchored to the user's trusted devices and
  trusted phone number.
- If a user loses all devices, they initiate recovery at
  `iforgot.apple.com`, which triggers a multi-step process:
  1. Confirm Apple ID and trusted phone number.
  2. Wait for a recovery contact or Apple's verification.
  3. Apple may require a 24-hour waiting period for security.
- Apple also supports **Recovery Contacts**: trusted people who can
  help verify identity during recovery (added in iOS 15).

**Strength**: Strong security with human verification.
**Weakness**: Can take hours to days. Not suitable for urgent access.

### 9.2 Google — Google Account Recovery

**Model**: Knowledge-and-device-based recovery.

- Google uses a combination of factors:
  1. Previously used passwords.
  2. Verification codes sent to backup email/phone.
  3. Device verification (devices previously signed in).
  4. Security questions (less common now).
  5. Google Prompt on other signed-in devices.
- Google has invested heavily in ML-based risk assessment to detect
  fraudulent recovery attempts.

**Strength**: Fast when the user has another signed-in device.
**Weakness**: Knowledge-based questions are weak against targeted attacks.

### 9.3 Microsoft — Microsoft Account Recovery

**Model**: Similar to Google — multi-factor recovery.

- Microsoft account recovery uses:
  1. Email or phone verification.
  2. Microsoft Authenticator app approval.
  3. Previously used device verification.
- For enterprise (Entra ID), admin-assisted recovery is the primary path.

**Strength**: Enterprise-friendly with Entra ID admin workflows.
**Weakness**: Consumer recovery relies on email/SMS (phishable).

### 9.4 1Password — Secret Key + Master Password

**Model**: Zero-knowledge with out-of-band secret.

- 1Password uses two secrets:
  1. **Master Password**: Chosen by the user (human-memorable).
  2. **Secret Key**: 128-bit random key generated at signup, stored in
     the Emergency Kit (a downloadable PDF).
- The vault encryption key is derived from both: `KDF(Master Password, Secret Key)`.
- 1Password's servers never receive the Secret Key — it's combined
  locally before any network transmission.
- **Recovery**: If a user loses all devices, they need the Emergency Kit
  (Secret Key) + their Master Password. Both are required; neither alone
  is sufficient.

**Strength**: Cryptographically robust. The Secret Key makes brute-force
impossible even if 1Password's servers are breached.
**Weakness**: User must safely store the Emergency Kit. If both the
Emergency Kit and Master Password are lost, recovery is impossible
(by design — zero-knowledge means 1Password cannot help).

### 9.5 Comparison Summary

| Approach | Recovery Speed | Security | User Burden | Phish-Resistant |
|---|---|---|---|---|
| Apple iCloud | Hours-Days | High | Medium | Yes (device-based) |
| Google Account | Minutes-Hours | Medium-High | Low-Medium | Partial (SMS phishable) |
| Microsoft Account | Minutes-Hours | Medium-High | Low-Medium | Partial (SMS phishable) |
| 1Password | Immediate (if Emergency Kit available) | Very High | Medium (store Emergency Kit) | Yes (zero-knowledge) |
| **GGID Recovery Codes** | **Immediate (if codes saved)** | **High** | **Low-Medium** | **Yes (bcrypt hashed)** |

### 9.6 FIDO Credential Exchange (CXP/CXF)

The FIDO Alliance's Credential Exchange Protocol (CXP) and Credential
Exchange Format (CXF), published as working drafts in October 2024,
define standards for securely transferring credentials between providers.

**Key properties**:
- Credentials are never transferred in the clear.
- Transfer is authenticated and authorized by both the source and
  destination credential providers.
- Supports passwords, passkeys, and other credential types.
- Developed by the Credential Provider Special Interest Group: 1Password,
  Apple, Bitwarden, Dashlane, Enpass, Google, Microsoft, NordPass, Okta,
  Samsung, SK Telecom.

**Impact on GGID**: Once CXP/CXF is finalized and widely implemented,
users will be able to move passkeys from Apple to Google, or from iCloud
Keychain to 1Password, reducing platform lock-in and simplifying recovery
for platform-switching users. GGID should monitor this standard and
provide guidance to users when cross-provider transfer becomes available.

---

## Appendix A: Migration Checklist for GGID

- [ ] Add `recovery_codes` table migration
- [ ] Add `recovery_enabled`, `recovery_codes_generated_at` columns to users
- [ ] Add `backup_eligible`, `backup_state` columns to passkey credential storage
- [ ] Implement `RecoveryCodeService` (Generate, Verify, Consume)
- [ ] Implement `SessionStore` for recovery sessions (Redis-backed)
- [ ] Add API endpoints (generate, verify, revoke, status)
- [ ] Integrate recovery code generation into passkey enrollment flow
- [ ] Add admin-assisted recovery endpoint (`/admin/users/{id}/passkeys/revoke-all`)
- [ ] Add audit events for all recovery operations
- [ ] Add rate limiting for recovery endpoints
- [ ] Add Console UI for recovery code setup and management
- [ ] Add Console UI for admin-assisted recovery
- [ ] Add tenant configuration for grace period settings
- [ ] Write integration tests for recovery flow
- [ ] Document recovery flow in user-facing help docs

## Appendix B: References

- **WebAuthn Level 3 (W3C)**: Backup Eligible and Backup State flags
  specification — `https://www.w3.org/TR/webauthn-3/`
- **FIDO Alliance CXP/CXF**: Credential Exchange Protocol and Format
  (October 2024 working draft) — `https://fidoalliance.org/`
- **Apple Platform Security**: iCloud Keychain security overview —
  `https://support.apple.com/guide/security/`
- **Google Password Manager**: Sync and security documentation —
  `https://support.google.com/accounts/`
- **1Password Security**: Secret Key architecture —
  `https://support.1password.com/secret-key-security/`
- **OWASP Authentication Cheat Sheet**: Recovery code best practices —
  `https://cheatsheetseries.owasp.org/`
- **passkeys.dev**: Developer resource for passkey implementation —
  `https://passkeys.dev/`
