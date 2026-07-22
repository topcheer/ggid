-- Migration 046: tenant_idp_configs
-- Table for per-tenant IdP (Identity Provider) configurations.
-- Used by social login (R1-02) to store Google/GitHub/etc OAuth client configs.
-- Schema matches social_handler.go query: SELECT config_json FROM tenant_idp_configs
--   WHERE tenant_id = $1 AND idp_type = 'oidc' AND name = $2 AND enabled = true

CREATE TABLE IF NOT EXISTS tenant_idp_configs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    idp_type     VARCHAR(20) NOT NULL DEFAULT 'oidc',  -- oidc, saml, ldap
    name         VARCHAR(100) NOT NULL DEFAULT '',     -- provider name: google, github, etc
    provider     VARCHAR(50) NOT NULL DEFAULT '',      -- canonical provider key
    client_id    VARCHAR(255) NOT NULL DEFAULT '',
    client_secret TEXT NOT NULL DEFAULT '',             -- encrypted at rest (future: KMS)
    config_json  JSONB DEFAULT '{}',                    -- full config (client_id, client_secret, scopes, redirect_uris)
    scopes       TEXT[] DEFAULT '{}',                   -- e.g. ['openid','email','profile']
    enabled      BOOLEAN DEFAULT true,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_idp_configs_tenant ON tenant_idp_configs(tenant_id);
CREATE INDEX idx_idp_configs_lookup ON tenant_idp_configs(tenant_id, idp_type, name, enabled);
