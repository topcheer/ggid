"use client";

import { useRemediationTracker } from "@ggid/sdk-react";
import { Wrench, AlertTriangle, TrendingUp } from "lucide-react";

export default function RemediationTrackerPage() {
  const { data, loading, error, refresh } = useRemediationTracker();

  if (loading) return <div className="p-8 text-gray-400">Loading remediation tracker...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Remediation Tracker</h1>
          <p className="text-sm text-gray-400 mt-1">Track remediation across vuln, pentest, audit, and compliance findings</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Wrench className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Total Items</p>
          <p className="text-xl font-bold">{data?.remediation_items?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <TrendingUp className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Completion Rate</p>
          <p className="text-xl font-bold text-green-400">{data?.completion_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Overdue</p>
          <p className="text-xl font-bold text-red-400">{data?.overdue_alerts?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Wrench className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">In Progress</p>
          <p className="text-xl font-bold text-yellow-400">{data?.remediation_items?.filter((r) => r.status === "in_progress").length ?? 0}</p>
        </div>
      </div>

      {/* Overdue Alerts */}
      {(data?.overdue_alerts?.length ?? 0) > 0 && (
        <div className="bg-red-900/20 border border-red-800 rounded-xl p-4 mb-6">
          <h2 className="text-sm font-semibold text-red-300 mb-2">Overdue Alerts</h2>
          <div className="space-y-1">
            {(data?.overdue_alerts ?? []).map((a, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className="text-red-400 font-mono">{a.finding_id}</span>
                <span className="text-gray-400">{a.source}</span>
                <span className="text-red-300">{a.days_overdue}d overdue</span>
                <span className="text-gray-500 ml-auto">Assigned: {a.assignee}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Remediation Items Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Remediation Items</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Source</th>
                <th className="text-left py-2 pr-3">Finding</th>
                <th className="text-left py-2 pr-3">Severity</th>
                <th className="text-left py-2 pr-3">Assignee</th>
                <th className="text-left py-2 pr-3">Due Date</th>
                <th className="text-left py-2 pr-3">Progress</th>
                <th className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.remediation_items ?? []).map((item, i) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-1.5 py-0.5 rounded " + (
                      item.source === "vuln" ? "bg-red-900 text-red-300" :
                      item.source === "pentest" ? "bg-orange-900 text-orange-300" :
                      item.source === "audit" ? "bg-blue-900 text-blue-300" :
                      "bg-purple-900 text-purple-300"
                    )}>{item.source}</span>
                  </td>
                  <td className="py-3 pr-3 text-xs">{item.finding}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs font-bold " + (
                      item.severity === "critical" ? "text-red-400" :
                      item.severity === "high" ? "text-orange-400" :
                      "text-yellow-400"
                    )}>{item.severity}</span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{item.assignee}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{item.due_date}</td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2 w-20">
                      <div className="flex-1 h-1.5 bg-gray-700 rounded-full">
                        <div className="h-full bg-blue-500 rounded-full" style={{ width: item.progress_pct + "%" }} />
                      </div>
                      <span className="text-xs">{item.progress_pct}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      item.status === "completed" ? "bg-green-900 text-green-300" :
                      item.status === "in_progress" ? "bg-yellow-900 text-yellow-300" :
                      item.status === "overdue" ? "bg-red-900 text-red-300" :
                      "bg-gray-700 text-gray-400"
                    )}>{item.status}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Per Team Breakdown */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">Per Team Breakdown</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-3">
          {(data?.per_team_breakdown ?? []).map((t) => (
            <div key={t.team} className="bg-gray-800 rounded-lg p-3 text-center">
              <p className="text-xs text-gray-400">{t.team}</p>
              <p className="text-lg font-bold">{t.total}</p>
              <p className="text-xs text-green-400">{t.completed} done</p>
              <div className="h-1 bg-gray-700 rounded-full mt-1">
                <div className="h-full bg-green-500 rounded-full" style={{ width: (t.total > 0 ? (t.completed / t.total) * 100 : 0) + "%" }} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
