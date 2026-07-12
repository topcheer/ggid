import { useState, useCallback } from "react";

export interface MatrixData {
  subjects: string[];
  resources: string[];
  cells: { subject: string; resource: string; coverage_pct: number; policies: number }[];
  uncovered: { subject: string; resource: string }[];
  redundant: { subject: string; resource: string; count: number }[];
  gaps_count: number;
}

export function useCoverageMatrix(baseUrl: string = "") {
  const [data, setData] = useState<MatrixData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMatrix = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/coverage-matrix`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchMatrix };
}
