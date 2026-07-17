"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  RefreshCw, Loader2, AlertCircle, X, KeyRound, Clock, Save, AlertOctagon,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SecretStatus {
  id: string;
  client_id: string;
  client_name: string;
  secret_hash: string;
  created_at: string;
  last_rotated: string;
  expires_at: string;
  grace_period_days: number;
  auto_rotate: boolean;
  rotation_interval_days: number;
  status: "active" | "expiring" | "expired";
}

export default function SecretRotationPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [secrets, setSecrets] = useState<SecretStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [rotating, setRotating] = useState<string | null>(null);
  const [savingId, setSavingId] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setSecrets(await apiFetch<SecretStatus[]>("/api/v1/oauth/secret-rotation").catch(() => [])); }
      catch { setError("Failed to load secrets"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleRotate = async (id: string) => {
    setRotating(id);
    try { await apiFetch(`/api/v1/oauth/secret-rotation/${id}/rotate`, { method: "POST" }); setSecrets(await apiFetch<SecretStatus[]>("/api/v1/oauth/secret-rotation").catch(() => secrets)); }
    catch { setError("Rotation failed"); }
    finally { setRotating(null); }
  };

  const handleToggleAuto = async (s: SecretStatus) => {
    setSavingId(s.id);
    try { await apiFetch(`/api/v1/oauth/secret-rotation/${s.id}`, { method: "PATCH", body: JSON.stringify({ auto_rotate: !s.auto_rotate }) }); setSecrets((prev) => prev.map((x) => x.id === s.id ? { ...x, auto_rotate: !x.auto_rotate } : x)); }
    catch { setError("Toggle failed"); }
    finally { setSavingId(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const statusColors: Record<string, string> = { active: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400", expiring: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400", expired: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400" };
  const expiring = secrets.filter((s) => s.status !== "active");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><RefreshCw className="h-6 w-6 text-blue-600" /> {t("secretRotation.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">OAuth client secret lifecycle with grace period and auto-rotation.</p>
      </div>

      {expiring.length > 0 && <div className="flex items-center gap-3 rounded-xl border border-yellow-200 bg-yellow-50 px-4 py-3 dark:border-yellow-800 dark:bg-yellow-900/20"><AlertOctagon className="h-5 w-5 text-yellow-600 shrink-0" /><span className="text-sm text-yellow-700 dark:text-yellow-400">{expiring.length} secret{expiring.length > 1 ? "s" : ""} need rotation.</span></div>}

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-600" /></div>
      : secrets.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No secrets to track.</p></div></div>
      : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800"><tr><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Client</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Created</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Last Rotated</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Expires</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Grace Period</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Auto-Rotate</th><th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th></tr></thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">{secrets.map((s) => (
              <tr key={s.id} className="bg-white dark:bg-gray-900">
                <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{s.client_name}</div><div className="text-xs text-gray-400 font-mono">{s.client_id.slice(0, 16)}</div></td>
                <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[s.status] || ""}`}>{s.status}</span></td>
                <td className="px-4 py-3 text-gray-400">{new Date(s.created_at).toLocaleDateString()}</td>
                <td className="px-4 py-3 text-gray-400">{s.last_rotated ? new Date(s.last_rotated).toLocaleDateString() : "—"}</td>
                <td className="px-4 py-3"><span className={s.expires_at && new Date(s.expires_at) < new Date() ? "text-red-500" : "text-gray-400"}>{s.expires_at ? new Date(s.expires_at).toLocaleDateString() : "—"}</span></td>
                <td className="px-4 py-3 text-gray-500">{s.grace_period_days}d</td>
                <td className="px-4 py-3"><button onClick={() => handleToggleAuto(s)} disabled={savingId === s.id} className={`flex items-center gap-1 text-xs font-medium ${s.auto_rotate ? "text-green-600" : "text-gray-400"}`}>{savingId === s.id ? <Loader2 className="h-3 w-3 animate-spin" /> : null}{s.auto_rotate ? "On" : "Off"}</button><div className="text-xs text-gray-400">every {s.rotation_interval_days}d</div></td>
                <td className="px-4 py-3 text-right"><button onClick={() => handleRotate(s.id)} disabled={rotating === s.id} className="flex items-center gap-1 text-xs text-blue-600 hover:underline">{rotating === s.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />} Rotate</button></td>
              </tr>
            ))}</tbody>
          </table>
        </div>
      )}
    </div>
  );
}
