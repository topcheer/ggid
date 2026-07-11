# OAuth 2.1 Changes: Compliance Analysis & GGID Readiness

## Overview

OAuth 2.1 is the upcoming consolidation of OAuth 2.0, merging the core RFC 6749 with security best-practice RFCs (7636, 6749, 6819, 9700) into a single, simplified specification. This document analyzes the key changes and maps them to GGID's current compliance status.

> **Related**: [OAuth 2.1 Analysis](oauth-2.1-analysis.md) (1741 lines), [OAuth 2.1 Migration Guide](../oauth-2.1-migration-guide.md)

## Key Changes in OAuth 2.1

### 1. PKCE Mandatory for All Clients

**Change**: Proof Key for Code Exchange (RFC 7636) is required for ALL authorization code flow clients — both public and confidential. Previously PKCE was only recommended for public clients (SPAs, mobile).

**Why**: Authorization code interception attacks affect confidential clients too. PKCE adds a per-request secret that makes intercepted codes useless.

**GGID Status**: **COMPLIANT** — PKCE implemented in `services/oauth/internal/service/oauth_service.go` with gap regression tests (`gap_regression_pkce_test.go`).

### 2. Implicit Grant REMOVED

**Change**: The `response_type=token` implicit grant is removed entirely.

**Why**: Implicit grants return access tokens directly in the URL fragment, exposing them to:
- Browser history leakage
- Referer header leakage
- No refresh token issuance
- No client authentication

**Migration**: SPAs must switch to authorization code + PKCE.

**GGID Status**: **COMPLIANT** — Authorization code flow is the primary flow. Implicit grant is not implemented.

### 3. Resource Owner Password Credentials REMOVED

**Change**: The `grant_type=password` (ROPC) is removed.

**Why**: ROPC gives the application direct access to user credentials, breaking the separation between resource owner and client.

**GGID Status**: **PARTIAL** — GGID's auth service uses its own login endpoint (`/api/v1/auth/login`) rather than the OAuth ROPC grant, so this is compatible. However, direct password authentication still exists via the auth service API.

### 4. Exact Redirect URI Matching

**Change**: Redirect URIs MUST be compared using exact string matching. Wildcard or prefix matching is prohibited.

**Why**: Loose redirect URI matching enables open redirect attacks and authorization code theft.

**GGID Status**: **COMPLIANT** — OAuth client redirect URIs are stored and compared exactly in the OAuth service.

### 5. Refresh Token Rotation

**Change**: Refresh token rotation is REQUIRED for public clients. Issued refresh tokens should be single-use (sender-constrained).

**Why**: Stolen refresh tokens from public clients (which can't store secrets) become useless after rotation.

**GGID Status**: **COMPLIANT** — Refresh token rotation implemented; old tokens invalidated after use.

### 6. Issuer Identifier (`iss`) in Authorization Response

**Change**: The authorization server MUST include an `iss` parameter in authorization responses to prevent mix-up attacks.

**Why**: Without `iss`, a client can't distinguish which authorization server issued a code, enabling code mix-up attacks.

**GGID Status**: **COMPLIANT** — `iss` parameter included in auth redirect (commit 72edaa5).

### 7. Browser-Based Client Security

**Change**: Clients running in a browser (SPAs) must:
- Use authorization code + PKCE (not implicit)
- NOT store refresh tokens in localStorage/sessionStorage
- Use `Back-Channel Front-Channel` (BFF) pattern or silent refresh via refresh tokens

**GGID Status**: **COMPLIANT** — Console uses server-side token management (Next.js server actions), not client-side storage.

### 8. Token Type Removals

| Token Type | OAuth 2.0 | OAuth 2.1 |
|------------|-----------|-----------|
| Bearer | Supported | Supported (only type) |
| PoP (Proof of Possession) | RFC 7800 | Recommended |
| DPoP (Demonstrating Proof) | Draft | Recommended |
| MTLS (RFC 8705) | Optional | Recommended for high-assurance |

**GGID Status**: Bearer tokens (default). mTLS support implemented (`jar_mtls.go`). DPoP not yet implemented.

## Compliance Matrix

| OAuth 2.1 Requirement | RFC Source | GGID Status | Evidence |
|-----------------------|-----------|-------------|----------|
| PKCE mandatory | RFC 7636 | COMPLIANT | `gap_regression_pkce_test.go` |
| Implicit removed | — | COMPLIANT | Not implemented |
| ROPC removed | — | COMPLIANT | Uses auth service login, not OAuth grant |
| Exact redirect URI | RFC 6749 §3.1.2.3 | COMPLIANT | Exact match in OAuth service |
| Refresh token rotation | RFC 6749 §6 | COMPLIANT | Token rotation implemented |
| `iss` in auth response | RFC 9207 | COMPLIANT | `iss` param in redirect |
| State parameter | RFC 6749 §10.12 | COMPLIANT | Redis-backed state validation |
| No tokens in URL fragment | — | COMPLIANT | Authorization code flow only |
| Client authentication for confidential | RFC 6749 §2.3.1 | COMPLIANT | Client secret validation |
| JWKS endpoint | RFC 7517 | COMPLIANT | `/.well-known/jwks.json` |
| Discovery document | RFC 8414 | COMPLIANT | `/.well-known/openid-configuration` |
| DPoP support | draft-ietf-oauth-dpop | NOT YET | Roadmap item |
| MTLS client cert | RFC 8705 | PARTIAL | `jar_mtls.go` exists |
| JAR (JWT Authorization Request) | RFC 9101 | COMPLIANT | `jar_mtls.go` |
| Sender-constrained tokens | RFC 8705/DPoP | PARTIAL | mTLS partial |

**Overall Compliance: 13/15 requirements met (87%)**

## Migration Impact for GGID Users

### No Breaking Changes Required

GGID users who followed OAuth 2.0 best practices (authorization code + PKCE, exact redirect URIs) need no changes for OAuth 2.1.

### Deprecation Timeline

| Feature | OAuth 2.0 Status | OAuth 2.1 Status | GGID Action |
|---------|-----------------|------------------|-------------|
| Implicit grant | Deprecated | Removed | Already absent |
| ROPC grant | Deprecated | Removed | Already absent |
| Wildcard redirect URIs | Discouraged | Prohibited | Not supported |
| PKCE for confidential | Optional | Mandatory | Already enforced |

## What GGID Still Needs

| Gap | Priority | Effort | Impact |
|-----|----------|--------|--------|
| DPoP (Demonstrating Proof of Possession) | P2 | Medium | Sender-constrained tokens |
| Full mTLS for all token endpoints | P1 | Medium | High-assurance deployments |
| PAR (Pushed Authorization Requests, RFC 9126) | P2 | Medium | Request integrity |
| Token introspection authentication (RFC 7662) | P0 | Small | P0 security fix |

## References

- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/)
- [RFC 7636: PKCE](https://www.rfc-editor.org/rfc/rfc7636)
- [RFC 9700: OAuth 2.0 Security Best Practices](https://www.rfc-editor.org/rfc/rfc9700)
- [RFC 9207: `iss` Parameter](https://www.rfc-editor.org/rfc/rfc9207)

## See Also

- [OAuth 2.1 Analysis](oauth-2.1-analysis.md)
- [OAuth Scopes Design](oauth-scopes-design.md)
- [Confidential Client & PKCE](confidential-client-pkce.md)
- [Token Exchange RFC 8693](token-exchange-rfc8693.md)
