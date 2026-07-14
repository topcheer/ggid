# Event-Driven Architecture

> How GGID uses NATS JetStream for audit events, webhook delivery, and async processing.

---

## Overview

```
┌──────────┐  Publish   ┌──────────────┐  Consume    ┌──────────────┐
│ Any Service│ ────────▶ │ NATS          │ ──────────▶│ Audit Service│
│ (auth,    │            │ JetStream     │            │ (hash chain  │
│  identity,│            │ (durable,     │            │  + persist)  │
│  policy)  │            │  at-least-once)│           └──────────────┘
└──────────┘            └──────┬───────┘
                               │ Push
                               ▼
                       ┌──────────────┐
                       │ Webhook      │
                       │ Delivery     │
                       │ (retry, HMAC)│
                       └──────────────┘
```

---

## NATS JetStream

GGID uses NATS JetStream as the event bus:

- **Stream:** `AUDIT_EVENTS` — all audit events
- **Durability:** Files-based storage (survives restarts)
- **Delivery:** At-least-once (consumers ack after processing)
- **Retention:** 7 days (configurable)

### Configuration

```bash
NATS_URL=nats://localhost:4222
NATS_STREAM_AUDIT=AUDIT_EVENTS
NATS_RETENTION=168h  # 7 days
``n
---

## Event Types

| Category | Action Format | Examples |
|----------|--------------|----------|
| **Authentication** | `user.*` | `user.login`, `user.logout`, `user.register`, `user.mfa_verify` |
| **User Management** | `user.*` | `user.create`, `user.update`, `user.delete`, `user.role_assign` |
| **Authorization** | `role.*`, `policy.*` | `role.create`, `role.update`, `policy.check` |
| **Organization** | `org.*` | `org.create`, `org.update`, `org.delete` |
| **OAuth** | `oauth.*` | `oauth.authorize`, `oauth.token`, `oauth.introspect` |
| **SCIM** | `scim.*` | `scim.user_create`, `scim.user_patch`, `scim.group_update` |
| **Security** | `security.*` | `security.rate_limit`, `security.ip_block`, `security.circuit_break` |

### Event Structure

```json
{
  "event_id": "evt_abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "actor_type": "user",
  "actor_id": "usr_abc123",
  "action": "user.login",
  "resource_type": "auth",
  "resource_id": "",
  "metadata": {
    "ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0",
    "success": true
  },
  "timestamp": "2025-07-11T12:00:00Z",
  "prev_hash": "sha256:abc...",
  "hash": "sha256:def..."
}
```

---

## Audit Pipeline

### 1. Event Publication (fire-and-forget)

```go
publisher := audit.NewPublisher(natsConn)
publisher.Publish(ctx, audit.NewEvent("user.login", "success", tenantID, userID))
// Non-blocking — event goes to NATS immediately
```

### 2. Event Consumption (durable consumer)

The Audit Service runs a durable consumer that:
1. Reads events from `AUDIT_EVENTS` stream
2. Computes hash chain (`SHA256(prev_hash + event_data)`)
3. Stores event + hash in PostgreSQL
4. Acks to NATS (marks as processed)

If the Audit Service crashes, NATS re-delivers unacked events.

### 3. Hash Chain Verification

```sql
SELECT verify_hash_chain() FROM audit_events WHERE tenant_id = $1;
-- Returns: { verified: true, tampered_events: [] }
```

---

## Webhook Delivery

When an event matches a registered webhook, the Gateway delivers it via HTTP POST:

### Delivery Flow

```
NATS Event → Gateway Webhook Matcher → HTTP POST to endpoint
                                     → HMAC signature in header
                                     → Retry on failure (3 attempts, exponential backoff)
```

### HMAC Signature Verification

```python
import hmac, hashlib

def verify_webhook(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

### Retry Strategy

| Attempt | Delay | |
|---------|-------|--|
| 1 | Immediate | |
| 2 | 5 seconds | |
| 3 | 30 seconds | |
| 4 | 2 minutes (final) | Gives up, logs failure |

---

## Event Sourcing for Compliance

The audit pipeline supports compliance requirements:

| Framework | Requirement | How GGID Meets It |
|-----------|------------|-------------------|
| **PCI-DSS** | Tamper-evident audit trail | Hash chain |
| **HIPAA** | Immutable access logs | Append-only + hash chain |
| **SOC 2** | Change tracking | Full event history |
| **GDPR** | Data access accountability | Actor + resource tracking |

---

## Monitoring

NATS monitoring at `http://localhost:8222/streaming`:

- Stream info: messages pending, total, consumer count
- Consumer lag: unacked messages
- Throughput: messages/sec

---

*See: [Microservices](microservices.md) | [Data Flow](data-flow.md) | [Audit Compliance](../guides/audit-compliance.md)*

*Last updated: 2025-07-11*

```
