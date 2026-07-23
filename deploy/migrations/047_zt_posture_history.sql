-- R2-04: Zero-trust posture scoring history (NIST 800-207)
CREATE TABLE IF NOT EXISTS zt_posture_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID,
    overall_score INT NOT NULL,
    identity_score INT NOT NULL,
    device_score INT NOT NULL,
    network_score INT NOT NULL,
    data_score INT NOT NULL,
    workload_score INT NOT NULL,
    grade TEXT NOT NULL,
    findings JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',
    evaluated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_zt_posture_tenant_time ON zt_posture_history (tenant_id, evaluated_at DESC);
COMMENT ON TABLE zt_posture_history IS 'Zero-trust posture score history for trend tracking';
