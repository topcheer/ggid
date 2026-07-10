# GGID Audit Compliance Guide

SOC 2, ISO 27001, and GDPR audit trail configuration, log retention policy,
immutable storage, and SIEM integration for GGID.

---

## Table of Contents

- [Overview](#overview)
- [Audit Event Schema](#audit-event-schema)
- [Compliance Framework Mapping](#compliance-framework-mapping)
- [Log Retention Policy](#log-retention-policy)
- [Immutable Storage](#immutable-storage)
- [SIEM Integration](#siem-integration)
- [Tamper Detection](#tamper-detection)

---

## Overview

GGID maintains a comprehensive, tamper-evident audit log of all security-relevant
events. Audit events are stored in PostgreSQL and streamed in real-time via NATS
JetStream for downstream SIEM consumption.

---

## Audit Event Schema

```json
{
  "id": 1234567890,
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "auth.login",
  "data": {
    "method": "password",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0...",
    "mfa_used": false,
    "success": true
  },
  "ip_address": "192.168.1.100",
  "prev_hash": "a1b2c3d4...",
  "hash": "e5f6g7h8...",
  "created_at": "2024-07-15T10:30:00Z"
}
```

### Event Types

| Category | Event Types |
|----------|------------|
| Authentication | `auth.login`, `auth.logout`, `auth.token_issued`, `auth.token_revoked`, `auth.mfa_challenge`, `auth.mfa_verify`, `auth.password_changed`, `auth.password_reset` |
| User Management | `user.created`, `user.updated`, `user.deleted`, `user.suspended`, `user.activated` |
| Role/Policy | `role.created`, `role.assigned`, `role.revoked`, `policy.created`, `policy.updated` |
| Organization | `org.created`, `org.member_added`, `org.member_removed` |
| Admin | `admin.config_changed`, `admin.api_key_created`, `admin.api_key_revoked` |

---

## Compliance Framework Mapping

### SOC 2 (Trust Services Criteria)

| SOC 2 Criteria | GGID Audit Coverage |
|----------------|-------------------|
| CC6.1 (Logical Access) | All login/logout events with IP, UA |
| CC6.2 (User Provisioning) | `user.created`, `role.assigned`, `role.revoked` |
| CC6.3 (Authorization) | Policy evaluation logged on access denial |
| CC6.6 (Boundary Protection) | Gateway logs all API requests |
| CC7.1 (Detection) | Real-time event stream via NATS |
| CC7.2 (Monitoring) | Prometheus metrics + alerting |
| CC8.1 (Change Management) | `admin.config_changed` events |

### ISO 27001

| ISO Control | GGID Audit Coverage |
|-------------|-------------------|
| A.9 (Access Control) | Auth events, role changes |
| A.12 (Operations Security) | System events, config changes |
| A.16 (Incident Management) | Failed logins, lockouts, rate limit hits |
| A.18 (Compliance) | Immutable audit trail, retention enforcement |

### GDPR

| GDPR Article | GGID Feature |
|-------------|-------------|
| Art. 6 (Lawful Basis) | Consent tracked in user metadata |
| Art. 15 (Right of Access) | DSAR export API (`GET /users/{id}/export`) |
| Art. 17 (Right to Erasure) | Anonymization API (preserves audit integrity) |
| Art. 25 (Privacy by Design) | PII redaction in logs, data minimization |
| Art. 30 (Records of Processing) | Audit log = processing records |
| Art. 32 (Security) | Encryption at rest, TLS in transit |
| Art. 33 (Breach Notification) | Alert rules for suspicious patterns |

---

## Log Retention Policy

| Tier | Retention | Storage |
|------|-----------|---------|
| Free | 30 days | PostgreSQL |
| Starter | 90 days | PostgreSQL + archival |
| Pro | 365 days | PostgreSQL + S3 archival |
| Enterprise | 7 years | PostgreSQL + WORM S3 |

### Configure Retention

```bash
curl -X PUT $API/api/v1/settings/audit-retention \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "hot_retention_days": 90,
    "cold_storage": "s3://ggid-audit-archive",
    "cold_retention_years": 7,
    "auto_archive": true
  }'
```

### Automated Archival

Events older than `hot_retention_days` are exported to cold storage and
removed from PostgreSQL. The hash chain is preserved by retaining the latest
hash from each batch.

---

## Immutable Storage

### WORM (Write Once Read Many) S3

For enterprise compliance, configure audit events to archive to WORM S3:

```bash
# AWS S3 Object Lock (compliance mode)
aws s3api create-bucket \
  --bucket ggid-audit-worm \
  --object-lock-enabled-for-bucket

aws s3api put-object-lock-configuration \
  --bucket ggid-audit-worm \
  --object-lock-configuration '{
    "ObjectLockEnabled": "Enabled",
    "Rule": {
      "DefaultRetention": {
        "Mode": "COMPLIANCE",
        "Years": 7
      }
    }
  }'
```

### Hash Chaining

Each audit event includes a SHA-256 hash of its content + the previous event's
hash, forming an unbreakable chain:

```
Event 1: hash = SHA256(data_1 + "")
Event 2: hash = SHA256(data_2 + hash_1)
Event 3: hash = SHA256(data_3 + hash_2)
...
```

Tampering with any event breaks the chain. Verification:

```bash
curl -X POST $API/api/v1/audit/verify-chain \
  -H "Authorization: Bearer $JWT" \
  -d '{"from": "2024-07-01", "to": "2024-07-31"}'
# {"verified": true, "events_checked": 456789}
```

---

## SIEM Integration

### Splunk

Configure GGID to forward audit events to Splunk via HTTP Event Collector:

```bash
curl -X PUT $API/api/v1/settings/siem \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "provider": "splunk",
    "hec_url": "https://splunk.corp.local:8088/services/collector",
    "hec_token": "YOUR_HEC_TOKEN",
    "events": ["auth.*", "user.*", "admin.*"],
    "batch_size": 100
  }'
```

### Elasticsearch / ELK

```bash
curl -X PUT $API/api/v1/settings/siem \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "provider": "elasticsearch",
    "url": "https://es.corp.local:9200",
    "index": "ggid-audit",
    "api_key": "YOUR_ES_API_KEY"
  }'
```

### Datadog

```bash
curl -X PUT $API/api/v1/settings/siem \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "provider": "datadog",
    "api_key": "YOUR_DD_API_KEY",
    "site": "datadoghq.com"
  }'
```

### SSE Stream (Custom Integration)

```bash
# Stream events in real-time via Server-Sent Events
curl -N $API/api/v1/audit/stream \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### NATS JetStream (Direct Subscription)

```go
// Subscribe to audit events directly from NATS
js, _ := jetstream.New(nc, jetstream.WithStream("GGID_EVENTS"))
cons, _ := js.CreateOrUpdateConsumer(ctx, "GGID_EVENTS",
    jetstream.ConsumerConfig{
        DurableName:   "siem-consumer",
        FilterSubject: "ggid.events.00000000-0000-0000-0000-000000000001.auth.>",
    })

cons.Consume(func(msg jetstream.Msg) {
    // Forward to your SIEM
    siemClient.Send(string(msg.Data()))
    msg.Ack()
})
```

---

## Tamper Detection

### Automated Integrity Checks

```bash
# Schedule daily integrity verification
curl -X POST $API/api/v1/audit/verify-chain \
  -H "Authorization: Bearer $JWT" \
  -d '{"from": "yesterday", "to": "today"}'
```

### Anomaly Detection Alert Rules

```yaml
# Alert on brute force attempts
- alert: BruteForceDetected
  expr: count(rate(ggid_auth_login_total{result="failure"}[5m])) by (ip) > 10
  for: 1m
  annotations:
    summary: "Possible brute force from {{ $labels.ip }}"

# Alert on admin config changes
- alert: AdminConfigChange
  expr: rate(ggid_audit_events{event_type="admin.config_changed"}[1m]) > 0
  for: 0s
  annotations:
    summary: "Admin configuration changed"
```

---

## References

- [Observability Guide](./observability.md) — Metrics and monitoring
- [Security Hardening](./security-hardening.md) — Production security
- [Database Schema](./database-schema.md) — audit_events table
