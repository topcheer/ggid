# Feature Matrix: Auth0 vs Keycloak vs GGID

> **Purpose**: Side-by-side comparison of three IAM platforms across 10 categories.
> Last updated: 2025-06-17

---

## Quick Summary

| Attribute | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **Type** | Commercial SaaS (Okta) | Open-Source (Red Hat / CNCF) | Open-Source (Go monorepo) |
| **Language** | Node.js / proprietary | Java (Quarkus) | Go 1.25 |
| **License** | Proprietary (SaaS) | Apache 2.0 | Apache 2.0 |
| **Deployment** | Cloud-hosted primary; self-hosted via CIAM appliance (Enterprise) | Self-hosted primary; managed by Cloud-IAM etc. | Self-hosted (Docker Compose) |
| **Multi-service** | Monolithic platform | Monolithic server | 7 microservices (gateway, identity, auth, oauth, policy, org, audit) |
| **Cost Model** | Per-MAU pricing, scales to 5+ figures/month at scale | Free + hosting costs | Free + hosting costs |

---

## 1. Authentication Methods

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **Password** | Full — bcrypt/argon2, breach detection, password policies | Full — configurable password policies, hashing | Full — register/login with hashed passwords |
| **MFA — TOTP** | Full — Google Authenticator, Authy, push via Guardian | Full — TOTP via FreeOTP/Google Authenticator, configurable OTP policies | Full — TOTP MFA (RFC 6238) |
| **MFA — SMS** | Full — via Twilio integration (push notification preferred) | Partial — via custom SPI / SMS provider | **WARNING: Not implemented** |
| **MFA — Email OTP** | Full — email-based OTP codes | Partial — via custom flow | **WARNING: Not implemented** |
| **MFA — Push** | Full — Auth0 Guardian push notifications | No native push | **WARNING: Not implemented** |
| **WebAuthn / Passkey** | Full — platform authenticators (Face ID, Touch ID), security keys, passkey support | Full — WebAuthn registration & login (security keys + platform authenticators) | Full — WebAuthn registration + verification flow |
| **Social Login** | Full — 30+ providers (Google, GitHub, Microsoft, Apple, Facebook, Twitter/X, LinkedIn, etc.) | Full — configurable identity providers (OIDC, SAML, social providers) | Full — 9 connectors (Google, GitHub, Microsoft, Apple, Discord, Slack, LinkedIn, GitLab, generic OIDC) |
| **LDAP / Active Directory** | Full — AD/LDAP connector (Enterprise) | Full — built-in LDAP/AD federation, user federation SPI | Full — LDAP provider with auto-provision, START-TLS |
| **Magic Link** | Full — passwordless email links | Partial — via custom authenticator flow | **WARNING: Not implemented** |
| **Biometric** | Full — via WebAuthn platform authenticators (Face ID, Touch ID) | Full — via WebAuthn platform authenticators | Full — via WebAuthn platform authenticators |
| **Passwordless** | Full — phone, email, WebAuthn passwordless flows | Partial — configurable flows | **WARNING: Not implemented (password required)** |

---

## 2. Identity Provider (IdP) Protocols

| Protocol | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **OIDC (OpenID Connect)** | Full — certified OIDC OP, all flows | Full — certified OIDC provider & relying party | Full — OAuth service implements OIDC flows |
| **OAuth 2.0** | Full — authorization code, PKCE, client credentials, device code, ROPC (deprecated) | Full — all standard grant types, token exchange (RFC 8693) | Full — authorization code, PKCE, client credentials |
| **SAML 2.0** | Full — SAML IdP & SP (Enterprise plan) | Full — SAML 2.0 IdP & SP, certified | Full — SAML IdP implementation |
| **SCIM 2.0** | Full — SCIM user provisioning (Enterprise) | Partial — via third-party extension (scim-for-keycloak) | **WARNING: Skeleton only** — basic endpoints, no full CRUD/bulk/filter |
| **WS-Federation** | Full — via WS-Fed addon | Full — WS-Federation support | **WARNING: Not implemented** |
| **Token Exchange** | Full — RFC 8693 token exchange | Full — token exchange grant type | **WARNING: Not implemented** |
| **Device Authorization** | Full — device code flow | Full — device authorization grant | **WARNING: Not implemented** |

---

## 3. Session Management

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **JWT** | Full — RS256/HS256, configurable claims, customizable via Actions | Full — RS256/ES256/HS256, configurable token claims | Full — JWT with configurable claims |
| **Refresh Tokens** | Full — rotating refresh tokens, absolute/sliding idle timeouts | Full — refresh tokens with rotation, reuse detection | Full — refresh token rotation |
| **Session Revocation** | Full — global logout, session revocation, token blacklist | Full — revoke sessions, backchannel logout (OIDC), token revocation endpoint | Partial — session revocation via audit/JWT blacklist. **WARNING: No backchannel logout** |
| **Sliding Sessions** | Full — configurable idle/absolute session timeouts | Full — configurable SSO session idle/timeout | Partial — token refresh extends session. **WARNING: No configurable idle timeout per-tenant** |
| **Concurrent Session Limits** | Full — per-user session limits (Enterprise) | Partial — via custom event listener / SPI | **WARNING: Not implemented** — no concurrent session limiting |
| **Single Logout (SLO)** | Full — front-channel & back-channel SLO | Full — front-channel & back-channel SLO | **WARNING: Not implemented** |
| **Token Introspection** | Full — RFC 7662 introspection endpoint | Full — token introspection endpoint | **WARNING: Not implemented** |
| **Session Management UI** | Full — admin dashboard with active sessions | Full — admin console session view | Partial — console sessions page (revoke individual sessions). Active session listing available |

---

## 4. Multi-Tenancy

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **Tenant Isolation Model** | Full — Organizations (B2B), separate tenants with shared/custom connections | Full — Realm-based isolation (each realm = independent tenant with own users/clients/IdPs) | Full — tenant_id-based isolation with PostgreSQL Row-Level Security (RLS) |
| **Tenant-Aware Routing** | Full — custom domains per organization, organization-specific login URLs | Full — per-realm URLs, multi-tenant SaaS via realm-per-tenant | Full — Gateway routes by tenant_id header (X-Tenant-ID) |
| **Per-Tenant Config** | Full — per-organization branding, connections, roles, MFA policies | Full — per-realm: themes, login flows, IdPs, roles, SMTP | **WARNING: Partial** — per-tenant auth provider chain (Local+LDAP), but no per-tenant branding or custom login flows |
| **Row-Level Security (RLS)** | N/A (cloud-managed) | N/A (realm = separate DB or schema) | Full — PostgreSQL 16 RLS policies enforced at database level |
| **Per-Tenant IdP** | Full — each organization can have its own SAML/OIDC connections | Full — each realm has independent IdP configurations | **WARNING: Partial** — LDAP provider configurable globally; per-tenant social/OIDC IdP not yet configurable |
| **Tenant Provisioning API** | Full — Management API for org CRUD | Full — admin REST API for realm CRUD | **WARNING: Not implemented** — no tenant management API |
| **Custom Domains** | Full — per-tenant custom domains (Enterprise) | Full — per-realm custom theme/URL | **WARNING: Not implemented** |

---

## 5. SCIM 2.0

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **User Provisioning** | Full — create/read/update/delete users via SCIM endpoints | Partial — via scim-for-keycloak extension | **WARNING: Skeleton** — user endpoints exist but limited CRUD |
| **Group Sync** | Full — group/group membership management | Partial — via extension | **WARNING: Skeleton** — groups endpoint exists but no full sync |
| **Bulk Operations** | Full — SCIM bulk endpoint (POST /Bulk) | Partial — via extension | **WARNING: Not implemented** |
| **Filter Support** | Full — SCIM filter (eq, co, sw, pr, AND, OR) | Partial — via extension | **WARNING: Not implemented** — no SCIM filter parsing |
| **PATCH Operations** | Full — SCIM PATCH for partial updates | Partial — via extension | **WARNING: Not implemented** |
| **Pagination** | Full — startIndex/count pagination | Partial — via extension | **WARNING: Not implemented** |
| **Enterprise User Schema** | Full — enterprise user extensions | Partial — via extension | **WARNING: Not implemented** |

> **Note**: Keycloak's SCIM support requires the third-party `scim-for-keycloak` extension, which is community-maintained and not part of the core distribution.

---

## 6. Audit & Compliance

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **Event Logging** | Full — all auth events logged (login, MFA, password changes, API calls) | Full — event listener SPI, admin events, user events | Full — all auth/CRUD events published to audit service |
| **Queue-Based Audit** | Full — via Actions webhooks / Log Streams (Datadog, Splunk, AWS, HTTP) | Partial — event listener SPI (JPA, JMS, custom) | Full — NATS JetStream event streaming (async, durable) |
| **Audit Query API** | Full — Log Management API (search, filter, export) | Full — admin REST API for events | Full — REST query API (GET /api/v1/audit/events) with filtering |
| **Compliance Reporting** | Full — compliance dashboards, SOC 2, HIPAA BAA (Enterprise) | Partial — events stored in DB, custom reporting via JPA listener | **WARNING: Not implemented** — no compliance report generation |
| **SIEM Integration** | Full — native integrations: Splunk, Datadog, Sumo Logic, HTTP webhook log streams | Partial — via event listener SPI (Splunk, Elastic, custom) | **WARNING: Partial** — events on NATS JetStream (consumable by SIEM) but no native SIEM connector |
| **Tamper-Proof Audit** | Full — immutable logs, WORM storage integration | Partial — DB-backed, not tamper-proof | **WARNING: Not implemented** — events stored in PostgreSQL (not tamper-proof) |
| **Real-Time Alerting** | Full — anomaly detection, brute-force alerts (Enterprise) | Partial — via event listener + external system | **WARNING: Not implemented** |
| **Data Retention Policies** | Full — configurable log retention | Partial — via DB cleanup SPI | **WARNING: Not implemented** |

---

## 7. API

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **REST API** | Full — Management API (v2), Authentication API, comprehensive CRUD | Full — Admin REST API, comprehensive management endpoints | Full — REST API for all services via Gateway |
| **GraphQL** | No — REST only | No — REST only | **WARNING: Not implemented** — REST + gRPC only |
| **gRPC** | No — REST only | No — REST only (Quarkus gRPC extension available but not for admin API) | Full — gRPC for policy, org, audit services |
| **API Versioning** | Full — `/api/v2/` with deprecation notices | Full — Admin REST API versioning | Full — `/api/v1/` versioning |
| **Rate Limiting** | Full — per-tenant rate limits (plan-based) | Partial — via built-in rate limiter or Infinispan | **WARNING: Partial** — auth service rate limits login attempts (~5/min). No API-wide rate limiting |
| **API Explorer / Playground** | Full — interactive API explorer | Full — Swagger/OpenAPI admin docs | **WARNING: Not implemented** |
| **Webhooks** | Full — Actions, Rules, Hooks for event-driven extensibility | Partial — event listener SPI | **WARNING: Not implemented** |
| **OpenAPI Spec** | Full — published OpenAPI/Swagger docs | Full — generated OpenAPI docs | **WARNING: Partial** — routes defined but no published OpenAPI spec |
| **Response Caching** | Full — CDN + edge caching | Partial — Infinispan cache layer | Partial — ETag + Last-Modified headers on gateway |

---

## 8. SDK

| Language | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **Go** | Community SDKs only | Community SDKs | Full — official Go SDK + middleware |
| **Java** | Community SDKs | Full — keycloak-admin-client, Quarkus/OIDC adapter | Full — official Java SDK |
| **Node.js** | Full — `auth0-node` (auth0, auth0-js, express-openid-connect) | Community SDKs | Full — official Node.js SDK |
| **Python** | Full — `auth0-python` | Community SDKs | **WARNING: Not available** |
| **.NET / C#** | Full — `auth0.net` | Community SDKs | **WARNING: Not available** |
| **Ruby** | Full — `omniauth-auth0` | Community SDKs | **WARNING: Not available** |
| **PHP** | Full — `auth0-PHP` | Community SDKs | **WARNING: Not available** |
| **Swift / iOS** | Full — `Auth0.swift` | Community SDKs | **WARNING: Not available** |
| **Android / Kotlin** | Full — `auth0-Android` | Community SDKs | **WARNING: Not available** |
| **React / Frontend** | Full — `auth0-react`, `auth0-nextjs`, `auth0-spa-js` | Community SDKs | **WARNING: Not available** — admin console built in Next.js but no public frontend SDK |
| **SDK Auto-Generation** | N/A | N/A | **WARNING: Not available** — manual SDKs |

> Auth0 has the broadest SDK ecosystem with official, maintained SDKs in 10+ languages. GGID has 3 official SDKs (Go, Java, Node).

---

## 9. Deployment

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **Cloud-Hosted (SaaS)** | Full — Auth0-managed cloud (US/EU/AU regions) | Partial — via third-party managed (Cloud-IAM, PhaseTwo) | **WARNING: Not available** — self-hosted only |
| **Self-Hosted** | Enterprise only — CIAM appliance (on-prem) | Full — primary deployment model | Full — primary deployment model |
| **Docker** | N/A (SaaS) | Full — official Docker image | Full — Docker Compose with all services |
| **Kubernetes** | Enterprise only | Full — community & vendor Helm charts | **WARNING: Not available** — Docker Compose only |
| **Helm Charts** | N/A | Full — community Helm chart (codecentric/keycloak) | **WARNING: Not available** |
| **Infrastructure** | Auth0-managed (AWS) | JGroups, Infinispan, PostgreSQL/MySQL/MariaDB/MSSQL | PostgreSQL 16, Redis 7, NATS JetStream, OpenLDAP |
| **Database** | Managed by Auth0 | PostgreSQL, MySQL, MariaDB, MSSQL | PostgreSQL 16 (with RLS) |
| **High Availability** | Full — multi-region by default | Full — active-passive / active-active clustering | **WARNING: Not configured** — Docker Compose single-node. Requires K8s for HA |
| **Horizontal Scaling** | Full — auto-scaled by Auth0 | Full — cluster mode | **WARNING: Not configured** — microservice architecture supports it but no K8s manifests |
| **CI/CD Integration** | Full — Management API, deploy CLI, Terraform provider | Full — realm export/import, Terraform provider | **WARNING: Not available** — no IaC tooling |
| **Monitoring** | Full — built-in dashboard, Datadog integration | Full — JMX, Micrometer, Prometheus endpoints | **WARNING: Not implemented** — health checks only |
| **Image Size** | N/A | ~500MB+ (JVM) | Lightweight Go binaries (~20-35MB per service) |

---

## 10. License & Governance

| Feature | Auth0 | Keycloak | GGID |
|---|---|---|---|
| **License** | Proprietary (closed-source SaaS, Okta subsidiary) | Apache 2.0 | Apache 2.0 |
| **Governance** | Okta Inc. (commercial) | Red Hat / CNCF Sandbox project | Open-source community |
| **Cost** | Freemium (7,000 MAU free) → Pro ($35+/mo) → Enterprise (custom) | Free | Free |
| **Enterprise Support** | Full — Okta SLA, dedicated support | Full — Red Hat support (RHBK / Red Hat Build of Keycloak) | **WARNING: Not available** — community support only |
| **Security Audits** | Full — SOC 2 Type II, ISO 27001, HIPAA, FedRAMP | Full — Red Hat security team | **WARNING: Not audited** |
| **Community** | Limited (proprietary) | Full — large CNCF/open-source community | **WARNING: Early stage** — small team |
| **Contributing** | Closed source | Open contributions, CLA-free | Open contributions |

---

## Gap Analysis & Priority Recommendations

### P0 — Critical Gaps (Blocking Enterprise Adoption)

| # | Gap | Impact | Recommendation |
|---|---|---|---|
| 1 | **No K8s / Helm deployment** | Cannot deploy to production-grade orchestration | Create Helm chart + K8s manifests for all 7 services + infra |
| 2 | **No HA configuration** | Single point of failure | Add K8s deployment with replicas, health probes, horizontal pod autoscaling |
| 3 | **No token introspection (RFC 7662)** | Resource servers cannot validate tokens offline | Implement `/oauth/introspect` endpoint in auth/oauth service |
| 4 | **No Single Logout (SLO)** | Cannot centrally terminate sessions across apps | Implement front-channel + back-channel SLO per OIDC spec |
| 5 | **No OpenAPI/Swagger spec published** | Developers cannot auto-generate clients | Generate OpenAPI spec from Go handlers and publish at `/swagger` |
| 6 | **SCIM 2.0 incomplete** | Cannot integrate with enterprise IdP provisioning (Azure AD, Okta) | Implement full SCIM CRUD, filtering, bulk, PATCH |

### P1 — Important Gaps (Needed for Competitive Parity)

| # | Gap | Impact | Recommendation |
|---|---|---|---|
| 7 | **No per-tenant branding/custom domains** | White-label login impossible | Add per-tenant theme config + custom domain routing in gateway |
| 8 | **No tenant management API** | Cannot programmatically create/manage tenants | Add CRUD API for tenant lifecycle |
| 9 | **No SLO / backchannel logout** | Session management incomplete | Full — implement OIDC backchannel logout |
| 10 | **No concurrent session limits** | Cannot enforce single-session or N-session policies | Track active sessions in Redis, enforce configurable limits |
| 11 | **No Magic Link / Passwordless** | Missing popular auth UX | Implement magic link via email service + token endpoint |
| 12 | **No SMS/Email OTP MFA** | Limited MFA options | Add SMS (Twilio) and email OTP providers to authprovider chain |
| 13 | **No GraphQL API** | Some teams prefer GraphQL | Add GraphQL gateway alongside REST + gRPC |
| 14 | **No Prometheus/Grafana monitoring** | No observability for production | Add Prometheus metrics exporter + Grafana dashboards |
| 15 | **No Terraform/IaC provider** | Cannot manage GGID config as code | Build Terraform provider wrapping Management API |
| 16 | **Python SDK missing** | Large ecosystem locked out | Generate Python SDK from OpenAPI spec |
| 17 | **No WS-Federation** | Some legacy enterprise systems blocked | Add WS-Fed passive requestor profile |

### P2 — Moderate Gaps (Nice to Have)

| # | Gap | Impact | Recommendation |
|---|---|---|---|
| 18 | **No native SIEM connector** | Audit events require manual NATS consumer | Ship NATS→Splunk/Datadog connector |
| 19 | **No compliance reporting** | Cannot generate audit reports for SOC 2/HIPAA | Add report generation to audit service |
| 20 | **No tamper-proof audit trail** | Audit logs could be modified | Append-only storage (e.g., write-once S3, blockchain anchor) |
| 21 | **No API-wide rate limiting** | API abuse possible | Add per-tenant rate limiting middleware in gateway |
| 22 | **No API explorer/playground** | Poor developer experience | Deploy Swagger UI / Redoc at gateway |
| 23 | **No device authorization flow** | Smart TV / CLI auth impossible | Implement RFC 8628 device code grant |
| 24 | **No token exchange (RFC 8693)** | Cannot delegate or impersonate | Implement token exchange grant type |
| 25 | **No React/Frontend SDK** | SPA integration requires manual API calls | Generate JS/TS SDK from OpenAPI spec |
| 26 | **No real-time alerting** | Security incidents undetected | NATS consumer → alert rules (brute-force, anomaly) |

### P3 — Future / Low Priority

| # | Gap | Impact | Recommendation |
|---|---|---|---|
| 27 | **No data retention policies** | Unbounded audit log growth | Add configurable TTL + archival to audit service |
| 28 | **No .NET/Ruby/PHP/Swift/Android SDKs** | Niche language ecosystems | Generate from OpenAPI when spec is published |
| 29 | **No cloud-hosted SaaS option** | Teams who prefer managed hosting must self-deploy | Consider offering managed GGID cloud |
| 30 | **No enterprise security audit** | No formal compliance certification | Pursue SOC 2 Type II when adoption warrants |
| 31 | **No per-tenant IdP config** | Cannot configure per-tenant social/OIDC providers | Multi-tenant auth provider registry |

---

## Scorecard Summary

| Category | Auth0 | Keycloak | GGID | Gap vs Auth0/Keycloak |
|---|---|---|---|---|
| Auth Methods | 10/10 | 8/10 | 7/10 | Missing: SMS/Email MFA, Magic Link, Passwordless |
| IdP Protocols | 6/6 | 6/6 | 3/6 | Missing: WS-Federation, Token Exchange, Device Auth |
| Session Mgmt | 8/8 | 7/8 | 4/8 | Missing: SLO, Introspection, Concurrent Limits, Configurable Idle Timeout |
| Multi-Tenancy | 7/7 | 6/7 | 4/7 | Missing: Tenant Mgmt API, Custom Domains, Per-Tenant Branding |
| SCIM 2.0 | 7/7 | 4/7 (extension) | 1/7 | Major gap: Skeleton only |
| Audit | 8/8 | 5/8 | 4/8 | Missing: Compliance Reporting, SIEM Connector, Tamper-Proof, Alerting |
| API | 8/8 | 6/8 | 5/8 | Missing: GraphQL, Webhooks, Rate Limiting, OpenAPI Spec |
| SDK | 10/10 | 3/10 | 3/10 | Major gap: Only 3 SDKs vs Auth0's 10+ |
| Deployment | 8/8 | 8/8 | 3/8 | Major gap: No K8s, Helm, HA, CI/CD, Monitoring |
| License | Proprietary | Apache 2.0 | Apache 2.0 | GGID license is competitive advantage |

### GGID Competitive Advantages

1. **Go performance**: Lightweight binaries (~20-35MB), fast startup, low memory vs JVM (Keycloak ~500MB+)
2. **Microservice architecture**: Independent scaling, fault isolation, technology diversity
3. **NATS JetStream audit**: Native async event streaming — modern, performant, SIEM-ready
4. **PostgreSQL RLS**: Database-enforced multi-tenant isolation (stronger than app-level)
5. **gRPC first-class**: Native gRPC for all internal services (Auth0/Keycloak are REST-only)
6. **Apache 2.0 + no MAU limits**: Free at any scale (unlike Auth0's per-MAU pricing)
7. **RBAC + ABAC engine**: Combined policy engine with both REST and gRPC APIs

### GGID Key Weaknesses

1. **SCIM 2.0 is a skeleton** — critical for enterprise IdP integration (P0)
2. **No K8s/Helm deployment** — blocks production adoption (P0)
3. **Limited SDKs** (3 vs 10+) — narrows developer ecosystem
4. **No managed SaaS** — teams preferring managed hosting have no option
5. **Missing session management features** (SLO, introspection, concurrent limits)
6. **No enterprise support or security certifications**

---

## Methodology

- Auth0 data sourced from [auth0.com/features](https://auth0.com/features), [auth0.com/docs](https://auth0.com/docs), and pricing pages.
- Keycloak data sourced from [keycloak.org](https://keycloak.org), [CNCF documentation](https://www.keycloak.org/documentation), and community blogs.
- GGID data sourced from project source code, test coverage reports, and architecture documentation.
- Feature assessments reflect the state of each platform as of June 2025.
