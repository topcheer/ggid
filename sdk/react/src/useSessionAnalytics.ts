import { useState, useCallback } from "react";
export interface SessionData { active_count: number; avg_duration_minutes: number; revocation_rate: number; peak_concurrent: number; peak_hour: string; by_platform: { platform: string; count: number }[]; by_location: { location: string; count: number }[]; top_users: { user_id: string; username: string; session_count: number; avg_duration: number }[]; }
export function useSessionAnalytics(baseUrl: string = "") {
  const [data, setData] = useState<SessionData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/session-analytics"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
