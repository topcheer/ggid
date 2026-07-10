# OID4VP and Credential Presentation

> Research note for GGID IAM. Focuses on credential **verification and
> presentation**. See [oid4vci-and-verifiable-credentials.md](oid4vci-and-verifiable-credentials.md)
> for issuance flows. All spec references as of 2024–2025.

## 1. Overview

**OID4VP** — OpenID for Verifiable Presentations (draft-ietf-oauth-oid4vp,
**draft 23**, late 2024). Defines how a **verifier** requests and receives
verifiable credentials from a **wallet** (holder's app) over OAuth 2.0.

| Protocol | Role | Direction |
|----------|------|-----------|
| OID4VCI | Issuer → Wallet | Credential **issuance** |
| **OID4VP** | **Verifier ↔ Wallet** | **Credential presentation** |
| SIOPv2 | Wallet as OP | Self-issued authentication |

Core flow:

```
 Verifier (RP)          Wallet (Holder)         Issuer
   |  authz request (presentation_definition) ->|
   |                                  |  user consents
   |  <- VP Token (credential) ------|
   |  verify + check status list ---->--> status
   |  create session                 |
```

**Key properties:** offline-verifiable signatures, selective disclosure,
holder binding (proof of possession), and status checking without tracking.

---

## 2. Verifier Flow

### Step 1: Presentation Definition

The verifier declares *what credentials it needs* using a
`presentation_definition` (DIF Presentation Exchange format):

```json
{
  "presentation_definition": {
    "id": "age-verification",
    "input_descriptors": [{
      "id": "gov-id",
      "format": { "ldp_vc": { "proof_type": ["Ed25519Signature2018"] } },
      "constraints": { "fields": [{
        "path": ["$.vc.credentialSubject.age"],
        "filter": { "type": "number", "minimum": 18 },
        "intent_to_retain": false
      }]}
    }]
  }
}
```

Each `input_descriptor` specifies:
- **id** — unique identifier for matching in the response
- **format** — accepted credential formats (ldp_vc, jwt_vc, dc+sd-jwt, mso_mdoc)
- **constraints.fields** — JSONPath expressions with JSON Schema filters
- **schema_uri** (optional) — restrict to a specific credential schema

### Step 2: Authorization Request

The verifier sends a signed authorization request. Large payloads use
`request_uri` (URL reference) to keep QR codes short. The `client_metadata`
field declares supported response formats (`vp_formats_supported`):

```
GET /authorize?response_type=vp_token
  &client_id=https://verifier.example.com
  &redirect_uri=https://verifier.example.com/callback
  &request_uri=https://verifier.example.com/request/jwt-xyz
  &client_metadata=<URL-encoded>
```

### Step 3: User Consents

The wallet parses the request and displays a human-readable prompt:

> **verifier.example.com** requests: Proof of age (18+)
> from your **Government ID** credential. Claims shared: `age`.

The user approves or denies. With **selective disclosure** (SD-JWT),
the wallet reveals only the filtered claims — e.g. prove `age >= 18`
without disclosing `birth_date`.

### Step 4: Presentation Response

The wallet generates a **Verifiable Presentation** (VP) containing the
credential(s) plus a **proof of possession** (holder binding):

- **Front-channel** (redirect): VP token sent via browser redirect to `redirect_uri`
- **Back-channel** (direct POST): wallet POSTs VP to verifier's `response_endpoint`

```json
{
  "vp_token": "<signed-verifiable-presentation>",
  "presentation_submission": {
    "definition_id": "age-verification",
    "descriptor_map": [{ "id": "gov-id", "format": "ldp_vc", "path": "$" }]
  }
}
```

The verifier validates:
1. **Signature** — credential signed by a trusted issuer (DID / X.509 key)
2. **Holder binding** — proof matches the key that holds the credential
3. **Status** — credential not revoked/suspended (see section 7)
4. **Freshness** — nonce in the proof prevents replay

---

## 3. Presentation Definition Format

### Input Descriptors

Input descriptors are evaluated against each credential in the wallet.
A credential satisfies a descriptor when **all** field constraints pass.

| Constraint | Purpose |
|-----------|---------|
| `path` (JSONPath) | Extract a claim from the credential |
| `filter` (JSON Schema) | Validate the extracted value |
| `purpose` | Human-readable reason for the request |
| `intent_to_retain` | Whether verifier will store the data |

### DIF Presentation Exchange

OID4VP adopts the **DIF Presentation Exchange** specification (v2) as its
primary definition language. Key features:

- **Submission requirements**: `from`/`from_select` rules ("present 1 of N",
  "present all from group A")
- **Format-agnostic**: W3C VC, SD-JWT VC, ISO mdoc

```json
{
  "submission_requirements": [{ "rule": "pick", "count": 1, "from": "A" }],
  "input_descriptors": [
    { "id": "passport",         "group": ["A"], "constraints": { ... } },
    { "id": "drivers-license",  "group": ["A"], "constraints": { ... } }
  ]
}
```

This tells the wallet: "present one credential — either a passport or a driver's license."

---

## 4. Digital Credentials API

The **Digital Credentials API** lets websites request verifiable credentials
directly from a platform wallet — without redirects, QR codes, or custom apps.

```javascript
const credential = await navigator.credentials.get({
  digital: {
    requests: [{
      protocol: "org.oid4vp",
      data: {
        presentation_definition: {
          id: "age-check",
          input_descriptors: [{
            id: "gov-id",
            constraints: { fields: [{ path: ["$.age"], filter: { minimum: 18 } }] }
          }]
        }
      }
    }]
  }
});
// credential.data → verifiable presentation
```

### Browser Support (2025)

| Browser | Status |
|---------|--------|
| Chrome 128–140 | Origin trial |
| **Chrome 141+** | **Shipped (stable)** |
| Edge | Following Chromium |
| Safari / Firefox | Not yet |

The API is **cross-platform**: on Android it delegates to Google Play
Services wallet integrations; on desktop it invokes wallet extensions or
system-level credential managers.

Privacy safeguards: user gesture required, browser mediates (no wallet fingerprinting),
origin-bound responses prevent cross-site tracking.

---

## 5. Cross-Device Presentation

### Same-Device (Seamless)

The wallet and verifier run on the same device — browser extension wallet,
platform wallet (OS-level), or PWA. No QR code or redirect needed.
The Digital Credentials API (section 4) enables this flow natively.

### QR Code Flow (Cross-Device)

Verifier displays an authorization request as a QR code. The user scans
it with their mobile wallet app:

```
 Desktop Browser              Mobile Wallet
   |  display QR code           |
   |                            |  scan → parse authz request
   |                            |  user approves presentation
   |  <- VP via deep-link -----|
   |  verify + create session   |
```

Mirrors the WebAuthn **hybrid transport** pattern — QR code contains a
session URL, presentation sent via redirect or direct POST.

### BLE Proximity (Privacy-Preserving)

- BLE handshake establishes an encrypted channel between phone and desktop
- Credential flows through BLE, **not** the verifier's server
- Prevents session-linking across verifiers

---

## 6. Selective Disclosure and Privacy

### SD-JWT (RFC 9901)

**SD-JWT** is now a published **RFC 9901** (2025). It enables per-claim
selective disclosure:

1. **Issuance**: each disclosable claim is salted and hashed. The SD-JWT
   contains hashes; actual values travel in `disclosures` arrays.
2. **Presentation**: holder chooses which claims to reveal, including
   the salted values alongside the SD-JWT.
3. **Verification**: verifier recomputes hashes for disclosed claims and
   confirms they match the signed SD-JWT.

```go
sd, err := sdjwt.Parse(presentationToken)
if err != nil { return err }

claims := sd.DisclosedClaims()  // only revealed claims visible
age, ok := claims["age"].(float64)
if !ok || age < 18 {
    return errors.New("age requirement not met")
}
if err := sd.VerifyHashes(); err != nil {
    return fmt.Errorf("disclosure verification failed: %w", err)
}
```

### BBS+ Signatures

**BBS+** (draft-irtf-cfrg-bbs-signature) provides **zero-knowledge proofs**:

- Prove `age >= 18` **without revealing** the actual age or birth date
- Each presentation generates an **unlinkable** proof — verifier cannot
  correlate two presentations from the same holder

| Feature | SD-JWT | BBS+ |
|---------|--------|------|
| Selective disclosure | Yes (reveal claims) | Yes (prove in ZK) |
| Unlinkability | No | **Yes** |
| Complexity | Low | High (pairing crypto) |
| Deployment | Widely available | Emerging |
| Go library | `sdjwt`, `jwx/v2` | `aries-framework-go` |

### Privacy Properties

- **Data minimization**: share only requested claims
- **Unlinkability** (BBS+): verifier cannot correlate sessions
- **No phone-home**: verification is offline — verifier never contacts the issuer
- **Holder binding**: proof-of-possession prevents stolen-credential replay

---

## 7. Status Verification

Before accepting a credential, the verifier must check it is not
**revoked** or **suspended**.

**Token Status List** (draft-ietf-oauth-status-list) provides a compact,
privacy-preserving mechanism:

1. Each credential contains a `status` field pointing to a status list
   URL and an index.
2. The verifier fetches the **entire status list** (a compressed bitstring,
   typically a few KB for millions of credentials). Checks the bit at
   the credential's index: `0` = valid, `1` = revoked.

```go
func checkStatus(ctx context.Context, vc *VerifiableCredential) error {
    s := vc.CredentialStatus
    if s.Type != "StatusList2021" { return nil }
    list, err := fetchStatusList(ctx, s.StatusListCredential) // cache + TTL
    if err != nil { return fmt.Errorf("status fetch failed: %w", err) }
    if list.IsRevoked(s.StatusListIndex) {
        return errors.New("credential revoked")
    }
    return nil
}
```

**Privacy**: fetching the entire list prevents issuer-side tracking.
**Offline**: cache with a TTL for bounded staleness.

---

## 8. GGID as Verifier

GGID can act as a **verifier** — accepting verifiable credentials as an
authentication mechanism alongside passwords, SSO, and WebAuthn.

**Use case**: A government employee presents an employee ID credential.
GGID verifies it, extracts claims, and creates a session.

### Integration Points

| Component | Responsibility |
|-----------|---------------|
| Gateway | New endpoint `POST /api/v1/auth/credential` |
| Auth service | Verify VP signature, check status, extract claims |
| Identity service | Create/update user from VC claims |

### Go Interface

```go
type CredentialVerifier interface {
    VerifyPresentation(ctx context.Context, vpToken string) (*PresentationResult, error)
}

type PresentationResult struct {
    Issuer     string
    HolderID   string
    Claims     map[string]any  // disclosed claims only
    ExpiresAt  time.Time
}

func (v *VCVerifier) VerifyPresentation(ctx context.Context, vpToken string) (*PresentationResult, error) {
    vc, err := parseVerifiablePresentation(vpToken)       // 1. signature
    if err != nil { return nil, fmt.Errorf("invalid VP: %w", err) }
    if _, ok := v.trustedIssuers[vc.Issuer]; !ok {        // 2. trust
        return nil, errors.New("untrusted issuer")
    }
    if err := vc.VerifyHolderBinding(); err != nil {       // 3. holder binding
        return nil, err
    }
    if err := v.checkStatus(ctx, vc); err != nil { return nil, err } // 4. status
    if vc.ExpiresAt.Before(time.Now()) { return nil, errors.New("expired") }
    return &PresentationResult{Issuer: vc.Issuer, HolderID: vc.HolderKeyHash(),
        Claims: vc.DisclosedClaims(), ExpiresAt: vc.ExpiresAt}, nil
}
```

### DB Schema

Store presentation **metadata** only — never the full credential:

```sql
CREATE TABLE credential_presentations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    user_id       UUID REFERENCES users(id),
    issuer        TEXT NOT NULL,
    credential_id TEXT NOT NULL,
    holder_hash   TEXT NOT NULL,
    claims_json   JSONB NOT NULL,         -- disclosed claims only
    presented_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(credential_id, holder_hash)
);
```

---

## 9. GGID Roadmap

| Phase | Scope | Effort |
|-------|-------|--------|
| **1** | Accept SD-JWT VP presentations. Verify signature, holder binding, expiry, status. Map claims to user. | ~1 week |
| **2** | Tenant-configurable presentation definitions API. Admins define accepted credentials/claims. | ~1 week |
| **3** | Digital Credentials API integration. Browser-native wallet from login page. | ~1 week |
| **4** | Cross-device BLE presentation. QR + proximity flow for desktop. | ~2 weeks |

**Phase 1**: `CredentialVerifier` interface + SD-JWT impl, `POST /api/v1/auth/credential` endpoint, trusted issuer registry (per-tenant), status list caching with TTL.
**Phase 2**: `GET/POST /api/v1/presentation-definitions` API, Console UI editor for input descriptors.

## References

- OID4VP draft 23: <https://openid.net/specs/openid-4-verifiable-presentations-1_0-23.html>
- DIF Presentation Exchange v2: <https://identity.foundation/presentation-exchange/>
- RFC 9901 SD-JWT: <https://datatracker.ietf.org/doc/rfc9901/>
- Token Status List: <https://datatracker.ietf.org/doc/draft-ietf-oauth-status-list/>
- BBS+ Signatures: <https://datatracker.ietf.org/doc/draft-irtf-cfrg-bbs-signature/>
- Digital Credentials API: <https://developer.chrome.com/blog/digital-credentials-api-shipped>
