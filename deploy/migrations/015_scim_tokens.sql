-- 015_scim_tokens.sql
-- SCIM bearer tokens for IdP-driven provisioning (Okta/Entra/Google).
-- Token format: ggid_scim_<base64url(32 bytes)>
-- Hash: Argon2id, plaintext returned only once at creation.

CREATE TABLE IF NOT EXISTS scim_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    name          TEXT NOT NULL,
    token_hash    TEXT NOT NULL,
    scopes        TEXT[] NOT NULL DEFAULT '{scim}',
    expires_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ,
    created_by    UUID NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scim_tokens_tenant  ON scim_tokens (tenant_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scim_tokens_hash   ON scim_tokens (token_hash) WHERE revoked_at IS NULL;

COMMENT ON TABLE scim_tokens IS 'SCIM bearer tokens for IdP provisioning integration';
