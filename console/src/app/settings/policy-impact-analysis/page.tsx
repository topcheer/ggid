"use client";

import { useState } from "react";
import { usePolicyImpactAnalysis } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Users, GitCompare, AlertTriangle, TrendingUp, ShieldCheck } from "lucide-react";

export default function PolicyImpactAnalysisPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = usePolicyImpactAnalysis();
  const [selectedPolicy, setSelectedPolicy] = useState("");

  if (loading) return <div className="p-8 text-gray-400">Loading policy impact analysis...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const analysis = selectedPolicy
    ? (data?.analyses ?? []).find((a: any) => a.policy_id === selectedPolicy)
    : (data?.analyses ?? [])[0];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Policy Impact Analysis</h1>
          <p className="text-sm text-gray-400 mt-1">Preview the impact of policy changes before deployment</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Policy Selector */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center gap-3">
          <label className="text-sm text-gray-400">Select Policy:</label>
          <select
            value={selectedPolicy || analysis?.policy_id || ""}
            onChange={(e) => setSelectedPolicy(e.target.value)}
            className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          >
            {(data?.analyses ?? []).map((a: any) => (
              <option key={a.policy_id} value={a.policy_id}>{a.policy_name}</option>
            ))}
          </select>
        </div>
      </div>

      {analysis && (
        <>
          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-gray-900 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-1 text-blue-400">
                <Users className="w-4 h-4" />
                <span className="text-xs text-gray-400">Affected Users</span>
              </div>
              <p className="text-2xl font-bold">{analysis.affected_users_count.toLocaleString()}</p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-1 text-green-400">
                <ShieldCheck className="w-4 h-4" />
                <span className="text-xs text-gray-400">Avg Risk Change</span>
              </div>
              <p className={"text-2xl font-bold " + (analysis.avg_risk_score_change >= 0 ? "text-red-400" : "text-green-400")}>
                {analysis.avg_risk_score_change >= 0 ? "+" : ""}{analysis.avg_risk_score_change.toFixed(1)}
              </p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-1 text-yellow-400">
                <AlertTriangle className="w-4 h-4" />
                <span className="text-xs text-gray-400">High-Risk Users</span>
              </div>
              <p className="text-2xl font-bold text-yellow-400">{analysis.high_risk_users}</p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-1 text-purple-400">
                <GitCompare className="w-4 h-4" />
                <span className="text-xs text-gray-400">Permission Deltas</span>
              </div>
              <p className="text-2xl font-bold">{analysis.permission_delta.length}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Permission Delta Table */}
            <div className="bg-gray-900 rounded-xl p-6">
              <h2 className="text-lg font-semibold mb-4">Permission Delta</h2>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-800 text-gray-400">
                      <th scope="col" className="text-left py-2 pr-4">User</th>
                      <th scope="col" className="text-right py-2 px-2 text-green-400">Added</th>
                      <th scope="col" className="text-right py-2 px-2 text-red-400">Removed</th>
                      <th scope="col" className="text-right py-2 px-2">Risk</th>
                    </tr>
                  </thead>
                  <tbody>
                    {analysis.permission_delta.slice(0, 10).map((d: any, i: number) => (
                      <tr key={i} className="border-b border-gray-800">
                        <td className="py-2 pr-4 text-gray-300">{d.user}</td>
                        <td className="text-right py-2 px-2 text-green-400">{d.added_perms}</td>
                        <td className="text-right py-2 px-2 text-red-400">{d.removed_perms}</td>
                        <td className={"text-right py-2 px-2 font-medium " + (d.risk_score_change >= 0 ? "text-red-400" : "text-green-400")}>
                          {d.risk_score_change >= 0 ? "+" : ""}{d.risk_score_change.toFixed(1)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <div className="space-y-6">
              {/* Before/After Comparison */}
              <div className="bg-gray-900 rounded-xl p-6">
                <h2 className="text-lg font-semibold mb-4">Before / After Comparison</h2>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-gray-400 mb-2">Current State</p>
                    {analysis.before.map((item: any, i: number) => (
                      <div key={i} className="bg-gray-800 rounded px-3 py-1.5 mb-1 text-xs">
                        <span className="text-gray-400">{item.metric}: </span>
                        <span className="font-medium">{item.value}</span>
                      </div>
                    ))}
                  </div>
                  <div>
                    <p className="text-xs text-blue-400 mb-2">After Change</p>
                    {analysis.after.map((item: any, i: number) => (
                      <div key={i} className="bg-gray-800 rounded px-3 py-1.5 mb-1 text-xs">
                        <span className="text-gray-400">{item.metric}: </span>
                        <span className={"font-medium " + (item.value !== analysis.before[i]?.value ? "text-yellow-400" : "")}>
                          {item.value}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* Timeline Projection */}
              <div className="bg-gray-900 rounded-xl p-6">
                <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
                  <TrendingUp className="w-5 h-5 text-blue-400" />
                  Risk Score Projection (14d)
                </h2>
                <div className="flex items-end gap-1 h-32">
                  {analysis.timeline_projection.map((score: any, i: number) => (
                    <div key={i} className="flex-1 flex flex-col items-center gap-1">
                      <div
                        className={"w-full rounded-t " + (score >= 70 ? "bg-red-500" : score >= 40 ? "bg-yellow-500" : "bg-green-500")}
                        style={{ height: `${score}%`, minHeight: "4px" }}
                        title={`Day ${i + 1}: ${score}`}
                      />
                    </div>
                  ))}
                </div>
                <div className="flex items-center justify-between mt-3 text-xs text-gray-500">
                  <span>Day 1</span>
                  <span>Day 14</span>
                </div>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
