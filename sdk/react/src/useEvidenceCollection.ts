import { useState, useCallback } from "react";
export interface Control { control_id: string; description: string; evidence_required: boolean; evidence_type: string; collection_status: "collected" | "pending" | "overdue"; last_collected: string | null; reviewer: string | null; }
export interface FrameworkData { framework: string; controls: Control[]; }
export function useEvidenceCollection(baseUrl: string = "") {
  const [data, setData] = useState<FrameworkData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchEvidence = useCallback(async (framework: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/evidence-collection?framework=" + encodeURIComponent(framework)); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const uploadEvidence = useCallback(async (controlId: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/evidence-collection/" + controlId + "/upload", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchEvidence, uploadEvidence };
}
