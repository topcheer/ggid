-- 050_missing_feature_tables.sql
-- Creates tables referenced by services but missing from DB schema.
-- These cause silent failures (empty results) when services query non-existent tables.
-- Idempotent: uses CREATE TABLE IF NOT EXISTS.

-- === OAuth consent ===
CREATE TABLE IF NOT EXISTS consent_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    client_id TEXT NOT NULL DEFAULT '',
    purpose TEXT NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    policy_version TEXT NOT NULL DEFAULT '1.0',
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    withdrawn_at TIMESTAMPTZ,
    withdrawn_reason TEXT
);
CREATE INDEX IF NOT EXISTS idx_consent_records_tenant_user_client
    ON consent_records(tenant_id, user_id, client_id, status);

-- === Webhook management ===
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    events TEXT[] DEFAULT '{}',
    secret TEXT,
    max_retries INT DEFAULT 3,
    batch_size INT DEFAULT 10,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    url TEXT NOT NULL,
    events TEXT[] DEFAULT '{}',
    secret TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_alert_webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    url TEXT,
    events TEXT[],
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- === Notifications ===
CREATE TABLE IF NOT EXISTS notification_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule TEXT,
    severity TEXT,
    channel TEXT,
    subject TEXT,
    status TEXT,
    sent_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notification_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    severity TEXT,
    channels TEXT[],
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS email_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    to_addr TEXT,
    subject TEXT,
    template TEXT,
    status TEXT,
    error TEXT,
    sent_at TIMESTAMPTZ DEFAULT now()
);

-- === SCIM ===
CREATE TABLE IF NOT EXISTS scim_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target TEXT,
    operation TEXT,
    ggid_user_id UUID,
    scim_user_id TEXT,
    status TEXT,
    error TEXT,
    executed_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scim_sync_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    target TEXT,
    cron TEXT,
    last_sync TIMESTAMPTZ,
    enabled BOOLEAN DEFAULT TRUE
);

-- === Security & compliance ===
CREATE TABLE IF NOT EXISTS password_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_password_history_user ON password_history(user_id);

CREATE TABLE IF NOT EXISTS key_rotation_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_type TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    rotated_at TIMESTAMPTZ,
    grace_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS passkey_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    user_id UUID,
    credential_id TEXT NOT NULL,
    public_key BYTEA,
    device_type TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS error_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level TEXT,
    message TEXT,
    resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS health_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service TEXT,
    status TEXT DEFAULT 'healthy',
    checked_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS compliance_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    framework TEXT,
    control TEXT,
    frequency TEXT,
    last_run TIMESTAMPTZ,
    next_run TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rls_policies (
    id TEXT PRIMARY KEY,
    table_name TEXT,
    policy_name TEXT,
    using_expr TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS security_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    type TEXT,
    config JSONB DEFAULT '{}',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- === Conditional access ===
CREATE TABLE IF NOT EXISTS consent_cascade_log (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    tenant_id TEXT,
    trigger_type TEXT,
    scope TEXT,
    actions JSONB DEFAULT '{}',
    affected_tokens TEXT[],
    affected_sessions TEXT[],
    notified_apps TEXT[],
    executed_at TIMESTAMPTZ DEFAULT now()
);

-- === Identity ===
CREATE TABLE IF NOT EXISTS joiner_onboarding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    user_id UUID,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS non_human_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    type TEXT,
    tenant_id UUID,
    credentials JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS privileged_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    tenant_id UUID,
    scope TEXT,
    mfa_verified BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_behavioral_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    login_patterns JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    type TEXT,
    secret TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- === Policy & governance ===
CREATE TABLE IF NOT EXISTS access_policies (
    id TEXT PRIMARY KEY,
    data JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS custom_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS pii_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name TEXT,
    column_name TEXT,
    pii_type TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name TEXT,
    retention_days INT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS role_tree (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID,
    parent_role_id UUID,
    depth INT DEFAULT 0
);

-- === SOAR ===
CREATE TABLE IF NOT EXISTS soar_playbooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    steps JSONB DEFAULT '[]',
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS soar_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    playbook_id UUID,
    status TEXT,
    result JSONB,
    executed_at TIMESTAMPTZ DEFAULT now()
);

-- === Tenant branding ===
CREATE TABLE IF NOT EXISTS tenant_branding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID UNIQUE,
    logo_url TEXT,
    primary_color TEXT,
    config JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- === Threat intel ===
CREATE TABLE IF NOT EXISTS threat_intel_indicators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT,
    value TEXT,
    severity TEXT,
    source TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- === Plugins ===
CREATE TABLE IF NOT EXISTS plugins (
    id TEXT PRIMARY KEY,
    name TEXT,
    version TEXT,
    enabled BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}',
    installed_at TIMESTAMPTZ DEFAULT now()
);

-- === GraphQL ===
CREATE TABLE IF NOT EXISTS graphql_query_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation TEXT,
    query_hash TEXT,
    complexity INT,
    depth INT,
    duration_ms INT,
    error TEXT,
    user_id TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- === Rate limiting ===
CREATE TABLE IF NOT EXISTS rate_limit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT,
    count INT,
    window_start TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- === Dependency scanning ===
CREATE TABLE IF NOT EXISTS dependency_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    component TEXT,
    version TEXT,
    vulnerabilities JSONB DEFAULT '[]',
    scanned_at TIMESTAMPTZ DEFAULT now()
);
