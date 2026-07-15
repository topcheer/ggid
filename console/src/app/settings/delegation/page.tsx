"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import {
  Users, UserCheck, Clock, Plus, X, AlertCircle, Loader2,
  Shield, ArrowRight, Trash2, XCircle, CheckCircle2,
} from "lucide-react";

interface Delegation {
  id: string;
  delegator_id: string;
  delegator_name: string;
  delegate_id: string;
  delegate_name: string;
  roles: string[];
  scope: string;
  created_at: string;
  expires_at: string;
  status: "active" | "expired" | "revoked";
  last_used?: string;
}

export default function DelegationPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDelegate, setShowDelegate] = useState(false);
  const [confirmRevoke, setConfirmRevoke] = useState<Delegation | null>(null);
  const [tab, setTab] = useState<"active" | "expired">("active");

  // Delegate form
  const [form, setForm] = useState({
    delegate_id: "",
    roles: "",
    scope: "",
    expires_hours: 24,
  });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ delegations?: Delegation[]; items?: Delegation[] }>("/api/v1/settings/delegations").catch(() => null);
      setDelegations(data?.delegations ?? data?.items ?? []);
    } catch {
      setError("Failed to load delegations");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleDelegate = async () => {
    if (!form.delegate_id.trim()) return;
    setCreating(true);
    try {
      const roles = form.roles.split(",").map((r) => r.trim()).filter(Boolean);
      await apiFetch("/api/v1/settings/delegations", {
        method: "POST",
        body: JSON.stringify({
          delegate_id: form.delegate_id,
          roles,
          scope: form.scope || "*",
          expires_hours: form.expires_hours,
        }),
      });
      setForm({ delegate_id: "", roles: "", scope: "", expires_hours: 24 });
      setShowDelegate(false);
      await load();
    } catch {
      setError("Failed to create delegation");
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (id: string) => {
    try {
      await apiFetch(`/api/v1/settings/delegations/${id}`, { method: "DELETE" });
      setConfirmRevoke(null);
      await load();
    } catch {
      setError("Failed to revoke delegation");
    }
  };

  const active = delegations.filter((d) => d.status === "active");
  const past = delegations.filter((d) => d.status !== "active");
  const display = tab === "active" ? active : past;

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <UserCheck className="h-6 w-6 text-indigo-600" /> {t("delegation.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("delegation.subtitle")}
          </p>
        </div>
        <button onClick={() => setShowDelegate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
          <Plus className="h-4 w-4" /> {t("delegation.delegate")}
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("delegation.active")}</p>
          <p className="mt-1 text-2xl font-bold text-green-600">{active.length}</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("delegation.expiring24h")}</p>
          <p className="mt-1 text-2xl font-bold text-orange-600">
            {active.filter((d) => { const h = (new Date(d.expires_at).getTime() - Date.now()) / 3600000; return h > 0 && h < 24; }).length}
          </p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium text-gray-400">{t("delegation.past")}</p>
          <p className="mt-1 text-2xl font-bold text-gray-500">{past.length}</p>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        <button onClick={() => setTab("active")} className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "active" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400 hover:text-gray-600"}`}>
          <CheckCircle2 className="h-4 w-4" /> {t("delegation.active")} ({active.length})
        </button>
        <button onClick={() => setTab("expired")} className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "expired" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400 hover:text-gray-600"}`}>
          <Clock className="h-4 w-4" /> {t("delegation.history")} ({past.length})
        </button>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : display.length === 0 ? (
        <div className={cardCls}>
          <div className="py-12 text-center">
            <UserCheck className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-4 text-sm text-gray-400">{tab === "active" ? t("delegation.noActive") : t("delegation.noHistory")}</p>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          {display.map((d) => {
            const expHours = (new Date(d.expires_at).getTime() - Date.now()) / 3600000;
            const expiringSoon = d.status === "active" && expHours > 0 && expHours < 24;
            return (
              <div key={d.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    {/* Delegator → Delegate flow */}
                    <div className="flex items-center gap-2">
                      <div className="flex items-center gap-1.5">
                        <div className="rounded-lg bg-blue-100 p-1.5 dark:bg-blue-900/30">
                          <Users className="h-4 w-4 text-blue-600" />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{d.delegator_name}</p>
                          <p className="text-xs text-gray-400">{t("delegation.delegator")}</p>
                        </div>
                      </div>
                      <ArrowRight className="h-4 w-4 text-gray-300" />
                      <div className="flex items-center gap-1.5">
                        <div className="rounded-lg bg-indigo-100 p-1.5 dark:bg-indigo-900/30">
                          <UserCheck className="h-4 w-4 text-indigo-600" />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{d.delegate_name}</p>
                          <p className="text-xs text-gray-400">{t("delegation.delegateLabel")}</p>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {d.status === "active" ? (
                      <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${expiringSoon ? "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400" : "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"}`}>
                        {expiringSoon ? t("delegation.expiring") : t("delegation.active")}
                      </span>
                    ) : (
                      <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500 dark:bg-gray-700">{d.status}</span>
                    )}
                    {d.status === "active" && (
                      <button onClick={() => setConfirmRevoke(d)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                        <XCircle className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>

                {/* Details */}
                <div className="mt-3 flex flex-wrap items-center gap-2">
                  {d.roles.map((r) => (
                    <span key={r} className="flex items-center gap-1 rounded-lg bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400">
                      <Shield className="h-3 w-3" />{r}
                    </span>
                  ))}
                  <span className="text-xs text-gray-400">Scope: {d.scope}</span>
                </div>

                <div className="mt-2 flex items-center gap-4 text-xs text-gray-400">
                  <span className="flex items-center gap-1"><Clock className="h-3 w-3" />Expires: {new Date(d.expires_at).toLocaleString()}</span>
                  {d.last_used && <span>Last used: {new Date(d.last_used).toLocaleString()}</span>}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Delegate modal */}
      {showDelegate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowDelegate(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Delegate Roles</h2>
              <button onClick={() => setShowDelegate(false)}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Delegate User ID</label>
                <input value={form.delegate_id} onChange={(e) => setForm((p) => ({ ...p, delegate_id: e.target.value }))} placeholder="user-uuid" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Roles (comma-separated)</label>
                <input value={form.roles} onChange={(e) => setForm((p) => ({ ...p, roles: e.target.value }))} placeholder="admin, auditor" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Scope</label>
                <input value={form.scope} onChange={(e) => setForm((p) => ({ ...p, scope: e.target.value }))} placeholder="* or specific resource" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Expires In (hours)</label>
                <div className="mt-2 flex gap-2">
                  {[4, 8, 24, 72].map((h) => (
                    <button key={h} onClick={() => setForm((p) => ({ ...p, expires_hours: h }))} className={`rounded-lg border px-3 py-1.5 text-sm font-medium ${form.expires_hours === h ? "border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400" : "border-gray-300 text-gray-500 dark:border-gray-600"}`}>
                      {h < 24 ? `${h}h` : `${h / 24}d`}
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowDelegate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleDelegate} disabled={!form.delegate_id.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <UserCheck className="h-4 w-4" />} Delegate
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke confirmation */}
      {confirmRevoke && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRevoke(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><XCircle className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">{t("delegation.confirmRevoke")}</h2>
                <p className="text-sm text-gray-500">Delegate <strong>{confirmRevoke.delegate_name}</strong> will lose roles: {confirmRevoke.roles.join(", ")} immediately.</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmRevoke(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("delegation.cancel")}</button>
              <button onClick={() => handleRevoke(confirmRevoke.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{t("delegation.revoke")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
