# SDK Ecosystem Gap Analysis and Priority Roadmap

> **Document type**: Competitive Analysis & Strategy
> **Date**: July 2025
> **Scope**: SDK coverage across GGID vs Auth0, Keycloak, Ory, Casdoor

---

## Executive Summary

GGID currently ships 4 SDKs (Go, Node.js/TypeScript, Python, Java) with varying
levels of maturity. While the Go and Node.js SDKs are production-quality with
JWT verification, RBAC middleware, and management APIs, the Python SDK lacks
role assignment and token refresh, and the Java SDK has no source code (only
a README and pom.xml). Competitors offer 6-12+ SDKs with broader language
coverage. This document analyzes the gap, ranks priorities, and proposes a
concrete roadmap to close the SDK ecosystem deficit within 2-3 quarters.

---

## 1. Competitor SDK Inventory

### Auth0 (12+ SDKs) — Industry Leader

Auth0 maintains the most comprehensive SDK ecosystem in the IAM space:

| SDK | Type | Key Features |
|-----|------|-------------|
| **auth0.js** | JavaScript (SPA) | WebAuth, cross-origin auth, popup/redirect |
| **@auth0/auth0-react** | React | `useAuth` hook, `withAuthenticationRequired` |
| **@auth0/auth0-angular** | Angular | HTTP interceptor, auth guard |
| **@auth0/auth0-vue** | Vue 3 | Composition API plugin |
| **@auth0/nextjs-auth0** | Next.js | SSR/SSG support, API routes, middleware |
| **node-auth0** | Node.js (server) | Management API, full CRUD |
| **go-auth0** | Go | Management API client |
| **auth0-python** | Python | Management API, async support |
| **Auth0.NET** | .NET | Management API, OWIN middleware |
| **omniauth-auth0** | Ruby | OmniAuth strategy |
| **auth0-swift** | iOS/Swift | Universal login, credentials |
| **auth0-android** | Android | WebAuthProvider, credentials manager |
| **auth0-flutter** | Flutter | Cross-platform auth |

**Key differentiator**: Auth0 separates SDK concerns into two tiers —
*client SDKs* (handle OAuth flows, token storage, session management in
browser/mobile) and *server SDKs* (call Management API for admin operations).
This separation is critical and GGID should adopt it.

### Keycloak (5 SDKs) — Enterprise Java Focus

| SDK | Type | Key Features |
|-----|------|-------------|
| **keycloak-admin-cli** | CLI | Full admin operations |
| **keycloak-admin-client** | Java | JAX-RS admin client |
| **keycloak-js** | JavaScript | Adapter for SPAs |
| **keycloak-nodejs-connect** | Node.js | Express middleware |
| **keycloak-spring-security** | Spring | Spring Security integration |

**Key differentiator**: Deep Java/Spring ecosystem integration. Spring Security
adapter is a major adoption driver for enterprise Java shops. However,
Keycloak's SDK coverage is narrow outside Java.

### Ory (6+ SDKs) — Auto-Generated

| SDK | Type | Generation Method |
|-----|------|------------------|
| **ory-go** | Go | OpenAPI codegen |
| **ory-client** | JS/TS | OpenAPI codegen (npm) |
| **ory-python** | Python | OpenAPI codegen (PyPI) |
| **ory-php** | PHP | OpenAPI codegen |
| **ory-dotnet** | .NET | OpenAPI codegen (NuGet) |
| **ory-ruby** | Ruby | OpenAPI codegen (RubyGems) |

**Key differentiator**: Ory auto-generates all SDKs from their OpenAPI spec
using a custom codegen pipeline. This means zero manual maintenance per SDK —
any API change triggers regeneration. The tradeoff is that generated SDKs are
thinner (no middleware, no framework integrations, no OAuth flow helpers).
GGID already has a 2,397-line OpenAPI spec at `docs/openapi.yaml` — this is
a viable path.

### Casdoor (8+ SDKs) — Broad Language Coverage

| SDK | Type |
|-----|------|
| **casdoor-go-sdk** | Go |
| **casdoor-js-sdk** | JavaScript |
| **casdoor-java-sdk** | Java |
| **casdoor-python-sdk** | Python |
| **casdoor-dotnet-sdk** | .NET |
| **casdoor-rust-sdk** | Rust |
| **casdoor-android-sdk** | Android |
| **casdoor-ios-sdk** | iOS |

**Key differentiator**: Casdoor covers more languages than any other open-source
IAM, including Rust and native mobile. Each SDK is hand-written, with Go as the
reference implementation. However, quality varies significantly across SDKs —
the Go and JS SDKs are well-maintained, while Rust/iOS have fewer contributors.

### Competitive Summary Table

| Feature | Auth0 | Keycloak | Ory | Casdoor | **GGID** |
|---------|-------|----------|-----|---------|----------|
| Total SDKs | 12+ | 5 | 6+ | 8+ | **4** |
| Client SDKs (SPA) | 4 (React/Angular/Vue/Next) | 1 | 0 | 1 | **0** |
| Server SDKs | 7+ | 4 | 6+ | 8+ | **4** |
| Mobile SDKs | 2 (iOS/Android) | 0 | 0 | 2 | **0** |
| Auto-generated | No | No | Yes (6) | No | **No** |
| OpenAPI spec | Yes | Yes | Yes | No | **Yes** |
| Rust SDK | No | No | No | Yes | **No** |
| .NET SDK | Yes | No | Yes | Yes | **No** |
| Ruby SDK | Yes | No | Yes | No | **No** |

---

## 2. GGID Current SDK Status

### SDK Inventory (as of source code review)

#### Go SDK (`sdk/go/`) — Production Grade

**Files**: `client.go` (629 lines), `middleware.go` (150 lines), plus tests

| Feature | Status | Notes |
|---------|--------|-------|
| JWT verification (online) | YES | RS256 via JWKS with caching (`WithJWKS(ttl)`) |
| JWT verification (offline) | YES | `ParseUnverified` claims extraction |
| Login (password grant) | YES | `Login(ctx, *LoginRequest)` |
| Token refresh | YES | `RefreshToken(ctx, refreshToken)` |
| Logout | YES | `Logout(ctx, accessToken)` |
| Register user | NO | Missing — only management `CreateUser` |
| User CRUD | YES | Create/Get/Update/Delete/List |
| Role management | YES | Create/List/Assign/Remove |
| Permission check | YES | `CheckPermission` via policy engine |
| Organization CRUD | YES | Create/List |
| HTTP middleware | YES | `Middleware()` — Bearer extraction + JWT verify |
| Role-based middleware | YES | `RequireRole()` — local JWT claims check |
| Scope-based middleware | YES | `RequireScope()` — local JWT claims check |
| Permission middleware | YES | `RequirePermission()` — calls policy engine |
| JWKS caching | YES | Thread-safe with `sync.RWMutex`, configurable TTL |
| Structured errors | YES | `APIError` with IsNotFound/IsUnauthorized/etc |
| Tenant context | YES | `MiddlewareConfig.TenantID` → `X-Tenant-ID` header |
| PKCE/OAuth flow | NO | Missing — no OIDC auth code flow helper |
| User info endpoint | NO | Missing — no `/userinfo` call |

**Verdict**: Most mature SDK. Complete for server-side management use cases.
Missing OAuth/OIDC client-side flow helpers.

#### Node.js/TypeScript SDK (`sdk/node/`) — Production Grade

**Files**: `client.ts`, `jwt.ts`, `middleware.ts`, `types.ts`, `index.ts`

| Feature | Status | Notes |
|---------|--------|-------|
| JWT verification | YES | Via `jose` library, JWKS caching built-in |
| Login | YES | `client.login(input)` |
| Register | YES | `client.register(username, email, password)` |
| Token refresh | YES | `client.refreshToken(refreshToken)` |
| Logout | YES | `client.logout(accessToken)` |
| User CRUD | YES | Full CRUD + pagination |
| Role management | YES | Create/List/Assign/Remove |
| Permission check | YES | `client.checkPermission(userId, resource, action)` |
| Organization CRUD | YES | Create/List |
| Express middleware | YES | `expressAuth()` — JWT verification |
| Role middleware | YES | `requireRole(role)` — local check |
| Permission middleware | YES | `requirePermission(config, resource, action)` |
| Structured errors | YES | `GGIDError` class with isNotFound/isRateLimited/etc |
| Tenant context | YES | Auto `X-Tenant-ID` header |
| PKCE/OAuth flow | NO | Missing |
| User info endpoint | NO | Missing |
| Timeout control | YES | AbortController with configurable timeout |
| NPM package | NO | No `package.json` — not yet published |

**Verdict**: Strong SDK with excellent TypeScript types. Missing NPM publish
configuration and OAuth flow helpers.

#### Python SDK (`sdk/python/`) — Partial

**Files**: `__init__.py`, `client.py`, `jwt.py`, `middleware.py`

| Feature | Status | Notes |
|---------|--------|-------|
| JWT verification | YES | RS256 via PyJWT, JWKS caching |
| Login | YES | `client.login(username, password)` |
| Register | YES | `client.register(username, email, password)` |
| Token refresh | NO | Missing |
| Logout | NO | Missing |
| User management | PARTIAL | list/get/create/delete — no update |
| Role management | PARTIAL | list only — no create/assign/remove |
| Permission check | YES | `client.check_permission(token, resource, action)` |
| Organization CRUD | NO | Missing |
| FastAPI middleware | YES | `GGIDMiddleware` (Starlette BaseHTTPMiddleware) |
| Flask decorator | PARTIAL | `@requires_auth` — token extraction only, no verify |
| Django decorator | PARTIAL | `ggid_login_required` — token extraction only, no verify |
| Structured errors | NO | Uses raw `httpx` exceptions |
| Tenant context | YES | Default tenant ID in constructor |
| PyPI package | NO | No `setup.py` or `pyproject.toml` |
| Async support | YES | httpx AsyncClient throughout |

**Verdict**: Has the infrastructure but is incomplete. Missing token refresh,
logout, role management, org management, and proper error handling. Flask and
Django middleware only extract tokens without verifying.

#### Java SDK (`sdk/java/`) — README Only

**Files**: `README.md`, `pom.xml` — **NO SOURCE CODE**

The README documents a full API (login, createUser, checkPermission, etc.) and
the `pom.xml` declares dependencies (jackson, java-jwt, okhttp), but there are
zero `.java` source files. This SDK exists only on paper.

**Verdict**: Not usable. Needs full implementation from scratch or
auto-generation.

### Feature Matrix Across GGID SDKs

| Feature | Go | Node.js | Python | Java |
|---------|----|---------|--------|----|
| JWT verification (JWKS) | YES | YES | YES | PLANNED |
| Login | YES | YES | YES | PLANNED |
| Token refresh | YES | YES | NO | PLANNED |
| Logout | YES | YES | NO | PLANNED |
| User CRUD | FULL | FULL | PARTIAL | PLANNED |
| Role management | FULL | FULL | PARTIAL | PLANNED |
| Org management | PARTIAL | PARTIAL | NO | PLANNED |
| Permission check | YES | YES | YES | PLANNED |
| HTTP middleware | YES | YES (Express) | YES (FastAPI) | PLANNED (Servlet) |
| Structured errors | YES | YES | NO | PLANNED |
| PKCE/OAuth flow | NO | NO | NO | NO |
| Package published | Go module | NPM (unpublished) | PyPI (unpublished) | Maven (unpublished) |
| **Maturity** | **Production** | **Production** | **Beta** | **Vaporware** |

---

## 3. SDK Feature Requirements

Every IAM SDK must provide the following capabilities. These are the
non-negotiable features that developers expect from an IAM SDK in 2025.

### 3.1 Core Authentication

| Requirement | Description | Priority |
|-------------|-------------|----------|
| **OIDC Auth Code Flow + PKCE** | Full redirect-based flow for web/mobile apps | P0 |
| **Token Validation** | RS256 JWT verification via JWKS with caching | P0 |
| **Token Refresh** | Automatic refresh when access token expires | P0 |
| **User Info Fetch** | Call `/userinfo` endpoint for profile data | P1 |
| **Session Management** | Token storage, expiry tracking, silent renew | P1 |

### 3.2 Authorization

| Requirement | Description | Priority |
|-------------|-------------|----------|
| **Role Check** | Verify user has a role (from JWT claims) | P0 |
| **Scope Check** | Verify user has an OAuth scope (from JWT claims) | P0 |
| **Permission Check** | Call policy engine for ABAC evaluation | P1 |
| **Tenant Context** | Propagate `X-Tenant-ID` automatically | P0 |

### 3.3 Framework Integration

| Requirement | Description | Priority |
|-------------|-------------|----------|
| **Middleware/Interceptor** | Drop-in auth for web frameworks | P0 |
| **Decorator/Annotation** | Per-route auth requirements | P1 |
| **Context/Request Injection** | User info available in request handlers | P0 |

### 3.4 Reference SDK Design (Go)

The following interface represents the minimal contract every GGID SDK should
implement. This serves as the reference specification for all language ports:

```go
// GGIDClient is the entry point for all SDK operations.
type GGIDClient interface {
    // --- Authentication ---
    // AuthCodeURL generates the OIDC authorization URL with PKCE challenge.
    AuthCodeURL(state, redirectURI string) (authURL, codeVerifier string)

    // ExchangeCode exchanges the auth code for tokens (completes PKCE flow).
    ExchangeCode(ctx, code, codeVerifier, redirectURI string) (*TokenSet, error)

    // Login performs password-based authentication (for service accounts).
    Login(ctx, username, password string) (*TokenSet, error)

    // RefreshToken refreshes an expired access token.
    RefreshToken(ctx, refreshToken string) (*TokenSet, error)

    // VerifyToken validates a JWT and returns user claims.
    // Uses JWKS caching. Verifies exp, nbf, iss, aud.
    VerifyToken(ctx, accessToken string) (*UserInfo, error)

    // UserInfo fetches the user's profile from the /userinfo endpoint.
    UserInfo(ctx, accessToken string) (*UserInfo, error)

    // --- Authorization ---
    // HasRole checks if the user has a role (from JWT claims, no API call).
    HasRole(userInfo *UserInfo, role string) bool

    // HasScope checks if the user has an OAuth scope.
    HasScope(userInfo *UserInfo, scope string) bool

    // CheckPermission calls the policy engine for ABAC evaluation.
    CheckPermission(ctx, userID, resource, action string) (bool, error)

    // --- Management (requires API key) ---
    // User CRUD operations
    CreateUser(ctx, req *CreateUserRequest) (*User, error)
    GetUser(ctx, userID string) (*User, error)
    UpdateUser(ctx, userID string, req *UpdateUserRequest) (*User, error)
    DeleteUser(ctx, userID string) error
    ListUsers(ctx, opts *ListOptions) (*PageResult[User], error)

    // Role CRUD operations
    CreateRole(ctx, req *CreateRoleRequest) (*Role, error)
    AssignRole(ctx, userID, roleID string) error
    RemoveRole(ctx, userID, roleID string) error

    // --- Framework Integration ---
    // Middleware returns an HTTP middleware that enforces JWT auth.
    Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler
}
```

### 3.5 Gap Against Reference Design

| Feature | Go | Node.js | Python | Java | Required |
|---------|----|---------|--------|------|---------|
| AuthCodeURL (PKCE) | MISSING | MISSING | MISSING | N/A | P0 |
| ExchangeCode | MISSING | MISSING | MISSING | N/A | P0 |
| UserInfo endpoint | MISSING | MISSING | MISSING | N/A | P1 |
| Token storage/renew | MISSING | MISSING | MISSING | N/A | P1 |

All four SDKs lack OAuth/OIDC client-side flow helpers. This is the single
biggest functional gap — without `AuthCodeURL` and `ExchangeCode`, developers
must implement the PKCE flow manually, which is error-prone and a security risk.

---

## 4. Priority Matrix

### Scoring Criteria (1-5 scale, 5 = highest)

| Criterion | Meaning |
|-----------|---------|
| **Market Demand** | Language popularity and developer requests |
| **Integration Difficulty** | Effort to build a production SDK (5 = easy) |
| **Competitive Necessity** | Required for parity with competitors |
| **Maintenance Cost** | Ongoing effort (5 = low cost) |
| **Total** | Sum of all scores |

### Language Priority Scoring

| Language | Demand | Difficulty | Comp. Necessity | Maint. Cost | Total | Rank |
|----------|--------|------------|-----------------|-------------|-------|------|
| **Go** (exists) | 4 | 5 | 5 | 4 | **18** | 1 |
| **TypeScript/Node.js** (exists) | 5 | 5 | 5 | 4 | **19** | 1 |
| **Python** (exists, needs work) | 5 | 4 | 5 | 4 | **18** | 1 |
| **Java** (vaporware) | 4 | 3 | 4 | 3 | **14** | 4 |
| **.NET / C#** | 4 | 3 | 5 | 3 | **15** | 5 |
| **Rust** | 2 | 2 | 3 | 3 | **10** | 8 |
| **Ruby** | 2 | 4 | 3 | 3 | **12** | 7 |
| **PHP** | 3 | 4 | 3 | 3 | **13** | 6 |
| **Swift (iOS)** | 3 | 2 | 3 | 2 | **10** | 8 |
| **Kotlin (Android)** | 3 | 2 | 3 | 2 | **10** | 8 |
| **Dart (Flutter)** | 3 | 3 | 2 | 2 | **10** | 8 |

### Priority Order

1. **P0 — Stabilize Existing**: Complete Python SDK, implement Java SDK
2. **P1 — Auto-Generate**: Use OpenAPI spec to generate .NET, Ruby, PHP SDKs
3. **P2 — Mobile**: Swift (iOS) + Kotlin (Android) — client-side auth
4. **P3 — Emerging**: Rust SDK for systems programmers
5. **P4 — Community**: Support community-maintained SDKs for niche languages

### Competitive Necessity Analysis

Languages where GGID has **no** SDK but every major competitor does:

| Language | Auth0 | Keycloak | Ory | Casdoor | Gap Impact |
|----------|-------|----------|-----|---------|------------|
| .NET | YES | No | YES | YES | **HIGH** — large enterprise market |
| Ruby | YES | No | YES | No | MEDIUM — Rails community |
| PHP | No | No | YES | No | MEDIUM — WordPress/Laravel |
| Swift | YES | No | No | YES | LOW-MEDIUM — iOS only |
| Kotlin | YES | No | No | YES | LOW-MEDIUM — Android only |

---

## 5. Minimum Viable SDK Set

### Recommendation: Ship 4 SDKs First

#### 1. Go SDK (Exists — Enhance)

**Why**: GGID itself is written in Go. The Go SDK serves as the reference
implementation. Go is the #2 backend language (after Node.js) in cloud-native
environments.

**Target audience**: Go microservice developers, DevOps engineers building
API gateways, backend teams in cloud-native shops.

**Key features to add**:
- OIDC Auth Code + PKCE flow (`AuthCodeURL`, `ExchangeCode`)
- `/userinfo` endpoint support
- Token storage interface (pluggable: memory, Redis)
- Auto-refresh on 401 (middleware-level)

**Effort**: 2-3 engineer-days

#### 2. Node.js/TypeScript SDK (Exists — Polish + Publish)

**Why**: JavaScript/TypeScript is the most popular backend language. Express
is the most-used web framework. Node.js SDKs have the highest NPM download
volume for any IAM platform.

**Target audience**: Full-stack JS/TS developers, Express/Fastify/Hono users,
Next.js API route developers, serverless (Lambda/Cloudflare Workers).

**Key features to add**:
- `package.json` with proper `@ggid/node` publish config
- OIDC Auth Code + PKCE flow
- Token storage interface (cookie, session, memory)
- Hono/Fastify middleware adapters (beyond Express)
- `/userinfo` endpoint support

**Effort**: 3-4 engineer-days

#### 3. Python SDK (Exists — Complete)

**Why**: Python is the #1 language for data science, ML, and scripting.
FastAPI is the fastest-growing Python web framework. Many IAM use cases
(Jupyter notebook auth, ML pipeline access control) require a Python SDK.

**Target audience**: FastAPI/Django/Flask developers, data platform teams,
ML/AI engineers needing identity-aware pipelines.

**Key features to add**:
- Token refresh + logout methods
- Full role management (create/assign/remove)
- Organization CRUD
- Structured error handling (custom exception classes)
- `setup.py` / `pyproject.toml` for PyPI publish
- Sync client variant (not just async)
- Flask/Django middleware with actual JWT verification (currently extracts
  token but does not verify)

**Effort**: 3-4 engineer-days

#### 4. Java SDK (Vaporware — Implement)

**Why**: Java remains the dominant language in enterprise. Spring Boot is
the most-used enterprise framework. Keycloak's primary adoption vector is
the Spring Security adapter. Without a Java SDK, GGID cannot compete for
enterprise Java accounts.

**Target audience**: Spring Boot developers, enterprise architects, banking/
insurance/government teams using Java.

**Key features**:
- Full client (login, token verify, refresh, logout)
- User/Role/Org CRUD
- Servlet Filter for JWT auth (`GGIDAuthFilter`)
- Spring Security integration (optional)
- Maven Central publication

**Effort**: 5-7 engineer-days (implementing from scratch)

### What NOT to Build Yet

- **Swift/Kotlin**: Mobile SDKs require OAuth flow management, secure token
  storage (Keychain/Keystore), and deep platform knowledge. Defer to Q3.
- **Rust**: Niche market; community SDK is more appropriate.
- **PHP/Ruby**: Auto-generate from OpenAPI spec; hand-written is not worth
  the maintenance cost for these language ecosystems.

---

## 6. SDK Generation Strategy

### Current State

GGID has:
- **OpenAPI 3.1.0 spec**: `docs/openapi.yaml` (2,397 lines, 84 endpoints)
- **gRPC proto definitions**: 6 proto files in `api/proto/`
- **Generated Go code**: `api/gen/` (gRPC stubs)

No SDK auto-generation pipeline exists.

### Recommended Approach: Hybrid Model

GGID should adopt a **hybrid SDK strategy** — auto-generate the API client
layer, then hand-write the framework integration and OAuth flow layer on top.

```
                    +---------------------+
                    |  OpenAPI Spec       |
                    |  (docs/openapi.yaml)|
                    +----------+----------+
                               |
              +----------------+----------------+
              |                                 |
    +---------v----------+           +----------v-----------+
    | openapi-generator  |           | Hand-written layer    |
    | (codegen)          |           | (middleware, OAuth,   |
    +----+---------------+           |  PKCE, token storage) |
         |                           +----------+------------+
    +----v------------------+                   |
    | Generated API client  |    +--------------v-----------+
    | (types, HTTP calls)   +----> Final SDK Package         |
    +-----------------------+    | (npm, PyPI, Maven, etc.) |
                                 +--------------------------+
```

### Auto-Generation Tools

| Tool | Languages | License | Notes |
|------|-----------|---------|-------|
| **openapi-generator** | 40+ | Apache 2.0 | Industry standard, widely used |
| **oapi-codegen** | Go | Apache 2.0 | Best Go generator, type-safe |
| **openapi-typescript** | TypeScript | MIT | Generates types from OpenAPI |
| **Swift OpenAPI Generator** | Swift | Apache 2.0 | Apple's official generator |

### Tradeoffs: Auto-Gen vs Hand-Written

| Aspect | Auto-Generated | Hand-Written |
|--------|---------------|-------------|
| **Speed to market** | Fast (1 day per language) | Slow (1 week per language) |
| **API coverage** | Complete (100% of endpoints) | Partial (only what's coded) |
| **Type safety** | Good (generated types) | Excellent (idiomatic) |
| **Framework integration** | None | Full (middleware, decorators) |
| **OAuth/PKCE flows** | None | Full |
| **Maintenance** | Low (regenerate on change) | High (manual updates) |
| **Developer experience** | Mediocre (generic API) | Excellent (idiomatic) |
| **Documentation** | Auto-generated (minimal) | Rich (hand-written) |

### Recommendation

1. **Use `openapi-generator`** for: .NET, Ruby, PHP, Rust — languages where
   GGID won't invest in hand-written SDKs
2. **Hand-write + use generated client internally** for: Go, Node.js, Python,
   Java — the 4 priority SDKs
3. **Add OpenAPI lint** to CI pipeline to ensure the spec stays in sync with
   actual API behavior

---

## 7. SDK Quality Standards

### Quality Checklist for Each New SDK

Every GGID SDK must meet these standards before publication:

#### 7.1 Core Requirements

- [ ] **JWT verification** with JWKS caching (RS256)
- [ ] **Login** (password grant)
- [ ] **Token refresh**
- [ ] **Logout** (token revocation)
- [ ] **User management** CRUD (create, get, update, delete, list)
- [ ] **Role management** (create, list, assign, remove)
- [ ] **Permission check** (policy engine integration)
- [ ] **Organization management** (create, list)
- [ ] **Tenant context** propagation (`X-Tenant-ID` header)

#### 7.2 Framework Integration

- [ ] **HTTP middleware/decorator** for JWT verification
- [ ] **Role-based** route protection (local JWT claims)
- [ ] **Permission-based** route protection (policy engine call)
- [ ] **Request context** injection of authenticated user

#### 7.3 Error Handling

- [ ] **Structured error type** with HTTP status code
- [ ] **Error classification** methods: `isNotFound()`, `isUnauthorized()`,
      `isForbidden()`, `isConflict()`, `isRateLimited()`
- [ ] **Network error** wrapping with context
- [ ] **Consistent error messages** across all SDKs

#### 7.4 Configuration

- [ ] **Builder/options pattern** for client construction
- [ ] **Custom HTTP client** injection (for testing, proxy support)
- [ ] **Configurable timeouts** (default 30s)
- [ ] **Configurable JWKS TTL** (default 15m)
- [ ] **Environment variable** support for all config values

#### 7.5 Testing

- [ ] **Unit tests** with mock HTTP server (>= 80% coverage)
- [ ] **Integration test** against running GGID instance
- [ ] **Error path tests** (404, 401, 403, 409, 429, 500)
- [ ] **JWKS caching** behavior tests
- [ ] **Concurrent** token verification tests

#### 7.6 Documentation

- [ ] **README** with quick start (3-line promise)
- [ ] **API reference** table (every method documented)
- [ ] **Code examples** for every common use case
- [ ] **Error handling** guide
- [ ] **Framework integration** guide (per supported framework)

#### 7.7 Packaging

- [ ] **Semantic versioning** (v1.0.0)
- [ ] **CHANGELOG.md** with per-release notes
- [ ] **Published** to language package registry
- [ ] **License file** (Apache 2.0)
- [ ] **CI pipeline** for build + test + publish

---

## 8. Community SDK Strategy

### Governance Model

GGID should adopt a tiered endorsement model similar to HashiCorp's Terraform
provider ecosystem:

#### Tier 1: Official SDKs (GGID Core Team Maintained)

- Maintained by the GGID core team
- Published under `@ggid/*` / `dev.ggid:*` / etc.
- Full CI/CD, documentation, and support
- Backward compatibility guaranteed within major versions
- **Current**: Go, Node.js, Python, Java (planned)

#### Tier 2: Endorsed SDKs (Community Maintained, GGID Reviewed)

- Maintained by trusted community contributors
- Published under contributor's namespace (e.g., `@someone/ggid-rust`)
- GGID team reviews major versions
- Listed in official documentation as "community-endorsed"
- **Criteria**: 90%+ test coverage, follows GGID SDK quality standards,
  maintainer responds to issues within 7 days, at least 100 GitHub stars
- **Example**: Ory's community SDKs, Terraform community providers

#### Tier 3: Community SDKs (Unendorsed)

- Any developer can create an SDK
- Not reviewed or endorsed by GGID team
- Listed in a "community SDKs" wiki page
- No compatibility guarantees
- **Example**: Early Casdoor SDKs before official adoption

### Quality Bar for Endorsement

A community SDK must meet ALL of the following to be considered for Tier 2
endorsement:

1. **Test coverage** >= 80%
2. **CI pipeline** with automated tests
3. **README** with quick start, API reference, error handling
4. **Published** to the language's package registry
5. **Semantic versioning** with CHANGELOG
6. **Open source** (Apache 2.0 or MIT)
7. **Active maintenance** (commits within last 30 days)
8. **No critical security issues** (code review by GGID team)

### Incentive Program

To encourage community SDK development:

- **Bounty program**: $500-$2000 per accepted Tier 2 SDK (sponsored by GGID)
- **Recognition**: Community SDK maintainers listed in CONTRIBUTORS.md
- **Conference talks**: Invite maintainers to present at GGID events
- **Priority support**: Tier 2 maintainers get direct access to GGID core team
- **Hackathon**: Annual "GGID SDK Hackathon" to bootstrap new language SDKs

---

## 9. Gap Analysis and Recommendations

### Gap Summary

| Gap | Current State | Target State | Severity |
|-----|---------------|-------------|----------|
| Java SDK | README only, no source | Full SDK with Servlet Filter | CRITICAL |
| Python SDK incomplete | Missing refresh/logout/roles/orgs | Full feature parity with Go | HIGH |
| No OAuth/PKCE flow | Missing in all SDKs | Present in all 4 SDKs | HIGH |
| No NPM/PyPI/Maven publish | Unpublished packages | Published to registries | HIGH |
| No .NET SDK | Missing | Auto-generated from OpenAPI | MEDIUM |
| No mobile SDKs | Missing | Swift + Kotlin (deferred) | LOW |
| No OpenAPI codegen pipeline | Manual SDK maintenance | Auto-gen for 6+ languages | MEDIUM |
| No client-side SPA SDK | Missing | React/Vue hooks (deferred) | MEDIUM |
| Flask/Django no JWT verify | Token extracted but not verified | Full JWKS verification | HIGH |
| Python no structured errors | Raw httpx exceptions | Custom exception classes | MEDIUM |

### Action Items with Effort Estimates

#### Phase 1: Stabilize (Q3 2025) — 3 weeks

| # | Task | Effort | Priority |
|---|------|--------|----------|
| 1 | Implement Java SDK (client + Servlet Filter) | 5-7 days | P0 |
| 2 | Complete Python SDK (refresh, logout, roles, orgs, errors) | 3-4 days | P0 |
| 3 | Add OIDC Auth Code + PKCE flow to Go + Node.js SDKs | 3 days | P0 |
| 4 | Fix Flask/Django middleware to actually verify JWTs | 1 day | P0 |
| 5 | Add `package.json` / `pyproject.toml` / `setup.py` | 1 day | P0 |
| 6 | Publish all 4 SDKs to package registries | 2 days | P0 |
| 7 | Add `/userinfo` endpoint support to all SDKs | 1 day | P1 |

**Total Phase 1 effort**: ~16-19 engineer-days (3-4 weeks with 1 engineer)

#### Phase 2: Expand (Q4 2025) — 3 weeks

| # | Task | Effort | Priority |
|---|------|--------|----------|
| 8 | Set up OpenAPI codegen pipeline (openapi-generator) | 2 days | P1 |
| 9 | Auto-generate .NET SDK from OpenAPI spec | 1 day | P1 |
| 10 | Auto-generate Ruby SDK from OpenAPI spec | 1 day | P2 |
| 11 | Auto-generate PHP SDK from OpenAPI spec | 1 day | P2 |
| 12 | Add Hono/Fastify adapters to Node.js SDK | 1 day | P1 |
| 13 | Add sync client variant to Python SDK | 1 day | P1 |
| 14 | Write SDK contribution guide + quality checklist | 1 day | P1 |
| 15 | Set up SDK CI pipeline (build + test + lint per SDK) | 2 days | P1 |

**Total Phase 2 effort**: ~10 engineer-days (2 weeks with 1 engineer)

#### Phase 3: Mobile + Community (Q1 2026) — 4 weeks

| # | Task | Effort | Priority |
|---|------|--------|----------|
| 16 | Swift SDK (iOS) with PKCE + Keychain storage | 7-10 days | P2 |
| 17 | Kotlin SDK (Android) with PKCE + EncryptedSharedPreferences | 7-10 days | P2 |
| 18 | Launch community SDK bounty program | 2 days | P2 |
| 19 | Add Spring Security adapter to Java SDK | 3 days | P1 |
| 20 | Add token storage abstraction (memory/Redis/cookie) | 3 days | P1 |

**Total Phase 3 effort**: ~22-28 engineer-days (4-5 weeks with 1 engineer)

### Success Metrics

| Metric | Current | Q3 Target | Q4 Target | Q1 2026 Target |
|--------|---------|-----------|-----------|----------------|
| Total SDKs | 4 (1 vaporware) | 4 (all functional) | 7 (auto-gen +3) | 9 (+mobile) |
| SDKs with PKCE flow | 0 | 4 | 4 | 6 |
| SDKs published to registry | 0 | 4 | 7 | 9 |
| NPM weekly downloads | 0 | 100 | 500 | 2,000 |
| PyPI monthly downloads | 0 | 50 | 300 | 1,000 |
| GitHub stars on SDK repos | 0 | 50 | 200 | 500 |
| Languages covered | 4 | 4 | 7 | 9 |
| Competitor parity languages | 4/12 (Auth0) | 4/12 | 7/12 | 9/12 |

---

## Conclusion

GGID's SDK ecosystem is at a critical inflection point. The Go and Node.js
SDKs demonstrate that the team can build production-quality SDKs — the patterns
are proven. The immediate priorities are clear:

1. **Stop shipping vaporware** — implement the Java SDK or remove it
2. **Complete the Python SDK** — it's 60% done and needs 3-4 days of work
3. **Add OAuth/PKCE flows** — this is the #1 feature gap vs all competitors
4. **Publish to registries** — unpublished packages might as well not exist
5. **Invest in OpenAPI codegen** — this is the force multiplier that lets GGID
   match Auth0's 12+ SDK coverage without multiplying maintenance cost

With a focused 10-week effort (3 phases), GGID can go from 4 incomplete SDKs
to 9 production SDKs covering 75% of Auth0's language coverage — while
leveraging auto-generation to keep maintenance costs manageable.

---

*Co-Authored-By: ggcode <noreply@ggcode.dev>*
