-- SCIM Groups tables for database-backed group persistence
-- SCIM-13: Replace mock groups with real DB-backed storage

CREATE TABLE IF NOT EXISTS scim_groups (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    external_id  VARCHAR(255),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,

    CONSTRAINT uq_scim_groups_tenant_display UNIQUE (tenant_id, display_name),
    CONSTRAINT uq_scim_groups_tenant_external UNIQUE (tenant_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_scim_groups_tenant ON scim_groups(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scim_groups_display ON scim_groups(tenant_id, display_name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS scim_group_members (
    id         UUID PRIMARY KEY,
    tenant_id  UUID NOT NULL,
    group_id   UUID NOT NULL REFERENCES scim_groups(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL,
    user_type  VARCHAR(50) NOT NULL DEFAULT 'User',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT uq_scim_group_members UNIQUE (tenant_id, group_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_scim_group_members_group ON scim_group_members(group_id, tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scim_group_members_user ON scim_group_members(user_id, tenant_id) WHERE deleted_at IS NULL;

COMMENT ON TABLE scim_groups IS 'SCIM 2.0 Group resources for enterprise directory sync';
COMMENT ON TABLE scim_group_members IS 'User-group membership mappings for SCIM Groups';
