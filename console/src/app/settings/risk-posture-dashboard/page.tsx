"use client";
import { useState, useEffect, useCallback } from "react";
import { Shield, TrendingDown, TrendingUp } from "lucide-react";
interface CategoryScore { category: string; score: number; max: number; }
interface Finding { id: string; finding: string; severity: "low" | "medium" | "high" | "critical"; age_days: number; owner: string; status: "open" | "remediated"; }
interface RiskData { overall_score: number; categories: CategoryScore[]; trending_risks: { name: string; trend: "up" | "down" }[]; mitigated_count: number; open_findings: Finding[]; trend_30d: { date: string; score: number }[]; }
const sevColors: Record<string, string> = { low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400", medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400", high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400", critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" };
export default function RiskPostureDashboardPage() {
  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/audit/risk-posture", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const scoreColor = data ? (data.overall_score >= 75 ? "#10b981" : data.overall_score >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  const maxTrend = Math.max(...(data?.trend_30d.map((d) => d.score) || [100]), 1);
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-red-500" /> Risk Posture</h1><p className="text-sm text-gray-500 mt-1">Overall security risk posture and findings.</p></div>
      {data && (<>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4"><div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={(data.overall_score / 100) * 176 + " 176"} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.overall_score}</span><span className="text-[10px] text-gray-400">score</span></div></div><div><span className="text-sm text-gray-500">Overall Risk Score</span><p className="text-xs text-gray-400 mt-1">{data.mitigated_count} mitigated</p></div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4 md:col-span-2"><h3 className="text-sm font-semibold mb-3">Risk by Category</h3><div className="space-y-2">{data.categories.map((c) => (<div key={c.category} className="flex items-center gap-2"><span className="text-xs capitalize w-20">{c.category}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-5 overflow-hidden"><div className="h-full rounded-full" style={{ width: (c.score / c.max) * 100 + "%", background: c.score / c.max >= 0.75 ? "#10b981" : c.score / c.max >= 0.5 ? "#f59e0b" : "#ef4444" }} /></div><span className="text-xs font-bold w-12 text-right">{c.score}/{c.max}</span></div>))}</div></div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Trending Risks</h3><div className="space-y-1">{data.trending_risks.map((r, i) => (<div key={i} className="flex items-center gap-2 text-sm">{r.trend === "up" ? <TrendingUp className="w-4 h-4 text-red-500" /> : <TrendingDown className="w-4 h-4 text-green-500" />}<span>{r.name}</span><span className={"text-xs ml-auto " + (r.trend === "up" ? "text-red-600" : "text-green-600")}>{r.trend}</span></div>))}</div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Risk Trend (30d)</h3><div className="flex items-end gap-1 h-20">{data.trend_30d.map((d, i) => (<div key={i} className="flex-1 bg-blue-400 dark:bg-blue-500 rounded-t" style={{ height: (d.score / maxTrend) * 100 + "%", minHeight: "2px" }} title={d.date + ": " + d.score} />))}</div></div>
        </div>
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Finding</th><th className="px-4 py-3 text-left font-medium">Severity</th><th className="px-4 py-3 text-left font-medium">Age</th><th className="px-4 py-3 text-left font-medium">Owner</th><th className="px-4 py-3 text-left font-medium">Status</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.open_findings.map((f) => (<tr key={f.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3">{f.finding}</td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + sevColors[f.severity]}>{f.severity}</span></td><td className="px-4 py-3 text-xs">{f.age_days}d</td><td className="px-4 py-3 text-xs font-mono">{f.owner}</td><td className="px-4 py-3"><span className={"text-xs " + (f.status === "open" ? "text-red-600" : "text-green-600")}>{f.status}</span></td></tr>))}</tbody></table></div>
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
