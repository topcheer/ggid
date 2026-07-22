-- GGID Combined Migration (auto-generated, idempotent)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS ltree;

-- Create enums
DO $$ BEGIN CREATE TYPE tenant_plan AS ENUM ('free', 'pro', 'enterprise'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'deleted'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE membership_status AS ENUM ('active', 'invited', 'removed'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(50) NOT NULL UNIQUE,
    plan        tenant_plan NOT NULL DEFAULT 'free',
    status      tenant_status NOT NULL DEFAULT 'active',
    settings    JSONB NOT NULL DEFAULT '{}',
    max_users   INT NOT NULL DEFAULT 50,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- Organizations table (LTREE hierarchical)
CREATE TABLE IF NOT EXISTS organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    parent_id   UUID REFERENCES organizations(id) ON DELETE SET NULL,
    name        VARCHAR(200) NOT NULL,
    path        LTREE NOT NULL,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orgs_tenant ON organizations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_orgs_parent ON organizations(parent_id);
CREATE INDEX IF NOT EXISTS idx_orgs_path ON organizations USING GIST(path);

-- Departments table (LTREE hierarchical within org)
CREATE TABLE IF NOT EXISTS departments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES departments(id) ON DELETE SET NULL,
    name        VARCHAR(200) NOT NULL,
    path        LTREE NOT NULL,
    manager_id  UUID,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_depts_org ON departments(org_id);
CREATE INDEX IF NOT EXISTS idx_depts_parent ON departments(parent_id);
CREATE INDEX IF NOT EXISTS idx_depts_path ON departments USING GIST(path);
CREATE INDEX IF NOT EXISTS idx_depts_manager ON departments(manager_id);

-- Teams table
CREATE TABLE IF NOT EXISTS teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    created_by  UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name)
);

CREATE INDEX IF NOT EXISTS idx_teams_org ON teams(org_id);

-- Memberships table
CREATE TABLE IF NOT EXISTS memberships (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    tenant_id   UUID NOT NULL,
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    dept_id     UUID REFERENCES departments(id) ON DELETE SET NULL,
    team_id     UUID REFERENCES teams(id) ON DELETE SET NULL,
    title       VARCHAR(100) DEFAULT '',
    status      membership_status NOT NULL DEFAULT 'invited',
    joined_at   TIMESTAMPTZ,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memberships_user ON memberships(user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_tenant ON memberships(tenant_id);
CREATE INDEX IF NOT EXISTS idx_memberships_org ON memberships(org_id);
CREATE INDEX IF NOT EXISTS idx_memberships_dept ON memberships(dept_id);
CREATE INDEX IF NOT EXISTS idx_memberships_team ON memberships(team_id);
CREATE INDEX IF NOT EXISTS idx_memberships_status ON memberships(status);

-- Enable RLS on tenant-scoped tables
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_orgs ON organizations
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_isolation_memberships ON memberships
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Enable UUID generation

-- Create enums
DO $$ BEGIN CREATE TYPE scope_type AS ENUM ('global', 'organization', 'department', 'team', 'resource'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE policy_effect AS ENUM ('allow', 'deny'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE principal_type AS ENUM ('user', 'role', 'group'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Roles table
CREATE TABLE IF NOT EXISTS roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    key             VARCHAR(64) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    description     TEXT DEFAULT '',
    system_role     BOOLEAN NOT NULL DEFAULT FALSE,
    parent_role_id  UUID REFERENCES roles(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_parent ON roles(parent_role_id);

-- Permissions table
CREATE TABLE IF NOT EXISTS permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    key             VARCHAR(128) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    resource_type   VARCHAR(50) NOT NULL,
    action          VARCHAR(50) NOT NULL,
    description     TEXT DEFAULT '',
    system_perm     BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS idx_permissions_tenant ON permissions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_permissions_resource_action ON permissions(resource_type, action);

-- Role-Permissions junction table
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    conditions      JSONB DEFAULT '{}',
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id);

-- User-Roles table
CREATE TABLE IF NOT EXISTS user_roles (
    user_id         UUID NOT NULL,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_type      scope_type NOT NULL,
    scope_id        UUID NOT NULL,
    granted_by      UUID NOT NULL,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id, scope_type, scope_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_scope ON user_roles(scope_type, scope_id);

-- Policies table (ABAC — AWS IAM style)
CREATE TABLE IF NOT EXISTS policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(100) NOT NULL,
    description     TEXT DEFAULT '',
    effect          policy_effect NOT NULL,
    actions         TEXT[] NOT NULL DEFAULT '{}',
    resources       TEXT[] NOT NULL DEFAULT '{}',
    conditions      JSONB DEFAULT '{}',
    priority        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_policies_tenant ON policies(tenant_id);

-- Policy attachments table
CREATE TABLE IF NOT EXISTS policy_attachments (
    policy_id       UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    principal_type  principal_type NOT NULL,
    principal_id    UUID NOT NULL,
    PRIMARY KEY (policy_id, principal_type, principal_id)
);

CREATE INDEX IF NOT EXISTS idx_policy_attachments_principal ON policy_attachments(principal_type, principal_id);

-- Enable Row Level Security on all tenant-scoped tables
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE policies ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_roles ON roles
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_isolation_permissions ON permissions
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenant_isolation_policies ON policies
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Identity Service: Initial Schema
-- Creates users, user_emails, user_external_identities, and email_verification_tokens tables
-- with PostgreSQL Row Level Security for multi-tenant isolation.

-- +migrate Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- === users ===
CREATE TABLE IF NOT EXISTS users (
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

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at DESC);

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
CREATE TABLE IF NOT EXISTS user_emails (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_emails_tenant_email_uk UNIQUE (email),
    CONSTRAINT user_emails_user_email_uk UNIQUE (user_id, email)
);

CREATE INDEX IF NOT EXISTS idx_user_emails_user ON user_emails (user_id);
CREATE INDEX IF NOT EXISTS idx_user_emails_primary ON user_emails (user_id, is_primary) WHERE is_primary = true;

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
CREATE TABLE IF NOT EXISTS user_external_identities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider        VARCHAR(50) NOT NULL,
    external_id     VARCHAR(255) NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    linked_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ext_identity_provider_uk UNIQUE (provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_ext_identity_user ON user_external_identities (user_id);
CREATE INDEX IF NOT EXISTS idx_ext_identity_provider ON user_external_identities (provider, external_id);

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
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id        UUID NOT NULL REFERENCES user_emails(id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_ver_token_hash ON email_verification_tokens (token_hash)
    WHERE consumed_at IS NULL;

-- No RLS on tokens — they are consumed by hash, not by tenant lookup.


-- credentials table: stores authentication credentials (passwords, etc.)
CREATE TABLE IF NOT EXISTS credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'password',
    identifier      VARCHAR(255) NOT NULL,        -- username or credential_id
    secret          TEXT NOT NULL,                 -- Argon2id hash for passwords
    metadata        JSONB DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ
);

-- Password history for reuse prevention
CREATE TABLE IF NOT EXISTS credential_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    secret          TEXT NOT NULL,                 -- previous password hash
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_credentials_tenant_user ON credentials(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_credentials_identifier   ON credentials(tenant_id, identifier);
CREATE UNIQUE INDEX IF NOT EXISTS uq_credentials_tenant_identifier_type ON credentials(tenant_id, identifier, type);
CREATE INDEX IF NOT EXISTS idx_cred_history_user       ON credential_history(tenant_id, user_id, created_at DESC);

COMMENT ON TABLE credentials IS 'Authentication credentials (password, passkey, etc.)';
COMMENT ON COLUMN credentials.secret IS 'Argon2id password hash or encrypted secret';

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
CREATE INDEX IF NOT EXISTS idx_sessions_user       ON sessions(tenant_id, user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_expires    ON sessions(expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE sessions IS 'Active and historical user sessions';
COMMENT ON COLUMN sessions.token_hash IS 'SHA-256 hash of the opaque session token';

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
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash   ON refresh_tokens(token_hash) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user   ON refresh_tokens(tenant_id, user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session ON refresh_tokens(session_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE refresh_tokens IS 'Opaque refresh tokens stored in DB (also mirrored in Redis for fast lookup)';
COMMENT ON COLUMN refresh_tokens.rotated_from IS 'ID of the previous token this was rotated from (rotation chain)';

-- Enable pgcrypto for gen_random_uuid()

-- Create enums
DO $$ BEGIN CREATE TYPE actor_type AS ENUM ('user', 'api_key', 'system', 'anonymous'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE event_result AS ENUM ('success', 'failure', 'denied'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Audit events table with monthly range partitioning
CREATE TABLE IF NOT EXISTS audit_events (
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    actor_type    actor_type NOT NULL DEFAULT 'user',
    actor_id      UUID,
    actor_name    VARCHAR(200) DEFAULT '',
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) DEFAULT '',
    resource_id   UUID,
    resource_name VARCHAR(200) DEFAULT '',
    result        event_result NOT NULL DEFAULT 'success',
    ip_address    INET,
    user_agent    TEXT DEFAULT '',
    request_id    VARCHAR(64) DEFAULT '',
    metadata      JSONB DEFAULT '{}',
    prev_hash     TEXT,
    event_hash    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partitions for current and next month
-- These will be replaced by a partitioning function in production,
-- but for the skeleton we create a few static partitions.

-- Default partition (catch-all for events outside defined partitions)
CREATE TABLE IF NOT EXISTS audit_events_default PARTITION OF audit_events DEFAULT;

-- Indexes on the parent table (inherited by all partitions)
CREATE INDEX IF NOT EXISTS idx_audit_tenant_created ON audit_events (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_events (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_events (action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_events (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_result ON audit_events (result);
CREATE INDEX IF NOT EXISTS idx_audit_request_id ON audit_events (request_id);

-- Create monthly partitions for 2025
-- In production, use pg_partman or a cron job to auto-create partitions.

-- January 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_01 PARTITION OF audit_events
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- February 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_02 PARTITION OF audit_events
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

-- March 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_03 PARTITION OF audit_events
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

-- April 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_04 PARTITION OF audit_events
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

-- May 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_05 PARTITION OF audit_events
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');

-- June 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_06 PARTITION OF audit_events
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

-- July 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_07 PARTITION OF audit_events
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');

-- August 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_08 PARTITION OF audit_events
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');

-- September 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_09 PARTITION OF audit_events
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');

-- October 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_10 PARTITION OF audit_events
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');

-- November 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_11 PARTITION OF audit_events
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

-- December 2025
CREATE TABLE IF NOT EXISTS audit_events_2025_12 PARTITION OF audit_events
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Seed system roles and permissions for a default tenant.
-- This migration can be run per-tenant during provisioning.

-- System roles
INSERT INTO roles (tenant_id, key, name, description, system_role) VALUES
    (SELECT id FROM tenants WHERE slug = 'default'), 'admin', 'Administrator', 'Full system access', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'editor', 'Editor', 'Read and write access, no admin', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'viewer', 'Viewer', 'Read-only access', TRUE)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- System permissions
INSERT INTO permissions (tenant_id, key, name, resource_type, action, description, system_perm) VALUES
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:users:read',    'Read Users',    'users',    'read',   'View user profiles', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:users:write',   'Write Users',   'users',    'write',  'Create/update users', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:users:delete',  'Delete Users',  'users',    'delete', 'Delete users', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:roles:read',    'Read Roles',    'roles',    'read',   'View roles', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:roles:write',   'Write Roles',   'roles',    'write',  'Create/update roles', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:orgs:read',     'Read Orgs',     'organizations', 'read', 'View organizations', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:orgs:write',    'Write Orgs',    'organizations', 'write', 'Manage organizations', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:audit:read',    'Read Audit',    'audit',    'read',   'View audit logs', TRUE),
    (SELECT id FROM tenants WHERE slug = 'default'), 'iam:policies:write', 'Write Policies', 'policies', 'write', 'Manage policies', TRUE)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND r.key = 'admin'
      AND p.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
ON CONFLICT DO NOTHING;

-- Assign read permissions to viewer role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND r.key = 'viewer'
      AND p.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND p.action = 'read'
ON CONFLICT DO NOTHING;

-- Assign read+write (non-admin) permissions to editor role
INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id FROM roles r, permissions p
    WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND r.key = 'editor'
      AND p.tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
      AND p.action IN ('read', 'write')
ON CONFLICT DO NOTHING;

-- OAuth/OIDC Service: Initial Schema
-- Creates oauth_clients, oauth_authorization_codes, and oidc_id_tokens tables.

-- +migrate Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- === oauth_clients ===
CREATE TABLE IF NOT EXISTS oauth_clients (
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

CREATE INDEX IF NOT EXISTS idx_oauth_clients_tenant ON oauth_clients (tenant_id) WHERE enabled = true;

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
CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
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

CREATE INDEX IF NOT EXISTS idx_oauth_codes_hash ON oauth_authorization_codes (code_hash) WHERE used = false;
CREATE INDEX IF NOT EXISTS idx_oauth_codes_expires ON oauth_authorization_codes (expires_at);

-- RLS
ALTER TABLE oauth_authorization_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_authorization_codes FORCE ROW LEVEL SECURITY;

CREATE POLICY oauth_codes_tenant_isolation ON oauth_authorization_codes
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- === oidc_id_tokens (audit only — tokens are stateless JWTs) ===
CREATE TABLE IF NOT EXISTS oidc_id_tokens (
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

CREATE INDEX IF NOT EXISTS idx_oidc_tokens_user ON oidc_id_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_oidc_tokens_client ON oidc_id_tokens (client_id);

-- RLS
ALTER TABLE oidc_id_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE oidc_id_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY oidc_tokens_tenant_isolation ON oidc_id_tokens
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );


-- MFA Devices table

-- +migrate Up

CREATE TABLE IF NOT EXISTS mfa_devices (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    name        VARCHAR(100) NOT NULL DEFAULT 'default',
    secret      TEXT NOT NULL,
    algorithm   VARCHAR(10) NOT NULL DEFAULT 'SHA1',
    digits      INT NOT NULL DEFAULT 6,
    period      INT NOT NULL DEFAULT 30,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mfa_devices_user ON mfa_devices (tenant_id, user_id) WHERE enabled = true;

-- updated_at trigger
CREATE TRIGGER mfa_devices_updated_at
    BEFORE UPDATE ON mfa_devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- RLS
ALTER TABLE mfa_devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_devices FORCE ROW LEVEL SECURITY;

CREATE POLICY mfa_devices_tenant_isolation ON mfa_devices
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );


INSERT INTO tenants (id, name, slug, plan, status, max_users)
VALUES (gen_random_uuid(), 'Default', 'default', 'enterprise', 'active', 10000)
ON CONFLICT (slug) DO NOTHING;
