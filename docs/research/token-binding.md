# Token Binding: Sender-Constrained Access Tokens

## Overview

Token binding prevents stolen access tokens from being used by anyone other than the original client. This document analyzes three token binding approaches — RFC 8471 (Token Binding), DPoP, and mTLS certificate binding — and maps them to GGID's capabilities.

> **Related**: [Token Binding & DPoP](token-binding-dpop.md), [Open Banking FAPI](open-banking-fapi.md), [OAuth 2.1 Changes](oauth-2.1-changes.md)

## The Problem

Bearer tokens (default OAuth 2.0) can be used by anyone who possesses them:

```
Attacker steals access token (via XSS, network interception, log leak)
  → Attacker sends: Authorization: Bearer <stolen_token>
  → Server: 200 OK (no way to verify WHO is sending it)
```

Token binding solves this by cryptographically linking the token to a client-held key.

## Approach 1: DPoP (Demonstrating Proof of Possession)

### Mechanism

The client generates a key pair and proves possession on every request:

```
1. Token request: Client includes DPoP header with proof JWT
   → Server adds cnf:{jkt:thumbprint} to access token

2. Every API request:
   Authorization: DPoP <access_token>
   DPoP: <signed_proof_jwt>

3. Proof JWT contains:
   - htm: HTTP method
   - htu: Full URL
   - iat: Timestamp
   - jti: Unique nonce (replay prevention)
   - ath: SHA-256(access_token) [FAPI 2.0]
```

### Stolen Token Scenario

```
Attacker steals access token + DPoP proof
  → Tries to replay
  → Server checks jti → already used → REJECTED
  → Tries new request
  → Server checks htu → different URL → REJECTED
  → Attacker doesn't have private key → can't sign new proof
```

### Properties

| Property | Value |
|----------|-------|
| Key type | ECDSA P-256 or Ed25519 |
| Per-request overhead | One signature verification (~0.5ms) |
| Replay protection | jti nonce in Redis |
| Browser support | Yes (WebCrypto API) |
| Mobile support | Yes |
| Infrastructure | None extra |

### GGID Status

**Not implemented**. Implementation requires:
1. Accept `DPoP` header on `/oauth/token` endpoint
2. Validate proof JWT (signature, htm, htu, iat, jti)
3. Store `jti` in Redis for replay detection
4. Add `cnf` claim to access token
5. Validate DPoP proof on protected resource requests

## Approach 2: mTLS Certificate Binding (RFC 8705)

### Mechanism

The client authenticates with a TLS client certificate. The access token is bound to the certificate's thumbprint:

```
1. Token request over mTLS:
   → Server extracts client cert from TLS handshake
   → Adds cnf:{x5t#S256:cert_thumbprint} to access token

2. Every API request over mTLS:
   → Server verifies client cert
   → Checks cert thumbprint matches cnf claim in token
   → If mismatch → REJECTED
```

### Properties

| Property | Value |
|----------|-------|
| Key type | X.509 certificate (RSA or ECDSA) |
| Per-request overhead | TLS handshake (amortized via session resumption) |
| Replay protection | TLS provides channel binding |
| Browser support | Difficult (client cert provisioning) |
| Mobile support | Difficult |
| Infrastructure | PKI / certificate authority |

### GGID Status

**Partially implemented**. mTLS support exists in `jar_mtls.go` for JAR requests. Full implementation requires:
1. mTLS on token endpoint
2. `cnf` claim with `x5t#S256` thumbprint
3. mTLS validation on resource server proxy
4. Certificate revocation checking (CRL/OCSP)

## Approach 3: RFC 8471 Token Binding (Deprecated)

### Mechanism

RFC 8471 defined a TLS extension for negotiated token binding. The browser proved possession of a key via the TLS channel.

### Status

**Deprecated**. Chrome removed Token Binding support in 2018. The IETF working group concluded. This approach is no longer viable.

**GGID**: Not implemented (correctly — deprecated standard).

## Comparison Matrix

| Feature | Bearer (default) | DPoP | mTLS (RFC 8705) | RFC 8471 |
|---------|-----------------|------|------------------|---------|
| Token theft protection | None | Yes | Yes | Yes |
| Replay protection | jti only | jti + URL binding | TLS channel | TLS channel |
| Browser support | Yes | Yes (WebCrypto) | Difficult | Removed |
| Mobile support | Yes | Yes | Difficult | N/A |
| PKI required | No | No | Yes | Yes |
| Standard | RFC 6749 | draft-ietf-oauth-dpop | RFC 8705 | RFC 8471 (dead) |
| Performance impact | None | +0.5ms/request | TLS overhead | TLS overhead |
| GGID status | Done | Not yet | Partial | N/A |

## FAPI 2.0 Requirements

FAPI 2.0 mandates sender-constrained tokens. Acceptable methods:

| Method | FAPI 2.0 Baseline | FAPI 2.0 Advanced |
|--------|-------------------|------------------|
| mTLS | Accepted | Accepted |
| DPoP | Accepted | Preferred |

**Recommendation**: Implement DPoP first (broader client support), then mTLS for regulated industries.

## Implementation Priority

| Priority | Feature | Effort | Enables |
|----------|---------|--------|---------|
| P1 | DPoP token binding | Medium | FAPI 2.0, general security |
| P2 | Full mTLS on token endpoint | Medium | UK Open Banking, enterprise |
| P2 | DPoP replay detection (Redis jti) | Small | Replay prevention |
| P3 | Certificate revocation (CRL/OCSP) | Medium | Enterprise mTLS |

## References

- [RFC 8705: mTLS](https://www.rfc-editor.org/rfc/rfc8705)
- [DPoP Draft](https://datatracker.ietf.org/doc/draft-ietf-oauth-dpop/)
- [RFC 8471: Token Binding (historical)](https://www.rfc-editor.org/rfc/rfc8471)

## See Also

- [Token Binding & DPoP](token-binding-dpop.md)
- [Open Banking FAPI](open-banking-fapi.md)
- [OAuth 2.1 Changes](oauth-2.1-changes.md)
- [Confidential Client & PKCE](confidential-client-pkce.md)
