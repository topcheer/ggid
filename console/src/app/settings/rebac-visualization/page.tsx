"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Network, Loader2, AlertCircle, X, RefreshCw, Users, Shield,
  Folder, Lock, ChevronRight, Search,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Relation {
  subject: string;
  subject_type: "user" | "group" | "role";
  relation: string;
  object: string;
  object_type: "resource" | "folder" | "policy";
  inherited: boolean;
}

interface CheckResult {
  allowed: boolean;
  reason: string;
  path: string[];
}

const typeIcons: Record<string, typeof Users> = {
  user: Users, group: Users, role: Shield,
  resource: Folder, folder: Folder, policy: Lock,
};

const typeColors: Record<string, string> = {
  user: "text-blue-500", group: "text-purple-500", role: "text-indigo-500",
  resource: "text-green-500", folder: "text-yellow-500", policy: "text-red-500",
};

export default function ReBACPage() {
  const t = useTranslations();
  const [relations, setRelations] = useState<Relation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [checkSubject, setCheckSubject] = useState("");
  const [checkRelation, setCheckRelation] = useState("view");
  const [checkObject, setCheckObject] = useState("");
  const [checkResult, setCheckResult] = useState<CheckResult | null>(null);
  const [checking, setChecking] = useState(false);

  const loadRelations = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/policy/relations?page_size=100", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setRelations(d.relations || d.items || []);
      }
    } catch { setError("Failed to load relations"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadRelations(); }, [loadRelations]);

  const runCheck = async () => {
    if (!checkSubject || !checkObject) return;
    setChecking(true);
    setCheckResult(null);
    try {
      const res = await fetch("/api/v1/policy/check", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ subject: checkSubject, relation: checkRelation, object: checkObject }),
      });
      if (res.ok) {
        setCheckResult(await res.json());
      } else {
        setCheckResult({ allowed: false, reason: "Check failed", path: [] });
      }
    } catch {
      setCheckResult({ allowed: false, reason: "Network error", path: [] });
    } finally { setChecking(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filtered = search
    ? relations.filter(r => r.subject.includes(search) || r.object.includes(search) || r.relation.includes(search))
    : relations;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Network className="h-6 w-6 text-purple-500" />
            ReBAC Permission Visualization
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Relationship-Based Access Control — view permission graphs and run policy checks.
          </p>
        </div>
        <button onClick={loadRelations} disabled={loading} aria-label="Refresh relations" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Permission check */}
      <div className={cardCls}>
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Search className="h-4 w-4" /> Permission Check</h2>
        <div className="flex flex-wrap items-end gap-3">
          <div><label className="text-xs font-medium text-gray-500">Subject</label><input aria-label="Check subject" type="text" value={checkSubject} onChange={e => setCheckSubject(e.target.value)} placeholder="user:alice" className="mt-1 w-40 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
          <ChevronRight className="h-4 w-4 text-gray-400 mb-2" />
          <div><label className="text-xs font-medium text-gray-500">Relation</label><input aria-label="Check relation" type="text" value={checkRelation} onChange={e => setCheckRelation(e.target.value)} placeholder="view" className="mt-1 w-28 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
          <ChevronRight className="h-4 w-4 text-gray-400 mb-2" />
          <div><label className="text-xs font-medium text-gray-500">Object</label><input aria-label="Check object" type="text" value={checkObject} onChange={e => setCheckObject(e.target.value)} placeholder="folder:reports/q4" className="mt-1 w-48 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
          <button onClick={runCheck} disabled={!checkSubject || !checkObject || checking} className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{checking ? <Loader2 className="h-4 w-4 animate-spin" /> : "Check"}</button>
        </div>
        {checkResult && (
          <div className="mt-4 flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
            <span className={"inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-bold " + (checkResult.allowed ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400")}>
              {checkResult.allowed ? <Shield className="h-4 w-4" /> : <Lock className="h-4 w-4" />}
              {checkResult.allowed ? "ALLOWED" : "DENIED"}
            </span>
            <span className="text-sm text-gray-500">{checkResult.reason}</span>
            {checkResult.path?.length > 0 && (
              <div className="ml-auto flex items-center gap-1 text-xs text-gray-400">
                {checkResult.path.map((p: any, i: number) => <span key={i} className="flex items-center gap-1">{i > 0 && <ChevronRight className="h-3 w-3" />}<span className="font-mono">{p}</span></span>)}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Relations</span><p className="mt-2 text-2xl font-bold">{relations.length}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Inherited</span><p className="mt-2 text-2xl font-bold text-purple-600">{relations.filter(r => r.inherited).length}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Subjects</span><p className="mt-2 text-2xl font-bold text-blue-600">{new Set(relations.map(r => r.subject)).size}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Objects</span><p className="mt-2 text-2xl font-bold text-green-600">{new Set(relations.map(r => r.object)).size}</p></div>
      </div>

      {/* Relations list */}
      <div className={cardCls}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase text-gray-400">Permission Relations</h2>
          <div className="relative">
            <Search className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" />
            <input aria-label="Search relations" type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder="Search..." className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" />
          </div>
        </div>
        {loading ? <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-purple-500" /></div> : filtered.length === 0 ? (
          <div className="py-8 text-center"><Network className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No relations found.</p></div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Subject</th>
                  <th scope="col" className="px-4 py-3 text-center font-medium">Relation</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Object</th>
                  <th scope="col" className="px-4 py-3 text-center font-medium">Type</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {filtered.map((r: any, i: number) => {
                  const SubIcon = typeIcons[r.subject_type] || Users;
                  const ObjIcon = typeIcons[r.object_type] || Folder;
                  return (
                    <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-4 py-3"><span className="flex items-center gap-1.5"><SubIcon className={"h-3.5 w-3.5 " + (typeColors[r.subject_type] || "")} /><span className="font-mono text-xs">{r.subject}</span></span></td>
                      <td className="px-4 py-3 text-center"><span className="inline-flex items-center gap-1"><ChevronRight className="h-3 w-3 text-gray-400" /><span className="px-1.5 py-0.5 rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400 font-mono text-xs">{r.relation}</span></span></td>
                      <td className="px-4 py-3"><span className="flex items-center gap-1.5"><ObjIcon className={"h-3.5 w-3.5 " + (typeColors[r.object_type] || "")} /><span className="font-mono text-xs">{r.object}</span></span></td>
                      <td className="px-4 py-3 text-center">{r.inherited ? <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 text-gray-500 dark:bg-gray-800">inherited</span> : <span className="px-1.5 py-0.5 rounded text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">direct</span>}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
