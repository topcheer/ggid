"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Lightbulb, Check, X, Sparkles, AlertTriangle, RotateCcw } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface Recommendation { id: string; type: "consolidate" | "split" | "create" | "delete"; affected_policies: string[]; reason: string; risk_reduction_score: number; confidence: number; before_summary: string; after_summary: string; }
const typeColors: Record<string, string> = { consolidate: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400", split: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400", create: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400", delete: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" };
export default function PolicyRecommendationPage() {
  const t = useTranslations();
  const [recs, setRecs] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/policy/recommendations", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json();
      setRecs(d.recommendations || d || []);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load recommendations"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const apply = async (id: string) => {
    try {
      const res = await fetch("/api/v1/policy/recommendations/" + id + "/apply", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setRecs(recs.filter((r: any) => r.id !== id));
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to apply recommendation"); }
  };
  const dismiss = async (id: string) => {
    try {
      const res = await fetch("/api/v1/policy/recommendations/" + id + "/dismiss", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setRecs(recs.filter((r: any) => r.id !== id));
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to dismiss recommendation"); }
  };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Lightbulb className="w-6 h-6 text-yellow-500" />{t("policyRecommendation.title")}</h1><p className="text-sm text-gray-500 mt-1">AI-powered policy optimization recommendations.</p></div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={fetchData} className="text-xs underline hover:text-red-700">Retry</button></div>}
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading recommendations...</div></div>}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {recs.map((r: any) => (
          <div key={r.id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <span className={"px-2 py-1 rounded text-xs font-medium " + typeColors[r.type]}>{r.type}</span>
              <span className="flex items-center gap-1 text-xs text-gray-500"><Sparkles className="w-3 h-3" /> {r.confidence}% confidence</span>
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400">{r.reason}</p>
            <div className="rounded border dark:border-gray-800 p-2 text-xs"><div><span className="text-red-500">Before:</span> {r.before_summary}</div><div className="mt-1"><span className="text-green-500">After:</span> {r.after_summary}</div></div>
            <div className="flex items-center gap-2"><span className="text-xs text-gray-500">Affected: {r.affected_policies.join(", ")}</span></div>
            <div className="flex items-center gap-2">
              <div className="flex-1 flex items-center gap-2">
                <span className="text-xs text-gray-500">Risk Reduction:</span>
                <div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className="h-full bg-green-500 rounded-full" style={{ width: r.risk_reduction_score + "%" }} /></div>
                <span className="text-xs font-bold text-green-600">+{r.risk_reduction_score}</span>
              </div>
            </div>
            <div className="flex gap-2">
              <button onClick={() => apply(r.id)} aria-label={`Apply recommendation ${r.id}`} className="flex-1 px-3 py-1.5 rounded-lg bg-green-600 text-white text-xs font-medium flex items-center justify-center gap-1"><Check className="w-3 h-3" /> Apply</button>
              <button onClick={() => dismiss(r.id)} aria-label={`Dismiss recommendation ${r.id}`} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-xs flex items-center gap-1"><X className="w-3 h-3" /> Dismiss</button>
            </div>
          </div>
        ))}
        {recs.length === 0 && !loading && <div className="col-span-full text-center text-gray-500 py-8">No recommendations.</div>}
      </div>
    </div>
  );
}
