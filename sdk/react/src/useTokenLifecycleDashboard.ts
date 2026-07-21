import { useState, useCallback } from "react";
export interface TokenDashboard { stages: { stage: string; count: number; color: string }[]; avg_lifetime_hours: number; refresh_rate: number; churn_30d: { date: string; value: number }[]; issuance_rate: number; revocation_rate: number; by_client: { client_name: string; active: number; expiring: number; revoked: number }[]; }
export function useTokenLifecycleDashboard(baseUrl: string = "") {
  const [data, setData] = useState<TokenDashboard | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/token-lifecycle"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
