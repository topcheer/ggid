import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface AnomalyEvent {
  id: string;
  type: string;
  severity: string;
  user: string;
  description: string;
  timestamp: string;
  confidence: number;
}

export interface DetectedPattern {
  pattern: string;
  count: number;
  auto_action: string;
}

export interface AnomalyDetectDashboardData {
  events: AnomalyEvent[];
  patterns: DetectedPattern[];
}

export function useAnomalyDetectDashboard() {
  const [data, setData] = useState<AnomalyDetectDashboardData | null>(null);
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
        events: [
          { id: "ae1", type: "impossible_travel", severity: "critical", user: "user_jd", description: "Login from Tokyo 5min after New York login", timestamp: "10m ago", confidence: 98 },
          { id: "ae2", type: "off_hours_access", severity: "medium", user: "user_sm", description: "Admin access at 3AM local time", timestamp: "1h ago", confidence: 82 },
          { id: "ae3", type: "new_device", severity: "low", user: "user_al", description: "First login from new device fingerprint", timestamp: "2h ago", confidence: 65 },
          { id: "ae4", type: "unusual_resource", severity: "high", user: "user_mk", description: "Accessed admin panel (first time)", timestamp: "3h ago", confidence: 91 },
        ],
        patterns: [
          { pattern: "off_hours_access", count: 23, auto_action: "require_mfa" },
          { pattern: "impossible_travel", count: 5, auto_action: "block" },
          { pattern: "new_device", count: 45, auto_action: "verify_email" },
          { pattern: "unusual_resource", count: 12, auto_action: "flag_review" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const acknowledge = useCallback((id: string) => { console.log("Ack", id); }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, acknowledge };
}
