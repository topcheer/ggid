# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 07:00
> **Status: Go SDK P0 FIXED. 6 SDKs + 6 demos still need work**

## Overall: Deploy 8/8 | CRUD 8/8 | **SDK Usage: 3/8** | **Signature Verification: 2/8 → 3/8 (Go fixed)**

## Critical Finding: Demos Don't Actually Use SDKs

The demos "work" at the HTTP level, but most bypass the SDK entirely with inline JWT decoding and raw HTTP calls. This means the demos do NOT validate GGID's SDK capabilities.

### Demo SDK Integration Audit

| Demo | SDK Imported | Token Verify | Permission Check | Auth Flow | HACKS |
|------|:-----------:|:------------:|:----------------:|:---------:|-------|
| Go | YES | **FIXED** ✅ VerifyToken requires JWKS (commit b82fa39f3) | Missing in demo | SDK has ClientCredentials() now; PKCE still manual HTTP in demo | Go SDK fixed: ClientCredentials() added, ParseUnverified removed, WithJWKS required. Demo still needs PKCE via SDK |
| Node | **NO** | Manual JWKS+crypto | Manual jwt.decode | Manual HTTP | middleware/auth.ts does NOT import SDK; reimplements verify from scratch |
| React | **NO** | atob() inline | atob() inline | N/A (SPA) | auth.ts decodes JWT with atob() — no SDK, no verification |
| Python | YES (import) | **Inline base64** | Inline base64 | Manual HTTP SAML | Imports GGIDClient but verify_token bypasses jwt_verifier.py |
| C# | **NO** | **Inline base64** | Inline base64 | Manual HTTP | Program.cs uses raw http.PostAsync; C# SDK HAS LoginAsync() + VerifyTokenAsync() but unused |
| Java | PARTIAL | **Inline base64** | GGIDUser.hasPermission() | Manual HTTP | Main.java splits JWT manually; AuthHandler uses HttpURLConnection |
| Ruby | YES | **SDK verify_token** ✅ | SDK has_permission? ✅ | Manual HTTP device | SDK used for verify, but device code flow is raw Net::HTTP |
| Rust | YES | **SDK verify_token** ✅ | Missing in demo | Manual HTTP exchange | SDK used for verify, but token exchange is raw HTTP |

### SDK OAuth2 Flow Coverage Matrix

| Flow | Go | Node | Python | Ruby | Rust | C# | Java |
|------|:--:|:----:|:------:|:----:|:----:|:--:|:----:|
| Auth Code + PKCE | Missing method | Missing method | Missing | YES | YES | YES (GetAuthorizeUrl) | YES |
| Client Credentials | **FIXED** ✅ | **MISSING** → assigned frontend | **MISSING** → assigned backend | **MISSING** → assigned backend | **MISSING** → assigned backend | **MISSING** → assigned backend | **MISSING** → assigned backend |
| Device Code | YES (ref only) | YES (ref only) | Missing | Missing | Missing | Missing | YES |
| Token Exchange | YES (agent) | YES (agent) | YES (agent) | YES (agent) | YES (agent) | YES (agent) | YES (agent) |
| Password Grant | Missing | YES (login) | YES (login) | YES (login) | YES (login) | YES (login) | YES (login) |
| SAML | YES | Missing | YES | Missing | Missing | YES | Missing |

**Critical Gap**: NO SDK has a `clientCredentials()` method — all 7 SDKs are missing M2M support.

### Security Issues

1. ~~**Go SDK `verifyTokenOffline`**~~: **FIXED** (commit b82fa39f3) — VerifyToken now requires WithJWKS(), fails closed if not configured. ClientCredentials() method added.
2. **5/8 demos decode JWT without signature verification**: Node, React, Python, C#, Java still decode JWT payload without verifying the RS256 signature. → assigned to backend (Python/Ruby/Rust/C#) + frontend (Node/React)
3. **4/8 demos don't import the SDK at all**: Node, React, C#, Python (imports but doesn't use verify). → assigned to frontend + backend

## Action Items (Priority Order)

### P0: SDK Signature Verification
1. ~~**Go SDK**~~: **FIXED** ✅ (commit b82fa39f3) — VerifyToken requires WithJWKS(), ClientCredentials() added
2. **C# demo**: Replace inline base64 decode with `GGIDClient.VerifyTokenAsync()` → assigned backend
3. **Java demo**: Replace `Main.verifyToken()` inline decode with `GGIDClient.verifyUser()` → assigned backend
4. **Node demo**: Replace manual JWKS+crypto with `GGIDClient.verifyToken()` → assigned frontend
5. **Python demo**: Use `jwt_verifier.verify()` instead of inline base64 → assigned backend
6. **React demo**: Use SDK's `verifyToken()` for token validation → assigned frontend

### P1: SDK OAuth2 Flow Methods
1. Add `ClientCredentials(clientID, clientSecret)` to ALL 7 SDKs (currently 0/7)
2. Add `DeviceCode(clientID)` + `PollDeviceToken(deviceCode)` to Python, Ruby, Rust, C# SDKs
3. Add `GetAuthorizeURL()` + `ExchangeCode()` to Go SDK
4. Add SAML SSO helper to Node, Ruby, Rust, Java SDKs

### P2: Demo Rewrite to Use SDK
1. Each demo must call SDK methods for ALL auth operations
2. No inline JWT decode, no raw HTTP to GGID API, no manual JWKS
3. Demos serve as SDK integration tests — if SDK has a gap, fix the SDK, don't hack the demo

### P3: IAM Platform Compliance
1. Token introspection endpoint should be used by resource servers
2. JWKS rotation should be transparent to SDK consumers
3. Permission checks should go through SDK's `checkPermission()` (PDP call) not just JWT claim read
