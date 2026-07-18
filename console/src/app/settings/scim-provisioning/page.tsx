"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ArrowRightLeft, Loader2, AlertCircle, X, RefreshCw, Plus, Check,
  Play, ChevronRight, Clock, CheckCircle2, XCircle, Zap,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface SCIMTarget { id: string; name: string; endpoint: string; status: string; last_sync: string; user_count: number; enabled: boolean; }
interface SyncLogEntry { id: string; target: string; operation: string; user: string; status: string; error?: string; timestamp: string; }

type Tab = "targets" | "sync" | "mapping";

const DEFAULT_MAPPING = [
  { ggid: "email", scim: "userName", required: true },
  { ggid: "display_name", scim: "displayName", required: true },
  { ggid: "email", scim: "emails[type eq \"work\"].value", required: true },
  { ggid: "status", scim: "active", required: true },
  { ggid: "groups", scim: "groups", required: false },
  { ggid: "department", scim: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department", required: false },
  { ggid: "title", scim: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:title", required: false },
  { ggid: "manager", scim: "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:manager", required: false },
];

export default function SCIMPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("targets");
  const [targets, setTargets] = useState<SCIMTarget[]>([]);
  const [syncLog, setSyncLog] = useState<SyncLogEntry[]>([]);
  const [mapping, setMapping] = useState(DEFAULT_MAPPING);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Form
  const [showForm, setShowForm] = useState(false);
  const [fName, setFName] = useState("");
  const [fEndpoint, setFEndpoint] = useState("");

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [tRes, lRes] = await Promise.all([
        fetch("/api/v1/identity/scim/sync-health", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/scim/config/sync", { headers: h }).catch(() => null),
      ]);
      if (tRes?.ok) { const d = await tRes.json(); setTargets(d.targets || d.apps || []); }
      if (lRes?.ok) { const d = await lRes.json(); setSyncLog(d.logs || d.operations || []); }
    } catch { setError(t("scim.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const triggerSync = async (targetId: string) => {
    setActionLoading(`sync-${targetId}`);
    try { await fetch(`/api/v1/identity/scim/config/sync`, { method: "POST", headers: H, body: JSON.stringify({ target_id: targetId }) }); loadData(); }
    catch { setError(t("scim.syncError")); }
    finally { setActionLoading(null); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ArrowRightLeft className="h-6 w-6 text-cyan-500" /> {t("scim.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("scim.subtitle")}</p>
      </div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "targets" as Tab, label: t("scim.targets"), icon: ArrowRightLeft },
          { id: "sync" as Tab, label: t("scim.syncLog"), icon: Clock },
          { id: "mapping" as Tab, label: t("scim.attributeMapping"), icon: Zap },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-cyan-600 text-cyan-600 dark:text-cyan-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {tb.label}</button>
        );})}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-500" /></div> : (<>

      {/* TARGETS */}
      {tab === "targets" && (
        <div>
          <div className="mb-4 flex items-center justify-between"><h2 className="text-sm font-semibold uppercase text-gray-400">{t("scim.downstreamApps")}</h2><button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-cyan-700"><Plus className="h-3 w-3" /> {t("scim.addTarget")}</button></div>
          {targets.length === 0 ? <div className={card}><div className="py-12 text-center"><ArrowRightLeft className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("scim.noTargets")}</p></div></div> : (
            <div className="space-y-2">{targets.map(tg => (
              <div key={tg.id} className={`${card} flex items-center justify-between !p-3`}>
                <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-cyan-100 dark:bg-cyan-900/30"><ArrowRightLeft className="h-4 w-4 text-cyan-500" /></div><div><div className="flex items-center gap-2"><span className="text-sm font-medium">{tg.name}</span><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${tg.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{tg.enabled ? "active" : "disabled"}</span></div><p className="text-xs text-gray-400 font-mono truncate max-w-md">{tg.endpoint}</p><p className="text-xs text-gray-400">{tg.user_count} {t("scim.users")} · {tg.last_sync ? new Date(tg.last_sync).toLocaleDateString() : t("scim.neverSynced")}</p></div></div>
                <button onClick={() => triggerSync(tg.id)} disabled={actionLoading === `sync-${tg.id}`} className="flex items-center gap-1 rounded-lg bg-cyan-600 px-2 py-1 text-xs font-medium text-white hover:bg-cyan-700 disabled:opacity-50">{actionLoading === `sync-${tg.id}` ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />} {t("scim.sync")}</button>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* SYNC LOG */}
      {tab === "sync" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> {t("scim.operationsHistory")}</h2>
          {syncLog.length === 0 ? <div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("scim.noSyncHistory")}</p></div> : (
            <div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("scim.target")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("scim.operation")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("scim.user")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("scim.status")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("scim.time")}</th></tr></thead>
            <tbody className="divide-y dark:divide-gray-800">{syncLog.map(l => (
              <tr key={l.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs font-medium">{l.target}</td><td className="px-3 py-3 text-center"><code className="text-xs font-mono">{l.operation}</code></td><td className="px-3 py-3 text-xs font-mono">{l.user}</td><td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${l.status === "success" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{l.status}</span></td><td className="px-3 py-3 text-xs text-gray-400">{new Date(l.timestamp).toLocaleString()}</td></tr>
            ))}</tbody></table></div>
          )}
        </div>
      )}

      {/* MAPPING */}
      {tab === "mapping" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> {t("scim.ggidToScimMapping")}</h2>
          <div className="space-y-2">{mapping.map((m: any, i: number) => (
            <div key={i} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
              <code className="text-xs font-mono text-gray-500 w-32">{m.ggid}</code>
              <ChevronRight className="h-3 w-3 text-gray-300" />
              <code className="flex-1 text-xs font-mono text-cyan-500">{m.scim}</code>
              {m.required && <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 text-red-600">{t("scim.required")}</span>}
            </div>
          ))}</div>
          <p className="mt-3 text-xs text-gray-400">{t("scim.mappingNote")}</p>
        </div>
      )}

      </>)}

      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-cyan-500" /> {t("scim.addTarget")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("scim.appName")}</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="Slack" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("scim.scimEndpoint")}</label><input type="text" value={fEndpoint} onChange={e => setFEndpoint(e.target.value)} placeholder="https://api.slack.com/scim/v2/Users" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={() => { setShowForm(false); loadData(); }} disabled={!fName} className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">{t("scim.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
