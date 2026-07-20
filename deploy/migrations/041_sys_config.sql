-- 041: System configuration key-value store
-- Stores runtime-configurable settings (WebAuthn RP ID, feature flags, etc.)

CREATE TABLE IF NOT EXISTS sys_config (
    key        VARCHAR(100) PRIMARY KEY,
    value      JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID
);

-- Seed default WebAuthn config from environment (can be overridden via API)
INSERT INTO sys_config (key, value) VALUES
    ('webauthn_config', '{"rp_id": "", "rp_origins": [], "rp_display_name": "GGID"}'::jsonb)
ON CONFLICT (key) DO NOTHING;

INSERT INTO sys_config (key, value) VALUES
    ('system_config', '{"initialized": false, "bootstrap_completed": false}'::jsonb)
ON CONFLICT (key) DO NOTHING;
