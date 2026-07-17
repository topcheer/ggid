"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  FileCheck, Loader2, AlertCircle, X, Upload, Download, Plus, Trash2, Save,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ABACCondition {
  attribute: string;
  operator: string;
  value: string;
}

interface ABACPolicy {
  id: string;
  name: string;
  description: string;
  effect: "allow" | "deny";
  conditions: ABACCondition[];
  enabled: boolean;
  created_at: string;
}

export default function ABACPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [policies, setPolicies] = useState<ABACPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<ABACPolicy | null>(null);
  const [importText, setImportText] = useState("");
  const [showImport, setShowImport] = useState(false);

  useEffect(() => {
    (async () => {
      try { setPolicies(await apiFetch<ABACPolicy[]>("/api/v1/policy/abac/policies").catch(() => [])); }
      catch { setError("Failed to load ABAC policies"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleSave = async () => {
    if (!editing) return;
    try {
      if (editing.id) {
        await apiFetch(`/api/v1/policy/abac/policies/${editing.id}`, { method: "PUT", body: JSON.stringify(editing) });
      } else {
        const created = await apiFetch<ABACPolicy>("/api/v1/policy/abac/policies", { method: "POST", body: JSON.stringify(editing) });
        setPolicies((p) => [...p, created]);
      }
      setEditing(null);
      setPolicies(await apiFetch<ABACPolicy[]>("/api/v1/policy/abac/policies").catch(() => policies));
    } catch { setError("Save failed"); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/policy/abac/policies/${id}`, { method: "DELETE" }); setPolicies((p) => p.filter((x) => x.id !== id)); }
    catch { setError("Delete failed"); }
  };

  const handleExport = () => {
    const blob = new Blob([JSON.stringify(policies, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = "abac-policies.json"; a.click();
    URL.revokeObjectURL(url);
  };

  const handleImport = async () => {
    try {
      const data = JSON.parse(importText);
      const arr = Array.isArray(data) ? data : [data];
      for (const p of arr) {
        await apiFetch("/api/v1/policy/abac/policies", { method: "POST", body: JSON.stringify({ ...p, id: "" }) });
      }
      setPolicies(await apiFetch<ABACPolicy[]>("/api/v1/policy/abac/policies").catch(() => policies));
      setShowImport(false); setImportText("");
    } catch { setError("Import failed — invalid JSON"); }
  };

  const addCondition = () => {
    if (!editing) return;
    setEditing({ ...editing, conditions: [...editing.conditions, { attribute: "", operator: "eq", value: "" }] });
  };

  const updateCondition = (idx: number, field: keyof ABACCondition, val: string) => {
    if (!editing) return;
    setEditing({ ...editing, conditions: editing.conditions.map((c, i) => i === idx ? { ...c, [field]: val } : c) });
  };

  const removeCondition = (idx: number) => {
    if (!editing) return;
    setEditing({ ...editing, conditions: editing.conditions.filter((_, i) => i !== idx) });
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><FileCheck className="h-6 w-6 text-indigo-600" /> {t("abac.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Attribute-Based Access Control policy management with import/export.</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={handleExport} className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"><Download className="h-4 w-4" /> Export</button>
          <button onClick={() => setShowImport(true)} className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"><Upload className="h-4 w-4" /> Import</button>
          <button onClick={() => setEditing({ id: "", name: "", description: "", effect: "allow", conditions: [], enabled: true, created_at: "" })} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Policy</button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : policies.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><FileCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No ABAC policies yet.</p></div></div>
      ) : (
        <div className="space-y-3">
          {policies.map((p) => (
            <div key={p.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="font-semibold text-gray-900 dark:text-white">{p.name}</h3>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${p.effect === "allow" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"}`}>{p.effect}</span>
                    {!p.enabled && <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">disabled</span>}
                  </div>
                  {p.description && <p className="mt-1 text-sm text-gray-500">{p.description}</p>}
                  {p.conditions.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-2">
                      {p.conditions.map((c, i) => (
                        <span key={i} className="rounded bg-gray-100 px-2 py-1 text-xs font-mono text-gray-600 dark:bg-gray-700 dark:text-gray-300">{c.attribute} {c.operator} {c.value}</span>
                      ))}
                    </div>
                  )}
                </div>
                <div className="flex gap-1">
                  <button onClick={() => setEditing({ ...p })} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-indigo-600 dark:hover:bg-gray-700"><FileCheck className="h-4 w-4" /></button>
                  <button onClick={() => handleDelete(p.id)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Import modal */}
      {showImport && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowImport(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">Import Policies (JSON)</h3><button onClick={() => setShowImport(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <textarea aria-label="Import text" value={importText} onChange={(e) => setImportText(e.target.value)} placeholder='[{ "name": "...", "effect": "allow", "conditions": [...] }]' className="h-48 w-full rounded-lg border border-gray-300 p-3 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
            <button onClick={handleImport} disabled={!importText.trim()} className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"><Upload className="h-4 w-4" /> Import</button>
          </div>
        </div>
      )}

      {/* Edit modal */}
      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditing(null)}>
          <div role="dialog" aria-modal="true" className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{editing.id ? "Edit Policy" : "New Policy"}</h3><button onClick={() => setEditing(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Name</label><input aria-label="editing" value={editing.name} onChange={(e) => setEditing({ ...editing, name: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Description</label><input aria-label="editing" value={editing.description} onChange={(e) => setEditing({ ...editing, description: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Effect</label><select aria-label="editing" value={editing.effect} onChange={(e) => setEditing({ ...editing, effect: e.target.value as "allow" | "deny" })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="allow">Allow</option><option value="deny">Deny</option></select></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Enabled</label><button onClick={() => setEditing({ ...editing, enabled: !editing.enabled })} className={`flex h-[38px] items-center rounded-lg px-4 text-sm font-medium ${editing.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{editing.enabled ? "Enabled" : "Disabled"}</button></div>
              </div>
              <div>
                <div className="mb-2 flex items-center justify-between"><label className="text-xs font-semibold uppercase text-gray-400">Conditions</label><button onClick={addCondition} className="flex items-center gap-1 text-xs text-indigo-600 hover:underline"><Plus className="h-3 w-3" /> Add</button></div>
                <div className="space-y-2">
                  {editing.conditions.map((c, idx) => (
                    <div key={idx} className="flex items-center gap-2">
                      <input aria-label="attribute" value={c.attribute} onChange={(e) => updateCondition(idx, "attribute", e.target.value)} placeholder="attribute" className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
                      <select aria-label="Select option" value={c.operator} onChange={(e) => updateCondition(idx, "operator", e.target.value)} className="rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="eq">==</option><option value="ne">!=</option><option value="in">in</option><option value="lt">&lt;</option><option value="gt">&gt;</option><option value="contains">contains</option></select>
                      <input aria-label="value" value={c.value} onChange={(e) => updateCondition(idx, "value", e.target.value)} placeholder="value" className="flex-1 rounded border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
                      <button onClick={() => removeCondition(idx)} className="rounded p-1 text-red-400 hover:bg-red-50"><Trash2 className="h-3 w-3" /></button>
                    </div>
                  ))}
                </div>
                {editing.conditions.length === 0 && <p className="text-xs text-gray-400">No conditions. Policy will match all requests.</p>}
              </div>
              <button onClick={handleSave} className="flex w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Save className="h-4 w-4" /> Save Policy</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
