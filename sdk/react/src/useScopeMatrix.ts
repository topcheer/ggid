import { useState, useCallback } from "react";

export interface MatrixData {
  clients: { client_id: string; client_name: string }[];
  scopes: string[];
  grants: Record<string, Record<string, boolean>>;
  usage: Record<string, Record<string, number>>;
}

export function useScopeMatrix(baseUrl: string = "") {
  const [data, setData] = useState<MatrixData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchMatrix = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/scope-matrix`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: MatrixData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateGrants = useCallback(async (grants: Record<string, Record<string, boolean>>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/scope-matrix`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ grants }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      if (data) setData({ ...data, grants });
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, data]);

  return { data, loading, error, fetchMatrix, updateGrants };
}
