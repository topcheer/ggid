"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { AlertOctagon, ChevronRight } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ConflictPair {
  id: string;
  policy_a: string;
  policy_b: string;
  overlap_type: "resource" | "action" | "subject" | "rule";
  severity: "low" | "medium" | "high" | "critical";
  resource_pattern: string;
  description: string;
}

const sevColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function PolicyConflictsPage() {
  const t = useTranslations();
  const [conflicts, setConflicts] = useState<ConflictPair[]>([]);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/conflicts", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setConflicts(data.conflicts || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const critical = conflicts.filter((c) => c.severity === "critical").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><AlertOctagon className="w-6 h-6 text-red-500" />{t("policyConflicts.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Detect overlapping or contradictory rules across active policies.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Conflicts</span><p className="text-xl font-bold mt-1">{conflicts.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Critical</span><p className="text-xl font-bold text-red-600 mt-1">{critical}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Unique Policies</span><p className="text-xl font-bold mt-1">{new Set(conflicts.flatMap((c) => [c.policy_a, c.policy_b])).size}</p></div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Policy A</th><th className="px-4 py-3 text-left font-medium">Policy B</th><th className="px-4 py-3 text-left font-medium">Overlap</th><th className="px-4 py-3 text-left font-medium">Severity</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {conflicts.map((c) => (
              <><tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30 cursor-pointer" onClick={() => setExpanded(expanded === c.id ? null : c.id)}><td className="px-4 py-3 font-mono text-xs font-medium">{c.policy_a}</td><td className="px-4 py-3 font-mono text-xs font-medium">{c.policy_b}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400">{c.overlap_type}</span></td><td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${sevColors[c.severity]}`}>{c.severity}</span></td></tr>{expanded === c.id && <tr className="bg-gray-50 dark:bg-gray-900/30"><td colSpan={4} className="px-8 py-3"><div className="space-y-1 text-xs"><div><span className="text-gray-500">Resource: </span><span className="font-mono">{c.resource_pattern}</span></div><div><span className="text-gray-500">Description: </span>{c.description}</div></div></td></tr>}</>
            ))}
            {conflicts.length === 0 && !loading && <tr><td colSpan={4} className="px-4 py-8 text-center text-gray-500">No conflicts detected.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
