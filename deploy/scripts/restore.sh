#!/bin/bash
# GGID Database Restore Script
# Usage: ./restore.sh <backup_file> [--confirm]
# WARNING: This will DROP and recreate the database!

set -euo pipefail

BACKUP_FILE="${1:?Usage: restore.sh <backup_file> [--confirm]}"
CONFIRM="${2:-}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ggid}"
DB_USER="${DB_USER:-ggid}"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2; }

if [ ! -f "$BACKUP_FILE" ]; then
    log "ERROR: Backup file not found: $BACKUP_FILE"
    exit 1
fi

if [ "$CONFIRM" != "--confirm" ]; then
    log "WARNING: This will DROP and recreate database '$DB_NAME'"
    log "To proceed, run: $0 $BACKUP_FILE --confirm"
    exit 1
fi

# Verify checksum if available
CHECKSUM_FILE="${BACKUP_FILE%.sql.gz}.sha256"
if [ -f "$CHECKSUM_FILE" ]; then
    log "Verifying checksum..."
    if ! sha256sum -c "$CHECKSUM_FILE" >/dev/null 2>&1; then
        log "ERROR: Checksum verification failed"
        exit 1
    fi
    log "Checksum OK"
fi

BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
log "Restoring $BACKUP_FILE ($BACKUP_SIZE) to $DB_NAME at $DB_HOST:$DB_PORT"

# Drop and recreate database
log "Dropping existing database..."
PGPASSWORD="${DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" || true
PGPASSWORD="${DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"

# Restore from backup
log "Restoring data..."
PGPASSWORD="${DB_PASSWORD:-}" pg_restore \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --no-owner \
    --no-privileges \
    --clean \
    --if-exists \
    "$BACKUP_FILE"

log "Restore complete. Verifying..."
TABLE_COUNT=$(PGPASSWORD="${DB_PASSWORD:-}" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" | xargs)
log "Database has $TABLE_COUNT tables in public schema."
log "Done."
