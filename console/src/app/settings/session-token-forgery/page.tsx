"use client";

import { useSessionTokenForgery } from "@ggid/sdk-react";
import { KeyRound, AlertTriangle, Shield, Activity, Ban } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SessionTokenForgeryPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useSessionTokenForgery();

  if (loading) return <div className="p-8 text-gray-400">Loading token forgery detection...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const methodColors: Record<string, string> = {
    signature_invalid: "bg-red-900 text-red-300",
    claim_mismatch: "bg-orange-900 text-orange-300",
    issuer_unknown: "bg-yellow-900 text-yellow-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Session Token Forgery Detection</h1>
          <p className="text-sm text-gray-400 mt-1">Detect forged or tampered session tokens</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <KeyRound className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Forged Tokens</p>
          <p className="text-xl font-bold text-red-400">{data?.forged_tokens?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Ban className="w-5 h-5 text-orange-400 mb-1" />
          <p className="text-xs text-gray-400">Blocked (24h)</p>
          <p className="text-xl font-bold">{data?.detection_stats?.blocked_attempts_24h ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Validation Failures</p>
          <p className="text-xl font-bold">{data?.detection_stats?.token_validation_failures ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Detection Rate</p>
          <p className="text-xl font-bold text-green-400">{data?.detection_stats?.detection_rate_pct ?? 0}%</p>
        </div>
      </div>

      {/* Forged Tokens Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Forged Tokens Detected</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Token (partial)</th>
                <th scope="col" className="text-left py-2 pr-3">Detection Method</th>
                <th scope="col" className="text-left py-2 pr-3">User Claimed</th>
                <th scope="col" className="text-left py-2 pr-3">Actual Source</th>
                <th scope="col" className="text-left py-2 pr-3">Timestamp</th>
              </tr>
            </thead>
            <tbody>
              {(data?.forged_tokens ?? []).map((t: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{t.token.substring(0, 32)}...</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (methodColors[t.detection_method] ?? "bg-gray-700 text-gray-300")}>
                      {t.detection_method}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-xs">{t.user_claimed}</td>
                  <td className="py-3 pr-3 font-mono text-xs text-gray-400">{t.actual_source}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{t.timestamp}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {(data?.forged_tokens?.length ?? 0) === 0 && (
          <p className="text-sm text-green-400 mt-2">No forged tokens detected</p>
        )}
      </div>
    </div>
  );
}
