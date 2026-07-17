"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  BarChart3, TrendingUp, Users, Activity, Loader2, AlertCircle, X,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface HourBucket {
  hour: string;
  count: number;
}

interface TopItem {
  name: string;
  count: number;
}

export default function AuditChartPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [hourly, setHourly] = useState<HourBucket[]>([]);
  const [topTypes, setTopTypes] = useState<TopItem[]>([]);
  const [topUsers, setTopUsers] = useState<TopItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const data = await apiFetch<{ hourly?: HourBucket[]; top_types?: TopItem[]; top_users?: TopItem[] }>("/api/v1/audit/stats/chart").catch(() => null);
        setHourly(data?.hourly ?? []);
        setTopTypes(data?.top_types ?? []);
        setTopUsers(data?.top_users ?? []);
      } catch {
        setError("Failed to load audit chart data");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const maxCount = Math.max(...hourly.map((h) => h.count), 1);
  const totalEvents = hourly.reduce((sum, h) => sum + h.count, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <BarChart3 className="h-6 w-6 text-indigo-600" /> Audit Activity Chart
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Events per hour (last 24h) with top event types and active users.</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : (
        <>
          {/* Summary */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-indigo-500" /><span className="text-xs font-semibold uppercase text-gray-400">Total Events</span></div><p className="mt-2 text-2xl font-bold text-indigo-600">{totalEvents}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><TrendingUp className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Avg/Hour</span></div><p className="mt-2 text-2xl font-bold text-green-600">{hourly.length > 0 ? Math.round(totalEvents / hourly.length) : 0}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Users className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active Users</span></div><p className="mt-2 text-2xl font-bold text-blue-600">{topUsers.length}</p></div>
          </div>

          {/* Bar chart */}
          <div className={cardCls}>
            <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Events Per Hour (24h)</h3>
            <div className="flex items-end gap-1 overflow-x-auto" style={{ height: 160 }}>
              {hourly.map((h, i) => (
                <div key={i} className="group relative flex flex-1 flex-col items-center" style={{ minWidth: 20 }}>
                  <div className="w-full rounded-t bg-indigo-500 transition-all hover:bg-indigo-600" style={{ height: `${(h.count / maxCount) * 120}px`, minHeight: 2 }}>
                    <div className="absolute -top-6 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-gray-800 px-1.5 py-0.5 text-xs text-white opacity-0 group-hover:opacity-100">{h.count}</div>
                  </div>
                  <span className="mt-1 text-[10px] text-gray-400">{h.hour}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            {/* Top event types */}
            <div className={cardCls}>
              <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Top Event Types</h3>
              {topTypes.length === 0 ? <p className="py-4 text-center text-sm text-gray-400">No data</p> : (
                <div className="space-y-2">
                  {topTypes.slice(0, 8).map((t, i) => {
                    const maxT = topTypes[0]?.count ?? 1;
                    return (
                      <div key={i}>
                        <div className="flex items-center justify-between text-sm"><span className="font-mono text-gray-600 dark:text-gray-300">{t.name}</span><span className="font-bold text-indigo-600">{t.count}</span></div>
                        <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-400" style={{ width: `${(t.count / maxT) * 100}%` }} /></div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Top users */}
            <div className={cardCls}>
              <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Most Active Users</h3>
              {topUsers.length === 0 ? <p className="py-4 text-center text-sm text-gray-400">No data</p> : (
                <div className="space-y-2">
                  {topUsers.slice(0, 8).map((u, i) => {
                    const maxU = topUsers[0]?.count ?? 1;
                    return (
                      <div key={i}>
                        <div className="flex items-center justify-between text-sm"><span className="text-gray-600 dark:text-gray-300">{u.name}</span><span className="font-bold text-blue-600">{u.count}</span></div>
                        <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-blue-400" style={{ width: `${(u.count / maxU) * 100}%` }} /></div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
