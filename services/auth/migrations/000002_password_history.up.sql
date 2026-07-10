-- Password history table for enforcing password rotation policies.
-- Stores the last N password hashes per user to prevent reuse.

-- +migrate Up

CREATE TABLE IF NOT EXISTS password_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user ON password_history (tenant_id, user_id, created_at DESC);

-- RLS for multi-tenant isolation.
ALTER TABLE password_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE password_history FORCE ROW LEVEL SECURITY;

CREATE POLICY password_history_tenant_isolation ON password_history
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );

-- +migrate Down

DROP POLICY IF EXISTS password_history_tenant_isolation ON password_history;
ALTER TABLE password_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE password_history NO FORCE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS password_history;
