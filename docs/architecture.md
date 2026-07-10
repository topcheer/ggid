# GGID Architecture

> C4 Model — System Context, Container, Component, and Deployment views.
> Built with [Mermaid](https://mermaid.js.org/) diagrams for GitHub/GitLab rendering.

---

## Overview

GGID is an Apache 2.0 open-source Identity and Access Management (IAM) platform
built as a Go microservices monorepo. It provides authentication, authorization,
policy enforcement, organization management, and audit logging for multi-tenant
SaaS applications.

**Key characteristics:**

- **7 microservices** — independently deployable, communicating via gRPC and REST
- **Multi-tenant** — PostgreSQL Row-Level Security (RLS) isolates tenant data
- **Event-driven** — NATS JetStream for async audit event streaming
- **Polyglot persistence** — PostgreSQL (primary), Redis (cache/sessions), LDAP (directory)

---

## 1. System Context (C4 Level 1)

The System Context view shows GGID as a single box and its relationships with
external actors and systems.

```mermaid
graph TB
    subgraph Actors
        EndUser[End Users<br/>Browser / Mobile / CLI]
        Admin[Tenant Administrators<br/>Admin Console]
        DevApp[Third-Party Apps<br/>OAuth / OIDC Client]
    end

    subgraph External
        IdP[External IdPs<br/>Google, GitHub, Microsoft, SAML]
        LDAP[LDAP / Active Directory]
        SMTP[SMTP Server<br/>Email Delivery]
    end

    GGID[GGID IAM Platform<br/>7 microservices + Admin Console]

    EndUser -->|HTTPS / gRPC| GGID
    Admin -->|HTTPS| GGID
    DevApp -->|OAuth 2.0 / OIDC| GGID

    GGID -->|SAML / OIDC| IdP
    GGID -->|LDAP bind| LDAP
    GGID -->|SMTP| SMTP

    style GGID fill:#4a90d9,color:#fff,stroke:#2c5f8a
```

### Actors

| Actor | Description | Protocol |
|-------|-------------|----------|
| End Users | authenticate, manage profile, access resources | HTTPS, gRPC |
| Tenant Admins | manage users, roles, orgs, policies via Console | HTTPS |
| Third-Party Apps | integrate via OAuth 2.0 / OIDC / SAML | HTTPS |

### External Dependencies

| System | Purpose |
|--------|---------|
| External IdPs | Social login (Google, GitHub, Microsoft, Apple, etc.) |
| LDAP / AD | Enterprise directory integration |
| SMTP Server | Transactional email (verification, reset, notifications) |

---

## 2. Container View (C4 Level 2)

The Container view breaks the system into its major deployable units.

```mermaid
graph LR
    subgraph Client
        Console[Admin Console<br/>Next.js 15<br/>:3000]
    end

    subgraph Gateway
        GW[API Gateway<br/>Go<br/>:8080]
    end

    subgraph Core Services
        Auth[Auth Service<br/>Go<br/>:9001]
        Identity[Identity Service<br/>Go + gRPC<br/>:8081 / :50051]
        OAuth[OAuth Service<br/>Go<br/>:9005]
        Policy[Policy Service<br/>Go + gRPC<br/>:8070 / :9070]
        Org[Org Service<br/>Go + gRPC<br/>:8071 / :9071]
        Audit[Audit Service<br/>Go + gRPC<br/>:8072 / :9072]
    end

    subgraph Infrastructure
        PG[(PostgreSQL 16<br/>:5432)]
        Redis[(Redis 7<br/>:6379)]
        NATS[(NATS JetStream<br/>:4222)]
    end

    Console -->|REST| GW
    GW -->|REST / gRPC| Auth
    GW -->|REST / gRPC| Identity
    GW -->|REST / gRPC| OAuth
    GW -->|REST / gRPC| Policy
    GW -->|REST / gRPC| Org
    GW -->|REST / gRPC| Audit

    Auth --> PG
    Auth --> Redis
    Identity --> PG
    OAuth --> PG
    Policy --> PG
    Org --> PG
    Audit --> PG

    Auth -.->|publish events| NATS
    GW -.->|publish events| NATS
    NATS -.->|consume| Audit

    style GW fill:#e74c3c,color:#fff
    style PG fill:#336791,color:#fff
    style Redis fill:#dc382d,color:#fff
    style NATS fill:#34a8c4,color:#fff
```

### Container Summary

| Container | Language | Port(s) | Role |
|-----------|----------|---------|------|
| Admin Console | TypeScript (Next.js 15) | 3000 | Web UI for tenant admins |
| API Gateway | Go | 8080 | JWT verification, routing, rate limiting |
| Auth Service | Go | 9001 | Register, login, JWT, MFA, refresh |
| Identity Service | Go | 8081, 50051 | User profiles, groups, SCIM 2.0 |
| OAuth Service | Go | 9005 | OAuth 2.0, OIDC, social login |
| Policy Service | Go | 8070, 9070 | RBAC + ABAC policy engine |
| Org Service | Go | 8071, 9071 | Organization tree, memberships |
| Audit Service | Go | 8072, 9072 | Event persistence, SSE streaming |
| PostgreSQL | — | 5432 | Primary data store with RLS |
| Redis | — | 6379 | Session cache, rate limiting, reset tokens |
| NATS JetStream | — | 4222 | Async audit event stream |

---

## 3. Component View (C4 Level 3)

The Component view zooms into the API Gateway and Auth Service to show their
internal structure.

### API Gateway Components

```mermaid
graph TB
    subgraph "API Gateway"
        Router[Router<br/>gorilla/mux]
        JWT[JWT Middleware<br/>Verify + extract claims]
        Tenant[Tenant Middleware<br/>Extract tenant_id]
        RateLimit[Rate Limiter<br/>Token bucket per IP]
        Proxy[Reverse Proxy<br/>httputil.ReverseProxy]
        AuditMW[Audit Middleware<br/>Publish to NATS]
        CORS[CORS Middleware]
    end

    Client[Client Request] --> CORS
    CORS --> RateLimit
    RateLimit --> Router
    Router --> JWT
    JWT --> Tenant
    Tenant --> AuditMW
    AuditMW --> Proxy
    Proxy -->|forward| Backend[Backend Service]

    style Router fill:#e74c3c,color:#fff
    style Proxy fill:#8e44ad,color:#fff
```

### Auth Service Components

```mermaid
graph TB
    subgraph "Auth Service"
        Handler[HTTP Handler<br/>REST endpoints]
        AuthService[Auth Service<br/>Login, Register, Refresh]
        PwdService[Password Service<br/>Hash, Verify, History]
        TokenService[Token Service<br/>JWT issue, refresh rotation]
        SessionService[Session Service<br/>Create, revoke, list]
        MFA[MFA Provider<br/>TOTP, WebAuthn]
        AuthProvider[Auth Provider Chain<br/>Local + LDAP]
        Hooks[Hook Manager<br/>Pre/Post webhooks]
    end

    Handler --> AuthService
    AuthService --> PwdService
    AuthService --> TokenService
    AuthService --> SessionService
    AuthService --> MFA
    AuthService --> AuthProvider
    AuthService --> Hooks

    PwdService --> CredRepo[(Credential Repo)]
    TokenService --> JWKS[(JWKS Keys)]
    SessionService --> Redis[(Redis)]
    AuthProvider --> LDAP[LDAP Server]

    style AuthService fill:#e74c3c,color:#fff
    style TokenService fill:#27ae60,color:#fff
```

---

## 4. Deployment View (C4 Level 4)

The Deployment view shows how containers are deployed in production.

### Docker Compose (Development)

```mermaid
graph TB
    subgraph "Docker Host"
        subgraph "Network: ggid-net"
            GW_C[Gateway :8080]
            Auth_C[Auth :9001]
            Identity_C[Identity :8081]
            OAuth_C[OAuth :9005]
            Policy_C[Policy :8070]
            Org_C[Org :8071]
            Audit_C[Audit :8072]
            Console_C[Console :3000]
            PG_C[(PostgreSQL :5432)]
            Redis_C[(Redis :6379)]
            NATS_C[(NATS :4222)]
        end
    end

    Internet[Internet] -->|:8080| GW_C
    Internet -->|:3000| Console_C

    GW_C --> Auth_C & Identity_C & OAuth_C & Policy_C & Org_C & Audit_C
    Auth_C --> PG_C & Redis_C
    Audit_C --> NATS_C

    style GW_C fill:#e74c3c,color:#fff
    style PG_C fill:#336791,color:#fff
```

### Kubernetes (Production)

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        subgraph "Ingress"
            ING[Nginx Ingress<br/>TLS termination]
        end

        subgraph "Frontend Namespace"
            CONSOLE_D[Console Deployment<br/>3 replicas]
        end

        subgraph "API Namespace"
            GW_D[Gateway Deployment<br/>3 replicas + HPA]
            AUTH_D[Auth Deployment<br/>3 replicas + HPA]
            IDENT_D[Identity Deployment<br/>2 replicas]
            OAUTH_D[OAuth Deployment<br/>2 replicas]
            POLICY_D[Policy Deployment<br/>2 replicas]
            ORG_D[Org Deployment<br/>2 replicas]
            AUDIT_D[Audit Deployment<br/>2 replicas]
        end

        subgraph "Data Namespace"
            PG_SS[(PostgreSQL StatefulSet<br/>Primary + Replica)]
            REDIS_SS[(Redis StatefulSet<br/>3-node cluster)]
            NATS_SS[(NATS StatefulSet<br/>3-node cluster)]
        end
    end

    Internet -->|HTTPS| ING
    ING --> GW_D
    ING --> CONSOLE_D
    GW_D --> AUTH_D & IDENT_D & OAUTH_D & POLICY_D & ORG_D & AUDIT_D

    style ING fill:#8e44ad,color:#fff
    style PG_SS fill:#336791,color:#fff
```

---

## 5. Data Flow: Authentication Request

```mermaid
sequenceDiagram
    participant U as User
    participant GW as API Gateway
    participant AU as Auth Service
    participant PG as PostgreSQL
    participant RD as Redis
    participant NT as NATS
    participant AU2 as Audit Service

    U->>GW: POST /api/v1/auth/login
    GW->>GW: Rate limit check
    GW->>AU: Forward (username, password)
    AU->>PG: Query credential by identifier
    PG-->>AU: Credential record
    AU->>AU: Verify bcrypt hash
    AU->>RD: Check rate limit / lockout
    AU->>AU: Issue JWT (access + refresh)
    AU->>RD: Store session
    AU->>NT: Publish audit event (user.login)
    AU-->>GW: 200 + JWT tokens
    GW-->>U: 200 + JWT tokens
    NT-->>AU2: Consume audit event
    AU2->>PG: Persist audit log
```

---

## 6. Data Flow: Policy Check

```mermaid
sequenceDiagram
    participant U as User
    participant GW as Gateway
    participant PO as Policy Service
    participant PG as PostgreSQL

    U->>GW: GET /api/v1/users (with JWT)
    GW->>GW: Verify JWT signature
    GW->>GW: Extract tenant_id, user_id
    GW->>PO: Check(subject=user_id, action=read, resource=users)
    PO->>PG: Load policies for tenant
    PG-->>PO: Active policies
    PO->>PO: Evaluate RBAC + ABAC rules
    PO-->>GW: Allow / Deny
    alt Allow
        GW->>GW: Forward to Identity Service
        GW-->>U: 200 + user list
    else Deny
        GW-->>U: 403 Forbidden
    end
```

---

## 7. Multi-Tenant Data Isolation

GGID uses PostgreSQL Row-Level Security (RLS) to enforce tenant isolation at
the database level. Every table includes a `tenant_id` column, and RLS policies
ensure queries only return rows matching the current tenant context.

```mermaid
graph LR
    subgraph "PostgreSQL RLS"
        T1[Tenant A Rows<br/>tenant_id = aaa-111]
        T2[Tenant B Rows<br/>tenant_id = bbb-222]
        T3[Tenant C Rows<br/>tenant_id = ccc-333]
    end

    AppA[Service with<br/>tenant_id = aaa-111] -->|SET LOCAL| T1
    AppB[Service with<br/>tenant_id = bbb-222] -->|SET LOCAL| T2

    style T1 fill:#27ae60,color:#fff
    style T2 fill:#e67e22,color:#fff
    style T3 fill:#95a5a6,color:#fff
```

### RLS Policy Example

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: users can only see their own tenant's data
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Set tenant context per transaction
SET LOCAL app.tenant_id = '00000000-0000-0000-0000-000000000001';
```

---

## 8. Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Language | Go | 1.25 |
| Frontend | Next.js + React + TypeScript | 15.x |
| Database | PostgreSQL | 16 |
| Cache | Redis | 7 |
| Message Queue | NATS JetStream | 2.10+ |
| Directory | OpenLDAP | 2.6 |
| Protocol | gRPC + REST + SSE | — |
| Serialization | Protocol Buffers + JSON | — |
| Auth | JWT (RS256) + OIDC + SAML 2.0 | — |

---

## 9. Service Communication Matrix

| From \ To | Gateway | Auth | Identity | OAuth | Policy | Org | Audit |
|-----------|---------|------|----------|-------|--------|-----|-------|
| **Gateway** | — | REST | REST/gRPC | REST | REST/gRPC | REST/gRPC | REST/gRPC |
| **Auth** | — | — | — | — | — | — | NATS pub |
| **Identity** | — | — | — | — | gRPC | — | NATS pub |
| **OAuth** | — | gRPC | — | — | — | — | NATS pub |
| **Policy** | — | — | — | — | — | — | NATS pub |
| **Org** | — | — | — | — | — | — | NATS pub |
| **Audit** | — | — | — | — | — | — | — |

- **REST**: Synchronous HTTP JSON
- **gRPC**: Synchronous Protocol Buffers
- **NATS pub**: Async fire-and-forget event publish

---

## 10. Cross-Cutting Concerns

### Observability

```mermaid
graph LR
    subgraph Services
        S1[Gateway]
        S2[Auth]
        S3[Identity]
        S4[Policy]
    end

    subgraph Observability Stack
        PROM[Prometheus<br/>Metrics scraping]
        LOKI[Loki / ELK<br/>Log aggregation]
        JAEGER[Jaeger / Tempo<br/>Distributed tracing]
        GRAF[Grafana<br/>Dashboards]
    end

    S1 & S2 & S3 & S4 -->|/metrics| PROM
    S1 & S2 & S3 & S4 -->|structured logs| LOKI
    S1 & S2 & S3 & S4 -->|OpenTelemetry| JAEGER
    PROM & LOKI & JAEGER --> GRAF
```

### Security Layers

1. **Edge**: TLS 1.3, HSTS, CSP headers (Ingress / Gateway)
2. **Authentication**: JWT RS256 verification at Gateway
3. **Authorization**: RBAC + ABAC policy check at Policy Service
4. **Tenant Isolation**: PostgreSQL RLS
5. **Audit**: NATS JetStream → Audit Service → append-only table
6. **Secrets**: Vault / Sealed Secrets / env vars (12-factor)

---

## References

- [C4 Model](https://c4model.com/) — Simon Brown's architecture visualisation framework
- [GGID Quick Start](./quick-start.md) — Get started in 5 minutes
- [Deployment Guide](./deployment.md) — Production deployment instructions
- [Security Whitepaper](./security-whitepaper.md) — Threat model and security controls
- [Data Model Design](./design/data-model.md) — Entity relationships and schema
