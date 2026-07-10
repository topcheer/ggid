# GGID vs Ory Ecosystem: Comprehensive Comparison

> **Research Document** — GGID IAM Suite vs Ory (Kratos, Hydra, Keto, Oathkeeper)
>
> Date: 2025 | Authors: GGID Research Team

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture Comparison](#2-architecture-comparison)
3. [Feature-by-Feature Comparison](#3-feature-by-feature-comparison)
4. [Developer Experience](#4-developer-experience-comparison)
5. [Community & Maturity](#5-community--maturity)
6. [Where GGID Wins](#6-where-ggid-wins)
7. [Where Ory Wins](#7-where-ory-wins)
8. [Strategic Recommendations](#8-strategic-recommendations-for-ggid)
9. [Summary Comparison Matrix](#9-summary-comparison-matrix)

---

## 1. Overview

### 1.1 Ory Ecosystem

Ory is one of the most prominent open-source Identity and Access Management platforms, written entirely in Go — the same language as GGID. It consists of four independently deployable components that together form a complete IAM stack:

| Component | Role | GitHub Stars (approx.) |
|---|---|---|
| **Kratos** | Identity management — registration, login, profile, MFA, social login, account recovery | ~11K |
| **Hydra** | OAuth 2.0 / OIDC server — token issuance, consent flows, client management | ~15K |
| **Keto** | Authorization server — Google Zanzibar-style relation tuples, ACL | ~5K |
| **Oathkeeper** | Identity-aware reverse proxy / API gateway — PEP (Policy Enforcement Point) | ~3K |

**Key facts:**
- Licensed under **Apache 2.0** (same as GGID)
- **~50K+ combined GitHub stars** across all repos
- Latest release: **v25.4.0** (2025) — all components released in lockstep
- **Hydra is OIDC Certified** (OpenID Foundation conformance)
- Backed by **Ory Corp** (VC-funded, Series A/B)
- **CNCF Sandbox** project status
- Headquarters: Berlin, Germany / San Francisco, USA

### 1.2 GGID IAM Suite

GGID is an open-source IAM platform built as a Go monorepo with 7 microservices:

| Service | Role |
|---|---|
| **gateway** | API gateway with JWT verification, rate limiting, circuit breaker, tenant routing |
| **identity** | User identity, profile management, credential storage |
| **auth** | Authentication — login, register, JWT issuance, MFA TOTP, LDAP, social, SAML, WebAuthn |
| **oauth** | OAuth 2.0 / OIDC provider |
| **policy** | RBAC + ABAC policy engine with REST API + gRPC |
| **org** | Organization / B2B management |
| **audit** | Audit trail via NATS JetStream + REST query API |

**Shared packages:** `pkg/crypto`, `pkg/tenant`, `pkg/errors`, `pkg/authprovider`, `pkg/social`, `pkg/saml`, `pkg/email`, `pkg/i18n`, `pkg/pii`, `pkg/notification`, `pkg/audit`

**Infrastructure:** PostgreSQL 16 (with Row-Level Security), Redis 7, NATS JetStream, OpenLDAP

---

## 2. Architecture Comparison

### 2.1 GGID Architecture

GGID follows a **monorepo microservices** pattern — all services share a single Go module with common packages:

```
┌─────────────────────────────────────────────────────────────┐
│                     GGID Architecture                        │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ Gateway  │──│ Identity │  │   Auth   │  │  OAuth   │    │
│  │ (proxy)  │  │ (users)  │  │ (login)  │  │ (OIDC)   │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       │              │              │              │         │
│  ┌────┴─────┐  ┌────┴─────┐  ┌────┴─────┐                  │
│  │  Policy  │  │   Org    │  │  Audit   │                  │
│  │ (RBAC/   │  │  (B2B)   │  │ (NATS)   │                  │
│  │  ABAC)   │  │          │  │          │                  │
│  └──────────┘  └──────────┘  └──────────┘                  │
│                                                              │
│  Shared: PostgreSQL (RLS) | Redis | NATS JetStream           │
└─────────────────────────────────────────────────────────────┘
```

**Characteristics:**
- Single Go module, shared packages
- All services share a PostgreSQL database with **Row-Level Security** for tenant isolation
- NATS JetStream for event-driven audit trail
- Built-in multi-tenancy from the ground up
- gRPC for internal service communication + REST for external APIs
- Admin Console (Next.js 15) with 7 dashboard pages

### 2.2 Ory Architecture

Ory follows a **decoupled services** pattern — each component is a standalone Go binary with its own database:

```
┌─────────────────────────────────────────────────────────────┐
│                     Ory Architecture                         │
│                                                              │
│                    ┌──────────────┐                          │
│  HTTP Traffic ────▶│  Oathkeeper  │ (reverse proxy / PEP)   │
│                    └──────┬───────┘                          │
│                           │                                  │
│           ┌───────────────┼───────────────┐                  │
│           ▼               ▼               ▼                  │
│    ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│    │   Kratos   │  │   Hydra    │  │    Keto    │           │
│    │ (identity) │  │ (OAuth2/   │  │ (authz /   │           │
│    │            │  │  OIDC)     │  │  Zanzibar) │           │
│    └─────┬──────┘  └─────┬──────┘  └─────┬──────┘           │
│          │               │               │                   │
│    ┌─────┴──────┐  ┌────┴───────┐  ┌────┴───────┐           │
│    │ Postgres   │  │ Postgres   │  │ Postgres   │           │
│    │ (own DB)   │  │ (own DB)   │  │ (own DB)   │           │
│    └────────────┘  └────────────┘  └────────────┘           │
│                                                              │
│  Each service: independent deploy, independent database      │
│  Communication: HTTP REST APIs                               │
└─────────────────────────────────────────────────────────────┘
```

**Characteristics:**
- 4 separate Git repositories, independently versioned (but released in lockstep)
- Each service has its **own database/schema** — no shared DB
- Communication via HTTP REST APIs (no shared Go packages)
- Cloud-native: Docker images, Helm charts, Kubernetes operators
- Ory Network (managed cloud) and Ory Enterprise License (self-hosted enterprise)
- Auto-generated SDKs from OpenAPI spec

### 2.3 Architecture Differences Summary

| Aspect | GGID | Ory |
|---|---|---|
| **Codebase** | Monorepo (single Go module) | Polyrepo (4 separate repos) |
| **Database** | Shared PostgreSQL with RLS | Separate DB per service |
| **Service count** | 7 microservices | 4 services |
| **Inter-service comm** | gRPC + REST | HTTP REST only |
| **Shared libraries** | Yes (11 shared packages) | No (each repo is standalone) |
| **Multi-tenancy** | Built-in (RLS + tenant_id) | Enterprise/cloud only (OSS is single-tenant) |
| **Event system** | NATS JetStream (built-in) | None (external logging) |
| **Deployment** | Docker Compose | Helm charts, K8s operators |
| **Build system** | Go Makefile + Docker | Bazel + Docker |

### 2.4 Service Mapping Table

| GGID Service | Ory Equivalent | Coverage | Notes |
|---|---|---|---|
| **gateway** | Oathkeeper | Partial | Both are PEP; Oathkeeper has richer authenticator/authorizer plugins |
| **identity** | Kratos | Partial | Kratos covers identity + auth flows together |
| **auth** | Kratos (partially) | Partial | Kratos handles login, MFA, recovery |
| **oauth** | Hydra | Subset | Hydra has more OAuth grants + OIDC certification |
| **policy** | Keto | Different model | Keto uses Zanzibar tuples; GGID uses RBAC+ABAC rules |
| **org** | *(no direct equivalent)* | — | GGID differentiator; Kratos added "organizations" concept in 2024 |
| **audit** | *(no built-in)* | — | Ory relies on external logging/observability stack |

---

## 3. Feature-by-Feature Comparison

### 3.1 Identity Management (Kratos vs GGID identity + auth)

| Feature | Ory Kratos | GGID (identity + auth) |
|---|---|---|
| **Registration** | Flow-based with customizable UI steps, webhooks, identity schema validation (JSON Schema) | API-driven registration with username/email + password |
| **Login methods** | Password, social (OIDC), WebAuthn/passkeys, TOTP, lookup codes | Password, social (9 providers), MFA TOTP, LDAP, SAML, WebAuthn |
| **Account recovery** | Self-service flows (email verification, recovery code) via Kratos flows | API-based recovery (password reset endpoint) |
| **Profile management** | JSON Schema-defined "traits" with UI form auto-generation | Struct-based fields with REST API |
| **Social login** | Built-in OIDC provider integration (Google, GitHub, Apple, etc.) | 9 providers: Google, GitHub, Microsoft, Discord, Slack, LinkedIn, GitLab, Apple, generic OIDC |
| **MFA** | TOTP, WebAuthn, lookup codes | TOTP, WebAuthn |
| **Account linking** | Automatic and manual identity linking | Per-provider linking |
| **Identity schema** | JSON Schema (extensible, user-defined) | Go struct fields (compiled) |
| **Webhooks** | Native webhooks on registration, login, etc. | Not built-in |
| **Session management** | Cookie-based sessions with configurable lifetime | JWT-based sessions with refresh tokens |
| **Privileged access** | Admin API for impersonation | Admin API for user management |
| **Email verification** | Built-in self-service flow | API-based verification |
| **Account import** | Identity import API | Bulk create via identity API |

**Kratos "Flow" Model:**
Kratos uses a unique "flow" abstraction — every identity interaction (registration, login, verification, recovery, settings) is a self-contained flow object. Each flow has:
- A configurable set of UI nodes (form fields)
- Webhooks triggered at each step
- Customizable redirect URLs and messages
- JSON Schema-driven form generation

This is powerful for SaaS applications that need custom-branded UI flows, but adds complexity for simple API-first use cases.

**GGID Advantage:** API-first simplicity. Registration and login are straightforward REST calls — no flow management, no redirect handling, no flow state to track.

### 3.2 OAuth/OIDC (Hydra vs GGID oauth)

| Feature | Ory Hydra | GGID oauth |
|---|---|---|
| **OIDC Certification** | ✅ Certified (OpenID Foundation) | ❌ Not certified |
| **Authorization Code Grant** | ✅ | ✅ |
| **Client Credentials Grant** | ✅ | ✅ |
| **Refresh Token Grant** | ✅ | ✅ |
| **Device Authorization Grant** | ✅ (RFC 8628) | ❌ |
| **Token Exchange (RFC 8693)** | ✅ | ❌ |
| **Resource Owner Password** | ✅ (deprecated, configurable) | ✅ |
| **Token format** | Opaque by default; can return JWTs | JWT |
| **Token signing** | RS256, ES256, HS256 | RS256 |
| **Client management** | Full CRUD via Admin API | REST API for client CRUD |
| **Consent flow** | Customizable consent screen with hooks | Basic consent handling |
| **PKCE** | ✅ Required/recommended | ✅ Supported |
| **DPoP** | ✅ (Demonstration of Proof-of-Possession) | ❌ |
| **Token introspection** | ✅ RFC 7662 | ❌ |
| **Token revocation** | ✅ RFC 7009 | ✅ |
| **JWKS endpoint** | ✅ Public keys for JWT verification | ✅ |
| **Scope management** | Fine-grained scope-per-audience | Basic scope support |

**Key Gap: OIDC Certification.** Hydra has passed the OpenID Foundation conformance test suite, making it suitable for regulated industries (finance, healthcare) that require certified OIDC providers. GGID's OAuth implementation is functional but uncertified. Achieving OIDC certification should be a GGID priority.

### 3.3 Access Control (Keto vs GGID policy)

| Feature | Ory Keto | GGID policy |
|---|---|---|
| **Authorization model** | Google Zanzibar relation tuples (`object#relation@subject`) | RBAC + ABAC rule engine |
| **Data model** | Relation tuples stored in PostgreSQL | Policy rules with conditions stored in PostgreSQL |
| **Query style** | `check(object, relation, subject)` → boolean | `evaluate(policy, subject, action, resource)` → allow/deny |
| **Performance** | In-memory graph cache, sub-ms checks for warm data | PostgreSQL query per evaluation |
| **Relationship types** | Direct, transitive (recursive graph traversal), contextual | Role-based, attribute-based, condition-based |
| **Scalability** | Designed for millions of tuples, O(1) cached lookups | Scales with PostgreSQL; needs caching for high QPS |
| **Policy language** | Tuple-based (no DSL) | Structured rules (no DSL, JSON/YAML config) |
| **Use cases** | "Can user X edit document Y?", team membership, file sharing, org hierarchy | "Does role X have permission Y on resource Z?", policy enforcement |
| **gRPC API** | ✅ | ✅ |
| **REST API** | ✅ | ✅ |
| **Watch/sync** | Real-time tuple writes and reads | Policy CRUD via REST |
| **Policy export/import** | ❌ | ✅ (planned) |

**Model Trade-off:**
- **Keto's Zanzibar model** excels at fine-grained, per-resource authorization: "user X can edit document Y." It builds a relationship graph and answers check queries in O(1) when cached. This is ideal for collaborative apps (Google Docs-style), file sharing, and complex org hierarchies.
- **GGID's RBAC+ABAC model** excels at policy-driven authorization: "users with role 'admin' can manage resources in tenant X." It's simpler to reason about, maps well to enterprise access policies, and integrates naturally with the tenant model.

### 3.4 Gateway/Proxy (Oathkeeper vs GGID gateway)

| Feature | Ory Oathkeeper | GGID gateway |
|---|---|---|
| **Role** | Identity-aware reverse proxy / PEP | API gateway + JWT verification proxy |
| **Authenticators** | Cookie session, JWT, OAuth2 introspection, anonymous, noop, bearer token, WebAuthn | JWT verification |
| **Authorizers** | Keto (Zanzibar), allow, deny, remote (custom JSON) | Policy service (RBAC/ABAC) |
| **Mutators** | Header injection, ID token (JWT), cookie, hydrated header | Tenant header injection |
| **Rate limiting** | ❌ (not built-in) | ✅ Sliding window rate limiting |
| **Circuit breaker** | ❌ | ✅ |
| **HTTP/3 support** | ❌ | ✅ |
| **Tenant routing** | ❌ | ✅ (X-Tenant-ID header + query param) |
| **Plugin system** | ✅ (extensible authenticators/authorizers/mutators) | Middleware chain |
| **Configuration** | YAML rules file (declarative) | Go code (programmatic) |
| **Access rules** | Per-route rule matching (URL method + path) | Route table with middleware |

**Oathkeeper's plugin architecture** is a significant advantage — developers can add custom authenticators, authorizers, or mutators. GGID's middleware chain is flexible but requires code changes for new authentication methods.

### 3.5 Multi-tenancy

| Aspect | GGID | Ory |
|---|---|---|
| **Native multi-tenancy** | ✅ Built-in from day one | ❌ OSS is single-tenant only |
| **Data isolation** | PostgreSQL Row-Level Security (RLS) | Separate databases or Ory Enterprise |
| **Tenant routing** | X-Tenant-ID header throughout stack | Not available in OSS |
| **Tenant-aware policies** | ✅ tenant_id in every policy/rule | Not in OSS |
| **Per-tenant configuration** | Planned | Enterprise/cloud only |
| **Multi-tenancy licensing** | Free (Apache 2.0) | Ory Network (SaaS) or Enterprise License (paid) |

> **Critical finding:** The Ory Kratos open-source version is explicitly **single-tenant only**. Its data model is not architected for multi-tenant data isolation. Multi-tenancy requires either Ory Network (managed cloud) or an Ory Enterprise License (self-hosted, paid). This is a **major competitive advantage** for GGID.

### 3.6 Audit & Observability

| Feature | GGID | Ory |
|---|---|---|
| **Built-in audit trail** | ✅ NATS JetStream event log | ❌ No built-in audit service |
| **Audit query API** | ✅ REST API for audit events | ❌ |
| **Event streaming** | ✅ NATS JetStream pub/sub | ❌ |
| **Log integration** | Application logs (structured) | Application logs + Ory Cloud has built-in logging |
| **Compliance reporting** | Planned | Ory Enterprise: audit logs |
| **Real-time alerts** | Planned (NATS consumer) | External (e.g., Datadog, ELK) |

GGID's event-driven audit trail via NATS JetStream is a significant differentiator. Every authenticated action, policy decision, and administrative operation can be published as an audit event. Ory has no equivalent in the open-source stack — organizations must build their own observability pipeline using external tools.

### 3.7 Organizations (B2B)

| Feature | GGID | Ory |
|---|---|---|
| **Dedicated org service** | ✅ | ❌ (Kratos "organizations" concept added 2024) |
| **Org hierarchy** | Planned | ✅ Organization hierarchies |
| **Delegated admin** | Planned | ✅ (Enterprise/cloud) |
| **Enterprise SSO per org** | ✅ SAML + OIDC per tenant | ✅ SAML 2.0 + OIDC federation |
| **Org-level roles** | ✅ tenant_id-scoped roles | ✅ |
| **Member management** | REST API | Kratos API |
| **SCIM provisioning** | Skeleton (SCIM 2.0) | ❌ (enterprise SSO only) |

Ory has invested significantly in B2B IAM with their "organizations" concept in Kratos (2024+), supporting organization hierarchies, delegated administration, and enterprise SSO onboarding. However, these features require Ory Network or Enterprise License — not available in the open-source version.

### 3.8 SDK Coverage

| Language | GGID | Ory |
|---|---|---|
| **Go** | ✅ | ✅ |
| **JavaScript/TypeScript** | ✅ (Node SDK) | ✅ |
| **Java** | ✅ | ✅ |
| **Python** | ❌ | ✅ |
| **PHP** | ❌ | ✅ |
| **Rust** | ❌ | ✅ |
| **.NET / C#** | ❌ | ✅ |
| **Dart** | ❌ | ❌ |
| **Generation method** | Hand-written | Auto-generated from OpenAPI spec |

Ory auto-generates SDKs from their OpenAPI specification, giving them broad language coverage with minimal maintenance effort. GGID's hand-written SDKs are higher quality per language but limited to 3 languages.

---

## 4. Developer Experience Comparison

| Aspect | Ory | GGID |
|---|---|---|
| **Managed cloud** | ✅ Ory Network (free tier + paid) | ❌ |
| **CLI tools** | ✅ Ory CLI (identity import, proxy, schema push) | ❌ |
| **Self-service UI** | ✅ React-based account UI kit (customizable) | ✅ Admin Console (Next.js, 7 pages) |
| **Documentation** | Extensive (ory.com/docs), guides, tutorials | Growing (docs/ directory) |
| **Quickstart** | `ory create` + Ory Network | `docker compose up -d` |
| **Local development** | Ory CLI proxy tunnel | Docker Compose (13 containers) |
| **API exploration** | Swagger UI for all services | REST APIs with OpenAPI (planned) |
| **Schema management** | JSON Schema UI for identity traits | Struct-based (compiled) |
| **Error handling** | Structured error responses with flow context | Structured error responses (pkg/errors) |
| **Testing** | Docker-based integration tests | 250+ test cases, 28 test suites |

**Ory's developer experience advantage** comes from:
1. Ory Network — zero-infrastructure onboarding
2. CLI tools for identity management and local proxying
3. Self-service UI kit for branded login/registration flows
4. Extensive documentation with practical examples

**GGID's developer experience strengths:**
1. Single `docker compose up -d` to run the entire stack
2. Monorepo makes cross-service changes straightforward
3. Admin Console with real API integration (not mock data)
4. gRPC + REST dual protocol for internal/external APIs

---

## 5. Community & Maturity

| Metric | Ory | GGID |
|---|---|---|
| **Founded** | 2014 (arekk on GitHub) | 2024 |
| **GitHub stars** | ~50K combined | Growing |
| **Contributors** | 500+ across repos | Small team |
| **Company backing** | Ory Corp (VC-backed, Series A/B) | Open-source community |
| **CNCF status** | Sandbox project | None |
| **Production users** | Tesla, GitHub (limited), Unity, Zalando | Early stage |
| **Commercial offering** | Ory Network (SaaS), Enterprise License | None (open-source only) |
| **Release cadence** | Quarterly major releases (v25.x) | Continuous |
| **Security audits** | Professional security audits | Community-driven |
| **Bug bounty** | Yes (via HackerOne) | No |
| **Conferences** | Ory Summit, conference talks | Community |

Ory has an 11-year head start and significant VC investment. Their community, documentation, and enterprise adoption are mature. GGID is newer but architecturally differentiated.

---

## 6. Where GGID Wins

### 6.1 Built-in Multi-tenancy ( Biggest Advantage )

GGID has multi-tenancy as a **first-class architectural principle**:
- PostgreSQL Row-Level Security (RLS) for database-level tenant isolation
- `tenant_id` threaded through every API, every policy, every audit event
- Tenant-aware routing in the gateway
- No additional licensing required

**Ory's OSS is explicitly single-tenant.** Multi-tenancy requires Ory Network (SaaS) or Enterprise License (paid self-hosted). This is a fundamental architectural limitation of the open-source Ory stack.

### 6.2 Event-Driven Audit Trail

NATS JetStream provides a built-in, durable, event-driven audit system:
- Every authenticated action publishes an audit event
- Events are queryable via REST API
- Supports real-time consumers for alerting/compliance
- No external infrastructure needed

Ory has **no built-in audit capability** — organizations must integrate external logging/observability tools (ELK, Datadog, Splunk).

### 6.3 Dedicated Organization/B2B Service

GGID has a dedicated `org` service for B2B use cases. While Ory Kratos added "organizations" in 2024, this feature is only available in Ory Network or Enterprise License — not in the open-source version.

### 6.4 Monorepo Simplicity

- Single `go build ./...` compiles everything
- Shared packages reduce code duplication
- Cross-service refactoring is trivial
- Single CI/CD pipeline
- One Docker Compose file for the entire stack

Ory's 4-repo approach means each service is versioned independently, requiring careful compatibility management.

### 6.5 RLS for Tenant Isolation

PostgreSQL Row-Level Security provides **database-enforced** tenant isolation — even if application code has a bug, the database prevents cross-tenant data leakage. This is more robust than application-level checks.

### 6.6 gRPC + REST Dual Protocol

GGID supports both gRPC (for internal service-to-service communication) and REST (for external APIs). Ory is REST-only.

---

## 7. Where Ory Wins

### 7.1 OIDC Certification

Hydra is **OIDC Certified** by the OpenID Foundation. This is required for:
- Regulated industries (finance, healthcare, government)
- Enterprise procurement requirements
- Interoperability guarantees

GGID's OAuth/OIDC implementation is functional but uncertified.

### 7.2 SDK Language Coverage

Ory offers **7+ SDK languages** (Go, JS/TS, Python, PHP, Rust, .NET, Java) auto-generated from OpenAPI. GGID offers 3 (Go, Node, Java), hand-written.

### 7.3 Cloud-Native Deployment

Ory provides:
- Production-grade Helm charts
- Kubernetes operators
- Ory Network (fully managed SaaS)
- Ory Enterprise License (self-hosted enterprise)

GGID has Docker Compose but lacks production-grade Kubernetes manifests and Helm charts.

### 7.4 Flow-Based Identity UI

Kratos's flow model enables:
- Customizable multi-step registration/login flows
- JSON Schema-driven UI form generation
- Webhooks at each flow step
- Branded self-service UIs

GGID's API-driven approach is simpler but less flexible for custom UI flows.

### 7.5 Keto's Zanzibar Model

For applications requiring fine-grained, per-resource authorization:
- O(1) cached relationship checks
- Millions of tuples
- Transitive relationship traversal
- Ideal for collaborative apps (Google Docs, Figma, Notion-style)

### 7.6 Community & Ecosystem

Ory has a 10+ year head start:
- 500+ contributors
- 50K+ GitHub stars
- Professional documentation
- Security audits
- Bug bounty program
- Conference presence (Ory Summit)

### 7.7 Oathkeeper Plugin System

Oathkeeper's extensible authenticator/authorizer/mutator plugin system allows adding custom authentication methods without modifying core code.

---

## 8. Strategic Recommendations for GGID

### 8.1 Five Priority Actions to Close the Gap

#### 1. Achieve OIDC Certification (HIGH PRIORITY)
- Run the OpenID Foundation conformance test suite against GGID's OAuth service
- Fix failing test cases
- Apply for official certification
- **Impact:** Unlocks enterprise/regulated industry adoption
- **Effort:** 2-4 engineer-months

#### 2. Add Token Introspection & Additional OAuth Grants
- Implement RFC 7662 (Token Introspection endpoint)
- Implement RFC 8628 (Device Authorization Grant) for IoT/CLI scenarios
- Implement RFC 8693 (Token Exchange) for delegation scenarios
- **Impact:** Feature parity with Hydra for OAuth use cases
- **Effort:** 1-2 engineer-months

#### 3. Auto-Generate SDKs from OpenAPI Spec
- Define complete OpenAPI 3.1 specification for all REST APIs
- Use code generators (openapi-generator, oapi-codegen) to produce multi-language SDKs
- Target: Go, TypeScript, Python, Java, Rust, .NET
- **Impact:** Closes the SDK language gap without hand-writing
- **Effort:** 1 engineer-month + ongoing maintenance

#### 4. Production Kubernetes Deployment (Helm Charts)
- Create Helm charts for all 7 services + infrastructure
- Add Kubernetes health checks, readiness probes, HorizontalPodAutoscaler
- Provide values.yaml for common deployment scenarios
- **Impact:** Production-ready cloud-native deployment story
- **Effort:** 1-2 engineer-months

#### 5. Flow-Based Identity API (Optional Differentiator)
- Add an optional flow abstraction layer on top of the auth API
- Support multi-step registration, progressive profiling, conditional logic
- JSON Schema-driven UI form definitions
- Webhooks on identity events
- **Impact:** Feature parity with Kratos for SaaS use cases
- **Effort:** 2-3 engineer-months

### 8.2 What GGID Should NOT Copy from Ory

- **Keto's Zanzibar model** — GGID's RBAC+ABAC is sufficient for enterprise access control. Zanzibar-style tuples add complexity for limited benefit unless building collaborative document apps. Instead, add caching to the existing policy engine.
- **Separate databases per service** — GGID's shared database with RLS is a strength, not a weakness. It enables cross-service transactions and simpler deployment.
- **4 separate repositories** — The monorepo approach is better for a project of GGID's size. Polyrepo adds operational overhead.
- **Kratos's flow abstraction as the ONLY option** — Keep the simple API-driven approach as the default; offer flows as an optional layer.
- **Opaque token format** — JWT tokens are better for modern microservice architectures (stateless verification).

### 8.3 Where GGID Should Differentiate

| Differentiator | Why GGID Wins | Action |
|---|---|---|
| **Multi-tenancy** | RLS + tenant_id throughout, free in OSS | Highlight in marketing; add tenant management UI |
| **Event-driven audit** | NATS JetStream built-in | Add compliance reporting templates; SIEM integration docs |
| **B2B organizations** | Dedicated org service, free in OSS | Build delegated admin UI; add org provisioning automation |
| **gRPC + REST** | Dual protocol for performance + accessibility | Highlight performance benchmarks |
| **Monorepo simplicity** | One build, one deploy, one CI | Create getting-started guides emphasizing simplicity |
| **SCIM 2.0** | Enterprise user provisioning standard | Complete SCIM 2.0 implementation; certify |

### 8.4 Competitive Positioning

```
                    ┌─────────────────────────────────┐
                    │         Target Market            │
   ┌────────────────┤  B2B SaaS / Enterprise IAM       │
   │                └────────────┬────────────────────┘
   │                             │
   ▼                             ▼
┌──────────────┐          ┌──────────────┐
│    GGID      │          │     Ory      │
│              │          │              │
│ • Multi-     │          │ • OIDC       │
│   tenancy    │          │   certified  │
│   (free OSS) │          │ • Managed    │
│ • B2B org    │          │   cloud      │
│ • Event audit│          │ • Large      │
│ • Monorepo   │          │   community  │
│   simplicity │          │ • K8s native │
│ • SCIM 2.0   │          │ • 7+ SDKs    │
└──────────────┘          └──────────────┘
     Open-source              Open-source
     self-hosted              + managed cloud
     multi-tenant             + enterprise license
```

**GGID's niche:** Self-hosted, multi-tenant, B2B IAM with built-in audit — without the licensing restrictions of Ory Enterprise.

---

## 9. Summary Comparison Matrix

| Category | Feature | GGID | Ory | Winner |
|---|---|---|---|---|
| **Identity** | Registration | API-driven | Flow-based | Ory (flexibility) |
| **Identity** | Social login | 9 providers | OIDC integration | Tie |
| **Identity** | MFA | TOTP, WebAuthn | TOTP, WebAuthn, lookup codes | Ory |
| **Identity** | Account recovery | API-based | Self-service flows | Ory |
| **Identity** | Identity schema | Go struct fields | JSON Schema (dynamic) | Ory |
| **OAuth** | OIDC certified | ❌ | ✅ | **Ory** |
| **OAuth** | OAuth grants | 3 (auth code, client creds, refresh) | 6+ (adds device, token exchange, etc.) | **Ory** |
| **OAuth** | Token introspection | ❌ | ✅ | **Ory** |
| **Authz** | Model | RBAC + ABAC | Zanzibar tuples | Tie (different strengths) |
| **Authz** | Performance | PostgreSQL queries | In-memory cache | **Ory** |
| **Gateway** | Authenticator plugins | Middleware chain | Plugin system | **Ory** |
| **Gateway** | Rate limiting | ✅ | ❌ | **GGID** |
| **Gateway** | HTTP/3 | ✅ | ❌ | **GGID** |
| **Multi-tenancy** | Native OSS support | ✅ (RLS) | ❌ (enterprise/cloud only) | **GGID** |
| **Audit** | Built-in event trail | ✅ (NATS JetStream) | ❌ | **GGID** |
| **B2B** | Dedicated org service | ✅ | Enterprise only | **GGID** |
| **SDK** | Languages | 3 | 7+ | **Ory** |
| **Deployment** | Docker Compose | ✅ | ✅ | Tie |
| **Deployment** | Helm/K8s | ❌ (planned) | ✅ | **Ory** |
| **Deployment** | Managed cloud | ❌ | ✅ (Ory Network) | **Ory** |
| **DX** | Admin Console | ✅ (Next.js) | ✅ (React UI kit) | Tie |
| **DX** | CLI tools | ❌ | ✅ | **Ory** |
| **Community** | GitHub stars | Growing | ~50K | **Ory** |
| **Community** | Contributors | Small team | 500+ | **Ory** |
| **License** | Open source | Apache 2.0 | Apache 2.0 | Tie |
| **Architecture** | Codebase | Monorepo | Polyrepo (4 repos) | GGID (simplicity) |
| **Architecture** | gRPC support | ✅ | ❌ | **GGID** |
| **Architecture** | Shared database | Yes (RLS) | No (separate per service) | GGID (multi-tenancy) |

**Score:** GGID wins 10 categories, Ory wins 12, tie 6.

---

## Appendix A: Ory Component Deep-Dive

### Kratos (Identity Server)
- **Purpose:** Headless identity management with self-service flows
- **Key concepts:** Identity, Traits (JSON Schema), Flows, Credentials, Sessions
- **Flow types:** Registration, Login, Verification, Recovery, Settings, Logout
- **Identity schema:** Arbitrary JSON Schema for custom user attributes
- **Credential types:** Password, OIDC, WebAuthn, TOTP, lookup_secret
- **Webhooks:** Pre/post hooks on registration, login, etc.
- **Database:** PostgreSQL, SQLite (dev), MySQL (deprecated)
- **Configuration:** YAML with extensive options

### Hydra (OAuth2/OIDC Server)
- **Purpose:** Certified OAuth 2.0 / OpenID Connect provider
- **Key concepts:** OAuth2 Client, Consent, Token, Session
- **Token strategy:** Opaque tokens (reference) by default; configurable JWT
- **Conformance:** Full OIDC Basic + Implicit + Hybrid + Configurable certification
- **Security:** Key rotation, JWKS endpoint, DPoP support
- **OEM use case:** Designed for "OAuth as a service" — issuing tokens for third-party clients
- **Consent flow:** Decoupled consent UI (separate from login)

### Keto (Authorization Server)
- **Purpose:** Google Zanzibar-style authorization with relation tuples
- **Data model:** `namespace:object#relation@subject`
- **Example:** `document:report.pdf#editor@user:alice`
- **Operations:** Write relation tuple, Check relation, Expand subjects, List relations
- **Performance:** In-memory graph cache for sub-ms checks
- **Scale:** Designed for billions of tuples
- **Limitation:** No policy language or ABAC conditions — purely tuple-based

### Oathkeeper (Reverse Proxy)
- **Purpose:** Identity-aware proxy that sits in front of APIs
- **Pipeline:** Request → Authenticate → Authorize → Mutate → Forward
- **Authenticators:** anonymous, cookie_session, bearer_token, jwt, oauth2_client_credentials, oauth2_introspection, noop, unauthorized
- **Authorizers:** allow, deny, keto_engine, remote_json, remote
- **Mutators:** cookie, header, hydrator, id_token, noop
- **Configuration:** YAML rules file with URL patterns and handler chains
- **Deployment:** Usually deployed as a sidecar or ingress controller

---

## Appendix B: GGID Service Deep-Dive

### Gateway
- JWT verification middleware
- Sliding window rate limiting
- Circuit breaker pattern
- Tenant routing (X-Tenant-ID header → query param/JSON body injection)
- HTTP/3 support
- Reverse proxy to all downstream services

### Identity
- User CRUD with profile management
- Credential storage (password hashes, social tokens)
- PII handling (pkg/pii for encryption-at-rest)
- Tenant-scoped user queries

### Auth
- Registration (username/email + password)
- Login with JWT issuance + refresh tokens
- MFA TOTP enrollment and verification
- LDAP integration (pkg/authprovider)
- Social login (pkg/social: 9 providers)
- SAML 2.0 SSO (pkg/saml)
- WebAuthn/Passkey registration and authentication
- Account recovery (API-based)

### OAuth
- OAuth 2.0 authorization server
- Authorization Code, Client Credentials, Refresh Token grants
- JWT token issuance (RS256)
- Client management (CRUD)
- JWKS endpoint

### Policy
- RBAC: roles with permissions, scoped to tenant
- ABAC: attribute-based conditions
- gRPC + REST API
- Policy export/import (planned)

### Org
- Organization CRUD
- Member management
- Org-level roles
- Enterprise SSO per organization

### Audit
- NATS JetStream event publisher
- REST query API for audit events
- Tenant-scoped audit trails
- Structured event format

---

## References

- [Ory Documentation](https://www.ory.com/docs)
- [Ory GitHub Organization](https://github.com/ory)
- [Ory v25.4.0 Release](https://www.ory.com/blog/ory-oss-v-25-4-0-launch-recap)
- [Ory B2B IAM Solution](https://www.ory.com/docs/solutions/solution-B2B)
- [Ory Kratos Multi-tenancy Guide](https://www.ory.com/docs/kratos/guides/multi-tenancy-multitenant)
- [Ory Hydra OAuth2 Server](https://www.ory.com/hydra)
- [GGID GitHub Repository](https://github.com/ggid/ggid)
- [OpenID Foundation Certification](https://openid.net/certification/)
- [Google Zanzibar Paper](https://research.google/pubs/pub48190/)

---

*This document is a living research artifact. Last updated: 2025. For the latest GGID feature status, refer to the [project README](../../README.md) and [feature matrix](../feature-matrix.md).*
