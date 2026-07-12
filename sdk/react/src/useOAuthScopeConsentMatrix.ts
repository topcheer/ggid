import { useState, useCallback } from "react";
export interface ScopeMatrix { scopes: string[]; consent_levels: string[]; assignments: Record<string, string>; risk_levels: Record<string, string>; }
export function useOAuthScopeConsentMatrix(baseUrl: string = "") {
  const [data, setData] = useState<ScopeMatrix | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchMatrix = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/scope-consent-matrix"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchMatrix };
}
