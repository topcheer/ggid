"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  Trash2, Loader2, AlertCircle, X, Search, AlertOctagon, CheckCircle, History,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SearchResult { user_id: string; username: string; email: string; record_count: number; }
interface ForgetRecord {
  id: string; user_id: string; username: string; email: string;
  status: "pending" | "processing" | "completed" | "failed";
  requested_by: string; requested_at: string; completed_at: string;
  records_deleted: number; errors: string[];
}

const statusColors: Record<string, string> = {
  completed: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  pending: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  processing: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  failed: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function GDPRForgetPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [query, setQuery] = useState("");
  const [searchResult, setSearchResult] = useState<SearchResult | null>(null);
  const [searching, setSearching] = useState(false);
  const [history, setHistory] = useState<ForgetRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [confirmUser, setConfirmUser] = useState<SearchResult | null>(null);
  const [executing, setExecuting] = useState(false);
  const [result, setResult] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setHistory(await apiFetch<ForgetRecord[]>("/api/v1/audit/gdpr-forget").catch(() => [])); }
      catch { setError("Failed to load history"); }
      finally { setLoading(false); }
    })();
  });

  const handleSearch = async () => {
    if (!query.trim()) return;
    setSearching(true); setSearchResult(null); setError(null);
    try { setSearchResult(await apiFetch<SearchResult>(`/api/v1/audit/gdpr-forget/search?q=${encodeURIComponent(query)}`)); }
    catch { setError("Search failed"); }
    finally { setSearching(false); }
  };

  const handleExecute = async () => {
    if (!confirmUser) return;
    setExecuting(true);
    try {
      await apiFetch("/api/v1/audit/gdpr-forget/execute", { method: "POST", body: JSON.stringify({ user_id: confirmUser.user_id }) });
      setResult("GDPR forget completed successfully");
      setHistory(await apiFetch<ForgetRecord[]>("/api/v1/audit/gdpr-forget").catch(() => history));
      setConfirmUser(null); setSearchResult(null);
    } catch { setResult("Execution failed"); }
    finally { setExecuting(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Trash2 className="h-6 w-6 text-red-600" /> {t("auditGdprForget.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Permanently delete all PII for a user. This action is irreversible.</p>
      </div>

      {/* Warning */}
      <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-900/20"><AlertOctagon className="h-5 w-5 text-red-600 shrink-0" /><p className="text-sm text-red-700 dark:text-red-400">This permanently deletes all user data including audit logs. This action cannot be undone.</p></div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Search */}
      <div className={cardCls}>
        <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Search User</h3>
        <div className="flex items-center gap-2">
          <div className="relative flex-1"><Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" /><input value={query} onChange={(e) => setQuery(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleSearch()} placeholder="Username, email, or user ID" className="w-full rounded-lg border border-gray-300 py-2 pl-10 pr-3 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
          <button onClick={handleSearch} disabled={!query.trim() || searching} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{searching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />} Search</button>
        </div>
        {searchResult && (
          <div className="mt-4 rounded-lg border border-gray-200 p-4 dark:border-gray-700">
            <div className="flex items-center justify-between">
              <div><div className="font-medium text-gray-900 dark:text-white">{searchResult.username}</div><div className="text-sm text-gray-400">{searchResult.email}</div><div className="mt-1 text-xs text-gray-500">{searchResult.record_count} records found</div></div>
              <button onClick={() => setConfirmUser(searchResult)} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Trash2 className="h-4 w-4" /> Delete</button>
            </div>
          </div>
        )}
        {result && <p className={`mt-3 flex items-center gap-2 text-sm ${result.includes("success") ? "text-green-600" : "text-red-600"}`}><CheckCircle className="h-4 w-4" />{result}</p>}
      </div>

      {/* Confirm modal */}
      {confirmUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmUser(null)}>
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="flex items-center gap-2 text-lg font-bold text-red-700 dark:text-red-400"><AlertOctagon className="h-5 w-5" /> Confirm Deletion</h3><button onClick={() => setConfirmUser(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <p className="mb-4 text-sm text-gray-600 dark:text-gray-300">You are about to permanently delete ALL data for <span className="font-bold text-red-600">{confirmUser.username}</span> ({confirmUser.email}). This includes {confirmUser.record_count} records and all associated audit logs.</p>
            <div className="rounded-lg bg-red-50 p-3 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-400">Type the username to confirm: {confirmUser.username}</div>
            <button onClick={handleExecute} disabled={executing} className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-red-600 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{executing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />} Confirm Permanent Deletion</button>
          </div>
        </div>
      )}

      {/* History */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><History className="h-4 w-4" /> Execution History</h2>
        {loading ? <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-red-600" /></div>
        : history.length === 0 ? <div className={cardCls}><div className="py-8 text-center text-sm text-gray-400">No deletion history.</div></div>
        : (
          <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800"><tr><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">User</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Records Deleted</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Requested</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Completed</th></tr></thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">{history.map((r) => (<tr key={r.id} className="bg-white dark:bg-gray-900"><td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{r.username}</div><div className="text-xs text-gray-400">{r.email}</div></td><td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[r.status] || ""}`}>{r.status}</span></td><td className="px-4 py-3 text-gray-500">{r.records_deleted}</td><td className="px-4 py-3 text-gray-400">{new Date(r.requested_at).toLocaleString()}</td><td className="px-4 py-3 text-gray-400">{r.completed_at ? new Date(r.completed_at).toLocaleString() : "—"}</td></tr>))}</tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
