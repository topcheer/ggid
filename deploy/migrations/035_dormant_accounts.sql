-- 035_dormant_accounts.sql
-- Dormant account lifecycle + ghost reconciliation.

CREATE TABLE IF NOT EXISTS user_lifecycle_state (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL UNIQUE,
    tenant_id       UUID,
    state           TEXT NOT NULL DEFAULT 'active', -- active, dormant, suspended, archived
    dormant_since   TIMESTAMPTZ,
    suspended_at    TIMESTAMPTZ,
    archived_at     TIMESTAMPTZ,
    last_login_at   TIMESTAMPTZ,
    notified_at     TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_lifecycle_state ON user_lifecycle_state(state);
CREATE INDEX IF NOT EXISTS idx_lifecycle_user ON user_lifecycle_state(user_id);

CREATE TABLE IF NOT EXISTS ghost_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    tenant_id       UUID,
    email           TEXT,
    recommendation  TEXT NOT NULL DEFAULT 'disable', -- disable, archive, investigate
    status          TEXT NOT NULL DEFAULT 'flagged', -- flagged, approved, actioned, dismissed
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    actioned_at     TIMESTAMPTZ,
    details         JSONB DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_ghost_status ON ghost_accounts(status);
