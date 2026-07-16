"use client";
import { useState, useEffect, useCallback } from "react";
import { Key, Plus, X, RotateCcw, Ban, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface ApiKey { id: string; name: string; scopes: string[]; created_at: string; expires_at: string; last_used: string | null; status: "active" | "expired" | "revoked"; usage_count: number; }
const statusColors: Record<string, string> = { active: "bg-green-100 dark:bg-green-900/30 dark:text-green-400", expired: "bg-gray-100 dark:bg-gray-800 dark:text-gray-400", revoked: "bg-red-100 dark:bg-red-900/30 dark:text-red-400" };
export default function APIKeyLifecyclePage() {
  const t = useTranslations();

  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: "", scopes: "", expires_at: "" });
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/api-keys", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) { setKeys([]); return; }
      const d = await res.json();
      setKeys(d.keys || d || []);
    } catch { setKeys([]); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const create = async () => {
    if (!form.name) return;
    try {
      const res = await fetch("/api/v1/auth/api-keys", { method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ name: form.name, scopes: form.scopes.split(",").map((s) => s.trim()).filter(Boolean), expires_at: form.expires_at }) });
      if (!res.ok) { setError("API key creation not available yet"); return; }
      setShowCreate(false); setForm({ name: "", scopes: "", expires_at: "" }); fetchData();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to create API key"); }
  };
  const rotate = async (id: string) => {
    try {
      const res = await fetch("/api/v1/auth/api-keys/" + id + "/rotate", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return;
      fetchData();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to rotate key"); }
  };
  const revoke = async (id: string) => {
    try {
      const res = await fetch("/api/v1/auth/api-keys/" + id, { method: "DELETE", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return;
      fetchData();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to revoke key"); }
  };
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Key className="w-6 h-6 text-blue-500" /> {t("apiKeyLifecycle.title")}</h1><p className="text-sm text-gray-500 mt-1">Manage API keys: create, rotate, revoke.</p></div>
        <button onClick={() => { setShowCreate(true); setError(null); }} aria-label="Create new API key" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Plus className="w-4 h-4" /> Create Key</button>
      </div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={() => { setError(null); fetchData(); }} className="text-xs underline hover:text-red-700">Retry</button></div>}
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading API keys...</div></div>}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr><th className="px-4 py-3 text-left font-medium">Name</th><th className="px-4 py-3 text-left font-medium">Scopes</th><th className="px-4 py-3 text-left font-medium">Created</th><th className="px-4 py-3 text-left font-medium">Expires</th><th className="px-4 py-3 text-left font-medium">Last Used</th><th className="px-4 py-3 text-left font-medium">Usage</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Actions</th></tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {keys.map((k) => (
              <tr key={k.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-medium">{k.name}</td><td className="px-4 py-3"><div className="flex flex-wrap gap-1 max-w-32">{k.scopes.map((s) => <span key={s} className="px-1 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{s}</span>)}</div></td><td className="px-4 py-3 text-xs text-gray-500">{k.created_at}</td><td className="px-4 py-3 text-xs text-gray-500">{k.expires_at}</td><td className="px-4 py-3 text-xs text-gray-500">{k.last_used || "-"}</td><td className="px-4 py-3 text-xs font-bold">{k.usage_count.toLocaleString()}</td>
                <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + statusColors[k.status]}>{k.status}</span></td>
                <td className="px-4 py-3">
                  <div className="flex gap-2">
                    {k.status === "active" && <>
                      <button onClick={() => rotate(k.id)} aria-label={`Rotate key ${k.name}`} className="text-xs text-blue-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> Rotate</button>
                      <button onClick={() => revoke(k.id)} aria-label={`Revoke key ${k.name}`} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Ban className="w-3 h-3" /> Revoke</button>
                    </>}
                  </div>
                </td>
              </tr>
            ))}
            {keys.length === 0 && !loading && <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No API keys.</td></tr>}
          </tbody>
        </table>
      </div>
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">Create API Key</h3><button onClick={() => setShowCreate(false)} aria-label="Close dialog" className="text-gray-400"><X className="w-5 h-5" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">Name</label><input type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} aria-label="API key name" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">Scopes (comma-separated)</label><input type="text" value={form.scopes} onChange={(e) => setForm({ ...form, scopes: e.target.value })} placeholder="read:users, write:roles" aria-label="API key scopes" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Expires At</label><input type="date" value={form.expires_at} onChange={(e) => setForm({ ...form, expires_at: e.target.value })} aria-label="API key expiration" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={create} disabled={!form.name} aria-label="Create API key" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50">Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
