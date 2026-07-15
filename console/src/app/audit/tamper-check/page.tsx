"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, CheckCircle, AlertOctagon, ShieldAlert,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TamperStatus {
  last_scan: string;
  total_events: number;
  verified: number;
  issues_count: number;
  integrity_pct: number;
  status: "verified" | "warning" | "failed";
}

interface TamperIssue {
  id: string;
  event_id: string;
  type: "hash_mismatch" | "gap_detected" | "chain_broken" | "timestamp_anomaly";
  severity: "low" | "medium" | "high" | "critical";
  description: string;
  detected_at: string;
  event_timestamp: string;
}

const sevColors: Record<string, string> = {
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function TamperCheckPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [status, setStatus] = useState<TamperStatus | null>(null);
  const [issues, setIssues] = useState<TamperIssue[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);

  useState(() => {
    (async () => {
      try {
        const [s, i] = await Promise.all([
          apiFetch<TamperStatus>("/api/v1/audit/tamper-check/status").catch(() => null),
          apiFetch<TamperIssue[]>("/api/v1/audit/tamper-check/issues").catch(() => []),
        ]);
        setStatus(s); setIssues(i);
      } catch { setError("Failed to load tamper check data"); }
      finally { setLoading(false); }
    })();
  });

  const handleScan = async () => {
    setScanning(true); setError(null);
    try {
      await apiFetch("/api/v1/audit/tamper-check/scan", { method: "POST" });
      const [s, i] = await Promise.all([apiFetch<TamperStatus>("/api/v1/audit/tamper-check/status"), apiFetch<TamperIssue[]>("/api/v1/audit/tamper-check/issues").catch(() => [])]);
      setStatus(s); setIssues(i);
    } catch { setError("Scan failed"); }
    finally { setScanning(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-green-600" /> {t("auditTamperCheck.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Verify audit log integrity via hash chain verification.</p>
        </div>
        <button onClick={handleScan} disabled={scanning} className="flex items-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50">{scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />} Run Scan</button>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Integrity banner */}
      {status && (
        <div className={`flex items-center gap-3 rounded-xl border px-4 py-4 ${status.status === "verified" ? "border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20" : status.status === "warning" ? "border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-900/20" : "border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20"}`}>
          {status.status === "verified" ? <CheckCircle className="h-6 w-6 text-green-600" /> : status.status === "warning" ? <ShieldAlert className="h-6 w-6 text-yellow-600" /> : <AlertOctagon className="h-6 w-6 text-red-600" />}
          <div>
            <div className={`font-medium ${status.status === "verified" ? "text-green-700 dark:text-green-400" : status.status === "warning" ? "text-yellow-700 dark:text-yellow-400" : "text-red-700 dark:text-red-400"}`}>Integrity: {status.integrity_pct}% — {status.status}</div>
            <div className="text-sm text-gray-500">{status.verified} of {status.total_events} events verified · {status.issues_count} issue{status.issues_count !== 1 ? "s" : ""} · Last scan: {status.last_scan ? new Date(status.last_scan).toLocaleString() : "never"}</div>
          </div>
        </div>
      )}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-green-600" /></div>
      : (
        <>
          {/* Stats */}
          {status && (
            <div className="grid grid-cols-3 gap-4">
              <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Total Events</div><p className="mt-2 text-2xl font-bold text-gray-700 dark:text-gray-200">{status.total_events.toLocaleString()}</p></div>
              <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Verified</div><p className="mt-2 text-2xl font-bold text-green-600">{status.verified.toLocaleString()}</p></div>
              <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Issues</div><p className="mt-2 text-2xl font-bold text-red-600">{status.issues_count}</p></div>
            </div>
          )}

          {/* Issues table */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Integrity Issues</h2>
            {issues.length === 0 ? (
              <div className={cardCls}><div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No integrity issues found.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Type</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Severity</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Description</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Event Time</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Detected</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {issues.map((iss) => (
                      <tr key={iss.id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3"><span className="font-mono text-xs text-gray-700 dark:text-gray-300">{iss.type.replace(/_/g, " ")}</span></td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${sevColors[iss.severity] || ""}`}>{iss.severity}</span></td>
                        <td className="px-4 py-3 text-gray-500">{iss.description}</td>
                        <td className="px-4 py-3 text-gray-400">{iss.event_timestamp ? new Date(iss.event_timestamp).toLocaleString() : "—"}</td>
                        <td className="px-4 py-3 text-gray-400">{new Date(iss.detected_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
