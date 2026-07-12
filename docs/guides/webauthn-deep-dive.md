# WebAuthn Deep Dive

This guide covers the FIDO2/WebAuthn implementation in GGID, including attestation formats, COSE algorithms, trust anchors, and advanced features.

## FIDO2 Architecture

WebAuthn is part of the FIDO2 specification, combining:

- **W3C WebAuthn** — Browser API for credential creation and assertion
- **CTAP2** — Client-to-Authenticator Protocol over USB/NFC/BLE

### Components

```
┌─────────────┐    ┌──────────────┐    ┌──────────────────┐
│   Browser   │───▶│  Authenticator │───▶│  GGID Server     │
│ (WebAuthn   │    │ (Passkey/      │    │ (Relying Party)  │
│  API)       │    │  Security Key) │    │                  │
└─────────────┘    └──────────────────┘    └──────────────────┘
     Client           Authenticator            Server (RP)
```

| Component | Role |
|---|---|
| Client | Browser/OS running WebAuthn API |
| Authenticator | Hardware or software that stores credentials and signs challenges |
| Server (RP) | GGID — verifies assertions, stores public keys |

## Attestation Formats

GGID supports all standard attestation formats:

### Packed

Most common format. Packs signature and certificate into a simple CBOR structure.

```json
{
  "fmt": "packed",
  "attStmt": {
    "alg": -7,
    "sig": "<ECDSA signature>",
    "x5c": ["<attestation certificate>"]
  }
}
```

### TPM

Used by Windows Hello. Contains TPM-generated attestation with TPM manufacturer certificate chain.

- Validates TPM is genuine hardware
- Checks TPM firmware version against known vulnerabilities
- Verifies AIK (Attestation Identity Key) certificate

### Android Key

Used by Android devices. Attestation from Android Keystore.

- Verifies device is Google-certified
- Checks attestation certificate chain to Google root
- Extracts device security level

### Apple

Used by iOS/macOS passkeys in iCloud Keychain.

```json
{
  "fmt": "apple",
  "attStmt": {
    "x5c": ["<Apple attestation cert>"]
  }
}
```

- Validates certificate chain to Apple root CA
- Checks `nonce` extension matches SHA-256 of authenticator data

### None

No attestation. Used when privacy is prioritized over authenticator provenance.

```json
{
  "fmt": "none",
  "attStmt": {}
}
```

GGID accepts `none` by default. Configure trust policy to require specific formats:

```yaml
webauthn:
  required_attestation: ["packed", "tpm"]
  trust_anchor_path: /etc/ggid/webauthn/trust-anchors/
```

## COSE Algorithm Registry

GGID supports the following COSE algorithms for credential public keys:

| COSE ID | Algorithm | Key Type | Notes |
|---|---|---|---|
| -7 | ES256 (ECDSA w/ SHA-256) | EC P-256 | Recommended |
| -8 | EdDSA (Ed25519) | OKP | Fast, modern |
| -35 | ES384 (ECDSA w/ SHA-384) | EC P-384 | High security |
| -36 | ES512 (ECDSA w/ SHA-512) | EC P-521 | Very high security |
| -257 | RS256 (RSASSA-PKCS1-v1.5 w/ SHA-256) | RSA 2048 | Widely supported |
| -258 | RS384 | RSA 3072 | |
| -259 | RS512 | RSA 4096 | |

```go
var supportedAlgs = []cose.Algorithm{
    cose.AlgES256,
    cose.AlgEdDSA,
    cose.AlgRS256,
}
```

## RPID + Origin Validation

### Relying Party ID (RPID)

The RPID must be a registrable domain suffix of the origin:

```
Origin:  https://auth.ggid.example.com
RPID:    ggid.example.com        ✓
RPID:    auth.ggid.example.com   ✓
RPID:    example.com             ✓
RPID:    other.com               ✗
```

```go
config := &webauthn.Config{
    RPID:          "ggid.example.com",
    RPDisplayName: "GGID Identity Platform",
    RPOrigins:     []string{"https://auth.ggid.example.com"},
}
```

### Origin Validation

GGID verifies that the `origin` in the client data JSON matches the configured allowed origins. Mismatched origins are rejected with `ERR_ORIGIN_MISMATCH`.

## Counter-Based Clone Detection

Each authenticator maintains a monotonically increasing signature counter. GGID tracks the counter:

```go
func verifyCounter(stored uint32, received uint32) error {
    if received > 0 && received <= stored {
        return ErrClonedAuthenticator  // Possible clone detected
    }
    return nil
}
```

- Counter = 0: Authenticator doesn't support clone detection (software keys)
- Counter > 0: Each assertion must have a higher counter than the previous

**Action on clone detection**: Revoke the credential immediately and notify the user.

## Token Binding

WebAuthn supports token binding to bind the credential to a specific TLS session:

```json
{
  "type": "webauthn.get",
  "challenge": "...",
  "origin": "https://auth.ggid.example.com",
  "tokenBinding": {
    "status": "supported",
    "id": "<TBID>"
  }
}
```

GGID verifies token binding ID matches the TLS session when present.

## Extensions

### appid

Allows FIDO U2F credentials to work with WebAuthn:

```json
{
  "extensions": {
    "appid": "https://u2f.ggid.example.com:443"
  }
}
```

### credProtect (credProtect)

Tiered protection for resident keys:

| Level | Name | Behavior |
|---|---|---|
| 0 | userVerificationOptional | Default |
| 1 | userVerificationRequired | UV required for use |
| 2 | userVerificationRequiredWithRogistics | UV required, credential removed on ROG |

### largeBlob

Stores large data associated with a credential (up to 2KB):

```json
{
  "extensions": {
    "largeBlob": {
      "support": "required"
    }
  }
}
```

### hmac-secret

Generates a secret tied to the credential for use as a derived key.

## UV (User Verification) Flags

The authenticator data contains UV and UP flags:

| Flag | Meaning |
|---|---|
| UP (User Present) | User touched the authenticator |
| UV (User Verified) | User verified (biometric, PIN) |

### UV Enforcement

```yaml
webauthn:
  user_verification: "required"  # or "preferred" or "discouraged"
```

- **required**: Only UV-verified credentials accepted
- **preferred**: UV preferred but not required
- **discouraged**: UV not needed (low friction)

## Backup State Flags

FIDO2 backup eligible/backup state flags indicate if the credential can be synced across devices:

| Flag | Meaning |
|---|---|
| BE (Backup Eligible) | Credential supports backup (multi-device passkeys) |
| BS (Backup State) | Credential is currently backed up |

```go
type AuthenticatorFlags struct {
    UserPresent     bool
    UserVerified    bool
    BackupEligible  bool
    BackupState     bool
}
```

**Policy consideration**: Some enterprises may want to disable backup-eligible credentials for high-security environments.

## Registration Flow

```
1. Server generates challenge → sends to client
2. Client calls navigator.credentials.create({challenge, ...})
3. Authenticator creates key pair, signs challenge with attestation key
4. Client sends attestation object + client data JSON to server
5. Server verifies attestation, stores public key
```

## Authentication Flow

```
1. Server generates challenge → sends to client
2. Client calls navigator.credentials.get({challenge, ...})
3. Authenticator signs challenge with credential private key
4. Client sends assertion + client data JSON to server
5. Server verifies signature against stored public key
```

## Security Considerations

- Always validate RPID and origin on every request
- Enforce UV for high-security operations (MFA escalation, admin actions)
- Rotate trust anchors periodically
- Monitor for clone detection events
- Rate-limit registration attempts to prevent authenticator exhaustion
- Store credential counters atomically to prevent race conditions