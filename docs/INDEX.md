# GGID Documentation Index

> Complete catalog of all GGID documentation. 362 docs organized by category.

---

## Quickstart (9 docs)

| Document | Description |
|----------|-------------|
| [5-Minute JWT](quickstart/5-minute-jwt.md) | Register, login, get JWT, call protected API — in 5 minutes |
| [Docker 5-Minute](quickstart/docker-5-min.md) | Zero to authenticated API call with Docker Compose |
| [RBAC Permissions](quickstart/rbac-permissions.md) | Create roles, assign permissions, check access in 5 minutes |
| [OAuth Login](quickstart/oauth-login.md) | OAuth 2.1 authorization code flow with PKCE end-to-end |
| [Go SDK](quickstart/go-sdk.md) | Add GGID auth to any Go app — JWT verify + middleware |
| [Node.js SDK](quickstart/node-sdk.md) | Add GGID auth to any Node.js app — express middleware |
| [SDK Quickstart](quickstart/sdk-quickstart.md) | All 4 SDKs side-by-side: Go, Node.js, Python, Java |
| [K3s Deploy](quickstart/k3s-deploy.md) | Get GGID running on K3s in under 10 minutes |
| [Developer Onboarding](quickstart/developer-onboarding.md) | 8-step onboarding: clone → build → Docker → first API call |

---

## Guides (8 docs)

| Document | Description |
|----------|-------------|
| [RBAC Guide](guides/role-based-access.md) | Complete RBAC: roles, permissions, hierarchy, policy check |
| [ABAC Policy](guides/abac-policy.md) | ABAC policies: syntax, evaluate, dry-run, compliance templates |
| [SDK Migration Guide](guides/sdk-migration-guide.md) | Auth0/Keycloak/Firebase to GGID API mapping |
| [Troubleshooting](guides/troubleshooting.md) | Common issues: JWT, DB, NATS, Gateway 502, tenant isolation |
| [Social Login Setup](guides/social-login-setup.md) | GitHub/Google/Microsoft/LDAP/OIDC provider configuration |
| [Webhook Setup](guides/webhook-setup.md) | Register webhooks, verify signatures, retry, idempotency |
| [Custom Claims](guides/custom-claims.md) | Standard claims, auth hooks, SDK reading, ABAC usage |
| [Multi-Tenant Setup](guides/multi-tenant-setup.md) | Tenant CRUD, RLS policies, per-tenant config |
| [Security Hardening](guides/security-hardening.md) | Production security checklist, SSRF, CSRF, rate limiting |
| [Frontend i18n](guides/frontend-i18n.md) | next-intl config, message key convention, LanguageSwitcher |
| [i18n Setup](guides/i18n-setup.md) | Backend i18n: pkg/i18n translator, string extraction |

---

## Integration Guides (3 docs)

| Document | Description |
|----------|-------------|
| [Express.js](integration-guides/express.md) | JWT verification middleware, role/scope guards, GGIDClient |
| [Gin](integration-guides/gin.md) | Go Gin middleware adapter, role checks, tenant-aware queries |
| [Spring Boot](integration-guides/spring-boot.md) | Java servlet filter, Spring Security integration |
| [3-Line Integration](quickstart/3-line-integration.md) | Add JWT auth in 3 lines: Go, Node.js, Python, Java |
| [External DB Setup](quickstart/external-db-setup.md) | Connect external PostgreSQL/Redis/NATS/LDAP |
| [Helm 5-Minute](quickstart/helm-5-min.md) | Deploy GGID to Kubernetes with Helm in 5 minutes |

---

## Examples (2 docs)

| Document | Description |
|----------|-------------|
| [Express.js Integration Example](examples/express-integration.md) | Full runnable Express app: auth, CRUD, permission check |
| [Go Backend Integration Example](examples/go-integration.md) | Full runnable Go server: SDK middleware, RequirePermission |
| [Python FastAPI Example](examples/python-integration.md) | Full runnable FastAPI app: GGIDMiddleware, get_current_user |
| [Java Spring Boot Example](examples/java-spring-integration.md) | Spring Security config, GGIDSecurityFilter, REST controller |
| [Go Gin Example](examples/go-gin-integration.md) | Gin middleware adapter, role/scope guards, CRUD |

---

## Tutorials (4 docs)

| Document | Description |
|----------|-------------|
| [Multi-Tenant Setup](tutorials/multi-tenant-setup.md) | Create tenants, configure RLS, per-tenant branding |
| [Custom Auth Provider](tutorials/custom-auth-provider.md) | Implement the AuthProvider interface with a custom provider |
| [Webhook Integration](tutorials/webhook-integration.md) | End-to-end webhook delivery with signature verification |
| [SAML SP Configuration](tutorials/saml-sp-configuration.md) | Configure SAML 2.0 Service Provider with IdP metadata |

---

## Deploy (10 docs)

| Document | Description |
|----------|-------------|
| [Docker](deploy/docker.md) | Compose explained, env vars, health checks, E2E walkthrough |
| [Kubernetes](deploy/kubernetes.md) | Helm chart, values reference, ingress, cert-manager, HPA |
| [K3s](deploy/k3s.md) | Full K3s guide: OrbStack, Traefik, registry, troubleshooting |
| [Bare Metal](deploy/bare-metal.md) | systemd units, nginx reverse proxy, Let's Encrypt |
| [Environment Variables](deploy/environment-variables.md) | Complete reference for all 7 services |
| [Docker Compose Override](deploy/docker-compose-override.md) | 8 override use cases with examples |
| [Helm Chart Guide](deploy/helm-chart-guide.md) | Helm install, values reference, upgrade, rollback |
| [Helm Reference](deploy/helm-reference.md) | Complete Helm values reference |
| [Production Checklist](deploy/production-checklist.md) | Pre-production verification + external infra checklist |
| [Registry Setup](deploy/registry-setup.md) | Private Docker registry setup |
| [Troubleshooting](deploy/troubleshooting.md) | 40+ issues across 6 categories |

---

## Architecture & Design (12 docs)

| Document | Description |
|----------|-------------|
| [Overview](architecture/overview.md) | System diagram, service responsibilities, data flow |
| [Security Overview](architecture/security-overview.md) | Auth flow, P0 security, RLS, audit hash chain, STRIDE |
| [Data Flow](architecture/data-flow.md) | Request flow diagrams: register, login, JWT verify, audit pipeline |
| [Microservices](architecture/microservices.md) | 7-service architecture: ports, deps, communication patterns |
| [ADR-0001: JWT RSA Shared Key](architecture/decision-record/0001-jwt-rsa-shared-key.md) | Why RSA for JWT signing |
| [ADR-0002: Multi-Tenancy](architecture/decision-record/0002-multi-tenancy.md) | RLS-based tenant isolation decision |
| [ADR-001: Database Choice](design/adr-001-database-choice.md) | PostgreSQL + RLS for multi-tenant isolation |
| [ADR-002: Event-Driven Audit](design/adr-002-event-driven-audit.md) | NATS JetStream for audit pipeline |
| [ADR-003: Provider Chain](design/adr-003-provider-chain.md) | Chain-of-responsibility for auth providers |
| [Authentication Flow](design/authentication-flow.md) | Login, MFA, token lifecycle sequence diagrams |
| [Data Model](design/data-model.md) | Entity relationships, table schemas |
| [Gateway Architecture](design/gateway-architecture.md) | Reverse proxy, middleware chain, rate limiting |
| [Multi-Tenant RLS](design/multi-tenant-rls.md) | Row-Level Security design and policies |
| [Policy Engine](design/policy-engine.md) | RBAC + ABAC evaluation algorithm |
| [Zero-Trust Implementation](design/zero-trust-implementation.md) | Trust boundaries, defense in depth |

---

## API Reference (5 docs)

| Document | Description |
|----------|-------------|
| [API Reference](api-reference.md) | Complete endpoint reference: 78+ endpoints across 7 services |
| [Admin API](api/admin-api.md) | User/role/org/policy/audit admin endpoints |
| [SCIM 2.0 API](api/scim-api.md) | Enterprise SCIM endpoints for HR system integration |
| [API Error Codes](api-error-codes.md) | 57 error codes across all services |
| [API Error Codes (v2)](api/error-codes.md) | Error codes with HTTP status mapping |
| [API Conventions](api-conventions.md) | URL patterns, pagination, filtering, versioning |
| [API Examples](api-examples.md) | curl examples for every endpoint category |

---

## SDK Reference (4 docs)

| Document | Description |
|----------|-------------|
| [Go SDK README](../sdk/go/README.md) | Go SDK: client, middleware, user/role/org CRUD |
| [Node.js SDK README](../sdk/node/README.md) | Node SDK: GGIDClient, expressAuth, JWTVerifier |
| [Python SDK README](../sdk/python/README.md) | Python SDK: FastAPI/Flask/Django middleware |
| [Java SDK README](../sdk/java/README.md) | Java SDK: GGIDClient, servlet filter, Spring Security |

---

## Research (120+ docs)

### Competitive Analysis
- [Auth0/Keycloak/GGID Matrix](research/auth0-keycloak-ggid-matrix.md) — 31-gap competitive analysis
- [Gap Closure Report](research/gap-closure-report.md) — Verification of all 31 gaps (24 DONE, 3 PARTIAL, 4 TODO)
- [Auth0 Comparison](research/auth0-comparison.md), [Keycloak Comparison](research/keycloak-comparison.md), [Ory Comparison](research/ggid-vs-ory.md)
- [Competitor Updates](research/competitor-update-2025.md), [Competitive Update 2026-07](research/competitive-update-2026-07.md)
- [Casdoor/Clerk/Logto](research/competitor-update-clerk-logto-casdoor.md)

### Security Research
- [JWT Algorithm Confusion](research/jwt-algorithm-confusion.md), [JWT Claim Validation](research/jwt-claim-validation.md)
- [OAuth Security BCP](research/oauth-security-recommendations-bcp.md), [OAuth State CSRF](research/oauth-state-csrf.md)
- [CSRF in IAM](research/cross-site-request-forgery-iam.md), [Session Fixation](research/session-fixation-prevention.md)
- [Credential Stuffing](research/credential-stuffing-iam.md), [Credential Theft Defense](research/credential-theft-defense.md)
- [SQL Injection Defense](research/sql-injection-iam-defense.md), [Entropy Audit](research/entropy-audit-iam.md)
- [DPoP (RFC 9449)](research/dpop-rfc9449.md), [Token Binding](research/token-binding-and-dpop.md)
- [Post-Quantum Cryptography](research/post-quantum-cryptography-iam.md)
- [HSM/KMS Integration](research/hsm-kms-integration.md)

### OAuth/OIDC Deep Dives
- [OAuth 2.1 Analysis](research/oauth-2.1-analysis.md), [OAuth 2.1 Migration](research/oauth-2-1-migration-guide.md)
- [PKCE Deep Dive](research/oauth-pkce-deep-dive.md), [PAR and JAR](research/par-and-jar-analysis.md)
- [Device Flow (RFC 8628)](research/device-flow-rfc8628.md), [Token Exchange (RFC 8693)](research/token-exchange-rfc8693.md)
- [OIDC Discovery](research/oidc-discovery-security.md), [OIDC CIBA](research/oidc-ciba-security.md)
- [OIDC Back-Channel Logout](research/oidc-back-channel-logout-security.md), [OIDC Session Management](research/openid-connect-session-management.md)
- [Bearer Token Usage (RFC 6750)](research/bearer-token-usage-rfc6750.md), [Native App Patterns (RFC 8252)](research/oauth-native-app-patterns-rfc8252.md)

### WebAuthn/FIDO2
- [Passkey Best Practices](research/webauthn-passkey-best-practices.md)
- [WebAuthn Attestation Chain](research/webauthn-attestation-chain.md), [Attestation Verification](research/webauthn-attestation-verification.md)
- [Passkey Recovery](research/passkey-recovery.md), [Passkey Sync Security](research/passkey-sync-security.md)
- [FIDO Metadata Service](research/fido-metadata-service.md), [FIDO2 Certification](research/fido2-certification-guide.md)
- [WebAuthn Roadmap v2](research/webauthn-roadmap-v2.md)

### Architecture & Strategy
- [Zero Trust IAM](research/zero-trust-iam.md), [Zero Trust Patterns](research/zero-trust-iam-patterns.md)
- [Edge Computing IAM](research/edge-computing-iam.md), [Service Mesh IAM](research/service-mesh-iam.md)
- [Privacy-Enhancing Tech](research/privacy-enhancing-technologies.md)
- [AI Agent Identity / MCP](research/ai-agent-identity-mcp.md), [AI Agent Identity Analysis](research/ai-agent-identity-analysis.md)
- [IAM Differentiation Strategy](research/iam-differentiation-strategy.md)
- [Market Positioning](research/market-positioning-analysis.md)

### Compliance & Auditing
- [Audit Tampering Detection](research/audit-tampering-detection.md), Audit Log Compliance
- [SOC2 Compliance](research/compliance-soc2-iam.md), [Compliance Automation](research/compliance-automation.md)
- [SIEM Security Monitoring](research/siem-security-monitoring.md)
- [Wire Audit](research/wire-audit.md) — Code exists but not wired patterns

### Other Research
- [Rate Limiting Strategies](research/rate-limiting-strategies.md), [Bot Protection](research/bot-protection-analysis.md)
- [Multi-Tenant Isolation](research/multi-tenant-isolation.md), [Multi-Tenant SAML](research/multi-tenant-saml.md)
- [Key Rotation](research/key-rotation-iam.md), [JWKS Key Rotation](research/jwks-key-rotation.md)
- [Token Replay Defense](research/token-replay-defense.md), [Token Lifecycle Security](research/token-lifecycle-security.md)
- [SDK Ecosystem Gap Analysis](research/sdk-ecosystem-gap-analysis.md)
- [i18n Gap Analysis](research/i18n-gap-analysis.md), [i18n Wiring Estimate](research/i18n-wiring-estimate.md)
- [OpenAPI Audit](research/openapi-audit.md), [Quickstart Timing Comparison](research/quickstart-timing-comparison.md)

---

## Operations (8 docs)

| Document | Description |
|----------|-------------|
| [Operations Runbook](operations-runbook.md) | Deploy, tenants, key rotation, backups, monitoring, emergencies |
| [Performance Tuning](performance-tuning.md) | Benchmark results, pool sizing formulas, profiling |
| [Backup & Recovery](backup-recovery.md) | Backup strategies, restore procedures |
| [Logging Guide](logging-guide.md) | slog structured logging, log levels, collection |
| [Observability Guide](observability-guide.md) | Metrics, tracing, dashboards |
| [Network Security](network-security.md) | Segmentation, firewall, mTLS, DDoS |
| [Vulnerability Management](vulnerability-management.md) | Scanning, SLAs, P0 patch history |
| [Incident Response](incident-response.md) | 5 playbooks, forensic tools, communication plan |

---

## Migration (4 docs)

| Document | Description |
|----------|-------------|
| [Auth0 Migration](migration-from-auth0.md) | Step-by-step Auth0 to GGID migration |
| [Keycloak Migration](migration-from-keycloak.md) | Keycloak to GGID migration guide |
| [Clerk Migration](migration-from-clerk.md) | Clerk to GGID migration guide |
| [OAuth 2.1 Migration](oauth-2.1-migration-guide.md) | OAuth 2.0 to 2.1 upgrade path |

---

## Other Key Docs

| Document | Description |
|----------|-------------|
| [Getting Started](getting-started.md) | Expanded getting started with architecture overview |
| [CHANGELOG](CHANGELOG.md) | All notable changes by version |
| [CONTRIBUTING](contributing.md) | How to contribute — code style, PR process |
| [Feature Matrix](feature-matrix.md) | Feature comparison across IAM platforms |
| [Glossary](glossary.md) | IAM terminology reference |
| [FAQ](faq.md) | Frequently asked questions |
| [Roadmap](roadmap.md) | Future development plans |
| [Team Backlog](team-backlog.md) | Work tracking and status |
| [Tech Debt](tech-debt.md) | Known technical debt items |

---

*362 total documents. Last updated: 2025-07-11*
