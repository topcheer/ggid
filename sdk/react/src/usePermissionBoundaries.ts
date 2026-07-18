import { useState, useCallback } from "react";

export interface PermissionBoundary {
  id: string;
  role: string;
  max_scopes: string[];
  denied_actions: string[];
  violation_count: number;
  last_updated: string;
}

export function usePermissionBoundaries(baseUrl: string = "") {
  const [boundaries, setBoundaries] = useState<PermissionBoundary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBoundaries = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/permission-boundaries`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setBoundaries(data.boundaries || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateBoundary = useCallback(async (id: string, boundary: Partial<PermissionBoundary>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/permission-boundaries/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(boundary),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setBoundaries((prev: any) => prev.map((b: any) => b.id === id ? { ...b, ...boundary } : b));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const createBoundary = useCallback(async (role: string, maxScopes: string[], deniedActions: string[]) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/permission-boundaries`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ role, max_scopes: maxScopes, denied_actions: deniedActions }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchBoundaries();
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchBoundaries]);

  const deleteBoundary = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/permission-boundaries/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setBoundaries((prev: any) => prev.filter((b: any) => b.id !== id));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { boundaries, loading, error, fetchBoundaries, updateBoundary, createBoundary, deleteBoundary };
}
