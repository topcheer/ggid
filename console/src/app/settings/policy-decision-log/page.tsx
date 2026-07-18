"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { ScrollText, Download, Filter, ShieldCheck, ShieldX } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Decision {
  id: string;
  timestamp: string;
  policy_id: string;
  subject: string;
  resource: string;
  action: string;
  decision: "allow" | "deny";
  matched_rules: string[];
  evaluation_time_ms: number;
}

export default function PolicyDecisionLogPage() {
  const t = useTranslations();
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterDecision, setFilterDecision] = useState("");
  const [filterPolicy, setFilterPolicy] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/policy/decision-log", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setDecisions(d.decisions || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filtered = decisions.filter((d: any) => { if (filterDecision && d.decision !== filterDecision) return false; if (filterPolicy && !d.policy_id.includes(filterPolicy)) return false; return true; });
  const policies = [...new Set(decisions.map((d: any) => d.policy_id))];

  const exportCSV = () => { const csv = ["timestamp,policy_id,subject,resource,action,decision,matched_rules,eval_ms", ...filtered.map((d: any) => [d.timestamp, d.policy_id, d.subject, d.resource, d.action, d.decision, d.matched_rules.join(";"), d.evaluation_time_ms].join(","))].join("\n"); const blob = new Blob([csv], { type: "text/csv" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = "policy-decisions.csv"; a.click(); };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ScrollText className="w-6 h-6 text-blue-500" />{t("policyDecisionLog.title")}</h1><p className="text-sm text-gray-500 mt-1">Policy evaluation decisions with matched rules and timing.</p></div>
        <button onClick={exportCSV} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><Download className="w-3.5 h-3.5" /> Export CSV</button>
      </div>

      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-gray-400" />
        <select aria-label="Filter" value={filterDecision} onChange={(e) => setFilterDecision(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Decisions</option><option value="allow">Allow</option><option value="deny">Deny</option></select>
        <select aria-label="Filter" value={filterPolicy} onChange={(e) => setFilterPolicy(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Policies</option>{policies.map((p: any) => <option key={p} value={p}>{p}</option>)}</select>
        <span className="text-sm text-gray-500">{filtered.length} entries</span>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Timestamp</th><th className="px-4 py-3 text-left font-medium">Policy</th><th className="px-4 py-3 text-left font-medium">Subject</th><th className="px-4 py-3 text-left font-medium">Resource</th><th className="px-4 py-3 text-left font-medium">Action</th><th className="px-4 py-3 text-left font-medium">Decision</th><th className="px-4 py-3 text-left font-medium">Rules</th><th className="px-4 py-3 text-left font-medium">Time</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{filtered.map((d: any) => (<tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 text-xs text-gray-400">{d.timestamp}</td><td className="px-4 py-3 font-mono text-xs">{d.policy_id}</td><td className="px-4 py-3 font-mono text-xs">{d.subject}</td><td className="px-4 py-3 font-mono text-xs">{d.resource}</td><td className="px-4 py-3 text-xs">{d.action}</td><td className="px-4 py-3">{d.decision === "allow" ? <span className="flex items-center gap-1 text-xs text-green-600"><ShieldCheck className="w-3.5 h-3.5" /> allow</span> : <span className="flex items-center gap-1 text-xs text-red-600"><ShieldX className="w-3.5 h-3.5" /> deny</span>}</td><td className="px-4 py-3"><div className="flex flex-wrap gap-1">{d.matched_rules.slice(0, 2).map((r: any, i: number) => <span key={i} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{r}</span>)}{d.matched_rules.length > 2 && <span className="text-xs text-gray-400">+{d.matched_rules.length - 2}</span>}</div></td><td className="px-4 py-3 text-xs text-gray-500">{d.evaluation_time_ms}ms</td></tr>))}{filtered.length === 0 && !loading && <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No decisions logged.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}
