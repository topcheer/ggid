"use client";

import { useState, useCallback, useRef } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Upload, Loader2, AlertCircle, X, Check, FileText, ChevronRight,
  CheckCircle2, XCircle, ArrowLeft, ArrowRight, Download,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const TENANT_ID = typeof window !== "undefined" ? localStorage.getItem("ggid_tenant_id") || "00000000-0000-0000-0000-000000000001" : "00000000-0000-0000-0000-000000000001";

interface ImportRow {
  row: number;
  email: string;
  name: string;
  status: "valid" | "invalid" | "warning";
  errors: string[];
}

interface ImportResult {
  total: number;
  valid: number;
  invalid: number;
  warnings: number;
  imported: number;
  failed: number;
  duration_ms: number;
}

type Step = "upload" | "map" | "preview" | "importing" | "done";

const TARGET_FIELDS = [
  { key: "email", label: "Email", required: true },
  { key: "first_name", label: "First Name", required: false },
  { key: "last_name", label: "Last Name", required: false },
  { key: "display_name", label: "Display Name", required: false },
  { key: "department", label: "Department", required: false },
  { key: "role", label: "Role", required: false },
  { key: "phone", label: "Phone", required: false },
];

const SAMPLE_CSV = `email,first_name,last_name,department,role
alice@company.com,Alice,Chen,Engineering,engineer
bob@company.com,Bob,Smith,Sales,rep
carol@company.com,Carol,Wong,Marketing,manager`;

const SAMPLE_JSON = JSON.stringify([
  { email: "alice@company.com", first_name: "Alice", last_name: "Chen", department: "Engineering", role: "engineer" },
  { email: "bob@company.com", first_name: "Bob", last_name: "Smith", department: "Sales", role: "rep" },
], null, 2);

export default function ImportWizardPage() {
  const t = useTranslations();
  const [step, setStep] = useState<Step>("upload");
  const [fileName, setFileName] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [fileType, setFileType] = useState<"json" | "csv">("csv");
  const [sourceFields, setSourceFields] = useState<string[]>([]);
  const [mapping, setMapping] = useState<Record<string, string>>({});
  const [rows, setRows] = useState<ImportRow[]>([]);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragOver, setDragOver] = useState(false);

  // Parse uploaded file
  const parseFile = useCallback((content: string, type: "json" | "csv") => {
    setError("");
    try {
      if (type === "json") {
        const data = JSON.parse(content);
        if (!Array.isArray(data) || data.length === 0) {
          setError("Invalid JSON: expected non-empty array");
          return;
        }
        const fields = Object.keys(data[0]);
        setSourceFields(fields);
        // Auto-map by name similarity
        const autoMap: Record<string, string> = {};
        fields.forEach((f) => {
          const match = TARGET_FIELDS.find((tf) =>
            tf.key === f || tf.key.replace(/_/g, "") === f.replace(/_/g, "").toLowerCase()
          );
          autoMap[f] = match ? match.key : "skip";
        });
        setMapping(autoMap);
        setRows(data.slice(0, 50).map((item: Record<string, unknown>, i: number) => ({
          row: i + 1,
          email: String(item.email || ""),
          name: [item.first_name, item.last_name].filter(Boolean).join(" ") || String(item.display_name || ""),
          status: item.email ? "valid" : "invalid",
          errors: item.email ? [] : ["Missing email"],
        })));
      } else {
        // CSV parse
        const lines = content.trim().split("\n");
        if (lines.length < 2) {
          setError("Invalid CSV: need header + at least 1 row");
          return;
        }
        const headers = lines[0].split(",").map((h) => h.trim().replace(/^["']|["']$/g, ""));
        setSourceFields(headers);
        const autoMap: Record<string, string> = {};
        headers.forEach((h) => {
          const match = TARGET_FIELDS.find((tf) =>
            tf.key === h || tf.key.replace(/_/g, "") === h.replace(/_/g, "").toLowerCase()
          );
          autoMap[h] = match ? match.key : "skip";
        });
        setMapping(autoMap);
        const parsedRows: ImportRow[] = lines.slice(1, 51).map((line: any, i: number) => {
          const vals = line.split(",").map((v) => v.trim().replace(/^["']|["']$/g, ""));
          const email = vals[headers.indexOf("email")] || "";
          return {
            row: i + 1,
            email,
            name: [vals[headers.indexOf("first_name")], vals[headers.indexOf("last_name")]].filter(Boolean).join(" "),
            status: email ? "valid" : "invalid",
            errors: email ? [] : ["Missing email"],
          };
        });
        setRows(parsedRows);
      }
    } catch {
      setError(`Failed to parse ${type.toUpperCase()} file`);
    }
  }, []);

  const handleFile = (file: File) => {
    const type = file.name.endsWith(".json") ? "json" : "csv";
    setFileType(type);
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      setFileContent(content);
      parseFile(content, type);
    };
    reader.readAsText(file);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files[0];
    if (file) handleFile(file);
  };

  const handleImport = async () => {
    setStep("importing");
    try {
      const res = await fetch(`${API_BASE}/api/v1/users/bulk-import`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({
          tenant_id: TENANT_ID,
          format: fileType,
          data: fileContent,
          mapping,
        }),
      });
      const data = await res.json();
      setResult({
        total: data.total || rows.length,
        valid: data.valid || rows.filter((r) => r.status === "valid").length,
        invalid: data.invalid || rows.filter((r) => r.status === "invalid").length,
        warnings: 0,
        imported: data.imported || data.valid || rows.filter((r) => r.status === "valid").length,
        failed: data.failed || 0,
        duration_ms: data.duration_ms || 0,
      });
      setStep("done");
    } catch {
      setError("Import failed");
      setStep("preview");
    }
  };

  const downloadSample = (type: "csv" | "json") => {
    const blob = new Blob([type === "csv" ? SAMPLE_CSV : SAMPLE_JSON], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `sample-users.${type}`;
    a.click();
  };

  const reset = () => {
    setStep("upload");
    setFileName("");
    setFileContent("");
    setSourceFields([]);
    setMapping({});
    setRows([]);
    setResult(null);
    setError("");
  };

  const steps = [
    { id: "upload", label: t("importWizard.steps.upload"), num: 1 },
    { id: "map", label: t("importWizard.steps.map"), num: 2 },
    { id: "preview", label: t("importWizard.steps.preview"), num: 3 },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-2">
            <Upload className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {t("importWizard.title")}
            </h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">
            {t("importWizard.description")}
          </p>
        </div>

        {/* Stepper */}
        <div className="flex items-center gap-2 mb-6">
          {steps.map((s: any, i: number) => {
            const isActive = step === s.id || (step === "importing" && s.id === "preview") || (step === "done" && s.id === "preview");
            const isPast = steps.findIndex((x) => x.id === step) > i || step === "done";
            return (
              <div key={s.id} className="flex items-center gap-2">
                {i > 0 && <ChevronRight className="w-4 h-4 text-gray-400" />}
                <div className={`flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium ${
                  isActive ? "bg-blue-600 text-white" : isPast ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-gray-200 dark:bg-gray-800 text-gray-500"
                }`}>
                  <span className="w-5 h-5 rounded-full flex items-center justify-center text-xs">
                    {isPast ? <Check className="w-3 h-3" /> : s.num}
                  </span>
                  {s.label}
                </div>
              </div>
            );
          })}
        </div>

        {error && (
          <div className="flex items-center gap-2 px-4 py-3 mb-4 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-700 dark:text-red-300 text-sm">
            <AlertCircle className="w-4 h-4" />
            {error}
          </div>
        )}

        {/* Step Content */}
        {step === "upload" && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-2">
              {t("importWizard.upload.title")}
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">
              {t("importWizard.upload.description")}
            </p>

            {/* Drop Zone */}
            <div
              onDrop={handleDrop}
              onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
              onDragLeave={() => setDragOver(false)}
              onClick={() => fileInputRef.current?.click()}
              className={`border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-colors ${
                dragOver ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-300 dark:border-gray-700 hover:border-blue-400"
              }`}
            >
              <Upload className="w-12 h-12 mx-auto mb-3 text-gray-400" />
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-1">
                {t("importWizard.upload.dragDrop")}
              </p>
              <p className="text-xs text-gray-400">
                {t("importWizard.upload.supportedFormats")} | {t("importWizard.upload.maxSize")}
              </p>
              <input
                ref={fileInputRef}
                type="file"
                accept=".json,.csv"
                onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); }}
                className="hidden"
              />
            </div>

            {fileName && (
              <div className="mt-4 flex items-center gap-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                <FileText className="w-5 h-5 text-blue-600" />
                <span className="text-sm text-gray-900 dark:text-white flex-1">{fileName}</span>
                <Check className="w-4 h-4 text-green-500" />
                <span className="text-xs text-gray-500">{rows.length} rows parsed</span>
              </div>
            )}

            {/* Sample Downloads */}
            <div className="mt-4 flex items-center gap-2">
              <span className="text-xs text-gray-500">Sample files:</span>
              <button onClick={() => downloadSample("csv")} className="text-xs text-blue-600 hover:underline flex items-center gap-1">
                <Download className="w-3 h-3" /> sample-users.csv
              </button>
              <button onClick={() => downloadSample("json")} className="text-xs text-blue-600 hover:underline flex items-center gap-1">
                <Download className="w-3 h-3" /> sample-users.json
              </button>
            </div>

            {fileName && (
              <button
                onClick={() => setStep("map")}
                className="mt-4 flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium text-sm"
              >
                {t("importWizard.steps.map")}
                <ArrowRight className="w-4 h-4" />
              </button>
            )}
          </div>
        )}

        {step === "map" && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-2">
              {t("importWizard.mapping.title")}
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">
              {t("importWizard.mapping.description")}
            </p>

            <div className="space-y-2">
              {sourceFields.map((field) => (
                <div key={field} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800">
                  <div className="flex-1">
                    <span className="text-sm font-medium text-gray-900 dark:text-white">{field}</span>
                  </div>
                  <ChevronRight className="w-4 h-4 text-gray-400" />
                  <select
                    value={mapping[field] || "skip"}
                    onChange={(e) => setMapping({ ...mapping, [field]: e.target.value })}
                    className="flex-1 px-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white"
                  >
                    <option value="skip">{t("importWizard.mapping.skip")}</option>
                    {TARGET_FIELDS.map((tf) => (
                      <option key={tf.key} value={tf.key}>
                        {t(`importWizard.mapping.${tf.key === "first_name" ? "firstName" : tf.key === "last_name" ? "lastName" : tf.key === "display_name" ? "displayName" : tf.key}`)}
                      </option>
                    ))}
                  </select>
                </div>
              ))}
            </div>

            <div className="flex gap-2 mt-4">
              <button
                onClick={() => setStep("upload")}
                className="flex items-center gap-1.5 px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium"
              >
                <ArrowLeft className="w-4 h-4" />
                {t("importWizard.preview.back")}
              </button>
              <button
                onClick={() => setStep("preview")}
                className="flex items-center gap-1.5 px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium"
              >
                {t("importWizard.steps.preview")}
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}

        {(step === "preview" || step === "importing") && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-2">
              {t("importWizard.preview.title")}
            </h3>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">
              {t("importWizard.preview.description")}
            </p>

            {/* Stats */}
            <div className="grid grid-cols-3 gap-3 mb-4">
              <StatCard label={t("importWizard.preview.totalRows")} value={rows.length} color="blue" />
              <StatCard label={t("importWizard.preview.validRows")} value={rows.filter((r) => r.status === "valid").length} color="green" />
              <StatCard label={t("importWizard.preview.invalidRows")} value={rows.filter((r) => r.status === "invalid").length} color="red" />
            </div>

            {/* Preview Table */}
            <div className="overflow-x-auto max-h-64 overflow-y-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-gray-50 dark:bg-gray-800">
                  <tr className="text-left">
                    <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("importWizard.preview.row")}</th>
                    <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">Email</th>
                    <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">Name</th>
                    <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("importWizard.preview.status")}</th>
                    <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("importWizard.preview.errors")}</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.row} className="border-b border-gray-100 dark:border-gray-800/50">
                      <td className="py-2 px-3 text-gray-500">{r.row}</td>
                      <td className="py-2 px-3 text-gray-900 dark:text-white">{r.email || "—"}</td>
                      <td className="py-2 px-3 text-gray-900 dark:text-white">{r.name || "—"}</td>
                      <td className="py-2 px-3">
                        {r.status === "valid" && <CheckCircle2 className="w-4 h-4 text-green-500" />}
                        {r.status === "invalid" && <XCircle className="w-4 h-4 text-red-500" />}
                        {r.status === "warning" && <AlertCircle className="w-4 h-4 text-yellow-500" />}
                      </td>
                      <td className="py-2 px-3 text-xs text-red-500">
                        {r.errors.join(", ") || t("importWizard.preview.noErrors")}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="flex gap-2 mt-4">
              <button
                onClick={() => setStep("map")}
                className="flex items-center gap-1.5 px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium"
              >
                <ArrowLeft className="w-4 h-4" />
                {t("importWizard.preview.back")}
              </button>
              <button
                onClick={handleImport}
                disabled={step === "importing"}
                className="flex items-center gap-2 px-6 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium"
              >
                {step === "importing" ? <Loader2 className="w-4 h-4 animate-spin" /> : <Upload className="w-4 h-4" />}
                {step === "importing" ? t("importWizard.preview.importing") : t("importWizard.preview.confirmImport")}
              </button>
            </div>
          </div>
        )}

        {step === "done" && result && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-8 text-center">
            <CheckCircle2 className="w-16 h-16 mx-auto mb-4 text-green-500" />
            <h3 className="text-lg font-bold text-gray-900 dark:text-white mb-2">
              {t("importWizard.preview.importSuccess")}
            </h3>

            <div className="grid grid-cols-4 gap-3 mt-6 max-w-md mx-auto">
              <StatCard label={t("importWizard.preview.totalRows")} value={result.total} color="blue" />
              <StatCard label={t("importWizard.preview.imported")} value={result.imported} color="green" />
              <StatCard label={t("importWizard.preview.failed")} value={result.failed} color="red" />
              <StatCard label="Time" value={`${result.duration_ms}ms`} color="gray" />
            </div>

            <button
              onClick={reset}
              className="mt-6 flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium mx-auto"
            >
              <Upload className="w-4 h-4" />
              {t("importWizard.title")}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({ label, value, color }: { label: string; value: number | string; color: "blue" | "green" | "red" | "gray" }) {
  const colors = {
    blue: "bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-300",
    green: "bg-green-50 text-green-700 dark:bg-green-950/30 dark:text-green-300",
    red: "bg-red-50 text-red-700 dark:bg-red-950/30 dark:text-red-300",
    gray: "bg-gray-50 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
  };
  return (
    <div className={`rounded-lg p-3 text-center ${colors[color]}`}>
      <div className="text-2xl font-bold">{value}</div>
      <div className="text-xs mt-0.5 opacity-80">{label}</div>
    </div>
  );
}
