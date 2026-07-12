import { useState, useCallback } from "react";
export interface RecoveryConfig { methods: string[]; verification_steps: string[]; enabled: boolean; }
export interface RecoveryCode { user_id: string; username: string; total: number; used: number; remaining: number; generated_at: string; }
export function useAccountRecovery(baseUrl: string = "") {
  const [config, setConfig] = useState<RecoveryConfig | null>(null);
  const [codes, setCodes] = useState<RecoveryCode[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/account-recovery"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setConfig(d.config); setCodes(d.recovery_codes || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { config, codes, loading, error, fetchData };
}
