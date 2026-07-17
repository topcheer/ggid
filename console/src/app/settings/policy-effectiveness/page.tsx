"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  BarChart3, Loader2, AlertCircle, X, TrendingUp, CheckCircle, XCircle,
} from "lucide-react";

interface PolicyStat {
  policy_id: string;
  policy_name: string;
  trigger_count: number;
  allow_count: number;
  deny_count: number;
  allow_rate: number;
  avg_eval_time_ms: number;
  last_triggered: string;
  top_rules: { rule: string; count: number; effect: "allow" | "deny" }[];
}

export default function PolicyEffectivenessPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [stats, setStats] = useState<PolicyStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setStats(await apiFetch<PolicyStat[]>("/api/v1/policy/effectiveness").catch(() => [])); }
      catch { setError("Failed to load effectiveness data"); }
      finally { setLoading(false); }
    })();
  }, []);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const totalTriggers = stats.reduce((s, p) => s + p.trigger_count, 0);
  const totalAllows = stats.reduce((s, p) => s + p.allow_count, 0);
  const totalDenies = stats.reduce((s, p) => s + p.deny_count, 0);
  const avgEvalTime = stats.length > 0 ? stats.reduce((s, p) => s + p.avg_eval_time_ms, 0) / stats.length : 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><BarChart3 className="h-6 w-6 text-indigo-600" />{t("policyEffectiveness.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Policy trigger counts, allow/deny ratios, and top matching rules.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : (
        <>
          {/* Overview stats */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><TrendingUp className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Triggers</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{totalTriggers.toLocaleString()}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><CheckCircle className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Allows</span></div><p className="mt-2 text-2xl font-bold text-green-600">{totalAllows.toLocaleString()}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><XCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Denies</span></div><p className="mt-2 text-2xl font-bold text-red-600">{totalDenies.toLocaleString()}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Avg Eval Time</div><p className="mt-2 text-2xl font-bold text-blue-600">{avgEvalTime.toFixed(1)}ms</p></div>
          </div>

          {/* Per-policy table */}
          {stats.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><BarChart3 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No policy data available.</p></div></div>
          : (
            <div className="space-y-3">{stats.map((p) => (
              <div key={p.policy_id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex-1"><div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{p.policy_name}</span><span className="font-mono text-xs text-gray-400">{p.policy_id.slice(0, 16)}</span></div>
                    <div className="mt-2 grid grid-cols-4 gap-4 text-sm">
                      <div><span className="text-xs text-gray-400">Triggers</span><div className="font-medium text-gray-900 dark:text-white">{p.trigger_count.toLocaleString()}</div></div>
                      <div><span className="text-xs text-gray-400">Allow Rate</span><div className={`font-medium ${p.allow_rate > 80 ? "text-green-600" : p.allow_rate < 50 ? "text-red-600" : "text-yellow-600"}`}>{p.allow_rate.toFixed(1)}%</div></div>
                      <div><span className="text-xs text-gray-400">Avg Time</span><div className="font-medium text-gray-500">{p.avg_eval_time_ms.toFixed(1)}ms</div></div>
                      <div><span className="text-xs text-gray-400">Last Triggered</span><div className="font-medium text-gray-400 text-xs">{p.last_triggered ? new Date(p.last_triggered).toLocaleDateString() : "—"}</div></div>
                    </div>
                  </div>
                </div>
                {/* Allow/deny bar */}
                <div className="mt-3 flex h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full bg-green-500" style={{ width: `${p.allow_rate}%` }} /><div className="h-full bg-red-500" style={{ width: `${100 - p.allow_rate}%` }} /></div>
                {/* Top rules */}
                {p.top_rules.length > 0 && (
                  <div className="mt-3 border-t border-gray-200 pt-3 dark:border-gray-700"><div className="mb-2 text-xs font-semibold uppercase text-gray-400">Top Rules</div><div className="flex flex-wrap gap-2">{p.top_rules.slice(0, 5).map((r, i) => (<span key={i} className={`rounded px-2 py-1 text-xs ${r.effect === "allow" ? "bg-green-100 text-green-600 dark:bg-green-900/30" : "bg-red-100 text-red-600 dark:bg-red-900/30"}`}>{r.rule} ({r.count})</span>))}</div></div>
                )}
              </div>
            ))}</div>
          )}
        </>
      )}
    </div>
  );
}
