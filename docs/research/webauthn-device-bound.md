# WebAuthn Device-Bound Credentials Research

## Overview

This document analyzes the landscape of WebAuthn credential types — synced passkeys vs device-bound credentials — and maps them to GGID's WebAuthn attestation verification implementation.

## Credential Types

### 1. Synced Passkeys (Multi-Device)

Synced passkeys are WebAuthn credentials that synchronize across devices via platform ecosystems:

| Platform             | Sync Mechanism         | Backup Available |
|----------------------|------------------------|------------------|
| Apple iCloud Keychain| iCloud sync            | Yes (encrypted)  |
| Google Password Mgr  | Google Account sync    | Yes (encrypted)  |
| Microsoft            | Windows Hello + Azure  | Partial          |
| 1Password            | Cloud vault            | Yes              |
| Dashlane             | Cloud vault            | Yes              |

**Key properties:**
- Portable across devices within the same ecosystem
- Private key never leaves the sync fabric in plaintext
- No attestation guarantee (the authenticator is a software key)
- `AAGUID` is typically `00000000-0000-0000-0000-000000000000` (zero GUID)
- Attestation format: `"none"` or platform-specific

### 2. Device-Bound Credentials (Single-Device)

Device-bound credentials are scoped to a single hardware authenticator:

| Authenticator       | Transport     | Attestation Format  |
|---------------------|---------------|---------------------|
| YubiKey 5 Series    | USB/NFC       | packed, fido-u2f    |
| SoloKeys            | USB/NFC       | packed              |
| Titan Security Key  | USB/NFC/BLE   | packed, fido-u2f    |
| TPM 2.0             | Platform      | tpm                 |
| Android Secure Key  | Platform      | android-key         |
| Android SafetyNet   | Platform      | android-safetynet   |
| Apple Touch/Face ID | Platform      | apple               |

**Key properties:**
- Private key is non-exportable (hardware-backed)
- Provides attestation proof (verifiable authenticator model)
- `AAGUID` is non-zero and identifies the authenticator model
- Attestation conveys trust chain to the relying party

## Attestation Conveyance

WebAuthn supports three attestation conveyance preferences:

| Preference     | Behavior                                         | Use Case                        |
|----------------|--------------------------------------------------|---------------------------------|
| `none`         | Server receives self-attestation or no attestation | Privacy-first, consumer apps  |
| `indirect`     | Server receives anonymized attestation           | Balanced privacy + trust        |
| `direct`       | Server receives full attestation chain           | Enterprise, regulated industries|

### GGID Recommendation

- **Consumer deployments**: Use `"none"` — privacy-first, avoids authenticator fingerprinting
- **Enterprise deployments**: Use `"direct"` — verify hardware authenticator, enforce policy
- **Regulated (HIPAA/PCI/PSD2)**: Use `"direct"` with allowlist of approved AAGUIDs

## GGID Attestation Verification

GGID implements all 7 standard WebAuthn attestation formats in `services/auth/internal/webauthn/attestation_formats.go`:

| Format              | Verification Logic                                         | Status    |
|---------------------|------------------------------------------------------------|-----------|
| `none`              | Accepts self-attestation, extracts public key             | Verified  |
| `packed`            | Verifies signature against attestation cert chain         | Verified  |
| `fido-u2f`          | Verifies ECDSA P-256 cert + U2F signature format          | Verified  |
| `android-key`       | Verifies key attestation extension (OID 1.3.6.1.4.1.11129) | Verified  |
| `android-safetynet` | Verifies JWS response with x5c certificate chain          | Verified  |
| `tpm`               | Verifies RSA/ECDSA cert + TPM-specific signature          | Verified  |
| `apple`             | Verifies Apple anonymized attestation format              | Verified  |

### Verification Flow

```
Client → Registration Response
         ↓
  VerifyAttestationFormat(format, authData, clientDataHash, alg, sig, certBytes)
         ↓
    switch format:
      case "none"       → VerifyNoneAttestation()
      case "packed"     → verifyPackedAttestation()
      case "fido-u2f"   → verifyFidoU2FAttestation()
      case "android-key"→ verifyAndroidKeyAttestation()
      case "android-safetynet" → verifyAndroidSafetynetAttestation()
      case "tpm"        → verifyTPMAttestation()
      case "apple"      → verifyAppleAttestation()
```

### Algorithm Support

GGID verifies COSE algorithm IDs:

| COSE Alg | Algorithm | Use              |
|----------|-----------|------------------|
| -7       | ES256     | ECDSA P-256      |
| -257     | RS256     | RSA 2048         |
| -8       | EdDSA     | Ed25519          |

## Synced vs Device-Bound: Security Implications

### Threat Model Differences

| Threat                | Synced Passkeys              | Device-Bound                  |
|-----------------------|------------------------------|-------------------------------|
| Phishing              | Resistant (origin-bound)     | Resistant (origin-bound)      |
| Device theft          | All synced devices at risk   | Only one device compromised   |
| Account recovery      | Sync fabric handles recovery | Requires re-registration      |
| Supply chain          | Platform vendor trust        | Hardware vendor trust         |
| Attestation guarantee | None (software key)          | Yes (hardware-backed)         |
| Compliance (PSD2 SCA) | May not qualify              | Qualifies (if certified)      |

### Enterprise Policy Recommendations

1. **Mandatory attestation**: Require `"direct"` conveyance for employee accounts
2. **AAGUID allowlist**: Only accept certified authenticators (FIDO Alliance L2+)
3. **User verification**: Require `userVerification: "required"` (biometric/PIN)
4. **Resident key policy**: Allow discoverable credentials for passwordless flows
5. **Backup eligibility**: For high-security contexts, reject backup-eligible credentials (device-bound only)

## Platform-Specific Behavior

### Apple Platform Authenticator

- Uses `"apple"` attestation format (anonymized)
- Synced via iCloud Keychain (backup eligible)
- Does NOT support `"fido-u2f"` format
- AAGUID is `00000000-...` when synced, non-zero when device-bound

### Google Platform Authenticator

- Uses `"android-key"` or `"android-safetynet"` format
- Synced via Google Password Manager (backup eligible)
- SafetyNet is deprecated in favor of Play Integrity API

### Hardware Security Keys

- Use `"packed"` or `"fido-u2f"` format
- NOT synced (device-bound by definition)
- Strongest attestation guarantee

## Future Considerations

### FIDO Alliance Passkey Standardization

- **Credential Exchange Protocol (CXF)**: Enabling passkey transfer between ecosystems
- **Hybrid transport**: QR code + BLE proximity for cross-device authentication
- **Conditional UI**: Autofill API integration for seamless passkey selection

### Post-Quantum Readiness

- Current WebAuthn algorithms (ES256, RS256, EdDSA) are not quantum-resistant
- NIST PQC standardization (ML-KEM, ML-DSA) will require WebAuthn spec updates
- GGID's modular attestation architecture supports future algorithm additions

## GGID Configuration Recommendations

```yaml
# Consumer deployment
webauthn:
  attestation_conveyance: "none"
  user_verification: "preferred"
  resident_key: "preferred"

# Enterprise deployment
webauthn:
  attestation_conveyance: "direct"
  user_verification: "required"
  resident_key: "required"
  aaguid_allowlist:
    - "cb69481e-8ff7-4039-93ec-0a2729a154a8"  # YubiKey 5
    - "08987058-cadc-4b81-b6e1-30de50dcbe96"  # Windows Hello
```

## References

- [W3C WebAuthn Level 3 (2024)](https://www.w3.org/TR/webauthn-3/)
- [FIDO Alliance Passkey Specifications](https://fidoalliance.org/passkeys/)
- [Apple Platform Security Guide](https://support.apple.com/guide/security/welcome/web)
- [Google Password Manager & Passkeys](https://developers.google.com/identity/fido)

## See Also

- [FIDO2 Passkey Ecosystem 2026](fido2-passkey-ecosystem-2026.md)
- [Passwordless Setup Guide](passwordless-setup.md)
- [Device-Bound SSO Analysis](device-bound-sso-analysis.md)
