"use client";

import { useAgentLifecycle } from "@ggid/sdk-react";
import { Bot, Plus, Key, AlertTriangle, Activity, RotateCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AgentLifecyclePage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAgentLifecycle();

  if (loading) return <div className="p-8 text-gray-400">Loading agent lifecycle...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Agent Lifecycle</h1>
          <p className="text-sm text-gray-400 mt-1">Manage AI agent identities, credentials, and behavior</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
            <Plus className="w-4 h-4" />
            Provision Agent
          </button>
          <button onClick={refresh} aria-label="Refresh agent lifecycle data" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Bot className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Total Agents</p>
          <p className="text-xl font-bold">{data?.registered_agents?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Active Agents</p>
          <p className="text-xl font-bold text-green-400">{data?.registered_agents?.filter((a) => a.status === "active").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Behavioral Alerts</p>
          <p className="text-xl font-bold text-red-400">{data?.behavioral_alerts?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <RotateCw className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Rotation Due</p>
          <p className="text-xl font-bold text-yellow-400">{data?.registered_agents?.filter((a) => a.rotation_due).length ?? 0}</p>
        </div>
      </div>

      {/* Registered Agents Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Registered Agents</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Agent</th>
                <th scope="col" className="text-left py-2 pr-3">Owner</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
                <th scope="col" className="text-left py-2 pr-3">Last Active</th>
                <th scope="col" className="text-left py-2 pr-3">Req/min</th>
                <th scope="col" className="text-left py-2 pr-3">Permissions</th>
                <th scope="col" className="text-left py-2 pr-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {(data?.registered_agents ?? []).map((a) => (
                <tr key={a.agent_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <Bot className="w-3 h-3 text-gray-500" />
                      <div>
                        <p className="text-sm font-medium">{a.name}</p>
                        <p className="text-xs text-gray-500 font-mono">{a.agent_id}</p>
                      </div>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{a.owner}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      a.status === "active" ? "bg-green-900 text-green-300" :
                      a.status === "suspended" ? "bg-red-900 text-red-300" :
                      "bg-gray-700 text-gray-400"
                    )}>
                      {a.status}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{a.last_active}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs " + (a.request_rate_per_min > 100 ? "text-red-400 font-medium" : "text-gray-400")}>
                      {a.request_rate_per_min}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex flex-wrap gap-0.5">
                      {a.permissions.slice(0, 3).map((p) => (
                        <span key={p} className="text-xs px-1 py-0.5 bg-gray-800 rounded">{p}</span>
                      ))}
                      {a.permissions.length > 3 && <span className="text-xs text-gray-500">+{a.permissions.length - 3}</span>}
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-1">
                      {a.rotation_due && <Key className="w-3 h-3 text-yellow-400" />}
                      <button className="text-xs text-red-400 hover:underline">Revoke</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Credential Rotation Schedule */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
          <RotateCw className="w-4 h-4 text-yellow-400" />
          Credential Rotation Schedule
        </h2>
        <p className="text-xs text-gray-400">Auto-rotate every {data?.credential_rotation_schedule?.interval_days ?? 90} days - Next: {data?.credential_rotation_schedule?.next_rotation ?? "N/A"}</p>
      </div>

      {/* Behavioral Alerts */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-4 h-4 text-red-400" />
          Behavioral Alerts
        </h2>
        <div className="space-y-2">
          {(data?.behavioral_alerts ?? []).map((alert, i) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <AlertTriangle className="w-3 h-3 text-yellow-400" />
              <div className="flex-1">
                <p className="text-sm font-medium">{alert.agent_name}</p>
                <p className="text-xs text-gray-400">{alert.pattern}</p>
              </div>
              <span className="text-xs text-gray-500">{alert.timestamp}</span>
            </div>
          ))}
          {(data?.behavioral_alerts?.length ?? 0) === 0 && (
            <p className="text-sm text-gray-500">No behavioral alerts</p>
          )}
        </div>
      </div>
    </div>
  );
}
