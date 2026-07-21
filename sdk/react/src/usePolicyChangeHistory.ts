import { useState, useCallback } from "react";

export interface ChangeEntry {
  version: string;
  changed_by: string;
  changed_at: string;
  change_type: "create" | "modify" | "delete";
  diff_summary: string;
  approved_by: string;
}

export function usePolicyChangeHistory(baseUrl: string = "") {
  const [history, setHistory] = useState<ChangeEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async (policyId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/policy/" + policyId + "/change-history");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setHistory(data.history || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const rollback = useCallback(async (policyId: string, version: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/policy/" + policyId + "/rollback", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ version }) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { history, loading, error, fetchHistory, rollback };
}
