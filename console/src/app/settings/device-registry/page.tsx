"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Smartphone, Loader2, AlertCircle, X, Trash2, Monitor, Tablet, HardDrive,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface DeviceEntry {
  id: string;
  fingerprint: string;
  user_id: string;
  username: string;
  platform: string;
  user_agent: string;
  last_seen: string;
  first_seen: string;
  session_count: number;
  trusted: boolean;
}

function PlatformIcon({ platform }: { platform: string }) {
  const p = platform.toLowerCase();
  if (p.includes("android") || p.includes("ios") || p.includes("iphone")) return <Smartphone className="h-4 w-4 text-purple-500" />;
  if (p.includes("ipad") || p.includes("tablet")) return <Tablet className="h-4 w-4 text-cyan-500" />;
  return <Monitor className="h-4 w-4 text-blue-500" />;
}

export default function DeviceRegistryPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [devices, setDevices] = useState<DeviceEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try { setDevices(await apiFetch<DeviceEntry[]>("/api/v1/auth/devices").catch(() => [])); }
      catch { setError("Failed to load device registry"); }
      finally { setLoading(false); }
    })();
  }, []);

  const handleRevoke = async (id: string) => {
    try { await apiFetch(`/api/v1/auth/devices/${id}`, { method: "DELETE" }); setDevices((p) => p.filter((d: any) => d.id !== id)); }
    catch { setError("Revoke failed"); }
  };

  const handleToggleTrust = async (device: DeviceEntry) => {
    try { await apiFetch(`/api/v1/auth/devices/${device.id}`, { method: "PATCH", body: JSON.stringify({ trusted: !device.trusted }) }); setDevices((p) => p.map((d: any) => d.id === device.id ? { ...d, trusted: !d.trusted } : d)); }
    catch { setError("Toggle failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><HardDrive className="h-6 w-6 text-blue-600" /> {t("big1.deviceRegistry.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("big1.deviceRegistry.allRegisteredDevicesWithFingerprintsSessionsAndTrustStatus")}</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-600" /></div>
      : (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">{t("big1.deviceRegistry.totalDevices")}</div><p className="mt-2 text-2xl font-bold text-blue-600">{devices.length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">{t("big1.deviceRegistry.trusted")}</div><p className="mt-2 text-2xl font-bold text-green-600">{devices.filter((d: any) => d.trusted).length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">{t("big1.deviceRegistry.activeSessions")}</div><p className="mt-2 text-2xl font-bold text-indigo-600">{devices.reduce((s: any, d: any) => s + d.session_count, 0)}</p></div>
          </div>

          {devices.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><HardDrive className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("big1.deviceRegistry.noRegisteredDevices")}</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.fingerprint")}</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.user")}</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.platform")}</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.sessions")}</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.lastSeen")}</th>
                  <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.trust")}</th>
                  <th scope="col" className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">{t("big1.deviceRegistry.actions")}</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {devices.map((d: any) => (
                    <tr key={d.id} className="bg-white dark:bg-gray-900">
                      <td className="px-4 py-3"><div className="flex items-center gap-2"><PlatformIcon platform={d.platform} /><span className="font-mono text-xs text-gray-700 dark:text-gray-300">{d.fingerprint.slice(0, 24)}</span></div></td>
                      <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{d.username}</div><div className="text-xs text-gray-400 font-mono">{d.user_id.slice(0, 12)}</div></td>
                      <td className="px-4 py-3"><div className="text-gray-700 dark:text-gray-300">{d.platform}</div><div className="text-xs text-gray-400 truncate max-w-[200px]">{d.user_agent}</div></td>
                      <td className="px-4 py-3"><span className="font-medium text-gray-700 dark:text-gray-300">{d.session_count}</span></td>
                      <td className="px-4 py-3 text-gray-400">{d.last_seen ? new Date(d.last_seen).toLocaleDateString() : "—"}</td>
                      <td className="px-4 py-3"><button onClick={() => handleToggleTrust(d)} className={`rounded-full px-2 py-0.5 text-xs font-medium ${d.trusted ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{d.trusted ? "Trusted" : "Untrusted"}</button></td>
                      <td className="px-4 py-3 text-right"><button onClick={() => handleRevoke(d.id)} className="text-red-400 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
