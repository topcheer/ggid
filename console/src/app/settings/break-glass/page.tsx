"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  AlertTriangle, Loader2, AlertCircle, X, Plus, CheckCircle, Clock, Bell, Lock,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface BreakGlassRequest {
  id: string;
  requester: string;
  requester_name: string;
  reason: string;
  scope: string;
  duration_minutes: number;
  status: "pending" | "approved" | "active" | "expired" | "denied" | "completed";
  requested_at: string;
  approved_by: string;
  approved_at: string;
  expires_at: string;
  completed_at: string;
  notifications_sent: number;
}

const statusColors: Record<string, string> = {
  pending: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  approved: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  active: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  expired: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
  denied: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  completed: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function BreakGlassPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [requests, setRequests] = useState<BreakGlassRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showRequest, setShowRequest] = useState(false);
  const [form, setForm] = useState({ reason: "", scope: "", duration_minutes: 60 });

  useState(() => {
    (async () => {
      try { setRequests(await apiFetch<BreakGlassRequest[]>("/api/v1/auth/break-glass").catch(() => [])); }
      catch { setError("Failed to load break-glass data"); }
      finally { setLoading(false); }
    })();
  });

  const handleRequest = async () => {
    try {
      const created = await apiFetch<BreakGlassRequest>("/api/v1/auth/break-glass", { method: "POST", body: JSON.stringify(form) });
      setRequests((p) => [created, ...p]);
      setShowRequest(false);
      setForm({ reason: "", scope: "", duration_minutes: 60 });
    } catch { setError("Request failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const activeReqs = requests.filter((r) => r.status === "active");
  const history = requests.filter((r) => r.status !== "active");
  const pendingCount = requests.filter((r) => r.status === "pending").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><AlertTriangle className="h-6 w-6 text-red-600" /> {t("breakGlass.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Emergency privileged access with full audit trail and admin notification.</p>
        </div>
        <button onClick={() => setShowRequest(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Plus className="h-4 w-4" /> Request Access</button>
      </div>

      {/* Warning banner */}
      <div className="flex items-center gap-3 rounded-xl border border-orange-200 bg-orange-50 px-4 py-3 dark:border-orange-800 dark:bg-orange-900/20"><AlertTriangle className="h-5 w-5 text-orange-600 shrink-0" /><p className="text-sm text-orange-700 dark:text-orange-400">Break-glass access grants temporary elevated privileges. All actions are fully audited and admin notifications are sent automatically.</p></div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Lock className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active</span></div><p className="mt-2 text-2xl font-bold text-red-600">{activeReqs.length}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Clock className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">Pending</span></div><p className="mt-2 text-2xl font-bold text-yellow-600">{pendingCount}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><CheckCircle className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{requests.length}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Bell className="h-4 w-4 text-purple-500" /><span className="text-xs font-semibold uppercase text-gray-400">Notified</span></div><p className="mt-2 text-2xl font-bold text-purple-600">{requests.reduce((s, r) => s + r.notifications_sent, 0)}</p></div>
          </div>

          {/* Active emergency access */}
          {activeReqs.length > 0 && (
            <div>
              <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-red-500"><AlertTriangle className="h-4 w-4" /> Active Emergency Access</h2>
              <div className="space-y-2">
                {activeReqs.map((r) => {
                  const remaining = r.expires_at ? Math.max(0, new Date(r.expires_at).getTime() - Date.now()) : 0;
                  const mins = Math.floor(remaining / 60000);
                  return (
                    <div key={r.id} className={`${cardCls} border-red-300 dark:border-red-800`}>
                      <div className="flex items-center justify-between">
                        <div><div className="flex items-center gap-2"><span className="font-medium text-gray-900 dark:text-white">{r.requester_name || r.requester.slice(0, 12)}</span><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[r.status]}`}>{r.status}</span></div><p className="mt-1 text-sm text-gray-500">{r.reason}</p><div className="mt-1 flex gap-4 text-xs text-gray-400"><span>Scope: {r.scope}</span><span className={`font-medium ${mins < 10 ? "text-red-500" : ""}`}>{mins} min remaining</span></div></div>
                        <Bell className="h-5 w-5 text-orange-400" />
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* History */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">History</h2>
            {history.length === 0 ? (
              <div className={cardCls}><div className="py-8 text-center"><AlertTriangle className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No past break-glass requests.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Requester</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Reason</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Scope</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Duration</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Requested</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Notified</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {history.map((r) => (
                      <tr key={r.id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{r.requester_name || r.requester.slice(0, 12)}</div></td>
                        <td className="px-4 py-3 text-gray-500">{r.reason}</td>
                        <td className="px-4 py-3 text-gray-500">{r.scope}</td>
                        <td className="px-4 py-3 text-gray-500">{r.duration_minutes}m</td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[r.status] || ""}`}>{r.status}</span></td>
                        <td className="px-4 py-3 text-gray-400">{new Date(r.requested_at).toLocaleString()}</td>
                        <td className="px-4 py-3 text-gray-400">{r.notifications_sent} admins</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}

      {/* Request modal */}
      {showRequest && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRequest(false)}>
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="flex items-center gap-2 text-lg font-bold text-gray-900 dark:text-white"><AlertTriangle className="h-5 w-5 text-red-600" /> Request Emergency Access</h3><button onClick={() => setShowRequest(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Reason for Emergency Access</label><textarea value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} rows={3} placeholder="Describe the emergency requiring elevated access..." className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Scope</label><input value={form.scope} onChange={(e) => setForm({ ...form, scope: e.target.value })} placeholder="e.g. prod-db, all-systems" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div className="w-32"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Duration (min)</label><input type="number" value={form.duration_minutes} onChange={(e) => setForm({ ...form, duration_minutes: parseInt(e.target.value) || 60 })} min={15} max={480} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              </div>
              <div className="flex items-center gap-2 rounded-lg bg-orange-50 px-3 py-2 text-xs text-orange-700 dark:bg-orange-900/20 dark:text-orange-400"><Bell className="h-3 w-3" /> All admins will be automatically notified upon submission.</div>
              <button onClick={handleRequest} disabled={!form.reason.trim() || !form.scope} className="flex w-full items-center justify-center gap-2 rounded-lg bg-red-600 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"><AlertTriangle className="h-4 w-4" /> Submit Emergency Request</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
