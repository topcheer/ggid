"use client";
import { useState, useCallback, useEffect } from "react";
import { Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check, CheckCircle, XCircle, Eye, TestTube, Activity, AlertTriangle, Download, Lock, Zap, Settings, Globe } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface DLPPolicy { id: string; name: string; data_class: "general" | "important" | "core"; scope: string; action: "allow" | "warn" | "block" | "encrypt"; enabled: boolean; triggered_24h: number; }
interface DLPEvent { id: string; user_id: string; data_type: string; operation: string; decision: "allow" | "warn" | "block" | "encrypt"; risk_score: number; timestamp: string; resource: string; detail: string; }
interface HeatmapCell { user: string; data_type: string; operations: number; max_decision: string; }
interface TestResult { decision: string; reason: string; risk_score: number; matched_policy: string; path: string[]; }

const classConfig = {
  general: { label: "General", color: "bg-green-100 dark:bg-green-900/30 text-green-600" },
  important: { label: "Important", color: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600" },
  core: { label: "Core", color: "bg-red-100 dark:bg-red-900/30 text-red-600" },
};

const actionConfig = {
  allow: { label: "Allow", color: "bg-green-100 dark:bg-green-900/30 text-green-600", icon: CheckCircle },
  warn: { label: "Warn", color: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600", icon: AlertTriangle },
  block: { label: "Block", color: "bg-red-100 dark:bg-red-900/30 text-red-600", icon: XCircle },
  encrypt: { label: "Encrypt", color: "bg-blue-100 dark:bg-blue-900/30 text-blue-600", icon: Lock },
};

type Tab = "policies" | "events" | "heatmap" | "tester";

export default function DLPPage() {
  const [tab, setTab] = useState<Tab>("policies");
  const [policies, setPolicies] = useState<DLPPolicy[]>([]);
  const [events, setEvents] = useState<DLPEvent[]>([]);
  const [heatmap, setHeatmap] = useState<HeatmapCell[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Policy form
  const [showForm, setShowForm] = useState(false);
  const [pName, setPName] = useState("");
  const [pClass, setPClass] = useState<DLPPolicy["data_class"]>("important");
  const [pScope, setPScope] = useState("all");
  const [pAction, setPAction] = useState<DLPPolicy["action"]>("warn");
  const [saving, setSaving] = useState(false);
  // Tester
  const [tUser, setTUser] = useState("user:alice");
  const [tDataType, setTDataType] = useState("customer_pii");
  const [tOperation, setTOperation] = useState("export");
  const [tResult, setTResult] = useState<TestResult | null>(null);
  const [testing, setTesting] = useState(false);
  // Actions
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [eventFilter, setEventFilter] = useState("all");

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [pRes, eRes, hmRes] = await Promise.all([
        fetch("/api/v1/auth/dlp/policies", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/dlp/events?page_size=100", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/dlp/heatmap", { headers: h }).catch(() => null),
      ]);
      if (pRes?.ok) { const d = await pRes.json(); setPolicies(d.policies || d.items || []); }
      if (eRes?.ok) { const d = await eRes.json(); setEvents(d.events || d.items || []); }
      if (hmRes?.ok) { const d = await hmRes.json(); setHeatmap(d.cells || d.items || []); }
    } catch { setError("Failed to load DLP data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const savePolicy = async () => {
    if (!pName) return;
    setSaving(true);
    try {
      await fetch("/api/v1/auth/dlp/policies", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ name: pName, data_class: pClass, scope: pScope, action: pAction }) });
      setShowForm(false); setPName(""); loadData();
    } catch { setError("Failed to save policy"); }
    finally { setSaving(false); }
  };

  const togglePolicy = async (id: string, enabled: boolean) => {
    setTogglingId(id);
    try {
      await fetch(`/api/v1/auth/dlp/policies/${id}`, { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ enabled: !enabled }) });
      setPolicies(prev => prev.map(p => p.id === id ? { ...p, enabled: !enabled } : p));
    } catch { /* noop */ }
    finally { setTogglingId(null); }
  };

  const runTest = async () => {
    setTesting(true); setTResult(null);
    try {
      const res = await fetch("/api/v1/auth/dlp/test", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ user_id: tUser, data_type: tDataType, operation: tOperation }) });
      if (res.ok) setTResult(await res.json());
      else setError("Test failed");
    } catch { setError("Network error"); }
    finally { setTesting(false); }
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filteredEvents = eventFilter === "all" ? events : events.filter(e => e.decision === eventFilter);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Shield className="h-6 w-6 text-red-500" /> Identity-Based DLP
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Data Loss Prevention — policy management, real-time event monitoring, risk heatmapping, and decision testing.
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
          { id: "policies" as Tab, label: "Policies", icon: Settings },
          { id: "events" as Tab, label: "Events", icon: Activity },
          { id: "heatmap" as Tab, label: "Risk Heatmap", icon: Globe },
          { id: "tester" as Tab, label: "Policy Tester", icon: TestTube },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-red-600 text-red-600 dark:text-red-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : (<>

      {/* POLICIES */}
      {tab === "policies" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> DLP Policies</h2>
            <button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700"><Plus className="h-3 w-3" /> Add Policy</button>
          </div>
          {policies.length === 0 ? (
            <div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No DLP policies configured.</p></div>
          ) : (
            <div className="space-y-2">{policies.map(p => {
              const aCfg = actionConfig[p.action];
              const cCfg = classConfig[p.data_class];
              const AIcon = aCfg.icon;
              return (
                <div key={p.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${aCfg.color}`}><AIcon className="h-4 w-4" /></div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{p.name}</span>
                        <span className={`px-1.5 py-0.5 rounded text-xs ${cCfg.color}`}>{cCfg.label}</span>
                        {!p.enabled && <span className="text-xs text-gray-400">disabled</span>}
                      </div>
                      <p className="text-xs text-gray-400">Scope: {p.scope} · {p.triggered_24h} triggers/24h</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${aCfg.color}`}>{aCfg.label}</span>
                    <button onClick={() => togglePolicy(p.id, p.enabled)} disabled={togglingId === p.id} aria-pressed={p.enabled}
                      className={`rounded-lg px-2 py-1 text-xs font-medium ${p.enabled ? "bg-green-50 text-green-700 dark:bg-green-950/20" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
                      {togglingId === p.id ? <Loader2 className="h-3 w-3 animate-spin" /> : p.enabled ? "Active" : "Off"}
                    </button>
                  </div>
                </div>
              );
            })}</div>
          )}
        </div>
      )}

      {/* EVENTS */}
      {tab === "events" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> DLP Event Stream</h2>
            <select aria-label="Filter decision" value={eventFilter} onChange={e => setEventFilter(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
              <option value="all">All Decisions</option><option value="block">Block</option><option value="warn">Warn</option><option value="encrypt">Encrypt</option><option value="allow">Allow</option>
            </select>
          </div>
          {filteredEvents.length === 0 ? (
            <div className="py-8 text-center"><Activity className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No DLP events.</p></div>
          ) : (
            <div className="overflow-x-auto max-h-[400px] overflow-y-auto"><table className="w-full text-sm">
              <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900/80"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">User</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Data Type</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Operation</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Decision</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Risk</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Time</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{filteredEvents.map(e => {
                const aCfg = actionConfig[e.decision] || actionConfig.allow;
                return (
                  <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-3 py-2 text-xs font-mono">{e.user_id}</td>
                    <td className="px-3 py-2 text-xs">{e.data_type}</td>
                    <td className="px-3 py-2 text-xs font-mono text-gray-500">{e.operation}</td>
                    <td className="px-3 py-2 text-center"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${aCfg.color}`}>{e.decision}</span></td>
                    <td className="px-3 py-2 text-center"><span className={`text-xs font-bold ${e.risk_score >= 70 ? "text-red-600" : e.risk_score >= 40 ? "text-yellow-600" : "text-green-600"}`}>{e.risk_score}</span></td>
                    <td className="px-3 py-2 text-xs text-gray-500">{new Date(e.timestamp).toLocaleTimeString()}</td>
                  </tr>
                );
              })}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* HEATMAP */}
      {tab === "heatmap" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Globe className="h-4 w-4" /> Risk Heatmap</h2>
          {heatmap.length === 0 ? (
            <div className="py-8 text-center"><Globe className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No heatmap data.</p></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead><tr><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">User</th><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Data Type</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Operations</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Max Decision</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{heatmap.map((cell, i) => {
                const aCfg = actionConfig[cell.max_decision as keyof typeof actionConfig] || actionConfig.allow;
                return (
                  <tr key={i}>
                    <td className="px-3 py-2 text-xs font-mono">{cell.user}</td>
                    <td className="px-3 py-2 text-xs">{cell.data_type}</td>
                    <td className="px-3 py-2 text-center"><div className="mx-auto h-6 rounded flex items-center justify-center text-xs font-bold text-white" style={{ width: `${Math.min(cell.operations * 15, 100)}px`, backgroundColor: cell.max_decision === "block" ? "#dc2626" : cell.max_decision === "warn" ? "#ca8a04" : "#2563eb" }}>{cell.operations}</div></td>
                    <td className="px-3 py-2 text-center"><span className={`px-1.5 py-0.5 rounded text-xs ${aCfg.color}`}>{cell.max_decision}</span></td>
                  </tr>
                );
              })}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* TESTER */}
      {tab === "tester" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TestTube className="h-4 w-4" /> Policy Tester</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">User ID</label>
                <input aria-label="Test user" type="text" value={tUser} onChange={e => setTUser(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-sm font-medium">Data Type</label>
                  <input aria-label="Data type" type="text" value={tDataType} onChange={e => setTDataType(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
                </div>
                <div>
                  <label className="text-sm font-medium">Operation</label>
                  <select aria-label="Operation" value={tOperation} onChange={e => setTOperation(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    <option value="read">read</option><option value="export">export</option><option value="share">share</option><option value="delete">delete</option><option value="print">print</option>
                  </select>
                </div>
              </div>
              <button onClick={runTest} disabled={testing} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">
                {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <TestTube className="h-4 w-4" />} Evaluate DLP
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Decision Result</h2>
            {tResult ? (
              <div>
                <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${tResult.decision === "block" ? "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30" : tResult.decision === "encrypt" ? "border-blue-300 bg-blue-50 dark:border-blue-700 dark:bg-blue-950/30" : tResult.decision === "warn" ? "border-yellow-300 bg-yellow-50 dark:border-yellow-700 dark:bg-yellow-950/30" : "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30"}`}>
                  {tResult.decision === "block" ? <XCircle className="h-8 w-8 text-red-500" /> : tResult.decision === "encrypt" ? <Lock className="h-8 w-8 text-blue-500" /> : tResult.decision === "warn" ? <AlertTriangle className="h-8 w-8 text-yellow-500" /> : <CheckCircle className="h-8 w-8 text-green-500" />}
                  <div>
                    <p className={`text-lg font-bold capitalize ${tResult.decision === "block" ? "text-red-700 dark:text-red-400" : tResult.decision === "encrypt" ? "text-blue-700 dark:text-blue-400" : tResult.decision === "warn" ? "text-yellow-700 dark:text-yellow-400" : "text-green-700 dark:text-green-400"}`}>{tResult.decision}</p>
                    <p className="text-xs text-gray-500">{tResult.reason}</p>
                  </div>
                  <span className={`ml-auto text-lg font-bold ${tResult.risk_score >= 70 ? "text-red-600" : tResult.risk_score >= 40 ? "text-yellow-600" : "text-green-600"}`}>{tResult.risk_score}</span>
                </div>
                {tResult.matched_policy && <p className="mt-3 text-xs text-gray-400">Matched policy: <span className="font-mono font-medium">{tResult.matched_policy}</span></p>}
                {tResult.path?.length > 0 && (
                  <div className="mt-2 flex items-center gap-1 flex-wrap">
                    <span className="text-xs text-gray-400">Path:</span>
                    {tResult.path.map((step, i) => <span key={i} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{step}</span>)}
                  </div>
                )}
              </div>
            ) : (
              <div className="py-8 text-center"><TestTube className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Configure and evaluate DLP decision.</p></div>
            )}
          </div>
        </div>
      )}

      </>)}

      {/* Policy form dialog */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-red-500" /> New DLP Policy</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Policy Name</label><input aria-label="Policy name" type="text" value={pName} onChange={e => setPName(e.target.value)} placeholder="Block PII export" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Data Class</label><select aria-label="Data class" value={pClass} onChange={e => setPClass(e.target.value as DLPPolicy["data_class"])} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="general">General</option><option value="important">Important</option><option value="core">Core</option></select></div>
                <div><label className="text-sm font-medium">Action</label><select aria-label="Action" value={pAction} onChange={e => setPAction(e.target.value as DLPPolicy["action"])} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="allow">Allow</option><option value="warn">Warn</option><option value="block">Block</option><option value="encrypt">Encrypt</option></select></div>
              </div>
              <div><label className="text-sm font-medium">Scope (group or "all")</label><input aria-label="Scope" type="text" value={pScope} onChange={e => setPScope(e.target.value)} placeholder="all" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={savePolicy} disabled={!pName || saving} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
