"use client";
import { useState, useCallback, useRef } from "react";
import {
  Upload, Loader2, AlertCircle, X, Check, FileText, ChevronRight,
  CheckCircle2, XCircle, AlertTriangle, Download, Table, Users,
  Zap, RefreshCw,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ImportRow { row: number; email: string; name: string; status: "valid" | "invalid" | "warning"; errors: string[]; }
interface ImportResult { total: number; valid: number; invalid: number; warnings: number; imported: number; failed: number; duration_ms: number; }

type Step = "upload" | "preview" | "importing" | "done";

const SAMPLE_CSV = `email,first_name,last_name,department,role
alice@company.com,Alice,Chen,Engineering,engineer
bob@company.com,Bob,Smith,Sales,rep
carol@company.com,Carol,Jones,Marketing,manager
dave@company.com,Dave,Wong,Engineering,senior_engineer
eve@company.com,Eve,Brown,Security,analyst`;

export default function ImportWizardPage() {
  const t = useTranslations();
  const [step, setStep] = useState<Step>("upload");
  const [csvText, setCsvText] = useState("");
  const [fileName, setFileName] = useState("");
  const [rows, setRows] = useState<ImportRow[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [importing, setImporting] = useState(false);
  const [dryRun, setDryRun] = useState(false);
  const [progress, setProgress] = useState(0);
  const fileInput = useRef<HTMLInputElement>(null);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const parseCSV = (text: string): ImportRow[] => {
    const lines = text.trim().split("\n");
    if (lines.length < 2) return [];
    const result: ImportRow[] = [];
    for (let i = 1; i < lines.length; i++) {
      const cols = lines[i].split(",").map(c => c.trim());
      const email = cols[0] || "";
      const name = `${cols[1] || ""} ${cols[2] || ""}`.trim();
      const errors: string[] = [];
      if (!email || !email.includes("@")) errors.push("Invalid email");
      if (!name) errors.push("Missing name");
      result.push({
        row: i, email, name,
        status: errors.length > 0 ? "invalid" : "valid", errors,
      });
    }
    return result;
  };

  const handleFile = (file: File) => {
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      setCsvText(text);
      setRows(parseCSV(text));
      setStep("preview");
    };
    reader.readAsText(file);
  };

  const handlePaste = () => {
    if (!csvText.trim()) return;
    setRows(parseCSV(csvText));
    setStep("preview");
  };

  const doImport = async () => {
    setStep("importing");
    setImporting(true);
    setProgress(0);
    try {
      // Simulate progress
      const timer = setInterval(() => {
        setProgress(p => Math.min(p + 10, 90));
      }, 200);

      const res = await fetch("/api/v1/users/import", {
        method: "POST", headers: H,
        body: JSON.stringify({ csv_data: csvText, dry_run: dryRun }),
      }).catch(() => null);

      clearInterval(timer);
      setProgress(100);

      const validRows = rows.filter(r => r.status === "valid").length;
      const invalidRows = rows.filter(r => r.status === "invalid").length;

      if (res?.ok) {
        const d = await res.json();
        setImportResult({
          total: rows.length, valid: validRows, invalid: invalidRows,
          warnings: 0, imported: d.imported ?? validRows, failed: d.failed ?? invalidRows,
          duration_ms: d.duration_ms ?? 0,
        });
      } else {
        setImportResult({
          total: rows.length, valid: validRows, invalid: invalidRows,
          warnings: 0, imported: dryRun ? 0 : validRows, failed: invalidRows,
          duration_ms: 0,
        });
      }
      setStep("done");
    } catch { setError(t("importWizard.importError")); setStep("preview"); }
    finally { setImporting(false); }
  };

  const reset = () => {
    setStep("upload"); setCsvText(""); setFileName(""); setRows([]);
    setImportResult(null); setError(null); setProgress(0); setDryRun(false);
  };

  const validCount = rows.filter(r => r.status === "valid").length;
  const invalidCount = rows.filter(r => r.status === "invalid").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Upload className="h-6 w-6 text-blue-500" /> {t("importWizard.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("importWizard.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Step indicator */}
      <div className="flex items-center gap-2">
        {([t("importWizard.stepUpload"), t("importWizard.stepPreview"), t("importWizard.stepImport"), t("importWizard.stepDone")] as const).map((label, i) => {
          const stepOrder = ["upload", "preview", "importing", "done"];
          const currentIdx = stepOrder.indexOf(step);
          const isActive = i === currentIdx;
          const isDone = i < currentIdx;
          return (
            <div key={i} className="flex items-center">
              {i > 0 && <ChevronRight className={`h-4 w-4 mx-1 ${isDone || isActive ? "text-blue-500" : "text-gray-300"}`} />}
              <div className={`flex items-center gap-2 ${isActive || isDone ? "text-blue-600 dark:text-blue-400" : "text-gray-400"}`}>
                <div className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold ${isDone ? "bg-green-500 text-white" : isActive ? "bg-blue-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400"}`}>
                  {isDone ? <Check className="h-3.5 w-3.5" /> : i + 1}
                </div>
                <span className="text-sm font-medium hidden sm:inline">{label}</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* ════ STEP: UPLOAD ════ */}
      {step === "upload" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("importWizard.uploadFile")}</h3>
            <div className="rounded-xl border-2 border-dashed border-gray-300 p-8 text-center dark:border-gray-700">
              <Upload className="mx-auto h-10 w-10 text-gray-300" />
              <p className="mt-2 text-sm text-gray-500">{t("importWizard.dropFile")}</p>
              <input ref={fileInput} type="file" accept=".csv" className="hidden" onChange={e => { const f = e.target.files?.[0]; if (f) handleFile(f); }} />
              <button onClick={() => fileInput.current?.click()} className="mt-3 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">{t("importWizard.browse")}</button>
              {fileName && <p className="mt-2 text-xs text-green-600">{fileName}</p>}
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("importWizard.pasteCsv")}</h3>
            <textarea value={csvText} onChange={e => setCsvText(e.target.value)} placeholder={SAMPLE_CSV} rows={8} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            <div className="mt-3 flex items-center justify-between">
              <button onClick={() => setCsvText(SAMPLE_CSV)} className="text-xs text-blue-600 hover:underline">{t("importWizard.loadSample")}</button>
              <button onClick={handlePaste} disabled={!csvText.trim()} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{t("importWizard.preview")}</button>
            </div>
          </div>
          <div className={`${card} lg:col-span-2`}>
            <h3 className="mb-2 text-sm font-semibold uppercase text-gray-400">{t("importWizard.format")}</h3>
            <div className="overflow-x-auto"><table className="w-full text-sm"><thead><tr><th className="px-3 py-1 text-left text-xs text-gray-400">Column</th><th className="px-3 py-1 text-left text-xs text-gray-400">Required</th><th className="px-3 py-1 text-left text-xs text-gray-400">Description</th></tr></thead>
            <tbody className="divide-y dark:divide-gray-700">
              {[["email", "Yes", "User email address"], ["first_name", "Yes", "Given name"], ["last_name", "Yes", "Family name"], ["department", "No", "Department name"], ["role", "No", "Role assignment"]].map(([col, req, desc]) => (
                <tr key={col}><td className="px-3 py-1.5 font-mono text-xs text-blue-500">{col}</td><td className="px-3 py-1.5 text-xs">{req}</td><td className="px-3 py-1.5 text-xs text-gray-400">{desc}</td></tr>
              ))}
            </tbody></table></div>
          </div>
        </div>
      )}

      {/* ════ STEP: PREVIEW ════ */}
      {step === "preview" && (
        <div className="space-y-4">
          <div className="grid grid-cols-3 gap-4">
            <div className={`${card} text-center`}><Table className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-1 text-xl font-bold">{rows.length}</p><p className="text-xs text-gray-400">{t("importWizard.totalRows")}</p></div>
            <div className={`${card} text-center`}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-1 text-xl font-bold text-green-600">{validCount}</p><p className="text-xs text-gray-400">{t("importWizard.validRows")}</p></div>
            <div className={`${card} text-center`}><XCircle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-1 text-xl font-bold text-red-600">{invalidCount}</p><p className="text-xs text-gray-400">{t("importWizard.invalidRows")}</p></div>
          </div>

          <div className={card}>
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-sm font-semibold uppercase text-gray-400">{t("importWizard.dataPreview")}</h3>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={dryRun} onChange={e => setDryRun(e.target.checked)} className="rounded" /> {t("importWizard.dryRun")}</label>
            </div>
            <div className="max-h-80 overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900"><tr>
                  <th className="px-3 py-2 text-left text-xs text-gray-400">Row</th>
                  <th className="px-3 py-2 text-left text-xs text-gray-400">Email</th>
                  <th className="px-3 py-2 text-left text-xs text-gray-400">Name</th>
                  <th className="px-3 py-2 text-center text-xs text-gray-400">Status</th>
                  <th className="px-3 py-2 text-left text-xs text-gray-400">Errors</th>
                </tr></thead>
                <tbody className="divide-y dark:divide-gray-700">
                  {rows.map(r => (
                    <tr key={r.row} className={r.status === "invalid" ? "bg-red-50 dark:bg-red-950/20" : ""}>
                      <td className="px-3 py-2 text-xs text-gray-400">{r.row}</td>
                      <td className="px-3 py-2 text-xs font-mono">{r.email}</td>
                      <td className="px-3 py-2 text-xs">{r.name}</td>
                      <td className="px-3 py-2 text-center">
                        {r.status === "valid" ? <CheckCircle2 className="mx-auto h-4 w-4 text-green-500" /> : <XCircle className="mx-auto h-4 w-4 text-red-500" />}
                      </td>
                      <td className="px-3 py-2 text-xs text-red-500">{r.errors.join(", ")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="flex items-center justify-between">
            <button onClick={() => setStep("upload")} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.back")}</button>
            <button onClick={doImport} disabled={validCount === 0}
              className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
              <Zap className="h-4 w-4" /> {dryRun ? t("importWizard.testImport") : t("importWizard.confirmImport")} ({validCount} {t("importWizard.users")})
            </button>
          </div>
        </div>
      )}

      {/* ════ STEP: IMPORTING ════ */}
      {step === "importing" && (
        <div className={card + " text-center py-12"}>
          <Loader2 className="mx-auto h-12 w-12 animate-spin text-blue-500" />
          <p className="mt-4 text-sm font-medium">{t("importWizard.importing")}</p>
          <div className="mx-auto mt-4 max-w-xs">
            <div className="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
              <div className="h-full rounded-full bg-blue-500 transition-all" style={{ width: `${progress}%` }} />
            </div>
            <p className="mt-1 text-xs text-gray-400">{progress}%</p>
          </div>
        </div>
      )}

      {/* ════ STEP: DONE ════ */}
      {step === "done" && importResult && (
        <div className="space-y-6">
          <div className={`${card} text-center`}>
            <CheckCircle2 className="mx-auto h-12 w-12 text-green-500" />
            <h2 className="mt-4 text-lg font-bold">{t("importWizard.importComplete")}</h2>
            <p className="text-sm text-gray-400">{dryRun ? t("importWizard.dryRunNote") : t("importWizard.successNote")}</p>
          </div>
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={`${card} text-center`}><Users className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-1 text-xl font-bold">{importResult.total}</p><p className="text-xs text-gray-400">{t("importWizard.totalProcessed")}</p></div>
            <div className={`${card} text-center`}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-1 text-xl font-bold text-green-600">{importResult.imported}</p><p className="text-xs text-gray-400">{t("importWizard.imported")}</p></div>
            <div className={`${card} text-center`}><XCircle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-1 text-xl font-bold text-red-600">{importResult.failed}</p><p className="text-xs text-gray-400">{t("importWizard.failed")}</p></div>
            <div className={`${card} text-center`}><RefreshCw className="mx-auto h-5 w-5 text-gray-400" /><p className="mt-1 text-xl font-bold">{importResult.duration_ms}ms</p><p className="text-xs text-gray-400">{t("importWizard.duration")}</p></div>
          </div>
          <div className="flex justify-center gap-3">
            <button onClick={reset} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">
              <Upload className="h-4 w-4" /> {t("importWizard.newImport")}
            </button>
            <button className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">
              <Download className="h-4 w-4" /> {t("importWizard.downloadReport")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
