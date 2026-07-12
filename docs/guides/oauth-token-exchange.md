# OAuth Token Exchange (RFC 8693)

This guide covers token exchange use cases, request parameters, response format, JWT-based exchange, delegation semantics, act claim, and GGID's implementation.

## Overview

RFC 8693 defines a token exchange grant type that allows a client to trade one token for another. This enables service-to-service delegation, impersonation, and token type conversion.

## Use Cases

### 1. Service-to-Service Delegation

A frontend service calls a backend service on behalf of a user:

```
User → Frontend (has user token) → Backend (needs delegated token)
Frontend exchanges user token for a delegated token scoped for Backend
```

### 2. Token Type Conversion

Exchange a JWT access token for an opaque token (or vice versa):

```
Client has JWT → exchanges for opaque token → uses with service that prefers opaque
```

### 3. Subject Token → Impersonation Token

Exchange a user token for an impersonation token:

```
Admin has own token → exchanges for token acting as another user
```

### 4. Cross-System Delegation

Exchange a token from one IdP for a token from another:

```
Client has IdP-A token → exchanges for IdP-B token → accesses IdP-B resources
```

## Request Parameters

### Token Exchange Request

```bash
POST /oauth/token
Content-Type: application/x-www-form-urlencoded
Authorization: Bearer <client_credentials>

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=eyJ...
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&actor_token=eyJ...
&actor_token_type=urn:ietf:params:oauth:token-type:access_token
&requested_token_type=urn:ietf:params:oauth:token-type:access_token
&audience=https://backend.ggid.example.com
&scope=read write
```

### Parameter Reference

| Parameter | Required | Description |
|---|---|---|
| `grant_type` | Yes | `urn:ietf:params:oauth:grant-type:token-exchange` |
| `subject_token` | Yes | The token being exchanged |
| `subject_token_type` | Yes | Type of subject token |
| `actor_token` | No | Token of the actor (delegator) |
| `actor_token_type` | No | Type of actor token |
| `requested_token_type` | No | Desired output token type |
| `audience` | No | Target audience for new token |
| `scope` | No | Requested scopes |
| `resource` | No | Target resource server |

### Token Type URIs

| Token Type | URI |
|---|---|
| Access token | `urn:ietf:params:oauth:token-type:access_token` |
| Refresh token | `urn:ietf:params:oauth:token-type:refresh_token` |
| JWT | `urn:ietf:params:oauth:token-type:jwt` |
| ID token | `urn:ietf:params:oauth:token-type:id_token` |
| SAML2 assertion | `urn:ietf:params:oauth:token-type:saml2` |
| SAML1 assertion | `urn:ietf:params:oauth:token-type:saml1` |

## Response Format

### Successful Response

```json
{
  "access_token": "eyJ...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "read write",
  "audience": "https://backend.ggid.example.com"
}
```

### Error Response

```json
{
  "error": "invalid_token",
  "error_description": "The subject token is invalid or expired"
}
```

### Error Codes

| Error | Description |
|---|---|
| `invalid_request` | Missing or invalid parameter |
| `invalid_token` | Subject/actor token invalid |
| `invalid_scope` | Requested scope not allowed |
| `unauthorized_client` | Client not allowed to exchange |
| `access_denied` | Exchange not permitted |

## JWT-Based Token Exchange

### Input JWT (Subject Token)

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid-1234",
  "aud": "frontend-service",
  "exp": 1700000600,
  "scope": "users:read"
}
```

### Output JWT (Exchanged Token)

```json
{
  "iss": "https://auth.ggid.example.com",
  "sub": "user-uuid-1234",
  "aud": "backend-service",
  "exp": 1700000900,
  "scope": "users:read",
  "act": {
    "sub": "frontend-service-client-id"
  }
}
```

## Delegation Semantics

### Subject vs Actor

| Concept | Who | Claim |
|---|---|---|
| Subject | The user on whose behalf the action is taken | `sub` |
| Actor | The service/client performing the delegation | `act.sub` |

### Delegation Chain

```
User (sub) → Frontend (act) → Backend (act.act) → Database
```

Each hop in the chain adds an `act` claim:

```json
{
  "sub": "user-uuid",
  "act": {
    "sub": "frontend-client",
    "act": {
      "sub": "backend-service"
    }
  }
}
```

### Delegation Depth Limit

```yaml
token_exchange:
  max_delegation_depth: 3
  reject_exceeding_depth: true
```

## Act Claim

### Structure

```json
{
  "act": {
    "sub": "frontend-service-client-id",
    "aud": "https://auth.ggid.example.com",
    "iss": "https://auth.ggid.example.com"
  }
}
```

### Validation

```go
func validateActClaim(claims jwt.MapClaims) error {
    act, ok := claims["act"].(map[string]interface{})
    if !ok {
        return nil  // No act claim (direct access, not delegated)
    }
    
    // Check delegation depth
    depth := getDelegationDepth(claims)
    if depth > maxDepth {
        return ErrMaxDelegationDepthExceeded
    }
    
    // Verify actor is authorized to exchange
    actorSub := act["sub"].(string)
    client := getClient(actorSub)
    if !client.TokenExchangeAllowed {
        return ErrClientCannotExchange
    }
    
    return nil
}

func getDelegationDepth(claims jwt.MapClaims) int {
    depth := 0
    current := claims
    for {
        act, ok := current["act"].(map[string]interface{})
        if !ok {
            break
        }
        depth++
        current = act
    }
    return depth
}
```

## GGID Token Exchange Implementation

### Handler

```go
func (s *OAuthService) HandleTokenExchange(r *http.Request) (*TokenResponse, error) {
    // Parse request
    subjectToken := r.FormValue("subject_token")
    subjectTokenType := r.FormValue("subject_token_type")
    audience := r.FormValue("audience")
    scope := r.FormValue("scope")
    actorToken := r.FormValue("actor_token")
    
    // Validate client
    client := authenticateClient(r)
    if client == nil {
        return nil, ErrInvalidClient
    }
    if !client.TokenExchangeAllowed {
        return nil, ErrUnauthorizedClient
    }
    
    // Validate subject token
    subjectClaims, err := s.parseToken(subjectToken)
    if err != nil {
        return nil, ErrInvalidToken
    }
    
    // Build new token
    newClaims := jwt.MapClaims{
        "iss": s.config.Issuer,
        "sub": subjectClaims["sub"],  // Preserve original subject
        "aud": audience,
        "exp": time.Now().Add(15 * time.Minute).Unix(),
        "iat": time.Now().Unix(),
        "scope": scope,
    }
    
    // Add act claim (delegation)
    if actorToken != "" {
        actorClaims, _ := s.parseToken(actorToken)
        newClaims["act"] = map[string]interface{}{
            "sub": actorClaims["sub"],
        }
    } else {
        // Client is the actor
        newClaims["act"] = map[string]interface{}{
            "sub": client.ID,
        }
    }
    
    // Copy delegation chain
    if existingAct, ok := subjectClaims["act"]; ok {
        newClaims["act"].(map[string]interface{})["act"] = existingAct
    }
    
    // Check delegation depth
    if getDelegationDepth(newClaims) > s.config.MaxDelegationDepth {
        return nil, ErrMaxDelegationDepthExceeded
    }
    
    // Sign new token
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, newClaims)
    token.Header["kid"] = s.config.KeyID
    signed, _ := token.SignedString(s.config.PrivateKey)
    
    return &TokenResponse{
        AccessToken:       signed,
        IssuedTokenType:   "urn:ietf:params:oauth:token-type:access_token",
        TokenType:         "Bearer",
        ExpiresIn:         900,
        Scope:             scope,
        Audience:          audience,
    }, nil
}
```

### Configuration

```yaml
token_exchange:
  enabled: true
  max_delegation_depth: 3
  allowed_clients:
    - "frontend-service"
    - "backend-service"
  allowed_audiences:
    - "https://backend.ggid.example.com"
    - "https://audit.ggid.example.com"
  token_lifetime: 15m
  audit: true
```

## Best Practices

1. **Limit delegation depth** — Prevent infinite delegation chains
2. **Validate subject token** — Ensure it's valid and not expired
3. **Check client authorization** — Not all clients can exchange tokens
4. **Restrict audiences** — Only allow exchange for known audiences
5. **Preserve subject** — Original user identity must be preserved
6. **Record act claim** — Always record who performed the exchange
7. **Audit all exchanges** — Log every token exchange event
8. **Short token lifetime** — Exchanged tokens should be short-lived
9. **Scope narrowing** — Exchanged token should have ≤ original scopes
10. **Test delegation chain** — Verify depth limits work correctly