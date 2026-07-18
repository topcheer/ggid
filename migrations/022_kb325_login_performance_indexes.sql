-- KB-325: Additional performance indexes for login + list hot paths
-- Complements 021_kb315_performance_indexes.sql

-- Login hot path: user lookup by username + tenant (auth_credentials join)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auth_credentials_tenant_identifier
  ON auth_credentials (tenant_id, identifier);

-- Login audit: recent auth events per user for brute-force analysis
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auth_events_user_tenant_time
  ON auth_events (user_id, tenant_id, created_at DESC);

-- OAuth tokens: token revocation lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_oauth_tokens_tenant_client_expires
  ON oauth_tokens (tenant_id, client_id, expires_at DESC);

-- OAuth clients: list by tenant with enabled filter
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_oauth_clients_tenant_enabled
  ON oauth_clients (tenant_id, enabled);

-- Audit events: list by tenant + time (dashboard pagination)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_events_tenant_time
  ON audit_events (tenant_id, created_at DESC);

-- Policy evaluations: recent per tenant for dashboard
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_policy_evaluations_tenant_time
  ON policy_evaluations (tenant_id, evaluated_at DESC);
