# Token Binding Comparison: DPoP vs mTLS vs PKI vs Cookie

Security model, deployment complexity, performance, fallback, and recommendation matrix.

## Overview

Token binding ties an access token to a specific client, making stolen tokens useless. This guide compares four approaches.

## Quick Comparison

| Feature | DPoP | mTLS | PKI (RFC 8705) | Cookie Binding |
|---------|------|------|---------|----------------|
| **Security** | High | Very High | Very High | Medium |
| **Deployment** | Easy (app layer) | Hard (TLS layer) | Hard (cert mgmt) | Easy |
| **Browser/SPA** | ✅ Web Crypto | ❌ No | ❌ No | ✅ Native |
| **Mobile** | ✅ | ⚠️ Complex | ⚠️ Complex | ⚠️ WebView |
| **Server-to-Server** | ✅ | ✅ | ✅ | N/A |
| **Key Management** | App generates | CA/PKI | CA/PKI | Server-set |
| **Revocation** | Rotate key | Revoke cert | Revoke cert | Delete cookie |
| **Performance** | +1ms (JWT verify) | 0ms (TLS native) | 0ms (TLS native) | 0ms |

## DPoP (Demonstrating Proof of Possession)

### How It Works

```
Client generates EC256 key pair (stored locally)
  → Every request signed with private key
  → JWT proof: {htu, htm, iat, jti, ath}
  → Server verifies proof matches token's cnf.jkt
```

### Pros

- Works in browsers (Web Crypto API)
- Works in mobile (platform keystore)
- No certificate infrastructure needed
- App-layer, no TLS changes

### Cons

- Each request needs proof JWT (minor overhead)
- Client must manage key lifecycle
- No hardware-backed security (software key)

### Best For

- SPAs and mobile apps
- Server-to-server without PKI
- Greenfield deployments

## mTLS (Mutual TLS)

### How It Works

```
Client and server authenticate via X.509 certificates at TLS layer
  → Token issued with cnf.x5t (cert thumbprint)
  → Every request requires valid client certificate
  → Token without matching cert rejected
```

### Pros

- Hardware-backed (HSM, TPM, smart card)
- Transparent to application code
- TLS-native, no app changes

### Cons

- Requires PKI infrastructure (CA, cert distribution)
- Not browser-compatible (no client cert in browser)
- Cert lifecycle management burden
- Hard to rotate

### Best For

- Server-to-server (microservices)
- High-security enterprise
- Financial/healthcare

## PKI (RFC 8705 OAuth mTLS)

### How It Works

Standardized mTLS for OAuth — uses certificate thumbprint in `cnf` claim:

```json
{
  "cnf": { "x5t#S256": "cert-thumbprint-sha256" }
}
```

Same as mTLS but formalized in RFC 8705 with specific OAuth flows.

### Difference from DPoP

| Aspect | PKI/mTLS | DPoP |
|--------|----------|------|
| Key type | X.509 certificate | Raw public key |
| Transport | TLS layer | Application layer (HTTP header) |
| Hardware backed | ✅ (TPM/HSM) | ❌ (software) |
| Browser support | ❌ | ✅ |

## Cookie Binding (TLS Token Binding / DPoP-Lite)

### How It Works

```
Login → Server sets cookie with device fingerprint
  → Subsequent requests must match fingerprint
  → Cookie: ggid_session=val; Secure; HttpOnly; SameSite=Strict
  → Server binds cookie to IP + UA hash
```

### Pros

- Zero client-side changes
- Works in all browsers
- Simple implementation

### Cons

- Not true proof-of-possession (cookie can be stolen)
- IP changes break sessions (mobile networks)
- User-Agent spoofing possible
- Weakest security model

### Best For

- Legacy applications
- Low-risk internal tools
- Supplement to other methods

## Recommendation Matrix

| Scenario | Recommended Binding | Rationale |
|----------|-------------------|-----------|
| SPA (React/Vue) | DPoP | Browser-native via Web Crypto |
| Mobile (iOS/Android) | DPoP | Platform keystore for key storage |
| Server-to-server | mTLS | Hardware-backed, TLS-native |
| Microservices | mTLS | Sidecar (service mesh) handles it |
| Legacy web app | Cookie binding | Simplest, no client changes |
| IoT device | mTLS | Device certificates |
| AI agent | DPoP | Programmatic, no cert infrastructure |
| Financial API | mTLS + DPoP | Defense in depth |

## Fallback Chain

```go
func extractBinding(r *http.Request, claims jwt.MapClaims) (string, error) {
    // 1. Try mTLS (strongest)
    if cert := r.TLS.PeerCertificates; len(cert) > 0 {
        return extractCertThumbprint(cert[0]), nil
    }

    // 2. Try DPoP
    if dpop := r.Header.Get("DPoP"); dpop != "" {
        return extractDPoPThumbprint(dpop, claims), nil
    }

    // 3. Try cookie binding
    if cookie := getCookieBinding(r); cookie != "" {
        return cookie, nil
    }

    // 4. No binding (bearer token)
    if requireBinding(claims) {
        return "", ErrBindingRequired
    }
    return "", nil // Bearer OK for low-risk
}
```

## Per-Client Enforcement

```bash
# Require specific binding per client
PATCH /api/v1/oauth/clients/{client_id}
{"require_token_binding": "dpop"}    // or "mtls", "cookie", "none"
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Binding failures | >5% → client misconfigured |
| mTLS cert expiry | <30 days → renew |
| DPoP key rotation | Track frequency |
| Bearer token usage (when binding required) | Any → security alert |

## See Also

- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
- [JWT Claim Validation](jwt-claim-validation.md)
- [Token Exchange Patterns](token-exchange-patterns.md)
