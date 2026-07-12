"use client";

import { useState, useCallback } from "react";
import { Gauge, Search, AlertTriangle, CheckCircle2 } from "lucide-react";

interface ThrottleData {
  user_id: string;
  username: string;
  is_throttled: boolean;
  delay_ms: number;
  failed_attempts: number;
  max_attempts: number;
  reset_at: string;
  reset_seconds: number;
  last_failed_ip: string;
}

export default function ThrottleStatusPage() {
  const [search, setSearch] = useState("");
  const [data, setData] = useState<ThrottleData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/auth/throttle-status?user=${encodeURIComponent(user)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  const iconBg = data?.is_throttled ? "bg-red-50 dark:bg-red-900/20" : "bg-green-50 dark:bg-green-900/20";
  const statusBadge = data?.is_throttled
    ? "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
    : "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400";
  const barColor = data?.is_throttled ? "bg-red-500" : "bg-yellow-500";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Gauge className="w-6 h-6 text-blue-500" /> Throttle Status</h1>
        <p className="text-sm text-gray-500 mt-1">Check login throttle state and rate limit countdown per user.</p>
      </div>

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input type="text" placeholder="Search by username..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {data && (
        <div className="rounded-lg border dark:border-gray-800 p-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className={`w-12 h-12 rounded-full flex items-center justify-center ${iconBg}`}>
                {data.is_throttled ? <AlertTriangle className="w-6 h-6 text-red-500" /> : <CheckCircle2 className="w-6 h-6 text-green-500" />}
              </div>
              <div>
                <h3 className="font-semibold text-lg">{data.username}</h3>
                <p className="text-xs text-gray-400">Last failed IP: {data.last_failed_ip || "-"}</p>
              </div>
            </div>
            <span className={`px-3 py-1 rounded-lg text-sm font-medium ${statusBadge}`}>{data.is_throttled ? "THROTTLED" : "NORMAL"}</span>
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4">
            <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><span className="text-xs text-gray-500">Failed Attempts</span><p className="text-xl font-bold mt-1">{data.failed_attempts}/{data.max_attempts}</p></div>
            <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><span className="text-xs text-gray-500">Delay</span><p className="text-xl font-bold mt-1">{data.delay_ms}<span className="text-sm text-gray-400">ms</span></p></div>
            <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><span className="text-xs text-gray-500">Reset In</span><p className="text-xl font-bold mt-1">{data.reset_seconds > 0 ? `${Math.floor(data.reset_seconds / 60)}m ${data.reset_seconds % 60}s` : "-"}</p></div>
            <div className="rounded-lg bg-gray-50 dark:bg-gray-900/50 p-3"><span className="text-xs text-gray-500">Reset At</span><p className="text-sm font-medium mt-1">{data.reset_at || "-"}</p></div>
          </div>

          {data.failed_attempts > 0 && (
            <div className="mt-4">
              <div className="flex items-center justify-between text-xs mb-1"><span className="text-gray-500">Attempt capacity</span><span>{data.failed_attempts}/{data.max_attempts}</span></div>
              <div className="w-full h-2 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden">
                <div className={`h-full rounded-full ${barColor}`} style={{ width: `${(data.failed_attempts / data.max_attempts) * 100}%` }} />
              </div>
            </div>
          )}
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No throttle data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to check throttle status.</p>}
    </div>
  );
}
