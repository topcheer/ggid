"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Users, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Building2, Clock, Ghost, ChevronRight, Ban, Play, AlertTriangle,
  CheckCircle2, XCircle, Download, Zap, UserX,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface HRConnector { id: string; name: string; type: string; enabled: boolean; last_sync_at?: string; created_at: string; }
interface DormantUser { user_id: string; email: string; last_active: string; state: "active" | "dormant" | "suspended" | "archived"; days_inactive: number; }
interface GhostUser { user_id: string; email: string; in_hr: boolean; action: string; }
interface SyncLogEntry { id: string; source: string; events: number; errors: number; status: string; timestamp: string; }

type Tab = "connectors" | "sync" | "dormant" | "ghosts";

const CONNECTOR_TYPES = ["workday", "bamboohr", "csv", "api", "ldap"];
const STATE_CFG: Record<string, { color: string; bg: string }> = {
  active: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  dormant: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  suspended: { color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30" },
  archived: { color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-800" },
};

export default function HRLifecyclePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("connectors");
  const [connectors, setConnectors] = useState<HRConnector[]>([]);
  const [syncLog, setSyncLog] = useState<SyncLogEntry[]>([]);
  const [dormant, setDormant] = useState<DormantUser[]>([]);
  const [ghosts, setGhosts] = useState<GhostUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Connector form
  const [showForm, setShowForm] = useState(false);
  const [fName, setFName] = useState("");
  const [fType, setFType] = useState("workday");

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [cRes, sRes, dRes, gRes] = await Promise.all([
        fetch("/api/v1/hr/connectors", { headers: h }).catch(() => null),
        fetch("/api/v1/hr/sync/log", { headers: h }).catch(() => null),
        fetch("/api/v1/hr/dormant", { headers: h }).catch(() => null),
        fetch("/api/v1/hr/reconcile", { headers: h }).catch(() => null),
      ]);
      if (cRes?.ok) { const d = await cRes.json(); setConnectors(d.connectors || []); }
      if (sRes?.ok) { const d = await sRes.json(); setSyncLog(d.logs || d.entries || []); }
      if (dRes?.ok) { const d = await dRes.json(); setDormant(d.users || d.dormant || []); }
      if (gRes?.ok) { const d = await gRes.json(); setGhosts(d.ghosts || d.users || []); }
    } catch { setError(t("hrLifecycle.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const createConnector = async () => {
    if (!fName) return;
    setActionLoading("create");
    try {
      await fetch("/api/v1/hr/connectors", { method: "POST", headers: H, body: JSON.stringify({ name: fName, type: fType, enabled: true }) });
      setShowForm(false); setFName(""); loadData();
    } catch { setError(t("hrLifecycle.createError")); }
    finally { setActionLoading(null); }
  };

  const triggerSync = async () => {
    setActionLoading("sync");
    try { await fetch("/api/v1/hr/sync", { method: "POST", headers: H }); loadData(); }
    catch { setError(t("hrLifecycle.syncError")); }
    finally { setActionLoading(null); }
  };

  const progressUser = async (userId: string, newState: string) => {
    setActionLoading(`prog-${userId}`);
    try { await fetch("/api/v1/hr/dormant", { method: "POST", headers: H, body: JSON.stringify({ user_id: userId, action: newState }) }); loadData(); }
    catch { /* noop */ }
    finally { setActionLoading(null); }
  };

  const reconcile = async () => {
    setActionLoading("reconcile");
    try { await fetch("/api/v1/hr/reconcile", { method: "POST", headers: H }); loadData(); }
    catch { /* noop */ }
    finally { setActionLoading(null); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Users className="h-6 w-6 text-indigo-500" /> {t("hrLifecycle.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("hrLifecycle.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "connectors" as Tab, label: t("hrLifecycle.connectors"), icon: Building2 },
          { id: "sync" as Tab, label: t("hrLifecycle.syncLog"), icon: RefreshCw },
          { id: "dormant" as Tab, label: `${t("hrLifecycle.dormant")} (${dormant.length})`, icon: Clock },
          { id: "ghosts" as Tab, label: `${t("hrLifecycle.ghosts")} (${ghosts.length})`, icon: Ghost },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* CONNECTORS */}
      {tab === "connectors" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase text-gray-400">{t("hrLifecycle.hrConnectors")}</h2>
            <button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><Plus className="h-3 w-3" /> {t("hrLifecycle.addConnector")}</button>
          </div>
          {connectors.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Building2 className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("hrLifecycle.noConnectors")}</p></div></div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{connectors.map(c => (
              <div key={c.id} className={card + " hover:shadow-md transition"}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Building2 className="h-5 w-5 text-indigo-500" /></div><div><h3 className="font-semibold text-sm">{c.name}</h3><p className="text-xs text-gray-400 capitalize">{c.type}</p></div></div>
                  <span className={`h-2 w-2 rounded-full ${c.enabled ? "bg-green-500 animate-pulse" : "bg-gray-400"}`} />
                </div>
                <div className="mt-3 text-xs text-gray-400">{c.last_sync_at ? `${t("hrLifecycle.lastSync")}: ${new Date(c.last_sync_at).toLocaleString()}` : t("hrLifecycle.neverSynced")}</div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* SYNC LOG */}
      {tab === "sync" && (
        <div>
          <div className="mb-4"><button onClick={triggerSync} disabled={actionLoading === "sync"} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{actionLoading === "sync" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} {t("hrLifecycle.triggerSync")}</button></div>
          {syncLog.length === 0 ? (
            <div className={card}><div className="py-8 text-center"><RefreshCw className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("hrLifecycle.noSyncHistory")}</p></div></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("hrLifecycle.time")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("hrLifecycle.source")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("hrLifecycle.events")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("hrLifecycle.errors")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("hrLifecycle.status")}</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{syncLog.map(l => (
                <tr key={l.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs">{new Date(l.timestamp).toLocaleString()}</td><td className="px-3 py-3 text-xs font-mono">{l.source}</td><td className="px-3 py-3 text-center text-xs font-mono">{l.events}</td><td className="px-3 py-3 text-center"><span className={`text-xs font-mono ${l.errors > 0 ? "text-red-600" : "text-gray-400"}`}>{l.errors}</span></td><td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${l.status === "success" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{l.status}</span></td></tr>
              ))}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* DORMANT */}
      {tab === "dormant" && (
        <div className="space-y-2">
          {dormant.length === 0 ? (
            <div className={card}><div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("hrLifecycle.noDormant")}</p></div></div>
          ) : dormant.map(d => {
            const cfg = STATE_CFG[d.state] || STATE_CFG.dormant;
            return (
              <div key={d.user_id} className={`${card} flex items-center justify-between !p-3`}>
                <div className="flex items-center gap-3">
                  <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><Clock className={`h-4 w-4 ${cfg.color}`} /></div>
                  <div><div className="flex items-center gap-2"><span className="text-xs font-mono">{d.email}</span><span className={`px-1.5 py-0.5 rounded text-xs ${cfg.bg} ${cfg.color}`}>{d.state}</span></div><p className="text-xs text-gray-400">{d.days_inactive} {t("hrLifecycle.daysInactive")} · {t("hrLifecycle.lastActive")}: {new Date(d.last_active).toLocaleDateString()}</p></div>
                </div>
                <div className="flex items-center gap-1">
                  {d.state === "dormant" && <button onClick={() => progressUser(d.user_id, "suspend")} disabled={actionLoading === `prog-${d.user_id}`} className="rounded-lg bg-orange-600 px-2 py-1 text-xs font-medium text-white hover:bg-orange-700">{t("hrLifecycle.suspend")}</button>}
                  {d.state === "suspended" && <button onClick={() => progressUser(d.user_id, "archive")} disabled={actionLoading === `prog-${d.user_id}`} className="rounded-lg bg-gray-600 px-2 py-1 text-xs font-medium text-white hover:bg-gray-700">{t("hrLifecycle.archive")}</button>}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* GHOSTS */}
      {tab === "ghosts" && (
        <div>
          <div className="mb-4"><button onClick={reconcile} disabled={actionLoading === "reconcile"} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{actionLoading === "reconcile" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} {t("hrLifecycle.runReconcile")}</button></div>
          {ghosts.length === 0 ? (
            <div className={card}><div className="py-8 text-center"><Ghost className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("hrLifecycle.noGhosts")}</p></div></div>
          ) : (
            <div className="space-y-2">{ghosts.map(g => (
              <div key={g.user_id} className={`${card} flex items-center justify-between !p-3`}>
                <div className="flex items-center gap-3"><div className="flex h-8 w-8 items-center justify-center rounded-lg bg-red-100 dark:bg-red-900/30"><UserX className="h-4 w-4 text-red-500" /></div><div><span className="text-xs font-mono">{g.email}</span><p className="text-xs text-gray-400">{t("hrLifecycle.notInHr")}</p></div></div>
                <button className="rounded-lg bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700">{t("hrLifecycle.disable")}</button>
              </div>
            ))}</div>
          )}
        </div>
      )}

      </>)}

      {/* Connector form */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-indigo-500" /> {t("hrLifecycle.addConnector")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("hrLifecycle.connectorName")}</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="Workday Production" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("hrLifecycle.connectorType")}</label><select value={fType} onChange={e => setFType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">{CONNECTOR_TYPES.map(tp => <option key={tp} value={tp}>{tp}</option>)}</select></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={createConnector} disabled={!fName || actionLoading === "create"} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : t("hrLifecycle.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
