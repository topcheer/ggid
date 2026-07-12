"use client";

import { useState } from "react";
import { useGDPRDataPortability } from "@ggid/sdk-react";
import { Download, FileJson, Clock, RefreshCw } from "lucide-react";

export default function GDPRDataPortabilityPage() {
  const { data, loading, error, refresh, generateExport } = useGDPRDataPortability();
  const [selectedScopes, setSelectedScopes] = useState<string[]>(["profile", "activity"]);

  if (loading) return <div className="p-8 text-gray-400">Loading data portability...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const scopeOptions = ["profile", "activity", "consents", "sessions", "audit_events"];
  const statusColors: Record<string, string> = {
    queued: "bg-gray-700 text-gray-300",
    processing: "bg-blue-900 text-blue-300",
    ready: "bg-green-900 text-green-300",
    expired: "bg-red-900 text-red-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">GDPR Data Portability</h1>
          <p className="text-sm text-gray-400 mt-1">Generate and manage data export requests (Article 20)</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Generate Export */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <FileJson className="w-4 h-4 text-blue-400" />
          Generate New Export
        </h2>
        <div className="space-y-3">
          <div>
            <p className="text-xs text-gray-400 mb-2">Data Scope (select what to include):</p>
            <div className="flex flex-wrap gap-2">
              {scopeOptions.map((scope) => (
                <button
                  key={scope}
                  onClick={() => setSelectedScopes(prev => prev.includes(scope) ? prev.filter(s => s !== scope) : [...prev, scope])}
                  className={"text-xs px-3 py-1.5 rounded-lg transition " + (
                    selectedScopes.includes(scope) ? "bg-blue-600 text-white" : "bg-gray-800 text-gray-400 hover:bg-gray-700"
                  )}
                >
                  {scope}
                </button>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-2">
            <select className="px-3 py-2 bg-gray-800 rounded-lg text-sm">
              <option>JSON</option><option>CSV</option><option>XML</option>
            </select>
            <button
              onClick={() => generateExport("current-user", selectedScopes)}
              className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
            >
              <Download className="w-4 h-4" /> Generate Export
            </button>
          </div>
          <p className="text-xs text-gray-500">Exports auto-expire after {data?.auto_expiry_days ?? 7} days</p>
        </div>
      </div>

      {/* Export Requests Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">Export Requests</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">User</th>
                <th className="text-left py-2 pr-3">Requested</th>
                <th className="text-left py-2 pr-3">Format</th>
                <th className="text-left py-2 pr-3">Scope</th>
                <th className="text-left py-2 pr-3">Status</th>
                <th className="text-left py-2 pr-3">Action</th>
              </tr>
            </thead>
            <tbody>
              {(data?.export_requests ?? []).map((r, i) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs">{r.user}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{r.requested_at}</td>
                  <td className="py-3 pr-3 text-xs">{r.format}</td>
                  <td className="py-3 pr-3">
                    <div className="flex gap-1">
                      {r.scope.map((s) => (
                        <span key={s} className="text-xs px-1 py-0.5 bg-gray-700 rounded text-gray-400">{s}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (statusColors[r.status] ?? "bg-gray-700")}>{r.status}</span>
                  </td>
                  <td className="py-3 pr-3">
                    {r.status === "ready" && r.download_link ? (
                      <a href={r.download_link} className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1">
                        <Download className="w-3 h-3" /> Download
                      </a>
                    ) : (
                      <span className="text-xs text-gray-500">--</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
