"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Gauge, Plus, Trash2, X, AlertCircle, Loader2, Check, Pencil,
  RefreshCw, ToggleLeft, ToggleRight,
} from "lucide-react";

interface RateLimit {
  id: string;
  path_pattern: string;
  method: string;
  requests_per_minute: number;
  burst: number;
  per_tenant: boolean;
  enabled: boolean;
}

export default function RateLimitsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [limits, setLimits] = useState<RateLimit[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);
  const [draft, setDraft] = useState<Partial<RateLimit>>({});

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ limits?: RateLimit[]; items?: RateLimit[] }>("/api/v1/policy/rate-limits").catch(() => null);
      setLimits(data?.limits ?? data?.items ?? []);
    } catch {
      setError(t("settings.failedLoadRateLimits"));
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const startEdit = (l: RateLimit) => {
    setEditing(l.id);
    setDraft({ ...l });
  };

  const saveEdit = async () => {
    if (!editing || !draft) return;
    try {
      await apiFetch(`/api/v1/policy/rate-limits/${editing}`, {
        method: "PATCH", body: JSON.stringify(draft),
      });
      setEditing(null);
      setDraft({});
      await load();
    } catch {
      setError(t("settings.failedSaveRateLimit"));
    }
  };

  const handleCreate = async () => {
    if (!draft.path_pattern) return;
    try {
      await apiFetch("/api/v1/policy/rate-limits", {
        method: "POST", body: JSON.stringify({
          path_pattern: draft.path_pattern,
          method: draft.method || "GET",
          requests_per_minute: draft.requests_per_minute ?? 60,
          burst: draft.burst ?? 10,
          per_tenant: draft.per_tenant ?? false,
        }),
      });
      setShowCreate(false);
      setDraft({});
      await load();
    } catch {
      setError(t("settings.failedCreateRateLimit"));
    }
  };

  const handleToggle = async (l: RateLimit) => {
    try {
      await apiFetch(`/api/v1/policy/rate-limits/${l.id}`, {
        method: "PATCH", body: JSON.stringify({ enabled: !l.enabled }),
      });
      await load();
    } catch {
      setError(t("settings.failedToggleRateLimit"));
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await apiFetch(`/api/v1/policy/rate-limits/${id}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError(t("settings.failedDeleteRateLimit"));
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Gauge className="h-6 w-6 text-indigo-600" /> {t("rateLimits.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("rateLimits.subtitle")}
          </p>
        </div>
        <button onClick={() => { setDraft({ method: "GET", requests_per_minute: 60, burst: 10, per_tenant: false }); setShowCreate(true); }} aria-label="Create new rate limit" className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
          <Plus className="h-4 w-4" /> {t("rateLimits.addLimit")}
        </button>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : limits.length === 0 ? (
        <div className={cardCls}>
          <div className="py-12 text-center">
            <Gauge className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-4 text-sm text-gray-400">{t("rateLimits.noLimits")}</p>
          </div>
        </div>
      ) : (
        <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr className="text-left text-xs font-semibold uppercase text-gray-500">
                <th scope="col" className="px-4 py-3">{t("rateLimits.pathPattern")}</th>
                <th scope="col" className="px-4 py-3">{t("rateLimits.method")}</th>
                <th scope="col" className="px-4 py-3">{t("rateLimits.reqMin")}</th>
                <th scope="col" className="px-4 py-3">{t("rateLimits.burst")}</th>
                <th scope="col" className="px-4 py-3">{t("rateLimits.perTenant")}</th>
                <th scope="col" className="px-4 py-3">{t("settings.enabled")}</th>
                <th scope="col" className="px-4 py-3 text-right">{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {limits.map((l) => (
                <tr key={l.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                  {editing === l.id ? (
                    <>
                      <td className="px-4 py-2"><input value={draft.path_pattern ?? ""} onChange={(e) => setDraft((p) => ({ ...p, path_pattern: e.target.value }))} className={inputCls} /></td>
                      <td className="px-4 py-2">
                        <select value={draft.method ?? "GET"} onChange={(e) => setDraft((p) => ({ ...p, method: e.target.value }))} className={inputCls}>
                          {["GET", "POST", "PUT", "PATCH", "DELETE", "*"].map((m) => <option key={m} value={m}>{m}</option>)}
                        </select>
                      </td>
                      <td className="px-4 py-2"><input type="number" value={draft.requests_per_minute ?? 60} onChange={(e) => setDraft((p) => ({ ...p, requests_per_minute: Number(e.target.value) }))} className={inputCls} /></td>
                      <td className="px-4 py-2"><input type="number" value={draft.burst ?? 10} onChange={(e) => setDraft((p) => ({ ...p, burst: Number(e.target.value) }))} className={inputCls} /></td>
                      <td className="px-4 py-2"><input type="checkbox" checked={draft.per_tenant ?? false} onChange={(e) => setDraft((p) => ({ ...p, per_tenant: e.target.checked }))} className="rounded border-gray-300 text-indigo-600" /></td>
                      <td className="px-4 py-2"><input type="checkbox" checked={draft.enabled ?? true} onChange={(e) => setDraft((p) => ({ ...p, enabled: e.target.checked }))} className="rounded border-gray-300 text-indigo-600" /></td>
                      <td className="px-4 py-2">
                        <div className="flex justify-end gap-1">
                          <button onClick={saveEdit} aria-label="Save edit" className="rounded-lg bg-green-600 px-2 py-1 text-xs text-white hover:bg-green-700"><Check className="h-3.5 w-3.5" /></button>
                          <button onClick={() => { setEditing(null); setDraft({}); }} aria-label="Cancel edit" className="rounded-lg px-2 py-1 text-xs text-gray-400"><X className="h-3.5 w-3.5" /></button>
                        </div>
                      </td>
                    </>
                  ) : (
                    <>
                      <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{l.path_pattern}</td>
                      <td className="px-4 py-3"><span className="rounded bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{l.method}</span></td>
                      <td className="px-4 py-3 font-medium text-gray-700 dark:text-gray-300">{l.requests_per_minute}</td>
                      <td className="px-4 py-3 text-gray-500">{l.burst}</td>
                      <td className="px-4 py-3">{l.per_tenant ? <ToggleRight className="h-5 w-5 text-indigo-600" /> : <ToggleLeft className="h-5 w-5 text-gray-300" />}</td>
                      <td className="px-4 py-3">
                        <button onClick={() => handleToggle(l)}>
                          <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${l.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>{l.enabled ? t("rateLimits.on") : t("rateLimits.off")}</span>
                        </button>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex justify-end gap-1">
                          <button onClick={() => startEdit(l)} aria-label={"Edit rate limit " + l.id} className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Pencil className="h-4 w-4" /></button>
                          <button onClick={() => setConfirmDelete(l.id)} aria-label={"Delete rate limit " + l.id} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button>
                        </div>
                      </td>
                    </>
                  )}
                </tr>
              ))}
              {showCreate && (
                <tr className="bg-indigo-50/50 dark:bg-indigo-900/10">
                  <td className="px-4 py-2"><input value={draft.path_pattern ?? ""} onChange={(e) => setDraft((p) => ({ ...p, path_pattern: e.target.value }))} placeholder="/api/v1/*" className={inputCls} /></td>
                  <td className="px-4 py-2">
                    <select value={draft.method ?? "GET"} onChange={(e) => setDraft((p) => ({ ...p, method: e.target.value }))} className={inputCls}>
                      {["GET", "POST", "PUT", "PATCH", "DELETE", "*"].map((m) => <option key={m} value={m}>{m}</option>)}
                    </select>
                  </td>
                  <td className="px-4 py-2"><input type="number" value={draft.requests_per_minute ?? 60} onChange={(e) => setDraft((p) => ({ ...p, requests_per_minute: Number(e.target.value) }))} className={inputCls} /></td>
                  <td className="px-4 py-2"><input type="number" value={draft.burst ?? 10} onChange={(e) => setDraft((p) => ({ ...p, burst: Number(e.target.value) }))} className={inputCls} /></td>
                  <td className="px-4 py-2"><input type="checkbox" checked={draft.per_tenant ?? false} onChange={(e) => setDraft((p) => ({ ...p, per_tenant: e.target.checked }))} className="rounded border-gray-300 text-indigo-600" /></td>
                  <td className="px-4 py-2" />
                  <td className="px-4 py-2">
                    <div className="flex justify-end gap-1">
                      <button onClick={handleCreate} disabled={!draft.path_pattern} aria-label="Create rate limit" className="rounded-lg bg-green-600 px-2 py-1 text-xs text-white hover:bg-green-700 disabled:opacity-50"><Check className="h-3.5 w-3.5" /></button>
                      <button onClick={() => { setShowCreate(false); setDraft({}); }} aria-label="Cancel create" className="rounded-lg px-2 py-1 text-xs text-gray-400"><X className="h-3.5 w-3.5" /></button>
                    </div>
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Mobile cards */}
      {!loading && limits.length > 0 && (
        <div className="space-y-3 md:hidden">
          {limits.map((l) => (
            <div key={l.id} className={cardCls}>
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs text-gray-700 dark:text-gray-300">{l.path_pattern}</span>
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${l.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>{l.enabled ? t("rateLimits.on") : t("rateLimits.off")}</span>
              </div>
              <div className="mt-2 flex items-center gap-4 text-xs text-gray-400">
                <span className="rounded bg-blue-100 px-1.5 py-0.5 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{l.method}</span>
                <span>{l.requests_per_minute} req/min</span>
                <span>{l.burst} burst</span>
              </div>
              <div className="mt-2 flex gap-2">
                <button onClick={() => startEdit(l)} aria-label={"Edit " + l.path_pattern} className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-600"><Pencil className="h-3 w-3" /> {t("common.edit")}</button>
                <button onClick={() => setConfirmDelete(l.id)} aria-label={"Delete " + l.path_pattern} className="flex items-center gap-1 rounded-lg border border-red-200 px-2 py-1 text-xs text-red-500"><Trash2 className="h-3 w-3" /> {t("common.delete")}</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">{t("rateLimits.deleteTitle")}</h2>
                <p className="text-sm text-gray-500">{t("rateLimits.deleteDesc")}</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmDelete(null)} aria-label="Cancel delete" className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("common.cancel")}</button>
              <button onClick={() => handleDelete(confirmDelete)} aria-label="Confirm delete" className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{t("common.delete")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
