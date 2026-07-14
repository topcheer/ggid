# WebAuthn Attestation Verification

This guide covers the complete verification flow for all 5 attestation formats, trust anchor chains, FIDO Metadata Service (MDS), AAGUID lookup, security levels, attestation bypass policies, and GGID's implementation.

## Overview

Attestation proves that an authenticator is genuine and not a counterfeit. During registration, the authenticator signs a challenge with its attestation private key, and the server verifies this signature against a trust anchor chain rooted at the authenticator manufacturer.

## 5 Attestation Formats

### 1. Packed

Most common format. Simple CBOR structure with signature and optional certificate chain.

```json
{
  "fmt": "packed",
  "attStmt": {
    "alg": -7,
    "sig": "<ECDSA or RSA signature>",
    "x5c": ["<leaf cert>", "<intermediate cert>", "..."],
    "ecdaaKeyId": null
}
```

**Verification Steps**:
1. Verify signature `sig` over authenticator data + client data hash
2. Verify certificate chain from `x5c` to root CA
3. Check leaf certificate's OID 1.3.6.1.4.1.45724.1.1.4 (aaguid) matches authenticator
4. Verify certificate is not expired
5. Check certificate against FIDO MDS revocation list

```go
func verifyPacked(attStmt map[string]interface{}, authData []byte, clientDataHash []byte) error {
    sig := attStmt["sig"].([]byte)
    alg := attStmt["alg"].(int)
    x5c := attStmt["x5c"].([]interface{})

    if len(x5c) == 0 {
        return ErrMissingAttestationCert
    }

    // Parse leaf certificate
    certBytes := x5c[0].([]byte)
    cert, err := x509.ParseCertificate(certBytes)
    if err != nil {
        return fmt.Errorf("parse attestation cert: %w", err)
    }

    // Verify certificate chain
    if err := verifyCertChain(x5c, trustedRoots); err != nil {
        return fmt.Errorf("cert chain verification failed: %w", err)
    }

    // Verify signature
    signedData := append(authData, clientDataHash...)
    pubKey := cert.PublicKey
    if err := verifySignature(alg, pubKey, signedData, sig); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    // Check AAGUID in cert extension
    aaguid := extractAAGUIDFromCert(cert)
    if !isKnownAAGUID(aaguid) {
        return ErrUnknownAuthenticator
    }

    return nil
}
```

### 2. TPM

Used by Windows Hello. Contains TPM-generated attestation with TPM certificate chain.

```json
{
  "fmt": "tpm",
  "attStmt": {
    "ver": "2.0",
    "alg": -7,
    "sig": "<signature>",
    "x5c": ["<TPM cert>", "..."],
    "info": "<TPS_TPM_INFO CBOR>",
    "pubArea": "<TPM public area>"
  }
}
```

**Verification Steps**:
1. Parse TPM public area (algorithm, parameters, unique)
2. Verify TPM certificate chain to manufacturer root
3. Check that AIK (Attestation Identity Key) cert is genuine
4. Verify signature over `authData + clientDataHash` using AIK public key
5. Validate TPM firmware version against known vulnerabilities
6. Check `info` field for TPM compliance

```go
func verifyTPM(attStmt map[string]interface{}, authData, clientDataHash []byte) error {
    ver := attStmt["ver"].(string)
    if ver != "2.0" {
        return ErrUnsupportedTPMVersion
    }

    // Parse pubArea
    pubArea := parseTPMPubArea(attStmt["pubArea"].([]byte))

    // Verify cert chain (TPM manufacturer roots)
    x5c := attStmt["x5c"].([]interface{})
    if err := verifyTPMCertChain(x5c); err != nil {
        return err
    }

    // Verify signature with AIK key
    sig := attStmt["sig"].([]byte)
    cert := parseCert(x5c[0])
    signedData := constructTPMSignedData(authData, clientDataHash, attStmt["info"])

    return verifySignature(-7, cert.PublicKey, signedData, sig)
}
```

### 3. Android Key

Used by Android devices. Attestation from Android Keystore.

```json
{
  "fmt": "android-key",
  "attStmt": {
    "alg": -7,
    "sig": "<signature>",
    "x5c": ["<Keystore attestation cert>", "..."]
  }
}
```

**Verification Steps**:
1. Verify certificate chain to Google Hardware Attestation root
2. Check `keyDescription` extension (OID 1.3.6.1.4.1.11129.2.1.17)
3. Verify `attestationChallenge` matches server challenge
4. Check `attestationSecurityLevel` (TEE vs StrongBox)
5. Verify device is Google-certified
6. Check for rooted/unlocked indicators

```go
func verifyAndroidKey(attStmt map[string]interface{}, challenge []byte, authData, clientDataHash []byte) error {
    x5c := attStmt["x5c"].([]interface{})
    cert := parseCert(x5c[0])

    // Verify chain to Google root
    if err := verifyCertChain(x5c, googleRoots); err != nil {
        return err
    }

    // Extract keyDescription extension
    keyDesc := extractAndroidKeyDescription(cert)

    // Verify challenge matches
    if !bytes.Equal(keyDesc.AttestationChallenge, challenge) {
        return ErrChallengeMismatch
    }

    // Check security level
    if keyDesc.AttestationSecurityLevel < SecurityLevelTEE {
        return ErrInsufficientSecurityLevel
    }

    // Verify signature
    sig := attStmt["sig"].([]byte)
    return verifySignature(-7, cert.PublicKey, append(authData, clientDataHash...), sig)
}
```

### 4. Apple

Used by iOS/macOS passkeys in iCloud Keychain.

```json
{
  "fmt": "apple",
  "attStmt": {
    "x5c": ["<Apple attestation cert>", "..."]
  }
}
```

**Verification Steps**:
1. Verify certificate chain to Apple Root CA
2. Check `nonce` extension in leaf certificate
3. Verify `nonce = SHA-256(authData + clientDataHash)`
4. Verify AAGUID matches Apple authenticator

```go
func verifyApple(attStmt map[string]interface{}, authData, clientDataHash []byte) error {
    x5c := attStmt["x5c"].([]interface{})
    cert := parseCert(x5c[0])

    // Verify chain to Apple root
    if err := verifyCertChain(x5c, appleRoots); err != nil {
        return err
    }

    // Verify nonce extension
    nonce := extractAppleNonce(cert)
    expected := sha256.Sum256(append(authData, clientDataHash...))
    if !bytes.Equal(nonce, expected[:]) {
        return ErrNonceMismatch
    }

    return nil
}
```

### 5. None

No attestation. Privacy-preserving — authenticator provenance is not verified.

```json
{
  "fmt": "none",
  "attStmt": {}
}
```

**Verification Steps**:
1. No verification needed
2. Accept credential with no attestation trust
3. Mark credential with `attestation_trust = "none"`

```go
func verifyNone() error {
    // No attestation to verify
    // Credential is accepted but has no provenance guarantee
    return nil
}
```

## Trust Anchor Chain

### Chain Structure

```
Manufacturer Root CA (self-signed, in trust store)
    └── Intermediate CA
        └── Authenticator Attestation Cert (leaf)
            └── Contains AAGUID extension
```

### Trust Store Management

```yaml
webauthn:
  trust_anchors:
    fido_mds:
      enabled: true
      url: "https://mds.fidoalliance.org/"
      refresh_interval: 24h
    custom_roots:
      - path: "/etc/ggid/webauthn/roots/yubico.pem"
      - path: "/etc/ggid/webauthn/roots/google.pem"
      - path: "/etc/ggid/webauthn/roots/apple.pem"
      - path: "/etc/ggid/webauthn/roots/ms-tpm.pem"
```

## FIDO Metadata Service (MDS)

### What is MDS?

FIDO Alliance maintains a Metadata Service (MDS) with information about all certified authenticators:
- AAGUID → authenticator model, manufacturer
- Security level (AAID/AAID certification)
- Certification status (FIDO Certified Level 1/2/Biometric)
- Root certificates for attestation chain verification
- Revocation status

### MDS Update Flow

```go
func updateMDS() error {
    // Download MDS blob (JWT signed by FIDO Alliance)
    blob, err := downloadMDSBlob("https://mds.fidoalliance.org/")
    if err != nil {
        return err
    }

    // Verify JWT signature with FIDO Alliance root
    metadata, err := verifyAndParseMDS(blob, fidoRootCert)
    if err != nil {
        return err
    }

    // Update trust store
    for _, entry := range metadata.Entries {
        aaguid := entry.AAGUID
        roots := entry.TrustChain
        status := entry.StatusReports

        if isRevoked(status) {
            removeTrustAnchor(aaguid)
        } else {
            updateTrustAnchor(aaguid, roots)
        }
    }

    return nil
}
```

## AAGUID Lookup

### What is AAGUID?

AAGUID (Authenticator Attestation GUID) is a 128-bit identifier unique to each authenticator model. It's embedded in:
- The authenticator data during registration
- The attestation certificate extension (OID 1.3.6.1.4.1.45724.1.1.4)

### Lookup Process

```go
func lookupAAGUID(aaguid string) (*AuthenticatorInfo, error) {
    // Check FIDO MDS
    info, ok := mdsCache.Get(aaguid)
    if !ok {
        return nil, ErrUnknownAuthenticator
    }

    // Check revocation status
    for _, report := range info.StatusReports {
        if report.Status == "REVOKED" {
            return nil, ErrRevokedAuthenticator
        }
        if report.Status == "FIDO_CERT_L1" {
            info.CertLevel = 1
        }
        if report.Status == "FIDO_CERT_L2" {
            info.CertLevel = 2
        }
    }

    return info, nil
}
```

### AAGUID Examples

| AAGUID | Authenticator | Manufacturer |
|---|---|---|
| 00000000-0000-0000-0000-000000000000 | Platform authenticator | Various |
| 00000000-0000-0000-0000-000000000001 | Android | Google |
| dd4ec289-e25e-5e20-bc6c-5e8d5e5e5e5e | YubiKey 5 | Yubico |
| ea9b8d60-3a1f-5e20-bc6c-5e8d5e5e5e5e | Windows Hello | Microsoft |

## Security Level

| Level | Name | Requirement | Trust |
|---|---|---|---|
| None | No attestation | `fmt: none` | Low |
| Self | Self-attestation | No cert chain | Medium |
| Basic | Basic attestation | Cert chain to manufacturer | High |
| Attestation CA | Attestation CA | Full chain + MDS verified | Very High |

## Attestation Bypass Policy

### Configuration

```yaml
webauthn:
  attestation_policy:
    required: false              # Allow "none" attestation
    required_formats: ["packed", "tpm"]  # If required, only accept these
    trust_threshold: "basic"     # Minimum trust level
    bypass_for_personal_devices: true   # Allow "none" for BYOD

    per_tenant:
      high_security:
        required: true
        required_formats: ["packed", "tpm", "android-key"]
        trust_threshold: "attestation_ca"
      enterprise:
        required: true
        required_formats: ["packed", "tpm"]
        trust_threshold: "basic"
      consumer:
        required: false
        trust_threshold: "none"
```

### Bypass Logic

```go
func shouldVerifyAttestation(tenantConfig *WebAuthnConfig, deviceType string) bool {
    // High-security tenants always require attestation
    if tenantConfig.Required {
        return true
    }

    // Allow bypass for personal devices if configured
    if deviceType == "byod" && tenantConfig.BypassForPersonalDevices {
        return false
    }

    // Default: verify if attestation present
    return true
}
```

## GGID Implementation

### Verification Pipeline

```go
func (s *WebAuthnService) VerifyAttestation(
    attObj *AttestationObject,
    authData []byte,
    clientDataHash []byte,
    challenge []byte,
    config *WebAuthnConfig,
) (*AttestationResult, error) {

    result := &AttestationResult{
        Format: attObj.Fmt,
    }

    switch attObj.Fmt {
    case "packed":
        if err := verifyPacked(attObj.AttStmt, authData, clientDataHash); err != nil {
            return nil, fmt.Errorf("packed verification: %w", err)
        }
        result.TrustLevel = "basic"

    case "tpm":
        if err := verifyTPM(attObj.AttStmt, authData, clientDataHash); err != nil {
            return nil, fmt.Errorf("tpm verification: %w", err)
        }
        result.TrustLevel = "basic"

    case "android-key":
        if err := verifyAndroidKey(attObj.AttStmt, challenge, authData, clientDataHash); err != nil {
            return nil, fmt.Errorf("android-key verification: %w", err)
        }
        result.TrustLevel = "basic"

    case "apple":
        if err := verifyApple(attObj.AttStmt, authData, clientDataHash); err != nil {
            return nil, fmt.Errorf("apple verification: %w", err)
        }
        result.TrustLevel = "basic"

    case "none":
        if config.Required {
            return nil, ErrAttestationRequired
        }
        if err := verifyNone(); err != nil {
            return nil, err
        }
        result.TrustLevel = "none"

    default:
        return nil, fmt.Errorf("unsupported attestation format: %s", attObj.Fmt)
    }

    // AAGUID lookup (skip for "none")
    if attObj.Fmt != "none" {
        aaguid := extractAAGUID(authData)
        info, err := lookupAAGUID(aaguid)
        if err != nil {
            if config.Required {
                return nil, fmt.Errorf("AAGUID lookup: %w", err)
            }
            // Log warning but continue
            result.Warning = "unknown authenticator"
        } else {
            result.AuthenticatorInfo = info
            result.CertLevel = info.CertLevel
        }
    }

    return result, nil
}
```

## Best Practices

1. **Verify full certificate chain** — Don't skip intermediate certificates
2. **Update FIDO MDS regularly** — New authenticators and revocations
3. **Check revocation status** — Don't trust revoked authenticators
4. **Set per-tenant policies** — High-security tenants need stricter requirements
5. **Allow "none" for consumer** — Privacy-conscious users may not want attestation
6. **Log attestation details** — AAGUID, format, trust level for audit
7. **Validate challenge match** — Prevent replay attacks
8. **Check certificate expiry** — Expired attestation certs should be rejected
9. **Handle format-specific requirements** — Each format has unique verification steps
10. **Monitor for new formats** — FIDO Alliance may add new attestation formats