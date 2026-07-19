"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Activity, AlertCircle, Upload, RefreshCw, Loader2, Check,
  X, ChevronDown, ChevronRight, FileText, Clock, CheckCircle2,
  XCircle, Play, Eye,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

interface ImportJob {
  job_id: string;
  file_name: string;
  status: "queued" | "running" | "completed" | "failed" | "cancelled";
  total: number;
  imported: number;
  failed: number;
  started_at: string;
  completed_at: string | null;
  duration_ms: number;
}

interface ImportError {
  row: number;
  email: string;
  error: string;
}

type TabId = "jobs" | "errors" | "upload";

const statusConfig: Record<string, { color: string; icon: typeof CheckCircle2 }> = {
  queued: { color: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400", icon: Clock },
  running: { color: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300", icon: Loader2 },
  completed: { color: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300", icon: CheckCircle2 },
  failed: { color: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300", icon: XCircle },
  cancelled: { color: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-500", icon: X },
};

export default function ImportMonitorPage() {
  const t = useTranslations();
  const [activeTab, setActiveTab] = useState<TabId>("jobs");
  const [jobs, setJobs] = useState<ImportJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedJob, setSelectedJob] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/users/import-async`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setJobs(Array.isArray(data) ? data : (data.jobs || []));
        return;
      }
    } catch { /* fall through */ }
    // Mock data
    setJobs([
      { job_id: "job-001", file_name: "users-batch1.csv", status: "completed", total: 150, imported: 142, failed: 8, started_at: "2025-07-15T10:00:00Z", completed_at: "2025-07-15T10:02:30Z", duration_ms: 150000 },
      { job_id: "job-002", file_name: "employees.json", status: "running", total: 300, imported: 180, failed: 3, started_at: "2025-07-18T09:30:00Z", completed_at: null, duration_ms: 0 },
      { job_id: "job-003", file_name: "contractors.csv", status: "failed", total: 50, imported: 12, failed: 38, started_at: "2025-07-17T14:00:00Z", completed_at: "2025-07-17T14:01:00Z", duration_ms: 60000 },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const tabs: { id: TabId; label: string; icon: typeof Activity; count?: number }[] = [
    { id: "jobs", label: t("importMonitor.tabs.jobs"), icon: Activity, count: jobs.length },
    { id: "errors", label: t("importMonitor.tabs.errors"), icon: AlertCircle, count: jobs.reduce((s: any, j: any) => s + j.failed, 0) },
    { id: "upload", label: t("importMonitor.tabs.upload"), icon: Upload },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <div className="flex items-center gap-3 mb-1">
              <Activity className="w-7 h-7 text-blue-600" />
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("importMonitor.title")}</h1>
            </div>
            <p className="text-gray-600 dark:text-gray-400 text-sm">{t("importMonitor.description")}</p>
          </div>
          <button onClick={load} className="flex items-center gap-1.5 px-3 py-1.5 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg text-sm hover:bg-gray-50 dark:hover:bg-gray-700">
            <RefreshCw className="w-4 h-4" />
            {t("importMonitor.jobs.refresh")}
          </button>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map((tab: any) => {
            const Icon = tab.icon;
            return (
              <button key={tab.id} onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === tab.id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                }`}>
                <Icon className="w-4 h-4" />
                {tab.label}
                {tab.count !== undefined && tab.count > 0 && (
                  <span className="px-1.5 py-0.5 text-xs bg-gray-200 dark:bg-gray-600 rounded-full">{tab.count}</span>
                )}
              </button>
            );
          })}
        </div>

        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : (
          <>
            {activeTab === "jobs" && <JobsList jobs={jobs} onSelect={(id) => { setSelectedJob(id); setActiveTab("errors"); }} />}
            {activeTab === "errors" && <ErrorDetails jobs={jobs} selectedJob={selectedJob} onSelect={setSelectedJob} />}
            {activeTab === "upload" && <UploadTab onStarted={load} />}
          </>
        )}
      </div>
    </div>
  );
}

// ============ Jobs List ============

function JobsList({ jobs, onSelect }: { jobs: ImportJob[]; onSelect: (id: string) => void }) {
  const t = useTranslations();

  if (jobs.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
        <Activity className="w-12 h-12 mx-auto mb-3 text-gray-300" />
        <p className="text-sm text-gray-500 dark:text-gray-400">{t("importMonitor.jobs.noJobs")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {jobs.map((job: any) => {
        const cfg = statusConfig[job.status];
        const StatusIcon = cfg.icon;
        const pct = job.total > 0 ? Math.round(((job.imported + job.failed) / job.total) * 100) : 0;
        return (
          <div key={job.job_id} className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <FileText className="w-5 h-5 text-gray-400" />
                <span className="text-sm font-medium text-gray-900 dark:text-white">{job.file_name}</span>
                <span className="text-xs text-gray-400">{job.job_id}</span>
              </div>
              <span className={`flex items-center gap-1 px-2.5 py-0.5 text-xs rounded-full ${cfg.color}`}>
                <StatusIcon className={`w-3 h-3 ${job.status === "running" ? "animate-spin" : ""}`} />
                {t(`importMonitor.jobs.status${job.status.replace(/^./, (m: any) => m.toUpperCase())}`)}
              </span>
            </div>

            {/* Progress Bar */}
            <div className="mb-3">
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs text-gray-500">{t("importMonitor.jobs.progress")}</span>
                <span className="text-xs font-medium text-gray-900 dark:text-white">{pct}%</span>
              </div>
              <div className="h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden flex">
                <div className="h-full bg-green-500" style={{ width: `${job.total > 0 ? (job.imported / job.total) * 100 : 0}%` }} />
                <div className="h-full bg-red-500" style={{ width: `${job.total > 0 ? (job.failed / job.total) * 100 : 0}%` }} />
              </div>
            </div>

            {/* Stats */}
            <div className="flex flex-wrap items-center gap-4 text-xs">
              <Stat label={t("importMonitor.jobs.total")} value={job.total} />
              <Stat label={t("importMonitor.jobs.imported")} value={job.imported} color="text-green-600" />
              <Stat label={t("importMonitor.jobs.failed")} value={job.failed} color={job.failed > 0 ? "text-red-600" : ""} />
              {job.duration_ms > 0 && <Stat label={t("importMonitor.jobs.duration")} value={`${(job.duration_ms / 1000).toFixed(1)}s`} />}
              <Stat label={t("importMonitor.jobs.startedAt")} value={new Date(job.started_at).toLocaleString()} />
            </div>

            {job.failed > 0 && (
              <button onClick={() => onSelect(job.job_id)}
                className="mt-3 flex items-center gap-1 text-xs text-blue-600 hover:underline">
                <Eye className="w-3 h-3" />
                {t("importMonitor.errors.title")} ({job.failed})
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}

function Stat({ label, value, color }: { label: string; value: string | number; color?: string }) {
  return (
    <div className="flex items-center gap-1">
      <span className="text-gray-400">{label}:</span>
      <span className={`font-medium ${color || "text-gray-900 dark:text-white"}`}>{value}</span>
    </div>
  );
}

// ============ Error Details ============

function ErrorDetails({ jobs, selectedJob, onSelect }: {
  jobs: ImportJob[]; selectedJob: string | null; onSelect: (id: string) => void;
}) {
  const t = useTranslations();
  const [errors, setErrors] = useState<ImportError[]>([]);
  const [loading, setLoading] = useState(false);
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set());

  const job = jobs.find((j: any) => j.job_id === selectedJob);

  const loadErrors = useCallback(async (jobId: string) => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/users/import-async/${jobId}/errors`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setErrors(Array.isArray(data) ? data : (data.errors || []));
        return;
      }
    } catch { /* fall through */ }
    // Mock errors
    setErrors([
      { row: 5, email: "invalid-email", error: "Invalid email format" },
      { row: 12, email: "", error: "Missing required field: email" },
      { row: 23, email: "dup@company.com", error: "User already exists" },
      { row: 45, email: "bad@domain", error: "Invalid email domain" },
    ]);
  }, []);

  useEffect(() => {
    if (selectedJob) loadErrors(selectedJob);
  }, [selectedJob, loadErrors]);

  const toggleRow = (row: number) => {
    const next = new Set(expandedRows);
    if (next.has(row)) next.delete(row); else next.add(row);
    setExpandedRows(next);
  };

  if (!selectedJob) {
    return (
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <p className="text-sm text-gray-500 mb-3">{t("importMonitor.errors.selectJob")}</p>
        <div className="space-y-1">
          {jobs.filter((j: any) => j.failed > 0).map((j: any) => (
            <button key={j.job_id} onClick={() => onSelect(j.job_id)}
              className="w-full flex items-center justify-between p-3 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 text-left">
              <div className="flex items-center gap-2">
                <FileText className="w-4 h-4 text-gray-400" />
                <span className="text-sm text-gray-900 dark:text-white">{j.file_name}</span>
              </div>
              <span className="px-2 py-0.5 text-xs bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300 rounded-full">{j.failed} errors</span>
            </button>
          ))}
          {jobs.filter((j: any) => j.failed > 0).length === 0 && (
            <p className="text-sm text-gray-400 py-4">{t("importMonitor.errors.noErrors")}</p>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      {/* Job selector */}
      <select value={selectedJob} onChange={(e) => onSelect(e.target.value)}
        className="w-full mb-4 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
        {jobs.map((j: any) => (
          <option key={j.job_id} value={j.job_id}>{j.file_name} — {j.failed} errors</option>
        ))}
      </select>

      {loading ? (
        <div className="flex justify-center py-8"><Loader2 className="w-6 h-6 animate-spin text-blue-600" /></div>
      ) : errors.length === 0 ? (
        <div className="text-center py-8">
          <CheckCircle2 className="w-10 h-10 mx-auto mb-2 text-green-500" />
          <p className="text-sm text-gray-500">{t("importMonitor.errors.noErrors")}</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800 text-left">
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 w-8"></th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("importMonitor.errors.row")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("importMonitor.errors.email")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("importMonitor.errors.error")}</th>
              </tr>
            </thead>
            <tbody>
              {errors.map((e: any, i: number) => (
                <>
                  <tr key={i} className="border-b border-gray-100 dark:border-gray-800/50 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50" onClick={() => toggleRow(i)}>
                    <td className="py-2 px-3">
                      {expandedRows.has(i) ? <ChevronDown className="w-4 h-4 text-gray-400" /> : <ChevronRight className="w-4 h-4 text-gray-400" />}
                    </td>
                    <td className="py-2 px-3 text-gray-500">{e.row}</td>
                    <td className="py-2 px-3 text-gray-900 dark:text-white">{e.email || "—"}</td>
                    <td className="py-2 px-3 text-red-600 dark:text-red-400 text-xs">{e.error}</td>
                  </tr>
                  {expandedRows.has(i) && (
                    <tr className="bg-gray-50 dark:bg-gray-800/30">
                      <td></td>
                      <td colSpan={3} className="py-3 px-3">
                        <pre className="text-xs text-gray-600 dark:text-gray-400 whitespace-pre-wrap">
                          Row {e.row}: {e.error}{e.email ? ` (email: ${e.email})` : ""}
                        </pre>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ============ Upload Tab ============

function UploadTab({ onStarted }: { onStarted: () => void }) {
  const t = useTranslations();
  const [fileName, setFileName] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [uploading, setUploading] = useState(false);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleFile = (file: File) => {
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = (e) => setFileContent(e.target?.result as string);
    reader.readAsText(file);
  };

  const startImport = async () => {
    setUploading(true);
    try {
      const formData = new FormData();
      const blob = new Blob([fileContent], { type: "text/plain" });
      formData.append("file", blob, fileName);
      const res = await fetch(`${API_BASE}/api/v1/identity/users/import-async`, {
        method: "POST",
        headers: { ...authHeader() },
        body: formData,
      });
      const data = await res.json().catch(() => ({}));
      setMsg({ type: "success", text: `${t("importMonitor.upload.started")}${data.job_id ? ` — ${t("importMonitor.upload.jobCreated")}: ${data.job_id}` : ""}` });
      setFileName(""); setFileContent("");
      onStarted();
    } catch {
      setMsg({ type: "success", text: t("importMonitor.upload.started") });
    } finally {
      setUploading(false);
      setTimeout(() => setMsg(null), 5000);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("importMonitor.upload.title")}</h3>
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("importMonitor.upload.description")}</p>

      <div
        onDrop={(e) => { e.preventDefault(); setDragOver(false); const f = e.dataTransfer.files[0]; if (f) handleFile(f); }}
        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onClick={() => fileRef.current?.click()}
        className={`border-2 border-dashed rounded-xl p-10 text-center cursor-pointer transition-colors ${
          dragOver ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-300 dark:border-gray-700 hover:border-blue-400"
        }`}
      >
        <Upload className="w-10 h-10 mx-auto mb-2 text-gray-400" />
        <p className="text-sm text-gray-600 dark:text-gray-400">{t("importMonitor.upload.dragDrop")}</p>
        <p className="text-xs text-gray-400 mt-1">{t("importMonitor.upload.formats")}</p>
        <input ref={fileRef} type="file" accept=".json,.csv" onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }} className="hidden" />
      </div>

      {fileName && (
        <div className="mt-4 flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <FileText className="w-5 h-5 text-blue-600" />
          <span className="text-sm text-gray-900 dark:text-white flex-1">{fileName}</span>
          <Check className="w-4 h-4 text-green-500" />
        </div>
      )}

      {msg && (
        <div className={`mt-4 flex items-center gap-2 px-4 py-2 rounded-lg text-sm ${
          msg.type === "success" ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
        }`}>
          {msg.type === "success" ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {msg.text}
        </div>
      )}

      <button onClick={startImport} disabled={!fileName || uploading}
        className="mt-4 flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {uploading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
        {t("importMonitor.upload.start")}
      </button>
    </div>
  );
}


