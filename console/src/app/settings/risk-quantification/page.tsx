"use client";
import { useState, useEffect, useCallback } from "react";
import { Gauge, Download, TrendingUp } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface RiskFactor { factor: string; weight: number; current_value: number; contribution: number; }
interface RiskData { composite_score: number; factors: RiskFactor[]; monte_carlo: { p50: number; p90: number; p99: number }; trend_30d: { date: string; score: number }[]; }

export default function RiskQuantificationPage() {
  const t = useTranslations();

  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/risk-quantification", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const scoreColor = data ? (data.composite_score >= 70 ? "#ef4444" : data.composite_score >= 40 ? "#f59e0b" : "#10b981") : "#3b82f6";
  const maxTrend = data ? Math.max(...data.trend_30d.map((t) => t.score), 1) : 1;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Gauge className="w-6 h-6 text-purple-500" /> {t("riskQuantification.title")}</h1><p className="text-sm text-gray-500 mt-1">Quantify risk with weighted factors and Monte Carlo simulation.</p></div>
        <button className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><Download className="w-4 h-4" /> Export</button>
      </div>

      {data && (<>
        <div className="flex items-center gap-6"><div className="relative w-28 h-28"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={(data.composite_score / 100) * 176 + " 176"} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.composite_score}</span><span className="text-[10px] text-gray-400">composite</span></div></div><div className="space-y-2"><div className="flex items-center gap-2 text-sm"><span className="text-gray-500">Monte Carlo P50:</span><span className="font-bold">{data.monte_carlo.p50}</span></div><div className="flex items-center gap-2 text-sm"><span className="text-gray-500">P90:</span><span className="font-bold text-yellow-600">{data.monte_carlo.p90}</span></div><div className="flex items-center gap-2 text-sm"><span className="text-gray-500">P99:</span><span className="font-bold text-red-600">{data.monte_carlo.p99}</span></div></div></div>

        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Risk Factors</h3><div className="space-y-2">{data.factors.map((f) => (<div key={f.factor} className="flex items-center gap-3"><span className="text-sm w-40 truncate">{f.factor}</span><div className="flex-1"><div className="flex items-center gap-2"><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-5"><div className="h-full rounded-full" style={{ width: (f.contribution / 100) * 100 + "%", background: f.contribution > 50 ? "#ef4444" : f.contribution > 25 ? "#f59e0b" : "#10b981" }} /></div><span className="text-xs font-bold w-12 text-right">{f.contribution.toFixed(0)}%</span></div></div></div>))}</div></div>

        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><TrendingUp className="w-4 h-4 text-gray-400" /> 30-Day Trend</h3><div className="flex items-end gap-1 h-24">{data.trend_30d.map((t: any, i: number) => (<div key={i} className="flex-1 rounded-t" style={{ height: (t.score / maxTrend) * 100 + "%", background: t.score >= 70 ? "#ef4444" : t.score >= 40 ? "#f59e0b" : "#10b981", minHeight: "2px" }} title={t.date + ": " + t.score} />))}</div></div>
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
