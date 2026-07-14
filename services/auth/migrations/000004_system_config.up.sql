-- System configuration table for hot-reloadable runtime parameters.
-- Stores per-tenant overrides for auth policy, rate limits, etc.
CREATE TABLE IF NOT EXISTS system_config (
    tenant_id   UUID        NOT NULL,
    key         TEXT        NOT NULL,
    value       TEXT        NOT NULL,
    value_type  TEXT        NOT NULL DEFAULT 'string', -- string | int | bool | duration | float
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, key)
);

-- Seed test environment defaults (tenant 00000000-0000-0000-0000-000000000001)
INSERT INTO system_config (tenant_id, key, value, value_type) VALUES
    ('00000000-0000-0000-0000-000000000001', 'auth.max_attempts',             '9999',  'int'),
    ('00000000-0000-0000-0000-000000000001', 'auth.lock_duration',            '1s',    'duration'),
    ('00000000-0000-0000-0000-000000000001', 'auth.rate_limit_per_minute',    '9999',  'int'),
    ('00000000-0000-0000-0000-000000000001', 'gateway.rate_limit_tokens',     '99999', 'float'),
    ('00000000-0000-0000-0000-000000000001', 'gateway.rate_limit_refill_per_sec', '99999', 'float')
ON CONFLICT (tenant_id, key) DO NOTHING;
