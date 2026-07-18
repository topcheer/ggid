"use client";

import { useState, useRef, useCallback, useMemo } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import Link from "next/link";
import {
  Upload,
  FileText,
  Check,
  X,
  ChevronRight,
  ChevronLeft,
  AlertCircle,
  Loader2,
  Download,
  ArrowRight,
  Database,
  Eye,
  PlayCircle,
  CheckCircle2,
  XCircle,
} from "lucide-react";

// ── Types ──────────────────────────────────────────────

interface GgidField {
  key: string;
  label: string;
  required: boolean;
}

const GGID_FIELDS: GgidField[] = [
  { key: "email", label: "Email", required: true },
  { key: "display_name", label: "Display Name", required: true },
  { key: "first_name", label: "First Name", required: false },
  { key: "last_name", label: "Last Name", required: false },
  { key: "phone", label: "Phone", required: false },
  { key: "department", label: "Department", required: false },
  { key: "title", label: "Title", required: false },
  { key: "role", label: "Role", required: false },
  { key: "external_id", label: "External ID", required: false },
];

type ColumnMap = Record<string, string>; // csvHeader -> ggidFieldKey

interface RowError {
  row: number;
  field: string;
  message: string;
}

interface ValidationResult {
  valid: number;
  errors: number;
  duplicates: number;
  errorList: RowError[];
}

type ImportStatus = "idle" | "importing" | "done";

interface ImportStats {
  created: number;
  skipped: number;
  failed: number;
}

// ── CSV Parser ─────────────────────────────────────────

function parseCsvLine(line: string): string[] {
  const result: string[] = [];
  let current = "";
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (ch === '"' && line[i + 1] === '"') {
      current += '"';
      i++;
    } else if (ch === '"') {
      inQuotes = !inQuotes;
    } else if (ch === "," && !inQuotes) {
      result.push(current.trim());
      current = "";
    } else {
      current += ch;
    }
  }
  result.push(current.trim());
  return result;
}

function parseCsvText(text: string): { headers: string[]; rows: string[][] } {
  const lines = text.trim().split(/\r?\n/).filter((l: any) => l.trim());
  if (lines.length === 0) return { headers: [], rows: [] };
  const headers = parseCsvLine(lines[0]);
  const rows = lines.slice(1).map(parseCsvLine);
  return { headers, rows };
}

// Auto-detect mapping based on header name
function autoDetectMapping(headers: string[]): ColumnMap {
  const mapping: ColumnMap = {};
  const aliases: Record<string, string[]> = {
    email: ["email", "e-mail", "mail", "email_address"],
    display_name: ["display_name", "displayname", "name", "full_name", "fullname"],
    first_name: ["first_name", "firstname", "given_name", "fname"],
    last_name: ["last_name", "lastname", "family_name", "lname"],
    phone: ["phone", "phone_number", "tel", "telephone", "mobile"],
    department: ["department", "dept", "division"],
    title: ["title", "job_title", "position"],
    role: ["role", "role_name", "group"],
    external_id: ["external_id", "ext_id", "employee_id", "emp_id"],
  };
  headers.forEach((header: any) => {
    const lower = header.toLowerCase().trim();
    for (const [field, names] of Object.entries(aliases)) {
      if (names.includes(lower)) {
        mapping[header] = field;
        break;
      }
    }
  });
  return mapping;
}

// ── Validation ─────────────────────────────────────────

function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function validateRow(
  row: string[],
  headers: string[],
  mapping: ColumnMap,
  rowIndex: number,
  seenEmails: Set<string>,
): RowError[] {
  const errors: RowError[] = [];
  const mapped: Record<string, string> = {};
  headers.forEach((h: any, i: any) => {
    const field = mapping[h];
    if (field) mapped[field] = row[i] || "";
  });

  // Check required fields
  GGID_FIELDS.forEach((f: any) => {
    if (f.required) {
      const val = mapped[f.key];
      if (!val || val.trim() === "") {
        errors.push({ row: rowIndex, field: f.key, message: `Missing required field: ${f.label}` });
      }
    }
  });

  // Validate email format
  if (mapped.email && !isValidEmail(mapped.email)) {
    errors.push({ row: rowIndex, field: "email", message: `Invalid email: ${mapped.email}` });
  }

  // Check duplicates
  if (mapped.email && isValidEmail(mapped.email)) {
    if (seenEmails.has(mapped.email.toLowerCase())) {
      errors.push({ row: rowIndex, field: "email", message: `Duplicate email: ${mapped.email}` });
    } else {
      seenEmails.add(mapped.email.toLowerCase());
    }
  }

  return errors;
}

function getRowFieldErrors(
  row: string[],
  headerIndex: number,
  header: string,
  mapping: ColumnMap,
  rowIndex: number,
  allErrors: RowError[],
): boolean {
  const field = mapping[header];
  if (!field) return false;
  return allErrors.some((e: any) => e.row === rowIndex && e.field === field);
}

// ── Component ──────────────────────────────────────────

const STEPS = ["Upload", "Field Mapping", "Preview", "Dry Run", "Import"];

export default function UserImportPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Step navigation
  const [step, setStep] = useState(0);

  // Upload state
  const [fileName, setFileName] = useState<string | null>(null);
  const [fileSize, setFileSize] = useState<number>(0);
  const [csvText, setCsvText] = useState("");
  const [pasteText, setPasteText] = useState("");
  const [dragOver, setDragOver] = useState(false);

  // Parsed data
  const [headers, setHeaders] = useState<string[]>([]);
  const [rows, setRows] = useState<string[][]>([]);
  const [mapping, setMapping] = useState<ColumnMap>({});

  // Validation state
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);

  // Import state
  const [importStatus, setImportStatus] = useState<ImportStatus>("idle");
  const [importProgress, setImportProgress] = useState(0);
  const [importStats, setImportStats] = useState<ImportStats>({ created: 0, skipped: 0, failed: 0 });

  // ── Upload handlers ──

  const processFile = useCallback((file: File) => {
    setFileName(file.name);
    setFileSize(file.size);
    const reader = new FileReader();
    reader.onload = (e) => {
      const text = e.target?.result as string;
      setCsvText(text);
      const parsed = parseCsvText(text);
      setHeaders(parsed.headers);
      setRows(parsed.rows);
      setMapping(autoDetectMapping(parsed.headers));
    };
    reader.readAsText(file);
  }, []);

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) processFile(file);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file && file.name.endsWith(".csv")) processFile(file);
  };

  const handlePasteSubmit = () => {
    const text = pasteText.trim();
    if (!text) return;
    setCsvText(text);
    setFileName("pasted-data.csv");
    setFileSize(new Blob([text]).size);
    const parsed = parseCsvText(text);
    setHeaders(parsed.headers);
    setRows(parsed.rows);
    setMapping(autoDetectMapping(parsed.headers));
    setStep(1);
  };

  // ── Mapping ──

  const handleMappingChange = (csvHeader: string, fieldKey: string) => {
    setMapping((prev) => {
      const next = { ...prev };
      if (fieldKey === "") {
        delete next[csvHeader];
      } else {
        // Remove this field from any other column
        Object.keys(next).forEach((k: any) => {
          if (next[k] === fieldKey) delete next[k];
        });
        next[csvHeader] = fieldKey;
      }
      return next;
    });
  };

  const requiredFieldsMapped = useMemo(() => {
    const mappedValues = new Set(Object.values(mapping));
    return GGID_FIELDS.filter((f: any) => f.required).every((f: any) => mappedValues.has(f.key));
  }, [mapping]);

  // ── Validation ──

  const allRowErrors = useMemo<RowError[]>(() => {
    if (headers.length === 0 || rows.length === 0) return [];
    const seenEmails = new Set<string>();
    const errs: RowError[] = [];
    rows.forEach((row: any, i: any) => {
      errs.push(...validateRow(row, headers, mapping, i + 2, seenEmails)); // +2: header + 1-indexed
    });
    return errs;
  }, [headers, rows, mapping]);

  const runFullValidation = useCallback(async () => {
    setValidating(true);
    await new Promise((r) => setTimeout(r, 400)); // brief loading for UX
    const seenEmails = new Set<string>();
    const errorList: RowError[] = [];
    let valid = 0;
    let dups = 0;

    rows.forEach((row: any, i: any) => {
      const rowErrs = validateRow(row, headers, mapping, i + 2, seenEmails);
      if (rowErrs.length === 0) {
        valid++;
      } else {
        const hasDup = rowErrs.some((e: any) => e.message.includes("Duplicate"));
        if (hasDup) dups++;
        errorList.push(...rowErrs);
      }
    });

    setValidationResult({
      valid,
      errors: errorList.length,
      duplicates: dups,
      errorList,
    });
    setValidating(false);
  }, [rows, headers, mapping]);

  // ── Import ──

  const startImport = useCallback(async () => {
    setImportStatus("importing");
    setImportProgress(0);
    let created = 0;
    let skipped = 0;
    let failed = 0;

    for (let i = 0; i < rows.length; i++) {
      const row = rows[i];
      const payload: Record<string, string> = {};
      headers.forEach((h: any, idx: any) => {
        const field = mapping[h];
        if (field) payload[field] = row[idx] || "";
      });

      // Skip rows with required field errors
      const rowErrs = allRowErrors.filter((e: any) => e.row === i + 2);
      if (rowErrs.length > 0) {
        skipped++;
        setImportStats({ created, skipped, failed });
        setImportProgress(i + 1);
        continue;
      }

      try {
        await apiFetch("/api/v1/users", {
          method: "POST",
          body: JSON.stringify({
            username: payload.email || payload.display_name,
            email: payload.email,
            display_name: payload.display_name || `${payload.first_name || ""} ${payload.last_name || ""}`.trim() || undefined,
            phone: payload.phone || undefined,
          }),
        });
        created++;
      } catch {
        failed++;
      }
      setImportStats({ created, skipped, failed });
      setImportProgress(i + 1);
    }

    setImportStatus("done");
  }, [rows, headers, mapping, allRowErrors, apiFetch]);

  const reset = () => {
    setStep(0);
    setFileName(null);
    setFileSize(0);
    setCsvText("");
    setPasteText("");
    setHeaders([]);
    setRows([]);
    setMapping({});
    setValidationResult(null);
    setImportStatus("idle");
    setImportProgress(0);
    setImportStats({ created: 0, skipped: 0, failed: 0 });
  };

  const canProceed = (s: number): boolean => {
    if (s === 0) return headers.length > 0 && rows.length > 0;
    if (s === 1) return requiredFieldsMapped;
    return true;
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  // ── Render ──

  return (
    <div className="max-w-5xl">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <Link href="/users" className="hover:text-brand-600">Users</Link>
            <ChevronRight className="h-3 w-3" />
            <span className="text-gray-600 dark:text-gray-300">{t("userImport.import")}</span>
          </div>
          <h1 className="mt-1 flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Upload className="h-6 w-6 text-brand-600" />
            User Import Wizard
          </h1>
        </div>
        <button
          onClick={reset}
          className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
        >
          Reset
        </button>
      </div>

      {/* Progress Steps Bar */}
      <div className="mb-8 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <div className="flex items-center justify-between">
          {STEPS.map((label: any, i: any) => (
            <div key={label} className="flex flex-1 items-center">
              <button
                onClick={() => i <= step && setStep(i)}
                disabled={i > step}
                className={`flex items-center gap-2 ${i <= step ? "cursor-pointer" : "cursor-not-allowed"}`}
              >
                <div
                  className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold transition-colors ${
                    i < step
                      ? "bg-brand-600 text-white"
                      : i === step
                        ? "bg-brand-600 text-white ring-4 ring-brand-100 dark:ring-brand-900"
                        : "bg-gray-200 text-gray-400 dark:bg-gray-700"
                  }`}
                >
                  {i < step ? <Check className="h-4 w-4" /> : i + 1}
                </div>
                <span
                  className={`hidden text-sm font-medium sm:inline ${
                    i <= step ? "text-gray-700 dark:text-gray-300" : "text-gray-400"
                  }`}
                >
                  {label}
                </span>
              </button>
              {i < STEPS.length - 1 && (
                <div
                  className={`mx-2 h-0.5 flex-1 ${i < step ? "bg-brand-600" : "bg-gray-200 dark:bg-gray-700"}`}
                />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Step Content */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        {/* ── Step 0: Upload ── */}
        {step === 0 && (
          <div className="space-y-6">
            <div>
              <h2 className="mb-1 text-lg font-semibold text-gray-800 dark:text-gray-200">
                Upload CSV File
              </h2>
              <p className="text-sm text-gray-500">
                Drag and drop a CSV file, or paste CSV content below.
              </p>
            </div>

            {/* Drop zone */}
            <div
              onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              className={`flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed py-12 transition-colors ${
                dragOver
                  ? "border-brand-500 bg-brand-50 dark:bg-brand-950"
                  : "border-gray-300 hover:border-brand-400 dark:border-gray-600"
              }`}
            >
              <FileText className="mb-3 h-10 w-10 text-gray-400" />
              <p className="text-sm font-medium text-gray-600 dark:text-gray-300">
                Drop CSV file here or click to browse
              </p>
              <p className="mt-1 text-xs text-gray-400">{t("userImport.supportsCsv")}</p>
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv"
                onChange={handleFileInput}
                className="hidden"
              />
            </div>

            {/* File info */}
            {fileName && (
              <div className="flex items-center gap-3 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900">
                <FileText className="h-5 w-5 text-brand-600" />
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-700 dark:text-gray-300">{fileName}</p>
                  <p className="text-xs text-gray-400">{formatSize(fileSize)} • {rows.length} rows</p>
                </div>
                <button
                  onClick={(e) => { e.stopPropagation(); reset(); }}
                  className="text-gray-400 hover:text-red-500"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            )}

            {/* Divider */}
            <div className="flex items-center gap-3">
              <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
              <span className="text-xs text-gray-400">OR</span>
              <div className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
            </div>

            {/* Paste textarea */}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-600 dark:text-gray-300">
                Paste CSV Content
              </label>
              <textarea
                value={pasteText}
                onChange={(e) => setPasteText(e.target.value)}
                placeholder={"email,display_name,phone\njohn@example.com,John Doe,+1234567890"}
                rows={5}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
              />
              {pasteText.trim() && (
                <button
                  onClick={handlePasteSubmit}
                  className="mt-2 flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
                >
                  Parse & Continue <ArrowRight className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        )}

        {/* ── Step 1: Field Mapping ── */}
        {step === 1 && (
          <div className="space-y-6">
            <div>
              <h2 className="mb-1 text-lg font-semibold text-gray-800 dark:text-gray-200">
                Map CSV Columns to GGID Fields
              </h2>
              <p className="text-sm text-gray-500">
                Auto-detected mappings are shown. Adjust as needed. Fields marked with <span className="text-red-500">*</span> are required.
              </p>
            </div>

            {!requiredFieldsMapped && (
              <div className="flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-400">
                <AlertCircle className="h-4 w-4" />
                Please map all required fields (marked with *) before continuing.
              </div>
            )}

            <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900">
                  <tr>
                    <th scope="col" className="px-4 py-2 text-left font-medium text-gray-500">CSV Column</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium text-gray-500">{t("userImport.sampleValue")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium text-gray-500">{t("userImport.mapTo")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                  {headers.map((header: any) => {
                    const sampleVal = rows[0]?.[headers.indexOf(header)] || "";
                    const mapped = mapping[header];
                    return (
                      <tr key={header} className={mapped ? "" : "bg-yellow-50 dark:bg-yellow-950/30"}>
                        <td className="px-4 py-2.5 font-medium text-gray-700 dark:text-gray-300">
                          {header}
                          {!mapped && (
                            <span className="ml-2 rounded bg-yellow-100 px-1.5 py-0.5 text-xs text-yellow-700 dark:bg-yellow-900 dark:text-yellow-400">
                              unmapped
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-2.5 text-gray-400">{sampleVal || "—"}</td>
                        <td className="px-4 py-2.5">
                          <select
                            value={mapped || ""}
                            onChange={(e) => handleMappingChange(header, e.target.value)}
                            className="rounded-lg border border-gray-300 px-2 py-1.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100"
                          >
                            <option value="">— Do not import —</option>
                            {GGID_FIELDS.map((f: any) => (
                              <option key={f.key} value={f.key}>
                                {f.label} {f.required ? "*" : ""}
                              </option>
                            ))}
                          </select>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* ── Step 2: Preview ── */}
        {step === 2 && (
          <div className="space-y-6">
            <div>
              <h2 className="mb-1 flex items-center gap-2 text-lg font-semibold text-gray-800 dark:text-gray-200">
                <Eye className="h-5 w-5 text-brand-600" />
                Preview Data (First 10 Rows)
              </h2>
              <p className="text-sm text-gray-500">
                Showing first {Math.min(10, rows.length)} of {rows.length} rows. Cells with errors are highlighted in red.
              </p>
            </div>

            <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900">
                  <tr>
                    <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">#</th>
                    {headers.filter((h: any) => mapping[h]).map((h: any) => (
                      <th scope="col" key={h} className="px-3 py-2 text-left font-medium text-gray-600 dark:text-gray-300">
                        {GGID_FIELDS.find((f: any) => f.key === mapping[h])?.label || h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                  {rows.slice(0, 10).map((row: any, i: any) => (
                    <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900">
                      <td className="px-3 py-2 text-xs text-gray-400">{i + 2}</td>
                      {headers.filter((h: any) => mapping[h]).map((h: any) => {
                        const idx = headers.indexOf(h);
                        const hasError = getRowFieldErrors(row, idx, h, mapping, i + 2, allRowErrors);
                        return (
                          <td
                            key={h}
                            className={`px-3 py-2 ${hasError ? "border-2 border-red-400 bg-red-50 dark:bg-red-950/30" : "text-gray-700 dark:text-gray-300"}`}
                          >
                            {row[idx] || "—"}
                          </td>
                        );
                      })}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {allRowErrors.length > 0 && (
              <div className="flex items-center gap-2 text-sm text-amber-600 dark:text-amber-400">
                <AlertCircle className="h-4 w-4" />
                {allRowErrors.length} validation issue{allRowErrors.length !== 1 ? "s" : ""} detected. Proceed to Dry Run for full details.
              </div>
            )}
          </div>
        )}

        {/* ── Step 3: Dry Run ── */}
        {step === 3 && (
          <div className="space-y-6">
            <div>
              <h2 className="mb-1 flex items-center gap-2 text-lg font-semibold text-gray-800 dark:text-gray-200">
                <PlayCircle className="h-5 w-5 text-brand-600" />
                Dry Run — Full Validation
              </h2>
              <p className="text-sm text-gray-500">
                Validate all {rows.length} rows without importing.
              </p>
            </div>

            {!validationResult && !validating && (
              <button
                onClick={runFullValidation}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-700"
              >
                <PlayCircle className="h-4 w-4" />
                Run Validation
              </button>
            )}

            {validating && (
              <div className="flex items-center gap-2 text-sm text-gray-500">
                <Loader2 className="h-4 w-4 animate-spin" />
                Validating {rows.length} rows...
              </div>
            )}

            {validationResult && (
              <>
                {/* Summary cards */}
                <div className="grid grid-cols-3 gap-4">
                  <div className="rounded-lg border border-green-200 bg-green-50 p-4 text-center dark:border-green-800 dark:bg-green-950">
                    <CheckCircle2 className="mx-auto mb-1 h-6 w-6 text-green-500" />
                    <p className="text-2xl font-bold text-green-700 dark:text-green-400">{validationResult.valid}</p>
                    <p className="text-xs text-gray-500">{t("userImport.validRows")}</p>
                  </div>
                  <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-center dark:border-red-800 dark:bg-red-950">
                    <XCircle className="mx-auto mb-1 h-6 w-6 text-red-500" />
                    <p className="text-2xl font-bold text-red-700 dark:text-red-400">{validationResult.errors}</p>
                    <p className="text-xs text-gray-500">{t("common.error")}</p>
                  </div>
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-center dark:border-amber-800 dark:bg-amber-950">
                    <AlertCircle className="mx-auto mb-1 h-6 w-6 text-amber-500" />
                    <p className="text-2xl font-bold text-amber-700 dark:text-amber-400">{validationResult.duplicates}</p>
                    <p className="text-xs text-gray-500">{t("userImport.duplicates")}</p>
                  </div>
                </div>

                {/* Error details */}
                {validationResult.errorList.length > 0 && (
                  <div>
                    <h3 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
                      Error Details ({validationResult.errorList.length})
                    </h3>
                    <div className="max-h-64 overflow-y-auto rounded-lg border border-gray-200 dark:border-gray-700">
                      <table className="w-full text-sm">
                        <thead className="sticky top-0 bg-gray-50 dark:bg-gray-900">
                          <tr>
                            <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("userImport.row")}</th>
                            <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("userImport.field")}</th>
                            <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("common.error")}</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                          {validationResult.errorList.slice(0, 100).map((e: any, i: any) => (
                            <tr key={i}>
                              <td className="px-3 py-1.5 text-gray-500">{e.row}</td>
                              <td className="px-3 py-1.5 text-gray-600 dark:text-gray-300">{e.field}</td>
                              <td className="px-3 py-1.5 text-red-600 dark:text-red-400">{e.message}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    {validationResult.errorList.length > 100 && (
                      <p className="mt-1 text-xs text-gray-400">
                        Showing first 100 of {validationResult.errorList.length} errors.
                      </p>
                    )}
                  </div>
                )}
              </>
            )}
          </div>
        )}

        {/* ── Step 4: Import ── */}
        {step === 4 && (
          <div className="space-y-6">
            {importStatus === "idle" && (
              <>
                <div>
                  <h2 className="mb-1 flex items-center gap-2 text-lg font-semibold text-gray-800 dark:text-gray-200">
                    <Database className="h-5 w-5 text-brand-600" />
                    Ready to Import
                  </h2>
                  <p className="text-sm text-gray-500">
                    {rows.length} rows will be processed. Valid rows will be created; rows with errors will be skipped.
                  </p>
                </div>

                {validationResult && (
                  <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-900">
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div>
                        <p className="text-lg font-bold text-green-600">{validationResult.valid}</p>
                        <p className="text-xs text-gray-400">to be created</p>
                      </div>
                      <div>
                        <p className="text-lg font-bold text-amber-600">{validationResult.duplicates}</p>
                        <p className="text-xs text-gray-400">duplicates (skipped)</p>
                      </div>
                      <div>
                        <p className="text-lg font-bold text-red-600">{validationResult.errors - validationResult.duplicates}</p>
                        <p className="text-xs text-gray-400">other errors (skipped)</p>
                      </div>
                    </div>
                  </div>
                )}

                <button
                  onClick={startImport}
                  className="flex items-center gap-2 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-700"
                >
                  <Upload className="h-4 w-4" />
                  Start Import
                </button>
              </>
            )}

            {importStatus === "importing" && (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <Loader2 className="h-4 w-4 animate-spin text-brand-600" />
                  Importing users... {importProgress}/{rows.length}
                </div>

                {/* Progress bar */}
                <div className="h-3 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div
                    className="h-full bg-brand-600 transition-all duration-300"
                    style={{ width: `${rows.length > 0 ? (importProgress / rows.length) * 100 : 0}%` }}
                  />
                </div>

                {/* Live counters */}
                <div className="grid grid-cols-3 gap-4">
                  <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-center dark:border-green-800 dark:bg-green-950">
                    <CheckCircle2 className="mx-auto mb-1 h-5 w-5 text-green-500" />
                    <p className="text-xl font-bold text-green-700 dark:text-green-400">{importStats.created}</p>
                    <p className="text-xs text-gray-500">Created</p>
                  </div>
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-center dark:border-amber-800 dark:bg-amber-950">
                    <AlertCircle className="mx-auto mb-1 h-5 w-5 text-amber-500" />
                    <p className="text-xl font-bold text-amber-700 dark:text-amber-400">{importStats.skipped}</p>
                    <p className="text-xs text-gray-500">Skipped</p>
                  </div>
                  <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-center dark:border-red-800 dark:bg-red-950">
                    <XCircle className="mx-auto mb-1 h-5 w-5 text-red-500" />
                    <p className="text-xl font-bold text-red-700 dark:text-red-400">{importStats.failed}</p>
                    <p className="text-xs text-gray-500">Failed</p>
                  </div>
                </div>
              </div>
            )}

            {importStatus === "done" && (
              <div className="space-y-6">
                <div className="flex flex-col items-center py-8">
                  <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-950">
                    <CheckCircle2 className="h-8 w-8 text-green-600" />
                  </div>
                  <h2 className="text-xl font-bold text-gray-800 dark:text-gray-200">{t("userImport.importComplete")}</h2>
                  <p className="mt-1 text-sm text-gray-500">
                    Processed {rows.length} rows
                  </p>
                </div>

                <div className="grid grid-cols-3 gap-4">
                  <div className="rounded-lg border border-green-200 bg-green-50 p-4 text-center dark:border-green-800 dark:bg-green-950">
                    <p className="text-3xl font-bold text-green-700 dark:text-green-400">{importStats.created}</p>
                    <p className="text-xs text-gray-500">{t("userImport.usersCreated")}</p>
                  </div>
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-center dark:border-amber-800 dark:bg-amber-950">
                    <p className="text-3xl font-bold text-amber-700 dark:text-amber-400">{importStats.skipped}</p>
                    <p className="text-xs text-gray-500">Skipped</p>
                  </div>
                  <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-center dark:border-red-800 dark:bg-red-950">
                    <p className="text-3xl font-bold text-red-700 dark:text-red-400">{importStats.failed}</p>
                    <p className="text-xs text-gray-500">Failed</p>
                  </div>
                </div>

                <div className="flex gap-3">
                  <Link
                    href="/users"
                    className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-6 py-2.5 text-sm font-medium text-white hover:bg-brand-700"
                  >
                    <Download className="h-4 w-4" />
                    View Users
                  </Link>
                  <button
                    onClick={reset}
                    className="rounded-lg border border-gray-300 px-6 py-2.5 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
                  >
                    Import Another File
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Navigation */}
        {step < 4 && !(step === 4 && importStatus === "importing") && (
          <div className="mt-8 flex justify-between border-t border-gray-100 pt-4 dark:border-gray-700">
            <button
              onClick={() => setStep(Math.max(0, step - 1))}
              disabled={step === 0}
              className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800"
            >
              <ChevronLeft className="h-4 w-4" />
              Back
            </button>
            {step < 3 && (
              <button
                onClick={() => canProceed(step) && setStep(step + 1)}
                disabled={!canProceed(step)}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </button>
            )}
            {step === 3 && validationResult && (
              <button
                onClick={() => setStep(4)}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                Proceed to Import
                <ChevronRight className="h-4 w-4" />
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
