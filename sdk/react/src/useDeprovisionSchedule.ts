import { useState, useCallback } from "react";

export interface DeprovisionJob {
  id: string;
  user_id: string;
  username: string;
  scheduled_at: string;
  reason: string;
  cascade_to_apps: boolean;
  notify_before_days: number;
  status: "scheduled" | "completed" | "cancelled";
}

export function useDeprovisionSchedule(baseUrl: string = "") {
  const [jobs, setJobs] = useState<DeprovisionJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchJobs = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/deprovision-schedule`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setJobs(data.jobs || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const createJob = useCallback(async (userId: string, scheduledAt: string, reason: string, cascadeToApps: boolean, notifyBeforeDays: number) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/deprovision-schedule`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ user_id: userId, scheduled_at: scheduledAt, reason, cascade_to_apps: cascadeToApps, notify_before_days: notifyBeforeDays }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const cancelJob = useCallback(async (id: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/deprovision-schedule/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { jobs, loading, error, fetchJobs, createJob, cancelJob };
}
