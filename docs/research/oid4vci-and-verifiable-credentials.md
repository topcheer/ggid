# OID4VCI and Verifiable Credentials

> Research note for GGID IAM. All spec references as of 2024–2025.

## 1. Overview

**Verifiable Credentials (VC)** — W3C standard (VC Data Model v2.0, 2024) for
tamper-evident digital credentials: diplomas, licenses, certifications, badges.
The problem: physical credentials are forgeable and hard to verify remotely.
VCs give the **holder** a self-contained, signed credential any **verifier**
can validate offline.

**OID4VCI** — IETF/OAuth WG protocol (draft-11, proposed final) for *issuing*
VCs via standard OAuth 2.1 / OIDC flows. Defines credential offers, the
credential endpoint, and proof-of-possession binding.

**Related specs (drafts as of 2024–2025):**

| Spec | Purpose |
|------|---------|
| OID4VCI | Issue VCs to a wallet via OAuth |
| OID4VP (draft-ietf-openid4vp) | Verifier requests VCs from wallet |
| SIOPv2 (draft-ietf-suites-siopv2) | Wallet as self-issued OpenID Provider |
| Token Status List (draft-ietf-oauth-status-list-21) | Compact revocation bitstring |
| SD-JWT VC (draft-ietf-oauth-sd-jwt-vc) | Selective-disclosure JWT-VC format |

```
 Issuer            Holder (Wallet)          Verifier
   |  credential offer ->|                     |
   |  <-- token request -|                     |
   |  -- access token -->|                     |
   |  <-- credential ----|                     |
   |                     |  -- presentation -->|  (offline verify)
```

## 2. VC Data Model (W3C v2.0)

**Core concepts:** Issuer (creates/signs), Subject/Holder (credential is about,
stored in wallet), Credential (claims + proof), Presentation (subset shown to
verifier), Verifier (validates).

### Credential JSON (LD-Proof format)

```json
{
  "@context": ["https://www.w3.org/ns/credentials/v2"],
  "id": "urn:uuid:f1c4a7e0-1234-5678-9abc-def012345678",
  "type": ["VerifiableCredential", "UniversityDegree"],
  "issuer": { "id": "did:web:university.example", "name": "Example University" },
  "issuanceDate": "2025-01-15T10:00:00Z",
  "credentialSubject": {
    "id": "did:key:z6MkpTHR8VNsBxYAA3...",
    "degree": { "type": "BachelorDegree", "name": "BSc Computer Science" }
  },
  "proof": {
    "type": "Ed25519Signature2020",
    "created": "2025-01-15T10:00:05Z",
    "verificationMethod": "did:web:university.example#key-1",
    "proofPurpose": "assertionMethod",
    "proofValue": "z58DAdFto9oqpHDJ5...="
  }
}
```

### Proof types

| Format | Mechanism | Selective Disclosure |
|--------|-----------|---------------------|
| LD-Proofs (Ed25519Signature2020) | JSON-LD + detached signature | No |
| JWT-VC | JWS with `vc` claim | No |
| SD-JWT VC | Salted claims, reveal selected | Yes |

### Issuer metadata (`.well-known/openid-credential-issuer`)

```json
{
  "credential_issuer": "https://issuer.example",
  "credential_endpoint": "https://issuer.example/credential",
  "credentials_supported": [{
    "format": "jwt_vc_json",
    "cryptographic_binding_methods_supported": ["did:key", "jwk"],
    "cryptographic_suites_supported": ["ES256", "EdDSA"],
    "display": [{ "name": "Employee ID", "locale": "en" }]
  }]
}
```

## 3. OID4VCI Issuance Flows

### 3.1 Credential Offer

Issuer creates a credential offer — JSON telling the wallet what's available
and how to get it. Delivered via deep link (`openid-credential-offer://`),
QR code, or URL redirect.

```json
{
  "credential_issuer": "https://issuer.example",
  "credential_configuration_ids": ["UniversityDegree"],
  "grants": {
    "urn:ietf:params:oauth:grant-type:pre-authorized_code": {
      "pre-authorized_code": "SplxlOBeZQQYbYS6WxSbIA",
      "tx_code": { "input_mode": "numeric", "length": 4 }
    }
  }
}
```

### 3.2 Pre-Authorized Code Flow

Best for **in-person/kiosk** scenarios — issuer pre-creates the code.

```
Issuer            Wallet
  |  QR (offer)      |
  |----------------->|
  |  POST /token (pre-authorized_code)
  |<-----------------|
  |  access_token    |
  |----------------->|
  |  POST /credential|
  |<-----------------|
  |  credential      |
  |----------------->|
```

No extra user auth needed — the code *is* the authorization. User may need a
transaction PIN (`tx_code`).

### 3.3 Authorization Code Flow

Best for **online** issuance where user must prove identity first. Standard
OAuth auth code flow with `authorization_details` (RAR, RFC 9396):

```json
[{ "type": "openid_credential", "format": "jwt_vc_json",
   "types": ["VerifiableCredential", "UniversityDegree"] }]
```

```
Wallet -> redirect to /authorize -> user authenticates -> auth code
Wallet -> POST /token -> access_token
Wallet -> POST /credential -> credential
```

### 3.4 Deferred Issuance

Credential not ready immediately (e.g., background check pending). Issuer
returns `transaction_id`; wallet polls:

```json
// First attempt:
{ "transaction_id": "tx_abc123" }
// Poll (not ready):
{ "error": "issuance_pending", "interval": 30 }
// Eventually:
{ "credential": "<JWT-VC>" }
```

## 4. Credential Endpoint

`POST /credential` — wallet submits request with access token + proof of
key possession.

**Request:**
```http
POST /credential
Authorization: Bearer <access_token>
{ "format": "jwt_vc_json",
  "credential_definition": { "type": ["VerifiableCredential", "EmployeeID"] },
  "proof": { "proof_type": "jwt", "jwt": "eyJ0eXAiOiJvcGVuaWQ0dmNp..." } }
```

**Response:**
```json
{ "format": "jwt_vc_json",
  "credential": "eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9...",
  "c_nonce": "worrying-otter-72",
  "c_nonce_expires_in": 86400 }
```

The `proof` field proves the wallet controls the DID key the credential binds
to. `c_nonce` is the challenge the issuer provides — the wallet signs it with
its key.

## 5. Token Status List (Bitstring)

**draft-ietf-oauth-status-list-21** — compact bitstring for revocation status.
Supports configurable status size (1-bit = valid/revoked; 2-bit = add
suspended/pending).

### How it works

1. Each credential gets a sequential **index**.
2. Issuer publishes a bitstring; position `index` = that credential's status.
3. `0` = valid, `1` = revoked. Bitstring is GZIP-compressed + base64url.

```
Index:   0  1  2  3  4  5  6  7 ...
Bits:    0  0  1  0  0  0  0  0 ...   (credential #2 revoked)
```

131,072 bits (16 KB compressed) tracks 131K credentials.

### Credential references the list

```json
"status": { "status_list": { "uri": "https://issuer.example/status/1234", "idx": 42 } }
```

### Privacy: k-anonymity

Verifier fetches the **entire** bitstring — it cannot tell which index it's
checking without the credential. No per-credential callback to issuer (unlike
OCSP). Verifier caches the bitstring locally.

| Feature | CRL | OCSP | Status List |
|---------|-----|------|-------------|
| Size | KB–MB | Per-query | 16 KB / 131K creds |
| Privacy | Exposes serials | Exposes lookup target | k-anonymity |
| Issuer contact | None | Yes | None (cached) |

### Go verification (sketch)

```go
func CheckStatus(idx int, uri string) (bool, error) {
    resp, err := http.Get(uri)
    if err != nil { return false, err }
    defer resp.Body.Close()
    var sl struct{ Bits string `json:"bits"` }
    json.NewDecoder(resp.Body).Decode(&sl)
    raw, _ := base64.RawURLEncoding.DecodeString(sl.Bits)
    gr, _ := gzip.NewReader(bytes.NewReader(raw))
    bits, _ := io.ReadAll(gr)
    return bits[idx/8]&(1<<(7-idx%8)) == 0, nil // true = valid
}
```

## 6. OID4VP (Verifiable Presentations)

**draft-ietf-openid4vp** — verifier requests credential presentations from wallet.

### presentation_definition

```json
{
  "presentation_definition": {
    "id": "employee-verification",
    "input_descriptors": [{
      "id": "employee_id",
      "format": { "jwt_vc_json": { "alg": ["ES256"] } },
      "constraints": { "fields": [
        { "path": ["$.type"], "filter": { "const": "EmployeeID" } },
        { "path": ["$.credentialSubject.department"] }
      ]}
    }]
  }
}
```

### Selective disclosure (SD-JWT)

Holder reveals only what the verifier needs:

```
Full claims: { name, email, department, salary, DOB, ... }
                   ↓ selective disclosure
Presented:   { name, department }
```

Each claim is individually salted + hashed; verifier re-computes hashes to
prove integrity without seeing undisclosed claims.

### Digital Bouncer (offline verification)

```
Holder -> VP (signed by wallet key) -> Verifier
  Verifier: verify issuer signature on VC
            check status list (cached)
            check wallet PoP signature
  Verifier -> access granted
```

No contact with the issuer. **SIOPv2**: wallet acts as self-issued OP —
responds with ID token signed by its own DID key, using VCs as claims.

## 7. GGID Integration Possibility

### GGID as Credential Issuer (extend `services/oauth`)

| Component | Implementation |
|-----------|----------------|
| `POST /credential` | Validates access token + proof, returns JWT-VC |
| `GET /.well-known/openid-credential-issuer` | Issuer metadata |
| `POST /credential-offer` | Generates offer JSON, stores in DB |
| `GET /status/:listId` | Publishes compressed bitstring |
| Token endpoint | Support `pre-authorized_code` grant type |

### DB schema (PostgreSQL)

```sql
CREATE TABLE credential_configurations (
    id UUID PRIMARY KEY, tenant_id UUID NOT NULL,
    type TEXT NOT NULL, format TEXT DEFAULT 'jwt_vc_json',
    signing_key JSONB NOT NULL, display JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE TABLE issued_credentials (
    id UUID PRIMARY KEY, tenant_id UUID NOT NULL,
    subject_did TEXT NOT NULL,
    configuration_id UUID REFERENCES credential_configurations(id),
    credential_jwt TEXT NOT NULL, status_index INTEGER NOT NULL,
    issued_at TIMESTAMPTZ DEFAULT now(), revoked_at TIMESTAMPTZ
);
CREATE TABLE credential_status_lists (
    id UUID PRIMARY KEY, tenant_id UUID NOT NULL, list_uri TEXT NOT NULL,
    bits BYTEA NOT NULL, capacity INTEGER DEFAULT 131072,
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE TABLE credential_offers (
    id UUID PRIMARY KEY, tenant_id UUID NOT NULL,
    pre_authorized_code TEXT NOT NULL UNIQUE,
    configuration_id UUID REFERENCES credential_configurations(id),
    expires_at TIMESTAMPTZ, consumed BOOLEAN DEFAULT false
);
```

### GGID as Verifier

- **Gateway**: `POST /vc/verify` — accepts presentation, validates issuer
  signature, checks status list, maps claims to GGID user profile.
- **Auth service**: OID4VP flow — accept VP as auth factor (step-up), similar
  to social login.

### Use cases

| Use case | Credential type | GGID role |
|----------|----------------|-----------|
| Employee ID card | `EmployeeID` | Issues |
| Access badge | `AccessBadge` | Issues |
| Professional certification | `Certification` | Issues |
| External diploma | `UniversityDegree` | Verifies |
| Government mDL | `mDL` (ISO 18013-5) | Verifies |

## 8. Ecosystem and Adoption

| Initiative | Status |
|------------|--------|
| EBSI (EU) | Pilot wallets; EUDI mandated by eIDAS 2.0 |
| mDL (ISO 18013-5) | Apple/Google Wallet support; US states piloting |
| EUDI Wallet | EU states must certify by end of 2026 |
| IETF OAuth VC family | OID4VCI draft-11 (proposed final), OID4VP draft-20 |
| W3C VC DM v2.0 | W3C Recommendation (2024) |

**Issuers:** governments (Estonia, Germany), universities (MIT, EU pilots),
enterprises (Microsoft Entra Verified ID).

**Wallet fragmentation:** Apple/Google platform wallets, third-party apps
(esatus, Trinsic), open-source SDKs (Veramo, Aries). Interoperability remains
the key challenge.

**Timeline:** government mandates 2025–2026, enterprise pilots 2024–2025,
mainstream adoption dependent on wallet UX + standardization.

## References

- W3C VC DM v2.0: https://www.w3.org/TR/vc-data-model-2.0/
- OID4VCI: https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html
- OID4VP: https://datatracker.ietf.org/doc/draft-ietf-openid4vp/
- Token Status List: https://datetracker.ietf.org/doc/draft-ietf-oauth-status-list/
- SD-JWT VC: https://datatracker.ietf.org/doc/draft-ietf-oauth-sd-jwt-vc/
- SIOPv2: https://datatracker.ietf.org/doc/draft-ietf-suites-siopv2/
