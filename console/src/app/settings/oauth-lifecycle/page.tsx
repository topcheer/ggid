"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Workflow, Loader2, AlertCircle, X, Pause, Play, RefreshCw, Clock, KeyRound, AlertOctagon,
} from "lucide-react";

interface ClientLifecycle {
  id: string;
  client_id: string;
  client_name: string;
  status: "active" | "suspended" | "expired" | "revoked";
  created_at: string;
  last_active: string;
  token_count: number;
  active_tokens: number;
  redirect_uris: string[];
  suspend_reason: string;
  suspended_at: string;
  expires_at: string;
}

const statusColors: Record<string, string> = {
  active: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  suspended: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  expired: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  revoked: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
};

export default function OAuthLifecyclePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [clients, setClients] = useState<ClientLifecycle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actioning, setActioning] = useState<string | null>(null);
  const [suspendTarget, setSuspendTarget] = useState<ClientLifecycle | null>(null);
  const [suspendReason, setSuspendReason] = useState("");

  useEffect(() => {
    (async () => {
      try { setClients(await apiFetch<ClientLifecycle[]>("/api/v1/oauth/clients/lifecycle").catch(() => [])); }
      catch { setError(t("oauthLifecycle.failedLoad")); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleSuspend = async () => {
    if (!suspendTarget) return;
    setActioning(suspendTarget.id);
    try { await apiFetch(`/api/v1/oauth/clients/${suspendTarget.client_id}/suspend`, { method: "POST", body: JSON.stringify({ reason: suspendReason }) }); setClients(await apiFetch<ClientLifecycle[]>("/api/v1/oauth/clients/lifecycle").catch(() => clients)); setSuspendTarget(null); setSuspendReason(""); }
    catch { setError("Suspend failed"); }
    finally { setActioning(null); }
  };

  const handleReinstate = async (clientId: string) => {
    setActioning(clientId);
    try { await apiFetch(`/api/v1/oauth/clients/${clientId}/reinstate`, { method: "POST" }); setClients(await apiFetch<ClientLifecycle[]>("/api/v1/oauth/clients/lifecycle").catch(() => clients)); }
    catch { setError("Reinstate failed"); }
    finally { setActioning(null); }
  };

  const handleRevoke = async (clientId: string) => {
    setActioning(clientId);
    try { await apiFetch(`/api/v1/oauth/clients/${clientId}/revoke`, { method: "POST" }); setClients(await apiFetch<ClientLifecycle[]>("/api/v1/oauth/clients/lifecycle").catch(() => clients)); }
    catch { setError("Revoke failed"); }
    finally { setActioning(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const active = clients.filter((c) => c.status === "active").length;
  const suspended = clients.filter((c) => c.status === "suspended").length;
  const expired = clients.filter((c) => c.status === "expired").length;
  const totalTokens = clients.reduce((s, c) => s + c.active_tokens, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Workflow className="h-6 w-6 text-indigo-600" />{t("oauthLifecycle.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Monitor and manage OAuth client status: active, suspended, expired, revoked.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-4 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Play className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active</span></div><p className="mt-2 text-2xl font-bold text-green-600">{active}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Pause className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">Suspended</span></div><p className="mt-2 text-2xl font-bold text-yellow-600">{suspended}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Clock className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Expired</span></div><p className="mt-2 text-2xl font-bold text-red-600">{expired}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><KeyRound className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active Tokens</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{totalTokens}</p></div>
          </div>

          {/* Table */}
          {clients.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><Workflow className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No OAuth clients registered.</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Client</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Tokens (Active/Total)</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Last Active</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Expires</th>
                  <th scope="col" className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {clients.map((c) => (
                    <tr key={c.id} className="bg-white dark:bg-gray-900">
                      <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{c.client_name}</div><div className="text-xs text-gray-400 font-mono">{c.client_id.slice(0, 16)}</div></td>
                      <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[c.status] || ""}`}>{c.status}</span>{c.status === "suspended" && c.suspend_reason && <div className="mt-0.5 text-xs text-yellow-600">{c.suspend_reason}</div>}</td>
                      <td className="px-4 py-3"><span className="font-medium text-gray-900 dark:text-white">{c.active_tokens}</span><span className="text-gray-400"> / {c.token_count}</span></td>
                      <td className="px-4 py-3 text-gray-400">{c.last_active ? new Date(c.last_active).toLocaleDateString() : "—"}</td>
                      <td className="px-4 py-3 text-gray-400">{c.expires_at ? new Date(c.expires_at).toLocaleDateString() : "—"}</td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-2">
                          {c.status === "active" && <button onClick={() => setSuspendTarget(c)} disabled={actioning === c.id} className="flex items-center gap-1 rounded bg-yellow-100 px-2 py-1 text-xs text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30">{actioning === c.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <Pause className="h-3 w-3" />}Suspend</button>}
                          {c.status === "suspended" && <button onClick={() => handleReinstate(c.client_id)} disabled={actioning === c.id} className="flex items-center gap-1 rounded bg-green-100 px-2 py-1 text-xs text-green-700 hover:bg-green-200 dark:bg-green-900/30">{actioning === c.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}Reinstate</button>}
                          {(c.status === "active" || c.status === "suspended") && <button onClick={() => handleRevoke(c.client_id)} disabled={actioning === c.id} className="flex items-center gap-1 rounded bg-red-100 px-2 py-1 text-xs text-red-700 hover:bg-red-200 dark:bg-red-900/30">{actioning === c.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <AlertOctagon className="h-3 w-3" />}Revoke</button>}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Suspend modal */}
      {suspendTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setSuspendTarget(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="flex items-center gap-2 text-lg font-bold text-gray-900 dark:text-white"><Pause className="h-5 w-5 text-yellow-600" /> Suspend Client</h3><button onClick={() => setSuspendTarget(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <p className="mb-4 text-sm text-gray-500">Suspending <span className="font-medium text-gray-900 dark:text-white">{suspendTarget.client_name}</span> will immediately invalidate all active tokens.</p>
            <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Reason</label><textarea aria-label="Reason for suspension..." value={suspendReason} onChange={(e) => setSuspendReason(e.target.value)} rows={3} placeholder="Reason for suspension..." className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
            <button onClick={handleSuspend} disabled={actioning === suspendTarget.id} className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-yellow-600 py-2 text-sm font-medium text-white hover:bg-yellow-700 disabled:opacity-50">{actioning === suspendTarget.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Pause className="h-4 w-4" />}Confirm Suspend</button>
          </div>
        </div>
      )}
    </div>
  );
}
