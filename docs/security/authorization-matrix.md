# Authorization Matrix

**Version**: v1.0-stable  
**KB**: KB-363  

---

## Authorization Layers

| Layer | Scope | Mechanism |
|-------|-------|-----------|
| Gateway (global) | All `/api/v1/admin/*` paths | `hasAdminScope()` — checks JWT for `admin` or `ggid:admin` scope |
| Gateway (per-request) | JWT claims → headers | `JWTClaimExtraction` injects `X-User-Role`, `X-Is-Admin`, `X-Scopes` |
| Policy service | `roles/assign` | `isAdminRequest()` — checks `X-User-Role`, `X-Is-Admin`, `X-Scopes` |
| Tenant isolation | All data queries | `X-Tenant-ID` header from JWT claims; PG RLS |

---

## Admin-Only Endpoints

### Gateway-Level Protection (`hasAdminScope`)

All paths matching `/api/v1/admin/*` require `admin` or `ggid:admin` in JWT scopes:

| Endpoint | Method | Access | Notes |
|----------|--------|--------|-------|
| `/api/v1/admin/routes` | GET | Admin | List gateway routes |
| `/api/v1/admin/routes` | POST | Admin | Add route |
| `/api/v1/admin/routes/{prefix}` | DELETE | Admin | Remove route |
| `/api/v1/admin/routes/{prefix}/toggle` | POST | Admin | Enable/disable route |
| `/api/v1/admin/stats` | GET | Admin | Gateway statistics |
| `/api/v1/admin/rate-limits` | GET/POST | Admin | Rate limit tiers |

### Policy-Level Protection (`isAdminRequest`)

| Endpoint | Method | Access | Notes |
|----------|--------|--------|-------|
| `/api/v1/roles/assign` | POST | Admin | Assign role to user |

### Auth Key Management

| Endpoint | Method | Access | Notes |
|----------|--------|--------|-------|
| `/api/v1/auth/key-rotation/rotate/{type}` | POST | Admin* | JWT required, no explicit admin check |
| `/api/v1/auth/key-rotation/history` | GET | Admin* | JWT required |
| `/api/v1/auth/key-rotation/keys` | GET | Admin* | JWT required |

*These should have explicit admin checks added in v1.1.

---

## Public Endpoints (No JWT Required)

| Endpoint | Method | Rationale |
|----------|--------|-----------|
| `/healthz`, `/healthz/*` | GET | Health check |
| `/.well-known/jwks.json` | GET | JWKS public keys |
| `/.well-known/openid-configuration` | GET | OIDC discovery |
| `/api/v1/auth/login` | POST | Authentication |
| `/api/v1/auth/register` | POST | Registration |
| `/api/v1/auth/refresh` | POST | Token refresh |
| `/api/v1/auth/password/forgot` | POST | Password reset request |
| `/api/v1/auth/password/reset` | POST | Password reset |
| `/api/v1/auth/social/{provider}` | GET | Social login start |
| `/oauth/authorize` | GET | OAuth2 authorize |
| `/oauth/token` | POST | OAuth2 token exchange |
| `/docs`, `/api-docs` | GET | API documentation |

---

## Authenticated Endpoints (JWT Required, No Admin)

All endpoints not listed above require a valid JWT. These include:

| Category | Endpoints | Data Isolation |
|----------|-----------|----------------|
| Users | CRUD, import, lock/unlock | Tenant-scoped (RLS) |
| Roles | List, create, delete | Tenant-scoped |
| Policies | CRUD, evaluate | Tenant-scoped |
| Organizations | CRUD, tree | Tenant-scoped |
| Audit | Query, export, stream | Tenant-scoped |
| Sessions | List, revoke, force-logout | User-scoped |
| MFA | Setup, verify, disable, status | User-scoped |
| OAuth Clients | CRUD | Tenant-scoped |
| Agents | Register, token, scopes | Tenant-scoped |
| Profile | View, edit, linked accounts | User-scoped |

---

## Authorization Enforcement Details

### `hasAdminScope()` (Gateway)

```go
func (gw *Gateway) hasAdminScope(r *http.Request) bool {
    claims := ExtractJWTClaims(r)
    for _, sc := range claims.Scopes {
        if sc == "admin" || sc == "ggid:admin" {
            return true
        }
    }
    return false // strict — wildcard "*" does NOT pass
}
```

### `isAdminRequest()` (Policy)

Checks three sources in order:
1. `X-User-Role` header = `admin` or `superadmin`
2. `X-Is-Admin` header = `true`
3. `X-Scopes` header contains `admin`, `superadmin`, `roles:write`, or `*`

### JWT Claim Injection (Gateway)

`JWTClaimExtraction` middleware derives admin headers from JWT scopes:
- `admin` or `superadmin` or `roles:write` or `*` scope → sets `X-User-Role` + `X-Is-Admin`

---

## Known Gaps (v1.1 Backlog)

1. **Key rotation endpoints** — no explicit admin check (JWT-only)
2. **Gateway rate limit admin** — relies on path prefix only, should also check method
3. **OAuth client lifecycle** — no admin check on pause/resume/delete

---

## Test Results

| Test | Expected | Status |
|------|----------|--------|
| Admin token → `/api/v1/admin/stats` | 200 | PASS |
| Regular token → `/api/v1/admin/stats` | 403 | PASS |
| Admin token → `/api/v1/roles/assign` | 200 | PASS |
| Regular token → `/api/v1/roles/assign` | 403 | PASS |
| No token → any protected endpoint | 401 | PASS |
| Wildcard `*` scope → admin endpoint | 403 | PASS (strict) |
