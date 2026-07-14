# OAuth 2.1 Draft Analysis & GGID Migration Audit

**Document Type:** Research / Gap Analysis
**Subject:** OAuth 2.1 (draft-ietf-oauth-v2-1) vs GGID OAuth Service
**Status:** Draft-15 (latest revision as of 2024)
**Date:** 2025-01-XX

---

## Table of Contents

1. [Overview](#1-overview)
2. [Breaking Changes from OAuth 2.0](#2-breaking-changes-from-oauth-20)
3. [Retained Features](#3-retained-features)
4. [Security Improvements Made Mandatory](#4-security-improvements-made-mandatory)
5. [GGID Current OAuth Implementation Audit](#5-ggid-current-oauth-implementation-audit)
6. [Migration Plan: OAuth 2.0 to 2.1](#6-migration-plan-oauth-20-to-21)
7. [Impact on GGID SDKs](#7-impact-on-ggid-sdks)
8. [Comparison with Other Providers](#8-comparison-with-other-providers)
9. [Timeline and Strategy](#9-timeline-and-strategy)
10. [Appendix: Code Examples](#10-appendix-code-examples)

---

## 1. Overview

### 1.1 What is OAuth 2.1?

OAuth 2.1 is not a new protocol. It is a **consolidation** of the original OAuth 2.0
Authorization Framework (RFC 6749) combined with the security best practices and
errata that have accumulated over the past decade. The IETF OAuth Working Group
published the first draft in 2020 and has iterated through 15 revisions.

The core philosophy is captured in one phrase: **"good defaults, fewer footguns."**

Rather than introducing new capabilities, OAuth 2.1:

- **Removes** grant types and flows that have proven insecure or error-prone
- **Mandates** security mechanisms that were previously optional recommendations
- **Consolidates** information spread across dozens of RFCs into a single document
- **Simplifies** the developer experience by reducing the decision space

### 1.2 Document Status

| Attribute | Value |
|---|---|
| Draft | draft-ietf-oauth-v2-1-15 |
| Published by | IETF OAuth Working Group |
| Replaces | RFC 6749 (OAuth 2.0) |
| Current Status | Active draft (not yet RFC) |
| Expected RFC | 2025 (estimated) |

### 1.3 Key Documents Consolidated

OAuth 2.1 incorporates requirements from:

| Document | Topic | Status in 2.1 |
|---|---|---|
| RFC 6749 | OAuth 2.0 core framework | Base |
| RFC 6750 | Bearer Token Usage | Incorporated |
| RFC 6819 | OAuth 2.0 Threat Model | Informative |
| RFC 7636 | PKCE (Proof Key for Code Exchange) | Mandatory |
| RFC 7662 | Token Introspection | Retained |
| RFC 7009 | Token Revocation | Retained |
| RFC 7591 | Dynamic Client Registration | Retained |
| RFC 8628 | Device Authorization Grant | Retained |
| RFC 8693 | Token Exchange | Retained |
| RFC 8252 | Native App Best Practices | Incorporated |
| RFC 9700 | OAuth 2.0 Security Best Current Practice | Incorporated |
| draft-ietf-oauth-dpop | DPoP (Demonstrating Proof-of-Possession) | Recommended |

### 1.4 Why Migrate?

Even though OAuth 2.1 has not been published as a final RFC, its requirements
are already being adopted by:

- **Major cloud providers** (Auth0, Okta, Google)
- **Security auditors** (OAuth 2.0 Security BCP, RFC 9700, is already a formal RFC)
- **Application security frameworks** (OWASP ASVS 4.0+ references OAuth 2.1)
- **Regulatory environments** (PSD2, Open Banking reference OAuth 2.1 best practices)

Failing to align with OAuth 2.1 will result in:

1. Security audit findings against RFC 9700 recommendations
2. Incompatibility with modern OAuth client libraries that default to PKCE
3. Competitive disadvantage vs. providers who already enforce these standards

---

## 2. Breaking Changes from OAuth 2.0

### 2.1 Removed Grant Types

#### 2.1.1 Implicit Grant (response_type=token)

**OAuth 2.0 behavior:** The authorization server could return an access token
directly in the redirect URI fragment (`#access_token=...`) without a token
exchange step. This was designed for browser-based (JavaScript) applications
that could not securely store a client secret.

**Why it was removed:**

- Access tokens in the URL fragment are exposed to any JavaScript running on the page
- No mechanism to bind the token to the requesting client (no PKCE)
- Vulnerable to token substitution and redirect interception attacks
- Modern browsers with CORS and SameSite cookies make the authorization code
  flow with PKCE viable for SPAs

**OAuth 2.1 replacement:** Authorization Code flow with mandatory PKCE.

**GGID Impact:**

GGID's authorize endpoint at `services/oauth/internal/server/server.go:177` already
**rejects** `response_type=token`:

```go
// server.go line 177-179
if responseType != "code" {
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_response_type"})
    return
}
```

However, the OIDC discovery document at `oauth_service.go:293` still **advertises**
`"token"` and `"id_token"` as supported response types:

```go
// oauth_service.go line 293
ResponseTypesSupported: []string{"code", "token", "id_token"},
```

This is a **discovery document inconsistency** -- the server rejects what it advertises.

**Status:** Mostly compliant. Need to fix the discovery document.

#### 2.1.2 Resource Owner Password Credentials (ROPC, grant_type=password)

**OAuth 2.0 behavior:** The client collects the user's username and password
directly and sends them to the token endpoint, receiving tokens in exchange.

**Why it was removed:**

- Breaks the principle that clients should never see user credentials
- Enables phishing: malicious clients can capture passwords
- Prevents MFA enforcement (password alone does not support step-up auth)
- The auth code flow with PKCE is universally usable, even for first-party apps

**OAuth 2.1 replacement:** Authorization Code flow (with PKCE for public clients),
or Device Authorization Grant (RFC 8628) for input-constrained devices.

**GGID Impact:**

GGID's token endpoint switch at `server.go:325-386` does **not** include a
`grant_type=password` case. The switch falls through to the default:

```go
// server.go line 383-385
default:
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
    return
```

**Status:** Fully compliant. ROPC was never implemented.

#### 2.1.3 Summary of Removed Grant Types

| Grant Type | OAuth 2.0 | OAuth 2.1 | GGID Status |
|---|---|---|---|
| Implicit (`response_type=token`) | Supported | Removed | Not implemented (code only) |
| ROPC (`grant_type=password`) | Supported | Removed | Not implemented |
| Implicit (`response_type=id_token`) | Supported (OIDC) | Removed | Not implemented |

### 2.2 PKCE Mandatory for All Clients

#### 2.2.1 The Requirement

PKCE (Proof Key for Code Exchange, RFC 7636) was originally designed for mobile
and SPA applications that could not store a client secret securely. OAuth 2.1
makes PKCE **mandatory for all clients**, including confidential clients that
already authenticate with a client secret.

The flow works as follows:

```
1. Client generates code_verifier (random 43-128 char string)
2. Client derives code_challenge = BASE64URL(SHA256(code_verifier))
3. Client sends code_challenge + code_challenge_method=S256 in authorize request
4. Server stores code_challenge with the authorization code
5. At token exchange, client sends code_verifier
6. Server verifies: BASE64URL(SHA256(code_verifier)) == stored code_challenge
```

This prevents authorization code interception attacks: even if an attacker
intercepts the authorization code (e.g., via a malicious redirect), they cannot
exchange it without the code_verifier.

#### 2.2.2 code_challenge_method Restrictions

OAuth 2.1 requires `code_challenge_method` to be **S256** only. The `plain`
method is deprecated because it provides no cryptographic binding -- the
challenge equals the verifier, so intercepting the challenge is as good as
intercepting the verifier.

#### 2.2.3 GGID Impact

**PKCE infrastructure exists** but is **not enforced by default**:

- The config field `RequirePKCE` exists at `conf/conf.go:27`:
  ```go
  RequirePKCE    bool `yaml:"require_pkce"` // force PKCE for all authorization_code flows
  ```

- The `Default()` function at `conf/conf.go:41-54` does **NOT** set `RequirePKCE = true`.
  It defaults to `false`.

- The authorize endpoint at `server.go:182-190` only enforces PKCE when
  `cfg.RequirePKCE` is true:
  ```go
  if cfg.RequirePKCE && codeChallenge == "" {
      // reject: "PKCE is required"
  }
  ```

- The PKCE validation logic in `domain/models.go:100-118` correctly implements
  both S256 and plain verification:
  ```go
  func (c *AuthorizationCode) ValidatePKCE(verifier string) bool {
      if c.CodeChallenge == "" {
          return true // PKCE not required for this code
      }
      // ... S256 and plain verification
  }
  ```

- The token exchange at `oauth_service.go:248-251` validates PKCE:
  ```go
  if !code.ValidatePKCE(req.CodeVerifier) {
      return nil, errors.InvalidArgument("PKCE verification failed")
  }
  ```

**Gap:** PKCE is implemented but opt-in, not mandatory. OAuth 2.1 requires it
to be on by default.

### 2.3 Exact Redirect URI Matching

#### 2.3.1 The Requirement

OAuth 2.1 mandates that redirect URIs be compared using **exact string matching**.
No normalization, no wildcard matching, no path prefix matching. The registered
redirect URI and the requested redirect URI must be byte-for-byte identical.

Additionally:
- The redirect URI scheme **must** be `https` (except `http://localhost` for development)
- No dynamic redirect URI registration for public clients
- Loopback redirect URIs (`http://127.0.0.1:*`) are permitted with port flexibility

#### 2.3.2 GGID Impact

GGID's `ValidateRedirectURI` at `domain/models.go:59-67` uses exact string
comparison:

```go
func (c *OAuthClient) ValidateRedirectURI(uri string) bool {
    for _, r := range c.RedirectURIs {
        if r == uri {
            return true
        }
    }
    return false
}
```

This is **already OAuth 2.1 compliant** for the comparison logic.

**However, there are gaps:**

1. **No https scheme enforcement**: The code does not verify that the redirect
   URI uses `https://` (except for localhost). An `http://` redirect URI for
   a production domain would be accepted.

2. **No loopback port flexibility**: OAuth 2.1 allows `http://127.0.0.1:{port}`
   with any port if the registered URI uses `http://localhost` or
   `http://127.0.0.1`. GGID does exact match only.

3. **Dynamic registration accepts any URI**: The `DynamicClientRegister` method
   at `oauth_service.go:820-822` only checks that `RedirectURIs` is non-empty.
   It does not validate the scheme or hostname.

### 2.4 Refresh Token Rotation

#### 2.4.1 The Requirement

OAuth 2.1 strongly recommends **refresh token rotation**:

1. Each time a refresh token is used, the server issues a **new** refresh token
2. The old refresh token is **immediately invalidated**
3. If a previously-used (rotated) refresh token is presented again, the server
   detects **token reuse** and revokes the entire **token family** (all tokens
   derived from the same original authorization)

This mechanism detects token theft: if an attacker steals a refresh token and
uses it, the legitimate client's next refresh attempt will use a now-invalid
token, triggering family-wide revocation.

For public clients, the draft further recommends that refresh tokens be
**sender-constrained** using DPoP or mTLS to cryptographically bind them
to the client.

#### 2.4.2 GGID Impact

GGID's `RefreshToken` method at `oauth_service.go:582-626`:

```go
func (s *OAuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenResponse, error) {
    // ... client authentication ...

    // Parse the refresh token (simple format: "userID.randompart")
    parts := strings.SplitN(req.RefreshToken, ".", 2)
    userIDStr := parts[0]
    userID, err := uuid.Parse(userIDStr)

    // Issue new access token
    accessToken, expiresIn, err := s.issueAccessToken(userID, req.TenantID, client.ClientID)

    return &TokenResponse{
        AccessToken: accessToken,
        // NO new refresh token is issued!
        // NO rotation occurs
        // NO reuse detection exists
    }, nil
}
```

**Critical gaps:**

1. **No rotation**: The same refresh token can be used indefinitely. The response
   (`TokenResponse` at line 206-214) has a `RefreshToken` field but it is never
   populated by the `RefreshToken` method.

2. **No reuse detection**: There is no tracking of which refresh tokens have been
   used. A stolen token can be used forever.

3. **No token family**: There is no concept of a token family (all tokens derived
   from the same authorization). Family-wide revocation is impossible.

4. **Insecure token format**: The refresh token is `userID.randompart`, not a
   signed JWT or encrypted token. The `userID` is exposed in plaintext. An
   attacker who guesses the format can forge tokens.

5. **No sender-constraining**: Refresh tokens are bearer tokens with no DPoP
   or mTLS binding.

### 2.5 Browser Security Parameters

#### 2.5.1 State Parameter (REQUIRED)

OAuth 2.1 makes the `state` parameter **mandatory** in all authorization
requests. The `state` parameter serves as a CSRF token:

1. Client generates a random `state` value before redirecting to the authorize endpoint
2. Server echoes `state` back in the redirect
3. Client verifies the returned `state` matches the one it sent

Without state validation, an attacker can inject their own authorization code
into the victim's session (login CSRF attack).

#### 2.5.2 Nonce Parameter (OIDC)

For OpenID Connect flows, the `nonce` parameter is **mandatory** in the
authorization request. The nonce must be included in the resulting ID Token,
and the client must verify it matches.

This prevents token injection and replay attacks in OIDC.

#### 2.5.3 GGID Impact

**State parameter:**

- The authorize endpoint at `server.go:166` accepts `state` from query params
- It is passed through to `CreateAuthorizationCode` and echoed in the redirect
  at `server.go:280-282`
- But there is **no server-side validation** that `state` is present and non-empty
- The client is responsible for validating `state`, but GGID does not enforce
  its presence

```go
// server.go line 166 - state is read but never validated
state := r.URL.Query().Get("state")
```

**Nonce parameter:**

- The authorize endpoint at `server.go:168` accepts `nonce` from query params
- It is stored in the `AuthorizationCode` at `oauth_service.go:184`
- It is included in the ID Token at `oauth_service.go:371`
- But there is **no enforcement** that nonce is present for OIDC flows
- Client-side verification is the client's responsibility (correct per spec),
  but server should require nonce when `scope=openid` is requested

### 2.6 Token Endpoint Authentication

#### 2.6.1 The Requirement

OAuth 2.1 requires:

- **Confidential clients** MUST authenticate at the token endpoint using one of:
  - `client_secret_basic` (HTTP Basic Auth)
  - `client_secret_post` (form-encoded client_secret)
  - `client_secret_jwt` (JWT assertion with shared secret)
  - `private_key_jwt` (JWT assertion with private key)
  - `tls_client_auth` (mTLS)

- **Public clients** MUST use PKCE (no client_secret, no client_id-only auth)

- `client_id` alone is **not sufficient** authentication for confidential clients

#### 2.6.2 GGID Impact

GGID authenticates confidential clients by verifying the client secret at
`oauth_service.go:224-230`:

```go
if client.IsConfidential() {
    ok, _ := crypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
    if !ok {
        return nil, errors.Unauthenticated("invalid client credentials")
    }
}
```

**Gaps:**

1. **Only `client_secret_post` is effectively supported**: The secret is read
   from form data (`client_secret` form field). There is no HTTP Basic Auth
   extraction. The discovery document at line 299 advertises
   `client_secret_basic`, but the implementation does not parse Basic Auth
   headers.

2. **No JWT-based auth**: `client_secret_jwt` and `private_key_jwt` are not
   supported.

3. **No mTLS**: `tls_client_auth` is not supported.

4. **Public clients bypass all authentication**: For public clients, the
   `client_id` alone identifies the client at the token endpoint. OAuth 2.1
   requires PKCE for public clients, which is not enforced.

---

## 3. Retained Features

OAuth 2.1 retains the following features from OAuth 2.0 (with modifications):

### 3.1 Authorization Code Grant (with Mandatory PKCE)

The authorization code flow remains the primary grant type. The only change
is that PKCE is now mandatory.

**GGID Status:** Implemented at `oauth_service.go:152-193` (code creation) and
`oauth_service.go:217-278` (token exchange). PKCE is supported but not mandatory.

### 3.2 Client Credentials Grant

Used for machine-to-machine authentication. The client authenticates with its
secret and receives an access token without user involvement.

**GGID Status:** Implemented at `oauth_service.go:639-671`. Properly verifies
client secret and grant type support.

### 3.3 Refresh Token Grant

Allows clients to obtain new access tokens using a refresh token.

**GGID Status:** Implemented at `oauth_service.go:582-626`. Missing rotation
and reuse detection (see Section 2.4).

### 3.4 Device Authorization Grant (RFC 8628)

Used for devices with limited input capabilities (smart TVs, IoT devices).
The device displays a code, the user authorizes on a separate device.

**GGID Status:** Implemented at `oauth_service.go:989-1078`. Includes device
code creation, polling, approval flow, and proper RFC 8628 error codes.

### 3.5 Token Exchange (RFC 8693)

Allows exchanging one token for another with reduced scope or different
audience (delegation scenario).

**GGID Status:** Implemented at `oauth_service.go:912-941`. Currently a
skeleton -- issues an opaque token instead of a properly signed JWT with
audience constraints.

### 3.6 JWT Bearer Assertion Grant (RFC 7523)

Allows using a JWT assertion from a trusted third party to obtain an access
token.

**GGID Status:** Implemented at `oauth_service.go:1229-1304`. Parses the
assertion but does not verify the signature against the issuer's JWKS (uses
`ParseUnverified`).

### 3.7 PKCE (RFC 7636)

Proof Key for Code Exchange. Now mandatory.

**GGID Status:** Implemented at `domain/models.go:100-118`. Not enforced by
default.

### 3.8 DPoP (Demonstrating Proof-of-Possession)

A mechanism for sender-constraining access tokens using a client-generated
key pair. The client signs each HTTP request with its private key, proving
possession of the key.

**GGID Status:** **Not implemented.** This is a draft specification that
OAuth 2.1 recommends but does not mandate.

### 3.9 mTLS (Mutual TLS) Token Binding

Client certificates are used to authenticate the client at the token endpoint
and to bind tokens to the client's certificate.

**GGID Status:** **Not implemented.** Would require infrastructure changes
(TLS client certificate verification at the load balancer or reverse proxy).

### 3.10 Summary of Retained Features

| Feature | RFC | OAuth 2.1 Status | GGID Implementation |
|---|---|---|---|
| Authorization Code + PKCE | 6749 + 7636 | Mandatory | Implemented (PKCE optional) |
| Client Credentials | 6749 | Retained | Implemented |
| Refresh Token | 6749 | Retained (+ rotation rec.) | Implemented (no rotation) |
| Device Authorization | 8628 | Retained | Implemented |
| Token Exchange | 8693 | Retained | Skeleton |
| JWT Bearer | 7523 | Retained | Implemented (no sig verify) |
| PKCE | 7636 | Mandatory | Implemented (not enforced) |
| DPoP | draft-ietf-oauth-dpop | Recommended | Not implemented |
| mTLS | 8705 | Recommended | Not implemented |

---

## 4. Security Improvements Made Mandatory

### 4.1 PKCE for All Clients

**Requirement:** All clients using the authorization code flow MUST include
PKCE parameters (`code_challenge` and `code_challenge_method=S256`) in the
authorization request, and MUST provide `code_verifier` at the token endpoint.

**Scope:** This applies to both confidential and public clients. Even if a
client authenticates with a client secret, PKCE is still required. This is
defense-in-depth: the code secret protects against one class of attacks, and
PKCE protects against another (code interception).

**code_challenge_method:** Only `S256` is permitted. The `plain` method is
deprecated and should not be accepted by new implementations.

**GGID Gap:** PKCE is implemented but opt-in (`RequirePKCE` defaults to `false`).
The `plain` method is still accepted.

### 4.2 State Parameter Mandatory

**Requirement:** The `state` parameter MUST be included in every authorization
request. The authorization server MUST echo it back in the redirect response.

The `state` value should be:
- Cryptographically random (at least 128 bits of entropy)
- Bound to the user's session or the specific authorization request
- Validated by the client on callback (exact match)

**GGID Gap:** GGID accepts `state` and echoes it back, but does not enforce
its presence. An authorization request without `state` is accepted.

### 4.3 Strict Redirect URI Handling

**Requirement:** Redirect URIs must match exactly using simple string comparison.

Additional rules:
- Registered redirect URIs MUST use `https` scheme (except `http://localhost`)
- Loopback redirects (`http://127.0.0.1` or `http://[::1]`) are allowed for
  native apps with port flexibility
- No wildcard matching
- No path normalization (trailing slashes matter)
- Dynamic registration for public clients should not be allowed without
  verification

**GGID Status:** Exact string matching is implemented (compliant). Missing:
https scheme enforcement, loopback port flexibility, dynamic registration
URI validation.

### 4.4 Refresh Token Security

**Requirement:** Refresh tokens should be protected through:

1. **Rotation**: Issue a new refresh token on each use, invalidate the old one
2. **Reuse detection**: If a rotated token is reused, revoke the entire family
3. **Sender-constraining**: Bind refresh tokens to the client via DPoP or mTLS
   (strongly recommended for public clients)
4. **Lifetime limits**: Refresh tokens should have a maximum lifetime
5. **Scope narrowing**: Refreshed tokens should not grant more scopes than
   the original authorization

**GGID Gap:** None of these mechanisms are implemented. Refresh tokens are
infinite-lifetime bearer tokens with no rotation or reuse detection.

### 4.5 Consent

**Requirement:** OAuth 2.1 recommends explicit consent for each requested scope.

For first-party clients (owned by the same organization as the authorization
server), implicit consent may be acceptable. For third-party clients, the user
must explicitly approve each requested scope.

The `prompt` parameter (from OIDC) provides additional control:
- `prompt=none`: No UI; return error if consent/login is needed
- `prompt=consent`: Force consent screen
- `prompt=login`: Force re-authentication

**GGID Status:** Partial consent implementation exists at `server.go:233-254`.
Basic scopes (openid, profile, email, offline_access) are auto-approved. Extended
scopes require `consent=true` parameter. This is a reasonable first-party default
but does not properly implement per-scope consent or the `prompt` parameter.

### 4.6 Summary of Mandatory Security Improvements

| Security Feature | Requirement Level | GGID Status | Gap Severity |
|---|---|---|---|
| PKCE for all clients | MUST | Implemented, opt-in | High |
| code_challenge_method=S256 only | MUST | Accepts plain | Medium |
| State parameter mandatory | MUST | Accepted but not enforced | High |
| Exact redirect URI match | MUST | Implemented | None |
| HTTPS redirect URI scheme | MUST | Not enforced | Medium |
| Refresh token rotation | SHOULD | Not implemented | High |
| Refresh token reuse detection | SHOULD | Not implemented | High |
| Sender-constrained tokens | RECOMMENDED | Not implemented | Low |
| Explicit consent | RECOMMENDED | Partial | Medium |
| Confidential client auth | MUST | Partial (POST only) | Medium |

---

## 5. GGID Current OAuth Implementation Audit

### 5.1 Architecture Overview

GGID's OAuth service is implemented as a standalone microservice at
`services/oauth/`:

```
services/oauth/
├── cmd/
│   └── main.go              # Entry point
├── internal/
│   ├── conf/
│   │   └── conf.go          # Configuration (lines 1-54)
│   ├── domain/
│   │   └── models.go        # Domain types (lines 1-190)
│   ├── repository/
│   │   └── *.go             # PostgreSQL repositories
│   ├── service/
│   │   └── oauth_service.go # Business logic (lines 1-1304)
│   └── server/
│       └── server.go        # HTTP handlers (lines 1-1065)
└── go.mod
```

### 5.2 Supported Grant Types (Audit)

Examining `server.go:325-386` (token endpoint switch):

```go
switch grantType {
case "authorization_code":           // Implemented ✓
case "refresh_token":                // Implemented ✓
case "client_credentials":           // Implemented ✓
case "urn:ietf:params:oauth:grant-type:device_code":  // RFC 8628 ✓
case "urn:ietf:params:oauth:grant-type:jwt-bearer":   // RFC 7523 ✓
default:                             // unsupported_grant_type
}
```

**Grant types NOT in the switch (correctly absent):**
- `password` (ROPC) -- not implemented, returns `unsupported_grant_type`
- `implicit` -- not applicable (handled at authorize endpoint, rejected)

**Token Exchange (RFC 8693):**
The `ExchangeToken` method exists at `oauth_service.go:912-941` but is **not
wired** into the token endpoint switch. There is no
`urn:ietf:params:oauth:grant-type:token-exchange` case.

### 5.3 PKCE Support (Audit)

| Aspect | Location | Status |
|---|---|---|
| code_challenge accepted | `server.go:169` | Yes |
| code_challenge_method accepted | `server.go:170` | Yes |
| code_challenge stored | `oauth_service.go:182-183` | Yes |
| code_verifier validated | `oauth_service.go:248-251` | Yes |
| PKCE validation logic | `domain/models.go:100-118` | S256 + plain |
| RequirePKCE config | `conf/conf.go:27` | Defaults to false |
| Enforce S256 only | Not implemented | Accepts plain |

### 5.4 Redirect URI Validation (Audit)

| Aspect | Location | Status |
|---|---|---|
| Exact string match | `domain/models.go:59-67` | Yes (compliant) |
| https scheme check | Not implemented | Gap |
| Loopback port flexibility | Not implemented | Gap |
| Wildcard rejection | Implicit (exact match) | Compliant |
| Dynamic registration validation | `oauth_service.go:820` | Non-empty only |

### 5.5 Refresh Token Handling (Audit)

| Aspect | Location | Status |
|---|---|---|
| Token format | `oauth_service.go:605-613` | `userID.randompart` (insecure) |
| Rotation | Not implemented | Gap |
| Reuse detection | Not implemented | Gap |
| Token family tracking | Not implemented | Gap |
| Lifetime limit | Not implemented | Infinite |
| Sender-constraining | Not implemented | Bearer only |
| Scope narrowing | `oauth_service.go:624` | Client-requested scopes |

### 5.6 State and Nonce Enforcement (Audit)

| Aspect | Location | Status |
|---|---|---|
| State accepted | `server.go:166` | Yes |
| State echoed in redirect | `server.go:280-282` | Yes |
| State presence enforced | Not implemented | Gap |
| Nonce accepted | `server.go:168` | Yes |
| Nonce stored in code | `oauth_service.go:184` | Yes |
| Nonce in ID Token | `oauth_service.go:371` | Yes |
| Nonce presence enforced for OIDC | Not implemented | Gap |

### 5.7 Token Formats (Audit)

| Token Type | Format | Signing | Expiry | Location |
|---|---|---|---|---|
| Access Token | JWT | RS256 | 15 min | `oauth_service.go:325-358` |
| ID Token | JWT | RS256 | 1 hour | `oauth_service.go:361-384` |
| Refresh Token | Plaintext | None | Infinite | `oauth_service.go:605-613` |
| Device Token | JWT | RS256 | 1 hour | `oauth_service.go:1107-1130` |
| SAML Token | JWT | RS256 | 15 min | `oauth_service.go:523-527` |

### 5.8 Client Authentication Methods (Audit)

| Method | Supported | Location |
|---|---|---|
| `client_secret_basic` | Advertised only | `oauth_service.go:299` (not parsed) |
| `client_secret_post` | Yes | `server.go:315` (form field) |
| `client_secret_jwt` | No | -- |
| `private_key_jwt` | No | -- |
| `tls_client_auth` | No | -- |
| `none` (public client) | Yes | Advertised at line 299 |

### 5.9 Discovery Document Audit

The OIDC discovery document at `oauth_service.go:283-302`:

```go
ResponseTypesSupported:            []string{"code", "token", "id_token"},
GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials"},
TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
CodeChallengeMethodsSupported:     []string{"S256", "plain"},
```

**Issues:**

1. `ResponseTypesSupported` includes `"token"` and `"id_token"` -- these are
   not actually supported (implicit flow is rejected). Should be `["code"]` only.

2. `GrantTypesSupported` does not include `"urn:ietf:params:oauth:grant-type:device_code"`
   or `"urn:ietf:params:oauth:grant-type:jwt-bearer"` despite being implemented.

3. `CodeChallengeMethodsSupported` includes `"plain"` -- should be `["S256"]` for
   OAuth 2.1 compliance.

4. `TokenEndpointAuthMethodsSupported` includes `"client_secret_basic"` but the
   implementation does not parse Basic Auth headers.

### 5.10 Comprehensive Audit Table

| # | Feature | OAuth 2.1 Requirement | GGID Current State | Gap? | Priority |
|---|---|---|---|---|---|
| 1 | Implicit grant | Removed | Not implemented | No | -- |
| 2 | ROPC grant | Removed | Not implemented | No | -- |
| 3 | PKCE mandatory | MUST | Opt-in (RequirePKCE=false) | Yes | P0 |
| 4 | PKCE method S256 only | MUST | Accepts S256 + plain | Yes | P1 |
| 5 | Exact redirect URI match | MUST | Exact string match | No | -- |
| 6 | HTTPS redirect URI scheme | MUST | Not enforced | Yes | P1 |
| 7 | Loopback port flexibility | SHOULD | Not implemented | Yes | P2 |
| 8 | Refresh token rotation | SHOULD | Not implemented | Yes | P0 |
| 9 | Refresh token reuse detection | SHOULD | Not implemented | Yes | P0 |
| 10 | Sender-constrained tokens | RECOMMENDED | Not implemented | Yes | P2 |
| 11 | State parameter mandatory | MUST | Accepted, not enforced | Yes | P0 |
| 12 | Nonce mandatory for OIDC | MUST | Accepted, not enforced | Yes | P1 |
| 13 | Confidential client auth | MUST | POST only (no Basic) | Yes | P1 |
| 14 | Discovery doc accuracy | MUST | Inaccurate (3 issues) | Yes | P1 |
| 15 | Token exchange (RFC 8693) | Retained | Not wired to endpoint | Yes | P2 |
| 16 | JWT bearer signature verify | MUST | ParseUnverified | Yes | P1 |
| 17 | Refresh token format security | SHOULD | Plaintext userID | Yes | P0 |
| 18 | Refresh token lifetime limit | SHOULD | Infinite | Yes | P1 |
| 19 | Consent for extended scopes | RECOMMENDED | Partial | Yes | P2 |
| 20 | Token revocation persistence | SHOULD | In-memory only | Yes | P1 |
| 21 | Device authorization flow | Retained | Implemented | No | -- |
| 22 | Dynamic client registration | Retained | Implemented | No | -- |
| 23 | Token introspection (RFC 7662) | Retained | Implemented | No | -- |
| 24 | Token revocation (RFC 7009) | Retained | Implemented | No | -- |
| 25 | Back-channel logout | Retained | Implemented | No | -- |

**Summary:** 17 gaps identified. 6 are P0 (critical), 7 are P1 (high), 4 are P2 (medium).

---

## 6. Migration Plan: OAuth 2.0 to 2.1

### 6.1 Phase 1: PKCE Enforcement (P0, 1-2 weeks)

**Goal:** Make PKCE mandatory for all authorization code flows.

**Steps:**

1. **Change default config**: Set `RequirePKCE = true` in `conf.Default()`
   ```go
   // conf/conf.go
   func Default() *Config {
       cfg := &Config{}
       // ...
       cfg.RequirePKCE = true // OAuth 2.1: PKCE is mandatory
       return cfg
   }
   ```

2. **Add PKCE requirement for public clients** even when `RequirePKCE` is false:
   ```go
   // server.go authorize endpoint
   client, err := oauthSvc.GetClient(ctx, clientID)
   if err != nil { /* ... */ }

   if (cfg.RequirePKCE || !client.IsConfidential()) && codeChallenge == "" {
       // reject: PKCE required
   }
   ```

3. **Deprecate `plain` method**: Log a warning when `plain` is used, then
   reject it after a transition period.

4. **Update discovery document**: Change `CodeChallengeMethodsSupported` to
   `["S256"]`.

5. **Add PKCE to SDK**: All SDKs should generate and send PKCE parameters
   by default.

6. **Feature flag for backward compatibility**: Add a `PKCE_GRACE_PERIOD`
   flag that logs warnings instead of rejecting requests missing PKCE,
   for a 30-60 day transition.

**Affected files:**
- `services/oauth/internal/conf/conf.go` (line 27, 41-54)
- `services/oauth/internal/server/server.go` (lines 182-198)
- `services/oauth/internal/service/oauth_service.go` (lines 283-302)
- `sdk/go/ggid/` (OAuth client)
- `sdk/node/ggid/` (OAuth client)
- `sdk/java/ggid/` (OAuth client)

### 6.2 Phase 2: Remove Deprecated Flows (P1, 1 week)

**Goal:** Ensure no deprecated grant types are available and fix discovery doc.

**Steps:**

1. **Verify implicit is fully removed**: Already done (server.go:177-179).

2. **Verify ROPC is not available**: Already done (not in switch).

3. **Fix discovery document**:
   ```go
   // oauth_service.go GetDiscoveryConfig()
   ResponseTypesSupported: []string{"code"},  // was: ["code", "token", "id_token"]
   GrantTypesSupported: []string{
       "authorization_code",
       "refresh_token",
       "client_credentials",
       "urn:ietf:params:oauth:grant-type:device_code",
       "urn:ietf:params:oauth:grant-type:jwt-bearer",
   },
   CodeChallengeMethodsSupported: []string{"S256"},  // was: ["S256", "plain"]
   ```

4. **Remove `token` and `id_token` from response_types**: These are
   not supported and should not be advertised.

### 6.3 Phase 3: Tighten Security (P0-P1, 2-3 weeks)

**Goal:** Implement mandatory security controls.

#### 6.3.1 State Parameter Enforcement

```go
// server.go authorize endpoint
if state == "" {
    writeJSON(w, http.StatusBadRequest, map[string]string{
        "error":             "invalid_request",
        "error_description": "state parameter is required for CSRF protection",
    })
    return
}
```

#### 6.3.2 Nonce Enforcement for OIDC

```go
// server.go authorize endpoint
if contains(scopes, "openid") && nonce == "" {
    writeJSON(w, http.StatusBadRequest, map[string]string{
        "error":             "invalid_request",
        "error_description": "nonce parameter is required for OIDC flows",
    })
    return
}
```

#### 6.3.3 HTTPS Redirect URI Enforcement

```go
// domain/models.go ValidateRedirectURI
func (c *OAuthClient) ValidateRedirectURI(uri string) bool {
    parsed, err := url.Parse(uri)
    if err != nil {
        return false
    }

    // Require HTTPS except for localhost
    if parsed.Scheme != "https" {
        if parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" {
            return false
        }
    }

    for _, r := range c.RedirectURIs {
        if r == uri {
            return true
        }
    }
    return false
}
```

#### 6.3.4 Refresh Token Rotation + Reuse Detection

This is the most complex change. See Section 10.2 for implementation code.

**Requirements:**
- Generate a signed refresh token (JWT or encrypted token)
- Store refresh token metadata in the database (token family ID, used flag)
- On refresh: issue new access + refresh token, mark old token as used
- On reuse: detect previously-used token, revoke entire family
- Add refresh token lifetime (e.g., 30 days)

**Database changes:**
```sql
CREATE TABLE oauth_refresh_tokens (
    id            UUID PRIMARY KEY,
    tenant_id     UUID NOT NULL,
    token_hash    TEXT NOT NULL UNIQUE,
    family_id     UUID NOT NULL,
    client_id     UUID NOT NULL,
    user_id       UUID NOT NULL,
    scope         TEXT[],
    used          BOOLEAN DEFAULT FALSE,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    revoked_at    TIMESTAMPTZ
);

CREATE INDEX idx_refresh_tokens_family ON oauth_refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_hash ON oauth_refresh_tokens(token_hash);
```

### 6.4 Phase 4: Advanced Features (P2, 4-6 weeks)

#### 6.4.1 DPoP Support

**Overview:** DPoP (Demonstrating Proof-of-Possession) allows clients to
cryptographically bind tokens to their public key. Each HTTP request includes
a DPoP header with a JWT signed by the client's private key.

**Implementation plan:**
1. Accept `DPoP` header at authorize and token endpoints
2. Validate DPoP JWT structure (typ, alg, jti, htm, htu, iat)
3. Store the client's public key (thumbprint) with the token
4. Validate DPoP proof on resource access
5. Implement DPoP-bound refresh tokens

**Affected files:**
- New: `services/oauth/internal/service/dpop.go`
- Modified: `services/oauth/internal/server/server.go` (header parsing)
- Modified: `services/oauth/internal/service/oauth_service.go` (token binding)

#### 6.4.2 mTLS-Bound Tokens

**Overview:** Use TLS client certificates to authenticate the client and
bind tokens to the certificate.

**Implementation plan:**
1. Configure TLS with client certificate verification at the server level
2. Extract client certificate from TLS connection
3. Bind token to certificate thumbprint (x5t#S256)
4. Validate certificate on token use

**Infrastructure dependency:** Requires reverse proxy or load balancer
configuration to pass client certificates to the application.

#### 6.4.3 Enhanced Consent Management

**Implementation plan:**
1. Store consent records per user-client-scope combination
2. Implement `prompt` parameter (none, consent, login)
3. Add consent revocation endpoint
4. Per-scope consent UI in the admin console

**Database changes:**
```sql
CREATE TABLE oauth_consents (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    client_id   UUID NOT NULL,
    scope       TEXT[],
    granted_at  TIMESTAMPTZ DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ,
    UNIQUE(tenant_id, user_id, client_id)
);
```

### 6.5 Migration Phase Summary

| Phase | Duration | Priority | Description |
|---|---|---|---|
| Phase 1 | 1-2 weeks | P0 | PKCE enforcement, SDK updates |
| Phase 2 | 1 week | P1 | Remove deprecated flows, fix discovery doc |
| Phase 3 | 2-3 weeks | P0-P1 | State/nonce enforcement, HTTPS redirect, refresh token rotation |
| Phase 4 | 4-6 weeks | P2 | DPoP, mTLS, enhanced consent |
| **Total** | **8-12 weeks** | | Full OAuth 2.1 compliance |

---

## 7. Impact on GGID SDKs

### 7.1 Go SDK (`sdk/go/ggid/`)

**Changes needed:**

1. **PKCE generation**: Add `GeneratePKCE()` function that creates a code_verifier
   and derives the code_challenge using S256.

2. **Authorization URL builder**: Include `code_challenge` and `code_challenge_method`
   in the authorize URL by default.

3. **Token exchange**: Include `code_verifier` in the token request.

4. **State generation**: Add `GenerateState()` for random state values.

5. **Token validation**: Validate `state` on callback.

```go
// Example PKCE helper for Go SDK
package ggid

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
)

func GeneratePKCE() (verifier, challenge string, err error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", "", err
    }
    verifier = base64.RawURLEncoding.EncodeToString(b)
    h := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(h[:])
    return verifier, challenge, nil
}
```

### 7.2 Node SDK (`sdk/node/ggid/`)

**Changes needed:**

1. **PKCE generation**: Use `crypto.randomBytes` + SHA-256.
2. **State generation**: Use `crypto.randomBytes`.
3. **Authorization URL**: Include PKCE + state parameters.
4. **Callback handler**: Validate state, exchange code with verifier.

```typescript
import crypto from 'crypto';

export function generatePKCE(): { verifier: string; challenge: string } {
    const verifier = crypto.randomBytes(32).toString('base64url');
    const challenge = crypto.createHash('sha256').update(verifier).digest('base64url');
    return { verifier, challenge };
}

export function generateState(): string {
    return crypto.randomBytes(16).toString('hex');
}
```

### 7.3 Java SDK (`sdk/java/ggid/`)

**Changes needed:**

1. **PKCE generation**: Use `SecureRandom` + `MessageDigest`.
2. **State generation**: Same.
3. **Authorization URL builder**: Include PKCE + state.
4. **Callback handler**: Validate state, exchange code.

```java
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.util.Base64;

public class PKCEUtil {
    private static final SecureRandom random = new SecureRandom();

    public static String generateVerifier() {
        byte[] bytes = new byte[32];
        random.nextBytes(bytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }

    public static String generateChallenge(String verifier) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(verifier.getBytes(java.nio.charset.StandardCharsets.UTF_8));
            return Base64.getUrlEncoder().withoutPadding().encodeToString(hash);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException(e);
        }
    }
}
```

### 7.4 SDK Migration Timeline

| SDK | PKCE Support | State Support | Estimated Effort |
|---|---|---|---|
| Go | Add GeneratePKCE | Add GenerateState | 1 day |
| Node | Add generatePKCE | Add generateState | 1 day |
| Java | Add PKCEUtil | Add StateUtil | 1 day |

All SDKs should ship PKCE enabled by default. Existing applications using the
SDKs will automatically become OAuth 2.1 compliant once the server enforces PKCE.

---

## 8. Comparison with Other Providers

### 8.1 Auth0 (Okta)

| Feature | Auth0 Status |
|---|---|
| PKCE mandatory | Yes (enforced for all new clients since 2023) |
| Implicit grant | Disabled for new tenants |
| ROPC | Disabled for new tenants |
| Refresh token rotation | Supported (configurable per client) |
| DPoP | Not supported |
| mTLS | Supported (enterprise tier) |
| State enforcement | Recommended but not mandatory |
| Discovery document | Accurate |

**Assessment:** Auth0 is closest to OAuth 2.1 compliance among commercial providers.

### 8.2 Keycloak

| Feature | Keycloak Status |
|---|---|
| PKCE mandatory | Supported but not enforced by default |
| Implicit grant | Supported (can be disabled per client) |
| ROPC | Supported (deprecated in docs) |
| Refresh token rotation | Supported (configurable) |
| DPoP | Not supported |
| mTLS | Supported (with configuration) |
| State enforcement | Recommended but not mandatory |
| Discovery document | Accurate |

**Assessment:** Keycloak supports all OAuth 2.1 features but requires manual
configuration to enforce them. Not OAuth 2.1-compliant out of the box.

### 8.3 Ory Hydra

| Feature | Hydra Status |
|---|---|
| PKCE mandatory | Yes (enforced for public clients) |
| Implicit grant | Not supported |
| ROPC | Not supported |
| Refresh token rotation | Not supported (by design) |
| DPoP | Not supported |
| mTLS | Not supported |
| State enforcement | Handled by the login provider |
| Discovery document | Accurate |

**Assessment:** Hydra is the most OAuth 2.1-aligned open-source provider. It
removed implicit and ROPC from the start and enforces PKCE for public clients.

### 8.4 AWS Cognito

| Feature | Cognito Status |
|---|---|
| PKCE mandatory | Supported but not enforced |
| Implicit grant | Supported (still available) |
| ROPC | Supported (still available) |
| Refresh token rotation | Not supported |
| DPoP | Not supported |
| mTLS | Supported (with custom domains) |
| State enforcement | Client responsibility |
| Discovery document | Partial |

**Assessment:** Cognito is the least OAuth 2.1-aligned. It still supports all
deprecated flows by default. Migration would be difficult for AWS-locked customers.

### 8.5 GGID (Current)

| Feature | GGID Status |
|---|---|
| PKCE mandatory | Supported but opt-in (defaults off) |
| Implicit grant | Not implemented (compliant) |
| ROPC | Not implemented (compliant) |
| Refresh token rotation | Not implemented |
| DPoP | Not implemented |
| mTLS | Not implemented |
| State enforcement | Not enforced |
| Discovery document | Inaccurate (3 issues) |

**Assessment:** GGID is in a **good starting position** -- it never implemented
the deprecated flows. The main gaps are PKCE enforcement, refresh token
security, and state/nonce enforcement. With the Phase 1-3 migration, GGID
can surpass Keycloak and Cognito in OAuth 2.1 compliance.

### 8.6 Comparison Matrix

| Feature | Auth0 | Keycloak | Hydra | Cognito | GGID (target) |
|---|---|---|---|---|---|
| No Implicit | Yes | Configurable | Yes | No | Yes |
| No ROPC | Yes | Configurable | Yes | No | Yes |
| PKCE Mandatory | Yes | No | Yes (public) | No | Yes (target) |
| Refresh Rotation | Yes | Yes | No | No | Yes (target) |
| DPoP | No | No | No | No | Future |
| mTLS | Enterprise | Configurable | No | Yes (custom) | Future |
| Discovery Accurate | Yes | Yes | Yes | Partial | Yes (target) |

---

## 9. Timeline and Strategy

### 9.1 When to Start

**Recommendation: Start immediately.**

OAuth 2.1 is close to finalization (draft-15 is typically near last call).
RFC 9700 (OAuth 2.0 Security Best Current Practice) is already a published RFC
and contains most of the same requirements. Security auditors are already
referencing these requirements.

Delaying migration increases:
- Security risk (missing PKCE, no refresh token rotation)
- Technical debt (more clients to migrate later)
- Audit findings against RFC 9700

### 9.2 Backward Compatibility Strategy

**Approach: Feature flags with warning logs.**

1. **PKCE grace period (60 days):**
   - `OAUTH_PKCE_MODE=warn` -- log warning when PKCE is missing, but allow
   - `OAUTH_PKCE_MODE=enforce` -- reject requests without PKCE
   - Default: `warn` for 60 days, then `enforce`

2. **State parameter grace period (30 days):**
   - Same warn/enforce pattern

3. **Refresh token rotation:**
   - New tokens use rotation immediately
   - Existing tokens are grandfathered until used
   - Once used, the new rotated token enters the rotation cycle

4. **Discovery document:**
   - Update immediately (no backward compatibility needed)

### 9.3 Communication Plan

**For API consumers:**

1. **Developer blog post**: Announce OAuth 2.1 migration plan, timeline
2. **Changelog entry**: Document breaking changes
3. **Email notification**: Send to all registered client owners
4. **SDK release notes**: Highlight PKCE-by-default changes
5. **Migration guide**: Step-by-step document for updating clients
6. **Deprecation headers**: Add `Deprecation` and `Sunset` HTTP headers to
   responses for deprecated behavior

**Timeline for communication:**

| Timeframe | Action |
|---|---|
| T-90 days | Announce migration plan, publish timeline |
| T-60 days | Release SDK updates with PKCE by default |
| T-30 days | Enable warning mode (log deprecation warnings) |
| T-0 | Enable enforcement mode |
| T+30 days | Remove warning mode, enforcement only |

### 9.4 Feature Flag Configuration

```yaml
# oauth service config
oauth_2_1:
  pkce_mode: warn          # warn | enforce
  state_required: true     # enforce state parameter
  nonce_required: true     # enforce nonce for OIDC
  refresh_rotation: true   # enable refresh token rotation
  refresh_reuse_detection: true
  refresh_max_age: 720h    # 30 days
  https_redirect_only: true
  code_challenge_method: S256  # S256 only (reject plain)
```

### 9.5 Testing Strategy

1. **Unit tests**: PKCE generation/validation, state generation, refresh token
   rotation logic, reuse detection.

2. **Integration tests**: Full authorization code flow with PKCE, refresh token
   rotation, reuse detection triggers family revocation.

3. **Backward compatibility tests**: Verify existing clients (without PKCE)
   receive appropriate warnings during grace period.

4. **Security tests**: Attempt code interception without PKCE (should fail),
   replay rotated refresh token (should trigger family revocation).

---

## 10. Appendix: Code Examples

### 10.1 PKCE Validation (Go)

Current implementation at `domain/models.go:100-118`:

```go
// ValidatePKCE checks the provided verifier against the stored challenge.
func (c *AuthorizationCode) ValidatePKCE(verifier string) bool {
    if c.CodeChallenge == "" {
        return true // PKCE not required for this code
    }
    if verifier == "" {
        return false
    }
    switch c.CodeChallengeMethod {
    case "plain", "":
        return verifier == c.CodeChallenge
    case "S256":
        h := sha256.Sum256([]byte(verifier))
        encoded := base64.RawURLEncoding.EncodeToString(h[:])
        return encoded == c.CodeChallenge
    default:
        return false
    }
}
```

**Recommended OAuth 2.1 version (S256 only, no plain):**

```go
// ValidatePKCE checks the provided verifier against the stored challenge.
// OAuth 2.1: only S256 is supported. The plain method is deprecated.
func (c *AuthorizationCode) ValidatePKCE(verifier string) bool {
    if c.CodeChallenge == "" {
        // OAuth 2.1: PKCE is mandatory, so this should not happen.
        // Return false to reject codes without PKCE.
        return false
    }
    if verifier == "" {
        return false
    }
    // OAuth 2.1: only S256 is supported.
    if c.CodeChallengeMethod != "S256" {
        return false
    }
    h := sha256.Sum256([]byte(verifier))
    encoded := base64.RawURLEncoding.EncodeToString(h[:])
    // Use constant-time comparison to prevent timing attacks.
    return subtle.ConstantTimeCompare([]byte(encoded), []byte(c.CodeChallenge)) == 1
}
```

**Key changes:**
- Rejects codes without PKCE (`CodeChallenge == ""` returns `false`)
- Only accepts S256 method (no `plain`)
- Uses `subtle.ConstantTimeCompare` to prevent timing attacks

### 10.2 Refresh Token Rotation with Reuse Detection (Go)

```go
// RefreshTokenFamily represents a chain of refresh tokens derived from
// the same authorization. All tokens in a family share the same family_id.
type RefreshTokenFamily struct {
    FamilyID  uuid.UUID
    ClientID  uuid.UUID
    UserID    uuid.UUID
    Tokens    []RefreshTokenRecord
    Revoked   bool
}

// RefreshTokenRecord represents a single refresh token in a family.
type RefreshTokenRecord struct {
    ID        uuid.UUID
    FamilyID  uuid.UUID
    TokenHash string    // SHA-256 hash of the token
    Used      bool      // set to true after rotation
    ExpiresAt time.Time
    CreatedAt time.Time
    RevokedAt *time.Time
}

// RefreshToken issues new tokens using a refresh token.
// Implements OAuth 2.1 refresh token rotation with reuse detection.
func (s *OAuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenResponse, error) {
    // 1. Authenticate the client.
    client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
    if err != nil {
        return nil, errors.Unauthenticated("client authentication failed")
    }
    if client.IsConfidential() {
        ok, _ := crypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
        if !ok {
            return nil, errors.Unauthenticated("invalid client credentials")
        }
    }

    // 2. Look up the refresh token by hash.
    tokenHash := hashTokenSHA256(req.RefreshToken)
    record, err := s.refreshRepo.GetByHash(ctx, req.TenantID, tokenHash)
    if err != nil {
        return nil, errors.Unauthenticated("invalid refresh token")
    }

    // 3. Check expiry.
    if time.Now().After(record.ExpiresAt) {
        return nil, errors.InvalidArgument("refresh token has expired")
    }

    // 4. REUSE DETECTION: If this token was already used (rotated),
    //    revoke the ENTIRE token family.
    if record.Used {
        // This is a reuse attempt! The legitimate client already rotated
        // past this token. An attacker (or race condition) is trying to
        // reuse it. Revoke everything.
        _ = s.refreshRepo.RevokeFamily(ctx, req.TenantID, record.FamilyID)
        log.Printf("SECURITY: Refresh token reuse detected, family %s revoked",
            record.FamilyID)
        return nil, errors.InvalidArgument("refresh token reuse detected; token family revoked")
    }

    // 5. Mark the current token as used.
    record.Used = true
    if err := s.refreshRepo.MarkUsed(ctx, record.ID); err != nil {
        return nil, errors.Internal("mark token used", err)
    }

    // 6. Issue new access token.
    accessToken, expiresIn, err := s.issueAccessToken(record.UserID, req.TenantID, client.ClientID)
    if err != nil {
        return nil, err
    }

    // 7. Issue new refresh token (ROTATION).
    newRefreshToken := generateRefreshToken(record.UserID)
    newHash := hashTokenSHA256(newRefreshToken)
    newRecord := &RefreshTokenRecord{
        ID:        uuid.New(),
        FamilyID:  record.FamilyID, // same family
        TokenHash: newHash,
        Used:      false,
        ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
        CreatedAt: time.Now(),
    }
    if err := s.refreshRepo.Create(ctx, newRecord); err != nil {
        return nil, errors.Internal("create refresh token", err)
    }

    // 8. Return new tokens.
    return &TokenResponse{
        AccessToken:  accessToken,
        TokenType:    "Bearer",
        ExpiresIn:    expiresIn,
        RefreshToken: newRefreshToken, // new rotated token
        Scope:        joinScopes(req.Scope),
    }, nil
}

// generateRefreshToken creates a cryptographically random refresh token.
// Format: opaque random bytes encoded as base64url (no embedded user ID).
func generateRefreshToken(userID uuid.UUID) string {
    b := make([]byte, 48)
    _, _ = crand.Read(b)
    return "rt_" + base64.RawURLEncoding.EncodeToString(b)
}
```

### 10.3 HTTPS Redirect URI Enforcement (Go)

```go
// ValidateRedirectURI checks if the given redirect URI is registered and valid.
// OAuth 2.1: redirect URIs must use exact string matching.
// The scheme must be https, except for localhost/loopback addresses.
func (c *OAuthClient) ValidateRedirectURI(uri string) bool {
    // Parse the URI to check the scheme.
    parsed, err := url.Parse(uri)
    if err != nil {
        return false
    }

    // OAuth 2.1: require https scheme except for localhost.
    isLocal := isLocalhost(parsed.Hostname())
    if parsed.Scheme == "http" && !isLocal {
        return false
    }
    if parsed.Scheme != "https" && parsed.Scheme != "http" {
        return false // custom schemes not allowed for web clients
    }

    // OAuth 2.1: exact string matching.
    for _, registered := range c.RedirectURIs {
        if registered == uri {
            return true
        }
        // Allow loopback port flexibility: http://127.0.0.1:* matches any port.
        if isLoopbackRedirect(registered, uri) {
            return true
        }
    }
    return false
}

// isLocalhead returns true for localhost and loopback addresses.
func isLocalhost(hostname string) bool {
    switch hostname {
    case "localhost", "127.0.0.1", "::1":
        return true
    default:
        return false
    }
}

// isLoopbackRedirect checks if two redirect URIs match on loopback with
// different ports (RFC 8252 Section 7.3).
func isLoopbackRedirect(registered, requested string) bool {
    regURL, err1 := url.Parse(registered)
    reqURL, err2 := url.Parse(requested)
    if err1 != nil || err2 != nil {
        return false
    }
    if !isLocalhost(regURL.Hostname()) || !isLocalhost(reqURL.Hostname()) {
        return false
    }
    if regURL.Scheme != reqURL.Scheme {
        return false
    }
    if regURL.Path != reqURL.Path {
        return false
    }
    // Port is allowed to differ for loopback redirects.
    return regURL.Hostname() == reqURL.Hostname() || regURL.Hostname() == "localhost"
}
```

### 10.4 State and Nonce Enforcement (Go)

```go
// In the authorize endpoint handler (server.go):

// OAuth 2.1: state parameter is REQUIRED.
if state == "" {
    writeJSON(w, http.StatusBadRequest, map[string]string{
        "error":             "invalid_request",
        "error_description": "state parameter is required for CSRF protection",
    })
    return
}

// Validate state entropy (minimum 128 bits = 32 hex chars or 22 base64url chars).
if len(state) < 16 {
    writeJSON(w, http.StatusBadRequest, map[string]string{
        "error":             "invalid_request",
        "error_description": "state parameter must be at least 16 characters",
    })
    return
}

// OAuth 2.1 + OIDC: nonce parameter is REQUIRED for OIDC flows.
hasOpenIDScope := false
for _, s := range scopes {
    if s == "openid" {
        hasOpenIDScope = true
        break
    }
}
if hasOpenIDScope && nonce == "" {
    writeJSON(w, http.StatusBadRequest, map[string]string{
        "error":             "invalid_request",
        "error_description": "nonce parameter is required for OpenID Connect flows",
    })
    return
}
```

### 10.5 Discovery Document Fix (Go)

```go
// GetDiscoveryConfig returns the OAuth 2.1-compliant OIDC discovery document.
func (s *OAuthService) GetDiscoveryConfig() *domain.OIDCDiscoveryConfig {
    base := s.issuer
    return &domain.OIDCDiscoveryConfig{
        Issuer:                            s.issuer,
        AuthorizationEndpoint:             base + "/oauth/authorize",
        TokenEndpoint:                     base + "/oauth/token",
        UserInfoEndpoint:                  base + "/oauth/userinfo",
        JwksURI:                           base + "/oauth/jwks",
        RevocationEndpoint:                base + "/oauth/revoke",
        IntrospectionEndpoint:             base + "/oauth/introspect",
        // OAuth 2.1: only "code" is supported (implicit removed).
        ResponseTypesSupported:            []string{"code"},
        // OAuth 2.1: include all implemented grant types.
        GrantTypesSupported: []string{
            "authorization_code",
            "refresh_token",
            "client_credentials",
            "urn:ietf:params:oauth:grant-type:device_code",
            "urn:ietf:params:oauth:grant-type:jwt-bearer",
        },
        SubjectTypesSupported:             []string{"public"},
        IDTokenSigningAlgValues:           []string{"RS256"},
        ScopesSupported:                   []string{"openid", "profile", "email", "offline_access"},
        ClaimsSupported:                   []string{"sub", "email", "name", "picture", "groups", "preferred_username", "updated_at"},
        // OAuth 2.1: only client_secret_post is truly implemented.
        TokenEndpointAuthMethodsSupported: []string{"client_secret_post", "none"},
        // OAuth 2.1: only S256 is supported (plain deprecated).
        CodeChallengeMethodsSupported:     []string{"S256"},
    }
}
```

### 10.6 Client Secret Basic Auth Support (Go)

```go
// extractClientCredentials extracts client_id and client_secret from either
// HTTP Basic Auth header or form body (OAuth 2.1 token endpoint auth).
func extractClientCredentials(r *http.Request) (clientID, clientSecret string) {
    // Try HTTP Basic Auth first (Authorization: Basic base64(id:secret)).
    authHeader := r.Header.Get("Authorization")
    if strings.HasPrefix(authHeader, "Basic ") {
        decoded, err := base64.StdEncoding.DecodeString(authHeader[6:])
        if err == nil {
            parts := strings.SplitN(string(decoded), ":", 2)
            if len(parts) == 2 {
                return parts[0], parts[1]
            }
        }
    }

    // Fall back to form-encoded credentials.
    clientID = r.FormValue("client_id")
    clientSecret = r.FormValue("client_secret")
    return clientID, clientSecret
}
```

---

## References

1. **OAuth 2.1 Draft**: https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/
2. **OAuth 2.0 (RFC 6749)**: https://www.rfc-editor.org/rfc/rfc6749
3. **OAuth 2.0 Security BCP (RFC 9700)**: https://www.rfc-editor.org/rfc/rfc9700
4. **PKCE (RFC 7636)**: https://www.rfc-editor.org/rfc/rfc7636
5. **OAuth 2.1 Overview**: https://oauth.net/2.1/
6. **Device Authorization (RFC 8628)**: https://www.rfc-editor.org/rfc/rfc8628
7. **Token Exchange (RFC 8693)**: https://www.rfc-editor.org/rfc/rfc8693
8. **Token Introspection (RFC 7662)**: https://www.rfc-editor.org/rfc/rfc7662
9. **Token Revocation (RFC 7009)**: https://www.rfc-editor.org/rfc/rfc7009
10. **DPoP**: https://datatracker.ietf.org/doc/draft-ietf-oauth-dpop/
11. **mTLS (RFC 8705)**: https://www.rfc-editor.org/rfc/rfc8705
12. **Native App BCP (RFC 8252)**: https://www.rfc-editor.org/rfc/rfc8252
13. **Dynamic Client Registration (RFC 7591)**: https://www.rfc-editor.org/rfc/rfc7591
14. **JWT Bearer (RFC 7523)**: https://www.rfc-editor.org/rfc/rfc7523
15. **OAuth 2.0 vs 2.1 Migration Guide**: https://aembit.io/blog/oauth-2-1-guide-migration-security/

---

## GGID Source File References

| File | Lines | Description |
|---|---|---|
| `services/oauth/internal/conf/conf.go` | 1-54 | Configuration (RequirePKCE at line 27) |
| `services/oauth/internal/domain/models.go` | 1-190 | Domain types (PKCE at 100-118, RedirectURI at 59-67) |
| `services/oauth/internal/service/oauth_service.go` | 1-1304 | Business logic (all grant types, token issuance) |
| `services/oauth/internal/server/server.go` | 1-1065 | HTTP handlers (authorize, token, userinfo, etc.) |
| `services/oauth/internal/repository/` | -- | PostgreSQL repositories |

### Key Line References

| Feature | File | Line(s) |
|---|---|---|
| AuthorizeRequest struct | oauth_service.go | 138-150 |
| CreateAuthorizationCode | oauth_service.go | 152-193 |
| ExchangeAuthorizationCode | oauth_service.go | 217-278 |
| PKCE validation | domain/models.go | 100-118 |
| Redirect URI validation | domain/models.go | 59-67 |
| Refresh token grant | oauth_service.go | 582-626 |
| Client credentials grant | oauth_service.go | 639-671 |
| Device authorization | oauth_service.go | 989-1078 |
| Token exchange (RFC 8693) | oauth_service.go | 912-941 |
| JWT bearer (RFC 7523) | oauth_service.go | 1229-1304 |
| Discovery document | oauth_service.go | 283-302 |
| Token revocation | oauth_service.go | 539-556 |
| Authorize endpoint handler | server.go | 157-290 |
| Token endpoint handler | server.go | 293-396 |
| PKCE enforcement check | server.go | 182-198 |
| Consent screen | server.go | 233-254 |
| Dynamic registration | server.go | 577-593 |
| Device auth endpoint | server.go | 833-870 |

---

*End of document.*
