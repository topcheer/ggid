-- 033_unified_risk_engine.sql
-- Unified Risk Engine: policies + signals + scores.

CREATE TABLE IF NOT EXISTS risk_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    allow_threshold     INT NOT NULL DEFAULT 30,
    step_up_threshold   INT NOT NULL DEFAULT 60,
    strong_threshold    INT NOT NULL DEFAULT 85,
    weights         JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_risk_policy_tenant ON risk_policies(tenant_id);

CREATE TABLE IF NOT EXISTS risk_scores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID,
    user_id         TEXT NOT NULL,
    session_id      TEXT NOT NULL DEFAULT '',
    score           INT NOT NULL DEFAULT 0,
    level           TEXT NOT NULL DEFAULT 'low',
    decision        TEXT NOT NULL DEFAULT 'allow',
    signals         JSONB NOT NULL DEFAULT '[]',
    evaluated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_risk_scores_user ON risk_scores(user_id, evaluated_at DESC);
CREATE INDEX IF NOT EXISTS idx_risk_scores_tenant ON risk_scores(tenant_id, evaluated_at DESC);

COMMENT ON TABLE risk_policies IS 'Per-tenant risk thresholds + signal weights';
COMMENT ON TABLE risk_scores IS 'Risk evaluation results audit log';
