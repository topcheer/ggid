import { useState, useCallback } from "react";

export interface ScheduledExportJob {
  id: string;
  name: string;
  cron: string;
  format: "csv" | "json" | "parquet";
  filters: Record<string, string>;
  retention_days: number;
  destination: string;
  last_run: string;
  next_run: string;
}

export interface RetryPolicy {
  max_attempts: number;
  backoff_seconds: number;
}

export interface AuditExportScheduleConfig {
  scheduled_jobs: ScheduledExportJob[];
  max_concurrent: number;
  retry_policy: RetryPolicy;
  notification_on_complete: { enabled: boolean; webhook_url: string };
}

export function useAuditExportScheduleConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AuditExportScheduleConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/audit-export-schedule-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AuditExportScheduleConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/audit-export-schedule-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
