import { useState, useCallback } from "react";

export interface IntrospectionCaching {
  enabled: boolean;
  ttl_seconds: number;
  max_entries: number;
}

export interface ResourceServerAuth {
  resource_server: string;
  auth_required: boolean;
  scope: string;
}

export interface TokenIntrospectionDesign {
  caching: IntrospectionCaching;
  scope_filtering: boolean;
  per_resource_server_auth: ResourceServerAuth[];
  rate_limit_per_client: number;
  privacy_mode: boolean;
}

export function useTokenIntrospectionDesign(baseUrl: string = "") {
  const [config, setConfig] = useState<TokenIntrospectionDesign | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-introspection-design`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<TokenIntrospectionDesign>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-introspection-design`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
