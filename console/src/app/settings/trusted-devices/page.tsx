"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { ShieldCheck, Loader2, AlertCircle, X, Trash2, ToggleLeft, ToggleRight, Smartphone, Monitor } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TrustedDevice {
  id: string; user_id: string; username: string;
  fingerprint: string; platform: string; trusted_since: string;
  last_used: string; mfa_bypass_enabled: boolean;
}

export default function TrustedDevicesPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [devices, setDevices] = useState<TrustedDevice[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actioning, setActioning] = useState<string | null>(null);

  useState(() => { (async () => { try { setDevices(await apiFetch<TrustedDevice[]>("/api/v1/auth/trusted-devices").catch(() => [])); } catch { setError("Failed to load devices"); } finally { setLoading(false); } })(); });

  const handleRemove = async (id: string) => { setActioning(id); try { await apiFetch(`/api/v1/auth/trusted-devices/${id}`, { method: "DELETE" }); setDevices((p) => p.filter((d) => d.id !== id)); } catch { setError("Remove failed"); } finally { setActioning(null); } };
  const handleToggleMfa = async (d: TrustedDevice) => { setActioning(d.id); try { await apiFetch(`/api/v1/auth/trusted-devices/${d.id}`, { method: "PATCH", body: JSON.stringify({ mfa_bypass_enabled: !d.mfa_bypass_enabled }) }); setDevices((p) => p.map((x) => x.id === d.id ? { ...x, mfa_bypass_enabled: !x.mfa_bypass_enabled } : x)); } catch { setError("Toggle failed"); } finally { setActioning(null); } };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-green-600" /> Trusted Devices</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage devices with elevated trust and optional MFA bypass.</p></div>
      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-green-600" /></div> : devices.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><ShieldCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No trusted devices.</p></div></div> : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
          <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-800"><tr><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">User</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Fingerprint</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Platform</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Trusted Since</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Last Used</th><th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">MFA Bypass</th><th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th></tr></thead>
            <tbody className="divide-y divide-gray-200 dark:divide-gray-700">{devices.map((d) => (<tr key={d.id} className="bg-white dark:bg-gray-900"><td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{d.username}</div></td><td className="px-4 py-3 font-mono text-xs text-gray-500">{d.fingerprint.slice(0, 20)}</td><td className="px-4 py-3">{d.platform.toLowerCase().includes("android") || d.platform.toLowerCase().includes("ios") ? <Smartphone className="h-4 w-4 text-purple-500" /> : <Monitor className="h-4 w-4 text-blue-500" />}<span className="ml-1 text-gray-500">{d.platform}</span></td><td className="px-4 py-3 text-gray-400">{new Date(d.trusted_since).toLocaleDateString()}</td><td className="px-4 py-3 text-gray-400">{d.last_used ? new Date(d.last_used).toLocaleDateString() : "—"}</td><td className="px-4 py-3"><button onClick={() => handleToggleMfa(d)} disabled={actioning === d.id}>{actioning === d.id ? <Loader2 className="h-4 w-4 animate-spin" /> : d.mfa_bypass_enabled ? <ToggleRight className="h-6 w-6 text-green-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button></td><td className="px-4 py-3 text-right"><button onClick={() => handleRemove(d.id)} disabled={actioning === d.id} className="text-red-400 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></td></tr>))}</tbody>
          </table>
        </div>
      )}
    </div>
  );
}
