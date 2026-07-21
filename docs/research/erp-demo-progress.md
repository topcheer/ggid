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

## Next Target: Stable — monitoring for regressions

#### Round 18: dynamic RBAC commit (a0ab6ea19), 8/8 stable, no impact

#### Round 17 verification (core change check):
- New commits since last: a7584a360 (Console Settings), 633a2f401 (JWT scopes/roles fix), edea85e7c (RBAC ADR)
- Unstaged WIP: pkg/saml assertion signing refactor + OAuth trust chain validator (arch working)
- Core endpoints: OIDC discovery ✅, JWT claims ✅ (iss/aud/perms/roles), JWKS 2 keys ✅
- OIDC grant_types now includes `password` ✅
- **Impact on SDK/Demo: NONE** — SAML internal refactor + Console UI fixes
- 8/8 demos HTTP 200, 0 hacks confirmed

#### Round 19: 6 core commits (RBAC+refresh rotation+audit WORM), 8/8 stable
