"use client";

import { useState, useEffect, useCallback } from "react";
import { UserCog, Plus, X, Clock, CheckCircle, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Delegation {
  id: string;
  original_reviewer: string;
  delegated_to: string;
  scope: string;
  created_at: string;
  expires_at: string;
  status: "active" | "expired" | "revoked";
}

const statusConfig: Record<string, { color: string; icon: typeof CheckCircle }> = {
  active: { color: "text-green-600", icon: CheckCircle },
  expired: { color: "text-gray-500", icon: Clock },
  revoked: { color: "text-red-600", icon: AlertTriangle },
};

export default function DelegatedReviewPage() {
  const t = useTranslations();

  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ original_reviewer: "", delegated_to: "", scope: "", expires_at: "" });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/delegated-review", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setDelegations(data.delegations || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const create = async () => {
    if (!form.original_reviewer || !form.delegated_to) return;
    try { await fetch("/api/v1/policy/delegated-review", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowCreate(false); setForm({ original_reviewer: "", delegated_to: "", scope: "", expires_at: "" }); fetchData(); }
    catch { /* noop */ }
  };

  const revoke = async (id: string) => {
    try { await fetch(`/api/v1/policy/delegated-review/${id}`, { method: "DELETE", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  const activeCount = delegations.filter((d) => d.status === "active").length;
  const expiringSoon = delegations.filter((d) => { if (d.status !== "active") return false; const days = (new Date(d.expires_at).getTime() - Date.now()) / 86400000; return days <= 3 && days >= 0; }).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><UserCog className="w-6 h-6 text-indigo-500" /> {t("delegatedReview.title")}</h1><p className="text-sm text-gray-500 mt-1">Delegate access review responsibilities to other users with scoped permissions.</p></div>
        <button onClick={() => setShowCreate(true)}aria-label="Create new delegation" className="px-4 py-2 rounded-lg bg-indigo-600 text-white text-sm font-medium hover:bg-indigo-700 flex items-center gap-2"><Plus className="w-4 h-4" /> New Delegation</button>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Active</span><p className="text-xl font-bold text-green-600 mt-1">{activeCount}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Expiring Soon</span><p className="text-xl font-bold text-orange-600 mt-1">{expiringSoon}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total</span><p className="text-xl font-bold mt-1">{delegations.length}</p></div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Original Reviewer</th><th className="px-4 py-3 text-left font-medium">Delegated To</th><th className="px-4 py-3 text-left font-medium">Scope</th><th className="px-4 py-3 text-left font-medium">Created</th><th className="px-4 py-3 text-left font-medium">Expires</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{delegations.map((d) => { const cfg = statusConfig[d.status]; const Icon = cfg.icon; const isExpiringSoon = d.status === "active" && (new Date(d.expires_at).getTime() - Date.now()) / 86400000 <= 3; return (<tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{d.original_reviewer}</td><td className="px-4 py-3 text-indigo-600 font-medium">{d.delegated_to}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{d.scope}</span></td><td className="px-4 py-3 text-xs text-gray-500">{d.created_at}</td><td className="px-4 py-3 text-xs">{isExpiringSoon ? <span className="text-orange-600 font-medium">{d.expires_at} (soon)</span> : <span className="text-gray-500">{d.expires_at}</span>}</td><td className="px-4 py-3"><span className={`flex items-center gap-1 text-xs ${cfg.color}`}><Icon className="w-3.5 h-3.5" /> {d.status}</span></td><td className="px-4 py-3">{d.status === "active" && <button onClick={() => revoke(d.id)}aria-label={"Revoke delegation " + d.id} className="text-xs text-red-600 hover:underline">Revoke</button>}</td></tr>); })}{delegations.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No delegations.</td></tr>}</tbody>
        </table>
      </div>

      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><UserCog className="w-5 h-5 text-indigo-500" /> New Delegation</h3><button onClick={() => setShowCreate(false)} aria-label="Close dialog"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Original Reviewer</label><input type="text" value={form.original_reviewer} onChange={(e) => setForm({ ...form, original_reviewer: e.target.value })} placeholder="reviewer@example.com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Delegate To</label><input type="text" value={form.delegated_to} onChange={(e) => setForm({ ...form, delegated_to: e.target.value })} placeholder="delegate@example.com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Scope</label><input type="text" value={form.scope} onChange={(e) => setForm({ ...form, scope: e.target.value })} placeholder="review:all" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Expires At</label><input type="datetime-local" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowCreate(false)} aria-label="Cancel" className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={create} disabled={!form.original_reviewer || !form.delegated_to} aria-label="Create delegation" className="px-4 py-2 rounded-lg bg-indigo-600 text-white text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">Create</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
