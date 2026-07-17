-- 027_wasm_plugins.sql
-- WASM Plugin management: plugin registry + hook bindings.

CREATE TABLE IF NOT EXISTS wasm_plugins (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            TEXT NOT NULL UNIQUE,
    version         TEXT NOT NULL DEFAULT '1.0.0',
    author          TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    wasm_path       TEXT NOT NULL,
    wasm_hash       TEXT NOT NULL DEFAULT '',
    signature       TEXT NOT NULL DEFAULT '',
    config          JSONB NOT NULL DEFAULT '{}',
    hooks           TEXT[] NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    max_memory_mb   INT NOT NULL DEFAULT 16,
    timeout_ms      INT NOT NULL DEFAULT 100,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wasm_plugins_tenant ON wasm_plugins(tenant_id, enabled);
CREATE INDEX IF NOT EXISTS idx_wasm_plugins_name ON wasm_plugins(name);

CREATE TABLE IF NOT EXISTS wasm_plugin_hook_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    plugin_id       UUID NOT NULL REFERENCES wasm_plugins(id) ON DELETE CASCADE,
    hook_name       TEXT NOT NULL,
    priority        INT NOT NULL DEFAULT 100,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_hook_bindings_tenant ON wasm_plugin_hook_bindings(tenant_id, hook_name, enabled);
CREATE INDEX IF NOT EXISTS idx_hook_bindings_plugin ON wasm_plugin_hook_bindings(plugin_id);

COMMENT ON TABLE wasm_plugins IS 'WASM plugin registry with signing + resource limits';
COMMENT ON TABLE wasm_plugin_hook_bindings IS 'Per-hook plugin bindings with priority ordering';
