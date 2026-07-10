"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Server, Activity, CheckCircle2, XCircle, RefreshCw, TrendingUp } from "lucide-react";

interface ServiceHealth {
  name: string;
  url: string;
  status: "healthy" | "unhealthy" | "checking";
  latency?: number;
}

interface GatewayStats {
  total_requests?: number;
  total_errors?: number;
  uptime_seconds?: number;
  routes?: Record<string, { requests?: number; errors?: number }>;
}

const SERVICES: Omit<ServiceHealth, "status">[] = [
  { name: "Gateway", url: "http://localhost:8080/healthz" },
  { name: "Identity", url: "http://localhost:8081/healthz" },
  { name: "Auth", url: "http://localhost:9001/healthz" },
  { name: "OAuth", url: "http://localhost:9005/healthz" },
  { name: "Policy", url: "http://localhost:8070/healthz" },
  { name: "Org", url: "http://localhost:8071/healthz" },
  { name: "Audit", url: "http://localhost:8072/healthz" },
];

export default function MonitoringPage() {
  const { apiFetch } = useApi();
  const [services, setServices] = useState<ServiceHealth[]>([]);
  const [gwStats, setGwStats] = useState<GatewayStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Check each service healthz
      const healthChecks = await Promise.all(
        SERVICES.map(async (svc) => {
          const start = Date.now();
          try {
            const resp = await fetch(svc.url, { signal: AbortSignal.timeout(3000) });
            return {
              ...svc,
              status: resp.ok ? "healthy" as const : "unhealthy" as const,
              latency: Date.now() - start,
            };
          } catch {
            return { ...svc, status: "unhealthy" as const, latency: Date.now() - start };
          }
        })
      );
      setServices(healthChecks);

      // Load gateway stats
      try {
        const stats = await apiFetch<GatewayStats>("/api/v1/gateway/stats");
        setGwStats(stats);
      } catch {
        setGwStats(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const healthyCount = services.filter((s) => s.status === "healthy").length;
  const totalReqs = gwStats?.total_requests || 0;
  const totalErrors = gwStats?.total_errors || 0;
  const errorRate = totalReqs > 0 ? ((totalErrors / totalReqs) * 100).toFixed(2) : "0.00";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">System Monitoring</h1>
        <button
          onClick={loadData}
          disabled={loading}
          className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* Overview cards */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <OverviewCard
          icon={Server}
          label="Services Healthy"
          value={`${healthyCount}/${services.length || SERVICES.length}`}
          color={healthyCount === SERVICES.length ? "green" : healthyCount > 0 ? "amber" : "red"}
        />
        <OverviewCard
          icon={Activity}
          label="Total Requests"
          value={totalReqs.toLocaleString()}
          color="blue"
        />
        <OverviewCard
          icon={XCircle}
          label="Error Rate"
          value={`${errorRate}%`}
          color={parseFloat(errorRate) > 5 ? "red" : "green"}
        />
        <OverviewCard
          icon={TrendingUp}
          label="Uptime"
          value={gwStats?.uptime_seconds ? formatUptime(gwStats.uptime_seconds) : "-"}
          color="green"
        />
      </div>

      {/* Service health table */}
      <div className="mt-6 rounded-xl border border-gray-200 bg-white shadow-sm">
        <div className="border-b border-gray-200 p-4">
          <h2 className="text-sm font-semibold">Service Health</h2>
        </div>
        <table className="w-full">
          <thead className="border-b border-gray-100 bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Service</th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Status</th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Latency</th>
              <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">URL</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {(services.length > 0 ? services : SERVICES.map((s) => ({ ...s, status: "checking" as const }))).map((svc) => (
              <tr key={svc.name} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium">{svc.name}</td>
                <td className="px-4 py-3">
                  <span className={`flex items-center gap-1.5 text-xs font-medium ${
                    svc.status === "healthy" ? "text-green-700" : svc.status === "checking" ? "text-gray-400" : "text-red-700"
                  }`}>
                    {svc.status === "healthy" ? (
                      <CheckCircle2 className="h-4 w-4" />
                    ) : svc.status === "checking" ? (
                      <RefreshCw className="h-4 w-4 animate-spin" />
                    ) : (
                      <XCircle className="h-4 w-4" />
                    )}
                    {svc.status === "healthy" ? "Healthy" : svc.status === "checking" ? "Checking..." : "Unhealthy"}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-500">
                  {svc.latency != null ? `${svc.latency}ms` : "-"}
                </td>
                <td className="px-4 py-3 font-mono text-xs text-gray-400">{svc.url}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Route stats */}
      {gwStats?.routes && Object.keys(gwStats.routes).length > 0 && (
        <div className="mt-6 rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="border-b border-gray-200 p-4">
            <h2 className="text-sm font-semibold">Route Statistics</h2>
          </div>
          <table className="w-full">
            <thead className="border-b border-gray-100 bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Route</th>
                <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Requests</th>
                <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Errors</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {Object.entries(gwStats.routes).map(([route, stats]) => (
                <tr key={route} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-mono text-xs">{route}</td>
                  <td className="px-4 py-3 text-sm">{stats.requests || 0}</td>
                  <td className="px-4 py-3 text-sm text-red-600">{stats.errors || 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function OverviewCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  color: "green" | "red" | "amber" | "blue";
}) {
  const colorMap = {
    green: { bg: "bg-green-100", text: "text-green-600" },
    red: { bg: "bg-red-100", text: "text-red-600" },
    amber: { bg: "bg-amber-100", text: "text-amber-600" },
    blue: { bg: "bg-blue-100", text: "text-blue-600" },
  };
  const c = colorMap[color];
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold">{value}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  );
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days}d ${hours}h`;
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${mins}m`;
}
