import { useState, useCallback } from "react";
export interface DeprovisionItem { id: string; username: string; scheduled_date: string; reason: string; status: string; steps: { name: string; status: string }[]; }
export function useDeprovisionDashboard(baseUrl: string = "") {
  const [data, setData] = useState<{ scheduled: DeprovisionItem[]; in_progress: DeprovisionItem[]; completed_today: number; failed: DeprovisionItem[] } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/deprovision-dashboard"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const retry = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/deprovision/" + id + "/retry", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData, retry };
}
