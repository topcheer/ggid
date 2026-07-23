-- 047: API keys table for programmatic access (Argon2id hashed)
CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

COMMENT ON TABLE api_keys IS 'API keys for programmatic access (Argon2id hashed secrets)';
