-- MFA Devices table

-- +migrate Up

CREATE TABLE IF NOT EXISTS mfa_devices (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    name        VARCHAR(100) NOT NULL DEFAULT 'default',
    secret      TEXT NOT NULL,
    algorithm   VARCHAR(10) NOT NULL DEFAULT 'SHA1',
    digits      INT NOT NULL DEFAULT 6,
    period      INT NOT NULL DEFAULT 30,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mfa_devices_user ON mfa_devices (tenant_id, user_id) WHERE enabled = true;

-- updated_at trigger
CREATE TRIGGER mfa_devices_updated_at
    BEFORE UPDATE ON mfa_devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- RLS
ALTER TABLE mfa_devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_devices FORCE ROW LEVEL SECURITY;

CREATE POLICY mfa_devices_tenant_isolation ON mfa_devices
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- +migrate Down

DROP POLICY IF EXISTS mfa_devices_tenant_isolation ON mfa_devices;
ALTER TABLE mfa_devices DISABLE ROW LEVEL SECURITY;
ALTER TABLE mfa_devices NO FORCE ROW LEVEL SECURITY;

DROP TRIGGER IF EXISTS mfa_devices_updated_at ON mfa_devices;
DROP TABLE IF EXISTS mfa_devices;
