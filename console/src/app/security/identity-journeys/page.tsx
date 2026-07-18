"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import {
  Workflow, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  CheckCircle, XCircle, Play, Code, Eye, Zap, Settings, ChevronRight,
  ArrowRight, GitBranch, Clock, Shield, FileJson, Copy, Download,
  AlertTriangle, Sparkles, Layers,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface JourneyNode {
  id: string;
  type: string;
  label: string;
  x: number; y: number;
  config: Record<string, unknown>;
}

interface JourneyEdge {
  from: string; to: string; label?: string;
  condition?: string;
}

interface Journey {
  id: string;
  name: string;
  description: string;
  version: number;
  status: "draft" | "active" | "archived";
  yaml: string;
  nodes: JourneyNode[];
  edges: JourneyEdge[];
  updated_at: string;
  executions_30d: number;
}

interface DryRunResult {
  steps: { node_id: string; node_label: string; action: string; result: "continue" | "branch" | "end"; duration_ms: number; detail: string }[];
  final_decision: string;
  execution_path: string[];
  total_duration_ms: number;
}

const NODE_TYPES = [
  { type: "start", label: "Start", icon: Zap, color: "bg-green-500", category: "core" },
  { type: "end", label: "End", icon: CheckCircle, color: "bg-gray-500", category: "core" },
  { type: "auth_password", label: "Password Auth", icon: Shield, color: "bg-blue-500", category: "core" },
  { type: "auth_otp", label: "OTP Verify", icon: Shield, color: "bg-indigo-500", category: "core" },
  { type: "auth_passkey", label: "Passkey Auth", icon: Shield, color: "bg-purple-500", category: "core" },
  { type: "auth_social", label: "Social Login", icon: Shield, color: "bg-cyan-500", category: "core" },
  { type: "condition", label: "Condition (CEL)", icon: GitBranch, color: "bg-yellow-500", category: "core" },
  { type: "mfa_stepup", label: "MFA Step-up", icon: Shield, color: "bg-orange-500", category: "core" },
  { type: "assign_role", label: "Assign Role", icon: Settings, color: "bg-teal-500", category: "core" },
  { type: "provision", label: "JIT Provision", icon: Plus, color: "bg-pink-500", category: "core" },
  { type: "risk_score", label: "Risk Assessment", icon: AlertTriangle, color: "bg-red-500", category: "advanced" },
  { type: "device_check", label: "Device Posture", icon: Shield, color: "bg-violet-500", category: "advanced" },
  { type: "geo_fencing", label: "Geo Check", icon: Eye, color: "bg-fuchsia-500", category: "advanced" },
  { type: "ip_rep", label: "IP Reputation", icon: Shield, color: "bg-rose-500", category: "advanced" },
  { type: "webhook", label: "Webhook Call", icon: Code, color: "bg-lime-500", category: "advanced" },
  { type: "delay", label: "Delay/Timeout", icon: Clock, color: "bg-amber-500", category: "advanced" },
];

const TEMPLATES = [
  { id: "standard_login", name: "Standard Login", desc: "Password → MFA → Session", yaml: `name: standard-login\nversion: 1\nsteps:\n  - type: start\n  - type: auth_password\n  - type: condition\n    expr: "user.mfa_enabled"\n    on_true: mfa_stepup\n    on_false: assign_session\n  - type: mfa_stepup\n    id: mfa_stepup\n  - type: assign_role\n    id: assign_session\n  - type: end` },
  { id: "risk_mfa", name: "Risk-Driven MFA", desc: "Risk score → conditional MFA", yaml: `name: risk-driven-mfa\nversion: 1\nsteps:\n  - type: start\n  - type: auth_password\n  - type: risk_score\n  - type: condition\n    expr: "risk.score > 50"\n    on_true: mfa_stepup\n    on_false: assign_session\n  - type: mfa_stepup\n    id: mfa_stepup\n  - type: assign_role\n    id: assign_session\n  - type: end` },
  { id: "b2b_register", name: "B2B Registration", desc: "Org create → admin → branding", yaml: `name: b2b-register\nversion: 1\nsteps:\n  - type: start\n  - type: condition\n    expr: "org_exists == false"\n    on_true: provision\n    on_false: auth_password\n  - type: provision\n    id: provision\n  - type: assign_role\n  - type: end` },
  { id: "passwordless", name: "Passwordless Migration", desc: "Passkey → fallback password", yaml: `name: passwordless\nversion: 1\nsteps:\n  - type: start\n  - type: auth_passkey\n  - type: condition\n    expr: "!passkey_success"\n    on_true: auth_password\n    on_false: assign_session\n  - type: auth_password\n    id: auth_password\n  - type: assign_role\n    id: assign_session\n  - type: end` },
];

type Tab = "canvas" | "yaml" | "templates" | "dryrun";
type RightPanel = "nodes" | "properties" | "cel";

export default function IdentityJourneysPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("canvas");
  const [journeys, setJourneys] = useState<Journey[]>([]);
  const [activeJourney, setActiveJourney] = useState<Journey | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [rightPanel, setRightPanel] = useState<RightPanel>("nodes");
  const [yamlText, setYamlText] = useState("");
  const [yamlValid, setYamlValid] = useState<boolean | null>(null);
  const [saving, setSaving] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  // CEL editor
  const [celExpr, setCelExpr] = useState("user.role == \"admin\"");
  const [celVars, setCelVars] = useState<Record<string, string>>({ user_role: "admin", risk_score: "20", device_trust: "trusted" });
  // Dry run
  const [dryRunContext, setDryRunContext] = useState('{"user":{"role":"admin","mfa_enabled":true},"risk":{"score":20}}');
  const [dryResult, setDryResult] = useState<DryRunResult | null>(null);
  const [dryRunning, setDryRunning] = useState(false);
  const [highlightNode, setHighlightNode] = useState<string | null>(null);
  const canvasRef = useRef<HTMLDivElement>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/identity/journeys", { headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setJourneys(d.journeys || d.items || []); }
    } catch { setError("Failed to load journeys"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const validateYaml = (text: string) => {
    try { const lines = text.split("\n"); const hasName = lines.some(l => l.startsWith("name:")); const hasSteps = lines.some(l => l.trim().startsWith("- type:")); setYamlValid(hasName && hasSteps); }
    catch { setYamlValid(false); }
  };

  const saveJourney = async () => {
    if (!activeJourney) return;
    setSaving(true);
    try {
      await fetch(`/api/v1/identity/journeys/${activeJourney.id}`, {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ yaml: yamlText, name: activeJourney.name }),
      });
      loadData();
    } catch { setError("Failed to save"); }
    finally { setSaving(false); }
  };

  const createJourney = async () => {
    if (!newName) return;
    try {
      const res = await fetch("/api/v1/identity/journeys", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ name: newName, yaml: `name: ${newName}\nversion: 1\nsteps:\n  - type: start\n  - type: end` }),
      });
      if (res.ok) { setShowCreate(false); setNewName(""); loadData(); }
    } catch { setError("Failed to create"); }
  };

  const applyTemplate = (tmpl: typeof TEMPLATES[0]) => {
    setYamlText(tmpl.yaml);
    setYamlValid(true);
    setTab("yaml");
  };

  const runDryRun = async () => {
    if (!activeJourney) return;
    setDryRunning(true);
    setDryResult(null);
    try {
      const res = await fetch(`/api/v1/identity/journeys/${activeJourney.id}/dry-run`, {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ context: JSON.parse(dryRunContext) }),
      });
      if (res.ok) {
        const result: DryRunResult = await res.json();
        setDryResult(result);
        // Animate through steps
        for (const step of result.steps) {
          setHighlightNode(step.node_id);
          await new Promise(r => setTimeout(r, 500));
        }
        setHighlightNode(null);
      } else { setError("Dry run failed"); }
    } catch { setError("Invalid JSON or network error"); }
    finally { setDryRunning(false); }
  };

  const evaluateCEL = (): string => {
    let expr = celExpr;
    Object.entries(celVars).forEach(([k, v]) => { expr = expr.replace(new RegExp(k.replace(/_/g, "."), "g"), `"${v}"`); });
    try { return expr; } catch { return "Invalid"; }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const selectedNodeData = activeJourney?.nodes.find(n => n.id === selectedNode);
  const nodeTypeDef = (type: string) => NODE_TYPES.find(nt => nt.type === type);

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Workflow className="h-6 w-6 text-indigo-500" /> Identity Journey Editor</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Visual + YAML orchestration for identity flows. Drag nodes, edit conditions, simulate.</p>
        </div>
        <div className="flex items-center gap-2">
          <select aria-label="Select journey" value={activeJourney?.id || ""} onChange={e => { const j = journeys.find(j => j.id === e.target.value); setActiveJourney(j || null); setYamlText(j?.yaml || ""); }} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
            <option value="">Select journey...</option>
            {journeys.map(j => <option key={j.id} value={j.id}>{j.name} (v{j.version})</option>)}
          </select>
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New</button>
          <button onClick={loadData} disabled={loading} aria-label="Refresh" className="rounded-lg border border-gray-300 p-2 text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /></button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        {([
          { id: "canvas" as Tab, label: "Canvas", icon: Layers },
          { id: "yaml" as Tab, label: "YAML Editor", icon: Code },
          { id: "templates" as Tab, label: "Templates", icon: FileJson },
          { id: "dryrun" as Tab, label: "Dry Run", icon: Play },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition " + (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : !activeJourney ? (
        <div className={cardCls}><div className="py-12 text-center"><Workflow className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">Select or create a journey to begin.</p></div></div>
      ) : (<>

      {/* CANVAS TAB */}
      {tab === "canvas" && (
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
          {/* Canvas */}
          <div className="lg:col-span-3">
            <div ref={canvasRef} className="relative h-[500px] overflow-auto rounded-xl border-2 border-dashed border-gray-300 bg-gray-50 dark:border-gray-700 dark:bg-gray-900/50" style={{ backgroundImage: "radial-gradient(circle, #d1d5db 1px, transparent 1px)", backgroundSize: "20px 20px" }}>
              {/* Nodes */}
              {activeJourney.nodes.map((node: any, i: number) => {
                const ntd = nodeTypeDef(node.type);
                const NIcon = ntd?.icon || Zap;
                const isHighlighted = highlightNode === node.id;
                const isSelected = selectedNode === node.id;
                return (
                  <div key={node.id} onClick={() => { setSelectedNode(node.id); setRightPanel("properties"); }} className={"absolute cursor-pointer rounded-xl border-2 p-3 transition-all " + (isHighlighted ? "border-indigo-500 scale-110 shadow-lg ring-4 ring-indigo-200" : isSelected ? "border-indigo-400 shadow-md" : "border-gray-300 dark:border-gray-600") + " bg-white dark:bg-gray-800"} style={{ left: `${node.x}px`, top: `${node.y}px`, minWidth: "140px" }}>
                    <div className="flex items-center gap-2">
                      <div className={"flex h-7 w-7 items-center justify-center rounded-lg " + (ntd?.color || "bg-gray-500")}><NIcon className="h-4 w-4 text-white" /></div>
                      <div><p className="text-xs font-semibold text-gray-900 dark:text-white">{node.label}</p><p className="text-xs text-gray-400">{node.type}</p></div>
                    </div>
                  </div>
                );
              })}
              {/* Edges as SVG */}
              <svg className="pointer-events-none absolute inset-0 h-full w-full">
                {activeJourney.edges.map((edge: any, i: number) => {
                  const from = activeJourney.nodes.find(n => n.id === edge.from);
                  const to = activeJourney.nodes.find(n => n.id === edge.to);
                  if (!from || !to) return null;
                  const isOnPath = dryResult?.execution_path.includes(edge.from) && dryResult?.execution_path.includes(edge.to);
                  return <line key={i} x1={from.x + 70} y1={from.y + 30} x2={to.x + 70} y2={to.y + 30} stroke={isOnPath ? "#6366f1" : "#9ca3af"} strokeWidth={isOnPath ? 3 : 1.5} strokeDasharray={edge.condition ? "5" : "0"} markerEnd="url(#arrowhead)" />;
                })}
                <defs><marker id="arrowhead" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><polygon points="0 0, 8 3, 0 6" fill="#9ca3af" /></marker></defs>
              </svg>
            </div>
          </div>
          {/* Right panel */}
          <div className="space-y-2">
            <div className="flex gap-1">
              <button onClick={() => setRightPanel("nodes")} aria-pressed={rightPanel === "nodes"} className={"flex-1 rounded-lg px-2 py-1.5 text-xs font-medium " + (rightPanel === "nodes" ? "bg-indigo-600 text-white" : "border dark:border-gray-700")}>Nodes</button>
              <button onClick={() => setRightPanel("properties")} aria-pressed={rightPanel === "properties"} className={"flex-1 rounded-lg px-2 py-1.5 text-xs font-medium " + (rightPanel === "properties" ? "bg-indigo-600 text-white" : "border dark:border-gray-700")}>Props</button>
              <button onClick={() => setRightPanel("cel")} aria-pressed={rightPanel === "cel"} className={"flex-1 rounded-lg px-2 py-1.5 text-xs font-medium " + (rightPanel === "cel" ? "bg-indigo-600 text-white" : "border dark:border-gray-700")}>CEL</button>
            </div>
            {rightPanel === "nodes" && (
              <div className="space-y-1 max-h-[440px] overflow-y-auto">
                <p className="text-xs font-semibold uppercase text-gray-400 mb-1">Core Nodes</p>
                {NODE_TYPES.filter(n => n.category === "core").map(nt => { const NIcon = nt.icon; return <div key={nt.type} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700 cursor-grab hover:bg-gray-50 dark:hover:bg-gray-900"><div className={"flex h-6 w-6 items-center justify-center rounded " + nt.color}><NIcon className="h-3 w-3 text-white" /></div><span className="text-xs">{nt.label}</span></div>; })}
                <p className="text-xs font-semibold uppercase text-gray-400 mb-1 mt-2">Advanced Nodes</p>
                {NODE_TYPES.filter(n => n.category === "advanced").map(nt => { const NIcon = nt.icon; return <div key={nt.type} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700 cursor-grab hover:bg-gray-50 dark:hover:bg-gray-900"><div className={"flex h-6 w-6 items-center justify-center rounded " + nt.color}><NIcon className="h-3 w-3 text-white" /></div><span className="text-xs">{nt.label}</span></div>; })}
              </div>
            )}
            {rightPanel === "properties" && selectedNodeData && (
              <div className="rounded-lg border p-3 dark:border-gray-700">
                <p className="text-xs font-semibold uppercase text-gray-400 mb-2">Node Properties</p>
                <div className="space-y-2">
                  <div><label className="text-xs font-medium">Label</label><input aria-label="Node label" type="text" defaultValue={selectedNodeData.label} className="mt-1 w-full rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs" /></div>
                  <div><label className="text-xs font-medium">Type</label><p className="text-xs font-mono mt-1">{selectedNodeData.type}</p></div>
                  <div><label className="text-xs font-medium">Config (JSON)</label><textarea aria-label="Node config" defaultValue={JSON.stringify(selectedNodeData.config, null, 2)} rows={4} className="mt-1 w-full rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" /></div>
                </div>
              </div>
            )}
            {rightPanel === "cel" && (
              <div className="rounded-lg border p-3 dark:border-gray-700">
                <p className="text-xs font-semibold uppercase text-gray-400 mb-2">CEL Expression</p>
                <textarea aria-label="CEL expression" value={celExpr} onChange={e => setCelExpr(e.target.value)} rows={3} className="w-full rounded border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs font-mono" />
                <p className="text-xs font-semibold uppercase text-gray-400 mt-2 mb-1">Test Variables</p>
                <div className="space-y-1">{Object.entries(celVars).map(([k, v]) => <div key={k} className="flex items-center gap-1"><span className="text-xs font-mono flex-1">{k}</span><input aria-label={k} type="text" value={v} onChange={e => setCelVars({ ...celVars, [k]: e.target.value })} className="w-20 rounded border dark:border-gray-700 dark:bg-gray-900 px-1 py-0.5 text-xs font-mono" /></div>)}</div>
                <div className="mt-2 rounded bg-gray-900 p-2"><p className="text-xs text-green-400 font-mono">Evaluated: {evaluateCEL()}</p></div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* YAML EDITOR TAB */}
      {tab === "yaml" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <div className="lg:col-span-2">
            <textarea aria-label="YAML editor" value={yamlText} onChange={e => { setYamlText(e.target.value); validateYaml(e.target.value); }} rows={24} className={"w-full rounded-xl border-2 p-4 font-mono text-sm " + (yamlValid === false ? "border-red-400" : yamlValid === true ? "border-green-400" : "border-gray-300 dark:border-gray-700") + " dark:bg-gray-900"} spellCheck={false} />
            <div className="mt-2 flex items-center justify-between">
              <span className={"text-xs " + (yamlValid === false ? "text-red-500" : yamlValid === true ? "text-green-500" : "text-gray-400")}>{yamlValid === false ? "Invalid YAML — missing name or steps" : yamlValid === true ? "Valid" : "Type to validate"}</span>
              <button onClick={saveJourney} disabled={saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Save Journey</button>
            </div>
          </div>
          <div className="space-y-3">
            <div className={cardCls}>
              <h3 className="text-xs font-semibold uppercase text-gray-400 mb-2">Journey YAML Reference</h3>
              <pre className="text-xs text-gray-500 dark:text-gray-400 font-mono overflow-x-auto">{`name: <string>
version: <int>
description: <string>

steps:
  - type: <node_type>
    id: <optional_id>
    config:
      key: value
    on_true: <step_id>   # for conditions
    on_false: <step_id>
    timeout: <seconds>`}</pre>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-semibold uppercase text-gray-400">Available Node Types</p>
              <div className="flex flex-wrap gap-1">{NODE_TYPES.map(nt => <span key={nt.type} className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{nt.type}</span>)}</div>
            </div>
          </div>
        </div>
      )}

      {/* TEMPLATES TAB */}
      {tab === "templates" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {TEMPLATES.map(tmpl => (
            <div key={tmpl.id} className={cardCls + " hover:shadow-md transition cursor-pointer"} onClick={() => applyTemplate(tmpl)}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3"><div className="h-10 w-10 rounded-lg flex items-center justify-center bg-indigo-100 dark:bg-indigo-900/30"><Sparkles className="h-5 w-5 text-indigo-500" /></div><div><h3 className="font-semibold text-sm">{tmpl.name}</h3><p className="text-xs text-gray-400">{tmpl.desc}</p></div></div>
                <ChevronRight className="h-4 w-4 text-gray-300" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* DRY RUN TAB */}
      {tab === "dryrun" && (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className={cardCls}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Play className="h-4 w-4" /> Simulation Input</h3>
            <label className="text-sm font-medium">User Context (JSON)</label>
            <textarea aria-label="Dry run context" value={dryRunContext} onChange={e => setDryRunContext(e.target.value)} rows={8} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            <button onClick={runDryRun} disabled={dryRunning} className="mt-3 flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{dryRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} Run Simulation</button>
          </div>
          <div className={cardCls}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Execution Trace</h3>
            {dryResult ? (
              <div className="space-y-2">
                {dryResult.steps.map((step: any, i: number) => (
                  <div key={i} className={"flex items-center gap-3 rounded-lg border p-2 " + (highlightNode === step.node_id ? "border-indigo-400 bg-indigo-50 dark:bg-indigo-950/30" : "dark:border-gray-700")}>
                    <div className={"flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold " + (step.result === "end" ? "bg-gray-500" : "bg-indigo-500")}>{i + 1}</div>
                    <div className="flex-1"><p className="text-xs font-medium">{step.node_label}</p><p className="text-xs text-gray-400">{step.action} → {step.result} ({step.duration_ms}ms)</p>{step.detail && <p className="text-xs text-gray-400 italic">{step.detail}</p>}</div>
                  </div>
                ))}
                <div className="mt-3 rounded-lg border-2 p-3 dark:border-gray-600"><p className="text-xs font-semibold uppercase text-gray-400">Final Decision</p><p className={"text-lg font-bold " + (dryResult.final_decision === "allow" ? "text-green-600" : "text-red-600")}>{dryResult.final_decision}</p><p className="text-xs text-gray-400">Total: {dryResult.total_duration_ms}ms</p></div>
              </div>
            ) : <div className="py-8 text-center"><Play className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Run simulation to see execution trace.</p></div>}
          </div>
        </div>
      )}

      </>)}

      {/* Create dialog */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Workflow className="h-5 w-5 text-indigo-500" /> New Identity Journey</h3>
            <div className="mt-4"><label className="text-sm font-medium">Journey Name</label><input aria-label="Journey name" type="text" value={newName} onChange={e => setNewName(e.target.value)} placeholder="Custom Login Flow" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={createJourney} disabled={!newName} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">Create</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
