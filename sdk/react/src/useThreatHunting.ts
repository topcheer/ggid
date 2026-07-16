import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface HuntResult {
  entity: string;
  severity: string;
  description: string;
  timestamp: string;
}

export interface Hypothesis {
  id: string;
  hypothesis: string;
  status: string;
  evidence_count: number;
  conclusion: string;
}

export interface SavedHunt {
  name: string;
  last_run: string;
}

export interface ThreatHuntingData {
  hunt_results: HuntResult[];
  hypotheses: Hypothesis[];
  saved_hunts: SavedHunt[];
  watchlist: string[];
}

export function useThreatHunting() {
  const [data, setData] = useState<ThreatHuntingData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        hunt_results: [
          { entity: "185.220.101.45", severity: "high", description: "TOR exit node, 12 failed logins across 4 accounts", timestamp: "5m ago" },
          { entity: "svc.legacy", severity: "medium", description: "Service account accessing endpoints outside normal pattern", timestamp: "15m ago" },
        ],
        hypotheses: [
          { id: "h-001", hypothesis: "Insider using stolen credentials for lateral movement", status: "investigating", evidence_count: 8, conclusion: "" },
          { id: "h-002", hypothesis: "Compromised service account exfiltrating data", status: "confirmed", evidence_count: 15, conclusion: "Confirmed - account suspended, IR triggered" },
          { id: "h-003", hypothesis: "Automated scanner probing auth endpoints", status: "disproven", evidence_count: 3, conclusion: "False positive - legitimate pen test" },
        ],
        saved_hunts: [
          { name: "TOR connections after hours", last_run: "2h ago" },
          { name: "Mass login failures by IP", last_run: "4h ago" },
          { name: "Service accounts new IPs", last_run: "1d ago" },
        ],
        watchlist: ["203.0.113.50", "198.51.100.22", "svc.legacy", "temp.admin"],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
