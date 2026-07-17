"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  FileCode, Loader2, AlertCircle, X, Upload, Download, Play, Trash2, GitCompare,
} from "lucide-react";

interface PolicyFile {
  id: string;
  name: string;
  version: string;
  yaml: string;
  status: "active" | "draft" | "archived";
  updated_at: string;
}

export default function PolicyAsCodePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [policies, setPolicies] = useState<PolicyFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<{ id: string; name: string; yaml: string } | null>(null);
  const [diff, setDiff] = useState<string | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importYaml, setImportYaml] = useState("");
  const [showImport, setShowImport] = useState(false);

  useEffect(() => {
    (async () => {
      try { setPolicies(await apiFetch<PolicyFile[]>("/api/v1/policy/as-code").catch(() => [])); }
      catch { setError("Failed to load policies"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handlePreviewDiff = async () => {
    if (!editing) return;
    setPreviewing(true);
    try { const result = await apiFetch<{ diff: string }>(`/api/v1/policy/as-code/${editing.id}/diff`, { method: "POST", body: JSON.stringify({ yaml: editing.yaml }) }); setDiff(result.diff || "No changes detected."); }
    catch { setError("Diff failed"); }
    finally { setPreviewing(false); }
  };

  const handleImport = async () => {
    if (!importYaml.trim()) return;
    setImporting(true);
    try { await apiFetch("/api/v1/policy/as-code/import", { method: "POST", body: JSON.stringify({ yaml: importYaml, name: "imported-policy" }) }); setPolicies(await apiFetch<PolicyFile[]>("/api/v1/policy/as-code").catch(() => policies)); setShowImport(false); setImportYaml(""); }
    catch { setError("Import failed"); }
    finally { setImporting(false); }
  };

  const handleExport = async (id: string) => {
    try { const data = await apiFetch<{ yaml: string }>(`/api/v1/policy/as-code/${id}/export`); const blob = new Blob([data.yaml], { type: "text/yaml" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = `${id}.yaml`; a.click(); URL.revokeObjectURL(url); }
    catch { setError("Export failed"); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/policy/as-code/${id}`, { method: "DELETE" }); setPolicies((p) => p.filter((x) => x.id !== id)); }
    catch { setError("Delete failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><FileCode className="h-6 w-6 text-emerald-600" />{t("policyAsCode.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Define, version, and deploy access policies as YAML.</p>
        </div>
        <button onClick={() => setShowImport(true)} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"><Upload className="h-4 w-4" /> Import YAML</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-600" /></div>
      : (
        <div className="space-y-3">
          {policies.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><FileCode className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No policies defined yet.</p></div></div>
          ) : policies.map((p) => (
            <div key={p.id} className={cardCls}>
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{p.name}</span><span className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-gray-700">v{p.version}</span><span className={`rounded-full px-2 py-0.5 text-xs font-medium ${p.status === "active" ? "bg-green-100 text-green-700 dark:bg-green-900/30" : p.status === "draft" ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30" : "bg-gray-100 text-gray-500"}`}>{p.status}</span></div>
                  <p className="mt-1 text-xs text-gray-400">Updated {new Date(p.updated_at).toLocaleDateString()}</p>
                </div>
                <div className="flex gap-1">
                  <button onClick={() => { setEditing({ id: p.id, name: p.name, yaml: p.yaml }); setDiff(null); }} className="rounded p-1.5 text-gray-400 hover:text-emerald-600"><FileCode className="h-4 w-4" /></button>
                  <button onClick={() => handleExport(p.id)} className="rounded p-1.5 text-gray-400 hover:text-emerald-600"><Download className="h-4 w-4" /></button>
                  <button onClick={() => handleDelete(p.id)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"><Trash2 className="h-4 w-4" /></button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Import modal */}
      {showImport && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowImport(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-2xl rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="text-lg font-bold text-gray-900 dark:text-white">Import YAML Policy</h3><button onClick={() => setShowImport(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <textarea aria-label="Import yaml" value={importYaml} onChange={(e) => setImportYaml(e.target.value)} rows={12} placeholder={"name: my-policy\neffect: allow\nconditions:\n  - attribute: role\n    operator: eq\n    value: admin"} className="w-full rounded-lg border border-gray-300 p-3 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
            <button onClick={handleImport} disabled={!importYaml.trim() || importing} className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-emerald-600 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50">{importing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />} Import</button>
          </div>
        </div>
      )}

      {/* Edit + diff modal */}
      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditing(null)}>
          <div role="dialog" aria-modal="true" className="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-xl bg-white shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700"><h3 className="text-lg font-bold text-gray-900 dark:text-white">{editing.name}</h3><div className="flex gap-2"><button onClick={handlePreviewDiff} disabled={previewing} className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-600 dark:border-gray-600 dark:text-gray-300">{previewing ? <Loader2 className="h-4 w-4 animate-spin" /> : <GitCompare className="h-4 w-4" />} Preview Diff</button><button onClick={() => setEditing(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div></div>
            <div className="flex flex-1 gap-4 overflow-hidden p-6">
              <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">YAML Editor</label><textarea aria-label="Text input" value={editing.yaml} onChange={(e) => setEditing({ ...editing, yaml: e.target.value })} rows={20} className="w-full rounded-lg border border-gray-300 p-3 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              {diff !== null && (<div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Diff Preview</label><pre className="w-full overflow-auto rounded-lg border border-gray-300 p-3 font-mono text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" style={{ maxHeight: "500px" }}>{diff}</pre></div>)}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
