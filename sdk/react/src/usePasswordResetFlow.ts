import { useState, useCallback } from "react";
export interface ResetConfig { methods: { email_link: boolean; sms_code: boolean; security_questions: boolean; admin_reset: boolean }; token_expiry_minutes: number; require_mfa: boolean; reset_after_failed_attempts: number; notify_on_reset: boolean; }
export function usePasswordResetFlow(baseUrl: string = "") {
  const [config, setConfig] = useState<ResetConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/password-reset-config"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setConfig(d.config || d); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveConfig = useCallback(async (cfg: ResetConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/password-reset-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig, saveConfig };
}
