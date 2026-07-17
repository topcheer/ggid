-- 016_jit_requests.sql
-- PAM JIT Zero Standing Privileges — temporary role elevation requests.
-- Lifecycle: pending → approved/rejected → active → expired/revoked

CREATE TABLE IF NOT EXISTS jit_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    role_id         UUID NOT NULL,
    scope_type      TEXT NOT NULL DEFAULT 'tenant',
    scope_id        UUID,
    reason          TEXT NOT NULL,
    duration_min    INT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    approver_id     UUID,
    approved_at     TIMESTAMPTZ,
    activated_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    revoked_reason  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_jit_req_user   ON jit_requests (tenant_id, user_id, status);
CREATE INDEX IF NOT EXISTS idx_jit_req_expiry ON jit_requests (status, expires_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_jit_req_status ON jit_requests (tenant_id, status);

COMMENT ON TABLE jit_requests IS 'PAM JIT zero-standing privilege elevation requests';
