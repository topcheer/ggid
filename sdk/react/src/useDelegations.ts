import { useState, useCallback } from "react";

export interface Delegation {
  id: string;
  delegator: string;
  delegated_to: string;
  scope: string[];
  start_date: string;
  end_date: string;
  status: "active" | "expired" | "revoked" | "pending";
  reason: string;
}

export function useDelegations(baseUrl: string = "") {
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDelegations = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/delegations`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setDelegations(data.delegations || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createDelegation = useCallback(async (delegatedTo: string, scope: string[], startDate: string, endDate: string, reason?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/delegations`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ delegated_to: delegatedTo, scope, start_date: startDate, end_date: endDate, reason }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchDelegations();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchDelegations]);

  const revokeDelegation = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/delegations/${id}/revoke`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setDelegations((prev) => prev.map((d: any) => d.id === id ? { ...d, status: "revoked" } : d));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { delegations, loading, error, fetchDelegations, createDelegation, revokeDelegation };
}
