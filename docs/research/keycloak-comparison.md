# Keycloak vs GGID: Deep Competitive Analysis

> **Purpose**: Comprehensive engineering-level comparison of Keycloak and GGID across architecture, features, operations, and strategic positioning.
> Last updated: 2025-06-17
> Author: Competitive Analysis Team

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Architecture Comparison](#2-architecture-comparison)
3. [Feature Comparison Matrix (50+ Features)](#3-feature-comparison-matrix)
4. [Multi-Tenancy (Realms)](#4-multi-tenancy-realms)
5. [Identity Brokering](#5-identity-brokering)
6. [Authorization Services (UMA 2.0)](#6-authorization-services-uma-20)
7. [Plugin Ecosystem](#7-plugin-ecosystem)
8. [Community & Documentation](#8-community--documentation)
9. [Deployment & Operations](#9-deployment--operations)
10. [What GGID Can Learn from Keycloak](#10-what-ggid-can-learn-from-keycloak)
11. [Gap Analysis & Recommendations](#11-gap-analysis--recommendations)

---

## 1. Project Overview

### 1.1 Keycloak: History and Governance

Keycloak is one of the most widely adopted open-source Identity and Access Management
(IAM) platforms. Its trajectory from an internal Red Hat project to a CNCF community
project reflects the broader maturation of the identity ecosystem.

**Timeline:**

| Year | Milestone |
|------|-----------|
| **2014** | Keycloak launched by Red Hat as an open-source IAM solution, initially targeting Java EE developers who needed an alternative to proprietary SSO solutions. Built on the WildFly application server (JBoss). |
| **2015-2017** | Rapid feature expansion: OIDC certification, SAML 2.0, identity brokering, social login, multi-tenancy via realms. Community adoption accelerates. |
| **2018** | Red Hat begins productizing Keycloak as Red Hat Single Sign-On (RH-SSO), later renamed Red Hat Build of Keycloak (RHBK). Enterprise support contracts available. |
| **2020** | Keycloak accepted as a CNCF Sandbox project, increasing visibility and community governance. The CNCF sandbox provides infrastructure, CI/CD, and community governance without the strict incubation/graduation tiers of graduated projects like Kubernetes. |
| **2021-2022** | Major architecture shift: Keycloak moves from WildFly to Quarkus (supersonic subatomic Java). The new distribution (`kc.sh start`) replaces the WildFly JBoss module system with a single JAR-based runtime. This reduces startup time and memory footprint significantly. |
| **2023** | Map storage SPI introduced (preview), aiming to replace JPA for better performance at scale. Keycloak 22 ships with significant performance improvements. |
| **2024** | Keycloak 24+ continues Quarkus-first. New admin console (React-based, replacing the legacy Angular console). Declarative UP (User Profile) moves from preview to GA. |
| **2025** | Keycloak remains the de facto open-source IAM standard with 25,000+ GitHub stars, active development, and broad enterprise adoption. |

**Governance Structure:**

- **Primary Sponsor**: Red Hat (a subsidiary of IBM). Red Hat employees constitute
  the majority of core maintainers.
- **CNCF Affiliation**: CNCF Sandbox project since 2020. The sandbox tier is the
  entry point for early-stage projects; Keycloak has not progressed to Incubating
  or Graduated status.
- **License**: Apache License 2.0 — fully permissive, no copyleft requirements.
- **CLA**: No Contributor License Agreement required — contributions are under
  the Apache 2.0 license directly.

**Market Position:**

Keycloak occupies the unique position of being both a free open-source project
and a commercially supported enterprise product. This dual identity is a key
competitive advantage:

- **Enterprise customers** can purchase Red Hat Build of Keycloak (RHBK) for SLA
  support, security patches, and certification.
- **Community users** can self-host the upstream project at no cost.
- **Managed providers** (Cloud-IAM, PhaseTwo, Skrador) offer Keycloak-as-a-Service.

**Known Enterprise Adopters:**

| Organization | Use Case | Scale |
|---|---|---|
| **IBM** | Internal SSO across hundreds of products | 500K+ users |
| **Cisco** | DevNet identity, partner portal SSO | Enterprise |
| **Deutsche Bank** | Developer portal, internal tooling | Regulated financial |
| **Santander** | Engineering tooling, CI/CD access | Banking sector |
| **Eurostat** (EU Commission) | Statistical portal auth | Government |
| **MuleSoft** | Anypoint Platform integration | ISV |
| **GitLab** | Historical (migrated to custom) | DevTools |
| **VMware** (Broadcom) | Tanzu platform identity | Cloud infrastructure |

### 1.2 Community Metrics

| Metric | Keycloak | GGID |
|---|---|---|
| **GitHub Stars** | ~25,000+ | Early stage (< 500) |
| **Contributors** | 800+ unique contributors | < 10 core |
| **Downloads (Docker Hub)** | 1B+ total pulls | < 1,000 |
| **Releases/Year** | 4-6 major releases | Continuous (main branch) |
| **Open Issues** | ~2,000+ active | < 50 |
| **Mailing List / Forum** | 10,000+ members | None |
| **Stack Overflow Questions** | 20,000+ tagged | 0 |
| **Community Extensions** | 200+ third-party SPIs | 0 (early stage) |

### 1.3 Keycloak's Competitive Moat

Keycloak's moat is not any single feature but the combination of:

1. **Ten years of production hardening** — edge cases, protocol compliance, and
   security patches accumulated across thousands of deployments.
2. **Protocol certifications** — OIDC Certified (multiple profiles), FIDO2/WebAuthn
   certified, SAML 2.0 conformance tested.
3. **Enterprise ecosystem** — RHBK, training, consulting partners, documentation.
4. **Integration breadth** — Spring Security, Quarkus OIDC, Vert.x, Node.js,
   React, Angular adapters, though many are community-maintained.
5. **Knowledge base** — thousands of blog posts, tutorials, conference talks,
   and Stack Overflow answers that make it the "safe default" for open-source IAM.

### 1.4 GGID's Position

GGID is a greenfield IAM platform built in Go, designed to learn from Keycloak's
strengths while avoiding its architectural debt. Its core differentiators:

- **Go microservices** instead of Java monolith (10x lower memory, faster startup)
- **PostgreSQL RLS** for multi-tenant isolation (database-enforced, not application-level)
- **NATS JetStream** for audit event streaming (async, durable, SIEM-ready)
- **gRPC first-class** for all internal services
- **Apache 2.0** with no MAU limits

GGID is early-stage but architecturally sound. Its competitive position is that
of a modern, cloud-native alternative to Keycloak — not a feature-for-feature clone.

---

## 2. Architecture Comparison

### 2.1 High-Level Architecture Diagrams

#### Keycloak Architecture

```
                    ┌─────────────────────────────────────────────────┐
                    │            Keycloak Server (Monolith)            │
                    │                                                   │
                    │  ┌─────────────────────────────────────────────┐ │
                    │  │           Quarkus Application               │ │
                    │  │                                             │ │
                    │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐ │ │
                    │  │  │ Auth     │  │ Admin    │  │ Realms   │ │ │
                    │  │  │ Flows    │  │ Console  │  │ Mgmt     │ │ │
                    │  │  └────┬─────┘  └────┬─────┘  └────┬─────┘ │ │
                    │  │       │              │              │       │ │
                    │  │  ┌────┴──────────────┴──────────────┴────┐  │ │
                    │  │  │          SPI Framework                 │  │ │
                    │  │  │  (Authenticator, User Storage,         │  │ │
                    │  │  │   Event Listener, Protocol Mapper)     │  │ │
                    │  │  └────┬──────────────┬──────────────┬────┘  │ │
                    │  │       │              │              │       │ │
                    │  │  ┌────┴────┐  ┌──────┴───┐  ┌───────┴────┐ │ │
                    │  │  │ JPA     │  │ Infini-  │  │ JTA        │ │ │
                    │  │  │ Storage │  │ span     │  │ Tx Mgmt    │ │ │
                    │  │  └────┬────┘  └────┬─────┘  └────────────┘ │ │
                    │  └───────┼─────────────┼──────────────────────┘ │
                    └──────────┼─────────────┼────────────────────────┘
                               │             │
                    ┌──────────┴────┐  ┌─────┴──────────┐
                    │  PostgreSQL   │  │ Infinispan     │
                    │  / MySQL      │  │ (Clustering,   │
                    │  / MariaDB    │  │  Cache, Sess.) │
                    │  / MSSQL      │  │                │
                    └───────────────┘  └────────────────┘
```

#### GGID Architecture

```
                    ┌─────────────────────────────────────────────────┐
                    │              API Gateway (Go)                    │
                    │  ┌─────────────────────────────────────────┐    │
                    │  │  JWT Validation │ Rate Limit │ CORS     │    │
                    │  │  Tenant Routing │ WASM Plugins │ gRPC  │    │
                    │  └────────────────────┬────────────────────┘    │
                    └───────────────────────┼─────────────────────────┘
                                            │
                ┌───────────┬───────────────┼───────────────┬────────────┐
                │           │               │               │            │
        ┌───────┴───┐ ┌─────┴────┐ ┌────────┴──────┐ ┌─────┴────┐ ┌─────┴─────┐
        │ Auth Svc  │ │Identity  │ │ OAuth/OIDC    │ │ Policy   │ │ Org       │
        │ (Go)      │ │ Svc (Go) │ │ Svc (Go)      │ │ Svc (Go) │ │ Svc (Go)  │
        │ :9001     │ │ :8081    │ │ :9005         │ │ :8070    │ │ :8071     │
        └─────┬─────┘ └────┬─────┘ └───────┬───────┘ └────┬─────┘ └─────┬─────┘
              │            │               │              │             │
              └────────────┴───────────────┼──────────────┴─────────────┘
                                           │
                    ┌──────────────────────┼──────────────────────┐
                    │                      │                      │
            ┌───────┴──────┐      ┌────────┴────────┐   ┌────────┴────────┐
            │ PostgreSQL   │      │ Redis 7         │   │ NATS JetStream  │
            │ 16 (RLS)     │      │ (Sessions,      │   │ (Audit Events,  │
            │              │      │  Rate Limit,     │   │  CIBA Polling,  │
            │              │      │  Token Cache)    │   │  Key Rotation)  │
            └──────────────┘      └─────────────────┘   └────────────────┘
```

### 2.2 Technology Stack Comparison

| Layer | Keycloak | GGID |
|---|---|---|
| **Language** | Java 17+ (Quarkus) | Go 1.25 |
| **Runtime** | JVM (GraalVM native image optional) | Native binary |
| **Application Framework** | Quarkus (CDI, JAX-RS) | Standard library + chi/mux |
| **Database** | PostgreSQL, MySQL, MariaDB, MSSQL | PostgreSQL 16 only |
| **ORM** | JPA / Hibernate | pgx (direct SQL) |
| **Cache** | Infinispan (distributed) | Redis 7 |
| **Messaging** | JMS / JGroups | NATS JetStream |
| **HTTP Server** | Undertow (embedded in Quarkus) | net/http (Go standard library) |
| **Admin Console** | React (new) / Angular (legacy) | Next.js 15 |
| **Clustering** | JGroups + Infinispan distributed cache | None (microservices stateless) |
| **SPI / Extension** | Java ServiceLoader SPI | Go interfaces + WASM plugins |
| **Build Tool** | Maven | Go modules |
| **Container Image** | ~500-800MB | ~20-35MB per service |
| **Startup Time** | 5-30 seconds (JVM), 1-2s (native) | < 1 second |
| **Memory at Idle** | 300-500MB heap minimum | 20-50MB per service |

### 2.3 Architectural Tradeoffs

#### Monolith vs Microservices

**Keycloak's Monolith Advantages:**
- **Operational simplicity**: One process, one container, one log stream. Deploy,
  upgrade, and scale as a unit.
- **No network hops**: All internal calls are in-process method invocations. No
  serialization, no network latency, no partial failure modes between services.
- **Simpler transactions**: JTA distributed transactions span multiple domain
  objects within the same process. No saga patterns or eventual consistency
  needed.
- **Easier debugging**: Single stack trace from request entry to database query.
  No distributed tracing required to understand request flow.
- **Simpler infrastructure**: One health check endpoint, one set of environment
  variables, one deployment manifest.

**Keycloak's Monolith Disadvantages:**
- **Scaling is all-or-nothing**: Cannot scale authentication independently of
  admin console or SAML processing. If login traffic spikes but admin usage
  is low, the entire monolith must scale.
- **Deployment coupling**: A security patch to the SAML module requires
  restarting the entire server, dropping all active sessions.
- **Technology lock-in**: Everything is Java. Cannot use a different language
  for a specialized subsystem (e.g., Rust for crypto, Go for concurrency).
- **Memory overhead**: Even a small deployment pays the full JVM overhead.
  Minimum 300MB heap for production, often 1GB+ under load.

**GGID's Microservices Advantages:**
- **Independent scaling**: Scale the auth service (high traffic, stateless) to
  50 replicas while keeping the audit service (low write volume) at 2 replicas.
- **Fault isolation**: A crash in the SAML parser does not affect OAuth token
  issuance. Each service is in its own process with its own failure domain.
- **Technology diversity**: Each service is Go, but the architecture allows
  future services to be written in other languages if needed.
- **Independent deployment**: Deploy a fix to the OAuth service without
  touching the identity service. Blue-green and canary per service.
- **Right-sized resources**: Each service gets exactly the CPU/memory it needs.
  No shared JVM heap.

**GGID's Microservices Disadvantages:**
- **Operational complexity**: 7 services + 3 infrastructure components = 10
  moving parts to deploy, monitor, and debug. Requires a container
  orchestrator (Kubernetes) for production.
- **Network overhead**: Internal gRPC calls add serialization and network
  latency (~1-5ms per hop). The gateway-to-service-to-database path has
  2 network hops instead of 0.
- **Partial failure modes**: The gateway must handle backend service timeouts,
  circuit breaking, and retries. A failed audit service should not block
  authentication.
- **Distributed transactions**: No ACID transactions across services. The
  system must use saga patterns, outbox patterns, or eventual consistency.
- **Debugging complexity**: Requires distributed tracing (OpenTelemetry) to
  understand request flow across services.

#### Performance: JVM vs Go Binary

**Startup Time:**

| Platform | Cold Start | Warm Restart |
|---|---|---|
| Keycloak (JVM) | 5-30 seconds | 3-10 seconds |
| Keycloak (GraalVM native) | 1-3 seconds | < 1 second |
| GGID (per service) | < 1 second | < 0.5 seconds |

Keycloak's JVM startup is dominated by:
1. Class loading and bytecode verification (~2-5s)
2. CDI container initialization (~1-3s)
3. JPA/Hibernate schema validation (~1-5s)
4. JGroups cluster join (~2-10s if clustering)
5. Infinispan cache initialization (~1-5s)

GGID services start in < 1 second because:
1. Go compiles to a single static binary — no class loading
2. No DI container — dependencies wired manually in `main()`
3. pgx opens a connection pool lazily on first query
4. No clustering protocol to negotiate at startup

**Memory Footprint:**

| Platform | Idle Memory | Under Load (1000 req/s) |
|---|---|---|
| Keycloak (JVM, default) | 400-600MB | 1-2GB |
| Keycloak (JVM, tuned) | 300MB | 800MB-1GB |
| Keycloak (GraalVM native) | 80-150MB | 200-500MB |
| GGID (7 services total) | 140-350MB | 350-700MB |

The memory comparison is nuanced. Keycloak is a single process doing everything;
GGID distributes the same work across 7 processes. At idle, GGID's total footprint
is lower because Go binaries have minimal runtime overhead. Under load, GGID's
advantage grows because Go's goroutine model uses ~2KB per concurrent connection
versus Java's thread-per-request model (1MB default stack per thread) or even
virtual threads (Project Loom).

**Throughput:**

Throughput benchmarks are highly workload-dependent, but general observations:

| Workload | Keycloak | GGID |
|---|---|---|
| Token introspection (cached) | ~5,000-10,000 req/s | ~15,000-30,000 req/s (estimated) |
| Login (DB write) | ~500-1,000 req/s | ~1,000-2,000 req/s (estimated) |
| JWKS serving | ~10,000+ req/s | ~20,000+ req/s (estimated) |

GGID's throughput advantage comes from:
- Go's efficient goroutine scheduler (M:N threading)
- Direct SQL (pgx) instead of ORM (Hibernate)
- No reflection-based serialization (struct tags)
- Lower GC pause times (Go's concurrent GC vs JVM's G1/ZGC)

Note: GGID throughput figures are estimated from architecture analysis, not
measured benchmarks. Keycloak figures are from community benchmarks.

### 2.4 Startup Flow Comparison

**Keycloak Startup Sequence:**

```
1. JVM bootstrap                    [2-5s]
2. Quarkus CDI container init       [1-3s]
3. Database connection pool (Agroal)[1-2s]
4. JPA/Hibernate schema validation  [1-5s]
5. Realm cache warm-up (Infinispan) [1-3s]
6. JGroups cluster join (if HA)     [2-10s]
7. HTTP server bind                 [0.5s]
8. Ready to serve                   ------
Total:                              [8-28s]
```

**GGID Startup Sequence (per service):**

```
1. Binary exec + Go runtime init    [0.05s]
2. Config load (env vars / YAML)    [0.02s]
3. pgxpool creation (lazy connect)  [0.01s]
4. Dependency wiring (manual DI)    [0.01s]
5. HTTP server bind                 [0.01s]
6. Ready to serve                   ------
Total per service:                  [0.1-0.5s]
```

This startup time difference is not just academic. It directly affects:
- **Auto-scaling speed**: GGID can spin up new replicas in < 1s vs Keycloak's 10-30s.
- **CI/CD feedback loops**: Integration tests with GGID are 10-50x faster.
- **Disaster recovery**: GGID services come back online almost instantly after a crash.

---

## 3. Feature Comparison Matrix

### 3.1 Authentication Methods

| # | Feature | Keycloak | GGID | Winner |
|---|---|---|---|---|
| 1 | **Password (bcrypt/argon2)** | Full — PBKDF2, argon2 (24+), configurable iterations | Full — bcrypt + password policy (min length, complexity, history, breach check) | **Tie** |
| 2 | **TOTP MFA** | Full — FreeOTP/Google Authenticator, configurable OTP policies | Full — RFC 6238 TOTP | **Tie** |
| 3 | **SMS MFA** | Partial — via custom Authenticator SPI (Authy, Twilio) | Not implemented | **Keycloak** |
| 4 | **Email OTP MFA** | Partial — via custom flow | Not implemented | **Keycloak** |
| 5 | **Push MFA** | No native support | Not implemented | **Neither** |
| 6 | **WebAuthn / Passkeys** | Full — FIDO2 certified, platform + roaming authenticators | Full — WebAuthn registration + verification (`pkg/webauthn`) | **Keycloak** (certified) |
| 7 | **Biometric (platform authenticator)** | Full — via WebAuthn | Full — via WebAuthn | **Tie** |
| 8 | **Social Login** | Full — configurable IdPs (OIDC, SAML, social) | Full — 9 connectors (Google, GitHub, Microsoft, Apple, Discord, Slack, LinkedIn, GitLab, generic OIDC) | **Tie** |
| 9 | **LDAP / Active Directory** | Full — built-in federation, User Federation SPI, LDIF import | Full — LDAP provider with auto-provision, START-TLS, configurable filters | **Keycloak** (SPI extensibility) |
| 10 | **Kerberos / SPNEGO** | Full — Kerberos/SPNEGO integration for Windows SSO | Not implemented | **Keycloak** |
| 11 | **Magic Link (passwordless)** | Partial — via custom authenticator flow | Not implemented | **Keycloak** |
| 12 | **Passwordless** | Partial — WebAuthn-only flows, configurable | Not implemented | **Keycloak** |
| 13 | **Password breach detection** | Partial — via HaveIBeenPwned SPI (community) | Full — `password_breach.go` checks against HaveIBeenPwned | **GGID** |
| 14 | **Password policy** | Full — regex, length, complexity, history, expiry, not-username | Full — configurable: min length, require upper/lower/digit/symbol, max age, history, breach check | **Tie** |
| 15 | **Custom authentication flows** | Full — Authentication Flow SPI, per-flow steps, conditional sub-flows | Partial — provider chain (Local + LDAP), no visual flow builder | **Keycloak** |

**Authentication Scorecard:**

| Category | Keycloak | GGID |
|---|---|---|
| Total Features | 15 | 15 |
| Full | 8 | 6 |
| Partial | 5 | 1 |
| Not Implemented | 1 (SMS) | 5 (SMS, Email OTP, Kerberos, Magic Link, Passwordless) |
| **Winner** | **Keycloak** | |

Keycloak's authentication depth comes from its Authentication Flow SPI, which allows
administrators to compose arbitrary authentication sequences (e.g., "password OR
WebAuthn OR Kerberos → optional OTP → update profile if changed"). GGID's provider
chain is simpler (ordered list of providers tried sequentially) and lacks the
conditional branching and step composition of Keycloak's flow engine.

### 3.2 Protocols and Standards

| # | Feature | Keycloak | GGID | Winner |
|---|---|---|---|---|
| 16 | **OIDC (OpenID Connect)** | Full — certified OP and RP, all flows (code, implicit, hybrid), OIDC session management, CIBA, backchannel logout | Full — authorization code, PKCE, consent screen, CIBA flow, JWKS, rotating keys | **Keycloak** (certified) |
| 17 | **OAuth 2.0** | Full — all grant types, token exchange (RFC 8693), device authorization | Full — authorization code, client credentials, PKCE, RFC 7592 client management | **Keycloak** (more grant types) |
| 18 | **SAML 2.0** | Full — IdP and SP, signed/encrypted assertions, SLO, metadata generation, certified | Full — IdP and SP, signed assertions, AuthnRequest generation, metadata, SP-initiated SSO | **Keycloak** (encryption, certified) |
| 19 | **SCIM 2.0** | Partial — via third-party `scim-for-keycloak` extension (community-maintained) | Partial — skeleton only (basic user/group endpoints, no filtering/bulk/PATCH) | **Tie** (both incomplete) |
| 20 | **WS-Federation** | Full — passive requestor profile | Not implemented | **Keycloak** |
| 21 | **Token Exchange (RFC 8693)** | Full — built-in token exchange grant type | Not implemented | **Keycloak** |
| 22 | **Device Authorization (RFC 8628)** | Full — device code flow | Not implemented | **Keycloak** |
| 23 | **OAuth 2.1 compliance** | Partial — Keycloak 24+ updated for OAuth 2.1 recommendations | In progress — migration guide exists | **Tie** |
| 24 | **Token Introspection (RFC 7662)** | Full — introspection endpoint | Not implemented (P0 gap) | **Keycloak** |
| 25 | **Backchannel Logout (OIDC)** | Full — backchannel logout endpoint | Not implemented | **Keycloak** |
| 26 | **Front-channel Logout** | Full | Not implemented | **Keycloak** |
| 27 | **JWT (signed)** | Full — RS256, ES256, HS256, PS256 | Full — RS256, ES256, configurable key rotation | **Tie** |
| 28 | **JWKS endpoint** | Full — `/.well-known/jwks.json` | Full — `/.well-known/jwks.json` with rotation | **Tie** |
| 29 | **OIDC Discovery** | Full — `/.well-known/openid-configuration` | Full — `/.well-known/openid-configuration` | **Tie** |
| 30 | **Pushed Authorization Requests (PAR)** | Full — RFC 9126 support (Keycloak 24+) | Not implemented | **Keycloak** |
| 31 | **JWT-Secured Authorization Request (JAR)** | Full — RFC 9101 support | Partial — JAR validation in OAuth service | **Keycloak** |
| 32 | **DPoP (RFC 9449)** | Full — Demonstrating Proof-of-Possession | Not implemented | **Keycloak** |
| 33 | **mTLS client auth** | Full — mutual TLS client certificate bound tokens | Partial — mTLS support in OAuth service | **Keycloak** |

**Protocol Scorecard:**

| Category | Keycloak | GGID |
|---|---|---|
| Total Features | 18 | 18 |
| Full | 14 | 8 |
| Partial | 2 | 3 |
| Not Implemented | 0 | 5 |
| **Winner** | **Keycloak** (clear) | |

Keycloak's protocol coverage is its strongest technical advantage. A decade of
standards compliance work means it supports virtually every OAuth/OIDC extension.
GGID covers the core flows well (authorization code, PKCE, client credentials,
SAML, basic CIBA) but lacks many RFC extensions that enterprise integrations
require.

### 3.3 User Management and Federation

| # | Feature | Keycloak | GGID | Winner |
|---|---|---|---|---|
| 34 | **User CRUD** | Full — admin REST API, user profile (declarative UP) | Full — identity service with REST API | **Tie** |
| 35 | **User attributes / custom fields** | Full — declarative user profile (JSON schema-based) | Full — user metadata (JSON key-values) | **Keycloak** (schema validation) |
| 36 | **Group management** | Full — hierarchical groups, group roles, group attributes | Full — organizations (org service) with memberships | **Tie** |
| 37 | **Role management** | Full — realm roles, client roles, composite roles, role mapping | Full — RBAC roles with inheritance, role-permission mapping, scope types | **Tie** |
| 38 | **User federation (LDAP)** | Full — LDAP/AD federation with sync modes (full, read-only), configurable mappers | Full — LDAP provider with auto-provision, configurable user filter, START-TLS | **Keycloak** (sync modes) |
| 39 | **Custom user storage SPI** | Full — User Storage SPI for custom backends (e.g., custom DB, REST API) | Not implemented (provider chain is auth-only, not storage) | **Keycloak** |
| 40 | **User import/export** | Full — realm export/import (JSON), partial user sync | Partial — no built-in bulk import/export | **Keycloak** |
| 41 | **Brute-force protection** | Full — configurable brute-force detection, temporary lockout, permanent lockout | Full — rate limiter (~5 attempts/min), Redis-based lockout | **Tie** |
| 42 | **Account console (self-service)** | Full — account console (profile, password, MFA, sessions, applications) | Full — Next.js 15 console with 30+ pages (dashboard, users, roles, orgs, audit, settings, sessions, API keys) | **GGID** (modern UI) |
| 43 | **Email verification** | Full — verify email required action, configurable SMTP | Full — email service with templates | **Tie** |
| 44 | **Password reset (forgot password)** | Full — reset via email link, configurable policies | Full — forgot password flow with token | **Tie** |
| 45 | **Identity linking** | Full — link multiple IdPs to one account | Partial — provider chain supports linking via `MustLink` and `LinkedUser` fields | **Keycloak** (admin UI for linking) |

### 3.4 Authorization

| # | Feature | Keycloak | GGID | Winner |
|---|---|---|---|---|
| 46 | **RBAC** | Full — realm/client roles, composite roles, role groups | Full — role-permission model, role inheritance, scoped assignments (global/org/dept/team/resource) | **GGID** (scope-based) |
| 47 | **ABAC** | Partial — via JavaScript policies in Authorization Services | Full — AWS IAM-style policies with conditions, effect (allow/deny), priority | **GGID** (simpler model) |
| 48 | **UMA 2.0 (User-Managed Access)** | Full — resources, scopes, permissions, policies, ticket-based authz | Not implemented | **Keycloak** |
| 49 | **Fine-grained authorization** | Full — Authorization Services (resources, scopes, policies, permissions, JS policies) | Partial — policy engine evaluates conditions but lacks resource/scope model | **Keycloak** |
| 50 | **Policy evaluation API** | Full — Authorization Services REST API + token endpoint with authz claims | Full — policy evaluation endpoint (REST + gRPC), CheckRequest/CheckResult | **Tie** |
| 51 | **Resource-based protection** | Full — protect paths/resources with permission tickets | Not implemented | **Keycloak** |
| 52 | **Group-based roles** | Full — assign roles to groups, inherited by members | Full — role assignments scoped to org/dept/team/resource | **Tie** |

### 3.5 Administration and APIs

| # | Feature | Keycloak | GGID | Winner |
|---|---|---|---|---|
| 53 | **Admin REST API** | Full — comprehensive CRUD for all resources (realms, clients, users, roles, groups, IdPs, events, sessions) | Full — REST API for all services via gateway (auth, identity, policy, org, audit, oauth) | **Tie** |
| 54 | **gRPC API** | No — REST only (Quarkus gRPC extension exists but not used for admin) | Full — gRPC for policy, org, audit services | **GGID** |
| 55 | **GraphQL API** | No | Partial — GraphQL resolver in gateway middleware | **GGID** |
| 56 | **API versioning** | Full — admin API versioned | Full — `/api/v1/` with deprecation middleware | **Tie** |
| 57 | **OpenAPI/Swagger** | Full — generated OpenAPI docs | Full — OpenAPI spec at `/api-docs`, Swagger UI at `/docs` | **Tie** |
| 58 | **Rate limiting** | Partial — built-in rate limiter or Infinispan-based | Full — tiered rate limiting, tenant-aware, sliding window, per-route config | **GGID** |
| 59 | **Webhooks** | Partial — event listener SPI (push to external HTTP) | Partial — webhook events documented but no native webhook delivery | **Tie** |
| 60 | **Themes/UI customization** | Full — theme SPI (HTML/CSS/JS themes for login, account, admin, email) | Partial — brand customization docs, hosted login/register pages in gateway templates | **Keycloak** |
| 61 | **Internationalization (i18n)** | Full — 30+ language bundles, theme-specific messages | Full — i18n package (`pkg/i18n`), console with en/zh message bundles | **Tie** |
| 62 | **Email templates** | Full — FreeMarker templates for all email types | Full — email template system (`pkg/email`) | **Tie** |
| 63 | **Event logging** | Full — user events, admin events, event listener SPI | Full — all events published to NATS JetStream + audit service | **GGID** (async, streaming) |
| 64 | **Admin audit log** | Full — admin events stored in DB | Full — audit service with REST query API | **Tie** |
| 65 | **Metrics (Prometheus)** | Full — Micrometer + Prometheus endpoint | Full — Prometheus metrics at `/metrics` | **Tie** |
| 66 | **Distributed tracing (OpenTelemetry)** | Full — OTel integration (Keycloak 24+) | Full — OpenTelemetry middleware (`otel.go`) | **Tie** |

### 3.6 Deployment and Operations

| # | Feature | Keycloak | GGID | Winner |
|---|---|---|---|---|
| 67 | **Docker image** | Full — official `quay.io/keycloak/keycloak` | Full — Docker Compose with all 7 services | **Tie** |
| 68 | **Kubernetes operator** | Full — official Keycloak Operator | Not available (Docker Compose only) | **Keycloak** |
| 69 | **Helm chart** | Full — community charts (codecentric, Bitnami) | Not available | **Keycloak** |
| 70 | **Database support** | Full — PostgreSQL, MySQL, MariaDB, MSSQL, Oracle | PostgreSQL 16 only | **Keycloak** |
| 71 | **Clustering / HA** | Full — JGroups + Infinispan, active-active clustering | Not configured (architecture supports it but no K8s manifests) | **Keycloak** |
| 72 | **High availability** | Full — multi-node clusters with shared DB | Not available | **Keycloak** |
| 73 | **Backup/recovery** | Full — DB-backed, standard DB backup procedures | Partial — backup docs exist, no automated tooling | **Keycloak** |
| 74 | **Terraform provider** | Full — `mrparkers/keycloak` Terraform provider | Not available | **Keycloak** |
| 75 | **CI/CD realm import/export** | Full — realm JSON export/import at startup | Not available | **Keycloak** |
| 76 | **Image size** | ~500-800MB (JVM) | ~20-35MB per service (total ~200MB) | **GGID** |
| 77 | **Container startup** | 5-30 seconds | < 1 second per service | **GGID** |
| 78 | **Memory at idle** | 300-600MB | 140-350MB (all 7 services) | **GGID** |
| 79 | **Horizontal scaling** | Full — cluster mode, stateless nodes | Full — microservices are stateless, scale independently | **GGID** (independent scaling) |
| 80 | **Monitoring** | Full — JMX, Micrometer, Prometheus, Grafana | Full — Prometheus, health checks, OpenTelemetry | **Tie** |
| 81 | **Multi-database** | Full — 5 database engines | PostgreSQL only (intentional: RLS is PostgreSQL-specific) | **Keycloak** (flexibility) / **GGID** (simplicity) |
| 82 | **Database migrations** | Full — Liquibase (automatic schema upgrade) | Full — SQL migration files per service | **Tie** |

### 3.7 Scorecard Summary

| Category | Keycloak | GGID | Winner |
|---|---|---|---|
| Authentication Methods | 13/15 | 8/15 | **Keycloak** |
| Protocols & Standards | 16/18 | 11/18 | **Keycloak** |
| User Management | 11/12 | 9/12 | **Keycloak** (slight) |
| Authorization | 6/7 | 4/7 | **Keycloak** (UMA) |
| Administration & APIs | 9/14 | 12/14 | **GGID** (gRPC, GraphQL, rate limiting) |
| Deployment & Ops | 11/16 | 8/16 | **Keycloak** (K8s, HA, Terraform) |
| **Total** | **66/82** | **52/82** | **Keycloak** |

Keycloak wins on feature breadth — it has been building features for 10+ years.
GGID wins on modern architecture, developer experience (gRPC, GraphQL, Go), and
operational efficiency (memory, startup, image size). The strategic question for
GGID is which of Keycloak's features are table-stakes (must-have for enterprise
adoption) versus long-tail features that only matter for specific use cases.

---

## 4. Multi-Tenancy (Realms)

### 4.1 Keycloak Realm Model

Keycloak's multi-tenancy is built on the concept of **realms**. A realm is a
fully isolated tenant with its own:

- Users, credentials, and sessions
- Roles and groups
- Clients (applications)
- Identity providers (OIDC, SAML, social)
- Authentication flows
- Themes (login, account, admin, email)
- Events and audit log
- SMTP configuration
- Required actions (verify email, update profile, etc.)
- Authorization services (resources, scopes, policies)

**Realm Isolation Characteristics:**

| Property | Implementation |
|---|---|
| **Data isolation** | Each realm's data is in the same database but separated by `realm_id` column (JPA queries always filter by realm). Optionally, different databases per realm (advanced configuration). |
| **Configuration isolation** | Each realm has completely independent configuration. No shared state between realms. |
| **Session isolation** | Sessions are scoped to a realm. SSO only works within the same realm. |
| **URL isolation** | Each realm has its own URL prefix (`/realms/{realm-name}/`). Custom domains per realm supported. |
| **Key isolation** | Each realm has its own signing keys (RS256/ES256). No key sharing across realms. |
| **Theme isolation** | Each realm can have a completely different look and feel. |
| **Event isolation** | Events are logged per-realm. No cross-realm visibility without admin API queries. |

**Realm Management:**

```bash
# Create a realm via admin REST API
curl -X POST http://localhost:8080/admin/realms \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "realm": "acme-corp",
    "enabled": true,
    "sslRequired": "external",
    "loginTheme": "acme-brand",
    "eventsEnabled": true,
    "enabledEventTypes": ["LOGIN", "LOGOUT", "REGISTER"]
  }'

# Configure realm-specific LDAP federation
curl -X POST http://localhost:8080/admin/realms/acme-corp/components \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "acme-ldap",
    "providerId": "ldap",
    "providerType": "org.keycloak.storage.UserStorageProvider",
    "config": {
      "connectionUrl": ["ldap://ldap.acme.com:389"],
      "bindDn": ["cn=admin,dc=acme,dc=com"],
      "bindCredential": ["secret"],
      "baseDn": ["ou=users,dc=acme,dc=com"]
    }
  }'
```

**Realm Advantages:**
- **Complete tenant autonomy**: Each tenant's admin can configure their own auth
  flows, themes, and IdPs without affecting other tenants.
- **No shared schema risk**: A bug in realm isolation means one tenant sees
  another's data — Keycloak uses `realm_id` filtering everywhere, but it's
  application-level, not database-level.
- **Per-tenant key rotation**: Each realm can rotate its signing keys independently.

**Realm Disadvantages:**
- **Database proliferation**: Each realm adds rows to shared tables. With 1,000
  realms and 100K users per realm, the `USER_ENTITY` table has 100M rows. Queries
  are filtered by `realm_id` but there's no physical partitioning.
- **No cross-realm operations**: Cannot query users across realms in a single
  API call. No "global admin" view of all tenants.
- **Operational complexity at scale**: Thousands of realms require careful
  management of cache (Infinispan local-only cache per node), event storage,
  and key management.
- **Per-realm schema**: When Keycloak upgrades, it runs Liquibase migrations that
  affect all realms. A schema change is global.

### 4.2 GGID Tenant Model

GGID's multi-tenancy is fundamentally different. Instead of realm-level isolation
at the application layer, GGID uses **PostgreSQL Row-Level Security (RLS)** for
database-enforced tenant isolation.

**GGID Tenant Implementation:**

From `pkg/tenant/tenant.go`:

```go
type IsolationLevel string

const (
    IsolationShared   IsolationLevel = "shared"   // All tenants share one DB, isolated by RLS
    IsolationSchema   IsolationLevel = "schema"   // Tenant gets a dedicated PostgreSQL schema
    IsolationDatabase IsolationLevel = "database"  // Tenant gets a dedicated database instance
)

type Context struct {
    TenantID       uuid.UUID
    IsolationLevel IsolationLevel
    SchemaName     string           // for schema-level isolation
    Settings       map[string]any   // tenant-specific settings
}
```

The tenant context is propagated through every request:

1. **Gateway**: Extracts `tenant_id` from JWT claims (priority) or `X-Tenant-ID`
   header, attaches to request context.
2. **Each service**: Extracts tenant from context, sets PostgreSQL session variable
   `app.tenant_id`.
3. **PostgreSQL RLS policies**: Every table has an RLS policy like:
   ```sql
   CREATE POLICY tenant_isolation ON users
     USING (tenant_id = current_setting('app.tenant_id')::uuid);
   ```
4. **Database enforcement**: Even if a service has a bug (e.g., missing WHERE
   clause), PostgreSQL blocks cross-tenant data access at the database level.

**Tenant Context Propagation in Gateway:**

From `services/gateway/internal/router/router.go`:

```go
proxy.Director = func(req *http.Request) {
    originalDirector(req)
    if userID, ok := middleware.UserIDFromRequest(req); ok {
        req.Header.Set("X-User-ID", userID.String())
    }
    if tenantID, ok := middleware.TenantIDFromRequest(req); ok {
        req.Header.Set("X-Tenant-ID", tenantID)
        // Inject as query param for GET requests
        q := req.URL.Query()
        if q.Get("tenant_id") == "" {
            q.Set("tenant_id", tenantID)
            req.URL.RawQuery = q.Encode()
        }
        // Inject into JSON body for POST/PUT/PATCH requests
        injectTenantIntoBody(req, tenantID)
    }
}
```

**GGID Tenant Advantages:**
- **Database-level enforcement**: RLS is enforced by PostgreSQL, not application
  code. A bug in a SQL query cannot leak cross-tenant data — PostgreSQL blocks
  it at the storage layer.
- **No schema duplication**: All tenants share the same tables. Adding a new
  tenant is just inserting a row in the `tenants` table. No realm creation
  ceremony.
- **Shared queries**: Can query across tenants (with explicit superadmin
  context) — useful for platform operators who need a global view.
- **Simpler at scale**: 10,000 tenants in one database is fine — RLS adds a
  simple WHERE clause, not separate tables.
- **Flexible isolation**: Can upgrade from shared → schema → database isolation
  per tenant if needed (e.g., for high-security tenants).

**GGID Tenant Disadvantages:**
- **No per-tenant configuration**: Currently, all tenants share the same auth
  provider chain, themes, and IdP configurations. No per-tenant LDAP, per-tenant
  SAML IdP, or per-tenant theme.
- **No tenant management API**: No REST endpoint to create/list/delete tenants.
  Tenant creation requires database-level operations.
- **No per-tenant keys**: All tenants share the same JWT signing key. Key
  rotation affects all tenants simultaneously.
- **No custom domains**: No per-tenant URL routing or custom domain support.
- **No per-tenant rate limiting**: While tier-based rate limiting exists
  (`tier_ratelimit.go`), it's not fully per-tenant configurable via API.

### 4.3 Multi-Tenancy Comparison Matrix

| Capability | Keycloak Realms | GGID RLS |
|---|---|---|
| **Data isolation mechanism** | Application-level (JPA realm_id filter) | Database-level (PostgreSQL RLS) |
| **Isolation strength** | Medium (app bug can leak data) | **High** (DB blocks cross-tenant access) |
| **Per-tenant configuration** | **Full** (independent auth flows, themes, IdPs, SMTP) | Minimal (auth provider chain only) |
| **Per-tenant keys** | Yes (independent key pairs per realm) | No (shared keys) |
| **Per-tenant themes** | Yes (full HTML/CSS/JS themes) | No (shared hosted pages) |
| **Per-tenant IdPs** | Yes (independent OIDC/SAML/LDAP per realm) | No (global auth provider chain) |
| **Tenant management API** | Yes (admin REST API for realm CRUD) | No |
| **Custom domains** | Yes (per-realm URL) | No |
| **Cross-tenant queries** | No (realm-scoped queries only) | Yes (superadmin context) |
| **Scalability (10K+ tenants)** | Challenging (shared tables, cache pressure) | Good (RLS adds WHERE clause, no extra objects) |
| **Operational complexity** | Medium (realm export/import, per-realm config) | Low (just a tenant_id column) |
| **Upgrade impact** | Global (all realms affected) | Global (all tenants share schema) |
| **Multi-database support** | No (single database, multi-realm) | Designed for (shared/schema/database tiers) |

### 4.4 Analysis

Keycloak's realm model is the gold standard for **tenant autonomy** — each
tenant gets a fully self-service environment. This is ideal for SaaS platforms
where each customer (tenant) needs to configure their own identity providers,
authentication flows, and branding.

GGID's RLS model is the gold standard for **data security** — the database
itself enforces isolation, making cross-tenant data leakage virtually impossible.
This is ideal for compliance-heavy environments (healthcare, finance) where
data isolation is a regulatory requirement.

**GGID's multi-tenancy gap is significant for SaaS use cases.** Without
per-tenant IdP configuration, per-tenant themes, and a tenant management API,
GGID cannot serve the "B2B SaaS where each customer brings their own IdP"
use case that Keycloak handles well.

---

## 5. Identity Brokering

### 5.1 Keycloak's Identity Brokering

Identity brokering is one of Keycloak's most powerful features. It allows
Keycloak to act as an intermediary (broker) between users and external identity
providers, translating between protocols and unifying identity.

**Key Brokering Capabilities:**

1. **Protocol Translation**: A user authenticates via SAML IdP, Keycloak issues
   an OIDC token to the application. The application only needs to speak OIDC.
   Keycloak handles the SAML-to-OIDC conversion internally.

2. **Chained Brokering**: User → Keycloak Realm A → Keycloak Realm B → External
   SAML IdP. Each hop translates the identity without re-authentication.

3. **Social Login Brokering**: Built-in connectors for Google, GitHub, Microsoft,
   Facebook, Instagram, GitLab, PayPal, Apple, Twitter/X, Stack Overflow, and
   any OIDC/SAML IdP. Each connector handles the OAuth2/OIDC dance and maps
   claims to Keycloak user attributes.

4. **First Broker Login Flow**: When a user first authenticates via a broker,
   Keycloak runs a configurable flow that can:
   - Create a new local user (JIT provisioning)
   - Link to an existing user (if email matches)
   - Require manual account linking
   - Deny access

5. **Post Broker Login Flow**: After broker authentication, run additional
   steps (e.g., require OTP, update profile, check custom conditions).

6. **Token Exchange via Broker**: Exchange an external IdP token for a Keycloak
   token using the token exchange grant type.

7. **Stored Tokens**: Keycloak can store the external IdP's access/refresh tokens
   for later use (e.g., to call Google APIs on behalf of the user).

8. **SAML Brokering**: Keycloak acts as a SAML SP to an external SAML IdP and
   as a SAML IdP to downstream SAML SPs — full SAML-to-SAML brokering.

**Keycloak Brokering Architecture:**

```
┌──────────┐     OIDC      ┌──────────┐     SAML      ┌──────────┐
│   App    │ ←──────────→  │ Keycloak │ ←──────────→  │  SAML    │
│ (Client) │               │  Realm   │               │   IdP    │
└──────────┘               └────┬─────┘               └──────────┘
                                │
                           OIDC/LDAP
                                │
                           ┌────┴─────┐
                           │  Active  │
                           │Directory │
                           └──────────┘
```

The application only sees OIDC. Keycloak translates between all protocols
internally.

### 5.2 GGID's Identity Federation

GGID has solid social login support and SAML capabilities, but its identity
brokering is less mature than Keycloak's.

**GGID Social Login (`pkg/social/`):**

From the source code, GGID implements a Connector interface:

```go
type Connector interface {
    ID() string
    DisplayName() string
    GetAuthURL(ctx context.Context, state string, redirectURI string) (string, error)
    HandleCallback(ctx context.Context, code string, state string, redirectURI string) (*UserInfo, error)
}
```

Implemented connectors:
- **Google** — OAuth2 with OpenID Connect
- **GitHub** — OAuth2
- **Microsoft** — OAuth2 with Microsoft Graph
- **Apple** — OAuth2 with Sign in with Apple
- **Discord** — OAuth2
- **Slack** — OAuth2
- **LinkedIn** — OAuth2
- **GitLab** — OAuth2
- **Generic OIDC** — any OpenID Connect provider

**GGID SAML Support (`pkg/saml/`):**

GGID implements SAML 2.0 as both IdP and SP:
- `sp.go` — Service Provider configuration, AuthnRequest generation, metadata
- `signed_assertion.go` — Signed assertion parsing and verification
- `idp_initiated.go` — IdP-initiated SSO flow
- `assertion.go` — SAML assertion types

**GGID Auth Provider Chain (`pkg/authprovider/`):**

```go
type Chain struct {
    providers []Provider
}

func (c *Chain) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
    var lastErr error
    for _, p := range c.providers {
        result, err := p.Authenticate(ctx, creds)
        if err == nil {
            return result, nil
        }
        lastErr = err
    }
    return nil, lastErr
}
```

The chain tries providers sequentially: Local → LDAP → (future: OIDC, SAML).
The `AuthResult` includes JIT provisioning signals:

```go
type AuthResult struct {
    ExternalID  string         // ID in external system (LDAP DN, OIDC sub)
    Provider    ProviderType
    Attributes  map[string]any // synced attributes
    MustLink    bool           // needs linking to a local account
    NewUser     bool           // first-time login, JIT provisioning
    LinkedUser  *uuid.UUID     // already linked
    Roles       []string       // mapped roles
}
```

### 5.3 Brokering Gap Analysis

| Capability | Keycloak | GGID | Gap Severity |
|---|---|---|---|
| Protocol translation (SAML ↔ OIDC) | Full | Not implemented | **High** — critical for enterprise |
| Chained brokering (multi-hop) | Full | Not implemented | Medium |
| Social login (9+ providers) | Full (30+) | Full (9 connectors) | Low (core providers covered) |
| JIT provisioning on first login | Full (configurable flow) | Partial (AuthResult.NewUser flag exists) | Medium |
| Account linking (existing user) | Full (email-based linking, manual linking UI) | Partial (MustLink/LinkedUser fields exist) | Medium |
| First broker login flow | Full (configurable steps) | Not implemented | **High** |
| Post broker login flow | Full (configurable steps) | Not implemented | Medium |
| Token storage (external IdP tokens) | Full (stores access/refresh tokens per IdP) | Not implemented | Medium |
| SAML-to-OIDC conversion | Full (broker translates internally) | Not implemented | **High** |
| SAML-to-SAML brokering | Full | Not implemented | Medium |
| OIDC-to-SAML brokering | Full | Not implemented | Medium |
| Per-tenant IdP configuration | Full (each realm has independent IdPs) | Not implemented | **Critical** for SaaS |
| Home realm discovery | Full (email domain → realm mapping) | Not implemented | Medium |
| IdP redirector (choose IdP based on hints) | Full | Not implemented | Low |

### 5.4 Recommendations

The most impactful brokering gaps for GGID are:

1. **SAML ↔ OIDC protocol translation** — This is what makes Keycloak the "universal
   identity hub." Enterprises want their apps to speak one protocol (OIDC) while
   Keycloak/GGID federates with whatever IdP the customer uses (SAML, LDAP, etc.).

2. **Per-tenant IdP configuration** — For SaaS platforms, each customer (tenant)
   needs to configure their own Azure AD / Okta / SAML IdP. Without this, GGID
   cannot serve the B2B SaaS identity federation use case.

3. **First Broker Login flow** — Configurable steps for what happens when a user
   first authenticates via an external IdP (JIT vs. deny vs. link).

---

## 6. Authorization Services (UMA 2.0)

### 6.1 Keycloak Authorization Services

Keycloak's Authorization Services is a full-featured fine-grained authorization
system built on top of the User-Managed Access (UMA) 2.0 specification. It goes
far beyond simple RBAC.

**Core Concepts:**

1. **Resources**: Protected objects (e.g., "User Account", "Financial Report",
   "Bank Account:12345"). Resources have types, URIs, and optional icons.

2. **Scopes**: Fine-grained actions on resources (e.g., "read", "write", "delete",
   "execute"). A resource can have multiple scopes.

3. **Policies**: Rules that evaluate to allow or deny:
   - **Role-based policy**: Allow if user has role X.
   - **User-based policy**: Allow if user is in the allowed list.
   - **Time-based policy**: Allow only between 9am-5pm.
   - **JavaScript policy**: Custom logic in JavaScript (evaluated by Nashorn/ GraalVM).
   - **Aggregated policy**: Combine multiple policies (AND, OR, NOT).
   - **Client-based policy**: Allow if client (application) matches.
   - **Group-based policy**: Allow if user is in group X.
   - **Regex-based policy**: Match resource URI against regex.

4. **Permissions**: Link policies to resources/scopes:
   - **Resource-based permission**: Apply policies to a specific resource.
   - **Scope-based permission**: Apply policies to a specific scope.
   - **Resource-type-based permission**: Apply policies to all resources of a type.

5. **Permission Tickets (UMA)**: When a user tries to access a resource they
   don't have permission for, Keycloak issues a permission ticket. The resource
   owner can then grant access, creating a dynamic authorization flow.

**Keycloak Authorization Flow:**

```
User → App → Keycloak Authz endpoint
                      │
                      ├─ Evaluate all applicable policies
                      │   ├─ Role policies
                      │   ├─ Time policies
                      │   ├─ JavaScript policies
                      │   └─ Aggregated policies
                      │
                      ├─ Decision: ALLOW or DENY
                      │
                      └─ Return RPT (Requesting Party Token)
                         with authorization claims
```

**JavaScript Policy Example (Keycloak):**

```javascript
var context = $evaluation.getContext();
var identity = context.getIdentity();
var attributes = identity.getAttributes();
var role = attributes.getValue('role').asString();

if (role === 'admin') {
    $evaluation.grant();
} else {
    var resource = context.getResource();
    var ownerId = resource.getOwner();
    var userId = identity.getId();

    if (ownerId === userId) {
        $evaluation.grant();
    } else {
        $evaluation.deny();
    }
}
```

### 6.2 GGID Policy Engine

GGID's policy engine (`services/policy/`) implements a hybrid RBAC + ABAC model
inspired by AWS IAM policies.

**Core Domain Model:**

From `services/policy/internal/domain/models.go`:

```go
// RBAC
type Role struct {
    ID           uuid.UUID
    TenantID     uuid.UUID
    Key          string
    Name         string
    Description  string
    SystemRole   bool
    ParentRoleID *uuid.UUID   // role inheritance
}

type Permission struct {
    ID           uuid.UUID
    TenantID     uuid.UUID
    Key          string
    Name         string
    ResourceType string
    Action       string       // e.g., "read", "write", "delete"
}

type UserRole struct {
    UserID    uuid.UUID
    RoleID    uuid.UUID
    ScopeType ScopeType       // global, organization, department, team, resource
    ScopeID   uuid.UUID
    ExpiresAt *time.Time
}

// ABAC
type Policy struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    Name        string
    Description string
    Effect      Effect          // "allow" or "deny"
    Actions     []string        // e.g., ["s3:GetObject", "iam:ListUsers"]
    Resources   []string        // e.g., ["arn:ggid:s3:::my-bucket/*"]
    Conditions  map[string]any  // e.g., {"ip": "10.0.0.0/8"}
    Priority    int             // higher = evaluated first
}

// Evaluation
type CheckRequest struct {
    UserID       uuid.UUID
    TenantID     uuid.UUID
    ResourceType string
    Action       string
    Resource     string
    Conditions   map[string]any
}

type CheckResult struct {
    Allowed   bool
    Reason    string
    MatchedBy string  // e.g., "role:admin" or "policy:deny-sensitive"
}
```

**GGID Policy Evaluation:**

The policy service evaluates requests through a two-stage process:

1. **RBAC check**: Does the user have a role that grants this permission (via
   role-permission mapping with optional conditions)?
2. **ABAC check**: Do any policies explicitly allow or deny this action?
   - Deny policies are evaluated first (higher default priority: 100)
   - Allow policies are evaluated if no deny matches
   - If both match, the higher priority wins

### 6.3 Authorization Comparison

| Capability | Keycloak Authz Services | GGID Policy Engine |
|---|---|---|
| **RBAC** | Realm/client roles, composite roles, role groups | Roles with inheritance, scoped assignments (global/org/dept/team/resource) |
| **ABAC** | JavaScript policies, attribute-based conditions | AWS IAM-style policies with conditions (IP, time, etc.) |
| **UMA 2.0** | Full — resources, scopes, permissions, tickets | Not implemented |
| **Resource model** | Resources with URIs, types, scopes | Permissions with resource_type + action |
| **Policy languages** | JavaScript (Nashorn/GraalVM), Drools (legacy) | Declarative JSON (Conditions map) |
| **Dynamic authorization** | Permission tickets (owner grants access at runtime) | Not implemented |
| **Scope-based authz** | Full (scopes per resource) | Not implemented |
| **Policy aggregation** | AND, OR, NOT policy composition | Priority-based (deny > allow, configurable priority) |
| **Time-based policies** | Full (time-range conditions) | Partial (conditions map supports time values) |
| **IP-based policies** | Via JavaScript policy | Full (Conditions can specify IP CIDR) |
| **Resource ownership** | Full (resources have owners, owners can grant access) | Not implemented |
| **Policy evaluation API** | REST API + token endpoint with authz claims | REST + gRPC CheckRequest/CheckResult |
| **Admin UI for policies** | Full (React admin console) | Partial (console policy management pages) |
| **JavaScript policies** | Full (GraalVM sandboxed) | Not implemented |
| **Token-based authz claims** | RPT (Requesting Party Token) with permissions | Not implemented |

### 6.4 Analysis

Keycloak's Authorization Services is significantly more mature and feature-rich.
The UMA 2.0 model (resources → scopes → permissions → policies) provides a
complete authorization framework that can handle complex enterprise scenarios
like:

- "Only the owner of a document can delete it"
- "Users in the Finance group can read financial reports during business hours"
- "API access requires a valid scope token with `report:read` scope"

GGID's policy engine is simpler and more developer-friendly (AWS IAM-style
declarative JSON vs JavaScript policies), but it lacks the resource/scope model
and dynamic authorization (permission tickets) that Keycloak provides.

**For most use cases**, GGID's RBAC + ABAC model is sufficient. The AWS IAM-style
policy is intuitive and powerful. The gap matters for:

- **Healthcare** — patient consent management (UMA permission tickets)
- **Collaborative apps** — document sharing with owner-based grants
- **API marketplaces** — scope-based API access control

---

## 7. Plugin Ecosystem

### 7.1 Keycloak SPI (Service Provider Interface)

Keycloak's extensibility is built on Java's ServiceLoader SPI mechanism. There
are 30+ SPI extension points:

**Major SPI Extension Points:**

| SPI | Purpose | Example Use Case |
|---|---|---|
| **Authenticator SPI** | Custom authentication steps | SMS OTP, biometric, risk-based auth |
| **User Storage SPI** | Custom user backends | REST API-backed user store, custom DB |
| **Credential Provider SPI** | Custom credential types | Hardware security keys, custom OTP |
| **Event Listener SPI** | Custom event handling | Push to Splunk, custom audit log |
| **Protocol Mapper SPI** | Custom token claims | Add custom claims to JWT/SAML |
| **Identity Provider SPI** | Custom social/enterprise IdP | Custom OIDC provider, proprietary SSO |
| **Realm Resource SPI** | Custom admin REST endpoints | Tenant-specific admin operations |
| **Theme Resource SPI** | Custom themes | Branded login pages, email templates |
| **Required Action SPI** | Custom required actions | Force password change, accept terms |
| **Client Registration SPI** | Custom client registration policies | Restrict redirect URIs, enforce naming |
| **Email Template SPI** | Custom email templates | Localized emails, branded templates |
| **Export/Import SPI** | Custom realm import/export | Migrate from proprietary system |
| **JPA Entity SPI** | Custom database entities | Add custom tables managed by Keycloak |
| **PublicKeyStorage SPI** | Custom key storage | HSM-backed key management |
| **Client Policy SPI** | Client registration/validation policies | Enforce client configuration standards |

**SPI Development Experience:**

Creating a Keycloak SPI requires:

1. Java project with Keycloak dependencies (Maven/Gradle)
2. Implement the SPI interface (e.g., `Authenticator`)
3. Register via `META-INF/services/` file
4. Package as JAR
5. Deploy to `providers/` directory or mount as volume
6. Configure via admin console or `keycloak.json`

**Example: Custom SMS Authenticator SPI:**

```java
public class SmsAuthenticator implements Authenticator {
    @Override
    public void authenticate(AuthenticationFlowContext context) {
        String phoneNumber = context.getUser().getFirstAttribute("phone");
        String code = generateOtp();
        sendSms(phoneNumber, code);
        context.getAuthenticationSession().setAuthNote("sms_code", code);
        context.challenge(context.form().createForm("sms-otp.ftl"));
    }

    @Override
    public void action(AuthenticationFlowContext context) {
        String submittedCode = context.getHttpRequest().getDecodedFormParameters()
            .getFirst("otp_code");
        String expectedCode = context.getAuthenticationSession().getAuthNote("sms_code");
        if (submittedCode.equals(expectedCode)) {
            context.success();
        } else {
            context.failureChallenge(AuthenticationFlowError.INVALID_CREDENTIALS,
                context.form().setError("Invalid code").createForm("sms-otp.ftl"));
        }
    }
    // ... other methods
}
```

### 7.2 GGID Extension Points

GGID's extensibility model is fundamentally different — Go interfaces instead of
Java SPI, plus WASM plugins for sandboxed extensions.

**Go Interface Extension Points:**

| Interface | Package | Purpose |
|---|---|---|
| `authprovider.Provider` | `pkg/authprovider` | Custom auth backends (local, LDAP, OIDC, SAML) |
| `social.Connector` | `pkg/social` | Custom social login providers |
| `domain.KeyProvider` | `services/oauth` | Custom JWT signing key management |
| `PolicyRepo` | `services/policy` | Custom policy storage backend |
| `CredentialRepo` | `services/auth` | Custom credential storage |

**WASM Plugin System:**

From `services/gateway/internal/middleware/wasm_plugin.go`:

```go
type WasmPluginConfig struct {
    Name     string            `json:"name"`
    WasmPath string            `json:"wasm_path"`
    Config   map[string]string `json:"config"`
    Enabled  bool              `json:"enabled"`
}

type WasmPluginPhase string
const (
    PhaseRequest  WasmPluginPhase = "request"  // before proxying to backend
    PhaseResponse WasmPluginPhase = "response" // after receiving backend response
)
```

GGID's WASM plugin host:
- Uses Wazero (pure Go WebAssembly runtime, no CGO)
- Plugins are sandboxed — no network or filesystem access beyond what the host provides
- Plugins can run in request phase (modify request before forwarding) or response phase
  (modify response before returning to client)
- Plugins expose `init()` function that returns metadata (name, version, hooks)
- Plugin context provides request/response data as JSON

**WASM Plugin Development:**

```go
// Plugin is compiled to .wasm from any language (Rust, Go, AssemblyScript, C)
// The host calls the plugin's process() function with a JSON context

// PluginContext (passed to plugin):
type PluginContext struct {
    Method      string            `json:"method"`
    Path        string            `json:"path"`
    Headers     map[string]string `json:"headers"`
    Body        string            `json:"body"`
    StatusCode  int               `json:"status_code,omitempty"`
    Phase       string            `json:"phase"`
}
```

### 7.3 Plugin Ecosystem Comparison

| Aspect | Keycloak SPI | GGID Interfaces + WASM |
|---|---|---|
| **Language** | Java only | Go interfaces (Go plugins) + WASM (any language) |
| **Extension points** | 30+ SPI types | 5 Go interfaces + WASM request/response hooks |
| **Sandboxing** | None (SPI runs in-process with full JVM access) | WASM plugins are sandboxed (no network/FS access) |
| **Deployment** | JAR in providers/ directory | .wasm file + config JSON |
| **Runtime isolation** | None (SPI shares JVM with Keycloak) | WASM module is isolated from host process |
| **Plugin languages** | Java only | Any language compiling to WASM (Rust, Go, C, AssemblyScript, Zig) |
| **Security risk** | High (malicious SPI can access DB, filesystem, network) | Low (WASM sandbox restricts access) |
| **Performance** | Native (in-process, no serialization) | Overhead (WASM serialization + sandbox boundary) |
| **Plugin marketplace** | ~200+ community extensions | None (early stage) |
| **Documentation** | Extensive SPI docs, examples | Partial (`plugin-development.md`, `plugin-api-reference.md`) |
| **Developer experience** | Mature (Maven archetypes, IntelliJ support) | Early (manual WASM compilation) |

### 7.4 Analysis

Keycloak's SPI ecosystem is vastly more mature — 30+ extension points, 200+
community extensions, and a decade of documentation. This is a significant
competitive moat. Enterprises often choose Keycloak specifically because they
need a custom SPI (e.g., a proprietary user store, a specific SMS provider).

GGID's WASM approach is architecturally superior for security (sandboxed plugins)
and language flexibility (any WASM-compatible language), but the ecosystem is
non-existent. The WASM approach is better suited for request/response middleware
plugins than for deep authentication flow customization.

**For GGID to compete**, it should:
1. Expand Go interface extension points (event listeners, protocol mappers, required actions)
2. Document the WASM plugin SDK thoroughly
3. Create a plugin marketplace or registry
4. Provide starter templates for common plugin types (SMS auth, custom event sink, protocol mapper)

---

## 8. Community & Documentation

### 8.1 Keycloak Documentation

Keycloak has extensive documentation developed over 10+ years:

| Resource | Quality | Completeness |
|---|---|---|
| **Official docs (keycloak.org/docs)** | High — comprehensive, well-organized | Very complete — covers all features |
| **Server Administration Guide** | Excellent — detailed admin operations | 400+ pages |
| **Developer Guide** | Good — SPI development, REST API | 300+ pages |
| **Authorization Services Guide** | Good — UMA 2.0 explained | 150+ pages |
| **Getting Started Guide** | Excellent — step-by-step tutorial | Beginner-friendly |
| **Upgrading Guide** | Critical — migration between versions | Detailed per-version changes |
| **API Documentation** | Full — OpenAPI/Swagger | Complete admin + token endpoints |
| **Examples repository** | Good — demo SPIs, themes | Covers common use cases |

**Community Support Channels:**

| Channel | Activity Level | Response Quality |
|---|---|---|
| **Keycloak mailing list** | Active — 100+ messages/month | High — core maintainers participate |
| **GitHub Discussions** | Active — questions answered regularly | Good |
| **Stack Overflow** | 20,000+ tagged questions | High — community-driven |
| **Keycloak blog** | Regular posts (monthly+) | High quality |
| **Conference talks** | Frequent at Devnexus, JBCNConf, KubeCon | Good |
| **Reddit r/keycloak** | Active | Community quality varies |
| **Third-party tutorials** | Hundreds of blog posts | Variable quality |

**Issue Tracker Activity:**

- GitHub issues: ~2,000+ open, ~500+ closed per release
- Average time to first response: 1-3 days
- Bug fix rate: Most critical bugs fixed within one release cycle (1-2 months)
- Feature request rate: High — many requests for new features

**Release Cadence:**

| Type | Frequency | Support Duration |
|---|---|---|
| Major releases (e.g., 24.0) | Every 2-3 months | Latest 2 versions supported |
| Patch releases (e.g., 24.0.5) | As needed | Security fixes backported |
| Long-term support | Via RHBK | 5+ years (Red Hat product) |

### 8.2 GGID Documentation

GGID has an impressive documentation set for its maturity level:

| Resource | Status | Quality |
|---|---|---|
| **Architecture docs** | Comprehensive — C4 model, ADRs, design docs | Good — 100+ docs |
| **API reference** | Full — OpenAPI spec, API examples, error codes | Good |
| **Getting started** | Quick start guide exists | Adequate |
| **Deployment guide** | Docker Compose focused | Partial (no K8s yet) |
| **Migration guides** | From Auth0, Keycloak, Clerk | Good — detailed mapping tables |
| **SDK guides** | Go, Java, Node.js | Adequate |
| **Security docs** | Security whitepaper, hardening guide, checklist | Good |
| **Tutorials** | Custom auth provider, multi-tenant setup, SAML, webhook | Good — practical |
| **Research docs** | 140+ research documents on IAM topics | Good — emerging trends |

From the docs directory listing, GGID has 100+ documentation files covering:
- Architecture (C4, decisions, deployment)
- API (reference, conventions, examples, error codes, rate limiting)
- Authentication (guide, MFA, WebAuthn, SAML, OAuth flows)
- Operations (backup, disaster recovery, observability, production hardening)
- Security (architecture, audit, compliance, vulnerability management)
- Integration (SDKs, webhooks, LDAP, SCIM, social login)
- Migration (from Auth0, Keycloak, Clerk, OAuth 2.1)

### 8.3 Documentation Comparison

| Aspect | Keycloak | GGID |
|---|---|---|
| **Documentation depth** | Deep — 10+ years of accumulated docs | Good for project age — 100+ docs |
| **Getting started experience** | Excellent — Docker quick start | Good — Docker Compose |
| **API documentation** | Complete — Swagger UI built in | Good — OpenAPI spec + Swagger UI |
| **SPI/Plugin documentation** | Extensive | Partial |
| **Community knowledge** | Massive — StackOverflow, blogs, talks | Minimal |
| **Migration documentation** | Limited (migration to Keycloak from other systems) | Good — migration from Keycloak, Auth0, Clerk |
| **Real-world examples** | Thousands of community tutorials | Limited to official docs |
| **Video tutorials** | Many (YouTube, conference recordings) | None |
| **Responsive community** | Yes — mailing list, discussions | Early stage |

---

## 9. Deployment & Operations

### 9.1 Keycloak Deployment Options

**Container Deployment:**

```bash
# Official Quarkus distribution
docker run -p 8080:8080 \
  -e KEYCLOAK_ADMIN=admin \
  -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:latest start-dev

# Production with database
docker run -p 8080:8080 \
  -e KC_DB=postgres \
  -e KC_DB_URL=jdbc:postgresql://db:5432/keycloak \
  -e KC_DB_USERNAME=keycloak \
  -e KC_DB_PASSWORD=password \
  -e KEYCLOAK_ADMIN=admin \
  -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:latest start \
    --optimized \
    --hostname=auth.example.com \
    --proxy=edge
```

**Kubernetes Operator:**

Keycloak has an official Kubernetes Operator that manages:
- Keycloak cluster deployment and scaling
- Realm import/export
- Database connection management
- TLS certificate management
- Backup configuration

```yaml
apiVersion: k8s.keycloak.org/v2alpha1
kind: Keycloak
metadata:
  name: keycloak
spec:
  instances: 3
  db:
    vendor: postgres
    host: postgres-db
    usernameSecret:
      name: keycloak-db-secret
      key: username
    passwordSecret:
      name: keycloak-db-secret
      key: password
  ingress:
    enabled: true
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt
  features:
    authorization: true
    webauthn: true
```

**Clustering:**

Keycloak clusters use:
- **JGroups** for node discovery and cluster communication
- **Infinispan** for distributed cache (sessions, login failures, offline tokens)
- **Database** as the source of truth
- **Load balancer** distributes traffic across nodes

```
                    ┌─────────────────┐
                    │  Load Balancer  │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
        ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐
        │ Keycloak  │ │ Keycloak  │ │ Keycloak  │
        │  Node 1   │ │  Node 2   │ │  Node 3   │
        └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
                    ┌────────┴────────┐
                    │  Infinispan     │
                    │  (distributed   │
                    │   cache)        │
                    └────────┬────────┘
                             │
                    ┌────────┴────────┐
                    │   PostgreSQL    │
                    │   (shared)      │
                    └─────────────────┘
```

**Database Support:**

| Database | Support | Notes |
|---|---|---|
| PostgreSQL | Full (recommended) | Best performance, native JSON support |
| MySQL | Full | Popular but some limitations |
| MariaDB | Full | MySQL-compatible |
| MSSQL | Full | Microsoft SQL Server |
| Oracle | Full (RHBK only) | Enterprise only |

### 9.2 GGID Deployment

**Docker Compose (primary deployment):**

```bash
cd deploy && docker compose up -d
```

This starts 12 containers:
- 7 microservices (gateway, identity, auth, oauth, policy, org, audit)
- 1 admin console (Next.js)
- 4 infrastructure (PostgreSQL, Redis, NATS, OpenLDAP)

**GGID Container Architecture:**

```
┌─────────────────────────────────────────────────────┐
│                  Docker Compose                      │
│                                                      │
│  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌────────┐ │
│  │ Gateway │  │Identity  │  │  Auth   │  │ OAuth  │ │
│  │ :8080   │  │ :8081    │  │ :9001   │  │ :9005  │ │
│  └─────────┘  └──────────┘  └─────────┘  └────────┘ │
│                                                      │
│  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌────────┐ │
│  │ Policy  │  │   Org    │  │ Audit   │  │Console │ │
│  │ :8070   │  │ :8071    │  │ :8072   │  │ :3000  │ │
│  └─────────┘  └──────────┘  └─────────┘  └────────┘ │
│                                                      │
│  ┌───────────────┐  ┌───────┐  ┌──────┐  ┌────────┐ │
│  │ PostgreSQL 16 │  │ Redis │  │ NATS │  │  LDAP  │ │
│  │ :5432         │  │ :6379 │  │:4222 │  │  :389  │ │
│  └───────────────┘  └───────┘  └──────┘  └────────┘ │
└─────────────────────────────────────────────────────┘
```

**Kubernetes (not yet available):**

GGID's architecture is designed for Kubernetes (stateless microservices, health
checks at `/healthz/live`, `/healthz/ready`, `/healthz/deep`, Prometheus
metrics at `/metrics`) but no Helm chart or K8s manifests exist yet.

### 9.3 Operational Complexity Comparison

| Aspect | Keycloak | GGID |
|---|---|---|
| **Components to manage** | 1 app + 1 DB (+ optional Infinispan) | 7 services + 4 infrastructure = 11 |
| **Single point of failure** | DB only (if clustered) | Each service is a potential SPOF |
| **Monitoring** | JMX + Micrometer + Prometheus | Prometheus + health checks |
| **Log aggregation** | Single log stream per instance | 7 log streams (one per service) |
| **Configuration** | Single config (CLI flags, env vars, keycloak.conf) | Per-service config (env vars each) |
| **Database migrations** | Liquibase (automatic) | SQL files per service (manual) |
| **Backup** | DB backup only | DB + Redis + NATS (event stream) |
| **Upgrade** | Replace JAR, restart | Update each service independently |
| **Horizontal scaling** | Add Keycloak nodes (JGroups auto-join) | Scale each service independently |
| **Debugging** | Single JVM (thread dumps, heap dumps) | Distributed tracing required |
| **Resource planning** | Size one JVM (heap, CPU) | Size each service independently |
| **Network configuration** | One port (8080) + DB port | 7 service ports + 4 infra ports |

### 9.4 Deployment Tradeoff Analysis

Keycloak's monolithic deployment is simpler for small-to-medium deployments.
One container, one database, one log stream. The operational overhead is minimal.

GGID's microservice deployment is more complex but offers:
- **Independent scaling**: Scale auth service to 50 replicas while keeping
  audit at 2 — saves resources vs scaling the entire monolith
- **Independent upgrades**: Patch the OAuth service's SAML parser without
  restarting the auth service
- **Fault isolation**: A memory leak in the audit service doesn't affect
  authentication
- **Technology diversity**: Can rewrite the policy engine in Rust without
  affecting other services

**The operational complexity tax of GGID's 7 services is real.** It requires:
- A container orchestrator (Kubernetes, Nomad, or Docker Swarm) — not just
  Docker Compose
- Distributed tracing (OpenTelemetry, Jaeger) to debug cross-service issues
- Centralized logging (ELK, Loki) to aggregate 7 log streams
- Service mesh (Istio, Linkerd) or API gateway for mTLS, circuit breaking,
  retry logic between services

**For small teams** (< 5 engineers), Keycloak's monolith is the right choice.
**For platform teams** managing a large microservices estate, GGID's
architecture fits naturally into their existing Kubernetes/Docker infrastructure.

---

## 10. What GGID Can Learn from Keycloak

### 10.1 Features That Matter Most (From 10 Years of Production Use)

Keycloak's decade of production deployment has revealed which features matter
most for enterprise adoption. GGID should prioritize these:

1. **Protocol completeness over feature breadth.** Enterprises don't care about
   30 social login providers; they care about token introspection (RFC 7662),
   backchannel logout, and token exchange (RFC 8693). These RFCs are
   non-negotiable for resource server integration. Keycloak learned this the
   hard way — many of these were added after enterprise customers demanded them.

2. **Realm/tenant management API.** The ability to programmatically create,
   configure, and manage tenants is essential for SaaS platforms. Keycloak's
   admin REST API for realm CRUD is one of its most-used features. GGID needs
   this urgently.

3. **Per-tenant configuration.** Real-world multi-tenancy means each tenant has
   different identity providers, authentication flows, and branding. A
   "one-size-fits-all" auth configuration cannot serve a B2B SaaS platform.

4. **Identity brokering (protocol translation).** The single feature that makes
   Keycloak indispensable is its ability to federate any IdP (SAML, OIDC, LDAP,
   social) and present a unified OIDC interface to applications. This is the
   "universal identity hub" value proposition.

5. **SCIM 2.0 for enterprise provisioning.** Azure AD, Okta, and other enterprise
   IdPs use SCIM for automated user provisioning. Without full SCIM support,
   GGID cannot integrate with enterprise directory sync.

6. **Themes and white-labeling.** Enterprises need to customize the login page
   with their own branding, terms of service, and messaging. Keycloak's theme
   SPI (full HTML/CSS/JS themes) is essential for white-label deployments.

7. **Clustering and high availability from day one.** Production IAM systems
   cannot have downtime. Keycloak's clustering (Infinispan + JGroups) was built
   early and has been battle-tested. GGID needs K8s-native HA configuration.

### 10.2 Mistakes Keycloak Made That GGID Should Avoid

1. **JPA/Hibernate as the only data layer.** Keycloak's use of JPA/Hibernate
   adds significant overhead: N+1 queries, lazy loading, second-level cache
   complexity. GGID's direct SQL (pgx) is already better, but should resist
   adding an ORM. Keep SQL close to the metal.

2. **Infinispan cache as a critical path.** Keycloak's distributed cache
   (Infinispan) is a source of operational complexity — cache invalidation bugs,
   split-brain scenarios, memory tuning. GGID should use Redis carefully and
   treat it as an optimization layer, not a source of truth.

3. **Realm explosion at scale.** Keycloak's realm model doesn't scale well past
   ~1,000 realms per deployment. Cache pressure, database table size, and
   configuration management become nightmares. GGID's RLS model is inherently
   more scalable for high-tenant-count deployments — keep it that way.

4. **Upgrade pain between major versions.** Keycloak upgrades (especially
   WildFly to Quarkus, legacy to new admin console, map storage migration) are
   notoriously painful. GGID should:
   - Maintain strict API versioning (`/api/v1/`)
   - Provide database migration scripts with rollback support
   - Never break backward compatibility within a major version
   - Document breaking changes with detailed migration guides

5. **Angular legacy admin console.** Keycloak maintained two admin consoles
   (Angular legacy + React new) for years, creating confusion and maintenance
   burden. GGID should maintain only one admin console (Next.js) and never
   split effort across multiple UIs.

6. **Too many configuration options.** Keycloak has hundreds of configuration
   options (keycloak.conf, CLI flags, environment variables, realm settings).
   This creates analysis paralysis for new users. GGID should maintain sensible
   defaults and minimize required configuration.

7. **SPI complexity.** Keycloak's SPI system, while powerful, has a steep
   learning curve (Java ServiceLoader, META-INF/services, classloading issues).
   GGID's WASM plugin approach is cleaner but needs more extension points and
   better documentation.

8. **Slow startup time hurts CI/CD.** Keycloak's 10-30 second startup makes
   integration tests painfully slow. Teams resort to running a persistent
   Keycloak instance for tests instead of spinning up per-test. GGID's < 1s
   startup is a massive advantage — never let it degrade.

9. **Database migration pain.** Keycloak uses Liquibase, which sometimes
   fails mid-migration, leaving the database in an inconsistent state.
   GGID should use simple, idempotent SQL migration scripts that can be
   safely re-run.

10. **Over-reliance on Java-specific tools.** Keycloak's deep coupling to
    the Java ecosystem (JPA, Hibernate, JTA, JGroups) makes it hard to
    evolve. GGID's Go foundation avoids this — Go's standard library and
    minimal runtime mean fewer external dependencies.

### 10.3 What Keycloak Still Struggles With

Understanding Keycloak's current pain points reveals GGID's competitive opportunities:

1. **Performance at scale.** Keycloak struggles with high-throughput token
   introspection (>10,000 req/s). The JVM overhead and Hibernate query overhead
   create bottlenecks. GGID's Go + pgx stack can handle significantly higher
   throughput for this specific workload.

2. **Memory consumption.** Keycloak's minimum production heap is 300-500MB,
   often 1-2GB under load. This makes it expensive to run many instances (for
   high-availability or per-tenant isolation). GGID's 20-35MB per service is
   10-20x more efficient.

3. **Startup time.** Even with Quarkus, Keycloak takes 5-15 seconds to start.
   GraalVM native images reduce this to 1-3 seconds but add build complexity
   (reflection configuration, longer build times). GGID's < 1s startup is
   a structural advantage.

4. **Upgrade complexity.** Major version upgrades (e.g., 22 → 24) often
   require database migrations, cache format changes, and configuration updates.
   Downtime is expected. GGID's microservice architecture allows rolling
   upgrades with zero downtime.

5. **Multi-tenancy at scale.** Keycloak's realm model struggles with 1,000+
   realms due to cache pressure and database growth. GGID's RLS model is
   inherently more scalable.

6. **Developer experience for extensions.** Writing Keycloak SPIs requires
   Java expertise, Maven build setup, and understanding of Keycloak's internal
   classloading. GGID's WASM plugins can be written in any language with a
   WASM compiler target.

7. **Observability gaps.** Despite Micrometer support, Keycloak's internal
   request flow is opaque. Debugging why a specific authentication flow failed
   requires reading Java stack traces and Hibernate SQL logs. GGID's structured
   logging and distributed tracing (OpenTelemetry) provide better observability.

8. **Realms are not truly isolated.** Keycloak's realms share the same database
   tables with `realm_id` filtering. A JPA bug or custom SPI can leak cross-realm
   data. GGID's PostgreSQL RLS provides database-level enforcement that is
   immune to application-level bugs.

---

## 11. Gap Analysis & Recommendations

### 11.1 Priority Matrix

Based on the competitive analysis, the following gaps are prioritized by
enterprise adoption impact:

#### P0 — Critical (Blocks Enterprise Adoption)

| # | Gap | Keycloak Has It | Impact | Recommendation | Effort |
|---|---|---|---|---|---|
| 1 | **Token Introspection (RFC 7662)** | Yes | Resource servers cannot validate tokens offline. Every enterprise API gateway needs this. | Add `/oauth/introspect` endpoint in OAuth service | Low (2-3 days) |
| 2 | **Backchannel Logout (OIDC)** | Yes | Cannot centrally terminate sessions. Critical for security compliance. | Implement OIDC backchannel logout in auth service | Medium (1-2 weeks) |
| 3 | **Tenant Management API** | Yes | Cannot programmatically create/manage tenants. Blocks SaaS platform use case. | Add CRUD API + service for tenant lifecycle | Medium (1 week) |
| 4 | **Per-Tenant IdP Configuration** | Yes | Each tenant needs their own OIDC/SAML/LDAP IdP. Blocks B2B SaaS. | Extend tenant model with per-tenant auth provider registry | High (3-4 weeks) |
| 5 | **SCIM 2.0 Full Implementation** | Partial (extension) | Cannot integrate with Azure AD/Okta automated provisioning. | Implement full SCIM CRUD, filtering, bulk, PATCH | High (3-4 weeks) |
| 6 | **Kubernetes/Helm Deployment** | Yes | Cannot deploy to production-grade orchestration. | Create Helm chart + K8s manifests for all services | Medium (2 weeks) |
| 7 | **High Availability Configuration** | Yes | Single point of failure in current deployment. | Add K8s deployment with replicas, HPA, health probes | Medium (2 weeks) |
| 8 | **Token Exchange (RFC 8693)** | Yes | Cannot delegate or impersonate tokens. Needed for service-to-service auth. | Implement token exchange grant type in OAuth service | Medium (1-2 weeks) |

#### P1 — Important (Needed for Competitive Parity)

| # | Gap | Keycloak Has It | Impact | Recommendation | Effort |
|---|---|---|---|---|---|
| 9 | **Per-Tenant Themes/Branding** | Yes | Cannot white-label login pages. | Add theme system with per-tenant HTML/CSS templates | Medium (2 weeks) |
| 10 | **Custom Domains** | Yes | Cannot route per-tenant custom domains to GGID. | Add domain routing in gateway + DNS verification | Medium (1 week) |
| 11 | **SAML ↔ OIDC Protocol Translation** | Yes | Cannot serve as universal identity hub. Core value proposition. | Implement broker that translates between protocols | High (4-6 weeks) |
| 12 | **First Broker Login Flow** | Yes | Cannot configure JIT provisioning/linking behavior per IdP. | Add configurable flow for first-time broker login | Medium (2 weeks) |
| 13 | **Device Authorization (RFC 8628)** | Yes | Cannot auth on smart TVs, CLIs, IoT devices. | Implement device code flow in OAuth service | Low (3-5 days) |
| 14 | **Per-Tenant Key Isolation** | Yes | All tenants share one JWT signing key. Security concern for multi-tenant SaaS. | Add per-tenant key management in key provider | Medium (1-2 weeks) |
| 15 | **Terraform Provider** | Yes | Cannot manage GGID config as infrastructure-as-code. | Build Terraform provider wrapping management API | Medium (2 weeks) |
| 16 | **Kerberos/SPNEGO** | Yes | Cannot do Windows desktop SSO. Important for enterprise AD environments. | Add Kerberos/SPNEGO auth provider | High (3-4 weeks) |
| 17 | **SMS/Email OTP MFA** | Partial | Limited MFA options. SMS is table-stakes for many enterprises. | Add Twilio/SMS and email OTP providers to auth chain | Medium (1 week) |
| 18 | **Concurrent Session Limits** | Partial | Cannot enforce single-session or N-session policies. | Track active sessions in Redis, enforce limits | Low (3-5 days) |

#### P2 — Moderate (Nice to Have)

| # | Gap | Keycloak Has It | Impact | Recommendation | Effort |
|---|---|---|---|---|---|
| 19 | **UMA 2.0 Authorization** | Yes | Missing fine-grained, user-managed access control. | Extend policy engine with resources/scopes/tickets | High (6-8 weeks) |
| 20 | **JavaScript Policies** | Yes | Cannot express complex authorization logic. | Add JS policy evaluation (embedded V8/QuickJS) | Medium (2 weeks) |
| 21 | **Passwordless / Magic Link** | Partial | Missing modern auth UX trends. | Implement magic link + passwordless flow | Medium (1 week) |
| 22 | **WS-Federation** | Yes | Some legacy enterprise systems need it. | Add WS-Fed passive requestor profile | Medium (2 weeks) |
| 23 | **DPoP (RFC 9449)** | Yes | Proof-of-possession for enhanced token security. | Implement DPoP token binding | Medium (1-2 weeks) |
| 24 | **Home Realm Discovery** | Yes | User enters email, system routes to correct IdP. | Email domain → tenant/IdP mapping | Low (3-5 days) |
| 25 | **Tamper-Proof Audit Trail** | No | Audit logs could be modified. | Append-only storage + hash chain anchoring | Medium (1-2 weeks) |
| 26 | **Compliance Reporting** | Partial | Cannot generate SOC 2/HIPAA audit reports. | Add report generation to audit service | Medium (1-2 weeks) |
| 27 | **SIEM Native Connectors** | Partial | NATS events require manual consumer setup. | Ship NATS → Splunk/Datadog connectors | Low (3-5 days) |
| 28 | **Permission Ticket (UMA)** | Yes | Dynamic resource owner grants. | Implement UMA permission ticket flow | Medium (2 weeks) |
| 29 | **Realm/Config Export-Import** | Yes | Cannot backup/restore tenant configurations. | Add config export/import API | Medium (1 week) |
| 30 | **OpenTelemetry Distributed Tracing** | Yes | Cross-service request tracing needs improvement. | Wire OTel spans across all services | Medium (1-2 weeks) |

### 11.2 Strategic Recommendations

#### Short-Term (Next 3 Months) — Focus on P0 Gaps

1. **Implement token introspection** — This is the single most impactful gap.
   Every resource server (API backend) needs to validate tokens without the
   full JWT verification dance. A 3-day investment unlocks massive enterprise
   integration potential.

2. **Add tenant management API + per-tenant IdP config** — These two together
   unlock the B2B SaaS use case, which is GGID's most natural market
   (modern SaaS companies building multi-tenant platforms in Go).

3. **Publish Helm chart + K8s manifests** — No enterprise deploys IAM via
   Docker Compose in production. A Helm chart is the minimum viable
   deployment artifact for production adoption.

4. **Complete SCIM 2.0** — Without full SCIM, GGID cannot integrate with
   Azure AD, Okta, or Google Workspace automated user provisioning. This is
   a hard requirement for enterprise identity management.

#### Medium-Term (3-6 Months) — Competitive Differentiation

5. **Build identity brokering with protocol translation** — This is the "universal
   identity hub" capability that makes Keycloak indispensable. GGID should aim
   to support: OIDC IdP → GGID → OIDC client, SAML IdP → GGID → OIDC client,
   LDAP → GGID → OIDC client.

6. **Add per-tenant themes and custom domains** — White-label support is
   essential for SaaS platforms that want their login pages to match their
   brand.

7. **Achieve OIDC certification** — Formal certification (basic, config, dynamic,
   form modes, hybrid profiles) is a trust signal for enterprise buyers.
   Keycloak and Auth0 both have this. GGID should too.

8. **Build Terraform provider** — Infrastructure-as-code is standard practice.
   A Terraform provider enables GGID to be managed alongside other infrastructure.

#### Long-Term (6-12 Months) — Feature Parity and Beyond

9. **Implement UMA 2.0 authorization** — For healthcare, finance, and
   collaborative applications that need dynamic, user-managed access control.

10. **Build plugin marketplace** — GGID's WASM plugin system is architecturally
    superior to Keycloak's SPI (sandboxed, language-agnostic). Capitalize on
    this by creating a plugin registry with starter templates and documentation.

11. **Pursue SOC 2 Type II certification** — Enterprise buyers require formal
    security certifications. This is a 6-12 month process but opens doors.

12. **Build managed SaaS offering** — A cloud-hosted GGID would compete
    directly with Auth0 and Cloud-IAM (managed Keycloak). GGID's Go
    architecture (low memory, fast startup) is a cost advantage for a
    managed service.

### 11.3 GGID's Sustainable Competitive Advantages

Despite Keycloak's feature lead, GGID has several structural advantages that
are difficult to replicate:

1. **Go performance**: 10-20x lower memory, 10-50x faster startup, 2-3x higher
   throughput. These are structural — you can't make Java as efficient as Go
   for this workload.

2. **PostgreSQL RLS**: Database-enforced multi-tenant isolation is strictly
   more secure than application-level filtering. This is a compliance
   differentiator (HIPAA, SOC 2, GDPR).

3. **Microservice architecture**: Independent scaling, fault isolation, and
   technology diversity. This is the cloud-native standard — Keycloak's
   monolith is a legacy architecture.

4. **NATS JetStream audit**: Native async event streaming for audit logs is
   more performant and SIEM-ready than Keycloak's database-backed event log.

5. **gRPC first-class**: Native gRPC for internal services is a developer
   experience advantage. Keycloak and Auth0 are REST-only.

6. **WASM plugin sandbox**: Sandboxed, language-agnostic plugins are more
   secure than Keycloak's in-process Java SPI.

7. **Apache 2.0 with no MAU limits**: Free at any scale. Auth0 charges per-MAU;
   Keycloak is free but operationally expensive (JVM resources).

### 11.4 The Path to Competitive Parity

GGID cannot match Keycloak's 10-year feature head start in the short term.
The strategic path is:

1. **Own the Go-native IAM niche** — Be the best IAM for Go/Kubernetes/cloud-native
   teams. Keycloak's Java foundation is a turnoff for Go shops.

2. **Win on operational efficiency** — 10x lower memory, 50x faster startup,
   10x smaller images. These matter at scale and in cost-sensitive environments.

3. **Achieve protocol parity on the RFCs that matter** — Token introspection,
   backchannel logout, token exchange, device authorization. These are
   non-negotiable for enterprise integration.

4. **Differentiate on security architecture** — RLS, WASM sandbox, NATS
   audit streaming. These are architecturally superior approaches that
   Keycloak cannot easily replicate.

5. **Build the ecosystem incrementally** — Helm charts, Terraform provider,
   SDKs, plugin marketplace. Each ecosystem investment compounds over time.

---

## Appendix A: Feature Count Summary

| Category | Keycloak Full | GGID Full | Gap |
|---|---|---|---|
| Authentication Methods | 8/15 | 6/15 | 2 features |
| Protocols & Standards | 14/18 | 8/18 | 6 features |
| User Management | 11/12 | 9/12 | 2 features |
| Authorization | 6/7 | 4/7 | 2 features |
| Administration & APIs | 9/14 | 12/14 | GGID leads by 3 |
| Deployment & Ops | 11/16 | 8/16 | 3 features |
| **Total** | **59/82** | **47/82** | **12 features** |

## Appendix B: Keycloak Version History (Relevant Releases)

| Version | Date | Key Changes |
|---|---|---|
| 1.0 | 2014 | Initial release (WildFly) |
| 2.0 | 2016 | Identity brokering, social login expansion |
| 3.0 | 2017 | OIDC certification, fine-grained admin permissions |
| 4.0 | 2018 | Authorization Services (UMA 2.0) GA |
| 6.0 | 2019 | WebAuthn support, account console v2 |
| 8.0 | 2019 | Token exchange, client policies |
| 12.0 | 2020 | Quarkus preview, new admin console preview |
| 17.0 | 2022 | Quarkus GA (WildFly deprecated), map storage preview |
| 20.0 | 2023 | New admin console (React) GA, declarative user profile |
| 22.0 | 2023 | Multi-site active-active clustering (preview) |
| 24.0 | 2024 | PAR support, OAuth 2.1 alignment, DPoP |
| 26.0 | 2025 | Continued performance improvements, OTel enhancements |

## Appendix C: References

- [Keycloak Official Documentation](https://www.keycloak.org/documentation)
- [Keycloak GitHub Repository](https://github.com/keycloak/keycloak)
- [Keycloak Authorization Services Guide](https://www.keycloak.org/docs/latest/authorization_services/)
- [Keycloak Server Administration Guide](https://www.keycloak.org/docs/latest/server_admin/)
- [Keycloak SPI Documentation](https://www.keycloak.org/docs/latest/server_development/)
- [Red Hat Build of Keycloak](https://www.redhat.com/en/technologies/jboss-middleware/red-hat-build-of-keycloak)
- [CNCF Sandbox Projects](https://www.cncf.io/sandbox-projects/)
- [GGID Source Code](https://github.com/ggid/ggid) (internal)
- [GGID Architecture Documentation](../architecture.md)
- [GGID Feature Matrix](../feature-matrix.md)
- [Auth0-Keycloak-GGID Comparison Matrix](./auth0-keycloak-ggid-matrix.md)
- [Keycloak Migration Guide](../migration-from-keycloak.md)

---

*This document is a competitive analysis based on publicly available information
about Keycloak and direct source code analysis of the GGID project. Keycloak
feature assessments reflect the state of Keycloak as of version 26 (2025).
GGID assessments reflect the source code state as of June 2025.*
