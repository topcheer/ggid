"use client";

import { useDataRetentionDashboard } from "@ggid/sdk-react";
import { Clock, Calendar, Archive, Download } from "lucide-react";

export default function DataRetentionDashboardPage() {
  const { data, loading, error, refresh } = useDataRetentionDashboard();

  if (loading) return <div className="p-8 text-gray-400">Loading retention dashboard...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const actionIcons: Record<string, string> = {
    archive: "bg-blue-900 text-blue-300",
    delete: "bg-red-900 text-red-300",
    anonymize: "bg-purple-900 text-purple-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Data Retention Dashboard</h1>
          <p className="text-sm text-gray-400 mt-1">Manage data retention policies and automated purges</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Retention Policies */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Clock className="w-4 h-4 text-blue-400" />
          Retention Policies
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Data Type</th>
                <th className="text-left py-2 pr-3">Retention</th>
                <th className="text-left py-2 pr-3">Action</th>
                <th className="text-left py-2 pr-3">Legal Basis</th>
              </tr>
            </thead>
            <tbody>
              {(data?.retention_policies ?? []).map((p, i) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs font-medium">{p.data_type}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{p.retention_days} days</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (actionIcons[p.action] ?? "bg-gray-700")}>{p.action}</span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-500">{p.legal_basis}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Storage by Age + Upcoming Purges */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Storage Usage by Age</h2>
          <div className="space-y-2">
            {(data?.storage_usage_by_age ?? []).map((s, i) => (
              <div key={i} className="flex items-center gap-3">
                <span className="text-xs w-20 text-gray-400">{s.age_range}</span>
                <div className="flex-1 h-4 bg-gray-800 rounded">
                  <div className="h-full bg-blue-500 rounded" style={{ width: s.pct + "%" }} />
                </div>
                <span className="text-xs w-16 text-right">{s.size_gb}GB</span>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Calendar className="w-4 h-4 text-red-400" />
            Upcoming Purges
          </h2>
          <div className="space-y-2">
            {(data?.upcoming_purges ?? []).map((p, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">{p.data_type}</span>
                  <span className="text-xs text-gray-400">{p.date}</span>
                </div>
                <p className="text-xs text-gray-500 mt-0.5">{p.affected_records.toLocaleString()} records</p>
              </div>
            ))}
            {(data?.upcoming_purges?.length ?? 0) === 0 && <p className="text-sm text-green-400">No purges scheduled</p>}
          </div>
        </div>
      </div>

      {/* Compliance + Purge History */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Compliance Status</h2>
          <div className="flex flex-wrap gap-2">
            {(data?.compliance_status ?? []).map((c) => (
              <span key={c.framework} className={"text-xs px-2 py-1 rounded " + (c.compliant ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                {c.framework} {c.compliant ? "OK" : "GAP"}
              </span>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Archive className="w-4 h-4 text-purple-400" />
            Purge History
          </h2>
          <div className="space-y-1">
            {(data?.purge_history ?? []).map((p, i) => (
              <div key={i} className="flex items-center justify-between text-xs">
                <span className="text-gray-400">{p.date} - {p.data_type}</span>
                <span className="text-gray-500">{p.records_purged.toLocaleString()} purged</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
