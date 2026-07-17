-- 013_sessions_jti.sql
-- CAE Phase 2: Add JTI (JWT ID) tracking to sessions for session revocation.
-- Allows the SessionRevocationManager to:
--   1. Look up active JTIs for a user when revoking
--   2. Add those JTIs to the Redis blocklist (ZSET)
--   3. Gateway CAECheck middleware will reject revoked tokens immediately

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS jti       TEXT;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS token_exp  TIMESTAMPTZ;

-- Fast lookup of active sessions by jti (for CAE revocation check).
CREATE INDEX IF NOT EXISTS idx_sessions_jti
    ON sessions (jti)
    WHERE revoked_at IS NULL AND jti IS NOT NULL;

COMMENT ON COLUMN sessions.jti      IS 'JWT ID (jti claim) of the access token issued for this session';
COMMENT ON COLUMN sessions.token_exp IS 'Expiry timestamp of the access token (used as Redis ZSET score for auto-cleanup)';
