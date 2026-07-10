# FIDO Alliance Interoperability Report — GGID WebAuthn

> **Scope**: How GGID interacts with diverse FIDO2/WebAuthn authenticators, attestation
> formats, and the FIDO Alliance Metadata Service. Includes implementation gaps and
> a phased improvement roadmap.

---

## 1. Overview

WebAuthn is the W3C standard that forms the browser-facing half of **FIDO2**;
the other half is CTAP2 (Client-to-Authenticator Protocol), which transports
assertions over USB, NFC, BLE, or the hybrid (caBLE) channel.

Authenticators fall into two categories:

| Category | Description | Examples |
|---|---|---|
| **Platform** | Built into the device; non-removable | Touch ID, Face ID, Windows Hello, Android Biometric |
| **Cross-platform** | Removable hardware tokens | YubiKey, Google Titan, Feitian ePass, SoloKey |

The relying party (GGID) must accept whichever authenticator the user's browser
presents. Each device differs in:

- **Transports** (`usb`, `nfc`, `ble`, `internal`, `hybrid`, `smart-card`)
- **Attestation formats** (`none`, `packed`, `tpm`, `android-key`, `apple`, …)
- **Backup/sync capability** (iCloud Keychain, Google Password Manager, device-bound)
- **AAGUID** (authenticator model identifier — may be all-zeros for platform)
- **Certification status** (FIDO Alliance L1/L2 certified or not)

GGID's handler stores `AttestationType`, `AAGUID`, `BackupEligible`,
`BackupState`, `UserVerified`, and `Transports` for each credential, but does
not yet cross-reference them against the FIDO Metadata Service.

---

## 2. Platform Authenticator Matrix

| Platform | Biometric | Backup Eligible | Sync Fabric | Conditional UI | Hybrid Transport | AAGUID |
|---|---|---|---|---|---|---|
| Apple Touch ID / Face ID | Touch ID, Face ID | Yes | iCloud Keychain | Safari 16+ | Yes (iOS↔Mac) | Often all-zeros |
| Windows Hello | Face, Fingerprint, PIN | No (device-bound) | — | Chrome 107+, Edge | Limited (Android pairing) | Per Windows version |
| Android Biometric | Fingerprint, Face | Yes (Q+) | Google Password Mgr | Chrome 107+ | Yes (Android↔desktop) | Varies by OEM |
| Chrome on Linux | PIN (libfido2) | No | — | Chrome 107+ | Yes (phone pairing) | Varies |

### Apple Touch ID / Face ID
- Available on macOS and iOS via Safari.
- Backup eligible: true — credentials sync through iCloud Keychain.
- Conditional mediation: Safari 16+.
- Hybrid transport: an iPhone can act as an authenticator for a Mac via the
  `hybrid` transport (caBLE), enabling cross-device passkey flows.
- AAGUID: frequently all-zeros (Apple suppresses model-level identification
  for privacy). Attestation format is `apple` (anonymous attestation).

### Windows Hello
- Available on Windows 10/11 via Edge or Chrome.
- Biometrics (Face/Fingerprint) fall back to PIN.
- Backup eligible: false by default — credentials are device-bound.
- Conditional mediation: Chrome 107+, Edge.
- Attestation format: `tpm` (Trusted Platform Module).
- AAGUID: specific per Windows build.

### Android Biometric
- Available on Android 7+ via Chrome.
- Backup eligible: true on Android Q+ (Google Password Manager sync).
- Hybrid transport: Android can serve as an authenticator for a desktop
  browser via the `hybrid` transport.
- Attestation format: `android-key` (newer) or `android-safetynet` (legacy).
- AAGUID: varies by device manufacturer (Samsung, Pixel, etc.).

### Chrome on Linux
- Platform authenticator support is limited; typically falls back to PIN via
  `libfido2`.
- Most users rely on USB/NFC security keys (cross-platform) instead.
- Backup eligible: false (no sync fabric).
- Attestation format: `none` (by default) or `packed` (if using a security key).

---

## 3. Cross-Platform Authenticator Matrix

| Device | Connectors | Transports | Attestation | Backup | Approx. Cost |
|---|---|---|---|---|---|
| **YubiKey 5 Series** | USB-A, USB-C, NFC, Lightning | `usb`, `nfc` | `packed` (RSA/ECDSA) | No | $40–75 |
| **Google Titan** | USB-C/NFC (BLE on older) | `usb`, `nfc` | `packed` | No | $25–40 |
| **Feitian ePass** | USB-A/C, NFC | `usb`, `nfc` | `packed` | No | $20–50 |
| **SoloKey** | USB-A/C, NFC | `usb`, `nfc` | `packed` | No | $15–40 |

### YubiKey 5 Series (Yubico)
- The most widely deployed FIDO2 security key.
- Supports FIDO2/CTAP2, U2F, OTP, OpenPGP, PIV (multi-protocol).
- FIDO Alliance L2 certified.
- Attestation: `packed` with RSA or ECDSA signatures and a full certificate
  chain rooted at the Yubico FIDO root CA.
- AAGUID varies by form factor (USB-A vs USB-C vs NFC-only).

### Google Titan (Google)
- USB-C/NFC key with optional BLE bridge on older models.
- FIDO2 certified.
- Attestation: `packed`.
- AAGUID is registered in the FIDO MDS.

### Feitian ePass (Feitian)
- Budget-friendly alternative to YubiKey.
- FIDO2 certified.
- Attestation: `packed`.
- AAGUID registered in MDS.

### SoloKey (Open Source)
- Open-source firmware (Somu and Tap variants).
- Attestation: `packed` — user can self-sign root for personal deployments.
- Not always FIDO Alliance certified (depends on batch).
- AAGUID registered for certified batches.

---

## 4. AAGUID Reference

The **AAGUID** is a 128-bit UUID embedded in the attested credential data
during registration. It identifies the authenticator model (make + firmware).
The FIDO Alliance **Metadata Service (MDS)** maps AAGUIDs to:

- Device name and manufacturer
- Certification level (L1, L2, L3)
- Supported attestation root certificates
- Status reports (revoked, compromised)

GGID currently stores `AAGUID []byte` on each credential but does not look it
up against MDS.

### Common AAGUIDs (representative)

| AAGUID | Device |
|---|---|
| `00000000-0000-0000-0000-000000000000` | Apple platform authenticator (anonymous) |
| `08987058-cadc-4b81-b6e1-3de6692c6130` | YubiKey 5 USB-A |
| `cb69481e-8ff7-4039-93ec-021c1e90a7f4` | YubiKey 5C |
| `fa2b99dc-9e39-4257-8f92-4b3090f0a8ab` | YubiKey 5 NFC |
| `73bb0cd4-e502-49b8-9c67-3b4a0e82c3d1` | YubiKey 5C NFC |
| `149a2021-8ef6-4a3e-bd2f-aa4d3ffd4b4d` | Google Titan v1 |
| `dd4ec289-e01d-41c9-bb89-70fa845d4bf2` | Feitian ePass FIDO |
| `8876631b-d4a0-427f-833b-f819374e3c79` | SoloKey Tap |
| `00000000-0000-0000-0000-000000000000` | Windows Hello (varies by build) |
| `bada5566-a7aa-401f-bd96-45619a55420d` | Android Authenticator (Google) |

### Go: Fetching MDS Metadata

```go
// Fetch FIDO MDS v3 blob (JWT-embedded metadata BLOB).
// Endpoint: https://mds.fidoalliance.org/
func fetchMDS(ctx context.Context) ([]byte, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET",
        "https://mds.fidoalliance.org/", nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

// The response is a JWS-signed JWT whose payload (base64) contains
// a JSON array of MetadataBLOBPayloadEntry objects, each with an aaguid
// field. Parse with a JWT library, verify the FIDO root signature, then
// cache the resulting map[string]MetadataEntry in Redis.
```

Recommended caching strategy: fetch once daily, store in Redis with a TTL,
and look up AAGUIDs at registration time to enrich the stored credential
with a human-readable device name.

---

## 5. Attestation Formats

| Format | Typical Source | Privacy | Verification Complexity |
|---|---|---|---|
| `none` | Platform (Apple, Google, default) | High | Trivial |
| `packed` | Hardware keys (YubiKey, Titan) | Low | Medium |
| `tpm` | Windows Hello | Low | High |
| `android-key` | Android Keystore | Medium | Medium |
| `android-safetynet` | Legacy Android | Low | Medium (deprecated) |
| `fido-u2f` | Legacy U2F keys | Low | Low |
| `apple` | Apple platform | High | Low |

### packed
Most common for hardware security keys. Contains a signature over the
authenticator data + client data hash, using RS256, ES256, or PS256.
Includes a certificate chain: authenticator cert -> intermediate -> FIDO
root CA.

**GGID action**: `go-webauthn` verifies the signature and checks the
cert chain against the trusted roots configured in `webauthn.Config`.
GGID should additionally cross-reference the AAGUID against MDS for
certification status.

### none
No attestation data — the authenticator declines to prove its identity.
This is the default recommendation from the W3C spec for consumer-facing
RPs and is what most platform authenticators use.

**GGID action**: Accept without verification. The AAGUID may still be
present in the attested credential data for identification purposes
(often all-zeros).

### tpm
Used by Windows Hello. Contains TPM2 structures with certification info.
The certificate chain is rooted at the TPM manufacturer's CA, and the
 endorsement key cert is signed by a trusted intermediate.

**GGID action**: `go-webauthn` parses the TPM2 structure and verifies
the signature. Full validation requires pinning Microsoft's TPM root
certificates and optionally checking the endorsement key against MDS.

### android-key
Used by Android devices with Keystore-backed keys. Contains a keymaster
certificate chain rooted at Google's hardware attestation root.

**GGID action**: `go-webauthn` validates the cert chain against
Google's root. GGID should accept but note that hardware attestation
requires Google Play Services.

### android-safetynet
Deprecated in favor of `android-key`. Uses Google SafetyNet (now Play
Integrity API) to provide device integrity attestation.

**GGID action**: Accept but log a warning recommending `android-key`.
Monitor for removal in future WebAuthn spec revisions.

### fido-u2f
Legacy format from U2F security keys (pre-CTAP2). Simple and
well-understood: a single signature over the key handle + RP ID hash.

**GGID action**: Accept for backward compatibility with older keys.

### apple
Apple's anonymous attestation. Contains a nonce and timestamp but no
device-specific information, preserving user privacy.

**GGID action**: `go-webauthn` verifies the nonce. Accept without
further device identification.

---

## 6. Certificate Chain Validation

The trust model for WebAuthn attestation is:

```
FIDO Alliance Root CA
    └── Authenticator Manufacturer Intermediate CA
        └── Individual Authenticator Certificate (per device)
```

### Validation Steps

1. **Extract cert chain** from the attestation statement.
2. **Verify signatures**: each cert signed by its parent.
3. **Check trust roots**: the root must be a FIDO Alliance-trusted CA
   (downloaded from the MDS or pinned in configuration).
4. **Check revocation**: CRL or OCSP for the leaf cert.
5. **Check expiry**: all certs must be within validity period.
6. **Cross-reference MDS**: confirm the AAGUID is listed as certified
   and not revoked.

### Go Implementation Sketch

```go
func verifyAttestationCertChain(leaf, intermediate *x509.Certificate,
    roots *x509.CertPool) error {
    opts := x509.VerifyOptions{
        Roots:         roots,          // FIDO Alliance root CAs
        Intermediates: x509.NewCertPool(),
        KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
    }
    if intermediate != nil {
        opts.Intermediates.AddCert(intermediate)
    }
    _, err := leaf.Verify(opts)
    return err
}
```

`go-webauthn` performs these checks internally when an attestation root
pool is configured via `webauthn.Config.AttRootCAs`. GGID does not
currently configure this pool, meaning all attestation formats pass
without cert-chain validation.

---

## 7. GGID Implementation Status

### What GGID's handler currently does

Based on `services/auth/internal/webauthn/handler.go`:

| Feature | Status | Code Reference |
|---|---|---|
| Full registration + authentication flow | Done | `beginRegistration`, `finishRegistration`, `beginAuthentication`, `finishAuthentication` |
| Attestation verification (signature + origin) | Done (delegated) | `h.wbn.CreateCredential()` / `ValidateLogin()` |
| Sign-count clone detection | Done | WA-2 check in `finishAuthentication` |
| Credential transports persisted | Done | WA-8: `credential.Transport` |
| Backup eligible / backup state flags | Done | WA-1: stored on `Credential` |
| User verification flag | Done | WA-1: `UserVerified` |
| AAGUID stored | Done | `credential.Authenticator.AAGUID` |
| Exclusion of existing credentials | Done | WA-3: `excludeCreds` |
| Resident key preference | Done | WA-4: `ResidentKeyRequirementPreferred` |
| Credential auto-naming from User-Agent | Done | WA-7: `generateCredentialName()` |
| Error classification | Done | `classifyError()` |
| Related Origin Requests (ROR) | Done | WA-11: `/.well-known/webauthn` |
| Mobile asset links | Done | WA-12: `assetlinks.json`, `apple-app-site-association` |
| Discoverable credential auth | Done | WA-15: ephemeral user flow |

### What is missing

| Gap | Priority | Description |
|---|---|---|
| **MDS integration** | High | No FIDO Metadata Service lookup; AAGUIDs stored but never resolved to device names or certification status |
| **Cert chain validation** | Medium | `go-webauthn` supports `AttRootCAs` but GGID does not configure a root pool; all attestation formats pass without chain validation |
| **Per-tenant attestation policy** | Medium | No config to restrict accepted attestation formats per tenant (e.g., `none`-only for consumer, `packed` for enterprise) |
| **AAGUID display in UI** | Low | Console shows credential name but not resolved device model |

### go-webauthn library responsibilities

The `github.com/go-webauthn/webauthn` library handles:
- CBOR decoding of authenticator responses
- COSE key parsing and verification
- Attestation format verification for all standard formats
- Signature verification over authenticator data + client data hash
- Origin and RP ID matching
- Optional cert chain validation (when `AttRootCAs` is set)

GGID must add on top: MDS integration, per-tenant policy enforcement,
and user-facing device naming.

---

## 8. Recommendations

### Phase 1: Accept All Formats (Parse-Only) — Current
GGID already accepts all attestation formats because `AttRootCAs` is not
configured. This is the W3C-recommended default for consumer RPs. No
change needed, but **store the attestation format** explicitly for
auditing (already done via `AttestationType`).

### Phase 2: MDS Integration (High Priority)
```go
// Fetch MDS blob daily, cache in Redis.
type MDSCache struct {
    rdb    *redis.Client
    entries map[string]MetadataEntry // aaguid -> metadata
}

// At registration time, look up the AAGUID:
func (c *MDSCache) Lookup(aaguid []byte) (*MetadataEntry, bool) {
    key := uuid.FromBytesOrNil(aaguid).String()
    entry, ok := c.entries[key]
    return &entry, ok
}
```
This enables GGID to display device names in the Console and flag
uncertified or revoked authenticators.

### Phase 3: Cert Chain Validation (Medium Priority)
Download FIDO Alliance root CAs from the MDS blob and configure them:
```go
wconfig := &webauthn.Config{
    RPDisplayName: rpName,
    RPID:          rpID,
    RPOrigins:     origins,
    AttRootCAs:    fidoRootPool, // x509.CertPool from MDS
}
```
This enables full `packed` and `tpm` cert-chain verification for
high-security tenants while keeping `none` accepted for others.

### Phase 4: Per-Tenant Attestation Policy
Add tenant-level configuration to control accepted attestation formats:

```go
type AttestationPolicy struct {
    AllowedFormats []string // e.g. ["none", "packed", "tpm"]
    RequireMDS     bool     // reject uncertified authenticators
    RequireUV      string   // "required", "preferred", "discouraged"
}
```

Consumer tenants: `["none", "packed"]` (lenient).
Enterprise tenants: `["packed", "tpm"]` + `RequireMDS: true` (strict).

---

*Last updated: 2025 — based on GGID WebAuthn handler at `services/auth/internal/webauthn/handler.go` and FIDO Alliance specifications (CTAP 2.1, W3C WebAuthn Level 3).*
