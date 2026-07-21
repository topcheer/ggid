import { useState, useEffect, useCallback } from "react";

export interface PolicySnapshot {
  id: string;
  policy_id: string;
  version: number;
  description: string;
  created_at: string;
  created_by: string;
}

export function usePolicySnapshots(baseUrl: string = "") {
  const [snapshots, setSnapshots] = useState<PolicySnapshot[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSnapshots = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/snapshots`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setSnapshots(data.snapshots || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createSnapshot = useCallback(async (policyId: string, description?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/snapshots`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ policy_id: policyId, description }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchSnapshots();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchSnapshots]);

  const rollback = useCallback(async (snapshotId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/snapshots/${snapshotId}/rollback`, {
        method: "POST",
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchSnapshots();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchSnapshots]);

  useEffect(() => {
    fetchSnapshots();
  }, [fetchSnapshots]);

  return { snapshots, loading, error, fetchSnapshots, createSnapshot, rollback };
}
