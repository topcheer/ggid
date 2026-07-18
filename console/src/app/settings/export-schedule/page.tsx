"use client";
import { useState, useEffect, useCallback } from "react";
import { Download, Plus, Trash2, Cloud, Mail, Link as LinkIcon, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface ExportJob { id: string; format: string; schedule_cron: string; filters: string; retention_days: number; destination_type: string; destination_config: string; last_export_at: string | null; status: "active" | "paused" | "failed"; }
const statusColors: Record<string, string> = { active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400", paused: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400", failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" };
const destIcons: Record<string, typeof Cloud> = { s3: Cloud, https: LinkIcon, email: Mail };
export default function ExportSchedulePage() {
  const t = useTranslations();

  const [jobs, setJobs] = useState<ExportJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ format: "csv", schedule_cron: "0 2 * * *", filters: "", retention_days: 30, destination_type: "s3", destination_config: "" });
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/audit/export-schedule", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json();
      setJobs(d.jobs || d || []);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load export schedules"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const create = async () => {
    setError(null);
    try {
      const res = await fetch("/api/v1/audit/export-schedule", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) });
      if (!res.ok) return null;
      setShowCreate(false); fetchData();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to create schedule"); }
  };
  const remove = async (id: string) => {
    try {
      const res = await fetch(`/api/v1/audit/export-schedule/${id}`, { method: "DELETE", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      fetchData();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to delete schedule"); }
  };
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Download className="w-6 h-6 text-blue-500" /> {t("big1.exportSchedule.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.exportSchedule.scheduleRecurringAuditDataExportsToExternalDestinations")}</p></div>
        <button onClick={() => { setShowCreate(true); setError(null); }} aria-label="Create new export schedule" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" />{t("big1.exportSchedule.newExport")}</button>
      </div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => { setError(null); fetchData(); }} className="text-xs underline hover:text-red-700">{t("big1.exportSchedule.retry")}</button></div>}
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">{t("big1.exportSchedule.loadingExportSchedules")}</div></div>}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("big1.exportSchedule.format")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.exportSchedule.schedule")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.exportSchedule.retention")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.exportSchedule.destination")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.exportSchedule.lastExport")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.exportSchedule.status")}</th><th className="px-4 py-3 text-left font-medium"></th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{jobs.map((j: any) => { const DestIcon = destIcons[j.destination_type] || Cloud; return (
            <tr key={j.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs font-mono bg-gray-100 dark:bg-gray-800">{j.format}</span></td><td className="px-4 py-3 font-mono text-xs">{j.schedule_cron}</td><td className="px-4 py-3 text-xs text-gray-500">{j.retention_days}{t("big1.exportSchedule.d")}</td><td className="px-4 py-3"><span className="flex items-center gap-1 text-xs"><DestIcon className="w-3 h-3 text-gray-400" />{j.destination_type}</span></td><td className="px-4 py-3 text-xs text-gray-400">{j.last_export_at || "Never"}</td><td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[j.status]}`}>{j.status}</span></td><td className="px-4 py-3"><button onClick={() => remove(j.id)} aria-label={`Delete export schedule ${j.id}`} className="text-red-500 hover:text-red-700"><Trash2 className="w-4 h-4" /></button></td></tr>
          ); })}{jobs.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">{t("big1.exportSchedule.noExportSchedules")}</td></tr>}</tbody>
        </table>
      </div>
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("big1.exportSchedule.newExportSchedule")}</h3><button onClick={() => setShowCreate(false)} aria-label="Close dialog" className="text-gray-400"><Plus className="w-5 h-5 rotate-45" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">{t("big1.exportSchedule.format")}</label><select value={form.format} onChange={(e) => setForm({ ...form, format: e.target.value })} aria-label="Export format" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="csv">{t("big1.exportSchedule.csv")}</option><option value="json">{t("big1.exportSchedule.json")}</option><option value="parquet">{t("big1.exportSchedule.parquet")}</option></select></div>
              <div><label className="text-sm font-medium">{t("big1.exportSchedule.scheduleCron")}</label><input type="text" value={form.schedule_cron} onChange={(e) => setForm({ ...form, schedule_cron: e.target.value })} placeholder="0 2 * * *" aria-label="Schedule cron" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("big1.exportSchedule.filtersOptional")}</label><input type="text" value={form.filters} onChange={(e) => setForm({ ...form, filters: e.target.value })} placeholder="event_type=login" aria-label="Filters" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("big1.exportSchedule.retentionDays")}</label><input type="number" min={1} value={form.retention_days} onChange={(e) => setForm({ ...form, retention_days: parseInt(e.target.value) || 0 })} aria-label="Retention days" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("big1.exportSchedule.destinationType")}</label><select value={form.destination_type} onChange={(e) => setForm({ ...form, destination_type: e.target.value })} aria-label="Destination type" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm"><option value="s3">{t("big1.exportSchedule.s3")}</option><option value="https">{t("big1.exportSchedule.httpsWebhook")}</option><option value="email">{t("big1.exportSchedule.email")}</option></select></div>
              <div><label className="text-sm font-medium">{t("big1.exportSchedule.destinationConfig")}</label><input type="text" value={form.destination_config} onChange={(e) => setForm({ ...form, destination_config: e.target.value })} placeholder="s3://bucket/path or https://url or email@addr" aria-label="Destination config" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("big1.exportSchedule.cancel")}</button><button onClick={create} aria-label="Create schedule" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700">{t("big1.exportSchedule.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
