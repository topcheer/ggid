import { useState, useCallback } from "react";

export interface PerClientTtl {
  client_id: string;
  client_name: string;
  ttl_seconds: number;
}

export interface CacheStats {
  hits: number;
  misses: number;
  evictions: number;
  size: number;
}

export interface OAuthIntrospectionCacheConfig {
  cache_key_strategy: "token_hash" | "token_jti" | "client_token";
  ttl_seconds: number;
  max_entries: number;
  invalidation_on_revocation: boolean;
  per_client_ttl_override: PerClientTtl[];
  stampede_prevention: boolean;
  cache_stats: CacheStats;
}

export function useOAuthIntrospectionCacheConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthIntrospectionCacheConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-introspection-cache-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthIntrospectionCacheConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-introspection-cache-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
