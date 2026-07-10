# GGID Database Optimization Guide

Performance tuning for PostgreSQL in GGID deployments.

---

## pgxpool Connection Pool Configuration

### Recommended Settings

```go
config, _ := pgxpool.ParseConfig(databaseURL)
config.MaxConns = 25          // (CPU cores * 2) + disk_spindles
config.MinConns = 5           // keep warm connections
config.MaxConnLifetime = 1 * time.Hour
config.MaxConnIdleTime = 30 * time.Minute
config.HealthCheckPeriod = 30 * time.Second
pool, _ := pgxpool.NewWithConfig(ctx, config)
```

### Pool Sizing Formula

```
Optimal pool size = ((core_count × 2) + effective_disk_count)
```

| Deployment | Cores | Recommended MaxConns |
|-----------|:-----:|:----:|
| Dev (2 vCPU) | 2 | 10 |
| Small (4 vCPU, SSD) | 4 | 20 |
| Medium (8 vCPU, SSD) | 8 | 25 |
| Large (16 vCPU, NVMe) | 16 | 50 |

### Multiple Services

Each GGID service has its own pool. Total connections:

```
Total = Gateway(0) + Auth(25) + Identity(15) + OAuth(15) + Policy(15) + Org(15) + Audit(15)
      ≈ 100 connections
```

PostgreSQL `max_connections` should be ≥ 150 (with headroom).

---

## Index Strategy

### Rule: tenant_id First

Every multi-tenant index starts with `tenant_id`:

```sql
-- Correct: tenant_id first (RLS uses this)
CREATE INDEX idx_users_tenant_email ON users (tenant_id, email);

-- Wrong: email first (RLS can't use efficiently)
CREATE INDEX idx_users_email ON users (email);
```

### Core Indexes

```sql
-- Users
CREATE INDEX idx_users_tenant_username ON users (tenant_id, username);
CREATE INDEX idx_users_tenant_email    ON users (tenant_id, email);
CREATE INDEX idx_users_tenant_status   ON users (tenant_id, status);

-- Roles + Permissions
CREATE INDEX idx_roles_tenant_key       ON roles (tenant_id, key);
CREATE INDEX idx_role_perm_tenant_role  ON role_permissions (tenant_id, role_id);
CREATE INDEX idx_role_perm_tenant_res   ON role_permissions (tenant_id, resource);
CREATE INDEX idx_user_roles_tenant_user ON user_roles (tenant_id, user_id);

-- Organizations (LTREE)
CREATE INDEX idx_orgs_tenant_parent ON organizations (tenant_id, parent_id);
CREATE INDEX idx_orgs_path          ON organizations USING gist (path);

-- Policies
CREATE INDEX idx_policies_tenant_name ON policies (tenant_id, name);

-- Audit (time-range heavy)
CREATE INDEX idx_audit_tenant_time   ON audit_events (tenant_id, created_at DESC);
CREATE INDEX idx_audit_tenant_action ON audit_events (tenant_id, action);
CREATE INDEX idx_audit_tenant_actor  ON audit_events (tenant_id, actor_id);
```

### Partial Indexes

For frequently-filtered subsets:

```sql
CREATE INDEX idx_users_active ON users (tenant_id, email)
    WHERE status = 'active';
```

### EXPLAIN ANALYZE Workflow

```sql
-- Always verify queries use indexes
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM users WHERE tenant_id = '...' AND email = 'john@example.com';

-- Look for:
-- ✓ Index Scan (index hit)
-- ✗ Seq Scan (full table scan — bad)
-- ✓ Index Only Scan (covering index — best)
```

---

## Query Optimization

### Batch Inserts

```sql
-- Slow: one row at a time
INSERT INTO audit_events (...) VALUES (...);  -- x1000

-- Fast: bulk insert
INSERT INTO audit_events (...)
VALUES (...), (...), (...), (...);  -- single query
```

### N+1 Query Prevention

```go
// Bad: N+1 (one query per user)
for _, userID := range userIDs {
    user := getUser(userID)  // query per iteration
}

// Good: batch fetch
users := getUsersByIDs(userIDs)  // single query: WHERE id = ANY($1)
```

### JSONB Query Optimization

```sql
-- Use GIN index for JSONB containment queries
CREATE INDEX idx_policies_conditions ON policies USING gin (conditions);

-- Efficient JSONB query
SELECT * FROM policies WHERE conditions @> '{"department": "engineering"}';
```

### COUNT Optimization

Large tables (audit_events) should avoid `COUNT(*)`:

```sql
-- Slow on millions of rows
SELECT COUNT(*) FROM audit_events WHERE tenant_id = '...';

-- Fast: use estimated count
SELECT reltuples::bigint FROM pg_class WHERE relname = 'audit_events';

-- Fast: filtered estimate
EXPLAIN SELECT * FROM audit_events WHERE tenant_id = '...' AND created_at > NOW() - INTERVAL '1 day';
-- Parse rows estimate from EXPLAIN output
```

---

## RLS Performance Impact

### Overhead

RLS adds a `current_setting('app.tenant_id')::uuid` filter to every query:

```sql
-- Without RLS:
SELECT * FROM users WHERE email = 'x';
-- Uses index on email

-- With RLS (transparent):
SELECT * FROM users WHERE email = 'x'
    AND tenant_id = current_setting('app.tenant_id')::uuid;
-- Needs index on (tenant_id, email)
```

**Measured overhead:** 3-8% per query when indexes are properly configured.

### Optimization

1. **Always include tenant_id in indexes** (see above)
2. **Set tenant context once per transaction** (`SET LOCAL`)
3. **Avoid SELECT *** — query only needed columns
4. **Use covering indexes** for hot paths:

```sql
CREATE INDEX idx_users_covering ON users (tenant_id, email)
    INCLUDE (username, status, display_name);
```

---

## Partitioning

### When to Partition

| Table | Threshold | Strategy |
|-------|-----------|----------|
| `audit_events` | > 10M rows | Range partition by `created_at` (monthly) |
| `users` | > 100M rows | Hash partition by `id` (rarely needed) |

### Audit Events Partitioning

```sql
CREATE TABLE audit_events (
    id          UUID DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- ...
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE audit_events_2024_07 PARTITION OF audit_events
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');

CREATE TABLE audit_events_2024_08 PARTITION OF audit_events
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
```

### Automated Partition Management

Use `pg_partman` extension for automatic creation/dropping:

```sql
CREATE EXTENSION pg_partman;

SELECT partman.create_parent(
    'public.audit_events',
    'created_at',
    'native',
    'monthly'
);
```

### Partition Pruning

PostgreSQL automatically prunes partitions for time-range queries:

```sql
-- Only scans 2024_07 partition
SELECT * FROM audit_events
WHERE created_at >= '2024-07-01' AND created_at < '2024-08-01';
```

---

## PostgreSQL Server Configuration

### memory tuning

```ini
# postgresql.conf (production)
shared_buffers = 4GB              # 25% of total RAM
effective_cache_size = 12GB       # 75% of total RAM
work_mem = 64MB                   # per-sort/hash
maintenance_work_mem = 512MB      # for VACUUM/CREATE INDEX
wal_buffers = 16MB
```

### WAL / Checkpoint

```ini
max_wal_size = 4GB
min_wal_size = 1GB
checkpoint_completion_target = 0.9
wal_compression = on
```

### Autovacuum

```ini
autovacuum = on
autovacuum_naptime = 30s
autovacuum_vacuum_scale_factor = 0.05    # vacuum at 5% dead tuples
autovacuum_analyze_scale_factor = 0.02   # analyze at 2% changes
```

### Connection Limits

```ini
max_connections = 200             # GGID needs ~100 + headroom
superuser_reserved_connections = 5
```

---

## Monitoring

### Key Metrics

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity;

-- Connections per database
SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;

-- Slow queries (> 1s)
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Table bloat
SELECT relname, n_live_tup, n_dead_tup,
       round(n_dead_tup::float / n_live_tup * 100, 2) AS bloat_pct
FROM pg_stat_user_tables
WHERE n_live_tup > 1000
ORDER BY bloat_pct DESC;
```

### Index Usage

```sql
-- Unused indexes (candidates for removal)
SELECT indexrelname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```
