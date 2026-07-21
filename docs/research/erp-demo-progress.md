# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 04:00
> **Status: 8/8 CRUD VERIFIED**

## Overall: Code 8/8 | Deploy 8/8 | CRUD 8/8 ✅ | Auth Adapted 6/8

| # | Lang | Tenant | Auth Method | Auth | k8s | CRUD | Notes |
|---|------|--------|-------------|------|-----|------|-------|
| 1 | Go | 0001 | OAuth2 PKCE | ⏳ shen | ✅ | ✅ | inv+create+dash+audit |
| 2 | Node | 0002 | Client Creds (M2M) | ✅ | ✅ | ✅ | JWKS local verify |
| 3 | React | 0003 | SPA PKCE | ✅ | ✅ | ✅ | via Node backend API |
| 4 | Python | 0004 | SAML SSO | ✅ | ✅ | ✅ | inv+create |
| 5 | C# | 0005 | Password Grant | ✅ | ✅ | ✅ | inv+create |
| 6 | Java | 0006 | SAML SSO | ✅ | ✅ | ✅ | inv+create+dash, /api/ prefix |
| 7 | Ruby | 0007 | Device Code | ⏳ guardian | ✅ | ✅ | inv+create+dash+audit |
| 8 | Rust | 0008 | Token Exchange | ✅ | ✅ | ✅ | inv+create+dash+audit |

## Auth Method Adaptation

| Demo | Required | Status | Owner |
|------|----------|--------|-------|
| Go | PKCE (OIDC) | ⏳ Pending | shen_frontend |
| Node | Client Credentials | ✅ Done | — |
| React | SPA PKCE | ✅ Done | — |
| Python | SAML SSO | ✅ Done | arch |
| C# | Password Grant | ✅ Done | — |
| Java | SAML SSO | ✅ Done | backend |
| Ruby | Device Code | ⏳ Pending | guardian |
| Rust | Token Exchange | ✅ Done | backend |

## Infrastructure

### 8/8 Pods Running, 8/8 Ingress Configured
- All demos accessible via https://erp-{lang}.iot2.win
- All tenant IDs correctly configured in k8s deployments

### Tenant Setup (8 Independent Tenants)
- 8 ERP tenants (0001-0008) with admin users (admin_{lang})
- Each tenant: ERP Admin role + 9 permissions (inventory:*, orders:*, audit:read, dashboard:read)
- Built-in roles: Administrator, Tenant Admin, User, Viewer
- JWT claims fully separated: permissions / roles / scopes

## Key Fixes This Session
1. **JWT permissions/scope separation** — auth service `getUserScopesAndPermissions()` returns independent claims
2. **All SDK fallback removed** — no scope-as-permission logic anywhere
3. **Ruby** — Sinatra before filter fix + SDK Accept-Encoding header
4. **Node** — JWKS-based local JWT verification (crypto.createPublicKey)
5. **Java** — verifyToken reads `permissions` claim directly
6. **Rust** — JWKS endpoint + Claims struct + runtime env vars + Axum route fix
7. **8 ERP tenants** bootstrapped with users, roles, permissions

## Remaining Work (Delegated to Team)
1. **Go PKCE** — shen_frontend (DM sent)
2. **Ruby Device Code** — guardian_security (DM sent)
3. **React SPA** — API works via Node backend; standalone SPA API routes optional
