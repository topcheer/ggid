-- 023_dlp_policies.sql
-- Data Loss Prevention: policies + event log.

CREATE TABLE IF NOT EXISTS dlp_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    trigger         TEXT NOT NULL,
    conditions      JSONB NOT NULL DEFAULT '{}',
    action          TEXT NOT NULL DEFAULT 'log',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dlp_policies_tenant ON dlp_policies(tenant_id, enabled);

CREATE TABLE IF NOT EXISTS dlp_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    policy_id           UUID,
    user_id             TEXT,
    user_name           TEXT,
    trigger             TEXT NOT NULL,
    resource_type       TEXT,
    data_classification TEXT,
    action_taken        TEXT NOT NULL,
    reason              TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dlp_events_tenant ON dlp_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dlp_events_action ON dlp_events(tenant_id, action_taken);

COMMENT ON TABLE dlp_policies IS 'DLP policy definitions';
COMMENT ON TABLE dlp_events IS 'DLP enforcement event log';
