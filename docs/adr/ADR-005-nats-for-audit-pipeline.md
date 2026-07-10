# ADR-005: NATS JetStream for Audit Event Pipeline

- **Status:** Accepted
- **Date:** 2024-02-05

## Context

GGID needs to record audit events for every security-relevant action (login,
role changes, user modifications, policy decisions). These events must be:

- **Reliable** — no events lost, even during service restarts
- **Non-blocking** — audit logging must not slow down the primary operation
- **Queryable** — events must be searchable by action, actor, time range
- **Exportable** — CSV export and real-time streaming for SIEM integration

Options considered:

1. **Direct database writes** — each service writes audit events to PostgreSQL
2. **Redis pub/sub** — fire-and-forget event publication
3. **NATS JetStream** — persistent message streaming with at-least-once delivery
4. **Kafka** — enterprise-grade event streaming

### Forces

- Services should not block on audit writes (latency-sensitive auth flows)
- Events must survive service crashes (durability)
- The platform should be lightweight (no Kafka dependency for small deployments)
- NATS is already used by some team members and has a small footprint
- The audit service needs to both store events in PostgreSQL and stream them
to subscribers (SIEM, dashboards)

## Decision

We chose **NATS JetStream** as the audit event transport.

### Design

- **Publisher**: Each service (auth, identity, policy, org) publishes audit
events to a NATS JetStream stream (`audit-events`) using the shared
`pkg/audit.Publisher`. Publication is **best-effort** — if NATS is down,
the service continues operating (events are skipped, not queued).
- **Consumer**: The audit service runs a JetStream durable consumer that
reads events from the stream and persists them to PostgreSQL. The consumer
uses at-least-once delivery with manual acknowledgment.
- **Query API**: The audit service exposes REST endpoints for querying
stored events with filtering by action, actor, time range, resource type.
- **Streaming**: Server-Sent Events (SSE) endpoint for real-time event streaming.

## Consequences

### Positive

- **Decoupled**: Services don't know about PostgreSQL or the audit schema —
they just publish to NATS
- **Resilient**: JetStream persists messages to disk. If the audit consumer
crashes, events are buffered in the stream and processed on restart
- **Non-blocking**: Publishing is asynchronous; auth flows aren't slowed
- **Extensible**: Multiple consumers can read from the same stream (SIEM,
analytics, alerting) without modifying services
- **Lightweight**: NATS binary is ~15MB, much lighter than Kafka

### Negative

- Best-effort publishing means events may be lost if NATS is down during a
service operation (acceptable for audit vs. critical for transactions)
- Adds NATS as an infrastructure dependency
- Event ordering is per-subject, not global (usually fine for audit)
- JetStream persistence requires disk space management

### Neutral

- The `pkg/audit.Publisher` interface allows swapping NATS for Kafka or
SQS without changing service code
- Default stream retention is 7 days; configurable via NATS config
- The audit service gracefully degrades when NATS is unavailable
