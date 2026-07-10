#!/bin/bash
# Generate combined migration SQL from all service migrations.
# Output is made idempotent so the Docker migrate init container can run
# safely against an already-initialized database.
set -e
cd "$(dirname "$0")/.."

mkdir -p deploy/migrations

{
  echo "-- GGID Combined Migration (auto-generated, idempotent)"
  echo ""
  sed '/-- +migrate Down/,$d' services/org/migrations/000001_init_extensions.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/org/migrations/000002_create_org_tables.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/policy/migrations/000001_init_extensions.up.sql | grep -v "CREATE EXTENSION" || true
  echo ""
  sed '/-- +migrate Down/,$d' services/policy/migrations/000002_create_policy_tables.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/identity/migrations/000001_initial_schema.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/auth/migrations/000001_create_credentials.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/auth/migrations/000002_create_sessions.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/auth/migrations/000003_create_refresh_tokens.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/audit/migrations/000001_init_extensions.up.sql | grep -v "CREATE EXTENSION" || true
  echo ""
  sed '/-- +migrate Down/,$d' services/audit/migrations/000002_create_partitions.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/policy/migrations/000003_seed_system_roles_permissions.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/oauth/migrations/000001_initial_schema.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/auth/migrations/000001_mfa_devices.up.sql 2>/dev/null || true
  echo ""
  echo "INSERT INTO tenants (id, name, slug, plan, status, max_users)"
  echo "VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default', 'enterprise', 'active', 10000)"
  echo "ON CONFLICT (slug) DO NOTHING;"
} | python3 -c "
import sys, re
sql = sys.stdin.read()

# CREATE TYPE -> wrap in DO block for idempotency
def make_type_idempotent(m):
    return 'DO \$\$ BEGIN ' + m.group(0) + ' EXCEPTION WHEN duplicate_object THEN NULL; END \$\$;'
sql = re.sub(r'CREATE TYPE \S+ AS ENUM \([^)]*\);', make_type_idempotent, sql, flags=re.DOTALL)

# CREATE TABLE -> CREATE TABLE IF NOT EXISTS
sql = re.sub(r'CREATE TABLE (?!IF NOT EXISTS)', 'CREATE TABLE IF NOT EXISTS ', sql)

# CREATE INDEX -> CREATE INDEX IF NOT EXISTS (handles UNIQUE too)
sql = re.sub(r'CREATE (UNIQUE )?INDEX (?!IF NOT EXISTS)', lambda m: f'CREATE {m.group(1) or \"\"}INDEX IF NOT EXISTS ', sql)

# CREATE SEQUENCE -> CREATE SEQUENCE IF NOT EXISTS
sql = re.sub(r'CREATE SEQUENCE (?!IF NOT EXISTS)', 'CREATE SEQUENCE IF NOT EXISTS ', sql)

sys.stdout.write(sql)
" > deploy/migrations/01_all_up.sql

wc -l deploy/migrations/01_all_up.sql
