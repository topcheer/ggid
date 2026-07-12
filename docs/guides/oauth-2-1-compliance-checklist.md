# OAuth 2.1 Compliance Checklist

Per-item pass/fail criteria for OAuth 2.1 requirements: PKCE, implicit/ROPC removal, redirect URIs, refresh rotation, DPoP, PAR, JAR, state enforcement.

## PKCE (Mandatory)

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 1 | PKCE required for all authorization code flows | `require_pkce: true` globally | ✅ |
| 2 | Only S256 challenge method accepted | `plain` rejected with error | ✅ |
| 3 | code_verifier is 43-128 chars, high entropy | Validated at token endpoint | ✅ |
| 4 | code_verifier not sent in authorization request | Only challenge sent | ✅ |
| 5 | New verifier per authorization request | Not reused across sessions | ✅ |

## Implicit & ROPC Removal

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 6 | `response_type=token` (implicit) rejected | Returns error on authorize | ✅ |
| 7 | `response_type=id_token` rejected | Returns error | ✅ |
| 8 | `grant_type=password` (ROPC) rejected | Returns error on token endpoint | ✅ |
| 9 | Only `response_type=code` and `code id_token` allowed | Validated in config | ✅ |

## Redirect URI Exact Match

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 10 | Exact string match (no wildcards) | `redirect_uri` must match registered | ✅ |
| 11 | HTTPS required (except localhost) | HTTP rejected for production | ✅ |
| 12 | No fragment (#) in redirect URI | Validated on registration | ✅ |
| 13 | Localhost exemption for development | `http://localhost:*` allowed | ✅ |

## Refresh Token Rotation

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 14 | New refresh token on each refresh | Old token invalidated | ✅ |
| 15 | Reuse detection revokes family | Family-based revocation | ✅ |
| 16 | Rotation grace period for race conditions | 30s window | ✅ |
| 17 | Confidential clients: sender-constrained | DPoP or mTLS binding | ✅ |

## Token Security

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 18 | Access tokens ≤15 min TTL | `access_token_ttl: 900s` | ✅ |
| 19 | Refresh tokens ≤30 day TTL (confidential) | `refresh_token_ttl: 30d` | ✅ |
| 20 | `aud` claim enforced per service | Token rejected if wrong audience | ✅ |
| 21 | `iss` claim validated | Must match GGID issuer | ✅ |
| 22 | `jti` anti-replay via Redis blacklist | Checked on every request | ✅ |
| 23 | Algorithm restricted to RS256/ES256 | `none` and HS256 rejected | ✅ |

## State Parameter

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 24 | State parameter required on authorize | Request rejected if missing | ✅ |
| 25 | State validated on callback | Constant-time comparison | ✅ |
| 26 | State bound to session | Cookie/sessionStorage | ✅ |
| 27 | State single-use | Invalidated after callback | ✅ |

## DPoP / Token Binding

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 28 | DPoP supported for all client types | Proof JWT verified | ✅ |
| 29 | Per-client DPoP enforcement | `require_dpop: true` option | ✅ |
| 30 | mTLS supported as alternative | Client cert thumbprint in cnf | ✅ |
| 31 | Bearer tokens not default for SPA/mobile | DPoP recommended | ⚠️ Recommended |

## PAR (Pushed Authorization Requests)

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 32 | PAR endpoint available | `POST /par` returns request_uri | ✅ |
| 33 | Per-client PAR enforcement | `require_par: true` option | ✅ |
| 34 | request_uri single-use, 60s TTL | Validated on authorize | ✅ |

## JAR (JWT-Secured Authorization Requests)

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 35 | Signed request object supported | `request` parameter accepted | ✅ |
| 36 | `iss` must equal `client_id` | Validated in JAR processing | ✅ |
| 37 | `aud` must match authorization server | Validated | ✅ |
| 38 | `exp` required, max 60 min | Validated | ✅ |

## Discovery & Metadata

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 39 | `.well-known/openid-configuration` published | All endpoints listed | ✅ |
| 40 | JWKS endpoint available | `/.well-known/jwks.json` | ✅ |
| 41 | `require_pushed_authorization_requests` in metadata | Advertises PAR support | ✅ |
| 42 | DPoP signing alg supported advertised | Listed in metadata | ✅ |

## Dynamic Client Registration

| # | Requirement | Pass Criteria | Status |
|---|------------|---------------|--------|
| 43 | RFC 7591 registration endpoint | `POST /register` | ✅ |
| 44 | RFC 7592 management (read/update/delete) | RAT-protected | ✅ |
| 45 | Software statement support | Signed JWT accepted | ✅ |
| 46 | Scope restrictions by registration type | Tiered scope access | ✅ |

## Summary

| Category | Total | Pass | Warning | Fail |
|----------|-------|------|---------|------|
| PKCE | 5 | 5 | 0 | 0 |
| Flow Restrictions | 4 | 4 | 0 | 0 |
| Redirect URI | 4 | 4 | 0 | 0 |
| Refresh Rotation | 4 | 4 | 0 | 0 |
| Token Security | 6 | 6 | 0 | 0 |
| State | 4 | 4 | 0 | 0 |
| Token Binding | 4 | 3 | 1 | 0 |
| PAR | 3 | 3 | 0 | 0 |
| JAR | 4 | 4 | 0 | 0 |
| Discovery | 4 | 4 | 0 | 0 |
| DCR | 4 | 4 | 0 | 0 |
| **Total** | **46** | **45** | **1** | **0** |

## See Also

- [OAuth PKCE Deep Dive](oauth-pkce-deep-dive.md)
- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [OAuth Refresh Token Rotation](oauth-refresh-token-rotation.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
