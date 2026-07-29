-- Add composite index for OAuth refresh token lookup by tenant_id + token_hash.
-- Without this index, the query in pg_repo.go (GetRefreshToken) does a full table scan.

CREATE INDEX IF NOT EXISTS idx_oidc_refresh_tokens_tenant_hash
    ON oidc_refresh_tokens(tenant_id, token_hash)
    WHERE revoked = false;
