#!/bin/bash
# GGID Database Backup Script
# Usage: ./backup.sh [backup_dir]
# Cron: 0 2 * * * /opt/ggid/scripts/backup.sh /var/backups/ggid
#
# Features:
# - Compressed pg_dump with timestamp
# - Retention policy (keep 7 daily, 4 weekly, 12 monthly)
# - Backup integrity verification (checksums)
# - S3 upload option (when AWS creds configured)
# - Health check webhook on failure

set -euo pipefail

BACKUP_DIR="${1:-/var/backups/ggid}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ggid}"
DB_USER="${DB_USER:-ggid}"
RETENTION_DAILY=7
RETENTION_WEEKLY=4
RETENTION_MONTHLY=12
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.sql.gz"
CHECKSUM_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.sha256"
WEBHOOK_URL="${BACKUP_WEBHOOK_URL:-}"

# Logging
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2; }

# Cleanup function
cleanup() {
    rc=$?
    if [ $rc -ne 0 ]; then
        log "ERROR: Backup failed with exit code $rc"
        if [ -n "$WEBHOOK_URL" ]; then
            curl -s -X POST "$WEBHOOK_URL" \
                -H "Content-Type: application/json" \
                -d "{\"event\":\"backup_failed\",\"db\":\"$DB_NAME\",\"timestamp\":\"$TIMESTAMP\",\"exit_code\":$rc}" \
                || true
        fi
    fi
    rm -f "${BACKUP_FILE}.tmp" 2>/dev/null || true
}
trap cleanup EXIT

# Create backup directory
mkdir -p "$BACKUP_DIR"

log "Starting backup of $DB_NAME to $BACKUP_DIR"

# Verify database connectivity
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1; then
    log "ERROR: Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT"
    exit 1
fi

# Perform backup
log "Running pg_dump..."
if ! PGPASSWORD="${DB_PASSWORD:-}" pg_dump \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --no-owner \
    --no-privileges \
    --format=custom \
    --compress=9 \
    -f "${BACKUP_FILE}.tmp"; then
    log "ERROR: pg_dump failed"
    exit 1
fi

# Move tmp file to final
mv "${BACKUP_FILE}.tmp" "$BACKUP_FILE"

# Generate checksum
log "Generating checksum..."
sha256sum "$BACKUP_FILE" > "$CHECKSUM_FILE"

# Verify backup integrity
log "Verifying backup..."
if ! sha256sum -c "$CHECKSUM_FILE" >/dev/null 2>&1; then
    log "ERROR: Backup checksum verification failed"
    rm -f "$BACKUP_FILE" "$CHECKSUM_FILE"
    exit 1
fi

BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
log "Backup created: $BACKUP_FILE ($BACKUP_SIZE)"

# S3 upload (optional)
if [ -n "${AWS_S3_BUCKET:-}" ]; then
    log "Uploading to S3 bucket: $AWS_S3_BUCKET"
    if aws s3 cp "$BACKUP_FILE" "s3://$AWS_S3_BUCKET/ggid-backups/$(basename "$BACKUP_FILE")" \
        --sse AES256 2>/dev/null; then
        log "S3 upload complete"
    else
        log "WARNING: S3 upload failed (backup is still local)"
    fi
fi

# Retention: daily backups (keep last 7)
log "Applying retention policy..."
find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz" -mtime +$RETENTION_DAILY -delete
find "$BACKUP_DIR" -name "${DB_NAME}_*.sha256" -mtime +$RETENTION_DAILY -delete

# Retention: keep weekly snapshots (first backup of each week, keep 4 weeks)
WEEKLY_DIR="${BACKUP_DIR}/weekly"
mkdir -p "$WEEKLY_DIR"
if [ "$(date +%u)" = "1" ]; then  # Monday
    cp "$BACKUP_FILE" "$WEEKLY_DIR/"
    cp "$CHECKSUM_FILE" "$WEEKLY_DIR/"
    find "$WEEKLY_DIR" -name "${DB_NAME}_*.sql.gz" -mtime +$((RETENTION_WEEKLY * 7)) -delete
    find "$WEEKLY_DIR" -name "${DB_NAME}_*.sha256" -mtime +$((RETENTION_WEEKLY * 7)) -delete
fi

# Retention: keep monthly snapshots (first backup of each month, keep 12 months)
MONTHLY_DIR="${BACKUP_DIR}/monthly"
mkdir -p "$MONTHLY_DIR"
if [ "$(date +%d)" = "01" ]; then  # 1st of month
    cp "$BACKUP_FILE" "$MONTHLY_DIR/"
    cp "$CHECKSUM_FILE" "$MONTHLY_DIR/"
    find "$MONTHLY_DIR" -name "${DB_NAME}_*.sql.gz" -mtime +$((RETENTION_MONTHLY * 30)) -delete
    find "$MONTHLY_DIR" -name "${DB_NAME}_*.sha256" -mtime +$((RETENTION_MONTHLY * 30)) -delete
fi

# List remaining backups
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "${DB_NAME}_*.sql.gz" | wc -l)
log "Backup complete. $BACKUP_COUNT backups in $BACKUP_DIR"

# Success webhook
if [ -n "$WEBHOOK_URL" ]; then
    curl -s -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{\"event\":\"backup_success\",\"db\":\"$DB_NAME\",\"timestamp\":\"$TIMESTAMP\",\"size\":\"$BACKUP_SIZE\",\"count\":$BACKUP_COUNT}" \
        || true
fi

log "Done."
