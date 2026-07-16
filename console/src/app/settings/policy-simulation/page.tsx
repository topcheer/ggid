"use client";

import { useTranslations } from "@/lib/i18n";
import { useState } from "react";
import { Play, Plus, Trash2, Check, X, Minus, GitCompare } from "lucide-react";

interface SimRule {
  id: string;
  effect: "allow" | "deny";
  resource: string;
  action: string;
  condition: string;
}

interface SimResult {
  subject: string;
  resource: string;
  action: string;
  before: "allow" | "deny";
  after: "allow" | "deny";
  status: "would_allow" | "would_deny" | "unchanged";
}

export default function PolicySimulationPage() {
  const [rules, setRules] = useState<SimRule[]>([{ id: "1", effect: "allow", resource: "documents:*", action: "read", condition: "dept == 'engineering'" }]);
  const [results, setResults] = useState<SimResult[] | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");
  const t = useTranslations();

  const retry = () => { setError(""); runSim(); };

  const addRule = () => setRules([...rules, { id: Date.now().toString(), effect: "allow", resource: "", action: "", condition: "" }]);
  const removeRule = (id: string) => setRules(rules.filter((r) => r.id !== id));
  const updateRule = (id: string, field: keyof SimRule, val: string) => setRules(rules.map((r) => r.id === id ? { ...r, [field]: val } : r));

  const runSim = async () => {
    setRunning(true); setError("");
    try {
      const res = await fetch("/api/v1/policy/simulate", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ rules }) });
      if (!res.ok) return null;
      const data = await res.json(); setResults(data.results || data || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to run simulation");
    } finally { setRunning(false); }
  };

  const wouldAllow = results?.filter((r) => r.status === "would_allow") || [];
  const wouldDeny = results?.filter((r) => r.status === "would_deny") || [];
  const unchanged = results?.filter((r) => r.status === "unchanged") || [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Play className="w-6 h-6 text-blue-500" /> {t("policySimulation.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("policySimulation.subtitle")}</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Proposed rules editor */}
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold">{t("policySimulation.proposedRules")} ({rules.length})</h3>
            <button aria-label="Plus" onClick={addRule} className="p-1 rounded bg-blue-600 text-white"><Plus className="w-4 h-4" /></button>
          </div>
          <div className="space-y-2">
            {rules.map((r) => (
              <div key={r.id} className="rounded-lg border dark:border-gray-700 p-3 space-y-2">
                <div className="flex items-center gap-2">
                  <select aria-label="Select option" value={r.effect} onChange={(e) => updateRule(r.id, "effect", e.target.value)} className="px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-xs">
                    <option value="allow">ALLOW</option>
                    <option value="deny">DENY</option>
                  </select>
                  <input aria-label="resource" type="text" value={r.resource} onChange={(e) => updateRule(r.id, "resource", e.target.value)} placeholder="resource" className="flex-1 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-xs font-mono" />
                  <button onClick={() => removeRule(r.id)} className="p-1 text-red-400"><Trash2 className="w-3 h-3" /></button>
                </div>
                <div className="flex items-center gap-2">
                  <input aria-label="action" type="text" value={r.action} onChange={(e) => updateRule(r.id, "action", e.target.value)} placeholder="action" className="flex-1 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-xs font-mono" />
                  <input aria-label="condition" type="text" value={r.condition} onChange={(e) => updateRule(r.id, "condition", e.target.value)} placeholder="condition" className="flex-1 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-xs font-mono" />
                </div>
              </div>
            ))}
          </div>
          <button onClick={runSim} disabled={running} className="w-full mt-3 px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center justify-center gap-2" aria-label="Run policy simulation"><Play className="w-4 h-4" /> {running ? t("policySimulation.simulating") : t("policySimulation.runSimulation")}</button>
        </div>

        {/* Results */}
        <div className="space-y-3">
          {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span>{error}</span><button aria-label="action" onClick={retry} className="text-xs underline hover:text-red-700">{t("policySimulation.retry")}</button></div>}
          {running && (
            <div className="rounded-lg border dark:border-gray-800 p-8 text-center">
              <div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" />
              <div className="text-sm text-gray-500">{t("policySimulation.runningSim")}</div>
            </div>
          )}
          {!running && results ? (
            <>
              <div className="grid grid-cols-3 gap-2">
                <div className="rounded-lg bg-green-50 dark:bg-green-900/20 p-3 text-center"><span className="text-xs text-gray-500">{t("policySimulation.wouldAllow")}</span><p className="text-xl font-bold text-green-600">{wouldAllow.length}</p></div>
                <div className="rounded-lg bg-red-50 dark:bg-red-900/20 p-3 text-center"><span className="text-xs text-gray-500">{t("policySimulation.wouldDeny")}</span><p className="text-xl font-bold text-red-600">{wouldDeny.length}</p></div>
                <div className="rounded-lg bg-gray-50 dark:bg-gray-800 p-3 text-center"><span className="text-xs text-gray-500">{t("policySimulation.unchanged")}</span><p className="text-xl font-bold text-gray-500">{unchanged.length}</p></div>
              </div>
              <div className="rounded-lg border dark:border-gray-800 max-h-80 overflow-y-auto">
                <div className="divide-y dark:divide-gray-800">
                  {results.map((r, i) => (
                    <div key={i} className="px-3 py-2 text-xs">
                      <div className="flex items-center gap-2">
                        {r.status === "would_allow" ? <Check className="w-3 h-3 text-green-500" /> : r.status === "would_deny" ? <X className="w-3 h-3 text-red-500" /> : <Minus className="w-3 h-3 text-gray-400" />}
                        <span className="font-mono truncate">{r.subject} → {r.action} on {r.resource}</span>
                      </div>
                      <div className="flex items-center gap-2 ml-5 mt-0.5 text-gray-400">
                        <span className={r.before === "allow" ? "text-green-600" : "text-red-600"}>{r.before}</span>
                        <GitCompare className="w-3 h-3" />
                        <span className={r.after === "allow" ? "text-green-600" : "text-red-600"}>{r.after}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </>
          ) : (
            <div className="rounded-lg border dark:border-gray-800 p-8 text-center text-sm text-gray-500">{t("policySimulation.runToSeeResults")}</div>
          )}
        </div>
      </div>
    </div>
  );
}
