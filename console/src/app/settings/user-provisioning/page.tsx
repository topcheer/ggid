"use client";
import { useState, useEffect, useCallback } from "react";
import { UserPlus, Play, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ProvRule { id: string; source: string; trigger: string; action: string; target_app: string; enabled: boolean; }
interface QueueItem { id: string; user: string; app: string; status: "pending" | "processing" | "completed" | "failed"; error: string | null; }

export default function UserProvisioningPage() {
  const t = useTranslations();

  const [rules, setRules] = useState<ProvRule[]>([]);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [dryRun, setDryRun] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/provisioning", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setRules(d.rules || []); setQueue(d.queue || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const retryItem = async (id: string) => {
    try { await fetch("/api/v1/identity/provisioning/queue/" + id + "/retry", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><UserPlus className="w-6 h-6 text-green-500" /> {t("userProvisioning.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage automated provisioning rules across connected apps.</p></div>
        <label className="flex items-center gap-2 text-sm"><input aria-label="Dry run" type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} className="rounded" /> Dry Run</label>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Source</th><th className="px-4 py-3 text-left font-medium">Trigger</th><th className="px-4 py-3 text-left font-medium">Action</th><th className="px-4 py-3 text-left font-medium">Target App</th><th className="px-4 py-3 text-left font-medium">Enabled</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{rules.map((r) => (<tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{r.source}</span></td><td className="px-4 py-3 text-xs text-gray-500">{r.trigger}</td><td className="px-4 py-3 text-xs font-medium">{r.action}</td><td className="px-4 py-3 font-mono text-xs">{r.target_app}</td><td className="px-4 py-3">{r.enabled ? <span className="text-xs text-green-600">Yes</span> : <span className="text-xs text-gray-400">No</span>}</td></tr>))}{rules.length === 0 && !loading && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No provisioning rules.</td></tr>}</tbody></table></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Provisioning Queue</h3><div className="space-y-2">{queue.map((q) => (<div key={q.id} className="flex items-center gap-3 rounded-lg border dark:border-gray-700 p-2"><div className="flex-1"><div className="flex items-center gap-2"><span className="text-sm font-medium">{q.user}</span><span className="text-xs text-gray-400">{"->"}</span><span className="text-sm font-mono text-gray-500">{q.app}</span></div>{q.error && <p className="text-xs text-red-500 mt-0.5 flex items-center gap-1"><AlertCircle className="w-3 h-3" /> {q.error}</p>}</div><span className={"px-2 py-0.5 rounded text-xs " + (q.status === "completed" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : q.status === "failed" ? "bg-red-100 dark:bg-red-900/30 dark:text-red-400" : "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400")}>{q.status}</span>{q.status === "failed" && <button onClick={() => retryItem(q.id)} className="text-xs text-blue-600 hover:underline flex items-center gap-1"><Play className="w-3 h-3" /> Retry</button>}</div>))}{queue.length === 0 && <p className="text-xs text-gray-500">Queue is empty.</p>}</div></div>
    </div>
  );
}
