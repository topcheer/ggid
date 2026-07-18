-- 037_verification_tokens.sql
CREATE TABLE IF NOT EXISTS verification_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    token       TEXT NOT NULL UNIQUE,
    type        TEXT NOT NULL, -- email_verification, password_reset
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_verif_token ON verification_tokens(token);
CREATE INDEX IF NOT EXISTS idx_verif_user ON verification_tokens(user_id, type);
