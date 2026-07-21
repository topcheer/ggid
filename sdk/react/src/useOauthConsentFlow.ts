import { useState, useCallback } from "react";
export interface ConsentConfig { logo_url: string; privacy_policy_url: string; tos_url: string; show_skip_consent: boolean; remember_consent_duration_days: number; scope_descriptions: Record<string, string>; pre_approved_apps: { client_id: string; client_name: string }[]; }
export function useOauthConsentFlow(baseUrl: string = "") {
  const [config, setConfig] = useState<ConsentConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/consent-flow-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: ConsentConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/consent-flow-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(cfg); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
