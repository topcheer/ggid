import { useState, useCallback } from "react";

export interface ElevationRequest {
  id: string;
  user_id: string;
  username: string;
  role: string;
  duration_minutes: number;
  justification: string;
  status: "pending" | "approved" | "active" | "rejected" | "expired";
  requested_at: string;
  expires_at: string;
  remaining_minutes: number;
}

export function useJITElevation(baseUrl: string = "") {
  const [requests, setRequests] = useState<ElevationRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/jit-elevation`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setRequests(data.requests || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const submitRequest = useCallback(async (role: string, durationMinutes: number, justification: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/jit-elevation`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role, duration_minutes: durationMinutes, justification }),
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
      const res = await fetch(`${baseUrl}/api/v1/policy/jit-elevation/${id}/approve`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setRequests((prev: any) => prev.map((r: any) => r.id === id ? { ...r, status: "active" } : r));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const reject = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/jit-elevation/${id}/reject`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setRequests((prev: any) => prev.map((r: any) => r.id === id ? { ...r, status: "rejected" } : r));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { requests, loading, error, fetchRequests, submitRequest, approve, reject };
}
