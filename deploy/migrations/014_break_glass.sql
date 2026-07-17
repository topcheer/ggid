-- 014_break_glass.sql
-- Migrate break-glass records from in-memory array to Postgres.
-- Tracks emergency access activations for SOC audit trail.

CREATE TABLE IF NOT EXISTS break_glass_records (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    requester         UUID NOT NULL,
    requester_name    TEXT,
    reason            TEXT NOT NULL,
    scope             TEXT NOT NULL DEFAULT '',
    duration_minutes  INT NOT NULL DEFAULT 60,
    activated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at    TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_break_glass_tenant_time ON break_glass_records (tenant_id, activated_at DESC);
CREATE INDEX IF NOT EXISTS idx_break_glass_status      ON break_glass_records (tenant_id, status) WHERE status = 'active';

COMMENT ON TABLE break_glass_records IS 'Emergency break-glass access activations for SOC audit';
