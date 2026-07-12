# Token Binding Strategies

Receiver-constrained tokens, DPoP proof, mTLS bound tokens, token tagging, per-device binding, browser session binding, API client fingerprinting, and revocation on binding mismatch.

## Overview

Bearer tokens can be used by anyone who steals them. Token binding ties a token to a specific holder, making stolen tokens useless.

## Binding Methods

| Method | Layer | Browser | Mobile | Server-to-Server |
|--------|-------|---------|--------|-----------------|
| DPoP | Application | ✅ | ✅ | ✅ |
| mTLS | Transport | ❌ | ⚠️ | ✅ |
| Cookie binding | HTTP | ✅ | ⚠️ | ❌ |
| Token tagging | Application | ✅ | ✅ | ✅ |

## Receiver-Constrained Tokens

Token includes a `cnf` (confirmation) claim that the resource server verifies:

```json
{
  "sub": "user-uuid",
  "cnf": {
    "jkt": "hash-of-dpop-public-key"
  }
}
```

Server checks that the proof (DPoP header or mTLS cert) matches the `cnf` claim.

## DPoP Proof

```json
// DPoP JWT Header
{"typ": "dpop+jwt", "alg": "ES256", "jwk": {"kty": "EC", ...}}

// DPoP JWT Payload
{
  "htu": "https://api.ggid.dev/v1/users",
  "htm": "GET",
  "iat": 1700000000,
  "jti": "unique-nonce-per-request",
  "ath": "sha256(access_token)"
}
```

## mTLS Bound Tokens

```json
{"cnf": {"x5t#S256": "sha256-of-client-cert"}}
```

Every API call requires the client certificate at TLS layer. Token without matching cert is rejected.

## Token Tagging

Lightweight binding without cryptographic proof:

```json
{
  "cnf": {
    "tag": "device-fingerprint-hash"
  }
}
```

```go
func verifyTag(claims jwt.MapClaims, r *http.Request) error {
    tag := claims["cnf"].(map[string]interface{})["tag"].(string)
    actualTag := hashDeviceFingerprint(r)
    if tag != actualTag {
        return ErrBindingMismatch
    }
    return nil
}
```

Weaker than DPoP but requires no client-side crypto. Useful for legacy clients.

## Per-Device Binding

```bash
# Login binds token to device
POST /api/v1/auth/login
{
  "username": "jane@corp.com",
  "password": "...",
  "device_fingerprint": "hash-from-client"
}
# → Token includes cnf.tag = device_fingerprint
```

If token is used from a different device, fingerprint won't match → rejected.

## Browser Session Binding

```http
Set-Cookie: ggid_session=...; Secure; HttpOnly; SameSite=Strict
X-Session-Bind: hash(IP + UA + TLS_JA3)
```

```go
func verifySessionBinding(r *http.Request, session *Session) error {
    expected := hash(r.RemoteAddr + r.UserAgent + getTLSFingerprint(r))
    if session.BindingHash != expected {
        // IP changed (mobile) → allow with step-up
        if sameSubnet(session.IP, r.RemoteAddr) {
            requireStepUp(session)
        } else {
            return ErrBindingMismatch
        }
    }
    return nil
}
```

## Revocation on Binding Mismatch

```go
func (g *Gateway) verifyBinding(r *http.Request, claims jwt.MapClaims) error {
    cnf, ok := claims["cnf"].(map[string]interface{})
    if !ok { return nil } // No binding (bearer token)
    
    // Try DPoP
    if dpop := r.Header.Get("DPoP"); dpop != "" {
        return verifyDPoP(dpop, cnf["jkt"].(string))
    }
    
    // Try mTLS
    if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
        return verifyCert(r.TLS.PeerCertificates[0], cnf["x5t#S256"].(string))
    }
    
    // Try tag
    if tag, ok := cnf["tag"]; ok {
        return verifyTag(tag.(string), r)
    }
    
    // Binding required but no proof provided
    return ErrBindingRequired
}
```

### Mismatch Actions

| Scenario | Action |
|----------|--------|
| DPoP proof invalid | Reject (401) |
| mTLS cert mismatch | Reject (401) |
| Device fingerprint changed | Require step-up MFA |
| IP changed (same subnet) | Allow (mobile network) |
| IP changed (different country) | Reject + revoke + alert |

## Per-Client Binding Policy

```bash
PATCH /api/v1/oauth/clients/{client_id}
{"token_binding": "dpop"}  # dpop, mtls, tag, none
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Binding verification failures | >1% → misconfigured client |
| Bearer tokens (no binding) | Track for migration |
| Binding mismatch → revocation | Any → investigate token theft |

## See Also

- [Token Binding Comparison](token-binding-comparison.md)
- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Session Security](session-security.md)
