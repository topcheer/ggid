import { useState, useCallback } from "react";

export interface OrgNode {
  id: string;
  name: string;
  title: string;
  email: string;
  manager_id: string | null;
  department: string;
  children?: OrgNode[];
}

export function useOrgChart(baseUrl: string = "") {
  const [tree, setTree] = useState<OrgNode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTree = useCallback(async (orgId: string) => {
    if (!orgId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/org/orgs/${orgId}/chart`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: OrgNode = await res.json();
      setTree(data);
    } catch (e: any) {
      setError(e.message);
      setTree(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { tree, loading, error, fetchTree };
}
