"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  ScanSearch, Loader2, AlertCircle, X, Play, Database, ShieldCheck,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PIIFinding {
  id: string;
  field: string;
  entity: string;
  entity_type: string;
  pii_type: string;
  severity: "low" | "medium" | "high" | "critical";
  count: number;
  sample: string;
  discovered_at: string;
}

interface ScanResult {
  scan_id: string;
  status: string;
  started_at: string;
  completed_at: string;
  total_findings: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  findings: PIIFinding[];
}

const severityColors: Record<string, string> = {
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function PIIScanPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [result, setResult] = useState<ScanResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);

  useState(() => {
    (async () => {
      try { setResult(await apiFetch<ScanResult>("/api/v1/audit/pii-scan/results").catch(() => null)); }
      catch { setError("Failed to load PII scan data"); }
      finally { setLoading(false); }
    })();
  });

  const handleScan = async () => {
    setScanning(true);
    try {
      await apiFetch("/api/v1/audit/pii-scan/run", { method: "POST" });
      setResult(await apiFetch<ScanResult>("/api/v1/audit/pii-scan/results").catch(() => null));
    } catch { setError("Scan failed to start"); }
    finally { setScanning(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ScanSearch className="h-6 w-6 text-purple-600" /> {t("auditPiiScan.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Automated discovery of personally identifiable information across all data stores.</p>
        </div>
        <button onClick={handleScan} disabled={scanning} className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} Run Scan</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-600" /></div>
      : result ? (
        <>
          {/* Severity cards */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Critical</span></div><p className="mt-2 text-2xl font-bold text-red-600">{result.critical}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">High</span></div><p className="mt-2 text-2xl font-bold text-orange-600">{result.high}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">Medium</span></div><p className="mt-2 text-2xl font-bold text-yellow-600">{result.medium}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertCircle className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Low</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{result.low}</p></div>
          </div>

          {/* Scan info */}
          <div className={cardCls}>
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2"><Database className="h-4 w-4 text-gray-400" /><span className="text-gray-500">Scan ID: <span className="font-mono text-gray-700 dark:text-gray-300">{result.scan_id.slice(0, 12)}</span></span></div>
              <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${result.status === "completed" ? "text-green-600 bg-green-100 dark:bg-green-900/30" : "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30"}`}>{result.status}</span>
            </div>
            <div className="mt-2 flex gap-6 text-xs text-gray-400">
              <span>Started: {new Date(result.started_at).toLocaleString()}</span>
              {result.completed_at && <span>Completed: {new Date(result.completed_at).toLocaleString()}</span>}
              <span>Total findings: {result.total_findings}</span>
            </div>
          </div>

          {/* Findings table */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Findings</h2>
            {result.findings.length === 0 ? (
              <div className={cardCls}><div className="py-12 text-center"><ShieldCheck className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">No PII findings detected.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Field</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Entity</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">PII Type</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Severity</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Count</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Sample</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {result.findings.map((f) => (
                      <tr key={f.id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{f.field}</td>
                        <td className="px-4 py-3"><span className="text-gray-900 dark:text-white">{f.entity}</span><span className="block text-xs text-gray-400">{f.entity_type}</span></td>
                        <td className="px-4 py-3 text-gray-500">{f.pii_type}</td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${severityColors[f.severity] || ""}`}>{f.severity}</span></td>
                        <td className="px-4 py-3 font-bold text-gray-700 dark:text-gray-300">{f.count}</td>
                        <td className="px-4 py-3 font-mono text-xs text-gray-400">{f.sample}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      ) : <div className={cardCls}><div className="py-12 text-center"><ScanSearch className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No scan data yet. Click "Run Scan" to start.</p></div></div>}
    </div>
  );
}
