# ADR-001: Microservices over Monolith

- **Status:** Accepted
- **Date:** 2024-01-15

## Context

GGID is an Identity and Access Management platform with distinct functional domains:
authentication, user identity management, policy/RBAC evaluation, organization
management, audit logging, and OAuth/OIDC federation.

We considered two architectural styles:

1. **Monolith** — single binary, single database, shared codebase
2. **Microservices** — independently deployable services, each with its own
gRPC/REST API, communicating through the API Gateway

### Forces

- Different domains have different scalability profiles (auth is high-QPS,
low-latency; audit is write-heavy, eventual-consistency)
- Team members need to work independently without merge conflicts
- Some services may need different runtime characteristics (e.g., audit needs
NATS consumer, auth needs Redis)
- Open-source users may want to deploy only specific components
- Operational complexity must remain manageable for small teams

## Decision

We chose **microservices with a shared API Gateway**.

The system is decomposed into 7 services:

| Service | Responsibility |
|---------|---------------|
| Gateway | JWT verification, reverse proxy, rate limiting |
| Identity | User CRUD, SCIM 2.0 |
| Auth | Register, login, JWT issuance, MFA, passwordless |
| OAuth | OAuth2/OIDC, JWKS, SAML 2.0 |
| Policy | RBAC + ABAC engine, roles, permissions |
| Org | Tenants, org tree, departments, teams |
| Audit | Event query, NATS consumer, retention |

All services share:
- PostgreSQL database (with RLS for tenant isolation)
- Go module with shared `pkg/` packages
- Consistent `internal/{config, domain, service, server}` structure

## Consequences

### Positive

- Independent scaling: auth can scale to 10 replicas while audit stays at 2
- Team parallelism: each developer owns a service with clear boundaries
- Selective deployment: users can deploy only the services they need
- Blast radius: a bug in audit doesn't take down authentication
- Technology flexibility: services could be rewritten independently

### Negative

- Operational complexity: 7 services to monitor, deploy, and debug
- Network latency: gateway adds a proxy hop (~1-2ms overhead)
- Shared database coupling: schema migrations affect multiple services
- Distributed tracing needed for debugging cross-service issues

### Neutral

- Go's single-binary model means the deployment artifacts are small (~20-35MB each)
- Docker Compose makes local development straightforward
