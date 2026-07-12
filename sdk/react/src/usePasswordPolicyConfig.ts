import { useState, useCallback } from "react";
export interface PasswordConfig { min_length: number; require_uppercase: boolean; require_lowercase: boolean; require_digit: boolean; require_special: boolean; check_dictionary: boolean; check_breach: boolean; expiry_days: number; history_count: number; per_role: { role: string; min_length: number; expiry_days: number }[]; }
export function usePasswordPolicyConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PasswordConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/password-policy-config"); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: PasswordConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/password-policy-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(cfg); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
