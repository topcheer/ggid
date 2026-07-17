"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Network, Loader2, AlertCircle, X, RefreshCw, Users, Shield,
  Folder, Lock, ChevronRight, Search, Plus, Trash2, Check, XCircle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Tuple {
  namespace: string;
  object: string;
  relation: string;
  subject: string;
  subject_id?: string;
}

interface CheckResult {
  allowed: boolean;
  reason?: string;
  trace?: string[];
}

const typeIcons: Record<string, typeof Users> = {
  user: Users, group: Users, role: Shield,
  resource: Folder, folder: Folder, policy: Lock,
  document: Folder,
};

const nsColors: Record<string, string> = {
  user: "text-blue-500", group: "text-purple-500", role: "text-indigo-500",
  resource: "text-green-500", folder: "text-yellow-500", document: "text-cyan-500",
};

export default function ReBACPage() {
  const t = useTranslations();
  const [tuples, setTuples] = useState<Tuple[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  // Check form
  const [checkNs, setCheckNs] = useState("document");
  const [checkObj, setCheckObj] = useState("");
  const [checkRel, setCheckRel] = useState("view");
  const [checkSubj, setCheckSubj] = useState("");
  const [checkResult, setCheckResult] = useState<CheckResult | null>(null);
  const [checking, setChecking] = useState(false);
  // Create form
  const [showCreate, setShowCreate] = useState(false);
  const [newNs, setNewNs] = useState("document");
  const [newObj, setNewObj] = useState("");
  const [newRel, setNewRel] = useState("viewer");
  const [newSubj, setNewSubj] = useState("");
  const [creating, setCreating] = useState(false);
  const [deletingKey, setDeletingKey] = useState<string | null>(null);

  const loadTuples = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/identity/tuples", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setTuples(d.tuples || d.items || []);
      }
    } catch { setError("Failed to load tuples"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadTuples(); }, [loadTuples]);

  const runCheck = async () => {
    if (!checkObj || !checkSubj) return;
    setChecking(true);
    setCheckResult(null);
    try {
      const res = await fetch("/api/v1/identity/check", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({
          namespace: checkNs,
          object: checkObj,
          relation: checkRel,
          subject: checkSubj,
        }),
      });
      if (res.ok) {
        const data = await res.json();
        setCheckResult({ allowed: data.allowed ?? false, reason: data.reason, trace: data.trace || data.path });
      } else {
        setCheckResult({ allowed: false, reason: "Check failed" });
      }
    } catch {
      setCheckResult({ allowed: false, reason: "Network error" });
    } finally { setChecking(false); }
  };

  const createTuple = async () => {
    if (!newObj || !newSubj) return;
    setCreating(true);
    try {
      const res = await fetch("/api/v1/identity/tuples", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ namespace: newNs, object: newObj, relation: newRel, subject: newSubj }),
      });
      if (res.ok) {
        setShowCreate(false); setNewObj(""); setNewSubj(""); setNewRel("viewer");
        loadTuples();
      } else { setError("Failed to create tuple"); }
    } catch { setError("Network error"); }
    finally { setCreating(false); }
  };

  const deleteTuple = async (tup: Tuple) => {
    const key = `${tup.namespace}:${tup.object}:${tup.relation}:${tup.subject}`;
    setDeletingKey(key);
    try {
      await fetch("/api/v1/identity/tuples", {
        method: "DELETE",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ namespace: tup.namespace, object: tup.object, relation: tup.relation, subject: tup.subject }),
      });
      setTuples(prev => prev.filter(t => `${t.namespace}:${t.object}:${t.relation}:${t.subject}` !== key));
    } catch { setError("Failed to delete tuple"); }
    finally { setDeletingKey(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filtered = search
    ? tuples.filter(t => t.subject.includes(search) || t.object.includes(search) || t.relation.includes(search) || t.namespace.includes(search))
    : tuples;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Network className="h-6 w-6 text-purple-500" />
            ReBAC Permission Graph
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Relationship-Based Access Control — manage tuples and check permissions via Zanzibar-style engine.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-purple-600 px-3 py-2 text-sm font-medium text-white hover:bg-purple-700">
            <Plus className="h-4 w-4" /> Add Relation
          </button>
          <button onClick={loadTuples} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Permission Check */}
      <div className={cardCls}>
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Search className="h-4 w-4" /> Permission Check</h2>
        <div className="flex flex-wrap items-end gap-3">
          <div>
            <label className="text-xs font-medium text-gray-500">Namespace</label>
            <select aria-label="Check namespace" value={checkNs} onChange={e => setCheckNs(e.target.value)} className="mt-1 block rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
              <option value="document">document</option>
              <option value="folder">folder</option>
              <option value="resource">resource</option>
              <option value="policy">policy</option>
            </select>
          </div>
          <div><label className="text-xs font-medium text-gray-500">Object</label><input aria-label="Check object" type="text" value={checkObj} onChange={e => setCheckObj(e.target.value)} placeholder="report-q4" className="mt-1 w-40 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
          <div><label className="text-xs font-medium text-gray-500">Relation</label><input aria-label="Check relation" type="text" value={checkRel} onChange={e => setCheckRel(e.target.value)} placeholder="view" className="mt-1 w-28 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
          <ChevronRight className="h-4 w-4 text-gray-400 mb-2" />
          <div><label className="text-xs font-medium text-gray-500">Subject</label><input aria-label="Check subject" type="text" value={checkSubj} onChange={e => setCheckSubj(e.target.value)} placeholder="user:alice" className="mt-1 w-40 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
          <button onClick={runCheck} disabled={!checkObj || !checkSubj || checking} className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{checking ? <Loader2 className="h-4 w-4 animate-spin" /> : "Check"}</button>
        </div>
        {checkResult && (
          <div className="mt-4 rounded-lg border p-3 dark:border-gray-700">
            <span className={"inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-sm font-bold " + (checkResult.allowed ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400")}>
              {checkResult.allowed ? <Check className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}
              {checkResult.allowed ? "ALLOWED" : "DENIED"}
            </span>
            {checkResult.reason && <span className="ml-3 text-sm text-gray-500">{checkResult.reason}</span>}
            {checkResult.trace && checkResult.trace.length > 0 && (
              <div className="mt-2 flex items-center gap-1 text-xs text-gray-400">
                <span className="font-medium">Path:</span>
                {checkResult.trace.map((p, i) => <span key={i} className="flex items-center gap-1">{i > 0 && <ChevronRight className="h-3 w-3" />}<span className="font-mono">{p}</span></span>)}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Tuples</span><p className="mt-2 text-2xl font-bold">{tuples.length}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Subjects</span><p className="mt-2 text-2xl font-bold text-blue-600">{new Set(tuples.map(t => t.subject)).size}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Objects</span><p className="mt-2 text-2xl font-bold text-green-600">{new Set(tuples.map(t => `${t.namespace}:${t.object}`)).size}</p></div>
      </div>

      {/* Tuples table */}
      <div className={cardCls}>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase text-gray-400">Permission Tuples</h2>
          <div className="relative">
            <Search className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" />
            <input aria-label="Search tuples" type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder="Search..." className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" />
          </div>
        </div>
        {loading ? <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-purple-500" /></div> : filtered.length === 0 ? (
          <div className="py-8 text-center"><Network className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No permission tuples found.</p></div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Subject</th>
                  <th scope="col" className="px-4 py-3 text-center font-medium">Relation</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Object</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {filtered.map((tup, i) => {
                  const SubIcon = typeIcons[tup.subject?.split(":")[0]] || Users;
                  const ObjIcon = typeIcons[tup.namespace] || Folder;
                  const key = `${tup.namespace}:${tup.object}:${tup.relation}:${tup.subject}`;
                  return (
                    <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-4 py-3"><span className="flex items-center gap-1.5"><SubIcon className={"h-3.5 w-3.5 " + (nsColors[tup.subject?.split(":")[0]] || "text-gray-400")} /><span className="font-mono text-xs">{tup.subject}</span></span></td>
                      <td className="px-4 py-3 text-center"><span className="inline-flex items-center gap-1"><ChevronRight className="h-3 w-3 text-gray-400" /><span className="px-1.5 py-0.5 rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400 font-mono text-xs">{tup.relation}</span></span></td>
                      <td className="px-4 py-3"><span className="flex items-center gap-1.5"><ObjIcon className={"h-3.5 w-3.5 " + (nsColors[tup.namespace] || "text-gray-400")} /><span className="font-mono text-xs">{tup.namespace}:{tup.object}</span></span></td>
                      <td className="px-4 py-3 text-right">
                        <button onClick={() => deleteTuple(tup)} disabled={deletingKey === key} aria-label="Delete tuple" className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50">
                          {deletingKey === key ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Create tuple dialog */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-purple-500" /> Add Permission Relation</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Subject *</label><input aria-label="Tuple subject" type="text" value={newSubj} onChange={e => setNewSubj(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Relation</label><input aria-label="Tuple relation" type="text" value={newRel} onChange={e => setNewRel(e.target.value)} placeholder="viewer" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
                <div><label className="text-sm font-medium">Namespace</label><select aria-label="Tuple namespace" value={newNs} onChange={e => setNewNs(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="document">document</option><option value="folder">folder</option><option value="resource">resource</option></select></div>
              </div>
              <div><label className="text-sm font-medium">Object *</label><input aria-label="Tuple object" type="text" value={newObj} onChange={e => setNewObj(e.target.value)} placeholder="report-q4" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <p className="mt-3 text-xs text-gray-400">Creates: <span className="font-mono">{newSubj || "user:?"} → {newRel} → {newNs}:{newObj || "object"}</span></p>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={createTuple} disabled={!newObj || !newSubj || creating} className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : "Create"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
