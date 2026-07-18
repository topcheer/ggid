"use client";

import { useAuditEventCorrelation } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Network, Zap, GitBranch } from "lucide-react";

export default function AuditEventCorrelationPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditEventCorrelation();

  if (loading) return <div className="p-8 text-gray-400">Loading event correlation...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Event Correlation</h1>
          <p className="text-sm text-gray-400 mt-1">Detect patterns across correlated events</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Engine Status */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4"><Network className="w-5 h-5 text-green-400 mb-1" /><p className="text-xs text-gray-400">Engine Status</p><p className={"text-sm font-bold " + (data?.engine_status === "running" ? "text-green-400" : "text-yellow-400")}>{data?.engine_status ?? "--"}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Correlated Incidents (24h)</p><p className="text-xl font-bold text-blue-400">{data?.correlated_incidents?.length ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Active Rules</p><p className="text-xl font-bold">{data?.correlation_rules?.length ?? 0}</p></div>
      </div>

      {/* Correlated Incidents */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><GitBranch className="w-4 h-4 text-blue-400" /> Correlated Incidents</h2>
        <div className="space-y-3">
          {(data?.correlated_incidents ?? []).map((inc) => (
            <div key={inc.id} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-semibold">{inc.title}</h3>
                <span className={"text-xs px-2 py-0.5 rounded " + (inc.severity === "critical" ? "bg-red-900 text-red-300" : inc.severity === "high" ? "bg-orange-900 text-orange-300" : "bg-yellow-900 text-yellow-300")}>{inc.severity}</span>
              </div>
              <p className="text-xs text-gray-400 mb-2">{inc.description}</p>
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <span>{inc.event_count} events correlated</span>
                <span>via {inc.correlation_key}</span>
                <span className="ml-auto">{inc.timestamp}</span>
              </div>
              {/* Event chain */}
              <div className="flex items-center gap-1 mt-2 flex-wrap">
                {inc.events.map((ev: any, i: number) => (
                  <span key={i} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded font-mono text-gray-400">{ev}</span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Correlation Rules */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Zap className="w-4 h-4 text-yellow-400" /> Correlation Rules</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b border-gray-800 text-gray-400">
            <th scope="col" className="text-left py-2 pr-3">Rule</th><th className="text-left py-2 pr-3">Window</th><th className="text-left py-2 pr-3">Min Events</th><th className="text-left py-2 pr-3">Action</th>
          </tr></thead>
          <tbody>
            {(data?.correlation_rules ?? []).map((r) => (
              <tr key={r.rule} className="border-b border-gray-800">
                <td className="py-3 pr-3 text-sm">{r.rule}</td>
                <td className="py-3 pr-3 text-xs text-gray-400">{r.window}</td>
                <td className="py-3 pr-3 text-xs text-gray-400">{r.min_events}</td>
                <td className="py-3 pr-3"><span className="text-xs px-2 py-0.5 rounded bg-blue-900 text-blue-300">{r.action}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
