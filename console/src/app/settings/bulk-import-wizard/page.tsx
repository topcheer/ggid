"use client";

import { useState, useCallback, useRef } from "react";
import {
  Upload, FileText, ChevronRight, ChevronLeft, Check, Loader2, X,
  AlertCircle, Download, Eye, ArrowRight, Lock, Shield, Zap,
  CheckCircle, XCircle, AlertTriangle, RefreshCw,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ParsedRow { [key: string]: string }
interface ImportProgress {
  total: number; processed: number; succeeded: number; failed: number;
  status: "idle" | "running" | "completed" | "failed";
  errors: { row: number; field: string; error: string; suggestion: string }[];
}

const STEPS = ["Upload", "Field Mapping", "Role Mapping", "Password Hash", "Preview", "Import"];
const GGID_FIELDS = ["username", "email", "first_name", "last_name", "display_name", "phone", "department", "title", "employee_id", "status"];
const HASH_ALGOS = [
  { id: "argon2id", name: "Argon2id", desc: "Recommended. GGID native format.", recommended: true },
  { id: "bcrypt", name: "bcrypt", desc: "Cost factor 10-12. Transparent re-hash on login." },
  { id: "pbkdf2", name: "PBKDF2", desc: "Iterations 10000+. Common in AWS Cognito." },
  { id: "scrypt", name: "scrypt", desc: "N=16384. Used by Auth0." },
  { id: "ssha", name: "LDAP SSHA", desc: "OpenLDAP/389DS salted SHA." },
  { id: "plain", name: "Plain Text", desc: "NOT recommended. Will be hashed immediately.", warning: true },
];
const PLAYBOOKS = [
  { id: "auth0", name: "Auth0 → GGID", mapping: { user_id: "username", email: "email", given_name: "first_name", family_name: "last_name", nickname: "display_name", phone_number: "phone" }, hash: "scrypt" },
  { id: "keycloak", name: "Keycloak → GGID", mapping: { username: "username", email: "email", firstName: "first_name", lastName: "last_name", attributes_phone: "phone" }, hash: "pbkdf2" },
  { id: "ldap", name: "LDAP → GGID", mapping: { uid: "username", mail: "email", givenName: "first_name", sn: "last_name", cn: "display_name", telephoneNumber: "phone" }, hash: "ssha" },
];

const ROLE_PRESETS = [
  { source_value: "admin", ggid_role: "admin" },
  { source_value: "manager", ggid_role: "manager" },
  { source_value: "user", ggid_role: "user" },
  { source_value: "viewer", ggid_role: "viewer" },
];

export default function BulkImportWizard() {
  const t = useTranslations();
  const [step, setStep] = useState(0);
  const [fileName, setFileName] = useState("");
  const [fileContent, setFileContent] = useState("");
  const [parsedHeaders, setParsedHeaders] = useState<string[]>([]);
  const [parsedRows, setParsedRows] = useState<ParsedRow[]>([]);
  const [fieldMapping, setFieldMapping] = useState<Record<string, string>>({});
  const [roleField, setRoleField] = useState("");
  const [roleMapping, setRoleMapping] = useState<Record<string, string>>({...ROLE_PRESETS.reduce((a, r) => ({...a, [r.source_value]: r.ggid_role}), {})});
  const [hashAlgo, setHashAlgo] = useState("argon2id");
  const [progress, setProgress] = useState<ImportProgress>({ total: 0, processed: 0, succeeded: 0, failed: 0, status: "idle", errors: [] });
  const [error, setError] = useState<string | null>(null);
  const [importing, setImporting] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const handleFile = (file: File) => {
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = () => {
      const content = reader.result as string;
      setFileContent(content);
      // Parse CSV
      const lines = content.split("\n").filter(l => l.trim());
      if (lines.length === 0) return;
      const headers = lines[0].split(",").map(h => h.trim().replace(/"/g, ""));
      setParsedHeaders(headers);
      const rows = lines.slice(1, 11).map(line => {
        const vals = line.split(",").map(v => v.trim().replace(/"/g, ""));
        const row: ParsedRow = {};
        headers.forEach((h: any, i: number) => { row[h] = vals[i] || ""; });
        return row;
      });
      setParsedRows(rows);
      // Auto-map fields
      const autoMap: Record<string, string> = {};
      headers.forEach(h => {
        const lower = h.toLowerCase();
        const match = GGID_FIELDS.find(f => lower.includes(f) || f.includes(lower));
        if (match) autoMap[h] = match;
      });
      setFieldMapping(autoMap);
      // Auto-detect role field
      const roleCandidate = headers.find(h => /role|group|type/i.test(h));
      if (roleCandidate) setRoleField(roleCandidate);
    };
    reader.readAsText(file);
  };

  const applyPlaybook = (pbId: string) => {
    const pb = PLAYBOOKS.find(p => p.id === pbId);
    if (!pb) return;
    const mapping: Record<string, string> = {};
    Object.entries(pb.mapping).forEach(([srcField, ggidField]) => {
      const matched = parsedHeaders.find(h => h.toLowerCase().includes(srcField.toLowerCase()));
      if (matched) mapping[matched] = ggidField;
    });
    setFieldMapping(mapping);
    setHashAlgo(pb.hash);
  };

  const runDryRun = async () => {
    setProgress({ ...progress, status: "running", total: parsedRows.length, processed: 0, succeeded: 0, failed: 0, errors: [] });
    // Simulate validation
    let succeeded = 0; const errors: ImportProgress["errors"] = [];
    for (let i = 0; i < parsedRows.length; i++) {
      await new Promise(r => setTimeout(r, 200));
      const row = parsedRows[i];
      const email = row[Object.keys(fieldMapping).find(k => fieldMapping[k] === "email") || ""] || "";
      if (!email.includes("@")) {
        errors.push({ row: i + 2, field: "email", error: "Invalid email format", suggestion: "Ensure email field contains @ symbol" });
      } else { succeeded++; }
      setProgress(p => ({ ...p, processed: i + 1, succeeded, failed: errors.length, errors }));
    }
    setProgress(p => ({ ...p, status: "completed" }));
  };

  const executeImport = async () => {
    setImporting(true);
    try {
      const res = await fetch("/api/v1/identity/users/bulk-import", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ file_content: fileContent, field_mapping: fieldMapping, role_mapping: roleMapping, hash_algorithm: hashAlgo, dry_run: false }),
      });
      if (res.ok) { const d = await res.json(); setProgress(d); }
      else { setError("Import failed"); }
    } catch { setError("Network error"); }
    finally { setImporting(false); }
  };

  const downloadErrors = () => {
    const csv = "row,field,error,suggestion\n" + progress.errors.map(e => `${e.row},${e.field},"${e.error}","${e.suggestion}"`).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = "import-errors.csv"; a.click();
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Upload className="h-6 w-6 text-indigo-500" /> Bulk User Import</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Migrate users from CSV/JSON with field mapping, role mapping, and password hash support.</p>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Step indicator */}
      <div className="flex items-center gap-1">
        {STEPS.map((s: any, i: number) => (
          <div key={i} className="flex items-center gap-1 flex-1">
            <div className={"flex h-8 w-8 items-center justify-center rounded-full text-xs font-bold transition " + (i < step ? "bg-green-600 text-white" : i === step ? "bg-indigo-600 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400")}>
              {i < step ? <Check className="h-4 w-4" /> : i + 1}
            </div>
            <span className={"text-xs font-medium hidden sm:block " + (i <= step ? "text-gray-900 dark:text-white" : "text-gray-400")}>{s}</span>
            {i < STEPS.length - 1 && <div className={"h-0.5 flex-1 " + (i < step ? "bg-green-600" : "bg-gray-200 dark:bg-gray-700")} />}
          </div>
        ))}
      </div>

      {/* Step content */}
      <div className={cardCls}>
        {/* STEP 0: Upload */}
        {step === 0 && (
          <div className="space-y-4">
            <div className="flex flex-wrap gap-2">
              <span className="text-sm font-medium mr-2">Quick start:</span>
              {PLAYBOOKS.map(pb => (
                <button key={pb.id} onClick={() => applyPlaybook(pb.id)} className="flex items-center gap-1 rounded-lg border border-indigo-200 px-3 py-1.5 text-xs font-medium text-indigo-700 hover:bg-indigo-50 dark:border-indigo-800 dark:text-indigo-400 dark:hover:bg-indigo-950/30"><Zap className="h-3 w-3" /> {pb.name}</button>
              ))}
            </div>
            <div className="rounded-xl border-2 border-dashed border-gray-300 p-8 text-center dark:border-gray-700"
              onDragOver={e => { e.preventDefault(); e.currentTarget.classList.add("border-indigo-500", "bg-indigo-50"); }}
              onDragLeave={e => { e.currentTarget.classList.remove("border-indigo-500", "bg-indigo-50"); }}
              onDrop={e => { e.preventDefault(); e.currentTarget.classList.remove("border-indigo-500", "bg-indigo-50"); if (e.dataTransfer.files[0]) handleFile(e.dataTransfer.files[0]); }}>
              <Upload className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-3 text-sm text-gray-500">Drag & drop CSV or JSON file here</p>
              <p className="text-xs text-gray-400">or</p>
              <button onClick={() => fileRef.current?.click()} className="mt-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">Browse Files</button>
              <input ref={fileRef} type="file" accept=".csv,.json" className="hidden" onChange={e => { if (e.target.files?.[0]) handleFile(e.target.files[0]); }} />
            </div>
            {fileName && (
              <div className="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 dark:bg-green-950/20">
                <FileText className="h-5 w-5 text-green-500" />
                <div><p className="text-sm font-medium text-green-700 dark:text-green-400">{fileName}</p><p className="text-xs text-gray-400">{parsedRows.length}+ rows detected · {parsedHeaders.length} columns</p></div>
              </div>
            )}
          </div>
        )}

        {/* STEP 1: Field Mapping */}
        {step === 1 && (
          <div className="space-y-4">
            <h2 className="text-sm font-semibold uppercase text-gray-400">Map source columns to GGID fields</h2>
            <div className="grid gap-3 sm:grid-cols-2">
              {parsedHeaders.map(header => (
                <div key={header} className="flex items-center gap-2 rounded-lg border p-3 dark:border-gray-700">
                  <span className="flex-1 font-mono text-xs text-gray-600 dark:text-gray-400">{header}</span>
                  <ArrowRight className="h-3 w-3 text-gray-400" />
                  <select aria-label={`Map ${header}`} value={fieldMapping[header] || ""} onChange={e => setFieldMapping({ ...fieldMapping, [header]: e.target.value })} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs">
                    <option value="">Skip</option>
                    {GGID_FIELDS.map(f => <option key={f} value={f}>{f}</option>)}
                  </select>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* STEP 2: Role Mapping */}
        {step === 2 && (
          <div className="space-y-4">
            <div><label className="text-sm font-medium">Role source column</label><select aria-label="Role field" value={roleField} onChange={e => setRoleField(e.target.value)} className="mt-1 block rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="">None</option>{parsedHeaders.map(h => <option key={h} value={h}>{h}</option>)}</select></div>
            {roleField && (
              <div className="space-y-2">
                <h3 className="text-sm font-semibold uppercase text-gray-400">Role Mapping</h3>
                {Object.entries(roleMapping).map(([src, dst]) => (
                  <div key={src} className="flex items-center gap-2 rounded-lg border p-2 dark:border-gray-700">
                    <span className="flex-1 font-mono text-xs">{src}</span>
                    <ArrowRight className="h-3 w-3 text-gray-400" />
                    <select aria-label={`Role ${src}`} value={dst} onChange={e => setRoleMapping({ ...roleMapping, [src]: e.target.value })} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-xs"><option>admin</option><option>manager</option><option>user</option><option>viewer</option></select>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* STEP 3: Password Hash */}
        {step === 3 && (
          <div className="space-y-4">
            <h2 className="text-sm font-semibold uppercase text-gray-400">Password Hash Algorithm</h2>
            <div className="space-y-2">
              {HASH_ALGOS.map(algo => (
                <button key={algo.id} onClick={() => setHashAlgo(algo.id)} aria-pressed={hashAlgo === algo.id} className={"flex w-full items-center gap-3 rounded-xl border-2 p-4 text-left transition " + (hashAlgo === algo.id ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 hover:border-gray-300 dark:border-gray-700")}>
                  <div className={"flex h-10 w-10 items-center justify-center rounded-lg " + (hashAlgo === algo.id ? "bg-indigo-600" : "bg-gray-200 dark:bg-gray-700")}><Lock className={"h-5 w-5 " + (hashAlgo === algo.id ? "text-white" : "text-gray-400")} /></div>
                  <div className="flex-1"><div className="flex items-center gap-2"><span className="font-medium text-gray-900 dark:text-white">{algo.name}</span>{algo.recommended && <span className="px-1.5 py-0.5 rounded text-xs bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">Recommended</span>}{algo.warning && <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">Caution</span>}</div><p className="text-xs text-gray-400">{algo.desc}</p></div>
                  {hashAlgo === algo.id && <Check className="h-5 w-5 text-indigo-500" />}
                </button>
              ))}
            </div>
            <div className="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30"><p className="text-xs text-blue-700 dark:text-blue-400"><Shield className="inline h-3 w-3 mr-1" />Passwords are transparently re-hashed to Argon2id on first login for maximum security.</p></div>
          </div>
        )}

        {/* STEP 4: Preview + Dry Run */}
        {step === 4 && (
          <div className="space-y-4">
            <h2 className="text-sm font-semibold uppercase text-gray-400">Preview & Validate</h2>
            <div className="overflow-x-auto rounded-lg border dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr>{GGID_FIELDS.filter(f => Object.values(fieldMapping).includes(f)).map(f => <th key={f} scope="col" className="px-3 py-2 text-left font-medium text-xs">{f}</th>)}</tr></thead>
                <tbody className="divide-y dark:divide-gray-800">
                  {parsedRows.slice(0, 5).map((row: any, i: number) => (
                    <tr key={i}>{GGID_FIELDS.filter(f => Object.values(fieldMapping).includes(f)).map(f => {
                      const srcHeader = Object.entries(fieldMapping).find(([, v]) => v === f)?.[0] || "";
                      return <td key={f} className="px-3 py-2 text-xs font-mono">{row[srcHeader] || "—"}</td>;
                    })}</tr>
                  ))}
                </tbody>
              </table>
            </div>
            <button onClick={runDryRun} disabled={progress.status === "running"} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{progress.status === "running" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />} Dry Run Validation</button>
            {progress.status !== "idle" && (
              <div className="rounded-lg border p-4 dark:border-gray-700">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium">{progress.processed}/{progress.total} processed</span>
                  <span className={"text-sm font-bold " + (progress.failed > 0 ? "text-yellow-600" : "text-green-600")}>{progress.succeeded} ok · {progress.failed} errors</span>
                </div>
                <div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={"h-full rounded-full transition-all " + (progress.failed > 0 ? "bg-yellow-500" : "bg-green-500")} style={{ width: `${progress.total ? (progress.processed / progress.total) * 100 : 0}%` }} /></div>
                {progress.errors.length > 0 && (
                  <div className="mt-3"><div className="flex items-center justify-between"><span className="text-xs font-semibold uppercase text-gray-400">Errors</span><button onClick={downloadErrors} className="flex items-center gap-1 text-xs text-indigo-600 hover:underline"><Download className="h-3 w-3" /> Download CSV</button></div>
                    <div className="mt-2 max-h-32 overflow-y-auto space-y-1">{progress.errors.map((e: any, i: number) => <div key={i} className="flex items-center gap-2 text-xs"><XCircle className="h-3 w-3 text-red-500" /><span>Row {e.row}: <span className="text-red-600">{e.error}</span> — <span className="text-gray-400">{e.suggestion}</span></span></div>)}</div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* STEP 5: Execute Import */}
        {step === 5 && (
          <div className="space-y-4 text-center">
            <div className="mx-auto h-16 w-16 rounded-full bg-indigo-100 dark:bg-indigo-950/30 flex items-center justify-center"><Upload className="h-8 w-8 text-indigo-600" /></div>
            <h2 className="text-lg font-semibold">Ready to Import</h2>
            <div className="mx-auto max-w-sm rounded-lg border p-4 text-left dark:border-gray-700 space-y-1 text-sm">
              <div><span className="text-gray-400">File:</span> {fileName}</div>
              <div><span className="text-gray-400">Rows:</span> {parsedRows.length}+</div>
              <div><span className="text-gray-400">Mapped fields:</span> {Object.keys(fieldMapping).filter(k => fieldMapping[k]).length}</div>
              <div><span className="text-gray-400">Hash:</span> {HASH_ALGOS.find(a => a.id === hashAlgo)?.name}</div>
              <div><span className="text-gray-400">Role field:</span> {roleField || "none"}</div>
            </div>
            <button onClick={executeImport} disabled={importing} className="flex items-center gap-2 rounded-lg bg-green-600 px-6 py-3 text-sm font-bold text-white hover:bg-green-700 disabled:opacity-50 mx-auto">{importing ? <Loader2 className="h-5 w-5 animate-spin" /> : <CheckCircle className="h-5 w-5" />} Execute Import</button>
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="flex justify-between">
        <button onClick={() => setStep(Math.max(0, step - 1))} disabled={step === 0} className="flex items-center gap-1 rounded-lg border border-gray-300 px-4 py-2 text-sm disabled:opacity-30 dark:border-gray-700"><ChevronLeft className="h-4 w-4" /> Back</button>
        {step < STEPS.length - 1 && <button onClick={() => setStep(step + 1)} disabled={step === 0 && !fileName} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">Next <ChevronRight className="h-4 w-4" /></button>}
      </div>
    </div>
  );
}
