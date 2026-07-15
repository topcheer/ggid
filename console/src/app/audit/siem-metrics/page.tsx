"use client";

import { useState, useEffect, useCallback } from "react";
import { Send, AlertCircle, Clock, Server, Activity, Calendar } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SIEMMetrics {
  total_forwarded: number;
  failed: number;
  avg_latency_ms: number;
  uptime_pct: number;
  error_breakdown: { error_type: string; count: number }[];
  destinations: {
    id: string;
    name: string;
    endpoint: string;
    protocol: string;
    forwarded: number;
    failed: number;
    last_status: string;
    avg_latency_ms: number;
  }[];
}

export default function SIEMMetricsPage() {
  const t = useTranslations();

  const [data, setData] = useState<SIEMMetrics | null>(null);
  const [loading, setLoading] = useState(false);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params = startDate && endDate ? `?start=${startDate}&end=${endDate}` : "";
      const res = await fetch(`/api/v1/audit/siem-metrics${params}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [startDate, endDate]);

  useEffect(() => {
    const end = new Date();
    const start = new Date(); start.setDate(start.getDate() - 7);
    setStartDate(start.toISOString().split("T")[0]);
    setEndDate(end.toISOString().split("T")[0]);
  }, []);

  useEffect(() => {
    if (startDate && endDate) fetchData();
  }, [startDate, endDate, fetchData]);

  const maxErrorCount = data ? Math.max(...data.error_breakdown.map((e) => e.count), 1) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Send className="w-6 h-6 text-blue-500" /> {t("auditSiemMetrics.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Monitor SIEM forwarding health, latency, and per-destination status.</p>
      </div>

      {/* Date range */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <Calendar className="w-4 h-4 text-gray-400" />
          <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          <span className="text-gray-400">to</span>
          <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <button onClick={fetchData} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? "Loading..." : "Refresh"}</button>
      </div>

      {data && (
        <>
          {/* Stats cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Forwarded</span><Send className="w-5 h-5 text-green-400" /></div>
              <p className="text-2xl font-bold mt-1 text-green-600">{data.total_forwarded.toLocaleString()}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Failed</span><AlertCircle className="w-5 h-5 text-red-400" /></div>
              <p className="text-2xl font-bold mt-1 text-red-600">{data.failed.toLocaleString()}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Avg Latency</span><Clock className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.avg_latency_ms}<span className="text-base text-gray-400">ms</span></p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Uptime</span><Activity className="w-5 h-5 text-gray-400" /></div>
              <p className={`text-2xl font-bold mt-1 ${data.uptime_pct >= 99 ? "text-green-600" : data.uptime_pct >= 95 ? "text-yellow-600" : "text-red-600"}`}>{data.uptime_pct.toFixed(1)}%</p>
            </div>
          </div>

          {/* Error breakdown chart */}
          {data.error_breakdown.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-3">Error Breakdown</h3>
              <div className="space-y-2">
                {data.error_breakdown.map((e, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <span className="text-xs text-gray-500 w-48 truncate">{e.error_type}</span>
                    <div className="flex-1 h-6 rounded bg-gray-100 dark:bg-gray-800 overflow-hidden">
                      <div className="h-full rounded bg-red-500 flex items-center justify-end px-2" style={{ width: `${(e.count / maxErrorCount) * 100}%` }}>
                        <span className="text-xs text-white font-medium">{e.count}</span>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Per-destination table */}
          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th className="px-4 py-3 text-left font-medium">Destination</th>
                  <th className="px-4 py-3 text-left font-medium">Protocol</th>
                  <th className="px-4 py-3 text-left font-medium">Forwarded</th>
                  <th className="px-4 py-3 text-left font-medium">Failed</th>
                  <th className="px-4 py-3 text-left font-medium">Avg Latency</th>
                  <th className="px-4 py-3 text-left font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {data.destinations.map((d) => (
                  <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Server className="w-3 h-3 text-gray-400" />
                        <div>
                          <span className="font-medium">{d.name}</span>
                          <p className="text-xs text-gray-400 font-mono">{d.endpoint}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{d.protocol}</span></td>
                    <td className="px-4 py-3 text-green-600 font-medium">{d.forwarded.toLocaleString()}</td>
                    <td className="px-4 py-3 text-red-600">{d.failed}</td>
                    <td className="px-4 py-3">{d.avg_latency_ms}ms</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded text-xs ${d.last_status === "healthy" ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"}`}>{d.last_status}</span>
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
