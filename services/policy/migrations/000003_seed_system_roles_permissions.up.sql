-- Seed system roles and permissions for a default tenant.
-- This migration can be run per-tenant during provisioning.

-- System roles
INSERT INTO roles (tenant_id, key, name, description, system_role) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin', 'Administrator', 'Full system access', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'editor', 'Editor', 'Read and write access, no admin', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'viewer', 'Viewer', 'Read-only access', TRUE)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- System permissions
INSERT INTO permissions (tenant_id, key, name, resource_type, action, description, system_perm) VALUES
    ('00000000-0000-0000-0000-000000000001', 'iam:users:read',    'Read Users',    'users',    'read',   'View user profiles', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:users:write',   'Write Users',   'users',    'write',  'Create/update users', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:users:delete',  'Delete Users',  'users',    'delete', 'Delete users', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:roles:read',    'Read Roles',    'roles',    'read',   'View roles', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:roles:write',   'Write Roles',   'roles',    'write',  'Create/update roles', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:orgs:read',     'Read Orgs',     'organizations', 'read', 'View organizations', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:orgs:write',    'Write Orgs',    'organizations', 'write', 'Manage organizations', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:audit:read',    'Read Audit',    'audit',    'read',   'View audit logs', TRUE),
    ('00000000-0000-0000-0000-000000000001', 'iam:policies:write', 'Write Policies', 'policies', 'write', 'Manage policies', TRUE)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = '00000000-0000-0000-0000-000000000001'
      AND r.key = 'admin'
      AND p.tenant_id = '00000000-0000-0000-0000-000000000001'
ON CONFLICT DO NOTHING;

-- Assign read permissions to viewer role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = '00000000-0000-0000-0000-000000000001'
      AND r.key = 'viewer'
      AND p.tenant_id = '00000000-0000-0000-0000-000000000001'
      AND p.action = 'read'
ON CONFLICT DO NOTHING;

-- Assign read+write (non-admin) permissions to editor role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = '00000000-0000-0000-0000-000000000001'
      AND r.key = 'editor'
      AND p.tenant_id = '00000000-0000-0000-0000-000000000001'
      AND p.action IN ('read', 'write')
ON CONFLICT DO NOTHING;
