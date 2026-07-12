"use client";

import { useState, useEffect, useCallback } from "react";
import { Clock, Smartphone, Globe, Activity, AlertTriangle } from "lucide-react";

interface LoginPatternData {
  time_of_day: { hour: number; count: number }[];
  device_usage: { device: string; count: number }[];
  geo_distribution: { country: string; city: string; count: number; lat?: number; lng?: number }[];
  frequency_trend: { date: string; logins: number }[];
  anomalies: { type: string; description: string; severity: "low" | "medium" | "high" }[];
}

const sevColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

const deviceColors = ["#3b82f6", "#8b5cf6", "#10b981", "#f59e0b", "#ef4444"];

export default function LoginPatternsPage() {
  const [data, setData] = useState<LoginPatternData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/login-patterns", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxHourCount = Math.max(...(data?.time_of_day.map((d) => d.count) || [1]), 1);
  const totalDevices = data?.device_usage.reduce((s, d) => s + d.count, 0) || 1;
  const maxFreq = Math.max(...(data?.frequency_trend.map((d) => d.logins) || [1]), 1);
  const freqPoints = data?.frequency_trend.map((d, i) => `${(i / (data.frequency_trend.length - 1 || 1)) * 200},${40 - (d.logins / maxFreq) * 35}`).join(" ") || "";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> Login Patterns</h1>
        <p className="text-sm text-gray-500 mt-1">Analyze authentication patterns and detect anomalies.</p>
      </div>

      {data && (
        <>
          {data.anomalies.length > 0 && (
            <div className="space-y-2">
              {data.anomalies.map((a, i) => (
                <div key={i} className="rounded-lg border border-yellow-200 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20 p-3 flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 text-yellow-500" />
                  <span className="text-sm flex-1"><strong>{a.type}:</strong> {a.description}</span>
                  <span className={`px-2 py-0.5 rounded text-xs ${sevColors[a.severity]}`}>{a.severity}</span>
                </div>
              ))}
            </div>
          )}

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Clock className="w-4 h-4 text-gray-400" /> Time of Day Distribution</h3>
              <div className="flex items-end gap-1 h-32">
                {data.time_of_day.map((d) => (
                  <div key={d.hour} className="flex-1 flex flex-col items-center gap-1">
                    <div className="w-full bg-blue-500 rounded-t" style={{ height: `${(d.count / maxHourCount) * 100}%` }} title={`${d.count} logins`} />
                    <span className="text-[8px] text-gray-400">{d.hour}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Smartphone className="w-4 h-4 text-gray-400" /> Device Usage</h3>
              <div className="space-y-2">
                {data.device_usage.map((d, i) => (
                  <div key={d.device} className="flex items-center gap-2">
                    <span className="w-3 h-3 rounded" style={{ background: deviceColors[i % deviceColors.length] }} />
                    <span className="text-sm flex-1">{d.device}</span>
                    <span className="text-sm font-bold">{d.count}</span>
                    <span className="text-xs text-gray-400">({((d.count / totalDevices) * 100).toFixed(0)}%)</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Globe className="w-4 h-4 text-gray-400" /> Geographic Distribution</h3>
              <div className="space-y-1">
                {data.geo_distribution.map((g, i) => (
                  <div key={i} className="flex items-center gap-2 text-sm">
                    <span className="font-mono text-xs bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">{g.country}</span>
                    <span className="flex-1">{g.city}</span>
                    <span className="font-bold">{g.count}</span>
                  </div>
                ))}
                {data.geo_distribution.length === 0 && <p className="text-xs text-gray-400">No geo data.</p>}
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Activity className="w-4 h-4 text-gray-400" /> Login Frequency (30d)</h3>
              <svg viewBox="0 0 200 40" className="w-full h-20">
                <polyline fill="none" stroke="#3b82f6" strokeWidth={2} points={freqPoints} />
              </svg>
            </div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
