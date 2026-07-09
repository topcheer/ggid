-- sessions table: tracks active user sessions
CREATE TABLE IF NOT EXISTS sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    token_hash      VARCHAR(128) NOT NULL,          -- session token SHA-256 hash
    device_info     JSONB DEFAULT '{}',
    ip_address      INET,
    user_agent      TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata        JSONB DEFAULT '{}'              -- MFA verified, auth context, etc.
);

-- Indexes
CREATE INDEX idx_sessions_user       ON sessions(tenant_id, user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_expires    ON sessions(expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE sessions IS 'Active and historical user sessions';
COMMENT ON COLUMN sessions.token_hash IS 'SHA-256 hash of the opaque session token';
