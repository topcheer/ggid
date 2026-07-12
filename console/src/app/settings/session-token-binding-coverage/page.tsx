"use client";

import { useSessionTokenBindingCoverage } from "@ggid/sdk-react";
import { Shield, AlertTriangle } from "lucide-react";

export default function SessionTokenBindingCoveragePage() {
  const { data, loading, error, refresh } = useSessionTokenBindingCoverage();

  if (loading) return <div className="p-8 text-gray-400">Loading token binding coverage...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Token Binding Coverage</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor DPoP/mTLS token binding adoption</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Coverage Gauge + Donut */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6 text-center">
          <Shield className="w-8 h-8 text-blue-400 mx-auto mb-2" />
          <p className="text-xs text-gray-400 mb-1">Overall Binding Coverage</p>
          <p className={"text-4xl font-bold " + ((data?.coverage_pct ?? 0) >= 80 ? "text-green-400" : "text-yellow-400")}>{data?.coverage_pct ?? 0}%</p>
          <div className="w-full bg-gray-800 rounded-full h-3 mt-3">
            <div className="bg-blue-600 h-3 rounded-full" style={{ width: (data?.coverage_pct ?? 0) + "%" }} />
          </div>
          <p className="text-xs text-gray-500 mt-2">Threshold: {data?.compliance_threshold ?? 90}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h3 className="text-sm font-semibold mb-4">Bound vs Unbound</h3>
          <div className="flex items-center gap-6">
            <svg width="120" height="120" viewBox="0 0 120 120">
              <circle cx="60" cy="60" r="45" fill="none" stroke="#1f2937" strokeWidth="12" />
              <circle cx="60" cy="60" r="45" fill="none" stroke="#3b82f6" strokeWidth="12" strokeDasharray={((data?.coverage_pct ?? 0) / 100) * 283 + " " + 283} transform="rotate(-90 60 60)" />
              <text x="60" y="65" textAnchor="middle" className="fill-white text-lg font-bold">{data?.coverage_pct ?? 0}%</text>
            </svg>
            <div className="space-y-2">
              <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-blue-600" /><span className="text-xs">Bound: {data?.bound_tokens ?? 0}</span></div>
              <div className="flex items-center gap-2"><span className="w-3 h-3 rounded-full bg-gray-700" /><span className="text-xs">Unbound: {data?.unbound_tokens ?? 0}</span></div>
            </div>
          </div>
        </div>
      </div>

      {/* Per-Client Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Per-Client Coverage</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b border-gray-800 text-gray-400">
            <th className="text-left py-2 pr-3">Client</th><th className="text-left py-2 pr-3">Bound %</th><th className="text-left py-2 pr-3">Method</th><th className="text-left py-2 pr-3">Last Checked</th>
          </tr></thead>
          <tbody>
            {(data?.per_client ?? []).map((c) => (
              <tr key={c.client} className="border-b border-gray-800">
                <td className="py-3 pr-3 text-sm">{c.client}</td>
                <td className="py-3 pr-3">
                  <div className="flex items-center gap-2"><div className="w-20 bg-gray-800 rounded-full h-1.5"><div className={"h-1.5 rounded-full " + (c.bound_pct >= 90 ? "bg-green-600" : c.bound_pct >= 50 ? "bg-yellow-600" : "bg-red-600")} style={{ width: c.bound_pct + "%" }} /></div><span className="text-xs">{c.bound_pct}%</span></div>
                </td>
                <td className="py-3 pr-3 text-xs text-gray-400">{c.binding_method}</td>
                <td className="py-3 pr-3 text-xs text-gray-400">{c.last_checked}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Unbound Tokens */}
      {data?.unbound_list && data.unbound_list.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-red-400" /> Unbound Active Tokens</h2>
          <div className="space-y-1">
            {data.unbound_list.map((t) => (
              <div key={t.token_id} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
                <span className="font-mono text-red-400">{t.token_id}</span>
                <span className="text-gray-400">Client: {t.client}</span>
                <span className="text-gray-500 ml-auto">Issued: {t.issued_at}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
