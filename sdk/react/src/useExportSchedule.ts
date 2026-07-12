import { useState, useCallback } from "react";

export interface ExportJob {
  id: string;
  format: string;
  schedule_cron: string;
  filters: string;
  retention_days: number;
  destination_type: string;
  destination_config: string;
  last_export_at: string | null;
  status: "active" | "paused" | "failed";
}

export function useExportSchedule(baseUrl: string = "") {
  const [jobs, setJobs] = useState<ExportJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchJobs = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/export-schedule`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setJobs(data.jobs || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const createJob = useCallback(async (payload: Omit<ExportJob, "id" | "last_export_at" | "status">) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/export-schedule`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const deleteJob = useCallback(async (id: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/export-schedule/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { jobs, loading, error, fetchJobs, createJob, deleteJob };
}
