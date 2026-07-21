import { useState, useCallback } from "react";
export interface MatrixData { roles: string[]; permissions: string[]; assignments: Record<string, Record<string, string>>; }
export function useRolePermissionMatrix(baseUrl: string = "") {
  const [data, setData] = useState<MatrixData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchMatrix = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/role-permission-matrix"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchMatrix };
}
