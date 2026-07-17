-- 034_hr_connectors.sql
-- HR Connector Framework: sync log + connector config.

CREATE TABLE IF NOT EXISTS hr_connectors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL, -- workday, bamboohr, csv
    config          JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    last_sync_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS hr_sync_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id    UUID NOT NULL REFERENCES hr_connectors(id) ON DELETE CASCADE,
    source          TEXT NOT NULL,
    event_type      TEXT NOT NULL, -- hired, terminated, dept_change, manager_change
    employee_id     TEXT NOT NULL,
    ggid_user_id    TEXT,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, processed, failed
    details         JSONB DEFAULT '{}',
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_hr_sync_connector ON hr_sync_log(connector_id, synced_at DESC);
CREATE INDEX IF NOT EXISTS idx_hr_sync_status ON hr_sync_log(status);

COMMENT ON TABLE hr_connectors IS 'HR system connector configurations';
COMMENT ON TABLE hr_sync_log IS 'HR event sync audit log';
