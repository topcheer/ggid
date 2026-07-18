"use client";
import { useState, useCallback, useEffect } from "react";
import { Network, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check, CheckCircle, XCircle, Shield, Zap, Eye, TestTube, ArrowRight, ChevronRight, Clock, AlertTriangle, Code, Lock, Users } from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface DelegationPolicy { id: string; subject: string; actor: string; actions: string[]; resource: string; max_scope: string; enabled: boolean; }
interface ActiveSession { id: string; subject: string; actor: string; chain_depth: number; scopes: string[]; issued_at: string; expires_at: string; status: "active" | "expiring"; }
interface AuditEntry { id: string; event: "created" | "used" | "revoked"; subject: string; actor: string; chain_depth: number; timestamp: string; detail: string; anomaly: boolean; }
interface SimResult { allowed: boolean; token_claims: Record<string, unknown>; effective_scopes: string[]; chain: string[]; warnings: string[]; }

type Tab = "chain" | "policy" | "sessions" | "audit" | "simulator";

export default function DelegationPage() {
  const [tab, setTab] = useState<Tab>("chain");
  const [policies, setPolicies] = useState<DelegationPolicy[]>([]);
  const [sessions, setSessions] = useState<ActiveSession[]>([]);
  const [audit, setAudit] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Simulator
  const [simSubject, setSimSubject] = useState("user:alice");
  const [simActor, setSimActor] = useState("service:api-gateway");
  const [simActions, setSimActions] = useState("read:data");
  const [simResult, setSimResult] = useState<SimResult | null>(null);
  const [simRunning, setSimRunning] = useState(false);
  // Actions
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [pRes, sRes, aRes] = await Promise.all([
        fetch("/api/v1/policies/delegations", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/delegation/sessions", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/delegation/audit", { headers: h }).catch(() => null),
      ]);
      if (pRes?.ok) { const d = await pRes.json(); setPolicies(d.delegations || d.items || []); }
      if (sRes?.ok) { const d = await sRes.json(); setSessions(d.sessions || d.items || []); }
      if (aRes?.ok) { const d = await aRes.json(); setAudit(d.entries || d.items || []); }
    } catch { setError("Failed to load delegation data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const runSim = async () => {
    setSimRunning(true); setSimResult(null);
    try {
      const res = await fetch("/api/v1/auth/delegation/simulate", {
        method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ subject: simSubject, actor: simActor, actions: simActions.split(",") }),
      });
      if (res.ok) setSimResult(await res.json());
      else setError("Simulation failed");
    } catch { setError("Network error"); }
    finally { setSimRunning(false); }
  };

  const revokeSession = async (id: string) => {
    setRevokingId(id);
    try {
      await fetch(`/api/v1/auth/delegation/sessions/${id}/revoke`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setSessions(prev => prev.filter(s => s.id !== id));
    } catch { setError("Failed to revoke"); }
    finally { setRevokingId(null); }
  };

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  // Chain visualization data
  const chainNodes = sessions.length > 0 ? sessions.slice(0, 3).map(s => ({ label: s.subject, actor: s.actor, depth: s.chain_depth })) : [
    { label: "user:alice", actor: "service:api-gw", depth: 1 },
    { label: "service:api-gw", actor: "service:db-proxy", depth: 2 },
    { label: "service:db-proxy", actor: "—", depth: 3 },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Network className="h-6 w-6 text-indigo-500" /> Fine-grained Delegation
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Token exchange delegation chains, policy configuration, active sessions, and audit trail.
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
          { id: "chain" as Tab, label: "Delegation Chain", icon: Network },
          { id: "policy" as Tab, label: "Policies", icon: Shield },
          { id: "sessions" as Tab, label: "Active Sessions", icon: Zap },
          { id: "audit" as Tab, label: "Audit Trail", icon: Clock },
          { id: "simulator" as Tab, label: "Simulator", icon: TestTube },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* CHAIN VISUALIZATION */}
      {tab === "chain" && (
        <div className={card}>
          <h2 className="mb-6 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Network className="h-4 w-4" /> Active Delegation Chains</h2>
          <div className="flex items-center justify-center gap-2 flex-wrap">
            {chainNodes.map((node: any, i: number) => (
              <div key={i} className="flex items-center gap-2">
                <div className="flex flex-col items-center rounded-xl border-2 border-indigo-300 dark:border-indigo-700 bg-indigo-50 dark:bg-indigo-950/30 p-3 min-w-[120px]">
                  <Users className="h-6 w-6 text-indigo-500" />
                  <p className="mt-1 text-xs font-mono font-medium">{node.label}</p>
                  <span className="mt-1 px-1.5 py-0.5 rounded text-xs bg-indigo-100 dark:bg-indigo-900/40 text-indigo-600">depth {node.depth}</span>
                </div>
                {i < chainNodes.length - 1 && (
                  <div className="flex flex-col items-center">
                    <ArrowRight className="h-5 w-5 text-gray-400" />
                    <span className="text-xs text-gray-400">act claim</span>
                  </div>
                )}
              </div>
            ))}
          </div>
          <div className="mt-6 rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3">
            <p className="text-xs font-semibold uppercase text-gray-400 mb-2">JWT act Claim Structure</p>
            <pre className="overflow-x-auto text-xs text-gray-600 dark:text-gray-400 font-mono">{`{
  "sub": "user:alice",
  "act": {
    "sub": "service:api-gateway",
    "act": {
      "sub": "service:db-proxy"
    }
  }
}`}</pre>
          </div>
        </div>
      )}

      {/* POLICIES */}
      {tab === "policy" && (
        <div className={card}>
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> Delegation Policies</h2>
            <button className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700"><Plus className="h-3 w-3" /> Add Policy</button>
          </div>
          {policies.length === 0 ? (
            <div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No delegation policies configured.</p></div>
          ) : (
            <div className="space-y-2">{policies.map(p => (
              <div key={p.id} className="rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs text-blue-600 dark:text-blue-400">{p.subject}</span>
                    <ArrowRight className="h-3 w-3 text-gray-400" />
                    <span className="font-mono text-xs text-purple-600 dark:text-purple-400">{p.actor}</span>
                  </div>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${p.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{p.enabled ? "Active" : "Disabled"}</span>
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
                  <span className="text-gray-400">Actions:</span>
                  {p.actions?.map(a => <span key={a} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-700 font-mono">{a}</span>)}
                  <span className="text-gray-400 ml-2">Resource:</span>
                  <span className="font-mono">{p.resource}</span>
                  <span className="text-gray-400 ml-2">Max scope:</span>
                  <span className="px-1 py-0.5 rounded bg-orange-100 dark:bg-orange-900/30 font-mono text-orange-600">{p.max_scope}</span>
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* ACTIVE SESSIONS */}
      {tab === "sessions" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Active Delegation Sessions</h2>
          {sessions.length === 0 ? (
            <div className="py-8 text-center"><Zap className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No active delegation sessions.</p></div>
          ) : (
            <div className="space-y-2">{sessions.map(s => (
              <div key={s.id} className={`flex items-center justify-between rounded-lg border p-3 ${s.status === "expiring" ? "border-yellow-300 dark:border-yellow-700" : "dark:border-gray-700"}`}>
                <div className="flex items-center gap-3">
                  <div className="flex flex-col items-center">
                    <div className="flex h-7 w-7 items-center justify-center rounded-full bg-indigo-100 dark:bg-indigo-900/30 text-xs font-bold text-indigo-600">{s.chain_depth}</div>
                    <span className="text-xs text-gray-400">depth</span>
                  </div>
                  <div>
                    <div className="flex items-center gap-1">
                      <span className="font-mono text-xs text-blue-600 dark:text-blue-400">{s.subject}</span>
                      <ArrowRight className="h-3 w-3 text-gray-400" />
                      <span className="font-mono text-xs text-purple-600 dark:text-purple-400">{s.actor}</span>
                    </div>
                    <div className="mt-1 flex items-center gap-2 text-xs text-gray-400">
                      <Clock className="h-3 w-3" />
                      <span>Expires: {new Date(s.expires_at).toLocaleString()}</span>
                      {s.status === "expiring" && <span className="text-yellow-600 font-medium">expiring soon</span>}
                    </div>
                    <div className="mt-1 flex gap-1">{s.scopes?.map(sc => <span key={sc} className="px-1 py-0.5 rounded bg-orange-100 dark:bg-orange-900/30 text-xs font-mono text-orange-600">{sc}</span>)}</div>
                  </div>
                </div>
                <button onClick={() => revokeSession(s.id)} disabled={revokingId === s.id} className="flex items-center gap-1 rounded-lg bg-red-50 px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-100 dark:bg-red-950/20 disabled:opacity-50">
                  {revokingId === s.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <Lock className="h-3 w-3" />} Revoke
                </button>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* AUDIT */}
      {tab === "audit" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> Delegation Audit Trail</h2>
          {audit.length === 0 ? (
            <div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No delegation audit events.</p></div>
          ) : (
            <div className="space-y-2">{audit.map(e => (
              <div key={e.id} className={`flex items-start gap-3 rounded-lg border p-3 ${e.anomaly ? "border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-950/10" : "dark:border-gray-700"}`}>
                <div className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold ${e.event === "created" ? "bg-blue-100 text-blue-600 dark:bg-blue-900/30" : e.event === "used" ? "bg-green-100 text-green-600 dark:bg-green-900/30" : "bg-red-100 text-red-600 dark:bg-red-900/30"}`}>
                  {e.event === "created" ? <Plus className="h-3.5 w-3.5" /> : e.event === "used" ? <Check className="h-3.5 w-3.5" /> : <XCircle className="h-3.5 w-3.5" />}
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-bold capitalize">{e.event}</span>
                    <span className="font-mono text-xs text-blue-600 dark:text-blue-400">{e.subject}</span>
                    <ArrowRight className="h-3 w-3 text-gray-400" />
                    <span className="font-mono text-xs text-purple-600 dark:text-purple-400">{e.actor}</span>
                    <span className="px-1 rounded text-xs bg-indigo-100 dark:bg-indigo-900/30">d{e.chain_depth}</span>
                    {e.anomaly && <AlertTriangle className="h-3.5 w-3.5 text-red-500" />}
                  </div>
                  <p className="mt-0.5 text-xs text-gray-400">{e.detail}</p>
                  <p className="text-xs text-gray-400">{new Date(e.timestamp).toLocaleString()}</p>
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* SIMULATOR */}
      {tab === "simulator" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TestTube className="h-4 w-4" /> Delegation Simulator</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Subject (original user)</label>
                <input aria-label="Subject" type="text" value={simSubject} onChange={e => setSimSubject(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="text-sm font-medium">Actor (delegate service)</label>
                <input aria-label="Actor" type="text" value={simActor} onChange={e => setSimActor(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="text-sm font-medium">Actions (comma-separated)</label>
                <input aria-label="Actions" type="text" value={simActions} onChange={e => setSimActions(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <button onClick={runSim} disabled={simRunning} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {simRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <TestTube className="h-4 w-4" />} Simulate Delegation
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Result</h2>
            {simResult ? (
              <div>
                <div className={`flex items-center gap-3 rounded-xl border-2 p-4 ${simResult.allowed ? "border-green-300 bg-green-50 dark:border-green-700 dark:bg-green-950/30" : "border-red-300 bg-red-50 dark:border-red-700 dark:bg-red-950/30"}`}>
                  {simResult.allowed ? <CheckCircle className="h-8 w-8 text-green-500" /> : <XCircle className="h-8 w-8 text-red-500" />}
                  <p className={`text-lg font-bold ${simResult.allowed ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}`}>{simResult.allowed ? "ALLOWED" : "DENIED"}</p>
                </div>
                {simResult.chain?.length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs font-semibold text-gray-400 mb-1">Delegation Chain</p>
                    <div className="flex items-center gap-1 flex-wrap">
                      {simResult.chain.map((node: any, i: number) => (
                        <span key={i} className="flex items-center gap-1">
                          {i > 0 && <ArrowRight className="h-3 w-3 text-gray-400" />}
                          <span className="px-1.5 py-0.5 rounded bg-indigo-100 dark:bg-indigo-900/30 text-xs font-mono">{node}</span>
                        </span>
                      ))}
                    </div>
                  </div>
                )}
                {simResult.effective_scopes?.length > 0 && (
                  <div className="mt-3">
                    <p className="text-xs font-semibold text-gray-400 mb-1">Effective Scopes (narrowed)</p>
                    <div className="flex flex-wrap gap-1">{simResult.effective_scopes.map(s => <span key={s} className="px-1.5 py-0.5 rounded bg-orange-100 dark:bg-orange-900/30 text-xs font-mono text-orange-600">{s}</span>)}</div>
                  </div>
                )}
                {simResult.warnings?.length > 0 && (
                  <div className="mt-3 rounded-lg bg-yellow-50 p-3 dark:bg-yellow-950/20">
                    {simResult.warnings.map((w: any, i: number) => <p key={i} className="text-xs text-yellow-700 dark:text-yellow-400">{w}</p>)}
                  </div>
                )}
              </div>
            ) : (
              <div className="py-8 text-center"><TestTube className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Configure and simulate delegation.</p></div>
            )}
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
