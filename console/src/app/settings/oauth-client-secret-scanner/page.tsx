"use client";

import { useOAuthClientSecretScanner } from "@ggid/sdk-react";
import { Search, ShieldAlert, RefreshCw, AlertTriangle, CheckCircle, RotateCw, Code, GitBranch } from "lucide-react";

export default function OAuthClientSecretScannerPage() {
  const { data, loading, error, refresh, autoRotateExposed } = useOAuthClientSecretScanner();

  if (loading) return <div className="p-8 text-gray-400">Loading secret scanner...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Client Secret Scanner</h1>
          <p className="text-sm text-gray-400 mt-1">Scan codebase and git history for exposed OAuth client secrets</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => autoRotateExposed()}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition"
          >
            <RotateCw className="w-4 h-4" />
            Auto-Rotate Exposed
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Scan Settings */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Code className="w-4 h-4" />
            <span className="text-xs text-gray-400">Codebase Scan</span>
          </div>
          <p className="text-lg font-bold">{data?.codebase_scan_enabled ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <GitBranch className="w-4 h-4" />
            <span className="text-xs text-gray-400">Git History Scan</span>
          </div>
          <p className="text-lg font-bold">{data?.git_history_scan_enabled ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <RefreshCw className="w-4 h-4" />
            <span className="text-xs text-gray-400">Scan Frequency</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.scan_frequency ?? "daily"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Secrets Found</span>
          </div>
          <p className="text-lg font-bold">{data?.secrets_found?.length ?? 0}</p>
        </div>
      </div>

      {/* Scan Results Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <ShieldAlert className="w-5 h-5 text-yellow-400" />
          Scan Results
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-4">Client ID</th>
                <th className="text-left py-2 pr-4">Secret in Source</th>
                <th className="text-left py-2 pr-4">Last Rotated</th>
                <th className="text-left py-2 pr-4">Exposure Risk</th>
              </tr>
            </thead>
            <tbody>
              {(data?.scan_results ?? []).map((r) => (
                <tr key={r.client_id} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-mono text-blue-400">{r.client_id}</td>
                  <td className="py-3 pr-4">
                    {r.secret_in_source ? (
                      <span className="flex items-center gap-1 text-red-400"><AlertTriangle className="w-3 h-3" /> Yes</span>
                    ) : (
                      <span className="flex items-center gap-1 text-green-400"><CheckCircle className="w-3 h-3" /> No</span>
                    )}
                  </td>
                  <td className="py-3 pr-4 text-gray-300">{r.last_rotated_days > 0 ? `${r.last_rotated_days} days ago` : "Never"}</td>
                  <td className="py-3 pr-4">
                    <span
                      className={"text-xs px-2 py-0.5 rounded " + (
                        r.exposure_risk === "critical" ? "bg-red-900 text-red-300" :
                        r.exposure_risk === "high" ? "bg-orange-900 text-orange-300" :
                        r.exposure_risk === "medium" ? "bg-yellow-900 text-yellow-300" :
                        "bg-green-900 text-green-300"
                      )}
                    >
                      {r.exposure_risk}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Secrets Found List */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Search className="w-5 h-5 text-red-400" />
          Secrets Found in Source
        </h2>
        <div className="space-y-2">
          {(data?.secrets_found ?? []).map((s, i) => (
            <div key={i} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between mb-1">
                <code className="text-xs text-blue-400 font-mono">{s.file}:{s.line}</code>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    s.severity === "critical" ? "bg-red-900 text-red-300" :
                    "bg-yellow-900 text-yellow-300"
                  )}
                >
                  {s.severity}
                </span>
              </div>
              <p className="text-xs font-mono text-gray-400 bg-gray-900 rounded px-2 py-1">{s.preview_masked}</p>
            </div>
          ))}
          {(data?.secrets_found ?? []).length === 0 && (
            <p className="text-sm text-gray-500 text-center py-4">No secrets found in source code.</p>
          )}
        </div>
      </div>
    </div>
  );
}
