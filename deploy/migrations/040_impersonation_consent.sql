-- 040: Impersonation Consent System
-- Enables platform admins to access tenant data with explicit tenant admin consent.

CREATE TABLE IF NOT EXISTS tenant_access_consents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    granted_to  VARCHAR(255) NOT NULL,   -- 'platform_admin' or specific admin user_id
    granted_by  UUID NOT NULL,            -- tenant admin user_id
    scope       VARCHAR(20) NOT NULL DEFAULT 'support',  -- support | audit | full
    expires_at  TIMESTAMPTZ,              -- optional expiry, NULL = until revoked
    revoked_at  TIMESTAMPTZ,              -- when revoked
    reason      TEXT,                     -- why consent was granted
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_consent_scope CHECK (scope IN ('support', 'audit', 'full'))
);

CREATE INDEX IF NOT EXISTS idx_consents_tenant ON tenant_access_consents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_consents_active ON tenant_access_consents(tenant_id, granted_to)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_consents_granted_by ON tenant_access_consents(granted_by);

CREATE TABLE IF NOT EXISTS impersonation_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    impersonator_id  UUID NOT NULL,       -- platform admin user_id
    target_user_id   UUID,                -- optional: specific user being impersonated
    consent_id       UUID REFERENCES tenant_access_consents(id) ON DELETE SET NULL,
    reason           TEXT NOT NULL,        -- mandatory reason
    scope            VARCHAR(20) NOT NULL DEFAULT 'support',
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMPTZ,          -- NULL = active session
    ip_address       INET,
    user_agent       TEXT
);

CREATE INDEX IF NOT EXISTS idx_impersonation_active ON impersonation_sessions(impersonator_id, tenant_id)
    WHERE ended_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_impersonation_tenant ON impersonation_sessions(tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_impersonation_consent ON impersonation_sessions(consent_id);
