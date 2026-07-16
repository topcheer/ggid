"use client";

import { useState, useEffect, useCallback } from "react";
import { Users, Plus, Trash2, X, Save, Calendar, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Delegation {
  id: string;
  delegator: string;
  delegated_to: string;
  scope: string[];
  start_date: string;
  end_date: string;
  status: "active" | "expired" | "revoked" | "pending";
  reason: string;
}

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  expired: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  revoked: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
};

export default function DelegationsPage() {
  const t = useTranslations();

  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [revokeId, setRevokeId] = useState<string | null>(null);
  const [form, setForm] = useState({ delegated_to: "", scope: "", start_date: "", end_date: "", reason: "" });

  const fetchDelegations = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/delegations", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setDelegations(data.delegations || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchDelegations(); }, [fetchDelegations]);

  const saveDelegation = async () => {
    const payload = {
      delegated_to: form.delegated_to,
      scope: form.scope.split(",").map((s) => s.trim()).filter(Boolean),
      start_date: form.start_date,
      end_date: form.end_date,
      reason: form.reason,
    };
    try {
      const url = editId ? `/api/v1/identity/delegations/${editId}` : "/api/v1/identity/delegations";
      const method = editId ? "PUT" : "POST";
      await fetch(url, {
        method,
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(payload),
      });
      setShowCreate(false);
      setEditId(null);
      setForm({ delegated_to: "", scope: "", start_date: "", end_date: "", reason: "" });
      fetchDelegations();
    } catch { /* noop */ }
  };

  const startEdit = (d: Delegation) => {
    setEditId(d.id);
    setForm({ delegated_to: d.delegated_to, scope: d.scope.join(", "), start_date: d.start_date, end_date: d.end_date, reason: d.reason });
    setShowCreate(true);
  };

  const doRevoke = async () => {
    if (!revokeId) return;
    try {
      await fetch(`/api/v1/identity/delegations/${revokeId}/revoke`, {
        method: "POST",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      setDelegations((prev) => prev.map((d) => d.id === revokeId ? { ...d, status: "revoked" } : d));
      setRevokeId(null);
    } catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-blue-500" /> {t("delegations.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Manage access delegations between users with scope and time limits.</p>
        </div>
        <button onClick={() => { setEditId(null); setForm({ delegated_to: "", scope: "", start_date: "", end_date: "", reason: "" }); setShowCreate(true); }} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> New Delegation</button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total</span><p className="text-2xl font-bold mt-1">{delegations.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Active</span><p className="text-2xl font-bold mt-1 text-green-600">{delegations.filter((d) => d.status === "active").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Pending</span><p className="text-2xl font-bold mt-1 text-yellow-600">{delegations.filter((d) => d.status === "pending").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Revoked</span><p className="text-2xl font-bold mt-1 text-red-600">{delegations.filter((d) => d.status === "revoked").length}</p></div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">Delegated To</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Scope</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Start</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">End</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {delegations.map((d) => (
              <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3"><span className="font-medium">{d.delegated_to}</span><p className="text-xs text-gray-400">from {d.delegator}</p></td>
                <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{d.scope.map((s, i) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{s}</span>)}</div></td>
                <td className="px-4 py-3 text-gray-500">{d.start_date}</td>
                <td className="px-4 py-3 text-gray-500">{d.end_date}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[d.status]}`}>{d.status}</span></td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <button onClick={() => startEdit(d)} className="text-xs text-blue-600 hover:underline">Edit</button>
                    {d.status !== "revoked" && d.status !== "expired" && (
                      <button onClick={() => setRevokeId(d.id)} className="text-xs text-red-600 hover:underline">Revoke</button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {delegations.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No delegations found.</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Create/Edit modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Shield className="w-5 h-5 text-blue-500" /> {editId ? "Edit" : "New"} Delegation</h3>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Delegate To</label><input aria-label="username" type="text" value={form.delegated_to} onChange={(e) => setForm({ ...form, delegated_to: e.target.value })} placeholder="username" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Scope (comma-separated)</label><input aria-label="read:users, write:roles" type="text" value={form.scope} onChange={(e) => setForm({ ...form, scope: e.target.value })} placeholder="read:users, write:roles" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium flex items-center gap-1"><Calendar className="w-3 h-3" /> Start</label><input aria-label="form" type="date" value={form.start_date} onChange={(e) => setForm({ ...form, start_date: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
                <div><label className="text-sm font-medium flex items-center gap-1"><Calendar className="w-3 h-3" /> End</label><input aria-label="form" type="date" value={form.end_date} onChange={(e) => setForm({ ...form, end_date: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              </div>
              <div><label className="text-sm font-medium">Reason</label><input aria-label="Vacation coverage" type="text" value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} placeholder="Vacation coverage" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={saveDelegation} disabled={!form.delegated_to} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> Save</button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke confirmation */}
      {revokeId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setRevokeId(null)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="px-6 py-4"><p className="text-sm">Revoke this delegation? The delegatee will immediately lose all delegated permissions.</p></div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setRevokeId(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={doRevoke} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium hover:bg-red-700 flex items-center gap-1"><Trash2 className="w-4 h-4" /> Revoke</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
