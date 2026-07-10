# Credential Schemas and Exchange Formats

> **Scope:** This document covers the **credential format layer** — W3C VC Data Model 2.0,
> JSON-LD vs JWT-VC vs SD-JWT comparison, schema registry, DIDComm v2, and presentation
> exchange. Issuance and presentation *protocol* flows (OID4VCI/OID4VP) are documented
> separately in `oid4vci-credential-issuance.md` and `oid4vp-credential-presentation.md`.

---

## 1. Overview

Verifiable Credentials (VCs) require two layers: a **data format** that defines how claims
are structured and secured, and an **exchange protocol** that governs how credentials flow
between issuer, holder, and verifier.

Three format camps dominate the ecosystem:

| Camp | Standard Body | Proof Mechanism | Key Feature |
|------|--------------|-----------------|-------------|
| **JSON-LD** | W3C | Data Integrity (Ed25519, BBS+) | Semantic interoperability |
| **JWT-VC** | IETF / W3C | JWS (RS256, ES256, EdDSA) | Simplicity, wide tooling |
| **SD-JWT** | IETF | JWS + hashed disclosures | Selective disclosure |

The choice of format directly affects:
- **Verification complexity** — canonicalization vs simple signature check
- **Selective disclosure** — can the holder reveal only specific claims?
- **Schema validation** — is the credential structure machine-verifiable?
- **Interoperability** — which ecosystems and wallets accept the format?

Exchange protocols sit above the format layer:
- **OID4VCI / OID4VP** — HTTP-based, OAuth/OIDC stack (covered in existing docs)
- **DIDComm v2** — transport-agnostic, DID-based encrypted messaging
- **Presentation Exchange (DIF)** — declarative request/response for credentials

---

## 2. W3C VC Data Model 2.0

The W3C VC Data Model 2.0 became a formal Recommendation in 2024. It defines the abstract
structure of a verifiable credential independent of the encoding format.

### Core Structure

```json
{
  "@context": ["https://www.w3.org/ns/credentials/v2"],
  "id": "urn:uuid:f1c0a890-1234-5678-9abc-def012345678",
  "type": ["VerifiableCredential", "UniversityDegree"],
  "issuer": "did:web:university.edu",
  "validFrom": "2025-01-15T00:00:00Z",
  "validUntil": "2026-01-15T00:00:00Z",
  "name": "Master of Science Degree",
  "credentialSchema": { "id": "https://university.edu/schemas/degree-v1.json", "type": "JsonSchema" },
  "credentialSubject": { "id": "did:key:z6Mk...", "degree": "MS", "field": "Computer Science" },
  "proof": {
    "type": "DataIntegrityProof",
    "cryptosuite": "eddsa-rdfc-2022",
    "verificationMethod": "did:web:university.edu#key-1",
    "proofPurpose": "assertionMethod",
    "proofValue": "z3Vt...base58signature..."
  }
}
```

### What Changed in v2 (2024)

| v1.1 Property | v2.0 Property | Rationale |
|---------------|---------------|-----------|
| `issuanceDate` | `validFrom` | Supports future-dated credentials (e.g., memberships starting later) |
| `expirationDate` | `validUntil` | Clearer semantics, optional |
| `credentialStatus` | `status` | Generalized status property (supports multiple status list types) |
| *(none)* | `name`, `description` | Human-readable metadata for wallet display |
| *(single proof)* | `proof` (array) | Multiple proofs supported (e.g., BBS+ + Ed25519) |

**Batch credentials:** v2 introduced `VerifiableCredential` containers that can wrap multiple
sub-credentials in a single document, reducing issuance overhead for correlated credentials.

**Data Integrity proofs** replaced the old `JwtProof` as the canonical proof mechanism for
JSON-LD credentials. Cryptosuites include:
- `eddsa-rdfc-2022` — Ed25519 with RDF canonicalization
- `bbs-2023` — BBS+ signatures with selective disclosure and unlinkability

---

## 3. Format Comparison: JSON-LD vs JWT-VC vs SD-JWT

### JSON-LD (Linked Data)

- **Proof type:** DataIntegrityProof (Ed25519, BBS+, ECDSA)
- **Canonicalization:** RDF Dataset Canonicalization v1.0 (formerly URDNA2015)
- **Structure:** Native JSON-LD with `@context` for semantic vocabulary
- **Pros:** Semantic interoperability, linked-data graph model, BBS+ selective disclosure
- **Cons:** Complex canonicalization (implementation landmines), larger payloads, harder to
  debug, RDF processing overhead
- **Used by:** EBSI (European Blockchain Services Infrastructure), Blockcerts (academic),
  governments (Ontario, Iceland)

### JWT-VC (JSON Web Token VC)

- **Proof type:** JWS (RS256, ES256, EdDSA)
- **Structure:** Compact JWT with `vc` claim containing credential payload
- **Payload:** JWT with `vc` claim: `{"iss": "did:web:uni.edu", "vc": {"type": ["VerifiableCredential", "UniversityDegree"], "credentialSubject": {"degree": "MS"}}}`
- **Pros:** Simple, compact, reuses existing JWS/JWT infrastructure, trivial verification
- **Cons:** No selective disclosure, correlation risk (same JWT reused), schema is convention-based
- **Used by:** OpenID Foundation, Microsoft Entra Verified ID, enterprise deployments

### SD-JWT (Selective Disclosure JWT)

- **Proof type:** JWS + per-claim salted hashes (disclosures)
- **Structure:** SD-JWT with `_sd` array of hashed claims + separate disclosure artifacts
- **How it works:**
  1. **Issuance:** Each disclosable claim is individually hashed with a random salt
  2. **Holder stores:** SD-JWT + all disclosure values
  3. **Presentation:** Holder includes only disclosures for claims they want to reveal
  4. **Verification:** Verifier sees only disclosed claims but can verify integrity of all
- **Pros:** Selective disclosure without JSON-LD complexity, backward-compatible with JWT
  infrastructure, adopted by EUDI wallet pilots
- **Cons:** Issuer must decide disclosable claims at issuance time, slightly larger than plain
  JWT, newer standard with evolving tooling, key-binding (KB-JWT) adds complexity
- **Used by:** EU Digital Identity Wallet (EUDI ARF), newer OID4VCI implementations

### Comparison Table

| Feature | JSON-LD (Data Integrity) | JWT-VC | SD-JWT |
|---------|------------------------|--------|--------|
| **Proof mechanism** | Data Integrity (embedded) | JWS (envelope) | JWS + hashed disclosures |
| **Selective disclosure** | BBS+ cryptosuite only | None | Native (per-claim) |
| **Unlinkability** | BBS+ only | None | KB-JWT (some) |
| **Canonicalization** | RDF Dataset Canonicalization | None | None |
| **Payload size** | Largest (~2-5 KB) | Smallest (~0.5-1 KB) | Medium (~1-2 KB) |
| **Verification complexity** | High (rdf-canon + crypto) | Low (verify JWS) | Medium (hash recompute) |
| **Schema binding** | JSON-LD context + JsonSchema | `vc.type` convention | `vct` (type string) |
| **Standard body** | W3C | IETF / W3C | IETF |
| **Maturity** | Mature (v2 Recommendation) | Mature (widest tooling) | Growing (RFC 9493 area) |
| **Tooling** | json-ld, digitalbazaar | go-jose, jose4j | go-sdjwt, sd-jwt-kotlin |
| **Ecosystem** | EBSI, governments | Entra, OpenID4VCI | EUDI Wallet |

---

## 4. Schema Registry

### Purpose

Credential schemas define: required fields, data types, validation rules, and claim semantics.
A schema registry is a central (or distributed) place to publish and discover schemas. Each
credential references its schema URI in the `credentialSchema` property.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://university.edu/schemas/degree-v1.json",
  "title": "UniversityDegree",
  "type": "object",
  "properties": {
    "degree": { "type": "string", "enum": ["BS", "MS", "PhD"] },
    "field": { "type": "string", "minLength": 2 },
    "honors": { "type": "string" }
  },
  "required": ["degree", "field"]
}
```
```

### Schema Formats

- **JSON Schema (draft 2020-12):** Most common — field types, required properties, constraints.
- **JSON-LD context:** Semantic vocabulary mapping property names to URIs.
- **XSD:** Legacy (SAML/XML world), rare in modern VC ecosystems.

Example JSON Schema:

| Model | Operator | Example | Pros | Cons |
|-------|----------|---------|------|------|
| **Central** | Federation/government | EBSI schema registry | Trust anchor, discoverable | Single point of failure |
| **Distributed** | Each issuer | `did:web` domain schemas | No central authority | Discovery harder |
| **Hybrid** | Federated + DID | Schema at DID endpoint | Balance of both | More complex |

**Schema validation flow:** Verifier fetches schema from URI → validates credential structure
→ rejects if structure doesn't match before checking proof/signature.

### Go: SchemaRegistry Interface

```go
type SchemaRegistry interface {
    GetSchema(ctx context.Context, uri string) (*Schema, error)
    ValidateCredential(ctx context.Context, cred map[string]any, schema *Schema) error
    Publish(ctx context.Context, schema *Schema) error
    List(ctx context.Context, tenantID string) ([]*Schema, error)
}

type Schema struct {
    ID         string          `json:"id"`
    Type       string          `json:"type"`       // e.g., "UniversityDegree"
    Format     string          `json:"format"`     // jwt-vc, sd-jwt, json-ld
    JSONSchema json.RawMessage `json:"json_schema"`
    Version    string          `json:"version"`
    CreatedAt  time.Time       `json:"created_at"`
}
```

---

## 5. DIDComm v2

### What is DIDComm

DIDComm v2 is a protocol for **encrypted, authenticated, transport-agnostic messaging** using
Decentralized Identifiers (DIDs). Messages are encrypted to keys referenced in the recipient's
DID Document, providing end-to-end confidentiality without a central broker.

- **Built on:** DID Documents, JWE/JWS, forward secrecy
- **Transport-agnostic:** HTTP, WebSocket, Bluetooth, NFC — any medium
- **Routing:** Supports mediator/relay for asynchronous delivery

### Credential Exchange via DIDComm

- **Issue credential:** Issuer sends a DIDComm message (`issue-credential/3.0/offer-credential`)
  encrypted to the holder's DID key. Holder accepts; credential delivered and stored in wallet.
- **Present credential:** Holder sends an encrypted presentation message to the verifier's DID.
- **Out-of-band (OOB):** QR code → DIDComm connection → credential exchange.

Flow: `Issuer (DID A) → OOB Invite → Holder (DID B) → encrypted credential exchange → Verifier (DID C)`

### Comparison with OID4VCI / OID4VP

| Aspect | OID4VCI / OID4VP | DIDComm v2 |
|--------|-------------------|------------|
| **Transport** | HTTP only | Any (HTTP, BLE, NFC, WS) |
| **Identity model** | OAuth issuer URL + client ID | DID-based mutual auth |
| **Discovery** | `.well-known/openid-configuration` | DID Document resolution |
| **Encryption** | TLS (transport) + ID token | End-to-end (JWE to DID key) |
| **Complexity** | Lower (familiar OAuth stack) | Higher (DID resolution, routing) |
| **Adoption trend** | Gaining (EUDI, enterprise) | Niche (Aries, gov pilots) |

**Recommendation:** For GGID, OID4VCI/OID4VP is the primary path. DIDComm is a future option
for offline or peer-to-peer scenarios where no HTTP infrastructure exists.

---

## 6. Presentation Exchange (DIF)

The DIF Presentation Exchange specification defines a **declarative framework** for requesting
and presenting verifiable credentials. It is format-agnostic — it works with JWT-VCs,
JSON-LD VCs, SD-JWT VCs, or any JSON claim format.

### Core Concepts

- **Presentation Definition:** What the verifier needs (credential types, constraints, predicates)
- **Presentation Submission:** What the holder provides (matching credentials + descriptor map)
- **Input Descriptor:** Defines a single credential requirement (type filter + constraints)

### Example: Age Verification

```json
{
  "presentation_definition": {
    "id": "age-verification-req",
    "input_descriptors": [{
      "id": "gov-id-descriptor",
      "constraints": {
        "fields": [{
          "path": ["$.credentialSubject.dob", "$.vc.credentialSubject.dob"],
          "filter": { "type": "string", "format": "date" }
        }]
      }
    }]
  }
}
```

Holder responds with a **presentation submission** mapping each input descriptor to the
specific credential. In OID4VP, the definition is embedded in the authorization request.

---

## 7. GGID Credential Architecture

### Format Strategy

| Phase | Format | Rationale | Effort |
|-------|--------|-----------|--------|
| 1 | **JWT-VC** | Simplest; reuses existing JWT/JWS infrastructure in `pkg/crypto` | ~1-2 weeks |
| 2 | **SD-JWT** | Selective disclosure without JSON-LD complexity; EUDI alignment | ~1 week |
| 3 | **OID4VCI/OID4VP** | Protocol integration (already researched) | ~1-2 weeks |
| 4 | **JSON-LD** | Government/academic interoperability (EBSI) | ~2-3 weeks |
| 5 | **DIDComm** | Peer-to-peer exchange, offline scenarios (future) | ~3-4 weeks |

**Guiding principle:** Don't try to support all three formats at once. Start with JWT-VC,
add SD-JWT when privacy-preserving disclosure is needed, and reserve JSON-LD for explicit
interop requirements.

### Schema Management

- **Per-tenant schemas:** Each tenant defines its credential types (e.g., tenant "university"
  defines `UniversityDegree`, tenant "hospital" defines `VaccinationRecord`).
- **Schema storage:** Stored in DB, referenced by URI (`/api/v1/credentials/schemas/{id}`).
- **Validation gate:** Validate against schema at both issuance time (issuer) and
  verification time (verifier) before accepting the credential.

### Go Implementation

```go
type CredentialIssuer interface {
    Issue(ctx context.Context, req IssueRequest) (*Credential, error)
    Revoke(ctx context.Context, credID string) error
}
type CredentialVerifier interface {
    Verify(ctx context.Context, raw []byte) (*VerifiedCredential, error)
}
type FormatRegistry interface {
    Register(format string, handler FormatHandler)
    Get(format string) (FormatHandler, error)
    Supports(format string) bool
}
type FormatHandler interface {
    Issue(cred *Credential, key crypto.Signer) ([]byte, error)
    Verify(raw []byte, pubKey crypto.PublicKey) (*Credential, error)
}
```

### DB Schema

```sql
CREATE TABLE credential_schemas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    type        VARCHAR(255) NOT NULL,
    format      VARCHAR(50) NOT NULL,  -- jwt-vc, sd-jwt, json-ld
    json_schema JSONB NOT NULL,
    version     VARCHAR(50) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, type, version)
);

CREATE TABLE issued_credentials (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    schema_id    UUID NOT NULL REFERENCES credential_schemas(id),
    format       VARCHAR(50) NOT NULL,
    payload      JSONB NOT NULL,
    subject_did  TEXT NOT NULL,
    status       VARCHAR(50) NOT NULL DEFAULT 'active',
    issued_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ
);
```

---

## 8. Interoperability Considerations

### Ecosystem Alignment

| Ecosystem | Preferred Format | DID Method | Notes |
|-----------|-----------------|------------|-------|
| **EBSI** (EU) | JSON-LD + Data Integrity | `did:ebsi` | European blockchain infrastructure; cross-border credentials |
| **EUDI Wallet** | SD-JWT VC | `did:key`, `did:jwk` | EU ARF mandates SD-JWT; pilots in DE, FR, NL, IT |
| **mDL (ISO 18013-5)** | CBOR / mDoc | N/A (offline) | Mobile driver's license; Apple/Google Wallet support |
| **Microsoft Entra Verified ID** | JWT-VC (deprecated→SD-JWT) | `did:web`, `did:ion` | Transitioning to SD-JWT per IETF drafts |
| **Blockcerts** | JSON-LD | `did:btcr`, `did:key` | Academic credentials; MIT-issued diplomas |

### Key Interop Risks

1. **Format fragmentation:** EU government use needs SD-JWT; academic/semantic use needs JSON-LD.
2. **DID method lock-in:** `did:ebsi` only resolves on EBSI; `did:web` is DNS-dependent; `did:key` is non-rotatable.
3. **Schema divergence:** Each ecosystem defines schemas differently (JSON Schema vs JSON-LD context vs CBOR CDDL).
4. **Test suites:** W3C CCG has Data Integrity test vectors; OpenID Foundation offers OID4VCI/OID4VP certification.

### mDL Note

Mobile driver's licenses (ISO 18013-5) use **CBOR** encoding, not JSON — a separate format
family. Supporting mDL requires a CBOR codec and ISO device engagement protocol. Apple Wallet
(iOS 15+) and Google Wallet support mDL natively.

---

## 9. Roadmap

| Phase | Scope | Format | Duration | Dependencies |
|-------|-------|--------|----------|--------------|
| **1** | JWT-VC format + schema registry | JWT-VC | 1-2 weeks | `pkg/crypto` (JWS signing) |
| **2** | SD-JWT format + selective disclosure | SD-JWT | ~1 week | Phase 1 (FormatRegistry) |
| **3** | OID4VCI/OID4VP protocol integration | JWT-VC, SD-JWT | 1-2 weeks | Phases 1-2; existing OID4VCI/OID4VP docs |
| **4** | JSON-LD format for government interop | JSON-LD | 2-3 weeks | RDF canonicalization library |
| **5** | DIDComm v2 exchange (future) | All | 3-4 weeks | DID resolution, DIDComm library |

**Phase 1:** `CredentialIssuer`/`CredentialVerifier` interfaces, JWT-VC `FormatHandler` (go-jose),
`SchemaRegistry` with DB storage, REST API for schema CRUD, integration tests (issue → verify → revoke).

**Phase 2:** SD-JWT `FormatHandler` with per-claim disclosure, updated `IssueRequest` for disclosable
claims, wallet disclosure selection API, EUDI-format compatibility tests.

---

## References

- [W3C VC Data Model 2.0](https://www.w3.org/TR/vc-data-model-2.0/) | [W3C Data Integrity](https://www.w3.org/TR/vc-data-integrity/)
- [IETF SD-JWT](https://datatracker.ietf.org/doc/draft-ietf-oauth-selective-disclosure-jwt/) | [SD-JWT VC](https://datatracker.ietf.org/doc/draft-ietf-oauth-sd-jwt-vc/)
- [DIF Presentation Exchange](https://identity.foundation/presentation-exchange/) | [DIDComm v2](https://identity.foundation/didcomm-messaging/spec/)
- [EUDI ARF](https://digital-strategy.ec.europa.eu/en/library/european-digital-identity-wallet-architecture-and-reference-framework)
