import { useState, useCallback } from "react";
export interface VerifConfig { methods: { document: boolean; face: boolean; kba: boolean; phone: boolean; email: boolean }; required_factors: number; confidence_threshold: number; risk_matrix: { level: string; factors: number }[]; }
export function useIdentityVerification(baseUrl: string = "") {
  const [config, setConfig] = useState<VerifConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/identity-verification"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setConfig(d.config || d); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { config, loading, error, fetchConfig };
}
