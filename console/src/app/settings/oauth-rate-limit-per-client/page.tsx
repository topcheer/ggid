"use client";

import { useOAuthRateLimitPerClient } from "@ggid/sdk-react";
import { Gauge, Shield, AlertTriangle, Activity, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthRateLimitPerClientPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useOAuthRateLimitPerClient();

  if (loading) return <div className="p-8 text-gray-400">Loading rate limits...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Per-Client Rate Limiting</h1>
          <p className="text-sm text-gray-400 mt-1">Configure rate limits, quotas, and throttling per OAuth client</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Throttle Response Config */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-blue-400" />
              <span className="text-sm text-gray-400">Throttle Response:</span>
              <span className="text-sm font-medium">HTTP {data?.throttle_response.status_code ?? 429}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-400">Retry-After:</span>
              <span className="text-sm font-medium">{data?.throttle_response.retry_after_seconds ?? 60}s</span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400">Whitelist IPs:</span>
            <div className="flex gap-1">
              {(data?.whitelist_ips ?? []).map((ip: any) => (
                <span key={ip} className="text-xs font-mono px-2 py-0.5 bg-green-900 text-green-300 rounded">{ip}</span>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* Rate Limits Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Gauge className="w-5 h-5 text-blue-400" />
          Rate Limits
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Client ID</th>
                <th scope="col" className="text-left py-2 pr-3">Req/min</th>
                <th scope="col" className="text-left py-2 pr-3">Burst</th>
                <th scope="col" className="text-left py-2 pr-3">Concurrent Tokens</th>
                <th scope="col" className="text-left py-2 pr-3">Daily Quota</th>
                <th scope="col" className="text-left py-2 pr-3">Current Usage</th>
              </tr>
            </thead>
            <tbody>
              {(data?.rate_limits ?? []).map((r: any) => {
                const usagePct = r.daily_quota > 0 ? Math.round((r.current_usage_today / r.daily_quota) * 100) : 0;
                return (
                  <tr key={r.client_id} className="border-b border-gray-800">
                    <td className="py-3 pr-3 font-mono text-xs text-blue-400">{r.client_id}</td>
                    <td className="py-3 pr-3 text-gray-300">{r.requests_per_min}</td>
                    <td className="py-3 pr-3 text-gray-300">{r.burst}</td>
                    <td className="py-3 pr-3 text-gray-300">{r.concurrent_tokens}</td>
                    <td className="py-3 pr-3 text-gray-300">{r.daily_quota.toLocaleString()}</td>
                    <td className="py-3 pr-3">
                      <div className="flex items-center gap-2">
                        <div className="w-16 bg-gray-700 rounded-full h-1.5">
                          <div
                            className={usagePct > 80 ? "bg-red-500" : usagePct > 60 ? "bg-yellow-500" : "bg-green-500"}
                            style={{ width: Math.min(usagePct, 100) + "%", height: "100%", borderRadius: "9999px" }}
                          />
                        </div>
                        <span className="text-xs text-gray-400">{usagePct}%</span>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Per-Endpoint Override */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Activity className="w-5 h-5 text-purple-400" />
          Per-Endpoint Override
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Client</th>
                <th scope="col" className="text-left py-2 pr-3">Endpoint</th>
                <th scope="col" className="text-left py-2 pr-3">Override Req/min</th>
                <th scope="col" className="text-left py-2 pr-3">Override Burst</th>
              </tr>
            </thead>
            <tbody>
              {(data?.per_endpoint_override ?? []).map((o: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{o.client_id}</td>
                  <td className="py-3 pr-3 text-gray-300 font-mono text-xs">{o.endpoint}</td>
                  <td className="py-3 pr-3 text-gray-300">{o.override_req_per_min ?? "-"}</td>
                  <td className="py-3 pr-3 text-gray-300">{o.override_burst ?? "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
