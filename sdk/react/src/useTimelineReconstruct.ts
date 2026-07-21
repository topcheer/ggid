import { useState, useCallback } from "react";

export interface ReconstructData {
  events: { id: string; timestamp: string; event_type: string; description: string; source: string; ip?: string; correlated_with?: string }[];
  correlation_chains: { chain_id: string; event_ids: string[]; pattern: string }[];
  gaps: { after_event: string; gap_minutes: number; severity: "low" | "medium" | "high" }[];
  anomaly_windows: { start: string; end: string; type: string }[];
}

export function useTimelineReconstruct(baseUrl: string = "") {
  const [data, setData] = useState<ReconstructData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reconstruct = useCallback(async (userId?: string, sessionId?: string) => {
    setLoading(true); setError(null);
    try {
      const params = new URLSearchParams();
      if (userId) params.set("user_id", userId);
      if (sessionId) params.set("session_id", sessionId);
      const res = await fetch(`${baseUrl}/api/v1/audit/timeline-reconstruct?${params}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, reconstruct };
}
