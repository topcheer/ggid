#!/bin/bash
# Seed script: creates admin user, system roles, and permission assignments
# Usage: bash deploy/seed.sh
set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-ggid}"
DB_PASSWORD="${DB_PASSWORD:-ggid}"
DB_NAME="${DB_NAME:-ggid}"

TENANT="00000000-0000-0000-0000-000000000001"

echo "=== GGID Seed Data ==="

# 1. Ensure default tenant exists
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
  "INSERT INTO tenants (id, name, slug, plan, status, max_users)
   VALUES ('$TENANT', 'Default', 'default', 'enterprise', 'active', 100000)
   ON CONFLICT (slug) DO NOTHING" 2>/dev/null || true

# 2. Ensure system roles exist
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
  "INSERT INTO roles (tenant_id, key, name, description, system_role)
   VALUES
     ('$TENANT', 'admin', 'Administrator', 'Full system access', true),
     ('$TENANT', 'manager', 'Manager', 'Manage users and roles', true),
     ('$TENANT', 'user', 'User', 'Basic user access', true)
   ON CONFLICT (tenant_id, key) DO NOTHING" 2>/dev/null || true

# 3. Ensure system permissions exist
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
  "INSERT INTO permissions (tenant_id, key, name, description)
   VALUES
     ('$TENANT', 'iam:users:read', 'Read Users', 'View user profiles'),
     ('$TENANT', 'iam:users:write', 'Write Users', 'Create and edit users'),
     ('$TENANT', 'iam:users:delete', 'Delete Users', 'Delete user accounts'),
     ('$TENANT', 'iam:roles:read', 'Read Roles', 'View roles and permissions'),
     ('$TENANT', 'iam:roles:write', 'Write Roles', 'Create and edit roles'),
     ('$TENANT', 'iam:orgs:read', 'Read Orgs', 'View organizations'),
     ('$TENANT', 'iam:orgs:write', 'Write Orgs', 'Create and edit organizations'),
     ('$TENANT', 'iam:audit:read', 'Read Audit', 'View audit logs'),
     ('$TENANT', 'iam:policies:write', 'Write Policies', 'Manage authorization policies'),
     ('$TENANT', 'iam:settings:read', 'Read Settings', 'View system settings'),
     ('$TENANT', 'iam:settings:write', 'Write Settings', 'Modify system settings')
   ON CONFLICT (tenant_id, key) DO NOTHING" 2>/dev/null || true

# 4. Assign all permissions to admin role
ADMIN_ROLE_ID=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
  "SELECT id FROM roles WHERE key = 'admin' AND tenant_id = '$TENANT'" 2>/dev/null | tr -d ' \n')

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c \
  "INSERT INTO role_permissions (tenant_id, role_id, permission_id)
   SELECT '$TENANT', '$ADMIN_ROLE_ID', id FROM permissions WHERE tenant_id = '$TENANT'
   ON CONFLICT DO NOTHING" 2>/dev/null || true

echo "=== Seed data complete ==="
echo "Admin user must be created via: POST /api/v1/auth/register"
echo '  {"username":"admin","email":"admin@ggid.dev","password":"$ADMIN_PASSWORD","name":"System Administrator"}'
echo ""
echo "Then assign admin role:"
echo "  See deploy/seed.sh for SQL"
