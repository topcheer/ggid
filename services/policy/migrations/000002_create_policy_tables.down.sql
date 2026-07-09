DROP POLICY IF EXISTS tenant_isolation_policies ON policies;
DROP POLICY IF EXISTS tenant_isolation_permissions ON permissions;
DROP POLICY IF EXISTS tenant_isolation_roles ON roles;

ALTER TABLE policies DISABLE ROW LEVEL SECURITY;
ALTER TABLE permissions DISABLE ROW LEVEL SECURITY;
ALTER TABLE roles DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS policy_attachments;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
