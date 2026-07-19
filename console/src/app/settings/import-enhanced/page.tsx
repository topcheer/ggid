"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import { EmptyState } from "@/components/EmptyState";
import {
  Upload, Loader2, AlertCircle, Check, X, FileText,
  Play, Download, RefreshCw, ChevronRight, CheckCircle2,
  XCircle, AlertTriangle, BarChart3, Clock,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

type Phase = "upload" | "precheck" | "precheck-result" | "importing" | "summary";

interface PreCheckResult {
  total: number; valid: number; invalid: number; skipped: number;
  duplicates: number; warnings: number;
  errors: { row: number; email: string; error: string }[];
}

interface ImportSummary {
  imported: number; failed: number; skipped: number;
  duration_ms: number; rate: number;
  errors: { type: string; count: number }[];
}

export default function ImportEnhancedPage() {
  const t = useTranslations();
  const [phase, setPhase] = useState<Phase>("upload");
  const [fileName, setFileName] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [preCheck, setPreCheck] = useState<PreCheckResult | null>(null);
  const [progress, setProgress] = useState(0);
  const [progressTotal, setProgressTotal] = useState(0);
  const [summary, setSummary] = useState<ImportSummary | null>(null);
  const [error, setError] = useState("");
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const handleFile = (file: File) => {
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = (e) => setFileContent(e.target?.result as string);
    reader.readAsText(file);
  };

  const runPreCheck = async () => {
    setPhase("precheck");
    setPreCheck(null);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/users/import-async/create`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ dry_run: true, data: fileContent }),
      });
      if (res.ok) { const d = await res.json(); setPreCheck(d); setPhase("precheck-result"); return; }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Pre-check failed");
    }
  };

  const startImport = async () => {
    setPhase("importing");
    setProgress(0);
    setProgressTotal(preCheck?.total || 0);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/users/import-async/create`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ dry_run: false, data: fileContent }),
      });
      const data = await res.json();
      if (data.job_id) { pollJob(data.job_id); return; }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Import failed");
      setPhase("precheck-result");
    }
  };

  const pollJob = (jobId: string) => {
    pollRef.current = setInterval(async () => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/identity/users/import-async/${jobId}`, { headers: { ...authHeader() } });
        if (res.ok) {
          const d = await res.json();
          setProgress(d.imported + d.failed);
          setProgressTotal(d.total);
          if (d.status === "completed" || d.status === "failed") {
            if (pollRef.current) clearInterval(pollRef.current);
            setSummary({
              imported: d.imported, failed: d.failed, skipped: d.skipped || 0,
              duration_ms: d.duration_ms || 0,
              rate: d.duration_ms > 0 ? Math.round((d.imported + d.failed) / (d.duration_ms / 1000)) : 0,
              errors: d.errors || [],
            });
            setPhase("summary");
          }
        }
      } catch { /* poll will retry */ }
    }, 1000);
  };

  useEffect(() => () => { if (pollRef.current) clearInterval(pollRef.current); }, []);

  const reset = () => {
    setPhase("upload"); setFileName(""); setFileContent(""); setPreCheck(null); setProgress(0); setSummary(null); setError("");
  };

  const downloadReport = () => {
    if (!summary) return;
    const report = `Import Report\n${"=".repeat(40)}\nDate: ${new Date().toISOString()}\nFile: ${fileName}\n\nTotal: ${progressTotal}\nImported: ${summary.imported}\nFailed: ${summary.failed}\nSkipped: ${summary.skipped}\nDuration: ${(summary.duration_ms / 1000).toFixed(1)}s\nRate: ${summary.rate} rows/sec\n\nError Breakdown:\n${summary.errors.map((e: any) => `  ${e.type}: ${e.count}`).join("\n")}\n`;
    const blob = new Blob([report], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = `import-report-${Date.now()}.txt`; a.click();
  };

  const steps = [
    { id: "upload", label: t("importWizard.steps.upload") },
    { id: "precheck", label: t("importEnhanced.preCheck.title") },
    { id: "importing", label: t("importEnhanced.progress.title") },
    { id: "summary", label: t("importEnhanced.summary.title") },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-3xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Upload className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">User Import</h1>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">Pre-check, import with live progress, and get a detailed summary report</p>
        </div>

        {/* Stepper */}
        <div className="flex items-center gap-2 mb-6">
          {steps.map((s: any, i: number) => {
            const stepIdx = ["upload", "precheck", "precheck-result", "importing", "summary"].indexOf(phase);
            const isActive = (phase === "precheck-result" && s.id === "precheck") || phase === s.id;
            const isPast = stepIdx > ["upload", "precheck", "importing", "summary"].indexOf(s.id);
            return (
              <div key={s.id} className="flex items-center gap-2">
                {i > 0 && <ChevronRight className="w-3 h-3 text-gray-300" />}
                <div className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium ${isActive ? "bg-blue-600 text-white" : isPast ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
                  {isPast ? <Check className="w-3 h-3" /> : <span>{i + 1}</span>}{s.label}
                </div>
              </div>
            );
          })}
        </div>

        {error && <div className="flex items-center gap-2 px-4 py-3 mb-4 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

        {/* Phase: Upload */}
        {phase === "upload" && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
            <div
              onDrop={(e) => { e.preventDefault(); const f = e.dataTransfer.files[0]; if (f) handleFile(f); }}
              onDragOver={(e) => e.preventDefault()}
              onClick={() => document.getElementById("file-input")?.click()}
              className="border-2 border-dashed rounded-xl p-10 text-center cursor-pointer border-gray-300 dark:border-gray-700 hover:border-blue-400"
            >
              <Upload className="w-10 h-10 mx-auto mb-2 text-gray-400" />
              <p className="text-sm text-gray-600 dark:text-gray-400">{t("importWizard.upload.dragDrop")}</p>
              <input id="file-input" type="file" accept=".json,.csv" onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }} className="hidden" />
            </div>
            {fileName && (
              <div className="mt-3 flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <FileText className="w-5 h-5 text-blue-600" />
                <span className="text-sm text-gray-900 dark:text-white flex-1">{fileName}</span>
                <Check className="w-4 h-4 text-green-500" />
              </div>
            )}
            <button onClick={runPreCheck} disabled={!fileName}
              className="mt-4 flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
              <Play className="w-4 h-4" />{t("importEnhanced.preCheck.runCheck")}
            </button>
          </div>
        )}

        {/* Phase: Pre-check running */}
        {phase === "precheck" && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12">
            <EmptyState icon={Loader2} title={t("importEnhanced.preCheck.checking")} />
          </div>
        )}

        {/* Phase: Pre-check results */}
        {phase === "precheck-result" && preCheck && (
          <div className="space-y-4">
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("importEnhanced.preCheck.results")}</h3>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-4">
                <StatCard label={t("importEnhanced.preCheck.total")} value={preCheck.total} color="text-blue-600" />
                <StatCard label={t("importEnhanced.preCheck.valid")} value={preCheck.valid} color="text-green-600" icon={CheckCircle2} />
                <StatCard label={t("importEnhanced.preCheck.invalid")} value={preCheck.invalid} color="text-red-500" icon={XCircle} />
                <StatCard label={t("importEnhanced.preCheck.skipped")} value={preCheck.skipped} color="text-gray-500" icon={AlertTriangle} />
                <StatCard label={t("importEnhanced.preCheck.duplicates")} value={preCheck.duplicates} color="text-orange-500" />
                <StatCard label={t("importEnhanced.preCheck.warnings")} value={preCheck.warnings} color="text-yellow-500" />
              </div>

              {preCheck.errors.length > 0 && (
                <div>
                  <h4 className="text-xs font-medium text-gray-500 mb-2">{t("importEnhanced.preCheck.issues")}:</h4>
                  <div className="max-h-40 overflow-y-auto rounded-lg border border-gray-200 dark:border-gray-800">
                    {preCheck.errors.map((e: any, i: number) => (
                      <div key={i} className="flex items-center gap-3 px-3 py-2 border-b border-gray-100 dark:border-gray-800/50 text-xs">
                        <span className="text-gray-400">Row {e.row}</span>
                        <span className="font-mono text-gray-900 dark:text-white">{e.email || "—"}</span>
                        <span className="text-red-500 ml-auto">{e.error}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="flex gap-2">
              <button onClick={() => setPhase("upload")} className="px-4 py-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg text-sm font-medium">
                {t("importEnhanced.preCheck.fixFirst")}
              </button>
              <button onClick={startImport} className="flex items-center gap-2 px-6 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg text-sm font-medium">
                <Play className="w-4 h-4" />{t("importEnhanced.preCheck.proceed")}
              </button>
            </div>
          </div>
        )}

        {/* Phase: Importing */}
        {phase === "importing" && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-8">
            <div className="text-center mb-6">
              <Loader2 className="w-12 h-12 mx-auto mb-3 text-blue-600 animate-spin" />
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("importEnhanced.progress.importing")}</h3>
              <p className="text-xs text-gray-500 mt-1">{t("importEnhanced.progress.processing", { current: progress, total: progressTotal })}</p>
            </div>
            <div className="max-w-md mx-auto">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs text-gray-500">{progress} / {progressTotal}</span>
                <span className="text-xs font-medium text-blue-600">{progressTotal > 0 ? Math.round((progress / progressTotal) * 100) : 0}%</span>
              </div>
              <div className="h-3 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                <div className="h-full bg-gradient-to-r from-blue-600 to-purple-600 rounded-full transition-all duration-300" style={{ width: `${progressTotal > 0 ? (progress / progressTotal) * 100 : 0}%` }} />
              </div>
            </div>
          </div>
        )}

        {/* Phase: Summary */}
        {phase === "summary" && summary && (
          <div className="space-y-4">
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
              <div className="text-center mb-6">
                <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-100 dark:bg-green-950/30 mb-3">
                  <CheckCircle2 className="w-8 h-8 text-green-500" />
                </div>
                <h3 className="text-lg font-bold text-gray-900 dark:text-white">{t("importEnhanced.progress.completed")}</h3>
              </div>

              {/* Summary stats */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
                <SummaryCard label={t("importEnhanced.summary.totalImported")} value={summary.imported} color="bg-green-50 text-green-700 dark:bg-green-950/30 dark:text-green-300" />
                <SummaryCard label={t("importEnhanced.summary.totalFailed")} value={summary.failed} color="bg-red-50 text-red-700 dark:bg-red-950/30 dark:text-red-300" />
                <SummaryCard label={t("importEnhanced.summary.totalSkipped")} value={summary.skipped} color="bg-gray-50 text-gray-700 dark:bg-gray-800 dark:text-gray-300" />
                <SummaryCard label={t("importEnhanced.summary.duration")} value={`${(summary.duration_ms / 1000).toFixed(1)}s`} color="bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-300" />
              </div>

              {/* Rate */}
              <div className="flex items-center justify-center gap-4 text-xs text-gray-500 mb-4">
                <span className="flex items-center gap-1"><Clock className="w-3 h-3" />{summary.rate} {t("importEnhanced.summary.rowsPerSecond")}</span>
              </div>

              {/* Error breakdown */}
              {summary.errors.length > 0 && (
                <div>
                  <h4 className="text-xs font-medium text-gray-500 mb-2 flex items-center gap-1"><BarChart3 className="w-3 h-3" />{t("importEnhanced.summary.errorBreakdown")}</h4>
                  <div className="space-y-1">
                    {summary.errors.map((e: any, i: number) => (
                      <div key={i} className="flex items-center justify-between px-3 py-1.5 rounded-lg bg-gray-50 dark:bg-gray-800/50 text-xs">
                        <span className="text-gray-700 dark:text-gray-300">{e.type}</span>
                        <span className="font-medium text-red-500">{e.count}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="flex gap-2">
              <button onClick={downloadReport} className="flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg text-sm font-medium">
                <Download className="w-4 h-4" />{t("importEnhanced.summary.downloadReport")}
              </button>
              <button onClick={reset} className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium">
                <RefreshCw className="w-4 h-4" />{t("importEnhanced.summary.newImport")}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============ Shared ============

function StatCard({ label, value, color, icon: Icon }: { label: string; value: number; color: string; icon?: typeof CheckCircle2 }) {
  return (
    <div className="p-3 rounded-lg bg-gray-50 dark:bg-gray-800/50">
      <div className="flex items-center gap-1 mb-1">{Icon && <Icon className={`w-3.5 h-3.5 ${color}`} />}<span className="text-xs text-gray-500">{label}</span></div>
      <div className={`text-xl font-bold ${color}`}>{value}</div>
    </div>
  );
}

function SummaryCard({ label, value, color }: { label: string; value: string | number; color: string }) {
  return (
    <div className={`rounded-lg p-3 text-center ${color}`}>
      <div className="text-2xl font-bold">{value}</div>
      <div className="text-xs mt-0.5 opacity-80">{label}</div>
    </div>
  );
}
