# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 03:40

## Overall: Code 8/8 | Deploy 8/8 | CRUD 7/8 | Auth Adapted 6/8

| # | Lang | Code | Tenant | Auth Method | Auth Status | k8s | CRUD | Notes |
|---|------|------|--------|-------------|-------------|-----|------|-------|
| 1 | Go | ✅ | 0001 | OAuth2 PKCE | ⏳ DM shen | ✅ | ✅ inv+create+dash+audit | Full CRUD verified |
| 2 | Node | ✅ | 0002 | Client Creds | ✅ M2M | ✅ | ✅ inv+create | JWT via JWKS local verify |
| 3 | React | ✅ | 0003 | SPA PKCE | ✅ PKCE | ✅ | ❌ 404 | SPA routing issue (no API routes) |
| 4 | Python | ✅ | 0004 | SAML SSO | ✅ SAML | ✅ | ✅ inv+create | Full CRUD verified |
| 5 | C# | ✅ | 0005 | Password Grant | ✅ Password | ✅ | ✅ inv+create | Full CRUD verified |
| 6 | Java | ✅ | 0006 | SAML SSO | ✅ SAML | ✅ | ✅ inv+create+dash | Fixed permissions claim |
| 7 | Ruby | ✅ | 0007 | Device Code | ⏳ DM guardian | ✅ | ✅ inv+create+dash+audit | Fixed before filter |
| 8 | Rust | ✅ | 0008 | Token Exchange | ✅ | ✅ | ✅ inv+create+dash+audit | Full CRUD verified |

## CRUD Verification Detail

| Demo | Inventory List | Inventory Create | Dashboard | Audit | Status |
|------|---------------|-----------------|-----------|-------|--------|
| Go | ✅ 200 | ✅ 201 | ✅ 200 | ✅ 200 | PASS |
| Node | ✅ 200 | ✅ 201 | N/A (404) | N/A | PASS |
| React | ❌ 404 | ❌ 404 | ❌ | ❌ | FAIL (SPA no API) |
| Python | ✅ 200 | ✅ 201 | N/A | N/A | PASS |
| C# | ✅ 200 | ✅ 201 | N/A (404) | N/A (404) | PASS |
| Java | ✅ 200 | ✅ 201 | ✅ 200 | ✅ | PASS |
| Ruby | ✅ 200 | ✅ 201 | ✅ 200 | ✅ 200 | PASS |
| Rust | ✅ 200 | ✅ 201 | ✅ 200 | ✅ 200 | PASS |

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

### All 8 Pods Running
- 8/8 deployments with correct tenant IDs
- 8/8 ingress routes configured
- All demos accessible via https://erp-{lang}.iot2.win

### Tenant Setup
- 8 ERP tenants created (0001-0008)
- Each tenant has admin_{lang} user with password q7Rf9Xk2Lm3pW8zBA
- Each tenant has ERP Admin role with 9 ERP permissions
- Each tenant has built-in roles (Administrator, Tenant Admin, User, Viewer)

### JWT Claims (deployed)
- `permissions`: fine-grained permission keys (inventory:read, orders:write, etc.)
- `roles`: role names (ERP Admin, Tenant Administrator, etc.)
- `scopes`: OAuth scopes only (role names, will be cleaned up)

## Fixes Applied This Session
1. JWT permissions/scope separation in auth service
2. All SDK fallback logic removed (Go, Java, Python, Ruby, React, Node, Rust, Console)
3. Ruby SDK Accept-Encoding header fix (empty HTTParty response)
4. Ruby Sinatra before filter fix (regex → string match)
5. Node demo JWKS-based JWT verification (removed GW verify dependency)
6. Java demo permissions claim fix (scope string → permissions array)
7. Rust SDK JWKS endpoint + Claims struct + runtime env vars
8. 8 ERP tenants with users, roles, permissions bootstrapped

## Known Issues / Next Steps
1. **React**: Next.js SPA has no API routes — needs serverless API or backend proxy
2. **Go PKCE**: Auth method adaptation pending (shen)
3. **Ruby Device Code**: Auth method adaptation pending (guardian)
4. **Dashboard/Audit**: Some demos don't implement these endpoints (Node, C#, Python)
5. **Multi-tenant isolation test**: Need to verify cross-tenant data isolation
