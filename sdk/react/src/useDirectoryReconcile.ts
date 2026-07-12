import { useState, useCallback } from "react";

export interface ReconcileData {
  orphaned_ids: { id: string; type: string; source: string; last_seen: string }[];
  duplicate_groups: { group_id: string; entries: { id: string; source: string; email: string }[]; suggested_merge_target: string }[];
  cleanup_plan: { action: string; count: number; risk: "low" | "medium" | "high" }[];
}

export function useDirectoryReconcile(baseUrl: string = "") {
  const [data, setData] = useState<ReconcileData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchReconcile = useCallback(async (dryRun: boolean = true) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/directory-reconcile?dry_run=${dryRun}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const execute = useCallback(async (dryRun: boolean, mergeStrategy: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/directory-reconcile/execute`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ dry_run: dryRun, merge_strategy: mergeStrategy }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchReconcile, execute };
}
