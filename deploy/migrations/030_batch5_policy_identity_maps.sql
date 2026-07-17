-- 030_batch5_policy_maps.sql
-- Batch 5: policy service in-memory map tables (JSONB generic stores).

CREATE TABLE IF NOT EXISTS policy_delegated_admins (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_abac_groups (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_permission_boundaries (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_certifications (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_campaign_results (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_bundles (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_approvals (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_snapshots (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_resource_tags (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS policy_inheritance (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());

-- Identity service batch 5
CREATE TABLE IF NOT EXISTS identity_delegations (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS identity_attestations (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS identity_user_preferences (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS identity_attribute_history (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS identity_did_registry (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS identity_templates (id TEXT PRIMARY KEY, tenant_id UUID, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
