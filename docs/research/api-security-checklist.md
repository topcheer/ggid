# API Security Checklist for IAM Systems

Based on OWASP API Security Top 10 (2023) and a source-code audit of GGID's
gateway, auth, and shared packages. Each section maps to GGID's actual
implementation with status ratings: **Pass**, **Partial**, **Gap**.

---

## 1. Overview

IAM APIs are high-value attack targets — a single weakness can expose user
data, enable privilege escalation, or cause cross-tenant leakage. This
checklist assesses GGID against the **OWASP API Security Top 10 (2023)**,
supplemented by deep-dives into OAuth scopes, rate limiting, input validation,
CORS/JWT/CSRF, secrets management, and audit logging. All findings reference
specific files in the GGID codebase with Go code snippets.

---

## 2. OWASP API Top 10 (2023) Audit

### API1: Broken Object Level Authorization (BOLA)

**What**: Attackers access objects they don't own by manipulating IDs.

**GGID**: **Partial**. Tenant isolation via PostgreSQL RLS + gateway tenant
resolution (`tenant_enhanced.go`). The `EnhancedTenantResolver` resolves
tenant from header, JWT claim, or subdomain. However, per-object ownership
checks (e.g., "can user A read user B?") are not centrally enforced.

**Fix**: Add ownership-verification middleware:
```go
func RequireOwner(paramName string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID, ok := UserIDFromRequest(r)
            if !ok { writeUnauthorized(w, "no user"); return }
            if chi.URLParam(r, paramName) != userID.String() && !hasScope(r, "admin:*") {
                writeForbidden(w, "access denied"); return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### API2: Broken Authentication

**GGID**: **Pass**. JWT validation (`middleware.go`) enforces RS256-only
(`jwt.WithValidMethods`), issuer, audience, and expiry (jwt/v5 default). JWKS
key rotation with background refresh. Argon2id password hashing (64MB, 3
iterations). MFA TOTP. Login rate-limited to 5/min, register to 3/min.

**Fix**: Add refresh token rotation and replay detection.

### API3: Broken Object Property Level Authorization

**What**: Mass assignment — clients set fields like `is_admin` in updates.

**GGID**: **Gap**. No explicit field-level allowlists. Request structs accept
all JSON-tagged fields.

**Fix**: Use separate input DTOs per endpoint:
```go
type UpdateUserInput struct {
    DisplayName string `json:"display_name"`
    Email       string `json:"email"`
    // NO Role, TenantID, IsActive — admin-only fields
}
```

### API4: Unrestricted Resource Consumption

**GGID**: **Pass**. Multiple rate-limiting layers: per-endpoint
(`ratelimit.go`: login 5/min, register 3/min, API 100/min), tiered
(`tier_ratelimit.go`: free 100/min, pro 1000/min), sliding window Redis-backed
(`sliding_ratelimit.go`), body size limiting (`bodysize.go`), and behavioral
bot detection (`botdetect.go`).

**Fix**: Enforce max page size on list endpoints.

### API5: Broken Function Level Authorization

**GGID**: **Partial**. RBAC + ABAC policy engine exists in `services/policy`,
but the gateway verifies JWT without enforcing role checks centrally.

**Fix**: Add gateway-level role enforcement:
```go
func RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userRole, _ := r.Context().Value(RoleKey).(string)
            for _, role := range roles {
                if userRole == role { next.ServeHTTP(w, r); return }
            }
            writeForbidden(w, "insufficient privileges")
        })
    }
}
```

### API6: Server-Side Request Forgery (SSRF)

**GGID**: **Gap**. SAML metadata, webhook delivery, and social avatar fetching
accept external URLs without allowlisting. No scheme or internal-IP validation.

**Fix**:
```go
func isSafeURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil { return err }
    if u.Scheme != "https" { return errors.New("https only") }
    if ip := net.ParseIP(u.Hostname()); ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
        return errors.New("internal address blocked")
    }
    return nil
}
```

### API7: Security Misconfiguration

**GGID**: **Partial**. Security headers present (HSTS, X-Content-Type-Options,
X-Frame-Options, Referrer-Policy). Per-tenant CORS with constant-time origin
comparison. But `DefaultCORSConfig()` uses `["*"]` — insecure for production.

**Fix**: Require explicit origins in production; add CSP for admin console.

### API8: Lack of Protection from Automated Threats

**GGID**: **Partial**. Bot detection blocks attack tools (sqlmap, nikto, nmap)
and tags legitimate crawlers. Behavioral detection rate-limits high-volume IPs.
No CAPTCHA or progressive challenge for repeated failures.

**Fix**: Add CAPTCHA after 3 failed logins. Rate-limit registration by domain.

### API9: Improper Inventory Management

**GGID**: **Partial**. API versioning middleware (`apiversion.go`) and OpenAPI
aggregator exist. No automated endpoint inventory or stale-API detection.

**Fix**: Generate API inventory from OpenAPI specs at build time.

### API10: Unsafe Consumption of APIs

**GGID**: **Gap**. Social login connectors consume third-party APIs without
response schema validation. No certificate pinning.

**Fix**: Validate third-party responses against expected schemas. Add circuit
breaker for outbound calls.

---

## 3. OAuth Scope Design

**Principle**: Least privilege — each token carries only needed scopes.

### Scope Taxonomy

| Category | Scope | Description |
|----------|-------|-------------|
| OIDC | `openid` | Enable OIDC flow, issue ID token |
| OIDC | `profile` | User profile claims |
| OIDC | `email` | Email claim |
| Read | `read:user` | List/view user profiles |
| Read | `read:role` | List/view roles |
| Write | `write:user` | Create/update users |
| Admin | `admin:*` | Full administrative access |
| Audit | `read:audit` | Query audit events |

### Scope Validation Middleware

```go
func RequireScopes(required ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            scopes, _ := r.Context().Value(ScopeKey).([]string)
            granted := make(map[string]bool, len(scopes))
            for _, s := range scopes { granted[s] = true }
            if granted["admin:*"] { next.ServeHTTP(w, r); return }
            for _, req := range required {
                if !granted[req] {
                    writeForbidden(w, "missing scope: "+req); return
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### GGID Status

OAuth service stores scopes on `OAuthClient.Scopes` and `AccessToken.Scope`
(`oauth/internal/domain/models.go`). Supported: `openid`, `profile`, `email`,
`read`, `write`. Token introspection returns `scope` claim. **Gap**: No
middleware enforces scopes at the gateway — they are stored but not checked.

**Fix**: Extract `scope` from JWT in `JWTAuth`, store in context, add
`RequireScopes` per route group.

---

## 4. Rate Limiting Strategy

GGID implements four complementary strategies:

| Level | Limit | Implementation |
|-------|-------|----------------|
| Login (anonymous) | 5 req/min | `ratelimit.go` |
| Register (anonymous) | 3 req/min | `ratelimit.go` |
| Free tier | 100 req/min | `tier_ratelimit.go` |
| Pro tier | 1000 req/min | `tier_ratelimit.go` |
| Enterprise | Unlimited | `tier_ratelimit.go` |

**Multi-dimensional**: Keys composed as `path:identifier` enabling per-IP,
per-user, per-tenant, or per-client limiting.

**Redis sliding window** (`sliding_ratelimit.go`): Implements `RateLimitStore`
interface using sorted sets for atomic count-and-add across distributed
instances. Falls back to in-memory for single-node deployments.

**Status**: In-memory limiter is default. Redis-backed, token bucket
(`token_bucket.go`), and tenant-specific (`tenant_ratelimit.go`) limiters are
available but must be explicitly configured.

**Fix**: Wire Redis as production default. Add per-client-id limiting.

---

## 5. Input Validation & Injection Prevention

### SQL Injection — Pass

All database access uses pgx parameterized queries (`$1`, `$2`). A grep for
`fmt.Sprintf.*SELECT|INSERT|UPDATE|DELETE` found 15 matches — all interpolate
**compile-time column-name constants** (e.g., `userColumns`), never user input:

```go
// Safe: column names are constants, user values use $1
query := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, userColumns)
```

### Password Validation — Pass

`PasswordPolicy` enforces configurable complexity (min length, upper/lower/
digit/special, blacklist, history). `StrengthScore` rates passwords 0-4.

### Remaining Gaps

| Threat | Status | Action |
|--------|--------|--------|
| XSS | Gap | Add Content-Security-Policy for console |
| Command injection | Pass | No `os/exec` with user input |
| Path traversal | Pass | No file paths from user input |
| Field length limits | Partial | Add `maxLen` validation struct tags |

---

## 6. CORS, JWT, CSRF

### CORS

| Aspect | Implementation | Status |
|--------|----------------|--------|
| Origin comparison | `subtle.ConstantTimeCompare` | Pass |
| Per-tenant origins | `TenantCORSStore` | Pass |
| Wildcard default | `["*"]` in `DefaultCORSConfig` | **Risk** |
| Credentials | Echoed origin when enabled | Pass |
| OPTIONS preflight | 204 NoContent | Pass |

### JWT

| Check | Status |
|-------|--------|
| RS256-only | Pass — `WithValidMethods` |
| Expiry/nbf/iat | Pass — jwt/v5 default |
| Issuer + audience | Pass — `WithIssuer`, `WithAudience` |
| JWKS key rotation | Pass — background refresh |
| `alg=none` rejected | Pass — RS256-only policy |
| Clock skew leeway | **Gap** — add `jwt.WithLeeway(30s)` |

### CSRF

Double-submit cookie pattern (`CSRFProtect`): safe methods set token cookie,
unsafe methods validate `X-CSRF-Token` header with constant-time comparison.
Cookie: `SameSite=Lax`, `Secure=true`. GGID is API-first (JWT in Authorization
header), so CSRF risk is inherently low — middleware is for the admin console.

---

## 7. Secrets Management

| Practice | Status | Details |
|----------|--------|---------|
| No hardcoded secrets | Pass | All via env vars / `keys.env` |
| `keys.env` pattern | Pass | Secrets in `keys.env`, not YAML |
| RSA signing keys | Pass | Loaded from file via env var |
| DB credentials | Pass | `DATABASE_URL` / `DB_HOST` env vars |
| API keys (social) | Partial | Not rotated |
| Key rotation policy | Gap | No automated rotation |
| Vault integration | Gap | No HashiCorp Vault / AWS SM |

**Fix**: Implement signing key rotation via JWKS `kid` versioning. The JWKS
endpoint already supports multiple keys — add a rotation workflow that
introduces a new key, waits for propagation, then retires the old one.

---

## 8. Audit Logging Coverage

GGID publishes audit events via NATS JetStream (`pkg/audit/publisher.go`):
structured JSON with `timestamp`, `user_id`, `tenant_id`, `action`,
`resource`, `result`, `ip_address`, `user_agent`.

| Event Type | Logged | Source |
|------------|--------|--------|
| Login (success/fail) | Yes | Auth service |
| Logout | Partial | Session invalidation only |
| User CRUD | Yes | Identity service |
| Role CRUD | Yes | Policy service |
| Org CRUD | Yes | Org service |
| MFA enable/disable | Partial | Auth service |
| Policy changes | **Gap** | No dedicated audit hook |
| Token revocation | **Gap** | Not audited |
| Admin actions | **Gap** | No elevated-action audit |

**Fix**: Add audit hooks for policy changes, token revocation, and admin
actions. Implement 90-day hot retention + 1-year cold archive to S3/GCS.

---

## 9. GGID Security Scorecard

| Area | OWASP | Status | Score | Action |
|------|-------|--------|-------|--------|
| Object-level authz | API1 | Partial | 6/10 | Per-object owner checks |
| Authentication | API2 | Pass | 9/10 | Refresh token rotation |
| Property authz | API3 | Gap | 3/10 | Per-endpoint input DTOs |
| Resource limits | API4 | Pass | 9/10 | Max page size |
| Function authz | API5 | Partial | 5/10 | Centralize role checks |
| SSRF | API6 | Gap | 2/10 | Validate outbound URLs |
| Misconfiguration | API7 | Partial | 6/10 | Remove wildcard CORS |
| Automated threats | API8 | Partial | 6/10 | CAPTCHA after failures |
| API inventory | API9 | Partial | 5/10 | Auto-generate inventory |
| 3rd-party APIs | API10 | Gap | 3/10 | Validate responses |
| OAuth scopes | — | Partial | 4/10 | Enforce at gateway |
| Rate limiting | — | Pass | 9/10 | Wire Redis as default |
| Input validation | — | Pass | 8/10 | Field length limits |
| CORS | — | Pass | 7/10 | Remove wildcard default |
| JWT | — | Pass | 9/10 | Add clock skew leeway |
| CSRF | — | Pass | 8/10 | N/A for API-only |
| Secrets | — | Pass | 7/10 | Key rotation workflow |
| Audit logging | — | Partial | 7/10 | Cover policy/admin events |

### Overall Posture: 7/10 — Strong foundation, targeted gaps

### Top 5 Gaps to Close

1. **Property-level authz (API3)** — Adopt per-endpoint input DTOs to prevent
   mass assignment. Highest risk, easiest fix.
2. **Function-level authz (API5)** — Centralize RBAC in gateway middleware
   rather than relying on each service.
3. **SSRF protection (API6)** — Validate outbound URLs against scheme and IP
   allowlists (webhooks, SAML metadata, social avatars).
4. **OAuth scope enforcement** — Wire existing scope infrastructure through
   gateway middleware for per-endpoint scope requirements.
5. **CORS default hardening (API7)** — Change `DefaultCORSConfig` from `*` to
   explicit origins; add CSP for the admin console.
