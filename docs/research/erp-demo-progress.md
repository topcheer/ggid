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
#### Round 20: auth_code refresh token fix (c78591362), 8/8 stable
#### Round 21: oauth refresh scope fix (bd7c3b647,14984c4e7), 8/8 stable, 0 hacks
#### Round 22: IAM review R1 (11 commits), discovery+introspection+PKCE+TOTP, 8/8 stable

## Dimension 1: Authentication Completeness (Round 23)
- Password grant: 6/7 tenants OK (Rust uses token_exchange, not password grant — correct)
- Client credentials (Node M2M): OK
- Token structure: access_token + token_type=Bearer + expires_in=900, consistent across all
- Refresh token: NOT issued on password grant (even with offline_access scope) — core behavior
- No-token 401: PASS
- Token usable: All tokens successfully verify and access demo APIs

### Issues Found
1. Go/Ruby/Rust inventory empty (items=0) — data initialization issue, not auth
2. Refresh token not issued on password grant — core layer decision
3. Node/Python/Java have seeded data (items=2-3), others don't

### Next Dimension: 2 — Authorization Boundaries (role + permission testing)

## Dimension 2: Authorization Boundaries (Round 24)
- Admin permissions: 9 items (inventory CRUD + orders CRUD + audit + dashboard) ✅
- Admin access inventory/orders: 200 ✅
- Fake token: 401 ✅
- Cross-demo admin permissions consistent: all 200 ✅
- C# my-permissions returns correct perms matching JWT ✅

### Issues Found
1. Go demo missing /api/my-permissions endpoint (other demos have it)
2. Go demo order approve uses PUT (other demos use POST) — API inconsistency
3. No viewer-level user to test 403 denial (all test users are admin)

### Next Dimension: 3 — Demo Functional Completeness

## Dimension 3: Demo Functional Completeness (Round 25)
- Inventory: Node(3)/Python(3)/C#(2)/Java(3) have data with correct fields ✅
- Go/Ruby inventory empty (data init issue)
- POST create + GET verify: C# PASS ✅
- my-permissions: C#/Python return correct perms ✅, Java missing endpoint
- Orders: real data but field naming inconsistent across demos

### Issues Found
1. Go/Ruby demo inventory empty — no seed data
2. Java missing /api/my-permissions endpoint
3. Orders field naming inconsistent: node(amount), python(qty), java(productName)
4. Rust demo uses erp-rust-exchange not erp-rust-demo for token exchange

### Next Dimension: 4 — Multi-tenant Isolation

## Dimension 4: Multi-tenant Isolation (Round 26)
- JWT tenant_id correctly set for each tenant ✅
- **CROSS-TENANT TOKEN ACCEPTED** — Go tenant token works on Java/C# demo ⚠️ SECURITY GAP
- GGID API cross-tenant: Go token + X-Tenant-ID:00000006 → 200 (gateway doesn't enforce tenant match)

### Root Cause
SDK verifyToken validates JWT signature + expiry but does NOT validate tenant_id.
Each demo accepts any valid GGID token regardless of tenant.

### Impact
- Low for demo (separate demo instances per tenant)
- HIGH for production — cross-tenant data access possible

### Recommendation
- SDK: add optional tenant_id verification to verifyToken (compare JWT tenant_id with configured tenant)
- Demo: pass expected tenant_id to SDK verifyToken
- Gateway: enforce X-Tenant-ID matches JWT tenant_id on API calls

### Next Dimension: 5 — SDK Cross-language Consistency

## Post-D4: Gateway tenant isolation fix verified (31c7e5c1e)
- Cross-tenant: 401 ✅ (was 200 before fix)
- Same-tenant: 200 ✅
- 8/8 demos still working ✅
- SDK layer: no action needed (gateway enforces tenant boundary)

## Dimension 5: SDK Cross-language Consistency (Round 27)
- login() return types: all return typed TokenSet/TokenResponse (except Python/Ruby return dict/Hash) ✅
- Token field names: all use snake_case JSON tags matching OAuth2 standard ✅
  Go: access_token/expires_in/token_type/refresh_token
  Node: same, Rust: same, C#: JsonPropertyName, Java: @JsonProperty
- verifyToken: all return Claims with permissions field ✅
  Go: UserInfo.Permissions, Node: JWTClaims.permissions, Python: JWTClaims.permissions
  Ruby: GGIDUser.permissions, Rust: Claims.permissions, C#: Claims.Permissions
  Java: GGIDUser.permissions
- API endpoints: all 7 SDKs use /api/v1/oauth/token ✅
- Python/Ruby return untyped dict/Hash (vs typed in other SDKs) — acceptable for dynamic languages

### Issues Found
1. Python/Ruby login() returns raw dict/Hash — no typed TokenSet (minor, language convention)
2. All SDKs consistent on endpoint paths and field names — GOOD

### Next Dimension: 6 — End-to-end User Experience

## Dimension 6: End-to-end User Experience (Round 28)
- Full E2E flow on C# demo: login→perms→read→create→verify→order ALL PASS ✅
- No-token rejection: 7/7 demos return 401 ✅
- Invalid/malformed token: rejected ✅
- 0 hacks confirmed ✅

### E2E Results
1. Login: token obtained ✅
2. Permissions: 9 items returned ✅
3. Read: 3 inventory items ✅
4. Create: new item created (id=p004) ✅
5. Verify: item visible in GET (4 items, E2E found) ✅
6. Order: created with status=pending ✅
7. No token: 401 on all demos ✅
8. Invalid token: 403 ✅

### ALL 6 DIMENSIONS COMPLETE — cycling back to Dimension 1

## Dimension Summary (Rounds 23-28)
- D1 Auth: 6/7 password grant OK, refresh token gap noted
- D2 AuthZ: admin perms consistent, Go demo missing my-permissions
- D3 Functional: 4/7 demos pass full content validation
- D4 Tenant isolation: GAP found → FIXED by arch (gateway enforces)
- D5 SDK consistency: all 7 SDKs aligned on field names + endpoints
- D6 E2E: full user flow verified, all security checks pass

## Dimension 1 R2: Auth Completeness (Round 29)
- Password grant: 5/5 tenants PASS (Bearer + 900s) ✅
- Client credentials (Node M2M): PASS ✅
- Token usable: 6/6 demos HTTP 200 ✅
- Issuer: https://ggid.iot2.win ✅
- 0 hacks

### Next Dimension: 2 — Authorization Boundaries

## Dimension 2 R2: Authorization Boundaries (Round 30)
- Admin perms: 9 items consistent ✅
- Cross-tenant: 401 ✅ (gateway enforces)
- Same-tenant: 200 ✅
- Fake token: 401 ✅
- All 7 demos admin access: inv=200 ord=200 ✅
- 0 hacks

### Next Dimension: 3 — Demo Functional Completeness

## Dimension 3 R2: Functional Completeness (Round 31)
- Inventory: Node(3)/Python(3)/C#(5)/Java(3) have data with fields ✅
- Go/Ruby still empty (known seed data issue, not regression)
- POST create→verify: C# PASS (id=p005, found in GET) ✅
- my-permissions: 9 perms accurate ✅
- Orders: Node(2)/Python(2)/Java(3) ✅
- 0 hacks, no regression from R1

### Next Dimension: 4 — Multi-tenant Isolation

## Dimension 4 R2: Multi-tenant Isolation (Round 32)
- JWT tenant_id correct for Go/Java ✅
- Cross-tenant Go→Java: 401 ✅
- Cross-tenant Java→Go: 401 ✅ (bidirectional verified)
- Same-tenant controls: both 200 ✅
- 0 hacks

### Next Dimension: 5 — SDK Cross-language Consistency
