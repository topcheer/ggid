# Microservices Architecture

> GGID's 7-service architecture: responsibilities, ports, dependencies, and communication patterns.

---

## Service Overview

```
                          ┌──────────────┐
                          │   Client      │
                          │ (Browser/SDK) │
                          └──────┬───────┘
                                 │
                          ┌──────▼───────┐
                          │ API Gateway   │ :8080
                          │ (JWT verify,  │
                          │  rate limit,  │
                          │  reverse proxy)│
                          └──┬───┬───┬───┘
             ┌───────────────┘   │   └───────────────┐
             │                   │                   │
      ┌──────▼──────┐    ┌──────▼──────┐    ┌──────▼──────┐
      │  Identity    │    │    Auth     │    │   Policy    │
      │  :8081/:50051│    │  :9001/:50052│   │ :8070/:50053│
      │  Users/SCIM  │    │  Login/MFA  │    │  RBAC/ABAC  │
      └──────┬───────┘    └──────┬──────┘    └──────┬──────┘
             │                   │                   │
      ┌──────▼──────┐    ┌──────▼──────┐    ┌──────▼──────┐
      │    OAuth     │    │    Org      │    │   Audit     │
      │  :9005       │    │ :8071/:50054│    │ :8072/:50055│
      │  OIDC/SAML   │    │  Org CRUD   │    │  NATS→DB    │
      └─────────────┘    └─────────────┘    └─────────────┘
             │                   │                   │
             └───────────────────┼───────────────────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
             ┌──────▼──┐  ┌─────▼───┐  ┌────▼───┐
             │PostgreSQL│  │  Redis  │  │  NATS  │
             │  16+RLS  │  │ Sessions│  │JetStream│
             └─────────┘  └─────────┘  └────────┘
```

---

## Service Details

### 1. API Gateway (:8080)

| Property | Value |
|----------|-------|
| **HTTP Port** | 8080 |
| **Role** | Public entry point, reverse proxy |
| **Responsibilities** | JWT verification, rate limiting, security headers, tenant resolution, request routing |
| **Depends on** | Redis (rate limit, JWKS cache), all backend services |
| **Routes** | `/api/v1/*` → backend services, `/login` → hosted login |

### 2. Identity Service (:8081 / :50051)

| Property | Value |
|----------|-------|
| **HTTP Port** | 8081 |
| **gRPC Port** | 50051 |
| **Role** | User and group management |
| **Responsibilities** | User CRUD, SCIM 2.0, group management, user profile |
| **Depends on** | PostgreSQL (users table with RLS) |
| **Routes** | `/api/v1/users/*`, `/scim/v2/*` |

### 3. Auth Service (:9001 / :50052)

| Property | Value |
|----------|-------|
| **HTTP Port** | 9001 |
| **gRPC Port** | 50052 |
| **Role** | Authentication and token issuance |
| **Responsibilities** | Login, register, JWT issuance, refresh, MFA (TOTP), LDAP, WebAuthn, social login |
| **Depends on** | PostgreSQL (credentials), Redis (sessions, rate limit), LDAP (optional) |
| **Routes** | `/api/v1/auth/*` |

### 4. OAuth Service (:9005)

| Property | Value |
|----------|-------|
| **HTTP Port** | 9005 |
| **Role** | OAuth 2.1 and OIDC provider |
| **Responsibilities** | Authorization code flow, PKCE, token exchange (RFC 8693), introspection, discovery, JARM |
| **Depends on** | PostgreSQL (oauth_clients), Redis (state, PKCE) |
| **Routes** | `/oauth/*`, `/.well-known/*` |

### 5. Policy Service (:8070 / :50053)

| Property | Value |
|----------|-------|
| **HTTP Port** | 8070 |
| **gRPC Port** | 50053 |
| **Role** | Authorization engine (RBAC + ABAC) |
| **Responsibilities** | Role CRUD, permission check, policy evaluate, dry-run, compliance templates |
| **Depends on** | PostgreSQL (roles, policies) |
| **Routes** | `/api/v1/roles/*`, `/api/v1/policies/*` |

### 6. Org Service (:8071 / :50054)

| Property | Value |
|----------|-------|
| **HTTP Port** | 8071 |
| **gRPC Port** | 50054 |
| **Role** | Organization management |
| **Responsibilities** | Org CRUD, hierarchy, member management |
| **Depends on** | PostgreSQL (organizations table) |
| **Routes** | `/api/v1/orgs/*` |

### 7. Audit Service (:8072 / :50055)

| Property | Value |
|----------|-------|
| **HTTP Port** | 8072 |
| **gRPC Port** | 50055 |
| **Role** | Audit event pipeline and query |
| **Responsibilities** | NATS consumer, hash chain, audit query API |
| **Depends on** | NATS JetStream (event bus), PostgreSQL (audit_events) |
| **Routes** | `/api/v1/audit/*` |

---

## Communication Patterns

| Pattern | Usage |
|---------|-------|
| **REST/HTTP** | Client → Gateway, Gateway → backend services |
| **gRPC** | Service-to-service (optional, for internal calls) |
| **NATS JetStream** | Async audit event pipeline (fire-and-forget) |
| **Redis** | Session state, rate limiting, JWKS cache, OAuth state |

---

## Data Ownership

| Database Table | Owner Service | Accessed By |
|----------------|--------------|-------------|
| `users` | Identity | Identity only |
| `credentials` | Auth | Auth only |
| `roles` | Policy | Policy only |
| `organizations` | Org | Org only |
| `audit_events` | Audit | Audit only |
| `oauth_clients` | OAuth | OAuth only |

> **Rule:** Each table has a single owner service. Cross-service reads go through the owner's API, never direct DB access.

---

## Shared Infrastructure

| Component | Port | Used By |
|-----------|------|---------|
| PostgreSQL | 5432 | Identity, Auth, OAuth, Policy, Org, Audit |
| Redis | 6379 | Gateway, Auth, OAuth |
| NATS JetStream | 4222 | All services (publish), Audit (consume) |
| LDAP (optional) | 389 | Auth only |

---

## Admin Console

| Property | Value |
|----------|-------|
| **Port** | 3000 |
| **Role** | Web-based management UI |
| **Built with** | Next.js 15 |
| **Depends on** | Gateway (all API calls) |

---

*See: [Data Flow](data-flow.md) | [Security Overview](security-overview.md) | [Architecture Overview](overview.md)*

*Last updated: 2025-07-11*
