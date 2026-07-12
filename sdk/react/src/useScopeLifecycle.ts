import { useState, useCallback } from "react";

export interface ScopeRequest {
  id: string;
  scope: string;
  requester: string;
  approver_chain: { approver: string; status: "pending" | "approved" | "rejected"; acted_at?: string }[];
  status: "pending" | "approved" | "rejected" | "expired";
  risk_level: "low" | "medium" | "high";
  requested_at: string;
  auto_expire_days: number;
  days_remaining: number;
}

export function useScopeLifecycle(baseUrl: string = "") {
  const [requests, setRequests] = useState<ScopeRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/scope-lifecycle`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setRequests(data.requests || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { requests, loading, error, fetchRequests };
}
