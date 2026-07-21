import { useState, useCallback } from "react";

export interface ImportDiff {
  added: string[];
  removed: string[];
  modified: string[];
}

export function usePolicyExport(baseUrl: string = "") {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const exportPolicies = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/export`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.text();
    } catch (e: any) {
      setError(e.message);
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const previewImport = useCallback(async (json: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/import-preview`, { method: "POST", headers: { "Content-Type": "application/json" }, body: json });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: ImportDiff = await res.json();
      return data;
    } catch (e: any) {
      setError(e.message);
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const executeImport = useCallback(async (json: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/import`, { method: "POST", headers: { "Content-Type": "application/json" }, body: json });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { loading, error, exportPolicies, previewImport, executeImport };
}
