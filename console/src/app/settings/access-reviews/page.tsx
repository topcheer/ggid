"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ClipboardCheck, Plus, Trash2, X, AlertCircle, Loader2, Check,
  Clock, Users, Calendar,
} from "lucide-react";

interface Campaign {
  id: string;
  name: string;
  scope: string;
  reviewer: string;
  deadline: string;
  status: "pending" | "in_progress" | "completed" | "overdue";
  total_items: number;
  reviewed_items: number;
  created_at: string;
}

const STATUS_COLOR: Record<string, string> = {
  pending: "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400",
  in_progress: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  completed: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  overdue: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

export default function AccessReviewsPage() {
  const { apiFetch } = useApi();
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<Campaign | null>(null);
  const [form, setForm] = useState({ name: "", scope: "", reviewer: "", deadline: "" });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ campaigns?: Campaign[]; items?: Campaign[] }>("/api/v1/audit/access-reviews").catch(() => null);
      setCampaigns(data?.campaigns ?? data?.items ?? []);
    } catch {
      setError("Failed to load access review campaigns");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.name.trim()) return;
    setCreating(true);
    try {
      await apiFetch("/api/v1/audit/access-reviews", { method: "POST", body: JSON.stringify(form) });
      setForm({ name: "", scope: "", reviewer: "", deadline: "" });
      setShowCreate(false);
      await load();
    } catch {
      setError("Failed to create campaign");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/audit/access-reviews/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError("Failed to delete campaign");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ClipboardCheck className="h-6 w-6 text-indigo-600" /> Access Reviews
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Periodic access recertification campaigns for compliance.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Campaign</button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : campaigns.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><ClipboardCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No active campaigns.</p></div></div>
      ) : (
        <div className="space-y-3">
          {campaigns.map((c) => {
            const progress = c.total_items > 0 ? Math.round((c.reviewed_items / c.total_items) * 100) : 0;
            return (
              <div key={c.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-800 dark:text-gray-200">{c.name}</span>
                      <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLOR[c.status] ?? STATUS_COLOR.pending}`}>{c.status.replace("_", " ")}</span>
                    </div>
                    <div className="mt-2 flex items-center gap-4 text-xs text-gray-400">
                      <span className="flex items-center gap-1"><Users className="h-3 w-3" />Scope: {c.scope || "all"}</span>
                      <span className="flex items-center gap-1"><ClipboardCheck className="h-3 w-3" />Reviewer: {c.reviewer}</span>
                      <span className="flex items-center gap-1"><Calendar className="h-3 w-3" />Deadline: {new Date(c.deadline).toLocaleDateString()}</span>
                    </div>
                  </div>
                  <button onClick={() => setConfirmDelete(c)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                </div>
                {/* Progress bar */}
                <div className="mt-3">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-gray-400">{c.reviewed_items} / {c.total_items} reviewed</span>
                    <span className="font-medium text-indigo-600">{progress}%</span>
                  </div>
                  <div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                    <div className="h-full rounded-full bg-indigo-500 transition-all" style={{ width: `${progress}%` }} />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !creating && setShowCreate(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">New Campaign</h2>
              <button onClick={() => setShowCreate(false)}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Name</label><input value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} placeholder="Q3 Access Review" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Scope (org/unit)</label><input value={form.scope} onChange={(e) => setForm((p) => ({ ...p, scope: e.target.value }))} placeholder="engineering" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Reviewer</label><input value={form.reviewer} onChange={(e) => setForm((p) => ({ ...p, reviewer: e.target.value }))} placeholder="compliance@company.com" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Deadline</label><input type="date" value={form.deadline} onChange={(e) => setForm((p) => ({ ...p, deadline: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleCreate} disabled={!form.name.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><div><h2 className="font-semibold text-gray-900 dark:text-white">Delete Campaign?</h2><p className="text-sm text-gray-500"><strong>{confirmDelete.name}</strong> and all review items will be removed.</p></div></div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
