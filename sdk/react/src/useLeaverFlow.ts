import { useState, useCallback } from "react";

export interface LeaverData {
  employee_id: string;
  employee_name: string;
  scheduled_date: string;
  cascade_to_apps: boolean;
  tasks: { id: string; label: string; done: boolean; status: string }[];
  completion_pct: number;
  overall_status: string;
}

export function useLeaverFlow(baseUrl: string = "") {
  const [data, setData] = useState<LeaverData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchLeaver = useCallback(async (userId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/leaver-flow?user_id=" + encodeURIComponent(userId));
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const triggerLeaver = useCallback(async (userId: string, scheduledDate: string, cascade: boolean) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/leaver-flow", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ user_id: userId, scheduled_date: scheduledDate, cascade_to_apps: cascade }) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchLeaver, triggerLeaver };
}
