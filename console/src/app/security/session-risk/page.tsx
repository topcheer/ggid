"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, Activity, TrendingUp, TrendingDown,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RiskFactor {
  type: string;
  detail: string;
  detected: boolean;
}

interface SessionRiskEntry {
  session_id: string;
  user_id: string;
  username: string;
  current_risk: number;
  previous_risk: number;
  risk_delta: number;
  factors: RiskFactor[];
  ip_address: string;
  device_id: string;
  location: string;
  last_evaluated: string;
  reevaluate_recommended: boolean;
}

function riskColor(score: number): string {
  const t = useTranslations();

  if (score >= 75) return "text-red-600";
  if (score >= 50) return "text-orange-600";
  if (score >= 25) return "text-yellow-600";
  return "text-green-600";
}

export default function SessionRiskPage() {
  const t = useTranslations();  const { apiFetch } = useApi();
  const [sessions, setSessions] = useState<SessionRiskEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [evaluating, setEvaluating] = useState<string | null>(null);
  const [evaluatingAll, setEvaluatingAll] = useState(false);

  useEffect(() => {
    (async () => {
      try { setSessions(await apiFetch<SessionRiskEntry[]>("/api/v1/auth/sessions/risk").catch(() => [])); }
      catch { setError("Failed to load session risk data"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleReevaluate = async (sessionId: string) => {
    setEvaluating(sessionId);
    try {
      const updated = await apiFetch<SessionRiskEntry>(`/api/v1/auth/sessions/${sessionId}/reevaluate`, { method: "POST" });
      setSessions((p) => p.map((s: any) => s.session_id === sessionId ? updated : s));
    } catch { setError("Re-evaluate failed"); }
    finally { setEvaluating(null); }
  };

  const handleReevaluateAll = async () => {
    setEvaluatingAll(true);
    try {
      await apiFetch("/api/v1/auth/sessions/reevaluate-all", { method: "POST" });
      setSessions(await apiFetch<SessionRiskEntry[]>("/api/v1/auth/sessions/risk").catch(() => sessions));
    } catch { setError("Batch re-evaluate failed"); }
    finally { setEvaluatingAll(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const flagged = sessions.filter((s: any) => s.reevaluate_recommended);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-orange-600" /> {t("securitySessionRisk.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Continuous risk assessment of active sessions based on contextual signals.</p>
        </div>
        <button onClick={handleReevaluateAll} disabled={evaluatingAll} className="flex items-center gap-2 rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700 disabled:opacity-50">{evaluatingAll ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />} Re-evaluate All</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-orange-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active Sessions</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{sessions.length}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Flagged</span></div><p className="mt-2 text-2xl font-bold text-red-600">{flagged.length}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><TrendingUp className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">Avg Risk</span></div><p className="mt-2 text-2xl font-bold text-orange-600">{sessions.length > 0 ? Math.round(sessions.reduce((s: any, e: any) => s + e.current_risk, 0) / sessions.length) : 0}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><TrendingDown className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Low Risk</span></div><p className="mt-2 text-2xl font-bold text-green-600">{sessions.filter((s: any) => s.current_risk < 25).length}</p></div>
          </div>

          {/* Sessions table */}
          {sessions.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><ShieldCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No active sessions.</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">User</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Current</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Previous</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Change</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Factors</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Location / IP</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Last Evaluated</th>
                  <th scope="col" className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {sessions.map((s: any) => (
                    <tr key={s.session_id} className={`bg-white dark:bg-gray-900 ${s.reevaluate_recommended ? "border-l-4 border-l-orange-400" : ""}`}>
                      <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{s.username}</div><div className="text-xs text-gray-400 font-mono">{s.session_id.slice(0, 12)}</div></td>
                      <td className="px-4 py-3"><span className={`text-lg font-bold ${riskColor(s.current_risk)}`}>{s.current_risk}</span></td>
                      <td className="px-4 py-3 text-gray-500">{s.previous_risk}</td>
                      <td className="px-4 py-3">
                        {s.risk_delta !== 0 && (
                          <span className={`inline-flex items-center gap-0.5 text-xs font-medium ${s.risk_delta > 0 ? "text-red-600" : "text-green-600"}`}>
                            {s.risk_delta > 0 ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
                            {s.risk_delta > 0 ? "+" : ""}{s.risk_delta}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{s.factors.filter((f: any) => f.detected).map((f: any) => <span key={f.type} className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-600 dark:bg-red-900/30">{f.type.replace(/_/g, " ")}</span>)}{s.factors.filter((f: any) => f.detected).length === 0 && <span className="text-xs text-green-500">clean</span>}</div></td>
                      <td className="px-4 py-3"><div className="text-xs text-gray-500">{s.location}</div><div className="text-xs text-gray-400 font-mono">{s.ip_address}</div></td>
                      <td className="px-4 py-3 text-xs text-gray-400">{s.last_evaluated ? new Date(s.last_evaluated).toLocaleTimeString() : "—"}</td>
                      <td className="px-4 py-3 text-right"><button onClick={() => handleReevaluate(s.session_id)} disabled={evaluating === s.session_id} className="flex items-center gap-1 text-xs text-orange-600 hover:underline">{evaluating === s.session_id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />} Re-eval</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
