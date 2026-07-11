# Multi-Database Deployment Guide

This guide covers PostgreSQL tuning, connection pooling, Row-Level Security (RLS) best practices, and read replica configuration for GGID production deployments.

## PostgreSQL Configuration

### Recommended PostgreSQL 16 Settings

```ini
# postgresql.conf — Production Settings

# Connection
max_connections = 200
shared_buffers = 2GB              # 25% of total RAM
work_mem = 64MB
maintenance_work_mem = 512MB

# WAL / Replication
wal_level = logical               # Required for logical replication
max_wal_senders = 10
max_replication_slots = 10
wal_keep_size = 4GB
checkpoint_completion_target = 0.9
checkpoint_timeout = 15min

# Query Planner
random_page_cost = 1.1            # SSD-optimized
effective_cache_size = 6GB        # 75% of total RAM
default_statistics_target = 200

# Autovacuum
autovacuum = on
autovacuum_max_workers = 6
autovacuum_naptime = 30s
autovacuum_vacuum_scale_factor = 0.05

# Logging
log_min_duration_statement = 250  # Log queries > 250ms
log_checkpoints = on
log_connections = off              # High volume in production
log_lock_waits = on
```

### Database Initialization

```sql
-- Create GGID database and roles
CREATE DATABASE ggid
  WITH ENCODING 'UTF8'
  LC_COLLATE 'en_US.UTF-8'
  LC_CTYPE 'en_US.UTF-8'
  TEMPLATE template0;

-- Application role (read-write)
CREATE ROLE ggid_app WITH LOGIN PASSWORD 'strong-password' CONNECTION LIMIT 100;

-- Read-only role (for analytics, replicas)
CREATE ROLE ggid_readonly WITH LOGIN PASSWORD 'strong-password';

-- Migration role
CREATE ROLE ggid_migrate WITH LOGIN PASSWORD 'strong-password';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE ggid TO ggid_migrate;
GRANT CONNECT ON DATABASE ggid TO ggid_app, ggid_readonly;
```

## Connection Pooling with pgxpool

### Why Connection Pooling?

Each GGID service (identity, auth, policy, org, audit) maintains its own database connection pool. Without pooling:

- Connection storms during traffic spikes
- PostgreSQL `max_connections` exhaustion
- 100-300ms overhead per new connection

### pgxpool Configuration

GGID services use `pgxpool` (part of jackc/pgx v5). Configure via environment variables or connection string:

```go
config, _ := pgxpool.ParseConfig(connString)
config.MaxConns = 25              // Max connections in pool
config.MinConns = 5               // Min idle connections
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = time.Minute
```

### Recommended Pool Sizes

| Service       | MaxConns | MinConns | Rationale                        |
|---------------|----------|----------|----------------------------------|
| gateway       | 5        | 1        | No direct DB access (JWT only)   |
| identity      | 25       | 5        | High read/write (user CRUD)      |
| auth          | 15       | 3        | Login bursts, session writes     |
| policy        | 20       | 5        | Policy evaluation queries        |
| org           | 15       | 3        | Moderate write volume            |
| audit         | 30       | 10       | High write volume (event insert) |
| **Total**     | **110**  | **27**   | Stays within max_connections=200 |

### PgBouncer (External Pooler)

For multi-instance deployments, add PgBouncer between services and PostgreSQL:

```ini
# pgbouncer.ini
[databases]
ggid = host=postgres dbname=ggid

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
pool_mode = transaction
max_client_conn = 500
default_pool_size = 25
reserve_pool_size = 5
reserve_pool_timeout = 3
server_idle_timeout = 300
```

**Transaction mode** is recommended. Session mode is needed if using:
- Prepared statements (server-side)
- Temporary tables
- `SET` commands

## Row-Level Security (RLS)

GGID uses PostgreSQL RLS for multi-tenant isolation. Every table has `tenant_id` and RLS policies enforce data isolation at the database level.

### RLS Pattern

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- App role bypasses RLS only for migrations
ALTER ROLE ggid_migrate BYPASSRLS;

-- Policy: users can only see their tenant's data
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.current_tenant')::uuid);
```

### Setting Tenant Context

GGID sets the tenant context per-request using `SET LOCAL`:

```go
// Before executing queries
_, err := pool.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = '%s'", tenantID))
```

> **Note**: PostgreSQL does not support parameterized `SET LOCAL` with `$1`. GGID uses `fmt.Sprintf` with validated UUID input to prevent SQL injection.

### RLS Best Practices

1. **Always FORCE RLS**: Even table owners should be subject to RLS
   ```sql
   ALTER TABLE users FORCE ROW LEVEL SECURITY;
   ```

2. **Index tenant_id**: Create indexes on `tenant_id` for query performance
   ```sql
   CREATE INDEX idx_users_tenant ON users(tenant_id);
   ```

3. **Never grant BYPASSRLS to app role**: Only migration/admin roles should bypass

4. **Test RLS isolation**: Verify cross-tenant queries return zero rows
   ```sql
   SET app.current_tenant = 'tenant-a';
   SELECT count(*) FROM users;  -- Should only return tenant-a users
   ```

5. **Audit RLS policy changes**: Track `CREATE POLICY` / `DROP POLICY` in audit log

## Read Replica Configuration

### Streaming Replication Setup

#### Primary (Master)

```ini
# postgresql.conf
wal_level = logical
max_wal_senders = 10
hot_standby = on
```

```sql
-- Create replication user
CREATE ROLE repl WITH REPLICATION LOGIN PASSWORD 'repl-password';
```

```conf
# pg_hba.conf
host replication repl 0.0.0.0/0 md5
```

#### Replica (Standby)

```bash
# Base backup
pg_basebackup -h primary-host -U repl -D /var/lib/postgresql/data -Fp -Xs -P -R

# postgresql.auto.conf (auto-generated by -R flag)
primary_conninfo = 'host=primary-host port=5432 user=repl'
```

### Read-Only Routing in GGID

For read-heavy workloads (audit queries, user listings), route reads to replicas:

```go
// Primary pool (writes + reads needing strong consistency)
primaryPool, _ := pgxpool.New(ctx, primaryConnString)

// Replica pool (eventual consistency reads)
replicaPool, _ := pgxpool.New(ctx, replicaConnString)

// Route based on operation type
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
    return s.replicaPool.QueryRow(ctx, "SELECT ...", id)  // Read from replica
}

func (s *Service) CreateUser(ctx context.Context, u *User) error {
    return s.primaryPool.QueryRow(ctx, "INSERT ...", u)   // Write to primary
}
```

### Replication Lag Monitoring

```sql
-- On primary: check replication status
SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn,
       (sent_lsn - replay_lsn) AS replication_lag
FROM pg_stat_replication;

-- On replica: check how far behind
SELECT now() - pg_last_xact_replay_timestamp() AS replication_delay;
```

Alert if replication lag exceeds 5 seconds.

## Performance Tuning Checklist

| Area                | Setting                          | Target                    |
|---------------------|----------------------------------|---------------------------|
| shared_buffers      | 25% of RAM                       | 2-8 GB                    |
| effective_cache_size| 75% of RAM                       | 6-24 GB                   |
| work_mem            | Per-sort memory                  | 32-128 MB                 |
| max_connections     | Total across all services        | 200-500                   |
| pgxpool MaxConns    | Per-service pool                 | 5-30                      |
| autovacuum          | Enabled, aggressive              | naptime=30s               |
| Index on tenant_id  | Every tenant-scoped table        | B-tree or hash            |
| Replication lag     | Alert threshold                  | < 5 seconds               |

## Backup Strategy

```bash
# Physical backup (pg_basebackup)
pg_basebackup -h primary -U backup -D /backups/$(date +%Y%m%d) -Fp -Xs -P -z

# Logical backup (pg_dump, per-tenant)
pg_dump -h primary -U ggid_app -Fc ggid -t users --where "tenant_id='UUID'" > users_backup.dump

# WAL archiving (continuous)
archive_mode = on
archive_command = 'aws s3 cp %p s3://ggid-wal-archive/%f'
```

## See Also

- [Docker Deployment](docker-deployment.md)
- [Backup and Restore](backup-restore.md)
- [Production Readiness Checklist](production-readiness-checklist.md)
- [Performance Tuning](performance-tuning.md)
