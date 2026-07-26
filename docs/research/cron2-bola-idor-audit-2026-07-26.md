# Cron-2 Attack Surface #0: BOLA/IDOR Deep Audit

**Date**: 2026-07-26  
**Auditor**: guardian_security_agent  
**Attack Surface**: Broken Object Level Authorization / Insecure Direct Object Reference  
**Methodology**: Attacker-perspective data flow tracing (not pattern matching)

## Executive Summary

Traced complete attack chains from gateway → service → database across 5 services. Found and fixed **2 P0** and **1 P1** vulnerabilities. All involve cross-tenant data access via header/query parameter manipulation.

| ID | Severity | Title | Status |
|----|----------|-------|--------|
| BOLA-01 | P0 | SCIM injectTenant header overrides token tenant context | FIXED |
| BOLA-02 | P0 | /scim/v2/Me ParseUnverified auth bypass | FIXED |
| BOLA-03 | P1 | Audit 8 endpoints accept tenant_id from query param without JWT validation | FIXED |
| BOLA-04 | P2 | Device code flow accepts tenant_id from form value | Mitigated (multi-step) |
| BOLA-05 | P2 | Gateway X-User-ID not cleared when JWT invalid | FIXED |

## BOLA-01: SCIM injectTenant Header Override (P0)

### Root Cause
`injectTenant()` in `services/identity/internal/scim/handler.go` read `tenant_id` from `X-Tenant-ID` HTTP header, overwriting the tenant context already set by `scimTokenAuth` from the SCIM bearer token's database record.

### Attack Chain
1. Attacker has valid SCIM token for tenant A (e.g., `ggid_scim_abc123`)
2. Sends `GET /scim/v2/Users` with `X-Tenant-ID: <tenant_B_UUID>` header
3. `scimTokenAuth` validates token → sets context `tenantID = A`
4. SCIM handler's `injectTenant` OVERWRITES context → `tenantID = B` (from header)
5. All DB queries use tenant B → RLS returns tenant B's user data

### Fix
`injectTenant` now checks `ggidtenant.FromContext(r.Context())` first. If token-set context exists (`tenantID != uuid.Nil`), returns it directly. Only falls back to `X-Tenant-ID` header if no token context exists.

Same fix applied to `tenantFromRequest` in `scim/groups.go` and `injectTenant` in `server/http.go`.

### Tests
3 security tests in `bola_security_test.go`:
- `TestInjectTenant_RespectsTokenContext` — proves token tenant not overwritten
- `TestInjectTenant_FallbackToHeader` — proves header fallback works for non-token auth
- `TestInjectTenant_MissingBothTokenAndHeader` — proves fail-closed

### Files Changed
- `services/identity/internal/scim/handler.go`
- `services/identity/internal/scim/groups.go`
- `services/identity/internal/server/http.go`
- `services/identity/internal/scim/bola_security_test.go`

## BOLA-02: /scim/v2/Me ParseUnverified Auth Bypass (P0)

### Root Cause
Commit `b77fe6177` (P2-14 SCIM /Me) implemented `/scim/v2/Me` using `jwt.NewParser().ParseUnverified()` to extract `sub` and `tenant_id` claims. Combined with the gateway routing `/scim/v2/` as a public path (JWT optional), this created a complete authentication bypass.

### Attack Chain
1. Gateway: `/scim/v2/` in `publicPaths` → `JWTAuth(required=false)`
2. Attacker sends forged JWT with `{sub: <victim_user_id>, tenant_id: <any_tenant>}` — no valid signature needed
3. Gateway JWT verification fails → `required=false` → request passes through **without setting context**
4. Gateway reverse-proxies to identity service (forwards forged `Authorization` header)
5. Identity `scimTokenAuth`: JWT doesn't have `ggid_scim_` prefix → passes through
6. `handleMe`: `ParseUnverified` trusts attacker-controlled claims → `GetUser(ctx, victim_user_id)` returns victim's full SCIM profile

### Fix (Two Parts)
1. **Gateway Director**: Always overwrite `X-User-ID` header — set verified value from JWT when valid, `Header.Del` when no valid JWT. Prevents client-side `X-User-ID` spoofing.
2. **handleMe**: Replaced `extractTokenSub` (ParseUnverified) with `extractVerifiedUser` — reads gateway-verified `X-User-ID` and `X-Tenant-ID` headers. Removed `jwt` import entirely.

### Lesson
`ParseUnverified` is **never safe** in a multi-service reverse-proxy architecture. The gateway verifies JWT and sets context values internally, but context values don't survive HTTP proxying — only headers do. Downstream services must either verify JWT themselves or trust gateway-injected verified headers.

### Files Changed
- `services/gateway/internal/router/router.go` (2 Director occurrences)
- `services/identity/internal/scim/handler.go`

## BOLA-03: Audit Endpoints Query Param Tenant Bypass (P1)

### Root Cause
8 audit service endpoints (`handleEvents`, `handleExport`, `handleStats`, `handleStream`, `handleCorrelate`, `handleSearch`, `handleMetrics`, compliance report) accepted `tenant_id` from URL query parameters without validating against the gateway-authenticated tenant. An attacker with a valid JWT for tenant A could query `?tenant_id=<tenant_B>` and read tenant B's audit logs.

### Fix
Added `resolveValidatedTenant()` helper:
- If `X-Tenant-ID` header present (gateway-validated against JWT), it takes precedence
- If query param `tenant_id` also present, it **must match** the header → 403 on mismatch
- If only header present, uses header
- If only query param present, uses query param (backward compat for tamper-check auto-discovery)

### Files Changed
- `services/audit/internal/server/http.go` (helper + 8 endpoint rewrites)

## BOLA-04: Device Code Flow (P2 — Mitigated)

OAuth device code handler (`device_code_handler.go`) accepts `tenant_id` from both `X-Tenant-ID` header and form value. However, the subsequent device authorization and token exchange flow involves multi-step validation (client_id lookup, user consent, device code polling) that mitigates direct exploitation. Tracked for future hardening.

## BOLA-05: Gateway X-User-ID Header Spoofing (P2 — Fixed)

### Root Cause
Gateway's `ReverseProxy.Director` only set `X-User-ID` when JWT was valid, but did NOT clear client-supplied `X-User-ID` when JWT was absent/invalid. This allowed downstream services that trust `X-User-ID` (identity service audit events) to be spoofed.

### Fix
Director now calls `req.Header.Del("X-User-ID")` when no valid JWT is present.

## Verified Safe

| Component | Mechanism |
|-----------|-----------|
| OAuth client lookup | RLS enforced (`setTenantRLS` + `WHERE client_id`) |
| Password reset | Token bound to tenant at creation (`tenantID:userID` in Redis) |
| RLS enforcement | `FORCED` on users, sessions, mfa_devices, oauth_clients, audit_events |
| Gateway tenant mismatch check | JWT tenant_id vs X-Tenant-ID header, platform:admin scope exception |

## Remaining Observations

1. **P2-15 JWT aud not wired**: `ParseAccessTokenWithAudience` added but all callers still use `ParseAccessToken` (no aud check). god_fullstack is migrating 5 callers.
2. **P2-16 PKCE plain**: Discovery declares `plain` but doesn't enforce. Non-interop-breaking.
3. **Device flow tenant_id**: Form value accepted without JWT validation (mitigated by multi-step flow).
4. **OAuth server.go tenant header**: 12+ locations read `X-Tenant-ID` directly — mitigated by gateway validation + RLS, but pattern should be centralized.

## Changed Files for Deploy

1. `services/identity/internal/scim/handler.go` (BOLA-01 + BOLA-02)
2. `services/identity/internal/scim/groups.go` (BOLA-01)
3. `services/identity/internal/scim/bola_security_test.go` (BOLA-01 tests)
4. `services/identity/internal/server/http.go` (BOLA-01)
5. `services/audit/internal/server/http.go` (BOLA-03)
6. `services/gateway/internal/router/router.go` (BOLA-05)
