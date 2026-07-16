"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Shield, Plus, Trash2, X, AlertCircle, Loader2, Check, Globe,
  Clock, Smartphone, AlertTriangle, ShieldCheck,
} from "lucide-react";

interface Policy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  conditions: {
    ip_ranges?: string[];
    time_window?: { start: string; end: string };
    device_trusted?: boolean;
    min_risk_score?: number;
  };
  action: "allow" | "deny" | "require_mfa";
  priority: number;
}

const ACTION_CONFIG = {
  allow: { label: "Allow", icon: ShieldCheck, color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  deny: { label: "Deny", icon: X, color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
  require_mfa: { label: "Require MFA", icon: Smartphone, color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30" },
};

export default function ConditionalAccessPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<Policy | null>(null);
  const [form, setForm] = useState({ name: "", description: "", action: "require_mfa" as const, ip_ranges: "", time_start: "", time_end: "", device_trusted: false, min_risk_score: 0 });
  const [creating, setCreating] = useState(false);

  useState(() => {
    (async () => {
      try {
        const data = await apiFetch<{ policies?: Policy[]; items?: Policy[] }>("/api/v1/policy/conditional-access").catch(() => null);
        setPolicies(data?.policies ?? data?.items ?? []);
      } catch { setError("Failed to load policies"); }
      finally { setLoading(false); }
    })();
  });

  const handleCreate = async () => {
    if (!form.name.trim()) return;
    setCreating(true);
    try {
      await apiFetch("/api/v1/policy/conditional-access", {
        method: "POST",
        body: JSON.stringify({
          name: form.name, description: form.description, action: form.action,
          conditions: {
            ip_ranges: form.ip_ranges ? form.ip_ranges.split(",").map((s) => s.trim()) : undefined,
            time_window: form.time_start ? { start: form.time_start, end: form.time_end } : undefined,
            device_trusted: form.device_trusted || undefined,
            min_risk_score: form.min_risk_score || undefined,
          },
        }),
      });
      setForm({ name: "", description: "", action: "require_mfa", ip_ranges: "", time_start: "", time_end: "", device_trusted: false, min_risk_score: 0 });
      setShowCreate(false);
      const data = await apiFetch<{ policies?: Policy[]; items?: Policy[] }>("/api/v1/policy/conditional-access").catch(() => null);
      setPolicies(data?.policies ?? data?.items ?? []);
    } catch { setError("Failed to create policy"); }
    finally { setCreating(false); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/policy/conditional-access/${id}`, { method: "DELETE" }); setConfirmDelete(null); setPolicies((p) => p.filter((x) => x.id !== id)); }
    catch { setError("Failed to delete"); }
  };

  const handleToggle = async (p: Policy) => {
    try { await apiFetch(`/api/v1/policy/conditional-access/${p.id}`, { method: "PATCH", body: JSON.stringify({ enabled: !p.enabled }) }); setPolicies((prev) => prev.map((x) => x.id === p.id ? { ...x, enabled: !x.enabled } : x)); }
    catch { setError("Failed to toggle"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-indigo-600" />{t("conditionalAccess.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Context-aware access policies with IP, time, device, and risk conditions.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Policy</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : policies.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Shield className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No conditional access policies.</p></div></div>
      : (
        <div className="space-y-3">
          {policies.map((p) => {
            const cfg = ACTION_CONFIG[p.action]; const ActionIcon = cfg.icon;
            return (
              <div key={p.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-800 dark:text-gray-200">{p.name}</span>
                      <span className={`flex items-center gap-1 rounded-full ${cfg.bg} px-2 py-0.5 text-xs font-medium ${cfg.color}`}><ActionIcon className="h-3 w-3" />{cfg.label}</span>
                      {!p.enabled && <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">Disabled</span>}
                    </div>
                    {p.description && <p className="mt-1 text-sm text-gray-400">{p.description}</p>}
                    <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-400">
                      {p.conditions.ip_ranges && <span className="flex items-center gap-1"><Globe className="h-3 w-3" />{p.conditions.ip_ranges.join(", ")}</span>}
                      {p.conditions.time_window && <span className="flex items-center gap-1"><Clock className="h-3 w-3" />{p.conditions.time_window.start}–{p.conditions.time_window.end}</span>}
                      {p.conditions.device_trusted && <span className="flex items-center gap-1"><Smartphone className="h-3 w-3" />Trusted device</span>}
                      {p.conditions.min_risk_score !== undefined && <span className="flex items-center gap-1"><AlertTriangle className="h-3 w-3" />Risk ≥ {p.conditions.min_risk_score}</span>}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <label className="relative inline-flex cursor-pointer items-center"><input type="checkbox" checked={p.enabled} onChange={() => handleToggle(p)} className="peer sr-only" /><div className="h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:transition-all peer-checked:bg-indigo-600 peer-checked:after:translate-x-full dark:bg-gray-700" /></label>
                    <button onClick={() => setConfirmDelete(p)} aria-label="Delete policy" className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                  </div>
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
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold text-gray-900 dark:text-white">New Access Policy</h2><button onClick={() => setShowCreate(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Name</label><input value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Description</label><input value={form.description} onChange={(e) => setForm((p) => ({ ...p, description: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Action</label><select value={form.action} onChange={(e) => setForm((p) => ({ ...p, action: e.target.value as typeof form.action }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white">{Object.entries(ACTION_CONFIG).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}</select></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">IP Ranges (comma-separated)</label><input value={form.ip_ranges} onChange={(e) => setForm((p) => ({ ...p, ip_ranges: e.target.value }))} placeholder="10.0.0.0/8, 192.168.0.0/16" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Time Start</label><input type="time" value={form.time_start} onChange={(e) => setForm((p) => ({ ...p, time_start: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Time End</label><input type="time" value={form.time_end} onChange={(e) => setForm((p) => ({ ...p, time_end: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              </div>
              <label className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input type="checkbox" checked={form.device_trusted} onChange={(e) => setForm((p) => ({ ...p, device_trusted: e.target.checked }))} className="rounded border-gray-300 text-indigo-600" />Require trusted device</label>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Min Risk Score</label><input type="number" value={form.min_risk_score} onChange={(e) => setForm((p) => ({ ...p, min_risk_score: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
            </div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setShowCreate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={handleCreate} disabled={!form.name.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}Create</button></div>
          </div>
        </div>
      )}

      {confirmDelete && <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}><div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}><div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><div><h2 className="font-semibold text-gray-900 dark:text-white">Delete Policy?</h2><p className="text-sm text-gray-500"><strong>{confirmDelete.name}</strong> will be removed.</p></div></div><div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button></div></div></div>}
    </div>
  );
}
