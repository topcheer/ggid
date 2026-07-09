-- Roles table
CREATE TABLE roles (
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

CREATE INDEX idx_roles_tenant ON roles(tenant_id);
CREATE INDEX idx_roles_parent ON roles(parent_role_id);

-- Permissions table
CREATE TABLE permissions (
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

CREATE INDEX idx_permissions_tenant ON permissions(tenant_id);
CREATE INDEX idx_permissions_resource_action ON permissions(resource_type, action);

-- Role-Permissions junction table
CREATE TABLE role_permissions (
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    conditions      JSONB DEFAULT '{}',
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role ON role_permissions(role_id);

-- User-Roles table
CREATE TABLE user_roles (
    user_id         UUID NOT NULL,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_type      scope_type NOT NULL,
    scope_id        UUID NOT NULL,
    granted_by      UUID NOT NULL,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id, scope_type, scope_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
CREATE INDEX idx_user_roles_scope ON user_roles(scope_type, scope_id);

-- Policies table (ABAC — AWS IAM style)
CREATE TABLE policies (
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

CREATE INDEX idx_policies_tenant ON policies(tenant_id);

-- Policy attachments table
CREATE TABLE policy_attachments (
    policy_id       UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    principal_type  principal_type NOT NULL,
    principal_id    UUID NOT NULL,
    PRIMARY KEY (policy_id, principal_type, principal_id)
);

CREATE INDEX idx_policy_attachments_principal ON policy_attachments(principal_type, principal_id);

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
