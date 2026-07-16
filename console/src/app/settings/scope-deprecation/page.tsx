"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Ban, Plus, X, Save, AlertTriangle } from "lucide-react";

interface DeprecatedScope {
  id: string;
  name: string;
  deprecated_at: string;
  sunset_date: string;
  replacement: string;
  usage_count: number;
  status: "active" | "deprecated" | "sunset";
}

interface AvailableScope {
  name: string;
}

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  deprecated: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  sunset: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ScopeDeprecationPage() {
  const t = useTranslations();
  const [scopes, setScopes] = useState<DeprecatedScope[]>([]);
  const [available, setAvailable] = useState<AvailableScope[]>([]);
  const [loading, setLoading] = useState(false);
  const [showDeprecate, setShowDeprecate] = useState(false);
  const [form, setForm] = useState({ name: "", replacement: "", sunset_date: "" });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/scope-deprecation", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setScopes(data.scopes || data || []); setAvailable(data.available_scopes || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const deprecate = async () => {
    if (!form.name) return;
    try { await fetch("/api/v1/oauth/scope-deprecation", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowDeprecate(false); setForm({ name: "", replacement: "", sunset_date: "" }); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Ban className="w-6 h-6 text-orange-500" />{t("scopeDeprecation.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage deprecated OAuth scopes with sunset dates and replacements.</p></div>
        <button onClick={() => setShowDeprecate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> Deprecate Scope</button>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Scope</th><th className="px-4 py-3 text-left font-medium">Deprecated</th><th className="px-4 py-3 text-left font-medium">Sunset Date</th><th className="px-4 py-3 text-left font-medium">Replacement</th><th className="px-4 py-3 text-left font-medium">Usage</th><th className="px-4 py-3 text-left font-medium">Status</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {scopes.map((s) => (
              <tr key={s.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs font-medium">{s.name}</td>
                <td className="px-4 py-3 text-gray-500">{s.deprecated_at || "-"}</td>
                <td className="px-4 py-3 text-gray-500">{s.sunset_date || "-"}</td>
                <td className="px-4 py-3 font-mono text-xs text-blue-600">{s.replacement || "-"}</td>
                <td className="px-4 py-3">{s.usage_count > 0 ? <span className="px-2 py-0.5 rounded text-xs bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400">{s.usage_count} uses</span> : <span className="text-xs text-green-600">Unused</span>}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[s.status]}`}>{s.status}</span></td>
              </tr>
            ))}
            {scopes.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No deprecated scopes.</td></tr>}
          </tbody>
        </table>
      </div>

      {showDeprecate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowDeprecate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><Ban className="w-5 h-5 text-orange-500" /> Deprecate Scope</h3><button onClick={() => setShowDeprecate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Scope to Deprecate</label><input aria-label="old:scope" type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="old:scope" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Replacement Scope</label><select aria-label="form" value={form.replacement} onChange={(e) => setForm({ ...form, replacement: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="">None</option>{available.map((s) => <option key={s.name} value={s.name}>{s.name}</option>)}</select></div>
              <div><label className="text-sm font-medium">Sunset Date</label><input aria-label="form" type="date" value={form.sunset_date} onChange={(e) => setForm({ ...form, sunset_date: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowDeprecate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={deprecate} disabled={!form.name} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 disabled:opacity-50 flex items-center gap-1"><Save className="w-4 h-4" /> Deprecate</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
