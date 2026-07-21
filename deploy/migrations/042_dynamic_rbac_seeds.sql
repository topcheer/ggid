-- 042: Dynamic RBAC — seed route permissions matching legacy hardcoded behavior
-- ADR: docs/design/adr-dynamic-rbac.md
--
-- Grants admin-level route permissions to admin roles across all tenants so
-- the gateway's dynamic RBAC resolver preserves the behavior previously
-- enforced by the hardcoded adminPrefixes list. Rows are per-tenant because
-- roles are per-tenant; ON CONFLICT makes this idempotent.
--
-- NOTE: intentionally admin-roles-only. Granting editor/viewer API-level
-- read/write would WIDEN access beyond legacy behavior (static fallback
-- denied them) — that is a product decision to be made separately.

WITH admin_roles AS (
    SELECT id FROM roles
    WHERE key IN ('admin', 'platform:admin', 'tenant:admin')
       OR name IN ('Administrator', 'Platform Administrator', 'Tenant Administrator')
)
INSERT INTO role_route_permissions (role_id, route_prefix, permission_level)
SELECT id, prefix, 'admin'
FROM admin_roles
CROSS JOIN (VALUES
    ('/api/v1/users'),
    ('/api/v1/audit/'),
    ('/api/v1/policies'),
    ('/api/v1/webhooks'),
    ('/api/v1/oauth/clients'),
    ('/api/v1/roles'),
    ('/api/v1/admin/'),
    ('/api/v1/settings/'),
    ('/api/v1/system/'),
    ('/api/v1/tenants'),
    ('/api/v1/impersonate')
) AS t(prefix)
ON CONFLICT (role_id, route_prefix) DO NOTHING;
