"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { CalendarClock, Plus, X, GitBranch, Bell } from "lucide-react";

interface DeprovisionJob {
  id: string;
  user_id: string;
  username: string;
  scheduled_at: string;
  reason: string;
  cascade_to_apps: boolean;
  notify_before_days: number;
  status: "scheduled" | "completed" | "cancelled";
}

const statusColors: Record<string, string> = {
  scheduled: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  completed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  cancelled: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
};

export default function DeprovisionSchedulePage() {
  const [jobs, setJobs] = useState<DeprovisionJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ user_id: "", scheduled_at: "", reason: "", cascade_to_apps: true, notify_before_days: 7 });
  const t = useTranslations();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/deprovision-schedule", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setJobs(d.jobs || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const create = async () => {
    if (!form.user_id || !form.scheduled_at) return;
    try { await fetch("/api/v1/identity/deprovision-schedule", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowCreate(false); setForm({ user_id: "", scheduled_at: "", reason: "", cascade_to_apps: true, notify_before_days: 7 }); fetchData(); }
    catch { /* noop */ }
  };

  const cancel = async (id: string) => {
    try { await fetch(`/api/v1/identity/deprovision-schedule/${id}`, { method: "DELETE", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); }
    catch { /* noop */ }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><CalendarClock className="w-6 h-6 text-orange-500" /> {t("deprovisionSchedule.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("deprovisionSchedule.subtitle")}</p></div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 flex items-center gap-2"><Plus className="w-4 h-4" /> {t("deprovisionSchedule.schedule")}</button>
      </div>

      <div className="relative pl-8">
        <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
        <div className="space-y-4">
          {jobs.map((job) => (
            <div key={job.id} className="relative">
              <div className={`absolute -left-5 w-4 h-4 rounded-full border-2 ${job.status === "scheduled" ? "bg-blue-500 border-blue-200" : job.status === "completed" ? "bg-green-500 border-green-200" : "bg-gray-400 border-gray-200"}`} />
              <div className="rounded-lg border dark:border-gray-800 p-4 ml-2">
                <div className="flex items-center justify-between">
                  <div><span className="font-semibold">{job.username}</span><p className="text-xs text-gray-400 font-mono">{job.user_id}</p></div>
                  <span className={`px-2 py-0.5 rounded text-xs ${statusColors[job.status]}`}>{job.status}</span>
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2 text-sm">
                  <div className="flex items-center gap-1"><CalendarClock className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">{t("deprovisionSchedule.scheduledLabel")}</span><span className="font-medium">{job.scheduled_at}</span></div>
                  <div className="flex items-center gap-1"><Bell className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">{t("deprovisionSchedule.notify")}</span><span className="font-medium">{job.notify_before_days}d before</span></div>
                  <div className="flex items-center gap-1 col-span-2"><GitBranch className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">{t("deprovisionSchedule.cascade")}</span><span className={`text-xs ${job.cascade_to_apps ? "text-green-600" : "text-gray-400"}`}>{job.cascade_to_apps ? "All apps" : "Identity only"}</span></div>
                  <div className="col-span-2"><span className="text-gray-500 text-xs">{t("deprovisionSchedule.reasonLabel")}</span><span className="text-sm ml-1">{job.reason}</span></div>
                </div>
                {job.status === "scheduled" && <button onClick={() => cancel(job.id)} className="mt-2 text-xs text-red-600 hover:underline">{t("deprovisionSchedule.cancel")}</button>}
              </div>
            </div>
          ))}
          {jobs.length === 0 && !loading && <p className="text-sm text-gray-500 py-4 ml-2">{t("deprovisionSchedule.noScheduled")}</p>}
        </div>
      </div>

      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Schedule Deprovisioning</h3><button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">{t("deprovisionSchedule.userId")}</label><input type="text" value={form.user_id} onChange={(e) => setForm({ ...form, user_id: e.target.value })} placeholder="usr-xxxx" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("deprovisionSchedule.scheduledAt")}</label><input type="datetime-local" value={form.scheduled_at} onChange={(e) => setForm({ ...form, scheduled_at: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("deprovisionSchedule.reason")}</label><input type="text" value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} placeholder="Contract end" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("deprovisionSchedule.notifyBefore")}</label><input type="number" min={0} value={form.notify_before_days} onChange={(e) => setForm({ ...form, notify_before_days: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={form.cascade_to_apps} onChange={(e) => setForm({ ...form, cascade_to_apps: e.target.checked })} className="rounded" /> Cascade to all connected apps</label>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("deprovisionSchedule.cancel")}</button><button onClick={create} disabled={!form.user_id || !form.scheduled_at} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 disabled:opacity-50">{t("deprovisionSchedule.schedule")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
