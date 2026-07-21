import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface LateralMovementPattern {
  id: string;
  user: string;
  resource_chain: string[];
  access_velocity: number;
  timeline: string;
  mitre_techniques: string[];
  kill_chain_stage: string;
  confidence_score: number;
  status: string;
}

export interface LateralMovementDetectData {
  detected_patterns: LateralMovementPattern[];
}

export function useLateralMovementDetect() {
  const [data, setData] = useState<LateralMovementDetectData | null>(null);
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
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        detected_patterns: [
          { id: "lm-001", user: "compromised.svc", resource_chain: ["web-app", "db-proxy", "file-share", "dc-server"], access_velocity: 12, timeline: "5m span", mitre_techniques: ["T1021.002", "T1075"], kill_chain_stage: "lateral_movement", confidence_score: 0.92, status: "blocked" },
          { id: "lm-002", user: "temp.admin", resource_chain: ["portal", "api-gateway", "k8s-secrets", "vault"], access_velocity: 8, timeline: "3m span", mitre_techniques: ["T1552.004", "T1528"], kill_chain_stage: "credential_access", confidence_score: 0.85, status: "investigating" },
          { id: "lm-003", user: "user.789", resource_chain: [ "wiki", "jira", "repo-access", "ci-runner"], access_velocity: 5, timeline: "8m span", mitre_techniques: ["T1078.004"], kill_chain_stage: "discovery", confidence_score: 0.68, status: "investigating" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
