# Token Binding & DPoP Research

> RFC 8471, DPoP (RFC 9449), mTLS sender-constrained tokens, and GGID's roadmap.

---

## Token Binding Mechanisms

### 1. DPoP (RFC 9449) — Demonstrating Proof-of-Possession

Browser apps prove they hold a private key by signing each request:

```
1. Client generates ECDSA keypair (stored in browser)
2. Each request: client signs (htm, htu, iat, jti) with private key
3. Server verifies signature against DPoP header
4. Access token bound to key thumbprint (cnf.jkt claim)
```

**GGID status**: DPoP implemented in OAuth service (verified in `dpop.go`). Coverage tests in `dpop_test.go`.

### 2. mTLS Sender-Constrained (RFC 8705)

Access token bound to client's TLS certificate:

```
Authorization: Bearer <token>
X-Tls-Cert-Hash: base64(sha256(client_cert))
```

Token includes `cnf.x5t#S256` claim matching cert hash.

**GGID status**: mTLS support exists in `pkg/transport` (verified in `grpc_tls_mtls_test.go`). Token binding claim not yet implemented.

### 3. Token Binding (RFC 8471-8473)

IETF Token Binding protocol — largely abandoned in favor of DPoP. Browser support was removed. **Not recommended.**

---

## Comparison

| Mechanism | Client Type | Browser Support | GGID Status |
|-----------|------------|-----------------|-------------|
| DPoP | SPA + mobile | Yes (WebCrypto) | Done |
| mTLS binding | Service-to-service | No | Partial (TLS yes, binding no) |
| Token Binding | Deprecated | Removed | N/A |

---

## GGID Roadmap

1. **DPoP** — ✅ Done. Used for browser-based OAuth flows.
2. **mTLS token binding** — Add `cnf.x5t#S256` claim to service-to-service tokens. Effort: 2 days.
3. **Token binding header verification** — Gateway checks DPoP/mTLS binding on all requests. Effort: 1 day.

Priority: P2 (mTLS binding for zero-trust service mesh).

---

*See: [Security Overview](../architecture/security-overview.md) | [Zero Trust Guide](zero-trust-architecture.md)*

*Last updated: 2025-07-11*
