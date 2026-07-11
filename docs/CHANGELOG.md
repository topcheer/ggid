# GGID Changelog

All notable changes to the GGID IAM Platform are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased] — Milestone: Documentation & Developer Experience

### Latest Changes (2026-07-26)

#### SCIM PATCH URN Colon Notation Fix (commit 513548b)
- Fixed `isAlpha` redeclaration conflict between `filter.go` and `patch.go`
- Fixed URN colon notation parsing for single-level sub-attributes (`urn:...:User:department`)
- Added multi-level nested attribute traversal in `setNestedAttr` (`manager.displayName`)

#### External Database/Middleware Support (commit d56efc9)
- All deployment methods (Docker, K8s/Helm, K3s, Terraform, Bare Metal) support external PostgreSQL, Redis, NATS, LDAP
- Docker Compose: `.env` file and override file support
- Helm: `externalDatabase`, `externalRedis`, `externalNats` values
- Terraform: `external_database_host`, `external_redis_host`, `external_nats_host` variables

#### Gap Regression Verification (15/20 verified)
- 3 gaps upgraded MEDIUM→HIGH confidence (28 regression tests)
- SCIM enterprise URN colon notation compliance (RFC 7644 §3.10)
- Audit hash chain tamper detection verified
- CSRF state validation (one-time use, cross-client isolation)
- HasScope enforcement (deny-by-default, wildcard, multi-scope)

#### i18n Expansion (827 keys total)
- Console i18n: 979→58 lines refactored to JSON-backed dictionaries
- 827 total translation keys across all console pages
- gen-i18n-dicts.py script for dictionary generation

### Gap Closure & Security Verification (2026-07-24)

#### Gap Regression Verification (3 gaps, 28 tests)
- **Audit Hash Chain** — 12 regression tests: repository wiring proof, field-level tamper detection, event deletion/insertion detection, cross-tenant isolation, replay attack defense (commit 76bc881)
- **CSRF State Validation** — 8 tests: happy path, one-time use (RFC 6749 §10.12), expired state, cross-client isolation, multiple concurrent states, expiry cleanup
- **HasScope Enforcement** — 8 tests: wildcard scope, multiple scopes, API key priority, JWT fallback, deny-by-default (P0 regression), empty scope list, 20-scope bypass attempt

#### i18n Refactoring (commits 133eaac, 33e1ce3)
- Console i18n: 979→58 lines in i18n.tsx, JSON-backed dictionaries (611 keys)
- 33 new keys for SSO/OAuth/API Keys/Certificates settings pages (commit 2e46047)
- gen-i18n-dicts.py script for dictionary generation

#### P0 Security Fixes — All Complete
- CSRF predictable entropy → crypto/rand.Read()
- Rate limiter wired into production Handler() chain
- SecurityHeaders wired into handler chain
- Tenant spoofing — JWT claim takes priority over X-Tenant-ID header
- Admin API scope check — hasAdminScope() guards /api/v1/admin/*
- OAuth state validation on token exchange (Redis-backed)
- JWT jti tracking with Redis SETNX (anti-replay)
- JWTSecret empty → log.Fatal (no silent bypass)
- HasScope() actual scope enforcement (no longer always true)
- iss parameter in auth redirect

#### Docker E2E — 11/11 PASS
- Full stack: 12 containers, 7 microservices + Console + 4 infrastructure
- Tests: gateway health, register, login+JWT, 401 without JWT, list users, create role, list roles, create org, audit query, wrong password 401, duplicate register 409
- Gateway tenant forwarding: injects tenant_id as query param AND JSON body field

#### Build Fixes
- StructuredLogger.Emit nil pointer fixed (commit 9a4f46e)
- Fast Argon2id hashing for auth service tests (commit 7e8e29f)
- Middleware chain order verification tests (commit 4b952b2)

### Documentation Expansion (170+ docs)

#### Quickstarts & Tutorials
- 5 copy-paste quickstart guides (5-min JWT, RBAC, OAuth, Go SDK, Node SDK)
- Unified SDK quickstart entry page (Go/Node/Python/Java side-by-side)
- 4 step-by-step tutorials (multi-tenant setup, custom auth provider, webhook integration, SAML SP config)
- Docker 5-minute quickstart with expected output for every command

#### Deployment Guides
- Docker deployment guide (compose explained, env vars, health checks, E2E walkthrough)
- Kubernetes/Helm deployment guide (values reference, ingress, cert-manager, HPA)
- K3s deployment guide (OrbStack, Traefik ingress, registry.iot2.win)
- Bare metal deployment guide (systemd units, nginx reverse proxy, Let's Encrypt)
- Docker Compose override guide (8 use cases with examples)
- Private Docker registry setup guide
- Complete environment variables reference (all 7 services)
- Deployment troubleshooting (40+ issues across 6 categories)
- Helm values complete reference
- Terraform module (main.tf, variables.tf, outputs.tf, README)

#### Architecture & Design
- Architecture overview for evaluators (system diagram, service responsibilities, data flow)
- 3 ADRs in docs/design/ (PostgreSQL+RLS, NATS JetStream audit, provider chain)
- ADR-0001 JWT RSA shared key decision record
- AI Agent Identity / MCP authentication design doc

#### Integration & Developer Experience
- 3 framework integration guides (Express.js, Gin, Spring Boot)
- Frontend i18n guide (next-intl config, message key convention, LanguageSwitcher)
- Backend i18n guide (pkg/i18n translator, 937 string extraction plan)
- Operations runbook (deploy, tenants, key rotation, backups, monitoring, emergencies)
- Authentication guide (all 8 methods, provider chain, session management)
- Data protection guide (encryption, PII, GDPR, key management)
- Incident response runbook (5 playbooks, forensic tools, communication plan)
- Network security guide (segmentation, firewall, mTLS planning, DDoS)
- Vulnerability management (scanning, SLAs, 19 P0 patches history)

#### API Reference
- Complete API reference: 78 endpoint subsections, 108+ endpoint references
- Complete error codes reference (57 error codes across 7 services)
- Performance tuning with benchmark_test.go results and pool sizing formulas

### Build & Test
- Makefile fix: exclude `sdk/examples/` from test targets (build constraint: ignore)
- 32/32 packages pass, 0 FAIL, 250+ test cases

---

## [Unreleased]

### Added

#### Documentation (40+ docs)
- Architecture Decisions (ADRs): 15 decisions with Context/Decision/Consequences
- Security Architecture: trust boundaries, defense in depth, JWT flow, tenant isolation, incident response
- Developer Environment: prerequisites, VS Code config, debugging tips, common errors
- RBAC Guide: role hierarchy, permission model, RBAC+ABAC evaluation, common patterns
- Webhook Guide: registration, 25+ event types, HMAC verification (Node/Python/Go), SSRF protection
- API Gateway Guide: reverse proxy, route table, middleware chain, tenant injection, circuit breaker
- API Reference expanded (+292 lines): Policy/ABAC, Tenant, Session, Notification, Webhook Delivery endpoints
- Multi-Tenancy expanded (+276 lines): tenant CRUD, migration patterns, per-tenant rate limiting
- 15+ security research documents (CIBA, back-channel logout, SAML metadata, MFA bypass, DPoP, etc.)

#### Console (Admin UI)
- Branding page: logo upload, 3-color picker, live preview
- Webhook tester with real-time delivery inspection
- User activity timeline with session history
- API explorer with interactive endpoint testing
- Certificate manager page
- Notification preview page
- Enhanced OAuth consent screen with history and custom text
- Enhanced impersonation with audit trail and history
- Access keys with scoped secrets and IP binding
- Enhanced group management with nesting and bulk operations
- SSO top-level route (`/sso`)

#### Security Fixes (P0)
- Database backup automation (P0)
- JWT access tokens now include scope claim (P0)
- SSRF protection wired into webhook delivery by default
- JWT signature verification enforced in `ValidateClientAssertion` (P0)
- DPoP (RFC 9449) proof validation for sender-constrained tokens

#### Infrastructure
- Prometheus `/metrics` endpoint on all 7 services
- `slog` structured logging for gateway
- Middleware coverage boosted: 89.4% → 90.0% (sprint24, 21 tests)

### Fixed
- OAuth build: `issueAccessToken` now accepts scope parameter across all call sites
- DPoP build: replaced custom ellipticCurve with `crypto/elliptic`, use `sha256.Sum256`
- Removed broken `coverage_sprint24_test.go` (stale API references from concurrent edits)
- Gateway `ClientIP()` now strips port from `RemoteAddr` (CIDR matching was failing)
- Register handler: duplicate email now returns 409 Conflict (was 500)
- Gateway tenant forwarding: injects `tenant_id` as both query param AND JSON body field
- Audit route alias: `/api/v1/audit` now works alongside `/api/v1/audit/events`
- Auth rate limiting: `429` after ~5 failed login attempts (working as designed)

### Changed
- SDK test coverage improved: 65.8% → 71.4% (put + RequirePermission tests)
- Docker Compose: 12 containers all healthy, 11/11 E2E tests pass
- NATS: added `-m 8222` flag for monitoring port (healthcheck)
- Policy/Org/Audit DB: use individual `DB_HOST`/`DB_PORT` env vars (not `DATABASE_URL`)
- Dockerfile: fixed EXPOSE ports (auth: 9001, policy: 8070/9070)
- Migrations: idempotent init container (skips if DB already has tables)

---

## [1.0.0] — 2024-07-10

### Overview

GGID v1.0 is the first stable release of the production-grade Identity and
Access Management platform. Built as 7 Go microservices with multi-tenant
PostgreSQL, Redis, and NATS JetStream infrastructure.

### Added

#### Microservices (7)

| Service | HTTP | gRPC | Responsibility |
|---------|------|------|---------------|
| **Gateway** | :8080 | — | JWT verification (RS256+JWKS), reverse proxy, rate limiting, CORS, circuit breaker, compression, OTel tracing, graceful shutdown |
| **Identity** | :8081 | :50051 | User CRUD, lock/unlock/activate, CSV import, SCIM 2.0 provisioning |
| **Auth** | :9001 | — | Register, login, JWT issuance, refresh token rotation, MFA (TOTP, Email OTP, WebAuthn), password reset, magic link, LDAP/AD |
| **OAuth** | :9005 | — | OAuth2/OIDC, JWKS, SAML 2.0 SP, OIDC discovery, token exchange (RFC 8693), consent screen |
| **Policy** | :8070 | :9070 | RBAC + ABAC engine, roles, permissions, policy check, policy import/export, attribute mapping |
| **Org** | :8071 | :9071 | Organizations, org tree (LTREE), departments, teams, memberships |
| **Audit** | :8072 | :9072 | Audit event query, statistics, CSV export, SSE streaming, retention, anomaly rules |

#### Authentication Methods (5+)

- **Password login** — Argon2id hashing, password policy enforcement
- **MFA: TOTP** — RFC 6238, Google Authenticator compatible
- **MFA: Email OTP** — One-time passwords via email
- **MFA: WebAuthn/Passkey** — FIDO2 attestation + assertion flows
- **Passwordless: Magic Link** — Email-based passwordless login
- **LDAP/AD** — Auth provider chain (Local + LDAP), auto-provision support
- **Social Login** — Google, GitHub, Discord, LinkedIn, Slack, Microsoft, GitLab
- **Generic OIDC** — Any OpenID Connect compliant IdP
- **SAML 2.0** — Service Provider with metadata exchange
- **IdP Federation** — External identity provider configuration

#### Authorization

- **RBAC** — Role hierarchy with parent/child inheritance
- **ABAC** — Attribute-based policy engine with conditions
- **Policy check** — Real-time permission evaluation (RBAC + ABAC combined)
- **ABAC evaluate** — Standalone attribute evaluation endpoint
- **Policy import/export** — JSON-based bulk policy management
- **SCIM 2.0** — `/scim/v2/Users` with filtering and pagination

#### Gateway Features

- JWT verification (RS256) with JWKS caching and key rotation
- Per-IP rate limiting (login: 5/min, register: 3/min, API: 100/min)
- Per-tenant rate limiting (configurable via REST API)
- CORS with configurable origins and preflight caching
- API key authentication (M2M)
- gRPC-Web protocol translation
- GraphQL query engine (fragments + variables)
- WebSocket proxy with session registry
- Request coalescing
- Shadow traffic support (`X-Shadow-Backend` header)
- Canary deployment routing
- Circuit breaker (per-backend)
- Response compression (gzip)
- Custom error pages (502/503/504 with request_id)
- Per-route body size limits
- Bot detection
- IP allowlist
- Session management middleware
- Hosted Universal Login pages (`/login`, `/register`, `/forgot-password`)
- Swagger UI + OpenAPI spec
- Prometheus histogram metrics per API
- OpenTelemetry tracing (W3C traceparent, OTLP HTTP exporter)
- Health check aggregation (version + uptime)
- Graceful shutdown (30s in-flight drain)

#### Infrastructure

- **PostgreSQL 16** with Row-Level Security (RLS) for multi-tenant isolation
- **Redis 7** for rate limiting, session cache, token blocklist, password reset tokens
- **NATS JetStream** for durable audit event pipeline (at-least-once delivery)
- **OpenLDAP** for directory integration testing

#### DevOps & Deployment

- **Docker Compose** — Full stack (13 containers), idempotent migrations, health checks
- **Kubernetes Helm Chart** — Deployments, Services, Ingress, HPA, PDB, NetworkPolicy, Secrets, ConfigMap
- **k6 Benchmark Suite** — 3 load testing scripts (login, API, mixed workload)
- **Prometheus Alert Rules** — 7 alert rules (high latency, error rate, backend down, etc.)
- **Grafana Dashboard** — Provisioned datasource + dashboard JSON
- **CI Pipeline** — govulncheck, Trivy container scanning, Helm lint
- **Docker E2E Test** — 11/11 tests (health, register, login, JWT, CRUD, audit, errors)

#### Admin Console (Next.js 15)

- Dashboard with service health and metrics
- User management (list, create, edit, lock/unlock, activate/deactivate, CSV import)
- Role management (create, permissions, hierarchy)
- Organization management (tree view, departments, teams, members)
- Audit log viewer with 5-dimension filtering
- OAuth client management
- Personal profile page (Profile/Security/Sessions tabs)
- Monitoring page
- Settings page (SMTP, branding, password policy, security)
- Login page with social icons, remember me
- Webhooks management page
- OAuth clients page

#### SDK (4 Languages)

- **Go** (`sdk/go/`) — Client API, JWT verification middleware, permission/role/scope checks
- **Node.js** (`sdk/node/`) — TypeScript, Express/Hono middleware, JWKS via jose
- **Java** (`sdk/java/`) — GGIDClient, Spring Security filter (`GGIDAuthFilter`)
- **Python** (`sdk/python/`) — JWT verification, Flask middleware, client API

#### Documentation (12 documents)

| Document | Description |
|----------|-------------|
| `docs/openapi.yaml` | OpenAPI 3.1 specification (all REST endpoints) |
| `docs/quick-start.md` | 5-minute guide from zero to authenticated API call |
| `docs/integration-guide.md` | Third-party developer integration guide |
| `docs/deployment.md` | Production deployment (Docker, K8s, TLS, backup, security) |
| `docs/security-hardening.md` | Security checklist and hardening guide |
| `docs/performance.md` | Performance tuning (DB, Redis, NATS, Go GC, pprof) |
| `docs/migration-guide.md` | Auth0/Keycloak → GGID migration |
| `docs/api-examples.md` | curl examples for every endpoint |
| `docs/troubleshooting.md` | FAQ-style troubleshooting guide |
| `docs/developer-guide.md` | Contributor guide (structure, testing, PR workflow) |
| `docs/console-guide.md` | Admin Console user manual |
| `docs/postman-collection.json` | Postman collection (64 requests, 9 folders) |
| `docs/adr/` | 5 Architecture Decision Records |

#### Shared Packages (`pkg/`)

| Package | Coverage | Description |
|---------|----------|-------------|
| `errors` | 100% | GGIDError type with codes + HTTP status mapping |
| `tenant` | 100% | Multi-tenant context propagation |
| `crypto` | 81.8% | Argon2id password hashing, AES-256-GCM encryption |
| `authprovider` | 88.1% | Auth provider chain: Local + LDAP |
| `audit` | — | NATS JetStream event publisher |

### Security

- Argon2id for password hashing (memory-hard, side-channel resistant)
- RS256 JWT signing (RSA 2048-bit)
- PostgreSQL RLS enforced per-tenant (defense in depth)
- Rate limiting on auth endpoints (brute-force protection)
- Password policy enforcement (complexity + history)
- Token blocklist via Redis (revocation support)
- Account lockout after failed login threshold
- CORS with configurable origin whitelist

### Changed

- Gateway injects `tenant_id` as both query param (GET) and JSON body field (POST/PUT/PATCH)
- Auth handler reads `username` field as credential identifier (not `email`)
- Policy roles require unique non-empty `key` field (`UNIQUE(tenant_id, key)`)
- PostgreSQL RLS enforced per-tenant via `SET LOCAL app.tenant_id`
- NATS JetStream monitoring port enabled (`-m 8222`)
- pgx v5 for PostgreSQL access (transaction-scoped settings for RLS)

### Fixed

- Register duplicate email/username returns 409 Conflict (was 500)
- Audit route alias added (`/api/v1/audit` in addition to `/api/v1/audit/events`)
- SCIM duplicate `writeSCIMError` declaration resolved
- Gateway coverage test stale references cleaned up
- Console `HOSTNAME=0.0.0.0` for Docker port binding
- Password service syntax error in `CheckHistory` function
- Docker Compose DB env vars: Policy/Org/Audit use `DB_HOST`/`DB_PORT` (not `DATABASE_URL`)
- Auth Dockerfile `EXPOSE 9001` (was 8082)
- Policy Dockerfile `EXPOSE 8070 9070` (was 8084 50054)

### Test Coverage

| Package | Coverage |
|---------|----------|
| `pkg/errors` | 100.0% |
| `pkg/tenant` | 100.0% |
| `audit/service` | 100.0% |
| `gateway/healthcheck` | 95.5% |
| `policy/service` | 93.9% |
| `auth/domain` | 92.9% |
| `pkg/i18n` | 90.2% |
| `pkg/notification` | 91.5% |
| `pkg/authprovider` | 88.1% |
| `org/service` | 87.3% |
| `gateway/middleware` | 84.0% |
| `gateway/router` | 82.5% |
| `gateway/webhooks` | 81.6% |
| `gateway/config` | 82.9% |
| `pkg/crypto` | 81.8% |
| `audit/handler` | 83.3% |
| `audit/server` | 71.8% |
| `pkg/saml` | 70.6% |
| `identity/service` | 72.3% |
| `oauth/service` | 72.0% |
| `auth/service` | 72.2% |

**Total: 15+ packages, 250+ test cases, 0 FAIL**

### E2E Test Results (Docker Compose)

| # | Test | Status |
|---|------|--------|
| 1 | Gateway healthz | PASS (200) |
| 2 | Register user | PASS (201) |
| 3 | Login + JWT | PASS (693 chars) |
| 4 | 401 without JWT | PASS |
| 5 | List users | PASS (200) |
| 6 | Create role | PASS (201) |
| 7 | List roles | PASS (200) |
| 8 | Create org | PASS (201) |
| 9 | Audit query | PASS (200) |
| 10 | Wrong password | PASS (401) |
| 11 | Duplicate register | PASS (409) |

### Docker Images

| Image | Size |
|-------|------|
| deploy-identity | 31.8 MB |
| deploy-auth | 27.4 MB |
| deploy-gateway | 18.3 MB |
| deploy-policy | 34.3 MB |
| deploy-org | 34.3 MB |
| deploy-audit | 34.2 MB |
| deploy-oauth | 23.6 MB |
| deploy-console | 212 MB |

---

## [0.1.0] — Phase 1-8 Initial Release

- 7 microservices (Go 1.25, gRPC + REST)
- Multi-tenant PostgreSQL 16 with Row-Level Security
- RBAC + ABAC policy engine with REST API + gRPC
- Organization tree with multi-tenant isolation (LTREE)
- Auth: register/login/JWT/refresh/MFA TOTP
- Audit: NATS JetStream consumer + REST query
- Admin Console (Next.js 15, 7 pages)
- Docker Compose containerization (13 services)
- Go SDK + Node.js SDK
- Integration tests via Gateway REST API
- E2E verified: register → login → JWT → CRUD → 401
