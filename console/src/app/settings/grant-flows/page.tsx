"use client";

import { useState, useEffect, useCallback } from "react";
import { Workflow, CheckCircle2, Clock, TrendingUp } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface FlowStat {
  flow: string;
  count: number;
  success_count: number;
  failure_count: number;
  success_rate: number;
  avg_duration_ms: number;
}

const flowColors: Record<string, string> = {
  authorization_code: "#3b82f6",
  client_credentials: "#10b981",
  refresh_token: "#8b5cf6",
  device_code: "#f59e0b",
  password: "#ef4444",
};

const flowLabels: Record<string, string> = {
  authorization_code: "Auth Code",
  client_credentials: "Client Credentials",
  refresh_token: "Refresh Token",
  device_code: "Device Code",
  password: "Password",
};

export default function GrantFlowsPage() {
  const t = useTranslations();

  const [stats, setStats] = useState<FlowStat[]>([]);
  const [loading, setLoading] = useState(false);
  const [timeRange, setTimeRange] = useState("7d");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/grant-flows/stats?range=${timeRange}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setStats(data.flows || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [timeRange]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxCount = Math.max(...stats.map((s) => s.count), 1);
  const maxDuration = Math.max(...stats.map((s) => s.avg_duration_ms), 1);
  const totalGrants = stats.reduce((sum, s) => sum + s.count, 0);
  const avgSuccessRate = stats.length > 0 ? stats.reduce((sum, s) => sum + s.success_rate, 0) / stats.length : 0;
  const totalSuccess = stats.reduce((sum, s) => sum + s.success_count, 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Workflow className="w-6 h-6 text-blue-500" /> {t("big1.grantFlows.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">{t("big1.grantFlows.oauthGrantFlowStatisticsAndPerformanceMetrics")}</p>
        </div>
        <select aria-label="Time range" value={timeRange} onChange={(e) => setTimeRange(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="24h">{t("big1.grantFlows.last24Hours")}</option>
          <option value="7d">{t("big1.grantFlows.last7Days")}</option>
          <option value="30d">{t("big1.grantFlows.last30Days")}</option>
        </select>
      </div>

      {/* Top stats cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">{t("big1.grantFlows.totalGrants")}</span><Workflow className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1">{totalGrants.toLocaleString()}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">{t("big1.grantFlows.avgSuccessRate")}</span><TrendingUp className="w-5 h-5 text-green-400" /></div>
          <p className={`text-2xl font-bold mt-1 ${avgSuccessRate >= 95 ? "text-green-600" : avgSuccessRate >= 80 ? "text-yellow-600" : "text-red-600"}`}>{avgSuccessRate.toFixed(1)}%</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">{t("big1.grantFlows.successful")}</span><CheckCircle2 className="w-5 h-5 text-green-400" /></div>
          <p className="text-2xl font-bold mt-1 text-green-600">{totalSuccess.toLocaleString()}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">{t("big1.grantFlows.activeFlows")}</span><Clock className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1">{stats.length}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Grant count bar chart */}
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <h3 className="font-semibold mb-4">{t("big1.grantFlows.grantCountByFlow")}</h3>
          <div className="space-y-3">
            {stats.map((s) => (
              <div key={s.flow} className="flex items-center gap-3">
                <span className="text-xs text-gray-500 w-28 truncate">{flowLabels[s.flow] || s.flow}</span>
                <div className="flex-1 h-7 rounded bg-gray-100 dark:bg-gray-800 overflow-hidden">
                  <div className="h-full rounded flex items-center justify-end px-2" style={{ width: `${(s.count / maxCount) * 100}%`, backgroundColor: flowColors[s.flow] || "#3b82f6" }}>
                    <span className="text-xs text-white font-medium">{s.count}</span>
                  </div>
                </div>
              </div>
            ))}
            {stats.length === 0 && !loading && <p className="text-sm text-gray-500">{t("big1.grantFlows.noDataAvailable")}</p>}
          </div>
        </div>

        {/* Avg duration bar chart */}
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <h3 className="font-semibold mb-4">{t("big1.grantFlows.avgDurationByFlow")}</h3>
          <div className="space-y-3">
            {stats.map((s) => (
              <div key={s.flow} className="flex items-center gap-3">
                <span className="text-xs text-gray-500 w-28 truncate">{flowLabels[s.flow] || s.flow}</span>
                <div className="flex-1 h-7 rounded bg-gray-100 dark:bg-gray-800 overflow-hidden">
                  <div className="h-full rounded flex items-center justify-end px-2" style={{ width: `${(s.avg_duration_ms / maxDuration) * 100}%`, backgroundColor: flowColors[s.flow] || "#3b82f6", opacity: 0.7 }}>
                    <span className="text-xs text-white font-medium">{s.avg_duration_ms}{t("big1.grantFlows.ms")}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Per-flow detail table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.grantFlows.flow")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.grantFlows.total")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.grantFlows.success")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.grantFlows.failure")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.grantFlows.successRate")}</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">{t("big1.grantFlows.avgDuration")}</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {stats.map((s) => (
              <tr key={s.flow} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <span className="w-3 h-3 rounded" style={{ backgroundColor: flowColors[s.flow] || "#3b82f6" }} />
                    <span className="font-medium">{flowLabels[s.flow] || s.flow}</span>
                    <span className="text-xs text-gray-400 font-mono">{s.flow}</span>
                  </div>
                </td>
                <td className="px-4 py-3 font-bold">{s.count.toLocaleString()}</td>
                <td className="px-4 py-3 text-green-600">{s.success_count}</td>
                <td className="px-4 py-3 text-red-600">{s.failure_count}</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${s.success_rate >= 95 ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" : s.success_rate >= 80 ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"}`}>{s.success_rate.toFixed(1)}%</span>
                </td>
                <td className="px-4 py-3">{s.avg_duration_ms}{t("big1.grantFlows.ms")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
