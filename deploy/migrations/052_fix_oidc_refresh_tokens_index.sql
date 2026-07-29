-- Fix the partial index from 051: its WHERE revoked = false predicate cannot be
-- used by GetRefreshToken, which queries (tenant_id, token_hash) without a
-- revoked filter — PostgreSQL only uses a partial index when the query implies
-- the predicate. Recreate as a plain composite index (R135 P2-1).

DROP INDEX IF EXISTS idx_oidc_refresh_tokens_tenant_hash;

CREATE INDEX IF NOT EXISTS idx_oidc_refresh_tokens_tenant_hash
    ON oidc_refresh_tokens(tenant_id, token_hash);
