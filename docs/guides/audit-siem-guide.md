# Audit & SIEM Integration Guide

This guide covers GGID's audit query API, compliance report generation, SIEM forwarder configuration, and hash chain verification.

> **Related**: [SIEM Integration](siem-integration.md), [SIEM Connector Design](../research/siem-connector-design.md)

## Overview

GGID provides a comprehensive audit subsystem:

| Component | Location | Purpose |
|-----------|----------|---------|
| Audit service | `services/audit/` | REST API for querying events |
| NATS publisher | `pkg/audit/publisher.go` | Async event publishing |
| SIEM forwarder | `pkg/audit/siem_forwarder.go` | Forward events to Splunk/Datadog/Elasticsearch |
| Hash chain | `services/audit/internal/service/hash_chain.go` | Tamper-evident integrity |
| Compliance reports | `services/audit/internal/compliance/` | SOC2, HIPAA, GDPR reports |
| Retention | `services/audit/internal/retention/retention.go` | Automated cleanup |

## Audit Query API

### List Events

```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/events?event_type=user.login&start_date=2025-01-01&end_date=2025-01-31&page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Filters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `event_type` | string | Filter by event type (e.g., `user.login`, `role.assign`) |
| `actor_id` | UUID | Filter by actor (user who performed action) |
| `resource_id` | UUID | Filter by target resource |
| `start_date` | date | Events after this date |
| `end_date` | date | Events before this date |
| `page` | int | Page number (1-based) |
| `page_size` | int | Results per page (max 100) |

### Event Types

| Category | Event Types |
|----------|-------------|
| Authentication | `user.login`, `user.logout`, `user.register`, `auth.mfa_verify`, `auth.mfa_enable` |
| User management | `user.create`, `user.update`, `user.delete`, `user.lock`, `user.unlock` |
| Roles | `role.create`, `role.delete`, `role.assign`, `role.revoke` |
| Organizations | `org.create`, `org.delete`, `member.add`, `member.remove` |
| Policies | `policy.create`, `policy.update`, `policy.evaluate` |
| OAuth | `oauth.consent`, `oauth.token_issue`, `oauth.token_revoke` |
| Agents | `agent.register`, `agent.token_exchange`, `agent.suspend` |
| Admin | `admin.config_change`, `admin.key_rotation`, `admin.retention_change` |

### Export Events

```bash
# CSV export
curl -X GET "https://api.ggid.example.com/api/v1/audit/export?format=csv&start_date=2025-01-01" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit_export.csv

# JSON export
curl -X GET "https://api.ggid.example.com/api/v1/audit/export?format=json" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit_export.json
```

## Compliance Reports

### Generate Report

```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/compliance/report?type=soc2&start_date=2025-01-01&end_date=2025-03-31" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Supported Report Types

| Type | Framework | Sections |
|------|-----------|----------|
| `soc2` | SOC 2 Type II | Access control, change management, operations |
| `hipaa` | HIPAA Security Rule | Access logs, audit controls, integrity |
| `gdpr` | GDPR | Data access, consent, retention, breach |

### Report Structure

```json
{
  "type": "soc2",
  "period": {"start": "2025-01-01", "end": "2025-03-31"},
  "summary": {
    "total_events": 154823,
    "unique_users": 342,
    "failed_logins": 1287,
    "privilege_changes": 45
  },
  "sections": [
    {
      "name": "Access Control",
      "controls": ["CC6.1", "CC6.2", "CC6.3"],
      "findings": [...]
    }
  ]
}
```

## SIEM Forwarder Configuration

### Supported SIEM Providers

| Provider | Format | Endpoint |
|----------|--------|----------|
| Splunk | CEF (Common Event Format) | HEC URL |
| Datadog | JSON logs | Datadog API |
| Elasticsearch | ECS (Elastic Common Schema) | `_bulk` API |

### Configuration

```go
import "github.com/ggid/ggid/pkg/audit"

config := audit.SIEMConfig{
    Provider:    audit.SIEMProviderSplunk,
    Endpoint:    "https://splunk.example.com:8088/services/collector",
    APIKey:      "splunk-hec-token",
    IndexName:   "ggid-audit",
    BatchSize:   100,
    FlushInterval: 5 * time.Second,
    MaxRetries:  3,
}

forwarder := audit.NewSIEMForwarder(config)
forwarder.Start(ctx)
defer forwarder.Stop()
```

### Environment Variables

```yaml
SIEM_PROVIDER: splunk              # splunk | datadog | elasticsearch
SIEM_ENDPOINT: https://splunk.example.com:8088/services/collector
SIEM_API_KEY: <hec-token>
SIEM_INDEX: ggid-audit
SIEM_BATCH_SIZE: 100
SIEM_FLUSH_INTERVAL: 5s
SIEM_MAX_RETRIES: 3
```

### CEF Output Format (Splunk)

```
CEF:0|GGID|IAM|1.0|100|User Login|3|act=login suser=alice@example.com \
src=192.168.1.50 rt=Jan 24 2025 14:30:00 UTC msg=Successful login \
cs1Label=TenantID cs1=00000000-0000-0000-0000-000000000001
```

### Datadog Output Format

```json
{
  "service": "ggid-audit",
  "timestamp": "2025-01-24T14:30:00Z",
  "ddsource": "ggid",
  "ddtags": ["env:prod", "tenant:default"],
  "message": {
    "event_type": "user.login",
    "actor": "alice@example.com",
    "source_ip": "192.168.1.50",
    "result": "success"
  }
}
```

## Hash Chain Verification

### How It Works

GGID implements a cryptographic hash chain for audit integrity:

```
Event₁ → hash₁ = SHA256(event₁_data)
Event₂ → hash₂ = SHA256(hash₁ + event₂_data)
Event₃ → hash₃ = SHA256(hash₂ + event₃_data)
...
```

Any tampering with an event breaks the chain.

### Verify Integrity

```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/integrity/verify" \
  -H "Authorization: Bearer $TOKEN"

# Response
{
  "valid": true,
  "events_verified": 154823,
  "chain_head_hash": "sha256:abc123...",
  "first_event": "2025-01-01T00:00:00Z",
  "last_event": "2025-01-31T23:59:59Z"
}
```

### Hash Chain Implementation

```go
// hash_chain.go
func ComputeHash(prevHash string, event AuditEvent) string {
    data := prevHash + event.Type + event.ActorID + event.Timestamp.Format(time.RFC3339)
    return sha256Hex(data)
}
```

## Alert Rules

### Create Alert Rule

```bash
curl -X POST "https://api.ggid.example.com/api/v1/audit/alerts/rules" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Failed login burst",
    "condition": "event_type=user.login AND result=failed COUNT > 10 IN 5m",
    "action": "email",
    "recipients": ["security@company.com"],
    "enabled": true
  }'
```

### Common Alert Rules

| Name | Condition | Severity |
|------|-----------|----------|
| Failed login burst | `user.login failed > 10 in 5m` | High |
| Privilege escalation | `role.assign AND role=admin` | Critical |
| Off-hours admin | `admin.* AND time BETWEEN 22:00-06:00` | Medium |
| New device login | `user.login AND new_device=true` | Medium |
| Impossible travel | `user.login AND impossible_travel=true` | High |
| Mass export | `audit.export COUNT > 5 in 1h` | Medium |

## Retention Management

```bash
# Get current retention policy
curl -X GET "https://api.ggid.example.com/api/v1/audit/retention" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Update retention
curl -X PUT "https://api.ggid.example.com/api/v1/audit/retention" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"max_age_days": 365, "max_count": 1000000}'

# Manual cleanup
curl -X POST "https://api.ggid.example.com/api/v1/audit/retention/apply" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# → {"deleted_by_age": 15423, "deleted_by_count": 0, "total_deleted": 15423}
```

See [Data Retention Policy Guide](data-retention-policy.md) for details.

## Monitoring Dashboard

Key metrics to monitor:

| Metric | Alert Threshold | SIEM Query |
|--------|----------------|------------|
| Events/sec | > 1000 | `count() GROUP BY 1m` |
| Failed logins | > 50 in 5m | `event_type=user.login result=failed` |
| Privilege changes | > 10 in 1h | `event_type=role.*` |
| Hash chain breaks | > 0 | `integrity.valid=false` |
| SIEM forward lag | > 60s | `forwarder.lag_seconds` |
| Export requests | > 10 in 1h | `event_type=audit.export` |

## See Also

- [SIEM Integration](siem-integration.md)
- [Data Retention Policy](data-retention-policy.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Compliance & Access Reviews](access-reviews.md)
