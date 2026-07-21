import { useState, useCallback } from "react";

export interface FrameworkInfo {
  framework: string;
  total_controls: number;
  covered: number;
  gaps: string[];
  coverage_pct: number;
}

export interface CoverageData {
  frameworks: FrameworkInfo[];
}

export function useFrameworkCoverage(baseUrl: string = "") {
  const [data, setData] = useState<CoverageData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCoverage = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/framework-coverage`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchCoverage };
}
