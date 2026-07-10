# OAuth 2.1 Migration Guide

OAuth 2.1 consolidates OAuth 2.0 (RFC 6749), best practices, and security
improvements (BCPs) into a single specification. This guide explains what
changed, what breaks, and how to make your GGID deployment fully compliant.

---

## Table of Contents

- [What Is OAuth 2.1?](#what-is-oauth-21)
- [Key Changes Summary](#key-changes-summary)
- [PKCE Required for All Flows](#pkce-required-for-all-flows)
- [Implicit Grant Removed](#implicit-grant-removed)
- [Redirect URI Exact Match](#redirect-uri-exact-match)
- [RS256+ Only (No HS256, No none)](#rs256-only-no-hs256-no-none)
- [Token Lifetime Restrictions](#token-lifetime-restrictions)
- [Refresh Token Rotation](#refresh-token-rotation)
- [State Parameter Required](#state-parameter-required)
- [GGID Compliance Checklist](#ggid-compliance-checklist)
- [Client Migration Steps](#client-migration-steps)
- [Breaking Changes Reference](#breaking-changes-reference)

---

## What Is OAuth 2.1?

OAuth 2.1 (draft-ietf-oauth-v2-1) is not a completely new protocol. It is a
consolidation of OAuth 2.0 with years of security best practices published as
BCPs (Best Current Practices). The goal: one spec that is secure by default,
with fewer footguns.

### RFCs Incorporated

| Source RFC/BPF | Topic |
|----------------|-------|
| RFC 6749 | OAuth 2.0 framework (base) |
| RFC 6750 | Bearer token usage |
| RFC 6819 | Threat model and security considerations |
| RFC 7636 | PKCE (Proof Key for Code Exchange) |
| RFC 7009 | Token revocation |
| RFC 7662 | Token introspection |
| RFC 8252 | Native app best practices |
| RFC 9700 | OAuth 2.0 security best practices |

---

## Key Changes Summary

| Feature | OAuth 2.0 | OAuth 2.1 | Impact |
|---------|-----------|-----------|--------|
| PKCE | Recommended for public clients | **Required for ALL clients** | High |
| Implicit grant | Supported | **Removed** | Breaking |
| Resource owner password grant | Supported | **Removed** | Breaking |
| Redirect URI matching | Prefix/wildcard allowed | **Exact match only** | Breaking |
| `state` parameter | Recommended | **Required** | Medium |
| Token `none` algorithm | Some implementations | **MUST reject** | Security |
| HS256 for JWTs | Common | **Discouraged, RS256+ only** | Medium |
| Nested JWTs inside access tokens | Common | Use opaque or structured | Low |
| Refresh token rotation | Optional | **Recommended (RFC 6749 §10.4)** | Medium |

---

## PKCE Required for All Flows

### What Changed

In OAuth 2.0, PKCE (RFC 7636) was recommended only for public clients (SPAs,
mobile apps). OAuth 2.1 mandates PKCE for **every** authorization code flow,
including confidential clients (server-side apps with client secrets).

### Why

PKCE prevents authorization code interception attacks. Without PKCE, a
malicious app on the same device can intercept the authorization code redirect
and exchange it for tokens.

### How PKCE Works

```
Client                          Authorization Server
  │                                    │
  │  1. Generate code_verifier         │
  │     (43-128 char random string)    │
  │                                    │
  │  2. code_challenge =               │
  │     BASE64URL(SHA256(verifier))    │
  │                                    │
  │  3. Authorization request          │
  │     with code_challenge            │
  │     and code_challenge_method=S256 │
  ├───────────────────────────────────►│
  │  4. Authorization code             │
  │◄───────────────────────────────────┤
  │                                    │
  │  5. Token request with             │
  │     code + code_verifier           │
  ├───────────────────────────────────►│
  │  6. Server verifies:               │
  │     SHA256(verifier) == challenge  │
  │                                    │
  │  7. Access token                   │
  │◄───────────────────────────────────┤
```

### Migration: Adding PKCE to Existing Clients

#### Server-Side (Confidential Client)

```python
# Before (OAuth 2.0 — no PKCE)
auth_url = f"{issuer}/authorize?response_type=code&client_id={client_id}&redirect_uri={redirect_uri}&state={state}"

# After (OAuth 2.1 — PKCE required)
import secrets, hashlib, base64

code_verifier = secrets.token_urlsafe(64)
code_challenge = base64.urlsafe_b64encode(
    hashlib.sha256(code_verifier.encode()).digest()
).rstrip(b'=').decode()

auth_url = (
    f"{issuer}/authorize?"
    f"response_type=code"
    f"&client_id={client_id}"
    f"&redirect_uri={redirect_uri}"
    f"&state={state}"
    f"&code_challenge={code_challenge}"
    f"&code_challenge_method=S256"
)

# Store code_verifier in session for token exchange
session['code_verifier'] = code_verifier
```

#### Native App (iOS/Android)

```swift
// iOS with AppAuth
let codeVerifier = OIDAuthorizationRequest.generateCodeVerifier()
let codeChallenge = OIDAuthorizationRequest.codeChallengeS256(forVerifier: codeVerifier)

let request = OIDAuthorizationRequest(
    configuration: config,
    clientId: clientId,
    scopes: ["openid", "profile"],
    redirectURL: redirectURI,
    responseType: .code,
    state: state,
    nonce: nonce,
    codeVerifier: codeVerifier,
    codeChallenge: codeChallenge,
    codeChallengeMethod: OIDOAuthorizationRequestCodeChallengeMethodS256
)
```

### GGID Configuration

```yaml
oauth:
  pkce:
    required: true              # Reject auth requests without code_challenge
    method_required: true       # code_challenge_method must be S256
    allowed_methods:            # plain is NOT allowed
      - "S256"
```

### Verification

```bash
# Request WITHOUT PKCE → should be rejected
curl "https://iam.example.com/oauth/authorize?response_type=code&client_id=app&redirect_uri=https://app.example.com/callback"

# Expected: 400 Bad Request
{
  "error": "invalid_request",
  "error_description": "code_challenge is required"
}
```

---

## Implicit Grant Removed

### What Changed

The implicit grant (`response_type=token`) is removed in OAuth 2.1. Clients
must use the authorization code flow with PKCE instead.

### Why

The implicit grant:
- Exposes access tokens in the URL fragment (browser history, referrer leaks)
- Cannot use PKCE (no code exchange step)
- Cannot issue refresh tokens safely
- Is vulnerable to token injection

### Migration

#### Before (Implicit)

```
# Old: redirect with token in fragment
https://app.example.com/callback#access_token=eyJhbG...

# Auth request
GET /authorize?response_type=token&client_id=app&redirect_uri=...
```

#### After (Authorization Code + PKCE)

```
# New: redirect with code, exchange for token server-side
https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyz

# Auth request
GET /authorize?response_type=code
    &client_id=app
    &redirect_uri=...
    &code_challenge=...
    &code_challenge_method=S256
    &state=...
```

### SPA Migration (Single Page Applications)

Modern SPAs should **not** use the implicit flow or even the authorization
code flow in the browser. Instead:

| Approach | Recommendation |
|----------|---------------|
| Backend-for-Frontend (BFF) | **Recommended** — token exchange happens server-side |
| Authorization Code + PKCE in SPA | Acceptable for read-only apps |
| Implicit | **Removed in OAuth 2.1** |

```
┌──────────────────────────────────────────────────┐
│                    BFF Pattern                    │
│                                                   │
│  Browser ──► SPA ──► BFF ──► Authorization Server │
│             (no     (holds  (token exchange +     │
│            tokens)  tokens)  refresh)              │
└──────────────────────────────────────────────────┘
```

---

## Redirect URI Exact Match

### What Changed

OAuth 2.0 allowed redirect URI prefix matching or wildcard patterns. OAuth 2.1
requires **exact string matching** of the redirect URI.

### Why

Prefix matching enables open redirect attacks. If `https://app.example.com/`
is allowed, an attacker could use `https://app.example.com.evil.com/callback`
to steal authorization codes.

### Migration

#### Register Exact URIs

```bash
# Register each callback URL individually
curl -X POST https://iam.example.com/api/v1/oauth/clients \
  -d '{
    "client_id": "my-app",
    "redirect_uris": [
      "https://app.example.com/auth/callback",
      "https://app.example.com/auth/callback?flow=login",
      "https://staging.example.com/auth/callback"
    ]
  }'
```

#### Native App Redirect URIs

Native apps use claimed HTTPS scheme or custom URI scheme:

```
com.example.app:/oauth2redirect
https://app.example.com/oauth2redirect
```

> Custom schemes must use reverse domain notation: `com.example.app`.

### GGID Configuration

```yaml
oauth:
  redirect_uri:
    exact_match: true          # Default in OAuth 2.1 mode
    case_sensitive: true
    allow_loopback: true       # http://127.0.0.1:* for native apps
    allowed_schemes:
      - "https"
      - "http"                 # Only for localhost/127.0.0.1
```

### Loopback Exception

For native apps using loopback redirect (`http://127.0.0.1:PORT/callback` or
`http://localhost:PORT/callback`), GGID allows port wildcard:

```bash
# Register once (no port)
redirect_uris: ["http://127.0.0.1/callback", "http://localhost/callback"]

# Any port works at runtime
# http://127.0.0.1:8080/callback  ✓
# http://127.0.0.1:9173/callback  ✓
# http://localhost:3000/callback  ✓
```

---

## RS256+ Only (No HS256, No none)

### What Changed

OAuth 2.1 requires asymmetric signing algorithms (RS256, ES256, EdDSA) for
JWTs. Symmetric algorithms (HS256) and `none` are prohibited.

### Why

| Algorithm | Problem |
|-----------|---------|
| `none` | Allows unsigned tokens — trivially forged |
| `HS256` | Client shared secret used for signing — any client can forge tokens |
| `RS256` | **Safe** — only the authorization server has the private key |

### Migration

#### Change JWT Signing Algorithm

```yaml
oauth:
  jwt:
    signing_algorithm: "RS256"   # or ES256, EdDSA
    # Do NOT use:
    # - HS256 (symmetric)
    # - none (unsigned)
```

#### Regenerate Client Secrets

Clients previously using HS256 shared their secret with the authorization
server. After switching to RS256, these secrets are no longer used for signing
but may still be used for client authentication:

```bash
# Rotate client secrets
curl -X POST https://iam.example.com/api/v1/oauth/clients/{id}/rotate-secret
```

### Supported Algorithms in GGID

| Algorithm | JWS alg | Key Type | Status |
|-----------|---------|----------|--------|
| RS256 | `RS256` | RSA 2048+ | **Default** |
| RS384 | `RS384` | RSA 2048+ | Supported |
| RS512 | `RS512` | RSA 2048+ | Supported |
| ES256 | `ES256` | EC P-256 | Supported |
| ES384 | `ES384` | EC P-384 | Supported |
| ES512 | `ES512` | EC P-521 | Supported |
| EdDSA | `EdDSA` | Ed25519 | Supported |
| HS256 | `HS256` | HMAC | **Rejected** |
| none | `none` | N/A | **Rejected** |

### Verification

```bash
# Check a token's algorithm
echo 'eyJhbG...' | cut -d. -f1 | base64 -d | jq .alg
# Should output: "RS256" (or ES256, EdDSA)
# NEVER: "HS256" or "none"
```

---

## Token Lifetime Restrictions

OAuth 2.1 best practices recommend shorter access token lifetimes.

| Token Type | OAuth 2.0 Common | OAuth 2.1 Recommended | GGID Default |
|------------|-----------------|----------------------|-------------|
| Access token | 1 hour | 5-15 minutes | 15 minutes |
| Refresh token | Unlimited | Session-bound, rotated | 24 hours |
| ID token | 1 hour | Short-lived | 15 minutes |
| Authorization code | 10 minutes | 30 seconds - 10 minutes | 5 minutes |

### Configuration

```yaml
oauth:
  token:
    access_token_lifetime: "15m"
    refresh_token_lifetime: "24h"
    id_token_lifetime: "15m"
    auth_code_lifetime: "5m"
    refresh_token_reuse_interval: "0s"  # Revoke immediately on reuse
```

---

## Refresh Token Rotation

OAuth 2.1 recommends refresh token rotation (RFC 6749 §10.4): each use of a
refresh token issues a new refresh token and invalidates the old one.

### Rotation Flow

```
1. Client sends refresh_token: RT-A
2. Server validates RT-A
3. Server issues new tokens:
   - access_token: AT-2
   - refresh_token: RT-B (new)
4. Server marks RT-A as used
5. If RT-A is used again → REVOKE entire token family
```

### Reuse Detection

If a used refresh token is presented again, GGID detects token theft and
revokes the entire token family:

```
# Attacker stole RT-A, uses it after legitimate client already rotated:
1. Legitimate client: RT-A → RT-B (RT-A now invalid)
2. Attacker: RT-A → DETECTED → revoke RT-A, RT-B, and all derived tokens
```

### Configuration

```yaml
oauth:
  refresh_token:
    rotation: true             # Rotate on each use
    reuse_detection: true      # Revoke family on reuse
    grace_period: "0s"         # No grace period (strict)
```

---

## State Parameter Required

### What Changed

The `state` parameter was recommended in OAuth 2.0 but optional. OAuth 2.1
makes it effectively required to prevent CSRF attacks.

### Why

Without `state`, an attacker can inject their own authorization code into the
victim's session (CSRF attack on the redirect endpoint).

### Migration

```python
# Generate state per authorization request
import secrets

state = secrets.token_urlsafe(32)
session['oauth_state'] = state

# Include in auth URL
auth_url = f"{issuer}/authorize?...&state={state}"

# Verify on callback
if request.args.get('state') != session.pop('oauth_state'):
    abort(400, "Invalid state parameter")
```

---

## GGID Compliance Checklist

Use this checklist to verify your GGID deployment is OAuth 2.1 compliant.

### Server-Side Configuration

- [ ] PKCE required for all clients (`oauth.pkce.required = true`)
- [ ] Only S256 challenge method allowed
- [ ] Implicit grant disabled
- [ ] Resource owner password grant disabled
- [ ] Redirect URI exact match enabled
- [ ] Loopback redirect allowed for native apps only
- [ ] JWT signed with RS256/ES256/EdDSA (not HS256)
- [ ] `none` algorithm rejected
- [ ] Access token lifetime ≤ 15 minutes
- [ ] Refresh token rotation enabled
- [ ] Refresh token reuse detection enabled
- [ ] State parameter required on authorization endpoint

### Client Registration

- [ ] Every client has PKCE enabled
- [ ] No client uses `response_type=token`
- [ ] No client uses `grant_type=password`
- [ ] All redirect URIs are exact (no wildcards)
- [ ] Confidential clients have strong secrets
- [ ] Public clients use PKCE only (no secret)

### Security

- [ ] HTTPS enforced on all endpoints
- [ ] Token endpoint requires client authentication
- [ ] Authorization endpoint rate-limited
- [ ] Redirect URI validated against allowlist
- [ ] CORS configured to only allow registered origins
- [ ] OpenID Connect `nonce` required for ID token validation

---

## Client Migration Steps

### Step 1: Inventory Existing Clients

```bash
# List all OAuth clients
curl https://iam.example.com/api/v1/oauth/clients \
  -H "Authorization: Bearer <admin-token>"

# Check which use implicit or password grants
curl https://iam.example.com/api/v1/oauth/clients?grant_type=implicit
curl https://iam.example.com/api/v1/oauth/clients?grant_type=password
```

### Step 2: Update Clients to Authorization Code + PKCE

```bash
# Update a client to disable implicit and password, enable PKCE
curl -X PATCH https://iam.example.com/api/v1/oauth/clients/{id} \
  -d '{
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "pkce_required": true,
    "redirect_uris": [
      "https://app.example.com/callback"
    ]
  }'
```

### Step 3: Fix Redirect URIs

```bash
# List current redirect URIs (check for wildcards)
curl https://iam.example.com/api/v1/oauth/clients/{id} | jq .redirect_uris

# Replace with exact matches
curl -X PATCH https://iam.example.com/api/v1/oauth/clients/{id} \
  -d '{
    "redirect_uris": [
      "https://app.example.com/auth/callback",
      "https://app.example.com/auth/callback?flow=signup"
    ]
  }'
```

### Step 4: Update JWT Algorithm

```bash
# Check current algorithm
curl https://iam.example.com/.well-known/openid-configuration | jq .id_token_signing_alg_values_supported

# Ensure RS256+ only
```

### Step 5: Enable OAuth 2.1 Mode

```yaml
oauth:
  version: "2.1"
  # This flag enables all OAuth 2.1 requirements:
  # - PKCE required
  # - Implicit disabled
  # - Password disabled
  # - Exact redirect URI match
  # - RS256+ only
```

### Step 6: Test

```bash
# Verify PKCE is required (should fail without it)
curl "https://iam.example.com/oauth/authorize?response_type=code&client_id=test&redirect_uri=https://app/callback"
# Expected: 400 invalid_request

# Verify implicit is disabled
curl "https://iam.example.com/oauth/authorize?response_type=token&client_id=test&redirect_uri=https://app/callback"
# Expected: 400 unsupported_response_type

# Verify redirect URI exact match
curl "https://iam.example.com/oauth/authorize?response_type=code&client_id=test&redirect_uri=https://evil.com/callback&code_challenge=xxx&code_challenge_method=S256"
# Expected: 400 invalid_request
```

---

## Breaking Changes Reference

### Quick Migration Matrix

| Old Behavior | OAuth 2.1 Replacement | Action Required |
|--------------|-----------------------|-----------------|
| `response_type=token` | `response_type=code` + PKCE | Update SPAs to auth code flow |
| `grant_type=password` | Authorization code + PKCE | Migrate mobile/desktop apps |
| Wildcard redirect URI | Exact redirect URI | Register each URI |
| No PKCE in server apps | PKCE for all clients | Add code_verifier/challenge |
| Optional `state` | Required `state` | Generate and verify state |
| HS256 tokens | RS256/ES256/EdDSA | Change signing algorithm |
| No refresh rotation | Rotate on each use | Enable rotation |
| Long access tokens | 15-minute max | Shorten lifetime |

### Deprecation Timeline

| Phase | Duration | Behavior |
|-------|----------|----------|
| Phase 1: Warning | 30 days | OAuth 2.0 clients get deprecation headers |
| Phase 2: Grace period | 30 days | Non-compliant requests warned but processed |
| Phase 3: Enforcement | Permanent | Non-compliant requests rejected |
