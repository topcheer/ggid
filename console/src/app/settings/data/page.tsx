"use client";

import { useState, useRef, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Upload,
  Download,
  FileJson,
  FileSpreadsheet,
  Loader2,
  AlertCircle,
  CheckCircle,
  XCircle,
  Database,
  ArrowRight,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type DataType = "users" | "groups" | "roles";

const TABS: { key: DataType; label: string; fields: string[] }[] = [
  {
    key: "users",
    label: "Users",
    fields: ["username", "email", "display_name", "phone", "status", "locale", "timezone"],
  },
  {
    key: "groups",
    label: "Groups",
    fields: ["name", "description", "parent_id", "org_id"],
  },
  {
    key: "roles",
    label: "Roles",
    fields: ["name", "key", "description", "priority"],
  },
];

interface ColumnMapping {
  csvColumn: string;
  ggidField: string;
}

interface DryRunResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  validCount: number;
  totalCount: number;
}

function parseCSV(text: string): { headers: string[]; rows: string[][] } {
  const lines = text.trim().split(/\r?\n/);
  if (lines.length === 0) return { headers: [], rows: [] };

  const parseLine = (line: string): string[] => {
    const result: string[] = [];
    let current = "";
    let inQuotes = false;
    for (let i = 0; i < line.length; i++) {
      const char = line[i];
      if (char === '"') {
        if (inQuotes && line[i + 1] === '"') {
          current += '"';
          i++;
        } else {
          inQuotes = !inQuotes;
        }
      } else if (char === "," && !inQuotes) {
        result.push(current);
        current = "";
      } else {
        current += char;
      }
    }
    result.push(current);
    return result;
  };

  const headers = parseLine(lines[0]);
  const rows = lines.slice(1).map(parseLine);
  return { headers, rows };
}

function rowsToObjects(
  headers: string[],
  rows: string[][],
  mappings: ColumnMapping[],
): Record<string, string>[] {
  return rows.map((row) => {
    const obj: Record<string, string> = {};
    mappings.forEach((m) => {
      if (m.ggidField) {
        const colIdx = headers.indexOf(m.csvColumn);
        if (colIdx >= 0 && colIdx < row.length) {
          obj[m.ggidField] = row[colIdx];
        }
      }
    });
    return obj;
  });
}

function objectsToCSV(objects: Record<string, unknown>[]): string {
  if (objects.length === 0) return "";
  const keys = Array.from(
    objects.reduce((set, obj) => {
      Object.keys(obj).forEach((k) => set.add(k));
      return set;
    }, new Set<string>()),
  );
  const escapeCSV = (val: unknown): string => {
    const s = val === null || val === undefined ? "" : String(val);
    if (s.includes(",") || s.includes('"') || s.includes("\n")) {
      return `"${s.replace(/"/g, '""')}"`;
    }
    return s;
  };
  const headerLine = keys.join(",");
  const dataLines = objects.map((obj) => keys.map((k) => escapeCSV(obj[k])).join(","));
  return [headerLine, ...dataLines].join("\n");
}

function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

export default function DataPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [activeTab, setActiveTab] = useState<DataType>("users");

  // Import state
  const [fileName, setFileName] = useState<string | null>(null);
  const [fileFormat, setFileFormat] = useState<"csv" | "json">("csv");
  const [csvHeaders, setCsvHeaders] = useState<string[]>([]);
  const [csvRows, setCsvRows] = useState<string[][]>([]);
  const [parsedJson, setParsedJson] = useState<Record<string, unknown>[]>([]);
  const [mappings, setMappings] = useState<ColumnMapping[]>([]);
  const [importError, setImportError] = useState<string | null>(null);
  const [progress, setProgress] = useState<string | null>(null);
  const [dryRunResult, setDryRunResult] = useState<DryRunResult | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const currentTab = TABS.find((t) => t.key === activeTab)!;

  const resetImportState = useCallback(() => {
    setFileName(null);
    setCsvHeaders([]);
    setCsvRows([]);
    setParsedJson([]);
    setMappings([]);
    setImportError(null);
    setProgress(null);
    setDryRunResult(null);
  }, []);

  const switchTab = (tab: DataType) => {
    setActiveTab(tab);
    resetImportState();
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    resetImportState();
    setFileName(file.name);

    const ext = file.name.split(".").pop()?.toLowerCase();
    const reader = new FileReader();
    reader.onload = (ev) => {
      const text = ev.target?.result as string;
      try {
        if (ext === "json") {
          setFileFormat("json");
          const parsed = JSON.parse(text);
          const arr = Array.isArray(parsed) ? parsed : [parsed];
          setParsedJson(arr);
          setProgress(`${arr.length} records loaded from JSON`);
        } else {
          setFileFormat("csv");
          const { headers, rows } = parseCSV(text);
          if (headers.length === 0) {
            setImportError("CSV file appears to be empty");
            return;
          }
          setCsvHeaders(headers);
          setCsvRows(rows);
          // Auto-map columns by fuzzy matching header names to fields
          const autoMappings: ColumnMapping[] = headers.map((h) => {
            const normalized = h.toLowerCase().replace(/[\s_-]/g, "");
            const match = currentTab.fields.find(
              (f) => f.toLowerCase().replace(/[\s_-]/g, "") === normalized,
            );
            return { csvColumn: h, ggidField: match || "" };
          });
          setMappings(autoMappings);
          setProgress(`${rows.length} records loaded from CSV`);
        }
      } catch {
        setImportError(`Failed to parse ${ext?.toUpperCase()} file`);
      }
    };
    reader.readAsText(file);
  };

  const updateMapping = (csvCol: string, ggidField: string) => {
    setMappings(mappings.map((m) => (m.csvColumn === csvCol ? { ...m, ggidField } : m)));
  };

  const getPreviewRecords = (): Record<string, string>[] => {
    if (fileFormat === "json") return parsedJson.slice(0, 5) as Record<string, string>[];
    return rowsToObjects(csvHeaders, csvRows.slice(0, 5), mappings);
  };

  const getAllRecords = (): Record<string, unknown>[] => {
    if (fileFormat === "json") return parsedJson;
    return rowsToObjects(csvHeaders, csvRows, mappings);
  };

  const handleDryRun = async () => {
    const records = getAllRecords();
    if (records.length === 0) {
      setImportError("No records to validate");
      return;
    }
    setProgress("Validating...");
    setDryRunResult(null);
    try {
      const data = await apiFetch<DryRunResult>(`/api/v1/${activeTab}/import?dry_run=true`, {
        method: "POST",
        body: JSON.stringify({ items: records, dry_run: true }),
      });
      setDryRunResult(data);
      setProgress(null);
    } catch {
      // If API doesn't support dry-run, do client-side validation
      const errors: string[] = [];
      const warnings: string[] = [];
      const requiredFields = activeTab === "users" ? ["username", "email"] : ["name"];
      records.forEach((rec: any, idx: number) => {
        requiredFields.forEach((f) => {
          if (!rec[f]) {
            errors.push(`Row ${idx + 1}: missing required field "${f}"`);
          }
        });
      });
      const result: DryRunResult = {
        valid: errors.length === 0,
        errors,
        warnings,
        validCount: records.length - errors.length,
        totalCount: records.length,
      };
      setDryRunResult(result);
      setProgress(null);
    }
  };

  const handleImport = async () => {
    const records = getAllRecords();
    if (records.length === 0) {
      setImportError("No records to import");
      return;
    }
    setProgress(`Importing ${records.length} ${activeTab}...`);
    setImportError(null);
    try {
      await apiFetch(`/api/v1/${activeTab}/import`, {
        method: "POST",
        body: JSON.stringify({ items: records }),
      });
      setProgress(`Successfully imported ${records.length} ${activeTab}`);
      setDryRunResult(null);
    } catch (err) {
      setProgress(null);
      setImportError(err instanceof Error ? err.message : "Import failed");
    }
  };

  const handleExportCSV = async () => {
    setProgress(`Exporting ${activeTab} as CSV...`);
    setImportError(null);
    try {
      const data = await apiFetch<{ items?: Record<string, unknown>[]; users?: Record<string, unknown>[]; roles?: Record<string, unknown>[]; groups?: Record<string, unknown>[] }>(
        `/api/v1/${activeTab}`,
      );
      const list = data.items || data[activeTab] || [];
      if (list.length === 0) {
        setProgress("No records to export");
        return;
      }
      const csv = objectsToCSV(list);
      downloadFile(csv, `${activeTab}_export.csv`, "text/csv");
      setProgress(`Exported ${list.length} ${activeTab} as CSV`);
    } catch (err) {
      setProgress(null);
      setImportError(err instanceof Error ? err.message : "CSV export failed");
    }
  };

  const handleExportJSON = async () => {
    setProgress(`Exporting ${activeTab} as JSON...`);
    setImportError(null);
    try {
      const data = await apiFetch<{ items?: Record<string, unknown>[]; users?: Record<string, unknown>[]; roles?: Record<string, unknown>[]; groups?: Record<string, unknown>[] }>(
        `/api/v1/${activeTab}`,
      );
      const list = data.items || data[activeTab] || [];
      if (list.length === 0) {
        setProgress("No records to export");
        return;
      }
      downloadFile(JSON.stringify(list, null, 2), `${activeTab}_export.json`, "application/json");
      setProgress(`Exported ${list.length} ${activeTab} as JSON`);
    } catch (err) {
      setProgress(null);
      setImportError(err instanceof Error ? err.message : "JSON export failed");
    }
  };

  const previewRecords = getPreviewRecords();
  const previewKeys = previewRecords.length > 0 ? Object.keys(previewRecords[0]) : [];
  const totalRecords = fileFormat === "json" ? parsedJson.length : csvRows.length;

  return (
    <div>
      <div className="mb-6">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Database className="h-6 w-6 text-brand-600" /> Import / Export Data
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          Bulk import and export users, groups, and roles via CSV or JSON.
        </p>
      </div>

      {/* Tabs */}
      <div className="mb-6 flex gap-1 border-b border-gray-200 dark:border-gray-700">
        {TABS.map((tab) => (
          <button
            key={tab.key}
            onClick={() => switchTab(tab.key)}
            className={`border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              activeTab === tab.key
                ? "border-brand-600 text-brand-600"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Import Section */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <Upload className="h-5 w-5 text-brand-600" /> Import {currentTab.label}
          </h2>

          {/* File Upload */}
          <div className="mb-4">
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,.json"
              onChange={handleFileUpload}
              className="hidden"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              className="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 px-4 py-8 text-sm text-gray-500 hover:border-brand-400 hover:bg-brand-50 dark:border-gray-600 dark:hover:border-brand-500 dark:hover:bg-gray-700"
            >
              <Upload className="h-5 w-5" />
              {fileName ? (
                <span className="font-medium text-gray-700 dark:text-gray-300">{fileName}</span>
              ) : (
                <span>Click to upload CSV or JSON file</span>
              )}
            </button>
          </div>

          {importError && (
            <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
              <AlertCircle className="h-4 w-4 shrink-0" /> {importError}
            </div>
          )}

          {/* CSV Column Mapping */}
          {fileFormat === "csv" && csvHeaders.length > 0 && (
            <div className="mb-4">
              <h3 className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                Column Mapping
              </h3>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-gray-200 dark:border-gray-700">
                      <th scope="col" className="px-2 py-1.5 text-left text-xs font-medium uppercase text-gray-500">
                        CSV Column
                      </th>
                      <th scope="col" className="px-2 py-1.5"></th>
                      <th scope="col" className="px-2 py-1.5 text-left text-xs font-medium uppercase text-gray-500">
                        GGID Field
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                    {mappings.map((m) => (
                      <tr key={m.csvColumn}>
                        <td className="px-2 py-1.5 text-sm text-gray-700 dark:text-gray-300">
                          {m.csvColumn}
                        </td>
                        <td className="px-2 py-1.5">
                          <ArrowRight className="h-3 w-3 text-gray-400" />
                        </td>
                        <td className="px-2 py-1.5">
                          <select
                            value={m.ggidField}
                            onChange={(e) => updateMapping(m.csvColumn, e.target.value)}
                            className="w-full rounded border border-gray-300 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                          >
                            <option value="">— Skip —</option>
                            {currentTab.fields.map((f) => (
                              <option key={f} value={f}>
                                {f}
                              </option>
                            ))}
                          </select>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Preview Table */}
          {previewRecords.length > 0 && (
            <div className="mb-4">
              <h3 className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                Preview (first {Math.min(5, previewRecords.length)} rows)
              </h3>
              <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
                <table className="w-full">
                  <thead className="bg-gray-50 dark:bg-gray-900">
                    <tr>
                      {previewKeys.map((k) => (
                        <th
                          key={k}
                          className="px-3 py-1.5 text-left text-xs font-medium uppercase text-gray-500"
                        >
                          {k}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                    {previewRecords.map((row: any, i: number) => (
                      <tr key={i}>
                        {previewKeys.map((k) => (
                          <td
                            key={k}
                            className="max-w-[200px] truncate px-3 py-1.5 text-xs text-gray-700 dark:text-gray-300"
                            title={row[k]}
                          >
                            {row[k] || "—"}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <p className="mt-1 text-xs text-gray-400">{totalRecords} total records in file</p>
            </div>
          )}

          {/* Dry Run Results */}
          {dryRunResult && (
            <div
              className={`mb-4 rounded-lg border p-4 ${
                dryRunResult.valid
                  ? "border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950"
                  : "border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950"
              }`}
            >
              <div className="flex items-center gap-2">
                {dryRunResult.valid ? (
                  <CheckCircle className="h-5 w-5 text-green-600" />
                ) : (
                  <AlertCircle className="h-5 w-5 text-amber-600" />
                )}
                <span className="text-sm font-semibold dark:text-gray-200">
                  Validation: {dryRunResult.validCount}/{dryRunResult.totalCount} records valid
                </span>
              </div>
              {dryRunResult.errors.length > 0 && (
                <ul className="mt-2 space-y-1">
                  {dryRunResult.errors.slice(0, 10).map((err: any, i: number) => (
                    <li key={i} className="flex items-start gap-1 text-xs text-red-600 dark:text-red-400">
                      <XCircle className="mt-0.5 h-3 w-3 shrink-0" /> {err}
                    </li>
                  ))}
                  {dryRunResult.errors.length > 10 && (
                    <li className="text-xs text-gray-400">
                      ... and {dryRunResult.errors.length - 10} more errors
                    </li>
                  )}
                </ul>
              )}
            </div>
          )}

          {/* Progress */}
          {progress && (
            <div className="mb-4 flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-400">
              <Loader2 className="h-4 w-4 animate-spin" /> {progress}
            </div>
          )}

          {/* Action Buttons */}
          {totalRecords > 0 && (
            <div className="flex gap-2">
              <button
                onClick={handleDryRun}
                className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 dark:text-gray-200"
              >
                Dry Run
              </button>
              <button
                onClick={handleImport}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                <Upload className="h-4 w-4" /> Import {totalRecords} records
              </button>
            </div>
          )}
        </div>

        {/* Export Section */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <Download className="h-5 w-5 text-brand-600" /> Export {currentTab.label}
          </h2>

          <p className="mb-4 text-sm text-gray-500">
            Download all {currentTab.label.toLowerCase()} data in your preferred format.
          </p>

          <div className="space-y-3">
            <button
              onClick={handleExportCSV}
              className="flex w-full items-center justify-between rounded-lg border border-gray-300 px-4 py-3 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 dark:text-gray-200"
            >
              <span className="flex items-center gap-2">
                <FileSpreadsheet className="h-5 w-5 text-green-600" />
                Export as CSV
              </span>
              <Download className="h-4 w-4 text-gray-400" />
            </button>

            <button
              onClick={handleExportJSON}
              className="flex w-full items-center justify-between rounded-lg border border-gray-300 px-4 py-3 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 dark:text-gray-200"
            >
              <span className="flex items-center gap-2">
                <FileJson className="h-5 w-5 text-blue-600" />
                Export as JSON
              </span>
              <Download className="h-4 w-4 text-gray-400" />
            </button>
          </div>

          <div className="mt-6 rounded-lg bg-gray-50 p-4 dark:bg-gray-900">
            <h3 className="mb-2 text-xs font-semibold uppercase text-gray-500">Supported Fields</h3>
            <div className="flex flex-wrap gap-1.5">
              {currentTab.fields.map((f) => (
                <span
                  key={f}
                  className="rounded bg-gray-200 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400"
                >
                  {f}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
