"use client";

import { usePIIDiscovery } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Database, Shield, AlertTriangle, RefreshCw } from "lucide-react";

export default function PIIDiscoveryPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = usePIIDiscovery();

  if (loading) return <div className="p-8 text-gray-400">Loading PII discovery...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">PII Discovery</h1>
          <p className="text-sm text-gray-400 mt-1">Automated discovery and classification of PII data</p>
        </div>
        <button onClick={refresh} className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">
          <RefreshCw className="w-4 h-4" /> Rescan
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Database className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Data Sources Scanned</p>
          <p className="text-xl font-bold">{data?.data_sources?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Coverage</p>
          <p className="text-xl font-bold text-green-400">{data?.coverage_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Unencrypted PII</p>
          <p className="text-xl font-bold text-red-400">{data?.unencrypted_pii_alerts?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Databases</p>
          <p className="text-xl font-bold">{data?.per_database_breakdown?.length ?? 0}</p>
        </div>
      </div>

      {/* Data Sources Scan Results */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Scan Results</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Table.Column</th>
                <th scope="col" className="text-left py-2 pr-3">PII Type</th>
                <th scope="col" className="text-left py-2 pr-3">Sample (Masked)</th>
                <th scope="col" className="text-left py-2 pr-3">Confidence</th>
                <th scope="col" className="text-left py-2 pr-3">Encrypted</th>
              </tr>
            </thead>
            <tbody>
              {(data?.data_sources ?? []).map((s: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs font-mono text-blue-400">{s.table}.{s.column}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs px-2 py-0.5 rounded bg-purple-900 text-purple-300">{s.pii_type}</span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400 font-mono">{s.sample_masked}</td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2 w-20">
                      <div className="flex-1 h-1.5 bg-gray-700 rounded-full">
                        <div className={"h-full rounded-full " + (s.confidence > 80 ? "bg-green-500" : s.confidence > 50 ? "bg-yellow-500" : "bg-red-500")} style={{ width: s.confidence + "%" }} />
                      </div>
                      <span className="text-xs">{s.confidence}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    {s.encrypted ? <span className="text-xs text-green-400">Yes</span> : <span className="text-xs text-red-400">No</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Unencrypted Alerts + Per DB Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold text-red-400 mb-3">Unencrypted PII Alerts</h2>
          <div className="space-y-2">
            {(data?.unencrypted_pii_alerts ?? []).map((a: any, i: number) => (
              <div key={i} className="flex items-center gap-2 bg-red-900/20 border border-red-800 rounded-lg p-2">
                <AlertTriangle className="w-4 h-4 text-red-400 flex-shrink-0" />
                <span className="text-xs text-gray-300 font-mono">{a.location}</span>
                <span className="text-xs text-red-300 ml-auto">{a.pii_type}</span>
              </div>
            ))}
            {(data?.unencrypted_pii_alerts?.length ?? 0) === 0 && <p className="text-sm text-green-400">No unencrypted PII found</p>}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Per Database Breakdown</h2>
          <div className="space-y-2">
            {(data?.per_database_breakdown ?? []).map((d: any) => (
              <div key={d.database} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-sm font-mono text-gray-300">{d.database}</span>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-gray-400">{d.pii_columns} columns</span>
                  <span className={"text-xs font-bold " + (d.unencrypted > 0 ? "text-red-400" : "text-green-400")}>{d.unencrypted} unencrypted</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
