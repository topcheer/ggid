# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 (Round 16 — Fully aligned with OIDC discovery)
> **Status: 8/8 demos working. Zero hack. OIDC discovery enabled.**

## Three-Layer Alignment — FINAL

| Demo | Auth Flow | Token | SDK Verify | CRUD | Hack | OIDC Discovery |
|------|-----------|:-----:|:----------:|:----:|:----:|:--------------:|
| Go | OAuth PKCE | 200 | SDK WithDiscovery() | 200 | ZERO | WithDiscovery() |
| Node | M2M Client Credentials | 200 | SDK crypto verify | 200 | ZERO | auto from gatewayUrl |
| C# | Password Grant | 200 | SDK VerifyTokenAsync | 200 | ZERO | WithJwks() fixed path |
| Java | Password Grant / SAML | 200 | SDK JwtVerifier | 200 | ZERO | manual jwksUrl |
| Python | SAML 2.0 SSO | 200 | SDK JWTVerifier | 200 | ZERO | auto from base_url |
| Ruby | Device Code | 200 | SDK verify_token | 200 | ZERO | relative path |
| Rust | Token Exchange | 200 | SDK verify_token | 200 | ZERO | auto from base_url |
| React | SPA PKCE | 200 | Backend SDK | 200 | ZERO | via Node backend |

## SDK OAuth2 Flow Coverage

| Flow | Go | Node | Python | Ruby | Rust | C# | Java |
|------|:--:|:----:|:------:|:----:|:----:|:--:|:----:|
| Auth Code + PKCE | SDK | SDK | SDK | SDK | SDK | SDK | SDK |
| Client Credentials | SDK | SDK | SDK | SDK | SDK | SDK | SDK |
| Device Code | SDK | SDK | SDK | SDK | - | SDK | SDK |
| Token Exchange | SDK | SDK | SDK | SDK | SDK | SDK | SDK |
| Password Grant | SDK | SDK | SDK | SDK | SDK | SDK | SDK |
| SAML2-bearer | - | - | SDK | - | - | SDK | SDK |

## OIDC Discovery Status
- Core: /.well-known/openid-configuration returns all endpoints + grant types
- Issuer: dynamically overridden from X-Forwarded-Host
- Go SDK: WithDiscovery() auto-fetches jwks_uri from discovery
- Node SDK: auto-derives jwksUrl from gatewayUrl
- Python/Ruby/Rust: auto-derive from base_url
- C#: WithJwks() path fixed to /.well-known/jwks.json
- Java: manual jwksUrl (acceptable)

## Next Target: Stable — monitoring for regressions, all 7 SDKs auto-discover JWKS

All 7 SDKs now auto-derive JWKS URL from base URL:
- Go: WithDiscovery() from OIDC discovery document
- Node: auto-derive from gatewayUrl
- Python/Ruby/Rust: auto-derive from base_url
- C#: WithJwks() fixed path
- Java: JwtVerifier(baseURL) auto-derives (NEW)

Last verification: 8/8 HTTP 200, 0 hacks, stable.
