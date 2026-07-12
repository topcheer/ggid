import { useState, useCallback } from "react";

export interface ForensicsData {
  hash_chain_verified: boolean;
  integrity_score: number;
  total_events: number;
  verified_events: number;
  tamper_evidence: { event_id: string; timestamp: string; type: string; description: string }[];
  insertion_gaps: { after_event: string; gap_duration: string; expected_events: number; actual_events: number }[];
  reorder_detected: { event_id: string; expected_seq: number; actual_seq: number }[];
}

export function useForensicsTimeline(baseUrl: string = "") {
  const [data, setData] = useState<ForensicsData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchForensics = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/forensics-timeline`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchForensics };
}
