"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Network, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Search, Shield, Zap, Code, Eye, ChevronRight, Database,
  CheckCircle2, XCircle, GitBranch, Layers, Filter, Copy,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface RelationTuple {
  id?: string; namespace: string; object: string;
  relation: string; subject: string; created_at?: string;
}
interface CheckResponse { allowed: boolean; reason?: string; }

type Tab = "playground" | "tuples" | "schema" | "explorer";

const COMMON_NAMESPACES = ["document", "folder", "user", "group", "project", "org"];
const COMMON_RELATIONS = ["owner", "editor", "viewer", "can_view", "can_edit", "can_delete", "parent", "member"];

const SAMPLE_SCHEMA = `namespace document {
  relation owner: user
  relation editor: user | group#member
  relation viewer: user | group#member
  
  permission can_view = viewer or editor or owner
  permission can_edit = editor or owner
  permission can_delete = owner
  permission can_share = owner or editor
}

namespace folder {
  relation parent: folder
  relation owner: user
  
  permission can_view = owner
}

namespace group {
  relation member: user
}`;

export default function ReBACPage() {
  const [tab, setTab] = useState<Tab>("playground");
  const [tuples, setTuples] = useState<RelationTuple[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Playground
  const [pgNs, setPgNs] = useState("document");
  const [pgObj, setPgObj] = useState("report-q4");
  const [pgRel, setPgRel] = useState("can_view");
  const [pgSubj, setPgSubj] = useState("user:alice");
  const [pgResult, setPgResult] = useState<CheckResponse | null>(null);
  const [checking, setChecking] = useState(false);

  // Tuple form
  const [showForm, setShowForm] = useState(false);
  const [tNs, setTNs] = useState("document");
  const [tObj, setTObj] = useState("");
  const [tRel, setTRel] = useState("viewer");
  const [tSubj, setTSubj] = useState("");

  // Explorer
  const [exMode, setExMode] = useState<"objects" | "subjects">("objects");
  const [exNs, setExNs] = useState("document");
  const [exRel, setExRel] = useState("can_view");
  const [exEntity, setExEntity] = useState("user:alice");
  const [exResults, setExResults] = useState<string[]>([]);

  // Tuple filter
  const [fNs, setFNs] = useState("");
  const [fRel, setFRel] = useState("");

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadTuples = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (fNs) params.set("namespace", fNs);
      if (fRel) params.set("relation", fRel);
      const res = await fetch(`/api/v1/identity/tuples?${params}`, { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setTuples(d.tuples || []); }
      setError(null);
    } catch { setError("Failed to load tuples"); }
    finally { setLoading(false); }
  }, [fNs, fRel]);

  useEffect(() => { loadTuples(); }, [loadTuples]);

  const runCheck = async () => {
    if (!pgNs || !pgObj || !pgRel || !pgSubj) return;
    setChecking(true); setPgResult(null);
    try {
      const res = await fetch("/api/v1/identity/check", {
        method: "POST", headers: H,
        body: JSON.stringify({ namespace: pgNs, object: pgObj, relation: pgRel, subject: pgSubj }),
      }).catch(() => null);
      if (res?.ok) {
        setPgResult(await res.json());
      } else {
        setPgResult({ allowed: false, reason: "ReBAC engine not configured or request failed" });
      }
    } catch { setError("Check failed"); }
    finally { setChecking(false); }
  };

  const addTuple = async () => {
    if (!tNs || !tObj || !tRel || !tSubj) return;
    setActionLoading("add");
    try {
      await fetch("/api/v1/identity/tuples", {
        method: "POST", headers: H,
        body: JSON.stringify({ namespace: tNs, object: tObj, relation: tRel, subject: tSubj }),
      });
      setShowForm(false); setTObj(""); setTSubj("");
      loadTuples();
    } catch { setError("Failed to add tuple"); }
    finally { setActionLoading(null); }
  };

  const deleteTuple = async (t: RelationTuple) => {
    setActionLoading(`del-${t.namespace}-${t.object}-${t.relation}-${t.subject}`);
    try {
      await fetch("/api/v1/identity/tuples", {
        method: "DELETE", headers: H,
        body: JSON.stringify({ namespace: t.namespace, object: t.object, relation: t.relation, subject: t.subject }),
      });
      loadTuples();
    } catch { setError("Failed to delete tuple"); }
    finally { setActionLoading(null); }
  };

  const runExplorer = async () => {
    setActionLoading("explorer");
    setExResults([]);
    try {
      const endpoint = exMode === "objects" ? "/api/v1/identity/list-objects" : "/api/v1/identity/list-subjects";
      const body = exMode === "objects"
        ? { namespace: exNs, relation: exRel, subject: exEntity }
        : { namespace: exNs, object: exEntity, relation: exRel };
      const res = await fetch(endpoint, { method: "POST", headers: H, body: JSON.stringify(body) }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setExResults(d.objects || d.subjects || []);
      }
    } catch { setError("Explorer query failed"); }
    finally { setActionLoading(null); }
  };

  const filteredTuples = tuples.filter(t => {
    if (fNs && t.namespace !== fNs) return false;
    if (fRel && t.relation !== fRel) return false;
    return true;
  });

  // Group tuples by namespace for visualization
  const nsGroups = filteredTuples.reduce((acc, t) => {
    const key = t.namespace;
    if (!acc[key]) acc[key] = [];
    acc[key].push(t);
    return acc;
  }, {} as Record<string, RelationTuple[]>);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Network className="h-6 w-6 text-cyan-500" /> ReBAC Console
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Zanzibar-style relationship-based access control — tuple store, permission playground, and schema editor.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "playground" as Tab, label: "Playground", icon: Zap },
          { id: "tuples" as Tab, label: "Tuples", icon: Database },
          { id: "schema" as Tab, label: "Schema Editor", icon: Code },
          { id: "explorer" as Tab, label: "Explorer", icon: Search },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-cyan-600 text-cyan-600 dark:text-cyan-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {/* ════ PLAYGROUND ════ */}
      {tab === "playground" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Permission Check</h2>
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm font-medium">Namespace</label>
                  <select value={pgNs} onChange={e => setPgNs(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {COMMON_NAMESPACES.map(ns => <option key={ns} value={ns}>{ns}</option>)}
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium">Relation / Permission</label>
                  <select value={pgRel} onChange={e => setPgRel(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {COMMON_RELATIONS.map(r => <option key={r} value={r}>{r}</option>)}
                  </select>
                </div>
              </div>
              <div>
                <label className="text-sm font-medium">Object</label>
                <input type="text" value={pgObj} onChange={e => setPgObj(e.target.value)} placeholder="report-q4" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="text-sm font-medium">Subject</label>
                <input type="text" value={pgSubj} onChange={e => setPgSubj(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <button onClick={runCheck} disabled={checking}
                className="flex items-center gap-2 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
                {checking ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} Check Permission
              </button>
            </div>
            <div className="mt-4 rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3">
              <p className="text-xs font-mono text-gray-500">Check: does <span className="text-cyan-500">{pgSubj}</span> have <span className="text-cyan-500">{pgRel}</span> on <span className="text-cyan-500">{pgNs}:{pgObj}</span>?</p>
            </div>
          </div>

          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> Result</h2>
            {pgResult ? (
              <div>
                <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${pgResult.allowed ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30"}`}>
                  {pgResult.allowed ? <CheckCircle2 className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                  <div>
                    <p className={`text-lg font-bold ${pgResult.allowed ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>
                      {pgResult.allowed ? "ALLOWED" : "DENIED"}
                    </p>
                    {pgResult.reason && <p className="text-xs text-gray-500 dark:text-gray-400">{pgResult.reason}</p>}
                  </div>
                </div>
                <div className="mt-4 text-xs text-gray-400">
                  <p className="flex items-center gap-1"><GitBranch className="h-3 w-3" /> Graph traversal resolves permission through relationship chains</p>
                </div>
              </div>
            ) : (
              <div className="py-8 text-center"><Zap className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Run a permission check to see the result.</p></div>
            )}
          </div>
        </div>
      )}

      {/* ════ TUPLES ════ */}
      {tab === "tuples" && (
        <div>
          <div className="mb-4 flex items-center justify-between gap-4 flex-wrap">
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-gray-400" />
              <select value={fNs} onChange={e => setFNs(e.target.value)} aria-label="Filter namespace" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
                <option value="">All namespaces</option>
                {COMMON_NAMESPACES.map(ns => <option key={ns} value={ns}>{ns}</option>)}
              </select>
              <select value={fRel} onChange={e => setFRel(e.target.value)} aria-label="Filter relation" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
                <option value="">All relations</option>
                {COMMON_RELATIONS.map(r => <option key={r} value={r}>{r}</option>)}
              </select>
            </div>
            <button onClick={() => { setTObj(""); setTSubj(""); setShowForm(true); }}
              className="flex items-center gap-1 rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-cyan-700">
              <Plus className="h-3 w-3" /> Add Tuple
            </button>
          </div>

          {loading ? <div className="flex justify-center py-8"><Loader2 className="h-8 w-8 animate-spin text-cyan-500" /></div> :
          filteredTuples.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Database className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No relation tuples found.</p></div></div>
          ) : (
            <div className="space-y-4">
              {Object.entries(nsGroups).map(([ns, items]) => (
                <div key={ns} className={card}>
                  <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold"><Layers className="h-4 w-4 text-cyan-500" /> {ns} <span className="text-xs text-gray-400">({items.length})</span></h3>
                  <div className="space-y-1">
                    {items.map((t, i) => (
                      <div key={i} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-900/30">
                        <div className="flex items-center gap-2 flex-wrap">
                          <code className="text-xs font-mono text-gray-500">{t.object}</code>
                          <ChevronRight className="h-3 w-3 text-gray-300" />
                          <span className="px-1.5 py-0.5 rounded bg-cyan-100 dark:bg-cyan-900/30 text-cyan-600 text-xs font-mono">{t.relation}</span>
                          <ChevronRight className="h-3 w-3 text-gray-300" />
                          <code className="text-xs font-mono text-gray-500">{t.subject}</code>
                        </div>
                        <button onClick={() => deleteTuple(t)} disabled={actionLoading === `del-${t.namespace}-${t.object}-${t.relation}-${t.subject}`}
                          aria-label={"Delete tuple " + t.object + " " + t.relation + " " + t.subject}
                          className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">
                          {actionLoading === `del-${t.namespace}-${t.object}-${t.relation}-${t.subject}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ════ SCHEMA EDITOR ════ */}
      {tab === "schema" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> Namespace Schema</h2>
            <textarea aria-label="Schema editor" defaultValue={SAMPLE_SCHEMA} rows={20}
              className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs leading-relaxed" />
            <div className="mt-3 flex gap-2">
              <button className="flex items-center gap-1 rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-cyan-700"><Check className="h-3 w-3" /> Validate Schema</button>
              <button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700"><Copy className="h-3 w-3" /> Copy</button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Schema Reference</h2>
            <div className="space-y-3 text-xs">
              <div className="rounded-lg border p-3 dark:border-gray-700">
                <p className="font-semibold text-sm mb-1">Relation</p>
                <code className="text-cyan-500">relation name: type</code>
                <p className="text-gray-400 mt-1">Defines a direct relationship. e.g. <code>owner: user</code></p>
              </div>
              <div className="rounded-lg border p-3 dark:border-gray-700">
                <p className="font-semibold text-sm mb-1">Permission</p>
                <code className="text-cyan-500">permission name = expression</code>
                <p className="text-gray-400 mt-1">Computed permission from relations. e.g. <code>can_view = viewer or editor</code></p>
              </div>
              <div className="rounded-lg border p-3 dark:border-gray-700">
                <p className="font-semibold text-sm mb-1">Union (or)</p>
                <code className="text-cyan-500">relation editor: user | group#member</code>
                <p className="text-gray-400 mt-1">Subject can be direct user or member of a group</p>
              </div>
              <div className="rounded-lg border p-3 dark:border-gray-700">
                <p className="font-semibold text-sm mb-1">Intersection (and)</p>
                <code className="text-cyan-500">permission can_delete = owner and not_guest</code>
                <p className="text-gray-400 mt-1">Both conditions must be satisfied</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ════ EXPLORER ════ */}
      {tab === "explorer" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Search className="h-4 w-4" /> Graph Explorer</h2>
            <div className="space-y-3">
              <div className="flex gap-2">
                <button onClick={() => setExMode("objects")} aria-pressed={exMode === "objects"}
                  className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium ${exMode === "objects" ? "border-cyan-500 bg-cyan-50 dark:bg-cyan-950/30 text-cyan-600" : "border-gray-300 dark:border-gray-700"}`}>
                  Find Objects (what can X access?)
                </button>
                <button onClick={() => setExMode("subjects")} aria-pressed={exMode === "subjects"}
                  className={`flex-1 rounded-lg border px-3 py-2 text-sm font-medium ${exMode === "subjects" ? "border-cyan-500 bg-cyan-50 dark:bg-cyan-950/30 text-cyan-600" : "border-gray-300 dark:border-gray-700"}`}>
                  Find Subjects (who has access?)
                </button>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm font-medium">Namespace</label>
                  <select value={exNs} onChange={e => setExNs(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {COMMON_NAMESPACES.map(ns => <option key={ns} value={ns}>{ns}</option>)}
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium">Relation</label>
                  <select value={exRel} onChange={e => setExRel(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {COMMON_RELATIONS.map(r => <option key={r} value={r}>{r}</option>)}
                  </select>
                </div>
              </div>
              <div>
                <label className="text-sm font-medium">{exMode === "objects" ? "Subject" : "Object"}</label>
                <input type="text" value={exEntity} onChange={e => setExEntity(e.target.value)} placeholder={exMode === "objects" ? "user:alice" : "report-q4"} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <button onClick={runExplorer} disabled={actionLoading === "explorer"}
                className="flex items-center gap-2 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
                {actionLoading === "explorer" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />} Explore
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Database className="h-4 w-4" /> Results ({exResults.length})</h2>
            {exResults.length > 0 ? (
              <div className="space-y-1">
                {exResults.map((r, i) => (
                  <div key={i} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700">
                    <ChevronRight className="h-3 w-3 text-cyan-400" />
                    <code className="text-xs font-mono">{r}</code>
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-8 text-center"><Search className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Run an exploration query.</p></div>
            )}
          </div>
        </div>
      )}

      {/* Add tuple dialog */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-cyan-500" /> Add Relation Tuple</h3>
            <div className="mt-4 space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Namespace</label>
                  <select value={tNs} onChange={e => setTNs(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {COMMON_NAMESPACES.map(ns => <option key={ns} value={ns}>{ns}</option>)}
                  </select>
                </div>
                <div><label className="text-sm font-medium">Relation</label>
                  <select value={tRel} onChange={e => setTRel(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {COMMON_RELATIONS.map(r => <option key={r} value={r}>{r}</option>)}
                  </select>
                </div>
              </div>
              <div><label className="text-sm font-medium">Object</label><input type="text" value={tObj} onChange={e => setTObj(e.target.value)} placeholder="report-q4" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div>
              <div><label className="text-sm font-medium">Subject</label><input type="text" value={tSubj} onChange={e => setTSubj(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <div className="mt-3 rounded-lg bg-cyan-50 dark:bg-cyan-900/20 p-2 text-xs text-cyan-600 dark:text-cyan-400">
              <code className="font-mono">{tNs || "..."}:{tObj || "..."}#<span className="font-bold">{tRel}</span>@{tSubj || "..."}</code>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={addTuple} disabled={!tObj || !tSubj || actionLoading === "add"} className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
                {actionLoading === "add" ? <Loader2 className="h-4 w-4 animate-spin" /> : "Add Tuple"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
