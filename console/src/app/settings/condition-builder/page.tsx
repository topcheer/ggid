"use client";
import { useState, useEffect } from "react";
import { GitBranch, Plus, X, Check, Code } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface Condition { id: string; attribute: string; operator: string; value: string; }
interface Group { id: string; logic: "AND" | "OR"; conditions: Condition[]; }
const operators = ["eq", "ne", "in", "contains", "gt", "lt", "gte", "lte"];
const attributes = ["user.role", "user.department", "user.location", "resource.type", "resource.owner", "action", "time.hour", "request.ip"];
export default function ConditionBuilderPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [groups, setGroups] = useState<Group[]>([]);
  const [validated, setValidated] = useState(false);
  const [showJson, setShowJson] = useState(false);

  useEffect(() => {
    fetch("/api/v1/policy/abac/condition-config", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setGroups(data.groups || []); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);
  const addCondition = (gid: string) => { setGroups(groups.map((g) => g.id === gid ? { ...g, conditions: [...g.conditions, { id: "c" + Date.now(), attribute: "user.role", operator: "eq", value: "" }] } : g)); };
  const removeCondition = (gid: string, cid: string) => { setGroups(groups.map((g) => g.id === gid ? { ...g, conditions: g.conditions.filter((c) => c.id !== cid) } : g)); };
  const updateCondition = (gid: string, cid: string, field: string, value: string) => { setGroups(groups.map((g) => g.id === gid ? { ...g, conditions: g.conditions.map((c) => c.id === cid ? { ...c, [field]: value } : c) } : g)); };
  const addGroup = () => { setGroups([...groups, { id: "g" + Date.now(), logic: "AND", conditions: [] }]); };
  const toggleLogic = (gid: string) => { setGroups(groups.map((g) => g.id === gid ? { ...g, logic: g.logic === "AND" ? "OR" : "AND" } : g)); };
  const jsonPreview = JSON.stringify(groups.map((g) => ({ logic: g.logic, conditions: g.conditions.map((c) => ({ attribute: c.attribute, operator: c.operator, value: c.value })) })), null, 2);
  if (loading) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Condition Builder</h1><p>Loading...</p></div>);
  if (error) return (<div className="p-6"><h1 className="text-2xl font-bold mb-4">Condition Builder</h1><p className="text-red-600">Error: {error}</p></div>);
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-purple-500" /> {t("conditionBuilder.title")}</h1><p className="text-sm text-gray-500 mt-1">Build ABAC policy conditions with AND/OR groups.</p></div><div className="flex gap-2"><button onClick={() => setShowJson(!showJson)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><Code className="w-4 h-4" /> JSON</button><button onClick={() => setValidated(true)} className="px-3 py-1.5 rounded-lg bg-green-600 text-white text-sm flex items-center gap-1"><Check className="w-4 h-4" /> Validate</button></div></div>
      {validated && <div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-3 text-sm text-green-700 dark:text-green-400">All conditions valid.</div>}
      <div className="space-y-3">{groups.map((g, gi) => (<div key={g.id} className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-3 mb-3"><button onClick={() => toggleLogic(g.id)} className={"px-3 py-1 rounded text-xs font-bold " + (g.logic === "AND" ? "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400" : "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400")}>{g.logic}</button><span className="text-xs text-gray-500">Group {gi + 1}</span></div><div className="space-y-2">{g.conditions.map((c) => (<div key={c.id} className="flex items-center gap-2"><select value={c.attribute} onChange={(e) => updateCondition(g.id, c.id, "attribute", e.target.value)} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono">{attributes.map((a) => <option key={a} value={a}>{a}</option>)}</select><select value={c.operator} onChange={(e) => updateCondition(g.id, c.id, "operator", e.target.value)} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono">{operators.map((o) => <option key={o} value={o}>{o}</option>)}</select><input type="text" value={c.value} onChange={(e) => updateCondition(g.id, c.id, "value", e.target.value)} placeholder="value" className="flex-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /><button onClick={() => removeCondition(g.id, c.id)} className="text-red-500 hover:text-red-700"><X className="w-4 h-4" /></button></div>))}<button onClick={() => addCondition(g.id)} className="text-xs text-purple-600 hover:underline flex items-center gap-1"><Plus className="w-3 h-3" /> Add Condition</button></div></div>))}<button onClick={addGroup} className="text-sm text-blue-600 hover:underline flex items-center gap-1"><Plus className="w-4 h-4" /> Add Group</button></div>
      {showJson && <div className="rounded-lg border dark:border-gray-800 p-4"><pre className="text-xs font-mono overflow-x-auto">{jsonPreview}</pre></div>}
    </div>
  );
}
