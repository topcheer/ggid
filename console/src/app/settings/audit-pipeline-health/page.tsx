"use client";

import { useAuditPipelineHealth } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Activity, ArrowRight, AlertTriangle, TrendingUp, Server, Zap } from "lucide-react";

export default function AuditPipelineHealthPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditPipelineHealth();

  if (loading) return <div className="p-8 text-gray-400">Loading pipeline health...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Pipeline Health</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor audit event processing pipeline stages</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Pipeline Visual */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex items-center justify-around gap-2 py-4">
          {(data?.pipeline_stages ?? []).map((stage, i) => (
            <div key={stage.name} className="flex items-center gap-2">
              <div
                className={"p-3 rounded-xl border-2 text-center min-w-[100px] " + (
                  stage.error_rate > 5 ? "bg-red-900/30 border-red-700" :
                  stage.queue_depth > 1000 ? "bg-yellow-900/30 border-yellow-700" :
                  "bg-green-900/30 border-green-700"
                )}
              >
                <p className="text-xs font-semibold">{stage.name}</p>
                <p className="text-xs text-gray-500 mt-0.5">{stage.throughput.toLocaleString()} ev/s</p>
              </div>
              {i < (data?.pipeline_stages?.length ?? 0) - 1 && (
                <ArrowRight className="w-4 h-4 text-gray-600" />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Per-Stage Metrics */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Stage Metrics</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Stage</th>
                <th className="text-right py-2 pr-3">Throughput</th>
                <th className="text-right py-2 pr-3">Latency</th>
                <th className="text-right py-2 pr-3">Error Rate</th>
                <th className="text-right py-2 pr-3">Queue Depth</th>
              </tr>
            </thead>
            <tbody>
              {(data?.pipeline_stages ?? []).map((s) => (
                <tr key={s.name} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-sm font-medium">{s.name}</td>
                  <td className="py-3 pr-3 text-right text-gray-300">{s.throughput.toLocaleString()} ev/s</td>
                  <td className="py-3 pr-3 text-right">
                    <span className={s.latency_ms > 100 ? "text-red-400" : s.latency_ms > 50 ? "text-yellow-400" : "text-green-400"}>
                      {s.latency_ms.toFixed(1)}ms
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-right">
                    <span className={s.error_rate > 5 ? "text-red-400 font-medium" : "text-green-400"}>
                      {s.error_rate.toFixed(2)}%
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-right">
                    <span className={s.queue_depth > 1000 ? "text-yellow-400 font-medium" : "text-gray-400"}>
                      {s.queue_depth.toLocaleString()}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Bottleneck Detection */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-4 h-4 text-yellow-400" />
            Bottleneck Detection
          </h2>
          <div className="space-y-2">
            {(data?.bottlenecks ?? []).map((b, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-1">
                  <span className={"w-2 h-2 rounded-full " + (b.severity === "critical" ? "bg-red-500" : "bg-yellow-500")} />
                  <p className="text-sm font-medium">{b.stage}</p>
                  <span className="text-xs text-gray-500 ml-auto capitalize">{b.severity}</span>
                </div>
                <p className="text-xs text-gray-400">{b.description}</p>
                <p className="text-xs text-blue-400 mt-1">Recommendation: {b.recommendation}</p>
              </div>
            ))}
            {(data?.bottlenecks?.length ?? 0) === 0 && (
              <p className="text-sm text-green-400">No bottlenecks detected</p>
            )}
          </div>
        </div>

        {/* Failover + Incident */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
              <Server className="w-4 h-4 text-blue-400" />
              Failover Status
            </h2>
            <div className="space-y-2">
              <div className="flex justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-400">Primary</span>
                <span className={"text-sm font-medium " + (data?.failover_status?.primary_healthy ? "text-green-400" : "text-red-400")}>
                  {data?.failover_status?.primary_healthy ? "Healthy" : "Unhealthy"}
                </span>
              </div>
              <div className="flex justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-400">Standby</span>
                <span className={"text-sm font-medium " + (data?.failover_status?.standby_ready ? "text-green-400" : "text-red-400")}>
                  {data?.failover_status?.standby_ready ? "Ready" : "Not Ready"}
                </span>
              </div>
              <div className="flex justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-400">Last Failover</span>
                <span className="text-sm text-gray-300">{data?.failover_status?.last_failover ?? "Never"}</span>
              </div>
            </div>
          </div>

          {data?.last_incident && (
            <div className="bg-gray-900 rounded-xl p-6">
              <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
                <Zap className="w-4 h-4 text-yellow-400" />
                Last Incident
              </h2>
              <div className="bg-gray-800 rounded-lg p-3">
                <p className="text-sm font-medium">{data.last_incident.description}</p>
                <p className="text-xs text-gray-400 mt-1">Duration: {data.last_incident.duration}</p>
                <p className="text-xs text-gray-500">Resolved: {data.last_incident.resolved_at}</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
