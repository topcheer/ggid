import { useState, useCallback } from "react";

export interface HijackData {
  user_id: string;
  username: string;
  confidence_score: number;
  events: { id: string; timestamp: string; event_type: string; description: string; details: string }[];
  recommended_actions: string[];
}

export function useHijackTimeline(baseUrl: string = "") {
  const [data, setData] = useState<HijackData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTimeline = useCallback(async (userId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/hijack-timeline?user_id=${encodeURIComponent(userId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchTimeline };
}
