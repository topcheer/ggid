-- 044: Task-E — refresh token rotation families (RFC 6749 §10.4)
-- Adds family_id to oidc_refresh_tokens so reuse detection can revoke
-- exactly one token family instead of the client's entire token set.

ALTER TABLE oidc_refresh_tokens ADD COLUMN IF NOT EXISTS family_id TEXT;
CREATE INDEX IF NOT EXISTS idx_oidc_refresh_tokens_family ON oidc_refresh_tokens(family_id);

-- Family registry (JSONB, shared with the /token-families API view).
CREATE TABLE IF NOT EXISTS oauth_token_families (
    id         TEXT PRIMARY KEY,
    data       JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
