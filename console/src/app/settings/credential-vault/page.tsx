"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, Loader2, AlertCircle, X, Plus, Eye, EyeOff, Trash2, Save, Lock, Activity,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface StoredCredential {
  id: string;
  key: string;
  description: string;
  created_at: string;
  last_accessed: string;
  access_count: number;
  expires_at: string;
}

export default function CredentialVaultPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [creds, setCreds] = useState<StoredCredential[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showStore, setShowStore] = useState(false);
  const [revealing, setRevealing] = useState<string | null>(null);
  const [revealedValue, setRevealedValue] = useState<string | null>(null);
  const [revealedId, setRevealedId] = useState<string | null>(null);
  const [form, setForm] = useState({ key: "", value: "", description: "" });
  const [storing, setStoring] = useState(false);

  useEffect(() => {
    (async () => {
      try { setCreds(await apiFetch<StoredCredential[]>("/api/v1/auth/credential-vault").catch(() => [])); }
      catch { setError("Failed to load credentials"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleStore = async () => {
    if (!form.key || !form.value) return;
    setStoring(true);
    try { await apiFetch("/api/v1/auth/credential-vault", { method: "POST", body: JSON.stringify(form) }); setCreds(await apiFetch<StoredCredential[]>("/api/v1/auth/credential-vault").catch(() => creds)); setShowStore(false); setForm({ key: "", value: "", description: "" }); }
    catch { setError("Store failed"); }
    finally { setStoring(false); }
  };

  const handleReveal = async (id: string) => {
    if (revealedId === id) { setRevealedId(null); setRevealedValue(null); return; }
    setRevealing(id);
    try { const data = await apiFetch<{ value: string }>(`/api/v1/auth/credential-vault/${id}/reveal`, { method: "POST" }); setRevealedId(id); setRevealedValue(data.value || ""); }
    catch { setError("Reveal failed"); }
    finally { setRevealing(null); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/auth/credential-vault/${id}`, { method: "DELETE" }); setCreds((p) => p.filter((c: any) => c.id !== id)); if (revealedId === id) { setRevealedId(null); setRevealedValue(null); } }
    catch { setError("Delete failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-amber-600" /> {t("credentialVault.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Secure encrypted storage for API keys, secrets, and service credentials.</p>
        </div>
        <button onClick={() => setShowStore(true)} className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700"><Plus className="h-4 w-4" /> Store</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-amber-600" /></div>
      : creds.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No stored credentials.</p></div></div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800"><tr>
              <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Key</th>
              <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Description</th>
              <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Created</th>
              <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Last Accessed</th>
              <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Access Count</th>
              <th scope="col" className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
            </tr></thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
              {creds.map((c: any) => (
                <tr key={c.id} className="bg-white dark:bg-gray-900">
                  <td className="px-4 py-3"><div className="flex items-center gap-2"><Lock className="h-3 w-3 text-gray-400" /><span className="font-mono text-sm font-medium text-gray-900 dark:text-white">{c.key}</span></div>{revealedId === c.id && revealedValue && <div className="mt-1 rounded bg-amber-50 px-2 py-1 font-mono text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-400">{revealedValue}</div>}</td>
                  <td className="px-4 py-3 text-gray-500">{c.description || "—"}</td>
                  <td className="px-4 py-3 text-gray-400">{new Date(c.created_at).toLocaleDateString()}</td>
                  <td className="px-4 py-3 text-gray-400">{c.last_accessed ? new Date(c.last_accessed).toLocaleDateString() : "—"}</td>
                  <td className="px-4 py-3"><span className="flex items-center gap-1 text-gray-600 dark:text-gray-300"><Activity className="h-3 w-3 text-gray-400" />{c.access_count}</span></td>
                  <td className="px-4 py-3 text-right">
                    <button onClick={() => handleReveal(c.id)} disabled={revealing === c.id} className="mr-2 text-gray-400 hover:text-amber-600">{revealing === c.id ? <Loader2 className="inline h-4 w-4 animate-spin" /> : revealedId === c.id ? <EyeOff className="inline h-4 w-4" /> : <Eye className="inline h-4 w-4" />}</button>
                    <button onClick={() => handleDelete(c.id)} className="text-red-400 hover:text-red-600"><Trash2 className="inline h-4 w-4" /></button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Store modal */}
      {showStore && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowStore(false)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between"><h3 className="flex items-center gap-2 text-lg font-bold text-gray-900 dark:text-white"><Lock className="h-5 w-5 text-amber-600" /> Store Credential</h3><button onClick={() => setShowStore(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button></div>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Key Name</label><input aria-label="AWS_SECRET_KEY" value={form.key} onChange={(e) => setForm({ ...form, key: e.target.value })} placeholder="AWS_SECRET_KEY" className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Value</label><textarea aria-label="secret value..." value={form.value} onChange={(e) => setForm({ ...form, value: e.target.value })} rows={3} placeholder="secret value..." className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Description (optional)</label><input aria-label="Production AWS credentials" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="Production AWS credentials" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              <button onClick={handleStore} disabled={!form.key || !form.value || storing} className="flex w-full items-center justify-center gap-2 rounded-lg bg-amber-600 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">{storing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Store Securely</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
