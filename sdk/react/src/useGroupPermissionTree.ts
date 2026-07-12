import { useState, useCallback } from "react";
export interface TreeNode { id: string; name: string; type: string; children?: TreeNode[]; permissions: string[]; }
export function useGroupPermissionTree(baseUrl: string = "") {
  const [tree, setTree] = useState<TreeNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchTree = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/group-permission-tree"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setTree(d.tree || d || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { tree, loading, error, fetchTree };
}
