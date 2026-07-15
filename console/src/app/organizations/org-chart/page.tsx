"use client";
import { useState, useEffect, useCallback } from "react";
import { Network, Search, ChevronDown, ChevronRight, Building2, User as UserIcon, Briefcase, AlertTriangle, RotateCcw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface OrgNode { id: string; name: string; title: string; email: string; manager_id: string | null; department: string; children?: OrgNode[]; }
interface TreeNodeProps { node: OrgNode; depth: number; collapsedIds: Set<string | null>; toggleNode: (id: string) => void; highlight: string; }
function TreeNode({ node, depth, collapsedIds, toggleNode, highlight }: TreeNodeProps) {
  const collapsed = collapsedIds.has(node.id);
  const hasChildren = node.children && node.children.length > 0;
  const isMatch = highlight && (node.name.toLowerCase().includes(highlight.toLowerCase()) || node.title.toLowerCase().includes(highlight.toLowerCase()));
  return (
    <div className="relative">
      <div className={`flex items-center gap-2 py-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900/30 ${isMatch ? "bg-yellow-50 dark:bg-yellow-900/20" : ""}`} style={{ paddingLeft: `${depth * 24 + 8}px` }}>
        {hasChildren ? (
          <button onClick={() => toggleNode(node.id)} aria-label={collapsed ? `Expand ${node.name}` : `Collapse ${node.name}`} className="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-800">
            {collapsed ? <ChevronRight className="w-4 h-4 text-gray-400" /> : <ChevronDown className="w-4 h-4 text-gray-400" />}
          </button>
        ) : (
          <div className="w-5" />
        )}
        <div className="w-8 h-8 rounded-full bg-blue-50 dark:bg-blue-900/20 flex items-center justify-center flex-shrink-0">
          <UserIcon className="w-4 h-4 text-blue-500" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">{node.name}</span>
            <span className="text-xs text-gray-400 flex items-center gap-0.5"><Briefcase className="w-3 h-3" />{node.title}</span>
          </div>
          <div className="flex items-center gap-2 text-xs text-gray-400">
            <span className="flex items-center gap-0.5"><Building2 className="w-3 h-3" />{node.department}</span>
            <span>·</span>
            <span className="font-mono">{node.email}</span>
          </div>
        </div>
      </div>
      {hasChildren && !collapsed && (
        <div className="relative">
          <div className="absolute left-[12px] top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" style={{ marginLeft: `${depth * 24}px` }} />
          {node.children!.map((child) => (
            <TreeNode key={child.id} node={child} depth={depth + 1} collapsedIds={collapsedIds} toggleNode={toggleNode} highlight={highlight} />
          ))}
        </div>
      )}
    </div>
  );
}
export default function OrgChartPage() {
  const t = useTranslations();
  const [orgs, setOrgs] = useState<{ id: string; name: string }[]>([]);
  const [selectedOrg, setSelectedOrg] = useState("");
  const [tree, setTree] = useState<OrgNode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [collapsedIds, setCollapsedIds] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  useEffect(() => {
    setError(null);
    fetch("/api/v1/org/orgs", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).then(async (res) => {
      if (!res.ok) { setError(`Failed to load orgs: HTTP ${res.status}`); return; }
      const data = await res.json();
      setOrgs(data.orgs || data || []);
    }).catch((e) => setError(e instanceof Error ? e.message : "Failed to load orgs"));
  }, []);
  const fetchTree = useCallback(async (orgId: string) => {
    if (!orgId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/org/orgs/${orgId}/chart`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) throw new Error(`Failed to load org chart: HTTP ${res.status}`);
      setTree(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load org chart"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { if (selectedOrg) fetchTree(selectedOrg); }, [selectedOrg, fetchTree]);
  const toggleNode = (id: string) => {
    setCollapsedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };
  const countNodes = (n: OrgNode | null): number => {
    if (!n) return 0;
    return 1 + (n.children || []).reduce((s, c) => s + countNodes(c), 0);
  };
  if (error) {
    return (
      <div className="p-8">
        <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
          <p className="text-red-700 dark:text-red-400 text-sm font-medium flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> Error: {error}</p>
          <button onClick={() => { setError(null); if (selectedOrg) fetchTree(selectedOrg); }} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button>
        </div>
      </div>
    );
  }
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-blue-500" /> {t("organizationsOrgChart.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Interactive org chart with expand/collapse and person search.</p>
      </div>
      <div className="flex items-center gap-3 flex-wrap">
        <select value={selectedOrg} onChange={(e) => setSelectedOrg(e.target.value)} aria-label="Select organization" className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">Select an org...</option>
          {orgs.map((o) => (<option key={o.id} value={o.id}>{o.name}</option>
          ))}
        </select>
        {tree && <span className="text-xs text-gray-500">{countNodes(tree)} people</span>}
        <div className="relative flex-1 max-w-xs ml-auto">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input type="text" placeholder="Search person..." value={search} onChange={(e) => setSearch(e.target.value)} aria-label="Search person" className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        {tree && (
          <div className="flex items-center gap-2">
            <button onClick={() => setCollapsedIds(new Set())} aria-label="Expand all nodes" className="text-xs text-blue-600 hover:underline">Expand All</button>
            <button onClick={() => {
              const allIds = new Set<string>();
              const collect = (n: OrgNode) => { allIds.add(n.id); (n.children || []).forEach(collect); };
              if (tree) collect(tree);
              setCollapsedIds(allIds);
            }} aria-label="Collapse all nodes" className="text-xs text-blue-600 hover:underline">Collapse All</button>
          </div>
        )}
      </div>
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading org chart...</div></div>}
      {tree && !loading && (
        <div className="rounded-lg border dark:border-gray-800 p-2 max-h-[600px] overflow-y-auto">
          <TreeNode node={tree} depth={0} collapsedIds={collapsedIds} toggleNode={toggleNode} highlight={search} />
        </div>
      )}
      {!tree && !loading && <p className="text-sm text-gray-500 text-center py-8">Select an org to view its chart.</p>}
    </div>
  );
}
