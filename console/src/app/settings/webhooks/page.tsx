"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Webhook, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Play, Clock, ChevronRight, RotateCcw, Zap, Bell,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface WebhookEndpoint { id: string; url: string; events: string[]; secret: string; status: "active" | "disabled"; last_delivery?: string; }
interface Delivery { id: string; endpoint: string; event: string; status: string; attempts: number; response_code: number; timestamp: string; }

type Tab = "endpoints" | "deliveries" | "catalog";

const EVENT_CATALOG = [
  { category: "user", events: ["user.created", "user.updated", "user.deleted", "user.suspended", "user.role_changed"] },
  { category: "session", events: ["session.created", "session.revoked", "session.expired", "session.anomaly"] },
  { category: "risk", events: ["risk.score_changed", "risk.threshold_exceeded", "risk.step_up_triggered"] },
  { category: "itdr", events: ["itdr.detection_triggered", "itdr.incident_created", "itdr.incident_resolved"] },
  { category: "consent", events: ["consent.granted", "consent.withdrawn", "consent.expired"] },
  { category: "policy", events: ["policy.evaluated", "policy.denied", "policy.changed"] },
];

export default function WebhooksPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("endpoints");
  const [endpoints, setEndpoints] = useState<WebhookEndpoint[]>([]);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const [showForm, setShowForm] = useState(false);
  const [fUrl, setFUrl] = useState("");
  const [fEvents, setFEvents] = useState<string[]>([]);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [eRes, dRes] = await Promise.all([
        fetch("/api/v1/webhooks", { headers: h }).catch(() => null),
        fetch("/api/v1/webhooks/deliveries", { headers: h }).catch(() => null),
      ]);
      if (eRes?.ok) { const d = await eRes.json(); setEndpoints(d.webhooks || d.endpoints || []); }
      if (dRes?.ok) { const d = await dRes.json(); setDeliveries(d.deliveries || d.items || []); }
    } catch { setError(t("webhooks.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const createEndpoint = async () => {
    if (!fUrl) return;
    setActionLoading("create");
    try { await fetch("/api/v1/webhooks", { method: "POST", headers: H, body: JSON.stringify({ url: fUrl, events: fEvents, secret: `whsec_${Date.now()}` }) }); setShowForm(false); setFUrl(""); setFEvents([]); loadData(); }
    catch { setError(t("webhooks.createError")); }
    finally { setActionLoading(null); }
  };

  const deleteEndpoint = async (id: string) => {
    setActionLoading(`del-${id}`); try { await fetch(`/api/v1/webhooks/${id}`, { method: "DELETE", headers: h }); loadData(); } catch { /* noop */ } finally { setActionLoading(null); }
  };

  const replayDelivery = async (id: string) => {
    setActionLoading(`replay-${id}`); try { await fetch(`/api/v1/webhooks/deliveries/${id}/replay`, { method: "POST", headers: H }); loadData(); } catch { /* noop */ } finally { setActionLoading(null); }
  };

  const toggleEvent = (ev: string) => setFEvents(prev => prev.includes(ev) ? prev.filter(e => e !== ev) : [...prev, ev]);
  const allEvents = EVENT_CATALOG.flatMap(c => c.events);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white dark:text-white"><Webhook className="h-6 w-6 text-purple-500" /> {t("webhooks.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("webhooks.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "endpoints" as Tab, label: `${t("webhooks.endpoints")} (${endpoints.length})`, icon: Webhook },
          { id: "deliveries" as Tab, label: t("webhooks.deliveryHistory"), icon: Clock },
          { id: "catalog" as Tab, label: t("webhooks.eventCatalog"), icon: Zap },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-purple-600 text-purple-600 dark:text-purple-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {tb.label}</button>
        );})}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-500" /></div> : (<>

      {/* ENDPOINTS */}
      {tab === "endpoints" && (
        <div>
          <div className="mb-4 flex items-center justify-between"><h2 className="text-sm font-semibold uppercase text-gray-400">{t("webhooks.configuredEndpoints")}</h2><button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-purple-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-purple-700"><Plus className="h-3 w-3" /> {t("webhooks.addEndpoint")}</button></div>
          {endpoints.length === 0 ? <div className={card}><div className="py-12 text-center"><Webhook className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("webhooks.noEndpoints")}</p></div></div> : (
            <div className="space-y-2">{endpoints.map(ep => (
              <div key={ep.id} className={`${card} flex items-center justify-between !p-3`}>
                <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-purple-100 dark:bg-purple-900/30"><Webhook className="h-4 w-4 text-purple-500" /></div><div><div className="flex items-center gap-2"><code className="text-xs font-mono truncate max-w-md">{ep.url}</code><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${ep.status === "active" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{ep.status}</span></div><div className="flex flex-wrap gap-1 mt-0.5">{(ep.events || []).slice(0, 4).map(ev => <span key={ev} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-700 dark:bg-gray-700 text-xs font-mono">{ev}</span>)}{(ep.events || []).length > 4 && <span className="text-xs text-gray-400">+{ep.events.length - 4}</span>}</div><p className="text-xs text-gray-400">{ep.last_delivery ? `${t("webhooks.lastDelivery")}: ${new Date(ep.last_delivery).toLocaleString()}` : t("webhooks.noDeliveries")}</p></div></div>
                <button onClick={() => deleteEndpoint(ep.id)} disabled={actionLoading === `del-${ep.id}`} aria-label="Delete" className="rounded p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">{actionLoading === `del-${ep.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}</button>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* DELIVERIES */}
      {tab === "deliveries" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("webhooks.endpoint")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("webhooks.event")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("webhooks.attempts")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("webhooks.code")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("webhooks.status")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("webhooks.time")}</th><th className="px-3 py-2 text-right text-xs text-gray-400"></th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{deliveries.map(d => (
            <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-700 dark:bg-gray-800 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs font-mono truncate max-w-xs">{d.endpoint}</td><td className="px-3 py-3"><code className="text-xs font-mono text-purple-500">{d.event}</code></td><td className="px-3 py-3 text-center text-xs font-mono">{d.attempts}</td><td className="px-3 py-3 text-center"><span className={`text-xs font-mono ${d.response_code >= 200 && d.response_code < 300 ? "text-green-600" : "text-red-600"}`}>{d.response_code || "—"}</span></td><td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${d.status === "delivered" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : d.status === "failed" ? "bg-red-100 dark:bg-red-900/30 text-red-600" : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600"}`}>{d.status}</span></td><td className="px-3 py-3 text-xs text-gray-400">{new Date(d.timestamp).toLocaleString()}</td><td className="px-3 py-3 text-right">{d.status !== "delivered" && <button onClick={() => replayDelivery(d.id)} disabled={actionLoading === `replay-${d.id}`} aria-label="Replay" className="rounded p-1 text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-900/20">{actionLoading === `replay-${d.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="h-3.5 w-3.5" />}</button>}</td></tr>
          ))}</tbody>
        </table></div>
      )}

      {/* CATALOG */}
      {tab === "catalog" && (
        <div className="space-y-4">{EVENT_CATALOG.map(cat => (
          <div key={cat.category} className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold"><span className="px-2 py-0.5 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-600 text-xs font-mono">{cat.category}.*</span></h3>
            <div className="flex flex-wrap gap-2">{cat.events.map(ev => <span key={ev} className="flex items-center gap-1 rounded-lg border px-2 py-1 text-xs dark:border-gray-700"><Bell className="h-3 w-3 text-gray-400" /><code className="font-mono">{ev}</code></span>)}</div>
          </div>
        ))}</div>
      )}

      </>)}

      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-lg rounded-xl bg-white dark:bg-gray-800 p-6 shadow-xl dark:bg-gray-800 max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white dark:text-white"><Plus className="h-5 w-5 text-purple-500" /> {t("webhooks.addEndpoint")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("webhooks.url")}</label><input type="text" value={fUrl} onChange={e => setFUrl(e.target.value)} placeholder="https://hooks.example.com/ggid" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("webhooks.subscribeEvents")}</label>
                <div className="mt-1 max-h-48 overflow-y-auto space-y-1 rounded-lg border dark:border-gray-700 p-2">{allEvents.map(ev => <label key={ev} className="flex items-center gap-2 cursor-pointer rounded p-1 hover:bg-gray-50 dark:hover:bg-gray-700 dark:bg-gray-800 dark:hover:bg-gray-900/50"><input type="checkbox" checked={fEvents.includes(ev)} onChange={() => toggleEvent(ev)} className="rounded" /><code className="text-xs font-mono">{ev}</code></label>)}</div>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 dark:border-gray-600 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={createEndpoint} disabled={!fUrl || actionLoading === "create"} className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : t("webhooks.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
