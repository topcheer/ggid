-- 032_unified_pdp.sql
-- Unified Policy Decision Point: decision audit log.

CREATE TABLE IF NOT EXISTS policy_decisions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID,
    subject         TEXT NOT NULL,
    resource        TEXT NOT NULL,
    action          TEXT NOT NULL,
    decision        TEXT NOT NULL,
    deny_reason     TEXT NOT NULL DEFAULT '',
    risk_score      INT NOT NULL DEFAULT 0,
    risk_overlay    TEXT NOT NULL DEFAULT '',
    context         JSONB NOT NULL DEFAULT '{}',
    evaluated_by    TEXT[] NOT NULL DEFAULT '{}',
    cache_hit       BOOLEAN NOT NULL DEFAULT FALSE,
    latency_ms      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_decisions_tenant ON policy_decisions(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_decisions_subject ON policy_decisions(subject, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_decisions_resource ON policy_decisions(resource, created_at DESC);

COMMENT ON TABLE policy_decisions IS 'Unified PDP authorization decision audit log';
