"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, ToggleLeft, ToggleRight,
  Sliders, Save,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface DetectionRule {
  id: string;
  rule_name: string;
  technique: string;
  description: string;
  enabled: boolean;
  threshold: number;
  time_window_minutes: number;
  last_triggered: string | null;
  triggers_24h: number;
}

export default function ITDRRulesPage() {
  const t = useTranslations();
  const [rules, setRules] = useState<DetectionRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [draft, setDraft] = useState<Partial<DetectionRule>>({});
  const [saving, setSaving] = useState(false);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const loadRules = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/audit/itdr/rules", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setRules(d.rules || d.items || []);
      }
    } catch { setError("Failed to load ITDR rules"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadRules(); }, [loadRules]);

  const toggleRule = async (id: string, enabled: boolean) => {
    setTogglingId(id);
    try {
      await fetch(`/api/v1/audit/itdr/rules/${id}`, {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ enabled: !enabled }),
      });
      setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !enabled } : r));
    } catch { setError("Failed to toggle rule"); }
    finally { setTogglingId(null); }
  };

  const saveEdit = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      await fetch(`/api/v1/audit/itdr/rules/${editing}`, {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify(draft),
      });
      setRules(prev => prev.map(r => r.id === editing ? { ...r, ...draft } as DetectionRule : r));
      setEditing(null); setDraft({});
    } catch { setError("Failed to save rule"); }
    finally { setSaving(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const enabledCount = rules.filter(r => r.enabled).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Shield className="h-6 w-6 text-purple-500" />
            ITDR Detection Rules
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Configure threat detection rules, thresholds, and enable/disable per tenant.</p>
        </div>
        <button onClick={loadRules} disabled={loading} aria-label="Refresh rules" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Total Rules</span><p className="mt-2 text-2xl font-bold">{rules.length}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Enabled</span><p className="mt-2 text-2xl font-bold text-green-600">{enabledCount}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Disabled</span><p className="mt-2 text-2xl font-bold text-gray-400">{rules.length - enabledCount}</p></div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-500" /></div> : rules.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Shield className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No detection rules configured.</p></div></div>
      ) : (
        <div className="space-y-3">
          {rules.map(r => (
            <div key={r.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">{r.rule_name}</span>
                    <span className="px-1.5 py-0.5 rounded text-xs font-mono bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">{r.technique}</span>
                    {r.triggers_24h > 0 && <span className="px-1.5 py-0.5 rounded text-xs bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400">{r.triggers_24h} triggers/24h</span>}
                  </div>
                  <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{r.description}</p>
                  <div className="mt-2 flex items-center gap-4 text-xs text-gray-400">
                    <span>Threshold: <span className="font-medium text-gray-600 dark:text-gray-300">{r.threshold}</span></span>
                    <span>Window: <span className="font-medium text-gray-600 dark:text-gray-300">{r.time_window_minutes}min</span></span>
                    {r.last_triggered && <span>Last: {new Date(r.last_triggered).toLocaleString()}</span>}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {editing === r.id ? (
                    <>
                      <input aria-label="Threshold" type="number" value={draft.threshold ?? r.threshold} onChange={e => setDraft({ ...draft, threshold: parseInt(e.target.value) || 0 })} className="w-16 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs" placeholder="Threshold" />
                      <input aria-label="Window minutes" type="number" value={draft.time_window_minutes ?? r.time_window_minutes} onChange={e => setDraft({ ...draft, time_window_minutes: parseInt(e.target.value) || 0 })} className="w-16 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs" placeholder="Window" />
                      <button onClick={saveEdit} disabled={saving} aria-label="Save" className="rounded-lg bg-green-600 p-1.5 text-white hover:bg-green-700 disabled:opacity-50">{saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />}</button>
                      <button onClick={() => { setEditing(null); setDraft({}); }} aria-label="Cancel edit" className="rounded-lg border p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"><X className="h-3.5 w-3.5" /></button>
                    </>
                  ) : (
                    <>
                      <button onClick={() => { setEditing(r.id); setDraft({ threshold: r.threshold, time_window_minutes: r.time_window_minutes }); }} aria-label={`Edit ${r.rule_name}`} className="rounded-lg border border-gray-300 p-1.5 text-gray-400 hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"><Sliders className="h-3.5 w-3.5" /></button>
                      <button onClick={() => toggleRule(r.id, r.enabled)} disabled={togglingId === r.id} aria-label={`${r.enabled ? "Disable" : "Enable"} ${r.rule_name}`} aria-pressed={r.enabled} className="flex items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium disabled:opacity-50">
                        {togglingId === r.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : r.enabled ? <ToggleRight className="h-5 w-5 text-green-500" /> : <ToggleLeft className="h-5 w-5 text-gray-400" />}
                        <span className={r.enabled ? "text-green-600 dark:text-green-400" : "text-gray-400"}>{r.enabled ? "Enabled" : "Disabled"}</span>
                      </button>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
