# Backup and Recovery Strategy

This guide covers backup types, targets, RPO/RTO per component, encryption, off-site replication, recovery procedures, disaster recovery runbook, verification automation, and GGID's backup strategy.

## Backup Types

| Type | Description | Frequency | Size |
|---|---|---|---|
| Full | Complete copy of all data | Weekly | Large |
| Incremental | Changes since last backup | Daily | Small |
| Differential | Changes since last full | Daily | Medium |
| Snapshot | Point-in-time copy | Hourly | Varies |

## Backup Targets

### Component Backup Matrix

| Component | Backup Method | Frequency | RPO | RTO |
|---|---|---|---|---|
| PostgreSQL | pg_dump + WAL archiving | Full: weekly, WAL: continuous | 5 min | 30 min |
| Redis | RDB + AOF | RDB: hourly, AOF: continuous | 1 min | 5 min |
| NATS JetStream | Stream snapshots | Hourly | 1 hour | 15 min |
| Config files | Git + file copy | On change | 0 | 5 min |
| JWT keys | KMS backup | On rotation | 0 | 1 min |
| Audit logs | Database backup + SIEM | Continuous | 0 | 1 hour |
| User uploads | S3 versioning | Continuous | 0 | 15 min |

## RPO/RTO Definitions

| Metric | Definition |
|---|---|
| RPO (Recovery Point Objective) | Maximum acceptable data loss |
| RTO (Recovery Time Objective) | Maximum acceptable downtime |

### Per-Component Targets

| Component | RPO | RTO | Justification |
|---|---|---|---|
| Auth service | 5 min | 15 min | Users can't login |
| Identity service | 5 min | 30 min | User data critical |
| OAuth service | 5 min | 15 min | Token issuance |
| Policy service | 1 hour | 1 hour | Cached decisions |
| Audit service | 0 | 1 hour | Compliance, no data loss |
| Gateway | 0 | 5 min | Single point of entry |

## Backup Encryption

### Encryption Configuration

```yaml
backup:
  encryption:
    enabled: true
    algorithm: "AES-256-GCM"
    key_management: "aws-kms"
    key_id: "backup-encryption-key"
    encrypt_in_transit: true
    tls_min_version: "TLS1.2"
```

### Encrypted Backup

```bash
#!/bin/bash
# PostgreSQL encrypted backup
pg_dump ggid | openssl enc -aes-256-gcm -salt -pbkdf2 -pass file:/etc/ggid/backup-key -out backup-$(date +%Y%m%d).sql.enc

# Verify backup
openssl enc -d -aes-256-gcm -pbkdf2 -pass file:/etc/ggid/backup-key -in backup-$(date +%Y%m%d).sql.enc | pg_restore --list
```

## Off-Site Replication

### Replication Strategy

```
Primary DC                    Off-Site
┌──────────┐    async         ┌──────────┐
│PostgreSQL│─────────────────▶│PostgreSQL│
│  (RW)    │   streaming rep  │  (RO)    │
└──────────┘                  └──────────┘
┌──────────┐    sync          ┌──────────┐
│  Redis   │─────────────────▶│  Redis   │
│  (RW)    │   replica        │  (RO)    │
└──────────┘                  └──────────┘
┌──────────┐    rsync         ┌──────────┐
│ Backups  │─────────────────▶│  S3/     │
│ (local)  │   every 1h       │ Glacier  │
└──────────┘                  └──────────┘
```

### Configuration

```yaml
backup:
  offsite:
    enabled: true
    destination: "s3://ggid-backups"
    replication_interval: 1h
    regions:
      primary: "us-east-1"
      secondary: "us-west-2"
    encryption: true
    lifecycle:
      hot: 7d      # S3 standard
      warm: 30d    # S3 IA
      cold: 365d   # S3 Glacier
      delete: 2555d  # 7 years
```

## Recovery Procedures

### PostgreSQL Point-in-Time Recovery

```bash
#!/bin/bash
# 1. Stop PostgreSQL
systemctl stop postgresql

# 2. Restore base backup
rm -rf /var/lib/postgresql/data/*
tar xzf /backups/base/base-20260712.tar.gz -C /var/lib/postgresql/data/

# 3. Configure recovery
cat > /var/lib/postgresql/data/recovery.signal << EOF
restore_command = 'cp /backups/wal/%f %p'
recovery_target_time = '2026-07-12 10:30:00'
recovery_target_action = 'promote'
EOF

# 4. Start PostgreSQL (begins recovery)
systemctl start postgresql

# 5. Monitor recovery
tail -f /var/log/postgresql/recovery.log
```

### Redis Recovery

```bash
#!/bin/bash
# RDB recovery
redis-cli SHUTDOWN NOSAVE
cp /backups/redis/dump.rdb /var/lib/redis/
chown redis:redis /var/lib/redis/dump.rdb
systemctl start redis

# AOF recovery (more precise)
cp /backups/redis/appendonly.aof /var/lib/redis/
systemctl start redis
```

### NATS Stream Recovery

```bash
#!/bin/bash
# Restore NATS JetStream from snapshot
nsc restore --dir /backups/nats/snapshot-20260712
systemctl restart nats
```

## Disaster Recovery Runbook

### DR Runbook Template

```markdown
## Disaster Recovery: [Scenario]

### Trigger
- [Detection criteria]
- [Who declares disaster]

### Immediate Actions (0-15 min)
1. [ ] Declare disaster (DR coordinator)
2. [ ] Notify incident response team
3. [ ] Switch DNS to DR site
4. [ ] Verify DR site is healthy

### Recovery Actions (15-60 min)
1. [ ] Restore PostgreSQL from backup
2. [ ] Verify data integrity
3. [ ] Start services in order: auth → identity → oauth → policy → audit → gateway
4. [ ] Verify each service health
5. [ ] Test authentication flow

### Post-Recovery (1-24 hours)
1. [ ] Monitor for errors
2. [ ] Verify all services operational
3. [ ] Communicate with users
4. [ ] Document timeline
5. [ ] Schedule post-mortem

### Contacts
- DR Coordinator: [name, phone]
- Security Team: [name, phone]
- Infrastructure: [name, phone]
```

### Service Recovery Order

```
1. PostgreSQL + Redis (infrastructure)
2. Auth Service (authentication)
3. Identity Service (user data)
4. OAuth Service (token issuance)
5. Policy Service (authorization)
6. Audit Service (compliance)
7. Gateway (entry point)
8. Console (admin UI)
```

## Backup Verification Automation

### Automated Verification

```yaml
backup:
  verification:
    enabled: true
    schedule: "daily"
    tests:
      - name: "pg_restore_test"
        command: "pg_restore --list backup-latest.sql.enc"
        expect: "success"
      - name: "redis_rdb_test"
        command: "redis-check-rdb /backups/redis/dump.rdb"
        expect: "OK"
      - name: "restore_dry_run"
        command: "pg_restore --dry-run backup-latest.sql"
        expect: "no errors"
    alert_on_failure: true
    notify: ["ops-team"]
```

### Restore Test

```bash
#!/bin/bash
# Monthly restore test
DATE=$(date +%Y%m01)
BACKUP_FILE="backup-${DATE}.sql.enc"

# Decrypt and restore to test database
openssl enc -d -aes-256-gcm -pbkdf2 -pass file:/etc/ggid/backup-key \
  -in /backups/${BACKUP_FILE} | \
  psql -h test-db ggid_test

# Verify row counts
EXPECTED=$(psql -h prod-db -t -c "SELECT count(*) FROM users")
ACTUAL=$(psql -h test-db -t -c "SELECT count(*) FROM users")

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "VERIFY FAILED: expected $EXPECTED, got $ACTUAL"
  exit 1
fi

echo "Backup verification PASSED"
```

## GGID Backup Strategy

### Configuration

```yaml
backup:
  enabled: true
  
  postgresql:
    method: "pg_dump + WAL archiving"
    full: weekly
    wal: continuous
    rpo: 5m
    rto: 30m
    encryption: true
  
  redis:
    method: "RDB + AOF"
    rdb: hourly
    aof: continuous
    rpo: 1m
    rto: 5m
  
  nats:
    method: "stream snapshots"
    frequency: hourly
    rpo: 1h
    rto: 15m
  
  offsite:
    destination: "s3://ggid-backups"
    replication: 1h
    lifecycle:
      hot: 7d
      warm: 30d
      cold: 365d
      delete: 2555d
  
  verification:
    schedule: daily
    monthly_restore_test: true
    alert_on_failure: true
  
  disaster_recovery:
    rto: 1h
    rpo: 5m
    dr_site: "us-west-2"
    dns_failover: true
    quarterly_dr_test: true
```

## Best Practices

1. **Automate backups** — Never rely on manual processes
2. **Encrypt all backups** — At rest and in transit
3. **Replicate off-site** — Don't keep all backups in one location
4. **Verify regularly** — Test restore at least monthly
5. **Document procedures** — Runbook for every recovery scenario
6. **Set RPO/RTO** — Know your recovery targets
7. **Test disaster recovery** — Quarterly DR exercise
8. **Use lifecycle policies** — Move old backups to cheaper storage
9. **Monitor backup health** — Alert on backup failures
10. **Keep audit backups longest** — Compliance requires long retention