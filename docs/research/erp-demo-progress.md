# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 08:20 (Round 5 — C# demo complete)
> **Status: Go+Node+C# SDK integration COMPLETE. Next: Java demo.**

## Overall: Deploy 8/8 | CRUD 8/8 | **SDK Usage: 6/8** | **Sig Verify: 6/8** | **Core GAPs: 0 (all fixed)**

### Completed (SDK integration done)
- **Go demo**: 6/6 — GetAuthorizeURL, ExchangeCode, VerifyToken, Login, Refresh via SDK
- **Node demo**: 4/4 — verifyToken, clientCredentials via SDK, zero scope fallback
- **C# demo**: 4/4 — LoginAsync, VerifyTokenAsync via SDK; SDK Claims adds Permissions field
- **Ruby demo**: 3/4 — verify_token + has_permission via SDK (device code still raw HTTP)
- **Rust demo**: 2/4 — verify_token via SDK (token exchange + perms still raw)

### Next Target: Java demo (score 2/4)
Java demo uses SDK GGIDUser for permissions but:
- Main.verifyToken() does raw HttpURLConnection to introspect (should use SDK JwtVerifier)
- AuthHandler uses raw HttpURLConnection for SAML + token exchange
- Needs: import SDK properly, use JwtVerifier.verifyUser() for token verify

---

## Layer 1: SDK Signature Verification

| SDK | Unsafe Pattern | Proper Verify | Status |
|-----|:---:|:---:|--------|
| Go | ParseUnverified for kid extraction only (safe) | YES — VerifyToken requires WithJWKS() | **FIXED** ✅ |
| Node | None in SDK | SDK has verifyToken() | OK (but demo doesn't use it) |
| Python | None | SDK has jwt_verifier.verify() | OK (but demo doesn't use it) |
| Ruby | None | SDK has verify_token() | OK |
| Rust | None | SDK has verify_token() | OK |
| C# | None | SDK has VerifyTokenAsync() | OK (but demo doesn't use it) |
| Java | None | SDK has JwtVerifier.verifyUser() | OK (but demo doesn't use it) |

## Layer 1: SDK OAuth2 Flow Coverage Matrix

| Flow | Go | Node | Python | Ruby | Rust | C# | Java | GAPs |
|------|:--:|:----:|:------:|:----:|:----:|:--:|:----:|------|
| Auth Code + PKCE | MISSING | MISSING | MISSING | YES | YES | YES | MISSING | 4 SDKs missing |
| Client Credentials | **YES** ✅ | **YES** ✅ | MISSING | MISSING | MISSING | MISSING | MISSING | 5 SDKs missing |
| Device Code | YES | YES | MISSING | MISSING | MISSING | MISSING | YES | 4 SDKs missing |
| Token Exchange | YES | YES | YES | YES | YES | YES | YES | All have it |
| Password Grant | YES (login) | YES (login) | YES (login) | YES (login) | YES (login) | MISSING | MISSING | 2 SDKs missing |
| SAML | YES | YES | YES | YES | YES | YES | YES | All have it |
| Permission Check | YES | YES | YES | YES | YES | YES | YES | All have it |
| Token Mgmt | YES | YES | YES | YES | YES | YES | YES | All have it |

### SDK Gaps Requiring Implementation

| Priority | SDK | Missing Method | Assigned To |
|----------|-----|----------------|-------------|
| P1 | Python | clientCredentials() | backend |
| P1 | Ruby | clientCredentials() | backend |
| P1 | Rust | clientCredentials() | backend |
| P1 | C# | clientCredentials() | backend |
| P1 | Java | clientCredentials() | backend |
| P1 | Go | getAuthorizeURL() + exchangeCode() | — |
| P1 | Node | getAuthorizeURL() + exchangeCode() | frontend |
| P1 | Python | getAuthorizeURL() + exchangeCode() | backend |
| P1 | Java | getAuthorizeURL() + exchangeCode() | backend |
| P1 | Python | deviceCode flow | backend |
| P1 | Ruby | deviceCode flow | backend |
| P1 | Rust | deviceCode flow | backend |
| P1 | C# | deviceCode flow | backend |
| P1 | C# | passwordGrant (login) | backend |
| P1 | Java | passwordGrant (login) | backend |

---

## Layer 2: Demo Hack Detection

### Inline JWT Decode (MUST use SDK verifyToken instead)

| Demo | File:Line | Current Hack | Should Use | Status |
|------|-----------|-------------|------------|--------|
| Go | orders.go:51 | `parts[1]` for path routing (NOT JWT, false positive) | N/A | OK |
| C# | Program.cs:77 | `token.Split('.')` + base64 decode + JSON parse for permissions | `GGIDClient.VerifyTokenAsync()` | **HACK** |
| Java | Main.java:73-79 | Changed to introspect endpoint ✅ (but still raw HttpURLConnection) | Should use SDK | **PARTIAL** |
| Java | AuthHandler.java:116 | Base64.decode for SAML response | SDK SAML handler | **HACK** |
| Python | main.py (stashed) | Changed to introspect_token() ✅ | Correct direction | **IN PROGRESS** |
| React | auth.ts (stashed) | Changed to backend introspect ✅ | Correct direction | **IN PROGRESS** |

### Manual JWKS / Crypto Verify (MUST use SDK instead)

| Demo | File:Line | Current Hack | Should Use | Status |
|------|-----------|-------------|------------|--------|
| Node | middleware/auth.ts:22-63 | Manual JWKS fetch + cache + crypto.createPublicKey + jwt.verify | `GGIDClient.verifyToken()` | **HACK** |

### Raw HTTP Calls to GGID API (MUST use SDK methods instead)

| Demo | File:Line | Current Hack | Should Use | Status |
|------|-----------|-------------|------------|--------|
| Node | routes/auth.ts:19 | `fetch(GGID_URL/api/v1/oauth/token)` | `GGIDClient.clientCredentials()` | **HACK** |
| Node | routes/auth.ts:37 | `fetch(GGID_URL/api/v1/auth/verify)` | `GGIDClient.verifyToken()` | **HACK** |
| Node | routes/auth.ts:48 | `fetch(GGID_URL/api/v1/oauth/introspect)` | `GGIDClient.introspectToken()` | **HACK** |
| React | auth.ts:42 | `fetch(GGID_URL/api/v1/oauth/introspect)` | Backend SDK introspect | **IN PROGRESS** |
| React | auth.ts:150 | `fetch(GGID_URL/api/v1/oauth/token)` | SDK PKCE flow | **HACK** |
| C# | Program.cs:54 | `http.PostAsync(ggidUrl/api/v1/auth/login)` | `GGIDClient.LoginAsync()` | **HACK** |
| Java | AuthHandler.java:95 | `HttpURLConnection(TOKEN_ENDPOINT)` | SDK OAuth method | **HACK** |
| Java | Main.java:79 | `HttpURLConnection(GGID_URL/api/v1/oauth/introspect)` | SDK introspect | **PARTIAL** |
| Ruby | app.rb:65,90 | `Net::HTTP` for device_authorize + token poll | SDK deviceCode (needs impl) | **HACK** (SDK gap) |

### Demo SDK Import Status

| Demo | Imports SDK | Uses SDK for verify | Uses SDK for auth flow | Uses SDK for perms | Score |
|------|:-----------:|:-------------------:|:----------------------:|:------------------:|:-----:|
| Go | YES | YES (WithJWKS) ✅ | NO (manual PKCE) | NO | 2/4 |
| Node | **NO** | NO | NO | NO | **0/4** |
| React | **NO** | partial (introspect) | NO | NO | 1/4 |
| Python | YES | partial (stashed fix) | NO | NO | 1/4 |
| C# | **NO** | NO | NO | NO | **0/4** |
| Java | PARTIAL | partial (introspect) | NO | YES (GGIDUser) | 2/4 |
| Ruby | YES | YES ✅ | NO | YES ✅ | **3/4** |
| Rust | YES | YES ✅ | NO | NO | 2/4 |

---

## Layer 3: IAM Platform Standard Compliance

### JWT Claims — PASS ✅
- iss: PRESENT (ggid-auth)
- aud: PRESENT (ggid)
- exp/iat: PRESENT
- sub: PRESENT
- tenant_id: PRESENT
- permissions: 9 items (independent claim)
- roles: 2 items (independent claim)
- scope: empty string (not used for permissions)

### OIDC Discovery — PARTIAL
- issuer: ✅ https://ggid.iot2.win
- jwks_uri: ✅ https://ggid.iot2.win/oauth/jwks
- token_endpoint: ✅
- authorization_endpoint: ✅
- device_authorization_endpoint: **MISSING** — should advertise RFC 8628 endpoint
- introspection_endpoint: ✅
- grant_types_supported: `['authorization_code', 'refresh_token', 'client_credentials']`
  - **MISSING**: `urn:ietf:params:oauth:grant-type:device_code`, `urn:ietf:params:oauth:grant-type:token-exchange`

### JWKS Endpoint — PASS ✅
- 2 keys, RS256, sig usage

### PDP (Policy Decision Point) — FAIL
- `/api/v1/policy/check` returns "quarantined" status — policy engine is in security hold
- SDK's `checkPermission()` cannot work until policy is un-quarantined

### Token Introspection — FAIL
- `/api/v1/oauth/introspect` returns `{"error":"invalid_client"}`
- Requires client authentication but no client credentials provided in introspect call
- Demos using introspect (Java, Python stashed fix, React stashed fix) will fail

### IAM Platform Gaps

| Priority | Issue | Impact |
|----------|-------|--------|
| P0 | Token Introspection returns invalid_client | Demos can't verify tokens via introspect |
| P0 | PDP policy engine quarantined | SDK checkPermission() non-functional |
| P1 | OIDC discovery missing device_authorization_endpoint | Clients can't auto-discover device flow |
| P1 | OIDC grant_types missing device_code + token_exchange | Flow discovery incomplete |

---

## Layer 4: Deploy & CRUD — PASS ✅

All 8 pods Running. All 7 backends return inventory data. Token issuance works.

---

## Action Items Summary

### Immediate (blocks demo SDK integration)
1. **Fix token introspection** — must return active+claims without requiring client auth for bearer token introspection (RFC 7662)
2. **Un-quarantine PDP** — policy engine must be active for checkPermission to work
3. **Update OIDC discovery** — add device_authorization_endpoint + missing grant types

### SDK Implementation (assigned)
4. **backend**: Python/Ruby/Rust/C#/Java — add clientCredentials(), deviceCode, getAuthorizeURL
5. **frontend**: Node/React — add getAuthorizeURL, rewrite demos to use SDK verifyToken

### Demo Rewrite (after SDK methods exist)
6. **Node demo**: Import SDK, replace manual JWKS+crypto, replace raw fetch calls
7. **C# demo**: Import SDK, replace inline base64 + raw http.PostAsync
8. **Java demo**: Use SDK JwtVerifier instead of raw HttpURLConnection
9. **Go demo**: Use SDK getAuthorizeURL + exchangeCode (when implemented)
10. **Ruby demo**: Use SDK deviceCode flow (when implemented)
11. **Rust demo**: Use SDK exchangeToken flow (when implemented)
