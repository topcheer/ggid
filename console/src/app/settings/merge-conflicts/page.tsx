"use client";

import { useState, useEffect, useCallback } from "react";
import { GitMerge, ArrowRight, Check, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ConflictRule {
  id: string;
  rule: string;
  resource: string;
  version_a_effect: string;
  version_b_effect: string;
  conflict_type: "contradictory" | "overlapping" | "redundant";
}

interface Policy {
  id: string;
  name: string;
}

const conflictColors: Record<string, string> = {
  contradictory: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  overlapping: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  redundant: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
};

const strategies = [
  { value: "a_wins", label: "Version A Wins" },
  { value: "b_wins", label: "Version B Wins" },
  { value: "merge", label: "Smart Merge" },
  { value: "manual", label: "Manual Resolution" },
];

export default function MergeConflictsPage() {
  const t = useTranslations();

  const [policies, setPolicies] = useState<Policy[]>([]);
  const [policyA, setPolicyA] = useState("");
  const [policyB, setPolicyB] = useState("");
  const [conflicts, setConflicts] = useState<ConflictRule[]>([]);
  const [strategy, setStrategy] = useState("merge");
  const [loading, setLoading] = useState(false);

  const fetchPolicies = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/policy/list", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setPolicies(data.policies || data || []); }
    } catch { /* noop */ }
  }, []);

  const fetchConflicts = useCallback(async () => {
    if (!policyA || !policyB) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/merge-conflicts?a=${encodeURIComponent(policyA)}&b=${encodeURIComponent(policyB)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setConflicts(data.conflicts || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [policyA, policyB]);

  useEffect(() => { fetchPolicies(); }, [fetchPolicies]);
  useEffect(() => { fetchConflicts(); }, [fetchConflicts]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitMerge className="w-6 h-6 text-purple-500" /> {t("mergeConflicts.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Detect and resolve overlapping rules between policy versions.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">Policy Version A</label><select value={policyA} onChange={(e) => setPolicyA(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Policy</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
          <div><label className="text-sm font-medium">Policy Version B</label><select value={policyB} onChange={(e) => setPolicyB(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Policy</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
        </div>
        <div className="flex items-center gap-3"><label className="text-sm font-medium">Strategy:</label><select value={strategy} onChange={(e) => setStrategy(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">{strategies.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}</select></div>
      </div>

      {conflicts.length > 0 && (
        <div className="rounded-lg border border-yellow-300 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20 p-3 text-sm text-yellow-700 dark:text-yellow-400">{conflicts.length} conflicts detected between selected versions</div>
      )}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Rule</th><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Version A Effect</th><th className="px-4 py-3 text-left font-medium"></th><th className="px-4 py-3 text-left font-medium">Version B Effect</th><th className="px-4 py-3 text-left font-medium">Type</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {conflicts.map((c) => (
              <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs font-medium">{c.rule}</td>
                <td className="px-4 py-3 text-gray-500 text-xs font-mono">{c.resource}</td>
                <td className="px-4 py-3"><span className="flex items-center gap-1"><Check className="w-3.5 h-3.5 text-green-500" /><span className="text-xs">{c.version_a_effect}</span></span></td>
                <td className="px-4 py-3"><ArrowRight className="w-4 h-4 text-gray-400" /></td>
                <td className="px-4 py-3"><span className="flex items-center gap-1"><X className="w-3.5 h-3.5 text-red-500" /><span className="text-xs">{c.version_b_effect}</span></span></td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${conflictColors[c.conflict_type]}`}>{c.conflict_type}</span></td>
              </tr>
            ))}
            {conflicts.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">{policyA && policyB ? "No conflicts." : "Select two policy versions."}</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
