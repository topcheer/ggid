"use client";

import { useAuditChainVerification } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ShieldCheck, CheckCircle, XCircle, AlertTriangle, Clock, Zap } from "lucide-react";

export default function AuditChainVerificationPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, verifyNow } = useAuditChainVerification();

  if (loading) return <div className="p-8 text-gray-400">Loading chain verification...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Chain Verification</h1>
          <p className="text-sm text-gray-400 mt-1">Verify tamper-evidence of audit log hash chain</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => verifyNow()}
            className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <Zap className="w-4 h-4" />
            Verify Now
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Last Verification Result */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <ShieldCheck className="w-4 h-4" />
            <span className="text-xs text-gray-400">Blocks Verified</span>
          </div>
          <p className="text-2xl font-bold">{data?.last_verification?.blocks_verified.toLocaleString() ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <XCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Blocks Failed</span>
          </div>
          <p className="text-2xl font-bold">{data?.last_verification?.blocks_failed ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Chain Integrity</span>
          </div>
          <p className={"text-2xl font-bold " + ((data?.last_verification?.chain_integrity_pct ?? 100) === 100 ? "text-green-400" : "text-red-400")}>
            {data?.last_verification?.chain_integrity_pct ?? 100}%
          </p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Auto-Verify</span>
          </div>
          <p className="text-sm font-bold font-mono">{data?.auto_verify_schedule ?? "0 0 * * *"}</p>
        </div>
      </div>

      {/* Alert Config */}
      <div className={"rounded-xl p-4 mb-6 flex items-center gap-3 " + (
        data?.alert_on_failure ? "bg-green-900/30 border border-green-800" : "bg-gray-900 border border-gray-800"
      )}>
        <AlertTriangle className={"w-5 h-5 " + (data?.alert_on_failure ? "text-green-400" : "text-gray-500")} />
        <span className="text-sm">Alert on failure: {data?.alert_on_failure ? "Enabled - will notify security team on chain integrity failure" : "Disabled"}</span>
      </div>

      {/* Verification Log Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Clock className="w-5 h-5 text-blue-400" />
          Verification Log
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Run ID</th>
                <th scope="col" className="text-left py-2 pr-3">Timestamp</th>
                <th scope="col" className="text-left py-2 pr-3">Result</th>
                <th scope="col" className="text-left py-2 pr-3">Duration</th>
                <th scope="col" className="text-left py-2 pr-3">Anomalies</th>
              </tr>
            </thead>
            <tbody>
              {(data?.verification_log ?? []).map((v: any) => (
                <tr key={v.run_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{v.run_id}</td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{v.timestamp}</td>
                  <td className="py-3 pr-3">
                    <span className={"flex items-center gap-1 text-xs " + (
                      v.result === "pass" ? "text-green-400" :
                      v.result === "fail" ? "text-red-400" :
                      "text-yellow-400"
                    )}>
                      {v.result === "pass" ? <CheckCircle className="w-3 h-3" /> :
                       v.result === "fail" ? <XCircle className="w-3 h-3" /> :
                       <AlertTriangle className="w-3 h-3" />}
                      {v.result.toUpperCase()}
                    </span>
                  </td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{v.duration_ms}ms</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs " + (v.anomalies_found > 0 ? "text-red-400 font-bold" : "text-gray-400")}>
                      {v.anomalies_found}
                    </span>
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
