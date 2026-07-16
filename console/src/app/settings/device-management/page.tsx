"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Smartphone, Ban, Trash2, AlertTriangle, RotateCcw } from "lucide-react";
interface Device { device_id: string; user_id: string; username: string; device_name: string; platform: string; last_seen: string; trust_level: "managed" | "byod" | "untrusted"; enrolled_at: string; fingerprint: string; }
const trustColors: Record<string, string> = { managed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400", byod: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400", untrusted: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" };
export default function DeviceManagementPage() {
  const t = useTranslations();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState("");
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/identity/devices", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const d = await res.json();
      setDevices(d.devices || d || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load devices");
    } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const revoke = async (id: string) => {
    try {
      const res = await fetch("/api/v1/identity/devices/" + id, { method: "DELETE", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setDevices(devices.filter((d) => d.device_id !== id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to revoke device");
    }
  };
  const revokeStale = async () => {
    try {
      const res = await fetch("/api/v1/identity/devices/stale/revoke", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      fetchData();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to revoke stale devices");
    }
  };
  const filtered = filter ? devices.filter((d) => d.trust_level === filter) : devices;
  const stale = devices.filter((d) => { const days = (Date.now() - new Date(d.last_seen).getTime()) / 86400000; return days > 30; }).length;
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Smartphone className="w-6 h-6 text-blue-500" /> {t("backend.deviceManagement.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Manage registered devices and trust levels.</p>
        </div>
        {stale > 0 && <button onClick={revokeStale} aria-label={`Revoke ${stale} stale devices`} className="px-4 py-2 rounded-lg bg-red-600 text-white text-sm font-medium flex items-center gap-2"><Trash2 className="w-4 h-4" /> Revoke {stale} Stale</button>}
      </div>
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button aria-label="action" onClick={fetchData} className="text-xs underline hover:text-red-700">{t("backend.deviceManagement.retry")}</button></div>}
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading devices...</div></div>}
      <div className="grid grid-cols-4 gap-4">{(["managed", "byod", "untrusted"] as const).map((t) => <div key={t} className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500 capitalize">{t}</span><p className="text-xl font-bold mt-1">{devices.filter((d) => d.trust_level === t).length}</p></div>)} <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend.deviceManagement.total")}</span><p className="text-xl font-bold mt-1">{devices.length}</p></div></div>
      <div className="flex items-center gap-2">
        <select value={filter} onChange={(e) => setFilter(e.target.value)} aria-label="Filter by trust level" className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="">{t("backend.deviceManagement.allTrustLevels")}</option><option value="managed">{t("backend.deviceManagement.managed")}</option><option value="byod">{t("backend.deviceManagement.byod")}</option><option value="untrusted">Untrusted</option>
        </select>
      </div>
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr><th scope="col" className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.device")}</th><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.platform")}</th><th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.lastSeen")}</th><th className="px-4 py-3 text-left font-medium">Trust</th><th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.fingerprint")}</th><th className="px-4 py-3 text-left font-medium">{t("backend.deviceManagement.action")}</th></tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {filtered.map((d) => (
              <tr key={d.device_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3"><span className="font-medium">{d.device_name}</span><p className="text-xs text-gray-400 font-mono">{d.device_id}</p></td>
                <td className="px-4 py-3">{d.username}</td>
                <td className="px-4 py-3 text-xs">{d.platform}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{d.last_seen}</td>
                <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + trustColors[d.trust_level]}>{d.trust_level}</span></td>
                <td className="px-4 py-3 font-mono text-xs text-gray-400 max-w-32 truncate">{d.fingerprint}</td>
                <td className="px-4 py-3"><button onClick={() => revoke(d.device_id)} aria-label={`Revoke device ${d.device_name}`} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Ban className="w-3 h-3" /> Revoke</button></td>
              </tr>
            ))}
            {filtered.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No devices.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
