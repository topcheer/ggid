-- OAuth/OIDC Service: Initial Schema
-- Creates oauth_clients, oauth_authorization_codes, and oidc_id_tokens tables.

-- +migrate Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- === oauth_clients ===
CREATE TABLE oauth_clients (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id                   UUID NOT NULL,
    client_id                   VARCHAR(64) NOT NULL UNIQUE,
    client_secret_hash          TEXT NOT NULL DEFAULT '',
    name                        VARCHAR(100) NOT NULL,
    type                        VARCHAR(20) NOT NULL DEFAULT 'confidential'
                                CHECK (type IN ('confidential', 'public')),
    grant_types                 TEXT[] NOT NULL DEFAULT '{authorization_code}',
    response_types              TEXT[] NOT NULL DEFAULT '{code}',
    redirect_uris               TEXT[] NOT NULL DEFAULT '{}',
    scopes                      TEXT[] NOT NULL DEFAULT '{openid,profile,email}',
    token_endpoint_auth_method  VARCHAR(50) NOT NULL DEFAULT 'client_secret_basic',
    metadata                    JSONB NOT NULL DEFAULT '{}',
    enabled                     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT oauth_clients_tenant_client_uk UNIQUE (tenant_id, client_id)
);

CREATE INDEX idx_oauth_clients_tenant ON oauth_clients (tenant_id) WHERE enabled = true;

-- updated_at trigger
CREATE TRIGGER oauth_clients_updated_at
    BEFORE UPDATE ON oauth_clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- RLS
ALTER TABLE oauth_clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_clients FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_clients_tenant_isolation ON oauth_clients
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- === oauth_authorization_codes ===
CREATE TABLE oauth_authorization_codes (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id               UUID NOT NULL,
    code_hash               VARCHAR(64) NOT NULL UNIQUE,
    client_id               UUID NOT NULL,
    user_id                 UUID NOT NULL,
    redirect_uri            TEXT NOT NULL,
    scope                   TEXT[] NOT NULL DEFAULT '{}',
    code_challenge          VARCHAR(256),
    code_challenge_method   VARCHAR(10) DEFAULT 'S256',
    nonce                   VARCHAR(128),
    expires_at              TIMESTAMPTZ NOT NULL,
    used                    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_codes_hash ON oauth_authorization_codes (code_hash) WHERE used = false;
CREATE INDEX idx_oauth_codes_expires ON oauth_authorization_codes (expires_at);

-- RLS
ALTER TABLE oauth_authorization_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_authorization_codes FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_codes_tenant_isolation ON oauth_authorization_codes
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- === oidc_id_tokens (audit only — tokens are stateless JWTs) ===
CREATE TABLE oidc_id_tokens (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    jti         VARCHAR(128) NOT NULL UNIQUE,
    user_id     UUID NOT NULL,
    client_id   UUID NOT NULL,
    tenant_id   UUID NOT NULL,
    scope       TEXT[] NOT NULL DEFAULT '{}',
    claims      JSONB NOT NULL DEFAULT '{}',
    expires_at  TIMESTAMPTZ NOT NULL,
    issued_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oidc_tokens_user ON oidc_id_tokens (user_id);
CREATE INDEX idx_oidc_tokens_client ON oidc_id_tokens (client_id);

-- RLS
ALTER TABLE oidc_id_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE oidc_id_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY oidc_tokens_tenant_isolation ON oidc_id_tokens
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- +migrate Down

DROP TABLE IF EXISTS oidc_id_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_clients;
