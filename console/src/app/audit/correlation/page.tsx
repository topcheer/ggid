"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Zap, Plus, Trash2, X, AlertCircle, Loader2, Check, Clock,
  Activity, AlertTriangle, ShieldAlert,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface CorrelationRule {
  id: string;
  name: string;
  description: string;
  event_pattern: string;
  time_window_minutes: number;
  threshold: number;
  severity: "critical" | "high" | "medium" | "low";
  enabled: boolean;
  last_triggered?: string;
}

interface CorrelatedGroup {
  id: string;
  rule_name: string;
  severity: "critical" | "high" | "medium" | "low";
  event_count: number;
  events: { id: string; type: string; user: string; ip: string; timestamp: string }[];
  first_event: string;
  last_event: string;
  description: string;
}

const SEV_CONFIG = {
  critical: { icon: ShieldAlert, color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", label: "Critical" },
  high: { icon: AlertTriangle, color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30", label: "High" },
  medium: { icon: Zap, color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", label: "Medium" },
  low: { icon: Activity, color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30", label: "Low" },
};

export default function EventCorrelationPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [rules, setRules] = useState<CorrelationRule[]>([]);
  const [results, setResults] = useState<CorrelatedGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [analyzing, setAnalyzing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<"rules" | "results">("rules");
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [form, setForm] = useState({ name: "", event_pattern: "", time_window_minutes: 15, threshold: 5, severity: "medium" as const });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ rules?: CorrelationRule[]; items?: CorrelationRule[] }>("/api/v1/audit/correlation/rules").catch(() => null);
      setRules(data?.rules ?? data?.items ?? []);
    } catch {
      setError("Failed to load correlation rules");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleAnalyze = async () => {
    setAnalyzing(true);
    try {
      const data = await apiFetch<{ groups?: CorrelatedGroup[]; items?: CorrelatedGroup[] }>("/api/v1/audit/correlation/analyze", { method: "POST" }).catch(() => null);
      setResults(data?.groups ?? data?.items ?? []);
      setTab("results");
    } catch {
      setError("Failed to analyze events");
    } finally {
      setAnalyzing(false);
    }
  };

  const handleCreate = async () => {
    if (!form.name.trim() || !form.event_pattern.trim()) return;
    setCreating(true);
    try {
      await apiFetch("/api/v1/audit/correlation/rules", { method: "POST", body: JSON.stringify(form) });
      setForm({ name: "", event_pattern: "", time_window_minutes: 15, threshold: 5, severity: "medium" });
      setShowCreate(false);
      await load();
    } catch {
      setError("Failed to create rule");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/audit/correlation/rules/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError("Failed to delete rule");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Zap className="h-6 w-6 text-indigo-600" /> Event Correlation
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Detect patterns across audit events using rules with time windows and thresholds.</p>
        </div>
        <div className="flex gap-2">
          <button onClick={handleAnalyze} disabled={analyzing} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            {analyzing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Activity className="h-4 w-4" />} Analyze
          </button>
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Rule</button>
        </div>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        <button onClick={() => setTab("rules")} className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "rules" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400"}`}><Zap className="h-4 w-4" /> Rules ({rules.length})</button>
        <button onClick={() => setTab("results")} className={`flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === "results" ? "border-indigo-600 text-indigo-600" : "border-transparent text-gray-400"}`}><Activity className="h-4 w-4" /> Results ({results.length})</button>
      </div>

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : tab === "rules" ? (
        rules.length === 0 ? (
          <div className={cardCls}><div className="py-12 text-center"><Zap className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No correlation rules defined.</p></div></div>
        ) : (
          <div className="space-y-3">
            {rules.map((r: any) => {
              const sev = SEV_CONFIG[r.severity];
              const SevIcon = sev.icon;
              return (
                <div key={r.id} className={cardCls}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-gray-800 dark:text-gray-200">{r.name}</span>
                        <span className={`flex items-center gap-1 rounded-full ${sev.bg} px-2 py-0.5 text-xs font-medium ${sev.color}`}><SevIcon className="h-3 w-3" />{sev.label}</span>
                        {!r.enabled && <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">Disabled</span>}
                      </div>
                      {r.description && <p className="mt-1 text-sm text-gray-400">{r.description}</p>}
                      <div className="mt-2 flex items-center gap-4 text-xs text-gray-400">
                        <span className="font-mono rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700">{r.event_pattern}</span>
                        <span className="flex items-center gap-1"><Clock className="h-3 w-3" />{r.time_window_minutes}min window</span>
                        <span>Threshold: {r.threshold}</span>
                        {r.last_triggered && <span>Last: {new Date(r.last_triggered).toLocaleString()}</span>}
                      </div>
                    </div>
                    <button onClick={() => setConfirmDelete(r.id)} aria-label="Delete rule" className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                  </div>
                </div>
              );
            })}
          </div>
        )
      ) : results.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Activity className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No correlated events. Run analysis to detect patterns.</p></div></div>
      ) : (
        <div className="space-y-4">
          {results.map((g: any) => {
            const sev = SEV_CONFIG[g.severity];
            const SevIcon = sev.icon;
            return (
              <div key={g.id} className={`${cardCls} ${g.severity === "critical" ? "border-red-200 dark:border-red-800" : ""}`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className={`rounded-lg ${sev.bg} p-1.5`}><SevIcon className={`h-4 w-4 ${sev.color}`} /></div>
                    <div>
                      <span className="font-medium text-gray-800 dark:text-gray-200">{g.rule_name}</span>
                      <p className="text-xs text-gray-400">{g.description}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <span className={`text-lg font-bold ${sev.color}`}>{g.event_count}</span>
                    <span className="text-xs text-gray-400"> events</span>
                  </div>
                </div>
                <div className="mt-3 space-y-1">
                  {g.events.slice(0, 5).map((e: any) => (
                    <div key={e.id} className="flex items-center gap-3 rounded-lg bg-gray-50 px-3 py-1.5 text-xs dark:bg-gray-900/30">
                      <span className="font-mono text-gray-500">{e.type}</span>
                      <span className="text-gray-400">{e.user}</span>
                      <span className="font-mono text-gray-400">{e.ip}</span>
                      <span className="ml-auto text-gray-400">{new Date(e.timestamp).toLocaleTimeString()}</span>
                    </div>
                  ))}
                  {g.events.length > 5 && <p className="text-center text-xs text-gray-400">+ {g.events.length - 5} more</p>}
                </div>
                <div className="mt-2 flex items-center gap-3 text-xs text-gray-400">
                  <span>From: {new Date(g.first_event).toLocaleString()}</span>
                  <span>To: {new Date(g.last_event).toLocaleString()}</span>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => !creating && setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">New Correlation Rule</h2>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Name</label><input aria-label="Brute-force detection" value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} placeholder="Brute-force detection" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Event Pattern (regex)</label><input aria-label="login.failed.*" value={form.event_pattern} onChange={(e) => setForm((p) => ({ ...p, event_pattern: e.target.value }))} placeholder="login.failed.*" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div className="grid grid-cols-3 gap-3">
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Window (min)</label><input aria-label="form" type="number" value={form.time_window_minutes} onChange={(e) => setForm((p) => ({ ...p, time_window_minutes: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Threshold</label><input aria-label="form" type="number" value={form.threshold} onChange={(e) => setForm((p) => ({ ...p, threshold: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Severity</label><select aria-label="form" value={form.severity} onChange={(e) => setForm((p) => ({ ...p, severity: e.target.value as typeof form.severity }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white">{Object.entries(SEV_CONFIG).map(([k, v]: any[]) => <option key={k} value={k}>{v.label}</option>)}</select></div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleCreate} disabled={!form.name.trim() || !form.event_pattern.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><p className="text-sm text-gray-500">Delete this correlation rule?</p></div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleDelete(confirmDelete)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
