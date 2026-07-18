"use client";

import { useAnomalyDetectDashboard } from "@ggid/sdk-react";
import { AlertTriangle, CheckCircle, XCircle, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AnomalyDetectDashboardPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, acknowledge } = useAnomalyDetectDashboard();

  if (loading) return <div className="p-8 text-gray-400">Loading anomaly dashboard...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const sevColors: Record<string, string> = { critical: "bg-red-900 text-red-300", high: "bg-orange-900 text-orange-300", medium: "bg-yellow-900 text-yellow-300", low: "bg-blue-900 text-blue-300" };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Anomaly Detection Dashboard</h1><p className="text-sm text-gray-400 mt-1">Real-time behavioral anomaly detection</p></div>
        <button onClick={refresh} aria-label="Refresh anomaly data" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        {(["critical", "high", "medium", "low"] as const).map((s: any) => {
          const count = data?.events?.filter((e: any) => e.severity === s).length ?? 0;
          return <div key={s} className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400 capitalize">{s}</p><p className={"text-xl font-bold " + (s === "critical" ? "text-red-400" : s === "high" ? "text-orange-400" : s === "medium" ? "text-yellow-400" : "text-blue-400")}>{count}</p></div>;
        })}
      </div>

      {/* Events Feed + Patterns */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-2 bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Activity className="w-4 h-4 text-green-400" /> Anomaly Events</h2>
          <div className="space-y-2">
            {(data?.events ?? []).map((e: any) => (
              <div key={e.id} className="flex items-start gap-3 bg-gray-800 rounded-lg p-3">
                <AlertTriangle className={"w-4 h-4 mt-0.5 " + (e.severity === "critical" ? "text-red-400" : "text-yellow-400")} />
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{e.type}</span>
                    <span className={"text-xs px-1.5 py-0.5 rounded " + (sevColors[e.severity] ?? "bg-gray-700")}>{e.severity}</span>
                  </div>
                  <p className="text-xs text-gray-400 mt-0.5">User: {e.user} - {e.description}</p>
                  <p className="text-xs text-gray-500">{e.timestamp} - Confidence: {e.confidence}%</p>
                </div>
                <button onClick={() => acknowledge(e.id)} className="text-xs px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded flex items-center gap-1"><CheckCircle className="w-3 h-3" /> Ack</button>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Detected Patterns</h2>
          <div className="space-y-2">
            {(data?.patterns ?? []).map((p: any) => (
              <div key={p.pattern} className="bg-gray-800 rounded p-2">
                <div className="flex items-center justify-between"><span className="text-xs font-medium">{p.pattern}</span><span className="text-xs text-gray-400">{p.count}</span></div>
                {p.auto_action && <span className="text-xs text-green-400">Auto: {p.auto_action}</span>}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
