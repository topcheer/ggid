-- 029_consent_management.sql
-- GDPR Art. 7 & Art. 17 compliant consent management.

CREATE TABLE IF NOT EXISTS consent_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         TEXT NOT NULL,
    client_id       TEXT NOT NULL DEFAULT '',
    purpose         TEXT NOT NULL,
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'active', -- active, withdrawn, expired
    policy_version  TEXT NOT NULL DEFAULT '1.0',
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ,
    withdrawn_at    TIMESTAMPTZ,
    withdrawn_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_consent_tenant_user ON consent_records(tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_consent_status ON consent_records(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_consent_client ON consent_records(client_id, user_id);

CREATE TABLE IF NOT EXISTS consent_purposes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    default_scopes  TEXT[] NOT NULL DEFAULT '{}',
    required        BOOLEAN NOT NULL DEFAULT FALSE,
    policy_version  TEXT NOT NULL DEFAULT '1.0',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_consent_purpose_name ON consent_purposes(tenant_id, name);

COMMENT ON TABLE consent_records IS 'GDPR Art. 7 consent records with withdrawal tracking';
COMMENT ON TABLE consent_purposes IS 'Catalog of consent purposes per tenant';
