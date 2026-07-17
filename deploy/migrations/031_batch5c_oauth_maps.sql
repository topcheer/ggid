-- 031_batch5c_oauth_maps.sql
-- Batch 5c: oauth service in-memory map tables (JSONB generic stores).

CREATE TABLE IF NOT EXISTS oauth_client_lifecycles (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_agent_reviews (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_client_deprecations (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_consent_overrides (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_usage_policies (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_token_families (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_client_versions (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_par_store (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_consent_receipts (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_client_events (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_revoke_cascades (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_client_certs (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_consent_screens (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_device_codes (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
CREATE TABLE IF NOT EXISTS oauth_agent_registrations (id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now());
