import { useState, useCallback } from "react";
export interface Milestone { id: string; name: string; status: "completed" | "in_progress" | "pending" | "overdue"; due_date: string; responsible_party: string; }
export interface FrameworkData { framework: string; milestones: Milestone[]; }
export function useComplianceTimeline(baseUrl: string = "") {
  const [data, setData] = useState<FrameworkData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async (framework: string) => {
    setLoading(true); setError(null);
    try { const res = await fetch(baseUrl + "/api/v1/audit/compliance-timeline?framework=" + encodeURIComponent(framework)); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); }
  }, [baseUrl]);
  return { data, loading, error, fetchData };
}
