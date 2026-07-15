"use client";

import { useGoldenTicketDetect } from "@ggid/sdk-react";
import { Ticket, ShieldAlert, Activity, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function GoldenTicketDetectPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useGoldenTicketDetect();

  if (loading) return <div className="p-8 text-gray-400">Loading Golden Ticket detection...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const anomalyColors: Record<string, string> = {
    issuer_mismatch: "bg-red-900 text-red-300",
    signature_anomaly: "bg-orange-900 text-orange-300",
    abnormal_claims: "bg-yellow-900 text-yellow-300",
    expiry_anomaly: "bg-purple-900 text-purple-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Golden Ticket Detection</h1>
          <p className="text-sm text-gray-400 mt-1">Detect forged Kerberos tickets and token anomalies</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Ticket className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Detected Forgeries</p>
          <p className="text-xl font-bold text-red-400">{data?.detected_forgeries?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">False Positive Rate</p>
          <p className="text-xl font-bold">{data?.false_positive_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <ShieldAlert className="w-5 h-5 text-orange-400 mb-1" />
          <p className="text-xs text-gray-400">Detection Rules</p>
          <p className="text-xl font-bold">{data?.detection_rules?.filter((r) => r.enabled).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Zap className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Revoke</p>
          <p className="text-sm font-bold">{data?.auto_revoke_enabled ? "Enabled" : "Disabled"}</p>
        </div>
      </div>

      {/* False Positive Rate Gauge */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex items-center gap-6">
          <div className="relative w-20 h-20">
            <svg className="w-20 h-20 -rotate-90" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="40" fill="none" stroke="#374151" strokeWidth="10" />
              <circle cx="50" cy="50" r="40" fill="none" stroke={data?.false_positive_rate_pct ?? 0 < 5 ? "#22c55e" : "#eab308"} strokeWidth="10" strokeDasharray={((data?.false_positive_rate_pct ?? 0) / 100 * 251.2) + " " + 251.2} strokeLinecap="round" />
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-lg font-bold">{data?.false_positive_rate_pct ?? 0}%</span>
            </div>
          </div>
          <div>
            <h2 className="text-sm font-semibold">False Positive Rate</h2>
            <p className="text-xs text-gray-400">Lower is better - target below 5%</p>
          </div>
        </div>
      </div>

      {/* Detected Forgeries Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Detected Forgeries</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">Token Hash</th>
                <th className="text-left py-2 pr-3">Anomaly Type</th>
                <th className="text-left py-2 pr-3">User</th>
                <th className="text-left py-2 pr-3">Source IP</th>
                <th className="text-left py-2 pr-3">Timestamp</th>
              </tr>
            </thead>
            <tbody>
              {(data?.detected_forgeries ?? []).map((f) => (
                <tr key={f.token_hash} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{f.token_hash.substring(0, 24)}...</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (anomalyColors[f.anomaly_type] ?? "bg-gray-700 text-gray-300")}>
                      {f.anomaly_type}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-xs">{f.user}</td>
                  <td className="py-3 pr-3 font-mono text-xs text-gray-400">{f.source_ip}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{f.timestamp}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {(data?.detected_forgeries?.length ?? 0) === 0 && (
          <p className="text-sm text-green-400 mt-2">No forged tickets detected</p>
        )}
      </div>

      {/* Detection Rules */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <ShieldAlert className="w-4 h-4 text-blue-400" />
          Detection Rules
        </h2>
        <div className="space-y-2">
          {(data?.detection_rules ?? []).map((r) => (
            <div key={r.rule_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <div>
                <p className="text-sm font-medium">{r.rule_name}</p>
                <p className="text-xs text-gray-400">{r.description}</p>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                {r.enabled ? "Active" : "Disabled"}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
