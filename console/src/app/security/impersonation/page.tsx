"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  UserRound, ArrowRight, Filter, Loader2, AlertCircle, X, Clock,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Entry {
  id: string;
  impersonator_name: string;
  target_name: string;
  started_at: string;
  ended_at: string | null;
  duration_seconds: number;
  ip_address: string;
  reason: string;
  actions_taken: number;
}

export default function ImpersonationLogPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [entries, setEntries] = useState<Entry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterImp, setFilterImp] = useState("");
  const [filterTarget, setFilterTarget] = useState("");

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (filterImp) params.set("impersonator", filterImp);
      if (filterTarget) params.set("target", filterTarget);
      const data = await apiFetch<{ entries?: Entry[]; items?: Entry[] }>(`/api/v1/audit/impersonation?${params.toString()}`).catch(() => null);
      setEntries(data?.entries ?? data?.items ?? []);
    } catch {
      setError("Failed to load impersonation log");
    } finally {
      setLoading(false);
    }
  };

  useState(() => { load(); });

  const formatDuration = (s: number) => s < 60 ? `${s}s` : s < 3600 ? `${Math.floor(s / 60)}m` : `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <UserRound className="h-6 w-6 text-indigo-600" /> Impersonation Log
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Audit trail of all user impersonation sessions.</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-3">
        <input value={filterImp} onChange={(e) => setFilterImp(e.target.value)} placeholder="Filter impersonator..." className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
        <input value={filterTarget} onChange={(e) => setFilterTarget(e.target.value)} placeholder="Filter target..." className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
        <button onClick={load} className="flex items-center gap-1.5 rounded-lg bg-indigo-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-indigo-700"><Filter className="h-4 w-4" />Apply</button>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : entries.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><UserRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No impersonation sessions recorded.</p></div></div>
      ) : (
        <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800"><tr className="text-left text-xs font-semibold uppercase text-gray-500">
              <th scope="col" className="px-4 py-3">Impersonator</th><th className="px-4 py-3"></th><th className="px-4 py-3">Target</th><th className="px-4 py-3">Started</th><th className="px-4 py-3">Duration</th><th className="px-4 py-3">Actions</th><th className="px-4 py-3">IP</th>
            </tr></thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {entries.map((e) => (
                <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                  <td className="px-4 py-3 font-medium text-gray-800 dark:text-gray-200">{e.impersonator_name}</td>
                  <td className="px-4 py-3"><ArrowRight className="h-4 w-4 text-gray-300" /></td>
                  <td className="px-4 py-3 font-medium text-indigo-600">{e.target_name}</td>
                  <td className="px-4 py-3 text-gray-500"><span className="flex items-center gap-1"><Clock className="h-3 w-3" />{new Date(e.started_at).toLocaleString()}</span></td>
                  <td className="px-4 py-3 text-gray-500">{e.ended_at ? formatDuration(e.duration_seconds) : <span className="text-green-600">Active</span>}</td>
                  <td className="px-4 py-3 text-gray-400">{e.actions_taken}</td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-400">{e.ip_address}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Mobile */}
      {!loading && entries.length > 0 && (
        <div className="space-y-3 md:hidden">
          {entries.map((e) => (
            <div key={e.id} className={cardCls}>
              <div className="flex items-center gap-2 text-sm">
                <span className="font-medium text-gray-800 dark:text-gray-200">{e.impersonator_name}</span>
                <ArrowRight className="h-3 w-3 text-gray-400" />
                <span className="font-medium text-indigo-600">{e.target_name}</span>
              </div>
              <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                <span>{new Date(e.started_at).toLocaleString()}</span>
                <span>{e.ended_at ? formatDuration(e.duration_seconds) : "Active"}</span>
                <span>{e.actions_taken} actions</span>
              </div>
              <p className="mt-0.5 font-mono text-xs text-gray-400">IP: {e.ip_address}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
