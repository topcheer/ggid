import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface SavedQuery {
  id: string;
  name: string;
  description: string;
  query_body: string;
  tags: string[];
  last_run: string | null;
  results_count: number;
  schedule: string | null;
}

export interface PopularQuery {
  name: string;
  description: string;
  run_count: number;
}

export interface AuditQueryLibraryData {
  saved_queries: SavedQuery[];
  popular_queries: PopularQuery[];
}

export function useAuditQueryLibrary() {
  const [data, setData] = useState<AuditQueryLibraryData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        saved_queries: [
          { id: "q1", name: "Failed Logins (24h)", description: "All failed authentication attempts in last 24 hours", query_body: "event_type = \"auth.login.failed\" AND timestamp > now() - 24h", tags: ["security", "auth"], last_run: "2h ago", results_count: 1250, schedule: "every 1h" },
          { id: "q2", name: "Admin Role Changes", description: "Track all admin role grants and revocations", query_body: "event_type = \"role.change\" AND target_role IN (\"admin\", \"superadmin\")", tags: ["compliance", "admin"], last_run: "1d ago", results_count: 42, schedule: "daily" },
          { id: "q3", name: "MFA Bypass Attempts", description: "Users attempting to bypass MFA", query_body: "event_type = \"mfa.bypass\" OR (event_type = \"mfa.challenge\" AND outcome = \"denied\")", tags: ["security", "mfa"], last_run: "3h ago", results_count: 18, schedule: null },
          { id: "q4", name: "Off-Hours Access", description: "Access between 22:00 and 06:00 UTC", query_body: "hour(timestamp) >= 22 OR hour(timestamp) < 6", tags: ["security", "anomaly"], last_run: "5h ago", results_count: 340, schedule: "daily" },
          { id: "q5", name: "Token Export Activity", description: "Bulk token or data export events", query_body: "event_type = \"data.export\" AND records > 1000", tags: ["compliance", "data"], last_run: "1d ago", results_count: 7, schedule: null },
          { id: "q6", name: "New Device Logins", description: "First-time logins from unrecognized devices", query_body: "event_type = \"auth.login\" AND device.is_new = true", tags: ["security", "device"], last_run: "30m ago", results_count: 89, schedule: "every 30m" },
        ],
        popular_queries: [
          { name: "Failed Logins (24h)", description: "All failed authentication attempts", run_count: 3420 },
          { name: "Admin Role Changes", description: "Admin role grants and revocations", run_count: 2180 },
          { name: "New Device Logins", description: "First-time device logins", run_count: 1950 },
          { name: "Off-Hours Access", description: "Access during 22:00-06:00 UTC", run_count: 1420 },
          { name: "SCIM Provisioning Events", description: "User provisioning/deprovisioning", run_count: 980 },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, isDemoData };
}
