# ADR-002: NATS JetStream for the Audit Pipeline

**Status:** Accepted
**Date:** 2024-Q1
**Deciders:** Architecture Team

---

## Context

GGID must record an audit event for every API request (who, what, when, from where, result). This creates high-volume, write-heavy traffic. The audit pipeline must:

1. **Not block the request path**: Audit logging must be asynchronous
2. **Be durable**: Events must survive service crashes
3. **Be replayable**: Consumers can reprocess historical events
4. **Support multiple consumers**: Audit storage, webhook delivery, SIEM forwarding
5. **Be operationally simple**: Minimal infrastructure overhead

### Alternatives Considered

#### Option A: Direct Database Writes (Synchronous)

```
Gateway → INSERT INTO audit_events → Response
```

**Pros:**
- Simplest implementation
- Immediate queryability
- No additional infrastructure

**Cons:**
- Adds 5-20ms to every request (database write)
- Database becomes bottleneck under high load
- No retry on transient failures
- Cannot fan out to multiple consumers

#### Option B: Direct Database Writes (Async Goroutine)

```
Gateway → go func() { INSERT INTO audit_events }() → Response
```

**Pros:**
- Non-blocking request path
- No new infrastructure

**Cons:**
- Events lost on process crash (goroutine never executes)
- No retry mechanism
- No fan-out to multiple consumers
- Backpressure under load (unbounded goroutines)

#### Option C: Kafka

```
Gateway → Kafka Topic → Audit Consumer → Database
```

**Pros:**
- Battle-tested for high-volume event streaming
- Excellent durability and replayability
- Rich ecosystem (Kafka Streams, Connect)

**Cons:**
- Heavy operational footprint (ZooKeeper/KRaft, multiple brokers)
- JVM overhead (or use Redpanda — but adds complexity)
- Overkill for GGID's audit volume (<10k events/sec)
- Resource-intensive for small deployments

#### Option D: NATS JetStream (Selected)

```
Gateway → NATS JetStream → Audit Consumer → Database
                          → Webhook Delivery
                          → SIEM Forwarder
```

**Pros:**
- Lightweight: single binary, no ZooKeeper
- Built-in persistence (file-based)
- Subjects-based routing (`audit.events.>`)
- Consumer groups for parallel processing
- Monitoring HTTP endpoint (port 8222)
- Durable consumers with acknowledgment

**Cons:**
- Smaller ecosystem than Kafka
- Less mature stream processing DSL
- JetStream configuration learning curve

---

## Decision

Choose **NATS JetStream** as the event bus for the audit pipeline.

### Architecture

```
┌──────────┐    ┌───────────────┐    ┌──────────────┐    ┌─────────────┐
│ Gateway  │───▶│    NATS       │───▶│ Audit Svc    │───▶│ PostgreSQL  │
│          │    │  JetStream    │    │ (consumer)   │    │ (storage)   │
└──────────┘    │               │    └──────────────┘    └─────────────┘
                │  audit.events │    ┌──────────────┐
                │  .>  (durable)│───▶│ Webhook      │───▶│ External    │
                │               │    │ Delivery     │    │ Endpoints  │
                └───────────────┘    └──────────────┘    └─────────────┘
```

### Implementation Details

1. **Stream**: `AUDIT` — durable stream on subject `audit.events.>`
2. **Gateway**: Publishes `AuditEvent` after every request (async, fire-and-forget to NATS)
3. **Audit Service**: Durable consumer processes events, writes to PostgreSQL
4. **Webhook Delivery**: Separate consumer delivers to registered webhook endpoints
5. **Retention**: 7-day JetStream retention for replay; 90-day PostgreSQL retention for queries

### Key Configuration

```yaml
# NATS JetStream stream
AUDIT:
  subjects: ["audit.events.>"]
  retention: limits
  max_age: 7d
  max_bytes: 10GB
  storage: file
  replicas: 1  # 3 for production

# Durable consumer
audit-consumer:
  deliver_policy: all
  ack_policy: explicit
  ack_wait: 30s
  max_deliver: 5
```

---

## Consequences

### Positive

- **Non-blocking**: Gateway publishes to NATS in <1ms, no database write on request path.
- **Durable**: Events survive gateway crashes. JetStream persists to disk.
- **Replayable**: New consumers can process historical events from the stream.
- **Multi-consumer**: Audit storage, webhook delivery, and SIEM forwarding are independent consumers.
- **Operational simplicity**: NATS is a single binary. Monitoring on `:8222/healthz`.
- **Backpressure handling**: NATS handles slow consumers with `max_deliver` and redelivery.

### Negative

- **Eventual consistency**: Audit events lag by 1-5 seconds. Not available immediately after request.
- **Additional infrastructure**: NATS must be deployed, monitored, and backed up.
- **Duplicate delivery**: At-least-once semantics means consumers must handle duplicates (via `event_id` deduplication).
- **No hash chain**: Events are not cryptographically chained (planned improvement for tamper evidence).

### Trade-off Analysis

| Factor | Direct DB (sync) | Kafka | NATS JetStream |
|--------|-----------------|-------|----------------|
| Request latency | +5-20ms | +1-3ms | +<1ms |
| Durability | DB only | Excellent | Good (file-based) |
| Operational complexity | Lowest | Highest | Medium |
| Fan-out | No | Yes | Yes |
| Replay | No | Yes | Yes |
| Resource footprint | None | High (JVM) | Low (single binary) |

---

## Future Considerations

1. **Audit hash chain**: Each event includes `previous_hash` (SHA-256) for tamper detection
2. **Kafka migration path**: If volume exceeds NATS capacity, the consumer interface allows swapping JetStream for Kafka without changing publishers
3. **Stream processing**: For real-time anomaly detection (brute force, unusual access patterns)

---

## References

- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [GGID Audit Guide](../audit-guide.md)
- [GGID Webhook Guide](../webhook-guide.md)
- Related: [event-driven-audit.md](./event-driven-audit.md), [event-sourcing.md](./event-sourcing.md)

---

*Last updated: 2025-07-11*
