# Decentralized Identity & Verifiable Credentials: Production Implementation Guide for GGID

> **Focus**: Upgrading GGID's existing in-memory VC/DID infrastructure (VCIssuer, DIDResolver, SD-JWT) into a production-grade decentralized identity system — DB-backed credential issuance, revocation status lists, DID method resolution with caching, OID4VCI/OID4VP flows, and EU Digital Identity Wallet compatibility.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§11), curl commands (§7).
>
> **Related**: `oid4vci-and-verifiable-credentials.md` (372 lines), `oid4vp-and-credential-presentation.md` (399 lines), `eu-digital-identity-wallet.md` (3734 lines), `credential-schemas-and-exchange.md` (402 lines).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: In-Memory VC/DID](#2-ggid-current-state-in-memory-vcdid)
3. [Gap Analysis](#3-gap-analysis)
4. [Proposed Architecture](#4-proposed-architecture)
5. [Credential Lifecycle](#5-credential-lifecycle)
6. [Endpoint Precondition Check](#6-endpoint-precondition-check)
7. [API Design + Curl Commands](#7-api-design--curl-commands)
8. [Database Schema](#8-database-schema)
9. [DID Method Support](#9-did-method-support)
10. [Revocation Status Lists](#10-revocation-status-lists)
11. [Implementation Backlog with DoD](#11-implementation-backlog-with-dod)
12. [Competitive Differentiation](#12-competitive-differentiation)
13. [Security Considerations](#13-security-considerations)

---

## 1. Executive Summary

Decentralized identity (DID) and Verifiable Credentials (VC) represent the next evolution of digital identity — user-controlled, cryptographically verifiable, and platform-independent. The EU's Digital Identity Wallet regulation (eIDAS 2.0) mandates Member States to issue VCs by 2026, creating urgent enterprise demand.

GGID has foundational VC/DID infrastructure:
- **VCIssuer** (`identity/service/vc_issuer.go:35`) — Ed25519-signed credential issuance, but **in-memory** (`sync.RWMutex` + 4 maps)
- **DIDResolver** (`identity/service/did_resolver.go:35`) — Resolves did:web, did:key, did:ion, but **in-memory cache** (`sync.RWMutex` + map)
- **SD-JWT** (`identity/server/sdjwt_handler.go:86`) — Selective disclosure JWT issue/verify ✅ (HMAC-based, needs upgrade to asymmetric)
- **VC handler** (`identity/server/vc_handler.go`) — VC API endpoints
- **DID handler** (`identity/server/did_handler.go`) — DID resolution endpoint

Critical issues:
1. **VCIssuer fully in-memory** — Credentials, revocations, and keys lost on restart
2. **DIDResolver cache in-memory** — Resolution results lost on restart
3. **No DB-backed credential store** — Issued VCs not persisted
4. **No status list (RFC 9114)** — Revocation tracking is a map, not a bitmap status list
5. **SD-JWT uses HMAC** — Should use ES256/EdDSA (asymmetric) for third-party verification
6. **No OID4VCI flow** — Credential issuance protocol not implemented
7. **No OID4VP flow** — Credential presentation/verification not implemented
8. **No DID:web HTTPS resolution** — did:web resolver stubbed, no real HTTPS fetch
9. **No credential schema registry** — No schema definitions for credential types
10. **No wallet integration** — No credential wallet endpoints

**Recommendation**: Upgrade to a production-grade VC/DID system with DB-backed credential store, status list revocation (RFC 9114), asymmetric SD-JWT, OID4VCI/OID4VP flows, and real DID method resolution.

**Estimated effort**: 4 sprints for MVP (DB + revocation + OID4VCI + OID4VP).

---

## 2. GGID Current State: In-Memory VC/DID

### Existing Components

| Component | File:Line | Status | Issue |
|-----------|-----------|--------|-------|
| VCIssuer | `identity/service/vc_issuer.go:35` | **In-memory** ❌ | `sync.RWMutex` + 4 maps (issued, revoked, keys, pubKeys) |
| IssueVC | `vc_issuer.go:68` | **Works** | Issues Ed25519-signed VC |
| SignVC | `vc_issuer.go:89` | **Works** | Ed25519 proof attachment |
| VerifyVC | `vc_issuer.go:120` | **Works** | Signature verification |
| ListIssuedVCs | `vc_issuer.go:151` | **In-memory** | Scans map for issuerDID |
| RevokeVC | `vc_issuer.go` | **In-memory** | Sets revoked[did]=true in map |
| DIDResolver | `identity/service/did_resolver.go:35` | **In-memory cache** ❌ | `sync.RWMutex` + `map[string]cachedDID` |
| ResolveDID | `did_resolver.go:50` | **Works** | Routes by method (web/key/ion) |
| resolveDIDWeb | `did_resolver.go:86` | **Stub** | No real HTTPS fetch |
| resolveDIDKey | `did_resolver.go:112` | **Works** | Cryptographic resolution |
| resolveDIDIon | `did_resolver.go:122` | **Stub** | No sidetree resolution |
| SD-JWT issue | `identity/server/sdjwt_handler.go:86` | **Works** ⚠️ | HMAC-SHA256 (should be asymmetric) |
| SD-JWT verify | `sdjwt_handler.go:174` | **Works** ⚠️ | HMAC verification |
| VC handler | `identity/server/vc_handler.go` | **Works** | VC CRUD API |
| DID handler | `identity/server/did_handler.go:13` | **Works** | DID resolution endpoint |
| RAR credential type | `oauth/rar_handler.go:196` | **Works** | "Issue Credential" in consent |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **In-memory VCIssuer** | Credentials, keys, revocations lost on restart |
| 2 | **In-memory DIDResolver** | Resolution cache lost on restart |
| 3 | **No status list (RFC 9114)** | Revocation doesn't scale; no bitmap |
| 4 | **SD-JWT uses HMAC** | Can't be verified by third parties |
| 5 | **No OID4VCI** | Can't issue credentials via standard protocol |
| 6 | **No OID4VP** | Can't verify credential presentations |
| 7 | **did:web stub** | No real HTTPS resolution |
| 8 | **No credential schema** | No type definitions for VC schemas |
| 9 | **No wallet endpoints** | No credential wallet API |
| 10 | **No trust framework** | No verifier trust registry |

---

## 3. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "Issue a 'UniversityDegree' credential to Alice" | In-memory, lost on restart | DB-backed credential + status list |
| 2 | "Verify Alice's credential at a third party" | HMAC can't be externally verified | Asymmetric (EdDSA/ES256) proof |
| 3 | "Revoke a credential" | Map entry, not discoverable | Status list (RFC 9114) bitmap |
| 4 | "EU wallet requests credential via OID4VCI" | No endpoint | OID4VCI credential endpoint |
| 5 | "Verifier requests credential presentation" | No endpoint | OID4VP authorization request |
| 6 | "Resolve did:web:corp.com" | Stub returns empty | HTTPS fetch + DID document parse |
| 7 | "Define schema for 'EmployeeID' credential" | No schema registry | Schema CRUD + validation |

---

## 4. Proposed Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │     Decentralized Identity Layer             │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Credential Store (PostgreSQL)        │    │
                    │  │  - verifiable_credentials table       │    │
                    │  │  - credential_schemas table           │    │
                    │  │  - credential_status_lists table      │    │
                    │  │  - did_documents table (cache)        │    │
                    │  │  - issuer_key_pairs table             │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  VC Issuer Service                    │    │
                    │  │  - DB-backed credential issuance      │    │
                    │  │  - Ed25519 / ES256 proof              │    │
                    │  │  - Schema validation                  │    │
                    │  │  - Status list revocation (RFC 9114)  │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────┐  ┌─────────────────┐   │
                    │  │  DID Resolver    │  │  SD-JWT Engine  │   │
                    │  │  (DB cache +     │  │  (asymmetric:   │   │
                    │  │   HTTPS fetch    │  │   EdDSA/ES256)  │   │
                    │  │   for did:web)   │  │                 │   │
                    │  └──────────────────┘  └─────────────────┘   │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  OID4VCI / OID4VP Endpoints           │    │
                    │  │                                      │    │
                    │  │  OID4VCI (issuance):                 │    │
                    │  │  ├── /.well-known/oauth-authorization-│    │
                    │  │  │   server                           │    │
                    │  │  ├── /credential-issuer-metadata      │    │
                    │  │  ├── /credential-offer                │    │
                    │  │  └── /credential                     │    │
                    │  │                                      │    │
                    │  │  OID4VP (presentation):              │    │
                    │  │  ├── /verify-presentation             │    │
                    │  │  └── /presentation-request            │    │
                    │  └──────────────────────────────────────┘    │
                    └──────────────────────────────────────────────┘
```

---

## 5. Credential Lifecycle

```
Schema Definition → Credential Issuance → Verification → Revocation
                                                              │
                    ┌─────────────────────────────────────────┘
                    │
                    ▼
1. Admin defines schema:
   POST /api/v1/vc/schemas { name: "UniversityDegree", fields: [...] }

2. Issuer issues credential:
   POST /api/v1/vc/issue { schema: "UniversityDegree", subject: "did:key:z...", claims: {...} }
   → Ed25519-signed VC stored in DB + assigned status list index

3. Holder stores credential in wallet

4. Verifier requests presentation:
   POST /api/v1/vc/verify-presentation { presentation: "{...}" }
   → Verifies proof + checks status list (not revoked) + checks schema

5. Issuer revokes credential:
   POST /api/v1/vc/{id}/revoke
   → Updates status list bitmap (index → revoked)

6. Verifier re-checks: fetches status list → credential now shows revoked
```

---

## 6. Endpoint Precondition Check

### Existing Endpoints (Upgrade)

| Endpoint | File:Line | Current | Target |
|----------|-----------|---------|--------|
| VC handler | `identity/server/vc_handler.go` | **Works** | DB-backed |
| DID resolution | `identity/server/did_handler.go:13` | **Works** | DB cache + HTTPS |
| SD-JWT issue | `identity/server/sdjwt_handler.go:86` | **HMAC** | Asymmetric |
| SD-JWT verify | `sdjwt_handler.go:174` | **HMAC** | Asymmetric |
| VC service | `identity/service/vc_issuer.go:35` | **In-memory** | DB-backed |
| DID service | `identity/service/did_resolver.go:35` | **In-memory** | DB cache |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/vc/schemas` | POST | Register credential schema | P0 |
| `/api/v1/vc/schemas` | GET | List schemas | P0 |
| `/api/v1/vc/schemas/{id}` | GET | Get schema definition | P0 |
| `/api/v1/vc/issue` | POST | Issue a credential (DB-backed) | P0 |
| `/api/v1/vc/{id}` | GET | Get credential by ID | P0 |
| `/api/v1/vc/{id}/revoke` | POST | Revoke credential | P0 |
| `/api/v1/vc/status-list/{issuer}` | GET | Get revocation status list | P0 |
| `/api/v1/vc/verify` | POST | Verify a credential | P0 |
| `/api/v1/vc/batch-verify` | POST | Batch verify credentials | P1 |
| `/.well-known/oauth-authorization-server` | GET | OID4VCI issuer metadata | P1 |
| `/credential-offer` | POST | OID4VCI credential offer | P1 |
| `/credential` | POST | OID4VCI credential endpoint | P1 |
| `/api/v1/vc/verify-presentation` | POST | OID4VP presentation verification | P1 |

---

## 7. API Design + Curl Commands

### Register Credential Schema

```bash
curl -X POST https://ggid.corp.com/api/v1/vc/schemas \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "UniversityDegree",
    "version": "1.0",
    "description": "Academic degree credential",
    "fields": [
      { "name": "degree", "type": "string", "required": true },
      { "name": "institution", "type": "string", "required": true },
      { "name": "graduationDate", "type": "date", "required": true },
      { "name": "gpa", "type": "number", "required": false }
    ]
  }'

# Response:
{ "schema_id": "sch_7f3a...", "name": "UniversityDegree", "version": "1.0", "status": "active" }
```

### Issue Credential (DB-backed)

```bash
curl -X POST https://ggid.corp.com/api/v1/vc/issue \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "schema_name": "UniversityDegree",
    "issuer_did": "did:web:university.edu",
    "subject_did": "did:key:z6Mk...",
    "claims": {
      "degree": "MS Computer Science",
      "institution": "Stanford University",
      "graduationDate": "2026-06-15",
      "gpa": 3.8
    },
    "expires_at": "2031-06-15T00:00:00Z"
  }'

# Response:
{
  "credential_id": "vc_9e8f7g6h-...",
  "credential": {
    "@context": ["https://www.w3.org/2018/credentials/v1"],
    "id": "https://ggid.corp.com/vc/vc_9e8f7g6h",
    "type": ["VerifiableCredential", "UniversityDegree"],
    "issuer": "did:web:university.edu",
    "credentialSubject": "did:key:z6Mk...",
    "issuanceDate": "2026-07-17T10:00:00Z",
    "expirationDate": "2031-06-15T00:00:00Z",
    "claims": { "degree": "MS CS", "institution": "Stanford", ... },
    "proof": {
      "type": "Ed25519Signature2020",
      "verificationMethod": "did:web:university.edu#key-1",
      "proofValue": "z7D1...",
      "proofPurpose": "assertionMethod"
    },
    "credentialStatus": {
      "type": "StatusList2021Entry",
      "statusListCredential": "https://ggid.corp.com/vc/status-list/did:web:university.edu",
      "statusListIndex": 42
    }
  }
}
```

### Verify Credential

```bash
curl -X POST https://ggid.corp.com/api/v1/vc/verify \
  -H "Content-Type: application/json" \
  -d '{ "credential": "{...VC JSON...}" }'

# Response:
{
  "valid": true,
  "checks": {
    "signature_valid": true,
    "not_expired": true,
    "not_revoked": true,
    "schema_valid": true
  },
  "verified_at": "2026-07-17T10:05:00Z"
}
```

### Revoke Credential

```bash
curl -X POST https://ggid.corp.com/api/v1/vc/vc_9e8f7g6h/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"reason": "credential_misuse"}'

# Response:
{ "status": "revoked", "revoked_at": "2026-07-17T11:00:00Z", "status_list_index": 42 }
```

### Status List (RFC 9114 / StatusList2021)

```bash
curl https://ggid.corp.com/api/v1/vc/status-list/did:web:university.edu

# Response (GZIP-compressed bitmap):
{
  "@context": ["https://w3id.org/vc/status-list/2021/v1"],
  "type": ["VerifiableCredential", "StatusList2021Credential"],
  "credentialSubject": {
    "type": "StatusList2021",
    "statusPurpose": "revocation",
    "encodedList": "H4sIAAAAAAAAA... (GZIP base64)"
  }
}
```

---

## 8. Database Schema

```sql
-- Credential schemas (type definitions)
CREATE TABLE vc_schemas (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    version             VARCHAR(16) NOT NULL DEFAULT '1.0',
    description         TEXT,
    fields              JSONB NOT NULL,               -- [{name, type, required}]
    json_schema         JSONB,                        -- Full JSON Schema for validation
    status              VARCHAR(16) DEFAULT 'active',
    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name, version)
);

-- Issuer key pairs (DB-backed, replaces in-memory maps)
CREATE TABLE vc_issuer_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    issuer_did          VARCHAR(256) NOT NULL,
    key_id              VARCHAR(128) NOT NULL,        -- #key-1, #key-2
    algorithm           VARCHAR(32) DEFAULT 'Ed25519',
    public_key_jwk      JSONB NOT NULL,               -- {kty, crv, x}
    private_key_enc     BYTEA NOT NULL,               -- Encrypted with tenant key
    status              VARCHAR(16) DEFAULT 'active', -- active, rotated, revoked
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at          TIMESTAMPTZ,
    UNIQUE(tenant_id, issuer_did, key_id)
);

-- Verifiable credentials (issued)
CREATE TABLE verifiable_credentials (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    schema_id           UUID REFERENCES vc_schemas(id),
    
    issuer_did          VARCHAR(256) NOT NULL,
    subject_did         VARCHAR(256) NOT NULL,
    credential_type     VARCHAR(128) NOT NULL,
    
    -- Full VC JSON (W3C compliant)
    credential_json     JSONB NOT NULL,
    
    -- Proof
    proof_type          VARCHAR(64) DEFAULT 'Ed25519Signature2020',
    key_id              VARCHAR(128),                 -- Key used for signing
    
    -- Status
    status              VARCHAR(16) NOT NULL DEFAULT 'active',
    -- 'active', 'revoked', 'expired', 'suspended'
    status_list_index   INT,                          -- Index in status bitmap
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    
    -- Lifecycle
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,
    
    -- Audit
    issued_by           UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Status lists (RFC 9114 / StatusList2021)
CREATE TABLE vc_status_lists (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    issuer_did          VARCHAR(256) NOT NULL,
    status_purpose      VARCHAR(32) DEFAULT 'revocation',
    
    -- Bitmap (GZIP-compressed base64)
    encoded_list        TEXT NOT NULL,
    size                INT NOT NULL DEFAULT 131072,  -- 16KB bitmap = 131072 credentials
    
    -- Last assigned index
    last_index          INT NOT NULL DEFAULT 0,
    
    -- Metadata
    credential_id       UUID,                         -- VC wrapping this status list
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, issuer_did, status_purpose)
);

-- DID documents (resolution cache — replaces in-memory)
CREATE TABLE did_documents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    did                 VARCHAR(512) NOT NULL,
    method              VARCHAR(32) NOT NULL,         -- 'web', 'key', 'ion', 'pkh'
    document_json       JSONB NOT NULL,
    
    -- Resolution source
    resolved_from       VARCHAR(32),                  -- 'https', 'dns', 'cache'
    resolution_error    TEXT,
    
    expires_at          TIMESTAMPTZ,                  -- Cache TTL
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, did)
);

-- Verification records (audit trail of credential verifications)
CREATE TABLE vc_verification_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    credential_id       VARCHAR(256),
    verifier            VARCHAR(256),                 -- Who verified
    result              VARCHAR(16) NOT NULL,         -- 'valid', 'invalid', 'revoked', 'expired'
    checks              JSONB,                        -- {signature, expiry, revocation, schema}
    error_detail        TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_vc_schemas_tenant ON vc_schemas (tenant_id, name, version);
CREATE INDEX idx_vc_tenant_issuer ON verifiable_credentials (tenant_id, issuer_did, status);
CREATE INDEX idx_vc_tenant_subject ON verifiable_credentials (tenant_id, subject_did, status);
CREATE INDEX idx_vc_status ON verifiable_credentials (tenant_id, status);
CREATE INDEX idx_vc_expiry ON verifiable_credentials (expires_at) WHERE status = 'active';
CREATE INDEX vc_status_lists_issuer ON vc_status_lists (tenant_id, issuer_did);
CREATE INDEX idx_did_documents_did ON did_documents (tenant_id, did);
CREATE INDEX idx_vc_verify_log_time ON vc_verification_log (tenant_id, created_at DESC);
CREATE INDEX idx_issuer_keys_did ON vc_issuer_keys (tenant_id, issuer_did, status);
```

---

## 9. DID Method Support

| Method | Current | Target | Priority |
|--------|---------|--------|----------|
| **did:web** | Stub (`did_resolver.go:86`) | HTTPS fetch `https://{domain}/.well-known/did.json` | P0 |
| **did:key** | Works (`did_resolver.go:112`) | ✅ Keep | ✅ |
| **did:ion** | Stub (`did_resolver.go:122`) | Sidetree resolution via ION node | P2 |
| **did:pkh** | Not implemented | Blockchain address derivation | P3 |
| **did:cheqd** | Not implemented | Cheqd network resolution | P3 |
| **did:ebsi** | Not implemented | EU EBSI resolution | P1 (EU compliance) |

### did:web Resolution

```go
func (r *DIDResolver) resolveDIDWeb(ctx context.Context, suffix string) (*DIDDocument, error) {
    // suffix = "corp.com:user:alice"
    parts := strings.SplitN(suffix, ":", 2)
    domain := parts[0]
    path := ""
    if len(parts) > 1 {
        path = "/" + strings.Join(strings.Split(parts[1], ":"), "/")
    }
    
    url := fmt.Sprintf("https://%s%s/.well-known/did.json", domain, path)
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("did:web HTTPS fetch failed: %w", err)
    }
    defer resp.Body.Close()
    
    var doc DIDDocument
    if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
        return nil, fmt.Errorf("did:web parse failed: %w", err)
    }
    
    return &doc, nil
}
```

---

## 10. Revocation Status Lists

### StatusList2021 (RFC 9114) Implementation

```
How it works:
1. Issuer creates a 16KB bitmap (131,072 bits = 131,072 credential slots)
2. Each issued credential gets index N (0-131071)
3. Bit N = 0 → credential valid; Bit N = 1 → credential revoked
4. Bitmap GZIP-compressed → base64 encoded → published as VC
5. Verifier fetches status list VC, decompresses, checks bit N

Advantages:
- O(1) revocation check
- Privacy-preserving (can't tell which specific credential is revoked)
- Compact (16KB covers 131K credentials)
- Cacheable (list changes rarely)
```

### Bitmap Operations

```go
// Set bit at index (revoke)
func setStatusBit(bitmap []byte, index int) {
    byteIndex := index / 8
    bitIndex := index % 8
    bitmap[byteIndex] |= 1 << bitIndex
}

// Check bit at index (verify)
func isStatusSet(bitmap []byte, index int) bool {
    byteIndex := index / 8
    bitIndex := index % 8
    return bitmap[byteIndex]&(1<<bitIndex) != 0
}
```

---

## 11. Implementation Backlog with DoD

### P0 — DB-Backed VC/DID Core (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | VC/DID DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No in-memory map | 2d |
| 2 | DB-backed VCIssuer | ✅ Uses verifiable_credentials table ✅ No sync.RWMutex ✅ Ed25519 keys in DB ✅ ≥3 tests | 4d |
| 3 | DB-backed DIDResolver | ✅ Uses did_documents table for cache ✅ HTTPS fetch for did:web ✅ ≥3 tests | 3d |
| 4 | Credential schema registry | ✅ CRUD for schemas ✅ JSON Schema validation on issue ✅ ≥3 tests | 3d |
| 5 | Status list (StatusList2021) | ✅ Bitmap allocation + GZIP ✅ Revocation sets bit ✅ Verify checks bit ✅ ≥3 tests | 3d |
| 6 | Asymmetric SD-JWT | ✅ ES256/EdDSA (not HMAC) ✅ Third-party verifiable ✅ ≥3 tests | 2d |

### P1 — OID4VCI + OID4VP (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | OID4VCI issuer metadata | ✅ `/.well-known/oauth-authorization-server` ✅ `/.well-known/credential-issuer` ✅ ≥3 tests | 2d |
| 8 | OID4VCI credential offer + issuance | ✅ Credential offer URI ✅ Pre-authorized code flow ✅ DB-backed ✅ ≥3 tests | 4d |
| 9 | OID4VP presentation verification | ✅ Verify presentation proof ✅ Check status list ✅ Return verification result ✅ ≥3 tests | 3d |

### P2 — Console UI + did:ebsi (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 10 | Console credential manager | ✅ Issue/list/revoke VCs ✅ Schema editor ✅ Status list viewer ✅ ≥3 tests | 4d |
| 11 | did:ebsi resolution | ✅ EBSI DID registry resolution ✅ EU compliance ✅ ≥3 tests | 3d |
| 12 | Verification dashboard | ✅ Verification history ✅ Success/failure metrics ✅ DB-backed ✅ ≥3 tests | 2d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 13 | BBS+ signatures | Selective disclosure + zero-knowledge proofs |
| 14 | did:ion resolution | Sidetree protocol via ION node |
| 15 | Trust framework registry | Verifier trust registry for known issuers |
| 16 | Universal Resolver integration | External universal resolver fallback |
| 17 | Credential exchange protocol | DIDComm v2 credential exchange |
| 18 | EU Wallet compatibility test | eIDAS 2.0 compliance certification |

---

## 12. Competitive Differentiation

| Feature | GGID (target) | Microsoft Entra | Okta | Auth0 | Keycloak |
|---------|---------------|-----------------|------|-------|----------|
| **VC issuance** | **DB-backed Ed25519** | Entra Verified ID | No | No | Yes (plugin) |
| **DID methods** | **web+key+ebsi** | did:ion, did:web | No | No | did:web |
| **Status list** | **StatusList2021** | Yes | No | No | No |
| **SD-JWT** | **Asymmetric** | No | No | No | No |
| **OID4VCI** | **Yes** | Yes | No | No | Partial |
| **OID4VP** | **Yes** | Yes | No | No | No |
| **Schema registry** | **DB-backed** | Yes | No | No | No |
| **EU Wallet** | **eIDAS 2.0 compatible** | In progress | No | No | No |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | Yes |

**Key differentiator**: GGID would be the only open-source IAM with a complete VC/DID stack — issuance, verification, revocation status lists, OID4VCI/OID4VP, schema registry, and multiple DID methods — including EU eIDAS 2.0 compatibility.

---

## 13. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Private key compromise** | Keys encrypted at rest (tenant key); rotation support |
| **Credential forgery** | Ed25519 proof verification; issuer DID must match |
| **Revocation gap** | Status list checked at verification time; short cache TTL |
| **Privacy leak in status list** | StatusList2021 bitmap preserves privacy (bulk revoke) |
| **SD-JWT disclosure leak** | Selective disclosure: holder chooses which claims to reveal |
| **DID document spoofing** | did:web verified via HTTPS + TLS; did:key cryptographically derived |
| **Schema injection** | JSON Schema validation on all issued credentials |
| **Replay attack** | VC includes issuance date + expiry + nonce |

---

## References

- [W3C Verifiable Credentials Data Model 1.1](https://www.w3.org/TR/vc-data-model/) — VC specification
- [W3C DID Core 1.0](https://www.w3.org/TR/did-core/) — DID specification
- [Status List 2021](https://www.w3.org/community/reports/credentials/CG-FINAL-vc-status-list-2021-20230102/) — Revocation bitmap
- [RFC 9496: SD-JWT](https://datatracker.ietf.org/doc/html/rfc9496) — Selective disclosure JWT
- [OID4VCI Draft](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html) — Credential issuance
- [OID4VP Draft](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html) — Credential presentation
- [eIDAS 2.0 / EUDI Wallet](https://digital-strategy.ec.europa.eu/en/policies/eu-digital-identity-wallet) — EU regulation
- [Ed25519Signature2020](https://w3c-ccg.github.io/lds-ed25519-2020/) — Linked data proof
- [GGID VCIssuer](../services/identity/internal/service/vc_issuer.go) — In-memory issuer at line 35
- [GGID DIDResolver](../services/identity/internal/service/did_resolver.go) — In-memory resolver at line 35
- [GGID SD-JWT Handler](../services/identity/internal/server/sdjwt_handler.go) — HMAC SD-JWT at line 86
- [GGID VC Handler](../services/identity/internal/server/vc_handler.go) — VC API endpoints
- [GGID DID Handler](../services/identity/internal/server/did_handler.go) — DID resolution endpoint at line 13
- [GGID OID4VCI Research](./oid4vci-and-verifiable-credentials.md) — Issuance protocol analysis
- [GGID OID4VP Research](./oid4vp-and-credential-presentation.md) — Presentation protocol analysis
- [GGID EU Digital Identity Wallet](./eu-digital-identity-wallet.md) — eIDAS 2.0 analysis (3734 lines)
- [GGID Credential Schemas](./credential-schemas-and-exchange.md) — Schema exchange analysis
