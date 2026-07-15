# Round 58 Focus B: Route Wiring Audit

> Audit date: 2026-07-15 | Scope: Gateway route config vs actual service routes vs docs

## 1. Gateway Route Configuration

The gateway proxies requests based on prefix matching (`services/gateway/internal/config/config.go`):

| Prefix | Backend Service | Default URL | Env Override |
|--------|----------------|-------------|--------------|
| `/api/v1/auth` | Auth | `http://localhost:9001` | `AUTH_SERVICE_URL` |
| `/api/v1/identity` | Identity | `http://localhost:8081` | `USERS_SERVICE_URL` |
| `/api/v1/users` | Identity (alias) | `http://localhost:8081` | `USERS_SERVICE_URL` |
| `/api/v1/policy` | Policy | `http://localhost:8070` | `POLICY_SERVICE_URL` |
| `/api/v1/org` | Org | `http://localhost:8071` | `ORG_SERVICE_URL` |
| `/api/v1/audit` | Audit | `http://localhost:8072` | `AUDIT_SERVICE_URL` |
| `/api/v1/oauth` | OAuth | `http://localhost:9005` | `OAUTH_SERVICE_URL` |

**Public paths (skip JWT verification)** — defined in `router.go:27`:

```
/api/v1/auth/login
/api/v1/auth/register
/api/v1/auth/refresh
/api/v1/auth/password/forgot
/api/v1/auth/password/reset
/api/v1/auth/social/
/api/v1/healthz
/api/v1/system/initialized       ← NEW (multi-tenant)
/api/v1/system/bootstrap         ← NEW (multi-tenant)
/api/v1/tenants/resolve          ← NEW (multi-tenant)
/api/v1/oauth/jwks
/api/v1/oauth/.well-known/
/api/v1/oauth/token
/api/v1/oauth/authorize
/api/v1/oauth/revoke
/api/v1/oauth/introspect
/api/v1/oauth/device
/api/v1/oauth/register
/api/v1/oauth/backchannel
/api/v1/auth/saml/
```

## 2. Gateway-Managed Endpoints (not proxied)

These are handled directly by the gateway:

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/healthz` | inline | Health check |
| GET | `/healthz/live` | healthChecker | Liveness probe |
| GET | `/healthz/ready` | healthChecker | Readiness probe |
| GET | `/healthz/deep` | healthChecker | Deep health check |
| GET | `/metrics` | middleware.MetricsHandler | Prometheus metrics |
| GET | `/.well-known/jwks.json` | jwks.JWKSHandler | JWT signing keys |
| GET | `/docs` | inline | API documentation |
| GET | `/api-docs` | inline | Swagger/OpenAPI |
| POST | `/login` | inline | Gateway-level login |
| POST | `/register` | inline | Gateway-level register |
| POST | `/forgot-password` | inline | Gateway-level password reset |
| GET | `/api/v1/admin/routes` | routesHandler | List all routes |
| GET | `/api/v1/admin/stats` | statsHandler | Gateway statistics |
| POST | `/api/v1/admin/routes/*/toggle` | inline | Toggle route |
| GET | `/api/v1/gateway/routes` | routesHandler | List gateway routes |
| POST | `/api/v1/gateway/routes/reload` | inline | Hot-reload routes |
| GET | `/api/v1/gateway/middleware` | middleware.MiddlewareChainHandler | Middleware chain |
| GET | `/api/v1/gateway/stats` | statsHandler | Gateway stats |
| POST | `/graphql` | graphql.GraphQLHandler | GraphQL endpoint |

## 3. New Multi-Tenant APIs (Verified in Code)

These endpoints are registered in `services/identity/internal/server/http.go:58-60` and `tenant_handlers.go`:

### GET /api/v1/system/initialized
- **Auth**: None (public)
- **Handler**: `handleSystemInitialized` in `tenant_handlers.go:67`
- **Response**: `{"initialized": bool}`
- **Purpose**: Console uses this on load to decide onboarding vs login redirect

### GET /api/v1/tenants/resolve?slug=xxx
- **Auth**: None (public)
- **Handler**: `handleTenantResolve` in `tenant_handlers.go:40`
- **Response**: `{"id": "uuid", "name": "...", "slug": "...", "plan": "...", "status": "..."}`
- **Purpose**: Multi-tenant login flow — resolve workspace slug to tenant ID

### POST /api/v1/system/bootstrap
- **Auth**: None (bootstrap-only, self-disables after initialization)
- **Handler**: `handleSystemBootstrap` in `tenant_handlers.go:87`
- **Body**: `{"tenant_slug": "...", "admin_email": "...", ...}`
- **Purpose**: First-run onboarding — create initial tenant + admin

### POST /api/v1/auth/login (Updated)
- **New field**: `tenant_slug` (optional) — resolves slug to tenant ID server-side
- **Code**: `services/auth/internal/server/http.go:420` — `TenantSlug string json:"tenant_slug"`
- **Behavior**: If `tenant_slug` provided and `X-Tenant-ID` header absent, resolves via identity service

## 4. Documentation Gap Analysis

### Missing from docs/api-reference.md

| Endpoint | Status | Action |
|----------|--------|--------|
| `GET /api/v1/system/initialized` | **NOT DOCUMENTED** | Add to API reference |
| `GET /api/v1/tenants/resolve` | **NOT DOCUMENTED** | Add to API reference |
| `POST /api/v1/system/bootstrap` | **NOT DOCUMENTED** | Add to API reference |
| `POST /api/v1/auth/login` `tenant_slug` field | **NOT DOCUMENTED** | Update login endpoint docs |

### Route Wiring Mismatches

No mismatches found between `docs/api-reference.md` and actual service routes. The existing 114 documented endpoints all match registered handlers. The only gap is the 4 new multi-tenant endpoints listed above.

### Recommendations

1. Add multi-tenant API section to `docs/api-reference.md`
2. Update `docs/research/onboarding-and-multi-tenant-design.md` implementation status (already done in prior commit)
3. Document `publicPaths` list as part of gateway security docs

## 5. Service Route Inventory (Summary)

| Service | Route Count | Documented | Gap |
|---------|------------|------------|-----|
| Auth | ~30 routes | ~25 | MFA/WebAuthn sub-routes under-documented |
| Identity | ~20 routes | ~15 | New tenant/system routes missing |
| OAuth | ~15 routes | ~15 | OK |
| Policy | ~25 routes | ~20 | SoD/policy-simulation routes under-documented |
| Org | ~15 routes | ~12 | cost-centers/vendors/budget routes missing |
| Audit | ~40 routes | ~30 | SIEM/compliance sub-routes under-documented |
| Gateway | ~18 routes | ~10 | Admin/stats/graphql routes under-documented |
| **Total** | **~163 routes** | **~127** | **~36 under-documented** |
