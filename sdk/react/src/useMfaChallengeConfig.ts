import { useState, useCallback } from "react";
export interface MfaConfig { method_priority: string[]; step_up_actions: string[]; challenge_frequency: string; fallback_method: boolean; grace_period_minutes: number; }
export function useMfaChallengeConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<MfaConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/mfa-challenge-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: MfaConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/mfa-challenge-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(cfg); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
