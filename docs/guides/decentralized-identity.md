# Decentralized Identity

W3C DID spec, DID methods, Verifiable Credentials Data Model 2.0, VC issuance/verification flow, ZKP for privacy, SSI, interoperability with traditional IAM, and migration path.

## W3C DID Spec Overview

A DID (Decentralized Identifier) is a URI that resolves to a DID document containing verification methods and service endpoints.

```json
{
  "id": "did:web:ggid.dev:users:jane",
  "verificationMethod": [{
    "id": "did:web:ggid.dev:users:jane#key-1",
    "type": "JsonWebKey",
    "controller": "did:web:ggid.dev:users:jane",
    "publicKeyJwk": {"kty": "EC", "crv": "P-256", "x": "...", "y": "..."}
  }],
  "service": [{
    "id": "did:web:ggid.dev:users:jane#ggid",
    "type": "GGIDIdentity",
    "serviceEndpoint": "https://auth.ggid.dev/did/jane"
  }]
}
```

## DID Methods

| Method | Resolution | Decentralized? | Use Case |
|--------|-----------|---------------|---------|
| `did:web` | DNS + HTTPS | Semi (DNS-based) | Enterprise, easy adoption |
| `did:ion` | Sidetree on Bitcoin | Yes (L1 anchor) | High verifiability |
| `did:key` | Embedded in DID | Yes (no registry) | Simple, ephemeral |
| `did:ethr` | Ethereum | Yes (on-chain) | Web3 native |

### did:web (Recommended for GGID)

```
did:web:ggid.dev:users:jane
  → Resolves via GET https://ggid.dev/.well-known/did-configuration
  → Returns DID document with public keys
```

## Verifiable Credentials (VC) Data Model 2.0

```json
{
  "@context": ["https://www.w3.org/ns/credentials/v2"],
  "type": ["VerifiableCredential", "EmployeeCredential"],
  "issuer": "did:web:ggid.dev",
  "credentialSubject": {
    "id": "did:web:ggid.dev:users:jane",
    "department": "Engineering",
    "title": "Senior Engineer",
    "startDate": "2023-06-01"
  },
  "proof": {
    "type": "DataIntegrityProof",
    "cryptosuite": "eddsa-rdfc-2022",
    "verificationMethod": "did:web:ggid.dev#key-1",
    "proofValue": "..."
  }
}
```

## VC Issuance Flow

```
1. User authenticates with GGID (password + MFA)
2. User requests VC: "Issue my employee credential"
3. GGID verifies identity + attributes
4. GGID creates VC signed with issuer DID key
5. User stores VC in their wallet (device or cloud)
6. User presents VC to verifier when needed
```

```bash
POST /api/v1/identity/vc/issue
{
  "credential_type": "EmployeeCredential",
  "subject_did": "did:web:ggid.dev:users:jane",
  "attributes": {"department": "Engineering", "title": "Senior Engineer"}
}
# → Returns signed VC
```

## VC Verification Flow

```
1. User presents VC to verifier (employer, partner, app)
2. Verifier checks:
   a. Issuer DID resolves to trusted GGID
   b. Proof signature valid
   c. VC not expired / not revoked
   d. Subject DID matches presenter
3. Verifier accepts/rejects
```

```go
func VerifyVC(vc *VerifiableCredential) error {
    // 1. Resolve issuer DID
    issuerDoc := resolveDID(vc.Issuer)

    // 2. Verify proof
    if err := verifyProof(vc.Proof, issuerDoc); err != nil {
        return ErrInvalidProof
    }

    // 3. Check revocation status
    if revoked := checkRevocation(vc.ID); revoked {
        return ErrRevoked
    }

    // 4. Check expiry
    if vc.ExpirationDate != "" && time.Now().After(parseTime(vc.ExpirationDate)) {
        return ErrExpired
    }

    return nil
}
```

## ZKP for Privacy-Preserving Attributes

See [Zero-Knowledge Proof Identity](zero-knowledge-proof-identity.md).

```
User has VC: department=Engineering, salary=$150K
User needs to prove: department=Engineering (without revealing salary)
→ Generate ZKP proof from VC
→ Verifier confirms department without seeing salary
```

## SSI (Self-Sovereign Identity)

| Principle | GGID Implementation |
|-----------|---------------------|
| User controls identity | User holds VC in own wallet |
| User controls data | User chooses what to present |
| Portable | VCs work across verifiers |
| Interoperable | W3C standards (DID + VC) |
| No single point of failure | DIDs resolve via multiple methods |

## Interoperability with Traditional IAM

| Traditional | Decentralized | Bridge |
|-------------|--------------|--------|
| User account (GGID) | DID | GGID issues VC to user's DID |
| OAuth scope | VC claim | VC presented in OAuth flow |
| SAML attribute | VC claim | VC verified at SP |
| Password auth | DID auth | Challenge-response with DID key |

### OAuth + VC Integration

```bash
# Client requests VC-backed scope
GET /authorize?scope=openid+vc:employee_credential
# → GGID verifies user holds valid EmployeeCredential VC
# → If valid, releases employment-related claims in ID token
```

## Migration Path

```
Phase 1: Traditional IAM (current)
  → GGID manages all identity, users authenticate via password/MFA

Phase 2: DID issuance (additive)
  → GGID issues DIDs to users
  → GGID issues VCs (employee, membership)
  → Users can use VCs externally

Phase 3: VC verification (inbound)
  → GGID accepts VCs from trusted external issuers
  → Users can authenticate with external VCs

Phase 4: SSI-native (future)
  → Users self-manage identity via wallet
  → GGID is one of many VC issuers
  → Cross-organizational trust via DID networks
```

## Monitoring

| Metric | Alert |
|--------|-------|
| DID resolution failures | >1% → DNS or endpoint issue |
| VC verification failures | >5% → expired or revoked |
| VC issuance rate | Track adoption |
| Revocation list sync | <1h lag |

## See Also

- [Zero-Knowledge Proof Identity](zero-knowledge-proof-identity.md)
- [Identity Federation Patterns](identity-federation-patterns.md)
- [Privacy by Design](privacy-by-design.md)
- [AI Agent Identity](ai-agent-identity.md)
