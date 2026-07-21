import { useState, useCallback } from "react";

export interface DriftData {
  framework: string;
  drift_score: number;
  changed_controls: { control_id: string; name: string; was_status: string; now_status: string; drift_score: number; risk_level: "low" | "medium" | "high" }[];
}

export function useComplianceDrift(baseUrl: string = "") {
  const [data, setData] = useState<DriftData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDrift = useCallback(async (framework: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/compliance-drift?framework=${encodeURIComponent(framework)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); setData(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchDrift };
}
