import { useState, useCallback } from "react";
export interface Recommendation { id: string; type: "consolidate" | "split" | "create" | "delete"; affected_policies: string[]; reason: string; risk_reduction_score: number; confidence: number; before_summary: string; after_summary: string; }
export function usePolicyRecommendation(baseUrl: string = "") {
  const [recs, setRecs] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchRecs = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/recommendations"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setRecs(data.recommendations || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const apply = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/recommendations/" + id + "/apply", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  const dismiss = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/recommendations/" + id + "/dismiss", { method: "POST" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { recs, loading, error, fetchRecs, apply, dismiss };
}
