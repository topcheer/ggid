"use client";

import { useState, useCallback, useEffect } from "react";
import {
  UserPlus, UserMinus, ArrowRightLeft, Activity, Loader2, AlertCircle, X,
  RefreshCw, Plus, Trash2, Play, Clock, CheckCircle, XCircle, Filter,
  Zap, TrendingUp, Settings, ChevronRight, Eye,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface LifecycleRule {
  id: string;
  name: string;
  trigger: "joiner" | "mover" | "leaver";
  trigger_condition: Record<string, unknown>;
  actions: { type: string; config: Record<string, unknown> }[];
  enabled: boolean;
  priority: number;
  created_at: string;
  last_executed: string | null;
  execution_count: number;
  success_rate: number;
}

interface ExecutionRecord {
  id: string;
  rule_id: string;
  rule_name: string;
  trigger: string;
  user_id: string;
  username: string;
  status: "success" | "failed" | "partial";
  duration_ms: number;
  executed_at: string;
  error: string | null;
  actions_taken: string[];
}

interface LifecycleEvent {
  id: string;
  event_type: "hr.join" | "hr.transfer" | "hr.leave" | "hr.title_change" | "hr.department_change";
  user_id: string;
  username: string;
  timestamp: string;
  matched_rules: string[];
  status: "pending" | "processing" | "completed" | "failed";
  payload: Record<string, unknown>;
}

interface DashboardStats {
  joiners_30d: number;
  movers_30d: number;
  leavers_30d: number;
  total_rules: number;
  active_rules: number;
  automation_coverage_pct: number;
  avg_execution_ms: number;
  success_rate_pct: number;
}

const triggerConfig = {
  joiner: { icon: UserPlus, color: "text-green-500", bg: "bg-green-100 dark:bg-green-900/30", label: "Joiner" },
  mover: { icon: ArrowRightLeft, color: "text-blue-500", bg: "bg-blue-100 dark:bg-blue-900/30", label: "Mover" },
  leaver: { icon: UserMinus, color: "text-red-500", bg: "bg-red-100 dark:bg-red-900/30", label: "Leaver" },
};

const statusConfig = {
  success: { icon: CheckCircle, color: "text-green-600", bg: "bg-green-50 dark:bg-green-950/20" },
  failed: { icon: XCircle, color: "text-red-600", bg: "bg-red-50 dark:bg-red-950/20" },
  partial: { icon: AlertCircle, color: "text-yellow-600", bg: "bg-yellow-50 dark:bg-yellow-950/20" },
  pending: { icon: Clock, color: "text-gray-400", bg: "bg-gray-50 dark:bg-gray-900" },
  processing: { icon: Loader2, color: "text-blue-500", bg: "bg-blue-50 dark:bg-blue-950/20" },
  completed: { icon: CheckCircle, color: "text-green-600", bg: "bg-green-50 dark:bg-green-950/20" },
};

type Tab = "dashboard" | "rules" | "events" | "history" | "dryrun";

export default function JMLLifecyclePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("dashboard");
  const [rules, setRules] = useState<LifecycleRule[]>([]);
  const [executions, setExecutions] = useState<ExecutionRecord[]>([]);
  const [events, setEvents] = useState<LifecycleEvent[]>([]);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Create rule form
  const [showCreate, setShowCreate] = useState(false);
  const [editRule, setEditRule] = useState<Partial<LifecycleRule> | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  // Dry run
  const [dryRunEvent, setDryRunEvent] = useState("hr.join");
  const [dryRunPayload, setDryRunPayload] = useState('{"user_id":"alice","department":"Engineering","title":"Senior Engineer"}');
  const [dryRunResult, setDryRunResult] = useState<{ matched_rules: LifecycleRule[]; actions: string[] } | null>(null);
  const [dryRunning, setDryRunning] = useState(false);

  const headers = () => ({ ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID });

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [rulesRes, execRes, eventsRes, statsRes] = await Promise.all([
        fetch("/api/v1/identity/lifecycle/rules", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/lifecycle/executions?page_size=50", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/lifecycle/events?page_size=50", { headers: h }).catch(() => null),
        fetch("/api/v1/identity/lifecycle/stats", { headers: h }).catch(() => null),
      ]);
      if (rulesRes?.ok) { const d = await rulesRes.json(); setRules(d.rules || d.items || []); }
      if (execRes?.ok) { const d = await execRes.json(); setExecutions(d.executions || d.items || []); }
      if (eventsRes?.ok) { const d = await eventsRes.json(); setEvents(d.events || d.items || []); }
      if (statsRes?.ok) { setStats(await statsRes.json()); }
    } catch { setError("Failed to load lifecycle data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const saveRule = async () => {
    if (!editRule?.name || !editRule?.trigger) return;
    setSaving(true);
    try {
      const method = editRule.id ? "PUT" : "POST";
      const url = editRule.id ? `/api/v1/identity/lifecycle/rules/${editRule.id}` : "/api/v1/identity/lifecycle/rules";
      await fetch(url, { method, headers: headers(), body: JSON.stringify(editRule) });
      setShowCreate(false); setEditRule(null);
      loadData();
    } catch { setError("Failed to save rule"); }
    finally { setSaving(false); }
  };

  const deleteRule = async (id: string) => {
    setDeletingId(id);
    try {
      await fetch(`/api/v1/identity/lifecycle/rules/${id}`, { method: "DELETE", headers: headers() });
      setRules(prev => prev.filter(r => r.id !== id));
    } catch { setError("Failed to delete rule"); }
    finally { setDeletingId(null); }
  };

  const toggleRule = async (id: string, enabled: boolean) => {
    try {
      await fetch(`/api/v1/identity/lifecycle/rules/${id}`, { method: "PUT", headers: headers(), body: JSON.stringify({ enabled: !enabled }) });
      setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !enabled } : r));
    } catch { setError("Failed to toggle rule"); }
  };

  const runDryRun = async () => {
    setDryRunning(true);
    setDryRunResult(null);
    try {
      const res = await fetch("/api/v1/identity/lifecycle/dry-run", {
        method: "POST", headers: headers(),
        body: JSON.stringify({ event_type: dryRunEvent, payload: JSON.parse(dryRunPayload) }),
      });
      if (res.ok) setDryRunResult(await res.json());
      else { const d = await res.json().catch(() => ({})); setError(d.error || "Dry run failed"); }
    } catch { setError("Invalid JSON payload or network error"); }
    finally { setDryRunning(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const tabs: { id: Tab; label: string; icon: typeof Activity }[] = [
    { id: "dashboard", label: "Dashboard", icon: Activity },
    { id: "rules", label: "Rules", icon: Settings },
    { id: "events", label: "Events", icon: Zap },
    { id: "history", label: "History", icon: Clock },
    { id: "dryrun", label: "Dry Run", icon: Play },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <UserPlus className="h-6 w-6 text-indigo-500" />
            JML Lifecycle Management
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Joiner-Mover-Leaver automation — rules, events, executions, and dry-run simulation.
          </p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh all" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {/* Error */}
      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        {tabs.map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
            className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition " +
              (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}>
            <Icon className="h-4 w-4" /> {tb.label}
          </button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* DASHBOARD TAB */}
      {tab === "dashboard" && stats && (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {(["joiners_30d", "movers_30d", "leavers_30d"] as const).map((key: any, i: number) => {
              const trigger = ["joiner", "mover", "leaver"][i];
              const cfg = triggerConfig[trigger as keyof typeof triggerConfig];
              const Icon = cfg.icon;
              return (
                <div key={key} className={cardCls + " flex items-center gap-4"}>
                  <div className={"flex h-12 w-12 items-center justify-center rounded-xl " + cfg.bg}><Icon className={"h-6 w-6 " + cfg.color} /></div>
                  <div><p className="text-xs font-semibold uppercase text-gray-400">{cfg.label}s (30d)</p><p className={"text-2xl font-bold " + cfg.color}>{stats[key]}</p></div>
                </div>
              );
            })}
          </div>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Total Rules</span><p className="mt-1 text-2xl font-bold">{stats.total_rules}</p></div>
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Active</span><p className="mt-1 text-2xl font-bold text-green-600">{stats.active_rules}</p></div>
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Auto Coverage</span><div className="mt-1 flex items-center gap-2"><span className="text-2xl font-bold">{stats.automation_coverage_pct}%</span><div className="h-2 w-16 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-500" style={{ width: `${stats.automation_coverage_pct}%` }} /></div></div></div>
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Avg Exec Time</span><p className="mt-1 text-2xl font-bold">{stats.avg_execution_ms}<span className="text-sm text-gray-400">ms</span></p></div>
          </div>
          <div className={cardCls}>
            <div className="flex items-center justify-between"><h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TrendingUp className="h-4 w-4" /> Success Rate</h2><span className={"text-lg font-bold " + (stats.success_rate_pct >= 95 ? "text-green-600" : stats.success_rate_pct >= 80 ? "text-yellow-600" : "text-red-600")}>{stats.success_rate_pct}%</span></div>
            <div className="mt-2 h-4 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={"h-full rounded-full transition-all " + (stats.success_rate_pct >= 95 ? "bg-green-500" : stats.success_rate_pct >= 80 ? "bg-yellow-500" : "bg-red-500")} style={{ width: `${stats.success_rate_pct}%` }} /></div>
          </div>
        </>
      )}

      {/* RULES TAB */}
      {tab === "rules" && (
        <>
          <div className="flex justify-end">
            <button onClick={() => { setEditRule({ name: "", trigger: "joiner", actions: [], enabled: true, priority: 10, trigger_condition: {} }); setShowCreate(true); }} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Rule</button>
          </div>
          {rules.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Settings className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No lifecycle rules configured.</p></div></div> : (
            <div className="space-y-3">
              {rules.map(rule => { const cfg = triggerConfig[rule.trigger]; const TIcon = cfg.icon; return (
                <div key={rule.id} className={cardCls}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className={"flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium " + cfg.bg + " " + cfg.color}><TIcon className="h-3 w-3" /> {cfg.label}</span>
                        <span className="font-medium text-gray-900 dark:text-white">{rule.name}</span>
                        {!rule.enabled && <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 text-gray-400 dark:bg-gray-800">disabled</span>}
                        <span className="text-xs text-gray-400">Priority: {rule.priority}</span>
                      </div>
                      <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-500">
                        <span>Executions: <span className="font-medium text-gray-700 dark:text-gray-300">{rule.execution_count}</span></span>
                        <span>Success: <span className={"font-medium " + (rule.success_rate >= 95 ? "text-green-600" : rule.success_rate >= 80 ? "text-yellow-600" : "text-red-600")}>{rule.success_rate}%</span></span>
                        {rule.last_executed && <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> {new Date(rule.last_executed).toLocaleString()}</span>}
                        {rule.actions?.map((a: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded font-mono bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{a.type}</span>)}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button onClick={() => toggleRule(rule.id, rule.enabled)} aria-pressed={rule.enabled} className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700">{rule.enabled ? "Disable" : "Enable"}</button>
                      <button onClick={() => { setEditRule(rule); setShowCreate(true); }} aria-label="Edit rule" className="rounded-lg border border-gray-300 p-1.5 text-gray-400 hover:bg-gray-100 dark:border-gray-700 dark:hover:bg-gray-800"><Settings className="h-3.5 w-3.5" /></button>
                      <button onClick={() => deleteRule(rule.id)} disabled={deletingId === rule.id} aria-label="Delete rule" className="rounded-lg bg-red-50 p-1.5 text-red-500 hover:bg-red-100 dark:bg-red-950/20 disabled:opacity-50">{deletingId === rule.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}</button>
                    </div>
                  </div>
                </div>
              ); })}
            </div>
          )}
        </>
      )}

      {/* EVENTS TAB */}
      {tab === "events" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> HR Event Stream</h2>
          {events.length === 0 ? <div className="py-8 text-center"><Zap className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No events processed yet.</p></div> : (
            <div className="space-y-2">
              {events.map(ev => { const cfg = statusConfig[ev.status] || statusConfig.pending; const SIcon = cfg.icon; return (
                <div key={ev.id} className={"flex items-start gap-3 rounded-lg border p-3 " + cfg.bg + " border-gray-200 dark:border-gray-700"}>
                  <SIcon className={"h-4 w-4 mt-0.5 shrink-0 " + cfg.color + (ev.status === "processing" ? " animate-spin" : "")} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs font-medium">{ev.event_type}</span>
                      <span className="text-xs text-gray-500">{ev.username || ev.user_id}</span>
                      <span className={"px-1.5 py-0.5 rounded text-xs " + cfg.bg + " " + cfg.color}>{ev.status}</span>
                    </div>
                    {ev.matched_rules?.length > 0 && <p className="mt-1 text-xs text-gray-500">Matched: {ev.matched_rules.join(", ")}</p>}
                    <p className="mt-0.5 text-xs text-gray-400">{new Date(ev.timestamp).toLocaleString()}</p>
                  </div>
                </div>
              ); })}
            </div>
          )}
        </div>
      )}

      {/* HISTORY TAB */}
      {tab === "history" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> Execution History</h2>
          {executions.length === 0 ? <div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No executions recorded.</p></div> : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Rule</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">Duration</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Actions</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Time</th>
                </tr></thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {executions.map(ex => { const cfg = statusConfig[ex.status] || statusConfig.failed; const SIcon = cfg.icon; return (
                    <tr key={ex.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-4 py-3 text-xs font-medium">{ex.rule_name}</td>
                      <td className="px-4 py-3 text-xs">{ex.username || ex.user_id}</td>
                      <td className="px-4 py-3"><span className={"flex items-center gap-1 text-xs " + cfg.color}><SIcon className="h-3 w-3" /> {ex.status}</span></td>
                      <td className="px-4 py-3 text-right text-xs font-mono">{ex.duration_ms}ms</td>
                      <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{ex.actions_taken?.map((a: any, i: number) => <span key={i} className="px-1 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-xs font-mono">{a}</span>)}</div></td>
                      <td className="px-4 py-3 text-xs text-gray-500">{new Date(ex.executed_at).toLocaleString()}</td>
                    </tr>
                  ); })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* DRY RUN TAB */}
      {tab === "dryrun" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Play className="h-4 w-4" /> Dry Run Simulator</h2>
          <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">Simulate an HR event to preview which rules match and what actions would execute.</p>
          <div className="space-y-4">
            <div><label className="text-sm font-medium">Event Type</label>
              <select aria-label="Event type" value={dryRunEvent} onChange={e => setDryRunEvent(e.target.value)} className="mt-1 block rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                <option value="hr.join">hr.join (Joiner)</option>
                <option value="hr.transfer">hr.transfer (Mover)</option>
                <option value="hr.leave">hr.leave (Leaver)</option>
                <option value="hr.title_change">hr.title_change</option>
                <option value="hr.department_change">hr.department_change</option>
              </select>
            </div>
            <div><label className="text-sm font-medium">Payload (JSON)</label>
              <textarea aria-label="Event payload" value={dryRunPayload} onChange={e => setDryRunPayload(e.target.value)} rows={5} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            </div>
            <button onClick={runDryRun} disabled={dryRunning} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{dryRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} Simulate</button>
          </div>
          {dryRunResult && (
            <div className="mt-6 space-y-4">
              <div className="rounded-lg border border-indigo-200 bg-indigo-50 p-4 dark:border-indigo-800 dark:bg-indigo-950/30">
                <h3 className="flex items-center gap-2 text-sm font-semibold text-indigo-700 dark:text-indigo-400"><Eye className="h-4 w-4" /> Simulation Result</h3>
                {dryRunResult.matched_rules?.length > 0 ? (
                  <div className="mt-3 space-y-2">
                    {dryRunResult.matched_rules.map((r: any, i: number) => (
                      <div key={i} className="flex items-center gap-2 rounded-lg bg-white p-3 dark:bg-gray-800">
                        <ChevronRight className="h-4 w-4 text-indigo-500" />
                        <div className="flex-1"><span className="font-medium text-sm">{r.name}</span><div className="flex gap-1 mt-1">{r.actions?.map((a: any, j: number) => <span key={j} className="px-1.5 py-0.5 rounded bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-400 text-xs font-mono">{a.type}</span>)}</div></div>
                      </div>
                    ))}
                  </div>
                ) : <p className="mt-2 text-sm text-gray-500">No rules matched this event.</p>}
              </div>
            </div>
          )}
        </div>
      )}

      </>)}

      {/* Create/Edit Rule Dialog */}
      {showCreate && editRule && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Settings className="h-5 w-5 text-indigo-500" /> {editRule.id ? "Edit Rule" : "New Lifecycle Rule"}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Rule Name *</label><input aria-label="Rule name" type="text" value={editRule.name || ""} onChange={e => setEditRule({ ...editRule, name: e.target.value })} placeholder="Auto-provision new hires" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Trigger</label><select aria-label="Trigger type" value={editRule.trigger || "joiner"} onChange={e => setEditRule({ ...editRule, trigger: e.target.value as LifecycleRule["trigger"] })} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="joiner">Joiner</option><option value="mover">Mover</option><option value="leaver">Leaver</option></select></div>
                <div><label className="text-sm font-medium">Priority</label><input aria-label="Priority" type="number" value={editRule.priority ?? 10} onChange={e => setEditRule({ ...editRule, priority: parseInt(e.target.value) || 10 })} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <div><label className="text-sm font-medium">Trigger Conditions (JSON)</label><textarea aria-label="Trigger conditions" value={JSON.stringify(editRule.trigger_condition || {}, null, 2)} onChange={e => { try { setEditRule({ ...editRule, trigger_condition: JSON.parse(e.target.value) }); } catch { /* allow editing invalid JSON */ } }} rows={4} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" /></div>
              <div><label className="text-sm font-medium">Actions (JSON array)</label><textarea aria-label="Actions" value={JSON.stringify(editRule.actions || [], null, 2)} onChange={e => { try { setEditRule({ ...editRule, actions: JSON.parse(e.target.value) }); } catch { /* allow editing */ } }} rows={4} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={saveRule} disabled={!editRule.name || saving} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
