"use client";

import { useSecurityMetricsDash } from "@ggid/sdk-react";
import { Clock, AlertTriangle, Shield, Download, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SecurityMetricsDashPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useSecurityMetricsDash();

  if (loading) return <div className="p-8 text-gray-400">Loading security metrics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Security Metrics Dashboard</h1>
          <p className="text-sm text-gray-400 mt-1">Executive security metrics and SLA tracking</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" /> Export Summary
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* MTTD / MTTR / Open Vulns / Patch Compliance */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">MTTD (Mean Time to Detect)</p>
          <p className="text-xl font-bold text-blue-400">{data?.mttd_minutes ?? 0}m</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">MTTR (Mean Time to Resolve)</p>
          <p className="text-xl font-bold text-green-400">{data?.mttr_hours ?? 0}h</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Open Vulnerabilities</p>
          <p className="text-xl font-bold text-red-400">{data?.open_vulns ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Patch Compliance</p>
          <p className="text-xl font-bold">{data?.patch_compliance_pct ?? 0}%</p>
        </div>
      </div>

      {/* Incidents 30d Trend + SLA Breaches */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6 lg:col-span-2">
          <h2 className="text-sm font-semibold mb-4">Incidents (30 days)</h2>
          <div className="flex items-end gap-1 h-32">
            {(data?.incidents_30d ?? []).map((v, i) => {
              const max = Math.max(...(data?.incidents_30d ?? [1]));
              return <div key={i} className="flex-1 bg-blue-500 rounded-t" style={{ height: max > 0 ? (v / max) * 100 + "%" : "0" }} />;
            })}
          </div>
          <p className="text-xs text-gray-500 mt-2">Total: {data?.incidents_30d?.reduce((a, b) => a + b, 0) ?? 0} incidents</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">SLA Breaches</h2>
          <p className="text-3xl font-bold text-red-400">{data?.sla_breaches ?? 0}</p>
          <p className="text-xs text-gray-400 mt-1">in last 30 days</p>
          <div className="mt-3 h-2 bg-gray-800 rounded-full">
            <div className="h-full bg-red-500 rounded-full" style={{ width: Math.min((data?.sla_breaches ?? 0) * 10, 100) + "%" }} />
          </div>
        </div>
      </div>

      {/* Top 10 Risks */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">Top 10 Risks</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">#</th>
                <th scope="col" className="text-left py-2 pr-3">Risk</th>
                <th scope="col" className="text-left py-2 pr-3">Category</th>
                <th scope="col" className="text-left py-2 pr-3">Score</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.top_10_risks ?? []).map((r, i) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs text-gray-500">{i + 1}</td>
                  <td className="py-3 pr-3 text-xs font-medium">{r.risk}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{r.category}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs font-bold " + (r.score >= 8 ? "text-red-400" : r.score >= 5 ? "text-yellow-400" : "text-green-400")}>{r.score}</span>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      r.status === "open" ? "bg-red-900 text-red-300" :
                      r.status === "mitigated" ? "bg-yellow-900 text-yellow-300" :
                      "bg-green-900 text-green-300"
                    )}>{r.status}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
