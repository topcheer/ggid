"use client";

import { useState, useEffect, useCallback } from "react";
import { KeyRound, Calendar, AlertTriangle, CheckCircle2, Clock, TrendingUp } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ResetAnalytics {
  total_resets: number;
  successful: number;
  failed: number;
  success_rate: number;
  avg_completion_time_ms: number;
  breach_triggered: number;
  method_breakdown: { method: string; count: number; percentage: number }[];
  daily_trend: { date: string; count: number }[];
}

const pieColors = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6"];

export default function PasswordResetAnalyticsPage() {
  const t = useTranslations();

  const [data, setData] = useState<ResetAnalytics | null>(null);
  const [loading, setLoading] = useState(false);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const fetchData = useCallback(async () => {
    if (!startDate || !endDate) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/auth/password-reset-analytics?start=${startDate}&end=${endDate}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [startDate, endDate]);

  useEffect(() => {
    const end = new Date();
    const start = new Date(); start.setDate(start.getDate() - 30);
    setStartDate(start.toISOString().split("T")[0]);
    setEndDate(end.toISOString().split("T")[0]);
  }, []);

  useEffect(() => {
    if (startDate && endDate) fetchData();
  }, [startDate, endDate, fetchData]);

  // Pie chart segments
  let cumulativePct = 0;
  const segments = data?.method_breakdown.map((m, i) => {
    const startAngle = (cumulativePct / 100) * 360;
    cumulativePct += m.percentage;
    const endAngle = (cumulativePct / 100) * 360;
    return { ...m, color: pieColors[i % pieColors.length], startAngle, endAngle };
  }) || [];

  const maxDaily = data ? Math.max(...data.daily_trend.map((d) => d.count), 1) : 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><KeyRound className="w-6 h-6 text-blue-500" /> {t("passwordResetAnalytics.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track password reset patterns, method usage, and breach-triggered events.</p>
      </div>

      {/* Date range */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <Calendar className="w-4 h-4 text-gray-400" />
          <input aria-label="Start date" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          <span className="text-gray-400">to</span>
          <input aria-label="End date" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <button onClick={fetchData} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? "Loading..." : "Refresh"}</button>
      </div>

      {data && (
        <>
          {/* Breach alert */}
          {data.breach_triggered > 0 && (
            <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-red-500" />
              <span className="font-semibold text-red-700 dark:text-red-400">{data.breach_triggered} breach-triggered password resets detected</span>
            </div>
          )}

          {/* Stats cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Total Resets</span><KeyRound className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.total_resets.toLocaleString()}</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Success Rate</span><TrendingUp className="w-5 h-5 text-green-400" /></div>
              <p className={`text-2xl font-bold mt-1 ${data.success_rate >= 90 ? "text-green-600" : data.success_rate >= 70 ? "text-yellow-600" : "text-red-600"}`}>{data.success_rate.toFixed(1)}%</p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Avg Completion</span><Clock className="w-5 h-5 text-gray-400" /></div>
              <p className="text-2xl font-bold mt-1">{data.avg_completion_time_ms}<span className="text-base text-gray-400">ms</span></p>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Failed</span><AlertTriangle className="w-5 h-5 text-red-400" /></div>
              <p className="text-2xl font-bold mt-1 text-red-600">{data.failed}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Method breakdown pie */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-4">Method Breakdown</h3>
              <div className="flex items-center gap-6">
                <div className="relative w-36 h-36">
                  <svg viewBox="0 0 100 100" className="w-full h-full -rotate-90">
                    {segments.map((seg, i) => {
                      if (seg.count === 0) return null;
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
                  {segments.map((seg, i) => (
                    <div key={i} className="flex items-center gap-2 text-sm">
                      <span className="w-3 h-3 rounded" style={{ backgroundColor: seg.color }} />
                      <span className="flex-1">{seg.method}</span>
                      <span className="text-gray-400">{seg.count} ({seg.percentage.toFixed(0)}%)</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Daily trend */}
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="font-semibold mb-4">Daily Reset Trend</h3>
              <div className="flex items-end gap-0.5 h-32">
                {data.daily_trend.slice(-30).map((d, i) => (
                  <div key={i} className="flex-1 bg-blue-500 rounded-t" style={{ height: `${(d.count / maxDaily) * 100}%`, minHeight: d.count > 0 ? "3px" : "0" }} title={`${d.date}: ${d.count} resets`} />
                ))}
              </div>
            </div>
          </div>
        </>
      )}

      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Select a date range.</p>}
    </div>
  );
}
