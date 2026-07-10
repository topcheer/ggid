# Audit Log Guide

Complete guide to GGID's audit logging system: event types, query API,
filtering, retention, export, compliance mapping, hash chain verification,
and NATS pipeline architecture.

> **See also**: [Audit Compliance](audit-compliance-guide.md) for
> regulatory requirements, [Webhook Events](webhook-events.md) for
> real-time event delivery.

---

## Table of Contents

- [Event Types](#event-types)
- [Query API](#query-api)
- [Filtering](#filtering)
- [Retention Policy](#retention-policy)
- [Export Formats](#export-formats)
- [Compliance Mapping](#compliance-mapping)
- [Hash Chain Verification](#hash-chain-verification)
- [NATS Pipeline Architecture](#nats-pipeline-architecture)

---

## Event Types

### Authentication Events

| Event | Trigger | Key Fields |
|------|---------|------------|
| `user.login` | Successful login | user_id, source_ip, method, mfa_used |
| `user.login.failed` | Failed login | username, source_ip, reason |
| `user.logout` | User logged out | user_id, session_id |
| `user.token.refreshed` | Token refreshed | user_id, client_id |
| `user.token.revoked` | Token revoked | user_id, reason |

### User Management Events

| Event | Trigger | Key Fields |
|------|---------|------------|
| `user.created` | User registered | user_id, username, source |
| `user.updated` | Profile modified | user_id, changes |
| `user.deleted` | User deleted | user_id, hard_delete |
| `user.activated` | Account activated | user_id |
| `user.deactivated` | Account deactivated | user_id, reason |
| `user.locked` | Account locked | user_id, failed_attempts |
| `user.unlocked` | Account unlocked | user_id, unlocked_by |

### Security Events

| Event | Trigger | Key Fields |
|------|---------|------------|
| `user.mfa.enabled` | MFA enrolled | user_id, method |
| `user.mfa.disabled` | MFA removed | user_id, method |
| `user.password.changed` | Password changed | user_id, mfa_verified |
| `user.password.reset` | Password reset | user_id, reset_method |
| `admin.impersonation.start` | Impersonation began | admin_id, target_id |
| `admin.impersonation.end` | Impersonation ended | admin_id, target_id |
| `security.token_reuse` | Refresh token reuse | user_id, family_id |

### Policy Events

| Event | Trigger | Key Fields |
|------|---------|------------|
| `role.assigned` | Role granted | user_id, role_id, scope |
| `role.revoked` | Role removed | user_id, role_id |
| `policy.created` | Policy created | policy_id, name |
| `policy.updated` | Policy modified | policy_id, changes |
| `policy.evaluated` | Access decision | user_id, resource, decision |

### Admin Events

| Event | Trigger | Key Fields |
|------|---------|------------|
| `admin.config.changed` | Tenant config modified | section, changes |
| `tenant.created` | New tenant | tenant_id, name |
| `tenant.suspended` | Tenant suspended | tenant_id, reason |
| `tenant.deleted` | Tenant deleted | tenant_id |

---

## Query API

### Basic Query

```bash
curl "https://iam.example.com/api/v1/audit/events?limit=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

```json
{
  "events": [
    {
      "id": "evt-uuid",
      "tenant_id": "00000000-...",
      "event_type": "user.login",
      "user_id": "550e8400-...",
      "source_ip": "192.168.1.50",
      "method": "password",
      "mfa_used": true,
      "timestamp": "2024-01-15T10:30:00Z",
      "hash": "sha256:abc123..."
    }
  ],
  "total": 15234,
  "page": 1,
  "page_size": 50
}
```

### Real-Time Stream (SSE)

```bash
curl -N "https://iam.example.com/api/v1/audit/events/stream" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: text/event-stream"
```

---

## Filtering

### Filter Parameters

| Parameter | Type | Example |
|-----------|------|---------|
| `event_type` | string | `user.login` |
| `user_id` | UUID | `550e8400-...` |
| `source_ip` | IP | `192.168.1.50` |
| `start` | timestamp | `2024-01-01T00:00:00Z` |
| `end` | timestamp | `2024-01-31T23:59:59Z` |
| `limit` | int | `50` (max 1000) |
| `page` | int | `1` |

### Combined Filters

```bash
curl "https://iam.example.com/api/v1/audit/events?
event_type=user.login.failed&
start=2024-01-15T00:00:00Z&
end=2024-01-15T23:59:59Z&
limit=100" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Retention Policy

| Storage Tier | Duration | Purpose |
|-------------|----------|---------|
| Hot (PostgreSQL) | 90 days | Fast query |
| Warm (archive table) | 1 year | Compliance queries |
| Cold (S3/ Glacier) | 7 years | Legal hold |

### Automated Archiving

```sql
-- Daily cron: move events older than 90 days to archive
INSERT INTO audit_events_archive
SELECT * FROM audit_events
WHERE created_at < NOW() - INTERVAL '90 days';

DELETE FROM audit_events
WHERE created_at < NOW() - INTERVAL '90 days';
```

### Compliance Retention

| Regulation | Min Retention | Notes |
|-----------|:------------:|-------|
| GDPR | Varies | Right to erasure may require deletion |
| SOC 2 | 1 year | Audit trail for access review |
| HIPAA | 6 years | PHI access logging |
| SOX | 7 years | Financial systems audit |
| PCI DSS | 1 year | Cardholder data access |

---

## Export Formats

### CSV Export

```bash
curl "https://iam.example.com/api/v1/audit/events/export?format=csv&start=2024-01-01T00:00:00Z&end=2024-01-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit-january.csv
```

### JSON Export

```bash
curl ".../export?format=json&start=2024-01-01T00:00:00Z&end=2024-01-31T23:59:59Z" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit-january.json
```

> Export limited to 100,000 events per request. Use date ranges to
> break up larger exports.

---

## Compliance Mapping

### GDPR

| Requirement | GGID Feature |
|------------|-------------|
| Article 30: Records of processing | Audit events log all data access |
| Article 32: Security of processing | Login/auth events, MFA tracking |
| Article 33: Breach notification | Real-time SSE stream + alerting |
| Right to erasure | `user.deleted` event cascades PII removal |

### SOC 2

| Control | GGID Audit Events |
|---------|-------------------|
| CC6.1: Logical access | `user.login`, `role.assigned` |
| CC6.6: Unauthorized access | `user.login.failed`, `user.locked` |
| CC7.2: Incident detection | `security.token_reuse`, SSE stream |
| CC8.1: Change management | `admin.config.changed`, `policy.updated` |

### HIPAA

| Requirement | GGID Feature |
|------------|-------------|
| §164.312(b): Audit controls | All events logged with user/IP/time |
| §164.312(c): Integrity | Hash chain verification |
| §164.308(a)(3): Workforce access | Role assignment/revocation audit |

---

## Hash Chain Verification

Each audit event includes a SHA-256 hash linking it to the previous event,
creating a tamper-evident chain.

### Chain Structure

```
Event 1: hash = SHA256(data_1 + "")
Event 2: hash = SHA256(data_2 + hash_1)
Event 3: hash = SHA256(data_3 + hash_2)
...
Event N: hash = SHA256(data_N + hash_{N-1})
```

### Verification

```bash
curl -X POST "https://iam.example.com/api/v1/audit/verify-chain" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{ "start": "2024-01-01T00:00:00Z", "end": "2024-01-31T23:59:59Z" }'
```

```json
{
  "verified": true,
  "events_checked": 15234,
  "first_hash": "sha256:abc...",
  "last_hash": "sha256:xyz...",
  "breaks": []
}
```

If tampering is detected:
```json
{
  "verified": false,
  "events_checked": 15234,
  "breaks": [
    { "event_id": "evt-uuid", "expected_hash": "...", "actual_hash": "..." }
  ]
}
```

---

## NATS Pipeline Architecture

```
┌─────────────┐     gRPC      ┌──────────┐    Publish    ┌───────────┐
│  Service     │──────────────►│  Audit    │─────────────►│   NATS    │
│  (auth,      │               │  Service  │              │ JetStream │
│   identity,  │               │ Publisher │              │           │
│   policy...) │               └──────────┘              └─────┬─────┘
│              │                                                 │
│              │     ┌──────────┐    Subscribe   ┌──────────────▼──────┐
│              │     │  Audit   │◄──────────────│  JetStream Consumer  │
│              │     │  Query   │               │  (durable, per-tenant)│
│              │     │  API     │               └──────────────────────┘
│              │     └────┬─────┘
└─────────────┘          │
                         ▼
                    ┌──────────┐
                    │PostgreSQL│
                    │(hot 90d) │
                    └──────────┘
```

### Pipeline Stages

1. **Service** generates audit event, publishes via gRPC to Audit Service
2. **Audit Publisher** validates event, publishes to NATS JetStream
3. **JetStream** persists event to stream (durable, replicated)
4. **Consumer** reads from stream, writes to PostgreSQL
5. **Query API** serves from PostgreSQL (fast indexed queries)

### JetStream Configuration

```yaml
nats:
  jetstream:
    stream: "AUDIT_EVENTS"
    subjects: ["audit.events.>"]
    retention: limits
    max_age: 2592000s    # 30 days
    replicas: 3
    consumer:
      durable: "audit-pg-consumer"
      filter_subject: "audit.events.>"
      ack_policy: explicit
      max_deliver: 3
```

### Reliability

| Feature | Behavior |
|---------|----------|
| At-least-once | Events delivered at least once (idempotent insert on event_id) |
| Ordering | Events ordered per-tenant via subject partitioning |
| Backpressure | JetStream handles flow control |
| Replay | Can replay from stream offset after DB recovery |
| Dead-letter | Failed events after 3 attempts → DLQ for manual inspection |

### Monitoring

```
ggid_audit_events_published_total
pggid_audit_events_stored_total
ggid_audit_publish_lag_seconds
ggid_audit_jetstream_pending
```
