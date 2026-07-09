-- Audit events table (monthly partitioned by created_at)
CREATE TABLE audit_events (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    actor_type      actor_type NOT NULL DEFAULT 'user',
    actor_id        UUID,
    actor_name      VARCHAR(200) DEFAULT '',
    action          VARCHAR(100) NOT NULL,
    resource_type   VARCHAR(50) DEFAULT '',
    resource_id     UUID,
    resource_name   VARCHAR(200) DEFAULT '',
    result          event_result NOT NULL DEFAULT 'success',
    ip_address      INET,
    user_agent      TEXT DEFAULT '',
    request_id      VARCHAR(64) DEFAULT '',
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create indexes on the parent table (applied to all partitions)
CREATE INDEX idx_audit_tenant_time ON audit_events(tenant_id, created_at DESC);
CREATE INDEX idx_audit_actor ON audit_events(actor_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit_events(tenant_id, action, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_events(tenant_id, resource_type, resource_id);
CREATE INDEX idx_audit_result ON audit_events(tenant_id, result, created_at DESC);
CREATE INDEX idx_audit_request_id ON audit_events(request_id);

-- Create initial monthly partitions for current year
-- This function creates partitions dynamically; called by a scheduled job in production.
DO $$
DECLARE
    month_start DATE;
    month_end DATE;
    partition_name TEXT;
    year INT;
    month INT;
BEGIN
    FOR year IN 2025..2026 LOOP
        FOR month IN 1..12 LOOP
            month_start := MAKE_DATE(year, month, 1);
            month_end := month_start + INTERVAL '1 month';
            partition_name := format('audit_events_%s_%02s', year, month);
            EXECUTE format(
                'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_events FOR VALUES FROM (%L) TO (%L)',
                partition_name, month_start, month_end
            );
        END LOOP;
    END LOOP;
END $$;

-- Enable RLS
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_audit ON audit_events
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
