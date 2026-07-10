# Changelog

All notable changes to GGID are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Social Login**: Google, GitHub, Discord, LinkedIn, Slack, GitLab connectors (`pkg/social/`)
- **Hosted Login**: Gateway serves `/login`, `/register`, `/forgot-password` pages
- **OAuth2**: Consent screen endpoint, RFC 8693 token exchange
- **Gateway**: gRPC-Web translation, per-tenant rate limiting, CORS preflight cache
- **Gateway**: Custom error pages (502/503/504), per-route body size limits
- **Gateway**: gRPC health check, HPA, PDB, NetworkPolicy (Helm)
- **Auth**: Hooks engine (pre/post login, pre-register), IdP federation
- **Auth**: Passwordless (email OTP), WebAuthn conditional mediation
- **Console**: Login page with social buttons + MFA flow
- **Console**: Dashboard with active sessions, login trends
- **Console**: User CSV export/import
- **Console**: Audit log advanced filtering (action/actor/result/date)
- **Console**: Personal profile page (Profile/Security/Sessions)
- **Console**: OAuth client management page
- **Helm Chart**: Full K8s deployment (deployments, services, ingress, secrets, HPA, PDB, NetworkPolicy)
- **k6 Benchmarks**: Register-login, API throughput, JWT verify burst scripts
- **Monitoring**: Prometheus alert rules, Grafana datasource provisioning
- **Documentation**: Architecture (C4 model), Security Whitepaper, Migration Guide, Production Hardening, Plugin System Design
- **SDKs**: Go, Node.js, Java, Python
- **CI/CD**: govulncheck, Trivy security scan, Helm chart lint
- **Docker**: `.dockerignore` for optimized build context

### Changed
- Gateway graceful shutdown: 30s drain timeout (was 10s)
- Middleware coverage: 81.0% (was 79%)
- Policy coverage: 93.3% (was 91.1%)
- OAuth coverage: 76.3% (was 67.9%)

### Fixed
- SCIM groups.go duplicate writeSCIMError declaration
- Gateway tenant_id forwarding for POST/PUT/PATCH (query param + JSON body injection)
- Auth register duplicate email returns 409 Conflict (was 500)
- Policy role creation requires unique key field
- Docker Compose: NATS monitoring port, DB env vars for Policy/Org/Audit
- Console: HOSTNAME=0.0.0.0 for Docker port mapping

## [0.1.0] — Phase 8

### Added
- 7 microservices: Gateway, Identity, Auth, OAuth, Policy, Org, Audit
- Admin Console (Next.js 15): Dashboard, Users, Roles, Orgs, Audit, Login, Settings
- gRPC + REST dual protocol for all services
- PostgreSQL 16 with Row-Level Security (multi-tenant isolation)
- Redis 7 for sessions, rate limiting, JWKS cache
- NATS JetStream for audit event streaming
- OpenLDAP for enterprise directory integration
- RBAC + ABAC policy engine with REST + gRPC APIs
- SCIM 2.0 skeleton (User + Group endpoints)
- Docker Compose: 13 containers, 11/11 E2E tests pass
- OpenAPI 3.0 spec + Swagger UI at /docs
- Apache 2.0 license
