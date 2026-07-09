-- Identity Service: Initial Schema
-- Creates users, user_emails, user_external_identities, and email_verification_tokens tables
-- with PostgreSQL Row Level Security for multi-tenant isolation.

-- +migrate Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- === users ===
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL,
    username        VARCHAR(64) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    phone           VARCHAR(20) DEFAULT '',
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'locked', 'disabled', 'deleted')),
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    phone_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    primary_email_id UUID,
    display_name    VARCHAR(200) DEFAULT '',
    avatar_url      VARCHAR(500) DEFAULT '',
    locale          VARCHAR(10) DEFAULT 'en',
    timezone        VARCHAR(50) DEFAULT 'UTC',
    last_login_at   TIMESTAMPTZ,
    last_login_ip   INET,
    password_hash   TEXT DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,

    -- Unique username and email per tenant
    CONSTRAINT users_tenant_username_uk UNIQUE (tenant_id, username),
    CONSTRAINT users_tenant_email_uk UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users (created_at DESC);

-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- === Row Level Security ===
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON users
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- === user_emails ===
CREATE TABLE user_emails (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_emails_tenant_email_uk UNIQUE (email),
    CONSTRAINT user_emails_user_email_uk UNIQUE (user_id, email)
);

CREATE INDEX idx_user_emails_user ON user_emails (user_id);
CREATE INDEX idx_user_emails_primary ON user_emails (user_id, is_primary) WHERE is_primary = true;

ALTER TABLE user_emails ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_emails FORCE ROW LEVEL SECURITY;

CREATE POLICY user_emails_tenant_isolation ON user_emails
    FOR ALL
    USING (
        user_id IN (
            SELECT id FROM users
            WHERE tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
        )
    );

-- Trigger: sync users.email when primary email changes
CREATE OR REPLACE FUNCTION sync_primary_email()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users SET email = NEW.email WHERE id = NEW.user_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_emails_primary_sync
    AFTER UPDATE OF is_primary ON user_emails
    FOR EACH ROW
    WHEN (NEW.is_primary = true)
    EXECUTE FUNCTION sync_primary_email();

-- === user_external_identities ===
CREATE TABLE user_external_identities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        VARCHAR(50) NOT NULL,
    external_id     VARCHAR(255) NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    linked_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ext_identity_provider_uk UNIQUE (provider, external_id)
);

CREATE INDEX idx_ext_identity_user ON user_external_identities (user_id);
CREATE INDEX idx_ext_identity_provider ON user_external_identities (provider, external_id);

ALTER TABLE user_external_identities ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_external_identities FORCE ROW LEVEL SECURITY;

CREATE POLICY ext_identity_tenant_isolation ON user_external_identities
    FOR ALL
    USING (
        user_id IN (
            SELECT id FROM users
            WHERE tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
        )
    );

-- === email_verification_tokens ===
CREATE TABLE email_verification_tokens (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id        UUID NOT NULL REFERENCES user_emails(id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_ver_token_hash ON email_verification_tokens (token_hash)
    WHERE consumed_at IS NULL;

-- No RLS on tokens — they are consumed by hash, not by tenant lookup.

-- +migrate Down

DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS user_external_identities;
DROP TABLE IF EXISTS user_emails;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS update_updated_at() CASCADE;
DROP FUNCTION IF EXISTS sync_primary_email() CASCADE;
