import { useState, useCallback } from "react";

export interface PerClientScopes {
  client_id: string;
  client_name: string;
  allowed_scopes: string[];
}

export interface OAuthTokenExchangeConfig {
  enabled: boolean;
  allowed_subject_token_types: string[];
  allowed_actor_token_types: string[];
  audience_restriction_policy: "strict" | "permissive" | "none";
  per_client_allowed_scopes: PerClientScopes[];
  max_delegation_depth: number;
}

export function useOAuthTokenExchangeConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthTokenExchangeConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-token-exchange-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthTokenExchangeConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-token-exchange-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
