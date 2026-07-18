"use client";

import { useState } from "react";
import { useComplianceEvidenceTracker } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { FileCheck, AlertTriangle, Clock, CheckCircle } from "lucide-react";

export default function ComplianceEvidenceTrackerPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useComplianceEvidenceTracker();
  const [activeTab, setActiveTab] = useState("SOC2");

  if (loading) return <div className="p-8 text-gray-400">Loading evidence tracker...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const frameworks = Object.keys(data?.frameworks ?? {});
  const activeMatrix = data?.frameworks?.[activeTab] ?? [];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Compliance Evidence Tracker</h1>
          <p className="text-sm text-gray-400 mt-1">Track evidence collection across compliance frameworks</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Overdue Alerts */}
      {(data?.overdue_alerts?.length ?? 0) > 0 && (
        <div className="bg-red-900/20 border border-red-800 rounded-xl p-4 mb-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-2 text-red-300">
            <AlertTriangle className="w-4 h-4" />
            Overdue Alerts ({data?.overdue_alerts?.length ?? 0})
          </h2>
          <div className="space-y-1">
            {(data?.overdue_alerts ?? []).map((a: any, i: number) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className="text-red-400 font-mono">{a.control_id}</span>
                <span className="text-gray-400">{a.framework}</span>
                <span className="text-red-300">{a.days_overdue} days overdue</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Framework Tabs */}
      <div className="flex gap-1 mb-4 overflow-x-auto pb-1">
        {frameworks.map((fw) => (
          <button
            key={fw}
            onClick={() => setActiveTab(fw)}
            className={"px-4 py-2 rounded-lg text-sm font-medium transition whitespace-nowrap " + (
              activeTab === fw ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
            )}
          >
            {fw}
          </button>
        ))}
      </div>

      {/* Evidence Matrix */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Evidence Matrix - {activeTab}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Control</th>
                <th scope="col" className="text-left py-2 pr-3">Evidence Type</th>
                <th scope="col" className="text-left py-2 pr-3">Last Collected</th>
                <th scope="col" className="text-left py-2 pr-3">Next Due</th>
                <th scope="col" className="text-left py-2 pr-3">Owner</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {activeMatrix.map((row: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{row.control}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{row.evidence_type}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{row.last_collected}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{row.next_due}</td>
                  <td className="py-3 pr-3 text-xs">{row.owner}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      row.status === "collected" ? "bg-green-900 text-green-300" :
                      row.status === "overdue" ? "bg-red-900 text-red-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}>
                      {row.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Auto Collection Rules */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <CheckCircle className="w-4 h-4 text-green-400" />
          Auto Collection Rules
        </h2>
        <div className="space-y-2">
          {(data?.auto_collection_rules ?? []).map((r) => (
            <div key={r.rule_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <div>
                <p className="text-sm font-medium">{r.rule_name}</p>
                <p className="text-xs text-gray-400">{r.description}</p>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                {r.enabled ? "Enabled" : "Disabled"}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
