DROP POLICY IF EXISTS tenant_isolation_audit ON audit_events;
ALTER TABLE audit_events DISABLE ROW LEVEL SECURITY;

-- Drop all partitions
DO $$
DECLARE
    part RECORD;
BEGIN
    FOR part IN SELECT inhrelid::regclass::text AS name FROM pg_inherits WHERE inhparent = 'audit_events'::regclass LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I', part.name);
    END LOOP;
END $$;

DROP TABLE IF EXISTS audit_events;
