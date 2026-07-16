"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useCallback } from "react";
import { Download, Upload, FileJson, GitCompare, Check, X } from "lucide-react";

export default function PolicyExportPage() {
  const t = useTranslations();
  const [exporting, setExporting] = useState(false);
  const [importJson, setImportJson] = useState("");
  const [diff, setDiff] = useState<{ added: string[]; removed: string[]; modified: string[] } | null>(null);
  const [importing, setImporting] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const retryExport = () => { setError(""); doExport(); };

  const doExport = useCallback(async () => {
    setExporting(true); setError("");
    try {
      const res = await fetch("/api/v1/policy/export", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url; a.download = `policies-${new Date().toISOString().split("T")[0]}.json`; a.click();
      URL.revokeObjectURL(url);
      setMessage("Export completed successfully.");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to export policies");
    } finally { setExporting(false); }
  }, []);

  const previewImport = useCallback(async () => {
    if (!importJson) return;
    setImporting(true); setError("");
    try {
      const res = await fetch("/api/v1/policy/import-preview", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: importJson });
      if (!res.ok) return null;
      setDiff(await res.json());
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to preview import");
    } finally { setImporting(false); }
  }, [importJson]);

  const doImport = async () => {
    setImporting(true); setError("");
    try {
      const res = await fetch("/api/v1/policy/import", { method: "POST", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: importJson });
      if (!res.ok) return null;
      setMessage("Import completed successfully."); setDiff(null); setImportJson("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to import policies");
    } finally { setImporting(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><FileJson className="w-6 h-6 text-blue-500" />{t("policyExport.title")}</h1><p className="text-sm text-gray-500 mt-1">Export policy configurations as JSON or import with diff preview.</p></div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Export */}
        <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
          <h3 className="font-semibold flex items-center gap-2"><Download className="w-4 h-4 text-blue-500" /> Export</h3>
          <p className="text-sm text-gray-500">Download all policies, roles, and permission rules as a JSON package.</p>
          <button aria-label="Download" onClick={doExport} disabled={exporting} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Download className="w-4 h-4" /> {exporting ? "Exporting..." : "Export All Policies"}</button>
        </div>

        {/* Import */}
        <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
          <h3 className="font-semibold flex items-center gap-2"><Upload className="w-4 h-4 text-green-500" /> Import</h3>
          <textarea aria-label="Paste JSON policy package..." value={importJson} onChange={(e) => setImportJson(e.target.value)} placeholder="Paste JSON policy package..." rows={5} className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" />
          <div className="flex items-center gap-2">
            <button onClick={previewImport} disabled={!importJson || importing} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><GitCompare className="w-4 h-4" /> Preview Diff</button>
            {diff && <button aria-label="action" onClick={doImport} disabled={importing} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"><Upload className="w-4 h-4" /> {importing ? "Importing..." : "Execute Import"}</button>}
          </div>
        </div>
      </div>

      {/* Diff results */}
      {diff && (
        <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
          <h3 className="font-semibold flex items-center gap-2"><GitCompare className="w-4 h-4" /> Import Diff Preview</h3>
          {diff.added.length > 0 && <div><h4 className="text-sm font-medium text-green-600 mb-1">Added ({diff.added.length})</h4>{diff.added.map((s, i) => <div key={i} className="text-xs font-mono text-green-600">+ {s}</div>)}</div>}
          {diff.removed.length > 0 && <div><h4 className="text-sm font-medium text-red-600 mb-1">Removed ({diff.removed.length})</h4>{diff.removed.map((s, i) => <div key={i} className="text-xs font-mono text-red-600 line-through">- {s}</div>)}</div>}
          {diff.modified.length > 0 && <div><h4 className="text-sm font-medium text-yellow-600 mb-1">Modified ({diff.modified.length})</h4>{diff.modified.map((s, i) => <div key={i} className="text-xs font-mono text-yellow-600">~ {s}</div>)}</div>}
          {diff.added.length === 0 && diff.removed.length === 0 && diff.modified.length === 0 && <p className="text-sm text-gray-500">No changes detected.</p>}
        </div>
      )}

      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span>{error}</span><button onClick={retryExport} className="text-xs underline hover:text-red-700">Retry</button></div>}
      {message && <div className="rounded-lg border border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-900/20 p-3 text-sm text-green-700 dark:text-green-400 flex items-center gap-2"><Check className="w-4 h-4" /> {message}</div>}
    </div>
  );
}
