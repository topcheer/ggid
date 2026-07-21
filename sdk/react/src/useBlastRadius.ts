import { useState, useCallback } from "react";

export interface BlastRadiusData {
  affected_users_count: number;
  affected_roles: string[];
  affected_resources: { name: string; type: string; children?: { name: string; type: string }[] }[];
  cascading_policies: string[];
}

export function useBlastRadius(baseUrl: string = "") {
  const [data, setData] = useState<BlastRadiusData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const analyze = useCallback(async (policyId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/blast-radius?id=${encodeURIComponent(policyId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, analyze };
}
