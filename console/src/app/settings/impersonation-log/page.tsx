"use client";
import { useState, useEffect, useCallback } from "react";
import { UserCog, Ban, AlertTriangle, Search } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface ImpersonationEvent { id: string; impersonator: string; target_user: string; start_at: string; end_at: string | null; duration_minutes: number; reason: string; ip_address: string; is_active: boolean; }
export default function ImpersonationLogPage() {
  const t = useTranslations();

  const [events, setEvents] = useState<ImpersonationEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/impersonation-log", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) throw new Error(`Failed to load impersonation log: HTTP ${res.status}`);
      const d = await res.json();
      setEvents(d.events || d || []);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load impersonation log"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const revoke = async (id: string) => {
    try {
      const res = await fetch("/api/v1/auth/impersonation-log/" + id + "/revoke", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) throw new Error(`Failed to revoke session: HTTP ${res.status}`);
      fetchData();
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to revoke session"); }
  };
  const filtered = search ? events.filter((e) => e.impersonator.includes(search) || e.target_user.includes(search)) : events;
  const active = events.filter((e) => e.is_active).length;
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><UserCog className="w-6 h-6 text-orange-500" /> {t("impersonationLog.title")}</h1><p className="text-sm text-gray-500 mt-1">Track admin impersonation sessions.</p></div>
      {active > 0 && <div className="rounded-lg border border-orange-300 dark:border-orange-800 bg-orange-50 dark:bg-orange-900/20 p-3 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" /><span className="font-semibold text-orange-700 dark:text-orange-400">{active} active impersonation session(s)</span></div>}
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span>{error}</span><button onClick={fetchData} className="text-xs underline hover:text-red-700">Retry</button></div>}
      <div className="relative max-w-xs">
        <Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search user..." aria-label="Search impersonation log" className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>
      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading log...</div></div>}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Impersonator</th><th className="px-4 py-3 text-left font-medium">Target</th><th className="px-4 py-3 text-left font-medium">Start</th><th className="px-4 py-3 text-left font-medium">Duration</th><th className="px-4 py-3 text-left font-medium">Reason</th><th className="px-4 py-3 text-left font-medium">IP</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{filtered.map((e) => (
            <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
              <td className="px-4 py-3 font-medium">{e.impersonator}</td><td className="px-4 py-3 text-orange-600 font-medium">{e.target_user}</td><td className="px-4 py-3 text-xs text-gray-500">{e.start_at}</td><td className="px-4 py-3 text-xs">{e.duration_minutes}m</td><td className="px-4 py-3 text-xs text-gray-500">{e.reason}</td><td className="px-4 py-3 font-mono text-xs text-gray-400">{e.ip_address}</td>
              <td className="px-4 py-3">{e.is_active ? <span className="px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400">Active</span> : <span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">Ended</span>}</td>
              <td className="px-4 py-3">{e.is_active && <button onClick={() => revoke(e.id)} aria-label={`Revoke impersonation for ${e.target_user}`} className="text-xs text-red-600 hover:underline flex items-center gap-1"><Ban className="w-3 h-3" /> Revoke</button>}</td>
            </tr>
          ))}{filtered.length === 0 && !loading && <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No events.</td></tr>}</tbody>
        </table>
      </div>
    </div>
  );
}
