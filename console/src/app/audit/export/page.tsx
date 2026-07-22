"use client";

import { useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Download, Loader2, AlertCircle, X, Check, FileText, Filter,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuditExportPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [format, setFormat] = useState<"csv" | "json">("csv");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [eventType, setEventType] = useState("");
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleExport = useCallback(async () => {
    setExporting(true);
    setError(null);
    setSuccess(false);
    try {
      const params = new URLSearchParams();
      params.set("format", format);
      if (dateFrom) params.set("date_from", dateFrom);
      if (dateTo) params.set("date_to", dateTo);
      if (eventType) params.set("event_type", eventType);

      const resp = await apiFetch<Response>(`/api/v1/audit/export?${params.toString()}`).catch(() => null);

      // Fallback: try direct blob download
      let blob: Blob;
      if (resp && resp instanceof Response && resp.ok) {
        blob = await resp.blob();
      } else {
        // Use apiFetch which returns parsed JSON - for binary we need fetch directly
        const tenantId = typeof window !== "undefined" ? localStorage.getItem("ggid_tenant_id") || "" : "";
        const token = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") || "" : "";
        const apiBase = typeof window !== "undefined" ? window.location.origin : "";
        const r = await fetch(`${apiBase}/api/v1/audit/export?${params.toString()}`, {
          headers: { Authorization: `Bearer ${token}`, "X-Tenant-ID": tenantId },
        });
        if (!r.ok) throw new Error(`Export failed (${r.status})`);
        blob = await r.blob();
      }

      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      const ts = new Date().toISOString().split("T")[0];
      a.href = url;
      a.download = `audit-export-${ts}.${format}`;
      a.click();
      URL.revokeObjectURL(url);
      setSuccess(true);
    } catch {
      setError("Export failed. Check filters and try again.");
    } finally {
      setExporting(false);
    }
  }, [format, dateFrom, dateTo, eventType, apiFetch]);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Download className="h-6 w-6 text-indigo-600" /> Audit Export
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Export audit events with optional filters for compliance and analysis.</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {success && (
        <div role="status" className="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-400">
          <Check className="h-4 w-4 shrink-0" />Export downloaded successfully.
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Export form */}
        <div className="lg:col-span-2">
          <div className={cardCls}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><Filter className="h-4 w-4" /> Filters</h3>
            <div className="space-y-4">
              {/* Format */}
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Format</label>
                <div className="mt-2 flex gap-3">
                  {(["csv", "json"] as const).map((f: any) => (
                    <button key={f} onClick={() => setFormat(f)} className={`flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium ${format === f ? "border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400" : "border-gray-300 text-gray-500 dark:border-gray-600"}`}>
                      <FileText className="h-4 w-4" />{f.toUpperCase()}
                    </button>
                  ))}
                </div>
              </div>

              {/* Date range */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300">From Date</label>
                  <input aria-label="date From" type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700 dark:text-gray-300">To Date</label>
                  <input aria-label="date To" type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                </div>
              </div>

              {/* Event type */}
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Event Type Filter (optional)</label>
                <input aria-label="e.g. login, role.update, policy.change" value={eventType} onChange={(e) => setEventType(e.target.value)} placeholder="e.g. login, role.update, policy.change" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
            </div>

            <div className="mt-6 flex justify-end">
              <button onClick={handleExport} disabled={exporting} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {exporting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />} Export {format.toUpperCase()}
              </button>
            </div>
          </div>
        </div>

        {/* Info card */}
        <div>
          <div className={cardCls}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><FileText className="h-4 w-4" /> Export Details</h3>
            <div className="space-y-2 text-xs text-gray-400">
              <p><strong className="text-gray-600 dark:text-gray-300">CSV:</strong> Spreadsheet-compatible, one event per row.</p>
              <p><strong className="text-gray-600 dark:text-gray-300">JSON:</strong> Structured format for programmatic processing.</p>
              <p><strong className="text-gray-600 dark:text-gray-300">Date Range:</strong> Leave blank to export all events.</p>
              <p><strong className="text-gray-600 dark:text-gray-300">Event Type:</strong> Filter by event action prefix.</p>
            </div>
            <div className="mt-4 rounded-lg bg-amber-50 p-3 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">
              Large exports may take several minutes. The file downloads automatically when ready.
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
