"use client";

import { useOAuthIntrospectionCache } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Database, Zap, Trash2, TrendingUp, AlertTriangle } from "lucide-react";

export default function OAuthIntrospectionCachePage() {
  const t = useTranslations();
  const { data, loading, error, refresh, purgeCache } = useOAuthIntrospectionCache();

  if (loading) return <div className="p-8 text-gray-400">Loading introspection cache...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Introspection Cache</h1>
          <p className="text-sm text-gray-400 mt-1">Token introspection caching for performance and scalability</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => purgeCache()}
            className="flex items-center gap-1 px-3 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
          >
            <Trash2 className="w-4 h-4" />
            Purge Cache
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Cache Config & Stats */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Database className="w-4 h-4" />
            <span className="text-xs text-gray-400">Enabled</span>
          </div>
          <p className="text-lg font-bold">{data?.cache_config.enabled ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">TTL</span>
          </div>
          <p className="text-lg font-bold">{data?.cache_config.ttl_seconds ?? 0}s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Database className="w-4 h-4" />
            <span className="text-xs text-gray-400">Max Entries</span>
          </div>
          <p className="text-lg font-bold">{((data?.cache_config.max_entries ?? 0) / 1000).toFixed(0)}K</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <TrendingUp className="w-4 h-4" />
            <span className="text-xs text-gray-400">Hit Rate</span>
          </div>
          <p className="text-lg font-bold text-green-400">{data?.hit_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Evictions/min</span>
          </div>
          <p className="text-lg font-bold">{data?.evictions_per_min ?? 0}</p>
        </div>
      </div>

      {/* Cache Size Bar */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-gray-400">Cache Size</span>
          <span className="text-sm font-medium">{(data?.cache_size_bytes ?? 0).toLocaleString()} bytes</span>
        </div>
        <div className="bg-gray-700 rounded-full h-2">
          <div
            className="bg-blue-500 rounded-full h-2"
            style={{ width: `${Math.min((data?.cache_size_bytes ?? 0) / 10485760 * 100, 100)}%` }}
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Cached Tokens Table */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Cached Tokens</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th scope="col" className="text-left py-2 pr-3">Token Hash</th>
                  <th scope="col" className="text-left py-2 pr-3">Client</th>
                  <th scope="col" className="text-left py-2 pr-3">Cached</th>
                  <th scope="col" className="text-left py-2 pr-3">Expires</th>
                </tr>
              </thead>
              <tbody>
                {(data?.cached_tokens ?? []).slice(0, 12).map((t) => (
                  <tr key={t.token_hash} className="border-b border-gray-800">
                    <td className="py-2 pr-3 font-mono text-xs text-blue-400">{t.token_hash.slice(0, 16)}</td>
                    <td className="py-2 pr-3 text-gray-300">{t.client}</td>
                    <td className="py-2 pr-3 text-gray-400 text-xs">{t.cached_at}</td>
                    <td className="py-2 pr-3 text-gray-400 text-xs">{t.expires_at}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Cache Invalidation Rules */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Cache Invalidation Rules</h2>
          <div className="space-y-2">
            {(data?.cache_invalidation_rules ?? []).map((rule, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium capitalize">{rule.trigger.replace(/_/g, " ")}</p>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      rule.action === "purge_entry" ? "bg-red-900 text-red-300" :
                      rule.action === "refresh" ? "bg-yellow-900 text-yellow-300" :
                      "bg-blue-900 text-blue-300"
                    )}
                  >
                    {rule.action}
                  </span>
                </div>
                <p className="text-xs text-gray-400">{rule.description}</p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
