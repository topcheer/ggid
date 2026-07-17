# Disaster Recovery & Backup Automation: RTO/RPO Targets, Restore Testing, and Chaos Engineering for GGID

> **Focus**: Comprehensive DR strategy — PostgreSQL backup (pg_dump + WAL archiving + PITR), Redis persistence, cross-region replication, automated restore testing, chaos engineering, and incident runbooks with concrete RTO < 4h / RPO < 15min targets.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: DoD per backlog item (§9).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Backup & DR](#2-ggid-current-state-backup--dr)
3. [Gap Analysis](#3-gap-analysis)
4. [PostgreSQL Backup Strategy](#4-postgresql-backup-strategy)
5. [Redis Persistence & Backup](#5-redis-persistence--backup)
6. [Cross-Region Replication](#6-cross-region-replication)
7. [RTO/RPO Architecture](#7-rtorpo-architecture)
8. [Automated Restore Testing](#8-automated-restore-testing)
9. [Chaos Engineering](#9-chaos-engineering)
10. [Incident Runbook Templates](#10-incident-runbook-templates)
11. [Implementation Backlog with DoD](#11-implementation-backlog-with-dod)
12. [Competitive Differentiation](#12-competitive-differentiation)

---

## 1. Executive Summary

GGID has **zero automated backup infrastructure** — no pg_dump schedule, no WAL archiving, no Redis persistence config, no restore testing. This is the highest-risk production gap: a single PostgreSQL disk failure could cause complete data loss.

**Current state:**
- Docker Compose for infra (PG, Redis, NATS) ✅
- Database migrations (SQL files in `deploy/migrations/`) ✅
- Health checks (`/healthz`, `/readyz`) ✅
- K8s deployment manifests (`deploy/k8s/`) ✅
- **No backup of any kind** ❌

**Recommendation**: Implement tiered backup strategy: PG full daily + WAL continuous (PITR), Redis AOF + RDB, encrypted off-site to S3, automated monthly restore tests, and chaos engineering for resilience validation.

---

## 2. GGID Current State: Backup & DR

| Component | Status | Risk |
|-----------|--------|------|
| PostgreSQL data | ❌ No backup | **Critical** — total data loss on disk failure |
| Redis cache | ❌ No persistence | Medium — cache rebuilds from DB |
| NATS streams | ❌ No persistence | Low — events are transient |
| Config files | ❌ Not backed up | Medium — reconstructable from Git |
| K8s manifests | ✅ In Git | Low — version controlled |
| Migration files | ✅ In Git | Low — version controlled |
| Restore testing | ❌ None | Critical — backups untested = no backups |
| DR runbook | ❌ None | Critical — no recovery procedure |

---

## 3. Gap Analysis

| # | Gap | RTO Impact | RPO Impact |
|---|-----|-----------|-----------|
| 1 | No PG backup | ∞ (unrecoverable) | ∞ (total loss) |
| 2 | No WAL archiving | Can't do PITR | Hours of data loss |
| 3 | No Redis persistence | Cache loss (rebuild 10-30 min) | N/A |
| 4 | No off-site backup | Single AZ failure = total loss | — |
| 5 | No encryption | Backup data exposed | — |
| 6 | No restore test | Backups may be corrupted | — |
| 7 | No replication | No failover target | — |
| 8 | No DR runbook | Recovery takes hours longer | — |

---

## 4. PostgreSQL Backup Strategy

### Tiered Backup Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  PostgreSQL Backup Pipeline                                  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Level 1: WAL Archiving (continuous)                │    │
│  │  • archive_mode = on                                │    │
│  │  • archive_command = 'wal-g wal-push %p'            │    │
│  │  • Every transaction archived to S3                  │    │
│  │  • RPO: < 15 minutes                                │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Level 2: Full Backup (daily, 02:00 UTC)            │    │
│  │  • pg_dump --format=custom --compress=9              │    │
│  │  • OR: pgBackRest / WAL-G full backup                │    │
│  │  • Uploaded to S3 (encrypted)                        │    │
│  │  • Retained: 30 daily + 12 monthly + 7 yearly       │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Level 3: Point-in-Time Recovery (PITR)             │    │
│  │  • Restore base backup + replay WAL                  │    │
│  │  • Recover to any point in time:                     │    │
│  │    recovery_target_time = '2026-07-17 10:30:00'     │    │
│  │  • RPO: < 15 min (WAL archive lag)                  │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### PG Configuration

```ini
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'wal-g wal-push %p'
archive_timeout = '300s'        # Force WAL switch every 5 min
max_wal_senders = 3

# Connection
max_connections = 200
shared_buffers = '4GB'
effective_cache_size = '12GB'
```

### Backup Script (Cron)

```bash
#!/bin/bash
# /opt/ggid/scripts/pg-backup.sh — Daily full backup

set -euo pipefail
DATE=$(date +%Y%m%d_%H%M%S)
BUCKET="s3://ggid-backups/postgresql"
PG_HOST="localhost"
PG_USER="ggid_backup"

# Full compressed backup
pg_dump --host=$PG_HOST --username=$PG_USER \
  --format=custom --compress=9 \
  --file="/tmp/ggid_${DATE}.dump" ggid

# Encrypt (age)
age -r "ssh-ed25519 AAAA..." \
  -o "/tmp/ggid_${DATE}.dump.age" \
  "/tmp/ggid_${DATE}.dump"

# Upload to S3
aws s3 cp "/tmp/ggid_${DATE}.dump.age" "$BUCKET/daily/"

# Cleanup local
rm "/tmp/ggid_${DATE}.dump" "/tmp/ggid_${DATE}.dump.age"

# Retention: delete daily backups older than 30 days
aws s3 ls "$BUCKET/daily/" | awk '{print $4}' | \
  while read f; do
    d=$(echo "$f" | grep -oP '\d{8}')
    [ $(( $(date +%Y%m%d) - d )) -gt 30 ] && \
      aws s3 rm "$BUCKET/daily/$f"
  done

echo "Backup completed: ggid_${DATE}.dump.age"
```

### Cron Schedule

```cron
# /etc/cron.d/ggid-backup
0 2 * * * ggid  /opt/ggid/scripts/pg-backup.sh           # Daily full 02:00
*/5 * * * * ggid /opt/ggid/scripts/pg-wal-check.sh        # WAL archive check
0 6 * * 6   ggid /opt/ggid/scripts/pg-monthly-snapshot.sh  # Weekly snapshot
0 3 1 * *   ggid /opt/ggid/scripts/pg-monthly-snapshot.sh  # Monthly snapshot
```

---

## 5. Redis Persistence & Backup

```conf
# redis.conf
# AOF (Append-Only File) — durability
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec            # Best balance of durability/performance
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb

# RDB (Snapshot) — point-in-time recovery
save 900 1                       # Snapshot if ≥1 key changed in 15 min
save 300 10                      # Snapshot if ≥10 keys changed in 5 min
save 60 10000                    # Snapshot if ≥10000 keys changed in 1 min
```

### Redis Backup Script

```bash
#!/bin/bash
# Save snapshot to disk → upload to S3
redis-cli BGSAVE
sleep 5  # Wait for background save
DATE=$(date +%Y%m%d_%H%M%S)
cp /var/lib/redis/dump.rdb "/tmp/redis_${DATE}.rdb"
age -r "ssh-ed25519 AAAA..." -o "/tmp/redis_${DATE}.rdb.age" "/tmp/redis_${DATE}.rdb"
aws s3 cp "/tmp/redis_${DATE}.rdb.age" "s3://ggid-backups/redis/daily/"
rm "/tmp/redis_${DATE}.rdb" "/tmp/redis_${DATE}.rdb.age"
```

---

## 6. Cross-Region Replication

### PostgreSQL Streaming Replication

```
Primary (us-east)          Standby (us-west)
  ├── Writes                 ├── Read-only queries
  ├── WAL stream ──────────▶ ├── WAL replay (async)
  └── Failover target ◀────── └── Promoted on failure

Configuration:
  Primary: max_wal_senders = 3, wal_level = replica
  Standby: primary_conninfo = 'host=primary port=5432'
  
Failover: pg_promote() on standby → becomes new primary
RTO: ~30 seconds (promotion time)
RPO: ~0 seconds (sync replication) or < 15min (async)
```

### Redis Replication

```
Primary → Replica (read-only)
Sentinel monitors → promotes replica on primary failure
```

---

## 7. RTO/RPO Architecture

### Target Metrics

| Tier | Component | RTO | RPO | Strategy |
|------|-----------|-----|-----|----------|
| **Tier 1** | PostgreSQL (primary) | 4h | 15min | pg_dump + WAL + PITR |
| **Tier 1** | PostgreSQL (HA) | 30s | 0s | Streaming replication + failover |
| **Tier 2** | Redis | 5min | 1min | AOF + restart from RDB |
| **Tier 2** | NATS | 2min | 0s | Stream replication |
| **Tier 3** | Config | 5min | 0s | Git (version controlled) |
| **Tier 3** | K8s manifests | 5min | 0s | Git + ArgoCD |

### What Supports RTO < 4h

```
RTO breakdown for total disaster recovery:
  1. Detect failure:        5 min (alerting)
  2. Provision new infra:   15 min (Terraform/k8s)
  3. Restore PG backup:     60 min (pg_restore + WAL replay)
  4. Restore Redis:         5 min (RDB load)
  5. Deploy services:       10 min (ArgoCD)
  6. Verify health:         10 min (smoke tests)
  7. DNS cutover:           5 min
  Total:                    ~110 min (under 4h target) ✅
```

---

## 8. Automated Restore Testing

### Monthly Restore Test (CI Job)

```yaml
# .github/workflows/restore-test.yml
name: Monthly Restore Test
on:
  schedule:
    - cron: '0 4 1 * *'  # 1st of month, 04:00 UTC

jobs:
  restore-test:
    steps:
      - name: Download latest backup
        run: aws s3 cp s3://ggid-backups/postgresql/daily/latest.dump.age .

      - name: Decrypt
        run: age -d -i key.txt latest.dump.age > latest.dump

      - name: Start fresh PostgreSQL
        run: docker run -d --name pg-restore -e POSTGRES_PASSWORD=test postgres:16

      - name: Restore backup
        run: |
          docker cp latest.dump pg-restore:/tmp/
          docker exec pg-restore pg_restore -U postgres -d ggid /tmp/latest.dump

      - name: Verify data integrity
        run: |
          # Check row counts
          USERS=$(docker exec pg-restore psql -U postgres -d ggid -t -c "SELECT COUNT(*) FROM users")
          [ "$USERS" -gt 0 ] || exit 1

          # Check hash chain integrity
          docker exec pg-restore psql -U postgres -d ggid -c "SELECT verify_hash_chain()"

          # Check audit events exist
          EVENTS=$(docker exec pg-restore psql -U postgres -d ggid -t -c "SELECT COUNT(*) FROM audit_events")
          [ "$EVENTS" -gt 0 ] || exit 1

      - name: Report
        run: echo "Restore test passed: all verification queries succeeded"
```

---

## 9. Chaos Engineering

### Fault Injection Tests

| Test | Method | Expected Behavior |
|------|--------|-------------------|
| Kill gateway pod | `kubectl delete pod -l app=gateway` | K8s restarts < 30s, no user impact |
| Kill PG primary | `kubectl delete pod -l app=postgres` | Replica promotes, < 60s outage |
| Network partition | Block traffic between services | Circuit breaker activates |
| Disk full (PG) | Fill disk to 100% | Alert fires, auto-scale disk |
| Redis flush | `redis-cli FLUSHALL` | Cache rebuilds from DB |
| High CPU | Stress test pod | HPA scales up |
| Clock skew | Shift pod clock | Audit timestamps checked |

### Chaos Test Script (litmus/gremlin)

```bash
#!/bin/bash
# chaos-test.sh — Run chaos experiments

echo "=== Experiment 1: Kill gateway pod ==="
kubectl delete pod -l app=gateway -n ggid
sleep 30
kubectl get pods -l app=gateway -n ggid  # Should show Running
curl -sf http://localhost:8080/healthz || exit 1

echo "=== Experiment 2: Kill auth pod ==="
kubectl delete pod -l app=auth -n ggid
sleep 30
curl -sf http://localhost:8080/healthz || exit 1

echo "All chaos experiments passed ✅"
```

---

## 10. Incident Runbook Templates

### Template: PostgreSQL Data Loss

```
SEVERITY: P0 — Complete data loss
TRIGGER: PG primary disk failure, no replica

STEPS:
1. Declare incident (Slack #incidents)
2. Provision new PG instance (Terraform)
3. Download latest backup from S3
4. Decrypt: age -d -i key.txt backup.dump.age > backup.dump
5. Restore: pg_restore -U ggid -d ggid -1 backup.dump
6. Replay WAL for PITR: set recovery_target_time
7. Start GGID services (ArgoCD)
8. Verify: row counts + hash chain + health checks
9. DNS cutover to new instance
10. Post-mortem within 48h

ESTIMATED RTO: 2-4 hours
```

### Template: Redis Failure

```
SEVERITY: P2 — Cache loss
TRIGGER: Redis unavailable

STEPS:
1. GGID automatically falls back to DB queries (slower)
2. Restart Redis pod: kubectl delete pod -l app=redis
3. Redis loads RDB snapshot (5 min)
4. Cache warms from traffic
5. Verify: rate limiting works, sessions valid

ESTIMATED RTO: 5-10 minutes
```

---

## 11. Implementation Backlog with DoD

### P0 — Critical Backup Infrastructure (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | PG full backup (pg_dump daily + S3) | ✅ Encrypted upload ✅ Retention policy ✅ Cron scheduled ✅ ≥3 test restores | 3d |
| 2 | PG WAL archiving (PITR) | ✅ archive_mode on ✅ WAL-G to S3 ✅ PITR tested ✅ ≥3 tests | 3d |
| 3 | Redis persistence (AOF + RDB) | ✅ AOF everysec ✅ RDB snapshots ✅ Backup to S3 ✅ ≥3 tests | 2d |
| 4 | Backup encryption (age/GPG) | ✅ All backups encrypted ✅ Key in Vault ✅ ≥3 tests | 2d |
| 5 | Off-site backup (S3/GCS) | ✅ Cross-region bucket ✅ Lifecycle policy ✅ ≥3 tests | 1d |

### P1 — HA + Restore Testing (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 6 | PG streaming replication | ✅ Standby replica ✅ Automated failover ✅ ≥3 tests | 4d |
| 7 | Automated monthly restore test | ✅ CI job runs restore ✅ Verification queries ✅ Alert on failure | 3d |
| 8 | DR runbook (3 scenarios) | ✅ PG loss + Redis loss + total region ✅ Step-by-step ✅ RTO verified | 2d |

### P2 — Chaos + Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 9 | Chaos engineering suite | 7 fault injection tests, automated |
| 10 | Blue-green deployment | Zero-downtime PG migration |
| 11 | Cross-region active-active | Multi-region write capability |
| 12 | Backup monitoring dashboard | Backup status + size + age |

---

## 12. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak |
|---------|---------------|------|-------|----------|
| **PG backup** | pg_dump + WAL PITR | Managed (AWS RDS) | Managed | Manual |
| **Redis backup** | AOF + RDB + S3 | Managed | Managed | Manual |
| **Off-site** | S3 cross-region | Multi-AZ | Multi-AZ | Manual |
| **Restore testing** | Monthly automated | Internal | Internal | No |
| **Chaos engineering** | 7 tests | Internal | Internal | No |
| **DR runbook** | 3 scenarios | Internal | Internal | No |
| **RTO/RPO** | 4h / 15min | ~1h / ~0min | ~1h / ~0min | Unknown |
| **Open source** | Yes | No | No | Yes |

---

## References

- [PostgreSQL Backup and Recovery](https://www.postgresql.org/docs/current/backup.html)
- [PostgreSQL PITR](https://www.postgresql.org/docs/current/continuous-archiving.html)
- [WAL-G](https://github.com/wal-g/wal-g) — PG backup to cloud
- [pgBackRest](https://pgbackrest.org/) — PG backup tool
- [Redis Persistence](https://redis.io/docs/management/persistence/)
- [age Encryption](https://github.com/FiloSottile/age) — Modern file encryption
- [Litmus Chaos](https://litmuschaos.io/) — Chaos engineering for K8s
- [GGID Deploy](../deploy/) — K8s manifests + migrations
- [GGID Production Hardening](./production-hardening-checklist.md) — Backup flagged as P0
