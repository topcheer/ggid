-- 024_memory_map_batch2.sql
-- Batch 2: remaining in-memory map migrations for audit + auth.

-- Audit: integrity, webhook deliveries, dsr requests, collect schedules, dedup
CREATE TABLE IF NOT EXISTS evidence_integrity (
    id TEXT PRIMARY KEY, evidence_id TEXT NOT NULL,
    hash TEXT, algorithm TEXT DEFAULT 'sha256',
    verified BOOLEAN DEFAULT FALSE, verified_by TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id TEXT PRIMARY KEY, tenant_id UUID,
    webhook_id TEXT, event_type TEXT, payload JSONB DEFAULT '{}',
    status TEXT DEFAULT 'pending', response_code INT,
    attempts INT DEFAULT 0, error TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dsr_requests (
    id TEXT PRIMARY KEY, tenant_id UUID,
    user_id TEXT, request_type TEXT DEFAULT 'access',
    status TEXT DEFAULT 'pending', details JSONB DEFAULT '{}',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS collect_schedules (
    id TEXT PRIMARY KEY, tenant_id UUID,
    name TEXT, source TEXT, interval_minutes INT DEFAULT 60,
    enabled BOOLEAN DEFAULT TRUE, last_run TIMESTAMPTZ,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS event_dedup (
    id TEXT PRIMARY KEY, event_hash TEXT UNIQUE,
    first_seen TIMESTAMPTZ DEFAULT now(),
    last_seen TIMESTAMPTZ DEFAULT now(),
    seen_count INT DEFAULT 1
);

-- Auth: device bindings, device trusts, geofence, travel events, login flows
CREATE TABLE IF NOT EXISTS auth_device_bindings (
    id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
    device_id TEXT, device_name TEXT, platform TEXT,
    trusted BOOLEAN DEFAULT FALSE, last_used TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_device_trusts (
    id TEXT PRIMARY KEY, device_id TEXT UNIQUE,
    user_id TEXT, trust_score INT DEFAULT 0,
    managed BOOLEAN DEFAULT FALSE, encrypted BOOLEAN DEFAULT FALSE,
    compliant_os BOOLEAN DEFAULT FALSE, jailbreak BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_geofence_rules (
    id TEXT PRIMARY KEY, tenant_id UUID,
    name TEXT, lat FLOAT, lng FLOAT, radius_meters INT DEFAULT 500,
    action TEXT DEFAULT 'warn', enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_travel_events (
    id TEXT PRIMARY KEY, user_id TEXT NOT NULL,
    login_time TIMESTAMPTZ, ip TEXT, country TEXT, city TEXT,
    latitude FLOAT, longitude FLOAT,
    flagged BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_login_flows (
    id TEXT PRIMARY KEY, tenant_id UUID,
    user_id TEXT, flow_type TEXT,
    step TEXT, status TEXT DEFAULT 'in_progress',
    ip TEXT, user_agent TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_devbind_user ON auth_device_bindings(user_id);
CREATE INDEX IF NOT EXISTS idx_devtrust_device ON auth_device_trusts(device_id);
CREATE INDEX IF NOT EXISTS idx_travel_user ON auth_travel_events(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_loginflow_tenant ON auth_login_flows(tenant_id, created_at DESC);
