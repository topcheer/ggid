-- 020_device_posture.sql
-- Device posture signals + compliance evaluation.

CREATE TABLE IF NOT EXISTS device_posture (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    device_id       TEXT NOT NULL,
    user_id         UUID,
    trust_level     TEXT NOT NULL DEFAULT 'unknown',
    compliance_score INT NOT NULL DEFAULT 0,
    compliant       BOOLEAN NOT NULL DEFAULT FALSE,
    checks          JSONB NOT NULL DEFAULT '{}',
    last_check_at   TIMESTAMPTZ,
    last_seen       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, device_id)
);
CREATE INDEX IF NOT EXISTS idx_device_posture_user ON device_posture(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_device_posture_device ON device_posture(tenant_id, device_id);

COMMENT ON TABLE device_posture IS 'ZTNA device posture signals and compliance evaluation results';
