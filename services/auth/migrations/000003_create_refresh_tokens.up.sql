-- refresh_tokens table: opaque refresh tokens with rotation support
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    session_id      UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    client_id       UUID,
    token_hash      VARCHAR(128) NOT NULL,          -- SHA-256 hash of the opaque token
    scope           TEXT[] DEFAULT ARRAY[]::TEXT[],
    expires_at      TIMESTAMPTZ NOT NULL,
    rotated_from    UUID,                            -- links to the previous token in rotation chain
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_refresh_tokens_hash   ON refresh_tokens(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_user   ON refresh_tokens(tenant_id, user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_session ON refresh_tokens(session_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE refresh_tokens IS 'Opaque refresh tokens stored in DB (also mirrored in Redis for fast lookup)';
COMMENT ON COLUMN refresh_tokens.rotated_from IS 'ID of the previous token this was rotated from (rotation chain)';
