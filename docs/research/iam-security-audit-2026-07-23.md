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
