"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { GitMerge, Copy, Trash2, Play, CheckSquare, FileText } from "lucide-react";

interface ReconcileData {
  orphaned_ids: { id: string; type: string; source: string; last_seen: string }[];
  duplicate_groups: { group_id: string; entries: { id: string; source: string; email: string }[]; suggested_merge_target: string }[];
  cleanup_plan: { action: string; count: number; risk: "low" | "medium" | "high" }[];
}

const riskColors: Record<string, string> = {
  low: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function DirectoryReconcilePage() {
  const [data, setData] = useState<ReconcileData | null>(null);
  const [loading, setLoading] = useState(false);
  const [dryRun, setDryRun] = useState(true);
  const [mergeStrategy, setMergeStrategy] = useState("newest");
  const [executing, setExecuting] = useState(false);
  const [executed, setExecuted] = useState(false);
  const t = useTranslations();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/directory-reconcile?dry_run=${dryRun}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [dryRun]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const execute = async () => {
    setExecuting(true);
    try {
      await fetch("/api/v1/identity/directory-reconcile/execute", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ dry_run: dryRun, merge_strategy: mergeStrategy }) });
      setExecuted(true);
    } catch { /* noop */ }
    finally { setExecuting(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><GitMerge className="w-6 h-6 text-indigo-500" /> {t("directoryReconcile.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("directoryReconcile.subtitle")}</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
        <label className="flex items-center gap-2 text-sm font-medium"><input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} className="rounded" /> Dry Run</label>
        <div className="flex items-center gap-2"><label className="text-sm font-medium">{t("directoryReconcile.mergeStrategy")}</label><select value={mergeStrategy} onChange={(e) => setMergeStrategy(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="newest">{t("directoryReconcile.newestEntry")}</option><option value="oldest">{t("directoryReconcile.oldestEntry")}</option><option value="most_complete">{t("directoryReconcile.mostComplete")}</option><option value="manual">{t("directoryReconcile.manualReview")}</option></select></div>
      </div>

      {data && (
        <>
          {data.orphaned_ids.length > 0 && (
            <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("directoryReconcile.id")}</th><th className="px-4 py-3 text-left font-medium">{t("directoryReconcile.type")}</th><th className="px-4 py-3 text-left font-medium">{t("directoryReconcile.source")}</th><th className="px-4 py-3 text-left font-medium">{t("directoryReconcile.lastSeen")}</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{data.orphaned_ids.map((o, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{o.id}</td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{o.type}</span></td><td className="px-4 py-3 text-xs text-gray-500">{o.source}</td><td className="px-4 py-3 text-xs text-gray-400">{o.last_seen}</td></tr>))}</tbody>
              </table>
            </div>
          )}

          {data.duplicate_groups.length > 0 && (
            <div className="space-y-3">
              <h3 className="text-sm font-semibold">{t("directoryReconcile.duplicateGroups")}</h3>
              {data.duplicate_groups.map((g) => (
                <div key={g.group_id} className="rounded-lg border dark:border-gray-800 p-4">
                  <div className="flex items-center justify-between mb-2"><span className="text-xs font-mono text-gray-400">Group: {g.group_id}</span><span className="text-xs text-green-600">{t("directoryReconcile.mergeTarget")} {g.suggested_merge_target}</span></div>
                  <div className="space-y-1">{g.entries.map((e) => (
                    <div key={e.id} className="flex items-center gap-2 text-sm"><Copy className="w-3 h-3 text-gray-400" /><span className="font-mono text-xs">{e.id}</span><span className="text-gray-500">{e.email}</span><span className="text-xs text-gray-400">({e.source})</span></div>
                  ))}</div>
                </div>
              ))}
            </div>
          )}

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><FileText className="w-4 h-4 text-gray-400" /> Cleanup Plan</h3>
            <div className="space-y-2">{data.cleanup_plan.map((p, i) => (
              <div key={i} className="flex items-center gap-3 text-sm"><CheckSquare className="w-4 h-4 text-gray-400" /><span className="flex-1">{p.action}</span><span className="font-bold">{p.count}</span><span className={`px-2 py-0.5 rounded text-xs ${riskColors[p.risk]}`}>{p.risk}</span></div>
            ))}</div>
          </div>

          <button onClick={execute} disabled={executing || executed} className={`px-4 py-2 rounded-lg text-white text-sm font-medium flex items-center gap-2 ${dryRun ? "bg-gray-600 hover:bg-gray-700" : "bg-red-600 hover:bg-red-700"} disabled:opacity-50`}><Play className="w-4 h-4" /> {executing ? "Executing..." : executed ? "Done" : dryRun ? "Simulate" : "Execute Cleanup"}</button>
          {executed && <span className="text-sm text-green-600 ml-2">{dryRun ? "Simulation complete." : "Cleanup executed."}</span>}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("directoryReconcile.loading")}</p>}
    </div>
  );
}
