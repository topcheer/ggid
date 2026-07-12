"use client";

import { useIdpFailoverConfig } from "@ggid/sdk-react";
import { Server, ArrowRight, Zap, Activity } from "lucide-react";

export default function IdpFailoverConfigPage() {
  const { data, loading, error, refresh, manualSwitch } = useIdpFailoverConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading IdP failover config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">IdP Failover Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Automatic and manual IdP failover management</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Primary / Secondary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        {data?.idp_cards?.map((idp) => (
          <div key={idp.name} className={"rounded-xl p-6 border " + (idp.role === "primary" ? "bg-blue-950 border-blue-800" : "bg-gray-900 border-gray-700")}>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <Server className={"w-6 h-6 " + (idp.status === "healthy" ? "text-green-400" : "text-red-400")} />
                <div>
                  <h3 className="text-sm font-semibold">{idp.name}</h3>
                  <p className="text-xs text-gray-400">{idp.role}</p>
                </div>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (idp.status === "healthy" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>{idp.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-gray-500">Latency</p>
                <p className={"text-sm font-medium " + (idp.latency_ms < 200 ? "text-green-400" : "text-yellow-400")}>{idp.latency_ms}ms</p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Health Score</p>
                <p className={"text-sm font-medium " + (idp.health_score >= 90 ? "text-green-400" : idp.health_score >= 70 ? "text-yellow-400" : "text-red-400")}>{idp.health_score}%</p>
              </div>
            </div>
            {idp.role !== "primary" && (
              <button onClick={() => manualSwitch(idp.name)} className="mt-4 w-full px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-xs font-medium transition">
                Promote to Primary
              </button>
            )}
          </div>
        ))}
      </div>

      {/* Failover Rules */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3">Failover Rules</h2>
        <div className="space-y-2">
          {(data?.failover_rules ?? []).map((r) => (
            <div key={r.trigger} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <Zap className="w-4 h-4 text-yellow-400" />
              <div className="flex-1">
                <p className="text-sm">{r.trigger}</p>
                <p className="text-xs text-gray-400">{r.condition}</p>
              </div>
              <ArrowRight className="w-3 h-3 text-gray-600" />
              <span className="text-xs px-2 py-0.5 rounded bg-blue-900 text-blue-300">{r.action}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Config + History */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Configuration</h2>
          <div className="space-y-2">
            <div className="flex justify-between"><span className="text-xs text-gray-400">Health Check Interval</span><span className="text-sm">{data?.health_check_interval ?? "30s"}</span></div>
            <div className="flex justify-between"><span className="text-xs text-gray-400">Auto Fallback</span><span className="text-sm">{data?.auto_fallback ? "Enabled" : "Disabled"}</span></div>
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Failover History</h2>
          <div className="space-y-1">
            {(data?.failover_history ?? []).map((h) => (
              <div key={h.id} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
                <span className="text-gray-500">{h.timestamp}</span>
                <span className="text-gray-300">{h.from}</span>
                <ArrowRight className="w-3 h-3 text-gray-600" />
                <span className="text-gray-300">{h.to}</span>
                <span className="text-gray-500 ml-auto">{h.reason}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
