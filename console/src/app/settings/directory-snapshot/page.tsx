"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { Camera, Users, TrendingUp, TrendingDown } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Snapshot {
  total_users: number;
  by_status: { status: string; count: number }[];
  by_org: { org: string; count: number }[];
  by_role: { role: string; count: number }[];
  changes_24h: { created: number; deleted: number; modified: number; net: number };
  snapshot_at: string;
}

const statusColors: Record<string, string> = {
  active: "#10b981", suspended: "#f59e0b", locked: "#ef4444", pending: "#3b82f6", inactive: "#9ca3af",
};

export default function DirectorySnapshotPage() {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [loading, setLoading] = useState(false);
  const t = useTranslations();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/directory-snapshot", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setSnapshot(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const total = snapshot?.by_status.reduce((s, d) => s + d.count, 0) || 1;
  const maxOrg = Math.max(...(snapshot?.by_org.map((d) => d.count) || [1]), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Camera className="w-6 h-6 text-cyan-500" /> {t("directorySnapshot.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("directorySnapshot.subtitle")}</p>
      </div>

      {snapshot && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Users className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">{t("directorySnapshot.totalUsers")}</span><p className="text-xl font-bold">{snapshot.total_users}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><TrendingUp className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">{t("directorySnapshot.created24h")}</span><p className="text-xl font-bold text-green-600">{snapshot.changes_24h.created}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><TrendingDown className="w-8 h-8 text-red-500" /><div><span className="text-sm text-gray-500">{t("directorySnapshot.deleted24h")}</span><p className="text-xl font-bold text-red-600">{snapshot.changes_24h.deleted}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><span className={`text-xl font-bold ${snapshot.changes_24h.net >= 0 ? "text-green-600" : "text-red-600"}`}>{snapshot.changes_24h.net >= 0 ? "+" : ""}{snapshot.changes_24h.net}</span><div><span className="text-sm text-gray-500">{t("directorySnapshot.netChange")}</span><p className="text-xs text-gray-400">{snapshot.changes_24h.modified} modified</p></div></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("directorySnapshot.byStatus")}</h3>
              <div className="flex items-center gap-4">
                <div className="relative w-32 h-32"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">{(() => { let offset = 0; return snapshot.by_status.map((d) => { const pct = d.count / total; const dash = pct * 176; const circle = <circle key={d.status} cx={32} cy={32} r={28} fill="none" stroke={statusColors[d.status] || "#ccc"} strokeWidth={8} strokeDasharray={`${dash} 176`} strokeDashoffset={-offset * 176} />; offset += pct; return circle; }); })()}</svg></div>
                <div className="space-y-1">{snapshot.by_status.map((d) => (<div key={d.status} className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded-full" style={{ background: statusColors[d.status] || "#ccc" }} /><span className="capitalize">{d.status}</span><span className="font-bold ml-auto">{d.count}</span></div>))}</div>
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("directorySnapshot.byOrg")}</h3>
              <div className="space-y-2">{snapshot.by_org.slice(0, 8).map((d) => (<div key={d.org} className="flex items-center gap-2"><span className="text-xs text-gray-500 w-24 truncate">{d.org}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-5 overflow-hidden"><div className="h-full bg-blue-500 rounded-full" style={{ width: `${(d.count / maxOrg) * 100}%` }} /></div><span className="text-xs font-bold w-8 text-right">{d.count}</span></div>))}</div>
            </div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("directorySnapshot.role")}</th><th className="px-4 py-3 text-left font-medium">{t("directorySnapshot.count")}</th><th className="px-4 py-3 text-left font-medium">{t("directorySnapshot.share")}</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{snapshot.by_role.map((d) => (<tr key={d.role} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{d.role}</td><td className="px-4 py-3 font-bold">{d.count}</td><td className="px-4 py-3"><div className="flex items-center gap-2"><div className="w-24 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className="h-full bg-purple-500 rounded-full" style={{ width: `${(d.count / total) * 100}%` }} /></div><span className="text-xs text-gray-500">{((d.count / total) * 100).toFixed(1)}%</span></div></td></tr>))}</tbody>
            </table>
          </div>

          <p className="text-xs text-gray-400 text-right">{t("directorySnapshot.snapshotTaken")} {snapshot.snapshot_at}</p>
        </>
      )}
      {!snapshot && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("directorySnapshot.loading")}</p>}
    </div>
  );
}
