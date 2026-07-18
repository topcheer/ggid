"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  BarChart3, Loader2, AlertCircle, X, KeyRound, Users, AlertOctagon, Activity,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ClientStat {
  client_id: string;
  client_name: string;
  total_tokens: number;
  active_tokens: number;
  unique_users: number;
  error_rate: number;
  total_requests: number;
  top_scopes: { scope: string; count: number }[];
  last_active: string;
}

export default function ClientAnalyticsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [stats, setStats] = useState<ClientStat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setStats(await apiFetch<ClientStat[]>("/api/v1/oauth/analytics").catch(() => [])); }
      catch { setError("Failed to load analytics"); }
      finally { setLoading(false); }
    })();
  }, []);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const totalTokens = stats.reduce((s, c) => s + c.total_tokens, 0);
  const totalUsers = stats.reduce((s, c) => s + c.unique_users, 0);
  const avgError = stats.length > 0 ? stats.reduce((s, c) => s + c.error_rate, 0) / stats.length : 0;
  const totalReqs = stats.reduce((s, c) => s + c.total_requests, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><BarChart3 className="h-6 w-6 text-indigo-600" /> {t("oauthClientAnalytics.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Usage metrics across all OAuth clients: tokens, users, errors, top scopes.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : (
        <>
          {/* Overview stats */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><KeyRound className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Tokens</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{totalTokens.toLocaleString()}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Users className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Unique Users</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{totalUsers.toLocaleString()}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertOctagon className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Avg Error Rate</span></div><p className="mt-2 text-2xl font-bold text-red-600">{avgError.toFixed(1)}%</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Requests</span></div><p className="mt-2 text-2xl font-bold text-green-600">{totalReqs.toLocaleString()}</p></div>
          </div>

          {/* Client table */}
          {stats.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><BarChart3 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No analytics data available.</p></div></div>
          ) : (
            <div className="space-y-3">
              {stats.map((c: any) => {
                const isExpanded = selected === c.client_id;
                return (
                  <div key={c.client_id} className={cardCls}>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{c.client_name}</span><span className="font-mono text-xs text-gray-400">{c.client_id.slice(0, 16)}</span></div>
                        <div className="mt-2 grid grid-cols-4 gap-4 text-sm">
                          <div><span className="text-xs text-gray-400">Tokens</span><div className="font-medium text-gray-900 dark:text-white">{c.total_tokens.toLocaleString()} <span className="text-xs text-green-500">({c.active_tokens} active)</span></div></div>
                          <div><span className="text-xs text-gray-400">Users</span><div className="font-medium text-gray-900 dark:text-white">{c.unique_users.toLocaleString()}</div></div>
                          <div><span className="text-xs text-gray-400">Error Rate</span><div className={`font-medium ${c.error_rate > 5 ? "text-red-600" : c.error_rate > 1 ? "text-yellow-600" : "text-green-600"}`}>{c.error_rate.toFixed(1)}%</div></div>
                          <div><span className="text-xs text-gray-400">Requests</span><div className="font-medium text-gray-900 dark:text-white">{c.total_requests.toLocaleString()}</div></div>
                        </div>
                      </div>
                      <button onClick={() => setSelected(isExpanded ? null : c.client_id)} className="text-xs text-indigo-600 hover:underline">{isExpanded ? "Hide scopes" : "Top scopes"}</button>
                    </div>
                    {/* Top scopes */}
                    {isExpanded && c.top_scopes.length > 0 && (
                      <div className="mt-3 border-t border-gray-200 pt-3 dark:border-gray-700">
                        <div className="mb-2 text-xs font-semibold uppercase text-gray-400">Top Scopes</div>
                        <div className="space-y-2">
                          {c.top_scopes.map((s: any) => (
                            <div key={s.scope}>
                              <div className="flex items-center justify-between text-xs"><span className="font-mono text-gray-600 dark:text-gray-300">{s.scope}</span><span className="font-medium text-gray-500">{s.count.toLocaleString()}</span></div>
                              <div className="mt-0.5 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-400" style={{ width: `${Math.min(100, (s.count / c.total_tokens) * 100)}%` }} /></div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                    {c.last_active && <div className="mt-2 text-xs text-gray-400">Last active: {new Date(c.last_active).toLocaleString()}</div>}
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}
    </div>
  );
}
