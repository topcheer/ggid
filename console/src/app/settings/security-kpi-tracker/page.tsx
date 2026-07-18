"use client";

import { useSecurityKPITracker } from "@ggid/sdk-react";
import { Target, TrendingUp, Download, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SecurityKPITrackerPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useSecurityKPITracker();

  if (loading) return <div className="p-8 text-gray-400">Loading KPI tracker...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Security KPI Tracker</h1>
          <p className="text-sm text-gray-400 mt-1">Track security KPIs against targets</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition" aria-label="Download board report">
            <Download className="w-4 h-4" /> Board Report
          </button>
          <button onClick={refresh} aria-label="Refresh KPI data" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* KPI Definitions */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Target className="w-4 h-4 text-blue-400" />
          KPI Definitions
        </h2>
        <div className="space-y-3">
          {(data?.kpi_definitions ?? []).map((k) => {
            const pct = k.target > 0 ? Math.min((k.current / k.target) * 100, 150) : 0;
            const onTrack = k.trend === "up" ? k.current >= k.target : k.current <= k.target;
            return (
              <div key={k.name} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <p className="text-sm font-medium">{k.name}</p>
                    <p className="text-xs text-gray-400">Owner: {k.owner}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-bold">
                      {k.current}<span className="text-xs text-gray-400">/{k.target} {k.unit}</span>
                    </p>
                    <span className={"text-xs " + (onTrack ? "text-green-400" : "text-red-400")}>
                      <TrendingUp className="w-3 h-3 inline mr-0.5" />{k.trend}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-1.5 bg-gray-700 rounded-full">
                    <div className={"h-full rounded-full " + (pct >= 100 ? "bg-green-500" : pct >= 70 ? "bg-yellow-500" : "bg-red-500")} style={{ width: Math.min(pct, 100) + "%" }} />
                  </div>
                  <span className="text-xs text-gray-400 w-10 text-right">{pct.toFixed(0)}%</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Monthly History + Alert Thresholds */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Monthly History</h2>
          <div className="flex items-end gap-2 h-32">
            {(data?.monthly_history ?? []).map((m: any, i: number) => {
              const max = Math.max(...(data?.monthly_history ?? []).map((x) => x.value), 1);
              return (
                <div key={i} className="flex-1 flex flex-col items-center">
                  <div className="w-full bg-blue-500 rounded-t" style={{ height: max > 0 ? (m.value / max) * 100 + "%" : "0" }} />
                  <span className="text-xs text-gray-500 mt-1">{m.month}</span>
                </div>
              );
            })}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-4 h-4 text-yellow-400" />
            Alert Thresholds
          </h2>
          <div className="space-y-2">
            {(data?.alert_thresholds ?? []).map((t) => (
              <div key={t.kpi} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-sm">{t.kpi}</span>
                <span className={"text-xs px-2 py-0.5 rounded " + (t.triggered ? "bg-red-900 text-red-300" : "bg-gray-700 text-gray-400")}>
                  {t.threshold} {t.triggered ? "- BREACHED" : ""}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
