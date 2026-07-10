# Changelog

All notable changes to GGID IAM Platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

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
