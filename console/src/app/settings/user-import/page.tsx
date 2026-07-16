"use client";
import { useState } from "react";
import { Upload, FileText, CheckCircle, AlertTriangle, X } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface Mapping { csv_column: string; user_attribute: string; }
interface ImportResults { created: number; updated: number; skipped: number; failed: number; errors: string[]; }
const userAttributes = ["username", "email", "first_name", "last_name", "department", "role", "phone"];
export default function UserImportPage() {
  const t = useTranslations();

  const [fileName, setFileName] = useState("");
  const [mappings, setMappings] = useState<Mapping[]>([{ csv_column: "name", user_attribute: "username" }, { csv_column: "email", user_attribute: "email" }]);
  const [dryRun, setDryRun] = useState(true);
  const [importing, setImporting] = useState(false);
  const [results, setResults] = useState<ImportResults | null>(null);
  const [preview, setPreview] = useState<{ errors: number; warnings: number; duplicates: number } | null>(null);
  const [importError, setImportError] = useState("");

  const onDrop = (e: React.DragEvent) => { e.preventDefault(); const file = e.dataTransfer.files[0]; if (file) { setFileName(file.name); setImportError(""); setPreview({ errors: 0, warnings: 2, duplicates: 1 }); } };
  const addMapping = () => setMappings([...mappings, { csv_column: "", user_attribute: "email" }]);
  const removeMapping = (i: number) => setMappings(mappings.filter((_, idx) => idx !== i));
  const updateMapping = (i: number, field: string, value: string) => setMappings(mappings.map((m, idx) => idx === i ? { ...m, [field]: value } : m));
  const doImport = async () => {
    setImporting(true);
    setImportError("");
    try {
      const res = await fetch("/api/v1/identity/users/import", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ file: fileName, mappings, dry_run: dryRun }) });
      if (!res.ok) return null;
      const data = await res.json();
      setResults({ created: data.created ?? 0, updated: data.updated ?? 0, skipped: data.skipped ?? 0, failed: data.failed ?? 0, errors: data.errors || [] });
    } catch (e) {
      setImportError(e instanceof Error ? e.message : "Failed to import users");
    } finally { setImporting(false); }
  };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Upload className="w-6 h-6 text-blue-500" /> {t("userImport.title")}</h1><p className="text-sm text-gray-500 mt-1">Bulk import users from CSV or JSON files.</p></div>
      {importError && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span>{importError}</span><button onClick={() => setImportError("")} className="text-xs underline hover:text-red-700">Dismiss</button></div>}
      {!fileName ? (
        <div onDrop={onDrop} onDragOver={(e) => e.preventDefault()} className="border-2 border-dashed dark:border-gray-700 rounded-lg p-12 text-center cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30">
          <Upload className="w-12 h-12 text-gray-400 mx-auto mb-2" />
          <p className="text-sm font-medium">Drop CSV/JSON here</p>
          <p className="text-xs text-gray-400 mt-1">or click to browse</p>
          <input aria-label="Input field" type="file" accept=".csv,.json" onChange={(e) => { const f = e.target.files?.[0]; if (f) { setFileName(f.name); setImportError(""); } }} className="hidden" id="user-import-file" />
        </div>
      ) : (
        <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-2">
          <FileText className="w-5 h-5 text-blue-500" />
          <span className="text-sm font-medium flex-1">{fileName}</span>
          <button onClick={() => { setFileName(""); setResults(null); setPreview(null); setImportError(""); }} aria-label="Remove file" className="text-gray-400"><X className="w-4 h-4" /></button>
        </div>
      )}
      {preview && <div className="grid grid-cols-3 gap-4"><div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Errors</span><p className={"text-lg font-bold " + (preview.errors > 0 ? "text-red-600" : "text-green-600")}>{preview.errors}</p></div><div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Warnings</span><p className={"text-lg font-bold " + (preview.warnings > 0 ? "text-yellow-600" : "text-green-600")}>{preview.warnings}</p></div><div className="rounded-lg border p-3 dark:border-gray-800"><span className="text-xs text-gray-500">Duplicates</span><p className={"text-lg font-bold " + (preview.duplicates > 0 ? "text-orange-600" : "text-green-600")}>{preview.duplicates}</p></div></div>}
      {fileName && <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><h3 className="text-sm font-semibold">Column Mapping</h3><div className="space-y-2">{mappings.map((m, i) => (<div key={i} className="flex items-center gap-2"><input type="text" value={m.csv_column} onChange={(e) => updateMapping(i, "csv_column", e.target.value)} placeholder="CSV column" aria-label={`CSV column ${i + 1}`} className="flex-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /><span className="text-gray-400 text-xs">{"->"}</span><select value={m.user_attribute} onChange={(e) => updateMapping(i, "user_attribute", e.target.value)} aria-label={`User attribute ${i + 1}`} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono">{userAttributes.map((a) => <option key={a} value={a}>{a}</option>)}</select><button onClick={() => removeMapping(i)} aria-label={`Remove mapping ${i + 1}`} className="text-red-500"><X className="w-4 h-4" /></button></div>))}<button onClick={addMapping} className="text-xs text-blue-600">Add Mapping</button></div><div className="flex items-center gap-4"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} className="rounded" /> Dry run (no changes)</label><button onClick={doImport} disabled={importing} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2" aria-label="Upload"><Upload className="w-4 h-4" /> {importing ? "Importing..." : dryRun ? "Test Import" : "Import"}</button></div></div>}
      {results && <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Import Results</h3><div className="grid grid-cols-4 gap-4 mb-3"><div><span className="text-xs text-gray-500">Created</span><p className="text-lg font-bold text-green-600">{results.created}</p></div><div><span className="text-xs text-gray-500">Updated</span><p className="text-lg font-bold text-blue-600">{results.updated}</p></div><div><span className="text-xs text-gray-500">Skipped</span><p className="text-lg font-bold text-yellow-600">{results.skipped}</p></div><div><span className="text-xs text-gray-500">Failed</span><p className="text-lg font-bold text-red-600">{results.failed}</p></div></div>{results.errors.length > 0 && <div className="space-y-1">{results.errors.map((e, i) => <div key={i} className="text-xs text-red-500 flex items-center gap-1"><AlertTriangle className="w-3 h-3" /> {e}</div>)}</div>}</div>}
    </div>
  );
}
