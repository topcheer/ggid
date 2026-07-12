"use client";

import { useState, useCallback } from "react";
import { Users, TrendingDown, Ban } from "lucide-react";

interface UserEntry {
  id: string;
  username: string;
  stage: "active" | "dormant" | "suspended" | "deactivated" | "pending";
  last_active: string;
  days_inactive: number;
  stage_since: string;
}

const stages = ["active", "dormant", "suspended", "deactivated", "pending"] as const;
const stageColors: Record<string, string> = {
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  dormant: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  suspended: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  deactivated: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  pending: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function UserLifecyclePage() {
  const [tab, setTab] = useState<string>("active");
  const [users, setUsers] = useState<UserEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [autoDays, setAutoDays] = useState(90);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/identity/user-lifecycle?stage=" + tab, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setUsers(d.users || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, [tab]);

  useState(() => { fetchData(); });

  const bulkAction = async (action: string) => {
    try { await fetch("/api/v1/identity/user-lifecycle/bulk", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ action, stage: tab, user_ids: users.map((u) => u.id) }) }); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-blue-500" /> User Lifecycle</h1><p className="text-sm text-gray-500 mt-1">Track user lifecycle stages with auto-deactivation rules.</p></div>

      <div className="flex items-center gap-2 flex-wrap">
        {stages.map((s) => <button key={s} onClick={() => setTab(s)} className={"px-4 py-2 rounded-lg text-sm font-medium capitalize " + (tab === s ? "bg-blue-600 text-white" : "border dark:border-gray-700")}>{s}</button>)}
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
        <TrendingDown className="w-5 h-5 text-orange-500" /><div><label className="text-sm font-medium">Auto-deactivate after</label><div className="flex items-center gap-2 mt-1"><input type="number" min={30} max={365} value={autoDays} onChange={(e) => setAutoDays(parseInt(e.target.value))} className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /><span className="text-sm text-gray-500">days of inactivity</span></div></div>
        <div className="ml-auto flex gap-2"><button onClick={() => bulkAction("deactivate")} className="px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700">Deactivate All</button><button onClick={() => bulkAction("notify")} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-xs">Notify All</button></div>
      </div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Stage</th><th className="px-4 py-3 text-left font-medium">Last Active</th><th className="px-4 py-3 text-left font-medium">Days Inactive</th><th className="px-4 py-3 text-left font-medium">Since</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{users.map((u) => (<tr key={u.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{u.username}</td><td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + stageColors[u.stage]}>{u.stage}</span></td><td className="px-4 py-3 text-xs text-gray-500">{u.last_active}</td><td className="px-4 py-3"><span className={"font-bold " + (u.days_inactive > 90 ? "text-red-600" : u.days_inactive > 60 ? "text-orange-600" : "text-gray-600")}>{u.days_inactive}d</span></td><td className="px-4 py-3 text-xs text-gray-400">{u.stage_since}</td></tr>))}{users.length === 0 && !loading && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">No users in this stage.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}
