import { useState, useCallback } from "react";

export interface PkceConfig {
  global_require_pkce: boolean;
  per_client: { client_id: string; client_name: string; required: boolean; challenge_method: "S256" | "plain" }[];
  exempted_clients: string[];
  compliance_pct: number;
}

export function usePkceEnforcement(baseUrl: string = "") {
  const [config, setConfig] = useState<PkceConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/pkce-enforcement");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const saveConfig = useCallback(async (cfg: PkceConfig) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/pkce-enforcement", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(cfg); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, saveConfig };
}
