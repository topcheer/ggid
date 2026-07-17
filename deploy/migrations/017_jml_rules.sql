-- 017_jml_rules.sql
-- JML (Joiner/Mover/Leaver) identity lifecycle orchestration engine.
-- HR events → rule matching → automated actions → audit trail.

CREATE TABLE IF NOT EXISTS lifecycle_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    name        TEXT NOT NULL,
    trigger     TEXT NOT NULL,               -- 'joiner', 'mover', 'leaver', 'rejoiner'
    conditions  JSONB NOT NULL DEFAULT '{}', -- {"department": "engineering", "source_idp": "workday"}
    actions     JSONB NOT NULL DEFAULT '[]', -- [{"type": "assign_role", "params": {"role_id": "..."}}]
    priority    INT NOT NULL DEFAULT 100,    -- lower = higher priority
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_jml_rules_trigger ON lifecycle_rules (tenant_id, trigger) WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS lifecycle_executions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    rule_id       UUID NOT NULL,
    user_id       UUID NOT NULL,
    trigger       TEXT NOT NULL,
    action_type   TEXT NOT NULL,
    action_params JSONB,
    result        TEXT NOT NULL,             -- 'success', 'failed', 'skipped'
    error         TEXT,
    executed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_jml_exec_user ON lifecycle_executions (tenant_id, user_id, executed_at DESC);

COMMENT ON TABLE lifecycle_rules IS 'JML identity orchestration rules';
COMMENT ON TABLE lifecycle_executions IS 'JML rule execution audit log';
