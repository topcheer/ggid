#!/bin/bash
set -euo pipefail

# GGID Database Backup Verification Script
# Usage: ./backup-verify.sh --source-host <pg-host> --target-host <temp-host> [options]
# Restores the latest backup to a temporary instance and verifies data integrity.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Defaults
SOURCE_HOST=""
TARGET_HOST=""
SOURCE_PORT=5432
TARGET_PORT=5432
DB_NAME="ggid"
DB_USER="ggid"
DB_PASSWORD=""
DUMP_FILE="/tmp/ggid-backup-verify-$$.sql"
TABLE_LIST="users,roles,organizations,api_keys,audit_logs,oauth_clients,policies,refresh_tokens"
SLACK_WEBHOOK=""

usage() {
    cat <<EOF
Usage: $0 --source-host <host> --target-host <host> [options]

Options:
  --source-host HOST     Source PostgreSQL host (required)
  --target-host HOST     Temporary restore host (required)
  --source-port PORT     Source port (default: 5432)
  --target-port PORT     Target port (default: 5432)
  --db-name NAME         Database name (default: ggid)
  --db-user USER         Database user (default: ggid)
  --db-password PASS     Database password (from env PG_PASSWORD if not set)
  --tables LIST          Comma-separated tables to verify (default: core tables)
  --slack-webhook URL    Slack webhook for failure alerts
  -h, --help             Show this help
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --source-host)  SOURCE_HOST="$2"; shift 2 ;;
        --target-host)  TARGET_HOST="$2"; shift 2 ;;
        --source-port)  SOURCE_PORT="$2"; shift 2 ;;
        --target-port)  TARGET_PORT="$2"; shift 2 ;;
        --db-name)      DB_NAME="$2"; shift 2 ;;
        --db-user)      DB_USER="$2"; shift 2 ;;
        --db-password)  DB_PASSWORD="$2"; shift 2 ;;
        --tables)       TABLE_LIST="$2"; shift 2 ;;
        --slack-webhook) SLACK_WEBHOOK="$2"; shift 2 ;;
        -h|--help)      usage ;;
        *)              echo "Unknown option: $1"; usage ;;
    esac
done

DB_PASSWORD="${DB_PASSWORD:-${PG_PASSWORD:-}}"
if [[ -z "$DB_PASSWORD" ]]; then echo "ERROR: PG_PASSWORD not set"; exit 1; fi
if [[ -z "$SOURCE_HOST" || -z "$TARGET_HOST" ]]; then echo "ERROR: --source-host and --target-host required"; usage; fi

export PGPASSWORD="$DB_PASSWORD"

alert() {
    local msg="$1"
    echo "ALERT: $msg" >&2
    if [[ -n "$SLACK_WEBHOOK" ]]; then
        curl -s -X POST "$SLACK_WEBHOOK" -H 'Content-Type: application/json' \
            -d "{\"text\":\"GGID Backup Verification FAILED: $msg\"}" || true
    fi
}

cleanup() {
    rm -f "$DUMP_FILE"
}
trap cleanup EXIT

echo "=== GGID Backup Verification ==="
echo "Source: $SOURCE_HOST:$SOURCE_PORT"
echo "Target: $TARGET_HOST:$TARGET_PORT"
echo "Database: $DB_NAME"
echo ""

# Step 1: Dump from source
echo "[1/4] Dumping from source..."
if ! pg_dump -h "$SOURCE_HOST" -p "$SOURCE_PORT" -U "$DB_USER" -d "$DB_NAME" -F p -f "$DUMP_FILE"; then
    alert "pg_dump from $SOURCE_HOST failed"
    exit 1
fi
echo "  Dump size: $(du -sh "$DUMP_FILE" | cut -f1)"

# Step 2: Restore to target
echo "[2/4] Restoring to target..."
if ! psql -h "$TARGET_HOST" -p "$TARGET_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$DUMP_FILE" >/dev/null 2>&1; then
    alert "psql restore to $TARGET_HOST failed"
    exit 1
fi
echo "  Restore complete"

# Step 3: Row count verification
echo "[3/4] Verifying row counts..."
IFS=',' read -ra TABLES <<< "$TABLE_LIST"
VERIFY_FAILED=0

for table in "${TABLES[@]}"; do
    table=$(echo "$table" | xargs) # trim whitespace
    SOURCE_COUNT=$(psql -h "$SOURCE_HOST" -p "$SOURCE_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null | xargs)
    TARGET_COUNT=$(psql -h "$TARGET_HOST" -p "$TARGET_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null | xargs)

    if [[ "$SOURCE_COUNT" == "$TARGET_COUNT" ]]; then
        echo "  ✅ $table: $SOURCE_COUNT rows (match)"
    else
        echo "  ❌ $table: source=$SOURCE_COUNT target=$TARGET_COUNT (MISMATCH)"
        VERIFY_FAILED=1
    fi
done

# Step 4: Checksum verification (sample critical table)
echo "[4/4] Checksum verification..."
CHECKSUM_SOURCE=$(psql -h "$SOURCE_HOST" -p "$SOURCE_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
    "SELECT md5(string_agg(id::text, ',' ORDER BY id)) FROM users;" 2>/dev/null | xargs)
CHECKSUM_TARGET=$(psql -h "$TARGET_HOST" -p "$TARGET_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
    "SELECT md5(string_agg(id::text, ',' ORDER BY id)) FROM users;" 2>/dev/null | xargs)

if [[ "$CHECKSUM_SOURCE" == "$CHECKSUM_TARGET" ]]; then
    echo "  ✅ users table checksum: match"
else
    echo "  ❌ users table checksum: MISMATCH"
    VERIFY_FAILED=1
fi

echo ""
if [[ $VERIFY_FAILED -eq 0 ]]; then
    echo "✅ Backup verification PASSED — all checks passed"
    exit 0
else
    alert "Row count or checksum mismatch detected"
    echo "❌ Backup verification FAILED"
    exit 1
fi
