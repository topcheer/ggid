"use client";

import { useState, useEffect, useCallback } from "react";
import { Clock, Gauge, TrendingUp } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface LifetimeData {
  avg_lifetime_minutes: number;
  median_lifetime_minutes: number;
  short_lived_count: number;
  long_lived_count: number;
  distribution: { range: string; count: number }[];
  per_client: { client_id: string; client_name: string; avg_minutes: number; token_count: number; short_pct: number }[];
}

export default function TokenLifetimePage() {
  const t = useTranslations();

  const [data, setData] = useState<LifetimeData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/token-lifetime", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxCount = data ? Math.max(...data.distribution.map((d: any) => d.count), 1) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Clock className="w-6 h-6 text-blue-500" /> {t("tokenLifetime.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Analyze token lifetimes across clients with distribution and short/long-lived breakdown.</p>
      </div>

      {data && (
        <>
          {/* Gauges */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex flex-col items-center">
              <span className="text-sm text-gray-500 mb-2 flex items-center gap-1"><Gauge className="w-4 h-4" /> Avg Lifetime</span>
              <p className="text-2xl font-bold">{data.avg_lifetime_minutes}<span className="text-base text-gray-400">m</span></p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex flex-col items-center">
              <span className="text-sm text-gray-500 mb-2 flex items-center gap-1"><Gauge className="w-4 h-4" /> Median</span>
              <p className="text-2xl font-bold">{data.median_lifetime_minutes}<span className="text-base text-gray-400">m</span></p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <span className="text-sm text-gray-500">Short-Lived (&lt;1h)</span>
              <p className="text-2xl font-bold mt-1 text-green-600">{data.short_lived_count}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <span className="text-sm text-gray-500">Long-Lived (&gt;24h)</span>
              <p className="text-2xl font-bold mt-1 text-orange-600">{data.long_lived_count}</p>
            </div>
          </div>

          {/* Distribution bar chart */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-4">Lifetime Distribution</h3>
            <div className="space-y-2">
              {data.distribution.map((d: any, i: number) => (
                <div key={i} className="flex items-center gap-3">
                  <span className="text-xs text-gray-500 w-24 text-right">{d.range}</span>
                  <div className="flex-1 h-6 rounded bg-gray-100 dark:bg-gray-800 overflow-hidden">
                    <div className="h-full rounded bg-blue-500 flex items-center justify-end px-2" style={{ width: `${(d.count / maxCount) * 100}%` }}><span className="text-xs text-white font-medium">{d.count}</span></div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Per-client table */}
          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Client</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Avg Lifetime</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Token Count</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Short-Lived %</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {data.per_client.map((c: any) => (
                  <tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-3 font-medium">{c.client_name}</td>
                    <td className="px-4 py-3">{c.avg_minutes}m</td>
                    <td className="px-4 py-3">{c.token_count}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <div className="w-20 h-2 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden"><div className="h-full bg-green-500" style={{ width: `${c.short_pct}%` }} /></div>
                        <span className="text-xs text-gray-400">{c.short_pct}%</span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
