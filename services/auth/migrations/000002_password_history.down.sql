-- Reverse of 000002_password_history.up.sql
DROP POLICY IF EXISTS password_history_tenant_isolation ON password_history;
ALTER TABLE password_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE password_history NO FORCE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS password_history;
