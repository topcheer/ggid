"use client";

import { useState, useEffect, useCallback } from "react";
import { Activity, Search, AlertTriangle, TrendingUp, Users, Clock } from "lucide-react";

interface HourlyBucket {
  hour: string;
  access_count: number;
  unique_users: number;
  is_anomaly: boolean;
}

interface FrequencyData {
  resource_id: string;
  resource_name: string;
  buckets: HourlyBucket[];
  total_accesses: number;
  avg_per_hour: number;
  anomaly_count: number;
  peak_hour: string;
  peak_count: number;
}

export default function AccessFrequencyPage() {
  const [search, setSearch] = useState("");
  const [data, setData] = useState<FrequencyData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (resource: string) => {
    if (!resource) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/policy/access-frequency?resource=${encodeURIComponent(resource)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchData(search);
  }, [search, fetchData]);

  const maxAccess = data ? Math.max(...data.buckets.map((b) => b.access_count), 1) : 1;
  const maxUsers = data ? Math.max(...data.buckets.map((b) => b.unique_users), 1) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> Access Frequency</h1>
        <p className="text-sm text-gray-500 mt-1">Hourly access patterns with anomaly detection per resource.</p>
      </div>

      {/* Resource search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input type="text" placeholder="Search by resource ID or name..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {data && (
        <>
          {/* Stats cards */}
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <span className="text-sm text-gray-500">Total Accesses</span>
              <p className="text-2xl font-bold mt-1">{data.total_accesses.toLocaleString()}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Avg/Hour</span><TrendingUp className="w-4 h-4 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.avg_per_hour.toFixed(1)}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Anomalies</span><AlertTriangle className="w-4 h-4 text-orange-400" /></div>
              <p className="text-2xl font-bold mt-1 text-orange-600">{data.anomaly_count}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Peak Hour</span><Clock className="w-4 h-4 text-gray-400" /></div>
              <p className="text-sm font-bold mt-1">{data.peak_hour}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Peak Count</span><TrendingUp className="w-4 h-4 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.peak_count}</p>
            </div>
          </div>

          {/* Hourly bar chart */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold">Hourly Access (30 days)</h3>
              <div className="flex items-center gap-4 text-xs text-gray-500">
                <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-blue-500" /> Access Count</span>
                <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-green-400" /> Unique Users</span>
                <span className="flex items-center gap-1"><AlertTriangle className="w-3 h-3 text-orange-500" /> Anomaly</span>
              </div>
            </div>
            <div className="flex items-end gap-0.5 h-48 relative">
              {data.buckets.map((b, i) => {
                const accessPct = (b.access_count / maxAccess) * 100;
                const userPct = (b.unique_users / maxUsers) * 100;
                return (
                  <div key={i} className="flex-1 flex flex-col items-center justify-end relative group" style={{ minWidth: "8px" }} title={`${b.hour}: ${b.access_count} accesses, ${b.unique_users} users`}>
                    {b.is_anomaly && <AlertTriangle className="absolute -top-5 w-4 h-4 text-orange-500" />}
                    {/* Unique users overlay bar */}
                    <div className="w-full bg-green-400 dark:bg-green-600 rounded-t" style={{ height: `${userPct}%`, minHeight: userPct > 0 ? "2px" : "0" }} />
                    {/* Access count bar */}
                    <div className="w-full bg-blue-500 dark:bg-blue-600" style={{ height: `${accessPct}%`, minHeight: accessPct > 0 ? "2px" : "0" }} />
                  </div>
                );
              })}
            </div>
            <div className="flex justify-between mt-2 text-xs text-gray-400">
              <span>{data.buckets[0]?.hour}</span>
              <span>{data.buckets[Math.floor(data.buckets.length / 2)]?.hour}</span>
              <span>{data.buckets[data.buckets.length - 1]?.hour}</span>
            </div>
          </div>

          {/* Anomaly list */}
          {data.anomaly_count > 0 && (
            <div className="rounded-lg border dark:border-gray-800">
              <div className="px-4 py-3 border-b dark:border-gray-800">
                <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-orange-500" /> Anomaly Hours ({data.anomaly_count})</h3>
              </div>
              <div className="divide-y dark:divide-gray-800 max-h-48 overflow-y-auto">
                {data.buckets.filter((b) => b.is_anomaly).map((b, i) => (
                  <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2"><AlertTriangle className="w-3 h-3 text-orange-500" /> {b.hour}</span>
                    <div className="flex items-center gap-3 text-xs text-gray-500">
                      <span>{b.access_count} accesses</span>
                      <span className="flex items-center gap-1"><Users className="w-3 h-3" /> {b.unique_users} users</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a resource to view access frequency.</p>}
      {loading && <p className="text-sm text-gray-500">Loading...</p>}
    </div>
  );
}
