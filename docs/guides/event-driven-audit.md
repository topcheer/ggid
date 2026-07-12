# Event-Driven Audit Architecture

NATS JetStream stream config, consumer patterns, deduplication, ordering guarantees, backpressure, replay, and exactly-once semantics.

## Architecture

```
Microservices (7 services)
    │  Publish audit events
    ▼
NATS JetStream
    │  AUDIT_EVENTS stream
    │  (durable, replicated, 7-day retention)
    │
    ├── Consumer 1: Audit Service (persist to PostgreSQL)
    ├── Consumer 2: SIEM Forwarder (forward to Splunk/QRadar)
    ├── Consumer 3: Hash Chain (compute integrity hash)
    └── Consumer 4: Real-time Alerting (anomaly detection)
```

## Stream Configuration

```yaml
stream:
  name: AUDIT_EVENTS
  subjects: ["audit.>"]
  retention: limits
  storage: file           # Persistent disk
  max_age: 168h           # 7 days in JetStream
  max_msgs: 10000000      # 10M messages
  max_bytes: 10GB
  replicas: 3             # HA (3-node cluster)
  duplicate_window: 5m    # Dedup by MsgId
  discard: old            # Drop oldest when full
```

### Subject Hierarchy

```
audit.user.created
audit.user.updated
audit.user.deleted
audit.user.login
audit.user.login_failed
audit.role.assigned
audit.role.revoked
audit.policy.evaluated
audit.session.created
audit.session.revoked
```

## Publisher Pattern

```go
type AuditPublisher struct {
    js nats.JetStream
}

func (p *AuditPublisher) Publish(event AuditEvent) error {
    data, _ := json.Marshal(event)
    
    subject := "audit." + event.Action
    
    _, err := p.js.Publish(subject, data,
        nats.MsgId(event.EventID),           // Dedup (idempotent)
        nats.AckWait(5*time.Second),         // Retry if unacked
        nats.MaxDeliver(10),                 // Max delivery attempts
    )
    
    if err != nil {
        log.Error("audit publish failed", err)
        // Fallback: write to local file for later replay
        writeToFallback(event)
    }
    return err
}
```

### Publish Best Practices

| Practice | Rationale |
|----------|-----------|
| Use `nats.MsgId(event.EventID)` | Dedup — prevents duplicate events |
| Handle publish failure | Fallback to local file |
| Never block on publish | Async — don't slow request path |
| Batch small events | Reduces overhead for high volume |

## Consumer Patterns

### Durable Push Consumer

```go
// Audit Service — durable, processes all events
sub, _ := js.Subscribe("audit.>", func(msg *nats.Msg) {
    event := parseEvent(msg)
    storeEvent(event)
    updateHashChain(event)
    msg.Ack()
}, 
    nats.Durable("AUDIT_PERSISTER"),
    nats.MaxDeliver(10),
    nats.AckWait(30*time.Second),
    nats.ManualAck(),
)
```

### Pull Consumer (Batch Processing)

```go
// Batch processor — pulls in batches of 100
sub, _ := js.PullSubscribe("audit.>", "AUDIT_BATCH",
    nats.Durable("AUDIT_BATCH"),
)

for {
    msgs, err := sub.Fetch(100, nats.MaxWait(5*time.Second))
    if err != nil { continue }
    
    events := make([]AuditEvent, len(msgs))
    for i, msg := range msgs {
        events[i] = parseEvent(msg)
    }
    
    // Batch insert to PostgreSQL
    batchStore(events)
    
    // Ack all
    for _, msg := range msgs {
        msg.Ack()
    }
}
```

### Queue Group (Load-Balanced)

```go
// Multiple workers share the consumer group
js.Subscribe("audit.>", handler,
    nats.Durable("AUDIT_WORKER"),
    nats.Queue("audit-workers"),  // Load-balanced across instances
)
```

## Deduplication

### Publisher-Side (MsgId)

```go
// NATS JetStream deduplicates by MsgId within duplicate_window (5 min)
js.Publish("audit.user.login", data, nats.MsgId("evt-uuid-123"))
// If published twice with same MsgId → stored once
```

### Consumer-Side

```go
func storeEvent(event AuditEvent) error {
    // PostgreSQL INSERT ... ON CONFLICT DO NOTHING
    _, err := db.Exec(`
        INSERT INTO audit_events (event_id, ...) 
        VALUES ($1, ...)
        ON CONFLICT (event_id) DO NOTHING
    `, event.EventID)
    return err
}
```

## Ordering Guarantees

### Per-Subject Ordering

NATS JetStream preserves ordering **within a single subject**:

```
audit.user.login (from user A): Event₁ → Event₂ → Event₃  ✓ Ordered
audit.user.login (from user B): Event₄ → Event₅ → Event₆  ✓ Ordered
Cross-subject: Event₃ and Event₄ may arrive in any order
```

### Per-Entity Ordering (Partitioning)

For strict per-entity ordering, use entity ID in subject:

```go
subject := fmt.Sprintf("audit.user.%s", event.Actor.UserID)
js.Publish(subject, data, nats.MsgId(event.EventID))
// Events for same user always arrive in order
```

## Backpressure

### MaxAckPending

```go
// Limit unacked messages per consumer
js.PullSubscribe("audit.>", "AUDIT_WORKER",
    nats.Durable("AUDIT_WORKER"),
    nats.MaxAckPending(1000),  // Max 1000 unacked
)
// Consumer stops receiving when 1000 unacked
```

### Monitoring Backpressure

| Metric | Alert |
|--------|-------|
| Pending messages | >10,000 → consumer slow |
| Redelivery count | >2 → processing failures |
| Ack wait timeouts | Spike → consumer overloaded |
| Consumer lag | >60s → scale consumers |

## Replay

### From Beginning

```bash
# Replay all events from start of stream
POST /api/v1/audit/replay
{
  "consumer": "AUDIT_PERSISTER",
  "from": "beginning",
  "subject_filter": "audit.user.*"
}
```

### From Timestamp

```bash
POST /api/v1/audit/replay
{
  "consumer": "AUDIT_PERSISTER",
  "from": "2025-01-15T10:00:00Z",
  "to": "2025-01-15T11:00:00Z"
}
```

### Replay Use Cases

- PostgreSQL data loss recovery
- New consumer backfill (e.g., adding SIEM forwarder)
- Re-computing hash chain after bug fix
- Testing new event processor

## Delivery Semantics

### At-Least-Once (Default)

NATS JetStream provides **at-least-once** delivery. Consumers must be idempotent (handle duplicates via event_id dedup).

### Effectively-Once (Consumer)

```go
// Consumer-side dedup achieves effectively-once:
func processEvent(event AuditEvent) error {
    // Check if already processed
    if processed := redis.SetNX(ctx, "processed:"+event.EventID, 1, 24*time.Hour); !processed {
        return nil // Already processed, skip
    }
    
    // Process
    doWork(event)
    return nil
}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Stream depth | <10,000 msgs | >50,000 → consumer lag |
| Publish latency | <5ms | >50ms → NATS overloaded |
| Consumer lag | <1s | >10s → scale consumers |
| Dedup hits | Track | High rate → publisher retrying |
| Redelivery rate | <1% | >5% → consumer crashes |

## See Also

- [Audit Log Architecture](audit-log-architecture.md)
- [Audit Tamper Detection](audit-tamper-detection.md)
- [Webhook Delivery Guarantees](webhook-delivery-guarantees.md)
- [SIEM Integration](siem-integration.md)
