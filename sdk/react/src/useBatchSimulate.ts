import { useState, useCallback } from "react";

export interface BatchData {
  results: { subject: string; resource: string; action: string; decision: "allow" | "deny"; matched_policy?: string }[];
  aggregate: { total: number; allowed: number; denied: number; mismatch_count: number };
}

export function useBatchSimulate(baseUrl: string = "") {
  const [data, setData] = useState<BatchData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const simulate = useCallback(async (subjects: string[], resources: string[], actions: string[]) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/batch-simulate`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ subjects, resources, actions }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, simulate };
}
