"use client";

import { useState, useEffect, useCallback } from "react";
import { RefreshCw, AlertTriangle, Activity, Server } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ScimHealth {
  endpoint_url: string;
  last_sync_at: string;
  provisioning_errors: { timestamp: string; user_id: string; error: string }[];
  user_counts: { synced: number; pending: number; failed: number };
  rate_limit: { remaining: number; reset_at: string };
  throughput_per_min: number;
  status: "healthy" | "degraded" | "error";
}

const statusColors: Record<string, string> = {
  healthy: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  degraded: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  error: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ScimSyncHealthPage() {
  const t = useTranslations();
  const [data, setData] = useState<ScimHealth | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/scim-sync-health", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Server className="w-6 h-6 text-blue-500" /> SCIM Sync Health</h1><p className="text-sm text-gray-500 mt-1">Monitor SCIM user provisioning sync status and errors.</p></div>
        <button onClick={fetchData} aria-label="Refresh data" className="px-3 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><RefreshCw className="w-4 h-4" /> Refresh</button>
      </div>

      {data && (
        <>
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center justify-between">
            <div><span className="text-sm text-gray-500">Endpoint</span><p className="font-mono text-sm mt-0.5">{data.endpoint_url}</p></div>
            <span className={`px-2 py-1 rounded text-xs font-medium ${statusColors[data.status]}`}>{data.status}</span>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Activity className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Throughput</span><p className="text-xl font-bold mt-1">{data.throughput_per_min}/min</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><RefreshCw className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Rate Limit</span><p className="text-xl font-bold mt-1">{data.rate_limit.remaining}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Last Sync</span><p className="text-sm font-medium mt-1">{data.last_sync_at}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Rate Reset</span><p className="text-sm font-medium mt-1">{data.rate_limit.reset_at}</p></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold mb-3">User Provisioning</h3>
            <div className="flex items-center gap-4">
              <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">{(() => { const total = data.user_counts.synced + data.user_counts.pending + data.user_counts.failed || 1; let offset = 0; const segments = [{ val: data.user_counts.synced, color: "#10b981" }, { val: data.user_counts.pending, color: "#f59e0b" }, { val: data.user_counts.failed, color: "#ef4444" }]; return segments.map((seg, i) => { const pct = seg.val / total; const dash = pct * 176; const circle = <circle key={i} cx={32} cy={32} r={28} fill="none" stroke={seg.color} strokeWidth={8} strokeDasharray={`${dash} 176`} strokeDashoffset={-offset * 176} />; offset += pct; return circle; }); })()}</svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold">{data.user_counts.synced + data.user_counts.pending + data.user_counts.failed}</span><span className="text-[9px] text-gray-400">total</span></div></div>
              <div className="space-y-2"><div className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded bg-green-500" /><span>Synced: <strong>{data.user_counts.synced}</strong></span></div><div className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded bg-yellow-500" /><span>Pending: <strong>{data.user_counts.pending}</strong></span></div><div className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded bg-red-500" /><span>Failed: <strong>{data.user_counts.failed}</strong></span></div></div>
            </div>
          </div>

          {data.provisioning_errors.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-red-500" /> Provisioning Errors</h3><div className="space-y-1">{data.provisioning_errors.map((e, i) => (<div key={i} className="flex items-center gap-2 text-sm"><span className="text-xs text-gray-400">{e.timestamp}</span><span className="font-mono text-xs text-gray-500">{e.user_id}</span><span className="text-red-600">{e.error}</span></div>))}</div></div>
          )}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
