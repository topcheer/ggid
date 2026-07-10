-- WebAuthn credentials table for Passkey registration.
-- Stores public key credentials registered via the WebAuthn API.
CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    name            TEXT,
    credential_id   BYTEA NOT NULL,
    public_key      BYTEA NOT NULL,
    transports      TEXT[] DEFAULT '{}',
    counter         BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ,
    UNIQUE(tenant_id, credential_id)
);

-- Row Level Security
ALTER TABLE webauthn_credentials ENABLE ROW LEVEL SECURITY;
CREATE POLICY webauthn_credentials_tenant_isolation ON webauthn_credentials
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Indexes
CREATE INDEX idx_webauthn_creds_user ON webauthn_credentials(tenant_id, user_id);
CREATE INDEX idx_webauthn_creds_cred_id ON webauthn_credentials(tenant_id, credential_id);
