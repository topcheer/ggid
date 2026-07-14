# Token Exchange Patterns (RFC 8693)

Guide for service-to-service delegation, audience restriction, and Security Token Service (STS) patterns in GGID.

## Overview

RFC 8693 defines token exchange — trading one token for another with different scope, audience, or subject. GGID uses this for internal service-to-service calls and user delegation chains.

## Core Flow

```
Client → Token Exchange Endpoint → New Token (different audience/scope)
         POST /api/v1/oauth/token
         grant_type=urn:ietf:params:oauth:grant-type:token-exchange
```

## Use Case 1: Service-to-Service

A frontend service needs to call a backend service on behalf of a user:

```bash
# Frontend has user's access token (audience: gateway)
# Needs token for backend service (audience: policy-svc)

POST /api/v1/oauth/token
{
  "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
  "subject_token": "eyJ...",
  "subject_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "audience": "policy-svc",
  "scope": "users:read"
}
# → {access_token: "new-eyJ...", audience: "policy-svc", expires_in: 300}
```

The new token:
- Different audience (`policy-svc`)
- Narrowed scope (never broader than original)
- Shorter TTL (max 5 min for service tokens)
- Includes `act` claim identifying the exchanging service

## Use Case 2: User Delegation Chain

User authorizes an agent/service to act on their behalf:

```json
// Original token (user)
{
  "sub": "user-uuid",
  "scope": "users:read users:write"
}

// Exchanged token (agent acting as user)
{
  "sub": "user-uuid",
  "act": {
    "sub": "agent-uuid",
    "scope": "users:read"  // narrowed
  },
  "scope": "users:read",
  "aud": "identity-svc"
}
```

Delegation chain depth limit: 3 (same as admin delegation).

## Use Case 3: External IdP Integration

```bash
# User authenticates via external SAML IdP
# GGID exchanges SAML assertion for GGID access token

POST /api/v1/oauth/token
{
  "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
  "subject_token": "<SAML assertion>",
  "subject_token_type": "urn:ietf:params:oauth:token-type:saml2",
  "audience": "ggid-gateway",
  "scope": "openid profile"
}
# → GGID JWT with mapped identity claims
```

## Audience Restriction

Tokens are only valid for their intended audience:

```go
func validateAudience(claims jwt.MapClaims, expectedAud string) error {
    aud, ok := claims["aud"].(string)
    if !ok {
        return ErrMissingAudience
    }
    if aud != expectedAud {
        return ErrAudienceMismatch
    }
    return nil
}
```

| Service | Expected Audience |
|---------|------------------|
| Gateway | `ggid-gateway` |
| Identity | `identity-svc` |
| Policy | `policy-svc` |
| Audit | `audit-svc` |

A token issued for `identity-svc` is rejected by `policy-svc`.

## Security Requirements

### Scope Narrowing

Exchanged tokens MUST have scope ⊆ original token scope:

```go
if !isSubset(requestedScope, subjectTokenScope) {
    return ErrScopeEscalation
}
```

### Token Binding

Exchanged tokens bind to the requester:

| Binding | Method |
|---------|--------|
| mTLS | `cnf:{"x5t":"..."}` — certificate thumbprint |
| DPoP | `cnf:{"jkt":"..."}` — JWK thumbprint |
| Token hash | `cnf:{"ath":"..."}` — hash of subject token |

Without matching binding, token is rejected.

### `act` Claim

Every exchanged token includes an `act` (actor) claim showing who performed the exchange:

```json
{
  "act": {
    "sub": "gateway-svc",
    "scope": "token-exchange"
  }
}
```

This creates an audit trail of delegation chains.

## Token Type Reference

| Token Type | URN |
|-----------|-----|
| Access token | `urn:ietf:params:oauth:token-type:access_token` |
| Refresh token | `urn:ietf:params:oauth:token-type:refresh_token` |
| ID token | `urn:ietf:params:oauth:token-type:id_token` |
| SAML 2.0 | `urn:ietf:params:oauth:token-type:saml2` |
| JWT | `urn:ietf:params:oauth:token-type:jwt` |

## Error Handling

| Error | Cause |
|-------|-------|
| `invalid_request` | Missing subject_token or audience |
| `invalid_grant` | Subject token expired or invalid |
| `invalid_scope` | Requested scope exceeds subject token |
| `access_denied` | Service not authorized for token exchange |
| `invalid_target` | Unknown or unauthorized audience |

## Caching Strategy

Exchanged tokens are short-lived (5 min) and should be cached briefly:

```go
cacheKey := hash(subjectToken + audience + scope)
if cached := tokenCache.Get(cacheKey); cached != nil {
    if time.Until(cached.Expiry) > 30*time.Second {
        return cached  // Use cached token
    }
}
newToken := exchangeToken(subjectToken, audience, scope)
tokenCache.Set(cacheKey, newToken, 4*time.Minute)
```

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [OAuth Scope Design](oauth-scope-design.md)
- AI Agent Identity
