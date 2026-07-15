"use client";

import { usePrivilegeEscalationDetect } from "@ggid/sdk-react";
import { ShieldAlert, TrendingUp, Zap, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function PrivilegeEscalationDetectPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = usePrivilegeEscalationDetect();

  if (loading) return <div className="p-8 text-gray-400">Loading privilege escalation detection...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Privilege Escalation Detection</h1>
          <p className="text-sm text-gray-400 mt-1">Detect unauthorized privilege escalation attempts</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <ShieldAlert className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Detected Events</p>
          <p className="text-xl font-bold text-red-400">{data?.detected_events?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <TrendingUp className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Avg Confidence</p>
          <p className="text-xl font-bold">
            {data?.detected_events?.length ? Math.round(data.detected_events.reduce((a, e) => a + e.confidence_score, 0) / data.detected_events.length * 100) : 0}%
          </p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">High Confidence</p>
          <p className="text-xl font-bold text-yellow-400">{data?.detected_events?.filter((e) => e.confidence_score > 0.8).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Zap className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Actions Taken</p>
          <p className="text-xl font-bold">{data?.detected_events?.filter((e) => e.action_taken !== "none").length ?? 0}</p>
        </div>
      </div>

      {/* Detected Events Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Detected Escalation Events</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">User</th>
                <th className="text-left py-2 pr-3">Role Change</th>
                <th className="text-left py-2 pr-3">Method</th>
                <th className="text-left py-2 pr-3">Patterns</th>
                <th className="text-left py-2 pr-3">Confidence</th>
                <th className="text-left py-2 pr-3">Action Taken</th>
                <th className="text-left py-2 pr-3">Timestamp</th>
              </tr>
            </thead>
            <tbody>
              {(data?.detected_events ?? []).map((e) => (
                <tr key={e.id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-sm font-medium">{e.user}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs text-gray-400">{e.from_role}</span>
                    <span className="text-xs text-gray-600 mx-1">{" -> "}</span>
                    <span className="text-xs text-red-400">{e.to_role}</span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{e.method}</td>
                  <td className="py-3 pr-3">
                    <div className="flex flex-wrap gap-1">
                      {e.patterns.map((p) => (
                        <span key={p} className={"text-xs px-1.5 py-0.5 rounded " + (
                          p === "mass_grant" ? "bg-red-900 text-red-300" :
                          p === "unusual_time" ? "bg-yellow-900 text-yellow-300" :
                          "bg-orange-900 text-orange-300"
                        )}>{p}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <div className="w-12 h-1.5 bg-gray-700 rounded-full">
                        <div className={"h-full rounded-full " + (e.confidence_score > 0.8 ? "bg-red-500" : e.confidence_score > 0.5 ? "bg-yellow-500" : "bg-green-500")} style={{ width: (e.confidence_score * 100) + "%" }} />
                      </div>
                      <span className="text-xs">{(e.confidence_score * 100).toFixed(0)}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      e.action_taken === "blocked" ? "bg-red-900 text-red-300" :
                      e.action_taken === "reverted" ? "bg-yellow-900 text-yellow-300" :
                      "bg-blue-900 text-blue-300"
                    )}>
                      {e.action_taken}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{e.timestamp}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Recommended Actions */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Zap className="w-4 h-4 text-yellow-400" />
          Recommended Actions
        </h2>
        <div className="space-y-2">
          {(data?.recommended_actions ?? []).map((a, i) => (
            <div key={i} className="flex items-start gap-3 bg-gray-800 rounded-lg p-3">
              <span className="text-xs font-bold text-blue-400 mt-0.5">{i + 1}.</span>
              <div className="flex-1">
                <p className="text-sm font-medium">{a.action}</p>
                <p className="text-xs text-gray-400">{a.reason}</p>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (
                a.priority === "critical" ? "bg-red-900 text-red-300" :
                a.priority === "high" ? "bg-orange-900 text-orange-300" :
                "bg-yellow-900 text-yellow-300"
              )}>
                {a.priority}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
