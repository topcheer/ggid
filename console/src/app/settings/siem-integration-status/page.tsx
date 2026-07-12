"use client";

import { useSiemIntegrationStatus } from "@ggid/sdk-react";
import { Activity, Server, AlertTriangle, CheckCircle, RefreshCw } from "lucide-react";

export default function SiemIntegrationStatusPage() {
  const { data, loading, error, refresh, testConnection } = useSiemIntegrationStatus();

  if (loading) return <div className="p-8 text-gray-400">Loading SIEM integration status...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">SIEM Integration Status</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor security event forwarding to SIEM platforms</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Overall Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Total Throughput (events/s)</span>
          </div>
          <p className="text-2xl font-bold">{data?.total_throughput ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Server className="w-4 h-4" />
            <span className="text-xs text-gray-400">Queue Depth</span>
          </div>
          <p className="text-2xl font-bold">{data?.total_queue_depth ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Retry Failures (24h)</span>
          </div>
          <p className="text-2xl font-bold">{data?.total_retry_failures ?? 0}</p>
        </div>
      </div>

      {/* Destinations */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Configured Destinations</h2>
        <div className="space-y-3">
          {(data?.destinations ?? []).map((dest) => (
            <div
              key={dest.id}
              className="bg-gray-800 rounded-lg p-4 border border-gray-700"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  {dest.status === "connected" ? (
                    <CheckCircle className="w-5 h-5 text-green-400" />
                  ) : (
                    <AlertTriangle className="w-5 h-5 text-red-400" />
                  )}
                  <div>
                    <h3 className="font-semibold">{dest.name}</h3>
                    <p className="text-xs text-gray-400">{dest.type}</p>
                  </div>
                </div>
                <button
                  onClick={() => testConnection(dest.id)}
                  className="px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium transition"
                >
                  Test Connection
                </button>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div>
                  <p className="text-xs text-gray-500 mb-1">Throughput</p>
                  <p className="text-sm font-medium">{dest.throughput.toLocaleString()} ev/s</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500 mb-1">Queue Depth</p>
                  <p className="text-sm font-medium">{dest.queue_depth.toLocaleString()}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500 mb-1">Retry Failures</p>
                  <p
                    className={
                      "text-sm font-medium " +
                      (dest.retry_failures > 0 ? "text-red-400" : "text-green-400")
                    }
                  >
                    {dest.retry_failures}
                  </p>
                </div>
              </div>

              <div className="mt-3 flex items-center gap-2">
                <span
                  className={
                    "px-2 py-0.5 rounded text-xs font-medium " +
                    (dest.status === "connected"
                      ? "bg-green-900 text-green-300"
                      : "bg-red-900 text-red-300")
                  }
                >
                  {dest.status}
                </span>
                <span className="text-xs text-gray-500">Last sync: {dest.last_sync}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
