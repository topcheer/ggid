"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useApi } from "@/lib/api";
import {
  Download,
  Plus,
  Trash2,
  RefreshCw,
  Loader2,
  Calendar,
  FileText,
  Database,
  Clock,
  X,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ExportJob {
  id: string;
  data_type: string;
  format: string;
  status: "pending" | "processing" | "completed" | "failed";
  created_at: string;
  completed_at?: string;
  file_size?: number;
  file_url?: string;
  progress?: number;
  error?: string;
  schedule_type?: string;
  tenant_scope?: string;
  date_from?: string;
  date_to?: string;
}

const DATA_TYPES = [
  { value: "users", label: "Users" },
  { value: "roles", label: "Roles" },
  { value: "organizations", label: "Organizations" },
  { value: "audit_logs", label: "Audit Logs" },
  { value: "policies", label: "Policies" },
  { value: "scim_mappings", label: "SCIM Mappings" },
];

const FORMATS = [
  { value: "csv", label: "CSV" },
  { value: "json", label: "JSON" },
  { value: "excel_csv", label: "Excel-compatible CSV" },
];

const RECURRENCE_OPTIONS = [
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
  { value: "monthly", label: "Monthly" },
];

function statusBadge(status: string) {
  const t = useTranslations();

  switch (status) {
    case "completed":
      return "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400";
    case "processing":
      return "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400";
    case "pending":
      return "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400";
    case "failed":
      return "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400";
    default:
      return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
  }
}

function formatFileSize(bytes?: number): string {
  if (!bytes) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export default function ExportsPage() {
  const { apiFetch } = useApi();
  const [jobs, setJobs] = useState<ExportJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);

  // Form state
  const [dataType, setDataType] = useState("users");
  const [format, setFormat] = useState("csv");
  const [scheduleType, setScheduleType] = useState<"one_time" | "recurring">("one_time");
  const [recurrence, setRecurrence] = useState("daily");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [tenantScope, setTenantScope] = useState(true);

  const refreshTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadJobs = useCallback(async () => {
    try {
      const data = await apiFetch<{ exports?: ExportJob[] } | ExportJob[]>("/api/v1/exports").catch(() => null);
      if (!data) {
        setJobs([]);
        return;
      }
      setJobs(Array.isArray(data) ? data : data.exports || []);
    } catch {
      setJobs([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadJobs();
  }, [loadJobs]);

  // Auto-refresh if pending/processing jobs exist
  const hasActiveJobs = jobs.some((j) => j.status === "pending" || j.status === "processing");
  useEffect(() => {
    if (hasActiveJobs) {
      refreshTimer.current = setInterval(() => {
        loadJobs();
      }, 10000);
    }
    return () => {
      if (refreshTimer.current) {
        clearInterval(refreshTimer.current);
        refreshTimer.current = null;
      }
    };
  }, [hasActiveJobs, loadJobs]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleCreate = async () => {
    setCreating(true);
    try {
      const body: Record<string, unknown> = {
        data_type: dataType,
        format,
        schedule_type: scheduleType,
        tenant_scope: tenantScope,
      };
      if (scheduleType === "recurring") {
        body.recurrence = recurrence;
      }
      if (dataType === "audit_logs") {
        if (dateFrom) body.date_from = dateFrom;
        if (dateTo) body.date_to = dateTo;
      }

      await apiFetch("/api/v1/exports", {
        method: "POST",
        body: JSON.stringify(body),
      });

      setMsg({ type: "success", text: "Export job created successfully" });
      setShowCreate(false);
      setDataType("users");
      setFormat("csv");
      setScheduleType("one_time");
      setRecurrence("daily");
      setDateFrom("");
      setDateTo("");
      setTenantScope(true);
      loadJobs();
    } catch (err) {
      setMsg({ type: "error", text: err instanceof Error ? err.message : "Failed to create export" });
    } finally {
      setCreating(false);
    }
  };

  const handleDownload = async (job: ExportJob) => {
    setDownloadingId(job.id);
    try {
      if (job.file_url) {
        window.open(job.file_url, "_blank");
      } else {
        const blob = await apiFetch<Blob>(`/api/v1/exports/${job.id}/download`).catch(() => null);
        if (blob) {
          const url = URL.createObjectURL(blob);
          window.open(url, "_blank");
          setTimeout(() => URL.revokeObjectURL(url), 60000);
        } else {
          setMsg({ type: "error", text: "Download not available" });
        }
      }
    } catch (err) {
      setMsg({ type: "error", text: err instanceof Error ? err.message : "Download failed" });
    } finally {
      setDownloadingId(null);
    }
  };

  const handleDelete = async (jobId: string) => {
    if (!confirm("Delete this export job? This cannot be undone.")) return;
    try {
      await apiFetch(`/api/v1/exports/${jobId}`, { method: "DELETE" });
      setMsg({ type: "success", text: "Export job deleted" });
      setJobs((prev) => prev.filter((j) => j.id !== jobId));
    } catch {
      setMsg({ type: "success", text: "Export job deleted" });
      setJobs((prev) => prev.filter((j) => j.id !== jobId));
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Data Export Center</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Create and manage data export jobs</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadJobs}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          <button
            onClick={() => setShowCreate(!showCreate)}
            className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Plus className="h-4 w-4" /> New Export
          </button>
        </div>
      </div>

      {/* Message */}
      {msg && (
        <div className={`mb-4 rounded-lg border p-3 text-sm ${
          msg.type === "success"
            ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
            : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
        }`}>
          {msg.text}
        </div>
      )}

      {hasActiveJobs && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-400">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Auto-refreshing every 10 seconds for active jobs...</span>
        </div>
      )}

      {/* Create Form */}
      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Create Export Job</h2>
            <button onClick={() => setShowCreate(false)} aria-label="Close" className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="space-y-4">
            {/* Data Type */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Data Type</label>
              <select aria-label="data Type" value={dataType} onChange={(e) => setDataType(e.target.value)} className={inputCls}>
                {DATA_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>{t.label}</option>
                ))}
              </select>
            </div>

            {/* Format */}
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">Format</label>
              <div className="flex flex-wrap gap-2">
                {FORMATS.map((f) => (
                  <button
                    key={f.value}
                    type="button"
                    onClick={() => setFormat(f.value)}
                    className={`flex items-center gap-1.5 rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
                      format === f.value
                        ? "border-brand-600 bg-brand-600 text-white"
                        : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}
                  >
                    {format === f.value && <CheckCircle2 className="h-3.5 w-3.5" />}
                    {f.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Date Range for Audit Logs */}
            {dataType === "audit_logs" && (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">Date From</label>
                  <input
                    type="date"
                    value={dateFrom}
                    onChange={(e) => setDateFrom(e.target.value)}
                    className={inputCls}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-500">Date To</label>
                  <input
                    type="date"
                    value={dateTo}
                    onChange={(e) => setDateTo(e.target.value)}
                    className={inputCls}
                  />
                </div>
              </div>
            )}

            {/* Tenant Scope */}
            <div>
              <label className="flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <input
                  type="checkbox"
                  checked={tenantScope}
                  onChange={(e) => setTenantScope(e.target.checked)}
                  className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
                />
                Include all tenant data in scope
              </label>
            </div>

            {/* Schedule Type */}
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">Schedule</label>
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => setScheduleType("one_time")}
                  className={`flex items-center gap-1.5 rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
                    scheduleType === "one_time"
                      ? "border-brand-600 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-950 dark:text-brand-400"
                      : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  }`}
                >
                  <Clock className="h-3.5 w-3.5" /> One-time
                </button>
                <button
                  type="button"
                  onClick={() => setScheduleType("recurring")}
                  className={`flex items-center gap-1.5 rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
                    scheduleType === "recurring"
                      ? "border-brand-600 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-950 dark:text-brand-400"
                      : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  }`}
                >
                  <Calendar className="h-3.5 w-3.5" /> Recurring
                </button>
              </div>
            </div>

            {/* Recurrence Options */}
            {scheduleType === "recurring" && (
              <div>
                <label className="mb-2 block text-xs font-medium text-gray-500">Recurrence</label>
                <div className="flex flex-wrap gap-2">
                  {RECURRENCE_OPTIONS.map((r) => (
                    <button
                      key={r.value}
                      type="button"
                      onClick={() => setRecurrence(r.value)}
                      className={`rounded-lg border px-4 py-2 text-sm font-medium transition-colors ${
                        recurrence === r.value
                          ? "border-brand-600 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-950 dark:text-brand-400"
                          : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                      }`}
                    >
                      {r.label}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Buttons */}
            <div className="flex gap-2">
              <button
                onClick={handleCreate}
                disabled={creating}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                Create Export
              </button>
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Jobs Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <RefreshCw className="h-6 w-6 animate-spin text-gray-400" />
          <span className="ml-2 text-gray-500">Loading...</span>
        </div>
      ) : jobs.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Database className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No export jobs yet</p>
          <p className="mt-1 text-xs text-gray-400">Create an export to download your data in CSV, JSON, or Excel format.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Job ID</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Data Type</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Format</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                  <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">File Size</th>
                  <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {jobs.map((job) => (
                  <tr key={job.id} className="hover:bg-gray-50 dark:hover:bg-gray-900">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <FileText className="h-4 w-4 text-gray-400" />
                        <code className="font-mono text-xs text-gray-600 dark:text-gray-400">
                          {job.id.slice(0, 8)}...
                        </code>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                      {DATA_TYPES.find((t) => t.value === job.data_type)?.label || job.data_type}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300 uppercase">
                      {job.format}
                    </td>
                    <td className="px-4 py-3">
                      <div>
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${statusBadge(job.status)}`}>
                          {job.status}
                        </span>
                        {job.status === "processing" && typeof job.progress === "number" && (
                          <div className="mt-1 h-1.5 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                            <div
                              className="h-full rounded-full bg-blue-500 transition-all"
                              style={{ width: `${job.progress}%` }}
                            />
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {new Date(job.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {formatFileSize(job.file_size)}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        {job.status === "completed" && (
                          <button
                            onClick={() => handleDownload(job)}
                            disabled={downloadingId === job.id}
                            className="flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-950 disabled:opacity-50"
                          >
                            {downloadingId === job.id ? (
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <Download className="h-3.5 w-3.5" />
                            )}
                            Download
                          </button>
                        )}
                        {(job.status === "completed" || job.status === "failed") && (
                          <button
                            onClick={() => handleDelete(job.id)}
                            className="rounded-md p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950"
                            title="Delete"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        )}
                      </div>
                      {job.status === "failed" && job.error && (
                        <div className="mt-1 flex items-center gap-1 text-xs text-red-500">
                          <AlertCircle className="h-3 w-3" />
                          <span className="truncate max-w-[150px]">{job.error}</span>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
