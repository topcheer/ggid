"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { Tag, Loader2, AlertCircle, X, Plus, Trash2, Save, Filter } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ResourceTagEntry {
  id: string; resource_path: string; resource_type: string;
  tags: Record<string, string>; updated_at: string;
}

export default function ResourceTagsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [tags, setTags] = useState<ResourceTagEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState("");
  const [showAssign, setShowAssign] = useState(false);
  const [assignPath, setAssignPath] = useState("");
  const [assignKey, setAssignKey] = useState("");
  const [assignVal, setAssignVal] = useState("");
  const [assigning, setAssigning] = useState(false);

  useState(() => { (async () => { try { setTags(await apiFetch<ResourceTagEntry[]>("/api/v1/policy/resource-tags").catch(() => [])); } catch { setError("Failed to load tags"); } finally { setLoading(false); } })(); });

  const handleAssign = async () => {
    if (!assignPath || !assignKey) return;
    setAssigning(true);
    try { await apiFetch("/api/v1/policy/resource-tags", { method: "POST", body: JSON.stringify({ resource_path: assignPath, tags: { [assignKey]: assignVal } }) }); setTags(await apiFetch<ResourceTagEntry[]>("/api/v1/policy/resource-tags").catch(() => tags)); setShowAssign(false); setAssignPath(""); setAssignKey(""); setAssignVal(""); }
    catch { setError("Assign failed"); } finally { setAssigning(false); }
  };
  const handleDelete = async (id: string) => { try { await apiFetch(`/api/v1/policy/resource-tags/${id}`, { method: "DELETE" }); setTags((p) => p.filter((t) => t.id !== id)); } catch { setError("Delete failed"); } };

  const filtered = filter ? tags.filter((t) => t.resource_path.includes(filter) || Object.values(t.tags).some((v) => v.includes(filter))) : tags;
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Tag className="h-6 w-6 text-purple-600" /> {t("resourceTags.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Tag resources with key-value metadata for policy targeting and filtering.</p></div><button onClick={() => setShowAssign(true)} className="flex items-center gap-2 rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700"><Plus className="h-4 w-4" /> Assign Tag</button></div>
      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}
      <div className="flex items-center gap-2"><Filter className="h-4 w-4 text-gray-400" /><input aria-label="Filter by resource or tag value..." value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="Filter by resource or tag value..." className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-600" /></div> : filtered.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Tag className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No tagged resources.</p></div></div> : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-800"><tr><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Resource</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Type</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Tags</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Updated</th><th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th></tr></thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">{filtered.map((t) => (<tr key={t.id} className="bg-white dark:bg-gray-900"><td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{t.resource_path}</td><td className="px-4 py-3 text-gray-500">{t.resource_type}</td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{Object.entries(t.tags).map(([k, v]) => <span key={k} className="rounded bg-purple-100 px-1.5 py-0.5 text-xs text-purple-600 dark:bg-purple-900/30">{k}: {v}</span>)}</div></td><td className="px-4 py-3 text-gray-400">{new Date(t.updated_at).toLocaleDateString()}</td><td className="px-4 py-3 text-right"><button onClick={() => handleDelete(t.id)} className="text-red-400 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></td></tr>))}</tbody>
          </table>
        </div>
      )}
      {showAssign && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowAssign(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">Assign Tag</h3><button onClick={() => setShowAssign(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Resource Path</label><input value={assignPath} onChange={(e) => setAssignPath(e.target.value)} placeholder="/api/v1/users/*" className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div className="flex gap-4"><div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Tag Key</label><input value={assignKey} onChange={(e) => setAssignKey(e.target.value)} placeholder="environment" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div><div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Tag Value</label><input value={assignVal} onChange={(e) => setAssignVal(e.target.value)} placeholder="production" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div></div>
              <button onClick={handleAssign} disabled={!assignPath || !assignKey || assigning} className="flex w-full items-center justify-center gap-2 rounded-lg bg-purple-600 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{assigning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Assign</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
