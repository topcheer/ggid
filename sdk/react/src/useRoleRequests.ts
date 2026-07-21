import { useState, useCallback } from "react";

export interface RoleRequest {
  id: string;
  requester: string;
  requester_email: string;
  requested_role: string;
  justification: string;
  status: "pending" | "approved" | "rejected" | "completed";
  approval_step: { step: number; total: number; current_approver: string };
  created_at: string;
  decided_at: string | null;
  is_mine: boolean;
}

export function useRoleRequests(baseUrl: string = "") {
  const [requests, setRequests] = useState<RoleRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/role-requests`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setRequests(data.requests || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createRequest = useCallback(async (requestedRole: string, justification: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/role-requests`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ requested_role: requestedRole, justification }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchRequests();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchRequests]);

  const approve = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/role-requests/${id}/approve`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchRequests();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchRequests]);

  const reject = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/role-requests/${id}/reject`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchRequests();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchRequests]);

  return { requests, loading, error, fetchRequests, createRequest, approve, reject };
}
