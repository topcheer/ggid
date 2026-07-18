import { useState, useCallback } from "react";

export interface ReviewExemption {
  id: string;
  role: string;
  reason: string;
  exempted_by: string;
  exempted_at: string;
  expires_at: string;
  days_remaining: number;
  status: "active" | "expired" | "revoked";
}

export function useReviewExemptions(baseUrl: string = "") {
  const [exemptions, setExemptions] = useState<ReviewExemption[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchExemptions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/review-exemptions`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setExemptions(data.exemptions || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createExemption = useCallback(async (role: string, reason: string, expiresAt: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/review-exemptions`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role, reason, expires_at: expiresAt }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchExemptions();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchExemptions]);

  const revokeExemption = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/review-exemptions/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setExemptions((prev: any) => prev.filter((e) => e.id !== id));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { exemptions, loading, error, fetchExemptions, createExemption, revokeExemption };
}
