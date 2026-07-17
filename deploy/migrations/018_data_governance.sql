-- 018_data_governance.sql
-- Data security law compliance: classification labels + DSR workflow.

CREATE TABLE IF NOT EXISTS data_classifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    classification  TEXT NOT NULL DEFAULT 'general',
    category        TEXT,
    lawful_basis    TEXT,
    retention_days  INT,
    cross_border    TEXT NOT NULL DEFAULT 'allowed',
    mask_rule       TEXT DEFAULT 'none',
    created_by      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, resource_type, resource_id)
);
CREATE INDEX IF NOT EXISTS idx_data_class_lookup ON data_classifications(tenant_id, resource_type, resource_id);

CREATE TABLE IF NOT EXISTS dsr_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    user_id       UUID NOT NULL,
    request_type  TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    details       JSONB DEFAULT '{}',
    handled_by    UUID,
    handled_at    TIMESTAMPTZ,
    result_data   JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dsr_tenant ON dsr_requests(tenant_id, status, created_at DESC);

COMMENT ON TABLE data_classifications IS 'Data resource classification labels for PIPL/GDPR compliance';
COMMENT ON TABLE dsr_requests IS 'Data Subject Rights requests (access/erasure/portability/etc)';
