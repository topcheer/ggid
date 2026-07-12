import { useState, useCallback } from "react";
export interface Challenge { id: string; title: string; control: string; severity: string; status: string; opened_at: string; compliance_impact: string; }
export function useAuditChallengeResponse(baseUrl: string = "") {
  const [open, setOpen] = useState<Challenge[]>([]);
  const [resolved, setResolved] = useState<Challenge[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/challenge-response"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setOpen(d.open || []); setResolved(d.resolved || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { open, resolved, loading, error, fetchData };
}
