# Changelog

All notable changes to GGID IAM Platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.2.0] — HSM/KMS Key Provider Release

### Added
- `pkg/crypto.KeyProvider` abstraction supporting local PEM, PKCS#11 HSM, AWS KMS, GCP KMS, Azure Key Vault, and HashiCorp Vault Transit.
- PKCS#11 KeyProvider implementation with SoftHSM2 integration tests (build tag `pkcs11`).
- Auth, OAuth, and Gateway services now initialize `crypto.NewKeyProvider` at startup via `GGID_KEY_PROVIDER` env var.
- JWKS endpoint dynamically serves the public key from the active KeyProvider.
- SoftHSM2 development environment under `deploy/softhsm2/`.
- `GGID_KEY_PROVIDER`, `GGID_PKCS11_LIB`, `GGID_PKCS11_SLOT`, `GGID_PKCS11_PIN`, `GGID_PKCS11_KEY_LABEL` configuration env vars.

### Changed
- `TokenService` and OAuth server now accept a `crypto.KeyProvider` instead of loading PEM files directly.
- Local RSA key auto-generation is now handled by the local KeyProvider fallback.

### Fixed
- Removed duplicate stub provider functions in `pkg/crypto`.
- Updated `TestTokenService_KeyFilesCreated` to match KeyProvider model (public key derived from private key).

---

## [0.2.1] — E2E Stability, SDK Alignment, and GeoIP Verification

### Added
- Agent Identity and Access Request (IGA) methods to Python, Java, Rust, Ruby, C#, Dart, and PHP SDKs.
- GeoIP middleware regression tests for private IP handling, country block/allow lists, and upstream header passthrough.

### Fixed
- CI lint failure: removed unused  function in .
- Docker Compose migrate container command syntax ( duplication) that prevented E2E tests from starting.
- Duplicate entry for SDK alignment gap (#20) in platform-completeness-report.md.

### Changed
- : all productization gaps now [DONE].
- : synchronized counts (Total 20 / Done 21 / Partial 0 / Remaining 0).

## [0.2.2] — Route Wiring + Release Workflow Fix

### Fixed
- Gateway route wiring: add , , and  routes to gateway config.
- Release workflow  job: run npm install/build in  instead of repo root.

### Changed
- docs/platform-completeness-report.md: finding #21 [DONE]; counts synchronized.
- docs/platform-scan-state.md: round 22, next focus E2E regression.

## [0.2.3] — CI Release Workflow Fix

### Fixed
- Release workflow  job now runs npm install/build in  with fallback from  to .
- Bump Node SDK version to 1.0.2 to avoid npm publish conflict.

### Changed
- docs/platform-scan-state.md: Round 22 E2E 11/11 PASS; advance to Round 23 Focus C (Middleware).

## [Unreleased]




### Added — Phase 9-10 Features

#### Authentication & Authorization
- Social login connectors: Google, GitHub, Discord, LinkedIn, Slack, Microsoft, GitLab
- Generic OIDC connector for any compliant IdP
- WebAuthn / Passkey skeleton (attestation + assertion flows)
- MFA: TOTP (RFC 6238), Email OTP, WebAuthn
- LDAP/AD integration (auth provider chain: Local + LDAP)
- SAML 2.0 Service Provider skeleton
- Auth hooks engine (pre-register, post-login webhook callbacks)
- IdP federation (OIDC-based external identity providers)
- Passwordless authentication (magic link + OTP)
- RFC 8693 Token Exchange (OAuth service)
- OAuth2 consent screen endpoint
- Account lockout + email lockout policies

#### Console (Admin UI)
- Dashboard with metrics cards
- User management (list, create, edit, lock/unlock)
- User CSV import/export
- Role management (create, list, assign)
- Organization management (tree view, departments, teams)
- Audit log viewer with advanced filtering (5 dimensions)
- OAuth client management
- Personal profile page (Profile/Security/Sessions tabs)
- Login page with social icons, remember me
- Monitoring page
- Settings page

#### Gateway
- JWT verification (RS256 + JWKS rotation)
- Per-tenant rate limiting (configurable via REST API)
- API key authentication (M2M)
- gRPC-Web protocol translation
- GraphQL query engine with fragments + variables
- Custom error pages (502/503/504 with request_id)
- CORS preflight cache (5min)
- Per-route body size limits
- Shadow traffic support (X-Shadow-Backend header)
- Request coalescing
- Session management middleware
- Response compression (gzip)
- Prometheus histogram metrics per API
- OpenTelemetry tracing (W3C traceparent)
- Health check aggregation (version + uptime)
- Graceful shutdown (30s in-flight drain)
- Hosted Universal Login pages (/login, /register, /forgot-password)
- Swagger UI + OpenAPI spec

#### Infrastructure & Deployment
- Docker Compose full stack (13 containers)
- Helm chart (deployments, services, ingress, HPA, PDB, NetworkPolicy, secrets, configmap)
- k6 performance benchmark suite (3 scripts)
- Prometheus alert rules (7 alerts)
- Grafana datasource provisioning
- CI: govulncheck + Trivy security scanning + Helm lint
- Docker E2E test script (11/11 tests pass)

#### Documentation
- C4 architecture diagrams (system context, container, component views)
- Security whitepaper (STRIDE threat model)
- Migration guide (Auth0/Keycloak/Clerk → GGID)
- Production hardening guide
- Plugin system design (webhook hooks, Go plugins, gRPC sidecar)
- Quick start guide (5-minute integration)
- Feature matrix (157 features × 10 platforms)
- Development roadmap (Phase 9-12)
- Contributing guide
- Team backlog (77 tasks)

#### SDK
- Go SDK (client, JWT verification, HTTP middleware)
- Node.js SDK (TypeScript, Express/Hono middleware)
- Java SDK (GGIDClient, Spring Security filter)
- Python SDK (jwt, client, Flask middleware)

### Changed
- Gateway injects tenant_id as both query param and JSON body field
- Auth handler reads `username` field as credential identifier
- Policy roles require unique `key` field (UNIQUE constraint)
- PostgreSQL RLS enforced per-tenant
- NATS JetStream monitoring port enabled (-m 8222)

### Fixed
- Register duplicate email returns 409 Conflict (was 500)
- Audit route alias (/api/v1/audit in addition to /api/v1/audit/events)
- SCIM duplicate writeSCIMError declaration
- Gateway stale coverage_final_test.go references
- Console HOSTNAME=0.0.0.0 for Docker port mapping

---

## [0.1.0] — Phase 1-8 Initial Release

- 7 microservices (Go 1.25, gRPC + REST)
- Multi-tenant PostgreSQL 16 with Row-Level Security
- RBAC + ABAC policy engine with REST API + gRPC
- Organization tree with multi-tenant isolation
- Auth: register/login/JWT/refresh/MFA TOTP
- Audit: NATS JetStream + REST query
- Admin Console (Next.js 15)
- Docker Compose containerization
