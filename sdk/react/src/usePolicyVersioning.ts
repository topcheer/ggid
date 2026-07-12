import { useState, useCallback } from "react";
export interface PolicyVersion { version_num: number; author: string; timestamp: string; change_summary: string; is_current: boolean; }
export function usePolicyVersioning(baseUrl: string = "") {
  const [versions, setVersions] = useState<PolicyVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchVersions = useCallback(async (policyId: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/" + policyId + "/versions"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setVersions(data.versions || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const activate = useCallback(async (policyId: string, versionNum: number) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/" + policyId + "/versions/" + versionNum + "/activate", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const rollback = useCallback(async (policyId: string, versionNum: number) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/" + policyId + "/versions/" + versionNum + "/rollback", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { versions, loading, error, fetchVersions, activate, rollback };
}
