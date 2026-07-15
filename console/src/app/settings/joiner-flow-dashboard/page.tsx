"use client";
import { useState, useEffect, useCallback } from "react";
import { UserPlus, Clock, AlertTriangle, CheckCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface OnboardingItem { id: string; employee: string; start_date: string; steps_completed: number; total_steps: number; blocked_items: string[]; provisioning: { app: string; status: "pending" | "done" | "failed" }[]; }
interface DashboardData { pending: OnboardingItem[]; completion_rate: number; avg_days_to_complete: number; upcoming_starts: { employee: string; start_date: string }[]; }
export default function JoinerFlowDashboardPage() {
  const t = useTranslations();

  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/identity/joiner-dashboard", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const gaugeColor = data ? (data.completion_rate >= 80 ? "#10b981" : data.completion_rate >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><UserPlus className="w-6 h-6 text-green-500" /> {t("big1.joinerFlowDashboard.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.joinerFlowDashboard.trackEmployeeOnboardingProgressAndProvisioningStatus")}</p></div>
      {data && (<>
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-4"><div className="relative w-20 h-20"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={(data.completion_rate / 100) * 176 + " 176"} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex items-center justify-center"><span className="text-lg font-bold" style={{ color: gaugeColor }}>{data.completion_rate}%</span></div></div><div><span className="text-sm text-gray-500">{t("big1.joinerFlowDashboard.completionRate")}</span></div></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.joinerFlowDashboard.avgDaysToComplete")}</span><p className="text-2xl font-bold mt-1">{data.avg_days_to_complete}</p></div>
          <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.joinerFlowDashboard.pending")}</span><p className="text-2xl font-bold text-orange-600 mt-1">{data.pending.length}</p></div>
        </div>
        <div className="space-y-3">{data.pending.map((item) => (<div key={item.id} className="rounded-lg border dark:border-gray-800 p-3"><div className="flex items-center justify-between"><div><span className="font-medium text-sm">{item.employee}</span><span className="text-xs text-gray-400 ml-2">{t("big1.joinerFlowDashboard.starts")}{item.start_date}</span></div><div className="flex items-center gap-2"><div className="w-24 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className="h-full bg-blue-500 rounded-full" style={{ width: (item.steps_completed / item.total_steps) * 100 + "%" }} /></div><span className="text-xs font-bold">{item.steps_completed}/{item.total_steps}</span></div></div>{item.blocked_items.length > 0 && <div className="mt-2 flex items-center gap-1 text-xs text-orange-600"><AlertTriangle className="w-3 h-3" />{t("big1.joinerFlowDashboard.blocked")}{item.blocked_items.join(", ")}</div>}<div className="mt-2 flex flex-wrap gap-1">{item.provisioning.map((p, i) => (<span key={i} className={"px-2 py-0.5 rounded text-xs " + (p.status === "done" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : p.status === "failed" ? "bg-red-100 dark:bg-red-900/30 dark:text-red-400" : "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400")}>{p.app}: {p.status}</span>))}</div></div>))}{data.pending.length === 0 && <p className="text-sm text-gray-500 text-center py-4">{t("big1.joinerFlowDashboard.noPendingOnboarding")}</p>}</div>
        {data.upcoming_starts.length > 0 && <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">{t("big1.joinerFlowDashboard.upcomingStarts")}</h3><div className="space-y-1">{data.upcoming_starts.map((u, i) => (<div key={i} className="flex items-center gap-2 text-sm"><Clock className="w-3.5 h-3.5 text-gray-400" /><span className="font-medium">{u.employee}</span><span className="text-xs text-gray-400">{u.start_date}</span></div>))}</div></div>}
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("big1.joinerFlowDashboard.loading")}</p>}
    </div>
  );
}
