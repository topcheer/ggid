"use client";
import { useState, useEffect, useCallback } from "react";
import { Network, ChevronRight, Search } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface TreeNode { id: string; name: string; type: "group" | "role" | "permission"; children?: TreeNode[]; permissions: string[]; }

export default function GroupPermissionTreePage() {
  const t = useTranslations();

  const [tree, setTree] = useState<TreeNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/group-permission-tree", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setTree(d.tree || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const toggle = (id: string) => { const next = new Set(expanded); if (next.has(id)) next.delete(id); else next.add(id); setExpanded(next); };

  const matchesSearch = (node: TreeNode): boolean => { if (!search) return true; if (node.name.toLowerCase().includes(search.toLowerCase())) return true; return (node.children || []).some(matchesSearch); };

  const renderNode = (node: TreeNode, depth: number): React.ReactElement | null => {
    if (!matchesSearch(node)) return null;
    const isExpanded = expanded.has(node.id);
    const hasChildren = (node.children || []).length > 0;
    const typeColors: Record<string, string> = { group: "text-blue-600", role: "text-purple-600", permission: "text-gray-500" };
    return (<div key={node.id}><div className="flex items-center gap-2 py-1" style={{ paddingLeft: depth * 20 }}>{hasChildren ? <button onClick={() => toggle(node.id)}><ChevronRight className={"w-4 h-4 text-gray-400 transition-transform " + (isExpanded ? "rotate-90" : "")} /></button> : <span className="w-4" />}<span className={"text-sm font-medium " + typeColors[node.type]}>{node.name}</span>{node.permissions.length > 0 && <div className="flex flex-wrap gap-1 ml-2">{node.permissions.slice(0, 5).map((p: any) => (<span key={p} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{p}</span>))}{node.permissions.length > 5 && <span className="text-xs text-gray-400">+{node.permissions.length - 5}</span>}</div>}{!hasChildren && <span className="text-xs text-gray-400 ml-2">({node.permissions.length}{t("big1.groupPermissionTree.perms")}</span>}</div>{isExpanded && hasChildren && <div>{node.children!.map((c: any) => renderNode(c, depth + 1))}</div>}</div>);
  };

  const totalPerms = (nodes: TreeNode[]): number => nodes.reduce((s: any, n: any) => s + n.permissions.length + totalPerms(n.children || []), 0);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Network className="w-6 h-6 text-blue-500" /> {t("big1.groupPermissionTree.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.groupPermissionTree.hierarchicalViewOfGroupsRolesAndInheritedPermissions")}</p></div>

      <div className="flex items-center gap-4"><div className="relative flex-1 max-w-xs"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input aria-label="Search..." type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search..." className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><span className="text-sm text-gray-500">{totalPerms(tree)}{t("big1.groupPermissionTree.totalPermissions")}</span></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><div className="space-y-0.5">{tree.map((n: any) => renderNode(n, 0))}{tree.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-4">{t("big1.groupPermissionTree.noData")}</p>}</div></div>
    </div>
  );
}
