DROP POLICY IF EXISTS tenant_isolation_memberships ON memberships;
DROP POLICY IF EXISTS tenant_isolation_orgs ON organizations;

ALTER TABLE memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE organizations DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS tenants;
