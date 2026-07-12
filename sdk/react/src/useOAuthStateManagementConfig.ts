import { useState, useCallback } from "react";

export interface PerFlowEncoding {
  flow: string;
  encoding: string;
}

export interface OAuthStateManagementConfig {
  state_length_bytes: number;
  binding_method: "session" | "cookie" | "jwt";
  state_ttl_seconds: number;
  per_flow_encoding: PerFlowEncoding[];
  validation_strictness: "strict" | "standard" | "lenient";
}

export function useOAuthStateManagementConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthStateManagementConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-state-management-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthStateManagementConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-state-management-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
