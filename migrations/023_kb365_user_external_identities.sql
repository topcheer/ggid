-- KB-365: user_external_identities table for social account linking
-- Stores linked social identity providers (Google, GitHub, Apple, etc.)

CREATE TABLE IF NOT EXISTS user_external_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- google, github, apple, oidc, etc.
    external_id TEXT NOT NULL, -- provider-specific user ID
    metadata JSONB DEFAULT '{}',
    linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, provider, external_id),
    UNIQUE(user_id, provider) -- one link per provider per user
);

CREATE INDEX IF NOT EXISTS idx_ext_ident_user ON user_external_identities(user_id);
CREATE INDEX IF NOT EXISTS idx_ext_ident_provider ON user_external_identities(tenant_id, provider);
