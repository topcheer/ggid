"use client";

import { useState } from "react";
import { useComplianceGapReport } from "@ggid/sdk-react";
import { Download, AlertCircle, CheckCircle, Clock, FileText } from "lucide-react";

export default function ComplianceGapReportPage() {
  const { data, loading, error, refresh } = useComplianceGapReport();
  const [selectedFramework, setSelectedFramework] = useState("SOC2");

  if (loading) return <div className="p-8 text-gray-400">Loading compliance gap report...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const severityColors: Record<string, string> = {
    critical: "bg-red-900 text-red-300",
    high: "bg-orange-900 text-orange-300",
    medium: "bg-yellow-900 text-yellow-300",
    low: "bg-blue-900 text-blue-300",
  };

  const filteredGaps = (data?.gaps ?? []).filter((g) => g.framework === selectedFramework);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Compliance Gap Report</h1>
          <p className="text-sm text-gray-400 mt-1">Identify and track compliance gaps across frameworks</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" />
            PDF
          </button>
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <FileText className="w-4 h-4" />
            CSV
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <p className="text-xs text-gray-400 mb-1">Total Gaps</p>
          <p className="text-2xl font-bold text-red-400">{data?.summary?.total_gaps ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <p className="text-xs text-gray-400 mb-1">Critical / High</p>
          <p className="text-2xl font-bold">
            <span className="text-red-400">{data?.summary?.by_severity?.critical ?? 0}</span>
            <span className="text-gray-500 mx-1">/</span>
            <span className="text-orange-400">{data?.summary?.by_severity?.high ?? 0}</span>
          </p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <p className="text-xs text-gray-400 mb-1">Medium / Low</p>
          <p className="text-2xl font-bold">
            <span className="text-yellow-400">{data?.summary?.by_severity?.medium ?? 0}</span>
            <span className="text-gray-500 mx-1">/</span>
            <span className="text-blue-400">{data?.summary?.by_severity?.low ?? 0}</span>
          </p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <p className="text-xs text-gray-400 mb-1">Resolved (30d)</p>
          <p className="text-2xl font-bold text-green-400">{data?.summary?.resolved_30d ?? 0}</p>
        </div>
      </div>

      {/* Framework Selector */}
      <div className="flex items-center gap-2 mb-6">
        {(data?.frameworks ?? []).map((fw) => (
          <button
            key={fw}
            onClick={() => setSelectedFramework(fw)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition ${
              selectedFramework === fw
                ? "bg-blue-600 text-white"
                : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            }`}
          >
            {fw}
          </button>
        ))}
      </div>

      {/* Gaps Table */}
      <div className="bg-gray-900 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800 text-gray-400">
              <th className="text-left py-3 px-4">Control ID</th>
              <th className="text-left py-3 px-4">Requirement</th>
              <th className="text-left py-3 px-4">Current State</th>
              <th className="text-left py-3 px-4">Severity</th>
              <th className="text-left py-3 px-4">Owner</th>
              <th className="text-left py-3 px-4">Deadline</th>
            </tr>
          </thead>
          <tbody>
            {filteredGaps.map((gap) => (
              <tr key={gap.control_id} className="border-b border-gray-800 hover:bg-gray-800/50">
                <td className="py-3 px-4 font-mono text-blue-400">{gap.control_id}</td>
                <td className="py-3 px-4">
                  <p className="font-medium">{gap.requirement}</p>
                  <p className="text-xs text-gray-400 mt-1">{gap.remediation_plan}</p>
                </td>
                <td className="py-3 px-4 text-gray-300">{gap.current_state}</td>
                <td className="py-3 px-4">
                  <span className={`text-xs px-2 py-0.5 rounded font-medium ${severityColors[gap.gap_severity] ?? "bg-gray-700 text-gray-300"}`}>
                    {gap.gap_severity}
                  </span>
                </td>
                <td className="py-3 px-4 text-gray-300">{gap.owner}</td>
                <td className="py-3 px-4 text-gray-300">{gap.deadline}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {filteredGaps.length === 0 && (
          <div className="p-12 text-center text-gray-500">No gaps found for {selectedFramework}.</div>
        )}
      </div>
    </div>
  );
}
