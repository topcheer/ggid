-- Migration 05: WebAuthn credentials table
-- Stores passkey/WebAuthn credentials for passwordless authentication.

CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    name            TEXT NOT NULL DEFAULT '',
    credential_id   BYTEA NOT NULL,
    public_key      BYTEA NOT NULL,
    transports      TEXT[] NOT NULL DEFAULT '{}',
    counter         INTEGER NOT NULL DEFAULT 0,
    aaguid          UUID,
    attestation_type TEXT NOT NULL DEFAULT 'none',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE
);

-- Indexes for efficient lookups.
CREATE INDEX IF NOT EXISTS idx_webauthn_tenant_user ON webauthn_credentials (tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_credential_id ON webauthn_credentials (credential_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_webauthn_cred_unique ON webauthn_credentials (tenant_id, credential_id);
