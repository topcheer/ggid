-- 049_geofence_tenant_rules.sql
-- Per-tenant geofencing rules with DB persistence.
-- Each rule is bound to a tenant_id for data residency enforcement.

CREATE TABLE IF NOT EXISTS geofence_rules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         TEXT NOT NULL,
    name              TEXT NOT NULL DEFAULT 'unnamed-rule',
    allowed_countries TEXT[] NOT NULL DEFAULT '{}',
    denied_regions    TEXT[] NOT NULL DEFAULT '{}',
    action            TEXT NOT NULL DEFAULT 'deny' CHECK (action IN ('allow', 'deny', 'mfa')),
    priority          INT NOT NULL DEFAULT 0,
    enabled           BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for tenant-scoped queries.
CREATE INDEX IF NOT EXISTS idx_geofence_rules_tenant ON geofence_rules (tenant_id, enabled);
