-- 026_policy_memory_maps.sql
-- Batch 3: Policy + identity in-memory map migrations.

-- Identity: lifecycle_rules, access_review_campaigns
CREATE TABLE IF NOT EXISTS lifecycle_rules_store (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_lifecycle_rules_tenant ON lifecycle_rules_store(tenant_id);

CREATE TABLE IF NOT EXISTS review_campaigns_store (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_review_campaigns_tenant ON review_campaigns_store(tenant_id);

-- Policy: conditional_access, access_requests, optimization_findings, auto_assignments
CREATE TABLE IF NOT EXISTS conditional_access_store (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_cond_access_tenant ON conditional_access_store(tenant_id);

CREATE TABLE IF NOT EXISTS access_requests_store (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_access_req_tenant ON access_requests_store(tenant_id);

CREATE TABLE IF NOT EXISTS access_optimization_store (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auto_assignments_store (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
