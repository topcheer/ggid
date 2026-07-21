"use client";
import { useState, useEffect } from "react";
import { Rocket, Loader2, Save, Upload, AlertCircle, CheckCircle, FileText } from "lucide-react";
import { useApi } from "@/lib/api";

interface ImportJob {
  id: string;
  status: "pending" | "running" | "completed" | "failed";
  total: number;
  processed: number;
  errors: number;
  created_at: string;
}

interface ImportConfig {
  dry_run: boolean;
  update_existing: boolean;
  send_welcome_email: boolean;
  default_role: string;
  default_org: string;
}

// Enhanced import page with dry-run support at /settings/import-enhanced
// Fixes 404 from settings grid navigation.
export default function EnhancedImportPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [config, setConfig] = useState<ImportConfig>({
    dry_run: true,
    update_existing: false,
    send_welcome_email: false,
    default_role: "user",
    default_org: "",
  });
  const [file, setFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<{ dry_run?: boolean; created?: number; updated?: number; errors?: number; total?: number; error_messages?: string[] } | null>(null);
  const [jobs, setJobs] = useState<ImportJob[]>([]);
  const [msg, setMsg] = useState<string | null>(null);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => {
    apiFetch<{ jobs?: ImportJob[] }>(`/api/v1/tenants/${TENANT_ID}/import/jobs`)
      .then((data) => {
        if (data?.jobs) setJobs(data.jobs);
      })
      .catch(() => {});
  }, []);

  const handleImport = async () => {
    if (!file) return;
    setImporting(true);
    setResult(null);
    setMsg(null);

    const formData = new FormData();
    formData.append("file", file);
    formData.append("dry_run", String(config.dry_run));
    formData.append("update_existing", String(config.update_existing));
    formData.append("send_welcome_email", String(config.send_welcome_email));
    formData.append("default_role", config.default_role);
    if (config.default_org) formData.append("default_org", config.default_org);

    try {
      const data = await apiFetch<any>(`/api/v1/tenants/${TENANT_ID}/import/users`, {
        method: "POST",
        body: formData,
      });
      setResult(data);
      setMsg(data.dry_run ? "Dry run completed — no changes made" : "Import completed");
    } catch (e) {
      setMsg("Import failed — check file format");
    }
    setImporting(false);
    setTimeout(() => setMsg(null), 5000);
  };

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Rocket className="h-6 w-6 text-blue-500" /> Enhanced Import
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Bulk import users from CSV/JSON with dry-run validation and progress tracking.
        </p>
      </div>

      <div className={card}>
        <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">Upload File</h3>
        <div className="space-y-4">
          <div>
            <label className="block w-full cursor-pointer rounded-lg border-2 border-dashed border-gray-300 p-8 text-center transition hover:border-blue-500 dark:border-gray-700">
              <input
                type="file"
                accept=".csv,.json"
                className="hidden"
                onChange={e => setFile(e.target.files?.[0] || null)}
              />
              {file ? (
                <div className="flex items-center justify-center gap-2 text-sm">
                  <FileText className="h-5 w-5 text-blue-500" />
                  <span className="font-medium">{file.name}</span>
                  <span className="text-gray-400">({(file.size / 1024).toFixed(1)} KB)</span>
                </div>
              ) : (
                <div className="text-gray-400">
                  <Upload className="mx-auto mb-2 h-8 w-8" />
                  <p className="text-sm">Click to upload CSV or JSON file</p>
                </div>
              )}
            </label>
          </div>

          <div className="space-y-3">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={config.dry_run} onChange={e => setConfig({ ...config, dry_run: e.target.checked })} className="rounded border-gray-300" />
              Dry run (validate without importing)
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={config.update_existing} onChange={e => setConfig({ ...config, update_existing: e.target.checked })} className="rounded border-gray-300" />
              Update existing users
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={config.send_welcome_email} onChange={e => setConfig({ ...config, send_welcome_email: e.target.checked })} className="rounded border-gray-300" />
              Send welcome email to new users
            </label>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-medium">Default Role</label>
              <select value={config.default_role} onChange={e => setConfig({ ...config, default_role: e.target.value })} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                <option value="user">User</option>
                <option value="admin">Admin</option>
                <option value="manager">Manager</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Default Organization</label>
              <input type="text" value={config.default_org} onChange={e => setConfig({ ...config, default_org: e.target.value })} placeholder="org-id (optional)" className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
            </div>
          </div>

          <button
            onClick={handleImport}
            disabled={!file || importing}
            className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {importing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {config.dry_run ? "Run Dry Run" : "Start Import"}
          </button>
        </div>
      </div>

      {msg && (
        <div className={`rounded-lg p-4 text-sm ${msg.includes("failed") ? "bg-red-50 text-red-600 dark:bg-red-950/30 dark:text-red-400" : "bg-green-50 text-green-600 dark:bg-green-950/30 dark:text-green-400"}`}>
          {msg}
        </div>
      )}

      {result && (
        <div className={card}>
          <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
            {result.dry_run ? <AlertCircle className="h-4 w-4 text-amber-500" /> : <CheckCircle className="h-4 w-4 text-green-500" />}
            {result.dry_run ? "Dry Run Results" : "Import Results"}
          </h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900 dark:text-white">{result.total || 0}</div>
              <div className="text-xs text-gray-400">Total</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">{result.created || 0}</div>
              <div className="text-xs text-gray-400">Created</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">{result.updated || 0}</div>
              <div className="text-xs text-gray-400">Updated</div>
            </div>
          </div>
          {result.errors ? (
            <div className="mt-3 text-sm text-red-600">{result.errors} errors</div>
          ) : null}
          {result.error_messages && result.error_messages.length > 0 && (
            <div className="mt-3 space-y-1">
              {result.error_messages.slice(0, 5).map((msg, i) => (
                <div key={i} className="text-xs text-red-500">{msg}</div>
              ))}
            </div>
          )}
        </div>
      )}

      {jobs.length > 0 && (
        <div className={card}>
          <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Import History</h3>
          <div className="space-y-2">
            {jobs.map(job => (
              <div key={job.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center gap-3">
                  <div className={`h-2 w-2 rounded-full ${job.status === "completed" ? "bg-green-500" : job.status === "running" ? "bg-blue-500" : job.status === "failed" ? "bg-red-500" : "bg-gray-400"}`} />
                  <span className="text-sm font-medium">{job.id}</span>
                </div>
                <div className="text-xs text-gray-400">
                  {job.processed}/{job.total} processed {job.errors > 0 && `· ${job.errors} errors`}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}