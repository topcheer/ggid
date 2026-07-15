"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  GitBranch, ChevronRight, ChevronDown, Users, Shield, Loader2,
  AlertCircle, X,
} from "lucide-react";

interface RoleNode {
  id: string;
  name: string;
  description: string;
  permissions: string[];
  children: RoleNode[];
  user_count: number;
}

function TreeNode({ node, depth }: { node: RoleNode; depth: number }) {
  const [open, setOpen] = useState(true);
  const hasChildren = node.children?.length > 0;
  return (
    <div>
      <div className="flex items-center gap-2 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/50" style={{ paddingLeft: depth * 24 + 12 }}>
        <button onClick={() => hasChildren && setOpen(!open)} className="shrink-0">
          {hasChildren ? (open ? <ChevronDown className="h-4 w-4 text-gray-400" /> : <ChevronRight className="h-4 w-4 text-gray-400" />) : <span className="inline-block w-4" />}
        </button>
        <Shield className="h-4 w-4 shrink-0 text-indigo-500" />
        <span className="flex-1 font-medium text-gray-800 dark:text-gray-200">{node.name}</span>
        <span className="hidden items-center gap-1 text-xs text-gray-400 sm:flex"><Users className="h-3 w-3" />{node.user_count}</span>
        <span className="hidden text-xs text-gray-400 sm:block">{node.permissions.length} perms</span>
      </div>
      {node.description && <p className="text-xs text-gray-400" style={{ paddingLeft: depth * 24 + 48 }}>{node.description}</p>}
      {open && hasChildren && node.children.map((child) => <TreeNode key={child.id} node={child} depth={depth + 1} />)}
    </div>
  );
}

export default function RoleHierarchyPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [tree, setTree] = useState<RoleNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try {
        const data = await apiFetch<{ tree?: RoleNode[]; nodes?: RoleNode[] }>("/api/v1/policy/roles/hierarchy").catch(() => null);
        setTree(data?.tree ?? data?.nodes ?? []);
      } catch {
        setError("Failed to load role hierarchy");
      } finally {
        setLoading(false);
      }
    })();
  });

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <GitBranch className="h-6 w-6 text-indigo-600" /> Role Hierarchy
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Role inheritance tree showing parent-child permission delegation.</p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : tree.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><GitBranch className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No role hierarchy defined.</p></div></div>
      ) : (
        <div className={cardCls}>
          {tree.map((node) => <TreeNode key={node.id} node={node} depth={0} />)}
        </div>
      )}
    </div>
  );
}
