"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  History, Loader2, AlertCircle, X, Calendar, Filter,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface BreakGlassHistory {
  id: string;
  requester: string;
  requester_name: string;
  reason: string;
  scope: string;
  duration_minutes: number;
  activated_at: string;
  deactivated_at: string;
  approver: string;
  approver_name: string;
  status: "completed" | "expired" | "revoked";
}

const statusColors: Record<string, string> = {
  completed: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  expired: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
  revoked: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function BreakGlassHistoryPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [records, setRecords] = useState<BreakGlassHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [filtering, setFiltering] = useState(false);

  useState(() => {
    (async () => {
      try { setRecords(await apiFetch<BreakGlassHistory[]>("/api/v1/auth/break-glass/history").catch(() => [])); }
      catch { setError("Failed to load history"); }
      finally { setLoading(false); }
    })();
  });

  const handleFilter = async () => {
    setFiltering(true);
    try {
      const params = new URLSearchParams();
      if (startDate) params.set("start", startDate);
      if (endDate) params.set("end", endDate);
      const q = params.toString() ? `?${params.toString()}` : "";
      setRecords(await apiFetch<BreakGlassHistory[]>(`/api/v1/auth/break-glass/history${q}`).catch(() => []));
    } catch { setError("Filter failed"); }
    finally { setFiltering(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const totalMinutes = records.reduce((s, r) => s + r.duration_minutes, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><History className="h-6 w-6 text-purple-600" /> {t("breakGlassHistory.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Historical record of all emergency access activations.</p>
      </div>

      {/* Date filter */}
      <div className={`${cardCls} flex items-end gap-3`}>
        <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Start Date</label><input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
        <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">End Date</label><input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
        <button onClick={handleFilter} disabled={filtering} className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{filtering ? <Loader2 className="h-4 w-4 animate-spin" /> : <Filter className="h-4 w-4" />} Filter</button>
        {(startDate || endDate) && <button onClick={() => { setStartDate(""); setEndDate(""); handleFilter(); }} className="text-sm text-gray-500 hover:underline">Clear</button>}
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Calendar className="h-4 w-4 text-purple-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Activations</span></div><p className="mt-2 text-2xl font-bold text-purple-600">{records.length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Total Duration</div><p className="mt-2 text-2xl font-bold text-gray-700 dark:text-gray-200">{Math.floor(totalMinutes / 60)}h {totalMinutes % 60}m</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Unique Requesters</div><p className="mt-2 text-2xl font-bold text-gray-700 dark:text-gray-200">{new Set(records.map((r) => r.requester)).size}</p></div>
          </div>

          {/* History table */}
          {records.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><History className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No break-glass history.</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Requester</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Reason</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Scope</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Duration</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Activated</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Deactivated</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Approver</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {records.map((r) => (
                    <tr key={r.id} className="bg-white dark:bg-gray-900">
                      <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{r.requester_name || r.requester.slice(0, 12)}</div></td>
                      <td className="px-4 py-3 text-gray-500">{r.reason}</td>
                      <td className="px-4 py-3"><span className="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{r.scope}</span></td>
                      <td className="px-4 py-3 text-gray-500">{r.duration_minutes}m</td>
                      <td className="px-4 py-3 text-gray-400">{new Date(r.activated_at).toLocaleString()}</td>
                      <td className="px-4 py-3 text-gray-400">{r.deactivated_at ? new Date(r.deactivated_at).toLocaleString() : "—"}</td>
                      <td className="px-4 py-3 text-gray-500">{r.approver_name || r.approver.slice(0, 12)}</td>
                      <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[r.status] || ""}`}>{r.status}</span></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
