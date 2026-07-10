# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records for the GGID IAM platform.

ADRs document significant architectural choices, their context, and consequences.

| ADR | Title | Status |
|-----|-------|--------|
| [001](./ADR-001-microservices-vs-monolith.md) | Microservices over Monolith | Accepted |
| [002](./ADR-002-grpc-and-rest.md) | gRPC for internal + REST for external | Accepted |
| [003](./ADR-003-jwt-over-server-sessions.md) | JWT over Server-Side Sessions | Accepted |
| [004](./ADR-004-rls-for-multi-tenancy.md) | PostgreSQL RLS for Multi-Tenant Isolation | Accepted |
| [005](./ADR-005-nats-for-audit-pipeline.md) | NATS JetStream for Audit Event Pipeline | Accepted |

## Format

Each ADR follows the [Michael Nygard template](https://github.com/joelparkerhenderson/architecture-decision-record/tree/main/src/templates/en/adr-template-michael-nygard):

- **Title**
- **Status** (Proposed, Accepted, Deprecated, Superseded)
- **Context**
- **Decision**
- **Consequences** (positive, negative, neutral)
