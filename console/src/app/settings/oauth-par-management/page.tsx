"use client";
import { useTranslations } from "@/lib/i18n";

import { useOAuthParManagement } from "@ggid/sdk-react";
import { Database, Zap, Trash2, AlertTriangle, CheckCircle } from "lucide-react";

export default function OAuthParManagementPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useOAuthParManagement();

  if (loading) return <div className="p-8 text-gray-400">Loading PAR management...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const hitRate = data?.par_hit_rate ?? 0;
  const hitColor = hitRate >= 90 ? "text-green-400" : hitRate >= 70 ? "text-yellow-400" : "text-red-400";

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">OAuth PAR Management</h1>
          <p className="text-sm text-gray-400 mt-1">Pushed Authorization Requests (RFC 9126) monitoring and management</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Top Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Database className="w-4 h-4" />
            <span className="text-xs text-gray-400">PAR Cache Size</span>
          </div>
          <p className="text-2xl font-bold">{(data?.par_cache_size ?? 0).toLocaleString()}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">PAR Hit Rate</span>
          </div>
          <p className={`text-2xl font-bold ${hitColor}`}>{hitRate}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Trash2 className="w-4 h-4" />
            <span className="text-xs text-gray-400">Expired Cleanup (24h)</span>
          </div>
          <p className="text-2xl font-bold">{data?.expired_cleanup_count ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Error Responses (24h)</span>
          </div>
          <p className="text-2xl font-bold">{data?.error_responses?.length ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Active Pushed Requests */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Active Pushed Requests</h2>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {(data?.active_pushed_requests ?? []).map((req) => (
              <div key={req.request_uri} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <code className="text-xs text-blue-400 truncate">{req.request_uri}</code>
                  <span
                    className={"text-xs px-2 py-0.5 rounded flex-shrink-0 ml-2 " + (
                      req.consumed
                        ? "bg-gray-700 text-gray-400"
                        : "bg-green-900 text-green-300"
                    )}
                  >
                    {req.consumed ? "Consumed" : "Active"}
                  </span>
                </div>
                <div className="flex items-center justify-between text-xs text-gray-400">
                  <span>{req.client_name}</span>
                  <span>
                    Pushed: {req.pushed_at} | Expires: {req.expires_at}
                  </span>
                </div>
              </div>
            ))}
            {(data?.active_pushed_requests ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No active pushed requests.</p>
            )}
          </div>
        </div>

        <div className="space-y-6">
          {/* Per-Client Usage */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">Per-Client Usage</h2>
            <div className="space-y-2">
              {(data?.per_client_usage ?? []).map((c) => (
                <div key={c.client_id} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm font-medium">{c.client_name}</span>
                    <span className="text-sm text-gray-400">{c.request_count}</span>
                  </div>
                  <div className="w-full bg-gray-700 rounded-full h-1">
                    <div
                      className="bg-blue-500 rounded-full h-1"
                      style={{
                        width: `${
                          (c.request_count / Math.max(...(data?.per_client_usage ?? [{ request_count: 1 }]).map((x) => x.request_count), 1)) * 100
                        }%`,
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Error Responses */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-red-400" />
              Error Responses (24h)
            </h2>
            <div className="space-y-2">
              {(data?.error_responses ?? []).map((err, i) => (
                <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  <CheckCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-sm font-medium font-mono">{err.error_code}</p>
                    <p className="text-xs text-gray-400">{err.description}</p>
                  </div>
                  <span className="text-sm font-bold text-red-400">{err.count}</span>
                </div>
              ))}
              {(data?.error_responses ?? []).length === 0 && (
                <p className="text-sm text-gray-500 text-center py-4">No errors in the last 24h.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
