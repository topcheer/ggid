"use client";

import { useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import {
  Users, Loader2, AlertCircle, X, Plus, Trash2, ShieldCheck, Clock,
} from "lucide-react";

interface Delegation {
  id: string;
  delegate: string;
  delegate_name: string;
  scope_type: "org" | "role" | "dept" | "global";
  scope_value: string;
  permissions: string[];
  granted_by: string;
  granted_at: string;
  expires_at: string;
  revoked: boolean;
}

const scopeColors: Record<string, string> = {
  org: "text-purple-600 bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400",
  role: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  dept: "text-cyan-600 bg-cyan-100 dark:bg-cyan-900/30 dark:text-cyan-400",
  global: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function DelegatedAdminPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showGrant, setShowGrant] = useState(false);
  const [revoking, setRevoking] = useState<string | null>(null);
  const [form, setForm] = useState({ delegate: "", scope_type: "org" as Delegation["scope_type"], scope_value: "", permissions: [] as string[], expires_at: "" });

  useState(() => {
    (async () => {
      try { setDelegations(await apiFetch<Delegation[]>("/api/v1/policy/delegated-admin").catch(() => [])); }
      catch { setError("Failed to load delegations"); }
      finally { setLoading(false); }
    })();
  });

  const togglePerm = (perm: string) => {
    setForm((f) => ({ ...f, permissions: f.permissions.includes(perm) ? f.permissions.filter((p) => p !== perm) : [...f.permissions, perm] }));
  };

  const handleGrant = async () => {
    try {
      const created = await apiFetch<Delegation>("/api/v1/policy/delegated-admin", { method: "POST", body: JSON.stringify(form) });
      setDelegations((p) => [created, ...p]);
      setShowGrant(false);
      setForm({ delegate: "", scope_type: "org", scope_value: "", permissions: [], expires_at: "" });
    } catch { setError("Grant failed"); }
  };

  const handleRevoke = async (id: string) => {
    setRevoking(id);
    try {
      await apiFetch(`/api/v1/policy/delegated-admin/${id}/revoke`, { method: "POST" });
      setDelegations((p) => p.map((d) => d.id === id ? { ...d, revoked: true } : d));
    } catch { setError("Revoke failed"); }
    finally { setRevoking(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const allPerms = ["read", "write", "admin", "full"];
  const active = delegations.filter((d) => !d.revoked);
  const expired = delegations.filter((d) => d.revoked || (d.expires_at && new Date(d.expires_at) < new Date()));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Users className="h-6 w-6 text-purple-600" /> {t("delegatedAdmin.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("delegatedAdmin.subtitle")}</p>
        </div>
        <button onClick={() => setShowGrant(true)}aria-label="Grant delegation" className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700"><Plus className="h-4 w-4" /> {t("delegatedAdmin.grant")}</button>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)}aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><ShieldCheck className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">{t("delegatedAdmin.active")}</span></div><p className="mt-2 text-2xl font-bold text-green-600">{active.length}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Clock className="h-4 w-4 text-orange-500" /><span className="text-xs font-semibold uppercase text-gray-400">{t("delegatedAdmin.expiringSoon")}</span></div><p className="mt-2 text-2xl font-bold text-orange-600">{delegations.filter((d) => !d.revoked && d.expires_at && new Date(d.expires_at).getTime() - Date.now() < 7 * 86400000).length}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><X className="h-4 w-4 text-gray-400" /><span className="text-xs font-semibold uppercase text-gray-400">{t("delegatedAdmin.revokedExpired")}</span></div><p className="mt-2 text-2xl font-bold text-gray-500">{expired.length}</p></div>
          </div>

          {/* Table */}
          {delegations.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><Users className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("delegatedAdmin.noGrants")}</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("delegatedAdmin.delegate")}</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("delegatedAdmin.scope")}</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("delegatedAdmin.permissions")}</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("delegatedAdmin.grantedBy")}</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("delegatedAdmin.expires")}</th>
                  <th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">{t("delegatedAdmin.actions")}</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {delegations.map((d) => (
                    <tr key={d.id} className={`bg-white dark:bg-gray-900 ${d.revoked ? "opacity-50" : ""}`}>
                      <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{d.delegate_name || d.delegate}</div><div className="text-xs text-gray-400 font-mono">{d.delegate.slice(0, 16)}</div></td>
                      <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${scopeColors[d.scope_type] || ""}`}>{d.scope_type}</span><div className="mt-0.5 text-xs text-gray-400">{d.scope_value || "—"}</div></td>
                      <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{d.permissions.map((p) => <span key={p} className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{p}</span>)}</div></td>
                      <td className="px-4 py-3 text-gray-400">{d.granted_by.slice(0, 12)}</td>
                      <td className="px-4 py-3"><span className={`${d.expires_at && new Date(d.expires_at) < new Date() ? "text-red-500" : "text-gray-400"}`}>{d.expires_at ? new Date(d.expires_at).toLocaleDateString() : "—"}</span>{d.revoked && <span className="ml-1 text-xs text-red-500">revoked</span>}</td>
                      <td className="px-4 py-3 text-right">{!d.revoked && <button onClick={() => handleRevoke(d.id)} disabled={revoking === d.id} aria-label={revoking === d.id ? "Revoking delegation" : "Revoke delegation"} className="text-red-500 hover:text-red-700">{revoking === d.id ? <Loader2 className="inline h-3 w-3 animate-spin" /> : t("delegatedAdmin.revoke")}</button>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Grant modal */}
      {showGrant && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowGrant(false)}>
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{t("delegatedAdmin.grantDelegation")}</h3><button onClick={() => setShowGrant(false)} aria-label="Close grant dialog"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Delegate (User ID)</label><input value={form.delegate} onChange={(e) => setForm({ ...form, delegate: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("delegatedAdmin.scopeType")}</label><select value={form.scope_type} onChange={(e) => setForm({ ...form, scope_type: e.target.value as Delegation["scope_type"] })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="org">Organization</option><option value="role">Role</option><option value="dept">Department</option><option value="global">Global</option></select></div>
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("delegatedAdmin.scopeValue")}</label><input value={form.scope_value} onChange={(e) => setForm({ ...form, scope_value: e.target.value })} placeholder="e.g. eng-dept" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              </div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Permissions</label><div className="flex gap-2">{allPerms.map((p) => <button key={p} onClick={() => togglePerm(p)} className={`rounded-lg px-3 py-1.5 text-xs font-medium ${form.permissions.includes(p) ? "bg-purple-600 text-white" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{p}</button>)}</div></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{t("delegatedAdmin.expiresAt")}</label><input type="datetime-local" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <button onClick={handleGrant} disabled={!form.delegate || form.permissions.length === 0} aria-label="Grant delegated access" className="flex w-full items-center justify-center gap-2 rounded-lg bg-purple-600 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50"><ShieldCheck className="h-4 w-4" /> {t("delegatedAdmin.grantAccess")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
