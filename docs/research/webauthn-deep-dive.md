# WebAuthn / FIDO2 Deep Dive

## Overview

This document provides a technical deep dive into WebAuthn/FIDO2 implementation details: attestation formats, device public key extension, conditional UI (autofill), hybrid transport, and the full registration/authentication lifecycle.

> **Related**: [WebAuthn Device-Bound](webauthn-device-bound.md), [WebAuthn Attestation Verification](webauthn-attestation-verification.md), [WebAuthn Attestation Chain](webauthn-attestation-chain.md), [Passwordless Setup](../guides/passwordless-setup.md)

## WebAuthn Registration Flow (Detailed)

### Phase 1: Ceremony Initiation

```
1. Relying Party (GGID Auth) generates:
   - challenge: 32 random bytes (crypto/rand)
   - rp.id: "ggid.example.com"
   - user.id: SHA-256(user_uuid)
   - user.name: "alice@example.com"
   - user.displayName: "Alice Chen"
   - pubKeyCredParams: [{alg: -7 (ES256)}, {alg: -257 (RS256)}]
   - authenticatorSelection:
     - authenticatorAttachment: "platform" | "cross-platform"
     - userVerification: "required" | "preferred" | "discouraged"
     - residentKey: "required" | "preferred" | "discouraged"
   - attestation: "none" | "indirect" | "direct"
   - excludeCredentials: [existing credential IDs]
```

### Phase 2: Browser API Call

```javascript
// navigator.credentials.create()
const credential = await navigator.credentials.create({
    publicKey: {
        challenge: Uint8Array.from(atob(challenge), c => c.charCodeAt(0)),
        rp: { id: "ggid.example.com", name: "GGID" },
        user: {
            id: Uint8Array.from(atob(userId), c => c.charCodeAt(0)),
            name: "alice@example.com",
            displayName: "Alice Chen"
        },
        pubKeyCredParams: [
            { type: "public-key", alg: -7 },   // ES256
            { type: "public-key", alg: -257 }   // RS256
        ],
        authenticatorSelection: {
            userVerification: "required",
            residentKey: "preferred"
        },
        attestation: "direct",
        excludeCredentials: existingCredentials
    }
});
```

### Phase 3: Authenticator Operation

The authenticator (hardware key or platform biometric):

1. Checks user presence (touch button / biometric)
2. Checks user verification (biometric match / PIN)
3. Generates a new key pair (non-exportable)
4. Creates attestation object:
   - `authData`: RP ID hash + flags + signCount + AAGUID + credential ID + public key (COSE)
   - `fmt`: attestation format identifier
   - `attStmt`: format-specific attestation statement
5. Returns `clientDataJSON`:
   - `type`: "webauthn.create"
   - `challenge`: base64url(challenge)
   - `origin`: "https://ggid.example.com"
   - `crossOrigin`: false

### Phase 4: Server Verification (GGID)

```go
func VerifyRegistration(parsed *ParsedRegistration) error {
    // 1. Verify challenge matches
    if parsed.Challenge != storedChallenge { return ErrChallengeMismatch }

    // 2. Verify origin
    if parsed.Origin != "https://ggid.example.com" { return ErrOriginMismatch }

    // 3. Verify clientDataJSON type
    if parsed.Type != "webauthn.create" { return ErrWrongType }

    // 4. Verify RP ID hash
    rpHash := sha256([]byte("ggid.example.com"))
    if !bytes.Equal(parsed.AuthData.RPIDHash, rpHash) { return ErrRPIDMismatch }

    // 5. Verify flags
    if !parsed.AuthData.Flags.UP { return ErrUserPresence }
    if !parsed.AuthData.Flags.UV && requiredUV { return ErrUserVerification }

    // 6. Verify attestation format
    err := VerifyAttestationFormat(
        parsed.Fmt,
        parsed.AuthData.Bytes,
        parsed.ClientDataHash,
        parsed.Alg,
        parsed.Sig,
        parsed.CertBytes,
    )
    return err
}
```

## Attestation Formats (All 7)

GGID verifies all 7 standard attestation formats in `services/auth/internal/webauthn/attestation.go`:

### Format: `none`

Self-attestation — no attestation chain. The public key is trusted directly.

```
authStmt = {} (empty)
```

**Use case**: Consumer applications (privacy-first).

### Format: `packed`

Compact attestation with signature from the authenticator's attestation key.

```
authStmt = {
    alg: -7 (ES256) or -257 (RS256),
    sig: <signature over authenticatorData + clientDataHash>,
    x5c: [<attestation cert chain>]  // present if not self-attested
}
```

**Verification**:
1. If `x5c` present: verify signature against leaf certificate's public key
2. If no `x5c` (self-attestation): verify signature against credential public key
3. Validate certificate chain to FIDO MDS root

### Format: `fido-u2f`

Legacy U2F format (used by older YubiKeys).

```
authStmt = {
    x5c: [<attestation cert>],
    sig: <signature over 0x00 + rpIdHash + clientDataHash + credentialId + publicKey>
}
```

**Note**: The signature includes an extra `0x00` byte prefix and uses the U2F-specific signing format.

### Format: `android-key`

Android Keystore attestation.

```
authStmt = {
    alg: -7,
    sig: <signature>,
    x5c: [<cert chain including Android Keystore attestation cert>]
}
```

**Verification**: Check for key attestation extension OID `1.3.6.1.4.1.11129.2.1.17`.

### Format: `android-safetynet`

Legacy Android SafetyNet response (deprecated, use Play Integrity API).

```
authStmt = {
    ver: "1.0",
    response: <JWS response from SafetyNet API>
}
```

**Verification**: Parse JWS, verify certificate chain, check `ctsProfileMatch` and `basicIntegrity`.

### Format: `tpm`

TPM 2.0 platform attestation (Windows Hello, enterprise laptops).

```
authStmt = {
    ver: "2.0",
    alg: -257 (RS256),
    sig: <TPM signature>,
    x5c: [<TPM attestation cert chain>],
    pubArea: <TPM2B_PUBLIC>,
    certInfo: <TPMS_ATTEST>
}
```

**Verification**: Verify TPM-specific structures (TPM2B_PUBLIC, TPMS_ATTEST, TPMT_SIGNATURE).

### Format: `apple`

Apple anonymized attestation (iOS, macOS).

```
authStmt = {
    alg: -7,
    sig: <signature>,
    x5c: [<Apple anonymized attestation cert>]
}
```

**Verification**: Verify nonce computation = SHA-256(authenticatorData + clientDataHash). Check that certificate contains Apple's OID extension.

## Device Public Key Extension (PRF)

The `devicePubKey` extension (W3C WebAuthn Level 3) allows the authenticator to expose a device-bound public key for encryption:

```
Extensions:
  devicePubKey:
    outputs:
      - pubKey: <COSE_Key>
        aaguid: <authenticator AAGUID>
```

**Use case**: End-to-end encryption where the server encrypts data to the device's public key.

**GGID status**: Not yet implemented (roadmap).

## Conditional UI (Autofill)

Conditional UI enables WebAuthn credentials to appear in browser autofill dropdowns, providing a seamless passwordless experience:

```javascript
// Before: User must click "Sign in with passkey" button
// After: Credential options appear in username field autofill

if (PublicKeyCredential.isConditionalMediationAvailable()) {
    const credential = await navigator.credentials.get({
        publicKey: {
            challenge: challenge,
            mediation: "conditional"  // Autofill UI
        }
    });
    // User selects credential from autofill dropdown
    // Browser prompts for biometric/PIN
}
```

**Browser support**:
- Chrome 121+ (desktop + Android)
- Safari 16+ (macOS + iOS)
- Edge 121+

**GGID Console**: Roadmap integration — passkeys will appear in the login form autofill.

## Hybrid Transport (Cross-Device)

Hybrid transport enables a mobile device to serve as an authenticator for a desktop browser:

```
Desktop Browser              Mobile Phone
     │                            │
     │  1. Display QR code        │
     │  (with Bluetooth pairing)  │
     │                            │
     │                     2. Scan QR code
     │                     3. BLE proximity check
     │                     4. Biometric verification
     │                            │
     │←──── BLE + cloud relay ────│
     │  5. Credential assertion   │
     │  (via hybrid transport)    │
```

**Protocol**: caBLE / BLE-LE + HTTPS relay (FIDO Cross-Device Authentication)

**GGID support**: Works automatically — hybrid is a client-side transport, transparent to the server.

## Sign Counter (Replay Detection)

Each authenticator maintains a monotonically increasing `signCount`:

```
Registration → signCount = 42
Auth #1     → signCount = 43
Auth #2     → signCount = 44
```

**Verification**: If `signCount` in the assertion is <= stored value, the credential may have been cloned.

```go
func (v *Verifier) checkSignCount(stored, received uint32) error {
    if received > 0 && received <= stored {
        return ErrSignCountRegressed // Possible cloned credential
    }
    return nil
}
```

**Limitation**: Some authenticators (software passkeys, iCloud Keychain) always return `signCount = 0`.

## COSE Algorithms

| COSE Alg ID | Name | Key Type | Use |
|-------------|------|----------|-----|
| -7 | ES256 | ECDSA P-256 | Default WebAuthn |
| -257 | RS256 | RSA 2048 | Hardware keys |
| -8 | EdDSA | Ed25519 | Modern keys |
| -35 | ES384 | ECDSA P-384 | High security |
| -36 | ES512 | ECDSA P-521 | Very high security |

## AAGUID (Authenticator Attestation GUID)

The AAGUID uniquely identifies an authenticator model:

| AAGUID | Device |
|--------|--------|
| `00000000-0000-0000-0000-000000000000` | Platform (synced passkey) |
| `cb69481e-8ff7-4039-93ec-0a2729a154a8` | YubiKey 5 (USB) |
| `08987058-cadc-4b81-b6e1-30de50dcbe96` | Windows Hello |
| `ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4` | Google Pixel |

GGID can enforce an AAGUID allowlist for enterprise deployments to restrict which authenticators are accepted.

## Multi-Device Credential Sync

When a credential is synced (iCloud Keychain, Google Password Manager):

| Property | Value |
|----------|-------|
| AAGUID | `00000000-...` (zero GUID) |
| Attestation | `"none"` |
| Backup eligible | `true` |
| Backup state | `true` (currently synced) |
| signCount | Always `0` |

This means synced credentials CANNOT be distinguished by hardware model. Enterprise policies that require hardware authenticators must check `backupEligible = false`.

## GGID WebAuthn Configuration

```yaml
webauthn:
  rp_id: "ggid.example.com"           # Must match browser origin domain
  rp_name: "GGID"
  origin: "https://ggid.example.com"  # Exact origin match
  attestation_conveyance: "direct"    # none | indirect | direct
  user_verification: "required"       # required | preferred | discouraged
  resident_key: "preferred"           # required | preferred | discouraged
  timeout: 60000                      # 60 seconds
  aaguid_allowlist: []                # Empty = accept all
```

## See Also

- [WebAuthn Device-Bound Credentials](webauthn-device-bound.md)
- [WebAuthn Attestation Verification](webauthn-attestation-verification.md)
- [FIDO2 Passkey Ecosystem 2026](fido2-passkey-ecosystem-2026.md)
- [Passwordless Setup Guide](../guides/passwordless-setup.md)
- [WebAuthn Roadmap v2](webauthn-roadmap-v2.md)
