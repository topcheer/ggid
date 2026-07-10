# Design: Audit Event Sourcing

> **Status:** Implemented

Complete design for GGID's audit event pipeline using NATS JetStream as the
event sourcing backbone.

---

## Overview

```
┌──────────┐  PublishAsync   ┌──────────────┐  Pull(100)   ┌──────────────┐
│ Services │ ──────────────► │ NATS          │ ───────────► │ Audit Service │
│ (Auth,   │                 │ JetStream     │              │ Consumer      │
│  Ident,  │                 │ Stream:       │              │               │
│  Policy, │                 │ AUDIT-EVENTS  │              │ INSERT → PG   │
│  Org)    │                 │ (file-backed) │              └──────────────┘
└──────────┘                 └──────┬───────┘                     │
                                   │                       Query / Export
                              Push │ (SSE)                      │
                                   ▼                     ┌──────┴──────┐
                            ┌─────────────┐              │ REST API    │
                            │ SIEM /      │              │ /audit/...  │
                            │ Analytics   │              └─────────────┘
                            │ Consumers   │
                            └─────────────┘
```

---

## NATS JetStream Configuration

### Stream Definition

| Property | Value | Rationale |
|----------|-------|-----------|
| Stream name | `AUDIT-EVENTS` | Uppercase convention |
| Subjects | `audit.>` | All audit event subjects |
| Retention | `limits` | Time/size-based expiry |
| Storage | `file` | Durable across NATS restarts |
| Max messages | 1,000,000 | Prevent unbounded growth |
| Max bytes | 5 GB | Storage cap |
| Max age | 7 days (604800s) | Compliance window |
| Discard policy | `old` | Drop oldest when limit reached |
| Replicas | 1 (RAFT when clustered) | Durability |

```conf
# nats-server.conf
jetstream {
    store_dir: "/data/nats"
    max_memory_store: 512MB
    max_file_store: 10GB
}
```

### Subject Hierarchy

Events are published with hierarchical subjects:

```
audit.events              → all events
audit.events.user         → user-related events
audit.events.user.login   → login events
audit.events.user.register → registration events
audit.events.role         → role-related events
audit.events.policy       → policy events
audit.events.org          → org events
audit.events.security     → security events (token theft, brute force)
```

Consumers can subscribe to specific subjects for filtered processing:

```go
// Subscribe only to security events
js.PullSubscribe("audit.events.security", "siem-consumer", ...)
```

---

## Event Schema

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "actor_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "actor_name": "admin",
  "action": "user.login",
  "result": "success",
  "resource_type": "auth",
  "resource_id": "",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
  "metadata": {
    "method": "password",
    "mfa_used": true,
    "session_id": "sess_abc123"
  },
  "timestamp": "2024-07-10T12:00:00.000Z"
}
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|:--------:|-------------|
| `id` | UUID | Yes | Unique event ID |
| `tenant_id` | UUID | Yes | Tenant scope |
| `actor_id` | UUID | No | User who triggered the action (null for system events) |
| `actor_name` | string | No | Username for display |
| `action` | string | Yes | Event type (e.g., `user.login`) |
| `result` | string | Yes | `success` or `failure` |
| `resource_type` | string | No | Type of affected resource |
| `resource_id` | string | No | ID of affected resource |
| `ip_address` | string | No | Client IP |
| `user_agent` | string | No | Client User-Agent |
| `metadata` | object | No | Additional event-specific data |
| `timestamp` | ISO8601 | Yes | When the event occurred |

---

## Consumer Groups

### Primary Consumer (Audit Service)

The main consumer that persists events to PostgreSQL:

```go
consumer, _ := js.PullSubscribe(
    "audit.events",          // subject
    "audit-pg-consumer",     // durable name
    nats.Durable("audit-pg-consumer"),
    nats.AckExplicit(),      // manual acknowledgment
    nats.MaxDeliver(3),      // retry up to 3 times
    nats.AckWait(30*time.Second),
)
```

**Batch processing:**

```go
for {
    batch, _ := consumer.Fetch(100, nats.MaxWait(5*time.Second))
    if len(batch) == 0 { continue }

    // Bulk insert to PostgreSQL
    events := make([]*AuditEvent, len(batch))
    for i, msg := range batch {
        json.Unmarshal(msg.Data, &events[i])
    }

    err := db.BulkInsertAuditEvents(ctx, events)
    if err != nil {
        // NAK all → redeliver
        for _, msg := range batch { msg.Nak() }
        continue
    }

    // Ack all
    for _, msg := range batch { msg.Ack() }
}
```

### SIEM Consumer (External)

Security teams can run a second durable consumer for SIEM forwarding:

```go
siemConsumer, _ := js.PullSubscribe(
    "audit.events.security",     // only security events
    "siem-forwarder",
    nats.Durable("siem-forwarder"),
    nats.AckExplicit(),
    nats.MaxDeliver(3),
)
```

### SSE Stream Consumer

For real-time dashboard streaming:

```go
sub, _ := js.Subscribe("audit.events", func(msg *nats.Msg) {
    // Push to connected SSE clients
    sseHub.Broadcast(msg.Data)
    msg.Ack()
}, nats.DeliverAll())
```

---

## Delivery Guarantees

### At-Least-Once Delivery

NATS JetStream provides at-least-once delivery:
- Messages are persisted to disk before acknowledgment
- If a consumer crashes, unacked messages are redelivered
- `MaxDeliver(3)` limits retries to prevent poison pills

### Ordering Guarantees

- **Per-subject ordering**: Events with the same subject are delivered in order
- **No global ordering**: Events across different subjects may arrive out of order
- For strict ordering, use a single subject and one consumer

### Idempotency

Events may be delivered more than once. The Audit Service deduplicates by `id`:

```sql
INSERT INTO audit_events (id, tenant_id, action, ...) VALUES (...)
ON CONFLICT (id) DO NOTHING;
```

---

## Replay Support

### Consumer Replay

JetStream supports replaying events from the beginning of the stream:

```go
// Create a new consumer that starts from the beginning
replayConsumer, _ := js.PullSubscribe(
    "audit.events",
    "replay-consumer",
    nats.Durable("replay-consumer"),
    nats.DeliverAll(),        // start from beginning
    nats.AckExplicit(),
)
```

### Use Cases

| Scenario | How |
|----------|-----|
| Rebuild PostgreSQL from scratch | Delete all rows, create replay consumer, process all events |
| Backfill SIEM | Create SIEM replay consumer with `DeliverAll` |
| Debug missing events | Create ephemeral consumer, replay specific time range |
| Test new consumer logic | Create test consumer, replay against test database |

### Time-Based Replay

```go
// Replay events from a specific time
startTime := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
consumer, _ := js.PullSubscribe(
    "audit.events",
    "time-replay",
    nats.Durable("time-replay"),
    nats.StartTime(startTime),
    nats.AckExplicit(),
)
```

---

## Reliability

### Failure Scenarios

| Scenario | Behavior | Data Loss? |
|----------|----------|:----------:|
| Service publishes event, NATS is up | Event persisted to JetStream | No |
| Service publishes event, NATS is down | `PublishAsync` silently fails (best-effort) | Yes (event lost) |
| NATS has event, consumer is down | Event buffered in JetStream (up to 7 days) | No |
| Consumer crashes mid-batch | Unacked events redelivered (MaxDeliver=3) | No |
| Consumer processes event, DB write fails | Event NAK'd → redelivered | No |
| PostgreSQL is down | Consumer can't persist → events buffer in NATS | No |
| JetStream disk full | `old` discard drops oldest events | Yes (oldest events) |

### Monitoring Consumer Lag

```bash
# Check consumer lag via NATS monitoring API
curl http://localhost:8222/jsz?consumers=true | \
  jq '.[] | .stream | .consumer[] | {
    name: .name,
    delivered: .delivered.stream_seq,
    ack_floor: .ack_floor.stream_seq,
    pending: (.delivered.stream_seq - .ack_floor.stream_seq),
    redelivered: .num_redelivered
  }'
```

**Alert threshold:** pending > 1000 (consumer is falling behind).

---

## Performance

| Metric | Value |
|--------|-------|
| Publish latency (async) | < 1ms |
| Consumer throughput (batch=100) | ~5,000 events/sec |
| Event size (avg) | ~200 bytes |
| JetStream disk per event | ~300 bytes (metadata overhead) |
| PostgreSQL insert (batch=100) | ~10ms |

### Tuning

1. **Batch size**: 100 is optimal for throughput vs latency balance
2. **Ack wait**: 30s default — increase if DB is slow
3. **Max deliver**: 3 prevents poison-pill infinite loops
4. **Stream max age**: 7 days balances compliance vs storage
5. **File storage**: Use SSD for JetStream store_dir
