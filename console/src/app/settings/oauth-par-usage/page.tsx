"use client";

import { useOAuthParUsage } from "@ggid/sdk-react";
import { Database, Zap, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthParUsagePage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useOAuthParUsage();
  if (loading) return <div className="p-8 text-gray-400">Loading PAR usage...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">OAuth PAR Usage</h1><p className="text-sm text-gray-400 mt-1">Pushed Authorization Request statistics</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4"><Database className="w-5 h-5 text-blue-400 mb-1" /><p className="text-xs text-gray-400">Total Pushed</p><p className="text-xl font-bold">{(data?.total_pushed ?? 0).toLocaleString()}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><Activity className="w-5 h-5 text-green-400 mb-1" /><p className="text-xs text-gray-400">Active Requests</p><p className="text-xl font-bold">{data?.active_requests?.length ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4 text-center"><Zap className="w-5 h-5 text-purple-400 mx-auto mb-1" /><p className="text-xs text-gray-400">Hit Rate</p><p className="text-xl font-bold text-green-400">{data?.hit_rate ?? 0}%</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Cache Size</p><p className="text-xl font-bold">{(data?.cache_size ?? 0).toLocaleString()}</p></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3">Active Requests</h2>
        <div className="space-y-2">
          {(data?.active_requests ?? []).map((r) => (
            <div key={r.request_uri} className="flex items-center gap-3 bg-gray-800 rounded p-3">
              <span className="text-xs font-mono text-blue-400 flex-1">{r.request_uri}</span>
              <span className="text-xs text-gray-400">{r.client}</span>
              <span className="text-xs text-gray-500">pushed: {r.pushed_at}</span>
              <span className={"text-xs px-1.5 py-0.5 rounded " + (r.consumed ? "bg-green-900 text-green-300" : "bg-yellow-900 text-yellow-300")}>{r.consumed ? "consumed" : "pending"}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Per-Client Usage</h2>
          <div className="space-y-2">
            {(data?.per_client ?? []).map((c) => (
              <div key={c.client} className="flex items-center gap-2">
                <span className="text-xs w-32">{c.client}</span>
                <div className="flex-1 bg-gray-800 rounded-full h-2"><div className="bg-blue-600 h-2 rounded-full" style={{ width: c.pct + "%" }} /></div>
                <span className="text-xs text-gray-400">{c.count}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Error Responses</h2>
          <div className="space-y-1">
            {(data?.errors ?? []).map((e) => (
              <div key={e.error} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs"><span className="font-mono text-red-400 flex-1">{e.error}</span><span className="text-gray-400">{e.count}</span></div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
