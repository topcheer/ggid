"use client";

import { useState, useCallback } from "react";
import { Search, Shield, GitBranch, Layers } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Permission {
  resource: string;
  action: string;
  source: string;
}

interface AccessData {
  subject: string;
  direct_permissions: Permission[];
  inherited_permissions: Permission[];
  effective_permissions: { resource: string; actions: string[] }[];
  via_groups: string[];
  via_roles: string[];
}

export default function AccessGraphPage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [data, setData] = useState<AccessData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchGraph = useCallback(async () => {
    if (!search) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/access-graph?subject=${encodeURIComponent(search)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [search]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Layers className="w-6 h-6 text-indigo-500" /> {t("accessGraph.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Visualize effective permissions for a subject including inherited access.</p>
      </div>

      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-md"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input aria-label="user:alice or role:admin" type="text" value={search} onChange={(e) => setSearch(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") fetchGraph(); }} placeholder="user:alice or role:admin" className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        <button aria-label="action" onClick={fetchGraph} disabled={loading || !search} className="px-4 py-2 rounded-lg bg-indigo-600 text-white text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">{t("backend3.accessGraph.analyze")}</button>
      </div>

      {data && (
        <>
          {(data.via_groups.length > 0 || data.via_roles.length > 0) && (
            <div className="flex flex-wrap gap-2">
              {data.via_groups.map((g) => <span key={g} className="px-2 py-1 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">via: {g}</span>)}
              {data.via_roles.map((r) => <span key={r} className="px-2 py-1 rounded text-xs bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400 font-mono">via: {r}</span>)}
            </div>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Shield className="w-4 h-4 text-green-500" /> Direct Permissions</h3>
              <div className="space-y-1">{data.direct_permissions.map((p: any, i: number) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs text-gray-500">{p.action}</span><span className="flex-1">{p.resource}</span><span className="text-xs text-gray-400">{p.source}</span></div>
              ))}{data.direct_permissions.length === 0 && <p className="text-xs text-gray-400">{t("backend3.accessGraph.none")}</p>}</div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><GitBranch className="w-4 h-4 text-purple-500" /> Inherited Permissions</h3>
              <div className="space-y-1">{data.inherited_permissions.map((p: any, i: number) => (
                <div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs text-gray-500">{p.action}</span><span className="flex-1">{p.resource}</span><span className="text-xs text-gray-400">via {p.source}</span></div>
              ))}{data.inherited_permissions.length === 0 && <p className="text-xs text-gray-400">{t("backend3.accessGraph.none")}</p>}</div>
            </div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">{t("backend3.accessGraph.effectivePermissionsSummary")}</h3>
            <div className="space-y-2">{data.effective_permissions.map((p) => (
              <div key={p.resource} className="flex items-center gap-2"><span className="font-mono text-sm flex-1">{p.resource}</span><div className="flex gap-1">{p.actions.map((a) => <span key={a} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{a}</span>)}</div></div>
            ))}{data.effective_permissions.length === 0 && <p className="text-xs text-gray-400">{t("backend3.accessGraph.none")}</p>}</div>
          </div>
        </>
      )}
      {!data && !loading && search && <p className="text-sm text-gray-500 text-center py-8">Click Analyze to view access graph.</p>}
    </div>
  );
}
