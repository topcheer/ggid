-- Seed system roles and permissions for a default tenant.
-- This migration can be run per-tenant during provisioning.

-- System roles
INSERT INTO roles (tenant_id, key, name, description, system_role) VALUES
    ((SELECT id FROM tenants WHERE slug = 'default'), 'admin', 'Administrator', 'Full system access', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'editor', 'Editor', 'Read and write access, no admin', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'viewer', 'Viewer', 'Read-only access', TRUE)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- System permissions
INSERT INTO permissions (tenant_id, key, name, resource_type, action, description, system_perm) VALUES
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:users:read',    'Read Users',    'users',    'read',   'View user profiles', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:users:write',   'Write Users',   'users',    'write',  'Create/update users', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:users:delete',  'Delete Users',  'users',    'delete', 'Delete users', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:roles:read',    'Read Roles',    'roles',    'read',   'View roles', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:roles:write',   'Write Roles',   'roles',    'write',  'Create/update roles', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:orgs:read',     'Read Orgs',     'organizations', 'read', 'View organizations', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:orgs:write',    'Write Orgs',    'organizations', 'write', 'Manage organizations', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:audit:read',    'Read Audit',    'audit',    'read',   'View audit logs', TRUE),
    ((SELECT id FROM tenants WHERE slug = 'default'), 'iam:policies:write', 'Write Policies', 'policies', 'write', 'Manage policies', TRUE)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND r.key = 'admin'
      AND p.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
ON CONFLICT DO NOTHING;

-- Assign read permissions to viewer role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND r.key = 'viewer'
      AND p.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND p.action = 'read'
ON CONFLICT DO NOTHING;

-- Assign read+write (non-admin) permissions to editor role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND r.key = 'editor'
      AND p.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND p.action IN ('read', 'write')
ON CONFLICT DO NOTHING;
