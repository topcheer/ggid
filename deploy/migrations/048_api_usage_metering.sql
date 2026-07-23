-- 048_api_usage_metering.sql
-- Per-tenant API usage metering: records every proxied request's
-- method, path, status, and latency for billing and analytics.

CREATE TABLE IF NOT EXISTS api_usage_log (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   TEXT        NOT NULL,
    method      TEXT        NOT NULL,
    path        TEXT        NOT NULL,
    status_code INT         NOT NULL,
    latency_ms  INT         NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns: per-tenant aggregation + time-range scans.
CREATE INDEX IF NOT EXISTS idx_api_usage_log_tenant_time ON api_usage_log (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_log_created_at  ON api_usage_log (created_at DESC);

-- Auto-cleanup: retain 30 days (audit service retention job handles deletion).
