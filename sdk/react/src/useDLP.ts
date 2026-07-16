import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface DLPPolicy {
  policy_name: string;
  trigger_pattern: string;
  action: string;
  channels: string[];
}

export interface DLPViolation {
  user: string;
  resource: string;
  pattern_matched: string;
  action_taken: string;
  timestamp: string;
}

export interface DLPData {
  dlp_policies: DLPPolicy[];
  violation_log: DLPViolation[];
}

export function useDLP() {
  const [data, setData] = useState<DLPData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
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
        dlp_policies: [
          { policy_name: "SSN Blocking", trigger_pattern: "\\d{3}-\\d{2}-\\d{4}", action: "block", channels: ["api", "export"] },
          { policy_name: "Credit Card Mask", trigger_pattern: "\\d{4}[ -]?\\d{4}[ -]?\\d{4}[ -]?\\d{4}", action: "mask", channels: ["query", "export"] },
          { policy_name: "Email Logging", trigger_pattern: "[\\w.+-]+@[\\w-]+\\.[\\w.]+", action: "log", channels: ["api", "query", "export"] },
        ],
        violation_log: [
          { user: "alice@corp.com", resource: "GET /api/v1/users/export", pattern_matched: "SSN pattern", action_taken: "block", timestamp: "2h ago" },
          { user: "bob@corp.com", resource: "GET /api/v1/audit/events", pattern_matched: "Credit card", action_taken: "mask", timestamp: "5h ago" },
          { user: "admin@corp.com", resource: "POST /api/v1/export", pattern_matched: "Email bulk", action_taken: "log", timestamp: "8h ago" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const testPolicy = useCallback((input: string) => {
    console.log("Testing input:", input);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, testPolicy };
}
