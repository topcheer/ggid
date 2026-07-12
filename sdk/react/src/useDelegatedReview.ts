import { useState, useCallback } from "react";

export interface Delegation {
  id: string;
  original_reviewer: string;
  delegated_to: string;
  scope: string;
  created_at: string;
  expires_at: string;
  status: "active" | "expired" | "revoked";
}

export function useDelegatedReview(baseUrl: string = "") {
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDelegations = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/delegated-review`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setDelegations(data.delegations || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const createDelegation = useCallback(async (originalReviewer: string, delegatedTo: string, scope: string, expiresAt: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/delegated-review`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ original_reviewer: originalReviewer, delegated_to: delegatedTo, scope, expires_at: expiresAt }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const revokeDelegation = useCallback(async (id: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/delegated-review/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { delegations, loading, error, fetchDelegations, createDelegation, revokeDelegation };
}
