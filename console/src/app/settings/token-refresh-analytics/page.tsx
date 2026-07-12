"use client";

import { useTokenRefreshAnalytics } from "@ggid/sdk-react";
import { Activity, Clock, XCircle, TrendingUp } from "lucide-react";

export default function TokenRefreshAnalyticsPage() {
  const { data, loading, error, refresh } = useTokenRefreshAnalytics();

  if (loading) return <div className="p-8 text-gray-400">Loading token refresh analytics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const successColor =
    (data?.refresh_success_rate ?? 0) >= 95
      ? "text-green-400"
      : (data?.refresh_success_rate ?? 0) >= 80
      ? "text-yellow-400"
      : "text-red-400";

  const maxRate = Math.max(...(data?.refresh_rate_per_hour ?? [1]), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Token Refresh Analytics</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor token refresh rates, success rates, and failure breakdown</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Top Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Avg Token Lifetime</span>
          </div>
          <p className="text-2xl font-bold">{data?.avg_token_lifetime_minutes ?? 0} min</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Refresh Success Rate</span>
          </div>
          <p className={`text-2xl font-bold ${successColor}`}>{data?.refresh_success_rate ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <TrendingUp className="w-4 h-4" />
            <span className="text-xs text-gray-400">Rotation Churn Rate</span>
          </div>
          <p className="text-2xl font-bold">{data?.rotation_churn_rate ?? 0}%</p>
        </div>
      </div>

      {/* Refresh Rate Chart */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Refresh Rate (per hour, 24h)</h2>
        <div className="flex items-end gap-1 h-40">
          {(data?.refresh_rate_per_hour ?? []).map((rate, i) => (
            <div key={i} className="flex-1 flex flex-col items-center gap-1">
              <div
                className="w-full rounded-t bg-blue-500 hover:bg-blue-400 transition-all"
                style={{ height: `${(rate / maxRate) * 100}%`, minHeight: "2px" }}
                title={`${i}:00 - ${rate} refreshes`}
              />
              {i % 3 === 0 && <span className="text-xs text-gray-500">{i}h</span>}
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Refresh by Client */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Refresh by Client</h2>
          <div className="space-y-2">
            {(data?.refresh_by_client ?? []).map((c) => (
              <div key={c.client_id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium">{c.client_name}</span>
                  <span className="text-sm text-gray-400">{c.refresh_count.toLocaleString()}</span>
                </div>
                <div className="w-full bg-gray-700 rounded-full h-1.5">
                  <div
                    className="bg-blue-500 rounded-full h-1.5"
                    style={{
                      width: `${
                        (c.refresh_count / Math.max(...(data?.refresh_by_client ?? [{ refresh_count: 1 }]).map((x) => x.refresh_count), 1)) * 100
                      }%`,
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Refresh Failures */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <XCircle className="w-5 h-5 text-red-400" />
            Refresh Failures (24h)
          </h2>
          <div className="space-y-3">
            {(data?.refresh_failures ?? []).map((f, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium font-mono">{f.error}</span>
                  <span className="text-sm font-bold text-red-400">{f.count}</span>
                </div>
                <p className="text-xs text-gray-400">{f.description}</p>
              </div>
            ))}
            {(data?.refresh_failures ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No refresh failures in the last 24h.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
