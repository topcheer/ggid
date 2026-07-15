"use client";

import { useAgentBehavioralMonitor } from "@ggid/sdk-react";
import { Activity, AlertTriangle, Bot, Gauge } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AgentBehavioralMonitorPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAgentBehavioralMonitor();

  if (loading) return <div className="p-8 text-gray-400">Loading agent behavioral monitor...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Agent Behavioral Monitor</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor AI agent behavior against established baselines</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Bot className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Monitored Agents</p>
          <p className="text-xl font-bold">{data?.monitored_agents?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Anomaly Alerts</p>
          <p className="text-xl font-bold text-red-400">{data?.anomaly_alerts?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Gauge className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Suspend Threshold</p>
          <p className="text-sm font-bold">{data?.auto_suspend_threshold ? (data.auto_suspend_threshold * 100).toFixed(0) + "%" : "N/A"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Avg Deviation</p>
          <p className="text-xl font-bold">
            {data?.monitored_agents?.length ? Math.round(data.monitored_agents.reduce((a, m) => a + m.deviation_score, 0) / data.monitored_agents.length * 100) : 0}%
          </p>
        </div>
      </div>

      {/* Monitored Agents */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Monitored Agents</h2>
        <div className="space-y-2">
          {(data?.monitored_agents ?? []).map((a) => (
            <div key={a.agent_id} className="flex items-center gap-4 bg-gray-800 rounded-lg p-3">
              <Bot className="w-4 h-4 text-gray-500" />
              <div className="flex-1">
                <p className="text-sm font-medium">{a.agent_name}</p>
                <p className="text-xs text-gray-400">{a.normal_baseline} vs {a.current_behavior}</p>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-500">Deviation:</span>
                <div className="w-16 h-1.5 bg-gray-700 rounded-full">
                  <div className={"h-full rounded-full " + (a.deviation_score > 0.7 ? "bg-red-500" : a.deviation_score > 0.4 ? "bg-yellow-500" : "bg-green-500")} style={{ width: (a.deviation_score * 100) + "%" }} />
                </div>
                <span className={"text-sm font-bold " + (a.deviation_score > 0.7 ? "text-red-400" : a.deviation_score > 0.4 ? "text-yellow-400" : "text-green-400")}>
                  {(a.deviation_score * 100).toFixed(0)}%
                </span>
              </div>
              {a.deviation_score > (data?.auto_suspend_threshold ?? 1) && (
                <span className="text-xs px-2 py-0.5 bg-red-900 text-red-300 rounded">Suspend</span>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Anomaly Alerts */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-4 h-4 text-red-400" />
          Anomaly Alerts
        </h2>
        <div className="space-y-2">
          {(data?.anomaly_alerts ?? []).map((alert, i) => (
            <div key={i} className="flex items-start gap-3 bg-gray-800 rounded-lg p-3">
              <AlertTriangle className="w-4 h-4 text-yellow-400 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium">{alert.agent_name}</p>
                  <span className={"text-xs px-1.5 py-0.5 rounded " + (
                    alert.type === "excessive_requests" ? "bg-red-900 text-red-300" :
                    alert.type === "unusual_api_pattern" ? "bg-orange-900 text-orange-300" :
                    "bg-yellow-900 text-yellow-300"
                  )}>
                    {alert.type}
                  </span>
                </div>
                <p className="text-xs text-gray-400 mt-0.5">{alert.description}</p>
                <p className="text-xs text-gray-500 mt-0.5">{alert.timestamp}</p>
              </div>
            </div>
          ))}
          {(data?.anomaly_alerts?.length ?? 0) === 0 && (
            <p className="text-sm text-gray-500">No anomalies detected</p>
          )}
        </div>
      </div>
    </div>
  );
}
