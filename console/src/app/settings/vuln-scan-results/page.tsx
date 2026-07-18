"use client";

import { useVulnScanResults } from "@ggid/sdk-react";
import { Bug, RefreshCw, Filter } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function VulnScanResultsPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useVulnScanResults();

  if (loading) return <div className="p-8 text-gray-400">Loading vulnerability scan...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const sevColors: Record<string, string> = {
    Critical: "bg-red-900 text-red-300",
    High: "bg-orange-900 text-orange-300",
    Medium: "bg-yellow-900 text-yellow-300",
    Low: "bg-blue-900 text-blue-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Vulnerability Scan Results</h1>
          <p className="text-sm text-gray-400 mt-1">Track vulnerability scans and findings</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-3 mb-6">
        <div className="bg-gray-900 rounded-xl p-3 text-center">
          <p className="text-xs text-gray-400">Total Findings</p>
          <p className="text-xl font-bold">{data?.findings?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-3 text-center">
          <p className="text-xs text-gray-400">Critical</p>
          <p className="text-xl font-bold text-red-400">{data?.findings?.filter((f) => f.severity === "Critical").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-3 text-center">
          <p className="text-xs text-gray-400">High</p>
          <p className="text-xl font-bold text-orange-400">{data?.findings?.filter((f) => f.severity === "High").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-3 text-center">
          <p className="text-xs text-gray-400">Medium</p>
          <p className="text-xl font-bold text-yellow-400">{data?.findings?.filter((f) => f.severity === "Medium").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-3 text-center">
          <p className="text-xs text-gray-400">Low</p>
          <p className="text-xl font-bold text-blue-400">{data?.findings?.filter((f) => f.severity === "Low").length ?? 0}</p>
        </div>
      </div>

      {/* Scan Runs */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Recent Scan Runs</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Date</th>
                <th scope="col" className="text-left py-2 pr-3">Scanner</th>
                <th scope="col" className="text-left py-2 pr-3">Scope</th>
                <th scope="col" className="text-left py-2 pr-3">Total</th>
                <th scope="col" className="text-left py-2 pr-3">Crit/High</th>
              </tr>
            </thead>
            <tbody>
              {(data?.scan_runs ?? []).map((r: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs text-gray-400">{r.date}</td>
                  <td className="py-3 pr-3 text-xs">{r.scanner}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{r.scope}</td>
                  <td className="py-3 pr-3 text-xs font-medium">{r.total}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs font-bold " + (r.critical_high > 0 ? "text-red-400" : "text-green-400")}>{r.critical_high}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Findings List */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Bug className="w-4 h-4 text-red-400" />
          Findings
        </h2>
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {(data?.findings ?? []).map((f: any, i: number) => (
            <div key={i} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-start justify-between mb-1">
                <div>
                  <p className="text-sm font-medium font-mono text-blue-400">{f.cve}</p>
                  <p className="text-xs text-gray-400 mt-0.5">{f.description}</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (sevColors[f.severity] ?? "bg-gray-700")}>{f.severity}</span>
              </div>
              <div className="flex items-center gap-3 mt-2">
                <span className="text-xs text-gray-500">CVSS: {f.cvss}</span>
                <span className="text-xs text-gray-500">Affected: {f.affected_component}</span>
                {f.fix_available && <span className="text-xs text-green-400">Fix available</span>}
                <span className={"text-xs px-1.5 py-0.5 rounded ml-auto " + (
                  f.status === "open" ? "bg-red-900 text-red-300" :
                  f.status === "fixed" ? "bg-green-900 text-green-300" :
                  "bg-gray-700 text-gray-300"
                )}>{f.status}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
