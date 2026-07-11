import { useState, useCallback } from "react";

export interface DSRRequest {
  id: string;
  type: "access" | "erasure" | "portability" | "rectification";
  user_id: string;
  username: string;
  email: string;
  status: "pending" | "in_progress" | "completed" | "overdue";
  created_at: string;
  due_date: string;
  days_remaining: number;
  notes: string;
}

export function useDSR(baseUrl: string = "") {
  const [requests, setRequests] = useState<DSRRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/dsr`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setRequests(data.requests || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createRequest = useCallback(async (type: DSRRequest["type"], userId: string, notes?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/dsr`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type, user_id: userId, notes }),
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

  const updateStatus = useCallback(async (id: string, status: DSRRequest["status"]) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/dsr/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setRequests((prev) => prev.map((r) => r.id === id ? { ...r, status } : r));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { requests, loading, error, fetchRequests, createRequest, updateStatus };
}
