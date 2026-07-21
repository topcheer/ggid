import { useState, useCallback } from "react";
export interface ScopeDriftData { unused_scopes: { scope: string; last_used_days_ago: number; severity: string }[]; unregistered_scopes: string[]; drift_trend_30d: { date: string; value: number }[]; recommendations: string[]; }
export function useScopeDrift(baseUrl: string = "") {
  const [data, setData] = useState<ScopeDriftData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchDrift = useCallback(async (clientId: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/scope-drift?client_id=" + encodeURIComponent(clientId)); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const revokeUnused = useCallback(async (clientId: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/scope-drift/revoke-unused", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ client_id: clientId }) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchDrift, revokeUnused };
}
