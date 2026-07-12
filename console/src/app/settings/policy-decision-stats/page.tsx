"use client";

import { usePolicyDecisionStats } from "@ggid/sdk-react";
import { CheckCircle, XCircle, Zap } from "lucide-react";

export default function PolicyDecisionStatsPage() {
  const { data, loading, error, refresh } = usePolicyDecisionStats();

  if (loading) return <div className="p-8 text-gray-400">Loading decision stats...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const allow = data?.allow_count ?? 0;
  const deny = data?.deny_count ?? 0;
  const total = allow + deny;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Policy Decision Statistics</h1><p className="text-sm text-gray-400 mt-1">Allow/deny ratios and evaluation metrics</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Donut + Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6 flex items-center justify-center">
          <svg width="160" height="160" viewBox="0 0 160 160">
            <circle cx="80" cy="80" r="55" fill="none" stroke="#1f2937" strokeWidth="16" />
            <circle cx="80" cy="80" r="55" fill="none" stroke="#22c55e" strokeWidth="16" strokeDasharray={(allow / total) * 346 + " " + 346} transform="rotate(-90 80 80)" />
            <circle cx="80" cy="80" r="55" fill="none" stroke="#ef4444" strokeWidth="16" strokeDasharray={(deny / total) * 346 + " " + 346} strokeDashoffset={-(allow / total) * 346} transform="rotate(-90 80 80)" />
            <text x="80" y="78" textAnchor="middle" className="fill-white text-sm">{total.toLocaleString()}</text>
            <text x="80" y="95" textAnchor="middle" className="fill-gray-400 text-xs">decisions</text>
          </svg>
        </div>
        <div className="bg-gray-900 rounded-xl p-6 space-y-3">
          <div className="flex items-center gap-2"><CheckCircle className="w-4 h-4 text-green-400" /><span className="text-sm">Allow: {allow.toLocaleString()}</span></div>
          <div className="flex items-center gap-2"><XCircle className="w-4 h-4 text-red-400" /><span className="text-sm">Deny: {deny.toLocaleString()}</span></div>
          <div className="flex items-center gap-2"><Zap className="w-4 h-4 text-purple-400" /><span className="text-sm">Cache Hit Rate: {data?.cache_hit_rate ?? 0}%</span></div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6 text-center">
          <p className="text-xs text-gray-400">Avg Eval Time</p>
          <p className={"text-3xl font-bold " + ((data?.avg_eval_time_ms ?? 0) > 100 ? "text-yellow-400" : "text-green-400")}>{data?.avg_eval_time_ms ?? 0}ms</p>
        </div>
      </div>

      {/* By Policy + By Resource */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">By Policy</h2>
          <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">Policy</th><th className="text-right py-2">Allow</th><th className="text-right py-2">Deny</th></tr></thead>
            <tbody>{(data?.by_policy ?? []).map((p) => <tr key={p.policy} className="border-b border-gray-800"><td className="py-2 text-sm">{p.policy}</td><td className="py-2 text-right text-green-400">{p.allow}</td><td className="py-2 text-right text-red-400">{p.deny}</td></tr>)}</tbody>
          </table>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">By Resource Type</h2>
          <div className="space-y-2">{(data?.by_resource_type ?? []).map((r) => (
            <div key={r.type} className="flex items-center gap-2"><span className="text-xs w-32">{r.type}</span><div className="flex-1 bg-gray-800 rounded-full h-2"><div className="bg-blue-600 h-2 rounded-full" style={{ width: r.pct + "%" }} /></div><span className="text-xs text-gray-400">{r.pct}%</span></div>
          ))}</div>
        </div>
      </div>

      {/* Top Denied Actions */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-3">Top Denied Actions</h2>
        <div className="space-y-1">{(data?.top_denied_actions ?? []).map((a) => (
          <div key={a.action} className="flex items-center gap-2 bg-gray-800 rounded p-2"><XCircle className="w-3 h-3 text-red-400" /><span className="text-xs font-mono flex-1">{a.action}</span><span className="text-xs text-gray-400">{a.count}</span></div>
        ))}</div>
      </div>
    </div>
  );
}
