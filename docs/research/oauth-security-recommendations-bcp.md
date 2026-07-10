# OAuth 2.0 Security Best Current Practice (RFC 9700) — GGID Compliance

> **Scope**: This document focuses exclusively on **RFC 9700** security
> recommendations, threat mitigations, and GGID's compliance status.
> OAuth 2.1 migration is covered separately in
> `oauth-2.1-analysis.md` and `oauth-2-1-migration-guide.md`.

---

## 1. Overview

RFC 9700, "Best Current Practice for OAuth 2.0 Security," was published in
2024, superseding RFC 6819. Its status is **Best Current Practice (BCP)** —
not an informational RFC, but the de-facto mandatory security baseline for
every OAuth 2.0 deployment.

**Key shifts from RFC 6819:**

| RFC 6819 (2013) | RFC 9700 (2024) |
|---|---|
| Recommendations and guidelines | Requirements ("MUST", "MUST NOT") |
| PKCE recommended for public clients | PKCE **required** for all clients |
| Implicit and ROPC discouraged | Implicit and ROPC **removed** |
| Redirect URI matching unspecified | Exact-match **mandatory** |

OAuth 2.1 (draft-ietf-oauth-v2-1) incorporates all of RFC 9700 by reference.
Implementing OAuth 2.1 means implementing RFC 9700.

---

## 2. PKCE for All Clients

RFC 9700 §2.1.1: PKCE (RFC 7636) is **REQUIRED** for every authorization code
flow — not just public clients. Confidential clients must also send
`code_challenge` and `code_verifier`. The `code_challenge_method` MUST be
`S256`; `plain` is deprecated and rejected.

**Threat mitigated**: Authorization code interception attack — an attacker
who intercepts the code (via malicious app, log, or redirect) cannot redeem
it without the `code_verifier`.

### GGID Compliance

```go
// domain/models.go — current behavior
func (c *OAuthClient) RequiresPKCE() bool {
    return c.RequirePKCE || c.IsPublic()  // public OR flagged
}

// oauth_service.go — CreateAuthorizationCode
if client.RequiresPKCE() && req.CodeChallenge == "" {
    return "", errors.InvalidArgument("code_challenge is required")
}
```

| Requirement | GGID Status | Compliant? |
|---|---|---|
| PKCE for public clients | Enforced via `RequiresPKCE()` | Yes |
| PKCE for confidential clients | Only if `RequirePKCE` flag set | **No** — not enforced by default |
| Reject `code_challenge_method=plain` | `plain` accepted; discovery advertises `["S256","plain"]` | **No** |
| Default to S256 when omitted | Defaults to S256 if empty | Yes |

**Gap**: Confidential clients can skip PKCE. Discovery advertises `plain`.
Fix: make `RequiresPKCE()` unconditionally `true`, remove `"plain"` from
`CodeChallengeMethodsSupported`, and reject non-S256 challenges.

---

## 3. Redirect URI Exact Match

RFC 9700 §2.1: `redirect_uri` MUST match a registered value exactly — no
wildcards, no path normalization, no trailing-slash tolerance. HTTPS is
required for all redirect URIs except `http://localhost` for development.

**Threat mitigated**: Open redirect via `redirect_uri` manipulation —
attacker crafts a similar URI to capture authorization codes.

### GGID Compliance

```go
// domain/models.go — exact string comparison
func (c *OAuthClient) ValidateRedirectURI(uri string) bool {
    for _, r := range c.RedirectURIs {
        if r == uri {   // simple == comparison
            return true
        }
    }
    return false
}
```

| Requirement | GGID Status | Compliant? |
|---|---|---|
| Exact string match | Yes — `r == uri` | Yes |
| Reject wildcards | Yes — no wildcard logic | Yes |
| HTTPS required (non-localhost) | **Not enforced** | **No** |
| Allow http://localhost | Implicitly allowed (no scheme check) | Yes |

**Gap**: No scheme validation. `http://evil.com` is accepted as a redirect
URI at registration time. Fix: add a `validateURIScheme()` check that
requires HTTPS except for `localhost`/`127.0.0.1`.

---

## 4. Deprecated Flows

### Implicit Grant (`response_type=token`)

RFC 9700 §2.1.2: The implicit grant is **REMOVED**. Tokens must not be
issued via the front-channel. Use authorization code + PKCE instead.

**Threat**: Token exposed in URL fragment (browser history, Referer header,
proxy logs). No client authentication.

### ROPC (`grant_type=password`)

RFC 9700 §2.1.1: The resource owner password credentials grant is
**REMOVED**. Use authorization code or device flow.

**Threat**: Client sees the user's raw password — prevents SSO, breaks
MFA, leaks credentials to every app.

### GGID Compliance

```go
// server.go — token endpoint grant types
switch grantType {
case "authorization_code":      // OK
case "refresh_token":           // OK
case "client_credentials":      // OK
case "urn:ietf:params:oauth:grant-type:device_code":  // OK
case "urn:ietf:params:oauth:grant-type:jwt-bearer":   // OK
default: return unsupported_grant_type  // password rejected implicitly
}
```

| Requirement | GGID Status | Compliant? |
|---|---|---|
| No implicit grant implementation | Not in token endpoint | Yes |
| No ROPC implementation | Rejected by default switch | Yes |
| Discovery does NOT advertise `token` | **Advertises `["code","token","id_token"]`** | **No** |
| Discovery does NOT advertise `password` | Not advertised | Yes |

**Gap**: `GetDiscoveryConfig()` advertises `"token"` in
`ResponseTypesSupported`, implying implicit support. Fix: remove `"token"`
and `"id_token"` from discovery; keep only `"code"`.

---

## 5. Refresh Token Security

RFC 9700 §2.2.2: Refresh token rotation is **RECOMMENDED**. On each use,
issue a new refresh token and invalidate the old one. If a previously-used
token is presented again, revoke the **entire family** (reuse detection).

Sender-constrained refresh tokens (DPoP per RFC 9445, or mTLS per RFC 8705)
are **RECOMMENDED** for high-security deployments.

### GGID Compliance

```go
// oauth_service.go — RefreshToken()
// 5. Reuse detection
if record.Used || record.Revoked {
    s.tokenRepo.RevokeAllRefreshTokens(ctx, req.TenantID, client.ID)
    return errors.Unauthenticated("refresh token reuse detected")
}
// 7. Mark old token as used (rotation)
s.tokenRepo.RevokeRefreshToken(ctx, req.TenantID, tokenHash)
// 9. Issue new refresh token (rotation)
newRecord := &domain.RefreshTokenRecord{
    ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
}
```

| Requirement | GGID Status | Compliant? |
|---|---|---|
| Refresh token rotation | Old token revoked, new one issued | Yes |
| Reuse detection | `RevokeAllRefreshTokens` on reuse | Yes |
| Max TTL (90 days) | 30 days default | Yes |
| Sender-constrained (DPoP/mTLS) | Not implemented | **No** (recommended) |

---

## 6. Mix-Up Mitigation

### The Attack

A user authenticates at multiple authorization servers (multi-provider SSO).
An attacker injects their authorization code into the victim's callback. The
RP exchanges the attacker's code, receiving a token for the attacker's
account. The victim believes they are logged in as themselves but are
actually operating inside the attacker's account.

### Mitigation: Issuer Identification

RFC 9700 §4: The RP MUST verify that tokens came from the expected AS.
The `state` parameter binds the authorization request to the RP's session.
The `iss` parameter in authorization responses (RFC 9207) identifies the AS.

### GGID Compliance

| Requirement | GGID Status | Compliant? |
|---|---|---|
| `state` parameter enforced | Yes — returns error if empty | Yes |
| `iss` in authorization response | Not implemented (RFC 9207) | **No** |
| Issuer claim in token responses | `iss` set in access/ID tokens | Yes |
| Issuer validation by RP (gateway) | Gateway validates `iss` in JWT | Yes |

**Gap**: RFC 9207 `iss` parameter not returned in authorization response.
This is an RP-side concern primarily, but GGID as an AS should include it.

---

## 7. Additional Security Requirements

### state Parameter
Enforced — `CreateAuthorizationCode` rejects empty `state`. The value is
cryptographically random (client responsibility).

### Token Lifetime
| Token Type | GGID Default | RFC 9700 | Compliant? |
|---|---|---|---|
| Access token (auth code) | 15 minutes | ≤15 min | Yes |
| Access token (device flow) | 1 hour | ≤15 min | **No** |
| Access token (JWT bearer) | 1 hour | ≤15 min | **No** |
| ID token | 1 hour | — | N/A |
| Refresh token | 30 days | ≤90 days | Yes |

### Audience Restriction
Access tokens include `aud` claim set to `client.ClientID`. Gateway validates
audience. This aligns with RFC 8707 (Resource Indicators).

### Confidential Client Authentication
```go
if client.IsConfidential() {
    ok, _ := crypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
    if !ok { return errors.Unauthenticated("invalid client credentials") }
}
```
Supports: `client_secret_basic`, `client_secret_post`, `private_key_jwt`,
`tls_client_auth`, `self_signed_tls_client_auth`.

### Browser Security (Gateway-Level)
```go
// gateway middleware — SecurityHeaders default config
CSP: "default-src 'self'; frame-ancestors 'none'",
// X-Frame-Options: DENY, X-Content-Type-Options: nosniff
```
CSP and clickjacking protection are enforced at the gateway. Token storage
(localStorage vs httpOnly cookie) is a client-side decision — GGID's SDK
documentation should recommend backend-session + httpOnly cookie.

### Scope Minimization
Scopes pass through from request without enforcement beyond client
allow-list. No automatic reduction logic. **Partial compliance.**

---

## 8. GGID Compliance Matrix

| # | RFC 9700 Requirement | GGID Status | Compliant? | Priority | Effort |
|---|---|---|---|---|---|
| 1 | PKCE for all clients | Public only | **No** | P0 | Low |
| 2 | S256 only (reject plain) | Plain accepted | **No** | P0 | Low |
| 3 | Redirect URI exact match | `==` comparison | Yes | — | — |
| 4 | HTTPS redirect URIs only | No scheme check | **No** | P0 | Low |
| 5 | No implicit grant | Not implemented | Yes | — | — |
| 6 | No ROPC | Not implemented | Yes | — | — |
| 7 | Discovery does not advertise removed flows | Advertises `token` | **No** | P1 | Trivial |
| 8 | Refresh token rotation | Implemented | Yes | — | — |
| 9 | Refresh reuse detection | Family revocation | Yes | — | — |
| 10 | state parameter mandatory | Enforced | Yes | — | — |
| 11 | Access token ≤15 min | 15 min (auth code); 1h (device/jwt) | **Partial** | P1 | Low |
| 12 | Scope minimization | Pass-through only | **Partial** | P1 | Medium |
| 13 | Audience restriction | `aud` claim set | Yes | — | — |
| 14 | Confidential client auth | Secret verified | Yes | — | — |
| 15 | Sender-constrained tokens | Not implemented | **No** | P2 | High |
| 16 | RFC 9207 `iss` in authz response | Not implemented | **No** | P1 | Low |
| 17 | CSP / X-Frame-Options | Gateway-level | Yes | — | — |
| 18 | Token storage guidance | Not documented | **No** | P1 | Low |

**Score: 9/18 fully compliant, 2 partial, 7 non-compliant.**

---

## 9. Threat Mitigation Summary

| Threat | RFC 9700 Mitigation | GGID Implementation |
|---|---|---|
| Code interception | PKCE (S256) | Public clients only; needs all-client enforcement |
| Open redirect | Exact URI match + HTTPS | Exact match done; HTTPS not enforced |
| CSRF | `state` parameter | Enforced |
| Token theft | Short TTL + sender-constrained | 15 min (auth code); no sender-constraining |
| Refresh theft | Rotation + reuse detection | Implemented with family revocation |
| Mix-up attack | Issuer identification | `state` enforced; `iss` response missing |
| Session fixation | Regenerate session ID | Auth service handles session refresh |
| XSS token theft | CSP + storage guidance | CSP at gateway; no SDK storage guidance |
| Code replay | Single-use code consumption | `ConsumeCode` atomic delete |
| Client impersonation | Client authentication | Secret verified for confidential clients |

---

## 10. Roadmap to Full Compliance

### P0 — Must Fix Immediately (Security-Critical)

| Item | Effort | Change |
|---|---|---|
| PKCE for all clients | Low | `RequiresPKCE()` → `return true` |
| Reject `plain` PKCE | Low | Remove from discovery; reject in `CreateAuthorizationCode` |
| HTTPS redirect URIs | Low | Add scheme validation in `ValidateRedirectURI` |

```go
// Proposed: enforce PKCE universally
func (c *OAuthClient) RequiresPKCE() bool { return true }

// Proposed: reject plain
if req.CodeChallengeMethod == "plain" {
    return "", errors.InvalidArgument("plain code_challenge_method is deprecated; use S256")
}

// Proposed: enforce HTTPS
if !strings.HasPrefix(uri, "https://") && !isLocalhost(uri) {
    return false // reject non-HTTPS
}
```

### P1 — Fix Within 1 Sprint

| Item | Effort | Change |
|---|---|---|
| Remove `token` from discovery ResponseTypesSupported | Trivial | Delete string from slice |
| Normalize device/JWT-bearer token TTL to 15 min | Low | Change `time.Hour` to `15 * time.Minute` |
| RFC 9207 `iss` parameter in authz response | Low | Add `iss` to redirect params |
| Token storage guidance in SDK docs | Low | Add security note to SDK README |

### P2 — Nice-to-Have (Advanced)

| Item | Effort | Change |
|---|---|---|
| Sender-constrained tokens (DPoP) | High | Implement RFC 9445 DPoP proof |
| Scope minimization engine | Medium | Add scope-reduction policy per client |
| mTLS client certificate binding | High | Implement RFC 8705 at token endpoint |

### Summary

**Current compliance: 9/18 requirements met (50%).** The three P0 items are
low-effort code changes (~20 lines total) that would bring GGID to 14/18
(78%). The remaining gaps (sender-constrained tokens, scope minimization)
require architectural work and are recommended for hardening but not
blocking for RFC 9700 baseline compliance.

**Estimated effort**: 1-2 days for P0+P1 (full baseline compliance), 2-4
weeks for P2 (advanced sender-constrained tokens).
