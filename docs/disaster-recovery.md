# GGID Disaster Recovery Guide

How to prepare for, respond to, and recover from disasters in GGID.

---

## RPO and RTO Targets

| Metric | Target | Description |
|--------|--------|-------------|
| **RPO** (Recovery Point Objective) | < 15 minutes | Maximum acceptable data loss |
| **RTO** (Recovery Time Objective) | < 1 hour | Maximum acceptable downtime |
| **RTO (critical)** | < 15 minutes | For auth-dependent services |

---

## Backup Strategy

### PostgreSQL

#### Daily Full Backup

```bash
#!/bin/bash
# Run daily at 02:00 via cron
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/backups/postgres

docker exec ggid-postgres pg_dump \
  -U ggid \
  -Fc \
  --no-owner \
  --no-privileges \
  ggid | gzip > "$BACKUP_DIR/ggid_$DATE.dump.gz"

# Retention: 30 days
find "$BACKUP_DIR" -name "ggid_*.dump.gz" -mtime +30 -delete
```

#### WAL Archiving (PITR)

For Point-In-Time Recovery (RPO < 15 min):

```ini
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'aws s3 cp %p s3://ggid-wal-archive/%f'
max_wal_senders = 3
```

#### Restoring from Backup

```bash
# 1. Stop GGID services
docker compose down

# 2. Restore PostgreSQL from dump
gunzip < /backups/postgres/ggid_20240710_020000.dump.gz | \
  docker exec -i ggid-postgres pg_restore -U ggid -d ggid --clean

# 3. Restore to specific point in time (PITR)
# Using WAL archives:
# Stop postgres, start in recovery mode with recovery_target_time
recovery_target_time = '2024-07-10 12:30:00'

# 4. Restart services
docker compose up -d
```

### JWT Signing Keys

```bash
# Backup RSA keys (store in Vault/KMS, NOT in git)
aws s3 cp /configs/rsa_private.pem s3://ggid-secrets/rsa_private_$(date +%Y%m%d).pem --sse aws:kms
aws s3 cp /configs/rsa_public.pem s3://ggid-secrets/rsa_public_$(date +%Y%m%d).pem --sse aws:kms
```

### Redis

Redis data is ephemeral (sessions, rate limits). No backup needed. On recovery:
- Users re-authenticate (sessions lost)
- Rate limit counters reset

### NATS JetStream

Audit events in JetStream are transient. The durable copy is in PostgreSQL. No backup needed for NATS.

---

## Cross-Region Replication

### PostgreSQL Streaming Replication

```
Primary (Region A) ──► Standby (Region B)
     │                         │
     ▼                         ▼
  Read/Write               Read-only (promoted on failover)
```

#### Setup

```ini
# Primary (postgresql.conf)
wal_level = replica
max_wal_senders = 10
synchronous_commit = on

# Standby (recovery.signal + postgresql.auto.conf)
primary_conninfo = 'host=primary.db.internal port=5432 user=replicator password=...'
restore_command = 'aws s3 cp s3://ggid-wal-archive/%f %p'
```

#### Failover (Promote Standby)

```bash
# 1. On standby:
pg_ctl promote -D /var/lib/postgresql/data

# 2. Update DNS to point to new primary
# 3. Update GGID services' DATABASE_URL
# 4. Restart services
```

### NATS Super-Cluster

```
Region A NATS ◄──► Region B NATS
   Gateway              Gateway
```

Audit events published in Region A are replicated to Region B automatically.

---

## Service Degradation Modes

When infrastructure fails, GGID degrades gracefully:

| Failure | Impact | Degradation Mode | Recovery |
|---------|--------|-----------------|----------|
| PostgreSQL down | Auth, user management, policy fail | Services crash (cannot operate without DB) | Restore DB, restart |
| Redis down | Rate limiting disabled, sessions lost | Auth still works (rate limits bypassed) | Restart Redis |
| NATS down | Audit events lost (best-effort) | All services continue normally | Restart NATS |
| Gateway down | No API access | Services still running, just unreachable | Restart Gateway |
| Auth Service down | No logins/registrations | Existing tokens still valid (Gateway verifies locally) | Restart Auth |
| Policy Service down | Permission checks fail | API calls to policy endpoints return 503 | Restart Policy |
| Audit Service down | No audit query | Events buffer in NATS (7 days) | Restart Audit |

### Critical Priority Matrix

| Service | Priority | RTO | Reason |
|---------|:--------:|-----|--------|
| PostgreSQL | P0 | Immediate | All services depend on it |
| Gateway | P0 | 5 min | Single entry point |
| Auth | P1 | 15 min | New logins blocked |
| Identity | P1 | 15 min | User management blocked |
| Redis | P2 | 30 min | Rate limits/sessions affected |
| Policy | P2 | 30 min | Permission checks affected |
| OAuth | P3 | 1 hour | SSO affected |
| Org | P3 | 1 hour | Org management affected |
| Audit | P3 | 1 hour | Audit query affected (NATS buffers) |
| NATS | P3 | 1 hour | Audit pipeline affected |

---

## Disaster Recovery Runbook

### Scenario: Primary Database Failure

```
1. DETECT: Prometheus alert "PostgreSQL Down"
2. VERIFY: Confirm primary is unreachable
3. PROMOTE: Promote standby replica to primary
4. UPDATE: Change DNS to point to new primary
5. RESTART: Restart all GGID services with new DATABASE_URL
6. VERIFY: Run smoke test (healthz → register → login → API call)
7. COMMUNICATE: Notify stakeholders of recovery
```

### Scenario: Complete Region Loss

```
1. ACTIVATE: Switch DNS to secondary region
2. VERIFY: Secondary region healthy (standby DB + GGID replicas)
3. PROMOTE: Promote standby DB to primary
4. SCALE: Scale GGID services to full capacity in secondary region
5. RESTORE: Begin rebuilding primary region from backups
6. SYNC: Re-establish replication from secondary back to new primary
7. FAILOVER: Switch DNS back to primary region
```

### Scenario: Data Corruption

```
1. STOP: Stop affected service
2. IDENTIFY: Determine corruption scope (single tenant vs all)
3. RESTORE: Restore from most recent backup
4. REPLAY: Apply WAL to recover to point before corruption
5. VERIFY: Test data integrity
6. RESTART: Start services
```

---

## Testing DR

### Quarterly DR Drill

1. **Backup restoration test**: Restore from backup to a test environment
2. **Failover test**: Promote standby, verify application works, fail back
3. **Latency test**: Measure actual RTO during simulated failure
4. **Data loss test**: Verify RPO by comparing primary vs restored data

### Automated Health Checks

```bash
#!/bin/bash
# DR health check (run hourly)
STATUS=0

# Check PostgreSQL
docker exec ggid-postgres pg_isready || STATUS=1

# Check Gateway
curl -sf http://localhost:8080/healthz || STATUS=1

# Check backup freshness
LATEST=$(ls -t /backups/postgres/ggid_*.dump.gz | head -1)
AGE=$(( ($(date +%s) - $(stat -c %Y "$LATEST")) / 3600 ))
if [ "$AGE" -gt 25 ]; then STATUS=1; fi

exit $STATUS
```
