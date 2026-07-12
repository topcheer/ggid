import { useState, useCallback } from "react";
export interface OnboardingItem { id: string; employee: string; start_date: string; steps_completed: number; total_steps: number; blocked_items: string[]; provisioning: { app: string; status: string }[]; }
export function useJoinerFlowDashboard(baseUrl: string = "") {
  const [data, setData] = useState<{ pending: OnboardingItem[]; completion_rate: number; avg_days_to_complete: number; upcoming_starts: { employee: string; start_date: string }[] } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/joiner-dashboard"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
