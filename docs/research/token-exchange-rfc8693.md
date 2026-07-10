# RFC 8693: OAuth 2.0 Token Exchange — Research & GGID Implementation Design

> **Status**: Research / Design Document
> **RFC**: [RFC 8693](https://datatracker.ietf.org/doc/html/rfc8693) — "OAuth 2.0 Token Exchange" (January 2020)
> **Category**: Standards Track
> **Authors**: M. Jones, A. Nadalin, M. Machulak, D. Campbell
> **Date**: 2025-01

---

## Table of Contents

1. [Overview](#1-overview)
2. [Token Exchange Flow](#2-token-exchange-flow)
3. [Delegation Semantics](#3-delegation-semantics)
4. [Use Cases](#4-use-cases)
5. [GGID Implementation Design](#5-ggid-implementation-design)
6. [Security Considerations](#6-security-considerations)
7. [Comparison with Other Implementations](#7-comparison-with-other-implementations)
8. [GGID Roadmap](#8-ggid-roadmap)

---

## 1. Overview

### 1.1 What Is RFC 8693?

RFC 8693 defines the **OAuth 2.0 Token Exchange** grant type — a standardized protocol
for exchanging one security token for another. It implements a Security Token Service
(STS) pattern within the OAuth 2.0 framework, allowing a client to present an existing
token (the **subject token**) and receive a new token with different scopes, audience,
or identity claims.

The core exchange operation:

```
subject_token  +  [actor_token]  →  new_token (reduced scope / new audience / act chain)
```

### 1.2 Purpose

Token Exchange solves several critical problems in distributed identity systems:

| Problem | Solution |
|---------|----------|
| **Privilege propagation** | Pass user identity through a chain of microservices without forwarding the original (high-privilege) token |
| **Audience restriction** | Issue a token that is valid *only* for a specific downstream service |
| **Privilege reduction** | Exchange a broad-scope token for a narrow-scope token (defense-in-depth) |
| **Delegation tracking** | Record *who acted on behalf of whom* via the `act` claim chain |
| **Cross-domain trust** | Exchange a token from one trusted domain into a token for another domain |
| **Impersonation** | Admin acts as a user while the audit trail records both identities |

### 1.3 Key Concepts

- **Subject Token**: The primary token being exchanged. Represents the entity for whom
  the new token is issued.
- **Actor Token** *(optional)*: A token representing the party performing the exchange.
  Present in delegation scenarios where a service acts on behalf of a user.
- **Act Claim**: A JWT claim (`act`) embedded in the issued token that records the
  actor chain — providing a cryptographic audit trail of delegation.
- **Token Type Identifiers**: URN-based identifiers that specify what kind of token is
  being presented and what kind is being requested.

### 1.4 Why Not Just Forward the Original Token?

Forwarding a user's access token to downstream services is an anti-pattern:

1. **Scope creep** — The downstream service receives *all* the user's permissions,
   violating least-privilege.
2. **No audience binding** — The token may be replayed against any service that trusts
   the issuer.
3. **No delegation record** — The downstream service cannot prove *who* delegated to it.
4. **Extended lifetime** — Long-lived tokens in transit increase replay risk.

Token Exchange addresses all four: the exchanged token has reduced scope, a specific
audience, an `act` claim recording delegation, and a shorter TTL.

---

## 2. Token Exchange Flow

### 2.1 Endpoint

Token Exchange uses the standard OAuth 2.0 token endpoint:

```
POST /oauth/token HTTP/1.1
Host: oauth.ggid.dev
Content-Type: application/x-www-form-urlencoded
X-Tenant-ID: 00000000-0000-0000-0000-000000000001
```

### 2.2 Grant Type

```
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
```

This URN must be registered in the client's `grant_types` list and handled by the
authorization server's token endpoint.

### 2.3 Request Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `grant_type` | **Yes** | Must be `urn:ietf:params:oauth:grant-type:token-exchange` |
| `subject_token` | **Yes** | The token being exchanged |
| `subject_token_type` | **Yes** | Type identifier of the subject token |
| `actor_token` | No | Token representing the actor (delegator/service) |
| `actor_token_type` | No (required if `actor_token` present) | Type of the actor token |
| `resource` | No | URI identifying the target service/resource |
| `audience` | No | Logical name of the target audience |
| `scope` | No | Space-delimited list of requested scopes |
| `requested_token_type` | No | Desired type of the issued token |

### 2.4 Token Type Identifiers

RFC 8693 defines the following type URNs:

| Identifier | Token Type |
|------------|------------|
| `urn:ietf:params:oauth:token-type:access_token` | OAuth 2.0 access token |
| `urn:ietf:params:oauth:token-type:refresh_token` | OAuth 2.0 refresh token |
| `urn:ietf:params:oauth:token-type:id_token` | OIDC ID Token |
| `urn:ietf:params:oauth:token-type:saml2` | SAML 2.0 assertion |
| `urn:ietf:params:oauth:token-type:jwt` | Generic JWT (no specific token type) |

### 2.5 Example Request

```http
POST /oauth/token HTTP/1.1
Host: oauth.ggid.dev
Content-Type: application/x-www-form-urlencoded
X-Tenant-ID: 00000000-0000-0000-0000-000000000001

grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Atoken-exchange
&subject_token=eyJhbGciOiJSUzI1NiIsImtpZCI6ImdnLW9hdXRoLWtpZC0yMDI1In0...
&subject_token_type=urn%3Aietf%3Aparams%3Aoauth%3Atoken-type%3Aaccess_token
&resource=https%3A%2F%2Fapi.orders.ggid.dev
&audience=orders-service
&scope=orders.read%20orders.write
&requested_token_type=urn%3Aietf%3Aparams%3Aoauth%3Atoken-type%3Aaccess_token
```

Decoded for readability:

```
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
subject_token=<JWT access token>
subject_token_type=urn:ietf:params:oauth:token-type:access_token
resource=https://api.orders.ggid.dev
audience=orders-service
scope=orders.read orders.write
requested_token_type=urn:ietf:params:oauth:token-type:access_token
```

### 2.6 Example Response

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6ImdnLW9hdXRoLWtpZC0yMDI1In0.eyJpc3MiOiJodHRwczovL29hdXRoLmdnaWQuZGV2Iiwic3ViIjoidXNlci0xMjM0NTYiLCJhdWQiOiJvcmRlcnMtc2VydmljZSIsInNjb3BlIjoib3JkZXJzLnJlYWQgb3JkZXJzLndyaXRlIiwiZXhwIjoxNzM3MDAwMDAwLCJpYXQiOjE3MzY5OTY0MDAsImFjdCI6eyJzdWIiOiJhcGktZ2F0ZXdheSJ9LCJqdGkiOiIwMWgxMngzNTBsIn0...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "N_A",
  "expires_in": 3600,
  "scope": "orders.read orders.write"
}
```

**Key response fields:**

| Field | Description |
|-------|-------------|
| `access_token` | The newly issued token |
| `issued_token_type` | URN identifying what kind of token was issued (echoes `requested_token_type` or the server's default) |
| `token_type` | `"N_A"` (not applicable) per RFC 8693; or `"Bearer"` if the token is a bearer token |
| `expires_in` | Lifetime of the issued token in seconds |
| `scope` | The actual scopes granted (may be a subset of what was requested) |
| `refresh_token` | Optional, only if the server supports refresh for exchanged tokens |

### 2.7 Decoded Issued JWT

```json
{
  "iss": "https://oauth.ggid.dev",
  "sub": "user-123456",
  "aud": "orders-service",
  "scope": "orders.read orders.write",
  "exp": 1737000000,
  "iat": 1736996400,
  "jti": "01h12x350l",
  "act": {
    "sub": "api-gateway"
  }
}
```

---

## 3. Delegation Semantics

### 3.1 The `act` Claim

The `act` (actor) claim is the heart of delegation in RFC 8693. It records the identity
of the party that requested the token exchange — the entity *acting on behalf of* the
subject.

```json
{
  "iss": "https://oauth.ggid.dev",
  "sub": "user-123",
  "aud": "billing-service",
  "act": {
    "sub": "api-gateway"
  }
}
```

**Reading this token:**
- `sub: "user-123"` — The token represents user-123's identity.
- `act.sub: "api-gateway"` — The API Gateway exchanged this token on user-123's behalf.
- The billing service knows: "The API Gateway is making this request on behalf of user-123."

### 3.2 Impersonation vs Delegation

RFC 8693 distinguishes two modes:

**Impersonation** — No `act` claim:
```json
{
  "sub": "user-123",
  "aud": "target-service"
}
```
The service receives a token that *looks like* it came directly from user-123. The actor
identity is not embedded in the token itself (though the authorization server logs it).
This is appropriate when the actor is fully trusted and the downstream service should
treat the request identically to a direct user request.

**Delegation** — With `act` claim:
```json
{
  "sub": "user-123",
  "aud": "target-service",
  "act": {
    "sub": "service-A"
  }
}
```
The downstream service can see that service-A is acting on behalf of user-123. This is
appropriate for service-to-service flows where the delegation chain must be visible.

### 3.3 Nested Delegation (Act Chain)

When a token is exchanged multiple times through a chain of services, each exchange
appends to the `act` chain:

```
Frontend → API Gateway → Orders Service → Payment Service
```

**Token issued to API Gateway (1st exchange):**
```json
{
  "sub": "user-123",
  "aud": "api-gateway",
  "act": null
}
```

**Token issued to Orders Service (2nd exchange):**
```json
{
  "sub": "user-123",
  "aud": "orders-service",
  "act": {
    "sub": "api-gateway"
  }
}
```

**Token issued to Payment Service (3rd exchange):**
```json
{
  "sub": "user-123",
  "aud": "payment-service",
  "act": {
    "sub": "orders-service",
    "act": {
      "sub": "api-gateway"
    }
  }
}
```

The nested `act` chain provides a complete delegation audit trail. The Payment Service
can verify the entire chain: user-123 delegated to api-gateway, which delegated to
orders-service, which is now calling payment-service.

### 3.4 Act Claim Rules

1. **Only the most recent actor is top-level.** The outermost `act.sub` is the
   immediate caller; inner entries represent prior delegations.
2. **The subject never changes.** `sub` always identifies the original user, not a
   service.
3. **Depth limits are recommended.** A deeply nested chain may indicate a misconfigured
   architecture. RFC 8693 recommends authorization servers enforce a maximum chain depth
   (typically 5-10 levels).
4. **`act` may contain additional claims.** Beyond `sub`, the `act` object may include
   `iss`, `aud`, or custom claims about the actor.

---

## 4. Use Cases

### 4.1 Service-to-Service Token Exchange

**Scenario:** A web frontend holds a user's JWT. It calls the API Gateway, which needs
to call an Orders Service. The Orders Service needs to call a Payment Service. Each
service should receive a token scoped *only* for itself, with the delegation chain
recorded.

```
  Frontend          API Gateway         Orders Service      Payment Service
     |                    |                    |                    |
     |  HTTP + Bearer     |                    |                    |
     |  (user JWT)        |                    |                    |
     |------------------->|                    |                    |
     |                    |                    |                    |
     |              Exchange user JWT         |                    |
     |              for gateway-scoped token   |                    |
     |              (act: none)                |                    |
     |                    |---POST /oauth/token-->                   |
     |                    |<--exchanged token---                    |
     |                    |                    |                    |
     |                    |  HTTP + Bearer     |                    |
     |                    |  (orders-scoped)   |                    |
     |                    |------------------->|                    |
     |                    |                    |                    |
     |                    |              Exchange again             |
     |                    |              (act: api-gateway)          |
     |                    |                    |---POST /oauth/token-->
     |                    |                    |<--exchanged token---|
     |                    |                    |                    |
     |                    |                    |  HTTP + Bearer     |
     |                    |                    |  (payment-scoped)  |
     |                    |                    |------------------->|
     |                    |                    |                    |
     |                    |                    |        Process    |
     |                    |                    |<-------------------|
     |                    |<-------------------|                    |
     |<-------------------|                    |                    |
```

**Why not forward the original token?** The original user JWT has `scope: "openid profile email users.read users.write"`. The Payment Service does not need (and should not have) `users.read`. The exchanged token has `scope: "payment.charge"` with `aud: "payment-service"` only.

### 4.2 Token Trading Down (Privilege Reduction)

**Scenario:** An admin user has a token with full administrative scopes. A background
job needs to export audit logs — a read-only operation. The job runner exchanges the
admin token for a minimal read-only token.

```
  Admin User         Background Job Runner     Audit Service
     |                         |                     |
     |  Admin JWT              |                     |
     |  (scope: *:admin)       |                     |
     |------------------------>|                     |
     |                         |                     |
     |                   Exchange for reduced scope  |
     |                   requested_scope: audit.read  |
     |                         |---POST /oauth/token-->
     |                         |<--reduced token------|
     |                         |                     |
     |                         |  Bearer (audit.read)|
     |                         |------------------->|
     |                         |                     |
     |                         |    Export logs     |
     |                         |<-------------------|
```

**Issued token:**
```json
{
  "sub": "admin-user-001",
  "aud": "audit-service",
  "scope": "audit.read",
  "exp": 1736998200,
  "iat": 1736996400,
  "act": {
    "sub": "job-runner-export-audit"
  }
}
```

**Key design principles:**
- Exchanged scope is always a **subset** of the subject token's scope. The authorization
  server must reject requests that try to widen scope.
- TTL is shorter (e.g., 30 minutes vs. 8 hours for the admin token).
- The `act` claim records the job runner identity.

### 4.3 Cross-Domain Token Exchange

**Scenario:** A multinational enterprise has subsidiaries "Company A" (auth.ggid-a.com)
and "Company B" (auth.ggid-b.com). An employee of Company A needs to access a Company B
API. Company B's OAuth server trusts Company A's tokens and exchanges them.

```
  Employee (Domain A)     Domain A AS        Domain B AS        Domain B API
       |                      |                   |                   |
       |  Login to Domain A   |                   |                   |
       |<----A JWT----------->|                   |                   |
       |                      |                   |                   |
       |  Exchange A JWT for B token              |                   |
       |------------------------------------->|    |                   |
       |                      |                   |                   |
       |              Domain B validates A JWT     |                   |
       |              via federation trust         |                   |
       |                      |                   |                   |
       |<-----B JWT issued-------------------------|                   |
       |                      |                   |                   |
       |  Request Domain B API with B JWT         |                   |
       |--------------------------------------------------------->|
       |                      |                   |                   |
```

**Trust requirements:**
- Domain B must have Domain A's JWKS public keys (via OIDC federation or direct JWKS
  import).
- Domain B maps Domain A's subject to a local identity (potentially via SCIM user
  provisioning).
- The exchanged token from Domain B has Domain B's issuer, not Domain A's.

**Example Domain B issued token:**
```json
{
  "iss": "https://auth.ggid-b.com",
  "sub": "federated:a-user-001@domain-a",
  "aud": "domain-b-api",
  "scope": "data.read",
  "act": {
    "sub": "federation-bridge",
    "iss": "https://auth.ggid-b.com"
  },
  "federation": {
    "original_iss": "https://auth.ggid-a.com",
    "original_sub": "a-user-001"
  }
}
```

### 4.4 Impersonation Workflows

**Scenario:** A support engineer needs to reproduce a user's bug by acting as that user.
The admin uses an impersonation token exchange, which records the admin's identity in
the `act` claim for audit, but the downstream service sees the user's identity as `sub`.

```
  Support Admin        GGID OAuth Server       Target App
       |                      |                    |
       |  Exchange my token    |                    |
       |  for user-123's token |                    |
       |  (act: admin-001)     |                    |
       |--------------------->|                    |
       |                      |                    |
       |               Verify admin has            |
       |               impersonate permission       |
       |                      |                    |
       |<----Impersonation JWT--                    |
       |                      |                    |
       |  Request app as user-123                  |
       |------------------------------------->|     |
       |                      |                    |
       |                      |   Audit log:       |
       |                      |   admin-001 acted   |
       |                      |   as user-123       |
       |                      |                    |
```

**Impersonation token (no `act` in standard impersonation):**
```json
{
  "sub": "user-123",
  "aud": "target-app",
  "scope": "profile data.read",
  "exp": 1736997300,
  "iat": 1736996400,
  "jti": "imp-01h12x350l"
}
```

**Delegation token (admin visible in `act`):**
```json
{
  "sub": "user-123",
  "aud": "target-app",
  "scope": "profile data.read",
  "exp": 1736997300,
  "iat": 1736996400,
  "jti": "imp-01h12x350l",
  "act": {
    "sub": "admin-001",
    "impersonation": true
  }
}
```

**Security controls for impersonation:**
1. **Explicit permission check** — The admin must have `iam:user:impersonate` permission
   (checked via Policy Engine).
2. **Time limits** — Impersonation tokens have a short TTL (5-15 minutes).
3. **Audit logging** — Every impersonation exchange is recorded with admin identity,
   target user, scopes, timestamp, and client IP.
4. **Consent (optional)** — Some implementations require the user's consent before
   impersonation.
5. **Rate limiting** — Impersonation requests are rate-limited per admin.

---

## 5. GGID Implementation Design

### 5.1 Current State

GGID already has a **skeleton implementation** of RFC 8693:

- `services/oauth/internal/service/oauth_service.go` defines:
  - `TokenExchangeRequestRFC8693` struct (lines 900-910)
  - `ExchangeToken()` method (lines 913-941)
- The token endpoint handler (`services/oauth/internal/server/server.go`, line 293)
  already handles a `switch grantType` with `authorization_code`, `refresh_token`,
  `client_credentials`, `urn:ietf:params:oauth:grant-type:device_code`, and
  `urn:ietf:params:oauth:grant-type:jwt-bearer`.

**Current gaps in the skeleton:**
1. The `ExchangeToken` method issues `"exchanged_" + uuid.New().String()` — a placeholder,
   not a real JWT.
2. No `act` claim is embedded in the issued token.
3. No actor token validation.
4. No scope narrowing enforcement.
5. No audience restriction.
6. No policy engine integration.
7. The token endpoint handler does not route `urn:ietf:params:oauth:grant-type:token-exchange`
   to `ExchangeToken()`.
8. No audit logging.

### 5.2 Architecture

```
                          ┌─────────────────────────────────────────┐
                          │         services/oauth (HTTP :9005)      │
                          │                                          │
  POST /oauth/token ────> │  server.go                               │
  grant_type=             │    └─ switch grantType                   │
    token-exchange        │         └─ case "urn:...:token-exchange" │
                          │              └─ oauthSvc.ExchangeToken() │
                          │                                          │
                          │  service/oauth_service.go                │
                          │    └─ ExchangeToken()                    │
                          │         ├─ 1. Validate subject_token JWT │
                          │         │    (pkg/crypto + KeyProvider)  │
                          │         ├─ 2. Validate actor_token       │
                          │         │    (if present)                │
                          │         ├─ 3. Policy check               │
                          │         │    (services/policy gRPC)      │
                          │         ├─ 4. Enforce scope narrowing    │
                          │         ├─ 5. Build new JWT with act     │
                          │         ├─ 6. Sign with KeyProvider      │
                          │         ├─ 7. Audit log to DB            │
                          │         └─ 8. Return TokenResponse       │
                          └─────────────────────────────────────────┘
```

### 5.3 Endpoint Registration

Add a new case to the existing grant type switch in
`services/oauth/internal/server/server.go` (currently at line 325):

```go
case "urn:ietf:params:oauth:grant-type:token-exchange":
    resp, tokenErr = oauthSvc.ExchangeToken(ctx, &service.TokenExchangeRequestRFC8693{
        TenantID:           tenantID,
        SubjectToken:       r.FormValue("subject_token"),
        SubjectTokenType:   r.FormValue("subject_token_type"),
        ActorToken:         r.FormValue("actor_token"),
        ActorTokenType:     r.FormValue("actor_token_type"),
        Resource:           r.FormValue("resource"),
        Audience:           r.FormValue("audience"),
        Scope:              scopes,
        RequestedTokenType: r.FormValue("requested_token_type"),
    })
```

Also register the grant type in the OIDC discovery document
(`domain.OIDCDiscoveryConfig.GrantTypesSupported`) and validate that the client's
`GrantTypes` list includes it (via `OAuthClient.SupportsGrantType()`).

### 5.4 Enhanced ExchangeToken Implementation

The existing `ExchangeToken()` method needs to be upgraded from a placeholder to a
full implementation. Here is the production design:

```go
// TokenExchangeRequestRFC8693 implements RFC 8693 token exchange parameters.
// Already defined at services/oauth/internal/service/oauth_service.go:900
type TokenExchangeRequestRFC8693 struct {
    TenantID           uuid.UUID
    SubjectToken       string
    SubjectTokenType   string
    ActorToken         string
    ActorTokenType     string
    Resource           string
    Audience           string
    Scope              []string
    RequestedTokenType string
}

// ExchangeToken implements RFC 8693 token exchange.
func (s *OAuthService) ExchangeToken(ctx context.Context, req *TokenExchangeRequestRFC8693) (*TokenResponse, error) {
    // ── 1. Validate required parameters ──
    if req.SubjectToken == "" {
        return nil, errors.InvalidArgument("subject_token is required")
    }
    if req.SubjectTokenType == "" {
        return nil, errors.InvalidArgument("subject_token_type is required")
    }

    // ── 2. Parse and validate the subject token (JWT) ──
    // ParseAccessToken already exists in oauth_service.go.
    subjectClaims, err := s.ParseAccessToken(req.SubjectToken)
    if err != nil {
        return nil, errors.Unauthenticated("invalid subject_token: " + err.Error())
    }

    sub := getStringClaim(subjectClaims, "sub")
    if sub == "" {
        return nil, errors.InvalidArgument("subject_token missing 'sub' claim")
    }

    subjectScope := strings.Split(getStringClaim(subjectClaims, "scope"), " ")

    // ── 3. Parse and validate the actor token (if present) ──
    var actorSub string
    if req.ActorToken != "" {
        actorClaims, err := s.ParseAccessToken(req.ActorToken)
        if err != nil {
            return nil, errors.Unauthenticated("invalid actor_token: " + err.Error())
        }
        actorSub = getStringClaim(actorClaims, "sub")
        if actorSub == "" {
            return nil, errors.InvalidArgument("actor_token missing 'sub' claim")
        }
    }

    // ── 4. Enforce scope narrowing (requested ⊆ subject's scopes) ──
    grantedScope := narrowScopes(req.Scope, subjectScope)
    if len(grantedScope) == 0 {
        return nil, errors.Forbidden("requested scope exceeds subject token scope")
    }

    // ── 5. Determine audience and TTL ──
    audience := req.Audience
    if audience == "" {
        audience = req.Resource
    }
    if audience == "" {
        return nil, errors.InvalidArgument("either resource or audience is required")
    }

    // Exchanged tokens are short-lived: min(subject TTL, 1 hour)
    ttl := 3600 // seconds
    subjectExp := getInt64Claim(subjectClaims, "exp")
    if subjectExp > 0 {
        remaining := subjectExp - time.Now().Unix()
        if remaining < int64(ttl) {
            ttl = int(remaining)
        }
    }

    // ── 6. Build the act claim chain ──
    actClaim := buildActClaim(subjectClaims, actorSub)

    // ── 7. Construct and sign the new JWT ──
    now := time.Now()
    claims := jwt.MapClaims{
        "iss":   s.issuer,
        "sub":   sub,
        "aud":   audience,
        "scope": strings.Join(grantedScope, " "),
        "iat":   now.Unix(),
        "exp":   now.Add(time.Duration(ttl) * time.Second).Unix(),
        "jti":   uuid.New().String(),
    }
    if actClaim != nil {
        claims["act"] = actClaim
    }

    // Sign using the existing KeyProvider (RS256).
    // KeyProvider.PrivateKey() and KeyID() are defined in domain/models.go.
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = s.keyProvider.KeyID()
    signedToken, err := token.SignedString(s.keyProvider.PrivateKey())
    if err != nil {
        return nil, fmt.Errorf("failed to sign exchanged token: %w", err)
    }

    // ── 8. Audit log (best-effort, non-blocking) ──
    // Uses the same pattern as the NATS audit publisher in pkg/audit.
    go s.auditTokenExchange(req.TenantID, sub, actorSub, audience, grantedScope)

    // ── 9. Return the RFC 8693 response ──
    return &TokenResponse{
        AccessToken: signedToken,
        TokenType:   "N_A",
        ExpiresIn:   ttl,
        Scope:       strings.Join(grantedScope, " "),
    }, nil
}

// narrowScopes returns the intersection of requested and allowed scopes.
func narrowScopes(requested, allowed []string) []string {
    allowedSet := make(map[string]bool, len(allowed))
    for _, s := range allowed {
        allowedSet[s] = true
    }
    var result []string
    for _, s := range requested {
        if allowedSet[s] {
            result = append(result, s)
        }
    }
    return result
}

// buildActClaim constructs or extends the act claim chain.
// If the subject token already has an act chain, the new actor is prepended.
func buildActClaim(subjectClaims jwt.MapClaims, actorSub string) map[string]any {
    if actorSub == "" {
        // No actor: check if subject token had an existing act chain to carry forward.
        if existingAct, ok := subjectClaims["act"].(map[string]any); ok {
            return existingAct
        }
        return nil
    }

    act := map[string]any{
        "sub": actorSub,
    }

    // If the subject token had an existing act chain, nest it.
    if existingAct, ok := subjectClaims["act"].(map[string]any); ok {
        act["act"] = existingAct
    }

    return act
}
```

### 5.5 Policy Engine Integration

The Policy Service (`services/policy`) already exposes a `Check` method via gRPC
at `:9070`. The request/response is defined in
`services/policy/internal/domain/models.go`:

```go
// CheckRequest is the input for a permission check.
type CheckRequest struct {
    UserID       uuid.UUID
    TenantID     uuid.UUID
    ResourceType string
    Action       string
    Resource     string
    Conditions   map[string]any
}
```

To integrate token exchange authorization, add a policy check before issuing the
exchanged token:

```go
// In ExchangeToken(), after validating tokens but before issuing:

// Check policy: is the subject allowed to exchange for this audience/scope?
policyReq := &policy.CheckRequest{
    TenantId:     req.TenantID.String(),
    UserId:       sub,
    ResourceType: "oauth:token-exchange",
    Action:       "exchange",
    Resource:     audience,
    Conditions: map[string]any{
        "requested_scope":  grantedScope,
        "requested_audience": audience,
        "subject_token_type": req.SubjectTokenType,
    },
}

policyResp, err := s.policyClient.Check(ctx, policyReq)
if err != nil {
    return nil, fmt.Errorf("policy check failed: %w", err)
}
if !policyResp.GetAllowed() {
    return nil, errors.Forbidden("token exchange denied by policy: " + policyResp.GetReason())
}
```

**Example ABAC policy for token exchange:**

```json
{
  "name": "allow-token-exchange-to-downstream",
  "effect": "allow",
  "actions": ["exchange"],
  "resources": ["orders-service", "payment-service", "audit-service"],
  "conditions": {
    "all_of": [
      {
        "field": "subject_token_type",
        "op": "eq",
        "value": "urn:ietf:params:oauth:token-type:access_token"
      },
      {
        "field": "requested_scope",
        "op": "subset_of",
        "value": ["orders.read", "orders.write", "payment.charge", "audit.read"]
      }
    ]
  },
  "priority": 100
}
```

### 5.6 Database Schema

Create a `token_exchange_audit` table for compliance and forensics:

```sql
-- services/oauth/migrations/00X_token_exchange.sql

CREATE TABLE IF NOT EXISTS token_exchange_audit (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    subject_sub     VARCHAR(255) NOT NULL,
    actor_sub       VARCHAR(255),
    audience        VARCHAR(255) NOT NULL,
    granted_scope   TEXT[] NOT NULL DEFAULT '{}',
    requested_scope TEXT[] NOT NULL DEFAULT '{}',
    subject_token_type  VARCHAR(255) NOT NULL,
    requested_token_type VARCHAR(255),
    issued_jti      VARCHAR(255) NOT NULL,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    client_ip       INET,
    success         BOOLEAN NOT NULL DEFAULT TRUE,
    error_reason    TEXT
);

CREATE INDEX idx_tea_tenant_subject ON token_exchange_audit(tenant_id, subject_sub);
CREATE INDEX idx_tea_tenant_audience ON token_exchange_audit(tenant_id, audience);
CREATE INDEX idx_tea_issued_at ON token_exchange_audit(issued_at DESC);
```

### 5.7 Discovery Document Update

Update the OIDC discovery document to advertise token exchange support:

```go
// In services/oauth/internal/server/server.go, buildHandler() discovery handler:
GrantTypesSupported: []string{
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:device_code",
    "urn:ietf:params:oauth:grant-type:jwt-bearer",
    "urn:ietf:params:oauth:grant-type:token-exchange",  // NEW
},
```

### 5.8 Key Provider Reuse

The OAuth service already loads RSA keys via `keyProvider` (defined in
`services/oauth/internal/server/server.go:41`) which implements `domain.KeyProvider`:

```go
type KeyProvider interface {
    PublicKey() *rsa.PublicKey
    PrivateKey() *rsa.PrivateKey
    KeyID() string
}
```

Exchanged tokens use the **same signing key** as regular OAuth tokens. This is correct
because:
1. The `aud` claim binds the token to a specific service, preventing cross-service
   replay.
2. Downstream services validate the token via the same JWKS endpoint
   (`/oauth/jwks`).
3. Key rotation applies to exchanged tokens automatically.

For multi-tenant deployments that require per-tenant signing keys, the `KeyProvider`
interface can be extended to a `TenantKeyProvider` that selects keys based on
`X-Tenant-ID`. This is a future enhancement.

---

## 6. Security Considerations

### 6.1 Token Replay Prevention

| Mechanism | Implementation |
|-----------|---------------|
| **`jti` (JWT ID)** | Every exchanged token gets a unique `jti` (UUID). Downstream services should cache seen `jti` values and reject duplicates. |
| **Short TTL** | Exchanged tokens are short-lived (default 1 hour, configurable). TTL is capped to the remaining lifetime of the subject token. |
| **Audience binding** | The `aud` claim restricts the token to a single service. Any other service receiving it must reject it. |
| **Replay window** | The authorization server can enforce a minimum time between exchange requests from the same subject for the same audience (anti-DoS). |

### 6.2 Scope Narrowing Guarantee

The authorization server **must** enforce that the issued token's scope is a subset
of the subject token's scope. The `narrowScopes()` function (Section 5.4) performs
this intersection. If the requested scope is entirely outside the subject's scope,
the request fails with a `403 Forbidden` error.

**Violations to prevent:**
- Requesting `admin.*` when the subject only has `user.*`
- Requesting `payment.write` when the subject only has `payment.read`
- Omitting scope (defaults to subject's full scope — acceptable but logged)

### 6.3 Audience Restriction

```
Rule: The issued token's audience must not be broader than the subject token's audience.
```

If the subject token has `aud: ["api-gateway", "orders-service"]`, the exchanged token
may target `orders-service` but not `admin-panel` (which was not in the original
audience).

### 6.4 Act Chain Depth Limits

Nested delegation can grow unboundedly. Recommended limits:
- **Maximum chain depth: 5** (configurable per tenant).
- If the subject token's act chain already has 5 levels, reject the exchange.
- Log a warning if chain depth exceeds 3.

### 6.5 Audit Trail Requirements

Every token exchange must produce an audit record with:

| Field | Source |
|-------|--------|
| `tenant_id` | `X-Tenant-ID` header |
| `subject_sub` | `sub` claim from subject token |
| `actor_sub` | `sub` claim from actor token (or "self" if no actor) |
| `audience` | `audience` or `resource` parameter |
| `granted_scope` | Final scope intersection |
| `issued_jti` | `jti` of the new token |
| `issued_at` / `expires_at` | Token timestamps |
| `client_ip` | `X-Forwarded-For` or `RemoteAddr` |
| `success` / `error_reason` | Whether the exchange succeeded |

These records are stored in the `token_exchange_audit` table and can also be published
to NATS JetStream via the existing `pkg/audit` publisher.

### 6.6 Key Management

| Concern | Recommendation |
|---------|----------------|
| **Signing key rotation** | Rotate RSA keys every 90 days. Keep old keys active for 2x max TTL to allow in-flight tokens to expire gracefully. |
| **Key compromise** | If the signing key is compromised, immediately rotate and revoke all active tokens via a `jti` blacklist or a global revocation event. |
| **Algorithm pinning** | Always use RS256 (or EdDSA). Never allow `alg: none` or HMAC algorithms for JWTs. |
| **JWKS endpoint security** | Serve `/oauth/jwks` over TLS only. Cache responses with short TTL (5 min). |

### 6.7 Rate Limiting

Token exchange endpoints should be rate-limited:
- Per-subject: max 60 exchanges per minute (prevents token flooding).
- Per-client: max 100 exchanges per minute.
- Per-IP: max 200 requests per minute.

GGID's existing rate limiter middleware
(`services/gateway/internal/middleware/sliding_ratelimit.go`) can be applied.

---

## 7. Comparison with Other Implementations

### 7.1 Auth0

Auth0 supports token exchange via the **client credentials** grant and a custom
"delegation" endpoint. Their approach differs from RFC 8693:

| Feature | Auth0 | RFC 8693 |
|---------|-------|----------|
| Grant type | `urn:ietf:params:oauth:grant-type:token-exchange` | Same |
| Actor token | Not supported in standard delegation | `actor_token` parameter |
| Act claim | `act` in JWT (limited support) | Full `act` chain |
| Scope narrowing | Enforced | Enforced |
| Cross-domain | Via Auth0 federation | Via OIDC federation |
| Custom claims | Rich support | Standard claims |

**Auth0 example:**
```
POST https://YOUR_DOMAIN/oauth/token
Content-Type: application/json

{
  "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
  "subject_token": "<token>",
  "subject_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "audience": "https://api.example.com",
  "scope": "read:data"
}
```

Auth0 has supported token exchange since 2023 and it is GA (generally available).

### 7.2 Keycloak

Keycloak has had **experimental** token exchange support since version 8.0 (2019). As
of Keycloak 25+ (2024), it remains flagged as preview/experimental:

| Feature | Keycloak | RFC 8693 |
|---------|----------|----------|
| Status | Preview (must enable with feature flag) | Standard |
| Actor token | Supported | `actor_token` parameter |
| Act claim | `act` claim with chain | Full `act` chain |
| Scope narrowing | Enforced | Enforced |
| Cross-domain | Via identity brokering | Via federation |
| Policy integration | Built-in authorization services | External policy engine |

**Keycloak example:**
```bash
# Must enable the preview feature first:
# bin/kc.sh start --features=token-exchange

curl -X POST https://keycloak.example.com/realms/myrealm/protocol/openid-connect/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "client_id=my-client" \
  -d "client_secret=..." \
  -d "subject_token=<token>" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "audience=target-client"
```

Keycloak's experimental status and feature-flag requirement make it less suitable
for production use compared to dedicated implementations.

### 7.3 AWS STS (Security Token Service)

AWS STS is the real-world analog that inspired RFC 8693. While not OAuth-based, the
concepts map directly:

| RFC 8693 Concept | AWS STS Equivalent |
|------------------|---------------------|
| `subject_token` | IAM role session credentials |
| `actor_token` | `AssumedRole` source identity |
| `audience` | Resource ARN |
| `scope` | IAM policy (inline or attached) |
| `act` claim | `aws:SourceIdentity` condition key |
| Token exchange endpoint | `sts:AssumeRole` API |
| Scope narrowing | IAM policy downscoping (session policy) |

**AWS STS AssumeRole example:**
```bash
aws sts assume-role \
  --role-arn "arn:aws:iam::123456789012:role/CrossAccountRole" \
  --role-session-name "MySession" \
  --duration-seconds 3600 \
  --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}'
```

AWS STS also supports **chained assumption** (equivalent to nested `act` claims) via
`AssumeRole` with session policies — each link in the chain can only reduce privileges,
never widen them. This is the same invariant as RFC 8693 scope narrowing.

### 7.4 Feature Comparison Matrix

| Feature | GGID (proposed) | Auth0 | Keycloak | AWS STS |
|---------|-----------------|-------|----------|---------|
| RFC 8693 compliant | Yes (planned) | Yes | Partial (preview) | N/A (different protocol) |
| `act` claim chain | Full | Limited | Full | Via condition keys |
| Actor token support | Phase 2 | No | Yes | N/A |
| Scope narrowing | Enforced | Enforced | Enforced | Session policies |
| Cross-domain | Phase 3 (OIDC federation) | Federation | Identity brokering | Cross-account roles |
| Policy integration | ABAC (Policy Engine) | Custom rules | Authorization Services | IAM policies |
| Audit logging | DB + NATS | Logs | Events | CloudTrail |
| Key rotation | RS256 via JWKS | Managed | Managed | KMS |

---

## 8. GGID Roadmap

### Phase 1: Basic Exchange (MVP)

**Goal:** Exchange a subject token for a scoped, audience-restricted token without
the `act` claim.

**Scope:**
- Route `urn:ietf:params:oauth:grant-type:token-exchange` in the token endpoint handler.
- Validate subject token (JWT via existing `ParseAccessToken`).
- Enforce scope narrowing (`narrowScopes()`).
- Issue real JWT (signed with `KeyProvider`, not a UUID placeholder).
- Short TTL (capped to subject token's remaining lifetime).
- Audit log to `token_exchange_audit` table.
- Register grant type in OIDC discovery document.

**Files to modify:**
| File | Change |
|------|--------|
| `services/oauth/internal/server/server.go` | Add `case "urn:...:token-exchange"` to grant type switch (~line 325) |
| `services/oauth/internal/service/oauth_service.go` | Upgrade `ExchangeToken()` method (lines 913-941) |
| `services/oauth/migrations/` | New migration for `token_exchange_audit` table |
| `services/oauth/internal/service/oauth_service.go` | Add `narrowScopes()`, `auditTokenExchange()` helpers |

**Effort:** 2-3 days

**Acceptance criteria:**
- `POST /oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`
  returns a valid JWT.
- The JWT has reduced scope (intersection of requested and subject scopes).
- The JWT has the correct `aud` claim.
- The JWT's `exp` does not exceed the subject token's `exp`.
- An audit record is written to the database.
- Existing grant types still work (regression-free).

### Phase 2: Delegation with Act Claim Chain

**Goal:** Full delegation semantics with `act` claim chain and actor token support.

**Scope:**
- Validate `actor_token` parameter.
- Build `act` claim chain via `buildActClaim()`.
- Enforce act chain depth limit (max 5).
- Policy engine integration: `CheckRequest` with `resourceType=oauth:token-exchange`.
- Publish audit events to NATS JetStream via `pkg/audit.Publisher`.
- Rate limiting on the exchange endpoint.

**Files to modify:**
| File | Change |
|------|--------|
| `services/oauth/internal/service/oauth_service.go` | Add `buildActClaim()`, actor token validation |
| `services/oauth/internal/server/server.go` | Wire Policy gRPC client |
| `services/oauth/internal/conf/conf.go` | Add `PolicyServiceAddr` config |

**Effort:** 3-4 days

**Acceptance criteria:**
- Exchanged token includes `act` claim when `actor_token` is provided.
- Nested `act` chain is correctly built from multiple exchanges.
- Chain depth > 5 returns an error.
- Policy engine approves/denies exchange based on ABAC rules.
- Audit events are published to NATS.

### Phase 3: Cross-Domain Trust via OIDC Federation

**Goal:** Exchange tokens across trusted GGID deployments or external IdPs.

**Scope:**
- OIDC Federation (RFC 8416 / draft-ietf-openid-federation) trust anchors.
- Import external JWKS for subject token validation.
- Subject mapping: map external `sub` to local federated identity.
- Issue a local JWT with the external identity recorded in a `federation` claim.
- Admin UI for configuring trust relationships (Console: Settings page).

**Files to create/modify:**
| File | Change |
|------|--------|
| `services/oauth/internal/service/federation.go` | New file: trust anchor management, JWKS import |
| `services/oauth/internal/domain/models.go` | Add `FederationTrust` domain model |
| `services/oauth/internal/repository/` | New `FederationTrustRepository` |
| `services/oauth/migrations/` | New migration for `federation_trusts` table |
| `console/src/app/settings/federation/` | New Console page |

**Effort:** 1-2 weeks

**Acceptance criteria:**
- Exchange a token from Domain A and receive a token from Domain B.
- The Domain B token has Domain B's issuer.
- The original Domain A identity is recorded in a `federation` claim.
- Trust relationships are configurable via Console.

### Summary Timeline

| Phase | Effort | Deliverable |
|-------|--------|-------------|
| Phase 1: Basic Exchange | 2-3 days | Scoped, audience-restricted JWT issuance |
| Phase 2: Delegation | 3-4 days | Act claim chain, policy integration, NATS audit |
| Phase 3: Cross-Domain | 1-2 weeks | OIDC federation, external JWKS, Console UI |
| **Total** | **~3 weeks** | **Full RFC 8693 compliance** |

---

## Appendix A: Test Vectors

### A.1 Subject Token (Input)

```json
Header: { "alg": "RS256", "kid": "gg-oauth-kid-2025", "typ": "JWT" }
Claims:
{
  "iss": "https://oauth.ggid.dev",
  "sub": "user-123456",
  "aud": "api-gateway",
  "scope": "openid profile email users.read users.write orders.read orders.write",
  "iat": 1736996400,
  "exp": 1737025200,
  "jti": "01h12x340a"
}
```

### A.2 Exchange Request

```http
POST /oauth/token HTTP/1.1
Content-Type: application/x-www-form-urlencoded
X-Tenant-ID: 00000000-0000-0000-0000-000000000001

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=<A.1 token>
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&audience=orders-service
&scope=orders.read+orders.write
&requested_token_type=urn:ietf:params:oauth:token-type:access_token
```

### A.3 Exchanged Token (Output)

```json
Header: { "alg": "RS256", "kid": "gg-oauth-kid-2025", "typ": "JWT" }
Claims:
{
  "iss": "https://oauth.ggid.dev",
  "sub": "user-123456",
  "aud": "orders-service",
  "scope": "orders.read orders.write",
  "iat": 1736996400,
  "exp": 1737000000,
  "jti": "01h12x350l",
  "act": {
    "sub": "api-gateway"
  }
}
```

### A.4 Nested Delegation (3rd Hop)

After API Gateway → Orders Service → Payment Service:

```json
Claims:
{
  "iss": "https://oauth.ggid.dev",
  "sub": "user-123456",
  "aud": "payment-service",
  "scope": "payment.charge",
  "iat": 1736996500,
  "exp": 1736999500,
  "jti": "01h12x360m",
  "act": {
    "sub": "orders-service",
    "act": {
      "sub": "api-gateway"
    }
  }
}
```

---

## Appendix B: References

1. **RFC 8693** — OAuth 2.0 Token Exchange. Jones, et al. January 2020.
   https://datatracker.ietf.org/doc/html/rfc8693

2. **RFC 9068** — JSON Web Token (JWT) Profile for OAuth 2.0 Access Tokens.
   Bertocci. May 2021.
   https://datatracker.ietf.org/doc/html/rfc9068

3. **RFC 7523** — JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication
   and Authorization Grants. Campbell, et al. May 2015.

4. **RFC 8416** — JSON Web Key (JWK) Set. Jones. July 2018.

5. **Karl McGuinness** — "Standardize `act` Across Assertion Grants and JWT Access
   Tokens." Analysis of delegation claim standardization.
   https://notes.karlmcguinness.com/notes/standardize-act-across-assertion-grants-and-jwt-access-tokens/

6. **AWS STS Documentation** — Security Token Service API Reference.
   https://docs.aws.amazon.com/STS/latest/APIReference/

7. **Keycloak Token Exchange Documentation** — Server Administration Guide.
   https://www.keycloak.org/docs/latest/securing_apps/#token-exchange

8. **Auth0 Token Exchange** — API documentation for token exchange.
   https://auth0.com/docs/get-started/authentication-and-authorization-flow/call-your-api-using-the-token-exchange-flow
