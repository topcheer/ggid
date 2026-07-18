"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  Route, Loader2, AlertCircle, X, ChevronRight, Folder, Lock, AlertOctagon, Search,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AccessPathNode {
  resource: string;
  resource_type: string;
  access_level: string;
  source: string;
  over_privileged: boolean;
  children: AccessPathNode[];
}

interface AccessPathResult {
  user_id: string;
  username: string;
  total_resources: number;
  over_privileged_count: number;
  paths: AccessPathNode[];
}

function PathNode({ node, depth }: { node: AccessPathNode; depth: number }) {
  const [expanded, setExpanded] = useState(depth < 1);
  const hasChildren = node.children.length > 0;
  return (
    <div>
      <div className="flex items-center gap-2 rounded px-2 py-1 hover:bg-gray-50 dark:hover:bg-gray-800" style={{ paddingLeft: `${depth * 20 + 8}px` }}>
        {hasChildren ? <button onClick={() => setExpanded(!expanded)}><ChevronRight className={`h-3 w-3 text-gray-400 transition-transform ${expanded ? "rotate-90" : ""}`} /></button> : <span className="w-3" />}
        {node.resource_type === "directory" ? <Folder className="h-4 w-4 text-blue-400" /> : <Lock className="h-4 w-4 text-gray-400" />}
        <span className={`flex-1 text-sm ${node.over_privileged ? "font-medium text-red-600" : "text-gray-700 dark:text-gray-300"}`}>{node.resource}</span>
        <span className={`rounded px-1.5 py-0.5 text-xs ${node.access_level === "admin" ? "bg-red-100 text-red-600 dark:bg-red-900/30" : node.access_level === "write" ? "bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{node.access_level}</span>
        <span className="text-xs text-gray-400">via {node.source}</span>
        {node.over_privileged && <AlertOctagon className="h-3 w-3 text-red-500" />}
      </div>
      {expanded && hasChildren && node.children.map((c: any, i: number) => <PathNode key={i} node={c} depth={depth + 1} />)}
    </div>
  );
}

export default function AccessPathsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [userId, setUserId] = useState("");
  const [result, setResult] = useState<AccessPathResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleAnalyze = async () => {
    if (!userId.trim()) return;
    setLoading(true); setError(null);
    try { setResult(await apiFetch<AccessPathResult>(`/api/v1/policy/access-paths?user_id=${encodeURIComponent(userId)}`)); }
    catch { setError("Analysis failed"); }
    finally { setLoading(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Route className="h-6 w-6 text-indigo-600" /> {t("securityAccessPaths.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Trace user privilege paths and identify over-privileged resources.</p>
      </div>

      {/* User selector */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input aria-label="Enter user ID or username" value={userId} onChange={(e) => setUserId(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleAnalyze()} placeholder="Enter user ID or username" className="w-full rounded-lg border border-gray-300 py-2 pl-10 pr-3 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
        </div>
        <button onClick={handleAnalyze} disabled={!userId.trim() || loading} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Route className="h-4 w-4" />} Analyze</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {result && (
        <>
          {/* Summary */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Total Resources</div><p className="mt-2 text-2xl font-bold text-indigo-600">{result.total_resources}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><AlertOctagon className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Over-Privileged</span></div><p className="mt-2 text-2xl font-bold text-red-600">{result.over_privileged_count}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">User</div><p className="mt-2 truncate text-lg font-bold text-gray-900 dark:text-white">{result.username}</p></div>
          </div>

          {result.over_privileged_count > 0 && (
            <div className="flex items-center gap-3 rounded-xl border border-orange-200 bg-orange-50 px-4 py-3 dark:border-orange-800 dark:bg-orange-900/20"><AlertOctagon className="h-5 w-5 text-orange-600 shrink-0" /><span className="text-sm text-orange-700 dark:text-orange-400">{result.over_privileged_count} resource{result.over_privileged_count > 1 ? "s" : ""} flagged as over-privileged. Review and reduce access scope.</span></div>
          )}

          {/* Path tree */}
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Privilege Path Tree</h3>
            <div className="space-y-0.5">{result.paths.map((node: any, i: number) => <PathNode key={i} node={node} depth={0} />)}</div>
          </div>
        </>
      )}

      {!result && !loading && !error && <div className={cardCls}><div className="py-12 text-center"><Route className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">Enter a user ID to trace their access paths.</p></div></div>}
    </div>
  );
}
