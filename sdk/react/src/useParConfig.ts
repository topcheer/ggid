import { useState, useCallback } from "react";
export interface ParConfig { require_par: boolean; par_lifetime_seconds: number; max_request_size_kb: number; per_client: { client_id: string; client_name: string; required: boolean }[]; exempted_clients: string[]; }
export function useParConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<ParConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/par-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: ParConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/par-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(cfg); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
