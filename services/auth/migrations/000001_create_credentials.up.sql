-- credentials table: stores authentication credentials (passwords, etc.)
CREATE TABLE IF NOT EXISTS credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'password',
    identifier      VARCHAR(255) NOT NULL,        -- username or credential_id
    secret          TEXT NOT NULL,                 -- Argon2id hash for passwords
    metadata        JSONB DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ
);

-- Password history for reuse prevention
CREATE TABLE IF NOT EXISTS credential_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    secret          TEXT NOT NULL,                 -- previous password hash
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_credentials_tenant_user ON credentials(tenant_id, user_id);
CREATE INDEX idx_credentials_identifier   ON credentials(tenant_id, identifier);
CREATE UNIQUE INDEX uq_credentials_tenant_identifier_type ON credentials(tenant_id, identifier, type);
CREATE INDEX idx_cred_history_user       ON credential_history(tenant_id, user_id, created_at DESC);

COMMENT ON TABLE credentials IS 'Authentication credentials (password, passkey, etc.)';
COMMENT ON COLUMN credentials.secret IS 'Argon2id password hash or encrypted secret';
