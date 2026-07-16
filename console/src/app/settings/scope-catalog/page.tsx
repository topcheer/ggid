"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { BookMarked, Plus, X, Ban } from "lucide-react";

interface ScopeDef {
  name: string;
  description: string;
  risk_level: "low" | "medium" | "high";
  created_by: string;
  usage_count: number;
  used_by_clients: string[];
  deprecated: boolean;
}

const riskColors: Record<string, string> = {
  low: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ScopeCatalogPage() {
  const t = useTranslations();
  const [scopes, setScopes] = useState<ScopeDef[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ name: "", description: "", risk_level: "low" });

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/scope-catalog", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setScopes(d.scopes || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const add = async () => {
    if (!form.name) return;
    try { await fetch("/api/v1/oauth/scope-catalog", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowAdd(false); setForm({ name: "", description: "", risk_level: "low" }); fetchData(); }
    catch { /* noop */ }
  };

  const deprecate = async (name: string) => {
    try { await fetch("/api/v1/oauth/scope-catalog/" + name + "/deprecate", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><BookMarked className="w-6 h-6 text-blue-500" />{t("scopeCatalog.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage OAuth scope definitions with risk levels and usage tracking.</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> Add Scope</button>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Name</th><th className="px-4 py-3 text-left font-medium">Description</th><th className="px-4 py-3 text-left font-medium">Risk</th><th className="px-4 py-3 text-left font-medium">Usage</th><th className="px-4 py-3 text-left font-medium">Used By</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{scopes.map((s) => (<tr key={s.name} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{s.name}{s.deprecated && <span className="ml-1 text-xs text-red-500">(deprecated)</span>}</td><td className="px-4 py-3 text-xs text-gray-500">{s.description}</td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + riskColors[s.risk_level]}>{s.risk_level}</span></td><td className="px-4 py-3"><span className="font-bold">{s.usage_count}</span><span className="text-xs text-gray-400 ml-1">clients</span></td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{s.used_by_clients.slice(0, 2).map((c, i) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{c}</span>)}{s.used_by_clients.length > 2 && <span className="text-xs text-gray-400">+{s.used_by_clients.length - 2}</span>}</div></td><td className="px-4 py-3">{!s.deprecated && <button onClick={() => deprecate(s.name)} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Ban className="w-3 h-3" /> Deprecate</button>}</td></tr>))}{scopes.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No scopes defined.</td></tr>}</tbody>
        </table>
      </div>

      {showAdd && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Add Scope</h3><button onClick={() => setShowAdd(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Scope Name</label><input aria-label="read:users" type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="read:users" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Description</label><input aria-label="form" type="text" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Risk Level</label><select aria-label="form" value={form.risk_level} onChange={(e) => setForm({ ...form, risk_level: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option></select></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={add} disabled={!form.name} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">Add</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
