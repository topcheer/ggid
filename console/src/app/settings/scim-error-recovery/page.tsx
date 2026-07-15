"use client";

import { useScimErrorRecovery } from "@ggid/sdk-react";
import { AlertTriangle, RotateCw, CheckCircle, Settings } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function ScimErrorRecoveryPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, retryError, bulkRetry } = useScimErrorRecovery();

  if (loading) return <div className="p-8 text-gray-400">Loading SCIM error recovery...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const statusColors: Record<string, string> = {
    pending: "bg-yellow-900 text-yellow-300",
    retrying: "bg-blue-900 text-blue-300",
    resolved: "bg-green-900 text-green-300",
    failed: "bg-red-900 text-red-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">SCIM Error Recovery</h1>
          <p className="text-sm text-gray-400 mt-1">Manage and retry failed SCIM provisioning operations</p>
        </div>
        <button onClick={bulkRetry} className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">
          <RotateCw className="w-4 h-4" /> Bulk Retry All
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        {(["pending", "retrying", "resolved", "failed"] as const).map((s) => (
          <div key={s} className="bg-gray-900 rounded-xl p-4">
            <p className="text-xs text-gray-400 capitalize">{s}</p>
            <p className={"text-xl font-bold " + (s === "failed" ? "text-red-400" : s === "resolved" ? "text-green-400" : s === "pending" ? "text-yellow-400" : "text-blue-400")}>
              {data?.error_queue?.filter((e) => e.status === s).length ?? 0}
            </p>
          </div>
        ))}
      </div>

      {/* Error Queue */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Error Queue</h2>
        <div className="space-y-2">
          {(data?.error_queue ?? []).map((e) => (
            <div key={e.id} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{e.operation}</span>
                  <span className={"text-xs px-1.5 py-0.5 rounded " + (statusColors[e.status] ?? "bg-gray-700")}>{e.status}</span>
                </div>
                <p className="text-xs text-gray-400 mt-0.5">Target: {e.target_app} - Error: {e.error_type} (retries: {e.retry_count})</p>
                <p className="text-xs text-gray-500">{e.timestamp}</p>
              </div>
              {e.status !== "resolved" && (
                <button onClick={() => retryError(e.id)} className="text-xs px-2 py-1 bg-gray-700 hover:bg-gray-600 rounded flex items-center gap-1">
                  <RotateCw className="w-3 h-3" /> Retry
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Error Patterns + Auto Retry Config */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Most Common Errors</h2>
          <div className="space-y-2">
            {(data?.error_patterns ?? []).map((p) => (
              <div key={p.error_type} className="flex items-center justify-between bg-gray-800 rounded p-2">
                <span className="text-xs font-mono text-red-400">{p.error_type}</span>
                <span className="text-xs text-gray-400">{p.count} occurrences</span>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Settings className="w-4 h-4" /> Auto-Retry Config</h2>
          <div className="space-y-2">
            <div className="flex justify-between"><span className="text-xs text-gray-400">Max Retries</span><span className="text-sm">{data?.auto_retry_config?.max_retries ?? 5}</span></div>
            <div className="flex justify-between"><span className="text-xs text-gray-400">Backoff</span><span className="text-sm">{data?.auto_retry_config?.backoff_seconds ?? 30}s exponential</span></div>
            <div className="flex justify-between"><span className="text-xs text-gray-400">Manual Override</span><span className="text-sm">{data?.auto_retry_config?.manual_override ? "Enabled" : "Disabled"}</span></div>
          </div>
        </div>
      </div>
    </div>
  );
}
