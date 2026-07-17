-- 036_rls_tenant_isolation.sql
-- Row-Level Security for multi-tenant isolation (defense-in-depth).
-- Applied to all tables with tenant_id column.

-- Helper function: safely enable RLS on a table.
DO $$
DECLARE
    tbl TEXT;
    rls_tables TEXT[] := ARRAY[
        'users', 'groups', 'group_members', 'roles', 'user_roles',
        'sessions', 'audit_events', 'oauth_clients', 'oauth_tokens',
        'policies', 'risk_scores', 'threat_indicators', 'consent_records',
        'scim_targets', 'device_posture_scores', 'soar_playbooks',
        'soar_executions', 'hr_connectors', 'hr_sync_log',
        'device_certificates', 'encrypted_fields', 'user_behavioral_baselines',
        'wasm_plugins', 'policy_decisions', 'risk_policies'
    ];
BEGIN
    FOREACH tbl IN ARRAY rls_tables LOOP
        BEGIN
            EXECUTE format('ALTER TABLE IF EXISTS %I ENABLE ROW LEVEL SECURITY', tbl);
            EXECUTE format('ALTER TABLE IF EXISTS %I FORCE ROW LEVEL SECURITY', tbl);
            EXECUTE format(
                'DROP POLICY IF EXISTS tenant_isolation ON %I', tbl
            );
            EXECUTE format(
                'CREATE POLICY tenant_isolation ON %I USING (tenant_id::text = current_setting(''app.tenant_id'', true))',
                tbl
            );
        EXCEPTION WHEN OTHERS THEN
            -- Table doesn't exist or already has RLS — skip.
        END;
    END LOOP;
END $$;

-- Create a role that bypasses RLS for admin operations.
-- Note: actual role creation requires superuser; this is documentation.
-- CREATE ROLE ggid_admin BYPASSRLS;

COMMENT ON SCHEMA public IS 'RLS enabled on tenant tables via migration 036';
