import { useState, useCallback } from "react";

export interface TreeNode {
  id: string;
  name: string;
  title: string;
  reports: TreeNode[];
  span_of_control: number;
  layer: number;
  is_orphan: boolean;
}

export interface OrgTreeData {
  root: TreeNode | null;
  total_layers: number;
  orphan_managers: { id: string; name: string }[];
  circular_detected: boolean;
  circular_path?: string[];
}

export function useReportingStructure(baseUrl: string = "") {
  const [data, setData] = useState<OrgTreeData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTree = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/reporting-structure`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchTree };
}
