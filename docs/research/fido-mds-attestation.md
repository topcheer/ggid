# FIDO MDS 3.0 Blob Processing and Attestation Validation — Go Implementation Guide

> **Scope:** Go implementation of MDS 3.0 blob fetch/parse/verify, root CA management,
> attestation cert chain validation, and AAGUID lookup. For policy see
> [fido-metadata-service.md](./fido-metadata-service.md); for attestation format
> theory see [webauthn-attestation-chain.md](./webauthn-attestation-chain.md).

## 1. Overview

Step-by-step Go implementation for integrating FIDO MDS 3.0 into GGID's
WebAuthn registration flow.

**Pipeline:** blob download (JWS JWT) → verify x5c to FIDO root → verify JWT sig
→ parse payload → build AAGUID map → lookup at registration → validate cert chain
→ check status reports for `REVOKED`.

**Goal:** GGID resolves any authenticator by AAGUID and validates its attestation
cert chain to the vendor root CA.

## 2. MDS Blob Fetch and Verify

- **Endpoint:** `https://fidoalliance.co/metadata/v3.0/blob` (configurable via env)
- **Returns:** JWS-signed JWT containing `MetadataBLOBPayload` JSON
- **Refresh:** check `nextUpdate` field; refresh when stale (weekly)

### JWT Verification (5 steps)

1. Download blob bytes via HTTP GET.
2. Split JWT into header/payload/signature; decode header to extract `x5c`.
3. Verify x5c chain terminates at the pinned FIDO Alliance root CA.
4. Verify JWT signature using leaf cert public key.
5. Parse payload JSON into `MetadataBLOBPayload`.

### FIDO Alliance Root CA

PEM-encoded self-signed X.509 root (RSA 4096-bit). Pin at build time under
`services/auth/internal/webauthn/certs/fido_root_ca.pem`.

### Go: FIDOMDSClient

```go
package webauthn

type FIDOMDSClient struct {
    rootCAs    *x509.CertPool     // pinned FIDO root CA
    blobURL    string             // default blob endpoint
    httpClient *http.Client       // 30s timeout
    cache      *MetadataCache
}

type MetadataCache struct {
    payload   *MetadataBLOBPayload
    aaguidMap map[string]*MetadataStatement
    expiresAt time.Time
}

func (c *FIDOMDSClient) FetchBlob(ctx context.Context) (*MetadataBLOBPayload, error) {
    // 1. Download
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.blobURL, nil)
    resp, err := c.httpClient.Do(req)
    if err != nil { return nil, fmt.Errorf("fetch blob: %w", err) }
    defer resp.Body.Close()
    blobBytes, _ := io.ReadAll(resp.Body)

    // 2. Split JWT + decode header
    parts := strings.Split(strings.TrimSpace(string(blobBytes)), ".")
    headerRaw, _ := base64.RawURLEncoding.DecodeString(parts[0])
    var header map[string]any
    json.Unmarshal(headerRaw, &header)

    // 3. Extract x5c + verify chain to FIDO root
    x5c := header["x5c"].([]interface{})
    leafDER, _ := base64.StdEncoding.DecodeString(x5c[0].(string))
    leafCert, _ := x509.ParseCertificate(leafDER)

    intermediates := x509.NewCertPool()
    for i := 1; i < len(x5c); i++ {
        if der, err := base64.StdEncoding.DecodeString(x5c[i].(string)); err == nil {
            if ic, _ := x509.ParseCertificate(der); ic != nil { intermediates.AddCert(ic) }
        }
    }
    _, err = leafCert.Verify(x509.VerifyOptions{
        Roots: c.rootCAs, Intermediates: intermediates,
        KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
    })
    if err != nil { return nil, fmt.Errorf("x5c chain failed: %w", err) }

    // 4. Verify JWT signature
    signed := parts[0] + "." + parts[1]
    sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
    if err := leafCert.CheckSignature(leafCert.SignatureAlgorithm, []byte(signed), sig); err != nil {
        return nil, fmt.Errorf("JWT sig failed: %w", err)
    }

    // 5. Parse payload + build cache
    payloadRaw, _ := base64.RawURLEncoding.DecodeString(parts[1])
    var payload MetadataBLOBPayload
    json.Unmarshal(payloadRaw, &payload)

    m := make(map[string]*MetadataStatement, len(payload.Entries))
    for i := range payload.Entries {
        m[payload.Entries[i].AAGUID] = &payload.Entries[i].MetadataStatement
    }
    nextUpdate, _ := time.Parse("2006-01-02", payload.NextUpdate)
    c.cache = &MetadataCache{payload: &payload, aaguidMap: m, expiresAt: nextUpdate.Add(24 * time.Hour)}
    return &payload, nil
}

// LoadRootCA pins the FIDO Alliance root from PEM bytes.
func LoadRootCA(pemBytes []byte) (*x509.CertPool, error) {
    pool := x509.NewCertPool()
    block, _ := pem.Decode(pemBytes)
    cert, err := x509.ParseCertificate(block.Bytes)
    if err != nil { return nil, err }
    pool.AddCert(cert)
    return pool, nil
}
```

## 3. Metadata BLOB Payload Structures

```go
type MetadataBLOBPayload struct {
    LegalHeader string              `json:"legalHeader"`
    NextUpdate  string              `json:"nextUpdate"`
    No          int                 `json:"no"`
    Entries     []MetadataBLOBEntry `json:"entries"`
}

type MetadataBLOBEntry struct {
    AAGUID                 string            `json:"aaguid"`
    MetadataStatement      MetadataStatement `json:"metadataStatement"`
    StatusReports          []StatusReport    `json:"statusReports"`
    TimeOfLastStatusChange string            `json:"timeOfLastStatusChange"`
}

type MetadataStatement struct {
    Description                 string   `json:"description"`
    ProtocolFamily              string   `json:"protocolFamily"`
    AuthenticationAlgorithms    []string `json:"authenticationAlgorithms"`
    AttestationTypes            []string `json:"attestationTypes"`
    KeyProtection               []string `json:"keyProtection"`
    AttestationRootCertificates []string `json:"attestationRootCertificates"` // PEM vendor roots
}

type StatusReport struct {
    Status        string `json:"status"`
    EffectiveDate string `json:"effectiveDate"`
    Certificate   string `json:"certificate,omitempty"`
    URL           string `json:"url,omitempty"`
}

type DeviceInfo struct {
    Name             string
    AAGUID           string
    ProtocolFamily   string
    Algorithms       []string
    KeyProtection    []string
    AttestationRoots []string
}
```

## 4. AAGUID Device Lookup Table

Hardcoded fallback for environments without MDS connectivity:

| AAGUID | Device | Algorithm | Key Protection |
|--------|--------|-----------|----------------|
| `cb69481e-8ff7-4039-93ec-0a2729a154a8` | YubiKey 5 NFC | secp256r1, eddsa | hardware |
| `73bb0cd4-e502-49b8-9c4f-b7122cb1b3ac` | YubiKey 5 NFC (alt fw) | secp256r1 | hardware |
| `08987058-cadc-49b2-ab47-ccb70c95c185` | YubiKey 5Ci | secp256r1 | hardware |
| `34f5766d-1536-4a24-9033-9c1ca3a0a1c9` | YubiKey 5 Series (USB-A/C) | secp256r1 | hardware |
| `dd4a7b44-ff2c-498e-9a72-78f3d7d9ee6c` | Google Titan | secp256r1 | hardware, secure_element |
| `8876631b-d5a3-4fee-9e85-71b00756b0e4` | Google Titan (Krypton) | secp256r1 | hardware |
| `de1e980d-9b50-4f0f-a682-193b2f9c490c` | Windows Hello | secp256r1 | software, tee |
| `fbfc3007-154e-4ecc-8c0b-6e0243cfc1b5` | Apple Passkey (iOS/macOS) | secp256r1, eddsa | secure_enclave |
| `adce0002-35bc-c60a-648b-0b25f1f05503` | Chrome on Android | secp256r1 | tee |
| `ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4` | Google Password Manager | secp256r1 | secure_element |
| `00000000-0000-0000-0000-000000000000` | Platform authenticator (generic) | varies | varies |
| `ffffffff-ffff-ffff-ffff-ffffffffffff` | AAGUID unset (U2F compat) | secp256r1 | varies |

```go
var fallbackAAGUIDs = map[string]DeviceInfo{
    "cb69481e-8ff7-4039-93ec-0a2729a154a8": {Name: "YubiKey 5 NFC", ProtocolFamily: "fido2",
        Algorithms: []string{"secp256r1", "eddsa"}, KeyProtection: []string{"hardware"}},
    "dd4a7b44-ff2c-498e-9a72-78f3d7d9ee6c": {Name: "Google Titan", ProtocolFamily: "fido2",
        Algorithms: []string{"secp256r1"}, KeyProtection: []string{"hardware", "secure_element"}},
    "fbfc3007-154e-4ecc-8c0b-a484-5d2f0fcb7756": {Name: "Apple Passkey", ProtocolFamily: "fido2",
        Algorithms: []string{"secp256r1", "eddsa"}, KeyProtection: []string{"secure_enclave"}},
    "de1e980d-9b50-4f0f-a682-193b2f9c490c": {Name: "Windows Hello", ProtocolFamily: "fido2",
        Algorithms: []string{"secp256r1"}, KeyProtection: []string{"software", "tee"}},
}

func (c *FIDOMDSClient) LookupDevice(aaguid string) (*DeviceInfo, error) {
    // 1. MDS cache
    if c.cache != nil {
        if stmt, ok := c.cache.aaguidMap[aaguid]; ok && stmt != nil {
            return &DeviceInfo{Name: stmt.Description, AAGUID: aaguid, ProtocolFamily: stmt.ProtocolFamily,
                Algorithms: stmt.AuthenticationAlgorithms, KeyProtection: stmt.KeyProtection,
                AttestationRoots: stmt.AttestationRootCertificates}, nil
        }
    }
    // 2. Fallback table
    if info, ok := fallbackAAGUIDs[aaguid]; ok { d := info; d.AAGUID = aaguid; return &d, nil }
    // 3. Allow all-zeros / empty
    if aaguid == "" || aaguid == "00000000-0000-0000-0000-000000000000" {
        return &DeviceInfo{Name: "Platform Authenticator", AAGUID: aaguid}, nil
    }
    return nil, fmt.Errorf("unknown AAGUID: %s", aaguid)
}
```

## 5. Attestation Certificate Chain Validation

At registration, the WebAuthn response includes an `x5c` chain (for `packed`, `tpm`,
`android-key`, `fido-u2f` formats). GGID validates this against MDS root CAs.

```go
var (
    ErrAuthenticatorRevoked = errors.New("authenticator revoked")
    ErrInvalidAttestation   = errors.New("attestation chain invalid")
)

func (c *FIDOMDSClient) VerifyAttestationChain(aaguid string, x5c [][]byte) (*x509.Certificate, error) {
    if len(x5c) == 0 { return nil, errors.New("empty x5c") }
    info, err := c.LookupDevice(aaguid)
    if err != nil { return nil, fmt.Errorf("unknown AAGUID %s", aaguid) }
    if c.IsRevoked(aaguid) { return nil, ErrAuthenticatorRevoked }

    // 3. Build root pool from MDS attestationRootCertificates
    roots := x509.NewCertPool()
    for _, pemCert := range info.AttestationRoots {
        roots.AppendCertsFromPEM([]byte(pemCert))
    }
    if len(info.AttestationRoots) == 0 { roots = c.rootCAs } // fallback to global FIDO root

    // 4. Parse leaf + intermediates
    leaf, err := x509.ParseCertificate(x5c[0])
    if err != nil { return nil, fmt.Errorf("parse leaf: %w", err) }
    intermediates := x509.NewCertPool()
    for _, der := range x5c[1:] {
        if ic, err := x509.ParseCertificate(der); err == nil { intermediates.AddCert(ic) }
    }

    // 5. Verify chain
    _, err = leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: intermediates,
        KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
    if err != nil { return nil, fmt.Errorf("%w: %v", ErrInvalidAttestation, err) }
    return leaf, nil
}

func (c *FIDOMDSClient) IsRevoked(aaguid string) bool {
    if c.cache == nil || c.cache.payload == nil { return false }
    critical := map[string]bool{"REVOKED": true, "USER_VERIFICATION_BYPASS": true,
        "ATTESTATION_KEY_COMPROMISE": true, "USER_KEY_REMOTE_COMPROMISE": true}
    for _, e := range c.cache.payload.Entries {
        if e.AAGUID == aaguid {
            for _, sr := range e.StatusReports { if critical[sr.Status] { return true } }
        }
    }
    return false
}
```

## 6. Status Report Monitoring

| Status | Severity | GGID Action |
|--------|----------|-------------|
| `FIDO_CERTIFIED` | Good | Allow registration |
| `NOT_FIDO_CERTIFIED` | Info | Allow, warn in console |
| `SELF_ASSERTION_SUBMITTED` | Low | Allow (vendor self-reported) |
| `USER_VERIFICATION_BYPASS` | **Critical** | Reject, alert admin |
| `ATTESTATION_KEY_COMPROMISE` | **Critical** | Reject, disable affected creds |
| `USER_KEY_REMOTE_COMPROMISE` | **Critical** | Reject, force re-registration |
| `REVOKED` | **Terminal** | Reject, disable all creds for AAGUID |

On blob refresh, GGID diffs old vs new status reports and emits admin alerts for
critical transitions. Revoked authenticators trigger credential disabling.

## 7. Integration with GGID WebAuthn Handler

GGID's `Handler` (`handler.go`) stores `AAGUID []byte` (line 37) and
`AttestationType string` (line 36). The go-webauthn library parses but does
**not** validate against MDS. Integration adds `mdsClient` + a hook in
`finishRegistration` after the `CreateCredential` call (line 535).

### Handler Changes

```go
type Handler struct {
    // ... existing fields (wbn, creds, sessions, origins, ...) ...
    mdsClient *FIDOMDSClient // NEW
    mdsConfig MDSConfig       // NEW
}

type MDSConfig struct {
    Enabled                 bool // global toggle
    RequireCertified        bool // reject NOT_FIDO_CERTIFIED
    AllowUncertifiedAAGUIDs bool // allow AAGUIDs missing from MDS
}
```

### Registration Hook (after `h.wbn.CreateCredential`)

```go
if h.mdsConfig.Enabled && h.mdsClient != nil {
    aaguidStr := formatAAGUID(credential.Authenticator.AAGUID)
    if x5c := extractX5C(parsedResponse); len(x5c) > 0 {
        if _, err := h.mdsClient.VerifyAttestationChain(aaguidStr, x5c); err != nil {
            writeClassifiedError(w, http.StatusBadRequest,
                fmt.Errorf("attestation verification: %w", err))
            return
        }
    }
    if info, _ := h.mdsClient.LookupDevice(aaguidStr); info != nil {
        cred.DeviceName = info.Name
    }
}
```

### Background Blob Refresh

```go
func (h *Handler) StartMDSRefresh(ctx context.Context) {
    if h.mdsClient == nil { return }
    ticker := time.NewTicker(24 * time.Hour)
    go func() {
        h.mdsClient.FetchBlob(ctx)
        for { select {
            case <-ticker.C: h.mdsClient.FetchBlob(ctx)
            case <-ctx.Done(): ticker.Stop(); return
        }}
    }()
}
```

## 8. Data Model

### PostgreSQL

```sql
CREATE TABLE fido_mds_cache (
    id SERIAL PRIMARY KEY,
    blob_jwt TEXT NOT NULL,
    blob_hash VARCHAR(64) NOT NULL,       -- SHA-256
    next_update DATE NOT NULL,
    last_fetched TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE fido_metadata_statements (
    aaguid UUID PRIMARY KEY,
    description TEXT NOT NULL,
    protocol_family VARCHAR(16) DEFAULT 'fido2',
    status VARCHAR(64),
    json_statement JSONB NOT NULL,
    last_updated TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE user_authenticators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    credential_id BYTEA NOT NULL,
    aaguid UUID,
    device_name TEXT,
    backup_eligible BOOLEAN DEFAULT false,
    backup_state BOOLEAN DEFAULT false,
    attestation_type VARCHAR(32),
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(tenant_id, credential_id)
);
CREATE INDEX idx_user_authenticators_aaguid ON user_authenticators(aaguid);
```

### Redis

```
fido:mds:aaguid:{aaguid}  → JSON MetadataStatement  (TTL: 24h)
fido:mds:status:{aaguid}  → latest status string     (TTL: 24h)
fido:mds:blob:hash        → SHA-256 of last blob
```

## 9. Implementation Roadmap

| Step | Task | Days | Priority |
|------|------|------|----------|
| 1 | `FIDOMDSClient` blob fetch + JWT/x5c verify | 2 | P2 |
| 2 | Payload parsing + AAGUID map + fallback | 1 | P2 |
| 3 | Attestation cert chain validation | 2 | P1 enterprise |
| 4 | WebAuthn handler integration | 1 | P1 enterprise |
| 5 | Background refresh + status monitoring | 1 | P2 |
| 6 | PostgreSQL + Redis persistence | 1 | P2 |
| **Total** | | **~8** | |

Consumer (attestation `"none"`) = P2. Enterprise (hardware assurance) = P1.

## References

- [FIDO MDS 3.0 Spec](https://fidoalliance.org/specs/fido-v2.1-rd-20210309/fido-metadata-service-v3.0-rd-20210309.html)
- GGID: [fido-metadata-service.md](./fido-metadata-service.md), [webauthn-attestation-chain.md](./webauthn-attestation-chain.md), `services/auth/internal/webauthn/handler.go`
