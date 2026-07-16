"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import { GitFork, Loader2, AlertCircle, X, ChevronRight, Folder, Lock, Plus, AlertOctagon } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface PolicyNode {
  id: string; name: string; parent_id: string;
  has_override: boolean; override_description: string;
  effect: "allow" | "deny"; children: PolicyNode[];
}

export default function PolicyInheritancePage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [policies, setPolicies] = useState<PolicyNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [selected, setSelected] = useState<PolicyNode | null>(null);
  const [overrideDesc, setOverrideDesc] = useState("");
  const [saving, setSaving] = useState(false);

  useState(() => { (async () => { try { setPolicies(await apiFetch<PolicyNode[]>("/api/v1/policy/inheritance").catch(() => [])); } catch { setError("Failed to load policy tree"); } finally { setLoading(false); } })(); });

  const toggleExpand = (id: string) => setExpanded((prev) => { const n = new Set(prev); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const handleSaveOverride = async () => {
    if (!selected) return;
    setSaving(true);
    try { await apiFetch(`/api/v1/policy/inheritance/${selected.id}/override`, { method: "POST", body: JSON.stringify({ description: overrideDesc }) }); setPolicies(await apiFetch<PolicyNode[]>("/api/v1/policy/inheritance").catch(() => policies)); setSelected(null); setOverrideDesc(""); }
    catch { setError("Override failed"); } finally { setSaving(false); }
  };

  const renderNode = (node: PolicyNode, depth: number): React.ReactNode => {
    const hasChildren = node.children.length > 0;
    return (<div key={node.id}>{<div className={`flex items-center gap-2 rounded px-2 py-1.5 hover:bg-gray-50 dark:hover:bg-gray-800 ${selected?.id === node.id ? "bg-indigo-50 dark:bg-indigo-900/20" : ""}`} style={{ paddingLeft: `${depth * 20 + 8}px` }}>{hasChildren ? <button onClick={() => toggleExpand(node.id)}><ChevronRight className={`h-3 w-3 text-gray-400 transition-transform ${expanded.has(node.id) ? "rotate-90" : ""}`} /></button> : <span className="w-3" />}{depth === 0 ? <Folder className="h-4 w-4 text-blue-400" /> : <Lock className="h-4 w-4 text-gray-400" />}<span className={`flex-1 text-sm ${node.has_override ? "font-medium text-orange-600" : "text-gray-700 dark:text-gray-300"}`}>{node.name}</span><span className={`rounded px-1.5 py-0.5 text-xs ${node.effect === "allow" ? "bg-green-100 text-green-600 dark:bg-green-900/30" : "bg-red-100 text-red-600 dark:bg-red-900/30"}`}>{node.effect}</span>{node.has_override && <span className="flex items-center gap-0.5 text-xs text-orange-500"><AlertOctagon className="h-3 w-3" />override</span>}<button onClick={() => { setSelected(node); setOverrideDesc(node.override_description); }} className="text-xs text-indigo-600 hover:underline">Override</button></div>}{expanded.has(node.id) && node.children.map((c) => renderNode(c, depth + 1))}</div>);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><GitFork className="h-6 w-6 text-blue-600" /> {t("policyInheritance.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Policy tree with parent-child inheritance and override management.</p></div>
      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-600" /></div> : policies.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><GitFork className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No policy tree.</p></div></div> : <div className={cardCls}><h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Policy Tree</h3><div className="max-h-[500px] overflow-y-auto">{policies.map((p) => renderNode(p, 0))}</div></div>}
      {/* Override modal */}
      {selected && (<div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setSelected(null)}><div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}><div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">Override: {selected.name}</h3><button onClick={() => setSelected(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div><div className="mb-3 rounded-lg bg-gray-50 p-3 text-sm dark:bg-gray-900"><span className="text-gray-400">Current effect:</span> <span className={`font-medium ${selected.effect === "allow" ? "text-green-600" : "text-red-600"}`}>{selected.effect}</span>{selected.parent_id && <div className="mt-1 text-xs text-gray-400">Inherited from parent</div>}</div><div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Override Description</label><textarea value={overrideDesc} onChange={(e) => setOverrideDesc(e.target.value)} rows={3} placeholder="Reason for overriding inherited policy..." className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div><button onClick={handleSaveOverride} disabled={saving} className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} Create Override</button></div></div>)}
    </div>
  );
}
