# Ory vs GGID: Deep Competitive Analysis

> **Research Document** — A comprehensive, source-level analysis of the Ory IAM ecosystem
> compared to the GGID IAM Suite.
>
> Date: 2025 | Authors: GGID Research Team
>
> **Scope**: This document goes significantly deeper than `ggid-vs-ory.md` (704 lines)
> by examining Ory's four projects (Kratos, Hydra, Keto, Oathkeeper) against GGID's
> seven microservices at the architectural, API, data-model, and developer-experience
> levels. Every comparison includes concrete code references from the GGID codebase.

---

## Table of Contents

1. [Ory Ecosystem Overview](#1-ory-ecosystem-overview)
2. [Architecture Philosophy Comparison](#2-architecture-philosophy-comparison)
3. [Kratos vs GGID Identity Service](#3-kratos-vs-ggid-identity-service)
4. [Hydra vs GGID OAuth Service](#4-hydra-vs-ggid-oauth-service)
5. [Keto vs GGID Policy Service](#5-keto-vs-ggid-policy-service)
6. [Oathkeeper vs GGID Gateway](#6-oathkeeper-vs-ggid-gateway)
7. [Developer Experience](#7-developer-experience)
8. [Self-Hosting vs Managed](#8-self-hosting-vs-managed)
9. [Performance](#9-performance)
10. [What GGID Can Learn from Ory](#10-what-ggid-can-learn-from-ory)
11. [Gap Analysis & Recommendations](#11-gap-analysis--recommendations)
12. [Appendix A: Feature Matrix](#appendix-a-feature-matrix)
13. [Appendix B: API Endpoint Comparison](#appendix-b-api-endpoint-comparison)
14. [Appendix C: Data Model Comparison](#appendix-c-data-model-comparison)

---

## 1. Ory Ecosystem Overview

### 1.1 History and Origins

Ory was founded by **Aeneas Rekkas** in **2014**, originally as an open-source project
on GitHub under the handle `arekkas`. The project began with **Hydra**, a dedicated
OAuth2/OIDC server, born from frustration with the complexity of bolting OAuth2 onto
existing identity systems. Hydra's core design principle was radical separation: the
OAuth2 provider should **never handle login UI** — that responsibility belongs to a
separate application. This principle shaped the entire Ory ecosystem.

**Timeline:**

| Year | Milestone |
|------|-----------|
| 2014 | Hydra v0.1 — first commit, OAuth2 server without login UI |
| 2017 | Kratos project started — identity management with JSON Schema traits |
| 2018 | Oathkeeper — zero-trust reverse proxy / PEP |
| 2019 | Keto — Google Zanzibar-style authorization engine |
| 2020 | Ory receives seed funding; team grows to 10+ |
| 2021 | Ory Corp incorporated; Series Seed extension |
| 2022 | **Ory Cloud** (later Ory Network) launched as managed SaaS; **Series A: $25M** |
| 2023 | Ory Enterprise License for self-hosted enterprise; organizations feature |
| 2024 | Unified SDK release; Kratos organizations; passkey support |
| 2025 | Ory Network scaling; v25.x unified release cadence |

### 1.2 The Four Projects

Ory consists of **four independently deployable** Go projects, each with its own
repository, release cycle (though synchronized since 2022), and database:

#### 1.2.1 Kratos — Identity Management

- **Repository**: `github.com/ory/kratos`
- **GitHub Stars**: ~11,000 (as of 2025)
- **Role**: User identity, registration, login, profile management, MFA, social login,
  account recovery, email/phone verification, session management
- **Key Innovation**: **JSON Schema-based identity traits** — instead of a fixed user
  model, Kratos lets you define your identity schema using JSON Schema, with fields
  automatically mapped to database columns and UI form generation
- **Flow Model**: Every identity interaction (registration, login, verification,
  recovery, settings) is a **self-contained flow object** with UI nodes, webhooks,
  and redirect URLs
- **Database**: PostgreSQL (primary), SQLite (development), CockroachDB (distributed)
- **Authentication**: Password, WebAuthn/passkeys, TOTP, lookup codes, social (OIDC)
- **Sessions**: Cookie-based (server-side sessions stored in DB or memory)

#### 1.2.2 Hydra — OAuth2/OIDC Provider

- **Repository**: `github.com/ory/hydra`
- **GitHub Stars**: ~15,000
- **Role**: OAuth 2.0 authorization server, OIDC provider, token issuance, consent
  flows, client management
- **Key Design**: **Login UI is external** — Hydra redirects to your application for
  login, then you call Hydra back with the accept/reject decision. This strict
  separation means Hydra never sees user credentials.
- **OIDC Certification**: Hydra is **certified by the OpenID Foundation** — passed
  the conformance test suite for multiple profiles
- **Token Types**: Opaque tokens (default) with introspection, or JWTs
- **Grant Types**: Authorization code, client credentials, refresh token, resource
  owner password (deprecated), device authorization (RFC 8628), token exchange
  (RFC 8693)
- **Advanced**: DPoP (Demonstration of Proof-of-Possession), PAR (Pushed
  Authorization Requests), JAR (JWT-Secured Authorization Request), RFC 7592
  (Dynamic Client Management)
- **Database**: PostgreSQL, SQLite, CockroachDB

#### 1.2.3 Keto — Authorization Engine

- **Repository**: `github.com/ory/keto`
- **GitHub Stars**: ~5,000
- **Role**: Google Zanzibar-style **relation tuple** authorization
- **Data Model**: `object#relation@subject` tuples — e.g.,
  `document:report.pdf#editor@user:alice`
- **Query Model**: `Check(object, relation, subject) -> boolean` with O(1) cached
  lookups for warm data
- **Architecture**: In-memory graph cache backed by PostgreSQL; supports transitive
  relationships (recursive graph traversal)
- **Use Cases**: Fine-grained per-resource authorization (Google Docs-style
  permissions), file sharing, org hierarchy, team membership
- **API**: gRPC + REST

#### 1.2.4 Oathkeeper — Identity-Aware Proxy

- **Repository**: `github.com/ory/oathkeeper`
- **GitHub Stars**: ~3,000
- **Role**: Zero-trust reverse proxy / Policy Enforcement Point (PEP)
- **Architecture**: Pluggable pipeline with three stages:
  1. **Authenticators**: cookie_session, jwt, oauth2_introspection, anonymous, noop,
     bearer_token, unauthorized
  2. **Authorizers**: allow, deny, keto (Zanzibar check), remote_json (custom HTTP)
  3. **Mutators**: header (inject headers), id_token (create JWT), cookie,
     hydrator (fetch additional data)
- **Configuration**: Declarative YAML access rules per route
- **Deployment**: Standalone binary or as a sidecar in Kubernetes

### 1.3 Ory Network (Managed SaaS)

Launched in 2022 as **Ory Cloud**, rebranded as **Ory Network**, this is the managed
SaaS offering:

- **Free Tier**: 10,000 monthly active users, basic features
- **Pro Tier**: $0.02/MAU after free tier, advanced features
- **Enterprise**: Custom pricing, SLA, dedicated infrastructure
- **Features**: All four services managed, global edge deployment, automatic
  scaling, project-level isolation (not true multi-tenant data isolation — each
  "project" is a separate namespace)

### 1.4 Funding and Business Model

| Round | Amount | Year | Lead Investor |
|-------|--------|------|---------------|
| Seed | Undisclosed | 2020 | — |
| Seed Extension | Undisclosed | 2021 | — |
| **Series A** | **$25M** | **2022** | **TCV** |
| Series B | Undisclosed | 2024 | — |

**Revenue Model:**
1. Ory Network (SaaS subscription — per MAU)
2. Ory Enterprise License (self-hosted enterprise features — multi-tenancy, audit
   logs, SSO, SLA)
3. Professional services (consulting, custom development)

### 1.5 Community and Open-Source Metrics

| Metric | Value |
|--------|-------|
| Total GitHub stars (all repos) | ~50,000+ |
| Contributors | 500+ |
| Open issues (all repos) | ~1,200 |
| Merged PRs | 5,000+ |
| CNCF status | Sandbox |
| License | Apache 2.0 |
| Release cadence | Quarterly major (v25.x) |
| Primary languages | Go (90%+), TypeScript (account UI) |
| Production users | Tesla, GitHub (limited), Unity, Zalando, Subway |

---

## 2. Architecture Philosophy Comparison

### 2.1 Fundamental Design Principles

The two platforms represent **fundamentally different architectural philosophies**.
Understanding these differences is critical for any team choosing between them.

#### Ory: "Compose What You Need"

Ory's design principle is **maximum flexibility through minimal coupling**. Each
project is a standalone binary with its own database. You can deploy Hydra without
Kratos (use your own login system). You can deploy Keto without anything else. You
can deploy Oathkeeper with Auth0 instead of Ory's identity stack.

**Core tenets:**
- **Single Responsibility**: Each service does one thing well
- **Own Database**: No shared database — each service owns its data
- **HTTP REST Only**: Services communicate exclusively through REST APIs
- **External Login UI**: Hydra never handles login — your app does
- **Pluggable**: Everything is pluggable through interfaces and configuration

**Tradeoffs:**
- **Advantage**: Extreme flexibility — mix-and-match with any existing system
- **Advantage**: Independent scaling — scale Hydra independently of Kratos
- **Advantage**: Blast radius isolation — a Kratos bug cannot corrupt Hydra's data
- **Disadvantage**: High operational complexity — 4 separate services to deploy,
  monitor, and debug
- **Disadvantage**: No shared types — each service redefines identity concepts
- **Disadvantage**: Integration burden — you must wire Kratos + Hydra together
  yourself (consent flow, login flow, session sharing)

#### GGID: "Integrated Out of the Box"

GGID's design principle is **opinionated integration**. All seven services are
designed to work together from day one. They share a Go module, common packages,
and a single PostgreSQL database with Row-Level Security for tenant isolation.

**Core tenets:**
- **Integrated Suite**: Services are designed to work together
- **Shared Database**: Single PostgreSQL with RLS for tenant isolation
- **gRPC + REST**: gRPC for internal communication, REST for external APIs
- **Multi-Tenant First**: `tenant_id` is threaded through every API, every policy,
  every audit event
- **Shared Packages**: 13 shared packages reduce duplication and ensure consistency

**Tradeoffs:**
- **Advantage**: Single `docker compose up -d` starts everything
- **Advantage**: Cross-service type safety — shared Go packages
- **Advantage**: Built-in multi-tenancy with database-level isolation
- **Advantage**: Event-driven audit trail via NATS JetStream
- **Disadvantage**: Less flexible — harder to use individual services standalone
- **Disadvantage**: Tighter coupling — changes to shared packages affect all services
- **Disadvantage**: Harder to scale individual services independently

### 2.2 Architectural Diagrams

#### Ory Topology

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Ory Ecosystem Topology                         │
│                                                                       │
│                       ┌──────────────────┐                           │
│    Client ──────────> │   Oathkeeper     │ (reverse proxy / PEP)    │
│                       │   (YAML rules)   │                           │
│                       └────────┬─────────┘                           │
│                                │                                      │
│              ┌─────────────────┼─────────────────┐                   │
│              ▼                 ▼                   ▼                   │
│      ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│      │   Kratos     │  │    Hydra     │  │    Keto      │           │
│      │  (identity)  │  │  (OAuth2/    │  │  (authz /    │           │
│      │              │  │   OIDC)      │  │  Zanzibar)   │           │
│      └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
│             │                 │                  │                    │
│      ┌──────┴───────┐  ┌─────┴──────┐   ┌──────┴───────┐            │
│      │ PostgreSQL   │  │ PostgreSQL │   │ PostgreSQL   │            │
│      │  (Kratos DB) │  │ (Hydra DB) │   │  (Keto DB)   │            │
│      └──────────────┘  └────────────┘   └──────────────┘            │
│                                                                       │
│      Communication: HTTP REST APIs only                               │
│      Each service: independent deploy, independent database           │
│      No shared Go packages between services                           │
└──────────────────────────────────────────────────────────────────────┘
```

#### GGID Topology

```
┌──────────────────────────────────────────────────────────────────────┐
│                        GGID Ecosystem Topology                        │
│                                                                       │
│                       ┌──────────────────┐                           │
│    Client ──────────> │    Gateway       │ (proxy + middleware)     │
│                       │  (JWT, rate-     │                           │
│                       │   limit, tenant) │                           │
│                       └────────┬─────────┘                           │
│                                │                                      │
│         ┌──────────┬───────────┼───────────┬──────────┐              │
│         ▼          ▼           ▼           ▼          ▼              │
│   ┌──────────┐┌──────────┐┌──────────┐┌──────────┐┌──────────┐     │
│   │ Identity ││   Auth   ││  OAuth   ││  Policy  ││   Org    │     │
│   │ (users,  ││ (login,  ││ (OIDC,   ││ (RBAC +  ││  (B2B    │     │
│   │  SCIM)   ││  MFA,    ││ clients, ││  ABAC    ││  orgs,   │     │
│   │          ││  social) ││  tokens) ││  engine) ││  teams)  │     │
│   └────┬─────┘└────┬─────┘└────┬─────┘└────┬─────┘└────┬─────┘     │
│        │           │           │           │           │             │
│        └───────────┴───────────┴───────────┴───────────┘             │
│                                │                                      │
│                    ┌───────────┴───────────┐                         │
│                    │     Shared Layer      │                         │
│                    │  PostgreSQL (RLS)     │                         │
│                    │  Redis (sessions)     │                         │
│                    │  NATS (audit events)  │                         │
│                    └───────────────────────┘                         │
│                                                                       │
│      Communication: gRPC (internal) + REST (external)                │
│      13 shared packages: crypto, tenant, errors, social, saml, ...   │
│      Built-in multi-tenancy via RLS                                  │
└──────────────────────────────────────────────────────────────────────┘
```

### 2.3 Codebase Structure Comparison

| Aspect | Ory | GGID |
|--------|-----|------|
| **Repository model** | Polyrepo (4 repos + shared config) | Monorepo (single Go module) |
| **Total Go files** | ~3,000+ across 4 repos | 456 files (55,495 non-test lines) |
| **Shared packages** | None between services | 13 packages (`pkg/`) |
| **Build system** | Bazel + Makefile + Docker | Makefile + Docker |
| **Dependency management** | Go modules per repo | Single `go.mod` for all services |
| **CI/CD** | GitHub Actions per repo | Single CI pipeline |
| **Cross-service refactoring** | Manual coordination | `go build ./...` validates all |

### 2.4 Database Strategy Comparison

This is one of the most fundamental architectural differences.

#### Ory: Database-Per-Service

Each Ory service has its own database schema and connection pool. This means:
- **Kratos** owns the `identities`, `identity_credentials`, `sessions`, `flows`
  tables
- **Hydra** owns the `oauth2_clients`, `oauth2_access_tokens`, `oidc_sessions`
  tables
- **Keto** owns the `relation_tuples` table
- **Oathkeeper** is stateless (no database)

**Implications:**
- No cross-service transactions (no way to atomically create an identity and an
  OAuth client)
- No cross-service foreign keys
- Each service manages its own migrations independently
- Potential for data inconsistency if services disagree on shared concepts
  (e.g., "what is a user ID?")
- Each service needs its own DB connection pool

#### GGID: Shared Database with RLS

All GGID services share a single PostgreSQL database. Tenant isolation is enforced
at the database level via **Row-Level Security (RLS)**:

```sql
-- Every table has a tenant_id column
-- RLS policies enforce:
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Implications:**
- Cross-service transactions are possible (create user + assign role atomically)
- Cross-service foreign keys enforce referential integrity
- Single migration pipeline (`deploy/migrations/`)
- RLS provides defense-in-depth: even if application code has a tenant isolation
  bug, the database prevents cross-tenant data leakage
- Simpler connection pool management (one pool, shared across services)

**GGID RLS in Practice:** Looking at GGID's codebase, every domain entity includes
a `TenantID` field:

```go
// From services/identity/internal/domain/user.go
type User struct {
    ID             uuid.UUID
    TenantID       uuid.UUID   // ← Threaded through every entity
    Username       string
    Email          string
    // ...
}
```

This is a **fundamental architectural advantage** for multi-tenant SaaS scenarios.
Ory's open-source version has no concept of tenant isolation — multi-tenancy
requires Ory Network (SaaS) or Enterprise License (paid).

### 2.5 Inter-Service Communication

| Aspect | Ory | GGID |
|--------|-----|------|
| **Protocol** | HTTP REST only | gRPC (internal) + REST (external) |
| **Serialization** | JSON | Protobuf (gRPC) + JSON (REST) |
| **Service discovery** | DNS / environment variables | DNS / environment variables |
| **Circuit breaking** | External (Istio, Linkerd) | Built-in gateway circuit breaker |
| **Request tracing** | External (Jaeger, OpenTelemetry) | Planned |
| **Timeout handling** | Per-service config | Per-route timeout config in gateway |

### 2.6 Philosophy Summary

| Dimension | Ory | GGID |
|-----------|-----|------|
| **Design principle** | Compose-what-you-need | Integrated-out-of-the-box |
| **Coupling** | Minimal (REST boundaries) | Moderate (shared packages, shared DB) |
| **Flexibility** | High (mix-and-match) | Medium (use the whole suite) |
| **Time-to-first-deploy** | Longer (wire services together) | Shorter (docker compose up) |
| **Multi-tenancy** | Enterprise/cloud only | Free (Apache 2.0) |
| **Operational complexity** | Higher (4 DBs, 4 deploys) | Lower (1 DB, 1 compose file) |
| **Scalability model** | Per-service independent | Shared database, service-level scale |

---

## 3. Kratos vs GGID Identity Service

This is the most complex comparison because Ory splits identity management across
Kratos (identity + auth flows) while GGID splits it across two services:
**identity** (user/org management, SCIM) and **auth** (login, register, MFA, social,
LDAP, SAML, WebAuthn).

### 3.1 Data Model Comparison

#### Kratos Identity Model

Kratos uses **JSON Schema** to define the identity model. The schema is configurable
at runtime — you push a new schema via API and Kratos handles the rest:

```json
{
  "$id": "https://example.com/identity.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Person",
  "type": "object",
  "properties": {
    "traits": {
      "type": "object",
      "properties": {
        "email": {
          "type": "string",
          "format": "email",
          "ory.sh/kratos": {
            "credentials": {
              "password": { "identifier": true },
              "webauthn": { "identifier": true }
            },
            "verification": { "via": "email" },
            "recovery": { "via": "email" }
          }
        },
        "name": { "type": "object", "properties": {
          "first": { "type": "string" },
          "last": { "type": "string" }
        }}
      },
      "required": ["email"],
      "additionalProperties": false
    }
  }
}
```

**Key aspects of Kratos's data model:**
- **Traits**: User-defined identity attributes, validated by JSON Schema
- **Credentials**: Separate from traits — password hash, WebAuthn keys, OIDC tokens
- **Schema-driven**: Adding a new field is a schema push, not a migration
- **Verifiable addresses**: Email/phone with verification state tracked separately
- **Recovery codes**: Generated and tracked per identity

#### GGID Identity Model

GGID uses **Go structs with compiled types**. The identity model is defined in
`services/identity/internal/domain/user.go`:

```go
type User struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    Username       string
    Email          string
    Phone          string
    Status         UserStatus   // active | locked | disabled | deleted
    EmailVerified  bool
    PhoneVerified  bool
    PrimaryEmailID *uuid.UUID
    DisplayName    string
    AvatarURL      string
    Locale         string
    Timezone       string
    ExternalID     string       // SCIM externalId
    LastLoginAt    *time.Time
    LastLoginIP    *netip.Addr
    PasswordHash   string       // Argon2id encoded hash
    CreatedAt      time.Time
    UpdatedAt      time.Time
    DeletedAt      *time.Time   // soft delete
}
```

**Key aspects of GGID's data model:**
- **Fixed schema**: Adding a new field requires code changes and a migration
- **Tenant-aware**: Every entity has `TenantID` built-in
- **SCIM-compatible**: `ExternalID` field for enterprise directory sync
- **Soft delete**: `DeletedAt` preserves data for compliance
- **Status lifecycle**: `UserStatus` enum with `CanAuthenticate()` method

**GGID also has richer identity models:**

```go
// From services/identity/internal/domain/group.go
type Group struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    Name        string
    DisplayName string
    Description string
    Members     []GroupMember
    ParentID    *uuid.UUID  // hierarchical groups
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 3.2 Registration Flows

#### Kratos Registration

Kratos registration is a **multi-step flow** with configurable UI nodes:

1. Client initiates registration: `POST /self-service/registration/api` or
   `GET /self-service/registration/browser`
2. Kratos returns a **flow object** with UI nodes (form fields)
3. Client renders UI from nodes (or uses Kratos's React account UI kit)
4. Client submits flow: `POST /self-service/registration?flow={flow_id}`
5. Kratos validates against identity schema, creates identity
6. Webhooks fire (if configured)
7. Client receives identity + session token (or redirect for browser flow)

**Kratos flow advantages:**
- Schema-driven: form fields auto-generated from JSON Schema
- Multi-step: can chain registration steps (e.g., email verification before
  activation)
- Webhooks: fire custom logic at each step
- Branded UI: customize the look-and-feel per tenant/organization
- Account linking: automatic or manual linking to existing accounts

#### GGID Registration

GGID registration is a **single-step REST API call**:

```
POST /api/v1/auth/register
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "secure-password",
  "tenant_id": "00000000-0000-0000-0000-000000000001"
}
```

The auth service (`services/auth/internal/service/auth_service.go`) processes
registration:
1. Validate password against password policy
2. Hash password with Argon2id
3. Create user via identity service
4. Store credential in auth service
5. Return JWT + refresh token

**GGID registration advantages:**
- Simpler: one API call, no flow state to manage
- API-first: perfect for SPAs and mobile apps
- Tenant-aware: `tenant_id` in every request
- No redirect handling needed

**GGID registration disadvantages vs Kratos:**
- No schema-driven form generation
- No multi-step flows
- No built-in webhooks on registration
- No branded UI kit for registration pages

### 3.3 Profile Management

| Feature | Kratos | GGID |
|---------|--------|------|
| **Schema-driven fields** | JSON Schema (runtime-configurable) | Go structs (compiled) |
| **Adding new fields** | Push new schema via API | Code change + migration |
| **UI form generation** | Auto-generated from schema | Manual (Admin Console) |
| **Profile update** | Settings flow (multi-step) | `PUT /api/v1/users/{id}` |
| **Avatar support** | Via traits (URL field) | `AvatarURL` field built-in |
| **Locale/timezone** | Via traits | `Locale`, `Timezone` built-in |

### 3.4 Account Recovery

#### Kratos Recovery Flow

Kratos has a dedicated **recovery flow** that is highly configurable:

1. User requests recovery: `POST /self-service/recovery/api`
2. Kratos creates recovery flow, sends recovery link/code via email
3. User clicks link or enters code
4. Kratos verifies, allows password reset
5. Optional: require second factor for recovery

**Kratos recovery features:**
- Recovery codes (generated ahead of time)
- Email-based recovery links
- SMS recovery (with twilio integration)
- Configurable expiry
- Webhooks on recovery events

#### GGID Recovery

GGID has API-based password recovery:

```
POST /api/v1/auth/password/forgot   { "email": "alice@example.com" }
POST /api/v1/auth/password/reset    { "token": "...", "new_password": "..." }
```

**GGID recovery gaps:**
- No recovery code generation (backup codes)
- No multi-channel recovery (SMS, voice)
- No recovery flow tracking
- No configurable recovery expiry per tenant

### 3.5 Email and Phone Verification

| Feature | Kratos | GGID |
|---------|--------|------|
| **Email verification** | Built-in flow with templates | `EmailVerified` boolean + API |
| **Phone verification** | Via traits + SMS | `PhoneVerified` boolean + API |
| **Verification templates** | Customizable per project | Planned |
| **Verification expiry** | Configurable | Not configurable |
| **Re-verification trigger** | On email change | On email change |

### 3.6 Session Management

| Feature | Kratos | GGID |
|---------|--------|------|
| **Session type** | Cookie-based (server-side) | JWT-based (stateless) |
| **Session storage** | Database/memory | Redis |
| **Session expiry** | Configurable per project | Configurable via token TTL |
| **Session revocation** | Delete session from DB | JWT blacklist in Redis |
| **Session listing** | Admin API | Console sessions page |
| **Concurrent sessions** | Configurable | Not limited |
| **Session device info** | User agent, IP tracked | Last login IP + time tracked |

**Kratos uses server-side sessions** stored in the database. This allows:
- Immediate revocation (delete session)
- Session listing and management
- Concurrent session limits
- Device fingerprinting

**GGID uses JWT-based sessions** with Redis-backed refresh tokens:
- Stateless verification (no DB lookup for JWT validation)
- Refresh token rotation
- JWT blacklist for revocation (via Redis)
- Lower latency for session validation (no DB lookup)

**Tradeoff:** GGID's JWT approach is faster for validation but makes session
revocation harder (must wait for JWT expiry + check blacklist). Kratos's
server-side sessions enable instant revocation but require a DB lookup on every
request.

### 3.7 Social Login

| Provider | Kratos | GGID |
|----------|--------|------|
| Google | Via OIDC provider config | `pkg/social/google.go` |
| GitHub | Via OIDC provider config | `pkg/social/github.go` |
| Apple | Via OIDC provider config | `pkg/social/apple.go` |
| Microsoft | Via OIDC provider config | `pkg/social/microsoft.go` |
| Discord | Via OIDC provider config | `pkg/social/discord.go` |
| Slack | Via OIDC provider config | `pkg/social/slack.go` |
| LinkedIn | Via OIDC provider config | `pkg/social/linkin.go` |
| GitLab | Via OIDC provider config | `pkg/social/gitlab.go` |
| Generic OIDC | Via OIDC provider config | `pkg/social/oidc.go` |
| Configuration | Runtime (YAML/API) | Runtime (per-tenant config) |

**Key difference:** Kratos configures social providers via a YAML file or API.
GGID has dedicated connector packages per provider with a registry pattern
(`pkg/social/registry.go`). Both approaches support all major providers.

### 3.8 Feature Comparison Summary

| Feature | Kratos | GGID (identity + auth) |
|---------|--------|----------------------|
| Registration | Flow-based, schema-driven | API-based, simpler |
| Login | Flow-based, multi-method | API-based, multi-method |
| MFA (TOTP) | Built-in | Built-in |
| MFA (WebAuthn) | Built-in (passkeys) | Built-in |
| MFA (SMS) | Via twilio | Not implemented |
| LDAP | Via external IdP | Built-in (`pkg/authprovider`) |
| SAML | Via Hydra federation | Built-in (`pkg/saml`) |
| Account recovery | Flow-based, multi-channel | API-based, email only |
| Email verification | Flow-based | API-based |
| Identity schema | JSON Schema (runtime) | Go structs (compiled) |
| Profile management | Settings flow | REST API |
| Webhooks | Built-in per flow | Not built-in |
| Social login | OIDC provider config | 9 dedicated connectors |
| Session model | Server-side (cookie) | JWT + Redis |
| Multi-tenancy | Enterprise only | Built-in (RLS) |
| Organizations | Enterprise only | Built-in `org` service |
| SCIM | Not built-in | Skeleton (endpoints exist) |

---

## 4. Hydra vs GGID OAuth Service

### 4.1 Design Philosophy

The most fundamental difference between Hydra and GGID's OAuth service is their
**separation of concerns** approach.

#### Hydra: Strict Separation

Hydra **never handles login UI**. When a user initiates an OAuth flow:

1. User visits `https://hydra/oauth2/auth?client_id=...&redirect_uri=...`
2. Hydra creates a **login request** and redirects to YOUR login URL
3. Your application authenticates the user (using Kratos or anything else)
4. Your application calls `PUT /oauth2/auth/requests/login/accept` with the
   user's identity
5. Hydra creates a **consent request** and redirects to YOUR consent URL
6. User grants/denies consent
7. Your application calls `PUT /oauth2/auth/requests/consent/accept`
8. Hydra issues authorization code and redirects back to client
9. Client exchanges code for tokens

This means **Hydra never sees user credentials**. Your login UI is completely
decoupled from the OAuth server.

#### GGID: Integrated Auth + OAuth

GGID's auth and OAuth services are separate but **designed to work together**:

1. User authenticates via auth service: `POST /api/v1/auth/login`
2. Auth service returns JWT
3. For OAuth flows, the OAuth service (`services/oauth/internal/service/oauth_service.go`)
   handles authorization code flow
4. OAuth service trusts JWT from auth service (via shared key)
5. No separate login redirect — the OAuth flow can use the existing JWT

**GGID's `OAuthService` struct** shows the integrated design:

```go
type OAuthService struct {
    clientRepo   repository.ClientRepository
    codeRepo     repository.AuthorizationCodeRepository
    tokenRepo    repository.IDTokenRepository
    keyProvider  domain.KeyProvider
    issuer       string
}
```

The OAuth service manages clients, authorization codes, and tokens — all in the
shared PostgreSQL database with tenant isolation.

### 4.2 OAuth Grant Types

| Grant Type | Hydra | GGID |
|------------|-------|------|
| Authorization Code | Yes | Yes |
| Authorization Code + PKCE | Yes (recommended) | Yes (`RequirePKCE` flag) |
| Client Credentials | Yes | Yes |
| Refresh Token | Yes | Yes |
| Resource Owner Password | Yes (deprecated) | Yes |
| Device Authorization (RFC 8628) | Yes | No |
| Token Exchange (RFC 8693) | Yes | No |
| CIBA (Client-Initiated Backchannel Auth) | Planned | Yes (`ciba.go`) |

**GGID actually leads in CIBA** — the OAuth service includes `ciba.go` for
client-initiated backchannel authentication, which Hydra does not yet fully
support. GGID also has:

- `dpop.go` — DPoP (Demonstration of Proof-of-Possession)
- `jar_mtls.go` — JAR (JWT-Secured Authorization Requests) + mTLS
- `par.go` — PAR (Pushed Authorization Requests)
- `rfc7523.go` — RFC 7523 JWT Profile for OAuth 2.0
- `key_rotation.go` — JWKS rotation with grace period
- `consent.go` — Consent management
- `logout.go` — RP-initiated logout

### 4.3 Token Management

| Feature | Hydra | GGID |
|---------|-------|------|
| **Default token format** | Opaque (with introspection) | JWT |
| **JWT support** | Configurable | Default |
| **Token signing** | RS256, ES256, HS256 | RS256 |
| **Token introspection** | RFC 7662 endpoint | Not implemented |
| **Token revocation** | RFC 7009 endpoint | Yes |
| **JWKS endpoint** | Yes (`/.well-known/jwks.json`) | Yes |
| **Key rotation** | Manual via admin API | `key_rotation.go` with grace period |
| **Access token lifetime** | Configurable per client | Configurable |
| **Refresh token rotation** | Yes | Yes |
| **Refresh token reuse detection** | Yes | Yes (Redis-backed) |

**GGID's key rotation** is notable — `services/oauth/internal/service/key_rotation.go`
implements a `RotatingKeyProvider` that:
- Maintains active and previous keys
- Supports a grace period where both old and new keys are valid
- Automatically generates new keys at configurable intervals
- Exposes old keys in JWKS for token verification during transition

### 4.4 Client Management

| Feature | Hydra | GGID |
|---------|-------|------|
| Client types | Confidential, Public | Confidential, Public |
| Client registration | Admin API + RFC 7591 | REST API |
| Dynamic client management | RFC 7592 | Not implemented |
| PKCE enforcement | Configurable | `RequirePKCE` per client |
| Client scopes | Per-audience scopes | Per-client scopes |
| Client metadata | Custom claims | `map[string]any` metadata |
| Client secret hashing | BCrypt | Argon2id |

**GGID client model** (`services/oauth/internal/domain/models.go`):

```go
type OAuthClient struct {
    ID                      uuid.UUID
    TenantID                uuid.UUID
    ClientID                string
    ClientSecretHash        string    // Argon2id hash
    Name                    string
    Type                    ClientType
    GrantTypes              []string
    ResponseTypes           []string
    RedirectURIs            []string
    Scopes                  []string
    TokenEndpointAuthMethod string
    Metadata                map[string]any
    RequirePKCE             bool
    Enabled                 bool
}
```

### 4.5 Consent Handling

#### Hydra Consent

Hydra has a **dedicated consent flow** with:
- Configurable consent UI (your application renders it)
- Per-scope consent tracking
- Remember consent (skip on subsequent requests)
- Consent revocation
- Webhooks on consent events

#### GGID Consent

GGID has consent handling in `services/oauth/internal/service/consent.go`:
- Basic consent screen support
- Per-client scope consent
- Consent stored in database
- No webhook on consent events

### 4.6 OIDC Certification

**This is Ory's most significant OAuth advantage.**

Hydra is **OIDC Certified** by the OpenID Foundation. It has passed conformance
tests for:
- Basic OP (OpenID Provider)
- Implicit OP
- Hybrid OP
- Config OP
- Form Post OP
- Dynamic OP

GGID's OAuth implementation is **not certified**. While functionally complete for
many use cases, certification is required for:
- Regulated industries (banking, healthcare, government)
- Enterprise procurement (many enterprises require OIDC certification)
- Interoperability guarantees
- Insurance/liability requirements

**Achieving OIDC certification should be GGID's top OAuth priority.** The process
involves running the OpenID Foundation conformance test suite, fixing failures,
and applying for certification.

### 4.7 Feature Comparison Summary

| Feature | Hydra | GGID |
|---------|-------|------|
| Authorization Code | Yes | Yes |
| PKCE | Yes | Yes |
| Client Credentials | Yes | Yes |
| Refresh Token | Yes | Yes |
| Resource Owner Password | Yes | Yes |
| Device Authorization | Yes | No |
| Token Exchange | Yes | No |
| CIBA | Planned | **Yes** |
| DPoP | Yes | **Yes** |
| PAR | Yes | **Yes** |
| JAR + mTLS | Yes | **Yes** |
| Key Rotation | Manual | **Automated** (grace period) |
| OIDC Certification | **Yes** | No |
| Token Introspection | **Yes** (RFC 7662) | No |
| Dynamic Client Registration | **Yes** (RFC 7591/7592) | No |
| Consent Flow | **Advanced** | Basic |
| Login UI Separation | **Strict** | Integrated |

**Key finding:** GGID has caught up significantly on advanced OAuth features
(CIBA, DPoP, PAR, JAR, key rotation). The remaining gaps are OIDC certification,
token introspection, device authorization grant, and dynamic client registration.

---

## 5. Keto vs GGID Policy Service

### 5.1 Authorization Model Comparison

This is the most architecturally divergent comparison — the two systems use
**completely different authorization paradigms**.

#### Keto: Google Zanzibar Relation Tuples

Keto implements Google's **Zanzibar** paper. Authorization is expressed as
relation tuples:

```
object#relation@subject
```

Examples:
```
document:annual-report#editor@user:alice
document:annual-report#viewer@user:bob
document:annual-report#viewer@group:finance#member
repo:ggid#owner@organization:ggid#member
folder:projects#parent@folder:root
```

**Checking permissions:**
```
POST /relation-tuples/check
{ "namespace": "document", "object": "annual-report",
  "relation": "editor", "subject_id": "user:alice" }
→ { "allowed": true }
```

**How it works:**
1. You write relation tuples to Keto (e.g., "alice is editor of report")
2. Keto stores tuples and builds an in-memory graph
3. Check queries traverse the graph: "is alice editor of report?" → O(1) if cached
4. Supports transitive relationships: "alice is member of finance, finance has
   viewer on report → alice can view report"

**Zanzibar strengths:**
- **Fine-grained per-resource authorization**: Perfect for "can user X do Y to
  resource Z?"
- **O(1) cached lookups**: Once the graph is warm, checks are sub-millisecond
- **Relationship traversal**: "Can user X access folder Y?" traverses parent
  relationships transitively
- **Scales to millions of tuples**: Designed for Google-scale systems
- **Expressive**: Can model complex sharing hierarchies, org structures, and
  delegation chains

**Zanzibar weaknesses:**
- **No policy language**: Everything is tuples — no "if-then" rules
- **No ABAC**: Cannot express attribute-based rules (e.g., "users in department X
  with clearance level Y can access resources marked Z")
- **No deny rules**: Zanzibar is allow-only; implementing deny requires careful
  tuple design
- **Operational complexity**: The tuple space grows linearly with resources ×
  relationships; managing millions of tuples requires operational discipline
- **No temporal policies**: Cannot express "allow only during business hours"

#### GGID: RBAC + ABAC Policy Engine

GGID's policy service (`services/policy/internal/service/evaluator.go`) implements
a hybrid RBAC + ABAC engine:

**Evaluation order (from source):**
1. Resolve user's roles including inherited ancestors
2. Collect permissions from all roles — if any matches, RBAC allows
3. Collect ABAC policies attached to the user and their roles
4. Deny policies always override allow
5. Default deny if no explicit allow

**RBAC component:**
- Users are assigned roles (via `UserRole` entity)
- Roles have permissions (via `RolePermission` entity)
- Roles support inheritance (ancestor chains via `GetAncestorChain()`)
- Example: `admin` role inherits `editor` role inherits `viewer` role

**ABAC component:**
- Policies have conditions (JSON/YAML rules)
- Policies have priority (deny defaults to higher priority)
- Policies attach to users or roles (`PolicyAttachment`)
- Example: "deny access to resources marked 'confidential' if user clearance <
  'secret'"

**GGID's `CheckRequest` and `CheckResult` model:**

```go
type CheckRequest struct {
    UserID   uuid.UUID
    TenantID uuid.UUID
    Action   string    // e.g., "read", "write", "delete"
    Resource string    // e.g., "documents:annual-report"
}

type CheckResult struct {
    Allowed  bool
    Reason   string
    MatchedBy string  // which policy/role allowed/denied
}
```

### 5.2 Expressiveness Comparison

| Authorization Pattern | Keto (Zanzibar) | GGID (RBAC+ABAC) |
|-----------------------|-----------------|-------------------|
| **Role-based** (users with role X can do Y) | Via tuples | Native |
| **Resource ownership** (user X owns resource Y) | Via tuples | Via policies |
| **Group membership** (members of group X can access Y) | Via tuples | Via roles + groups |
| **Hierarchical inheritance** (parent → child permissions) | Transitive tuples | Role inheritance |
| **Attribute-based** (if user.dept = X and resource.classification = Y) | Not supported | Native (ABAC) |
| **Deny rules** (explicitly deny access) | Not native | Native (deny overrides) |
| **Time-based** (allow only during business hours) | Not supported | Via conditions |
| **Contextual** (allow if request IP in range X) | Not supported | Via conditions |
| **Delegation** (user X delegates permission to user Y) | Via tuples | Via policies |
| **Per-object sharing** (share document Y with user Z) | Native | Requires policy per object |
| **Transitive membership** (org → group → subgroup → user) | Native | Via role inheritance |
| **Policy composition** (combine multiple rules) | Not supported | Native (priority + deny) |

### 5.3 Performance Comparison

| Metric | Keto | GGID |
|--------|------|------|
| **Check latency (warm)** | <1ms (cached) | 2-10ms (PostgreSQL query) |
| **Check latency (cold)** | 5-20ms (graph build) | 2-10ms (consistent) |
| **Throughput** | 100K+ checks/sec (cached) | ~10K checks/sec (DB-bound) |
| **Caching** | In-memory graph cache | Decision log cache |
| **Scalability limit** | Memory (tuple cache size) | PostgreSQL connection pool |
| **Tuple/policy count** | Millions of tuples | Thousands of policies |

**Keto's performance advantage** comes from its in-memory graph cache. Once tuples
are loaded, check queries are pure in-memory graph traversal — sub-millisecond
latency. This is ideal for high-volume, fine-grained authorization checks.

**GGID's performance model** queries PostgreSQL on every check. While this is
slower per-check, it provides:
- Consistent latency (no cold-start penalty)
- No memory pressure from large tuple sets
- Real-time policy updates (no cache invalidation needed)
- Transactional consistency with other services

**GGID has a decision log** for observability — `DecisionEntry` records every
evaluation with timestamp, user, action, resource, allowed/denied, and the matching
policy. This is valuable for audit and debugging but adds overhead.

### 5.4 API Design Comparison

#### Keto API

```
# Write a relation tuple
PUT /admin/relation-tuples
{ "namespace": "document", "object": "report.pdf",
  "relation": "editor", "subject_id": "alice" }

# Check a permission
POST /relation-tuples/check
{ "namespace": "document", "object": "report.pdf",
  "relation": "editor", "subject_id": "alice" }
→ { "allowed": true }

# List relations (expand)
GET /relation-tuples/expand?namespace=document&object=report.pdf&relation=editor
→ { "subjects": ["alice", "bob", "group:finance#member"] }
```

#### GGID API

```
# Create a role
POST /api/v1/roles
{ "tenant_id": "...", "name": "editor", "key": "doc-editor",
  "permissions": [{"action": "write", "resource": "documents"}] }

# Assign role to user
POST /api/v1/roles/{role_id}/users
{ "user_id": "alice" }

# Check permission
POST /api/v1/permissions/check
{ "user_id": "alice", "tenant_id": "...",
  "action": "write", "resource": "documents" }
→ { "allowed": true, "matched_by": "role:editor" }

# Create ABAC policy
POST /api/v1/policies
{ "tenant_id": "...", "name": "confidential-deny",
  "effect": "deny", "priority": 100,
  "conditions": {"resource.classification": "confidential",
                 "user.clearance": {"$lt": "secret"}} }
```

### 5.5 When to Choose Which

**Choose Keto when:**
- You need fine-grained per-resource authorization (Google Docs, Figma, Notion
  style)
- You have millions of resources with per-resource sharing rules
- You need transitive relationship traversal (org hierarchies)
- You don't need attribute-based policies
- You can tolerate the operational complexity of tuple management

**Choose GGID Policy when:**
- You need RBAC with role inheritance (enterprise access management)
- You need ABAC with attribute conditions (e.g., department, clearance level)
- You need explicit deny rules (compliance, regulatory)
- You need multi-tenant policy isolation (tenant_id in every policy)
- You want simpler operational model (PostgreSQL queries, no graph cache)

### 5.6 Could GGID Add Zanzibar?

GGID could add a Zanzibar-style authorization layer as an optional mode in the
policy service. The `evaluator.go` already has the `CheckRequest`/`CheckResult`
abstraction. A future enhancement could:

1. Add a `RelationTuple` entity and repository
2. Add an in-memory graph cache (using `github.com/dgraph-io/ristretto` or similar)
3. Extend `Evaluator.Check()` to query both the RBAC/ABAC engine and the relation
  graph
4. Merge results: if either engine allows, allow (configurable)

This would give GGID the best of both worlds — enterprise RBAC/ABAC plus
fine-grained Zanzibar-style authorization.

---

## 6. Oathkeeper vs GGID Gateway

### 6.1 Role Comparison

Both Oathkeeper and GGID's gateway act as **Policy Enforcement Points (PEP)** —
they intercept HTTP traffic and enforce authentication/authorization before
forwarding to backend services. However, their architectures are very different.

#### Oathkeeper: Plugin-Based Pipeline

Oathkeeper processes each request through a configurable pipeline:

```
Request → [Match Access Rule] → [Authenticator] → [Authorizer] → [Mutator] → [Forward to Backend]
```

**Access Rules** are defined in YAML:

```yaml
# oathkeeper.yml
access_rules:
  matching_handlers:
    - match:
        url: "http://<.*>/api/users/<.*>"
        methods: ["GET", "POST"]
      authenticators:
        - handler: cookie_session
        - handler: anonymous        # fallback if no session
      authorizer:
        handler: keto
        config:
          check_url: "http://keto:4466/relation-tuples/check"
      mutators:
        - handler: id_token
        - handler: header
          config:
            headers:
              X-User-ID: "{{ print .Subject }}"
```

**Available handlers:**

| Stage | Handlers |
|-------|----------|
| **Authenticators** | `cookie_session`, `jwt`, `oauth2_introspection`, `anonymous`, `noop`, `bearer_token`, `unauthorized` |
| **Authorizers** | `allow`, `deny`, `keto` (Zanzibar check), `remote_json` (custom HTTP) |
| **Mutators** | `header` (inject headers), `id_token` (create JWT), `cookie`, `hydrator` (fetch data), `noop` |

**Oathkeeper's key advantage:** the plugin system is extensible. You can build
custom authenticators, authorizers, or mutators as Go plugins. New authentication
methods can be added without modifying core code.

#### GGID Gateway: Middleware Chain

GGID's gateway (`services/gateway/internal/router/router.go`) uses a middleware
chain with programmatic configuration:

```go
type Gateway struct {
    cfg           *config.Config
    jwks          *middleware.JWKSClient
    proxies       map[string]*httputil.ReverseProxy
    timeouts      map[string]time.Duration
    healthChecker *healthcheck.Checker
    rateLimiter   *middleware.TenantBucketLimiter
    stats         *middleware.StatsCollector
    graphql       *middleware.GraphQLResolver
    sessionMgr    *middleware.SessionManager
}
```

**Gateway middleware** (from source):

| Middleware | File | Purpose |
|------------|------|---------|
| JWT verification | `router.go` | Verify JWT on protected routes |
| Rate limiting | `ratelimit.go` | Per-tenant sliding window rate limiting |
| Tenant context | `tenant_context.go` | Inject tenant_id from JWT/header |
| CORS | `per_tenant_cors.go` | Per-tenant CORS configuration |
| Gzip | `gzip.go` | Response compression |
| Circuit breaker | `router.go` | Per-service circuit breaking |
| Host validation | `host_validation.go` | DNS rebinding protection |
| IP allowlist | `ipallowlist.go` | IP-based access control |
| WASM plugin | `wasm_plugin.go` | WebAssembly plugin support |
| Response cache | `response_cache.go` | GET response caching |
| Metrics | `metrics.go` | Prometheus metrics |
| Recovery | `recovery.go` | Panic recovery |
| Error pages | `error_pages.go` | Custom error pages |
| gRPC-web | `grpcweb.go` | gRPC-Web transport |
| WebSocket proxy | `wsproxy.go` | WebSocket proxying |
| GraphQL | `graphql.go` | GraphQL query proxying |
| API key rotation | `apikey_rotation.go` | API key rotation |
| Sticky sessions | `sticky.go` | Session affinity |

### 6.2 Authentication Strategies

| Strategy | Oathkeeper | GGID Gateway |
|----------|------------|--------------|
| **JWT** | `jwt` authenticator | Built-in JWKS verification |
| **Cookie session** | `cookie_session` authenticator | Session manager |
| **OAuth2 introspection** | `oauth2_introspection` authenticator | Not implemented |
| **Bearer token** | `bearer_token` authenticator | Via JWT verification |
| **Anonymous** | `anonymous` authenticator | Public paths list |
| **API key** | Custom handler needed | `apikey_rotation.go` |
| **mTLS** | Custom handler needed | Planned |
| **WebAuthn** | Not built-in | Via auth service |

### 6.3 Authorization Strategies

| Strategy | Oathkeeper | GGID Gateway |
|----------|------------|--------------|
| **Zanzibar (Keto)** | `keto` authorizer | Not integrated |
| **RBAC/ABAC** | `remote_json` (custom HTTP) | Policy service integration |
| **Allow all** | `allow` authorizer | Public paths |
| **Deny all** | `deny` authorizer | Not implemented |
| **Remote HTTP** | `remote_json` authorizer | Policy service call |

### 6.4 Request Mutation

| Mutator | Oathkeeper | GGID Gateway |
|---------|------------|--------------|
| **Header injection** | `header` mutator | `X-Tenant-ID` injection |
| **ID token creation** | `id_token` mutator (JWT) | Not implemented |
| **Cookie injection** | `cookie` mutator | Not implemented |
| **Data hydration** | `hydrator` mutator (fetch data) | Not implemented |
| **Tenant context** | Not built-in | `tenant_context.go` |

### 6.5 Capabilities Beyond PEP

GGID's gateway has several capabilities that Oathkeeper lacks:

| Capability | Oathkeeper | GGID Gateway |
|------------|------------|--------------|
| **Rate limiting** | No | Yes (sliding window, per-tenant) |
| **Circuit breaker** | No | Yes (per-service) |
| **HTTP/3 support** | No | Yes |
| **Tenant routing** | No | Yes (X-Tenant-ID header + query param) |
| **Response caching** | No | Yes (`response_cache.go`) |
| **WASM plugins** | No | Yes (`wasm_plugin.go`) |
| **WebSocket proxy** | No | Yes (`wsproxy.go`) |
| **gRPC-Web** | No | Yes (`grpcweb.go`) |
| **GraphQL proxy** | No | Yes (`graphql.go`) |
| **Metrics/monitoring** | No (external) | Yes (Prometheus) |
| **Health checks** | No (external) | Yes (`healthcheck/`) |
| **Plugin system** | Yes (auth/authorizer/mutator) | No (middleware chain) |
| **Declarative config** | Yes (YAML rules) | No (Go code) |

### 6.6 Configuration Model

**Oathkeeper uses declarative YAML** — access rules are defined in a configuration
file and can be hot-reloaded. This is a significant advantage for operations:

```yaml
# Each route has its own auth/authz/mutation pipeline
- match: { url: "http://<.*>/api/admin/<.*>" }
  authenticators: [{ handler: jwt }]
  authorizer: { handler: keto }
  mutators: [{ handler: id_token }]
```

**GGID uses programmatic configuration** — routes are defined in Go code within
`router.go`. The `publicPaths` list is hardcoded:

```go
var publicPaths = []string{
    "/api/v1/auth/login",
    "/api/v1/auth/register",
    // ...
}
```

Adding a new route or changing authentication for an existing route requires a
code change and redeployment.

### 6.7 Verdict

**Oathkeeper wins on:**
- Plugin extensibility (custom authenticators/authorizers/mutators)
- Declarative YAML configuration (hot-reloadable)
- Fine-grained per-route authentication pipeline
- Keto integration for Zanzibar-style authorization

**GGID Gateway wins on:**
- Rate limiting (built-in, per-tenant)
- Circuit breaker (built-in, per-service)
- HTTP/3 support
- WebSocket proxying
- Response caching
- WASM plugin support
- Tenant routing
- Metrics/monitoring
- Health checks

The two gateways serve different needs. Oathkeeper is a **dedicated PEP** focused
on authentication/authorization flexibility. GGID's gateway is a **full-featured
API gateway** with auth capabilities plus operational features (rate limiting,
circuit breaking, caching, metrics).

---

## 7. Developer Experience

### 7.1 Five-Minute Experience

#### Ory: Getting Started

**Option A: Ory Network (Fastest)**

```bash
# Install Ory CLI
brew install ory/ory/ory

# Create account and project
ory create account
ory create project my-app

# Use immediately
ory create identity --schema email --traits '{"email":"alice@example.com"}'
```

**Time to first API call: ~5 minutes** (if you have the CLI installed).

**Option B: Self-Hosted (Slower)**

```bash
# Clone the quickstart
git clone https://github.com/ory/kratos
cd kratos

# Run with Docker Compose
docker compose up

# This starts: Kratos + PostgreSQL + Mailslurper
# But you still need Hydra for OAuth, Keto for authz, Oathkeeper for proxy
```

**Time to full stack running: 30-60 minutes** (wiring 4 services together).

The self-hosting experience is complex because:
1. Each service has its own config file
2. Services must be wired together (Kratos → Hydra consent flow)
3. Each service needs its own database
4. No single "unified" config — each service reads its own YAML
5. Identity schemas must be pushed to Kratos separately

#### GGID: Getting Started

```bash
# Clone the repo
git clone https://github.com/ggid/ggid
cd ggid

# Start the entire stack
cd deploy && docker compose up -d

# Wait for healthchecks
sleep 30

# Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"Password123!","tenant_id":"00000000-0000-0000-0000-000000000001"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"Password123!","tenant_id":"00000000-0000-0000-0000-000000000001"}'
```

**Time to first API call: ~5 minutes** (Docker Compose handles everything).

GGID's `docker-compose.yaml` starts **13 containers** (7 services + PostgreSQL +
Redis + NATS + LDAP + Console + MailHog) with one command. All services are
pre-wired, migrations run automatically, and the default tenant is seeded.

### 7.2 SDK Quality Comparison

#### Ory SDKs

Ory auto-generates SDKs from their OpenAPI specification using
`openapi-generator`. This gives them **7+ language SDKs**:

| Language | Package | Quality |
|----------|---------|---------|
| Go | `github.com/ory/client-go` | Good — auto-generated |
| JavaScript/TypeScript | `@ory/client` | Good — npm |
| Python | `ory-client` | Good — PyPI |
| PHP | `ory/client` | Good — Composer |
| Rust | `ory-client` | Good — crates.io |
| .NET/C# | `Ory.Client` | Good — NuGet |
| Java | `sh.ory.krautos` | Good — Maven |
| Dart | `ory-client` | Community — pub.dev |

**Ory SDK advantages:**
- Broad language coverage (7+ languages)
- Auto-generated — always in sync with API
- Published to package registries
- Type-safe in each language

**Ory SDK disadvantages:**
- Auto-generated code is verbose and less ergonomic
- Less hand-tuned for each language's idioms
- Generated code can be hard to read/debug
- Some edge cases in error handling

#### GGID SDKs

GGID has **hand-written SDKs** in 4 languages:

| Language | Package | Quality |
|----------|---------|---------|
| Go | `sdk/go/` | High — hand-written, idiomatic |
| Node/TypeScript | `sdk/node/` | High — hand-written, typed |
| Python | `sdk/python/` | High — hand-written, typed |
| Java | `sdk/java/` | High — hand-written, Maven |

**GGID SDK advantages:**
- Hand-written for idiomatic usage
- Includes middleware package (Go SDK: `sdk/go/middleware/`)
- Higher quality per language
- More ergonomic API design

**GGID SDK disadvantages:**
- Only 4 languages (vs Ory's 7+)
- Manual maintenance — must update for each API change
- No auto-generation pipeline
- Missing languages: PHP, Rust, .NET, Dart

The Go SDK (`sdk/go/client.go`) wraps REST API calls with typed methods:

```go
client := ggid.NewClient("https://api.ggid.io", "your-api-key")
user, err := client.Users.Create(ctx, &ggid.CreateUserInput{
    Username: "alice",
    Email:    "alice@example.com",
})
```

The middleware package (`sdk/go/middleware/`) provides HTTP middleware for
integrating GGID authentication into Go backend applications.

### 7.3 CLI Tooling

| Feature | Ory CLI | GGID |
|---------|---------|------|
| **Binary** | `ory` (brew install) | None |
| **Create project** | `ory create project` | N/A |
| **Identity management** | `ory create identity` | N/A |
| **Schema management** | `ory push schema` | N/A |
| **Proxy tunnel** | `ory proxy` (tunnel to Ory Network) | N/A |
| **Import/export** | `ory import identities` | N/A |
| **Dev server** | `ory dev` | `docker compose up` |

**Ory's CLI is a significant DX advantage.** It enables:
- Quick identity creation for testing
- Schema management without API calls
- Local proxy tunnel to Ory Network (use cloud APIs in local dev)
- Identity import/export for migration

GGID lacks a CLI tool entirely. All management is via REST API or the Admin
Console (Next.js).

### 7.4 Admin Console

| Feature | Ory Console | GGID Console |
|---------|-------------|--------------|
| **Framework** | Next.js (React) | Next.js 15 (React) |
| **Pages** | Dashboard, Identities, Projects, Settings | 20+ pages |
| **User management** | Yes | Yes |
| **Role management** | Yes (Enterprise) | Yes |
| **OAuth clients** | Yes | Yes |
| **Audit logs** | Enterprise only | Yes |
| **Organizations** | Enterprise only | Yes |
| **API explorer** | No | Yes |
| **Onboarding wizard** | No | Yes |
| **Branding** | Enterprise | Yes |
| **Monitoring** | Enterprise | Yes |
| **Certificates** | No | Yes |
| **Custom flows** | Yes | Yes |

GGID's console has more pages and features than Ory's open-source console.
Notably, GGID has:
- `access-keys/page.tsx` — API key management
- `api-explorer/page.tsx` — interactive API exploration
- `onboarding/page.tsx` — setup wizard
- `certificates/page.tsx` — certificate management
- `monitoring/page.tsx` — system health monitoring
- `flows/page.tsx` — custom auth flows
- `branding/page.tsx` — white-label branding

### 7.5 Documentation

| Aspect | Ory | GGID |
|--------|------|------|
| **Docs site** | ory.com/docs (extensive) | `docs/` directory (growing) |
| **Quickstart** | Multiple quickstarts per project | `docs/getting-started.md` |
| **API reference** | OpenAPI/Swagger UI per service | Planned |
| **Tutorials** | Step-by-step guides | `docs/tutorials/` (4 guides) |
| **Research docs** | — | 140+ research documents |
| **Architecture docs** | Multiple ADRs | Architecture docs in `docs/` |
| **Video tutorials** | YouTube channel | None |

---

## 8. Self-Hosting vs Managed

### 8.1 Deployment Models

#### Ory: Three Tiers

| Tier | Model | Multi-Tenancy | Cost |
|------|-------|---------------|------|
| **Ory Network (Free)** | Managed SaaS | Project-level isolation | Free up to 10K MAU |
| **Ory Network (Pro)** | Managed SaaS | Project-level isolation | $0.02/MAU |
| **Ory Enterprise** | Self-hosted license | True multi-tenancy | Custom pricing |
| **Ory OSS** | Self-hosted (Apache 2.0) | **Single-tenant only** | Free |

**Critical distinction:** Ory's open-source version is explicitly single-tenant.
Multi-tenancy requires either:
- Ory Network (managed SaaS with project-per-tenant isolation)
- Ory Enterprise License (paid self-hosted with multi-tenant features)

#### GGID: One Tier

| Tier | Model | Multi-Tenancy | Cost |
|------|-------|---------------|------|
| **GGID OSS** | Self-hosted (Apache 2.0) | **Built-in (RLS)** | Free |

GGID has no managed SaaS offering. Multi-tenancy is free and built-in.

### 8.2 Total Cost of Ownership (TCO)

#### Small Scale (1,000 users)

| Cost Component | Ory Network (Free) | Ory OSS (Self-Hosted) | GGID (Self-Hosted) |
|----------------|-------------------|----------------------|---------------------|
| Licensing | $0 (free tier) | $0 | $0 |
| Infrastructure | $0 (managed) | ~$200/mo (1 VM) | ~$200/mo (1 VM) |
| Operations | $0 (managed) | ~$2K/mo (engineer time) | ~$1K/mo (simpler ops) |
| **Total/month** | **$0** | **~$2,200** | **~$1,200** |

#### Medium Scale (50,000 MAU)

| Cost Component | Ory Network (Pro) | Ory OSS (Self-Hosted) | GGID (Self-Hosted) |
|----------------|-------------------|----------------------|---------------------|
| Licensing | ~$800/mo ($0.02/MAU above free) | $0 | $0 |
| Infrastructure | $0 (managed) | ~$1K/mo (2-3 VMs) | ~$1K/mo (2-3 VMs) |
| Operations | $0 (managed) | ~$4K/mo (engineer time) | ~$2K/mo (simpler ops) |
| **Total/month** | **~$800** | **~$5,000** | **~$3,000** |

#### Enterprise Scale (500,000 MAU, multi-tenant)

| Cost Component | Ory Network (Enterprise) | Ory Enterprise (Self-Hosted) | GGID (Self-Hosted) |
|----------------|-------------------------|------------------------------|---------------------|
| Licensing | Custom ($10K+/mo) | Custom ($10K+/mo) | $0 |
| Infrastructure | $0 (managed) | ~$5K/mo (K8s cluster) | ~$5K/mo (K8s cluster) |
| Operations | $0 (managed) | ~$10K/mo (DevOps team) | ~$5K/mo (simpler ops) |
| **Total/month** | **~$10K+** | **~$15K+** | **~$10K** |

**Key finding:** At enterprise scale with multi-tenancy requirements, GGID's TCO
is competitive because multi-tenancy is free. Ory requires either Enterprise
License (self-hosted) or Enterprise Network plan (managed) for multi-tenancy.

### 8.3 What GGID Would Need for a Managed Offering

To compete with Ory Network, GGID would need:

1. **Control Plane Service** — manage tenant provisioning, billing, and
   configuration at scale
2. **Multi-Region Deployment** — deploy GGID instances in multiple regions for
   latency and compliance
3. **Automated Tenant Isolation** — RLS policies provide data isolation, but
   managed offering needs tenant lifecycle management (create, suspend, delete)
4. **Usage Metering** — track MAU, API calls, storage per tenant for billing
5. **SLA Infrastructure** — monitoring, alerting, incident response, uptime
   guarantees
6. **Compliance Certifications** — SOC 2, ISO 27001, GDPR DPA, HIPAA BAA
7. **Customer Support** — documentation, ticketing, on-call support
8. **Data Residency** — EU, US, APAC data residency options
9. **Disaster Recovery** — automated backups, cross-region replication, RTO/RPO
10. **Self-Service Portal** — customer signup, billing, configuration UI

**Estimated effort:** 15-20 engineer-months for MVP managed offering.

### 8.4 GGID's Self-Hosting Advantage

GGID's self-hosting experience is simpler than Ory's:

| Aspect | Ory OSS (Self-Hosted) | GGID (Self-Hosted) |
|--------|----------------------|---------------------|
| **Services to deploy** | 4 (Kratos, Hydra, Keto, Oathkeeper) | 7 (but pre-wired) |
| **Databases** | 3-4 (one per service) | 1 (shared PostgreSQL) |
| **Config files** | 4 (one per service) | 1 (docker-compose.yaml) |
| **Inter-service wiring** | Manual (consent flow, login redirect) | Automatic |
| **Docker Compose** | Per-service (not unified) | Unified (13 containers, one command) |
| **Helm charts** | Production-grade (per service) | Skeleton (deploy/helm/) |
| **Kubernetes operator** | Community operator | None |
| **Migrations** | Per-service migration tools | Unified migration init container |

---

## 9. Performance

### 9.1 Architecture Performance Implications

Both Ory and GGID are written in **Go** — a compiled, garbage-collected language
with excellent concurrency support. This gives both platforms inherent performance
advantages over JVM-based (Keycloak) or Node.js-based (Auth0) alternatives.

However, the architectural differences create distinct performance profiles.

### 9.2 Latency Comparison

| Operation | Ory (Estimated) | GGID (Estimated) | Notes |
|-----------|----------------|-----------------|-------|
| **User registration** | 50-100ms | 30-80ms | GGID: fewer services involved |
| **User login** | 30-80ms | 20-60ms | GGID: direct DB lookup + JWT |
| **JWT verification** | 1-5ms | 1-5ms | Both: JWKS cached locally |
| **Session check (cookie)** | 5-15ms | N/A (JWT) | Kratos: DB lookup; GGID: stateless |
| **Permission check** | <1ms (cached) | 2-10ms | Keto: in-memory; GGID: DB query |
| **OAuth authorize** | 100-200ms | 50-150ms | Hydra: 2 redirects; GGID: integrated |
| **Token issuance** | 20-50ms | 10-40ms | Both: sign JWT + DB write |
| **User lookup by ID** | 5-15ms | 5-15ms | Both: PostgreSQL indexed query |

### 9.3 Database Performance

#### Ory: Separate Databases

Each Ory service has its own database. This means:
- **Connection pools**: 3-4 separate pools (Kratos, Hydra, Keto DBs)
- **No cross-service joins**: User data and OAuth tokens are in separate databases
- **Migration overhead**: Each service runs its own migrations
- **Resource isolation**: A spike in Hydra queries doesn't affect Kratos's DB
- **Scaling**: Each DB can be scaled independently

#### GGID: Shared Database with RLS

GGID uses a single PostgreSQL instance with RLS:
- **Connection pool**: One pool, shared across all services
- **Cross-service joins**: Possible (but not used heavily)
- **Single migration pipeline**: All migrations in `deploy/migrations/`
- **Resource contention**: A spike in one service can affect others
- **Scaling**: Scale PostgreSQL vertically, or use read replicas

**RLS Performance Impact:** PostgreSQL RLS adds a predicate to every query:
```sql
SELECT * FROM users WHERE tenant_id = current_setting('app.current_tenant_id')::uuid;
```
This adds ~0.1-0.5ms per query for the predicate evaluation. With proper
indexing on `tenant_id`, this is negligible.

### 9.4 Caching Strategy

| Component | Ory | GGID |
|-----------|-----|------|
| **Session cache** | In-memory (Kratos) | Redis |
| **Permission cache** | In-memory graph (Keto) | Decision log (in-memory) |
| **JWKS cache** | In-memory | `JWKSClient` with TTL |
| **Token cache** | Not cached (DB lookup) | JWT verification cache |
| **Rate limit state** | Not applicable | In-memory (per-instance) |
| **Config cache** | In-memory | In-memory |

**Keto's in-memory graph cache** is the most sophisticated caching in either
system. It builds a relationship graph in memory and serves check queries without
database access. This enables sub-millisecond permission checks at scale.

GGID's caching is simpler — Redis for sessions, in-memory for JWKS. Permission
checks always hit the database, which is slower but more consistent.

### 9.5 Throughput

| Metric | Ory (Estimated) | GGID (Estimated) |
|--------|----------------|-----------------|
| **Auth requests/sec** | ~10K/sec per instance | ~10K/sec per instance |
| **Permission checks/sec** | ~100K/sec (Keto cached) | ~5K/sec (DB-bound) |
| **Token issuance/sec** | ~5K/sec | ~5K/sec |
| **Max concurrent sessions** | Millions (DB-backed) | Millions (Redis-backed) |

Both platforms should handle comparable throughput for authentication operations.
The main difference is permission checking throughput, where Keto's cache gives it
a 20x advantage.

### 9.6 Resource Consumption

| Resource | Ory (4 services) | GGID (7 services) |
|----------|-----------------|-------------------|
| **Memory (idle)** | ~500MB total | ~400MB total |
| **Memory (active)** | ~1-2GB total | ~1-2GB total |
| **CPU (idle)** | ~5% total | ~5% total |
| **CPU (active)** | ~20-40% total | ~20-40% total |
| **DB connections** | 30-60 (3-4 pools × 10-15) | 15-25 (1 pool) |
| **Docker image size** | ~30-50MB per service | ~18-35MB per service |

GGID's Docker images are smaller (18-35MB vs 30-50MB) due to optimized multi-stage
builds and shared binary patterns.

### 9.7 Performance Optimization Opportunities for GGID

1. **Add Redis-backed permission caching** — cache policy evaluation results in
   Redis with a 60-second TTL. This would reduce permission check latency from
   2-10ms to <1ms for cached decisions.
2. **Add connection pooling** — use PgBouncer or Pgpool-II to reduce PostgreSQL
   connection overhead.
3. **Add JWT cache** — cache parsed JWTs by `jti` claim to avoid re-parsing on
   every request.
4. **Add rate limit Redis backend** — the current rate limiter is in-memory
   (`ratelimit.go`), which doesn't work across multiple gateway instances. Use
   Redis-backed rate limiting for multi-instance deployments.
5. **Consider Keto-style graph cache** — add an optional in-memory graph cache
   for the policy service, especially for RBAC role hierarchies.

---

## 10. What GGID Can Learn from Ory

### 10.1 API Design Philosophy

Ory's APIs follow a consistent pattern that GGID could adopt more broadly:

**Self-Service Flows**: Kratos uses flow-based APIs where each identity interaction
is a self-contained flow object. This pattern enables:
- Multi-step interactions (registration with email verification)
- State tracking (flow state persisted in DB)
- Customizable UI (flow nodes → form fields)
- Webhook integration (fire webhooks at each step)

GGID's current API is simpler (single-step REST calls) but less flexible. Adding
optional flow-based APIs for registration, recovery, and settings would bring
Kratos-level flexibility while maintaining the simple API as default.

**Error Handling**: Ory uses structured error responses with:
- Machine-readable error IDs
- Human-readable messages
- Flow context (which flow, which step)
- Debug information (development only)

GGID's `pkg/errors` package already provides structured errors. Extending it with
flow context and error IDs would improve DX.

### 10.2 Self-Service Flows

Kratos's self-service flow model is one of its most powerful features. GGID could
benefit from adopting a similar pattern for:

1. **Registration Flow** — multi-step registration with email verification before
   activation, custom fields per tenant, webhook on completion
2. **Recovery Flow** — multi-channel recovery (email, SMS, backup codes) with
   configurable steps and expiry
3. **Settings Flow** — profile update with verification (e.g., require password
   re-entry for sensitive changes)
4. **Verification Flow** — email/phone verification with configurable methods
   (link, code, QR)

**Implementation approach:** Add a `flows` service or package that manages flow
state, UI nodes, and webhook dispatch. Flows would be optional — the existing
simple API would remain as default.

### 10.3 SDK Generation Pipeline

Ory auto-generates SDKs from their OpenAPI spec. This gives them 7+ language SDKs
with minimal maintenance. GGID should:

1. **Define complete OpenAPI 3.1 spec** for all REST APIs
2. **Use `openapi-generator`** to produce SDKs for: Go, TypeScript, Python, Java,
   Rust, .NET, PHP, Dart
3. **Hand-tune generated code** for the top 3 languages (Go, TypeScript, Python)
4. **Publish to package registries** automatically in CI/CD
5. **Keep hand-written middleware SDK** (Go SDK `middleware/` package) as a
   separate, hand-maintained package

This would close the SDK language gap from 4 languages to 8+.

### 10.4 CLI Tooling

Ory's CLI (`ory`) is a significant DX advantage. GGID should build a `ggid` CLI
that provides:

```bash
# Tenant management
ggid tenant create --name "Acme Corp"
ggid tenant list
ggid tenant delete --id <uuid>

# User management
ggid user create --username alice --email alice@example.com
ggid user list --tenant <uuid>
ggid user import --file users.csv

# OAuth client management
ggid client create --name "My App" --redirect-uris "https://app.example.com/callback"
ggid client list

# Policy management
ggid policy create --file policy.yaml
ggid policy check --user alice --action read --resource documents

# Development
ggid dev start          # docker compose up
ggid dev migrate        # run migrations
ggid dev seed           # seed test data

# Configuration
ggid config show
ggid config set --key password.min_length --value 12
```

### 10.5 Declarative Configuration

Oathkeeper's YAML-based access rules enable hot-reloadable, declarative
configuration. GGID's gateway uses hardcoded route configuration in Go.

Adding a YAML-based configuration option for gateway routes would:
- Enable route changes without redeployment
- Allow GitOps-style configuration management
- Make routes auditable (version-controlled YAML)
- Support multi-environment configurations

### 10.6 Identity Schema (JSON Schema)

Kratos's JSON Schema-based identity model is extremely flexible. GGID's Go struct
model is type-safe but rigid. Consider adding:

1. **Optional custom attributes** — a `map[string]any` metadata field on the User
   entity, validated by a JSON Schema
2. **Per-tenant schemas** — each tenant can define custom identity attributes
3. **Schema migration** — update schema at runtime without code changes

### 10.7 Professional Security Audit

Ory has undergone professional security audits. GGID should:
1. Commission a security audit from a reputable firm
2. Publish the audit report (like Ory does)
3. Fix findings and publish remediation timeline
4. Establish a bug bounty program
5. Pursue SOC 2 Type II certification

### 10.8 OIDC Certification

Ory's Hydra is OIDC Certified. GGID should:
1. Run the OpenID Foundation conformance test suite
2. Fix failing test cases
3. Apply for official certification
4. Display certification badge in documentation
5. Market certification to enterprise customers

---

## 11. Gap Analysis & Recommendations

### 11.1 Priority Matrix

The following gaps are prioritized by **impact** (how much they matter for
adoption) and **effort** (engineer-months to implement):

| # | Gap | Impact | Effort | Priority |
|---|-----|--------|--------|----------|
| 1 | OIDC Certification | Critical | 2-4 months | **P0** |
| 2 | Token Introspection (RFC 7662) | High | 2 weeks | **P0** |
| 3 | Auto-generated SDKs (OpenAPI) | High | 1 month | **P1** |
| 4 | CLI tooling (`ggid` command) | Medium | 1 month | **P1** |
| 5 | Identity schema (JSON Schema traits) | Medium | 2 months | **P1** |
| 6 | Flow-based self-service APIs | Medium | 2-3 months | **P2** |
| 7 | Helm charts (production-grade) | Medium | 1 month | **P1** |
| 8 | Device Authorization Grant (RFC 8628) | Low | 2 weeks | **P2** |
| 9 | Token Exchange (RFC 8693) | Low | 2 weeks | **P2** |
| 10 | Zanzibar-style authorization mode | Medium | 2-3 months | **P2** |
| 11 | OAuth2 introspection authenticator (gateway) | Low | 1 week | **P2** |
| 12 | Security audit | High | 1 month + $ | **P1** |
| 13 | Dynamic Client Registration (RFC 7591/7592) | Low | 2 weeks | **P3** |
| 14 | Consent flow enhancement | Medium | 1 month | **P2** |
| 15 | Multi-step registration flows | Medium | 1-2 months | **P2** |
| 16 | Identity webhooks | Medium | 1 month | **P2** |
| 17 | Permission caching (Redis) | High | 2 weeks | **P1** |
| 18 | YAML-based gateway config | Low | 1 month | **P3** |
| 19 | Account recovery codes (backup codes) | Medium | 2 weeks | **P2** |
| 20 | Rate limit Redis backend | Medium | 1 week | **P1** |

### 11.2 Detailed Recommendations

#### P0: Immediate Actions (0-3 months)

**1. Achieve OIDC Certification**

This is the single most impactful action GGID can take. OIDC certification is a
hard requirement for:
- Banking and financial services (PSD2, Open Banking)
- Healthcare (HIPAA requires certified identity providers in some jurisdictions)
- Government procurement
- Enterprise security teams with certification requirements

**Steps:**
1. Clone the OpenID Foundation conformance test suite
2. Run tests against GGID's OAuth service
3. Fix failing tests (focus on: discovery endpoint, JWKS, ID token format,
   scope handling, nonce validation)
4. Apply for certification at certify.openid.net
5. Pay certification fee (~$1,500-$5,000 per profile)
6. Display certification badge prominently

**2. Implement Token Introspection (RFC 7662)**

Add a `POST /oauth/introspect` endpoint to the OAuth service:
```go
func (s *OAuthService) IntrospectToken(ctx context.Context, token string) (*IntrospectionResult, error) {
    // Parse JWT, validate signature, check expiry
    // Return active/inactive + claims
}
```

This is required for resource servers that need to validate opaque tokens or check
token state.

#### P1: High Priority (3-6 months)

**3. Auto-Generate SDKs from OpenAPI Spec**

1. Write comprehensive OpenAPI 3.1 spec for all REST APIs
2. Use `openapi-generator` to produce SDKs for 8+ languages
3. Publish to: npm, PyPI, Maven, Go module, crates.io, NuGet, Composer, pub.dev
4. Automate in CI/CD — regenerate on API changes

**4. Build `ggid` CLI**

1. Use `cobra` Go library for CLI framework
2. Implement commands: tenant, user, client, policy, dev, config
3. Support JSON output for scripting
4. Ship as single binary (Go) via Homebrew, Scoop, apt, yum

**5. Production Helm Charts**

1. Create Helm chart per service with configurable values
2. Add Kubernetes health checks, readiness probes, HPA
3. Create umbrella chart for full-stack deployment
4. Add Grafana dashboards (GGID already has `deploy/grafana/`)
5. Document deployment scenarios (single-node, HA, multi-region)

**6. Permission Caching (Redis)**

Add Redis-backed caching to the policy evaluator:
```go
func (e *Evaluator) Check(ctx context.Context, req *domain.CheckRequest) (*domain.CheckResult, error) {
    cacheKey := fmt.Sprintf("perm:%s:%s:%s", req.UserID, req.Action, req.Resource)
    if cached, err := redis.Get(ctx, cacheKey); err == nil {
        return cached, nil
    }
    // ... evaluate from DB ...
    redis.Set(ctx, cacheKey, result, 60*time.Second)
    return result, nil
}
```

**7. Rate Limit Redis Backend**

The current rate limiter (`ratelimit.go`) is in-memory. For multi-instance
deployments, add Redis-backed rate limiting:
```go
// Use redis_rate or tollbooth-redis for distributed rate limiting
```

**8. Professional Security Audit**

Commission a security audit from a reputable firm (e.g., Trail of Bits, Cure53,
NCC Group). Budget: $30K-$80K. Publish results transparently.

#### P2: Medium Priority (6-12 months)

**9. Identity Schema (JSON Schema traits)**

Add optional custom attributes to the User entity:
- Add `metadata jsonb` column to users table
- Add JSON Schema validation API
- Support per-tenant schema definitions
- Auto-generate form fields from schema (for Admin Console)

**10. Flow-Based Self-Service APIs**

Add optional flow-based APIs for registration, recovery, and settings:
- Create `flows` package with flow state management
- Support multi-step flows with configurable steps
- Add webhook dispatch at each step
- Maintain backward compatibility with existing simple APIs

**11. Zanzibar-Style Authorization (Optional Mode)**

Add relation tuple support as an optional authorization mode:
- Add `relation_tuples` table
- Add in-memory graph cache
- Extend `Evaluator.Check()` to query both engines
- Configure per-tenant which engine(s) to use

**12. Consent Flow Enhancement**

Improve consent handling in the OAuth service:
- Configurable consent UI
- Per-scope consent tracking
- Remember consent (skip on subsequent requests)
- Consent revocation API
- Webhook on consent events

**13. Account Recovery Codes**

Generate and track backup recovery codes:
- Generate 10 single-use codes on enrollment
- Display once with hash storage
- Track usage and alert on use
- Allow regeneration

**14. Identity Webhooks**

Add webhook dispatch on identity events:
- Registration, login, profile update, password change, MFA enrollment
- Configurable per tenant
- Retry with exponential backoff
- HMAC signature for verification

#### P3: Lower Priority (12+ months)

**15. YAML-Based Gateway Configuration**

Add declarative YAML configuration for gateway routes:
- Hot-reloadable route configuration
- GitOps-friendly
- Per-route authentication pipeline
- Migration tool from Go config to YAML

**16. Dynamic Client Registration (RFC 7591/7592)**

Add dynamic OAuth client registration endpoints:
- `POST /oauth/register` — create client dynamically
- `GET/PUT/DELETE /oauth/register/{client_id}` — manage registered clients
- Token-based authentication for management

### 11.3 GGID's Sustainable Competitive Advantages

Despite Ory's maturity and market position, GGID has several advantages that are
**architecturally difficult for Ory to replicate**:

1. **Built-in Multi-Tenancy (RLS)** — Ory's OSS is fundamentally single-tenant.
   Adding true multi-tenancy requires re-architecting the data model across all
   four services.

2. **Event-Driven Audit Trail (NATS)** — Ory has no built-in audit system.
   Adding one requires either bolting on NATS/Kafka or building a new service.

3. **Integrated Suite (monorepo)** — Ory's polyrepo architecture is intentional
   (flexibility), but it means integration is always the user's problem.

4. **Dedicated B2B/Org Service** — Ory Kratos added "organizations" in 2024, but
   only in Enterprise/Cloud. GGID's `org` service is free and open-source.

5. **gRPC + REST Dual Protocol** — Ory is REST-only. Adding gRPC to all four
   services is a massive undertaking.

6. **CIBA Support** — GGID has CIBA (`ciba.go`), Hydra does not fully support it.

7. **Automated Key Rotation** — GGID has `RotatingKeyProvider` with grace period.
   Hydra requires manual key rotation via admin API.

8. **WASM Plugin Support** — GGID gateway has WASM plugin support
   (`wasm_plugin.go`). Oathkeeper does not.

### 11.4 Ory's Sustainable Competitive Advantages

Conversely, Ory has advantages that are difficult for GGID to replicate quickly:

1. **OIDC Certification** — Requires passing conformance tests and paying
   certification fees. GGID can achieve this but it takes time.

2. **50K+ GitHub Stars & 500+ Contributors** — Community and ecosystem effects.
   GGID is newer and must build this over time.

3. **Managed SaaS (Ory Network)** — Multi-region, auto-scaling infrastructure
   with billing, SLA, compliance. GGID would need 15-20 engineer-months for MVP.

4. **Production Users (Tesla, GitHub, Unity, Zalando)** — Social proof and
   battle-testing at scale.

5. **JSON Schema Identity Model** — Runtime-configurable identity schema. GGID's
   Go struct model requires code changes for new fields.

6. **Keto's Zanzibar Engine** — In-memory graph cache with sub-ms checks at
   million-tuple scale. GGID would need 2-3 months to add an equivalent.

7. **Oathkeeper Plugin System** — Extensible authenticator/authorizer/mutator
   pipeline. GGID's middleware chain is less extensible.

8. **Professional Security Audits** — Published audit reports from reputable
   firms. GGID has community-driven security review.

---

## Appendix A: Feature Matrix

### Authentication Methods

| Method | Ory Kratos | GGID |
|--------|-----------|------|
| Password | Argon2id | Argon2id |
| TOTP MFA | Yes | Yes |
| WebAuthn/Passkey | Yes | Yes |
| SMS MFA | Via Twilio | No |
| Email OTP | Yes | No |
| Social Login | OIDC config | 9 connectors |
| LDAP | Via external IdP | Built-in |
| SAML | Via Hydra | Built-in |
| Magic Link | No | No |
| Passwordless | Via WebAuthn | No |

### OAuth/OIDC

| Feature | Ory Hydra | GGID |
|---------|-----------|------|
| Authorization Code | Yes | Yes |
| PKCE | Yes | Yes |
| Client Credentials | Yes | Yes |
| Refresh Token | Yes | Yes |
| ROPC | Yes | Yes |
| Device Auth (RFC 8628) | Yes | No |
| Token Exchange (RFC 8693) | Yes | No |
| CIBA | Planned | Yes |
| DPoP | Yes | Yes |
| PAR | Yes | Yes |
| JAR | Yes | Yes |
| OIDC Certified | Yes | No |
| Token Introspection | Yes | No |
| Dynamic Client Reg | Yes | No |
| Key Rotation | Manual | Automated |

### Authorization

| Feature | Ory Keto | GGID |
|---------|----------|------|
| Model | Zanzibar tuples | RBAC + ABAC |
| RBAC | Via tuples | Native |
| ABAC | No | Native |
| Deny rules | No | Native |
| In-memory cache | Yes | No |
| gRPC API | Yes | Yes |
| REST API | Yes | Yes |
| Multi-tenant | Enterprise only | Free (RLS) |

### Gateway/Proxy

| Feature | Ory Oathkeeper | GGID |
|---------|---------------|------|
| Plugin system | Yes | Middleware |
| YAML config | Yes | Go code |
| Rate limiting | No | Yes |
| Circuit breaker | No | Yes |
| HTTP/3 | No | Yes |
| WebSocket proxy | No | Yes |
| WASM plugins | No | Yes |
| Tenant routing | No | Yes |
| Health checks | No | Yes |
| Metrics | No | Yes |

### Multi-Tenancy

| Feature | Ory | GGID |
|---------|-----|------|
| OSS multi-tenancy | No | Yes (RLS) |
| Enterprise multi-tenancy | Yes (paid) | N/A (free) |
| Per-tenant config | Enterprise | Planned |
| Per-tenant IdP | Enterprise | Planned |
| Per-tenant branding | Enterprise | Yes (Console) |

### Developer Experience

| Feature | Ory | GGID |
|---------|-----|------|
| CLI | Yes (`ory`) | No |
| Managed SaaS | Yes (Ory Network) | No |
| SDK languages | 7+ | 4 |
| SDK generation | Auto (OpenAPI) | Hand-written |
| Admin Console | Yes | Yes (20+ pages) |
| Quickstart | 5 min (Network) | 5 min (Docker) |
| Documentation | Extensive | Growing |
| Community | 50K+ stars | Growing |

---

## Appendix B: API Endpoint Comparison

### Identity APIs

| Operation | Ory Kratos | GGID |
|-----------|-----------|------|
| Create identity | `POST /admin/identities` | `POST /api/v1/users` |
| Get identity | `GET /admin/identities/{id}` | `GET /api/v1/users/{id}` |
| List identities | `GET /admin/identities` | `GET /api/v1/users` |
| Update identity | `PUT /admin/identities/{id}` | `PUT /api/v1/users/{id}` |
| Delete identity | `DELETE /admin/identities/{id}` | `DELETE /api/v1/users/{id}` |
| Registration | `POST /self-service/registration` | `POST /api/v1/auth/register` |
| Login | `POST /self-service/login` | `POST /api/v1/auth/login` |
| Recovery | `POST /self-service/recovery` | `POST /api/v1/auth/password/forgot` |
| Verification | `POST /self-service/verification` | API-based |
| Settings | `POST /self-service/settings` | `PUT /api/v1/users/{id}` |
| Schema | `GET /schemas/{id}` | N/A (compiled) |

### OAuth APIs

| Operation | Ory Hydra | GGID |
|-----------|-----------|------|
| Authorize | `GET /oauth2/auth` | `GET /oauth/authorize` |
| Token | `POST /oauth2/token` | `POST /oauth/token` |
| Introspect | `POST /admin/oauth2/introspect` | Not implemented |
| Revoke | `POST /oauth2/revoke` | `POST /oauth/revoke` |
| JWKS | `GET /.well-known/jwks.json` | `GET /.well-known/jwks.json` |
| Discovery | `GET /.well-known/openid-configuration` | `GET /.well-known/openid-configuration` |
| Client create | `POST /admin/clients` | `POST /api/v1/oauth/clients` |
| Client list | `GET /admin/clients` | `GET /api/v1/oauth/clients` |
| Consent accept | `PUT /admin/oauth2/auth/requests/consent/accept` | Integrated |
| Login accept | `PUT /admin/oauth2/auth/requests/login/accept` | Integrated |

### Authorization APIs

| Operation | Ory Keto | GGID |
|-----------|----------|------|
| Check | `POST /relation-tuples/check` | `POST /api/v1/permissions/check` |
| Write tuple | `PUT /admin/relation-tuples` | N/A (RBAC model) |
| Expand | `GET /relation-tuples/expand` | N/A |
| Create role | N/A | `POST /api/v1/roles` |
| Assign role | N/A | `POST /api/v1/roles/{id}/users` |
| Create policy | N/A | `POST /api/v1/policies` |

---

## Appendix C: Data Model Comparison

### User/Identity Model

| Field | Ory Kratos | GGID |
|-------|-----------|------|
| ID | UUID | UUID |
| Schema | JSON Schema | Go struct |
| Email | trait (schema-defined) | `Email string` |
| Phone | trait (schema-defined) | `Phone string` |
| Username | trait (schema-defined) | `Username string` |
| Password hash | credential (separate) | `PasswordHash string` |
| Status | `state` (active/inactive) | `UserStatus` (active/locked/disabled/deleted) |
| Tenant | N/A (single-tenant) | `TenantID uuid.UUID` |
| External ID | N/A | `ExternalID string` (SCIM) |
| Created at | `created_at` | `CreatedAt time.Time` |
| Updated at | `updated_at` | `UpdatedAt time.Time` |
| Deleted at | N/A | `DeletedAt *time.Time` (soft delete) |
| Last login | `traits` or session | `LastLoginAt *time.Time` |
| Display name | trait | `DisplayName string` |
| Avatar | trait | `AvatarURL string` |
| Locale | trait | `Locale string` |
| Timezone | trait | `Timezone string` |
| Custom fields | Via traits (JSON Schema) | Not supported |

### OAuth Client Model

| Field | Ory Hydra | GGID |
|-------|-----------|------|
| Client ID | `client_id` | `ClientID string` |
| Client name | `client_name` | `Name string` |
| Client type | confidential/public | `ClientType` |
| Secret hash | BCrypt | Argon2id |
| Grant types | `grant_types` | `GrantTypes []string` |
| Response types | `response_types` | `ResponseTypes []string` |
| Redirect URIs | `redirect_uris` | `RedirectURIs []string` |
| Scopes | `scope` | `Scopes []string` |
| PKCE required | Configurable | `RequirePKCE bool` |
| Tenant | N/A (single-tenant) | `TenantID uuid.UUID` |
| Metadata | `metadata` | `Metadata map[string]any` |
| Token auth method | `token_endpoint_auth_method` | `TokenEndpointAuthMethod string` |
| Enabled | `active` | `Enabled bool` |

### Authorization Model

| Concept | Ory Keto | GGID |
|---------|----------|------|
| Primary unit | Relation tuple | Policy rule |
| Tuple format | `object#relation@subject` | `{effect, action, resource, conditions}` |
| Subject | user or group#relation | `UserID` or `RoleID` |
| Object | `namespace:object_id` | `Resource string` |
| Relationship | `relation` | `Action string` |
| Inheritance | Transitive tuples | Role ancestor chain |
| Deny support | Not native | `EffectDeny` with priority |
| Conditions | Not supported | JSON/YAML conditions (ABAC) |
| Tenant | N/A (single-tenant) | `TenantID` in every entity |

---

## Conclusion

Ory and GGID represent two legitimate but fundamentally different approaches to
building an IAM platform:

- **Ory** is the **mature, flexible, composable** choice — ideal for teams that
  want maximum flexibility, already have parts of their identity stack, or need
  OIDC certification. Its managed SaaS (Ory Network) provides zero-infrastructure
  onboarding. Its main weaknesses are operational complexity (4 services, 4
  databases), lack of open-source multi-tenancy, and no built-in audit trail.

- **GGID** is the **integrated, opinionated, multi-tenant-first** choice — ideal
  for teams building multi-tenant SaaS platforms, B2B applications, or anyone who
  wants built-in multi-tenancy, audit trails, and B2B organization management
  without enterprise licensing. Its main gaps are OIDC certification, SDK language
  coverage, CLI tooling, and managed SaaS offering.

The most impactful improvements GGID can make to close the gap with Ory are:
1. **OIDC Certification** (unlocks enterprise/regulated industries)
2. **Auto-generated SDKs** (closes language coverage gap)
3. **CLI tooling** (improves developer experience)
4. **Permission caching** (closes performance gap with Keto)
5. **Security audit** (builds enterprise trust)

Conversely, Ory's most difficult-to-replicate advantages for GGID to overcome are
community scale (50K+ stars, 500+ contributors), managed SaaS infrastructure, and
JSON Schema-based identity model.

**The IAM market is large enough for both approaches to coexist.** GGID's
multi-tenant-first architecture gives it a unique position that neither Ory OSS
(single-tenant) nor Auth0 (proprietary, expensive) can match. By closing the OIDC
certification and SDK gaps, GGID can position itself as the best open-source IAM
for multi-tenant SaaS applications.

---

*End of document — 1200+ lines of deep competitive analysis.*
