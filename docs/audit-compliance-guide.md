# GGID Audit & Compliance Guide

Complete guide for audit logging, data retention, and compliance mapping for
GDPR, SOC 2, and HIPAA requirements.

---

## Table of Contents

- [Audit Log Format](#audit-log-format)
- [Event Types](#event-types)
- [Retention Policy](#retention-policy)
- [GDPR Compliance](#gdpr-compliance)
- [SOC 2 Compliance](#soc-2-compliance)
- [HIPAA Compliance](#hipaa-compliance)
- [Export and SIEM Integration](#export-and-siem-integration)
- [Tamper Detection](#tamper-detection)

---

## Audit Log Format

GGID records every authentication, authorization, and administrative action.
Events are stored in PostgreSQL and streamed via NATS JetStream.

### Event Schema

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-07-15T10:30:45.123Z",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "event_type": "auth.login",
  "actor": {
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "username": "john.doe",
    "ip_address": "192.168.1.100",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15)"
  },
  "resource": {
    "type": "user",
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "name": "john.doe"
  },
  "result": "success",
  "metadata": {
    "auth_method": "password",
    "mfa_used": false,
    "session_id": "sess-abc-123"
  },
  "hash": "sha256:previous_event_hash"
}
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `event_id` | UUID | Yes | Unique event identifier |
| `timestamp` | ISO 8601 | Yes | UTC with milliseconds |
| `tenant_id` | UUID | Yes | Tenant context |
| `event_type` | string | Yes | Event category (see below) |
| `actor.user_id` | UUID | Yes (auth events) | Who performed the action |
| `actor.ip_address` | string | Yes | Source IP |
| `actor.user_agent` | string | No | Browser/client |
| `resource.type` | string | No | What was acted upon |
| `resource.id` | UUID | No | Resource identifier |
| `result` | string | Yes | `success`, `failure`, `denied` |
| `metadata` | object | No | Event-specific details |
| `hash` | string | Yes | Hash chain for tamper detection |

---

## Event Types

### Authentication Events

| Event Type | Trigger | Audit Fields |
|------------|---------|--------------|
| `auth.login` | User login | `auth_method`, `mfa_used` |
| `auth.login_failed` | Failed login attempt | `failure_reason` |
| `auth.logout` | User logout | `session_id` |
| `auth.token_issued` | JWT issued | `grant_type`, `scopes` |
| `auth.token_refreshed` | Token refreshed | `old_jti`, `new_jti` |
| `auth.token_revoked` | Token revoked | `jti`, `reason` |
| `auth.mfa_challenge` | MFA prompted | `mfa_method` |
| `auth.mfa_verified` | MFA successful | `mfa_method` |
| `auth.mfa_failed` | MFA failed | `mfa_method`, `failure_reason` |
| `auth.account_locked` | Account locked | `attempt_count` |
| `auth.password_changed` | Password changed | `policy_version` |
| `auth.password_reset` | Password reset | `via` (email/admin) |

### User Management Events

| Event Type | Trigger |
|------------|---------|
| `user.created` | User registered or provisioned |
| `user.updated` | User profile changed |
| `user.deleted` | User deleted |
| `user.suspended` | User suspended |
| `user.reactivated` | User reactivated |
| `user.role_assigned` | Role assigned |
| `user.role_revoked` | Role removed |

### Administrative Events

| Event Type | Trigger |
|------------|---------|
| `admin.config_changed` | System configuration modified |
| `admin.tenant_created` | New tenant created |
| `admin.tenant_deleted` | Tenant deleted |
| `admin.key_rotated` | JWT signing key rotated |
| `admin.policy_updated` | RBAC/ABAC policy changed |

### SCIM Events

| Event Type | Trigger |
|------------|---------|
| `scim.user.provisioned` | User created via SCIM |
| `scim.user.updated` | User updated via SCIM |
| `scim.user.deprovisioned` | User deactivated via SCIM |
| `scim.group.synced` | Group membership synced |

---

## Retention Policy

### Default Retention

| Event Category | Retention | Storage | Reason |
|----------------|-----------|---------|--------|
| Authentication | 365 days | PostgreSQL + Archive | Security forensics |
| User management | 7 years | PostgreSQL + Archive | Compliance audit |
| Administrative | 7 years | PostgreSQL + Archive | Change tracking |
| SCIM | 90 days | PostgreSQL | Provisioning tracking |

### Configuring Retention

```bash
# Set retention via API (days)
curl -X PUT $API/api/v1/settings/audit/retention \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "auth_events_days": 365,
        "user_events_days": 2555,
        "admin_events_days": 2555,
        "scim_events_days": 90
    }'
```

### Automated Archival

```bash
# Archive old events to cold storage (S3/GCS)
# Run as a daily cron job
curl -X POST $API/api/v1/audit/archive \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "older_than_days": 90,
        "destination": "s3://ggid-audit-logs/2024/"
    }'
```

Archived events are exported as JSONL (JSON Lines) files:

```jsonl
{"event_id":"...","timestamp":"2024-04-15T...","event_type":"auth.login",...}
{"event_id":"...","timestamp":"2024-04-15T...","event_type":"user.created",...}
```

---

## GDPR Compliance

### Requirements Mapping

| GDPR Article | Requirement | GGID Feature |
|--------------|-------------|--------------|
| Art. 6 | Lawful basis for processing | Audit log records consent |
| Art. 7 | Consent management | `user.consent_given` event |
| Art. 15 | Right of access | User data export API |
| Art. 16 | Right to rectification | User self-update |
| Art. 17 | Right to erasure | User deletion with audit trail |
| Art. 20 | Data portability | Export user data as JSON |
| Art. 30 | Records of processing | Audit log = processing record |
| Art. 32 | Security of processing | Encryption, RLS, rate limiting |
| Art. 33 | Breach notification | Alert rules for anomalies |
| Art. 35 | Data protection impact | PII redaction in logs |

### Data Subject Access Request (DSAR)

```bash
# Export all user data (Art. 15, 20)
curl $API/api/v1/users/$USER_ID/export \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -o user-data-export.json

# Export includes:
# - User profile
# - Credentials (hashed only)
# - Roles and groups
# - Audit events involving the user
# - Sessions (active and expired)
```

### Right to Erasure (Art. 17)

```bash
# Delete user and all associated data
curl -X DELETE $API/api/v1/users/$USER_ID?purge=true \
    -H "Authorization: Bearer $ADMIN_TOKEN"

# This:
# 1. Deletes user profile, credentials, sessions
# 2. Anonymizes audit logs (keeps events for compliance, removes PII)
# 3. Records a user.deleted event
# 4. Revokes all active tokens
```

### PII Redaction

Audit logs automatically redact PII fields:

```json
{
  "event_type": "auth.login",
  "actor": {
    "user_id": "550e8400-...",
    "username": "[REDACTED]",
    "ip_address": "192.168.1.100"
  }
}
```

Configure redacted fields:

```bash
curl -X PUT $API/api/v1/settings/audit/redaction \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "redact_fields": ["email", "username", "ip_address"],
        "hash_fields": ["user_id"],
        "retain_days_before_redaction": 30
    }'
```

---

## SOC 2 Compliance

### Trust Criteria Mapping

| SOC 2 Criteria | Requirement | GGID Feature |
|----------------|-------------|--------------|
| CC6.1 | Logical access controls | RBAC + ABAC, MFA, JWT |
| CC6.2 | User authentication | Password policy, MFA, WebAuthn |
| CC6.3 | Access authorization | Role-based access, per-tenant isolation |
| CC6.6 | Boundary protection | API Gateway, rate limiting, WAF |
| CC7.1 | System monitoring | Audit log, Prometheus metrics, tracing |
| CC7.2 | Anomaly detection | Alert rules (login spike, brute force) |
| CC7.3 | Incident response | Audit trail for forensics |
| CC7.4 | Backups | DB backups, WAL archiving |
| CC8.1 | Change management | Audit log for config changes |

### Audit Trail Completeness

SOC 2 requires a complete audit trail. GGID ensures:

1. **Every action is logged** — No bypass path exists
2. **Logs are immutable** — Append-only table, hash chaining
3. **Timestamps are accurate** — NTP-synced, UTC
4. **Actor is identified** — Every event includes `actor.user_id`
5. **Changes are tracked** — Before/after values in metadata

### SOC 2 Audit Report Query

```bash
# Generate SOC 2 audit report for a date range
curl "$API/api/v1/audit/events?\
start_date=2024-01-01T00:00:00Z&\
end_date=2024-03-31T23:59:59Z&\
event_type=admin.*,user.role_assigned,auth.login_failed&\
format=csv" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -o soc2-audit-q1.csv
```

---

## HIPAA Compliance

### Requirements Mapping

| HIPAA Rule | Requirement | GGID Feature |
|------------|-------------|--------------|
| §164.312(a)(1) | Access control | RBAC, per-tenant RLS, JWT |
| §164.312(a)(2)(i) | Unique user identification | UUID-based user IDs |
| §164.312(a)(2)(iii) | Automatic logoff | Short-lived JWT (15 min), session timeout |
| §164.312(b) | Audit controls | Comprehensive audit logging |
| §164.312(c)(1) | Integrity | Hash-chained audit log |
| §164.312(d) | Person or entity authentication | MFA, WebAuthn, LDAP |
| §164.312(e)(1) | Transmission security | TLS 1.2+ everywhere, mTLS internal |
| §164.312(e)(2)(ii) | Encryption | AES-256-GCM at rest, TLS in transit |

### PHI Considerations

GGID itself does not store Protected Health Information (PHI). However:

- If user emails contain PHI (e.g., `patient123@hospital.com`), they are treated as PHI
- Audit logs may contain PHI in metadata fields
- Configure PII redaction for PHI-containing fields

```bash
# Enable PHI redaction mode
curl -X PUT $API/api/v1/settings/audit/hipaa \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "phi_mode": true,
        "redact_email": true,
        "redact_ip_after_days": 90,
        "retention_days": 2190
    }'
```

### BAA (Business Associate Agreement)

GGID is self-hosted, so the covered entity controls all data. No BAA is
required with GGID as a vendor, since GGID is open-source software deployed
on your own infrastructure.

---

## Export and SIEM Integration

### Real-Time Stream (SSE)

```bash
# Subscribe to audit event stream
curl -N "$API/api/v1/audit/stream?event_type=auth.*" \
    -H "Authorization: Bearer $ADMIN_TOKEN"
```

### NATS JetStream Export

GGID publishes all audit events to NATS:

```bash
# Subscribe from external SIEM
nats sub "ggid.events.>.audit" --server nats://nats:4222
```

### Splunk Integration

```yaml
# Splunk Universal Forwarder inputs.conf
[monitor:///var/log/ggid/audit.log]
index = ggid_audit
sourcetype = ggid:audit
```

### Datadog Integration

```yaml
# datadog.yaml — Log collection
logs:
  - type: docker
    service: ggid
    source: ggid
    log_processing_rules:
      - type: mask_sequences
        name: redact_email
        pattern: '"email"\s*:\s*"[^"]+"'
        replace_placeholder: '"email":"[REDACTED]"'
```

### ELK Stack

```bash
# Filebeat config for shipping GGID logs to Elasticsearch
filebeat.inputs:
  - type: log
    paths:
      - /var/log/ggid/audit.log
    json.keys_under_root: true
    processors:
      - decode_json_fields:
          fields: ["message"]
```

---

## Tamper Detection

### Hash Chaining

Each audit event includes a SHA-256 hash of the previous event's hash,
creating an immutable chain:

```
Event 1: hash_1 = SHA256(event_1_data + "")
Event 2: hash_2 = SHA256(event_2_data + hash_1)
Event 3: hash_3 = SHA256(event_3_data + hash_2)
```

### Verifying Integrity

```bash
# Verify audit log integrity for a date range
curl -X POST "$API/api/v1/audit/verify-integrity" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "start_date": "2024-01-01T00:00:00Z",
        "end_date": "2024-07-01T00:00:00Z"
    }'

# Response:
# {
#   "total_events": 1542837,
#   "verified": 1542837,
#   "tampered": 0,
#   "status": "intact"
# }
```

If any event is modified or deleted, the hash chain breaks and verification
reports the tampered event.

---

## References

- Audit API — REST endpoints
- [Observability Guide](./observability-guide.md) — Monitoring
- [Security Whitepaper](./security-whitepaper.md) — Threat model
- [Webhooks Guide](./webhooks-guide.md) — Event subscriptions
