# NATS Event Catalog

Complete catalog of every NATS subject in GGID: payload schemas, consumer
patterns, idempotency, and ordering guarantees.

> **See also**: [Webhook Events](webhook-events.md) for HTTP delivery,
> [Audit Guide](audit-guide.md) for audit-specific events.

---

## Table of Contents

- [Subject Naming Convention](#subject-naming-convention)
- [User Events](#user-events)
- [Authentication Events](#authentication-events)
- [Role Events](#role-events)
- [Organization Events](#organization-events)
- [OAuth Events](#oauth-events)
- [Policy Events](#policy-events)
- [Audit Events](#audit-events)
- [Consumer Patterns](#consumer-patterns)
- [Idempotency](#idempotency)
- [Ordering Guarantees](#ordering-guarantees)

---

## Subject Naming Convention

```
{domain}.{category}.{event_type}.{tenant_id}
```

| Part | Example | Description |
|------|---------|-------------|
| domain | `ggid` | Fixed prefix |
| category | `user`, `auth`, `role` | Event category |
| event_type | `created`, `updated`, `deleted` | Specific event |
| tenant_id | UUID | Tenant routing key |

### Wildcard Subscriptions

```
ggid.user.>               # All user events
ggid.auth.>               # All auth events
ggid.*.>                  # All events (use sparingly)
ggid.user.created.>       # All user.created across tenants
ggid.>.{tenant_id}        # All events for specific tenant
```

---

## User Events

### ggid.user.created.{tenant_id}

```json
{
  "event_id": "evt-uuid",
  "event_type": "user.created",
  "timestamp": "2024-01-15T10:30:00Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "jane.doe",
    "email": "jane.doe@example.com",
    "display_name": "Jane Doe",
    "status": "active",
    "source": "api",
    "created_by": "admin-uuid"
  }
}
```

### ggid.user.updated.{tenant_id}

```json
{
  "event_type": "user.updated",
  "data": {
    "user_id": "550e8400-...",
    "changes": {
      "display_name": {"old": "Jane Doe", "new": "Jane Smith"},
      "department": {"old": null, "new": "Engineering"}
    },
    "updated_by": "admin-uuid"
  }
}
```

### ggid.user.deleted.{tenant_id}

```json
{
  "event_type": "user.deleted",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "hard_delete": false,
    "deleted_by": "admin-uuid"
  }
}
```

### ggid.user.activated / ggid.user.deactivated

```json
{
  "event_type": "user.deactivated",
  "data": {
    "user_id": "550e8400-...",
    "reason": "offboarding",
    "sessions_revoked": 3,
    "deactivated_by": "admin-uuid"
  }
}
```

### ggid.user.locked / ggid.user.unlocked

```json
{
  "event_type": "user.locked",
  "data": {
    "user_id": "550e8400-...",
    "failed_attempts": 5,
    "source_ip": "192.168.1.50",
    "auto_unlock_at": "2024-01-15T11:00:00Z"
  }
}
```

---

## Authentication Events

### ggid.auth.login.{tenant_id}

```json
{
  "event_type": "user.login",
  "data": {
    "user_id": "550e8400-...",
    "username": "jane.doe",
    "source_ip": "192.168.1.50",
    "method": "password",
    "mfa_used": true,
    "mfa_method": "totp",
    "session_id": "sess-uuid",
    "device_type": "desktop"
  }
}
```

### ggid.auth.login_failed.{tenant_id}

```json
{
  "event_type": "user.login.failed",
  "data": {
    "username": "jane.doe",
    "source_ip": "10.0.0.15",
    "reason": "invalid_password",
    "attempt_number": 3
  }
}
```

### ggid.auth.logout.{tenant_id}

```json
{
  "event_type": "user.logout",
  "data": {
    "user_id": "550e8400-...",
    "session_id": "sess-uuid"
  }
}
```

### ggid.auth.password_changed / ggid.auth.password_reset

```json
{
  "event_type": "user.password.reset",
  "data": {
    "user_id": "550e8400-...",
    "reset_method": "email_link",
    "reset_by": "admin-uuid"
  }
}
```

### ggid.auth.mfa_enabled / ggid.auth.mfa_disabled

```json
{
  "event_type": "user.mfa.enabled",
  "data": {
    "user_id": "550e8400-...",
    "method": "webauthn",
    "device_name": "MacBook Touch ID"
  }
}
```

---

## Role Events

### ggid.role.assigned.{tenant_id}

```json
{
  "event_type": "role.assigned",
  "data": {
    "user_id": "550e8400-...",
    "role_id": "role-uuid",
    "role_name": "admin",
    "scope": "tenant",
    "assigned_by": "admin-uuid",
    "expires_at": "2025-12-31T23:59:59Z"
  }
}
```

### ggid.role.revoked.{tenant_id}

```json
{
  "event_type": "role.revoked",
  "data": {
    "user_id": "550e8400-...",
    "role_id": "role-uuid",
    "role_name": "admin",
    "revoked_by": "admin-uuid",
    "reason": "role_review"
  }
}
```

---

## Organization Events

### ggid.org.member_added.{tenant_id}

```json
{
  "event_type": "org.member.added",
  "data": {
    "org_id": "org-uuid",
    "org_name": "Engineering",
    "user_id": "550e8400-...",
    "added_by": "admin-uuid"
  }
}
```

### ggid.org.member_removed.{tenant_id}

```json
{
  "event_type": "org.member.removed",
  "data": {
    "org_id": "org-uuid",
    "user_id": "550e8400-...",
    "removed_by": "admin-uuid"
  }
}
```

---

## OAuth Events

### ggid.oauth.token_issued.{tenant_id}

```json
{
  "event_type": "oauth.token.issued",
  "data": {
    "client_id": "web-app",
    "user_id": "550e8400-...",
    "grant_type": "authorization_code",
    "scopes": ["openid", "profile"],
    "expires_in": 900
  }
}
```

### ggid.oauth.token_refreshed.{tenant_id}

```json
{
  "event_type": "oauth.token.refreshed",
  "data": {
    "client_id": "web-app",
    "user_id": "550e8400-...",
    "family_id": "fam-abc-123"
  }
}
```

### ggid.oauth.token_reuse_detected.{tenant_id}

```json
{
  "event_type": "oauth.token.reuse_detected",
  "data": {
    "client_id": "web-app",
    "user_id": "550e8400-...",
    "family_id": "fam-abc-123",
    "revoked_count": 3,
    "severity": "critical"
  }
}
```

---

## Policy Events

### ggid.policy.evaluated.{tenant_id}

```json
{
  "event_type": "policy.evaluated",
  "data": {
    "user_id": "550e8400-...",
    "resource": "api:/v1/users",
    "action": "read",
    "decision": "allow",
    "policy_id": "policy-uuid",
    "evaluation_time_ms": 2
  }
}
```

---

## Audit Events

### ggid.audit.query.{tenant_id}

```json
{
  "event_type": "audit.query",
  "data": {
    "queried_by": "admin-uuid",
    "filters": {"event_type": "user.login"},
    "result_count": 50
  }
}
```

### ggid.audit.config_changed.{tenant_id}

```json
{
  "event_type": "admin.config.changed",
  "data": {
    "changed_by": "admin-uuid",
    "section": "password_policy",
    "changes": {"min_length": {"old": 8, "new": 12}}
  }
}
```

---

## Consumer Patterns

### Durable Consumer (Recommended)

```go
sub, err := js.Subscribe(
    "ggid.user.>",            // Subject filter
    handleUserEvent,           // Handler function
    nats.Durable("user-sync"), // Durable name (survives restart)
    nats.DeliverAll(),         // Deliver all historical events
    nats.ManualAck(),          // Explicit acknowledgment
    nats.MaxDeliver(3),        // Retry 3 times on failure
)
```

### Queue Group (Load Balanced)

```go
// Multiple consumers share workload
sub, _ := js.QueueSubscribe(
    "ggid.audit.>",
    "audit-processors",       // Queue group name
    handleAuditEvent,
    nats.Durable("audit-processor"),
)
```

### Filtered Consumer

```go
// Only user.created events
sub, _ := js.Subscribe(
    "ggid.user.created.>",
    handleUserCreated,
    nats.Durable("provisioning"),
)
```

### Per-Tenant Consumer

```go
// Only events for specific tenant
tenantID := "00000000-0000-0000-0000-000000000001"
sub, _ := js.Subscribe(
    fmt.Sprintf("ggid.>.%s", tenantID),
    handleTenantEvent,
    nats.Durable(fmt.Sprintf("tenant-%s", tenantID)),
)
```

---

## Idempotency

Every event includes a unique `event_id` (UUID). Consumers MUST handle
duplicates by tracking processed event IDs:

```go
var processed sync.Map

func handleEvent(msg *nats.Msg) {
    var event Event
    json.Unmarshal(msg.Data, &event)

    if _, ok := processed.Load(event.EventID); ok {
        msg.Ack()  // Already processed, acknowledge and skip
        return
    }

    processed.Store(event.EventID, true)
    processEvent(event)
    msg.Ack()
}
```

For production use, store processed event IDs in Redis or PostgreSQL
instead of in-memory maps.

---

## Ordering Guarantees

| Guarantee | Behavior |
|-----------|----------|
| Per-tenant ordering | Events for same tenant delivered in order |
| Per-user ordering | Events for same user_id delivered in order |
| Cross-tenant | NOT guaranteed — may arrive out of order |
| Cross-category | NOT guaranteed — user events may interleave with role events |
| Exactly-once | At-least-once delivery; deduplicate via event_id |
| Retry ordering | Retries do not block subsequent new events |

### JetStream Configuration

```yaml
nats:
  jetstream:
    stream: "GGID_EVENTS"
    subjects: ["ggid.>"]
    retention: limits
    max_age: 2592000s       # 30 days
    replicas: 3             # HA replication
    consumer:
      ack_policy: explicit
      max_deliver: 3
      ack_wait: 30s
```
