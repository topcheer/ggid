"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  CheckCircle, XCircle, Activity, Zap, Target, Lock, AlertTriangle,
  Clock, TrendingUp, Settings, ChevronRight, ArrowRight, GitBranch,
  ShieldOff, Eye, Download, Code,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface KillChainEvent {
  id: string;
  phase: "initial_access" | "lateral_movement" | "privilege_escalation" | "data_exfiltration" | "encryption";
  timestamp: string;
  user_id: string;
  description: string;
  mitre_technique: string;
  mitre_tactic: string;
  severity: "low" | "medium" | "high" | "critical";
}

interface CompositeRule {
  id: string;
  name: string;
  required_signals: { signal: string; min_count: number }[];
  time_window_minutes: number;
  output_severity: "critical" | "high";
  enabled: boolean;
  triggered_24h: number;
}

interface PlaybookStep {
  id: string;
  action: string;
  target: string;
  enabled: boolean;
  order: number;
}

interface ThreatMapCell {
  user: string;
  resource: string;
  count: number;
  severity: "low" | "medium" | "high" | "critical";
}

interface Incident {
  id: string;
  name: string;
  status: "active" | "contained" | "resolved";
  severity: "critical" | "high" | "medium";
  affected_users: number;
  created_at: string;
  actions_taken: string[];
}

const phaseConfig = {
  initial_access: { label: "Initial Access", color: "bg-orange-500", mitre: "TA0001" },
  lateral_movement: { label: "Lateral Movement", color: "bg-yellow-500", mitre: "TA0008" },
  privilege_escalation: { label: "Privilege Escalation", color: "bg-red-500", mitre: "TA0004" },
  data_exfiltration: { label: "Data Exfiltration", color: "bg-purple-500", mitre: "TA0010" },
  encryption: { label: "Encryption", color: "bg-gray-800", mitre: "TA0040" },
};

const sevColors: Record<string, string> = {
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
};

type Tab = "killchain" | "rules" | "playbook" | "heatmap" | "incidents";

export default function RansomwareDefensePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("killchain");
  const [chainEvents, setChainEvents] = useState<KillChainEvent[]>([]);
  const [rules, setRules] = useState<CompositeRule[]>([]);
  const [playbook, setPlaybook] = useState<PlaybookStep[]>([]);
  const [heatmap, setHeatmap] = useState<ThreatMapCell[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Rule builder
  const [showRuleBuilder, setShowRuleBuilder] = useState(false);
  const [newRuleName, setNewRuleName] = useState("");
  const [newRuleWindow, setNewRuleWindow] = useState(30);
  const [newRuleSignals, setNewRuleSignals] = useState<{ signal: string; min_count: number }[]>([{ signal: "failed_login", min_count: 5 }]);
  const [saving, setSaving] = useState(false);
  // Actions
  const [isolatingId, setIsolatingId] = useState<string | null>(null);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [chainRes, rulesRes, pbRes, heatRes, incRes] = await Promise.all([
        fetch("/api/v1/audit/itdr/kill-chain", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/itdr/composite-rules", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/itdr/playbook", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/itdr/threat-heatmap", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/itdr/incidents", { headers: h }).catch(() => null),
      ]);
      if (chainRes?.ok) { const d = await chainRes.json(); setChainEvents(d.events || d.items || []); }
      if (rulesRes?.ok) { const d = await rulesRes.json(); setRules(d.rules || d.items || []); }
      if (pbRes?.ok) { const d = await pbRes.json(); setPlaybook(d.steps || d.items || []); }
      if (heatRes?.ok) { const d = await heatRes.json(); setHeatmap(d.cells || d.items || []); }
      if (incRes?.ok) { const d = await incRes.json(); setIncidents(d.incidents || d.items || []); }
    } catch { setError("Failed to load ransomware defense data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const saveRule = async () => {
    if (!newRuleName) return;
    setSaving(true);
    try {
      await fetch("/api/v1/audit/itdr/composite-rules", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ name: newRuleName, required_signals: newRuleSignals, time_window_minutes: newRuleWindow, output_severity: "critical" }),
      });
      setShowRuleBuilder(false); setNewRuleName(""); loadData();
    } catch { setError("Failed to save rule"); }
    finally { setSaving(false); }
  };

  const togglePlaybook = async (id: string, enabled: boolean) => {
    setTogglingId(id);
    try {
      await fetch(`/api/v1/audit/itdr/playbook/${id}`, { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID }, body: JSON.stringify({ enabled: !enabled }) });
      setPlaybook(prev => prev.map(s => s.id === id ? { ...s, enabled: !enabled } : s));
    } catch { /* noop */ }
    finally { setTogglingId(null); }
  };

  const isolateIncident = async (id: string) => {
    setIsolatingId(id);
    try {
      await fetch(`/api/v1/audit/itdr/incidents/${id}/isolate`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setIncidents(prev => prev.map(i => i.id === id ? { ...i, status: "contained", actions_taken: [...i.actions_taken, "User isolated + JIT revoked + session revoked"] } : i));
    } catch { setError("Isolation failed"); }
    finally { setIsolatingId(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldOff className="h-6 w-6 text-red-500" /> Ransomware Defense Center</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Kill chain visualization, composite detection rules, auto-isolation playbook, and threat heatmapping.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "killchain" as Tab, label: "Kill Chain", icon: GitBranch },
          { id: "rules" as Tab, label: "Detection Rules", icon: Target },
          { id: "playbook" as Tab, label: "Isolation Playbook", icon: Zap },
          { id: "heatmap" as Tab, label: "Threat Heatmap", icon: Activity },
          { id: "incidents" as Tab, label: "Incident Response", icon: ShieldOff },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-red-600 text-red-600 dark:text-red-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : (<>

      {/* KILL CHAIN */}
      {tab === "killchain" && (
        <div className="space-y-4">
          {/* MITRE phase bar */}
          <div className="flex items-center gap-1">
            {Object.entries(phaseConfig).map(([key, cfg], i) => {
              const hasEvents = chainEvents.some(e => e.phase === key);
              return (
                <div key={key} className="flex items-center gap-1 flex-1">
                  <div className={"flex-1 rounded-lg p-3 text-center transition " + (hasEvents ? cfg.color + " text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>
                    <p className="text-xs font-medium">{cfg.label}</p>
                    <p className="text-xs opacity-75">{cfg.mitre}</p>
                    {hasEvents && <p className="text-xs mt-1 font-bold">{chainEvents.filter(e => e.phase === key).length} events</p>}
                  </div>
                  {i < Object.keys(phaseConfig).length - 1 && <ArrowRight className={"h-4 w-4 shrink-0 " + (hasEvents ? "text-red-500" : "text-gray-300")} />}
                </div>
              );
            })}
          </div>
          {/* Timeline */}
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> Attack Timeline</h2>
            {chainEvents.length === 0 ? <div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No kill chain events detected. System is clean.</p></div> : (
              <div className="space-y-2">{chainEvents.map(ev => { const cfg = phaseConfig[ev.phase]; return (
                <div key={ev.id} className="flex items-start gap-3 rounded-lg border p-3 dark:border-gray-700">
                  <div className={"mt-0.5 h-3 w-3 rounded-full shrink-0 " + cfg.color} />
                  <div className="flex-1">
                    <div className="flex items-center gap-2"><span className="font-mono text-xs font-bold text-gray-400">{cfg.mitre}</span><span className="font-medium text-sm">{cfg.label}</span><span className={"px-1.5 py-0.5 rounded text-xs font-medium " + sevColors[ev.severity]}>{ev.severity}</span></div>
                    <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{ev.description}</p>
                    <div className="mt-1 flex items-center gap-3 text-xs text-gray-400"><span>User: <span className="font-mono">{ev.user_id}</span></span><span>Mitre: <span className="font-mono">{ev.mitre_technique}</span></span><span>{new Date(ev.timestamp).toLocaleString()}</span></div>
                  </div>
                </div>
              ); })}</div>
            )}
          </div>
        </div>
      )}

      {/* COMPOSITE RULES */}
      {tab === "rules" && (
        <>
          <div className="flex justify-end"><button onClick={() => setShowRuleBuilder(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"><Plus className="h-4 w-4" /> New Composite Rule</button></div>
          {rules.length === 0 ? <div className={cardCls}><div className="py-8 text-center"><Target className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No composite detection rules configured.</p></div></div> : (
            <div className="space-y-3">{rules.map(rule => (
              <div key={rule.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2"><span className="font-medium text-gray-900 dark:text-white">{rule.name}</span><span className={"px-2 py-0.5 rounded text-xs font-medium " + sevColors[rule.output_severity]}>{rule.output_severity}</span>{!rule.enabled && <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 text-gray-400">disabled</span>}</div>
                    <div className="mt-2 space-y-1">{rule.required_signals?.map((sig: any, i: number) => (
                      <div key={i} className="flex items-center gap-2 text-xs"><ChevronRight className="h-3 w-3 text-gray-400" /><span className="font-mono text-blue-600 dark:text-blue-400">{sig.signal}</span><span className="text-gray-400">≥ {sig.min_count} times</span></div>
                    ))}<div className="flex items-center gap-2 text-xs"><Clock className="h-3 w-3 text-gray-400" /><span>within {rule.time_window_minutes} minutes</span><ArrowRight className="h-3 w-3 text-gray-400" /><span className="font-bold text-red-600">→ {rule.output_severity.toUpperCase()} ALERT</span></div></div>
                    <p className="mt-1 text-xs text-gray-400">Triggered {rule.triggered_24h} times in 24h</p>
                  </div>
                </div>
              </div>
            ))}</div>
          )}
        </>
      )}

      {/* ISOLATION PLAYBOOK */}
      {tab === "playbook" && (
        <div className="space-y-4">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Auto-Isolation Playbook</h2>
            <p className="text-sm text-gray-500 mb-4">When a ransomware precursor is detected, execute these actions automatically:</p>
            <div className="space-y-2">
              {(playbook.length > 0 ? playbook : [
                { id: "1", action: "CAE Revoke User Sessions", target: "auth-service", enabled: true, order: 1 },
                { id: "2", action: "Cancel Active JIT Elevations", target: "policy-service", enabled: true, order: 2 },
                { id: "3", action: "Lock User Account", target: "identity-service", enabled: true, order: 3 },
                { id: "4", action: "Revoke All OAuth Tokens", target: "oauth-service", enabled: true, order: 4 },
                { id: "5", action: "Send SOC Webhook Alert", target: "webhook", enabled: false, order: 5 },
              ]).map(step => (
                <div key={step.id} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex h-7 w-7 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30 text-xs font-bold text-red-600">{step.order}</div>
                  <div className="flex-1"><p className="text-sm font-medium">{step.action}</p><p className="text-xs text-gray-400">Target: <span className="font-mono">{step.target}</span></p></div>
                  <button onClick={() => togglePlaybook(step.id, step.enabled)} disabled={togglingId === step.id} aria-pressed={step.enabled} className={"flex items-center gap-1 rounded-lg px-3 py-1 text-xs font-medium " + (step.enabled ? "bg-green-50 text-green-700 dark:bg-green-950/20" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>
                    {togglingId === step.id ? <Loader2 className="h-3 w-3 animate-spin" /> : step.enabled ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
                    {step.enabled ? "Active" : "Disabled"}
                  </button>
                </div>
              ))}
            </div>
          </div>
          {/* Flow diagram */}
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Execution Flow</h3>
            <div className="flex items-center gap-2 flex-wrap text-xs">
              <span className="rounded-lg bg-red-100 dark:bg-red-900/30 px-3 py-1.5 font-medium text-red-700 dark:text-red-400">Detection</span>
              <ArrowRight className="h-3 w-3 text-gray-400" />
              <span className="rounded-lg bg-yellow-100 dark:bg-yellow-900/30 px-3 py-1.5 font-medium text-yellow-700 dark:text-yellow-400">Evaluate Rules</span>
              <ArrowRight className="h-3 w-3 text-gray-400" />
              <span className="rounded-lg bg-orange-100 dark:bg-orange-900/30 px-3 py-1.5 font-medium text-orange-700 dark:text-orange-400">Auto-Isolate</span>
              <ArrowRight className="h-3 w-3 text-gray-400" />
              <span className="rounded-lg bg-purple-100 dark:bg-purple-900/30 px-3 py-1.5 font-medium text-purple-700 dark:text-purple-400">Notify SOC</span>
              <ArrowRight className="h-3 w-3 text-gray-400" />
              <span className="rounded-lg bg-blue-100 dark:bg-blue-900/30 px-3 py-1.5 font-medium text-blue-700 dark:text-blue-400">Preserve Evidence</span>
            </div>
          </div>
        </div>
      )}

      {/* THREAT HEATMAP */}
      {tab === "heatmap" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> Active Threat Heatmap</h2>
          {heatmap.length === 0 ? <div className="py-8 text-center"><Activity className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No active threats detected.</p></div> : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead><tr><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">User</th><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Resource</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Detections</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Severity</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{heatmap.map((cell: any, i: number) => (
                <tr key={i}><td className="px-3 py-2 text-xs font-mono">{cell.user}</td><td className="px-3 py-2 text-xs font-mono">{cell.resource}</td><td className="px-3 py-2 text-center"><div className="mx-auto h-6 rounded flex items-center justify-center text-xs font-bold text-white" style={{ width: `${Math.min(cell.count * 20, 100)}px`, backgroundColor: cell.severity === "critical" ? "#dc2626" : cell.severity === "high" ? "#ea580c" : cell.severity === "medium" ? "#ca8a04" : "#2563eb" }}>{cell.count}</div></td><td className="px-3 py-2 text-center"><span className={"px-1.5 py-0.5 rounded text-xs " + sevColors[cell.severity]}>{cell.severity}</span></td></tr>
              ))}</tbody>
            </table></div>
          )}
        </div>
      )}

      {/* INCIDENT RESPONSE */}
      {tab === "incidents" && (
        <div className="space-y-3">
          {incidents.length === 0 ? <div className={cardCls}><div className="py-8 text-center"><CheckCircle className="mx-auto h-10 w-10 text-green-300" /><p className="mt-3 text-sm text-gray-400">No active incidents. System is clean.</p></div></div> : incidents.map(inc => (
            <div key={inc.id} className={"rounded-xl border-2 p-5 " + (inc.status === "active" ? "border-red-300 dark:border-red-700" : inc.status === "contained" ? "border-yellow-300 dark:border-yellow-700" : "border-green-300 dark:border-green-700")}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2"><span className={"px-2 py-0.5 rounded text-xs font-bold " + sevColors[inc.severity]}>{inc.severity.toUpperCase()}</span><span className="font-semibold text-gray-900 dark:text-white">{inc.name}</span><span className={"px-2 py-0.5 rounded text-xs " + (inc.status === "active" ? "bg-red-100 text-red-600 dark:bg-red-900/30" : inc.status === "contained" ? "bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30" : "bg-green-100 text-green-600 dark:bg-green-900/30")}>{inc.status}</span></div>
                  <p className="mt-2 text-xs text-gray-400">{inc.affected_users} users affected · Created {new Date(inc.created_at).toLocaleString()}</p>
                  {inc.actions_taken?.length > 0 && <div className="mt-2"><p className="text-xs font-semibold uppercase text-gray-400">Actions Taken:</p><div className="mt-1 flex flex-wrap gap-1">{inc.actions_taken.map((a: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-xs">{a}</span>)}</div></div>}
                </div>
                <div className="flex flex-col gap-2">
                  {inc.status === "active" && <button onClick={() => isolateIncident(inc.id)} disabled={isolatingId === inc.id} className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-2 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50">{isolatingId === inc.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Lock className="h-3.5 w-3.5" />} Isolate</button>}
                  <button className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-2 text-xs dark:border-gray-700"><Download className="h-3.5 w-3.5" /> Evidence</button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      </>)}

      {/* Rule builder dialog */}
      {showRuleBuilder && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRuleBuilder(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Target className="h-5 w-5 text-red-500" /> Composite Detection Rule</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Rule Name</label><input aria-label="Rule name" type="text" value={newRuleName} onChange={e => setNewRuleName(e.target.value)} placeholder="Ransomware Precursor" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Time Window (minutes)</label><input aria-label="Time window" type="number" min={1} max={1440} value={newRuleWindow} onChange={e => setNewRuleWindow(parseInt(e.target.value) || 30)} className="mt-1 w-24 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">Required Signals</label><div className="mt-1 space-y-2">{newRuleSignals.map((sig: any, i: number) => (
                <div key={i} className="flex items-center gap-2"><select aria-label={`Signal ${i+1}`} value={sig.signal} onChange={e => { const n = [...newRuleSignals]; n[i] = { ...sig, signal: e.target.value }; setNewRuleSignals(n); }} className="flex-1 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs"><option value="failed_login">failed_login</option><option value="lateral_move">lateral_move</option><option value="priv_escalation">priv_escalation</option><option value="mass_file_access">mass_file_access</option><option value="data_exfil">data_exfil</option><option value="encryption_detected">encryption_detected</option></select><span className="text-xs text-gray-400">≥</span><input aria-label={`Min count ${i+1}`} type="number" min={1} value={sig.min_count} onChange={e => { const n = [...newRuleSignals]; n[i] = { ...sig, min_count: parseInt(e.target.value) || 1 }; setNewRuleSignals(n); }} className="w-16 rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs" /></div>
              ))}</div><button onClick={() => setNewRuleSignals([...newRuleSignals, { signal: "lateral_move", min_count: 1 }])} className="mt-1 flex items-center gap-1 text-xs text-red-600 hover:underline"><Plus className="h-3 w-3" /> Add Signal</button></div>
            </div>
            <div className="mt-4 rounded-lg bg-red-50 p-3 dark:bg-red-950/30"><p className="text-xs text-red-700 dark:text-red-400">When all signals match within {newRuleWindow}min → generate <span className="font-bold">CRITICAL</span> ransomware precursor alert</p></div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowRuleBuilder(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={saveRule} disabled={!newRuleName || saving} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save Rule"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
