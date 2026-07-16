"use client";

import { useState, useCallback, useRef } from "react";
import { useUsers, useApi, type User } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import Link from "next/link";
import {
  Search,
  Plus,
  Lock,
  Unlock,
  Trash2,
  UserPlus,
  ChevronLeft,
  ChevronRight,
  Shield,
  Download,
  Upload,
  X,
  Cloud,
  ChevronDown,
  FileText,
  FileJson,
} from "lucide-react";

const PAGE_SIZE = 10;

// GGID fields available for CSV column mapping
const GGID_FIELDS = [
  { key: "username", label: "Username", required: true },
  { key: "email", label: "Email", required: true },
  { key: "display_name", label: "Display Name", required: false },
  { key: "phone", label: "Phone", required: false },
] as const;

type GgidFieldKey = (typeof GGID_FIELDS)[number]["key"];

export default function UsersPage() {
  const { users, loading, error, refresh } = useUsers();
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [page, setPage] = useState(0);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [batchRole, setBatchRole] = useState("");
  const [msg, setMsg] = useState<string | null>(null);
  const [roles, setRoles] = useState<{ id: string; key: string; name: string }[]>([]);

  // --- Legacy text import state (keep existing) ---
  const [showImport, setShowImport] = useState(false);
  const [importText, setImportText] = useState("");
  const [importResult, setImportResult] = useState<string | null>(null);

  // --- CSV import modal state ---
  const [showCsvImport, setShowCsvImport] = useState(false);
  const [csvData, setCsvData] = useState<string[][]>([]);
  const [csvHeaders, setCsvHeaders] = useState<string[]>([]);
  const [columnMapping, setColumnMapping] = useState<Record<string, GgidFieldKey>>({});
  const [csvImporting, setCsvImporting] = useState(false);
  const [csvImportResult, setCsvImportResult] = useState<string | null>(null);
  const csvFileRef = useRef<HTMLInputElement>(null);

  // --- Export dropdown state ---
  const [showExportMenu, setShowExportMenu] = useState(false);

  // Load roles for batch assign
  useCallback(async () => {
    const data = await apiFetch<{ roles?: { id: string; key: string; name: string }[] }>("/api/v1/roles").catch(() => ({ roles: [] }));
    setRoles(data.roles || []);
  }, [apiFetch]);

  const filtered = users.filter(
    (u) =>
      u.username.toLowerCase().includes(search.toLowerCase()) ||
      u.email.toLowerCase().includes(search.toLowerCase()),
  );

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paginated = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selected.size === paginated.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(paginated.map((u) => u.id)));
    }
  };

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    try {
      await apiFetch("/api/v1/users", {
        method: "POST",
        body: JSON.stringify({
          username: formData.get("username"),
          email: formData.get("email"),
          password: formData.get("password"),
        }),
      });
      setShowCreate(false);
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : t("users.createFailed"));
    }
  };

  const handleLock = async (userId: string, currentStatus: string) => {
    const action = currentStatus === "active" ? "lock" : "unlock";
    try {
      await apiFetch(`/api/v1/users/${userId}/${action}`, { method: "POST" });
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : `Failed to ${action} user`);
    }
  };

  const handleDelete = async (userId: string, username: string) => {
    if (!confirm(`Delete user "${username}"?`)) return;
    try {
      await apiFetch(`/api/v1/users/${userId}`, { method: "DELETE" });
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed");
    }
  };

  const handleBatchDelete = async () => {
    if (selected.size === 0) return;
    if (!confirm(`Delete ${selected.size} selected users?`)) return;
    try {
      await Promise.all([...selected].map((id) => apiFetch(`/api/v1/users/${id}`, { method: "DELETE" })));
      setSelected(new Set());
      setMsg(`Deleted ${selected.size} users`);
      refresh();
    } catch (err) {
      alert(err instanceof Error ? err.message : t("users.batchDeleteFailed"));
    }
  };

  const handleBatchAssignRole = async () => {
    if (selected.size === 0 || !batchRole) return;
    try {
      await Promise.all(
        [...selected].map((id) =>
          apiFetch(`/api/v1/users/${id}/roles`, { method: "POST", body: JSON.stringify({ role_id: batchRole }) }),
        ),
      );
      setMsg(`Role assigned to ${selected.size} users`);
      setSelected(new Set());
      setBatchRole("");
    } catch (err) {
      alert(err instanceof Error ? err.message : t("users.batchAssignFailed"));
    }
  };

  // --- Export helpers ---
  const handleExportCSV = () => {
    const header = "username,email,displayName,phone,status,created_at\n";
    const rows = filtered
      .map((u) =>
        [
          u.username || "",
          u.email || "",
          u.display_name || "",
          u.phone || "",
          u.status || "active",
          u.created_at || "",
        ]
          .map((v) => `"${String(v).replace(/"/g, '""')}"`)
          .join(","),
      )
      .join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "users_export.csv";
    a.click();
    URL.revokeObjectURL(url);
    setShowExportMenu(false);
  };

  const handleExportJSON = () => {
    const blob = new Blob([JSON.stringify(filtered, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "users_export.json";
    a.click();
    URL.revokeObjectURL(url);
    setShowExportMenu(false);
  };

  // --- Legacy text import (keep existing) ---
  const handleImportCSV = async () => {
    const lines = importText.trim().split("\n").filter((l) => l.trim() && !l.startsWith("username,"));
    let created = 0;
    const errors: string[] = [];
    for (let i = 0; i < lines.length; i++) {
      const [username, email, password] = lines[i].split(",").map((s) => s.trim());
      if (!username || !email) { errors.push(`Row ${i + 1}: missing username or email`); continue; }
      try {
        await apiFetch("/api/v1/users", {
          method: "POST",
          body: JSON.stringify({ username, email, password: password || "TempPass123!" }),
        });
        created++;
      } catch (err) {
        errors.push(`Row ${i + 1}: ${err instanceof Error ? err.message : "failed"}`);
      }
    }
    setImportResult(`Created ${created} users${errors.length ? `, ${errors.length} errors: ${errors.join("; ")}` : ""}`);
    setImportText("");
    refresh();
  };

  // --- CSV file import ---
  const handleCsvFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const text = ev.target?.result as string;
      parseCsv(text);
    };
    reader.readAsText(file);
    e.target.value = "";
  };

  const parseCsv = (text: string) => {
    const lines = text.trim().split(/\r?\n/).filter((l) => l.trim());
    if (lines.length === 0) return;

    // Parse CSV (basic: handle quoted fields)
    const parseLine = (line: string): string[] => {
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
    };

    const headers = parseLine(lines[0]);
    const dataRows = lines.slice(1).map(parseLine);

    setCsvHeaders(headers);
    setCsvData(dataRows);

    // Auto-detect column mapping
    const mapping: Record<string, GgidFieldKey> = {};
    headers.forEach((header) => {
      const lower = header.toLowerCase().trim();
      if (lower === "username" || lower === "user_name" || lower === "login") {
        mapping[header] = "username";
      } else if (lower === "email" || lower === "e-mail" || lower === "mail") {
        mapping[header] = "email";
      } else if (lower === "display_name" || lower === "displayname" || lower === "name" || lower === "full_name") {
        mapping[header] = "display_name";
      } else if (lower === "phone" || lower === "phone_number" || lower === "tel") {
        mapping[header] = "phone";
      }
    });
    setColumnMapping(mapping);
    setCsvImportResult(null);
    setShowCsvImport(true);
  };

  const handleMappingChange = (csvColumn: string, ggidField: GgidFieldKey | "") => {
    setColumnMapping((prev) => {
      const next = { ...prev };
      if (ggidField === "") {
        delete next[csvColumn];
      } else {
        next[csvColumn] = ggidField;
      }
      return next;
    });
  };

  const handleCsvImport = async () => {
    // Check required fields
    const mappedFields = Object.values(columnMapping);
    if (!mappedFields.includes("username")) {
      setCsvImportResult("Error: username field must be mapped");
      return;
    }
    if (!mappedFields.includes("email")) {
      setCsvImportResult("Error: email field must be mapped");
      return;
    }

    setCsvImporting(true);
    let created = 0;
    const errors: string[] = [];

    for (let i = 0; i < csvData.length; i++) {
      const row = csvData[i];
      const payload: Record<string, string> = {};
      csvHeaders.forEach((header, idx) => {
        const field = columnMapping[header];
        if (field) {
          payload[field] = row[idx] || "";
        }
      });

      if (!payload.username || !payload.email) {
        errors.push(`Row ${i + 2}: missing required field`);
        continue;
      }

      try {
        await apiFetch("/api/v1/users", {
          method: "POST",
          body: JSON.stringify({
            username: payload.username,
            email: payload.email,
            display_name: payload.display_name || undefined,
            phone: payload.phone || undefined,
            password: "TempPass123!",
          }),
        });
        created++;
      } catch (err) {
        errors.push(`Row ${i + 2}: ${err instanceof Error ? err.message : "failed"}`);
      }
    }

    setCsvImporting(false);
    setCsvImportResult(
      `Imported ${created} users${errors.length ? `, ${errors.length} errors: ${errors.slice(0, 5).join("; ")}${errors.length > 5 ? "..." : ""}` : ""}`,
    );
    if (created > 0) {
      refresh();
    }
  };

  const closeCsvImport = () => {
    setShowCsvImport(false);
    setCsvData([]);
    setCsvHeaders([]);
    setColumnMapping({});
    setCsvImportResult(null);
  };

  // --- SCIM source badge ---
  const getScimBadge = (user: User) => {
    const source = (user as unknown as Record<string, unknown>).scim_source as string | undefined;
    if (!source) return null;
    const lower = source.toLowerCase();
    let label = source;
    let colorClasses = "bg-blue-50 text-blue-600 border-blue-200";

    if (lower.includes("okta")) {
      label = "Okta";
      colorClasses = "bg-blue-50 text-blue-700 border-blue-300 dark:bg-blue-950 dark:text-blue-400 dark:border-blue-800";
    } else if (lower.includes("azure") || lower.includes("entra") || lower.includes("microsoft")) {
      label = "Azure AD";
      colorClasses = "bg-sky-50 text-sky-700 border-sky-300 dark:bg-sky-950 dark:text-sky-400 dark:border-sky-800";
    } else if (lower.includes("google")) {
      label = "Google";
      colorClasses = "bg-red-50 text-red-700 border-red-300 dark:bg-red-950 dark:text-red-400 dark:border-red-800";
    } else if (lower.includes("scim")) {
      label = "SCIM";
      colorClasses = "bg-purple-50 text-purple-700 border-purple-300 dark:bg-purple-950 dark:text-purple-400 dark:border-purple-800";
    }

    return (
      <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${colorClasses}`}>
        <Cloud className="h-3 w-3" />
        {label}
      </span>
    );
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold dark:text-gray-100">{t("users.title")}</h1>
        <div className="flex gap-2">
          {/* Export dropdown */}
          <div className="relative">
            <button
              onClick={() => setShowExportMenu(!showExportMenu)}
              className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              <Download className="h-4 w-4" /> {t("common.export")}
              <ChevronDown className="h-3.5 w-3.5" />
            </button>
            {showExportMenu && (
              <>
                <div className="fixed inset-0 z-10" onClick={() => setShowExportMenu(false)} />
                <div className="absolute right-0 z-20 mt-1 w-44 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800">
                  <button
                    onClick={handleExportCSV}
                    className="flex w-full items-center gap-2 px-4 py-2.5 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-700"
                  >
                    <FileText className="h-4 w-4 text-green-600" /> {t("users.exportCsv")}
                  </button>
                  <button
                    onClick={handleExportJSON}
                    className="flex w-full items-center gap-2 px-4 py-2.5 text-sm text-gray-700 hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-700"
                  >
                    <FileJson className="h-4 w-4 text-amber-600" /> {t("users.exportJson")}
                  </button>
                </div>
              </>
            )}
          </div>

          {/* Import CSV file */}
          <input
            ref={csvFileRef}
            type="file"
            accept=".csv"
            onChange={handleCsvFileSelect}
            className="hidden"
          />
          <button
            onClick={() => csvFileRef.current?.click()}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <Upload className="h-4 w-4" /> {t("users.importCsv")}
          </button>

          {/* Legacy text import */}
          <button
            onClick={() => setShowImport(!showImport)}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
            title="Paste CSV text"
          >
            <FileText className="h-4 w-4" /> {t("users.paste")}
          </button>

          <button
            onClick={() => setShowCreate(!showCreate)}
            className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <UserPlus className="h-4 w-4" /> {t("users.newUser")}
          </button>
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}

      {/* CSV Import Modal */}
      {showCsvImport && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold dark:text-gray-100">{t("users.importUsersCsv")}</h2>
            <button onClick={closeCsvImport} className="text-gray-400 hover:text-gray-600" aria-label="Close">
              <X className="h-5 w-5" />
            </button>
          </div>

          {/* Column mapping table */}
          <div className="mb-6">
            <h3 className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">{t("users.columnMapping")}</h3>
            <p className="mb-3 text-xs text-gray-500">
              {t("users.mappingHint")}
            </p>
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-200 dark:border-gray-700">
                  <th scope="col" className="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{t("users.csvColumn")}</th>
                  <th scope="col" className="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500">{t("users.ggidField")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {csvHeaders.map((header) => (
                  <tr key={header}>
                    <td className="px-3 py-2 text-sm font-medium text-gray-900 dark:text-gray-200">{header}</td>
                    <td className="px-3 py-2">
                      <select
                        value={columnMapping[header] || ""}
                        onChange={(e) => handleMappingChange(header, e.target.value as GgidFieldKey | "")}
                        className="rounded border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                      >
                        <option value="">{t("users.skip")}</option>
                        {GGID_FIELDS.map((field) => (
                          <option key={field.key} value={field.key}>
                            {field.label}
                            {field.required ? " (required)" : ""}
                          </option>
                        ))}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Preview */}
          <div className="mb-6">
            <h3 className="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("users.preview")} ({Math.min(5, csvData.length)}/{csvData.length})
            </h3>
            <div className="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
              <table className="w-full">
                <thead className="bg-gray-50 dark:bg-gray-700/50">
                  <tr>
                    {csvHeaders.map((header) => (
                      <th scope="col" key={header} className="px-3 py-2 text-left text-xs font-medium text-gray-500">
                        {header}
                        {columnMapping[header] && (
                          <span className="ml-1 text-brand-600">→ {columnMapping[header]}</span>
                        )}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                  {csvData.slice(0, 5).map((row, i) => (
                    <tr key={i}>
                      {csvHeaders.map((header, j) => (
                        <td key={header} className="px-3 py-2 text-sm text-gray-700 dark:text-gray-300">
                          {row[j] || "—"}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Import action */}
          <div className="flex items-center gap-3">
            <button
              onClick={handleCsvImport}
              disabled={csvImporting || csvData.length === 0}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {csvImporting ? (
                <>
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  {t("users.importing")}
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" /> {t("users.importUsers")} {csvData.length}
                </>
              )}
            </button>
            <button
              onClick={closeCsvImport}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
            >
              {t("common.cancel")}
            </button>
          </div>

          {csvImportResult && (
            <div className={`mt-4 rounded-lg border p-3 text-sm ${
              csvImportResult.startsWith("Error")
                ? "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
                : "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
            }`}>
              {csvImportResult}
            </div>
          )}
        </div>
      )}

      {/* Legacy paste import */}
      {showImport && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold">{t("users.importUsersCsv")}</h2>
            <button onClick={() => setShowImport(false)} aria-label="Close"><X className="h-4 w-4 text-gray-400" /></button>
          </div>
          <p className="mb-2 text-xs text-gray-500">{t("users.formatHint")}</p>
          <textarea
            value={importText}
            onChange={(e) => setImportText(e.target.value)}
            rows={6}
            placeholder={"alice,alice@example.com,Pass123!\nbob,bob@example.com,Pass123!"}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm"
          />
          <button onClick={handleImportCSV} disabled={!importText.trim()} className="mt-3 rounded-lg bg-brand-600 px-4 py-2 text-sm text-white hover:bg-brand-700 disabled:opacity-50">
            {t("users.importUsers")}
          </button>
          {importResult && <p className="mt-3 text-sm text-gray-600 dark:text-gray-400">{importResult}</p>}
        </div>
      )}

      {showCreate && (
        <form onSubmit={handleCreate} className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold">{t("users.createNew")}</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium">{t("users.usernameLbl")}</label>
              <input name="username" required className="w-full rounded-lg border border-gray-300 px-3 py-2" placeholder="johndoe" />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">{t("users.email")}</label>
              <input autoComplete="email" name="email" type="email" required className="w-full rounded-lg border border-gray-300 px-3 py-2" placeholder="john@example.com" />
            </div>
            <div className="col-span-2">
              <label className="mb-1 block text-sm font-medium">{t("users.passwordLbl")}</label>
              <input autoComplete="current-password" name="password" type="password" required minLength={12} className="w-full rounded-lg border border-gray-300 px-3 py-2" placeholder="At least 12 characters" />
            </div>
          </div>
          <div className="mt-4 flex gap-2">
            <button type="submit" className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">{t("common.create")}</button>
            <button type="button" onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:hover:bg-gray-700">{t("common.cancel")}</button>
          </div>
        </form>
      )}

      {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>}

      {/* Search + Batch toolbar */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="flex items-center gap-2">
          <Search className="h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder={t("users.searchPlaceholder")}
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(0); }}
            className="w-full max-w-xs rounded-lg border border-gray-300 px-3 py-2"
          />
        </div>
        {selected.size > 0 && (
          <div className="flex items-center gap-2 rounded-lg border border-amber-300 bg-amber-50 px-3 py-1.5">
            <span className="text-sm font-medium text-amber-800">{selected.size} {t("users.selected")}</span>
            <select
              value={batchRole}
              onChange={(e) => setBatchRole(e.target.value)}
              className="rounded border border-gray-300 px-2 py-1 text-xs"
            >
              <option value="">{t("users.assignRole")}</option>
              {roles.map((r) => (
                <option key={r.id} value={r.id}>{r.name || r.key}</option>
              ))}
            </select>
            <button onClick={handleBatchAssignRole} disabled={!batchRole} className="flex items-center gap-1 rounded bg-brand-600 px-2 py-1 text-xs text-white disabled:opacity-50">
              <Shield className="h-3 w-3" /> {t("users.assign")}
            </button>
            <button onClick={handleBatchDelete} className="flex items-center gap-1 rounded bg-red-600 px-2 py-1 text-xs text-white">
              <Trash2 className="h-3 w-3" /> {t("common.delete")}
            </button>
          </div>
        )}
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
        <table className="w-full">
          <thead className="border-b border-gray-200 bg-gray-50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left">
                <input
                  type="checkbox"
                  checked={selected.size === paginated.length && paginated.length > 0}
                  onChange={toggleSelectAll}
                  className="rounded"
                />
              </th>
              <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("users.userCol")}</th>
              <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.status")}</th>
              <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("users.sync")}</th>
              <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.created")}</th>
              <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{t("common.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {loading ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">{t("common.loading")}</td></tr>
            ) : paginated.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center">
                <div className="flex flex-col items-center gap-2">
                  <p className="text-gray-500 dark:text-gray-400">{t("users.noUsers")}</p>
                  <p className="text-xs text-gray-400 dark:text-gray-500">{t("users.noUsersHint")}</p>
                </div>
              </td></tr>
            ) : (
              paginated.map((user) => {
                const scimBadge = getScimBadge(user);
                return (
                  <tr key={user.id} className={`hover:bg-gray-50 ${selected.has(user.id) ? "bg-blue-50/40" : ""}`}>
                    <td className="px-4 py-3">
                      <input
                        type="checkbox"
                        checked={selected.has(user.id)}
                        onChange={() => toggleSelect(user.id)}
                        className="rounded"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <Link href={`/users/${user.id}`} className="flex items-center gap-3">
                        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium uppercase">
                          {user.username[0]}
                        </div>
                        <div>
                          <p className="text-sm font-medium hover:text-brand-600">{user.username}</p>
                          <p className="text-xs text-gray-500">{user.email}</p>
                        </div>
                      </Link>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                        user.status === "active" ? "bg-green-100 text-green-700" : user.status === "locked" ? "bg-red-100 text-red-700" : "bg-gray-100 text-gray-600"
                      }`}>
                        {user.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {scimBadge || <span className="text-xs text-gray-300">—</span>}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500">
                      {user.created_at ? new Date(user.created_at).toLocaleDateString() : "-"}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        {user.status === "active" ? (
                          <button onClick={() => handleLock(user.id, user.status)} title={t("users.lock")} className="rounded p-1.5 text-gray-400 hover:bg-gray-100">
                            <Lock className="h-4 w-4" />
                          </button>
                        ) : (
                          <button onClick={() => handleLock(user.id, user.status)} title={t("users.unlock")} className="rounded p-1.5 text-gray-400 hover:bg-gray-100">
                            <Unlock className="h-4 w-4" />
                          </button>
                        )}
                        <button onClick={() => handleDelete(user.id, user.username)} title={t("common.delete")} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <p className="text-sm text-gray-500">
            {t("users.showing")} {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, filtered.length)} / {filtered.length}
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => setPage(Math.max(0, page - 1))}
              disabled={page === 0}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm disabled:opacity-50"
            >
              <ChevronLeft className="h-4 w-4" /> {t("users.prev")}
            </button>
            <span className="flex items-center px-3 text-sm text-gray-500">
              {page + 1} / {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
              disabled={page >= totalPages - 1}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm disabled:opacity-50"
            >
              {t("users.next")} <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
