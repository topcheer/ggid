"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { AlertTriangle, Loader2, AlertCircle, X, Calendar, FileText, Eye } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface DeprecationStatus {
  id: string; client_id: string; client_name: string;
  status: "active" | "deprecated" | "sunset" | "retired";
  sunset_date: string; migration_guide: string;
  warning_message: string; deprecation_date: string;
  active_tokens: number; affected_users: number;
}

const statusColors: Record<string, string> = {
  active: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  deprecated: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  sunset: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  retired: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function ClientDeprecationPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [clients, setClients] = useState<DeprecationStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [preview, setPreview] = useState<DeprecationStatus | null>(null);

  useEffect(() => { (async () => { try { setClients(await apiFetch<DeprecationStatus[]>("/api/v1/oauth/client-deprecation").catch(() => [])); } catch { setError("Failed to load deprecation data"); } finally { setLoading(false); } })(); });

  const handleUpdateStatus = async (id: string, status: DeprecationStatus["status"]) => {
    try { await apiFetch(`/api/v1/oauth/client-deprecation/${id}`, { method: "PATCH", body: JSON.stringify({ status }) }); setClients((p) => p.map((c) => c.id === id ? { ...c, status } : c)); } catch { setError("Update failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const deprecated = clients.filter((c) => c.status !== "active");

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><AlertTriangle className="h-6 w-6 text-yellow-600" /> {t("clientDeprecation.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage OAuth client lifecycle deprecation, sunset dates, and migration guides.</p></div>
      {deprecated.length > 0 && <div className="flex items-center gap-3 rounded-xl border border-yellow-200 bg-yellow-50 px-4 py-3 dark:border-yellow-800 dark:bg-yellow-900/20"><AlertTriangle className="h-5 w-5 text-yellow-600 shrink-0" /><span className="text-sm text-yellow-700 dark:text-yellow-400">{deprecated.length} client{deprecated.length > 1 ? "s" : ""} in deprecation cycle.</span></div>}
      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-yellow-600" /></div> : clients.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><AlertTriangle className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No clients to manage.</p></div></div> : (
        <div className="space-y-3">{clients.map((c) => (
          <div key={c.id} className={`${cardCls} ${c.status !== "active" ? "border-l-4 border-l-yellow-400" : ""}`}>
            <div className="flex items-start justify-between">
              <div className="flex-1"><div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{c.client_name}</span><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[c.status]}`}>{c.status}</span></div>
                <div className="mt-2 flex flex-wrap gap-4 text-xs text-gray-400"><span className="flex items-center gap-1"><Calendar className="h-3 w-3" />Sunset: {c.sunset_date ? new Date(c.sunset_date).toLocaleDateString() : "—"}</span><span>Active tokens: {c.active_tokens}</span><span>Affected users: {c.affected_users}</span>{c.deprecation_date && <span>Deprecated: {new Date(c.deprecation_date).toLocaleDateString()}</span>}</div>
                {c.migration_guide && <div className="mt-2 flex items-center gap-1 text-xs text-blue-600"><FileText className="h-3 w-3" />{c.migration_guide.slice(0, 80)}</div>}
              </div>
              <div className="flex items-center gap-2"><button onClick={() => setPreview(c)} className="text-xs text-indigo-600 hover:underline"><Eye className="inline h-3 w-3" /> Warning</button><select aria-label="Select option" value={c.status} onChange={(e) => handleUpdateStatus(c.id, e.target.value as DeprecationStatus["status"])} className="rounded border border-gray-300 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200"><option value="active">Active</option><option value="deprecated">Deprecated</option><option value="sunset">Sunset</option><option value="retired">Retired</option></select></div>
            </div>
          </div>
        ))}</div>
      )}
      {/* Warning preview modal */}
      {preview && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setPreview(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="flex items-center gap-2 text-lg font-bold text-gray-900 dark:text-white"><AlertTriangle className="h-5 w-5 text-yellow-600" /> Deprecation Warning Preview</h3><button onClick={() => setPreview(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="rounded-lg border-2 border-dashed border-yellow-300 bg-yellow-50 p-4 dark:border-yellow-700 dark:bg-yellow-900/20">
              <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-yellow-600" /><span className="font-semibold text-yellow-700 dark:text-yellow-400">Deprecation Notice</span></div>
              <p className="mt-2 text-sm text-yellow-700 dark:text-yellow-300">{preview.warning_message || `Client "${preview.client_name}" is being deprecated. Please migrate to the new API by ${preview.sunset_date ? new Date(preview.sunset_date).toLocaleDateString() : "the specified date"}.`}</p>
              {preview.migration_guide && <p className="mt-2 text-xs text-yellow-600">Migration guide: {preview.migration_guide}</p>}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
