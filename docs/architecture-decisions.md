# Architecture Decision Records (ADRs)

> This document captures the key architectural decisions that shaped the GGID IAM platform. Each ADR follows the Context / Decision / Consequences format.

---

## Table of Contents

1. [ADR-001: Microservices Architecture](#adr-001-microservices-architecture)
2. [ADR-002: Dual gRPC + REST API](#adr-002-dual-grpc--rest-api)
3. [ADR-003: NATS JetStream for Event Bus](#adr-003-nats-jetstream-for-event-bus)
4. [ADR-004: PostgreSQL Row-Level Security for Multi-Tenancy](#adr-004-postgresql-row-level-security-for-multi-tenancy)
5. [ADR-005: JWT-Based Stateless Authentication](#adr-005-jwt-based-stateless-authentication)
6. [ADR-006: Multi-Tenant Isolation Strategy](#adr-006-multi-tenant-isolation-strategy)
7. [ADR-007: Go as Primary Language](#adr-007-go-as-primary-language)
8. [ADR-008: Next.js for Admin Console](#adr-008-nextjs-for-admin-console)
9. [ADR-009: Redis for Session and Rate Limiting](#adr-009-redis-for-session-and-rate-limiting)
10. [ADR-010: OpenLDAP for Directory Services](#adr-010-openldap-for-directory-services)
11. [ADR-011: Docker Compose for Local, Kubernetes for Production](#adr-011-docker-compose-for-local-kubernetes-for-production)
12. [ADR-012: OAuth 2.1 + OIDC for Authorization](#adr-012-oauth-21--oidc-for-authorization)
13. [ADR-013: RBAC + ABAC Policy Engine](#adr-013-rbac--abac-policy-engine)
14. [ADR-014: Audit Trail via NATS JetStream](#adr-014-audit-trail-via-nats-jetstream)
15. [ADR-015: SCIM 2.0 for Automated Provisioning](#adr-015-scim-20-for-automated-provisioning)

---

## ADR-001: Microservices Architecture

**Status:** Accepted

### Context

The IAM platform must support multiple identity concerns: user management, authentication, authorization, audit logging, and directory integration. A monolith would couple these domains tightly, making independent scaling and deployment difficult. Authentication traffic spikes (login storms) should not affect audit ingestion throughput. Different teams need to own different domains independently.

### Decision

Adopt a microservices architecture with 7 independently deployable services:

| Service | Responsibility | Port(s) |
|---------|---------------|---------|
| **Gateway** | API gateway, JWT verification, rate limiting, routing | 8080 |
| **Identity** | User lifecycle, SCIM provisioning, directory sync | 8081 / 50051 |
| **Auth** | Login, register, JWT issuance, MFA, WebAuthn | 9001 |
| **OAuth** | OAuth 2.1, OIDC, token introspection, revocation | 9005 |
| **Policy** | RBAC/ABAC policy evaluation, role management | 8070 / 9070 |
| **Org** | Organization hierarchy, membership management | 8071 / 9071 |
| **Audit** | Event ingestion, query, NATS publishing | 8072 / 9072 |

### Consequences

**Positive:**
- Independent scaling per domain (auth can scale during login storms)
- Team ownership boundaries are clear
- Fault isolation: one service crash does not take down others
- Technology choice flexibility per service

**Negative:**
- Operational complexity: 7 services to monitor, deploy, and debug
- Network latency for inter-service calls
- Distributed transaction challenges (compensating transactions needed)
- Requires service discovery and health checking

---

## ADR-002: Dual gRPC + REST API

**Status:** Accepted

### Context

Internal service-to-service communication needs low latency, strong typing, and binary efficiency. External clients (browsers, mobile apps, third-party integrations) need HTTP/JSON REST APIs. Supporting both protocols for every service increases code complexity but maximizes compatibility.

### Decision

Each service exposes both:
- **gRPC** for internal inter-service communication (HTTP/2, Protocol Buffers)
- **REST** for external client consumption (HTTP/1.1 + JSON)

The Gateway translates between protocols as needed. Protocol Buffers definitions live in `proto/` and are the single source of truth.

### Consequences

**Positive:**
- Internal calls are fast and type-safe
- External clients get standard REST + JSON
- Protocol Buffers provide contract-first development
- Gateway handles cross-protocol routing

**Negative:**
- Dual code paths per endpoint (gRPC handler + REST handler)
- Protocol Buffer regeneration step in build
- Debugging gRPC streaming is harder than REST
- More test surface area (must test both protocols)

---

## ADR-003: NATS JetStream for Event Bus

**Status:** Accepted

### Context

The audit service must ingest high-volume events without blocking request paths. The system needs reliable message delivery with persistence guarantees. Alternatives considered: Kafka (heavy operational footprint), RabbitMQ (limited persistence), Redis Streams (simpler but less durable).

### Decision

Use **NATS JetStream** as the event bus for:
- Audit event ingestion (durable stream with at-least-once delivery)
- Webhook delivery pipeline
- Inter-service async notifications
- Monitoring port 8222 for health checks

### Consequences

**Positive:**
- Lightweight single-binary deployment
- Built-in persistence (no external store)
- Subjects-based routing (`audit.events.>`, `webhook.delivery.>`)
- Consumer groups for parallel processing
- Monitoring HTTP endpoint on port 8222

**Negative:**
- Smaller ecosystem than Kafka
- Limited transformation capabilities (no stream processing DSL)
- JetStream configuration learning curve

---

## ADR-004: PostgreSQL Row-Level Security for Multi-Tenancy

**Status:** Accepted

### Context

Multi-tenancy requires strict data isolation between tenants. Application-level WHERE clauses are error-prone — one missed clause leaks data across tenants. Database-level enforcement provides defense in depth. PostgreSQL 16 has mature RLS support.

### Decision

Use **PostgreSQL Row-Level Security (RLS)** policies on every tenant-scoped table:

```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

Every database connection sets `app.tenant_id` via `SET LOCAL` before queries. This ensures queries can never return rows from a different tenant, even with a bug in the application layer.

### Consequences

**Positive:**
- Defense in depth: tenant isolation enforced at DB layer
- Cannot leak cross-tenant data even with application bugs
- Per-connection tenant context is explicit
- Works with existing PostgreSQL expertise

**Negative:**
- Performance overhead from RLS policy evaluation
- Must set `app.tenant_id` on every connection (no forgetting)
- `SET LOCAL` does not support parameterized queries (`$1`) — use `fmt.Sprintf`
- Debugging requires understanding RLS policies

---

## ADR-005: JWT-Based Stateless Authentication

**Status:** Accepted

### Context

Stateless authentication via JWT allows horizontal scaling without shared session state. Tokens are self-contained, reducing database lookups per request. However, revocation requires additional infrastructure since tokens are valid until expiry.

### Decision

Use **JWT (JSON Web Tokens)** with:
- **Access tokens**: Short-lived (15 minutes), carry tenant_id, user_id, scopes
- **Refresh tokens**: Longer-lived (7 days), stored in Redis for revocation
- **JTI tracking**: Redis SETNX for anti-replay on sensitive operations
- **Signing**: HMAC-SHA256 with server-side secret

### Consequences

**Positive:**
- Stateless: no session DB lookup per request
- Horizontal scaling without sticky sessions
- Token carries all authorization context (tenant, scopes)
- Standard format — widely supported

**Negative:**
- Cannot revoke access token before expiry (must wait for expiration)
- Token size increases with claims payload
- Secret compromise invalidates all tokens
- Must implement refresh token rotation carefully

---

## ADR-006: Multi-Tenant Isolation Strategy

**Status:** Accepted

### Context

The platform serves multiple organizations (tenants). Tenant data must be strictly isolated. Three approaches were considered: separate databases per tenant, separate schemas per tenant, and shared database with RLS. Separate databases scale poorly for thousands of tenants.

### Decision

**Shared database with PostgreSQL RLS** as the primary isolation mechanism, with tenant context propagated via:
1. **JWT claim** `tenant_id` (authoritative — takes priority over headers)
2. **HTTP header** `X-Tenant-ID` (for service-to-service only, JWT takes priority)
3. **Database session** `SET LOCAL app.tenant_id = '...'`

Default tenant: `00000000-0000-0000-0000-000000000001`

### Consequences

**Positive:**
- Single database instance supports all tenants
- Easy to add new tenants (just insert a row)
- RLS provides guaranteed isolation at the DB layer
- Backup/restore operates on all tenants together

**Negative:**
- Noisy neighbor risk: one tenant's heavy queries affect others
- Migration complexity: schema changes affect all tenants
- Must be disciplined about tenant_id on every row
- Backup granularity is all-or-nothing (no per-tenant backup)

---

## ADR-007: Go as Primary Language

**Status:** Accepted

### Context

The platform requires high performance, low memory footprint, strong concurrency, and fast compilation. The team has deep Go expertise. Alternatives: Java (JVM overhead), Rust (steeper learning curve, slower development velocity), Node.js (single-threaded, GC pauses).

### Decision

Use **Go 1.25** for all backend services. Key reasons:
- Excellent concurrency primitives (goroutines, channels)
- Fast compilation and small binary sizes
- Strong standard library for HTTP, crypto, and SQL
- Excellent gRPC and Protocol Buffers support
- Static linking produces container-ready binaries

### Consequences

**Positive:**
- Small Docker images (15-35 MB per service)
- Fast build times enable rapid iteration
- Excellent goroutine model for concurrent request handling
- Strong tooling (go test, go vet, pprof)

**Negative:**
- Verbose error handling (`if err != nil` everywhere)
- No generics until Go 1.18 (now available but ecosystem still adapting)
- Less mature ORM ecosystem compared to Java/Python
- Manual memory management decisions (e.g., value vs pointer)

---

## ADR-008: Next.js for Admin Console

**Status:** Accepted

### Context

The admin console needs server-side rendering for SEO on public pages, client-side interactivity for the dashboard, and a component library for rapid development. Alternatives: plain React SPA (no SSR), Vue (smaller ecosystem), Svelte (less mature).

### Decision

Use **Next.js 15** with:
- App Router for routing
- TypeScript for type safety
- Tailwind CSS for styling
- Server Components for initial data loading
- Client Components for interactive widgets

### Consequences

**Positive:**
- SSR for fast initial page load
- File-based routing is intuitive
- Rich ecosystem of React components
- API routes for BFF (Backend for Frontend) pattern
- TypeScript integration is first-class

**Negative:**
- Larger JavaScript bundle than vanilla React
- Next.js version upgrades can be breaking
- Requires Node.js runtime (not a static site)
- Complex build pipeline

---

## ADR-009: Redis for Session and Rate Limiting

**Status:** Accepted

### Context

Session management (token revocation, refresh tokens) and rate limiting require a fast, in-memory key-value store with TTL support. Redis is the de facto standard for this use case. Alternatives: etcd (more suited for config), Memcached (no persistence, no data structures).

### Decision

Use **Redis 7** for:
- Refresh token storage with TTL-based expiry
- JWT JTI (anti-replay) tracking
- Rate limiting counters (sliding window)
- Session revocation list
- OAuth state parameter storage

### Consequences

**Positive:**
- Sub-millisecond reads for token validation
- Built-in TTL eliminates cleanup logic
- Atomic operations (INCR, SETNX) for rate limiting
- Pub/Sub for real-time notifications
- Wide language support and operational maturity

**Negative:**
- Additional infrastructure component
- Memory-only: data loss on crash (mitigated by AOF persistence)
- Must handle Redis connection failures gracefully
- No complex queries (key-value only)

---

## ADR-010: OpenLDAP for Directory Services

**Status:** Accepted

### Context

Enterprise customers require LDAP integration for existing directory infrastructure (Active Directory, OpenLDAP). The auth service must authenticate users against LDAP directories while maintaining fallback to local credentials.

### Decision

Integrate **OpenLDAP** as:
- Optional identity provider in the auth chain
- Supports BIND DN authentication and user search
- Auto-provisioning of LDAP users into local database
- START_TLS support for encrypted LDAP connections
- Configurable user filter and base DN per deployment

### Consequences

**Positive:**
- Enterprise customers can integrate without migration
- Automatic user provisioning reduces admin overhead
- Supports both LDAP and LDAPS (START_TLS)
- Fallback to local auth when LDAP is unavailable

**Negative:**
- LDAP protocol complexity (filter syntax, attribute mapping)
- Additional infrastructure (OpenLDAP server)
- No real-time sync (authenticate-on-demand only)
- Password policies may conflict with local policies

---

## ADR-011: Docker Compose for Local, Kubernetes for Production

**Status:** Accepted

### Context

Developers need a simple local setup that mirrors production. Docker Compose provides this with a single `docker-compose.yml`. Production requires orchestration for scaling, self-healing, and rolling updates. Kubernetes is the industry standard for container orchestration.

### Decision

- **Local/Dev**: Docker Compose with 13 containers (7 services + 4 infrastructure + console + migrate)
- **Production**: Kubernetes with Helm charts
- **CI/CD**: Build Docker images, push to registry, deploy via Helm

### Consequences

**Positive:**
- `docker compose up -d` gets full stack running in <60 seconds
- Production parity: same containers run locally and in prod
- Helm charts enable declarative configuration management
- Easy to add sidecars (monitoring, logging) in Kubernetes

**Negative:**
- Two deployment manifests to maintain (Compose + Helm)
- Docker Compose lacks health check sophistication
- Kubernetes learning curve for operations team
- Resource overhead from containerization

---

## ADR-012: OAuth 2.1 + OIDC for Authorization

**Status:** Accepted

### Context

OAuth 2.1 consolidates OAuth 2.0 best practices and deprecates insecure flows. OIDC adds identity layer on top. Supporting both enables GGID to act as an identity provider (IdP) for third-party applications. PKCE is mandatory in OAuth 2.1.

### Decision

Implement **OAuth 2.1 + OIDC** with:
- Authorization Code flow with PKCE (mandatory)
- Client Credentials flow for service-to-service
- Device Authorization flow (RFC 8628) for IoT/CLI
- OIDC UserInfo and ID Token endpoints
- Token introspection (RFC 7662) and revocation (RFC 7009)
- Dynamic client registration (RFC 7591) and management (RFC 7592)
- Pushed Authorization Requests (JAR/RFC 9126)
- mTLS client certificate support

### Consequences

**Positive:**
- Standards-compliant: works with any OAuth/OIDC client library
- PKCE eliminates implicit flow security issues
- Acts as full IdP for relying parties
- Rich set of flows covers all use cases

**Negative:**
- Significant implementation surface area
- Must track RFC updates for compliance
- PKCE adds a round-trip for client metadata exchange
- Testing requires understanding of all flows

---

## ADR-013: RBAC + ABAC Policy Engine

**Status:** Accepted

### Context

Role-Based Access Control (RBAC) is simple but coarse-grained. Attribute-Based Access Control (ABAC) is fine-grained but complex. Enterprises need both: roles for broad access patterns, attributes for fine-grained exceptions.

### Decision

Implement a **hybrid RBAC + ABAC policy engine** with:
- Roles: named collections of permissions scoped to tenant
- Permissions: fine-grained action/resource pairs
- ABAC rules: attribute-based conditions (time, location, risk score)
- Policy evaluation: RBAC check first, ABAC refinement second
- REST and gRPC APIs for policy management

### Consequences

**Positive:**
- Covers both coarse and fine-grained access control
- Flexible enough for complex enterprise requirements
- Policy changes take effect immediately (no recompilation)
- Standard RBAC model is easy to understand

**Negative:**
- Two policy models increase complexity
- ABAC rule evaluation adds latency
- Debugging denied requests requires checking both layers
- Must maintain consistency between roles and attributes

---

## ADR-014: Audit Trail via NATS JetStream

**Status:** Accepted

### Context

Audit events must be durable, tamper-evident, and queryable. The gateway generates audit events for every request. Writing directly to the database on every request adds latency. An async pipeline decouples ingestion from storage.

### Decision

Implement an **audit pipeline** with:
1. Gateway generates `AuditEvent` per request
2. Publishes to NATS JetStream subject `audit.events.>`
3. Audit service consumes events from durable consumer
4. Events stored in PostgreSQL with query API
5. Webhook delivery for real-time SIEM integration

### Consequences

**Positive:**
- Async pipeline: no request latency from audit writes
- Durable: JetStream persists events even if audit service is down
- Replayable: consumers can reprocess historical events
- Queryable via REST API for compliance reporting

**Negative:**
- Eventual consistency: audit events may lag by seconds
- No cryptographic hash chain (planned improvement)
- NATS becomes a critical infrastructure dependency
- Must handle duplicate events (at-least-once delivery)

---

## ADR-015: SCIM 2.0 for Automated Provisioning

**Status:** Accepted

### Context

Enterprise customers use SCIM (System for Cross-domain Identity Management) to automate user provisioning from their HR systems (Workday, Okta, Azure AD). SCIM 2.0 is the RFC 7643/7644 standard. Without SCIM, IT admins must manually create users.

### Decision

Implement **SCIM 2.0** endpoints in the Identity service:
- `/scim/v2/Users` — CRUD for user resources
- `/scim/v2/Groups` — CRUD for group resources
- Bulk operations (`POST /scim/v2/Bulk`)
- Filtering (`?filter=userName eq "john"`)
- Pagination (`startIndex`, `count`)
- PATCH for partial updates
- ETag-based concurrency control

### Consequences

**Positive:**
- Standard protocol: works with Okta, Azure AD, Workday
- Eliminates manual user provisioning
- Deprovisioning (DELETE) automatically revokes access
- Bulk operations reduce API calls

**Negative:**
- SCIM filter parsing is complex (SCIM filter DSL)
- Must handle concurrent PATCH operations carefully
- External SCIM clients may have non-standard implementations
- Requires bearer token authentication for SCIM clients

---

## Decision Log

| ID | Decision | Date | Status |
|----|----------|------|--------|
| ADR-001 | Microservices architecture | 2024-Q1 | Accepted |
| ADR-002 | Dual gRPC + REST API | 2024-Q1 | Accepted |
| ADR-003 | NATS JetStream event bus | 2024-Q1 | Accepted |
| ADR-004 | PostgreSQL RLS for multi-tenancy | 2024-Q1 | Accepted |
| ADR-005 | JWT stateless authentication | 2024-Q1 | Accepted |
| ADR-006 | Multi-tenant isolation strategy | 2024-Q1 | Accepted |
| ADR-007 | Go as primary language | 2024-Q1 | Accepted |
| ADR-008 | Next.js admin console | 2024-Q1 | Accepted |
| ADR-009 | Redis for session/rate limiting | 2024-Q1 | Accepted |
| ADR-010 | OpenLDAP directory services | 2024-Q1 | Accepted |
| ADR-011 | Docker Compose + Kubernetes | 2024-Q1 | Accepted |
| ADR-012 | OAuth 2.1 + OIDC | 2024-Q1 | Accepted |
| ADR-013 | RBAC + ABAC policy engine | 2024-Q1 | Accepted |
| ADR-014 | NATS JetStream audit trail | 2024-Q1 | Accepted |
| ADR-015 | SCIM 2.0 provisioning | 2024-Q1 | Accepted |

---

## Future Considerations

Decisions under consideration for future ADRs:

- **Audit hash chain**: Cryptographic chaining of audit events for tamper evidence
- **gRPC TLS**: mTLS between services (currently plaintext in internal network)
- **WebAuthn passwordless**: Moving toward passwordless as default auth method
- **Multi-region deployment**: Active-active vs active-passive for DR
- **gRPC service mesh**: Istio or Linkerd for observability and traffic management
- **Database sharding**: Per-tenant database for very large tenants

---

*Last updated: 2025-07-11*
