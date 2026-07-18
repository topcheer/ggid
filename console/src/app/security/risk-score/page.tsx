"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldAlert, Loader2, AlertCircle, X, RefreshCw, Activity, Users,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RiskFactor {
  name: string;
  weight: number;
  value: number;
  description: string;
}

interface UserRiskScore {
  user_id: string;
  username: string;
  email: string;
  score: number;
  level: "low" | "medium" | "high" | "critical";
  factors: RiskFactor[];
  last_updated: string;
}

interface RiskSummary {
  total_users: number;
  average_score: number;
  high_risk_count: number;
  critical_count: number;
  top_factors: RiskFactor[];
  distribution: { level: string; count: number }[];
}

const levelColors: Record<string, string> = {
  low: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

function RiskGauge({ score }: { score: number }) {
  const clamped = Math.min(100, Math.max(0, score));
  const angle = (clamped / 100) * 180 - 90;
  const color = clamped >= 75 ? "#dc2626" : clamped >= 50 ? "#f97316" : clamped >= 25 ? "#eab308" : "#16a34a";
  return (
    <div className="relative flex flex-col items-center">
      <svg width="180" height="100" viewBox="0 0 180 100">
        <path d="M 10 90 A 80 80 0 0 1 170 90" fill="none" stroke="#e5e7eb" strokeWidth="12" strokeLinecap="round" />
        <path d="M 10 90 A 80 80 0 0 1 170 90" fill="none" stroke={color} strokeWidth="12" strokeLinecap="round" strokeDasharray={`${(clamped / 100) * 251} 251`} />
        <line x1="90" y1="90" x2={90 + 70 * Math.cos((angle - 90) * Math.PI / 180)} y2={90 + 70 * Math.sin((angle - 90) * Math.PI / 180)} stroke="#374151" strokeWidth="2" strokeLinecap="round" />
        <circle cx="90" cy="90" r="5" fill="#374151" />
      </svg>
      <div className="-mt-6 text-3xl font-bold" style={{ color }}>{clamped}</div>
      <div className="text-xs uppercase text-gray-400">Risk Score</div>
    </div>
  );
}

export default function RiskScorePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [summary, setSummary] = useState<RiskSummary | null>(null);
  const [users, setUsers] = useState<UserRiskScore[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [recalculating, setRecalculating] = useState<string | null>(null);
  const [selectedUser, setSelectedUser] = useState<UserRiskScore | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const [s, u] = await Promise.all([
          apiFetch<RiskSummary>("/api/v1/policy/risk-score/summary").catch(() => null),
          apiFetch<UserRiskScore[]>("/api/v1/policy/risk-score/users").catch(() => []),
        ]);
        setSummary(s); setUsers(u);
      } catch { setError("Failed to load risk score data"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleRecalculate = async (userId: string) => {
    setRecalculating(userId);
    try {
      await apiFetch("/api/v1/policy/risk-score/recalculate", { method: "POST", body: JSON.stringify({ user_id: userId }) });
      const updated = await apiFetch<UserRiskScore[]>("/api/v1/policy/risk-score/users").catch(() => []);
      setUsers(updated);
    } catch { setError("Recalculate failed"); }
    finally { setRecalculating(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const highRiskUsers = users.filter((u: any) => u.level === "high" || u.level === "critical");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldAlert className="h-6 w-6 text-orange-600" /> {t("securityRiskScore.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Continuous user risk assessment based on behavioral and contextual signals.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : (
        <>
          {/* Summary cards */}
          {summary && (
            <div className="grid grid-cols-4 gap-4">
              <div className={cardCls}><div className="flex items-center gap-2"><Users className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Users</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{summary.total_users}</p></div>
              <div className={cardCls}><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Average Score</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{(summary.average_score ?? 0).toFixed(1)}</p></div>
              <div className={cardCls}><div className="flex items-center gap-2"><ShieldAlert className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">High Risk</span></div><p className="mt-2 text-2xl font-bold text-orange-600">{summary.high_risk_count}</p></div>
              <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Critical</span></div><p className="mt-2 text-2xl font-bold text-red-600">{summary.critical_count}</p></div>
            </div>
          )}

          {/* Top factors */}
          {summary && (summary.top_factors?.length ?? 0) > 0 && (
            <div className={cardCls}>
              <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Top Risk Factors</h3>
              <div className="space-y-3">
                {summary.top_factors.map((f: any) => (
                  <div key={f.name}>
                    <div className="flex items-center justify-between text-sm"><span className="text-gray-600 dark:text-gray-300">{f.name}</span><span className="font-bold text-gray-800 dark:text-gray-200">{(f.weight * 100).toFixed(0)}% weight</span></div>
                    <div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-orange-400" style={{ width: `${Math.min(100, f.value * 100)}%` }} /></div>
                    <p className="mt-0.5 text-xs text-gray-400">{f.description}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Distribution */}
          {summary && (summary.distribution?.length ?? 0) > 0 && (
            <div className="grid grid-cols-4 gap-3">
              {summary.distribution.map((d: any) => (
                <div key={d.level} className={`${cardCls} text-center`}>
                  <div className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ${levelColors[d.level] || ""}`}>{d.level}</div>
                  <p className="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{d.count}</p>
                </div>
              ))}
            </div>
          )}

          {/* High-risk users table */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">High-Risk Users</h2>
            {highRiskUsers.length === 0 ? (
              <div className={cardCls}><div className="py-12 text-center"><ShieldAlert className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">No high-risk users detected.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">User</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Score</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Level</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Factors</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Updated</th>
                    <th scope="col" className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {highRiskUsers.map((u: any) => (
                      <tr key={u.user_id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{u.username}</div><div className="text-xs text-gray-400">{u.email}</div></td>
                        <td className="px-4 py-3"><span className={`text-lg font-bold ${u.score >= 75 ? "text-red-600" : "text-orange-600"}`}>{u.score}</span></td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${levelColors[u.level] || ""}`}>{u.level}</span></td>
                        <td className="px-4 py-3 text-gray-500">{u.factors.length} active</td>
                        <td className="px-4 py-3 text-gray-400">{new Date(u.last_updated).toLocaleDateString()}</td>
                        <td className="px-4 py-3 text-right">
                          <button onClick={() => setSelectedUser(u)} className="mr-2 text-xs text-indigo-600 hover:underline">Details</button>
                          <button onClick={() => handleRecalculate(u.user_id)} disabled={recalculating === u.user_id} className="text-xs text-gray-500 hover:text-indigo-600">{recalculating === u.user_id ? <Loader2 className="inline h-3 w-3 animate-spin" /> : <RefreshCw className="inline h-3 w-3" />}</button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* User detail modal */}
          {selectedUser && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setSelectedUser(null)}>
              <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
                <div className="mb-4 flex items-center justify-between">
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">{selectedUser.username}</h3>
                  <button onClick={() => setSelectedUser(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
                </div>
                <div className="mb-4 flex items-center justify-center"><RiskGauge score={selectedUser.score} /></div>
                <div className="space-y-3">
                  {selectedUser.factors.map((f: any) => (
                    <div key={f.name} className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                      <div className="flex items-center justify-between"><span className="text-sm font-medium text-gray-700 dark:text-gray-300">{f.name}</span><span className="text-sm font-bold text-gray-600">{(f.weight * 100).toFixed(0)}%</span></div>
                      <p className="mt-1 text-xs text-gray-400">{f.description}</p>
                      <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-orange-400" style={{ width: `${Math.min(100, f.value * 100)}%` }} /></div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
