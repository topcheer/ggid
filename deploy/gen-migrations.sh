#!/bin/bash
# Generate combined migration SQL from all service migrations
set -e
cd "$(dirname "$0")/.."

mkdir -p deploy/migrations

{
  sed '/-- +migrate Down/,$d' services/org/migrations/000001_init_extensions.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/org/migrations/000002_create_org_tables.up.sql
  echo ""
  sed '/-- +migrate Down/,$d' services/policy/migrations/000001_init_extensions.up.sql
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
  sed '/-- +migrate Down/,$d' services/audit/migrations/000001_init_extensions.up.sql
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
} > deploy/migrations/01_all_up.sql

wc -l deploy/migrations/01_all_up.sql
