# Auth Protocol Comparison

This guide compares SAML, OIDC, OAuth 2.1, WebAuthn, and FIDO2 — use case matrix, security features, performance, interoperability, migration paths, and GGID's multi-protocol support.

## Protocol Overview

| Protocol | Purpose | Token Format | Standard |
|---|---|---|---|
| SAML 2.0 | Federation/SSO | XML assertion | OASIS |
| OIDC | Identity layer on OAuth | JWT ID token | IETF |
| OAuth 2.1 | Authorization | Bearer token | IETF (draft) |
| WebAuthn | Passwordless auth | Public key crypto | W3C/FIDO |
| FIDO2 | Passwordless standard | CTAP + WebAuthn | FIDO Alliance |

## Use Case Matrix

| Use Case | SAML | OIDC | OAuth 2.1 | WebAuthn | FIDO2 |
|---|---|---|---|---|---|
| Enterprise SSO | Best | Good | N/A | No | No |
| Consumer SSO | Possible | Best | N/A | No | No |
| API authorization | No | Good | Best | No | No |
| Mobile auth | Poor | Best | Good | Yes | Yes |
| Passwordless | No | No | No | Best | Best |
| Service-to-service | No | Good | Best | No | No |
| Step-up auth | Possible | Best | No | Yes | Yes |
| Delegation | No | Good | Best | No | No |
| Device auth | No | Possible | No | Best | Best |

## Security Features Comparison

| Feature | SAML | OIDC | OAuth 2.1 | WebAuthn |
|---|---|---|---|---|
| Token signing | XML signature | JWT signature | JWT/opaque | Asymmetric crypto |
| Token encryption | XML encryption | JWE | JWE | N/A |
| Token binding | Holder-of-key | DPoP | DPoP/mTLS | Origin binding |
| Replay protection | NotOnOrAfter + ID | exp + jti + nonce | exp + jti + PKCE | Challenge-response |
| Phishing resistance | Medium | Medium | Medium | Very High |
| MFA integration | AuthnContext | acr/amr claims | Custom | Built-in (UV) |
| Mutual TLS | Possible | mTLS RFC 8705 | mTLS | N/A |
| CSRF protection | State + InResponseTo | state + nonce | state + PKCE | Origin check |

## Performance Comparison

| Metric | SAML | OIDC | OAuth 2.1 | WebAuthn |
|---|---|---|---|---|
| Login latency | 500-2000ms | 200-500ms | 200-500ms | 300-800ms |
| Token size | 5-20KB (XML) | 1-3KB (JWT) | 1-3KB | N/A |
| Parsing overhead | High (XML) | Low (JSON) | Low | Low |
| Bandwidth | High | Low | Low | Low |
| Server CPU (verify) | High (XML sig) | Medium (JWT sig) | Medium | Low (crypto) |

## Interoperability

| Protocol | Enterprise | Consumer | Mobile | IoT |
|---|---|---|---|---|
| SAML | Excellent (AD/Okta) | Poor | Poor | No |
| OIDC | Good | Excellent (Google) | Excellent | Good |
| OAuth 2.1 | Good | Excellent | Excellent | Good |
| WebAuthn | Good (Windows Hello) | Growing | Good (biometric) | No |

## Migration Paths

### SAML → OIDC

```
1. Add OIDC endpoints alongside SAML
2. Register apps with OIDC
3. Migrate app-by-app to OIDC
4. Deprecate SAML after all apps migrated
```

### Password-based → WebAuthn

```
1. Add WebAuthn as optional MFA
2. Promote passwordless enrollment
3. Make WebAuthn default for new users
4. Remove password requirement
```

### OAuth 2.0 → 2.1

```
1. Enforce PKCE for all clients
2. Disable implicit grant
3. Enable refresh token rotation
4. Enforce exact redirect URI match
```

## GGID Multi-Protocol Architecture

```
Client → Gateway → ┌─ SAML handler (for enterprise SSO)
                   ├─ OIDC handler (for identity + auth)
                   ├─ OAuth 2.1 handler (for API authz)
                   └─ WebAuthn handler (for passwordless)
                          ↓
                   Shared services:
                   - User store
                   - Token signing
                   - Audit trail
                   - Policy engine
```

### Configuration

```yaml
protocols:
  saml:
    enabled: true
    entity_id: "https://auth.ggid.example.com/saml/metadata"
  oidc:
    enabled: true
    discovery: "/.well-known/openid-configuration"
  oauth:
    enabled: true
    version: "2.1"
    pkce: "required"
  webauthn:
    enabled: true
    rp_id: "ggid.example.com"
    user_verification: "required"
```

## Best Practices

1. **Use OIDC for new apps** — Modern, JSON-based, better tooling
2. **Keep SAML for enterprise** — Many enterprises still use SAML
3. **WebAuthn for passwordless** — Phishing-resistant, future-proof
4. **OAuth 2.1 for API auth** — PKCE mandatory, no implicit
5. **Support multiple protocols** — Don't force one on all clients
6. **Share identity store** — One user database across all protocols
7. **Unify audit trail** — All protocol events in one audit log
8. **Plan migration gradually** — Don't break existing integrations
9. **Test interoperability** — Verify with real IdPs and clients
10. **Document protocol choice** — Help developers choose the right protocol