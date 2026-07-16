"use client";

import { useState, useCallback } from "react";
import { FlaskConical, Plus, X, Check, Ban } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SimResult {
  subject: string;
  resource: string;
  action: string;
  decision: "allow" | "deny";
  matched_policy?: string;
}

interface BatchData {
  results: SimResult[];
  aggregate: { total: number; allowed: number; denied: number; mismatch_count: number };
}

export default function BatchSimulatePage() {
  const t = useTranslations();

  const [subjects, setSubjects] = useState<string[]>([]);
  const [resources, setResources] = useState<string[]>([]);
  const [actions, setActions] = useState<string[]>([]);
  const [subjectInput, setSubjectInput] = useState("");
  const [resourceInput, setResourceInput] = useState("");
  const [actionInput, setActionInput] = useState("");
  const [data, setData] = useState<BatchData | null>(null);
  const [loading, setLoading] = useState(false);

  const add = (val: string, list: string[], setter: (v: string[]) => void) => { if (val && !list.includes(val)) { setter([...list, val]); } };
  const remove = (val: string, list: string[], setter: (v: string[]) => void) => setter(list.filter((x) => x !== val));

  const simulate = useCallback(async () => {
    if (subjects.length === 0 || resources.length === 0 || actions.length === 0) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/batch-simulate", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ subjects, resources, actions }) });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [subjects, resources, actions]);

  const Chip = ({ val, onRemove }: { val: string; onRemove: () => void }) => (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{val}<button aria-label="X" onClick={onRemove}><X className="w-3 h-3" /></button></span>
  );

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><FlaskConical className="w-6 h-6 text-teal-500" /> {t("batchSimulate.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Batch-evaluate policy decisions across multiple subjects, resources, and actions.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-4">
        <div><label className="text-sm font-medium">Subjects</label><div className="flex items-center gap-2 mt-1"><input aria-label="user:alice" type="text" value={subjectInput} onChange={(e) => setSubjectInput(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") { add(subjectInput, subjects, setSubjects); setSubjectInput(""); } }} placeholder="user:alice" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /><button onClick={() => { add(subjectInput, subjects, setSubjects); setSubjectInput(""); }} className="px-3 py-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm"><Plus className="w-4 h-4" /></button></div><div className="flex flex-wrap gap-1 mt-2">{subjects.map((s) => <Chip key={s} val={s} onRemove={() => remove(s, subjects, setSubjects)} />)}</div></div>
        <div><label className="text-sm font-medium">Resources</label><div className="flex items-center gap-2 mt-1"><input aria-label="doc:project-plan" type="text" value={resourceInput} onChange={(e) => setResourceInput(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") { add(resourceInput, resources, setResources); setResourceInput(""); } }} placeholder="doc:project-plan" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /><button onClick={() => { add(resourceInput, resources, setResources); setResourceInput(""); }} className="px-3 py-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm"><Plus className="w-4 h-4" /></button></div><div className="flex flex-wrap gap-1 mt-2">{resources.map((r) => <Chip key={r} val={r} onRemove={() => remove(r, resources, setResources)} />)}</div></div>
        <div><label className="text-sm font-medium">Actions</label><div className="flex items-center gap-2 mt-1"><input aria-label="read" type="text" value={actionInput} onChange={(e) => setActionInput(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter") { add(actionInput, actions, setActions); setActionInput(""); } }} placeholder="read" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /><button onClick={() => { add(actionInput, actions, setActions); setActionInput(""); }} className="px-3 py-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm"><Plus className="w-4 h-4" /></button></div><div className="flex flex-wrap gap-1 mt-2">{actions.map((a) => <Chip key={a} val={a} onRemove={() => remove(a, actions, setActions)} />)}</div></div>
        <button aria-label="action" onClick={simulate} disabled={loading || subjects.length === 0 || resources.length === 0 || actions.length === 0} className="px-4 py-2 rounded-lg bg-teal-600 text-white text-sm font-medium hover:bg-teal-700 disabled:opacity-50 flex items-center gap-2"><FlaskConical className="w-4 h-4" /> {loading ? "Simulating..." : "Run Simulation"}</button>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total</span><p className="text-xl font-bold mt-1">{data.aggregate.total}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Allowed</span><p className="text-xl font-bold text-green-600 mt-1">{data.aggregate.allowed}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Denied</span><p className="text-xl font-bold text-red-600 mt-1">{data.aggregate.denied}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Mismatches</span><p className="text-xl font-bold text-orange-600 mt-1">{data.aggregate.mismatch_count}</p></div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Subject</th><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Action</th><th className="px-4 py-3 text-left font-medium">Decision</th><th className="px-4 py-3 text-left font-medium">Policy</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.results.map((r, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{r.subject}</td><td className="px-4 py-3 font-mono text-xs">{r.resource}</td><td className="px-4 py-3 text-xs">{r.action}</td><td className="px-4 py-3">{r.decision === "allow" ? <span className="flex items-center gap-1 text-xs text-green-600"><Check className="w-3 h-3" /> Allow</span> : <span className="flex items-center gap-1 text-xs text-red-600"><Ban className="w-3 h-3" /> Deny</span>}</td><td className="px-4 py-3 font-mono text-xs text-gray-500">{r.matched_policy || "-"}</td></tr>))}</tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
