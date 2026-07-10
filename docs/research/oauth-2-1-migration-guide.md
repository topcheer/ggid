# OAuth 2.1 Migration Guide

> Practical migration checklist for moving GGID's OAuth/OIDC service to full
> OAuth 2.1 compliance. This doc focuses on **what to change and how** — for the
> detailed gap audit, see [oauth-2.1-analysis.md](./oauth-2.1-analysis.md).
>
> Status: OAuth 2.1 is still an IETF Internet Draft (draft-ietf-oauth-v2-1-15,
> Jan 2025). Its recommendations are already adopted by RFC 9700 and major IdPs.

---

## 1. What OAuth 2.1 Consolidates

OAuth 2.1 is not a new protocol — it consolidates a decade of RFCs, Best Current
Practice documents, and lessons learned into a single, normative specification.
The table below shows which RFCs are merged and their status.

| RFC | Title | What it contributed | OAuth 2.1 status |
|-----|-------|-------------------|-----------------|
| **6749** | OAuth 2.0 Core Framework | Base protocol, 4 grant types | Base framework, 2 flows removed |
| **6750** | Bearer Token Usage | `Authorization: Bearer` scheme | Retained unchanged |
| **8252** | Native Apps BCP | PKCE for mobile/desktop, custom URI schemes | PKCE now mandatory for ALL clients |
| **9700** | Security Best Current Practice | Threat model updates, redirect URI rules | Fully incorporated |
| **7636** | PKCE | `code_challenge` / `code_verifier` | Required, S256 only |
| **6819** | Threat Model (info) | Security considerations reference | Referenced as background |
| **7662** | Token Introspection | `/introspect` endpoint | Retained as optional |
| **7009** | Token Revocation | `/revoke` endpoint | Retained as optional |
| **8628** | Device Flow | `device_code` grant | Retained (replaces ROPC for CLI) |
| **8693** | Token Exchange | `grant_type=...jwt-bearer` delegation | Retained as optional |

**Key takeaway:** OAuth 2.1 removes complexity by eliminating insecure flows
(implicit, ROPC), makes PKCE mandatory, and codifies security best practices
that were previously scattered across 8+ RFCs.

---

## 2. Deprecated Flows

OAuth 2.1 removes two grant types entirely. All implementations must reject
these flows and migrate clients to alternatives.

### Implicit Grant (`response_type=token`)

**Why deprecated:**
- Access token exposed in URL fragment (visible to browser history, referrers, extensions)
- No client authentication — any party with the token can use it
- No PKCE support — vulnerable to authorization code interception
- Cannot issue refresh tokens (must re-authenticate each time)

**Replacement:** Authorization Code flow with PKCE.

**Migration impact on GGID:**
- GGID already rejects `response_type=token` at the authorize endpoint
  (server.go line 177: only `code` is accepted). **No code change needed here.**
- **Discovery document cleanup needed**: `response_types_supported` currently
  lists `"code", "token", "id_token"` — remove `"token"` to signal compliance.

### Resource Owner Password Credentials (ROPC)

**Why deprecated:**
- Client application sees the user's raw password
- Prevents SSO and multi-factor authentication
- Poor UX — users enter credentials into third-party apps
- Breaks phishing resistance (users trained to type passwords everywhere)

**Replacement:** Authorization Code + PKCE for interactive clients; Device Flow
(RFC 8628) for CLI/headless tools.

**Migration impact on GGID:**
- GGID does NOT implement ROPC — there is no `password` grant type in the
  token endpoint switch statement. **No code change needed.**
- The `grant_types_supported` in discovery already omits `password`.

**Both deprecated flows were never implemented in GGID**, so migration cost for
this section is limited to discovery document cleanup and client communication.

---

## 3. PKCE Mandatory for All Clients

PKCE (Proof Key for Code Exchange) was previously recommended for public clients
but optional for confidential ones. OAuth 2.1 makes it **mandatory for all
clients** regardless of type.

**Why mandatory:**
- Confidential clients (server-side apps) can also be compromised
- Code injection attacks work against all redirect flows, not just browser
- Defense in depth: PKCE is zero-cost and prevents entire attack classes

**Code challenge method:** Only `S256` is allowed. The `plain` method is
deprecated and must be rejected.

### GGID Current State

```go
// conf.go:27
RequirePKCE bool `yaml:"require_pkce"` // force PKCE for all auth_code flows

// domain/models.go:54 — PKCE auto-required for public clients
func (c *OAuthClient) RequiresPKCE() bool { return c.RequirePKCE || c.IsPublic() }
```

GGID has two layers of PKCE enforcement:
1. **Global config** (`RequirePKCE`): when true, rejects authorize requests
   without `code_challenge` for all clients (server.go line 184).
2. **Per-client flag** (`RequirePKCE`): individual clients can opt in.
3. **Public clients** always require PKCE via `RequiresPKCE()`.

### What Needs to Change

1. **Remove `plain` support** — currently accepted (server.go line 192, discovery
   line 321). Change validation to reject anything that isn't `S256`.

```go
// CURRENT (accepts plain):
if codeChallengeMethod != "S256" && codeChallengeMethod != "plain" {

// TARGET (S256 only, reject plain):
if codeChallengeMethod != "S256" {
    // reject: use code_challenge_method=S256
}

// Discovery document: remove "plain" from CodeChallengeMethodsSupported
CodeChallengeMethodsSupported: []string{"S256"},
```

2. **Set `RequirePKCE=true` as production default** — currently opt-in.

---

## 4. Other Key Requirements

### Exact Redirect URI Matching

OAuth 2.1 requires exact string comparison of `redirect_uri` against registered
URIs. No wildcard matching, no path normalization.

**GGID status:** Already compliant. `ValidateRedirectURI` does exact `==`
comparison (domain/models.go line 67-74). No change needed.

### HTTPS Required for Redirect URIs

All redirect URIs must use HTTPS. The only exception is `localhost`/`127.0.0.1`
for local development.

**GGID status:** **NOT enforced.** There is no scheme validation on redirect
URIs anywhere in the codebase. Need to add:

```go
func isValidRedirectScheme(uri string) bool {
    if strings.HasPrefix(uri, "https://") {
        return true
    }
    // Allow http://localhost and http://127.0.0.1 for dev
    if strings.HasPrefix(uri, "http://localhost") || strings.HasPrefix(uri, "http://127.0.0.1") {
        return true
    }
    return false
}
```

### State Parameter Mandatory

The `state` parameter is required for CSRF protection on all authorization
requests.

**GGID status:** Already enforced (oauth_service.go line 170-172).

### Refresh Token Rotation + Reuse Detection

Each refresh token use issues a new token and invalidates the old one. If a
previously-used token is presented, the entire token family is revoked (reuse
detection).

**GGID status:** **Already implemented** (oauth_service.go lines 626-697).
`RefreshToken()` rotates tokens and calls `RevokeAllRefreshTokens` on reuse
detection. Fully compliant.

### Client Authentication at Token Endpoint

Confidential clients must authenticate at the token endpoint using
`client_secret_basic` or `client_secret_post`.

**GGID status:** Implemented (oauth_service.go line 246-251). Both
`client_secret_basic` and `client_secret_post` are supported.

### No `client_secret` in Browser/URL

Public clients (SPAs, mobile apps) must use PKCE instead of a client secret.
**GGID status:** Compliant — public clients don't receive secrets at creation.

---

## 5. GGID Migration Checklist

| # | Item | Current State | Change Needed | Effort | Risk |
|---|------|--------------|---------------|--------|------|
| 1 | Audit existing OAuth clients for implicit/ROPC usage | Implicit rejected; ROPC not implemented | Query `oauth_clients` table, check `grant_types`/`response_types` columns for `token` or `password` values | S | L |
| 2 | Enable PKCE enforcement globally | `RequirePKCE` exists but defaults to `false` | Set `require_pkce: true` in production config | S | M |
| 3 | Deprecate `plain` code_challenge_method | Both `S256` and `plain` accepted | Reject `plain` in server.go + remove from discovery doc | S | M |
| 4 | Enforce exact redirect URI matching | Already doing exact `==` comparison | None — already compliant | — | — |
| 5 | Require HTTPS for redirect URIs | **Not enforced** — any scheme accepted | Add `isValidRedirectScheme()` check in authorize + client creation | S | M |
| 6 | Enforce `state` parameter on auth requests | Already enforced (oauth_service.go:170) | None — already compliant | — | — |
| 7 | Implement refresh token rotation + reuse detection | Already implemented with reuse detection | None — already compliant | — | — |
| 8 | Update discovery document (remove `token` response type, remove `plain` method) | Lists `"code", "token", "id_token"` and `"S256", "plain"` | Remove `"token"` from `ResponseTypesSupported`, remove `"plain"` from `CodeChallengeMethodsSupported` | S | L |
| 9 | Update SDK defaults to always send PKCE | SDKs may not generate `code_verifier` by default | Add PKCE generation in Go/Node/Java SDK `Authorize()` methods | M | L |
| 10 | Document breaking changes for API consumers | No deprecation notices | Add deprecation headers, publish migration timeline, notify client developers | M | L |

### Detailed Action Items

**Item 2 — Enable PKCE enforcement:**
```yaml
# config.yaml
oauth:
  require_pkce: true  # was false (opt-in)
```
Roll out via feature flag first to identify non-PKCE clients, then enforce.

**Item 3 — Reject `plain` method:**
```go
// server.go, authorize handler — replace the plain/S256 check:
if codeChallenge != "" && codeChallengeMethod != "S256" {
    writeJSON(w, http.StatusBadRequest, map[string]string{
        "error": "invalid_request",
        "error_description": "code_challenge_method must be S256",
    })
    return
}
```

**Item 5 — HTTPS validation on redirect URIs:**
Add scheme check in both `CreateClient` and the authorize endpoint. Reject
`http://` for any host other than `localhost`/`127.0.0.1`.

**Item 8 — Discovery document cleanup:**
```go
// oauth_service.go GetDiscoveryConfig()
ResponseTypesSupported:        []string{"code", "id_token"}, // removed "token"
CodeChallengeMethodsSupported: []string{"S256"},             // removed "plain"
```

---

## 6. SDK Impact

GGID ships SDKs for Go, Node, and Java. All three must be updated to generate
PKCE challenges by default.

### Go SDK
```go
// pkg/sdk/go/ — generate code_verifier + code_challenge automatically
func (c *Client) AuthorizeURL() (string, string, error) {
    verifier, err := generateCodeVerifier()
    // challenge = base64url(sha256(verifier))
    challenge := computeS256Challenge(verifier)
    // append code_challenge + code_challenge_method=S256 to auth URL
}
```

### Node SDK
```typescript
// pkg/sdk/node/ — same PKCE generation logic
const verifier = crypto.randomBytes(32).toString('base64url');
const challenge = crypto.createHash('sha256').update(verifier).digest('base64url');
```

### Java SDK
```java
// pkg/sdk/java/ — SecureRandom for verifier, MessageDigest for challenge
String verifier = Base64.getUrlEncoder().withoutPadding()
    .encodeToString(SecureRandom.getSeed(32));
String challenge = Base64.getUrlEncoder().withoutPadding()
    .encodeToString(MessageDigest.getInstance("SHA-256").digest(verifier.getBytes()));
```

### Migration for Existing SDK Users
1. **Minor version bump** — PKCE is added transparently, no API breaking change
2. **State parameter generation** — SDKs should auto-generate `state` if not provided
3. **Migration notice** — add deprecation log if `code_challenge` is not sent

---

## 7. Timeline

| Phase | Duration | Actions | Feature Flag |
|-------|----------|---------|--------------|
| **Phase 1: Audit** | Weeks 1-2 | Query all registered clients, identify implicit/ROPC usage, notify client developers | All flags off |
| **Phase 2: Preparation** | Weeks 3-4 | SDK PKCE updates, HTTPS validation, discovery doc cleanup | `oauth_strict_redirects=true` (new clients only) |
| **Phase 3: Soft enforcement** | Months 2-3 | Enable `require_pkce=true` with warning log for non-compliant clients (reject but log instead of hard error) | `require_pkce=warn` |
| **Phase 4: Hard enforcement** | Month 4 | Enable strict PKCE, reject `plain` method, reject non-HTTPS redirects | `require_pkce=enforce`, `reject_plain_method=true` |
| **Phase 5: Cleanup** | Month 6 | Remove backward-compatibility code paths, publish final migration notice | All flags removed, OAuth 2.1 strict mode is default |

### Backward Compatibility Window

- **3-6 months** recommended between Phase 3 (soft) and Phase 4 (hard)
- During this window, non-compliant requests should be **logged and rejected
  with a helpful error message** rather than silently accepted
- Monthly review of rejection logs to identify stragglers

### Feature Flag Strategy

```yaml
# config.yaml — phased rollout
oauth:
  require_pkce: warn        # Phase 3: log + reject
  reject_plain_method: false # Phase 3: accept with deprecation header
  require_https_redirects: false  # Phase 2: new clients only
  strict_mode: false        # Phase 4: flip all to true
```

Each flag can be toggled independently, allowing incremental enforcement without
a single big-bang deployment.

---

## Summary

GGID is already well-positioned for OAuth 2.1 compliance:

| Requirement | Status |
|------------|--------|
| No implicit grant | Already rejected |
| No ROPC | Never implemented |
| PKCE for public clients | Already enforced |
| State parameter mandatory | Already enforced |
| Exact redirect URI matching | Already compliant |
| Refresh token rotation | Already implemented |
| Refresh token reuse detection | Already implemented |

**Remaining work (4 items, all Small effort):**
1. Set `require_pkce: true` as production default
2. Reject `plain` code_challenge_method
3. Add HTTPS validation for redirect URIs
4. Clean up discovery document (remove `token` response type, `plain` method)

Total estimated effort: **2-3 developer days** for code changes, plus a
3-6 month rollout window for client migration.
