"use client";

import { useState, useEffect, useCallback } from "react";
import { Gauge, TrendingUp, Users, Building2 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface HighRiskUser {
  user_id: string;
  username: string;
  score: number;
  org: string;
  factors: string[];
}

interface RiskData {
  avg_score: number;
  high_risk_count: number;
  trends_7d: number[];
  high_risk_users: HighRiskUser[];
}

export default function RiskAggregatePage() {
  const t = useTranslations();

  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);
  const [view, setView] = useState<"user" | "org">("user");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/risk-aggregate?view=${view}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [view]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const scoreColor = data ? (data.avg_score >= 70 ? "#ef4444" : data.avg_score >= 40 ? "#f59e0b" : "#10b981") : "#3b82f6";
  const maxTrend = Math.max(...(data?.trends_7d || [1]), 1);
  const points = data?.trends_7d.map((v: any, i: number) => `${(i / (data.trends_7d.length - 1 || 1)) * 200},${50 - (v / maxTrend) * 45}`).join(" ") || "";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Gauge className="w-6 h-6 text-red-500" /> {t("riskAggregate.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Aggregate risk scoring across users and organizations.</p>
      </div>

      <div className="flex items-center gap-2">
        <button onClick={() => setView("user")} className={`px-3 py-1.5 rounded-lg text-sm font-medium flex items-center gap-1 ${view === "user" ? "bg-blue-600 text-white" : "border dark:border-gray-700"}`}><Users className="w-4 h-4" /> By User</button>
        <button onClick={() => setView("org")} className={`px-3 py-1.5 rounded-lg text-sm font-medium flex items-center gap-1 ${view === "org" ? "bg-blue-600 text-white" : "border dark:border-gray-700"}`}><Building2 className="w-4 h-4" /> By Org</button>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
              <div className="relative w-20 h-20"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${(data.avg_score / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold" style={{ color: scoreColor }}>{data.avg_score.toFixed(0)}</span></div></div>
              <div><span className="text-sm text-gray-500">Avg Risk Score</span><p className="text-xs text-gray-400 mt-1">across all {view === "user" ? "users" : "orgs"}</p></div>
            </div>
            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><TrendingUp className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">High Risk</span><p className="text-xl font-bold text-red-600">{data.high_risk_count}</p></div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between mb-2"><span className="text-sm text-gray-500">7-Day Trend</span></div><svg viewBox="0 0 200 50" className="w-full h-12"><polyline fill="none" stroke={scoreColor} strokeWidth={2} points={points} /></svg></div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{view === "user" ? "User" : "Organization"}</th><th className="px-4 py-3 text-left font-medium">Score</th><th className="px-4 py-3 text-left font-medium">Org</th><th className="px-4 py-3 text-left font-medium">Risk Factors</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.high_risk_users.map((u: any) => (<tr key={u.user_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{u.username}</span><p className="text-xs text-gray-400 font-mono">{u.user_id}</p></td><td className="px-4 py-3"><span className={`font-bold ${u.score >= 70 ? "text-red-600" : u.score >= 40 ? "text-orange-600" : "text-yellow-600"}`}>{u.score.toFixed(1)}</span></td><td className="px-4 py-3 text-xs text-gray-500">{u.org}</td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{u.factors.map((f: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{f}</span>)}</div></td></tr>))}{data.high_risk_users.length === 0 && <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500">No high-risk entries.</td></tr>}</tbody>
            </table>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
