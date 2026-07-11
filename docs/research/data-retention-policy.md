# Data Retention Policy

> Audit log retention, GDPR right-to-be-forgotten, and automated cleanup design for GGID.

---

## Compliance Requirements

| Regulation | Audit Retention | PII Deletion | Data Minimization |
|-----------|----------------|-------------|-------------------|
| **GDPR** | Not specified (reasonable) | 30 days from request | Only collect what's needed |
| **PCI-DSS** | 1 year minimum | N/A (cardholder data) | Quarterly review |
| **HIPAA** | 6 years | Patient request | Minimum necessary |
| **SOC 2** | 12 months | Policy-based | Annual review |
| **CCPA** | N/A | 45 days from request | Consumer choice |

---

## Audit Log Retention

### Configuration

```bash
# Default retention (days)
AUDIT_RETENTION_DAYS=365

# NATS stream retention
NATS_RETENTION=168h  # 7 days
```

### Cleanup Job

The Audit Service runs a daily cron that:

1. Queries `audit_events` where `timestamp < NOW() - retention_days`
2. Archives to cold storage (S3) before deletion
3. Deletes in batches of 1000 to avoid locking
4. Recalculates hash chain after deletion
5. Publishes `audit.retention_cleanup` event

```sql
-- Batch deletion with hash chain repair
BEGIN;
  DELETE FROM audit_events
  WHERE tenant_id = $1
    AND timestamp < NOW() - INTERVAL '365 days'
    AND id IN (
      SELECT id FROM audit_events
      WHERE tenant_id = $1
        AND timestamp < NOW() - INTERVAL '365 days'
      LIMIT 1000
    );
  -- Recalculate chain for remaining events
  PERFORM recalculate_hash_chain($1);
COMMIT;
```

---

## GDPR Right-to-be-Forgotten

### API Endpoint

```
DELETE /api/v1/users/{user_id}?gdpr=true
```

When `gdpr=true`, GGID performs **hard deletion** across all services:

### Deletion Pipeline

```
1. Identity Service:    Anonymize user record (username→"deleted", email→null, delete credentials)
2. Auth Service:        Revoke all sessions, delete refresh tokens, remove MFA devices
3. Audit Service:       Anonymize actor_id in events (usr_abc→"anonymized"), keep events for compliance
4. Policy Service:      Remove role assignments
5. OAuth Service:       Revoke tokens, delete consent records
```

**Note:** Audit events are **anonymized**, not deleted — regulatory compliance requires keeping the audit trail.

### Implementation

```go
func (s *IdentityService) GDPRDelete(ctx context.Context, userID uuid.UUID) error {
    // 1. Anonymize user
    user, _ := s.repo.Get(ctx, userID)
    user.Username = "anonymized_" + user.ID.String()[:8]
    user.Email = ""
    user.DisplayName = ""
    user.Status = "gdpr_deleted"
    user.DeletedAt = time.Now()
    s.repo.Update(ctx, user)

    // 2. Revoke sessions
    s.auth.RevokeAllSessions(ctx, userID)

    // 3. Anonymize audit events
    s.audit.AnonymizeActor(ctx, userID)

    // 4. Remove role assignments
    s.policy.RemoveAllRoles(ctx, userID)

    // 5. Audit the deletion itself
    s.audit.Publish(ctx, audit.NewEvent("user.gdpr_delete",
        "completed", user.TenantID, userID))

    return nil
}
```

### Response

```json
{
  "status": "deleted",
  "user_id": "usr_abc123",
  "anonymized": true,
  "audit_events_anonymized": 15423,
  "sessions_revoked": 3,
  "roles_removed": 2,
  "completed_at": "2025-07-11T12:00:00Z"
}
```

---

## Data Classification

| Data Type | Storage | Retention | Anonymizable |
|-----------|---------|-----------|-------------|
| User profile | PostgreSQL | Deleted on GDPR request | Yes |
| Credentials (hashes) | PostgreSQL | Deleted with user | Yes |
| Audit events | PostgreSQL + NATS | 365 days (configurable) | Anonymized only |
| Session data | Redis | 24 hours | Auto-expire |
| OAuth tokens | Redis + PostgreSQL | Token lifetime | Deleted with user |
| SCIM sync logs | PostgreSQL | 90 days | Anonymized |

---

## Proposed `retention_rules` Table

```sql
CREATE TABLE retention_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    data_type   VARCHAR(50) NOT NULL,    -- audit_events, user_sessions, scim_logs
    retention_days INTEGER NOT NULL DEFAULT 365,
    action      VARCHAR(20) NOT NULL DEFAULT 'delete',  -- delete, archive, anonymize
    archive_target VARCHAR(100),          -- s3://bucket, gs://bucket
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, data_type)
);
```

### Sample Rules

```sql
INSERT INTO retention_rules (tenant_id, data_type, retention_days, action) VALUES
('00000000-...', 'audit_events', 365, 'archive'),
('00000000-...', 'user_sessions', 1, 'delete'),
('00000000-...', 'scim_logs', 90, 'delete'),
('00000000-...', 'oauth_tokens', 7, 'delete');
```

---

## Monitoring

```promql
# Events pending cleanup
ggid_retention_pending{tenant="acme"}

# Cleanup job duration
rate(ggid_retention_cleanup_duration_seconds_sum[1h])

# GDPR deletion requests
increase(ggid_gdpr_deletions_total[24h])
```

Alert: `ggid_retention_cleanup_failed` → page on-call if cleanup job fails 2 consecutive runs.

---

*See: [Audit Compliance](../guides/audit-compliance.md) | [Event-Driven Architecture](../architecture/event-driven.md) | [Security Overview](../architecture/security-overview.md)*

*Last updated: 2025-07-11*
