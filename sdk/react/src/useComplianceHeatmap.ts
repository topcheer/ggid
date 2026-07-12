import { useState, useCallback } from "react";

export interface HeatmapData {
  framework: string;
  controls: string[];
  months: string[];
  scores: Record<string, Record<string, number>>;
}

export function useComplianceHeatmap(baseUrl: string = "") {
  const [data, setData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHeatmap = useCallback(async (framework: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/compliance-heatmap?framework=${encodeURIComponent(framework)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: HeatmapData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchHeatmap };
}
