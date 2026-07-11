# Disaster Recovery Guide

> RTO/RPO targets, backup procedures, and disaster recovery drills for GGID.

---

## RTO/RPO Targets

| Component | RTO | RPO |
|-----------|-----|-----|
| PostgreSQL | 30 min | 15 min |
| Redis | 5 min | 0 (ephemeral) |
| NATS JetStream | 15 min | 0 (file-backed) |
| JWT signing keys | 5 min | 0 (replicated) |

---

## Database Backup & Restore

### Backup (daily cron)

```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump $DATABASE_URL | gzip > /backups/ggid_${DATE}.sql.gz
# Upload to S3
aws s3 cp /backups/ggid_${DATE}.sql.gz s3://ggid-backups/
# Retain 30 days
find /backups -name "ggid_*.sql.gz" -mtime +30 -delete
```

### Restore

```bash
#!/bin/bash
# restore.sh
BACKUP=$1
aws s3 cp s3://ggid-backups/${BACKUP} - | gunzip | psql $DATABASE_URL
# Verify hash chain
curl -s http://localhost:8080/api/v1/audit/verify
```

---

## JWT Key Rotation

```bash
# 1. Add new key to JWKS (both keys active during transition)
ggid keys rotate
# 2. Wait for old tokens to expire (15 min)
# 3. Remove old key
ggid keys prune
```

In disaster: if signing key compromised:
1. Revoke all sessions (`FLUSHDB` on Redis session keys)
2. Deploy new signing key
3. Users re-authenticate

---

## Redis Failover

Use Redis Sentinel or Redis Cluster:

```bash
REDIS_HOST=redis-primary.internal
REDIS_SENTINEL=redis-sentinel-1,redis-sentinel-2,redis-sentinel-3
```

If Redis fails: sessions lost → users re-authenticate (graceful degradation).

---

## NATS JetStream Persistence

JetStream writes to disk — survives container restarts:

```yaml
nats:
  args: ["-js", "--store_dir", "/data", "--max_mem_store", "1GB"]
  volumeMounts:
    - name: nats-data
      mountPath: /data
```

---

## DR Drill (Quarterly)

1. Spin up DR environment in different region
2. Restore latest backup
3. Verify hash chain integrity
4. Run E2E test suite
5. Measure actual RTO
6. Document results

---

*See: [Production Checklist](../deploy/production-checklist.md) | [Backup Recovery](../backup-recovery.md) | [Operations Runbook](../operations-runbook.md)*

*Last updated: 2025-07-11*
