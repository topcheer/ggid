"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  FileText, Plus, Trash2, X, AlertCircle, Loader2, Check,
  Download, Clock, Calendar, Mail,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ComplianceSchedule {
  id: string;
  name: string;
  framework: string;
  frequency: string;
  recipients: string[];
  next_run: string;
  last_run?: string;
  format: string;
  enabled: boolean;
}

interface PastReport {
  id: string;
  schedule_name: string;
  framework: string;
  generated_at: string;
  size_kb: number;
  download_url: string;
}

const FRAMEWORKS = ["soc2", "hipaa", "gdpr", "iso27001", "pci"];
const FREQUENCIES = [
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
  { value: "monthly", label: "Monthly" },
  { value: "quarterly", label: "Quarterly" },
  { value: "annual", label: "Annual" },
];

export default function ComplianceReportsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [schedules, setSchedules] = useState<ComplianceSchedule[]>([]);
  const [pastReports, setPastReports] = useState<PastReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<ComplianceSchedule | null>(null);
  const [form, setForm] = useState({ name: "", framework: "soc2", frequency: "monthly", recipients: "", format: "pdf" });
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [schedRes, reportRes] = await Promise.all([
        apiFetch<{ schedules?: ComplianceSchedule[]; items?: ComplianceSchedule[] }>("/api/v1/audit/compliance/schedules").catch(() => ({ schedules: [] as ComplianceSchedule[], items: [] as ComplianceSchedule[] })),
        apiFetch<{ reports?: PastReport[]; items?: PastReport[] }>("/api/v1/audit/compliance/reports?limit=10").catch(() => ({ reports: [] as PastReport[], items: [] as PastReport[] })),
      ]);
      setSchedules(schedRes.schedules ?? schedRes.items ?? []);
      setPastReports(reportRes.reports ?? reportRes.items ?? []);
    } catch {
      setError("Failed to load compliance reports");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.name.trim()) return;
    setCreating(true);
    try {
      const recipients = form.recipients.split(",").map((r) => r.trim()).filter(Boolean);
      await apiFetch("/api/v1/audit/compliance/schedules", {
        method: "POST", body: JSON.stringify({ ...form, recipients }),
      });
      setForm({ name: "", framework: "soc2", frequency: "monthly", recipients: "", format: "pdf" });
      setShowCreate(false);
      await load();
    } catch {
      setError("Failed to create schedule");
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/audit/compliance/schedules/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError("Failed to delete schedule");
    }
  };

  const handleToggle = async (s: ComplianceSchedule) => {
    try {
      await apiFetch(`/api/v1/audit/compliance/schedules/${s.id}`, { method: "PATCH", body: JSON.stringify({ enabled: !s.enabled }) });
      await load();
    } catch {
      setError("Failed to toggle schedule");
    }
  };

  const handleDownload = (url: string) => {
    if (typeof window !== "undefined") window.open(url, "_blank");
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <FileText className="h-6 w-6 text-indigo-600" /> Compliance Reports
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Schedule automated compliance report generation and delivery.</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> New Schedule</button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Scheduled reports */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><Calendar className="h-4 w-4" /> Schedules ({schedules.length})</h2>
        {loading ? (
          <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-indigo-600" /></div>
        ) : schedules.length === 0 ? (
          <div className={cardCls}><div className="py-12 text-center"><FileText className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No scheduled reports.</p></div></div>
        ) : (
          <div className="space-y-3">
            {schedules.map((s) => (
              <div key={s.id} className={cardCls}>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-gray-800 dark:text-gray-200">{s.name}</span>
                      <span className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium uppercase text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{s.framework}</span>
                      <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{s.frequency}</span>
                      {!s.enabled && <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">Disabled</span>}
                    </div>
                    {s.recipients.length > 0 && (
                      <p className="mt-1 flex items-center gap-1 text-xs text-gray-400"><Mail className="h-3 w-3" />{s.recipients.join(", ")}</p>
                    )}
                    <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                      <span className="flex items-center gap-1"><Clock className="h-3 w-3" />Next: {new Date(s.next_run).toLocaleString()}</span>
                      {s.last_run && <span>Last: {new Date(s.last_run).toLocaleString()}</span>}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <label className="relative inline-flex cursor-pointer items-center">
                      <input type="checkbox" checked={s.enabled} onChange={() => handleToggle(s)} className="peer sr-only" />
                      <div className="h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:transition-all peer-checked:bg-indigo-600 peer-checked:after:translate-x-full dark:bg-gray-700" />
                    </label>
                    <button onClick={() => setConfirmDelete(s)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Past reports */}
      {pastReports.length > 0 && (
        <div>
          <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Recent Reports</h2>
          <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800"><tr className="text-left text-xs font-semibold uppercase text-gray-500">
                <th className="px-4 py-3">Schedule</th><th className="px-4 py-3">Framework</th><th className="px-4 py-3">Generated</th><th className="px-4 py-3">Size</th><th className="px-4 py-3 text-right">Download</th>
              </tr></thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {pastReports.map((r) => (
                  <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3 font-medium text-gray-700 dark:text-gray-300">{r.schedule_name}</td>
                    <td className="px-4 py-3"><span className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs uppercase text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{r.framework}</span></td>
                    <td className="px-4 py-3 text-gray-500">{new Date(r.generated_at).toLocaleString()}</td>
                    <td className="px-4 py-3 text-gray-400">{r.size_kb} KB</td>
                    <td className="px-4 py-3 text-right"><button onClick={() => handleDownload(r.download_url)} className="inline-flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"><Download className="h-3 w-3" /> Get</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">New Report Schedule</h2>
              <button onClick={() => setShowCreate(false)}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Name</label><input value={form.name} onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))} placeholder="Monthly SOC2 Report" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Framework</label><select value={form.framework} onChange={(e) => setForm((p) => ({ ...p, framework: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white">{FRAMEWORKS.map((f) => <option key={f} value={f}>{f.toUpperCase()}</option>)}</select></div>
                <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Frequency</label><select value={form.frequency} onChange={(e) => setForm((p) => ({ ...p, frequency: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white">{FREQUENCIES.map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}</select></div>
              </div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Recipients (comma-separated emails)</label><input value={form.recipients} onChange={(e) => setForm((p) => ({ ...p, recipients: e.target.value }))} placeholder="security@company.com, audit@company.com" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /></div>
              <div><label className="text-sm font-medium text-gray-700 dark:text-gray-300">Format</label><select value={form.format} onChange={(e) => setForm((p) => ({ ...p, format: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white">{["pdf", "csv", "json"].map((f) => <option key={f} value={f}>{f.toUpperCase()}</option>)}</select></div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleCreate} disabled={!form.name.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirm */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><div><h2 className="font-semibold text-gray-900 dark:text-white">Delete Schedule?</h2><p className="text-sm text-gray-500"><strong>{confirmDelete.name}</strong> will stop generating reports.</p></div></div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleDelete(confirmDelete.id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
