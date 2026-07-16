"use client";
import { useState, useCallback } from "react";
import { Search, Shield, TrendingUp, TrendingDown, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface RiskData { user_id: string; username: string; risk_score: number; trend: number; factors: { key: string; label: string; score: number; max: number }[]; }
export default function RiskProfilePage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/auth/risk-profile?user=${encodeURIComponent(user)}`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setData(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load risk profile"); }
    finally { setLoading(false); }
  }, []);
  const scoreColor = data ? (data.risk_score >= 70 ? "#ef4444" : data.risk_score >= 40 ? "#f59e0b" : "#10b981") : "#3b82f6";
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-blue-500" /> {t("riskProfile.title")}</h1><p className="text-sm text-gray-500 mt-1">User risk assessment across 5 security factors.</p></div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => { setError(null); fetchData(search); }} aria-label="Retry loading risk profile" className="text-xs underline hover:text-red-700">Retry</button></div>}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input type="text" placeholder="Search by username..." value={search} onChange={(e) => setSearch(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") fetchData(search); }} aria-label="Search by username" className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading risk profile...</div></div>}
      {data && (
        <div className="space-y-4">
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-6">
            <div className="relative w-24 h-24">
              <svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${data.risk_score * 1.76} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.risk_score}</span><span className="text-[10px] text-gray-400">/100</span></div>
            </div>
            <div>
              <h3 className="font-semibold">{data.username}</h3>
              <div className="flex items-center gap-1 mt-1 text-sm">Risk trend: {data.trend > 0 ? <span className="flex items-center gap-1 text-red-600"><TrendingUp className="w-4 h-4" /> +{data.trend} this week</span> : data.trend < 0 ? <span className="flex items-center gap-1 text-green-600"><TrendingDown className="w-4 h-4" /> {data.trend} this week</span> : <span className="text-gray-400">No change</span>}</div>
            </div>
          </div>
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3">Risk Factor Breakdown</h3>
            <div className="space-y-3">
              {data.factors.map((f) => (
                <div key={f.key}>
                  <div className="flex items-center justify-between text-sm mb-1"><span>{f.label}</span><span className="text-xs text-gray-400">{f.score}/{f.max}</span></div>
                  <div className="w-full h-3 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden"><div className="h-full rounded-full" style={{ width: `${(f.score / f.max) * 100}%`, backgroundColor: f.score / f.max >= 0.7 ? "#ef4444" : f.score / f.max >= 0.4 ? "#f59e0b" : "#10b981" }} /></div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
      {!data && !loading && search && <p className="text-sm text-gray-500">No risk data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to view their risk profile.</p>}
    </div>
  );
}
