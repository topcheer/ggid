# Backup and Restore Guide

PostgreSQL, Redis, NATS, and configuration backups, restore procedures, RTO/RPO targets, and DR drills.

## Backup Strategy

| Component | Method | Frequency | Retention | RPO |
|-----------|--------|-----------|-----------|-----|
| PostgreSQL | pg_basebackup + WAL archiving | Daily full + continuous WAL | 30 days + 12 monthly | <1 min (PITR) |
| Redis | RDB snapshot + AOF | Every 5 min | 7 days | 5 min |
| NATS JetStream | Stream snapshot | Hourly | 7 days | 1 hour |
| Config (env, k8s secrets) | GitOps + etcd backup | On change | 90 days | 0 (Git) |
| Audit events | Already in PostgreSQL (partitioned) | Included in PG backup | 7 years | <1 min |

## PostgreSQL Backup

### Full Backup

```bash
#!/bin/bash
# Daily full backup (physical)
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups/postgresql"
mkdir -p "$BACKUP_DIR"

pg_basebackup \
  -h localhost \
  -U backup_user \
  -D "$BACKUP_DIR/full_$DATE" \
  -Ft -z -P \
  -c fast

# Retain 30 daily + 12 monthly
find "$BACKUP_DIR" -name "full_*" -mtime +30 -not -name "full_*01_*" -delete
```

### WAL Archiving (Continuous PITR)

```ini
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'aws s3 cp %p s3://ggid-wal-archive/%f'
max_wal_senders = 3
```

### Logical Backup (for specific tables)

```bash
# Daily logical dump (supplemental)
pg_dump -h localhost -U ggid ggid \
  --format=custom \
  --compress=9 \
  --file="/backups/postgresql/logical_$(date +%Y%m%d).dump"

# Specific tables only
pg_dump -h localhost -U ggid ggid \
  --table=users \
  --table=roles \
  --format=custom \
  --file="/backups/users_roles_$(date +%Y%m%d).dump"
```

## Redis Backup

### RDB Snapshot

```bash
# Trigger manual snapshot
redis-cli BGSAVE

# Copy RDB file
cp /var/lib/redis/dump.rdb /backups/redis/redis_$(date +%Y%m%d_%H%M).rdb
```

### AOF (Append Only File)

```ini
# redis.conf
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec   # Best balance of durability + performance
```

## NATS JetStream Backup

```bash
#!/bin/bash
# Snapshot all streams
for stream in $(nats stream ls -n); do
  nats stream backup "$stream" "/backups/nats/${stream}_$(date +%Y%m%d_%H%M)"
done

# Retain 7 days
find /backups/nats -maxdepth 1 -mtime +7 -exec rm -rf {} \;
```

## Configuration Backup

```bash
#!/bin/bash
# Kubernetes secrets + configmaps
kubectl get secrets -n ggid -o yaml > /backups/config/secrets_$(date +%Y%m%d).yaml
kubectl get configmaps -n ggid -o yaml > /backups/config/configmaps_$(date +%Y%m%d).yaml

# .env files
tar -czf /backups/config/env_$(date +%Y%m%d).tar.gz \
  /opt/ggid/*/cmd/.env \
  /opt/ggid/deploy/.env*
```

## Restore Procedures

### PostgreSQL Restore (Full)

```bash
#!/bin/bash
# Step 1: Stop GGID services
kubectl scale deployment -n ggid --replicas=0 --all

# Step 2: Stop PostgreSQL
kubectl scale statefulset postgresql -n ggid --replicas=0

# Step 3: Restore from base backup
rm -rf /var/lib/postgresql/data/*
tar -xzf /backups/postgresql/full_20250115_030000/base.tar.gz -C /var/lib/postgresql/data/

# Step 4: Configure recovery
cat > /var/lib/postgresql/data/recovery.signal << EOF
restore_command = 'aws s3 cp s3://ggid-wal-archive/%f %p'
recovery_target_time = '2025-01-15T10:30:00Z'  # PITR target
EOF

# Step 5: Start PostgreSQL
kubectl scale statefulset postgresql -n ggid --replicas=1

# Step 6: Wait for recovery to complete
# PostgreSQL automatically promotes when target reached

# Step 7: Start GGID services
kubectl scale deployment -n ggid --replicas=N --all
```

### PostgreSQL Restore (Single Table)

```bash
# Restore specific table from logical backup
pg_restore \
  --host=localhost \
  --username=ggid \
  --dbname=ggid \
  --table=users \
  --clean --if-exists \
  /backups/postgresql/users_roles_20250115.dump
```

### Redis Restore

```bash
# Stop Redis
redis-cli SHUTDOWN NOSAVE

# Replace RDB file
cp /backups/redis/redis_20250115_0300.rdb /var/lib/redis/dump.rdb

# Start Redis
systemctl start redis

# Verify
redis-cli DBSIZE
```

### NATS Restore

```bash
# Stop NATS
kubectl scale statefulset nats -n ggid --replicas=0

# Restore stream snapshot
nats stream restore AUDIT_EVENTS /backups/nats/AUDIT_EVENTS_20250115_0300

# Start NATS
kubectl scale statefulset nats -n ggid --replicas=3
```

## RTO / RPO Targets

| Scenario | RPO | RTO | Method |
|----------|-----|-----|--------|
| Single table corruption | 0 | <15 min | Logical restore |
| Database failure | <1 min | <30 min | PITR from WAL |
| Redis failure | 5 min | <5 min | RDB restore or rebuild from PG |
| NATS failure | 1 hour | <10 min | Stream restore |
| Full region loss | <5 min | <2 hours | Cross-region backup restore |
| Ransomware | <1 day | <4 hours | Immutable backups (WORM) |

## DR Drill Procedure

### Quarterly DR Drill

```bash
#!/bin/bash
echo "=== DR DRILL: $(date) ==="

# 1. Provision isolated restore environment
echo "1. Provisioning restore environment..."
kubectl create namespace ggid-dr-drill

# 2. Restore latest backup
echo "2. Restoring PostgreSQL..."
pg_restore_test "$LATEST_BACKUP"

# 3. Verify data integrity
echo "3. Verifying data..."
psql -c "SELECT count(*) FROM users;" # Compare with production
psql -c "SELECT count(*) FROM audit_events WHERE created_at > NOW() - INTERVAL '1 day';"

# 4. Start services
echo "4. Starting services..."
kubectl scale deployment -n ggid-dr-drill --replicas=1 --all

# 5. Verify health
echo "5. Health check..."
sleep 30
curl -sf http://gateway:8080/healthz && echo "PASS" || echo "FAIL"

# 6. Run E2E tests
echo "6. E2E tests..."
bash deploy/e2e-docker-test.sh

# 7. Cleanup
echo "7. Cleanup..."
kubectl delete namespace ggid-dr-drill

echo "=== DR DRILL COMPLETE ==="
```

### Drill Checklist

- [ ] Restore completed within RTO target
- [ ] No data loss beyond RPO target
- [ ] All services pass health checks
- [ ] E2E tests pass
- [ ] Audit hash chain intact after restore
- [ ] Encryption keys accessible
- [ ] Document any issues + improvements

## Immutable Backups (Anti-Ransomware)

```bash
# S3 Object Lock (WORM — Write Once Read Many)
aws s3api put-object-lock-configuration \
  --bucket ggid-backups \
  --object-lock-configuration '{
    "ObjectLockEnabled": "Enabled",
    "Rule": {
      "DefaultRetention": {
        "Mode": "COMPLIANCE",
        "Days": 30
      }
    }
  }'

# Backups cannot be deleted or modified for 30 days
# Even root/admin cannot bypass COMPLIANCE mode
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Backup job failure | Any → page ops |
| Backup size deviation | >20% change → investigate |
| WAL archive gap | Lag >5 min |
| Restore test failure | DR drill fails |
| Backup storage >80% capacity | Plan capacity increase |

## See Also

- [Database Security](database-security.md)
- [Multi-Region Deployment](multi-region-deployment.md)
- [Disaster Recovery](disaster-recovery.md)
- [High Availability](high-availability.md)
