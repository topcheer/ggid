"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, Activity, TrendingUp, Trash2, X, AlertCircle, Loader2,
  RefreshCw, Clock, Database, ZapOff,
} from "lucide-react";

interface CacheStats {
  total_entries: number;
  hit_count: number;
  miss_count: number;
  hit_ratio: number;
  avg_response_time_ms: number;
  cache_size_mb: number;
  max_size_mb: number;
  entries: CacheEntry[];
}

interface CacheEntry {
  token_hash: string;
  client_id: string;
  scope: string;
  active: boolean;
  created_at: string;
  expires_at: string;
  last_accessed: string;
}

export default function IntrospectionPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [stats, setStats] = useState<CacheStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [invalidating, setInvalidating] = useState(false);
  const [confirmInvalidate, setConfirmInvalidate] = useState(false);
  const [invalidated, setInvalidated] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<CacheStats>("/api/v1/oauth/introspection/cache/stats").catch(() => null);
      if (data) setStats(data);
    } catch {
      setError("Failed to load introspection cache stats");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); const i = setInterval(load, 30000); return () => clearInterval(i); }, [load]);

  const handleInvalidate = async () => {
    setInvalidating(true);
    try {
      await apiFetch("/api/v1/oauth/introspection/cache/invalidate", { method: "POST" });
      setConfirmInvalidate(false);
      setInvalidated(true);
      setTimeout(() => setInvalidated(false), 3000);
      await load();
    } catch {
      setError("Failed to invalidate cache");
    } finally {
      setInvalidating(false);
    }
  };

  const handleInvalidateEntry = async (tokenHash: string) => {
    try {
      await apiFetch(`/api/v1/oauth/introspection/cache/${tokenHash}`, { method: "DELETE" });
      await load();
    } catch {
      setError("Failed to invalidate entry");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const utilization = stats ? (stats.cache_size_mb / stats.max_size_mb) * 100 : 0;

  if (loading && !stats) {
    return <div className="flex justify-center py-24"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <KeyRound className="h-6 w-6 text-indigo-600" /> {t("backend.introspection.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Monitor and manage the introspection cache for OAuth token validation.
          </p>
        </div>
        <button
          onClick={() => setConfirmInvalidate(true)}
          className="flex items-center gap-2 rounded-lg border border-red-300 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-900/20"
        >
          <ZapOff className="h-4 w-4" /> Invalidate All
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}
      {invalidated && (
        <div className="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-400">
          <RefreshCw className="h-4 w-4" /> Cache invalidated successfully. All tokens will be re-validated on next request.
        </div>
      )}

      {/* Stats cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className={cardCls}>
          <div className="flex items-center gap-2">
            <TrendingUp className="h-4 w-4 text-green-500" />
            <h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.hitRatio")}</h3>
          </div>
          <p className={`mt-2 text-3xl font-bold ${(stats?.hit_ratio ?? 0) >= 0.8 ? "text-green-600" : (stats?.hit_ratio ?? 0) >= 0.5 ? "text-yellow-600" : "text-red-600"}`}>
            {stats ? `${(stats.hit_ratio * 100).toFixed(1)}%` : "—"}
          </p>
          <div className="mt-1 flex gap-3 text-xs text-gray-400">
            <span>{stats?.hit_count ?? 0} hits</span>
            <span>{stats?.miss_count ?? 0} misses</span>
          </div>
        </div>

        <div className={cardCls}>
          <div className="flex items-center gap-2">
            <Database className="h-4 w-4 text-indigo-500" />
            <h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.cacheSize")}</h3>
          </div>
          <p className="mt-2 text-3xl font-bold text-indigo-600">{stats?.cache_size_mb.toFixed(1) ?? "—"} MB</p>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
            <div className="h-full rounded-full bg-indigo-500" style={{ width: `${utilization}%` }} />
          </div>
          <p className="mt-1 text-xs text-gray-400">{utilization.toFixed(0)}% of {stats?.max_size_mb ?? 0} MB</p>
        </div>

        <div className={cardCls}>
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-blue-500" />
            <h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.avgResponse")}</h3>
          </div>
          <p className="mt-2 text-3xl font-bold text-blue-600">{stats?.avg_response_time_ms.toFixed(1) ?? "—"} ms</p>
          <p className="mt-1 text-xs text-gray-400">Per introspection call</p>
        </div>

        <div className={cardCls}>
          <div className="flex items-center gap-2">
            <KeyRound className="h-4 w-4 text-purple-500" />
            <h3 className="text-xs font-semibold uppercase text-gray-500">{t("backend.introspection.entries")}</h3>
          </div>
          <p className="mt-2 text-3xl font-bold text-purple-600">{stats?.total_entries ?? 0}</p>
          <p className="mt-1 text-xs text-gray-400">{t("backend.introspection.activeCached")}</p>
        </div>
      </div>

      {/* Cache entries table */}
      <div>
        <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">{t("backend.introspection.cachedTokens")}</h2>
        {!stats?.entries || stats.entries.length === 0 ? (
          <div className={cardCls}>
            <div className="py-12 text-center">
              <Database className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-4 text-sm text-gray-400">No cached tokens. Cache is empty.</p>
            </div>
          </div>
        ) : (
          <>
            {/* Desktop table */}
            <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800">
                  <tr className="text-left text-xs font-semibold uppercase text-gray-500">
                    <th className="px-4 py-3">Token Hash</th>
                    <th className="px-4 py-3">{t("backend.introspection.clientId")}</th>
                    <th className="px-4 py-3">Scope</th>
                    <th className="px-4 py-3">Last Accessed</th>
                    <th className="px-4 py-3">{t("backend.introspection.expires")}</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3 text-right">{t("backend.introspection.action")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                  {stats.entries.map((e) => (
                    <tr key={e.token_hash} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                      <td className="px-4 py-3 font-mono text-xs text-gray-500">{e.token_hash.substring(0, 20)}...</td>
                      <td className="px-4 py-3 font-mono text-xs text-gray-600 dark:text-gray-300">{e.client_id.substring(0, 16)}...</td>
                      <td className="px-4 py-3">
                        <span className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{e.scope}</span>
                      </td>
                      <td className="px-4 py-3 text-gray-500">
                        <span className="flex items-center gap-1"><Clock className="h-3 w-3" />{new Date(e.last_accessed).toLocaleTimeString()}</span>
                      </td>
                      <td className="px-4 py-3 text-gray-500">{new Date(e.expires_at).toLocaleString()}</td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${e.active ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>
                          {e.active ? "Active" : "Expired"}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button onClick={() => handleInvalidateEntry(e.token_hash)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Mobile cards */}
            <div className="space-y-3 md:hidden">
              {stats.entries.map((e) => (
                <div key={e.token_hash} className={cardCls}>
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs text-gray-400">{e.token_hash.substring(0, 16)}...</span>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${e.active ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>{e.active ? "Active" : "Expired"}</span>
                  </div>
                  <div className="mt-2 space-y-1 text-xs">
                    <div className="flex justify-between"><span className="text-gray-400">Client:</span><span className="font-mono text-gray-500">{e.client_id.substring(0, 16)}...</span></div>
                    <div className="flex justify-between"><span className="text-gray-400">Scope:</span><span className="text-gray-500">{e.scope}</span></div>
                    <div className="flex justify-between"><span className="text-gray-400">Expires:</span><span className="text-gray-500">{new Date(e.expires_at).toLocaleString()}</span></div>
                  </div>
                  <button onClick={() => handleInvalidateEntry(e.token_hash)} className="mt-2 flex items-center gap-1 text-xs text-red-500">
                    <Trash2 className="h-3 w-3" /> Invalidate
                  </button>
                </div>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Invalidate all confirmation */}
      {confirmInvalidate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmInvalidate(false)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><ZapOff className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">Invalidate Entire Cache?</h2>
                <p className="text-sm text-gray-500">All {stats?.total_entries ?? 0} cached tokens will be removed. Subsequent introspection calls will query the authorization server directly until the cache rebuilds.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmInvalidate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("backend.introspection.cancel")}</button>
              <button onClick={handleInvalidate} disabled={invalidating} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">
                {invalidating ? <Loader2 className="h-4 w-4 animate-spin" /> : <ZapOff className="h-4 w-4" />} Invalidate All
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
