"use client";

import { useState, useEffect, useCallback } from "react";
import { Network, Plus, X, ChevronRight, Search } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface OrgUnit {
  id: string;
  name: string;
  member_count: number;
  budget: number;
  manager: string;
  children: OrgUnit[];
}

export default function OrgTreePage() {
  const t = useTranslations();

  const [tree, setTree] = useState<OrgUnit[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const [showAdd, setShowAdd] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/org/tree", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setTree(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const toggle = (id: string) => { const n = new Set(expanded); n.has(id) ? n.delete(id) : n.add(id); setExpanded(n); };

  const renderNode = (node: OrgUnit, depth: number = 0): React.ReactNode => {
    if (search && !node.name.toLowerCase().includes(search.toLowerCase())) { return node.children.flatMap((c) => c ? [renderNode(c, depth)] : []).find((x) => x) || null; }
    return (<div key={node.id}>
      <div className={"flex items-center gap-2 py-2 hover:bg-gray-50 dark:hover:bg-gray-900/30 rounded " + (depth > 0 ? "ml-" + (depth * 4) : "")} style={{ paddingLeft: depth * 20 }}>
        {node.children.length > 0 ? <button onClick={() => toggle(node.id)}><ChevronRight className={"w-4 h-4 text-gray-400 transition-transform " + (expanded.has(node.id) ? "rotate-90" : "")} /></button> : <span className="w-4" />}
        <Network className="w-4 h-4 text-blue-400" /><div className="flex-1"><span className="text-sm font-medium">{node.name}</span><span className="text-xs text-gray-400 ml-2">{node.member_count} members</span><span className="text-xs text-gray-400 ml-2">Manager: {node.manager}</span></div><span className="text-xs font-mono text-gray-400">${(node.budget / 1000).toFixed(0)}k</span>
      </div>
      {expanded.has(node.id) && node.children.map((c) => renderNode(c, depth + 1))}
    </div>);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-blue-500" /> {t("orgTree.title")}</h1><p className="text-sm text-gray-500 mt-1">Hierarchical organization structure with member counts and budgets.</p></div>
        <button onClick={() => setShowAdd(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> Add Unit</button>
      </div>

      <div className="relative max-w-xs"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search units..." className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>

      <div className="rounded-lg border dark:border-gray-800 p-4">{tree.map((n) => renderNode(n))}{tree.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No data.</p>}</div>

      {showAdd && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowAdd(false)}><div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
          <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Add Org Unit</h3><button onClick={() => setShowAdd(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
          <div className="px-6 py-4 space-y-3"><input type="text" placeholder="Unit name" className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /><input type="text" placeholder="Manager" className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /><input type="number" placeholder="Budget ($)" className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
          <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button><button onClick={() => setShowAdd(false)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">Add</button></div>
        </div></div>
      )}
    </div>
  );
}
