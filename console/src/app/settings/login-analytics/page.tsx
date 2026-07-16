"use client";

import { useState, useEffect, useCallback } from "react";
import { BarChart3, Calendar, TrendingUp, TrendingDown, Clock, CheckCircle, XCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AnalyticsData {
  total_attempts: number;
  successful: number;
  failed: number;
  success_rate: number;
  avg_duration_ms: number;
  method_breakdown: { method: string; count: number; percentage: number }[];
  failure_reasons: { reason: string; count: number }[];
  daily_trend: { date: string; success: number; failure: number }[];
}

const pieColors = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#ec4899"];

export default function LoginAnalyticsPage() {
  const t = useTranslations();

  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(false);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const fetchAnalytics = useCallback(async () => {
    if (!startDate || !endDate) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/auth/login-analytics?start=${startDate}&end=${endDate}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate]);

  useEffect(() => {
    // Default to last 30 days
    const end = new Date();
    const start = new Date();
    start.setDate(start.getDate() - 30);
    setStartDate(start.toISOString().split("T")[0]);
    setEndDate(end.toISOString().split("T")[0]);
  }, []);

  useEffect(() => {
    fetchAnalytics();
  }, [fetchAnalytics]);

  const maxFailureCount = data ? Math.max(...data.failure_reasons.map((f) => f.count), 1) : 1;

  // Build pie chart segments
  let cumulativePct = 0;
  const pieSegments = data?.method_breakdown.map((m, i) => {
    const startAngle = (cumulativePct / 100) * 360;
    cumulativePct += m.percentage;
    const endAngle = (cumulativePct / 100) * 360;
    return { ...m, color: pieColors[i % pieColors.length], startAngle, endAngle };
  }) || [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><BarChart3 className="w-6 h-6 text-blue-500" /> {t("loginAnalytics.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Analyze login patterns, method distribution, and failure reasons.</p>
      </div>

      {/* Date range picker */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <Calendar className="w-4 h-4 text-gray-400" />
          <input aria-label="Start date" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          <span className="text-gray-400">to</span>
          <input aria-label="End date" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <button onClick={fetchAnalytics} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? "Loading..." : "Refresh"}</button>
      </div>

      {data && (
        <>
          {/* Stats cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Total Attempts</span><BarChart3 className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.total_attempts.toLocaleString()}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Success Rate</span><TrendingUp className="w-5 h-5 text-green-500" /></div>
              <p className={`text-2xl font-bold mt-1 ${data.success_rate >= 90 ? "text-green-600" : data.success_rate >= 70 ? "text-yellow-600" : "text-red-600"}`}>{data.success_rate.toFixed(1)}%</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Avg Duration</span><Clock className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.avg_duration_ms}<span className="text-base text-gray-400">ms</span></p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Failed</span><XCircle className="w-5 h-5 text-red-400" /></div>
              <p className="text-2xl font-bold mt-1 text-red-600">{data.failed.toLocaleString()}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Method breakdown pie chart */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-4">Method Breakdown</h3>
              <div className="flex items-center gap-6">
                <div className="relative w-40 h-40">
                  <svg viewBox="0 0 100 100" className="w-full h-full -rotate-90">
                    {pieSegments.map((seg, i) => {
                      const r = 40, cx = 50, cy = 50;
                      const startRad = (seg.startAngle - 90) * Math.PI / 180;
                      const endRad = (seg.endAngle - 90) * Math.PI / 180;
                      const x1 = cx + r * Math.cos(startRad), y1 = cy + r * Math.sin(startRad);
                      const x2 = cx + r * Math.cos(endRad), y2 = cy + r * Math.sin(endRad);
                      const largeArc = seg.endAngle - seg.startAngle > 180 ? 1 : 0;
                      return <path key={i} d={`M${cx},${cy} L${x1},${y1} A${r},${r} 0 ${largeArc} 1 ${x2},${y2} Z`} fill={seg.color} stroke="white" strokeWidth={0.5} />;
                    })}
                  </svg>
                </div>
                <div className="flex-1 space-y-1">
                  {pieSegments.map((seg, i) => (
                    <div key={i} className="flex items-center gap-2 text-sm">
                      <span className="w-3 h-3 rounded" style={{ backgroundColor: seg.color }} />
                      <span className="flex-1">{seg.method}</span>
                      <span className="text-gray-400">{seg.count} ({seg.percentage.toFixed(0)}%)</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Failure reasons bar chart */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-4">Failure Reasons</h3>
              <div className="space-y-2">
                {data.failure_reasons.map((f, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <span className="text-xs text-gray-500 w-40 truncate">{f.reason}</span>
                    <div className="flex-1 h-6 rounded bg-gray-100 dark:bg-gray-800 overflow-hidden">
                      <div className="h-full rounded bg-red-500 flex items-center justify-end px-2" style={{ width: `${(f.count / maxFailureCount) * 100}%` }}>
                        <span className="text-xs text-white font-medium">{f.count}</span>
                      </div>
                    </div>
                  </div>
                ))}
                {data.failure_reasons.length === 0 && <p className="text-sm text-gray-500">No failures recorded.</p>}
              </div>
            </div>
          </div>

          {/* Daily trend */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-4">Daily Trend</h3>
            <div className="flex items-end gap-1 h-40">
              {data.daily_trend.slice(-30).map((d, i) => {
                const total = d.success + d.failure;
                const maxTotal = Math.max(...data.daily_trend.map((t) => t.success + t.failure), 1);
                const heightPct = (total / maxTotal) * 100;
                const successPct = total > 0 ? (d.success / total) * 100 : 0;
                return (
                  <div key={i} className="flex-1 group relative flex flex-col justify-end" style={{ height: `${heightPct}%`, minHeight: "4px" }} title={`${d.date}: ${d.success} success, ${d.failure} failure`}>
                    <div className="w-full bg-red-400 rounded-t" style={{ height: `${100 - successPct}%` }} />
                    <div className="w-full bg-green-500" style={{ height: `${successPct}%` }} />
                  </div>
                );
              })}
            </div>
            <div className="flex items-center gap-4 mt-2 text-xs">
              <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-green-500" /> Success</span>
              <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-red-400" /> Failure</span>
            </div>
          </div>
        </>
      )}

      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Select a date range to view analytics.</p>}
    </div>
  );
}
