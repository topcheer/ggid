import { useState, useCallback } from "react";
export interface RetentionRule { event_type: string; retention_days: number; action: "archive" | "delete" | "anonymize"; compliance_basis: string; }
export function useAuditRetentionPolicy(baseUrl: string = "") {
  const [data, setData] = useState<{ rules: RetentionRule[]; storage_used_gb: number; storage_limit_gb: number; per_tenant: boolean; purge_schedule: string } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchPolicy = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/retention-policy"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const savePolicy = useCallback(async (d: any) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/retention-policy", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(d) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchPolicy, savePolicy };
}
