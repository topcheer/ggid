# Audit Query Optimization

PostgreSQL partitioning, BRIN indexes, materialized views, cursor pagination, query plan analysis, and common anti-patterns.

## PostgreSQL Partitioning

### Range Partition by Month

```sql
CREATE TABLE audit_events (
    id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    actor_user_id UUID,
    action TEXT NOT NULL,
    result TEXT,
    details JSONB,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Monthly partitions
CREATE TABLE audit_events_2025_01 PARTITION OF audit_events
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE audit_events_2025_02 PARTITION OF audit_events
  FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
```

### Automated Partition Creation

```go
// Cron runs on 25th of each month — creates next month's partition
func ensureNextMonthPartition() error {
    next := firstDayOfNextMonth()
    partitionName := "audit_events_" + next.Format("2006_01")
    
    sql := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_events
        FOR VALUES FROM ('%s') TO ('%s')
    `, partitionName, next.Format("2006-01-02"), next.AddDate(0, 1, 0).Format("2006-01-02"))
    
    return db.Exec(sql).Error
}
```

### Benefits

| Benefit | Impact |
|---------|--------|
| Partition pruning | Queries with date range scan only relevant partitions |
| Easier archival | Detach old partition instead of DELETE |
| Smaller indexes | Each partition has its own indexes |
| Parallel scans | Multiple partitions scanned in parallel |

## BRIN Indexes

For time-series data, BRIN (Block Range Index) is far smaller than B-tree:

```sql
-- BRIN index on created_at (natural ordering)
CREATE INDEX idx_audit_brin_time ON audit_events USING BRIN (created_at)
  WITH (pages_per_range = 128);

-- B-tree for high-cardinality lookups
CREATE INDEX idx_audit_user_time ON audit_events_2025_01 (actor_user_id, created_at DESC);
CREATE INDEX idx_audit_action_time ON audit_events_2025_01 (action, created_at DESC);
CREATE INDEX idx_audit_tenant_time ON audit_events_2025_01 (tenant_id, created_at DESC);
```

### Index Size Comparison

| Index Type | Size (1M rows) | Query Speed | Best For |
|-----------|---------------|-------------|----------|
| B-tree | 120MB | Fast (exact match) | user_id, action |
| BRIN | 0.5MB | Good (range scan) | created_at |
| GIN (JSONB) | 80MB | Variable | details field queries |
| Hash | 60MB | Fast (exact) | event_id |

## Materialized Views for Aggregations

```sql
-- Pre-compute daily aggregations
CREATE MATERIALIZED VIEW audit_daily_stats AS
SELECT
    tenant_id,
    action,
    result,
    DATE(created_at) as day,
    COUNT(*) as event_count,
    COUNT(DISTINCT actor_user_id) as unique_users
FROM audit_events
WHERE created_at > NOW() - INTERVAL '90 days'
GROUP BY tenant_id, action, result, DATE(created_at)
WITH DATA;

-- Refresh hourly
REFRESH MATERIALIZED VIEW CONCURRENTLY audit_daily_stats;
```

### Index on Materialized View

```sql
CREATE INDEX idx_audit_stats_day ON audit_daily_stats (tenant_id, day DESC);
CREATE INDEX idx_audit_stats_action ON audit_daily_stats (tenant_id, action, day DESC);
```

### Aggregation Query

```bash
# Fast: serves from materialized view
GET /api/v1/audit/aggregations?group_by=action&from=2025-01-01&to=2025-01-31
# → SELECT * FROM audit_daily_stats WHERE day BETWEEN ... (instant)
```

## Cursor Pagination

### Offset vs Cursor Performance

```sql
-- BAD: Offset pagination (slow at high offsets)
SELECT * FROM audit_events
ORDER BY created_at DESC
LIMIT 100 OFFSET 100000;
-- → Scans 100,100 rows → 500ms

-- GOOD: Cursor pagination (consistent speed)
SELECT * FROM audit_events
WHERE created_at < '2025-01-15T10:30:00Z'
ORDER BY created_at DESC
LIMIT 100;
-- → Uses index → 2ms regardless of position
```

### Cursor Implementation

```go
func QueryWithCursor(cursor string, limit int) ([]Event, string, error) {
    query := db.Model(&AuditEvent{}).Limit(limit).Order("created_at DESC")
    
    if cursor != "" {
        // Decode cursor → timestamp of last item
        after := decodeCursor(cursor)
        query = query.Where("created_at < ?", after)
    }
    
    var events []AuditEvent
    query.Find(&events)
    
    var nextCursor string
    if len(events) == limit {
        nextCursor = encodeCursor(events[len(events)-1].CreatedAt)
    }
    
    return events, nextCursor, nil
}
```

## Query Plan Analysis

### EXPLAIN ANALYZE

```sql
EXPLAIN ANALYZE
SELECT * FROM audit_events
WHERE tenant_id = 'uuid'
  AND created_at >= '2025-01-01'
  AND created_at < '2025-02-01'
  AND action = 'user.login'
ORDER BY created_at DESC
LIMIT 50;

-- Good plan: Index Scan using idx_audit_tenant_time
-- Bad plan: Seq Scan on audit_events (full table scan)
```

### Red Flags

| Plan Output | Problem | Fix |
|-------------|---------|-----|
| Seq Scan | No index used | Add index on filter columns |
| Sort (external) | Sorting in memory exceeded | Add index on ORDER BY column |
| Nested Loop | Join without index | Add join index |
| Bitmap Heap Scan (large) | Many matching rows | Narrow WHERE clause |

## Common Anti-Patterns

### 1. No Date Filter

```sql
-- BAD: Scans all partitions
SELECT * FROM audit_events WHERE user_id = 'uuid';
-- Fix: Always include date range
SELECT * FROM audit_events
WHERE user_id = 'uuid' AND created_at > NOW() - INTERVAL '7 days';
```

### 2. LIKE with Leading Wildcard

```sql
-- BAD: Can't use index
SELECT * FROM audit_events WHERE action LIKE '%login%';
-- Fix: Use specific prefix or full-text search
SELECT * FROM audit_events WHERE action LIKE 'user.login%';
```

### 3. JSONB Deep Query Without GIN

```sql
-- BAD: Full table scan for JSONB field
SELECT * FROM audit_events WHERE details->>'ip' = '10.0.1.5';
-- Fix: Add GIN index
CREATE INDEX idx_audit_details ON audit_events USING GIN (details jsonb_path_ops);
```

### 4. COUNT(*) on Large Tables

```sql
-- BAD: Slow count
SELECT COUNT(*) FROM audit_events WHERE tenant_id = 'uuid';
-- Fix: Use materialized view estimate
SELECT event_count FROM audit_daily_stats WHERE tenant_id = 'uuid' AND day = CURRENT_DATE;
```

### 5. Selecting All Columns

```sql
-- BAD: Returns large JSONB details column
SELECT * FROM audit_events WHERE ...;
-- Fix: Only select needed columns
SELECT id, created_at, action, result FROM audit_events WHERE ...;
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Query latency p99 | <100ms | >500ms → analyze query plan |
| Partition count | 12-24 active | >36 → archive old |
| Index bloat | <20% | >30% → REINDEX |
| Materialized view refresh time | <30s | >2min → optimize |

## See Also

- [Audit Log Architecture](audit-log-architecture.md)
- [Audit Query API](audit-query-api.md)
- [Audit Tamper Detection](audit-tamper-detection.md)
- [Database Security](database-security.md)
