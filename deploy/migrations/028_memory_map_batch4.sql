-- 028_memory_map_batch4.sql
-- Batch 4: remaining 13 in-memory maps across 4 services.

-- audit: dashboard_widgets (reuse existing table from batch 1 if exists)
CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id TEXT PRIMARY KEY, tenant_id UUID,
    title TEXT, type TEXT,
    config JSONB DEFAULT '{}', position INT DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_retention_policies (
    id TEXT PRIMARY KEY, tenant_id UUID,
    name TEXT, description TEXT, category TEXT,
    retention_days INT DEFAULT 90,
    auto_delete BOOLEAN DEFAULT FALSE, enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY, tenant_id UUID,
    webhook_id TEXT, event_type TEXT, payload JSONB DEFAULT '{}',
    status TEXT DEFAULT 'pending', response_code INT,
    attempts INT DEFAULT 0, error TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- auth: device_bindings + device_trusts
CREATE TABLE IF NOT EXISTS auth_device_bindings (
    id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
    device_id TEXT, device_name TEXT, platform TEXT,
    trusted BOOLEAN DEFAULT FALSE, last_used TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_devbind_user ON auth_device_bindings(user_id);

CREATE TABLE IF NOT EXISTS auth_device_trusts (
    id TEXT PRIMARY KEY, device_id TEXT UNIQUE,
    user_id TEXT, trust_score INT DEFAULT 0,
    managed BOOLEAN DEFAULT FALSE, encrypted BOOLEAN DEFAULT FALSE,
    compliant_os BOOLEAN DEFAULT FALSE, jailbreak BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- oauth: 6 stores as generic JSONB tables
CREATE TABLE IF NOT EXISTS oauth_branding (
    id TEXT PRIMARY KEY, client_id TEXT NOT NULL,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_oauth_branding_client ON oauth_branding(client_id);

CREATE TABLE IF NOT EXISTS oauth_client_scopes (
    id TEXT PRIMARY KEY, client_id TEXT NOT NULL,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_dpop_bindings (
    id TEXT PRIMARY KEY,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_resource_allow (
    id TEXT PRIMARY KEY, client_id TEXT NOT NULL,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_custom_scopes (
    id TEXT PRIMARY KEY, scope_name TEXT NOT NULL,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_delegation_chains (
    id TEXT PRIMARY KEY,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);

-- policy: resource_acls + sod_rules_crud
CREATE TABLE IF NOT EXISTS policy_resource_acls (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS policy_sod_rule_pairs (
    id TEXT PRIMARY KEY, tenant_id UUID,
    data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
);
