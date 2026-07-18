"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useCallback } from "react";
import { History, RotateCcw, GitCompare } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ChangeEntry {
  version: string;
  changed_by: string;
  changed_at: string;
  change_type: "create" | "modify" | "delete";
  diff_summary: string;
  approved_by: string;
}

interface Policy { id: string; name: string; }

const typeColors: Record<string, string> = {
  create: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  modify: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  delete: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function PolicyChangeHistoryPage() {
  const t = useTranslations();
  const [policies] = useState<Policy[]>([{ id: "p1", name: "Data Access" }, { id: "p2", name: "Admin Access" }, { id: "p3", name: "External Partner" }]);
  const [policyId, setPolicyId] = useState("");
  const [history, setHistory] = useState<ChangeEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedVersions, setSelectedVersions] = useState<string[]>([]);

  const fetchHistory = useCallback(async () => {
    if (!policyId) return;
    setLoading(true);
    try { const res = await fetch("/api/v1/policy/" + policyId + "/change-history", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setHistory(d.history || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, [policyId]);

  const rollback = async (version: string) => {
    try { await fetch("/api/v1/policy/" + policyId + "/rollback", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ version }) }); fetchHistory(); }
    catch { /* noop */ }
  };

  const toggleCompare = (version: string) => { setSelectedVersions(selectedVersions.includes(version) ? selectedVersions.filter((v) => v !== version) : selectedVersions.length < 2 ? [...selectedVersions, version] : [selectedVersions[1], version]); };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><History className="w-6 h-6 text-purple-500" />{t("policyChangeHistory.title")}</h1><p className="text-sm text-gray-500 mt-1">Track policy changes with version diffs, rollback, and comparison.</p></div>

      <div className="flex items-center gap-3">
        <select aria-label="Policy id" value={policyId} onChange={(e) => setPolicyId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Policy</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select>
        {selectedVersions.length === 2 && <span className="text-sm text-blue-600 flex items-center gap-1"><GitCompare className="w-4 h-4" /> Compare {selectedVersions[0]} vs {selectedVersions[1]}</span>}
      </div>

      <div className="relative pl-8">
        <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
        <div className="space-y-4">
          {history.map((h: any, i: number) => (
            <div key={i} className="relative">
              <div className={"absolute -left-5 w-4 h-4 rounded-full border-2 " + (h.change_type === "create" ? "bg-green-500 border-green-200" : h.change_type === "modify" ? "bg-yellow-500 border-yellow-200" : "bg-red-500 border-red-200")} />
              <div className="rounded-lg border dark:border-gray-800 p-3 ml-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2"><span className="font-mono text-sm font-bold">v{h.version}</span><span className={"px-2 py-0.5 rounded text-xs " + typeColors[h.change_type]}>{h.change_type}</span></div>
                  <span className="text-xs text-gray-400">{h.changed_at}</span>
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{h.diff_summary}</p>
                <div className="flex items-center gap-3 mt-2"><span className="text-xs text-gray-500">by {h.changed_by}</span>{h.approved_by && <span className="text-xs text-green-600">approved by {h.approved_by}</span>}</div>
                <div className="flex gap-2 mt-2"><button onClick={() => rollback(h.version)} className="text-xs font-medium text-purple-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Rollback</button><button onClick={() => toggleCompare(h.version)} className={"text-xs flex items-center gap-1 " + (selectedVersions.includes(h.version) ? "text-blue-600 font-medium" : "text-gray-500 hover:text-blue-600")}><GitCompare className="w-3 h-3" /> {selectedVersions.includes(h.version) ? "Selected" : "Compare"}</button></div>
              </div>
            </div>
          ))}
          {history.length === 0 && !loading && <p className="text-sm text-gray-500 py-4 ml-2">{policyId ? "No changes recorded." : "Select a policy."}</p>}
        </div>
      </div>
    </div>
  );
}
