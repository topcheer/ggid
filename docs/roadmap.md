# GGID Roadmap

Development phases, completed features, and future plans for the GGID IAM Platform.

---

## Completed Phases

### Phase 1: Foundation (Done)

- Go 1.25 monorepo structure
- 7 microservice scaffolds (`cmd/` → `internal/{config,domain,service,handler,server}`)
- Shared packages: `pkg/errors`, `pkg/tenant`, `pkg/crypto`, `pkg/authprovider`
- PostgreSQL 16 with pgx v5
- gRPC + REST dual-protocol services
- Proto definitions for 6 services (`buf generate`)

### Phase 2: Identity & Authentication (Done)

- User registration and login (Argon2id password hashing)
- JWT issuance (RS256) with refresh token rotation
- JWKS endpoint (`/.well-known/jwks.json`)
- User CRUD: create, get, update, delete
- Account lifecycle: lock/unlock, activate/deactivate
- Password reset (forgot → token → reset)
- Password policy enforcement (complexity + history)

### Phase 3: Authorization Engine (Done)

- RBAC engine: roles, permissions, role hierarchy (parent/child inheritance)
- Wildcard resource matching (`documents:*`)
- Policy check API (`POST /api/v1/policies/check`)
- ABAC engine: attribute-based conditions (JSON)
- Policy CRUD: create, list, delete
- Policy import/export (JSON)
- Attribute mapping API

### Phase 4: Organization Management (Done)

- Organization CRUD
- Org tree (PostgreSQL LTREE for hierarchical queries)
- Departments within organizations
- Teams (cross-cutting groups)
- Membership management (add/remove members with titles)
- Org-level role scoping

### Phase 5: Audit Pipeline (Done)

- NATS JetStream event bus (durable stream, file-backed)
- `pkg/audit.Publisher` (best-effort, non-blocking)
- Audit service: JetStream durable consumer → PostgreSQL
- Query API with filtering (action, actor, result, time range, resource)
- Audit statistics (aggregated metrics)
- CSV export
- Server-Sent Events (SSE) real-time streaming
- Retention configuration API
- Anomaly detection rules API
- Audit integrity verification (hash chain)

### Phase 6: Multi-Tenancy & SSO (Done)

- PostgreSQL Row-Level Security (RLS) on all multi-tenant tables
- `SET LOCAL app.tenant_id` per-transaction context
- OAuth2/OIDC provider (authorization code, client credentials, refresh)
- SAML 2.0 Service Provider
- OIDC discovery document
- SCIM 2.0 user provisioning (`/scim/v2/Users`)
- LDAP/AD integration (auth provider chain: Local + LDAP)

### Phase 7: API Gateway (Done)

- JWT verification (RS256) with JWKS caching
- Reverse proxy with route table
- Per-IP rate limiting (login: 5/min, register: 3/min, API: 100/min)
- CORS with configurable origins
- Tenant context injection (query param + JSON body)
- Health check aggregation
- Graceful shutdown (30s in-flight drain)
- Prometheus metrics (`/metrics`)

### Phase 8: Docker Compose & E2E (Done)

- Full-stack Docker Compose (13 containers)
- Idempotent database migrations
- Docker healthchecks for all services
- RSA key generation init container
- E2E test suite: 11/11 tests passing
- Deploy scripts

### Phase 9: Advanced Features (Done)

- MFA: TOTP (RFC 6238), Email OTP, WebAuthn/Passkey (FIDO2)
- Passwordless: Magic link authentication
- Social login: Google, GitHub, Discord, LinkedIn, Slack, Microsoft, GitLab
- Generic OIDC IdP federation
- IdP configuration API (`/api/v1/idp/config`)
- Step-up authentication (challenge + verify)
- Session management (list, revoke, logout-all)
- Auth hooks engine (pre-registration, post-login, pre-token-issue)
- Email change flow (request + confirm)
- Phone OTP authentication
- Passwordless registration (WebAuthn-only accounts)

### Phase 10: Gateway Advanced (Done)

- Circuit breaker (per-backend, configurable thresholds)
- Response compression (gzip)
- API key authentication (M2M)
- gRPC-Web protocol translation
- GraphQL query engine (fragments + variables)
- WebSocket proxy with session registry
- Request coalescing (singleflight for identical GETs)
- Shadow traffic (`X-Shadow-Backend` header)
- Canary deployment routing
- Custom error pages (502/503/504 with request_id)
- Per-route body size limits
- Bot detection
- IP allowlist
- Hosted Universal Login pages (`/login`, `/register`, `/forgot-password`)
- Swagger UI + OpenAPI spec serving
- Prometheus histogram per API
- OpenTelemetry tracing (W3C traceparent, OTLP HTTP exporter)
- Versioned health check (includes version + uptime)
- Slow request detection middleware

### Phase 11: Production Hardening (Done)

- Security hardening guide (TLS, key rotation, CORS, rate limiting)
- Performance tuning guide (DB indexing, connection pools, pprof)
- Migration guide (Auth0/Keycloak → GGID)
- OWASP Top 10 security audit checklist
- k6 benchmark suite (3 load test scripts)
- Grafana dashboard (provisioned)
- Prometheus alert rules (7 alerts)
- Helm chart (Deployments, Services, Ingress, HPA, PDB, NetworkPolicy)
- govulncheck in CI
- Trivy container scanning
- Comprehensive documentation suite (20+ documents)
- SDK documentation (Go, Node.js, Java, Python)
- Admin Console (10 pages: Dashboard, Users, Roles, Orgs, Audit, Settings, Monitoring, OAuth Clients, Webhooks, Profile)

---

## Future Phases

### Phase 12: Plugin System (Planned — Q3 2024)

- **Go plugin SDK** — Compile-time plugins for service-level extension
- **gRPC plugin sidecar** — Out-of-process plugins via gRPC
- **Plugin marketplace** — Community plugin registry
- **Hot-reloadable hooks** — Update auth hooks without restart
- **Plugin lifecycle management** — Install, enable, disable, uninstall via API

### Phase 13: Multi-Region Deployment (Planned — Q4 2024)

- **Active-active multi-region** — Cross-region PostgreSQL replication (Citus/ Patroni)
- **Geo-distributed Gateway** — DNS-based routing to nearest region
- **Cross-region audit sync** — NATS super-cluster for global audit pipeline
- **Region-aware rate limiting** — Distributed rate limit via Redis cluster
- **Disaster recovery** — Automated failover with RTO < 5 min, RPO < 1 min

### Phase 14: Passwordless Everywhere (Planned — Q4 2024)

- **FIDO2 Enterprise Attestation** — Verify device attestation certificates
- **Device trust** — Register trusted devices, skip MFA for known devices
- **Adaptive authentication** — Risk-based step-up (location, device, behavior)
- **Biometric login** — Platform authenticator integration (Touch ID, Face ID, Windows Hello)
- **Passwordless by default** — Option to disable password login entirely
- **Recovery codes** — Backup codes for passwordless account recovery

### Phase 15: B2B Federation (Planned — Q1 2025)

- **Cross-tenant SSO** — Users from Tenant A can access Tenant B via federation
- **Organizations API** — B2B customer organization management (like Auth0 Organizations)
- **Tenant invitation flow** — Invite users to join a tenant
- **Delegated admin** — Tenant-scoped admin roles (without platform admin)
- **Connection templates** — Pre-configured SSO templates for common IdPs (Okta, Azure AD, Google Workspace)
- **Just-in-time (JIT) provisioning** — Auto-create users on first SSO login

### Phase 16: Fine-Grained Authorization GA (Planned — Q1 2025)

- **Relationship-Based Access Control (ReBAC)** — Google Zanzibar-style tuple-based authorization
- **Resource-level permissions** — Per-object ACLs (e.g., "user X can edit document Y")
- **Policy versioning GA** — Full version management with rollback and diff
- **Policy templates marketplace** — Pre-built compliance templates (SOC2, HIPAA, PCI)
- **Visual policy builder** — Drag-and-drop policy editor in Console
- **Policy simulation** — "What-if" testing before deploying policies
- **Real-time policy sync** — Push policy updates to all Gateway instances via NATS

### Phase 17+: Future Considerations

- **Hosted SaaS** — Fully managed GGID cloud
- **Mobile SDK** — iOS (Swift) and Android (Kotlin) SDKs
- **Privacy vault** — Tokenization for PII fields (e.g., SSN, credit card)
- **Zero-knowledge proofs** — Privacy-preserving authentication
- **Decentralized identity** — DID/VC support, verifiable credentials
- **AI-powered anomaly detection** — ML-based threat detection in audit pipeline

---

## Release Cadence

| Release | Focus | Target |
|---------|-------|--------|
| v1.0 | Production-ready core (Phase 1-11) | Done |
| v1.1 | Plugin system + bug fixes | Q3 2024 |
| v1.2 | Multi-region + adaptive auth | Q4 2024 |
| v2.0 | B2B federation + ReBAC | Q1 2025 |

---

## Community

- **Feature requests:** Open an issue with `feature-request` label
- **Bug reports:** Open an issue with `bug` label
- **Discussions:** GitHub Discussions
- **Contributing:** See [Contributing Quick Start](./contributing-quickstart.md)
