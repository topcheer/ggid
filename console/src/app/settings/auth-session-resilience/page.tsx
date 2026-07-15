"use client";

import { useAuthSessionResilience } from "@ggid/sdk-react";
import { Database, Server, ShieldCheck, Zap, Activity, AlertTriangle, Play } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthSessionResiliencePage() {
  const t = useTranslations();

  const { data, loading, error, refresh, testRecovery } = useAuthSessionResilience();

  if (loading) return <div className="p-8 text-gray-400">Loading session resilience...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Session Resilience</h1>
          <p className="text-sm text-gray-400 mt-1">Connection pools, failover, and degraded mode for session management</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Connection Pool Status */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Database className="w-5 h-5 text-blue-400" />
          Connection Pool Status
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-gray-800 rounded-lg p-4">
            <p className="text-xs text-gray-400 mb-1">Active Connections</p>
            <p className="text-2xl font-bold text-green-400">{data?.connection_pool_status?.active ?? 0}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <p className="text-xs text-gray-400 mb-1">Idle Connections</p>
            <p className="text-2xl font-bold text-blue-400">{data?.connection_pool_status?.idle ?? 0}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <p className="text-xs text-gray-400 mb-1">Max Pool Size</p>
            <p className="text-2xl font-bold text-yellow-400">{data?.connection_pool_status?.max ?? 0}</p>
          </div>
        </div>
        <div className="mt-3">
          <div className="w-full bg-gray-700 rounded-full h-2">
            <div
              className="bg-green-500 rounded-full h-2 transition-all"
              style={{
                width: `${((data?.connection_pool_status?.active ?? 0) / Math.max(data?.connection_pool_status?.max ?? 1, 1)) * 100}%`,
              }}
            />
          </div>
          <p className="text-xs text-gray-500 mt-1">Pool utilization</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Failover Config */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Server className="w-5 h-5 text-purple-400" />
            Failover Configuration
          </h2>
          <div className="space-y-3">
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-300">Primary Store</span>
                <span className="text-sm font-medium text-green-400">{data?.session_failover_config?.primary_redis ?? "N/A"}</span>
              </div>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-300">Fallback Store</span>
                <span className="text-sm font-medium text-yellow-400">{data?.session_failover_config?.fallback_memory ?? "N/A"}</span>
              </div>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-300">Grace Period (Outage)</span>
                <span className="text-sm font-medium">{data?.grace_period_during_outage ?? 0}s</span>
              </div>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-300 flex items-center gap-2">
                  <ShieldCheck className="w-3 h-3" />
                  Offline Token Validation
                </span>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    data?.offline_token_validation ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                  )}
                >
                  {data?.offline_token_validation ? "Enabled" : "Disabled"}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Degraded Mode + Recovery Test */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-yellow-400" />
              Degraded Mode Indicators
            </h2>
            <div className="space-y-2">
              {(data?.degraded_mode_indicators ?? []).map((ind, i) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center gap-2">
                    {ind.active ? (
                      <Zap className="w-3 h-3 text-red-400" />
                    ) : (
                      <Activity className="w-3 h-3 text-green-400" />
                    )}
                    <span className="text-sm text-gray-300">{ind.indicator}</span>
                  </div>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      ind.active ? "bg-red-900 text-red-300" : "bg-green-900 text-green-300"
                    )}
                  >
                    {ind.active ? "Triggered" : "Normal"}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-3">Session Recovery Test</h2>
            <p className="text-sm text-gray-400 mb-3">Simulate a Redis outage and verify fallback behavior.</p>
            <button
              onClick={() => testRecovery()}
              className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
            >
              <Play className="w-4 h-4" />
              Run Recovery Test
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
