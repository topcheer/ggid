# Open Banking & FAPI: Research and GGID Readiness

## Overview

Open Banking standards (FAPI 2.0, PAR, DPoP) define high-security OAuth profiles for financial APIs. This document analyzes these standards and maps them to GGID's current capabilities and compliance gaps.

> **Related**: [Token Binding & DPoP](token-binding-dpop.md), [Open Banking PSD2](open-banking-psd2.md), [OAuth 2.1 Changes](oauth-2.1-changes.md), [Confidential Client PKCE](confidential-client-pkce.md)

## FAPI 2.0 Security Profile

The Financial-grade API (FAPI) 2.0 is the OpenID Foundation's high-security OAuth profile, succeeding FAPI 1.0 (RFC 8374). It mandates sender-constrained tokens, PKCE, and strict validation.

### FAPI 2.0 Requirements

| Requirement | Description | GGID Status |
|-------------|-------------|-------------|
| Authorization code + PKCE | Mandatory for all clients | Done |
| Sender-constrained tokens | DPoP or mTLS-bound tokens | Partial (mTLS in jar_mtls.go) |
| PAR (Pushed Auth Requests) | Request pushed to AS before redirect | Not implemented |
| No implicit grant | Removed (same as OAuth 2.1) | Done (not implemented) |
| Exact redirect URI matching | No wildcards | Done |
| `iss` parameter in auth response | Prevents mix-up attacks | Done |
| Intent ID | Resource-specific consent | Partial (consent store) |
| JWT-Secured Authorization Request (JAR) | RFC 9101 | Done (jar_mtls.go) |
| Token expiration ≤ 10 min | Short-lived access tokens | Configurable |
| Refresh token rotation | Single-use | Done |

**Compliance: 7/10 requirements met (70%)**

## PAR (Pushed Authorization Requests, RFC 9126)

### Problem PAR Solves

Standard authorization requests pass all parameters in the URL query string:

```
GET /authorize?response_type=code&client_id=xxx&redirect_uri=xxx
  &scope=accounts&state=xxx&code_challenge=xxx&code_challenge_method=S256
  &nonce=xxx&claims=...HTTP/1.1
```

**Risks**:
- URL length limits (especially with JAR)
- Parameters logged in proxy/web server access logs
- Parameters visible in browser history/referer

### PAR Flow

```
Client                    Authorization Server         Browser
  │                            │                          │
  │── POST /par ──────────────→│                          │
  │   request_uri=pushed       │                          │
  │   (all auth params in body)│                          │
  │                            │  Store request           │
  │←── 201 {request_uri} ──────│                          │
  │                            │                          │
  │── Redirect browser ──────────────────────────────────→│
  │   /authorize?request_uri=xxx                            │
  │                            │←── GET /authorize?request_uri=xxx ──│
  │                            │  Look up stored request   │
  │                            │  Process auth flow        │
```

### GGID Gap

PAR is not yet implemented. Adding it requires:
1. `POST /oauth/par` endpoint — stores request, returns `request_uri`
2. Modify `/oauth/authorize` to accept `request_uri` parameter
3. Request storage in Redis with TTL (60 seconds)
4. Validate `request_uri` is consumed exactly once

## DPoP (Demonstrating Proof-of-Possession)

### How DPoP Works

DPoP binds access tokens to a client's key pair, preventing token theft:

```
1. Client generates ECDSA P-256 key pair (DPoP key)

2. For each request, client creates a DPoP proof JWT:
   Header: { typ: "dpop+jwt", alg: "ES256", jwk: <public_key> }
   Body: {
     htm: "POST",
     htu: "https://api.ggid.example.com/api/v1/users",
     iat: 1706104200,
     jti: "random-unique-id",
     ath: sha256(access_token)  // FAPI 2.0 addition
   }

3. Client sends:
   Authorization: DPoP <access_token>
   DPoP: <dpop-proof-jwt>

4. Server verifies:
   - Proof signature matches JWK in header
   - htm matches request method
   - htu matches request URL
   - iat within acceptable window
   - jti not replayed
   - ath matches access token hash
   - Access token contains cnf.jkt = thumbprint of JWK
```

### DPoP vs mTLS

| Aspect | DPoP | mTLS (RFC 8705) |
|--------|------|------------------|
| Transport | Any (HTTP) | TLS with client cert |
| Key type | ECDSA/EdDSA | X.509 certificate |
| Infrastructure | None extra | PKI infrastructure |
| Mobile friendly | Yes | Difficult (cert provisioning) |
| Browser friendly | Yes | No |
| Performance | 1 extra verify per request | TLS overhead |
| Standard | draft-ietf-oauth-dpop | RFC 8705 |

### GGID Status

DPoP is not yet implemented. mTLS sender-constrained tokens are partially implemented in `jar_mtls.go`.

**Implementation plan**:
1. Accept `DPoP` header on token endpoint
2. Validate proof JWT signature and claims
3. Add `cnf` claim to access token: `{"jkt": "<thumbprint>"}`
4. Validate DPoP proof on resource requests
5. Replay detection via `jti` in Redis

## FAPI 2.0 Baseline vs Advanced

### Baseline Profile

| Requirement | GGID Status |
|-------------|-------------|
| OAuth 2.1 compliant | Yes (87%) |
| PKCE mandatory | Yes |
| Sender-constrained (mTLS or DPoP) | Partial |
| `iss` in auth response | Yes |
| State parameter | Yes (Redis-backed) |
| No implicit/ROPC | Yes |
| Refresh token rotation | Yes |

### Advanced Profile (Higher Security)

| Requirement | GGID Status |
|-------------|-------------|
| PAR mandatory | Not implemented |
| DPoP mandatory (not mTLS) | Not implemented |
| JAR mandatory | Partial |
| Intent ID | Partial |
| Consent management | Done (consent store) |
| Transaction signing | Not implemented |

## Open Banking Regional Standards

### UK Open Banking (OBIE)

- Based on FAPI 1.0 Final
- Requires mutually-signed TLS (directory)
- OIDC hybrid flow (code id_token)
- Custom claims: `openbanking_intent_id`

### PSD2 (EU)

- SCA (Strong Customer Authentication) mandatory
- Dynamic linking for payment initiation
- Requires QWACS (Qualified Certificates)
- GGID readiness: Needs transaction signing support

### Brazil Open Finance

- Based on FAPI 1.0 Implementer's Draft 3
- Requires DPoP
- mTLS for transport security
- GGID gap: DPoP implementation needed

### Australia CDR

- FAPI 1.0 Final
- PAR required
- Refresh token lifetime ≤ 28 days
- GGID gap: PAR implementation needed

## GGID Open Banking Readiness Score

| Area | Compliance | Score |
|------|-----------|-------|
| OAuth 2.1 base | 87% | B+ |
| PKCE | 100% | A |
| Sender-constrained tokens | 30% | D- |
| PAR | 0% | F |
| JAR | 70% | C+ |
| Consent management | 90% | A- |
| Certificate handling | 60% | C |
| Transaction signing | 0% | F |
| **Overall FAPI 2.0** | **~48%** | **D+** |

## Priority Implementation

| Priority | Feature | Effort | Enables |
|----------|---------|--------|---------|
| P1 | DPoP | Medium | Sender-constrained tokens, Brazil OF |
| P1 | PAR | Small | All FAPI 2.0 profiles, Australia CDR |
| P2 | Full JAR | Small | Request integrity |
| P2 | Full mTLS on all endpoints | Medium | UK Open Banking |
| P3 | Transaction signing | Large | PSD2 payment initiation |
| P3 | QWAC support | Large | EU PSD2 compliance |

## References

- [FAPI 2.0 Security Profile](https://openid.net/specs/fapi-2_0-security-profile.html)
- [RFC 9126: PAR](https://www.rfc-editor.org/rfc/rfc9126)
- [DPoP Draft](https://datatracker.ietf.org/doc/draft-ietf-oauth-dpop/)
- [RFC 9101: JAR](https://www.rfc-editor.org/rfc/rfc9101)
- [UK Open Banking Standards](https://standards.openbanking.org.uk/)

## See Also

- [Token Binding & DPoP](token-binding-dpop.md)
- [Open Banking PSD2](open-banking-psd2.md)
- [OAuth 2.1 Changes](oauth-2.1-changes.md)
- [Confidential Client & PKCE](confidential-client-pkce.md)
