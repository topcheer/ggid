import { useState, useCallback, useEffect } from "react";

export interface CacheConfig {
  enabled: boolean;
  ttl_seconds: number;
  max_entries: number;
}

export interface CachedToken {
  token_hash: string;
  client: string;
  cached_at: string;
  expires_at: string;
}

export interface InvalidationRule {
  trigger: string;
  action: string;
  description: string;
}

export interface OAuthIntrospectionCacheData {
  cache_config: CacheConfig;
  hit_rate_pct: number;
  cache_size_bytes: number;
  evictions_per_min: number;
  cached_tokens: CachedToken[];
  cache_invalidation_rules: InvalidationRule[];
}

export function useOAuthIntrospectionCache() {
  const [data, setData] = useState<OAuthIntrospectionCacheData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        cache_config: { enabled: true, ttl_seconds: 60, max_entries: 10000 },
        hit_rate_pct: 87,
        cache_size_bytes: 4194304,
        evictions_per_min: 12,
        cached_tokens: [
          { token_hash: "sha256-a1b2c3d4e5f6", client: "client-web-001", cached_at: "2m ago", expires_at: "in 58s" },
          { token_hash: "sha256-g7h8i9j0k1l2", client: "client-mobile-002", cached_at: "5m ago", expires_at: "in 30s" },
          { token_hash: "sha256-m3n4o5p6q7r8", client: "client-api-003", cached_at: "1m ago", expires_at: "in 59s" },
          { token_hash: "sha256-s9t0u1v2w3x4", client: "client-spa-005", cached_at: "30s ago", expires_at: "in 30s" },
        ],
        cache_invalidation_rules: [
          { trigger: "token_revoked", action: "purge_entry", description: "Remove cached entry when token is revoked" },
          { trigger: "client_disabled", action: "purge_all_client", description: "Purge all entries for disabled client" },
          { trigger: "ttl_expired", action: "evict", description: "Auto-evict expired entries" },
          { trigger: "scope_changed", action: "refresh", description: "Refresh entry when token scopes change" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const purgeCache = useCallback(async () => {
    console.log("Purging introspection cache");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, purgeCache };
}
