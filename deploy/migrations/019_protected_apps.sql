-- 019_protected_apps.sql
-- Access Broker / ZTNA: protected applications + access logs.

CREATE TABLE IF NOT EXISTS protected_apps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    upstream_url        TEXT NOT NULL,
    icon                TEXT,
    description         TEXT,
    auth_mode           TEXT NOT NULL DEFAULT 'jwt',
    access_policy       JSONB NOT NULL DEFAULT '{}',
    inject_headers      JSONB NOT NULL DEFAULT '[]',
    health_check_path   TEXT DEFAULT '/health',
    health_check_interval INT DEFAULT 30,
    health_status       TEXT DEFAULT 'unknown',
    rate_limit_per_min  INT DEFAULT 100,
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_by          UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_protected_apps_slug ON protected_apps(tenant_id, slug) WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS app_access_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    app_id            UUID NOT NULL,
    user_id           UUID,
    user_name         TEXT,
    method            TEXT NOT NULL,
    path              TEXT NOT NULL,
    status_code       INT NOT NULL,
    response_time_ms  INT,
    ip_address        TEXT,
    user_agent        TEXT,
    pdp_decision      TEXT,
    pdp_reason        TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_app_logs_app_time ON app_access_logs(tenant_id, app_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_logs_user ON app_access_logs(tenant_id, user_id, created_at DESC);

COMMENT ON TABLE protected_apps IS 'ZTNA Access Broker protected applications';
COMMENT ON TABLE app_access_logs IS 'Access Broker per-request audit logs';
