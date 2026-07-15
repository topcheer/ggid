"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  UserCog,
  Loader2,
  CheckCircle2,
  XCircle,
  Clock,
  Filter,
  Download,
  AlertCircle,
  X,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ImpersonationRecord {
  id: string;
  impersonator_id: string;
  impersonator_name: string;
  impersonated_id: string;
  impersonated_name: string;
  reason: string;
  started_at: string;
  ended_at?: string;
  duration_seconds?: number;
  ip_address: string;
  user_agent: string;
  status: "active" | "completed" | "terminated";
}

export default function ImpersonationLogPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [records, setRecords] = useState<ImpersonationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = statusFilter ? `?status=${statusFilter}` : "";
      const data = await apiFetch<{ records?: ImpersonationRecord[] }>(`/api/v1/audit/impersonation${params}`).catch(() => ({ records: [] }));
      setRecords(data.records ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load impersonation records");
    } finally {
      setLoading(false);
    }
  }, [apiFetch, statusFilter]);

  useEffect(() => { load(); }, [load]);

  const handleExport = () => {
    const csv = [
      "ID,Impersonator,Impersonated,Reason,Started,Ended,Duration(s),IP,Status",
      ...records.map((r) => `${r.id},${r.impersonator_name},${r.impersonated_name},${r.reason},${r.started_at},${r.ended_at ?? ""},${r.duration_seconds ?? ""},${r.ip_address},${r.status}`),
    ].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `impersonation-log-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds) return "—";
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filtered = statusFilter ? records.filter((r) => r.status === statusFilter) : records;

  const statusIcon = (status: string) => {
    if (status === "active") return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    if (status === "terminated") return <XCircle className="h-4 w-4 text-red-500" />;
    return <Clock className="h-4 w-4 text-gray-400" />;
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <UserCog className="h-7 w-7 text-indigo-600" /> Impersonation Log
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Audit trail of all user impersonation sessions.
          </p>
        </div>
        <button onClick={handleExport} aria-label="Export impersonation log as CSV" className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
          <Download className="mr-1 inline h-4 w-4" /> Export
        </button>
      </div>

      {/* Filter */}
      <div className="flex items-center gap-3">
        <Filter className="h-4 w-4 text-gray-400" />
        <select
          aria-label="Filter by status"
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
        >
          <option value="">All Statuses</option>
          <option value="active">Active</option>
          <option value="completed">Completed</option>
          <option value="terminated">Terminated</option>
        </select>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : filtered.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <UserCog className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">No impersonation records found.</p>
        </div>
      ) : (
        <div className={`${cardCls} overflow-hidden p-0`}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs uppercase text-gray-400 dark:border-gray-700">
                  <th className="px-4 py-3">Impersonator</th>
                  <th className="px-4 py-3">Impersonated</th>
                  <th className="px-4 py-3">Reason</th>
                  <th className="px-4 py-3">Started</th>
                  <th className="px-4 py-3">Duration</th>
                  <th className="px-4 py-3">IP</th>
                  <th className="px-4 py-3">Status</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((r) => (
                  <tr key={r.id} className="border-b border-gray-100 dark:border-gray-700/50">
                    <td className="px-4 py-3">
                      <div className="font-medium text-gray-800 dark:text-gray-200">{r.impersonator_name}</div>
                      <div className="text-xs text-gray-400">{r.impersonator_id.slice(0, 12)}</div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="font-medium text-gray-800 dark:text-gray-200">{r.impersonated_name}</div>
                      <div className="text-xs text-gray-400">{r.impersonated_id.slice(0, 12)}</div>
                    </td>
                    <td className="px-4 py-3 text-gray-600 dark:text-gray-300">{r.reason || "—"}</td>
                    <td className="px-4 py-3 text-xs text-gray-400">{new Date(r.started_at).toLocaleString()}</td>
                    <td className="px-4 py-3 text-gray-600 dark:text-gray-300">{formatDuration(r.duration_seconds)}</td>
                    <td className="px-4 py-3 text-xs text-gray-400">{r.ip_address}</td>
                    <td className="px-4 py-3">
                      <span className="flex items-center gap-1 text-xs capitalize text-gray-600 dark:text-gray-300">
                        {statusIcon(r.status)} {r.status}
                      </span>
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
