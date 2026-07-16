"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldOff, Plus, Trash2, X, Save, AlertTriangle, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ReviewExemption {
  id: string;
  role: string;
  reason: string;
  exempted_by: string;
  exempted_at: string;
  expires_at: string;
  days_remaining: number;
  status: "active" | "expired" | "revoked";
}

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  expired: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  revoked: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ReviewExemptionsPage() {
  const t = useTranslations();

  const [exemptions, setExemptions] = useState<ReviewExemption[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [revokeId, setRevokeId] = useState<string | null>(null);
  const [form, setForm] = useState({ role: "", reason: "", expires_at: "" });

  const fetchExemptions = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/review-exemptions", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setExemptions(data.exemptions || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchExemptions(); }, [fetchExemptions]);

  const createExemption = async () => {
    if (!form.role) return;
    try {
      await fetch("/api/v1/policy/review-exemptions", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(form),
      });
      setShowCreate(false);
      setForm({ role: "", reason: "", expires_at: "" });
      fetchExemptions();
    } catch { /* noop */ }
  };

  const doRevoke = async () => {
    if (!revokeId) return;
    try {
      await fetch(`/api/v1/policy/review-exemptions/${revokeId}`, {
        method: "DELETE",
        headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      setExemptions((prev) => prev.filter((e) => e.id !== revokeId));
      setRevokeId(null);
    } catch { /* noop */ }
  };

  const active = exemptions.filter((e) => e.status === "active");
  const expiring = active.filter((e) => e.days_remaining <= 7);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldOff className="w-6 h-6 text-orange-500" /> {t("reviewExemptions.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Manage roles exempted from access review campaigns.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> New Exemption</button>
      </div>

      {/* Expiring alert */}
      {expiring.length > 0 && (
        <div className="rounded-lg border border-orange-200 dark:border-orange-900 bg-orange-50 dark:bg-orange-900/20 p-4">
          <div className="flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /><span className="font-semibold text-orange-700 dark:text-orange-400">{expiring.length} exemption{expiring.length > 1 ? "s" : ""} expiring within 7 days</span></div>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total</span><p className="text-2xl font-bold mt-1">{exemptions.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Active</span><p className="text-2xl font-bold mt-1 text-green-600">{active.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Expired</span><p className="text-2xl font-bold mt-1 text-gray-400">{exemptions.filter((e) => e.status === "expired").length}</p></div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th className="px-4 py-3 text-left font-medium">Role</th>
              <th className="px-4 py-3 text-left font-medium">Reason</th>
              <th className="px-4 py-3 text-left font-medium">Exempted By</th>
              <th className="px-4 py-3 text-left font-medium">Expires</th>
              <th className="px-4 py-3 text-left font-medium">Days Left</th>
              <th className="px-4 py-3 text-left font-medium">Status</th>
              <th className="px-4 py-3 text-left font-medium">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {exemptions.map((e) => (
              <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs font-medium">{e.role}</td>
                <td className="px-4 py-3 max-w-xs truncate" title={e.reason}>{e.reason || "-"}</td>
                <td className="px-4 py-3">{e.exempted_by}</td>
                <td className="px-4 py-3 text-gray-500">{e.expires_at}</td>
                <td className="px-4 py-3">
                  {e.status === "expired" ? <span className="text-gray-400">-</span> :
                   <span className={`font-bold flex items-center gap-1 text-xs ${e.days_remaining <= 3 ? "text-red-600" : e.days_remaining <= 7 ? "text-orange-600" : "text-gray-500"}`}><Clock className="w-3 h-3" />{e.days_remaining}d</span>}
                </td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[e.status]}`}>{e.status}</span></td>
                <td className="px-4 py-3">
                  {e.status === "active" && <button onClick={() => setRevokeId(e.id)} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Trash2 className="w-3 h-3" /> Revoke</button>}
                </td>
              </tr>
            ))}
            {exemptions.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No exemptions found.</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold">New Review Exemption</h3>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Role</label><input type="text" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} placeholder="admin" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Reason</label><textarea value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} rows={3} placeholder="Service account requiring continuous privileged access" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Expires At</label><input type="date" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={createExemption} disabled={!form.role} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke confirmation */}
      {revokeId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setRevokeId(null)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="px-6 py-4"><p className="text-sm">Revoke this exemption? The role will be included in future access review campaigns.</p></div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setRevokeId(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={doRevoke} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700">Revoke</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
