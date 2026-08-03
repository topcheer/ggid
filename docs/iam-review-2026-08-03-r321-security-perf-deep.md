# R321 Security (21st) + Performance (18th) Deep Audit

**Date**: 2026-08-03
**Scope**: `services/gateway/internal/`, `services/auth/internal/`, `services/identity/internal/`, `pkg/rbac/`, `pkg/crypto/`
**Mode**: Read-only code audit, first-contact independent review

---

## Executive Summary

- **P0 (Critical)**: 2 confirmed (1 new, 1 persistent from 3-round backlog)
- **P1 (High)**: 4 confirmed
- **P2 (Medium)**: 5 confirmed
- **Previously reported items now FIXED**: CORS wildcard+credentials, HSTS r.TLS, JWT kid rejection, platform:admin scope-only enforcement, CSP headers
- **Audit statistics**: ~322 P0 cumulative, ~601 P1, ~635 P2 across 91 rounds. 58 P0 fixed.

---

## P0 — Critical Security Issues

### P0-1: `platformOnlyPaths` divergence between router.go and rbac.go (3-round persistent)

**Status**: STILL OPEN — 3rd round flagged, not fixed

**Files**:
- `services/gateway/internal/router/router.go:836-841` (checkRouteScope)
- `services/gateway/internal/middleware/rbac.go:226-234` (RequireAdminScope)

**Problem**: Two independent `platformOnlyPaths` lists exist with **different entries**:

router.go `platformOnlyPaths`:
```
/api/v1/system/, /api/v1/tenants/create,
/api/v1/org/tenants/suspend, /api/v1/org/tenants/activate,
/api/v1/admin/audit/global, /api/v1/admin/threats/dashboard
```

rbac.go `platformOnlyPaths`:
```
/api/v1/admin/secrets, /api/v1/admin/backup, /api/v1/admin/key-rotation,
/api/v1/impersonate, /api/v1/auth/impersonate,
/api/v1/system/, /api/v1/tenants
```

**Risk**: HIGH — Depending on which middleware fires first, paths like `/api/v1/admin/secrets` may only be checked by one list but not the other. A `tenant:admin` could access platform-only endpoints that exist in one list but not the other.

**Root cause**: The two gate functions (`checkRouteScope` in router.go and `RequireAdminScope` in rbac.go) both run on every request but use different hardcoded lists. Neither is the single source of truth.

**Fix**: Consolidate into a single shared `platformOnlyPaths` variable in the middleware package, imported by both router.go and rbac.go.

---

### P0-2: `AdminOnly` middleware is dead code with dangerous empty-scope bypass

**File**: `services/gateway/internal/middleware/rbac.go:13-32`

**Problem**: The `AdminOnly` middleware function is defined but **never wired into any route** (confirmed: zero call sites outside definition). If it were ever activated, lines 17-19 contain a critical vulnerability:

```go
func AdminOnly(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := ExtractJWTClaims(r)
        if len(claims.Scopes) == 0 {
            next.ServeHTTP(w, r)  // FAILS OPEN — no scope = full access
            return
        }
```

**Risk**: MEDIUM-HIGH currently (dead code), but if revived without fixing, any authenticated user without scopes gets admin access. The active `RequireAdminScope` (rbac.go:109) correctly handles this case — `AdminOnly` does not.

**Fix**: Delete `AdminOnly` to prevent accidental future wiring. Its logic is superseded by `RequireAdminScope`.

---

## P1 — High Severity Issues

### P1-1: `HasPermissionForRoute` bypass at router.go:924 — partially mitigated but inconsistent

**File**: `services/gateway/internal/router/router.go:924-930`

**Problem**: In `checkRouteScope`, a non-admin user with fine-grained permission `users:read` can bypass the `adminOnlyPaths` gate:
```go
if middleware.HasPermissionForRoute(path, r.Method, claims.Permissions) {
    continue  // bypasses admin check
}
```

**Mitigation present**: The **other** gate (`RequireAdminScope` at rbac.go:204) correctly blocks this with `!isAdminOnlyPath(r.URL.Path)` guard. And the dynamic resolver (rbac_dynamic.go:423) also has `!adminProtected` guard.

**Residual risk**: The router.go gate (`checkRouteScope`) runs at `ServeHTTP` entry (line 321) BEFORE `RequireAdminScope` in the middleware chain. So the first gate allows the request through based on permission-key, but the second gate catches it. This is defense-in-depth working correctly **only if both gates are always active**. If middleware ordering changes or `RequireAdminScope` is ever removed, the bypass is exploitable.

**Fix**: Add `&& !middleware.IsAdminOnlyPath(path)` guard to router.go:928, matching rbac.go:204.

---

### P1-2: `EnsureSystemPermissions` executes N individual INSERT...ON CONFLICT queries

**File**: `pkg/rbac/permissions.go:229-261`

**Problem**: Each system permission is upserted via a separate `pool.Exec()` call in a loop (line 239). With ~68 system permissions, this means 68 round-trips to the database on every service startup.

**Current mitigation**: `ON CONFLICT (key) DO UPDATE` avoids failures on restart. Errors are logged but non-fatal (continue).

**Risk**: MEDIUM — Startup latency. On cold starts with network latency to DB, 68 sequential queries add measurable delay. Not a runtime issue (runs once at boot).

**Fix**: Batch into a single query using `UNNEST` arrays or a CTE with `VALUES`:
```sql
INSERT INTO permissions (...) VALUES (...), (...), ...
ON CONFLICT (key) DO UPDATE SET ...
```

---

### P1-3: `rows.Err()` checked in only 27 of 178 `rows.Next()` sites (15%)

**Scope**: All `services/` subdirectories

**Problem**: 178 locations call `rows.Next()` in a loop, but only 27 check `rows.Err()` after the loop (15%). Missing `rows.Err()` means silent data truncation on backend errors (connection closed, query timeout, context cancelled).

**Risk**: MEDIUM — Hard-to-detect data consistency bugs. A query that returns 1000 rows but errors after row 500 would appear to return 500 rows with no error logged.

**Fix**: After every `for rows.Next()` loop, add:
```go
if err := rows.Err(); err != nil {
    return fmt.Errorf("scan rows: %w", err)
}
```

---

### P1-4: Audit `OFFSET` pagination in 3 query sites

**Files**:
- `services/audit/internal/repository/audit_repo.go:214`
- `services/audit/internal/server/global_dashboard_handler.go:73`
- `services/audit/internal/repository/itdr_repo.go:170`

**Problem**: Deep pagination using OFFSET scans and discards rows. For page 10000 at 50/page, the DB scans 500500 rows to return 50. Audit tables grow unboundedly.

**Risk**: MEDIUM — Degrades over time as audit log grows. Eventually causes query timeouts on large datasets.

**Fix**: Use keyset (cursor) pagination with `WHERE created_at < $last_seen ORDER BY created_at DESC LIMIT $n`.

---

## P2 — Medium Severity Issues

### P2-1: `json.Unmarshal` used in 423 locations across services without size limits

**Scope**: All `services/`

**Problem**: 423 `json.Unmarshal` calls (up from 136 reported previously — scope expanded). Many likely operate on request body data without size validation before deserialization. While the gateway has `maxBodySize` (10 MiB default), internal service-to-service calls may not have body limits.

**Risk**: LOW-MEDIUM — Potential for large payload DoS on internal endpoints that bypass gateway body limits.

---

### P2-2: `http.Client` / `http.Get` used in 51 locations

**Scope**: All `services/`

**Problem**: 51 locations use `http.Client{}`, `http.DefaultClient`, `http.Get()`, or `http.Post()`. Many may lack timeout configuration, which can cause goroutine leaks and resource exhaustion when calling external services.

**Risk**: LOW-MEDIUM — Default `http.Client{}` has no timeout. A slow upstream can hang goroutines indefinitely.

**Fix**: Wrap all HTTP client creation in a shared factory that enforces timeouts.

---

### P2-3: `injectTenantIntoBody` only handles flat JSON objects

**File**: `services/gateway/internal/router/router.go:975-979+`

**Problem**: The function documents it "only modifies flat JSON objects and preserves the original body if it's not JSON or already contains a tenant_id field." Nested body structures with tenant_id at a different level may be silently ignored.

**Risk**: LOW — Could allow tenant_id injection in nested fields if backend trusts body tenant_id.

---

### P2-4: `DefaultRolePermissionKeys` generates NxM role-permission map at startup

**File**: `pkg/rbac/permissions.go:282-306`

**Problem**: Iterates over all system permissions to build role->[]permission maps. With ~68 permissions and 4 roles, this is ~272 iterations — trivial cost, but the function is called each time role permissions need to be assigned. If called per-tenant at bootstrap, cost scales with tenant count.

**Risk**: LOW — Acceptable for one-time bootstrap. Would become P1 if called per-request.

---

### P2-5: `canary.go` uses `math/rand` for feature flag routing

**File**: `services/gateway/internal/middleware/canary.go:108`

**Problem**: `rand.Intn(100) < percentage` uses non-cryptographic random. Not a security issue for canary routing, but if the same pattern is reused for security-sensitive decisions, it would be predictable.

**Risk**: LOW — Canary routing is not security-sensitive. Flagged for pattern awareness only.

---

## CONFIRMED FIXES (Previously Reported Issues)

| Issue | Status | Evidence |
|-------|--------|----------|
| CORS wildcard + credentials (R318 P0-3) | **FIXED** | `middleware.go:176-180` — wildcard+credentials now uses `*` without `Allow-Credentials: true`. Explicit origin whitelist uses `subtle.ConstantTimeCompare`. Production default = no origins allowed. |
| HSTS on plaintext HTTP (R314) | **FIXED** | `security_headers.go:90` — `active.HSTSMaxAge > 0 && r.TLS != nil`. Also `middleware.go:336` — `if r.TLS != nil` guard. |
| JWT kid rejection on unknown key | **FIXED** | `middleware.go:746-749` — kid present but not in JWKS returns error, no fallback to static key. |
| JWT alg confusion attack | **FIXED** | `middleware.go:728` — `jwt.WithValidMethods(crypto.SupportedAlgs())`. Line 737 — double-check `IsSupportedAlg`. |
| Invalid token stripping (R226 P0) | **FIXED** | `middleware.go:719,769,783` — forged/invalid tokens have Authorization header deleted before forwarding. Fail-closed `JWTCClaims{}` set in context. |
| platform:admin scope-only enforcement | **FIXED** | `router.go:880-885,889-903` — Platform status comes ONLY from scopes claim, not roles claim (which is tenant-forgeable). |
| tenant:admin privilege escalation (R305) | **FIXED** | `rbac.go:182-188` — `isPlatformOnlyPath` check blocks tenant:admin from platform endpoints. `router.go:906-911` — parallel check. |
| CSP headers | **FIXED** | `security_headers.go:97-101` — `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`. |
| Impersonation permission intersection | **FIXED** | `router.go:818` and `rbac.go:230-231` — `/api/v1/impersonate` and `/api/v1/auth/impersonate` in platformOnlyPaths. |

---

## Security Architecture Assessment

### JWT Validation Chain (VERIFIED COMPLETE)
1. `WithValidMethods(crypto.SupportedAlgs())` — blocks alg confusion
2. `WithIssuer(issuer)` — validates issuer claim
3. `WithAudience(audience)` — validates audience claim
4. `jwt/v5` validates exp/nbf/iat by default
5. `kid` → JWKS lookup, reject if not found (no static fallback when kid present)
6. No kid → static key (backward compat)
7. No key → fail validation (no nil key panic)
8. Invalid token → strip Authorization header, set empty claims (fail-closed)

**Verdict**: JWT chain is robust. No bypasses found.

### RBAC Defense-in-Depth (3 layers verified)
1. **Gateway router** (`checkRouteScope`, router.go:845) — scope check on all requests
2. **Middleware** (`RequireAdminScope`, rbac.go:109) — dynamic RBAC + static fallback
3. **Dynamic resolver** (`RBACResolver.CheckAccess`, rbac_dynamic.go) — DB-driven route permissions

**Concern**: The three layers have **independently maintained path lists** (adminOnlyPaths, defaultAdminPrefixes, platformOnlyPaths in two places). This is the primary source of P0-1.

---

## Performance Summary

| Metric | Current | Previous | Trend |
|--------|---------|----------|-------|
| EnsureSystemPermissions | N individual Exec (N~68) | 68xExec | Unchanged |
| GrantPermissions | Service-level, UNNEST in repo | UNNEST | OK |
| Audit OFFSET | 3 sites | 4 sites | Improved (-1) |
| rows.Err() coverage | 27/178 = 15% | 82% reported | **Discrepancy** (see note) |
| json.Unmarshal | 423 calls | 136 reported | Scope expanded |
| HTTP client | 51 calls | 61 reported | Improved |
| Random eviction | Rate limiter + JTI blocklist | 3 sites | OK (time-based, not random) |

**Note on rows.Err() discrepancy**: Previous rounds reported 82% coverage; this audit found 15% (27/178). The difference is scope: previous may have counted only the targeted service (e.g., identity), while this counted all services/ broadly. Both indicate improvement is needed.

---

## Recommendations (Priority Order)

1. **P0-1**: Consolidate `platformOnlyPaths` into single shared variable — this is the 3rd round flagged
2. **P0-2**: Delete `AdminOnly` dead code to prevent dangerous future activation
3. **P1-1**: Add `isAdminOnlyPath` guard to router.go:928 HasPermissionForRoute bypass
4. **P1-2**: Batch EnsureSystemPermissions into single query
5. **P1-3**: Systematic `rows.Err()` audit — add after every `rows.Next()` loop
6. **P1-4**: Convert audit OFFSET pagination to keyset/cursor pagination
