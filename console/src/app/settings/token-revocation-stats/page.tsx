"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Ban, TrendingUp } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Stats {
  total_revocations: number;
  by_reason: { reason: string; count: number }[];
  by_client: { client_id: string; client_name: string; count: number }[];
  trend_30d: { day: string; count: number }[];
  peak_revocation_hour: number;
}

const reasonColors: Record<string, string> = {
  user_initiated: "#3b82f6", admin: "#8b5cf6", expired: "#10b981", security_event: "#ef4444",
};

export default function TokenRevocationStatsPage() {
  const t = useTranslations();
  const [data, setData] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/token-revocation-stats", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxTrend = Math.max(...(data?.trend_30d.map((t) => t.count) || [1]), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Ban className="w-6 h-6 text-red-500" />{t("tokenRevocationStats.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track token revocation events across the platform.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Revocations</span><p className="text-xl font-bold text-red-600 mt-1">{data.total_revocations.toLocaleString()}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Peak Hour</span><p className="text-xl font-bold mt-1">{String(data.peak_revocation_hour).padStart(2, "0")}:00</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 col-span-2"><span className="text-sm text-gray-500">30-Day Trend</span><div className="flex items-end gap-0.5 mt-2 h-12">{data.trend_30d.map((t: any, i: number) => (<div key={i} className="flex-1 bg-red-400 dark:bg-red-500 rounded-t" style={{ height: (t.count / maxTrend) * 100 + "%", minHeight: "2px" }} title={t.day + ": " + t.count} />))}</div></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">By Reason</h3><div className="space-y-2">{data.by_reason.map((r) => (<div key={r.reason} className="flex items-center gap-2"><span className="w-3 h-3 rounded" style={{ background: reasonColors[r.reason] || "#ccc" }} /><span className="text-xs capitalize flex-1">{r.reason.replace("_", " ")}</span><span className="font-bold text-sm">{r.count}</span></div>))}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">By Client</h3><div className="space-y-2">{data.by_client.map((c) => (<div key={c.client_id} className="flex items-center gap-2"><span className="text-xs font-medium flex-1">{c.client_name}</span><span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 font-bold">{c.count}</span></div>))}</div></div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
