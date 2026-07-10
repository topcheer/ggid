# WebAuthn / Passkey Best Practices for 2024

> **Research Document** — Comprehensive guide for implementing WebAuthn/Passkey authentication
> in production systems, with specific recommendations for GGID IAM Suite.
>
> **Author:** GGID Dev Team
> **Last Updated:** 2024
> **Spec Versions Covered:** WebAuthn Level 2 (W3C Recommendation, April 2021) and Level 3 (Working Draft)

---

## Table of Contents

1. [WebAuthn Level 2/3 Spec Updates](#1-webauthn-level-23-spec-updates)
2. [Conditional UI (Autofill)](#2-conditional-ui-autofill)
3. [Hybrid Transport (ctap2qr)](#3-hybrid-transport-ctap2qr)
4. [Device Registration Flows](#4-device-registration-flows)
5. [Relying Party (RP) Best Practices](#5-relying-party-rp-best-practices)
6. [Migration & Coexistence](#6-migration--coexistence)
7. [Security Considerations](#7-security-considerations)
8. [Implementation Libraries](#8-implementation-libraries)
9. [GGID Implementation Roadmap](#9-ggid-implementation-roadmap)

---

## 1. WebAuthn Level 2/3 Spec Updates

### Level 1 → Level 2 (Published April 2021)

WebAuthn Level 2 was promoted to a W3C Recommendation in April 2021. It is a series of
fixes and enhancements to the original L1 spec, focused on expanding functionality for
specific enterprise and consumer use cases.

#### Key Changes in Level 2

| Feature | Description | Impact |
|---|---|---|
| **Enterprise Attestation** | Allows RPs to request uniquely identifying information from authenticators during registration. Designed for controlled enterprise deployments where organizations wish to tie registrations to specific authenticators. Two modes: Vendor-Facilitated EA and Platform-Managed EA. | Enables device management in corporate environments. Privacy concern: usage tracking — mitigated by authenticator pre-loading. |
| **Apple Anonymous Attestation** | New attestation statement format (`apple`) added to the spec. | Enables iOS/macOS platform authenticators to provide verifiable attestation without user identification. |
| **Cross-Origin iFrame Support** | Allows `get()` commands (authentication only, not credential creation) from cross-origin iFrames. | Enables payment flows, embedded checkout, and third-party authentication widgets. |
| **LargeBlob Extension** | Allows RPs to store opaque, mutable data associated with a credential on the authenticator. | Useful for issuing client-side certificates or storing encrypted configuration. |
| **credProps Extension** | Returns properties of a created credential, including whether it is discoverable (resident key). | Critical for passkey UX — RPs need to know if a credential supports Conditional UI. |
| **Discoverable Credentials (`residentKey`)** | Renamed from "Resident Key". Options: `discouraged`, `preferred`, `required`. | Discoverable credentials are the foundation of passkeys and usernameless login. |
| **FIDO AppID Exclusion Extension** | Migration bridge between legacy U2F and WebAuthn/FIDO2. Allows RPs to exclude U2F credentials during registration. | Prevents duplicate credentials when migrating from U2F to WebAuthn. |
| **Signature Count Semantics** | Clarified that `signCount` is a *hint*, not a guarantee. Authenticators may return 0 or a monotonically increasing counter. RP enforcement is recommended but not mandatory. | RPs should track but not reject on count=0. If count > 0, verify it increased. |

#### Attestation Format Support (as of L3 draft)

| Format | Since | Purpose |
|---|---|---|
| `packed` | L1 | Generic, widely supported |
| `tpm` | L1 | TPM-based platform authenticators (Windows Hello) |
| `android-key` | L1 | Android platform authenticators |
| `android-safetynet` | L1 | Legacy Android SafetyNet |
| `fido-u2f` | L1 | Legacy U2F security keys |
| `none` | L1 | No attestation (privacy-preserving default) |
| `apple` | L2 | Apple Anonymous attestation |
| `compound` | L3 (draft) | Compound attestation for hardware-backed credentials |

### Level 3 (Working Draft)

Level 3 is currently a Working Draft and introduces several forward-looking features:

#### Key Features in Level 3 Draft

1. **User Agent Hints**: The `hints` parameter in `PublicKeyCredentialCreationOptions`
   and `PublicKeyCredentialRequestOptions` lets RPs suggest the type of authenticator
   to use (`security-key`, `client-device`, `hybrid`). This improves UX by guiding
   browsers to show the right UI faster.

   ```javascript
   // Suggest the user authenticate with a phone via hybrid transport
   navigator.credentials.get({
     publicKey: {
       // ... other options ...
       hints: ["hybrid"]
     }
   });
   ```

2. **Signal Methods (RP ID Validation)**: Asynchronous RP ID validation algorithm
   lets `signal` methods validate RP IDs in parallel, improving performance.

3. **LargeBlob as First-Class**: LargeBlob storage extension moved from experimental
   to recommended.

4. **PRF (Pseudo-Random Function) Extension**: Allows web applications to derive
   secret keys directly from a passkey during authentication. Enables end-to-end
   encryption use cases where the RP derives symmetric keys from the authenticator.

5. **Credential Properties Extension Updates**: Enhanced `credProps` to return more
   metadata about created credentials.

6. **Deprecation of Several Extensions**: Removed `txAuthGeneric`, `authnSel`,
   `exts`, `uvi`, `loc`, `biometricPerfBounds` due to lack of adoption or
   privacy/security concerns. The `uvm` (User Verification Method) extension was
   also deprecated in L3.

#### Signature Count Best Practice (L2/L3)

```go
// Recommended signature count verification logic
func verifySignCount(storedCount, receivedCount uint32) error {
    if receivedCount > 0 {
        // Authenticator supports counter — enforce monotonicity
        if receivedCount <= storedCount {
            return fmt.Errorf("possible cloned credential: counter did not increase (stored=%d, received=%d)",
                storedCount, receivedCount)
        }
    }
    // If receivedCount == 0, authenticator doesn't support counters — skip check
    return nil
}
```

> **GGID Status**: The current implementation stores `Counter` but does not enforce
> monotonicity checks during authentication. See [GGID Roadmap](#9-ggid-implementation-roadmap).

---

## 2. Conditional UI (Autofill)

Conditional UI (also called "Conditional Mediation" or "Passkey Autofill") integrates
passkey authentication into the browser's native autofill dropdown, letting users
select a passkey alongside password suggestions in the same input field.

### How It Works

The flow has two phases:

**Phase 1 — Page Load (background):**
1. Client checks `PublicKeyCredential.isConditionalMediationAvailable()`
2. Client calls server's conditional UI endpoint to get `PublicKeyCredentialRequestOptions`
3. Client calls `navigator.credentials.get()` with `mediation: "conditional"`
4. Browser populates autofill dropdown with available passkeys (no modal dialog)

**Phase 2 — User Interaction:**
5. User focuses the username input field
6. Browser shows passkey suggestions mixed with saved passwords
7. User selects a passkey → device biometric prompt (Face ID / Touch ID / Windows Hello)
8. Signed assertion sent to server for verification
9. Server validates signature → user logged in

### Browser Support

| Browser | Minimum Version | Platform |
|---|---|---|
| Chrome | 107+ | Windows, macOS, Linux |
| Safari | 16+ | macOS, iOS |
| Firefox | 122+ (behind flag in earlier versions) | Windows, macOS, Linux |
| Edge | 107+ | Windows, macOS |
| Chrome (Android) | 107+ | Android (via Credential Manager) |
| Safari (iOS) | 16+ | iOS |

### Implementation Pattern

#### HTML (Autofill Token)

```html
<!-- The "webauthn" autocomplete token signals the browser to surface passkeys -->
<label for="username">Username</label>
<input
  type="text"
  id="username"
  name="username"
  autocomplete="username webauthn"
/>

<label for="password">Password</label>
<input
  type="password"
  id="password"
  name="password"
  autocomplete="current-password webauthn"
/>
```

#### JavaScript (Conditional UI Flow)

```javascript
// Global AbortController for cancellation
let conditionalAbortController = new AbortController();

async function initConditionalUI() {
  // 1. Check browser support
  if (!window.PublicKeyCredential ||
      !PublicKeyCredential.isConditionalMediationAvailable) {
    return; // Fall back to regular login
  }

  const isCMA = await PublicKeyCredential.isConditionalMediationAvailable();
  if (!isCMA) {
    return; // Fall back to regular login
  }

  // 2. Get options from server (allowCredentials should be empty for discoverable)
  const response = await fetch('/api/v1/webauthn/auth/begin', {
    method: 'POST',
    headers: { 'X-Tenant-ID': tenantId },
  });
  const { publicKey } = await response.json();

  // 3. Start conditional mediation (non-blocking)
  try {
    const credential = await navigator.credentials.get({
      publicKey: publicKey,
      mediation: 'conditional',
      signal: conditionalAbortController.signal,
    });

    // 4. Send assertion to server
    const authResponse = await fetch('/api/v1/webauthn/auth/finish', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantId,
      },
      body: JSON.stringify(credential),
    });

    if (authResponse.ok) {
      window.location.href = '/dashboard';
    }
  } catch (err) {
    if (err.name === 'AbortError') {
      // Expected when user clicks a button or starts modal flow
      console.log('Conditional UI aborted — user chose another path');
    } else {
      console.error('Conditional UI error:', err);
    }
  }
}

// Handle modal login button click — abort conditional UI
document.getElementById('loginBtn').addEventListener('click', () => {
  conditionalAbortController.abort();
  conditionalAbortController = new AbortController(); // Reset for next time
  // Start modal WebAuthn flow...
});

// Initialize on page load
initConditionalUI();
```

### Key Requirements

1. **Discoverable Credentials Only**: Conditional UI requires resident keys
   (discoverable credentials). Non-resident keys cannot populate the autofill menu
   because authenticators don't store user-specific data for them.

2. **Empty `allowCredentials`**: For true usernameless login, send an empty
   `allowCredentials` array so the browser discovers all matching passkeys.

3. **AbortController**: Always create a globally-scoped `AbortController` to cancel
   the conditional request when the user chooses an alternative login path. A fresh
   controller must be created each time — reusing an aborted one causes immediate
   cancellation.

4. **No Timeout**: Do not set `timeout` parameters — users may take time to decide.

5. **Privacy**: Websites cannot detect whether a passkey exists until the user
   actively selects one from the autofill menu.

> **GGID Priority**: **HIGH** — Conditional UI is the single highest-impact UX
> improvement for passkey adoption. The server-side `beginAuthentication` endpoint
> already supports empty `allowCredentials`. The console frontend needs the
> `autocomplete="username webauthn"` token and the conditional mediation JS.

---

## 3. Hybrid Transport (ctap2qr)

Hybrid transport (formerly called "caBLE" — Cloud-Assisted Bluetooth Low Energy)
enables cross-device passkey authentication. A user on a device without their passkey
can authenticate using a phone that has the passkey stored locally.

### How It Works

The hybrid transport involves two devices:

- **Client Device** (e.g., public computer, friend's laptop) — displays a QR code
- **Authenticator Device** (e.g., smartphone) — has the passkey, scans the QR code

```
┌──────────────┐         ┌──────────────────┐         ┌──────────┐
│  Client      │         │   Cloud Service   │         │ Phone    │
│  (Desktop)   │         │   (FIDO tunnel)   │         │(has key) │
│              │         │                   │         │          │
│ 1.Show QR ──┼────────►│                   │◄────────┼ 2.Scan   │
│              │         │ 3.Establish       │         │  QR code │
│              │         │   tunnel          │         │          │
│              │         │                   │         │ 4.BLE    │
│ 5.BLE check ◄┼─────────┼───────────────────┼────────►│  proximity│
│              │         │                   │         │          │
│              │◄────────┼───────────────────┼─────────┼ 6.Sign   │
│              │         │ 7.Encrypted data  │         │  challenge│
│ 8.Done!      │         │   over internet   │         │          │
└──────────────┘         └──────────────────┘         └──────────┘
```

### QR Code Flow (Step by Step)

1. **Initiate**: Client device shows a "Scan QR code" button or automatically
   displays a QR code when no local passkey is found.

2. **Generate QR Code**: The client generates a QR code encoding a unique,
   time-sensitive session identifier. The QR contains a CBOR-encoded handshake
   with ephemeral secrets. The payload uses a `FIDO:` URI scheme:
   ```
   FIDO:/078241338926040702789239694720083010994762289...
   ```

3. **Scan**: User scans the QR code with their phone (via camera app or browser).
   The QR is **one-time use** to prevent replay attacks.

4. **Tunnel Establishment**: The phone communicates with the authentication server
   through the cloud service to establish an encrypted internet tunnel between
   the two devices.

5. **BLE Proximity Check**: Bluetooth Low Energy is used **only for proximity
   verification**, not for data exchange. This prevents remote attackers from
   initiating authentication. All authentication data flows over the encrypted
   internet tunnel.

6. **Sign Challenge**: The phone signs the server's challenge using the passkey's
   private key. The private key **never leaves the phone**.

7. **Validate**: Server validates the signature using the stored public key.

### Critical Design Points

| Aspect | Detail |
|---|---|
| **BLE Role** | Proximity check ONLY. Not a data channel. Prevents remote MITM. |
| **Data Path** | All authentication data flows over encrypted internet tunnel. |
| **Private Keys** | Never leave the originating device. Only signatures are transmitted. |
| **QR Code** | One-time use, time-sensitive. Contains ephemeral handshake secrets. |
| **Security** | Combination of QR (session identity) + BLE (physical proximity) + asymmetric crypto (challenge-response). |

### Platform Support Matrix (as of 2024)

| OS / Browser | As Authenticator (Phone) | As Client (Desktop) |
|---|---|---|
| Android + Chrome | Yes (scan QR) | Yes (display QR) |
| iOS + Safari | Yes (scan QR) | Yes (display QR) |
| macOS + Chrome | No | Yes (display QR) |
| macOS + Safari | No | Yes (display QR) |
| Windows + Chrome | No | Yes (display QR) |
| Windows + Edge | No | Yes (display QR) |

### Known Issues

1. **Bluetooth Unavailable**: If the client device lacks Bluetooth, hybrid transport
   cannot work (dead-end with confusing error on some Windows versions).
2. **`transports: [internal]` Does Not Suppress QR**: Testing shows that setting
   `transports: [internal]` does NOT reliably prevent QR code display on Chrome/Edge
   on Windows or Safari on macOS/iOS. The browser still shows the QR option.
3. **First-Time Confusion**: Most non-technical users don't understand the flow.
   Education is critical.

> **GGID Priority**: **MEDIUM** — Hybrid transport works automatically when the
> RP supports WebAuthn. No server-side changes needed. GGID should ensure the
> `transports` array returned during registration is correctly persisted and
> forwarded in `allowCredentials`.

---

## 4. Device Registration Flows

### Discoverable Credentials vs Server-Side Credentials

| Aspect | Discoverable (Resident Key) | Server-Side (Non-Resident) |
|---|---|---|
| Stored on authenticator | Public key + user metadata | Public key only |
| Usernameless login | Supported | Not supported |
| Conditional UI | Supported | Not supported |
| Authenticator storage | Consumes device storage | Minimal |
| Scalability | Limited by device capacity | Unlimited |
| Passkey sync (iCloud/Google) | Supported | Not synced |

**Recommendation for 2024**: Always use `residentKey: "preferred"` (or `"required"`
for passwordless-first). Platform authenticators (Face ID, Touch ID, Windows Hello)
support discoverable credentials by default.

### Registration Best Practices

#### 1. Authenticator Selection

```go
// Recommended authenticator selection parameters
authenticatorSelection := protocol.AuthenticatorSelection{
    ResidentKey:       protocol.ResidentKeyRequirementPreferred,  // "preferred"
    UserVerification:  protocol.VerificationPreferred,            // "preferred"
    AuthenticatorAttachment: protocol.Platform,                   // "platform" or omitted
}
```

- Use `preferred` for `residentKey` — allows fallback for authenticators with
  limited storage.
- Use `preferred` for `userVerification` — accepts both UV and non-UV credentials.
  Use `required` only for high-security contexts (banking, admin).
- Omit `authenticatorAttachment` to allow both platform and cross-platform
  (security keys), or set `"platform"` for consumer passkey flows.

#### 2. Attestation Conveyance

| Preference | Privacy | When to Use |
|---|---|---|
| `none` | Highest — no attestation data | Default for consumer apps. Maximum privacy. |
| `indirect` | Medium — anonymized attestation | Rarely used in practice. |
| `direct` | Lower — full attestation | Enterprise, regulated industries. Enables device tracking. |
| `enterprise` | Lowest — unique device ID | Corporate device management only. |

```go
// Consumer app (recommended default)
attestation: protocol.PreferNoAttestation  // "none"

// Enterprise deployment
attestation: protocol.PreferDirectAttestation  // "direct"
```

**2024 Consensus**: Use `attestation: "none"` for consumer-facing applications.
Attestation adds complexity (FIDO Metadata Service, certificate chain validation)
and provides marginal security benefit for most use cases. The cryptographic
binding to the RP ID is the primary security mechanism, not attestation.

#### 3. Backup Eligibility & Backup State

New in WebAuthn Level 2, these flags indicate whether a credential can be synced
across devices (backup eligible) and whether it is currently backed up:

```go
// After registration, check backup flags
credential := parsedResponse.Response

if credential.AuthenticatorFlags.BackupEligible {
    // This credential is synced (e.g., via iCloud Keychain, Google Password Manager)
    // Good UX: user can recover across devices
}

if credential.AuthenticatorFlags.BackupState {
    // The credential is currently backed up to a cloud service
}
```

**Recommendation**: Store both `backupEligible` and `backupState` flags. Use them
for:
- UX guidance (show "synced" badge in credential management UI)
- Security policy (enterprise may require non-backup-eligible credentials)
- Account recovery planning (backup-eligible credentials survive device loss)

#### 4. Exclude Existing Credentials

Prevent duplicate registrations by passing `excludeCredentials` with the user's
existing credential IDs:

```go
// In BeginRegistration, exclude existing credentials
excludeCredentials := []protocol.CredentialDescriptor{}
for _, cred := range user.WebAuthnCredentials() {
    excludeCredentials = append(excludeCredentials, protocol.CredentialDescriptor{
        CredentialID:     cred.ID,
        Type:             protocol.PublicKeyCredentialType,
        Transport:        cred.Transport,
    })
}
```

#### 5. Full Registration Flow Example (Go)

```go
func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
    // ... tenant/user resolution ...

    user, _ := h.buildWebAuthnUser(ctx, tenantID, userID)

    options, sessData, err := h.wbn.BeginRegistration(
        user,
        // Recommended options for 2024 passkey registration
        webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
            ResidentKey:       protocol.ResidentKeyRequirementPreferred,
            UserVerification:  protocol.VerificationPreferred,
        }),
        // Use "none" attestation for consumer apps
        webauthn.WithAttestation(protocol.PreferNoAttestation),
        // Exclude existing credentials to prevent duplicates
        webauthn.WithExclusions(excludeCredentials),
        // Request credProps extension to know if credential is discoverable
        webauthn.WithExtensions(map[string]any{
            "credProps": true,
        }),
    )
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // Store session data
    // ...
}
```

> **GGID Priority**: **HIGH** — Update `beginRegistration` to use explicit
> `AuthenticatorSelection` with `residentKey: preferred`, add `excludeCredentials`,
> and persist `backupEligible`/`backupState` flags. Currently the handler calls
> `BeginRegistration(user)` with no options.

---

## 5. Relying Party (RP) Best Practices

### RP ID Configuration

The RP ID is a domain string that scopes credentials. It must be an eTLD+1 or higher.

```go
wconfig := &webauthn.Config{
    RPDisplayName: "GGID IAM",
    RPID:          "ggid.example.com",          // Must match your domain
    RPOrigins:     []string{                     // Authorized origins
        "https://ggid.example.com",
        "https://console.ggid.example.com",
    },
}
```

**Rules:**
- RP ID can be the current hostname or a parent domain (eTLD+1 or higher).
- Cannot use IP addresses or public suffixes (e.g., `github.io`).
- Using a broader RP ID (e.g., `example.com`) lets passkeys work across subdomains
  (`login.example.com`, `shop.example.com`).
- The RP ID in both `create()` and `get()` must be consistent.

#### Multi-Origin Setups (Related Origin Requests)

If your service spans multiple eTLD+1 domains (e.g., `example.com` and `example.co.jp`),
use Related Origin Requests (ROR):

1. Choose one primary RP ID (e.g., `example.com`).
2. Host a configuration file at `https://example.com/.well-known/webauthn`:

```json
{
  "origins": [
    "https://www.example.co.jp",
    "https://shop.example"
  ]
}
```

3. Always use the primary RP ID in all WebAuthn calls, regardless of which
   origin the user is on.

### Origin Validation

**Critical**: The RP MUST validate the `origin` field in `clientDataJSON`. Never
accept unexpected origins — this could allow credential theft via malicious sites.

```go
// go-webauthn handles origin validation automatically when RPOrigins is set correctly
wconfig := &webauthn.Config{
    RPOrigins: []string{
        "https://ggid.example.com",
        // Do NOT use wildcards or overly broad patterns
    },
}
```

**Common Mistakes:**
- Adding `http://` origins in production (WebAuthn requires HTTPS, except localhost).
- Forgetting to update origins when deploying to a new domain.
- Using `*` or regex patterns for origin matching (not supported by WebAuthn).

### Mobile App Integration

#### Android (Digital Asset Links)

Host `https://<RP_ID>/.well-known/assetlinks.json`:

```json
[
  {
    "relation": [
      "delegate_permission/common.get_login_creds",
      "delegate_permission/common.handle_all_urls"
    ],
    "target": {
      "namespace": "android_app",
      "package_name": "com.ggid.console",
      "sha256_cert_fingerprints": [
        "AB:CD:EF:..."
      ]
    }
  }
]
```

#### iOS (Associated Domains)

Host `https://<RP_ID>/.well-known/apple-app-site-association`:

```json
{
  "webcredentials": {
    "apps": ["TEAMID.com.ggid.console"]
  }
}
```

### Credential Management UX

#### Naming Credentials

Always let users name their passkeys with friendly names:

```go
// After registration, prompt for a friendly name
type RenameCredentialRequest struct {
    Name string `json:"name"` // e.g., "MacBook Pro (Work)"
}
```

Best practices for credential names:
- Auto-generate a default name based on the user agent (e.g., "Chrome on macOS")
- Let users rename via the credential management UI
- Display the name, creation date, and last-used date in the list

#### Deletion

- Always confirm before deletion
- If the credential being deleted is the last one, warn the user about lockout risk
- Provide alternative authentication method for users who delete their last passkey

#### Reauthentication Patterns

- **Step-up authentication**: Require passkey re-auth for sensitive operations
  (e.g., changing email, deleting account). Use `userVerification: "required"`.
- **Session timeout**: For long-lived sessions, require passkey re-auth after
  a configurable idle period.
- **Transaction confirmation**: Use WebAuthn for high-value transaction signing.

> **GGID Priority**: **HIGH** — The current `Credential` struct does not store
> `backupEligible` or `backupState`. RP Origins are hardcoded. Mobile app
> integration (DAL/AASA) files are not configured. Credential naming needs
> auto-generation.

---

## 6. Migration & Coexistence

### Phased Migration Strategy

The transition from passwords to passkeys should be gradual. Industry consensus
recommends a 4-phase approach:

#### Phase 1: Offer Passkeys Alongside Passwords (Dual-Stack)

```
Login Page:
  ┌─────────────────────────────────────┐
  │  [Passkey autofill in username field] │
  │                                       │
  │  Username: [________________]         │
  │  Password: [________________]         │
  │                                       │
  │  [Sign In]    [Sign in with Passkey]  │
  │                                       │
  │  Don't have an account? Sign up       │
  └─────────────────────────────────────┘
```

- Keep the existing password login fully functional
- Add a "Sign in with Passkey" button for users who have registered passkeys
- Implement Conditional UI for seamless autofill
- Do not remove or degrade password login

#### Phase 2: Encourage Passkey Enrollment (Post-Login Nudge)

After successful password login, prompt users to create a passkey:

```
"Create a passkey for faster, more secure sign-in?"
  [Create Passkey]    [Not Now]
```

Key principles:
- **Timing matters**: Show the prompt after successful login, not before
- **Frequency**: Show once per session, max 2-3 times before going dormant
- **Value proposition**: Explain benefits (faster login, phishing-resistant)
- **No pressure**: "Not Now" must be prominent and respected

#### Phase 3: Passwordless-First (Passkey Prominent)

```
Login Page:
  ┌─────────────────────────────────────┐
  │         [Passkey autofill here]       │
  │  Username: [________________]         │
  │                                       │
  │  [Sign In]                            │
  │                                       │
  │  ── or ──                             │
  │  [Use password instead]               │
  │  [Sign in with Passkey]               │
  └─────────────────────────────────────┘
```

- Passkeys become the primary authentication method
- Password login is available as a fallback ("Use password instead")
- Conditional UI is the default entry point
- New users are offered passkey enrollment during sign-up

#### Phase 4: Password Deprecation (Optional)

- Only for users who have multiple passkeys registered
- Requires robust account recovery mechanism
- May not be appropriate for all applications

### Account Recovery Without Passwords

Passwordless authentication needs a recovery strategy. Options:

| Method | Security | UX | Complexity |
|---|---|---|---|
| **Email magic link** | Medium | High | Low |
| **Phone OTP** | Medium | High | Medium |
| **Recovery codes** (generated at enrollment) | High | Medium | Medium |
| **Admin-assisted recovery** | High | Low | Low |
| **Second passkey on another device** | High | Medium | Medium |

**Recommendation**: Combine email magic link (convenience) with recovery codes
(security) for consumer apps. Enterprise apps should use admin-assisted recovery.

```go
// Recovery code generation
func generateRecoveryCodes(count int) []string {
    codes := make([]string, count)
    for i := 0; i < count; i++ {
        // Format: XXXX-XXXX-XXXX (alphanumeric, no ambiguous chars)
        codes[i] = generateCode(4) + "-" + generateCode(4) + "-" + generateCode(4)
    }
    return codes
}
```

### Coexistence Architecture for GGID

```
┌─────────────────────────────────────────────────────────┐
│                     Login Flow                           │
│                                                          │
│  1. User visits /login                                   │
│  2. Conditional UI fires (if passkey exists)             │
│     ├── Passkey selected → WebAuthn auth → JWT issued    │
│     └── No passkey / user types username → password form │
│  3. Password login → JWT issued                          │
│  4. Post-login: offer passkey enrollment (Phase 2)       │
│  5. User with passkeys: "Use passkey" button (Phase 3)   │
└─────────────────────────────────────────────────────────┘
```

> **GGID Priority**: **HIGH** — GGID currently supports WebAuthn registration
> and authentication but lacks Conditional UI on the console frontend and
> post-login enrollment nudges. The auth service already handles both password
> and WebAuthn paths.

---

## 7. Security Considerations

### Phishing Resistance

WebAuthn credentials are **cryptographically bound to the RP ID**. A passkey created
for `example.com` cannot be used on `evil-site.com`, even if the user is tricked
into visiting the malicious site.

```
Legitimate site (example.com):
  Authenticator signs challenge bound to rpId: "example.com"

Phishing site (evil-site.com):
  Requests authentication with rpId: "evil-site.com"
  → Authenticator has no credential for this rpId
  → Authentication fails
```

This is the **fundamental security advantage** of passkeys over passwords, OTPs,
and push notifications. Unlike passwords (which users can type anywhere), passkeys
cannot be phished because the browser enforces the RP ID binding.

### Replay Attack Prevention

Each WebAuthn ceremony includes a **server-generated challenge** (random nonce).
The authenticator signs this challenge along with other data. The signed challenge
is single-use:

1. Server generates random 32+ byte challenge
2. Authenticator signs `SHA-256(authenticatorData || clientDataHash)`
3. `clientDataJSON` contains the challenge, origin, and type
4. Server verifies the signature matches the challenge it generated
5. The challenge is consumed after use

A captured assertion cannot be replayed because:
- The challenge is unique per session
- The session is deleted after verification
- The signature includes a timestamp-bound origin

### Credential Stuffing Elimination

Passkeys eliminate credential stuffing because:
- No shared secret to steal from a breached database
- Public keys are not useful to attackers (they can't authenticate with them)
- Each RP has unique key pairs (no reuse across sites)

### Device Attestation Trade-offs

| Approach | Pros | Cons | Recommendation |
|---|---|---|---|
| **`none`** | Privacy-preserving, simple, fast | Cannot verify authenticator model | Default for consumer apps |
| **`direct`** | Device attestation, metadata checking | Privacy concern, requires FIDO MDS integration, breaks some flows | Enterprise/regulated only |
| **`enterprise`** | Full device management | Maximum privacy erosion, corporate-only authenticators | Corporate device management |

**2024 Consensus**: The cryptographic binding to RP ID is the primary security
mechanism. Attestation provides device provenance but is unnecessary for most
consumer applications. The `none` preference is the recommended default.

### Signature Count (Clone Detection)

```go
// Recommended: store and check sign count for clone detection
func (h *Handler) finishAuthentication(w http.ResponseWriter, r *http.Request) {
    // ... parse assertion ...

    credential, err := h.wbn.ValidateLogin(user, *sd.data, parsedResponse)
    if err != nil {
        writeError(w, http.StatusUnauthorized, err.Error())
        return
    }

    // Clone detection: if sign count > 0, it must increase
    if credential.Authenticator.SignCount > 0 {
        storedCred, _ := h.creds.GetCredentialByID(ctx, tenantID, credential.ID)
        if storedCred != nil && credential.Authenticator.SignCount <= storedCred.Counter {
            // POTENTIAL CREDENTIAL CLONE — log security event
            log.Security("webauthn clone detected: credential %s counter not increasing",
                credential.ID)
            // Option 1: Reject authentication
            // Option 2: Allow but alert (depends on security policy)
        }
    }

    // Update stored counter
    h.creds.UpdateCounter(ctx, tenantID, credential.ID, credential.Authenticator.SignCount)
}
```

> **GGID Priority**: **MEDIUM** — Signature count is stored but not checked.
> Clone detection should be added for production security.

### CVE-2024-9956 Awareness

In late 2024, CVE-2024-9956 disclosed a potential passkey account takeover
vulnerability in mobile browsers. The issue relates to cross-app credential
interception on mobile platforms. Mitigations:
- Ensure RP ID is correctly configured (not too broad)
- Validate origin strictly
- Keep browsers/OS updated
- Consider adding server-side risk scoring for unusual login patterns

---

## 8. Implementation Libraries

### Server-Side: Go

#### go-webauthn (Recommended)

**Repository**: https://github.com/go-webauthn/webauthn
**Import**: `github.com/go-webauthn/webauthn`
**Status**: FIDO2 Conformant, actively maintained, supports WebAuthn L3 features

**Features**:
- Full registration and authentication ceremony support
- All attestation formats (packed, tpm, android-key, android-safetynet, fido-u2f, none, apple, compound)
- Extensions: appid, appidExclude, credProps, largeBlob (manual)
- Credential Record support (backupEligible, backupState, userVerified flags)
- Metadata service provider integration for attestation verification
- FIDO2 conformant (tested against official conformance tools)

**Supported Go versions**: 1.24, 1.25, 1.26

```go
import (
    "github.com/go-webauthn/webauthn/protocol"
    "github.com/go-webauthn/webauthn/webauthn"
)

// Initialize
wconfig := &webauthn.Config{
    RPDisplayName: "GGID IAM",
    RPID:          "auth.ggid.dev",
    RPOrigins:     []string{"https://auth.ggid.dev"},
}

wbn, err := webauthn.New(wconfig)
if err != nil {
    log.Fatal(err)
}

// Registration
options, sessionData, err := wbn.BeginRegistration(user,
    webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
        ResidentKey:      protocol.ResidentKeyRequirementPreferred,
        UserVerification: protocol.VerificationPreferred,
    }),
    webauthn.WithAttestation(protocol.PreferNoAttestation),
)

// Finish registration
parsedResponse, err := protocol.ParseCredentialCreationResponseBody(r.Body)
credential, err := wbn.CreateCredential(user, *sessionData, parsedResponse)

// Authentication
options, sessionData, err := wbn.BeginLogin(user)

// Finish authentication
parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
credential, err := wbn.ValidateLogin(user, *sessionData, parsedResponse)
```

**User Interface**:

```go
// Your user model must implement webauthn.User
type MyUser struct {
    ID          uuid.UUID
    Username    string
    DisplayName string
    Credentials []webauthn.Credential
}

func (u *MyUser) WebAuthnID() []byte                    { return u.ID[:] }
func (u *MyUser) WebAuthnName() string                  { return u.Username }
func (u *MyUser) WebAuthnDisplayName() string           { return u.DisplayName }
func (u *MyUser) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
```

**Credential Record** (store ALL these fields):

```go
type webauthn.Credential struct {
    ID                []byte              // Credential ID
    PublicKey         []byte              // COSE-encoded public key
    AttestationType   string              // e.g., "none", "packed"
    Authenticator     Authenticator       // AAGUID, SignCount, Attachment, transports
    Flags             CredentialFlags     // UserVerified, BackupEligible, BackupState
    Attestation       Attestation         // Attestation object, clientDataJSON
    Transport         []protocol.AuthenticatorTransport
}
```

#### duo-labs/webauthn (Legacy)

**Repository**: https://github.com/duo-labs/webauthn
**Status**: Predecessor to go-webauthn, archived/unmaintained. Do NOT use for new projects.

### Client-Side: JavaScript/TypeScript

#### SimpleWebAuthn (Recommended)

**Repository**: https://github.com/MasterKale/SimpleWebAuthn
**NPM**: `@simplewebauthn/browser`, `@simplewebauthn/server`

**Features**:
- Browser-side helpers for `startRegistration()` and `startAuthentication()`
- Automatic base64 encoding/decoding
- TypeScript types for all WebAuthn objects
- Conditional UI support
- Cross-browser compatibility shims

```javascript
import { startRegistration, startAuthentication } from '@simplewebauthn/browser';

// Registration
const resp = await fetch('/api/v1/webauthn/register/begin', { method: 'POST' });
const options = await resp.json();
const attResp = await startRegistration({ optionsJSON: options });
const verifyResp = await fetch('/api/v1/webauthn/register/finish', {
  method: 'POST',
  body: JSON.stringify(attResp),
});

// Authentication with Conditional UI
const resp = await fetch('/api/v1/webauthn/auth/begin', { method: 'POST' });
const options = await resp.json();
const asseResp = await startAuthentication(
  { optionsJSON: options },
  true  // useBrowserAutofill = true (Conditional UI)
);
const verifyResp = await fetch('/api/v1/webauthn/auth/finish', {
  method: 'POST',
  body: JSON.stringify(asseResp),
});
```

#### Native (No Library)

WebAuthn is a native browser API. No library is strictly needed for the client side:

```javascript
// Registration (no library)
const credential = await navigator.credentials.create({
  publicKey: creationOptions
});

// Authentication (no library)
const credential = await navigator.credentials.get({
  publicKey: requestOptions,
  mediation: 'conditional'  // Conditional UI
});
```

Libraries add convenience (base64 handling, TypeScript types, error normalization)
but the raw API is usable directly.

### Other Server-Side Libraries

| Language | Library | Notes |
|---|---|---|
| Node.js | `@simplewebauthn/server`, `@fido2-lib` | Most popular, well-documented |
| Python | `fido2` (Yubico), `django-fido` | Official Yubico library |
| Java | `java-webauthn-server` (Yubico) | Enterprise-grade |
| Rust | `webauthn-rs` | Growing ecosystem |
| Ruby | `webauthn-ruby` | Rails-friendly |

---

## 9. GGID Implementation Roadmap

Based on the analysis of GGID's current WebAuthn implementation
(`services/auth/internal/webauthn/handler.go`), here are prioritized improvements:

### Priority: CRITICAL

| # | Task | Current State | Target State |
|---|---|---|---|
| 1 | **Add `backupEligible` and `backupState` to Credential struct** | Not stored | Store both flags in `webauthn_credentials` table |
| 2 | **Enforce signature count monotonicity** | Counter stored but not checked | Check `receivedCount > storedCount` when count > 0 |
| 3 | **Add `excludeCredentials` to registration** | Not passed | Exclude user's existing credential IDs |
| 4 | **Add `AuthenticatorSelection` to registration** | Default (no explicit params) | `residentKey: preferred`, `userVerification: preferred` |

### Priority: HIGH

| # | Task | Current State | Target State |
|---|---|---|---|
| 5 | **Conditional UI on console frontend** | Not implemented | `autocomplete="username webauthn"` + conditional mediation JS |
| 6 | **Post-login passkey enrollment nudge** | Not implemented | Prompt users after password login to create passkey |
| 7 | **Credential auto-naming** | Name from query param only | Auto-generate from User-Agent (e.g., "Chrome on macOS") |
| 8 | **Persist transports from registration** | Stores `Attachment` only | Store actual transports array (`internal`, `hybrid`, `usb`, `nfc`) |
| 9 | **Make RP Origins configurable** | Hardcoded `"https://" + rpID` | Configurable list from environment/config |
| 10 | **Redis-backed session store** | In-memory `map[string]*sessionData` | Redis with TTL for multi-instance deployments |

### Priority: MEDIUM

| # | Task | Current State | Target State |
|---|---|---|---|
| 11 | **Related Origin Requests (ROR)** | Not configured | Host `.well-known/webauthn` for multi-domain support |
| 12 | **Mobile app integration (DAL/AASA)** | Not configured | Host assetlinks.json and apple-app-site-association |
| 13 | **Account recovery (recovery codes)** | Not implemented | Generate and store recovery codes during passkey enrollment |
| 14 | **FIDO Metadata Service integration** | Not configured | Optional: verify authenticator attestation against FIDO MDS |
| 15 | **Credential transport forwarding** | Not forwarded in `allowCredentials` | Include transports from stored credentials in auth options |
| 16 | **Security event logging** | Not logged | Log clone detection, registration, authentication events to audit |

### Priority: LOW

| # | Task | Current State | Target State |
|---|---|---|---|
| 17 | **PRF extension** | Not supported | Derive symmetric keys from passkeys for E2E encryption |
| 18 | **LargeBlob extension** | Not supported | Store opaque data with credentials |
| 19 | **Enterprise attestation** | Not supported | Support `attestation: "enterprise"` for corporate deployments |
| 20 | **WebAuthn error classification** | Generic error messages | Classify errors (AbortError, NotAllowedError, SecurityError) for UX |

### Current GGID WebAuthn Gap Analysis

```go
// CURRENT (simplified):
func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
    user, _ := h.buildWebAuthnUser(ctx, tenantID, userID)
    options, sessData, err := h.wbn.BeginRegistration(user)  // No options!
    // ...
}

// RECOMMENDED:
func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
    user, _ := h.buildWebAuthnUser(ctx, tenantID, userID)

    // Build excludeCredentials from existing credentials
    existingCreds := user.WebAuthnCredentials()
    excludeCreds := make([]protocol.CredentialDescriptor, len(existingCreds))
    for i, c := range existingCreds {
        excludeCreds[i] = protocol.CredentialDescriptor{
            CredentialID: c.ID,
            Type:         protocol.PublicKeyCredentialType,
            Transport:    c.Transport,
        }
    }

    options, sessData, err := h.wbn.BeginRegistration(
        user,
        webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
            ResidentKey:      protocol.ResidentKeyRequirementPreferred,
            UserVerification: protocol.VerificationPreferred,
        }),
        webauthn.WithAttestation(protocol.PreferNoAttestation),
        webauthn.WithExclusions(excludeCreds),
    )
    // ...
}
```

```go
// CURRENT credential persistence (simplified):
cred := &Credential{
    CredentialID: credential.ID,
    PublicKey:    credential.PublicKey,
    Transports:   []string{string(credential.Authenticator.Attachment)}, // Wrong! Stores attachment, not transports
    Counter:      credential.Authenticator.SignCount,
}

// RECOMMENDED:
transports := make([]string, len(credential.Transport))
for i, t := range credential.Transport {
    transports[i] = string(t)
}

cred := &Credential{
    CredentialID:    credential.ID,
    PublicKey:       credential.PublicKey,
    Transports:      transports,               // Actual transports: "internal", "hybrid", "usb", "nfc"
    Counter:         credential.Authenticator.SignCount,
    BackupEligible:  credential.Flags.BackupEligible,   // NEW
    BackupState:     credential.Flags.BackupState,       // NEW
    UserVerified:    credential.Flags.UserVerified,       // NEW
    AttestationType: credential.AttestationType,          // NEW
}
```

### Database Migration

```sql
-- Add new columns to webauthn_credentials table
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS backup_state BOOLEAN DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS user_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS attestation_type TEXT DEFAULT 'none';
ALTER TABLE webauthn_credentials ADD COLUMN IF NOT EXISTS aaguid BYTEA;
```

---

## Appendix: References

### Specifications
- [WebAuthn Level 2 (W3C Recommendation)](https://www.w3.org/TR/webauthn-2/)
- [WebAuthn Level 3 (Working Draft)](https://www.w3.org/TR/webauthn-3/)
- [CTAP 2.1 Specification](https://fidoalliance.org/specs/fido-v2.1-ps-20210615/fido-client-to-authenticator-protocol-v2.1-ps-20210615.html)
- [FIDO2 / Passkeys (FIDO Alliance)](https://fidoalliance.org/passkeys/)

### Implementation Guides
- [Chrome: WebAuthn Conditional UI](https://developer.chrome.com/docs/identity/webauthn-conditional-ui)
- [web.dev: RP ID Deep Dive](https://web.dev/articles/webauthn-rp-id)
- [Yubico: WebAuthn Level 2 Features](https://developers.yubico.com/WebAuthn/Concepts/WebAuthn_Level_2_Features_and_Enhancements.html)
- [Corbado: Conditional UI Technical Explanation](https://www.corbado.com/blog/webauthn-conditional-ui-passkeys-autofill)
- [Corbado: Hybrid Transport](https://www.corbado.com/blog/webauthn-passkey-qr-code)
- [passkeys.dev](https://passkeys.dev/)

### Libraries
- [go-webauthn (Go)](https://github.com/go-webauthn/webauthn)
- [SimpleWebAuthn (JS/TS)](https://github.com/MasterKale/SimpleWebAuthn)
- [Yubico java-webauthn-server (Java)](https://github.com/webauthn4j/webauthn4j)
- [webauthn-rs (Rust)](https://github.com/kanidm/webauthn-rs)

### Security
- [CVE-2024-9956: Passkey Account Takeover in Mobile Browsers](https://nvd.nist.gov/vuln/detail/CVE-2024-9956)
- [FIDO Alliance: Passkey Security White Paper](https://fidoalliance.org/white-paper-multi-device-fido-credentials/)
- [Phishing-Resistant MFA Guide](https://www.loginradius.com/blog/identity/how-phishing-resistant-authentication-works)
