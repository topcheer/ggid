import { useState, useCallback } from "react";
export interface LockoutConfig { max_failed_attempts: number; lockout_duration_minutes: number; progressive_backoff: boolean; captcha_trigger_attempts: number; auto_unlock_minutes: number; per_endpoint: { endpoint: string; max_attempts: number; duration_minutes: number }[]; ip_allowlist: string[]; }
export function useLockoutPolicy(baseUrl: string = "") {
  const [config, setConfig] = useState<LockoutConfig | null>(null);
  const [lockouts, setLockouts] = useState<{ username: string; ip: string; locked_at: string; unlock_at: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchPolicy = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/lockout-policy"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setConfig(d.config); setLockouts(d.active_lockouts || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const savePolicy = useCallback(async (c: LockoutConfig) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/lockout-policy", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(c) }); if (!res.ok) throw new Error("HTTP " + res.status); setConfig(c); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { config, lockouts, loading, error, fetchPolicy, savePolicy };
}
