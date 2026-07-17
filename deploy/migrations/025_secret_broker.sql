-- 025_secret_broker.sql
-- Zero-Trust Secret Brokering: targets + grants.

CREATE TABLE IF NOT EXISTS secret_targets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    name              TEXT NOT NULL,
    type              TEXT NOT NULL, -- db, ssh, cloud, api_key
    connection_config JSONB NOT NULL DEFAULT '{}',
    ttl_seconds       INT NOT NULL DEFAULT 3600,
    default_role      TEXT NOT NULL DEFAULT '',
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_secret_targets_tenant ON secret_targets(tenant_id, enabled);

CREATE TABLE IF NOT EXISTS secret_grants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    target_id   UUID NOT NULL REFERENCES secret_targets(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT '',
    credential  TEXT NOT NULL, -- encrypted short-lived credential
    jit_request_id UUID,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_secret_grants_tenant ON secret_grants(tenant_id, expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_secret_grants_user ON secret_grants(user_id, revoked);
CREATE INDEX IF NOT EXISTS idx_secret_grants_target ON secret_grants(target_id, revoked);

COMMENT ON TABLE secret_targets IS 'Zero-Trust secret broker targets (DB/SSH/cloud/API)';
COMMENT ON TABLE secret_grants IS 'Short-lived dynamic credential grants';
