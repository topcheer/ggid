"use client";
import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { UserMinus, Clock, CheckCircle, AlertTriangle, RotateCcw } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface DeprovisionItem { id: string; username: string; scheduled_date: string; reason: string; status: "scheduled" | "in_progress" | "completed" | "failed"; steps: { name: string; status: "pending" | "done" | "failed" }[]; }
interface DashboardData { scheduled: DeprovisionItem[]; in_progress: DeprovisionItem[]; completed_today: number; failed: DeprovisionItem[]; }
const statusColors: Record<string, string> = { scheduled: "bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400", in_progress: "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400", completed: "bg-green-100 dark:bg-green-900/30 dark:text-green-400", failed: "bg-red-100 dark:bg-red-900/30 dark:text-red-400" };
export default function DeprovisionDashboardPage() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const t = useTranslations();
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/identity/deprovision-dashboard", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const retry = async (id: string) => { try { await fetch("/api/v1/identity/deprovision/" + id + "/retry", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); } catch { /* noop */ } };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><UserMinus className="w-6 h-6 text-red-500" /> {t("deprovisionDashboard.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("deprovisionDashboard.subtitle")}</p></div>
      {data && (<>
        <div className="grid grid-cols-4 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("deprovisionDashboard.scheduled")}</span><p className="text-xl font-bold mt-1">{data.scheduled.length}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("deprovisionDashboard.inProgress")}</span><p className="text-xl font-bold text-yellow-600 mt-1">{data.in_progress.length}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("deprovisionDashboard.completedToday")}</span><p className="text-xl font-bold text-green-600 mt-1">{data.completed_today}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("deprovisionDashboard.failed")}</span><p className="text-xl font-bold text-red-600 mt-1">{data.failed.length}</p></div>
        </div>
        {data.failed.length > 0 && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">{data.failed.length} failed deprovisioning items need attention</span></div>}
        <div className="space-y-3">{[...data.scheduled, ...data.in_progress].map((item: any) => (<div key={item.id} className="rounded-lg border dark:border-gray-800 p-3"><div className="flex items-center justify-between"><div><span className="font-medium text-sm">{item.username}</span><span className="text-xs text-gray-400 ml-2">{item.scheduled_date}</span></div><div className="flex items-center gap-2"><span className={"px-2 py-0.5 rounded text-xs " + statusColors[item.status]}>{item.status}</span></div></div><p className="text-xs text-gray-500 mt-1">{item.reason}</p><div className="mt-2 flex flex-wrap gap-1">{item.steps.map((s: any, i: number) => (<span key={i} className={"px-1.5 py-0.5 rounded text-xs " + (s.status === "done" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : s.status === "failed" ? "bg-red-100 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 dark:bg-gray-800")}>{s.name}: {s.status}</span>))}</div></div>))}</div>
        {data.failed.length > 0 && <div><h3 className="text-sm font-semibold mb-2">{t("deprovisionDashboard.failedItems")}</h3><div className="space-y-2">{data.failed.map((f: any) => (<div key={f.id} className="rounded-lg border border-red-200 dark:border-red-900 p-3 flex items-center justify-between"><div><span className="font-medium text-sm">{f.username}</span><p className="text-xs text-red-500">{f.steps.filter((s: any) => s.status === "failed").map((s: any) => s.name).join(", ")}</p></div><button onClick={() => retry(f.id)} className="px-3 py-1 rounded text-xs bg-blue-600 text-white flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Retry</button></div>))}</div></div>}
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("deprovisionDashboard.loading")}</p>}
    </div>
  );
}
