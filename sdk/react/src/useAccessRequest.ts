import { useState, useCallback } from "react";
export interface AccessReq { id: string; target_role: string; justification: string; duration_days: number; approver: string; status: "pending" | "approved" | "rejected" | "expired"; submitted_at: string; expires_at: string; days_remaining: number; comments: { author: string; text: string; timestamp: string }[]; }
export function useAccessRequest(baseUrl: string = "") {
  const [requests, setRequests] = useState<AccessReq[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchRequests = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/access-request"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setRequests(data.requests || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const submitRequest = useCallback(async (payload: { target_role: string; justification: string; duration_days: number; approver: string }) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/access-request", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const decide = useCallback(async (id: string, decision: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/access-request/" + id, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ decision }) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { requests, loading, error, fetchRequests, submitRequest, decide };
}
