# Database Security Guide

## PostgreSQL Hardening

### Connection Security

```ini
# postgresql.conf
ssl = on
ssl_cert_file = '/etc/postgresql/server.crt'
ssl_key_file = '/etc/postgresql/server.key'
password_encryption = scram-sha-256

# pg_hba.conf — only TLS connections from app subnet
hostssl ggid ggid_app 10.0.0.0/8 scram-sha-256
hostssl all  all       0.0.0.0/0   reject
```

### Row-Level Security (RLS)

GGID uses RLS for tenant isolation:

```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

Each request sets `app.tenant_id` via `SET LOCAL`:

```go
tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
```

### Connection Pooling

```yaml
pool:
  max_conns: 25
  min_conns: 5
  max_conn_lifetime: 30m
  max_conn_idle_time: 5m
```

## Encryption

### At Rest

| Layer | Method |
|-------|--------|
| Disk | LUKS / cloud KMS |
| Column-level | pgcrypto for PII fields |
| Backups | AES-256-GCM |

### In Transit

| Connection | Protocol |
|-----------|----------|
| App ↔ DB | TLS 1.3 |
| gRPC services | mTLS |
| External APIs | TLS 1.2+ |

## Access Control

### Least Privilege

| Role | Permissions |
|------|------------|
| `ggid_app` | SELECT, INSERT, UPDATE, DELETE on app tables |
| `ggid_migrate` | DDL on app tables |
| `ggid_readonly` | SELECT only (analytics) |
| `ggid_backup` | pg_dump, no table access |

### Credential Rotation

```bash
# Rotate DB password quarterly
ALTER ROLE ggid_app WITH PASSWORD 'new-secure-password';
# Update Kubernetes secret, rolling restart
```

## Audit Logging

```sql
CREATE TABLE db_audit_log (
  id BIGSERIAL PRIMARY KEY,
  actor UUID,
  action TEXT,
  table_name TEXT,
  record_id UUID,
  changes JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Backup & Recovery

| Type | Frequency | Retention |
|------|-----------|-----------|
| Full backup | Daily | 30 days |
| WAL archive | Continuous | 7 days |
| PITR | Available | Up to 7 days |
| Monthly snapshot | Monthly | 12 months |

## Monitoring

- Connection count alerts (>80% of max)
- Slow query log (>1s)
- Failed login attempts (>10/min)
- RLS policy violations (should be 0)
- Replication lag (>5s)

## See Also

- [Secrets Management](secrets-management.md)
- [HSM Integration](hsm-integration.md)
- [Disaster Recovery](disaster-recovery.md)
