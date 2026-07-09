#!/bin/bash
# GGID Database Initialization Script
# Runs all migration UP sections, stripping any embedded DOWN sections.
set -euo pipefail

DB_URL="${1:-${GGID_DATABASE_URL:-postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable}}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

MIGRATIONS=(
    "services/org/migrations/000001_init_extensions.up.sql"
    "services/org/migrations/000002_create_org_tables.up.sql"
    "services/policy/migrations/000001_init_extensions.up.sql"
    "services/policy/migrations/000002_create_policy_tables.up.sql"
    "services/identity/migrations/000001_initial_schema.up.sql"
    "services/auth/migrations/000001_create_credentials.up.sql"
    "services/auth/migrations/000002_create_sessions.up.sql"
    "services/auth/migrations/000003_create_refresh_tokens.up.sql"
    "services/audit/migrations/000001_init_extensions.up.sql"
    "services/audit/migrations/000002_create_partitions.up.sql"
    "services/policy/migrations/000003_seed_system_roles_permissions.up.sql"
    "services/oauth/migrations/000001_initial_schema.up.sql"
    "services/auth/migrations/000001_mfa_devices.up.sql"
)

echo "=== GGID Database Migration ==="
echo "Database: $DB_URL"
echo ""

for migration in "${MIGRATIONS[@]}"; do
    file="$PROJECT_ROOT/$migration"
    if [ ! -f "$file" ]; then
        echo "SKIP: $migration (file not found)"
        continue
    fi
    echo "RUN:  $migration"
    # Strip everything from "-- +migrate Down" onwards (inclusive)
    sed '/-- +migrate Down/,$d' "$file" | psql "$DB_URL" -v ON_ERROR_STOP=1 -q
done

echo ""
echo "=== Seeding default tenant ==="
psql "$DB_URL" -v ON_ERROR_STOP=1 -c "
INSERT INTO tenants (id, name, slug, plan, status, max_users)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default', 'enterprise', 'active', 10000)
ON CONFLICT (slug) DO NOTHING;
" -q

echo ""
echo "=== Migration complete ==="
