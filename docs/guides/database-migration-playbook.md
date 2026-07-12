# Database Migration Playbook

Zero-downtime schema changes, expand-contract pattern, backward compatibility, rollback procedures, data backfill, lock management, and large table strategies.

## Golden Rule

**Never break backward compatibility in a single migration.** Deploy schema changes and code changes in separate releases.

## Expand-Contract Pattern

```
Phase 1 (Expand): Add new schema (both old and new coexist)
Phase 2 (Migrate): Deploy code that uses new schema, backfill data
Phase 3 (Contract): Remove old schema (after code fully migrated)
```

### Example: Rename Column

```sql
-- Phase 1: EXPAND (migration v1)
ALTER TABLE users ADD COLUMN display_name_new TEXT;
-- Old column (display_name) still exists, app uses it

-- Phase 2: MIGRATE (app deploy + backfill)
UPDATE users SET display_name_new = display_name WHERE display_name_new IS NULL;
-- App deployed to read display_name_new, fallback to display_name

-- Phase 3: CONTRACT (migration v2, after all app instances updated)
ALTER TABLE users DROP COLUMN display_name;
ALTER TABLE users RENAME COLUMN display_name_new TO display_name;
```

## Zero-Downtime Operations

### Add Column (Safe)

```sql
-- ✅ Safe: nullable column without default
ALTER TABLE users ADD COLUMN department TEXT;

-- ✅ Safe: nullable with default (PG 11+, no table rewrite)
ALTER TABLE users ADD COLUMN status TEXT DEFAULT 'active';

-- ❌ DANGEROUS: NOT NULL without default (locks table)
ALTER TABLE users ADD COLUMN required_field TEXT NOT NULL;

-- ✅ Safe: Multi-step
-- Step 1: Add nullable
ALTER TABLE users ADD COLUMN required_field TEXT;
-- Step 2: Backfill
UPDATE users SET required_field = 'default' WHERE required_field IS NULL;
-- Step 3: Add NOT NULL
ALTER TABLE users ALTER COLUMN required_field SET NOT NULL;
```

### Add Index (Safe)

```sql
-- ✅ Safe: CONCURRENTLY doesn't lock table
CREATE INDEX CONCURRENTLY idx_users_email_lower
  ON users (LOWER(email));

-- ❌ DANGEROUS: Locks table for writes
CREATE INDEX idx_users_email_lower ON users (LOWER(email));
```

### Alter Column Type (Multi-Step)

```sql
-- ❌ DANGEROUS: Rewrites entire table
ALTER TABLE users ALTER COLUMN employee_id TYPE INTEGER;

-- ✅ Safe: Multi-step expand-contract
ALTER TABLE users ADD COLUMN employee_id_int INTEGER;
UPDATE users SET employee_id_int = employee_id::INTEGER;
ALTER TABLE users DROP COLUMN employee_id;
ALTER TABLE users RENAME COLUMN employee_id_int TO employee_id;
```

## Lock Management

### Lock Types

| Operation | Lock Level | Blocks |
|-----------|-----------|--------|
| SELECT | AccessShare | Nothing |
| INSERT/UPDATE/DELETE | RowExclusive | Other writes (briefly) |
| ALTER TABLE | AccessExclusive | Everything (dangerous) |
| CREATE INDEX CONCURRENTLY | Share | Only writes |

### Lock Duration

```sql
-- Statement lock: released after each statement
BEGIN;
UPDATE users SET status = 'active' WHERE id = 'uuid';
COMMIT;

-- DDL lock: held until transaction ends
BEGIN;
ALTER TABLE users ADD COLUMN phone TEXT; -- Lock held until COMMIT
COMMIT;
```

### Avoiding Long Locks

```sql
-- Set lock timeout (auto-abort if can't get lock)
SET lock_timeout = '3s';
ALTER TABLE users ADD COLUMN phone TEXT;
-- If another transaction holds the lock for >3s, this aborts
```

## Data Backfill

### Batched Backfill

```go
func backfillColumn(db *sql.DB) error {
    batchSize := 1000
    for {
        result, err := db.Exec(`
            UPDATE users 
            SET display_name = email 
            WHERE display_name IS NULL 
            LIMIT $1
        `, batchSize)
        if err != nil { return err }
        
        rows, _ := result.RowsAffected()
        if rows == 0 { break } // Done
        
        time.Sleep(100 * time.Millisecond) // Throttle
    }
    return nil
}
```

### Large Table Backfill

```sql
-- For millions of rows, use keyset pagination
DECLARE done BOOLEAN DEFAULT FALSE;
WHILE NOT done DO
  UPDATE users SET display_name = email
  WHERE id IN (
    SELECT id FROM users WHERE display_name IS NULL LIMIT 1000
  );
  SET done = ROW_COUNT() = 0;
  DO SLEEP(0.1);
END WHILE;
```

## Large Table Strategies

### Partitioning

```sql
-- Partition large tables before migration
CREATE TABLE audit_events_new (...) PARTITION BY RANGE (created_at);

-- Migrate data per partition (minimal locking)
INSERT INTO audit_events_new_2025_01 
  SELECT * FROM audit_events WHERE created_at >= '2025-01-01' AND created_at < '2025-02-01';

-- Swap table names (brief lock)
BEGIN;
ALTER TABLE audit_events RENAME TO audit_events_old;
ALTER TABLE audit_events_new RENAME TO audit_events;
COMMIT;
```

### Online Schema Change Tools

| Tool | Mechanism | Use Case |
|------|-----------|----------|
| pg_repack | Shadow table + trigger | Rebuild bloated tables |
| pg_partman | Automated partitioning | Time-series tables |
| pg-osc | Shadow table + CDC | Large ALTER TABLE |

## Rollback Procedures

### Migration Rollback

```bash
# Each migration has a down migration
migrate down -version 003   # Roll back to v003

# If rollback requires data restoration:
pg_restore -d ggid /backups/pre-migration.dump
```

### Emergency Rollback

```sql
-- If column rename broke app:
ALTER TABLE users RENAME COLUMN display_name TO display_name_old;
ALTER TABLE users RENAME COLUMN display_name_new TO display_name;
-- App still reads display_name, now mapped to new column
```

## Migration Testing

### Pre-Production Checklist

- [ ] Test on copy of production data
- [ ] Verify migration time (should be < maintenance window)
- [ ] Test app with old schema (backward compat)
- [ ] Test app with new schema
- [ ] Test rollback
- [ ] Check for long-running queries that might block DDL

### CI Pipeline

```yaml
migration_test:
  steps:
    - name: create-test-db
      run: createdb ggid_migration_test
    
    - name: apply-all-migrations
      run: migrate -path migrations -database $TEST_DB up
    
    - name: seed-large-data
      run: ./scripts/seed-1M-rows.sh
    
    - name: apply-new-migration
      run: migrate -path migrations -database $TEST_DB up 1
    
    - name: verify-data-integrity
      run: ./scripts/verify-data.sh
    
    - name: test-rollback
      run: migrate -path migrations -database $TEST_DB down 1
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Migration duration | >5 min for non-partitioned tables |
| Lock wait time | >5s → blocking issue |
| Failed migrations | Any → investigate |
| Table bloat post-migration | >30% → schedule VACUUM/REINDEX |

## See Also

- [Backup and Restore](backup-and-restore.md)
- [Zero-Downtime Deployment](zero-downtime-deployment.md)
- [Canary Deployment Strategy](canary-deployment-strategy.md)
- [Database Security](database-security.md)
