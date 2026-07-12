"use client";

import { useSocialProviderStats } from "@ggid/sdk-react";
import { Users, TrendingUp, Zap } from "lucide-react";

export default function SocialProviderStatsPage() {
  const { data, loading, error, refresh } = useSocialProviderStats();

  if (loading) return <div className="p-8 text-gray-400">Loading social provider stats...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const maxLogin = Math.max(...(data?.providers ?? []).map((p) => p.login_count_30d), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Social Provider Statistics</h1>
          <p className="text-sm text-gray-400 mt-1">Login analytics across social identity providers</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Provider Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
        {(data?.providers ?? []).map((p) => (
          <div key={p.name} className="bg-gray-900 rounded-xl p-5">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold">{p.name}</h3>
              <span className={"text-xs px-2 py-0.5 rounded " + (p.status === "active" ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>{p.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-3 mb-3">
              <div>
                <p className="text-xs text-gray-500">Users</p>
                <p className="text-lg font-bold">{p.user_count.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Logins (30d)</p>
                <p className="text-lg font-bold text-blue-400">{p.login_count_30d.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Success Rate</p>
                <p className={"text-sm font-bold " + (p.success_rate >= 95 ? "text-green-400" : "text-yellow-400")}>{p.success_rate}%</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Avg Latency</p>
                <p className="text-sm font-bold">{p.avg_latency_ms}ms</p>
              </div>
            </div>
            {/* Login bar */}
            <div className="mb-2">
              <div className="w-full bg-gray-800 rounded-full h-1.5">
                <div className="bg-blue-600 h-1.5 rounded-full" style={{ width: (p.login_count_30d / maxLogin * 100) + "%" }} />
              </div>
            </div>
            <p className="text-xs text-gray-400 flex items-center gap-1"><TrendingUp className="w-3 h-3 text-green-400" /> {p.new_users_30d} new users (30d)</p>
          </div>
        ))}
      </div>

      {/* Top Errors */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Zap className="w-4 h-4 text-yellow-400" /> Top Errors</h2>
        <div className="space-y-2">
          {(data?.top_errors ?? []).map((e) => (
            <div key={e.error} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <span className="text-xs font-mono text-red-400 flex-1">{e.error}</span>
              <span className="text-xs text-gray-400">{e.provider}</span>
              <span className="text-xs font-bold text-gray-300">{e.count}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
