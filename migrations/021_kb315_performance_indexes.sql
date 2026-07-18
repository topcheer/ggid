-- KB-315: Missing indexes for hot-path queries
-- Run these migrations to improve dashboard + NHI + session query performance.

-- Dashboard stats: users count by tenant + deleted_at
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_tenant_deleted 
  ON users (tenant_id) WHERE deleted_at IS NULL;

-- Dashboard stats: sessions in last 24h (time-based scan)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_created_recent 
  ON sessions (created_at DESC);

-- Dashboard stats: auth events by time (login success/failure counts)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auth_events_created_type 
  ON auth_events (created_at DESC, event_type);

-- NHI baselines lookup by NHI ID (risk evaluation hot path)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_nhi_baselines_nhiid 
  ON nhi_behavior_baselines (nhi_id, endpoint);

-- NHI risk scores lookup (frequent reads during evaluation)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_nhi_risk_scores_nhiid 
  ON nhi_risk_scores (nhi_id, evaluated_at DESC);

-- CCM results: latest per control (DISTINCT ON optimization)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ccm_results_control_time 
  ON ccm_results (tenant_id, control_id, checked_at DESC);

-- Conditional access policies: enabled + priority ordering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cap_enabled_priority 
  ON conditional_access_policies (tenant_id, priority DESC) 
  WHERE enabled = true;

-- Privileged operations: recent per tenant
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_privil_op_tenant_time 
  ON privileged_operations (tenant_id, timestamp DESC);

-- CAE evaluations: recent per tenant
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cae_eval_tenant_time 
  ON cae_evaluations (tenant_id, evaluated_at DESC);

-- Joiner dashboard: user lookups by status + tenant
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_status_tenant 
  ON users (status, tenant_id) WHERE deleted_at IS NULL;
