# WebAuthn Server Implementation

Ceremony flows, challenge generation, attestation verification, assertion verification, counter tracking, and credential storage.

## Overview

WebAuthn (FIDO2) enables passwordless authentication using public-key cryptography. GGID implements the server-side (Relying Party) logic for registration and authentication ceremonies.

## Registration Ceremony

### Step 1: Generate Challenge

```bash
POST /api/v1/auth/webauthn/register/begin
{
  "user_id": "uuid",
  "device_name": "iPhone 15",
  "authenticator_attachment": "platform"  // or "cross-platform"
}
# → 200
# {
#   "challenge": "base64url-challenge",
#   "rp": {
#     "name": "GGID",
#     "id": "ggid.dev"
#   },
#   "user": {
#     "id": "base64url-user-id",
#     "name": "jane@corp.com",
#     "displayName": "Jane Doe"
#   },
#   "pubKeyCredParams": [
#     {"type":"public-key","alg":-7},   // ES256
#     {"type":"public-key","alg":-257}  // RS256
#   ],
#   "timeout": 60000,
#   "attestation": "none",  // or "direct" for high-security
#   "excludeCredentials": [...],  // Prevent duplicate registration
#   "authenticatorSelection": {
#     "authenticatorAttachment": "platform",
#     "userVerification": "required",
#     "residentKey": "preferred"
#   }
# }
```

### Step 2: Client Creates Credential

```javascript
// Browser calls WebAuthn API
const credential = await navigator.credentials.create({ publicKey: options });
// credential contains: id, rawId, response.{attestationObject, clientDataJSON}
```

### Step 3: Verify Registration Response

```bash
POST /api/v1/auth/webauthn/register/complete
{
  "id": "base64url-credential-id",
  "rawId": "base64url-raw-id",
  "response": {
    "attestationObject": "base64url-attestation",
    "clientDataJSON": "base64url-clientdata"
  },
  "type": "public-key"
}
# → 201 {"credential_id": "uuid", "device_name": "iPhone 15"}
```

### Server-Side Verification

```go
func VerifyRegistration(response WebAuthnRegistrationResponse, expectedChallenge string) (*Credential, error) {
    // 1. Parse clientDataJSON
    clientData := parseClientData(response.ClientDataJSON)

    // 2. Verify clientData
    if clientData.Type != "webauthn.create" {
        return nil, ErrWrongCeremonyType
    }
    if clientData.Challenge != expectedChallenge {
        return nil, ErrChallengeMismatch
    }
    if clientData.Origin != expectedOrigin {
        return nil, ErrOriginMismatch
    }

    // 3. Parse attestation object
    attObj := parseAttestation(response.AttestationObject)

    // 4. Verify authenticator data
    authData := attObj.AuthData
    rpIDHash := authData[:32]
    if !bytes.Equal(rpIDHash, sha256(expectedRPID)) {
        return nil, ErrRPIDMismatch
    }
    if authData.Flags&USER_PRESENT == 0 {
        return nil, ErrUserNotPresent
    }
    if authData.Flags&USER_VERIFIED == 0 && requireUV {
        return nil, ErrUserNotVerified
    }

    // 5. Extract public key
    pubKey := extractPublicKey(authData)

    // 6. Verify attestation (if requested)
    if attObj.Fmt != "none" {
        if err := verifyAttestation(attObj, clientData); err != nil {
            return nil, err
        }
    }

    // 7. Store credential
    return &Credential{
        ID:        response.ID,
        PublicKey: pubKey,
        Counter:   authData.Counter,
        Transport: response.Transports,
    }, nil
}
```

## Authentication Ceremony

### Step 1: Generate Challenge

```bash
POST /api/v1/auth/webauthn/authenticate/begin
{
  "user_id": "uuid",
  "user_verification": "required"
}
# → 200
# {
#   "challenge": "base64url-challenge",
#   "rpId": "ggid.dev",
#   "allowCredentials": [
#     {"type":"public-key","id":"base64url-cred-id","transports":["internal"]}
#   ],
#   "timeout": 60000,
#   "userVerification": "required"
# }
```

### Conditional UI (Discoverable Credentials)

```bash
POST /api/v1/auth/webauthn/authenticate/begin
{
  "user_verification": "required"
  // No allowCredentials → discoverable credentials
}
```

### Step 2: Client Gets Assertion

```javascript
const assertion = await navigator.credentials.get({ publicKey: options });
// assertion contains: id, rawId, response.{authenticatorData, clientDataJSON, signature, userHandle}
```

### Step 3: Verify Assertion

```bash
POST /api/v1/auth/webauthn/authenticate/complete
{
  "id": "base64url-cred-id",
  "rawId": "base64url-raw-id",
  "response": {
    "authenticatorData": "base64url-authdata",
    "clientDataJSON": "base64url-clientdata",
    "signature": "base64url-signature",
    "userHandle": "base64url-user-handle"
  },
  "type": "public-key"
}
# → 200 {"access_token":"...","refresh_token":"..."}
```

### Server-Side Verification

```go
func VerifyAssertion(response WebAuthnAssertionResponse, expectedChallenge string, cred *Credential) error {
    // 1. Verify clientData
    clientData := parseClientData(response.ClientDataJSON)
    if clientData.Type != "webauthn.get" { return ErrWrongCeremonyType }
    if clientData.Challenge != expectedChallenge { return ErrChallengeMismatch }
    if clientData.Origin != expectedOrigin { return ErrOriginMismatch }

    // 2. Parse authenticator data
    authData := parseAuthData(response.AuthenticatorData)
    if !bytes.Equal(authData.RPIDHash, sha256(expectedRPID)) { return ErrRPIDMismatch }
    if authData.Flags&USER_PRESENT == 0 { return ErrUserNotPresent }

    // 3. Verify signature
    signedData := append(response.AuthenticatorData, sha256(response.ClientDataJSON)...)
    if !verifySignature(cred.PublicKey, signedData, response.Signature) {
        return ErrInvalidSignature
    }

    // 4. Check counter (clone detection)
    if authData.Counter <= cred.Counter && cred.Counter > 0 {
        return ErrClonedAuthenticator  // Possible credential clone!
    }

    // 5. Update counter
    cred.Counter = authData.Counter

    return nil
}
```

## Counter Tracking

Each authentication returns a monotonically increasing counter. If a subsequent assertion has a lower counter, the authenticator may have been cloned.

```go
func checkCounter(stored, received uint32) error {
    if received == 0 {
        return nil  // Some authenticators don't support counters
    }
    if received <= stored {
        log.Warn("possible credential clone detected")
        return ErrClonedAuthenticator
    }
    return nil
}
```

## Credential Storage

```sql
CREATE TABLE webauthn_credentials (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    credential_id BYTEA NOT NULL UNIQUE,  -- Base64url decoded
    public_key BYTEA NOT NULL,            -- CBOR encoded
    counter INTEGER NOT NULL DEFAULT 0,
    transports TEXT[],                    -- ['internal', 'hybrid', 'usb', 'nfc']
    device_name TEXT,
    attestation_format TEXT,              -- 'none', 'packed', 'fido-u2f', 'tpm'
    aaguid TEXT,                          -- Authenticator model identifier
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    INDEX idx_webauthn_user (user_id),
    INDEX idx_webauthn_cred_id (credential_id)
);
```

## Attestation Formats

| Format | Trust Model | Verification |
|--------|------------|-------------|
| `none` | No attestation | Skip verification (most common) |
| `packed` | Self or CA attested | Verify signature chain |
| `fido-u2f` | U2F device | Verify X.509 certificate |
| `tpm` | TPM hardware | Verify TPM attestation |
| `android-key` | Android keystore | Verify Android key attestation |
| `apple` | Apple device | Verify Apple anonymized attestation |

GGID supports all formats but defaults to `"none"` for privacy. High-security deployments can require `"direct"` attestation.

## Security Considerations

| Threat | Mitigation |
|--------|-----------|
| Replay attack | Challenge is single-use, nonce-based |
| MITM | Origin binding in clientDataJSON |
| Phishing | RP ID (domain) bound — credential only works on real domain |
| Credential clone | Counter tracking detects cloned authenticators |
| Brute force | Not possible — private key never leaves authenticator |

## Monitoring

| Metric | Alert |
|--------|-------|
| Registration failure rate | >10% |
| Authentication failure rate | >5% |
| Clone detection (counter) | Any → security alert |
| Credential never used (>90 days) | Flag for cleanup |
| Attestation verification failures | Spike → check trusted CAs |

## See Also

- [Passwordless Auth Architecture](passwordless-auth-architecture.md)
- [WebAuthn Recovery](webauthn-recovery.md)
- [MFA Architecture](mfa-architecture.md)
- [Multi-Factor Step-Up Design](multi-factor-step-up-design.md)
