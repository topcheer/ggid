# FIDO Metadata Service (MDS) 3.0 — Research & Implementation Plan

> Research document for integrating FIDO Alliance Metadata Service into GGID's
> WebAuthn authentication flow. Targets the `webauthn.Handler` at
> `services/auth/internal/webauthn/handler.go`.

---

## 1. Overview

The **FIDO Metadata Service (MDS) 3.0** is a centralized, FIDO Alliance-maintained
registry of certified authenticators. It allows Relying Parties (RPs) to verify
that a WebAuthn credential was produced by a genuine, certified device and to
check its security status.

**Key facts:**
- Maintained by the FIDO Alliance
- Endpoint: `https://fidoalliance.co/metadata/` (redirects to the latest BLOB)
- Format: a single JWS-signed JWT (the "BLOB") containing metadata for all
  certified authenticators worldwide
- Refresh cadence: `nextUpdate` field in the BLOB header, typically 7–30 days
- Purpose: RP can verify authenticator certification, capabilities, and security
  level; detect revoked or compromised devices

**Why GGID needs this:** Today GGID's `webauthn.Handler` calls
`h.wbn.CreateCredential()` which performs cryptographic attestation verification
but does **not** check whether the authenticator is FIDO-certified, look up its
metadata, or enforce any authenticator policy. The AAGUID is stored
(`Credential.AAGUID`, line 37) but never resolved to a device name or validated
against any registry. This document proposes closing that gap.

---

## 2. Metadata BLOB Structure

### BLOB Format

The MDS BLOB is a compact JWT (JWS-signed). It has three parts:

```
eyJhbGciOiJFUzI1NiIsIng1Y3QiOlsi...header x5c chain...
eyJub0VudHJpZXMiOjI1MDAs...payload (base64url JSON)...
base64url(signature)
```

**JWT Header:** `{ "alg": "ES256", "x5c": ["...blob-signing-cert...", "...intermediate...", "...root..."] }`
- The `x5c` array is a certificate chain leading up to the FIDO Alliance Root CA.

**JWT Payload** contains two top-level objects:

| Field | Type | Description |
|-------|------|-------------|
| `legalHeader` | string | Legal notice / usage terms |
| `no` | int | BLOB sequence number (monotonic) |
| `nextUpdate` | string (date) | When this BLOB should be refreshed (`YYYY-MM-DD`) |
| `entries` | array | Array of `MetadataBLOBEntry` objects |

### Metadata Statement Fields (per entry)

Each `entry.metadataStatement` describes one authenticator model:

```json
{
  "aaguid": "cb69481e-8ff7-4039-93ec-021c1b4a4c4a",
  "description": "YubiKey 5 NFC",
  "authenticatorVersion": 2,
  "protocolFamily": "fido2",
  "authenticationAlgorithms": ["rsassa-pkcs1-v1_5-with-sha-256", "es256"],
  "userVerificationDetails": [["User Verify"] [["biometric","any"], ["passcode","any"]]],
  "keyProtection": ["hardware", "secureElement"],
  "matcherProtection": "onChip",
  "attachmentHint": ["external", "wireless", "nfc"],
  "attestationRootCertificates": ["MIID..."],
  "icon": "data:image/svg+xml;base64,..."
}
```

| Field | Purpose |
|-------|---------|
| `aaguid` | Authenticator model identifier (UUID); the primary lookup key |
| `description` | Human-readable name for UI display |
| `authenticatorVersion` | Firmware version (for revocation scope) |
| `protocolFamily` | `"fido2"`, `"uaf"`, or `"u2f"` |
| `authenticationAlgorithms` | Supported signature algorithms (RS256, ES256, EdDSA) |
| `userVerificationDetails` | Supported UV methods (biometric, passcode) |
| `keyProtection` | `hardware`, `secureElement`, `tee` |
| `matcherProtection` | `onChip`, `tee`, `software` |
| `attestationRootCertificates` | Root CAs used to validate this model's attestation certs |

### Status Reports

Each entry also carries `statusReports` — an array of lifecycle events:

```json
{
  "statusReports": [
    { "status": "FIDO_CERTIFIED", "certificateNumber": "0", "certificationDescriptor": "YubiKey 5 NFC", "report": "2019-08-21T00:00:00" }
  ],
  "timeOfLastStatusChange": "2019-08-21"
}
```

**Critical statuses the RP must check:**

| Status | Action |
|--------|--------|
| `NOT_FIDO_CERTIFIED` | Warn — device failed or never attempted certification |
| `FIDO_CERTIFIED` | Accept — passed functional certification |
| `SELF_ASSERTION_SUBMITTED` | Caution — vendor self-reported, not independently verified |
| `REVOKED` | **Reject** — certification revoked |
| `USER_VERIFICATION_BYPASS` | **Reject** — UV can be bypassed |
| `ATTESTATION_KEY_COMPROMISE` | **Reject** — attestation keys leaked |
| `UPDATE_AVAILABLE` | Warn — firmware update recommended |

---

## 3. Trust Anchor and Certificate Chain

### Trust Chain

Three layers of certificates underpin the MDS:

```
FIDO Alliance Root CA (self-signed)
├── BLOB Signing Certificate (signs the metadata BLOB JWT)
└── Attestation CA Certificates (per-manufacturer; sign authenticator certs)
    ├── Yubico Attestation CA
    ├── Google Hardware Attestation CA
    ├── Feitian Attestation CA
    └── ...
```

1. **FIDO Alliance Root CA** — the ultimate trust anchor. A self-signed root
   certificate published by the FIDO Alliance. This is embedded in the RP.

2. **BLOB Signing Certificate** — issued by the FIDO root, used only to sign the
   metadata BLOB JWT. Its validity proves the BLOB is authentic.

3. **Attestation CA Certificates** — per-manufacturer CAs that sign individual
   authenticator attestation certificates during registration. These are listed
   in each metadata statement's `attestationRootCertificates` field.

### Validation Flow

1. Download BLOB from FIDO MDS endpoint → verify JWT signature against FIDO root (validate `x5c` chain)
2. Parse entries into `map[AAGUID]MetadataStatement`
3. At registration: extract AAGUID from attested credential data
4. Look up AAGUID in local MDS cache
5. Check `statusReports` — reject if `REVOKED`, `USER_VERIFICATION_BYPASS`, or `ATTESTATION_KEY_COMPROMISE`
6. Get `attestationRootCertificates` for this model
7. Validate authenticator's attestation cert chain against MDS-provided root

**GGID gap:** Step 8 is entirely missing. The `go-webauthn` library's
`CreateCredential` validates the attestation signature but does not verify the
attestation certificate against a manufacturer root from MDS. For `attestation:
"none"` (the common consumer case) this is fine. For enterprise deployments
requiring `attestation: "direct"`, the MDS root check is essential.

---

## 4. AAGUID Lookup

### How AAGUID Works

The **AAGUID** (Authenticator Attestation GUID) is a 128-bit UUID burned into
authenticator hardware at manufacturing time. It identifies the authenticator
**model**, not an individual device. It appears in the attested credential data
during WebAuthn registration.

**Platform authenticators** (Windows Hello, iCloud Keychain, Android) may set
AAGUID to all-zeros (`00000000-0000-0000-0000-000000000000`) for privacy, in
which case MDS lookup is not possible.

### Lookup Process

On registration, extract AAGUID from `authData.attestedCredentialData.aaguid` (16 bytes) and query the local cache:
- **Found + `FIDO_CERTIFIED`** → accept, show device name from `description`
- **Found + `REVOKED`** → reject or flag for admin review
- **Not found** → accept with warning (unknown device) or reject per policy

### Common AAGUIDs Table

| AAGUID | Device | Attachment | Key Protection |
|--------|--------|-----------|----------------|
| `cb69481e-8ff7-4039-93ec-021c1b4a4c4a` | YubiKey 5 NFC | external/nfc/usb | hardware |
| `cb69481e-8ff7-4039-93ec-021c1b4a4c4a` | YubiKey 5 NFC (alt) | external | hardware |
| `0bb43545-fc0f-42b6-a1f2-f7e3e57b2e30` | YubiKey 5 Nano | external/usb | hardware |
| `2a0bc837-b3c3-4f40-9f80-d4fc4d8a79ac` | YubiKey 5Ci | external/lightning | hardware |
| `08987058-cadc-4b81-b6e1-30de50dcbe96` | Windows Hello | platform | software/tee |
| `9ddd1817-af5a-4672-a2b9-3e3dd95000a9` | Windows Hello (alt) | platform | tee |
| `adce0002-35bc-c60a-648b-0b25f1f05503` | Chrome on Android | platform | tee |
| `dd4ec289-e01d-41c9-bb89-70fa845d4bf2` | iCloud Keychain | platform (cross-device) | secureEnclave |
| `fbfc3007-154e-4ecc-8c0b-6e02455b1c43` | Google Titan (USB) | external/usb | hardware |
| `f8a011f3-8c0a-4d0c-bc71-4d2c80b87dbf` | Feitian ePass FIDO | external/usb | hardware |
| `8876631b-d4a0-427f-5773-0ec71d9c0c1c` | SoloKey Tap | external/nfc | hardware |
| `6d44ba9b-f6ec-2e49-b930-0c8fe980cb77` | SoloKey USB | external/usb | hardware |

> Note: AAGUIDs can vary by firmware version. The community-maintained list at
> `github.com/passkeydeveloper/passkey-authenticator-aaguids` is a good
> supplementary source for platform authenticators that may not be in MDS.

---

## 5. GGID Authenticator Policy

### Current State in GGID

Examining `handler.go`, the WebAuthn registration flow currently:
- Calls `h.wbn.CreateCredential()` for cryptographic verification (line 535)
- Stores `AttestationType` and `AAGUID` in the `Credential` struct (lines 36-37)
- Generates device names from User-Agent via `generateCredentialName()` (line 199)
- Does **not** perform any MDS lookup, device naming by AAGUID, or policy enforcement

### Proposed Per-Tenant Policy Levels

| Level | Description | Use Case |
|-------|-------------|----------|
| `none` | Accept all authenticators (default) | Consumer apps |
| `certified` | Only FIDO-certified (check MDS status) | Enterprise SaaS |
| `enterprise` | Certified + specific AAGUID allowlist | Corporate managed devices |
| `government` | Certified + hardware-backed only (`keyProtection: hardware`) | Regulated industries |

Config stored per tenant:

```sql
CREATE TABLE authenticator_policies (
    tenant_id        UUID PRIMARY KEY REFERENCES tenants(id),
    policy_level     VARCHAR(32) NOT NULL DEFAULT 'none',
    allowed_aaguids  JSONB DEFAULT '[]'::jsonb,   -- for "enterprise" level
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Enforcement Points

1. **Registration** (`finishRegistration`, line 500): After `CreateCredential`
   succeeds, check AAGUID against tenant policy before persisting.
2. **Authentication** (`finishAuthentication`, line 668): Verify authenticator
   is still certified (periodic status re-check).
3. **Background**: Re-check all enrolled authenticators against updated MDS,
   flag any newly-revoked devices.

### Go Implementation Sketch

```go
type AuthenticatorPolicy interface {
    ShouldAccept(aaguid string, mds *MetadataService) (bool, string)
}

type TenantPolicy struct {
    Level          PolicyLevel   // none | certified | enterprise | government
    AllowedAAGUIDs []string      // for "enterprise" level
}

func (p *TenantPolicy) ShouldAccept(aaguid string, mds *MetadataService) (bool, string) {
    switch p.Level {
    case PolicyNone:
        return true, "accepted (no policy)"
    case PolicyCertified, PolicyEnterprise:
        stmt, found := mds.Lookup(aaguid)
        if !found { return false, "not in FIDO registry" }
        if stmt.IsRevoked() { return false, "certification revoked" }
        if p.Level == PolicyEnterprise && !slices.Contains(p.AllowedAAGUIDs, aaguid) {
            return false, "not in enterprise allowlist"
        }
        return true, stmt.Description
    }
    return true, "accepted"
}
```

---

## 6. BLOB Refresh Strategy

The BLOB header includes `nextUpdate` — a date after which the BLOB is stale.
The FIDO Alliance publishes new BLOBs roughly monthly.

**Strategy:**

1. **On startup**: Check `nextUpdate` of cached BLOB. If past, download fresh.
2. **Background goroutine**: Check daily; refresh when `nextUpdate` is within
   3 days (proactive refresh to avoid a window where BLOB expires).
3. **Fallback**: If the FIDO endpoint is unreachable, continue using the cached
   BLOB with a logged warning. Never block authentication on MDS availability.
4. **Cache**: Store the raw JWT in Redis (or DB) so all instances share one copy.

```go
func (s *MetadataService) RefreshIfNeeded(ctx context.Context) error {
    if s.cache != nil && time.Until(s.cache.NextUpdate) > 72*time.Hour {
        return nil // still fresh
    }
    blob, err := s.fetchBlob(ctx)
    if err != nil && s.cache != nil {
        log.Warn("MDS unreachable, using stale cache", "err", err)
        return nil
    }
    return s.verifyAndStore(ctx, blob)
}
```

---

## 7. Implementation Design

### Data Model

```sql
CREATE TABLE fido_metadata_cache (
    id              INT PRIMARY KEY DEFAULT 1,  -- always row 1
    blob_jwt        TEXT NOT NULL,              -- raw JWS JWT
    blob_hash       VARCHAR(64) NOT NULL,       -- SHA-256 of payload
    num_entries     INT NOT NULL,
    next_update     DATE NOT NULL,
    last_fetched    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- authenticator_policies: see Section 5
```

### Core Go Types and Functions

```go
type MetadataService struct {
    endpoint string           // "https://fidoalliance.co/metadata/"
    rootCAs  *x509.CertPool   // FIDO Alliance Root CA
    cache    map[string]*MetadataStatement
    store    MetadataStore    // DB or Redis persistence
}

type MetadataStatement struct {
    AAGUID                      string   `json:"aaguid"`
    Description                 string   `json:"description"`
    AuthenticatorVersion        int      `json:"authenticatorVersion"`
    ProtocolFamily              string   `json:"protocolFamily"`
    AuthenticationAlgorithms   []string `json:"authenticationAlgorithms"`
    KeyProtection               []string `json:"keyProtection"`
    MatcherProtection           string   `json:"matcherProtection"`
    AttestationRootCertificates []string `json:"attestationRootCertificates"`
}

func (s *MetadataService) FetchBlob(ctx context.Context) (string, error)
func (s *MetadataService) VerifyBlob(blobJWT string) (*MetadataBLOBPayload, error)
func (s *MetadataService) ParseBlob(p *MetadataBLOBPayload) map[string]*MetadataStatement
func (s *MetadataService) Lookup(aaguid string) (*MetadataStatement, bool)
func (s *MetadataService) CheckAuthenticator(aaguid string, p AuthenticatorPolicy) (bool, string)
func (s *MetadataService) RefreshIfNeeded(ctx context.Context) error
```

### Integration with WebAuthn Handler

In `finishRegistration` (line 534), after `CreateCredential` succeeds:

```go
if h.mds != nil {
    aaguidStr := uuid.Must(uuid.FromBytes(credential.Authenticator.AAGUID)).String()
    policy, _ := h.policyResolver.Get(ctx, tenantID)
    if accepted, reason := h.mds.CheckAuthenticator(aaguidStr, policy); !accepted {
        writeError(w, http.StatusForbidden, fmt.Sprintf("authenticator rejected: %s", reason))
        return
    }
    if stmt, ok := h.mds.Lookup(aaguidStr); ok && stmt.Description != "" {
        name = stmt.Description // override User-Agent-derived name
    }
}
```

---

## 8. Roadmap

| Phase | Scope | Effort |
|-------|-------|--------|
| **Phase 1** | MDS BLOB download + JWT verification + DB/Redis cache (no enforcement) | 2 days |
| **Phase 2** | AAGUID lookup + device naming in WebAuthn handler + Admin Console UI | 1–2 days |
| **Phase 3** | Per-tenant authenticator policy enforcement (`none`/`certified`/`enterprise`) | 3 days |
| **Phase 4** | Status report monitoring — background re-check of enrolled authenticators, alert admin on revoked devices | 2 days |
| **Total** | | **8–10 days** |

**Phase 1–2** delivers immediate value (accurate device names instead of
"Chrome on macOS"). **Phase 3–4** adds enterprise security posture. The design
is backward-compatible — default policy `none` means zero behavior change.

### Dependencies

- No new Go modules: JWT via `github.com/golang-jwt/jwt/v5` (in `go.mod`), X.509 via stdlib.
- FIDO Root CA certificate: embed as a constant or config file.
- Community AAGUID list (`github.com/passkeydeveloper/passkey-authenticator-aaguids`): optional JSON for platform authenticators with zero AAGUID.
