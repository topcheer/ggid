"use client";

import { useState, useEffect, useCallback } from "react";
import { UserMinus, Check, Clock, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Task {
  id: string;
  label: string;
  done: boolean;
  status: "pending" | "in_progress" | "done" | "failed";
}

interface LeaverData {
  employee_id: string;
  employee_name: string;
  scheduled_date: string;
  cascade_to_apps: boolean;
  tasks: Task[];
  completion_pct: number;
  overall_status: "scheduled" | "in_progress" | "completed" | "failed";
}

const statusIcons: Record<string, string> = {
  pending: "text-gray-400", in_progress: "text-blue-500 animate-pulse", done: "text-green-500", failed: "text-red-500",
};

export default function LeaverFlowPage() {
  const t = useTranslations();

  const [employeeId, setEmployeeId] = useState("");
  const [data, setData] = useState<LeaverData | null>(null);
  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<{ user_id: string; username: string }[]>([]);

  const fetchUsers = useCallback(async () => {
    try { const res = await fetch("/api/v1/identity/users", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setUsers(d.users || d || []); } } catch { /* noop */ }
  }, []);

  const fetchLeaver = useCallback(async () => {
    if (!employeeId) return;
    setLoading(true);
    try { const res = await fetch("/api/v1/identity/leaver-flow?user_id=" + encodeURIComponent(employeeId), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, [employeeId]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);
  useEffect(() => { fetchLeaver(); }, [fetchLeaver]);

  const toggleTask = (id: string) => { if (!data) return; setData({ ...data, tasks: data.tasks.map((t) => t.id === id ? { ...t, done: !t.done, status: !t.done ? "done" : "pending" } : t), completion_pct: Math.round((data.tasks.filter((t) => t.id !== id ? t.done : !t.done).length / data.tasks.length) * 100) }); };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><UserMinus className="w-6 h-6 text-red-500" /> {t("leaverFlow.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage employee offboarding with deprovisioning checklist and cascade.</p></div>

      <select value={employeeId} onChange={(e) => setEmployeeId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Employee</option>{users.map((u) => <option key={u.user_id} value={u.user_id}>{u.username}</option>)}</select>

      {data && (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Employee</span><p className="font-bold mt-1">{data.employee_name}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Scheduled</span><p className="font-bold mt-1 flex items-center gap-1"><Clock className="w-4 h-4 text-gray-400" />{data.scheduled_date}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Completion</span><div className="flex items-center gap-2 mt-1"><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-3 overflow-hidden"><div className={"h-full rounded-full " + (data.completion_pct === 100 ? "bg-green-500" : data.completion_pct > 50 ? "bg-blue-500" : "bg-orange-500")} style={{ width: data.completion_pct + "%" }} /></div><span className="font-bold text-sm">{data.completion_pct}%</span></div></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4 space-y-2">
            {data.tasks.map((t) => (<div key={t.id} className="flex items-center gap-3 p-2 rounded hover:bg-gray-50 dark:hover:bg-gray-900/30"><button onClick={() => toggleTask(t.id)} className={"w-5 h-5 rounded border-2 flex items-center justify-center " + (t.done ? "bg-green-500 border-green-500" : "border-gray-300 dark:border-gray-600")}>{t.done && <Check className="w-3 h-3 text-white" />}</button><span className={"text-sm flex-1 " + (t.done ? "line-through text-gray-400" : "")}>{t.label}</span>{t.status === "failed" && <AlertCircle className="w-4 h-4 text-red-500" />}<span className={"text-xs " + statusIcons[t.status]}>{t.status}</span></div>))}
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4"><label className="flex items-center gap-3 cursor-pointer"><input type="checkbox" checked={data.cascade_to_apps} onChange={() => setData({ ...data, cascade_to_apps: !data.cascade_to_apps })} className="rounded" /><span className="text-sm font-medium">Cascade to all connected apps</span><span className="text-xs text-gray-500">(disable accounts in all provisioned applications)</span></label></div>
        </>
      )}
      {!data && !loading && employeeId && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
      {!employeeId && <p className="text-sm text-gray-500 text-center py-8">Select an employee to manage offboarding.</p>}
    </div>
  );
}
