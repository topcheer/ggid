"use client";
import { useTranslations } from "@/lib/i18n";

import { useOAuthScopeAnalytics } from "@ggid/sdk-react";
import { BarChart3, Grid3x3, TrendingUp, AlertCircle } from "lucide-react";

export default function OAuthScopeAnalyticsPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useOAuthScopeAnalytics();

  if (loading) return <div className="p-8 text-gray-400">Loading scope analytics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const heatColors = ["bg-gray-800", "bg-blue-900/50", "bg-blue-700/60", "bg-blue-500/70", "bg-blue-400/80"];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Scope Analytics</h1>
          <p className="text-sm text-gray-400 mt-1">Analyze OAuth scope usage, correlations, and trends</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <BarChart3 className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Total Scopes</p>
          <p className="text-xl font-bold">{data?.scope_usage?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <TrendingUp className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Total Requests (30d)</p>
          <p className="text-xl font-bold">{data?.scope_usage?.reduce((a, s) => a + s.requested_count, 0).toLocaleString() ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertCircle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Unused Scopes</p>
          <p className="text-xl font-bold text-red-400">{data?.unused_scopes?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Grid3x3 className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Correlations</p>
          <p className="text-xl font-bold">{data?.scope_correlation?.length ?? 0}</p>
        </div>
      </div>

      {/* Scope Usage Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Scope Usage (30d)</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Scope</th>
                <th scope="col" className="text-right py-2 pr-3">Requested</th>
                <th scope="col" className="text-right py-2 pr-3">Granted</th>
                <th scope="col" className="text-right py-2 pr-3">Denied</th>
                <th scope="col" className="text-left py-2 pr-3">Deny Reasons</th>
                <th scope="col" className="text-right py-2 pr-3">Avg/Token</th>
              </tr>
            </thead>
            <tbody>
              {(data?.scope_usage ?? []).map((s: any) => {
                const denyPct = s.requested_count > 0 ? (s.denied_count / s.requested_count) * 100 : 0;
                return (
                  <tr key={s.scope_name} className="border-b border-gray-800">
                    <td className="py-3 pr-3 font-mono text-xs text-blue-400">{s.scope_name}</td>
                    <td className="py-3 pr-3 text-right text-gray-300">{s.requested_count.toLocaleString()}</td>
                    <td className="py-3 pr-3 text-right text-green-400">{s.granted_count.toLocaleString()}</td>
                    <td className="py-3 pr-3 text-right">
                      <span className={denyPct > 10 ? "text-red-400 font-medium" : "text-gray-400"}>{s.denied_count.toLocaleString()}</span>
                    </td>
                    <td className="py-3 pr-3">
                      <div className="flex flex-wrap gap-1">
                        {s.deny_reasons.map((r: any) => (
                          <span key={r} className="text-xs px-1.5 py-0.5 bg-gray-800 rounded">{r}</span>
                        ))}
                      </div>
                    </td>
                    <td className="py-3 pr-3 text-right text-gray-400">{s.avg_per_token.toFixed(1)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Correlation Heatmap */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Scope Correlation Heatmap</h2>
          <div className="overflow-x-auto">
            <div className="inline-block">
              {(data?.scope_correlation ?? []).map((row: any, i: number) => (
                <div key={i} className="flex">
                  {row.map((val: any, j: number) => (
                    <div
                      key={j}
                      className={"w-12 h-12 flex items-center justify-center text-xs font-medium border border-gray-800 " + heatColors[Math.min(Math.floor(val * 5), 4)]}
                      title={"Correlation: " + (val * 100).toFixed(0) + "%"}
                    >
                      {(val * 100).toFixed(0)}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Unused Scopes */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <AlertCircle className="w-4 h-4 text-red-400" />
            Unused Scopes (0 requests in 30d)
          </h2>
          <div className="flex flex-wrap gap-2">
            {(data?.unused_scopes ?? []).map((s: any) => (
              <span key={s} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700 font-mono text-gray-400">
                {s}
              </span>
            ))}
            {(data?.unused_scopes?.length ?? 0) === 0 && (
              <p className="text-xs text-gray-500">All scopes have been requested</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
