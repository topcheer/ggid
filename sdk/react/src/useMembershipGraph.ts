import { useState, useCallback } from "react";

export interface GraphData {
  group_id: string;
  group_name: string;
  total_depth: number;
  direct_members: { id: string; name: string; type: "user" | "group" }[];
  nested_groups: { id: string; name: string; depth: number; children?: { id: string; name: string }[] }[];
  parent_groups: { id: string; name: string }[];
  circular_detected: boolean;
  circular_path?: string[];
}

export function useMembershipGraph(baseUrl: string = "") {
  const [data, setData] = useState<GraphData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGraph = useCallback(async (groupId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/membership-graph?group_id=${encodeURIComponent(groupId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchGraph };
}
