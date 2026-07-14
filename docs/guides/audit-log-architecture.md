# Audit Log Architecture

Event schema, NATS JetStream pipeline, hash chain integrity, storage, query optimization, retention, and forensics.

## Architecture Overview

```
Service ──event──▶ NATS JetStream ──▶ Audit Consumer ──▶ PostgreSQL
                      │                      │
                      │                      ├──▶ Hash Chain (integrity)
                      │                      └──▶ SIEM Forwarder
                      │
                      └── retention: 7 days
```

Events flow from all 7 microservices through NATS JetStream to the audit service, which persists them to PostgreSQL and optionally forwards to SIEM.

## Event Schema

```json
{
  "event_id": "evt-uuid-v7",
  "timestamp": "2025-01-15T10:30:00.123456789Z",
  "actor": {
    "user_id": "uuid",
    "tenant_id": "uuid",
    "ip": "10.0.1.5",
    "user_agent": "Mozilla/5.0...",
    "session_id": "sess-abc"
  },
  "action": "user.create",
  "resource": {
    "type": "user",
    "id": "uuid",
    "name": "jane@corp.com"
  },
  "result": "success",
  "details": {
    "source": "console",
    "method": "POST",
    "path": "/api/v1/users"
  },
  "metadata": {
    "request_id": "req-xyz",
    "trace_id": "trace-123",
    "risk_score": 15
  }
}
```

### Field Requirements

| Field | Required | Description |
|-------|----------|-------------|
| event_id | Yes | UUIDv7 (time-ordered) |
| timestamp | Yes | RFC 3339 with nanoseconds |
| actor.user_id | Yes | Who performed the action |
| actor.tenant_id | Yes | Multi-tenant isolation |
| actor.ip | Yes | Source IP |
| action | Yes | Verb + resource (e.g., `user.create`) |
| result | Yes | `success`, `failure`, `denied` |
| resource.type | Yes | Entity type |
| resource.id | Conditional | Required for success/failure on specific entity |

## NATS JetStream Pipeline

### Stream Configuration

```yaml
stream:
  name: AUDIT_EVENTS
  subjects: ["audit.>"]
  retention: limits
  max_age: 168h        # 7 days in JetStream
  max_msgs: 10000000   # 10M messages
  max_bytes: 10GB
  storage: file
  replicas: 3          # HA
  duplicate_window: 5m # Dedup by event_id
```

### Publisher (Per Service)

```go
func PublishAuditEvent(js nats.JetStream, event AuditEvent) error {
    data, _ := json.Marshal(event)
    _, err := js.Publish("audit."+event.Action, data,
        nats.MsgId(event.EventID),  // Dedup
        nats.AckWait(5*time.Second),
    )
    return err
}
```

### Consumer (Audit Service)

```go
sub, _ := js.PullSubscribe("audit.>", "AUDIT_WORKER",
    nats.Durable("AUDIT_WORKER"),
    nats.MaxDeliver(10),         // Retry up to 10x
    nats.AckWait(30*time.Second),
)

for {
    msgs, _ := sub.Fetch(100, nats.MaxWait(5*time.Second))
    for _, msg := range msgs {
        event := parseEvent(msg)
        storeEvent(event)           // PostgreSQL
        updateHashChain(event)      // Integrity
        msg.Ack()
    }
}
```

## Hash Chain Integrity

Each event's hash is chained to the previous, making tampering detectable:

```go
type ChainEntry struct {
    EventID    string
    EventHash  string
    PrevHash   string
    Sequence   int64
}

func updateHashChain(event AuditEvent) {
    prev := getLatestChainEntry()

    payload := fmt.Sprintf("%s|%s|%s|%s",
        event.EventID, event.Timestamp, event.Action, prev.EventHash)
    hash := sha256.Sum256([]byte(payload))

    entry := ChainEntry{
        EventID:   event.EventID,
        EventHash: hex.EncodeToString(hash[:]),
        PrevHash:  prev.EventHash,
        Sequence:  prev.Sequence + 1,
    }
    db.Insert(entry)
}
```

### Verification

```bash
# Verify chain integrity (periodic or on-demand)
GET /api/v1/audit/verify-chain
# → {"status":"valid","verified_events":1452390,"broken_links":0}

# If broken_links > 0 → tampering detected → security alert
```

## Storage Strategy

| Layer | Purpose | Retention | Query Pattern |
|-------|---------|-----------|---------------|
| NATS JetStream | Buffer + delivery | 7 days | Sequential |
| PostgreSQL (hot) | Active queries | 1 year | Indexed by actor, action, time |
| PostgreSQL archive (cold) | Compliance retention | 7 years | Infrequent, by export |
| SIEM forward | External analysis | Per SIEM policy | Not stored locally |

### PostgreSQL Partitioning

```sql
-- Partition by month for query performance
CREATE TABLE audit_events (...) PARTITION BY RANGE (created_at);

CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Indexes per partition
CREATE INDEX ON audit_events_2025_01 (actor_user_id, created_at DESC);
CREATE INDEX ON audit_events_2025_01 (action, created_at DESC);
CREATE INDEX ON audit_events_2025_01 (tenant_id, created_at DESC);
```

## Query Optimization

```bash
# Fast queries use indexed columns
GET /api/v1/audit/events?user_id=uuid&from=2025-01-01&to=2025-01-31
GET /api/v1/audit/events?action=user.create&result=denied
GET /api/v1/audit/events?tenant_id=uuid&limit=100

# Slow queries (avoid without filters)
GET /api/v1/audit/events  # No filters → full scan, rate-limited
```

### Query Performance Tips

- Always include `tenant_id` (enables partition pruning)
- Use `created_at` range to narrow partition scope
- Limit result sets (max 1000 per request, use pagination)
- Export large datasets asynchronously

## Retention Policy

```sql
-- Automated nightly cleanup
-- Move events older than 1 year to archive table
INSERT INTO audit_events_archive
  SELECT * FROM audit_events
  WHERE created_at < NOW() - INTERVAL '1 year';

DELETE FROM audit_events
  WHERE created_at < NOW() - INTERVAL '1 year';

-- Purge archive after 7 years (compliance retention met)
DELETE FROM audit_events_archive
  WHERE created_at < NOW() - INTERVAL '7 years';
```

| Data Class | Hot Retention | Archive | Total |
|-----------|---------------|---------|-------|
| Security events | 1 year | 7 years | 7 years |
| Auth events | 1 year | 7 years | 7 years |
 Admin actions | 1 year | 7 years | 7 years |
| Debug logs | 30 days | — | 30 days |

## Forensics Workflow

### Investigation Steps

```
1. Identify suspect (user_id or IP)
2. Query all events for actor within timeframe
   GET /api/v1/audit/events?user_id=uuid&from=...&to=...
3. Cross-reference with session events
   GET /api/v1/audit/events?session_id=sess-abc
4. Check hash chain integrity
   GET /api/v1/audit/verify-chain
5. Export evidence package
   GET /api/v1/audit/export?user_id=uuid&format=csv
```

### Evidence Export

```bash
GET /api/v1/audit/export
  ?user_id=uuid
  &from=2025-01-01
  &to=2025-01-31
  &format=csv    # or json, pdf
  &include_chain=true
# → Signed evidence package (GPG signature)
```

## See Also

- [Audit API](../api/audit-api.md)
- [SIEM Integration](siem-integration.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [Identity Threat Detection](identity-threat-detection.md)
