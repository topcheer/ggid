import { useState, useCallback } from "react";

export interface PkceClientEntry {
  client_id: string;
  client_name: string;
  required: boolean;
  method: "S256" | "plain" | "none";
}

export interface PkceDeepDiveConfig {
  code_challenge_method: "S256" | "plain";
  per_client_enforcement: PkceClientEntry[];
  migrate_non_pkce_clients: boolean;
  compliance_pct: number;
}

export function usePkceDeepDiveConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PkceDeepDiveConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/pkce-deep-dive-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<PkceDeepDiveConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/pkce-deep-dive-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
