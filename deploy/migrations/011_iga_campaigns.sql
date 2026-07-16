-- Migration 11: IGA Access Review Campaigns
-- Persists access review campaigns for SOX/等保 compliance.

CREATE TABLE IF NOT EXISTS iga_campaigns (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    scope        TEXT NOT NULL DEFAULT '',
    scope_id     TEXT NOT NULL DEFAULT '',
    reviewer_id  UUID,
    deadline     TIMESTAMPTZ,
    status       TEXT NOT NULL DEFAULT 'active',
    decision     TEXT,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    submitted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_iga_tenant_status ON iga_campaigns (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_iga_reviewer ON iga_campaigns (reviewer_id, status);
