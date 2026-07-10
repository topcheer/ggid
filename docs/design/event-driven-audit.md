# Design: Event-Driven Audit Pipeline

> **Status:** Implemented | **ADR:** [ADR-005](../adr/ADR-005-nats-for-audit-pipeline.md)

## Problem Statement

GGID must record audit events for every security-relevant action (login, role
change, user modification, policy decision). The pipeline must be:

- **Reliable** — events survive service crashes
- **Non-blocking** — audit logging must not slow down the primary operation
- **Queryable** — events searchable by action, actor, time range
- **Extensible** — multiple consumers (SIEM, analytics) without modifying services

## Architecture

```
┌──────────────┐  publish   ┌──────────────┐  consume   ┌──────────────┐
│ Auth Service │ ─────────► │ NATS         │ ─────────► │ Audit Service │
│ Identity Svc │            │ JetStream    │            │ (PostgreSQL)  │
│ Policy Svc   │            │ (durable)    │            └──────────────┘
│ Org Svc      │            │              │                 │
└──────────────┘            └──────┬───────┘                 │
                                   │                   query / export
                                   │                         │
                              subscribe                  ┌──────┴──────┐
                                   │                     │  REST API   │
                            ┌──────┴──────┐              │ /audit/...  │
                            │   SIEM /    │              └─────────────┘
                            │  Analytics  │
                            └─────────────┘
```

## Components

### 1. Publisher (in each service)

Each service uses the shared `pkg/audit.Publisher` to publish events:

```go
// pkg/audit/publisher.go
type Publisher interface {
    Publish(ctx context.Context, event *AuditEvent) error
    PublishAsync(ctx context.Context, event *AuditEvent)
}

type AuditEvent struct {
    TenantID     uuid.UUID
    ActorID      uuid.UUID
    ActorName    string
    Action       string         // e.g. "user.login"
    Result       string         // "success" or "failure"
    ResourceType string
    ResourceID   string
    IPAddress    string
    UserAgent    string
    Metadata     map[string]any
    Timestamp    time.Time
}
```

**Best-effort publishing:** If NATS is down, `PublishAsync` returns immediately
without blocking the primary operation. The event is lost but the service
continues operating.

### 2. NATS JetStream (transport)

- **Stream name:** `AUDIT-EVENTS`
- **Subject:** `audit.>`
- **Retention:** 7 days (configurable)
- **Storage:** File-backed (durable across NATS restarts)
- **Max messages:** 1,000,000
- **Discard policy:** Old (drop oldest when limit reached)

```conf
jetstream {
    store_dir: "/data/nats"
}
```

### 3. Consumer (Audit Service)

The Audit service runs a **durable consumer** that:
1. Reads events from the JetStream stream
2. Persists them to PostgreSQL
3. Acknowledges (ack) successful writes
4. Re-delivers unacked events on crash recovery

```go
// Batch processing for efficiency
consumer, _ := js.PullSubscribe("audit.events", "audit-consumer",
    nats.Durable("audit-consumer"),
    nats.AckExplicit(),
    nats.MaxDeliver(3),
)

for {
    batch, _ := consumer.Fetch(100, nats.MaxWait(5*time.Second))
    for _, msg := range batch {
        var event AuditEvent
        json.Unmarshal(msg.Data, &event)
        err := db.InsertAuditEvent(ctx, &event)
        if err != nil {
            msg.Nak()  // negative ack → redeliver
            continue
        }
        msg.Ack()
    }
}
```

### 4. Query API (Audit Service REST)

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/audit/events` | Query with filters (action, actor, time range) |
| `GET /api/v1/audit/events/{id}` | Get single event |
| `GET /api/v1/audit/stats` | Aggregated statistics |
| `GET /api/v1/audit/export` | CSV export |
| `GET /api/v1/audit/stream` | SSE real-time streaming |
| `GET/PUT /api/v1/audit/retention` | Retention configuration |
| `GET/POST /api/v1/audit/rules` | Anomaly detection rules |

## Event Taxonomy

| Action | Triggered By | Service |
|--------|-------------|--------|
| `user.login` | Successful login | Auth |
| `user.login_failed` | Failed login attempt | Auth |
| `user.register` | New user registered | Auth |
| `user.logout` | User logs out | Auth |
| `user.create` | User created via API | Identity |
| `user.update` | User profile updated | Identity |
| `user.delete` | User deleted | Identity |
| `user.lock` | Account locked | Identity |
| `user.unlock` | Account unlocked | Identity |
| `role.create` | Role created | Policy |
| `role.assign` | Role assigned to user | Policy |
| `policy.create` | Policy created | Policy |
| `policy.check` | Permission evaluated | Policy |
| `org.create` | Organization created | Org |
| `org.member.add` | Member added to org | Org |
| `password.reset` | Password reset | Auth |
| `password.change` | Password changed | Auth |
| `mfa.enable` | MFA enabled | Auth |
| `mfa.disable` | MFA disabled | Auth |

## Reliability Guarantees

| Scenario | Behavior |
|----------|---------|
| Service publishes event | Non-blocking (PublishAsync) |
| NATS is down | Event is lost (best-effort), service continues |
| NATS is up, consumer is down | Event buffered in JetStream (up to 7 days) |
| Consumer crashes mid-processing | Event redelivered (manual ack, MaxDeliver=3) |
| Consumer processes but DB write fails | Event NAK'd → redelivered |
| PostgreSQL is down | Consumer can't persist → events buffer in NATS |

## Performance Characteristics

- **Publish latency:** < 1ms (async, non-blocking)
- **Consume throughput:** ~5000 events/sec (batch of 100)
- **Storage:** ~200 bytes per event in PostgreSQL
- **JetStream overhead:** ~1MB per 5000 events on disk

## Extensibility

Multiple consumers can subscribe to the same stream:

```
AUDIT-EVENTS stream
  ├── audit-consumer (GGID Audit Service → PostgreSQL)
  ├── siem-consumer  (Splunk / Datadog forwarding)
  ├── analytics      (real-time dashboards)
  └── alerting       (anomaly detection, Slack alerts)
```

New consumers can be added without modifying any service — just create a new
JetStream durable consumer.
