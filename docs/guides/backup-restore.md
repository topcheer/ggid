# Backup & Restore Guide

> PostgreSQL, Redis, NATS, and JWT key backup procedures with cron scripts.

---

## PostgreSQL

### Daily Backup (cron)

```bash
#!/bin/bash
# /opt/ggid/scripts/backup-db.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/backups/postgres
mkdir -p $BACKUP_DIR

pg_dump $DATABASE_URL | gzip > $BACKUP_DIR/ggid_${DATE}.sql.gz

# Upload to S3
aws s3 cp $BACKUP_DIR/ggid_${DATE}.sql.gz s3://ggid-backups/postgres/

# Retain 30 days
find $BACKUP_DIR -name "ggid_*.sql.gz" -mtime +30 -delete
echo "Backup complete: ggid_${DATE}.sql.gz"
```

### Cron

```cron
0 2 * * * /opt/ggid/scripts/backup-db.sh >> /var/log/ggid-backup.log 2>&1
```

### Restore

```bash
#!/bin/bash
# restore-db.sh <backup-file>
BACKUP=$1
aws s3 cp s3://ggid-backups/postgres/${BACKUP} - | gunzip | psql $DATABASE_URL
curl -s http://localhost:8080/api/v1/audit/verify | jq .verified
```

---

## Redis

### RDB Snapshot

```bash
redis-cli BGSAVE
# Copy dump.rdb to backup location
cp /var/lib/redis/dump.rdb /backups/redis/dump_$(date +%Y%m%d).rdb
```

### AOF (Append Only File)

```bash
# Enable AOF for point-in-time recovery
redis-cli CONFIG SET appendonly yes
redis-cli CONFIG SET appendfsync everysec
```

---

## NATS JetStream

```bash
# Backup stream data
curl http://localhost:8222/jsz?streams=true > /backups/nats/streams_$(date +%Y%m%d).json

# For full backup: copy JetStream store directory
tar -czf /backups/nats/jetstream_$(date +%Y%m%d).tar.gz /data/nats/
```

---

## JWT Signing Keys

```bash
# Backup RSA keys
mkdir -p /backups/keys
cp /etc/ggid/keys/private.pem /backups/keys/private_$(date +%Y%m%d).pem
cp /etc/ggid/keys/public.pem /backups/keys/public_$(date +%Y%m%d).pem

# Encrypt
gpg --encrypt --recipient security@example.com /backups/keys/private_*.pem
rm /backups/keys/private_*.pem  # Keep only encrypted
```

---

## Full Backup Script

```bash
#!/bin/bash
# /opt/ggid/scripts/backup-all.sh
echo "=== GGID Full Backup $(date) ==="
/opt/ggid/scripts/backup-db.sh
/opt/ggid/scripts/backup-redis.sh
/opt/ggid/scripts/backup-nats.sh
/opt/ggid/scripts/backup-keys.sh
echo "=== Backup Complete ==="
```

```cron
0 2 * * * /opt/ggid/scripts/backup-all.sh
```

---

## Restore Procedure

1. Restore PostgreSQL from latest backup
2. Restore Redis from RDB (if sessions need preserving)
3. Restore NATS store directory (optional — audit events archived in DB)
4. Restore JWT keys (or generate new — forces re-authentication)
5. Verify: `curl http://localhost:8080/healthz`
6. Verify: `curl http://localhost:8080/api/v1/audit/verify`

---

*See: [Disaster Recovery](disaster-recovery.md) | [Production Checklist](production-readiness-checklist.md)*

*Last updated: 2025-07-11*
