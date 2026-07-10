# WebAuthn Implementation Guide

Complete guide for implementing passkey and security-key authentication using
WebAuthn (Web Authentication) / FIDO2 in GGID. Covers registration and
authentication ceremonies, attestation formats, credential management, recovery
codes, and Relying Party (RP) configuration.

---

## Table of Contents

- [Overview](#overview)
- [WebAuthn Architecture](#webauthn-architecture)
- [Relying Party Configuration](#relying-party-configuration)
- [Registration Ceremony](#registration-ceremony)
- [Authentication Ceremony](#authentication-ceremony)
- [Attestation Formats](#attestation-formats)
- [Credential Management](#credential-management)
- [Recovery Codes](#recovery-codes)
- [Multi-Device Support](#multi-device-support)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

---

## Overview

WebAuthn is a W3C standard for passwordless authentication using public-key
cryptography. GGID acts as a **Relying Party (RP)** — users register
authenticators (platform authenticators like Touch ID / Windows Hello, or
roaming authenticators like YubiKeys) and authenticate by proving possession of
the private key.

### Key Benefits

| Feature | Password | OTP / TOTP | WebAuthn |
|---------|----------|------------|----------|
| Phishing resistant | No | No | **Yes** |
| Replay resistant | No | No | **Yes** |
| Server breach safe | No | Partial | **Yes** |
| User friction | High | Medium | **Low** |

### Standards

- **W3C WebAuthn Level 2** (April 2021): core browser API
- **FIDO2 CTAP 2.1**: client-to-authenticator protocol
- **RFC 8809**: Registry for WebAuthn extension identifiers

---

## WebAuthn Architecture

```
Browser                    GGID Server              Authenticator
  │                            │                         │
  │  1. Start registration     │                         │
  ├───────────────────────────►│                         │
  │  2. PublicKeyCredential    │                         │
  │     CreationOptions (challenge, RP info)             │
  │◄───────────────────────────┤                         │
  │  3. navigator.credentials.create()                   │
  │     User touches authenticator                       │
  ├──────────────────────────────────────────────────────►│
  │  4. Attestation object + client data                 │
  │◄─────────────────────────────────────────────────────┤
  │  5. Send attestation      │                         │
  ├───────────────────────────►│                         │
  │  6. Verify attestation,   │                         │
  │     store credential      │                         │
  │  7. Success               │                         │
  │◄───────────────────────────┤                         │
```

### Components

| Component | Role |
|-----------|------|
| **Authenticator** | Hardware or software that generates and stores key pairs (Touch ID, Windows Hello, YubiKey, Titan) |
| **Client (Browser)** | Mediates between RP and authenticator via `navigator.credentials` API |
| **Relying Party (GGID)** | Verifies authenticator responses, stores public keys |
| **Authenticator Model** | Stored metadata: AAGUID, attestation format, transport hints |

---

## Relying Party Configuration

### RP ID

The RP ID is the domain that authenticators bind credentials to. A credential
registered for `iam.example.com` can only be used on `iam.example.com` (and
subdomains if configured).

```yaml
webauthn:
  rp_id: "iam.example.com"
  rp_name: "GGID Identity Platform"
  rp_origins:
    - "https://iam.example.com"
    - "https://console.example.com"
  timeout: 60000          # milliseconds
  user_verification: "preferred"  # required | preferred | discouraged
  attestation: "none"     # none | indirect | direct
```

### Configuration via Environment Variables

```bash
# .env
WEBAUTHN_RP_ID=iam.example.com
WEBAUTHN_RP_NAME="GGID Identity Platform"
WEBAUTHN_RP_ORIGINS=https://iam.example.com,https://console.example.com
WEBAUTHN_TIMEOUT=60000
WEBAUTHN_USER_VERIFICATION=preferred
WEBAUTHN_ATTESTATION=none
```

### RP ID Rules

| Rule | Example |
|------|---------|
| Must be a valid domain | `iam.example.com` |
| Credentials scoped to exact domain | RP ID `example.com` works on `app.example.com` |
| Cannot use IP addresses | `192.168.1.10` is invalid |
| Cannot use localhost in production | Use `localhost` only for development |
| Must match the origin's registrable domain | Origin `https://app.example.com` → RP ID can be `example.com` |

### Multi-Origin Support

```yaml
webauthn:
  rp_id: "example.com"
  rp_origins:
    - "https://iam.example.com"
    - "https://app.example.com"
    - "https://admin.example.com"
```

All origins must share the RP ID as their registrable domain suffix.

---

## Registration Ceremony

### Step 1: Server Generates Options

GGID generates a `PublicKeyCredentialCreationOptions` challenge:

```json
{
  "challenge": "base64url-encoded-random-bytes",
  "rp": {
    "id": "iam.example.com",
    "name": "GGID Identity Platform"
  },
  "user": {
    "id": "base64url-encoded-user-id",
    "name": "user@example.com",
    "displayName": "Jane Doe"
  },
  "pubKeyCredParams": [
    { "type": "public-key", "alg": -7 },
    { "type": "public-key", "alg": -257 }
  ],
  "timeout": 60000,
  "attestation": "none",
  "authenticatorSelection": {
    "authenticatorAttachment": "platform",
    "userVerification": "preferred",
    "residentKey": "preferred",
    "requireResidentKey": false
  },
  "excludeCredentials": [
    {
      "type": "public-key",
      "id": "base64url-existing-credential-id"
    }
  ]
}
```

### API Endpoint

```
POST /api/v1/webauthn/register/begin
Authorization: Bearer <jwt>

Response: PublicKeyCredentialCreationOptions
```

### Step 2: Browser Creates Credential

```javascript
// Frontend code
const options = await fetch('/api/v1/webauthn/register/begin', {
  method: 'POST',
  headers: { Authorization: `Bearer ${token}` }
}).then(r => r.json());

// Decode base64url fields
const publicKey = {
  ...options,
  challenge: base64urlToBuffer(options.challenge),
  user: {
    ...options.user,
    id: base64urlToBuffer(options.user.id)
  },
  excludeCredentials: options.excludeCredentials?.map(c => ({
    ...c,
    id: base64urlToBuffer(c.id)
  }))
};

const credential = await navigator.credentials.create({ publicKey });
```

### Step 3: Server Verifies Attestation

```
POST /api/v1/webauthn/register/finish
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "id": "credential-id-base64url",
  "rawId": "credential-id-base64url",
  "type": "public-key",
  "response": {
    "attestationObject": "...",
    "clientDataJSON": "..."
  }
}
```

### Verification Steps

GGID performs these checks on the attestation:

1. **Parse clientDataJSON**: verify `type === "webauthn.create"`
2. **Verify origin**: must match a configured RP origin
3. **Verify challenge**: must match the challenge sent in Step 1
4. **Parse attestationObject**: extract authenticator data, fmt, attStmt
5. **Verify RP ID hash**: `SHA256(rp_id) === authData.rpIdHash`
6. **Verify flags**: `UP` (user present) must be set; check `UV` (user verified)
7. **Verify attestation signature** (if `fmt !== "none"`)
8. **Store credential**: credential ID, public key, counter, AAGUID

### Credential Storage Schema

```sql
CREATE TABLE webauthn_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    user_id         UUID NOT NULL REFERENCES users(id),
    credential_id   BYTEA NOT NULL,         -- unique credential identifier
    public_key      BYTEA NOT NULL,          -- COSE-encoded public key
    attestation_format VARCHAR(32) NOT NULL,  -- none, packed, tpm, etc.
    aaguid          UUID,                    -- authenticator model ID
    sign_count      BIGINT NOT NULL DEFAULT 0,
    transports      TEXT[],                  -- usb, nfc, ble, internal
    device_type     VARCHAR(32),             -- platform, roaming
    backed_up       BOOLEAN DEFAULT FALSE,
    name            VARCHAR(128),            -- user-assigned label
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ,
    UNIQUE(tenant_id, credential_id)
);
```

---

## Authentication Ceremony

### Step 1: Server Generates Assertion Challenge

```
POST /api/v1/webauthn/authenticate/begin
Content-Type: application/json

{ "username": "user@example.com" }
```

```json
{
  "challenge": "base64url-challenge",
  "rpId": "iam.example.com",
  "timeout": 60000,
  "userVerification": "preferred",
  "allowCredentials": [
    {
      "type": "public-key",
      "id": "base64url-credential-id",
      "transports": ["internal", "hybrid"]
    }
  ]
}
```

> **Note**: If `allowCredentials` is empty, the browser shows all passkeys
> for the RP ID (discoverable credentials / resident keys).

### Step 2: Browser Gets Assertion

```javascript
const options = await fetch('/api/v1/webauthn/authenticate/begin', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username })
}).then(r => r.json());

const publicKey = {
  ...options,
  challenge: base64urlToBuffer(options.challenge),
  allowCredentials: options.allowCredentials?.map(c => ({
    ...c,
    id: base64urlToBuffer(c.id)
  }))
};

const assertion = await navigator.credentials.get({ publicKey });
```

### Step 3: Server Verifies Assertion

```
POST /api/v1/webauthn/authenticate/finish
Content-Type: application/json

{
  "id": "credential-id-base64url",
  "rawId": "credential-id-base64url",
  "type": "public-key",
  "response": {
    "authenticatorData": "...",
    "clientDataJSON": "...",
    "signature": "...",
    "userHandle": "..."
  }
}
```

### Verification Steps

1. **Parse clientDataJSON**: verify `type === "webauthn.get"`
2. **Verify origin and challenge**: same as registration
3. **Parse authenticatorData**: extract RP ID hash, flags, counter
4. **Verify RP ID hash**: matches configured RP ID
5. **Verify user presence flag** (`UP`)
6. **Verify signature**: `signature` over `authData || SHA256(clientDataJSON)` using stored public key
7. **Check counter**: `counter > stored_counter` (replay protection)
8. **Update counter and last_used_at**
9. **Issue JWT** session token

### Counter-Based Replay Protection

```go
if assertion.Response.SignCount != 0 {
    if assertion.Response.SignCount <= credential.SignCount {
        return errors.New("possible cloned authenticator: sign count did not increase")
    }
    credential.SignCount = assertion.Response.SignCount
}
```

> Some authenticators always report 0. GGID skips the counter check when both
> the stored and received counters are 0.

---

## Attestation Formats

GGID supports the following attestation formats:

| Format | Description | Attestation Trust |
|--------|-------------|-------------------|
| `none` | No attestation data | None — recommended for most deployments |
| `packed` | Compact Packed format | Self or attestation CA |
| `fido-u2f` | Legacy U2F format | FIDO MDS lookup |
| `tpm` | Trusted Platform Module | Hardware attestation |
| `android-key` | Android Keymaster | Google Play Integrity |
| `android-safetynet` | Android SafetyNet | Google JWT |
| `apple` | Apple App Attest | Apple CA chain |
| `none` (hybrid) | Platform authenticators | None |

### Attestation Conveyance Preference

```yaml
webauthn:
  attestation: "none"  # Default — privacy-preserving
```

| Value | Use Case |
|-------|----------|
| `none` | Privacy-first, consumer applications. No device info sent. |
| `indirect` | Anonymized attestation CA. Moderate device trust. |
| `direct` | Full attestation. Device identifiable. Enterprise/government. |

### COSE Algorithms

GGID advertises these algorithms in `pubKeyCredParams`:

| alg | Name | Curve / Type |
|-----|------|--------------|
| -7 | ES256 | ECDSA w/ SHA-256, P-256 |
| -35 | ES384 | ECDSA w/ SHA-384, P-384 |
| -36 | ES512 | ECDSA w/ SHA-512, P-521 |
| -257 | RS256 | RSA PKCS#1 v1.5 w/ SHA-256 |
| -258 | RS384 | RSA PKCS#1 v1.5 w/ SHA-384 |
| -259 | RS512 | RSA PKCS#1 v1.5 w/ SHA-512 |
| -37 | PS256 | RSA-PSS w/ SHA-256 |
| -8 | EdDSA | Ed25519 |

---

## Credential Management

### List User Credentials

```
GET /api/v1/webauthn/credentials
Authorization: Bearer <jwt>
```

```json
{
  "credentials": [
    {
      "id": "cred-uuid",
      "name": "MacBook Touch ID",
      "device_type": "platform",
      "aaguid": "adce0001-35bc-c60a-648b-0b25f1f05503",
      "transports": ["internal"],
      "created_at": "2024-01-15T10:30:00Z",
      "last_used_at": "2024-06-20T14:22:00Z",
      "sign_count": 142
    }
  ]
}
```

### Rename a Credential

```
PATCH /api/v1/webauthn/credentials/{id}
Authorization: Bearer <jwt>
Content-Type: application/json

{ "name": "Personal YubiKey 5C" }
```

### Delete a Credential

```
DELETE /api/v1/webauthn/credentials/{id}
Authorization: Bearer <jwt>
```

### Deletion Guard

GGID prevents removing the last credential if it is the user's only
authentication factor:

```json
{
  "error": "cannot_remove_last_factor",
  "message": "This is your only authentication factor. Add another method before removing it."
}
```

---

## Recovery Codes

When a user registers a WebAuthn credential, GGID generates recovery codes as a
fallback for lost or broken authenticators.

### Generation

```
POST /api/v1/webauthn/recovery-codes/regenerate
Authorization: Bearer <jwt>
```

```json
{
  "recovery_codes": [
    "7HXK-Q2M5-9RJT",
    "BPWL-3NF8-4KVD",
    "CZRA-7YQ2-6MTS",
    "DSXE-1LU5-8NFG",
    "EVGY-4PK2-9HWJ",
    "FTBH-6QRM-2XLA",
    "GUIC-8SDN-5OKE",
    "HOJF-3TPL-7QVR"
  ]
}
```

### Recovery Code Properties

| Property | Value |
|----------|-------|
| Format | `XXXX-XXXX-XXXX` (12 alphanumeric chars) |
| Count | 8 codes per user |
| Storage | SHA-256 hashed (never plaintext) |
| Single use | Each code consumed on first use |
| Entropy | ~62 bits (36^12 with checksum) |

### Using a Recovery Code

```
POST /api/v1/auth/recover
Content-Type: application/json

{
  "username": "user@example.com",
  "recovery_code": "7HXK-Q2M5-9RJT"
}
```

On success, GGID:
1. Marks the recovery code as used
2. Issues a temporary session (10-minute TTL)
3. Prompts the user to register a new WebAuthn credential
4. Generates a new set of recovery codes

---

## Multi-Device Support

### Registering Multiple Authenticators

Users can register any number of authenticators. Common patterns:

| Pattern | Authenticators |
|---------|----------------|
| Primary + backup | Touch ID (platform) + YubiKey (roaming) |
| Cross-platform | Touch ID + Windows Hello |
| Shared workstation | Roaming key used across machines |

### Synchronizable Passkeys

Platform authenticators (Apple iCloud Keychain, Google Password Manager) can sync
passkeys across devices. GGID detects this via the `BS` (backup eligible) and
`BU` (backup state) flags:

```
flags: BE=1, BS=1  →  passkey is synced across devices
```

### Hybrid Transport

Apple and Google support a **hybrid** transport where a phone authenticates via
QR code + Bluetooth proximity to a desktop:

```json
{
  "transports": ["hybrid", "internal"]
}
```

---

## Security Considerations

### User Verification

| Level | Flag | Behavior |
|-------|------|----------|
| Required | `UV=1` | Authentication fails without biometric/PIN |
| Preferred | `UV=1` or `UV=0` | Attempts UV, falls back to presence only |
| Discouraged | `UV=0` | Presence only, lowest friction |

### Phishing Resistance

WebAuthn credentials are cryptographically bound to the RP ID. A credential
registered for `iam.example.com` **cannot** be used on `iam.evil.com`, even if
the user is tricked into visiting the malicious site. This eliminates phishing
as an attack vector for WebAuthn-authenticated sessions.

### Clone Detection

The sign counter mechanism detects cloned authenticators:

```
If SignCount(new_assertion) <= SignCount(stored):
    → CLONE DETECTED
    → Revoke credential
    → Alert user and security team
```

### Rate Limiting

| Endpoint | Rate Limit |
|----------|------------|
| `/webauthn/register/begin` | 10/min per user |
| `/webauthn/register/finish` | 10/min per user |
| `/webauthn/authenticate/begin` | 20/min per IP |
| `/webauthn/authenticate/finish` | 20/min per IP |

### Session Binding

After successful WebAuthn authentication, GGID binds the JWT to the credential:

```json
{
  "amr": ["hwk", "user"],
  "cnf": {
    "jkt": "thumbprint-of-credential-public-key"
  }
}
```

---

## Troubleshooting

### "SecurityError: The operation is insecure"

The origin does not match the RP ID or is not in the allowed list.

**Fix**: Ensure `WEBAUTHN_RP_ORIGINS` includes the exact origin (scheme + host +
port) the browser is using.

### "NotAllowedError: The operation either timed out or is not permitted"

User cancelled, or the authenticator does not support the requested parameters.

**Fix**: Check `userVerification` setting. Try `preferred` instead of `required`.

### "InvalidStateError: The authenticator was previously registered"

The credential already exists (duplicate registration attempt).

**Fix**: The `excludeCredentials` list should prevent this. Ensure it is
populated from existing user credentials.

### Counter Desync After Firmware Update

Some authenticators reset their counter after firmware updates.

**Fix**: Admin endpoint to reset a credential's stored counter:

```
POST /api/v1/admin/webauthn/credentials/{id}/reset-counter
```

### Cross-Origin iframe Issues

WebAuthn inside an iframe requires the `publickey-credentials-create` and
`publickey-credentials-get` Permissions Policy:

```html
<iframe
  allow="publickey-credentials-create; publickey-credentials-get"
  src="https://auth.example.com">
</iframe>
```

### Safari quirks

Safari requires HTTPS with a valid certificate (no self-signed in production).
For local development, use `localhost` which Safari treats as a secure context.
