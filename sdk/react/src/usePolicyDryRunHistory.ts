import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface DryRunEntry {
  run_id: string;
  policy: string;
  subject: string;
  decision: string;
  executed_by: string;
  timestamp: string;
  duration_ms: number;
}

export interface PolicyDryRunHistoryData {
  history: DryRunEntry[];
  saved_run_templates: string[];
}

export function usePolicyDryRunHistory() {
  const [data, setData] = useState<PolicyDryRunHistoryData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        history: [
          { run_id: "run-001", policy: "prod-access-mfa", subject: "alice.chen", decision: "allow", executed_by: "admin.bob", timestamp: "10m ago", duration_ms: 12 },
          { run_id: "run-002", policy: "admin-ip-restriction", subject: "carol.jones", decision: "deny", executed_by: "admin.bob", timestamp: "30m ago", duration_ms: 8 },
          { run_id: "run-003", policy: "weekend-block", subject: "dave.wilson", decision: "deny", executed_by: "system.test", timestamp: "1h ago", duration_ms: 15 },
          { run_id: "run-004", policy: "data-export-stepup", subject: "eve.brown", decision: "allow", executed_by: "eve.brown", timestamp: "2h ago", duration_ms: 10 },
          { run_id: "run-005", policy: "geo-fencing", subject: "frank.lee", decision: "not_applicable", executed_by: "admin.bob", timestamp: "3h ago", duration_ms: 6 },
          { run_id: "run-006", policy: "prod-access-mfa", subject: "grace.kim", decision: "allow", executed_by: "grace.kim", timestamp: "4h ago", duration_ms: 11 },
        ],
        saved_run_templates: ["Admin access test", "Weekend policy test", "Geo-fence validation", "MFA enforcement check"],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const replayRun = useCallback(async (_runId: string) => {
    console.log("Replaying run:", _runId);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, replayRun };
}
