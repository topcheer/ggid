# Backup & Disaster Recovery — Technical Guide

> Feature: Automated Backup System + DR Readiness
> Console: `/admin/backup`

## What It Does

GGID's backup system provides automated PostgreSQL backups with point-in-time recovery (PITR), WAL archiving to S3-compatible storage, and disaster recovery readiness checks. Administrators can trigger manual backups, restore from snapshots, and verify DR readiness from the console.

## Backup Strategy

### Full Backups
- **Frequency**: Daily (configurable)
- **Method**: `pg_dump` with custom format (compressed, parallel)
- **Retention**: 30 days (configurable)
- **Destination**: S3-compatible storage (MinIO, AWS S3, GCS)

### WAL Archiving
- **Frequency**: Continuous (every WAL segment)
- **Method**: PostgreSQL `archive_command` → S3
- **Retention**: 7 days of WAL files
- **Purpose**: Point-in-time recovery between full backups

### Restore Points
- **Named snapshots**: Admin can create named restore points before risky operations
- **Timestamp recovery**: Restore to any point within WAL retention window

## RTO/RPO Targets

| Metric | Target | Description |
|--------|--------|-------------|
| **RPO** (Recovery Point Objective) | < 5 minutes | Maximum data loss tolerance |
| **RTO** (Recovery Time Objective) | < 30 minutes | Maximum time to restore service |

RPO achieved via continuous WAL archiving. RTO achieved via automated restore scripts.

## DR Readiness Checks

The system continuously verifies DR readiness:

| Check | Frequency | Alert |
|-------|-----------|-------|
| Last successful backup < 24h | Hourly | Warning if > 24h |
| WAL archiving active | Every 5 min | Critical if stalled |
| S3 connectivity | Every 5 min | Critical if unreachable |
| Backup integrity (test restore) | Weekly | Critical if corrupt |
| Disk space for backups | Hourly | Warning at 80% |

## Restore Process

1. **Select backup**: Choose full backup timestamp or named restore point.
2. **Stop services**: Scale down GGID pods.
3. **Restore PostgreSQL**: `pg_restore` from S3 + apply WAL to target timestamp.
4. **Verify**: Run integrity checks on restored data.
5. **Restart services**: Scale up pods.
6. **Validate**: Run health checks + smoke tests.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/admin/backup` | GET | List backups |
| `/api/v1/admin/backup` | POST | Trigger manual backup |
| `/api/v1/admin/backup/restore` | POST | Restore from backup |
| `/api/v1/admin/backup/dr-status` | GET | DR readiness status |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List backups
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/admin/backup" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Trigger manual backup
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/admin/backup" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Check DR readiness
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/admin/backup/dr-status" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Backups failing | S3 credentials or disk space | Verify S3 config; free disk space |
| WAL archiving stalled | `archive_command` error | Check PostgreSQL logs; verify S3 connectivity |
| Restore fails | Corrupt backup or version mismatch | Use earlier backup; verify PG version matches |
| DR status critical | Backup overdue or integrity check failed | Trigger manual backup; run integrity check |

## Best Practices

- **Test restores monthly**: A backup you can't restore is no backup.
- **Monitor WAL lag**: If WAL archiving falls behind, RPO increases.
- **Use named restore points**: Before migrations or risky changes, create a restore point.
- **Encrypt backups**: Enable S3 server-side encryption (SSE-KMS).
- **Cross-region replication**: For multi-region DR, replicate backups to a second region.
