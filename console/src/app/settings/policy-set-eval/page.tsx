"use client";

import { useState } from "react";
import { Play, ShieldCheck, ShieldX, MinusCircle, ChevronRight } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface PolicyResult {
  policy_id: string;
  policy_name: string;
  decision: "allow" | "deny" | "no_match";
  matched_rule: string;
}

const decisionConfig: Record<string, { color: string; icon: typeof ShieldCheck; label: string }> = {
  allow: { color: "text-green-600", icon: ShieldCheck, label: "Allow" },
  deny: { color: "text-red-600", icon: ShieldX, label: "Deny" },
  no_match: { color: "text-gray-500", icon: MinusCircle, label: "No Match" },
};

export default function PolicySetEvalPage() {
  const t = useTranslations();

  const [subject, setSubject] = useState("");
  const [resource, setResource] = useState("");
  const [action, setAction] = useState("");
  const [results, setResults] = useState<PolicyResult[]>([]);
  const [finalDecision, setFinalDecision] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);

  const evaluate = async () => {
    if (!subject || !resource) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/set-evaluate", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ subject, resource, action: action || "access" }) });
      if (res.ok) { const data = await res.json(); setResults(data.results || []); setFinalDecision(data.final_decision || null); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Play className="w-6 h-6 text-blue-500" /> {t("policySetEval.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Batch-evaluate a request against all active policies in a policy set.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div><label className="text-sm font-medium">Subject</label><input aria-label="user:alice@example.com" type="text" value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="user:alice@example.com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Resource</label><input aria-label="doc:project-plan" type="text" value={resource} onChange={(e) => setResource(e.target.value)} placeholder="doc:project-plan" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Action</label><input aria-label="access (default)" type="text" value={action} onChange={(e) => setAction(e.target.value)} placeholder="access (default)" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <button aria-label="Play" onClick={evaluate} disabled={loading || !subject || !resource} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Play className="w-4 h-4" /> {loading ? "Evaluating..." : "Evaluate"}</button>
      </div>

      {finalDecision && (() => {
        const cfg = decisionConfig[finalDecision] || decisionConfig.no_match;
        const Icon = cfg.icon;
        return (
          <div className={`rounded-lg border-2 p-4 flex items-center gap-3 ${finalDecision === "allow" ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : finalDecision === "deny" ? "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20" : "border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/20"}`}><Icon className={`w-8 h-8 ${cfg.color}`} /><div><span className="text-sm text-gray-500">Final Decision</span><p className={`text-xl font-bold ${cfg.color}`}>{cfg.label}</p></div></div>
        );
      })()}

      {results.length > 0 && (
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Policy</th><th className="px-4 py-3 text-left font-medium">Decision</th><th className="px-4 py-3 text-left font-medium">Matched Rule</th></tr></thead>
            <tbody className="divide-y dark:divide-gray-800">
              {results.map((r) => {
                const cfg = decisionConfig[r.decision] || decisionConfig.no_match;
                const Icon = cfg.icon;
                return (
                  <><tr key={r.policy_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30 cursor-pointer" onClick={() => setExpanded(expanded === r.policy_id ? null : r.policy_id)}><td className="px-4 py-3"><div className="flex items-center gap-2"><ChevronRight className={`w-4 h-4 text-gray-400 transition-transform ${expanded === r.policy_id ? "rotate-90" : ""}`} /><div><span className="font-medium">{r.policy_name}</span><p className="text-xs text-gray-400 font-mono">{r.policy_id}</p></div></div></td><td className="px-4 py-3"><span className={`flex items-center gap-1 font-medium ${cfg.color}`}><Icon className="w-4 h-4" /> {cfg.label}</span></td><td className="px-4 py-3 font-mono text-xs text-gray-600 dark:text-gray-400">{r.matched_rule}</td></tr>{expanded === r.policy_id && <tr className="bg-gray-50 dark:bg-gray-900/30"><td colSpan={3} className="px-12 py-3 text-xs text-gray-500"><pre className="font-mono">{JSON.stringify(r, null, 2)}</pre></td></tr>}</>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
