# WebAuthn/FIDO2 Enterprise Features: Implementation Guide for GGID

> **Focus**: Production implementation of enterprise WebAuthn features — Conditional UI, device public keys, enterprise attestation (AAGUID allowlist), hybrid transport (QR+Bluetooth), and passkey recovery — building on GGID's existing `passkey_handler.go`.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `fido2-passkey-deep-dive.md` (theory), `auth/webauthn/handler.go:477` (existing impl).
>
> **Checklist Compliance**: DoD per backlog item (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: WebAuthn](#2-ggid-current-state-webauthn)
3. [Gap Analysis](#3-gap-analysis)
4. [Conditional UI (Autofill)](#4-conditional-ui-autofill)
5. [Device Public Key (DPK)](#5-device-public-key-dpk)
6. [Enterprise Attestation (AAGUID)](#6-enterprise-attestation-aaguid)
7. [Hybrid Transport (QR + Bluetooth)](#7-hybrid-transport-qr--bluetooth)
8. [Passkey Recovery](#8-passkey-recovery)
9. [Testing Strategy](#9-testing-strategy)
10. [Implementation Backlog with DoD](#10-implementation-backlog-with-dod)
11. [Competitive Differentiation](#11-competitive-differentiation)

---

## 1. Executive Summary

GGID has a **mature WebAuthn implementation** — `auth/webauthn/handler.go` (500+ lines) supports registration, authentication, multi-device, and user verification. However, it lacks enterprise features needed for large deployments.

**Existing WebAuthn:**
- Registration begin/finish (`handler.go:477`) ✅
- Authentication begin/finish ✅
- Discoverable credentials (passkeys) ✅
- User verification (UV) flag ✅
- Multiple authenticators per user ✅
- Exclusion list (prevent re-registration) ✅

**Missing enterprise features:**
1. **Conditional UI** — No browser autofill support (autocomplete="webauthn")
2. **Device public key (DPK)** — No per-device keypair augmentation
3. **Enterprise attestation** — No AAGUID allowlist / authenticator model restriction
4. **Hybrid transport** — No QR + Bluetooth cross-device auth
5. **Passkey recovery** — No temporary access pass or re-enrollment flow
6. **Authenticator metadata** — No FIDO MDS (Metadata Service) integration

**Recommendation**: Add Conditional UI autofill, AAGUID allowlist, temporary access pass recovery, and FIDO MDS integration. Hybrid transport is browser-native (no GGID changes needed).

---

## 2. GGID Current State

| Component | File | Status |
|-----------|------|--------|
| WebAuthn handler | `auth/webauthn/handler.go:477` | ✅ 500+ lines |
| Registration | handler BeginRegistration/FinishRegistration | ✅ |
| Authentication | handler BeginLogin/FinishLogin | ✅ |
| Passkey support | `auth/server/passkey_handler.go` | ✅ |
| Passwordless | `auth/server/passwordless_*.go` | ✅ |
| Device attestation | `auth/server/device_attest_handler.go:10` | ⚠️ Basic TPM |

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No Conditional UI | Users must click "sign in with passkey" |
| 2 | No AAGUID allowlist | Any authenticator accepted (consumer-grade) |
| 3 | No DPK | No per-device proof beyond credential |
| 4 | No passkey recovery | Lost device = locked out |
| 5 | No FIDO MDS | Can't verify authenticator certification level |
| 6 | No temporary access pass | No recovery mechanism |

---

## 4. Conditional UI (Autofill)

### What It Is

Browser autofills passkey credentials in the username field — user just authenticates with biometrics without clicking any button.

### Frontend Change (Console + Login Page)

```html
<!-- Add autocomplete="webauthn" to username input -->
<input
  type="text"
  name="username"
  autocomplete="webauthn"
  placeholder="Username or passkey"
/>
```

### Backend Support

```go
// Allow PublicKeyCredentialRequestOptions with empty allowCredentials
// (for discoverable credentials / passkey autofill)
func BeginPasskeyLogin(ctx context.Context) (*PublicKeyCredentialRequestOptions, error) {
    options := &webauthn.PublicKeyCredentialRequestOptions{
        Challenge:          generateChallenge(),
        UserVerification:   webauthn.VerificationRequired,
        AllowCredentials:   []webauthn.CredentialDescriptor{},  // Empty = autofill
        // Browser will show all matching passkeys via Conditional UI
    }
    return options, nil
}
```

### API Change

```bash
# New endpoint: begin passkey login with autofill
POST /api/v1/auth/passkey/login/begin
# No username required — browser shows passkey picker
# Returns: PublicKeyCredentialRequestOptions with empty allowCredentials

# Existing: finish passkey login (unchanged)
POST /api/v1/auth/passkey/login/finish
```

---

## 5. Device Public Key (DPK)

### What It Is

Each authenticator provides a per-device public key (DPK) that signs the assertion. This proves the same physical device was used, independent of the credential.

### Implementation

```go
// During registration: extract DPK from attestation
type DevicePublicKey struct {
    KeyType     string  // "EC2" or "RSA"
    Algorithm   int     // COSE algorithm
    PublicKey   []byte  // Raw public key
    AAGUID      string  // Authenticator model ID
}

// Store DPK alongside WebAuthn credential
func StoreCredential(cred *WebAuthnCredential) error {
    if cred.DevicePublicKey != nil {
        // Store DPK for later verification
        storeDevicePublicKey(cred.UserID, cred.ID, cred.DevicePublicKey)
    }
}

// During authentication: verify DPK signature
func VerifyAssertion(assertion *AssertionResponse) error {
    // Standard WebAuthn verification...
    
    // DPK verification (if present)
    if assertion.DevicePublicKeySig != nil {
        dpk := getStoredDPK(assertion.CredentialID)
        if !verifyDPKSignature(dpk, assertion.AuthenticatorData, assertion.DevicePublicKeySig) {
            return ErrInvalidDevicePublicKey
        }
    }
    return nil
}
```

---

## 6. Enterprise Attestation (AAGUID)

### AAGUID Allowlist

```sql
CREATE TABLE webauthn_aaguid_allowlist (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    aaguid          VARCHAR(36) NOT NULL,     -- Authenticator Attestation GUID
    name            VARCHAR(128),             -- "YubiKey 5 NFC" / "Windows Hello"
    certification_level VARCHAR(16),          -- 'L1', 'L2', 'L3'
    trusted         BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, aaguid)
);
```

### Enforcement

```go
func ValidateAttestation(att *webauthn.Attestation, tenantID uuid.UUID) error {
    allowlist := getAAGUIDAllowlist(tenantID)
    
    if len(allowlist) == 0 {
        return nil  // No allowlist = allow all
    }
    
    aaguid := att.AuthenticatorData.AAGUID.String()
    if _, ok := allowlist[aaguid]; !ok {
        return fmt.Errorf("authenticator model %s not in allowlist", aaguid)
    }
    
    return nil
}
```

### AAGUID Examples

| AAGUID | Authenticator |
|--------|-------------|
| `00000000-0000-0000-0000-000000000000` | None (consumer) |
| `cb69481c-5b04-40c8-b3a5-3b3f8a4f5e3a` | Windows Hello |
| `ee882879-721c-4913-9775-3571376e4364` | YubiKey 5 |
| `ad8ce93e-356a-4b5e-a612-4c868238fa0c` | iCloud Keychain |

---

## 7. Hybrid Transport (QR + Bluetooth)

### What It Is

User scans QR code on desktop → phone authenticates via biometric → desktop receives credential. This is **browser-native** (Chrome/Safari/Edge handle the transport).

### GGID Support (Minimal)

No changes needed — the WebAuthn API handles hybrid transport transparently. The server just sees a normal WebAuthn assertion. The only requirement:

```go
// Ensure options don't restrict transport
options := &webauthn.PublicKeyCredentialRequestOptions{
    // Don't set TransportHints — let browser decide
    // Hybrid transport automatically available if:
    //   1. User has phone with passkey
    //   2. Desktop browser supports hybrid (Chrome 126+, Safari 17+)
}
```

---

## 8. Passkey Recovery

### Temporary Access Pass (TAP)

```
Scenario: User loses phone with only passkey

1. Admin issues TAP:
   POST /api/v1/auth/recovery/temporary-access-pass
   { user_id: "uuid", expires_in_minutes: 15, uses: 1 }

2. User uses TAP to authenticate:
   POST /api/v1/auth/login
   { tap: "tap_abc123" }  // Instead of password/passkey

3. After TAP login, user must re-enroll new passkey:
   POST /api/v1/auth/webauthn/register/begin
   → Standard registration flow

4. Old passkey credentials marked as "lost" (revoked)
```

### TAP Database

```sql
CREATE TABLE temporary_access_passes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    pass_hash       VARCHAR(256) NOT NULL,    -- bcrypt hash
    max_uses        INT DEFAULT 1,
    uses_count      INT DEFAULT 0,
    expires_at      TIMESTAMPTZ NOT NULL,
    issued_by       UUID NOT NULL,            -- Admin who issued
    issued_reason   TEXT,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### Multi-Device Encouragement

```bash
# API: Check if user has only one passkey → recommend adding backup
GET /api/v1/auth/webauthn/credentials/recovery-status

# Response:
{
  "credentials_count": 1,
  "recommend_backup": true,
  "message": "Add a second passkey to prevent lockout"
}
```

---

## 9. Testing Strategy

### Virtual Authenticators (Chrome DevTools)

```python
# Selenium/Playwright: use virtual authenticator
protocol = "webauthn:virtualAuthenticatorOptions="
options = {
    "protocol": "ctap2",
    "transport": "internal",
    "hasResidentKey": True,
    "hasUserVerification": True,
    "isUserVerified": True
}
# Register virtual authenticator in browser
driver.execute_cdp_cmd("WebAuthn.addVirtualAuthenticator", {"options": options})
```

### WebAuthn Test Matrix

| Test | Method | Expected |
|------|--------|----------|
| Registration with UV | Virtual authenticator | Success, UV=true |
| Registration without UV | Virtual (UV=false) | Rejected (if required) |
| AAGUID allowlist | Virtual with specific AAGUID | Blocked if not in list |
| Passkey autofill | Browser Conditional UI | Shows passkey picker |
| TAP recovery | Issue TAP → login → re-enroll | Full recovery works |
| Lost device | Revoke credential → try auth | Rejected |

---

## 10. Implementation Backlog with DoD

### P0 — Conditional UI + Recovery (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Conditional UI autofill | ✅ autocomplete="webauthn" ✅ Empty allowCredentials API ✅ ≥3 tests | 2d |
| 2 | Temporary Access Pass (TAP) | ✅ Admin issue TAP ✅ TAP login ✅ Auto-expire ✅ DB-backed ✅ ≥3 tests | 3d |
| 3 | Passkey recovery status | ✅ Check credential count ✅ Backup recommendation ✅ ≥3 tests | 1d |

### P1 — Enterprise Attestation (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 4 | AAGUID allowlist DB + enforcement | ✅ CRUD for allowlist ✅ Registration checks AAGUID ✅ ≥3 tests | 3d |
| 5 | FIDO MDS integration | ✅ Fetch metadata ✅ Verify certification level ✅ ≥3 tests | 3d |
| 6 | Device Public Key (DPK) | ✅ Extract DPK from attestation ✅ Verify on auth ✅ ≥3 tests | 3d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 7 | Multi-device sync | Sync passkeys across user's devices |
| 8 | WebAuthn analytics | Registration rate, UV success rate, authenticator distribution |
| 9 | Passwordless migration | Auto-suggest passkey after password login |

---

## 11. Competitive Differentiation

| Feature | GGID (target) | Auth0 | Okta | Microsoft Entra | Keycloak |
|---------|---------------|-------|------|-----------------|----------|
| **Conditional UI** | Target | Yes | Yes | Yes | No |
| **AAGUID allowlist** | Target | Yes | Yes | Yes | No |
| **Passkey recovery (TAP)** | Target | Custom | Yes | Yes | No |
| **Device public key** | Target | No | No | Yes | No |
| **Hybrid transport** | Browser-native | ✅ | ✅ | ✅ | ✅ |
| **FIDO MDS** | Target | Yes | Yes | Yes | No |
| **Open source** | Yes | No | No | No | Yes |

---

## References

- [WebAuthn Level 3 (W3C)](https://www.w3.org/TR/webauthn-3/) — Specification
- [CTAP 2.1](https://fidoalliance.org/specs/fido-v2.1-ps/) — Client-to-Authenticator Protocol
- [FIDO Metadata Service](https://fidoalliance.org/metadata/) — Authenticator certification
- [Conditional UI Spec](https://github.com/w3c/webauthn/wiki/Explainer:-WebAuthn-Conditional-UI) — Autofill
- [Device Public Key](https://w3c.github.io/webauthn/#sctn-device-publickey) — DPK extension
- [Virtual Authenticators](https://www.w3.org/TR/webauthn-3/#sctn-verifying-authenticator-models) — Testing
- [GGID WebAuthn Handler](../services/auth/internal/webauthn/handler.go) — At line 477
- [GGID Passkey Handler](../services/auth/internal/server/passkey_handler.go) — Existing
- [GGID FIDO2 Deep Dive](./fido2-passkey-deep-dive.md) — Theoretical analysis
- [GGID FIDO2 Certification Guide](./fido2-certification-guide.md) — Certification prep
