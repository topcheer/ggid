"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import { Zap, Loader2, AlertCircle, X, Plus, Trash2, Save, Play, ToggleLeft, ToggleRight } from "lucide-react";

interface CorrelationRule {
  id: string; name: string; pattern: string; window_minutes: number;
  threshold: number; enabled: boolean; action: string;
  created_at: string; last_triggered: string; trigger_count: number;
}

export default function CorrelationRulesPage() {
  const { apiFetch } = useApi();
  const [rules, setRules] = useState<CorrelationRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<CorrelationRule | null>(null);
  const [testing, setTesting] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ matched: boolean; matches: number; ruleId: string } | null>(null);

  useState(() => { (async () => { try { setRules(await apiFetch<CorrelationRule[]>("/api/v1/audit/correlation-rules").catch(() => [])); } catch { setError("Failed to load rules"); } finally { setLoading(false); } })(); });

  const handleSave = async () => {
    if (!editing) return;
    try {
      if (editing.id) { await apiFetch(`/api/v1/audit/correlation-rules/${editing.id}`, { method: "PUT", body: JSON.stringify(editing) }); }
      else { await apiFetch("/api/v1/audit/correlation-rules", { method: "POST", body: JSON.stringify(editing) }); }
      setEditing(null); setRules(await apiFetch<CorrelationRule[]>("/api/v1/audit/correlation-rules").catch(() => rules));
    } catch { setError("Save failed"); }
  };
  const handleDelete = async (id: string) => { try { await apiFetch(`/api/v1/audit/correlation-rules/${id}`, { method: "DELETE" }); setRules((p) => p.filter((r) => r.id !== id)); } catch { setError("Delete failed"); } };
  const handleTest = async (id: string) => { setTesting(id); setTestResult(null); try { const r = await apiFetch<{ matched: boolean; matches: number }>(`/api/v1/audit/correlation-rules/${id}/test`, { method: "POST" }); setTestResult({ ...r, ruleId: id }); } catch { setError("Test failed"); } finally { setTesting(null); } };
  const toggle = (r: CorrelationRule) => { setRules((prev) => prev.map((x) => x.id === r.id ? { ...x, enabled: !x.enabled } : x)); apiFetch(`/api/v1/audit/correlation-rules/${r.id}`, { method: "PUT", body: JSON.stringify({ enabled: !r.enabled }) }).catch(() => {}); };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Zap className="h-6 w-6 text-orange-600" /> Correlation Rules</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Event correlation rules with pattern matching, time windows, and thresholds.</p></div><button onClick={() => setEditing({ id: "", name: "", pattern: "", window_minutes: 5, threshold: 3, enabled: true, action: "alert", created_at: "", last_triggered: "", trigger_count: 0 })} className="flex items-center gap-2 rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700"><Plus className="h-4 w-4" /> New Rule</button></div>
      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-orange-600" /></div> : rules.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Zap className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No correlation rules.</p></div></div> : (
        <div className="space-y-2">{rules.map((r) => (
          <div key={r.id} className={cardCls}>
            <div className="flex items-start justify-between">
              <div className="flex-1"><div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{r.name}</span><span className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-gray-700">action: {r.action}</span>{r.trigger_count > 0 && <span className="rounded bg-orange-100 px-1.5 py-0.5 text-xs text-orange-600 dark:bg-orange-900/30">{r.trigger_count} triggers</span>}</div>
                <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-gray-400"><span className="font-mono">pattern: {r.pattern}</span><span>window: {r.window_minutes}m</span><span>threshold: {r.threshold}</span></div>
              </div>
              <div className="flex items-center gap-2"><button onClick={() => handleTest(r.id)} disabled={testing === r.id} className="text-xs text-indigo-600 hover:underline">{testing === r.id ? <Loader2 className="inline h-3 w-3 animate-spin" /> : <Play className="inline h-3 w-3" />} Test</button><button onClick={() => setEditing({ ...r })} className="text-xs text-gray-500 hover:text-orange-600">Edit</button><button onClick={() => toggle(r)}>{r.enabled ? <ToggleRight className="h-6 w-6 text-green-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button><button onClick={() => handleDelete(r.id)} className="text-red-400 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></div>
            </div>
            {testResult && testResult.ruleId === r.id && <div className={`mt-2 rounded-lg px-3 py-1.5 text-sm ${testResult.matched ? "bg-orange-50 text-orange-700 dark:bg-orange-900/20" : "bg-gray-50 text-gray-500 dark:bg-gray-900"}`}>{testResult.matched ? `Matched ${testResult.matches} events` : "No matches found"}</div>}
          </div>
        ))}</div>
      )}
      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditing(null)}>
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{editing.id ? "Edit Rule" : "New Rule"}</h3><button onClick={() => setEditing(null)}><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Name</label><input value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Pattern (regex or event type)</label><input value={editing.pattern} onChange={(e) => setEditing({ ...editing, pattern: e.target.value })} placeholder="failed_login" className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Window (min)</label><input type="number" value={editing.window_minutes} onChange={(e) => setEditing({ ...editing, window_minutes: parseInt(e.target.value) || 5 })} min={1} max={1440} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Threshold</label><input type="number" value={editing.threshold} onChange={(e) => setEditing({ ...editing, threshold: parseInt(e.target.value) || 3 })} min={1} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Action</label><select value={editing.action} onChange={(e) => setEditing({ ...editing, action: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="alert">Alert</option><option value="block">Block</option><option value="notify">Notify</option><option value="incident">Create Incident</option></select></div>
              </div>
              <button onClick={handleSave} className="flex w-full items-center justify-center gap-2 rounded-lg bg-orange-600 py-2 text-sm font-medium text-white hover:bg-orange-700"><Save className="h-4 w-4" /> Save Rule</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
