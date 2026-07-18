"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  AlertTriangle, Loader2, AlertCircle, X, ChevronDown, ChevronRight, CheckCircle, TrendingUp,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RelatedEvent {
  event_id: string;
  action: string;
  timestamp: string;
}

interface Anomaly {
  id: string;
  type: string;
  description: string;
  severity: "low" | "medium" | "high" | "critical";
  confidence: number;
  status: "active" | "dismissed" | "escalated" | "resolved";
  user_id: string;
  related_events: RelatedEvent[];
  detected_at: string;
  metadata: Record<string, string>;
}

const sevColors: Record<string, string> = {
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

const statusColors: Record<string, string> = {
  active: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  dismissed: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
  escalated: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  resolved: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
};

export default function AnomaliesPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [actioning, setActioning] = useState<string | null>(null);
  const [actionModal, setActionModal] = useState<{ anomaly: Anomaly; type: "dismiss" | "escalate" } | null>(null);
  const [note, setNote] = useState("");

  useEffect(() => {
    (async () => {
      try { setAnomalies(await apiFetch<Anomaly[]>("/api/v1/audit/anomalies").catch(() => [])); }
      catch { setError("Failed to load anomalies"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleAction = async () => {
    if (!actionModal) return;
    setActioning(actionModal.anomaly.id);
    try {
      await apiFetch(`/api/v1/audit/anomalies/${actionModal.anomaly.id}/${actionModal.type}`, { method: "POST", body: JSON.stringify(actionModal.type === "dismiss" ? { reason: note } : { note }) });
      setAnomalies((p) => p.map((a: any) => a.id === actionModal.anomaly.id ? { ...a, status: actionModal.type === "dismiss" ? "dismissed" : "escalated" } : a));
      setActionModal(null); setNote("");
    } catch { setError("Action failed"); }
    finally { setActioning(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const active = anomalies.filter((a: any) => a.status === "active");
  const escalated = anomalies.filter((a: any) => a.status === "escalated");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><AlertTriangle className="h-6 w-6 text-orange-600" /> {t("auditAnomalies.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">ML-based behavioral anomaly detection with confidence scoring.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-orange-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Active</div><p className="mt-2 text-2xl font-bold text-orange-600">{active.length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Escalated</div><p className="mt-2 text-2xl font-bold text-red-600">{escalated.length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Avg Confidence</div><p className="mt-2 text-2xl font-bold text-indigo-600">{anomalies.length > 0 ? Math.round(anomalies.reduce((s: any, a: any) => s + a.confidence, 0) / anomalies.length) : 0}%</p></div>
          </div>

          {/* Anomalies list */}
          {anomalies.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><CheckCircle className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">No anomalies detected.</p></div></div>
          ) : (
            <div className="space-y-2">
              {anomalies.map((a: any) => (
                <div key={a.id} className={cardCls}>
                  <div className="flex items-start justify-between">
                    <div className="flex flex-1 items-start gap-3">
                      <button onClick={() => setExpanded(expanded === a.id ? null : a.id)} className="mt-0.5">{expanded === a.id ? <ChevronDown className="h-4 w-4 text-gray-400" /> : <ChevronRight className="h-4 w-4 text-gray-400" />}</button>
                      <div className="flex-1">
                        <div className="flex items-center gap-2"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${sevColors[a.severity] || ""}`}>{a.severity}</span><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[a.status] || ""}`}>{a.status}</span><span className="text-sm font-medium text-gray-900 dark:text-white">{a.type}</span></div>
                        <p className="mt-1 text-sm text-gray-500">{a.description}</p>
                        <div className="mt-1 flex items-center gap-3 text-xs text-gray-400"><span className="flex items-center gap-0.5"><TrendingUp className="h-3 w-3" />{a.confidence}% confidence</span><span>User: {a.user_id.slice(0, 12)}</span><span>{a.related_events.length} related events</span><span>{new Date(a.detected_at).toLocaleString()}</span></div>
                        {/* Expanded related events */}
                        {expanded === a.id && a.related_events.length > 0 && (
                          <div className="mt-3 space-y-1 rounded-lg bg-gray-50 p-3 dark:bg-gray-900">{a.related_events.map((e: any) => (<div key={e.event_id} className="flex items-center justify-between text-xs"><span className="font-mono text-gray-500">{e.event_id.slice(0, 16)}</span><span className="text-gray-400">{e.action}</span><span className="text-gray-400">{new Date(e.timestamp).toLocaleTimeString()}</span></div>))}</div>
                        )}
                      </div>
                    </div>
                    {a.status === "active" && (<div className="flex gap-2"><button onClick={() => { setActionModal({ anomaly: a, type: "dismiss" }); setNote(""); }} className="rounded bg-gray-100 px-3 py-1 text-xs text-gray-600 dark:bg-gray-700">Dismiss</button><button onClick={() => { setActionModal({ anomaly: a, type: "escalate" }); setNote(""); }} className="rounded bg-red-100 px-3 py-1 text-xs text-red-600 dark:bg-red-900/30">Escalate</button></div>)}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Action modal */}
          {actionModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setActionModal(null)}>
              <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
                <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold capitalize text-gray-900 dark:text-white">{actionModal.type}: {actionModal.anomaly.type}</h3><button onClick={() => setActionModal(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">{actionModal.type === "dismiss" ? "Reason" : "Escalation Note"}</label><textarea aria-label="Text input" value={note} onChange={(e) => setNote(e.target.value)} rows={3} placeholder={actionModal.type === "dismiss" ? "Why is this anomaly not a concern?" : "Why is this being escalated?"} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <button aria-label="action" onClick={handleAction} disabled={actioning === actionModal.anomaly.id} className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-orange-600 py-2 text-sm font-medium text-white capitalize hover:bg-orange-700 disabled:opacity-50">{actioning === actionModal.anomaly.id ? <Loader2 className="h-4 w-4 animate-spin" /> : null}{actionModal.type}</button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
