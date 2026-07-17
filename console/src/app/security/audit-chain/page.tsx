"use client";
import { useAuditChainVerification } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ShieldCheck, CheckCircle, XCircle, AlertTriangle, Clock, Zap, RefreshCw, Loader2 } from "lucide-react";

export default function AuditChainPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, verifyNow } = useAuditChainVerification();

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading) return <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div>;
  if (error) return (
    <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
      <AlertTriangle className="h-4 w-4 shrink-0" /> {error}
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ShieldCheck className="h-6 w-6 text-indigo-500" /> Audit Chain Verification
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Verify tamper-evidence of the audit log hash chain.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => verifyNow()} aria-label="Verify chain now"
            className="flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700">
            <Zap className="w-4 h-4" /> Verify Now
          </button>
          <button onClick={refresh} aria-label="Refresh"
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium dark:border-gray-700">
            <RefreshCw className="w-4 h-4" /> Refresh
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div className={card}>
          <div className="flex items-center gap-2 text-blue-400"><ShieldCheck className="h-4 w-4" /><span className="text-xs text-gray-400">Blocks Verified</span></div>
          <p className="mt-1 text-2xl font-bold">{data?.last_verification?.blocks_verified?.toLocaleString() ?? 0}</p>
        </div>
        <div className={card}>
          <div className="flex items-center gap-2 text-red-400"><XCircle className="h-4 w-4" /><span className="text-xs text-gray-400">Blocks Failed</span></div>
          <p className="mt-1 text-2xl font-bold">{data?.last_verification?.blocks_failed ?? 0}</p>
        </div>
        <div className={card}>
          <div className="flex items-center gap-2 text-green-400"><CheckCircle className="h-4 w-4" /><span className="text-xs text-gray-400">Chain Integrity</span></div>
          <p className={"mt-1 text-2xl font-bold " + ((data?.last_verification?.chain_integrity_pct ?? 100) === 100 ? "text-green-600" : "text-red-600")}>
            {data?.last_verification?.chain_integrity_pct ?? 100}%
          </p>
        </div>
        <div className={card}>
          <div className="flex items-center gap-2 text-purple-400"><Clock className="h-4 w-4" /><span className="text-xs text-gray-400">Auto-Verify</span></div>
          <p className="mt-1 text-sm font-bold font-mono">{data?.auto_verify_schedule ?? "0 0 * * *"}</p>
        </div>
      </div>

      {/* Alert config */}
      <div className={"rounded-xl p-4 flex items-center gap-3 " + (
        data?.alert_on_failure
          ? "bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800"
          : "bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-800"
      )}>
        <AlertTriangle className={"h-5 w-5 " + (data?.alert_on_failure ? "text-green-500" : "text-gray-400")} />
        <span className="text-sm text-gray-600 dark:text-gray-300">
          {data?.alert_on_failure ? "Alert on failure: Enabled — security team will be notified on chain integrity failure" : "Alert on failure: Disabled"}
        </span>
      </div>

      {/* Verification log */}
      <div className={card}>
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> Verification Log</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/50">
              <tr className="border-b dark:border-gray-700 text-gray-400">
                <th scope="col" className="text-left px-3 py-2 text-xs font-medium">Run ID</th>
                <th scope="col" className="text-left px-3 py-2 text-xs font-medium">Timestamp</th>
                <th scope="col" className="text-left px-3 py-2 text-xs font-medium">Result</th>
                <th scope="col" className="text-left px-3 py-2 text-xs font-medium">Duration</th>
                <th scope="col" className="text-left px-3 py-2 text-xs font-medium">Anomalies</th>
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-gray-700">
              {(data?.verification_log ?? []).map((v) => (
                <tr key={v.run_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-3 py-3 font-mono text-xs text-indigo-500">{v.run_id}</td>
                  <td className="px-3 py-3 text-gray-400 text-xs">{v.timestamp}</td>
                  <td className="px-3 py-3">
                    <span className={"flex items-center gap-1 text-xs " + (
                      v.result === "pass" ? "text-green-600" :
                      v.result === "fail" ? "text-red-600" : "text-yellow-600"
                    )}>
                      {v.result === "pass" ? <CheckCircle className="w-3 h-3" /> :
                       v.result === "fail" ? <XCircle className="w-3 h-3" /> :
                       <AlertTriangle className="w-3 h-3" />}
                      {v.result.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-3 py-3 text-gray-400 text-xs">{v.duration_ms}ms</td>
                  <td className="px-3 py-3">
                    <span className={"text-xs " + (v.anomalies_found > 0 ? "text-red-600 font-bold" : "text-gray-400")}>
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
