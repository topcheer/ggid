"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Shield, Plus, Trash2, Edit2, Loader2, Save, Check, Play,
  ChevronUp, ChevronDown, Zap, AlertCircle, X,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type TabId = "policies" | "editor" | "tester";
type PolicyAction = "allow" | "deny" | "require_mfa" | "require_step_up" | "block";

interface Condition {
  id: string; operand: string; operator: string; value: string;
}

interface Policy {
  id: string; name: string; priority: number; enabled: boolean;
  action: PolicyAction; conditions: Condition[]; logic: "AND" | "OR";
}

const OPERANDS = [
  { value: "device_posture", label: "Device Posture Score", type: "number" },
  { value: "risk_score", label: "Risk Score", type: "number" },
  { value: "user_group", label: "User Group", type: "text" },
  { value: "ip_range", label: "IP Range", type: "text" },
  { value: "time_of_day", label: "Time of Day", type: "number" },
  { value: "location", label: "Location (Country)", type: "text" },
  { value: "mfa_enrolled", label: "MFA Enrolled", type: "boolean" },
  { value: "auth_method", label: "Authentication Method", type: "text" },
];

const ACTIONS: { value: PolicyAction; color: string }[] = [
  { value: "allow", color: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" },
  { value: "require_mfa", color: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300" },
  { value: "require_step_up", color: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300" },
  { value: "deny", color: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300" },
  { value: "block", color: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300" },
];

let condId = 0;
const newCondId = () => `c${Date.now()}_${condId++}`;

export default function ConditionalAccessPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("policies");
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Policy | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/conditional-access/policies`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setPolicies(d.policies || d || []); return; }
    } catch { /* mock */ }
    setPolicies([
      { id: "p1", name: "High Risk MFA Required", priority: 1, enabled: true, action: "require_mfa", logic: "OR",
        conditions: [
          { id: "c1", operand: "device_posture", operator: "<", value: "70" },
          { id: "c2", operand: "risk_score", operator: ">", value: "60" },
        ] },
      { id: "p2", name: "Block Suspicious IPs", priority: 2, enabled: true, action: "block", logic: "OR",
        conditions: [{ id: "c3", operand: "ip_range", operator: "in", value: "10.0.0.0/8,192.168.0.0/16" }] },
      { id: "p3", name: "Off-Hours Step-Up", priority: 3, enabled: false, action: "require_step_up", logic: "AND",
        conditions: [
          { id: "c4", operand: "time_of_day", operator: "<", value: "8" },
          { id: "c5", operand: "time_of_day", operator: ">", value: "18" },
        ] },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const tabs: { id: TabId; label: string; icon: typeof Shield }[] = [
    { id: "policies", label: t("conditionalAccess.tabs.policies"), icon: Shield },
    { id: "editor", label: t("conditionalAccess.tabs.editor"), icon: Edit2 },
    { id: "tester", label: t("conditionalAccess.tabs.tester"), icon: Play },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Shield className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("conditionalAccess.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("conditionalAccess.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "policies" && (
          <PoliciesList policies={policies} loading={loading}
            onEdit={(p) => { setEditing(p); setTab("editor"); }}
            onAdd={() => { setEditing(null); setTab("editor"); }}
            onToggle={(id) => setPolicies(policies.map((p) => p.id === id ? { ...p, enabled: !p.enabled } : p))}
            onMove={(id, dir) => {
              const idx = policies.findIndex((p) => p.id === id);
              const next = [...policies];
              const target = dir === "up" ? idx - 1 : idx + 1;
              if (target >= 0 && target < next.length) { [next[idx], next[target]] = [next[target], next[idx]]; setPolicies(next); }
            }}
            onDelete={(id) => { setPolicies(policies.filter((p) => p.id !== id)); }}
          />
        )}
        {tab === "editor" && (
          <PolicyEditor editing={editing}
            onSave={(p) => {
              if (editing) { setPolicies(policies.map((x) => x.id === p.id ? p : x)); }
              else { setPolicies([...policies, { ...p, id: `p${Date.now()}` }]); }
              setTab("policies");
            }}
            onCancel={() => setTab("policies")}
          />
        )}
        {tab === "tester" && <PolicyTester policies={policies} />}
      </div>
    </div>
  );
}

// ============ Policies List ============

function PoliciesList({ policies, loading, onEdit, onAdd, onToggle, onMove, onDelete }: {
  policies: Policy[]; loading: boolean;
  onEdit: (p: Policy) => void; onAdd: () => void;
  onToggle: (id: string) => void; onMove: (id: string, dir: "up" | "down") => void;
  onDelete: (id: string) => void;
}) {
  const t = useTranslations();

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("conditionalAccess.policies.title")}</h3>
        <button onClick={onAdd} className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium">
          <Plus className="w-4 h-4" />{t("conditionalAccess.policies.addPolicy")}
        </button>
      </div>

      {policies.length === 0 ? (
        <div className="text-center py-12"><Shield className="w-12 h-12 mx-auto mb-3 text-gray-300" /><p className="text-sm text-gray-500">{t("conditionalAccess.policies.noPolicies")}</p></div>
      ) : (
        <div className="space-y-2">
          {policies.map((p, i) => {
            const actionCfg = ACTIONS.find((a) => a.value === p.action);
            return (
              <div key={p.id} className="flex items-center gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                {/* Priority + arrows */}
                <div className="flex flex-col items-center gap-0.5">
                  <button onClick={() => onMove(p.id, "up")} disabled={i === 0} className="text-gray-400 hover:text-gray-600 disabled:opacity-30"><ChevronUp className="w-4 h-4" /></button>
                  <span className="text-xs font-bold text-gray-500">{p.priority}</span>
                  <button onClick={() => onMove(p.id, "down")} disabled={i === policies.length - 1} className="text-gray-400 hover:text-gray-600 disabled:opacity-30"><ChevronDown className="w-4 h-4" /></button>
                </div>
                {/* Name + conditions */}
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{p.name}</div>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {p.conditions.map((c, ci) => (
                      <span key={c.id} className="text-xs text-gray-500 dark:text-gray-400">
                        {ci > 0 && <span className="mx-1 font-medium text-blue-500">{p.logic}</span>}
                        {c.operand} {c.operator} {c.value}
                      </span>
                    ))}
                  </div>
                </div>
                {/* Action */}
                <span className={`px-2.5 py-0.5 text-xs rounded-full ${actionCfg?.color || ""}`}>{p.action.replace(/_/g, " ")}</span>
                {/* Enabled toggle */}
                <button onClick={() => onToggle(p.id)}
                  className={`relative w-10 h-6 rounded-full transition-colors ${p.enabled ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
                  <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${p.enabled ? "translate-x-4" : ""}`} />
                </button>
                {/* Actions */}
                <button onClick={() => onEdit(p)} className="p-1.5 hover:bg-gray-100 dark:hover:bg-gray-800 rounded"><Edit2 className="w-4 h-4 text-gray-500" /></button>
                <button onClick={() => { if (confirm(t("conditionalAccess.policies.confirmDelete"))) onDelete(p.id); }} className="p-1.5 hover:bg-red-50 dark:hover:bg-red-950 rounded"><Trash2 className="w-4 h-4 text-red-500" /></button>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ============ Policy Editor ============

function PolicyEditor({ editing, onSave, onCancel }: {
  editing: Policy | null; onSave: (p: Policy) => void; onCancel: () => void;
}) {
  const t = useTranslations();
  const [name, setName] = useState(editing?.name || "");
  const [priority, setPriority] = useState(editing?.priority || 10);
  const [action, setAction] = useState<PolicyAction>(editing?.action || "require_mfa");
  const [logic, setLogic] = useState<"AND" | "OR">(editing?.logic || "AND");
  const [conditions, setConditions] = useState<Condition[]>(editing?.conditions || [{ id: newCondId(), operand: "device_posture", operator: "<", value: "70" }]);

  const addCondition = () => setConditions([...conditions, { id: newCondId(), operand: "risk_score", operator: ">", value: "50" }]);
  const removeCondition = (id: string) => setConditions(conditions.filter((c) => c.id !== id));
  const updateCondition = (id: string, field: keyof Condition, value: string) =>
    setConditions(conditions.map((c) => c.id === id ? { ...c, [field]: value } : c));

  const save = () => {
    if (!name) return;
    onSave({ id: editing?.id || "", name, priority, enabled: editing?.enabled ?? true, action, logic, conditions });
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("conditionalAccess.editor.title")}</h3>

      {/* Name + Priority */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="md:col-span-2">
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("conditionalAccess.editor.name")}</label>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("conditionalAccess.editor.namePlaceholder")}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("conditionalAccess.editor.priority")}</label>
          <input type="number" value={priority} onChange={(e) => setPriority(parseInt(e.target.value) || 10)} min={1}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
      </div>

      {/* Conditions */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="text-sm font-semibold text-gray-900 dark:text-white">{t("conditionalAccess.editor.conditions")}</label>
          <div className="flex items-center gap-2">
            <div className="flex gap-1">
              {(["AND", "OR"] as const).map((l) => (
                <button key={l} onClick={() => setLogic(l)}
                  className={`px-2 py-0.5 text-xs font-medium rounded ${logic === l ? "bg-blue-600 text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-500"}`}>{l}</button>
              ))}
            </div>
            <button onClick={addCondition} className="flex items-center gap-1 px-2 py-1 text-xs text-blue-600 hover:underline">
              <Plus className="w-3 h-3" />{t("conditionalAccess.editor.addCondition")}
            </button>
          </div>
        </div>
        <div className="space-y-2">
          {conditions.map((c, i) => (
            <div key={c.id} className="flex items-center gap-2 p-2 rounded-lg bg-gray-50 dark:bg-gray-800/50">
              {i > 0 && <span className="px-2 py-0.5 text-xs font-bold text-blue-500 bg-blue-50 dark:bg-blue-950/30 rounded">{logic}</span>}
              <select value={c.operand} onChange={(e) => updateCondition(c.id, "operand", e.target.value)}
                className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
                {OPERANDS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
              <select value={c.operator} onChange={(e) => updateCondition(c.id, "operator", e.target.value)}
                className="px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white">
                <option value="<">&lt;</option><option value=">">&gt;</option>
                <option value="=">=</option><option value="≠">≠</option>
                <option value="in">in</option><option value="contains">contains</option>
              </select>
              {OPERANDS.find((o) => o.value === c.operand)?.type === "number" ? (
                <input type="range" min={0} max={100} value={parseInt(c.value) || 0} onChange={(e) => updateCondition(c.id, "value", e.target.value)}
                  className="flex-1 max-w-[200px]" />
              ) : (
                <input type="text" value={c.value} onChange={(e) => updateCondition(c.id, "value", e.target.value)} placeholder="value"
                  className="flex-1 px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
              )}
              {OPERANDS.find((o) => o.value === c.operand)?.type === "number" && (
                <span className="text-xs font-medium text-gray-700 dark:text-gray-300 w-8 text-right">{c.value}</span>
              )}
              <button onClick={() => removeCondition(c.id)} className="p-1 text-red-500 hover:bg-red-50 dark:hover:bg-red-950 rounded"><X className="w-3 h-3" /></button>
            </div>
          ))}
        </div>
      </div>

      {/* Action */}
      <div>
        <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">{t("conditionalAccess.editor.action")}</label>
        <div className="flex flex-wrap gap-2">
          {ACTIONS.map((a) => (
            <button key={a.value} onClick={() => setAction(a.value)}
              className={`px-3 py-1.5 rounded-lg border-2 text-sm font-medium transition-all ${
                action === a.value ? "border-blue-500 " + a.color : "border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400"
              }`}>
              {a.value.replace(/_/g, " ")}
            </button>
          ))}
        </div>
      </div>

      {/* Preview */}
      <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <span className="text-xs font-medium text-gray-500 mb-1 block">{t("conditionalAccess.editor.preview")}</span>
        <code className="text-xs text-gray-900 dark:text-white">
          IF {conditions.map((c, i) => `${i > 0 ? ` ${logic} ` : ""}${c.operand} ${c.operator} ${c.value}`).join("")} THEN {action.toUpperCase()}
        </code>
      </div>

      {/* Actions */}
      <div className="flex gap-2">
        <button onClick={save} disabled={!name}
          className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
          <Save className="w-4 h-4" />{t("conditionalAccess.editor.save")}
        </button>
        <button onClick={onCancel} className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium">
          {t("conditionalAccess.editor.selectNew")}
        </button>
      </div>
    </div>
  );
}

// ============ Policy Tester ============

function PolicyTester({ policies }: { policies: Policy[] }) {
  const t = useTranslations();
  const [input, setInput] = useState({
    device_posture: 75, risk_score: 40, user_group: "engineers",
    ip_address: "192.168.1.100", time_of_day: 14, country: "CN",
    mfa_enrolled: true, auth_method: "passkey",
  });
  const [result, setResult] = useState<{ action: PolicyAction; policy?: Policy } | null>(null);
  const [evaluating, setEvaluating] = useState(false);

  const evaluate = () => {
    setEvaluating(true);
    setTimeout(() => {
      let matched: Policy | undefined;
      for (const p of [...policies].sort((a, b) => a.priority - b.priority)) {
        if (!p.enabled) continue;
        const results = p.conditions.map((c) => {
          const inputVal = (input as Record<string, unknown>)[c.operand];
          const numVal = typeof inputVal === "number" ? inputVal : parseFloat(String(inputVal)) || 0;
          const condVal = parseFloat(c.value) || 0;
          switch (c.operator) {
            case "<": return numVal < condVal;
            case ">": return numVal > condVal;
            case "=": return String(inputVal) === c.value;
            case "≠": return String(inputVal) !== c.value;
            case "in": return c.value.split(",").map((v) => v.trim()).includes(String(inputVal));
            case "contains": return String(inputVal).includes(c.value);
            default: return false;
          }
        });
        const isMatch = p.logic === "AND" ? results.every(Boolean) : results.some(Boolean);
        if (isMatch) { matched = p; break; }
      }
      setResult({ action: matched?.action || "allow", policy: matched });
      setEvaluating(false);
    }, 400);
  };

  const inputFields: { key: string; label: string; type: "number" | "text" | "boolean" }[] = [
    { key: "device_posture", label: t("conditionalAccess.tester.devicePosture"), type: "number" },
    { key: "risk_score", label: t("conditionalAccess.tester.riskScore"), type: "number" },
    { key: "user_group", label: t("conditionalAccess.tester.userGroup"), type: "text" },
    { key: "ip_address", label: t("conditionalAccess.tester.ipAddress"), type: "text" },
    { key: "time_of_day", label: t("conditionalAccess.tester.timeOfDay"), type: "number" },
    { key: "country", label: t("conditionalAccess.tester.country"), type: "text" },
    { key: "mfa_enrolled", label: t("conditionalAccess.tester.mfaEnrolled"), type: "boolean" },
    { key: "auth_method", label: t("conditionalAccess.tester.authMethod"), type: "text" },
  ];

  const resultColors: Record<string, string> = {
    allow: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    require_mfa: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    require_step_up: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
    deny: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
    block: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  };

  return (
    <div className="space-y-4">
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("conditionalAccess.tester.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("conditionalAccess.tester.description")}</p>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
          {inputFields.map((f) => (
            <div key={f.key}>
              <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{f.label}</label>
              {f.type === "number" ? (
                <div className="flex items-center gap-2">
                  <input type="range" min={0} max={100} value={(input as Record<string, number>)[f.key]}
                    onChange={(e) => setInput({ ...input, [f.key]: parseInt(e.target.value) })} className="flex-1" />
                  <span className="text-xs font-medium text-gray-900 dark:text-white w-8 text-right">{(input as Record<string, number>)[f.key]}</span>
                </div>
              ) : f.type === "boolean" ? (
                <button onClick={() => setInput({ ...input, [f.key]: !(input as Record<string, boolean>)[f.key] })}
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium ${(input as Record<string, boolean>)[f.key] ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-gray-100 text-gray-500 dark:bg-gray-800"}`}>
                  {(input as Record<string, boolean>)[f.key] ? "true" : "false"}
                </button>
              ) : (
                <input type="text" value={(input as Record<string, string>)[f.key]} onChange={(e) => setInput({ ...input, [f.key]: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
              )}
            </div>
          ))}
        </div>

        <button onClick={evaluate} disabled={evaluating}
          className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
          {evaluating ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
          {evaluating ? t("conditionalAccess.tester.evaluating") : t("conditionalAccess.tester.evaluate")}
        </button>
      </div>

      {result && (
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3">{t("conditionalAccess.tester.result")}</h3>
          <div className="flex items-center gap-4">
            <div className={`px-6 py-3 rounded-xl text-lg font-bold uppercase ${resultColors[result.action]}`}>
              {t(`conditionalAccess.tester.result${result.action.replace(/_./g, (m) => m[1].toUpperCase()).replace(/^./, (m) => m.toUpperCase())}`)}
            </div>
            {result.policy ? (
              <div>
                <div className="text-xs text-gray-500">{t("conditionalAccess.tester.matchedPolicy")}</div>
                <div className="text-sm font-medium text-gray-900 dark:text-white">{result.policy.name}</div>
              </div>
            ) : (
              <span className="text-sm text-gray-400">{t("conditionalAccess.tester.noMatch")}</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
