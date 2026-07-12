import { useState, useCallback } from "react";
export interface RotationEntry { client_id: string; client_name: string; interval_days: number; max_age_hours: number; notify_before_hours: number; auto_rotate: boolean; last_rotated: string; }
export function useTokenRotationPolicy(baseUrl: string = "") {
  const [data, setData] = useState<{ clients: RotationEntry[]; upcoming: { client_name: string; scheduled_at: string; overdue: boolean }[]; compliance_pct: number } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchPolicy = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/token-rotation-policy"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const savePolicy = useCallback(async (clients: RotationEntry[]) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/token-rotation-policy", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(clients) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchPolicy, savePolicy };
}
