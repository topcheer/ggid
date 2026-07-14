# Data Retention Policy Guide

This guide covers configuring audit log retention, GDPR right-to-erasure workflows, and automated data cleanup in GGID.

> **Related**: [Data Retention Policy Research](../research/data-retention-policy.md) (compliance framework analysis)

## Overview

GGID implements a structured retention system in `services/audit/internal/retention/retention.go` that enforces time-based and count-based deletion of audit events. This guide explains how to configure and operate it.

## Compliance Requirements

| Regulation | Audit Retention | PII Erasure | Data Minimization |
|------------|----------------|-------------|-------------------|
| **GDPR** | Not specified (reasonable) | 30 days from request | Only collect what's needed |
| **PCI-DSS** | 1 year minimum | N/A | Quarterly review |
| **HIPAA** | 6 years | Patient request | Minimum necessary |
| **SOC 2** | 12 months | Policy-based | Annual review |
| **CCPA/CPRA** | N/A | 45 days from request | Consumer choice |
| **SOX** | 7 years | N/A | Financial controls |
| **ISO 27001** | 12 months minimum | Policy-based | Need-to-know |

## Audit Log Retention

### Configuration

GGID's `RetentionPolicy` struct supports two deletion modes:

```yaml
# Environment variables
AUDIT_RETENTION_DAYS: 365          # Delete events older than N days (0 = disabled)
AUDIT_RETENTION_MAX_COUNT: 1000000 # Keep at most N events (0 = unlimited)
```

### How It Works

The retention system runs in two phases:

1. **Delete by age**: Removes events with `created_at < NOW() - retention_days`
2. **Delete excess by count**: If total events exceed max_count, removes oldest

```go
type RetentionPolicy struct {
    Enabled   bool          // Master switch
    MaxAge    time.Duration // Delete events older than this
    MaxCount  int64         // Keep at most this many events
}
```

### Running Retention

Retention is applied via a periodic scheduler or manual API call:

```bash
# Manual trigger
curl -X POST https://api.ggid.example.com/api/v1/audit/retention/apply \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response
{
  "deleted_by_age": 15423,
  "deleted_by_count": 0,
  "total_deleted": 15423
}
```

### Scheduling Automated Cleanup

```yaml
# Kubernetes CronJob — runs daily at 2 AM
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ggid-retention-cleanup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: ggid-audit:latest
            command: ["./audit", "--run-retention"]
            env:
            - name: AUDIT_RETENTION_DAYS
              value: "365"
```

### NATS JetStream Retention

Audit events flow through NATS JetStream before persistence. Configure stream retention separately:

```yaml
# NATS stream config
nats:
  stream:
    name: "AUDIT"
    retention: "limits"          # limits | interest | work-queue
    max_age: "168h"              # 7 days in NATS
    max_msgs: 1000000
    max_bytes: 10737418240       # 10 GB
    storage: "file"              # file-based persistence
```

## GDPR Right to Erasure (Article 17)

### Erasure Workflow

```
Data Subject Request → Admin Console → Anonymize PII → Retain Audit (Legal Basis) → Confirm
```

### What Must Be Deleted

| Data Type | Erasure Required | Exception |
|-----------|-----------------|-----------|
| User profile (name, email, phone) | Yes | None |
| Credentials (password hash) | Yes | None |
| MFA devices/secrets | Yes | None |
| OAuth consents | Yes | None |
| User sessions | Yes | None |
| Audit log entries | No | Legitimate interest (legal compliance) |
| Transaction records | No | Contractual/legal obligation |

### Executing Erasure

```bash
# Anonymize a user (GDPR Article 17)
curl -X DELETE https://api.ggid.example.com/api/v1/users/$USER_ID/gdpr-erase \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# This:
# 1. Anonymizes PII fields (email → "erased@anonymous", name → "ERASED")
# 2. Deletes MFA devices
# 3. Revokes all sessions (jti blacklist)
# 4. Revokes all OAuth tokens
# 5. Keeps audit log with anonymized actor reference
```

### PII Obfuscation in Audit Logs

Audit events containing PII are obfuscated before persistence:

```go
// pii.Obfuscate masks email, phone, SSN in audit payloads
masked := pii.Obfuscate(event.Data)
// "user@example.com" → "u***@e******.com"
// "+1234567890"      → "+1234567***"
```

## Data Classification

| Classification | Examples | Retention | Encryption |
|----------------|----------|-----------|------------|
| **Public** | Org name, public keys | Indefinite | TLS in transit |
| **Internal** | Role names, policy rules | 2 years | TLS + at-rest |
| **Confidential** | User email, audit logs | 1-7 years | TLS + at-rest + RLS |
| **Restricted** | Passwords, MFA secrets | Until account deletion | TLS + at-rest + encryption |

## Database-Level Cleanup

### Scheduled Vacuum

```sql
-- After retention deletion, reclaim space
VACUUM ANALYZE audit_events;

-- For large deletions, use VACUUM FULL (locks table)
VACUUM FULL audit_events;  -- Run during maintenance window only
```

### Partition by Time

For high-volume audit tables, use time-based partitioning:

```sql
CREATE TABLE audit_events (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    data JSONB
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Drop old partitions (instant, no VACUUM needed)
DROP TABLE audit_events_2024_01;
```

## Monitoring & Alerting

### Key Metrics

| Metric | Alert Threshold |
|--------|----------------|
| Audit events per day | > 1M (capacity planning) |
| Retention deletion count | = 0 for 3 days (retention broken?) |
| Audit table size | > 100 GB |
| GDPR erasure queue | > 0 pending > 30 days |
| Oldest unprocessed event | > retention_days (backlog) |

## Retention Audit Checklist

- [ ] Retention period documented per regulation
- [ ] Automated retention job running (daily)
- [ ] GDPR erasure endpoint tested
- [ ] PII obfuscation verified in audit events
- [ ] Database partition strategy implemented
- [ ] Backup retention aligns with data retention
- [ ] Legal hold mechanism documented
- [ ] Retention compliance reviewed annually

## See Also

- [Data Retention Policy Research](../research/data-retention-policy.md)
- [Backup and Restore](backup-restore.md)
- Audit Configuration
- [Security Audit Checklist](security-audit-checklist.md)
