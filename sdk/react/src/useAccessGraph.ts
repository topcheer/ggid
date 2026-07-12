import { useState, useCallback } from "react";

export interface AccessData {
  subject: string;
  direct_permissions: { resource: string; action: string; source: string }[];
  inherited_permissions: { resource: string; action: string; source: string }[];
  effective_permissions: { resource: string; actions: string[] }[];
  via_groups: string[];
  via_roles: string[];
}

export function useAccessGraph(baseUrl: string = "") {
  const [data, setData] = useState<AccessData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGraph = useCallback(async (subject: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-graph?subject=${encodeURIComponent(subject)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchGraph };
}
