"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { DEFAULT_TENANT_ID } from "@/lib/api-config";
import { useI18n } from "@/lib/i18n";
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
  { name: "Gateway", url: "/healthz" },
  { name: "Identity", url: "/healthz/deep" },
  { name: "Auth", url: "/healthz/deep" },
  { name: "OAuth", url: "/healthz/deep" },
  { name: "Policy", url: "/healthz/deep" },
  { name: "Org", url: "/healthz/deep" },
  { name: "Audit", url: "/healthz/deep" },
];

export default function MonitoringPage() {
  const { apiFetch } = useApi();
  const { t } = useI18n();
  const [services, setServices] = useState<ServiceHealth[]>([]);
  const [gwStats, setGwStats] = useState<GatewayStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Use gateway /healthz/deep for all service checks (works in all deployments)
      const start = Date.now();
      try {
        const resp = await fetch("/healthz/deep", {
          signal: AbortSignal.timeout(5000),
          headers: { "X-Tenant-ID": DEFAULT_TENANT_ID },
        });
        const deepData = await resp.json();
        const allServices = deepData.services || {};
        const healthChecks: ServiceHealth[] = Object.entries(allServices).map(([key, svc]: [string, any]) => ({
          name: key.charAt(0).toUpperCase() + key.slice(1),
          url: key,
          status: svc.status === "healthy" ? "healthy" as const : "unhealthy" as const,
          latency: svc.latency_ms || (Date.now() - start),
        }));
        setServices(healthChecks);
      } catch {
        // Fallback to simple gateway check
        const healthChecks = SERVICES.map(svc => ({ ...svc, status: "unhealthy" as const, latency: 0 }));
        setServices(healthChecks);
      }

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

  const healthyCount = services.filter((s: any) => s.status === "healthy").length;
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
          className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
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
          label={t("monitoring.servicesHealthy")}
          value={`${healthyCount}/${services.length || SERVICES.length}`}
          color={healthyCount === SERVICES.length ? "green" : healthyCount > 0 ? "amber" : "red"}
        />
        <OverviewCard
          icon={Activity}
          label={t("monitoring.totalRequests")}
          value={totalReqs.toLocaleString()}
          color="blue"
        />
        <OverviewCard
          icon={XCircle}
          label={t("monitoring.errorRate")}
          value={`${errorRate}%`}
          color={parseFloat(errorRate) > 5 ? "red" : "green"}
        />
        <OverviewCard
          icon={TrendingUp}
          label={t("monitoring.uptime")}
          value={gwStats?.uptime_seconds ? formatUptime(gwStats.uptime_seconds) : "-"}
          color="green"
        />
      </div>

      {/* Service health table */}
      <div className="mt-6 rounded-xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 shadow-sm">
        <div className="border-b border-gray-200 p-4">
          <h2 className="text-sm font-semibold">Service Health</h2>
        </div>
        <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-b border-gray-100 bg-gray-50 dark:border-gray-800 dark:bg-gray-800">
            <tr>
              <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Service</th>
              <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Status</th>
              <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Latency</th>
              <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">URL</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {(services.length > 0 ? services : SERVICES.map((s: any) => ({ ...s, status: "checking" as const }))).map((svc: any) => (
              <tr key={svc.name} className="hover:bg-gray-50 dark:hover:bg-gray-700 dark:bg-gray-800">
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
                    {svc.status === "healthy" ? t("monitoring.healthy") : svc.status === "checking" ? t("monitoring.checking") : t("monitoring.unhealthy")}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                  {svc.latency != null ? `${svc.latency}ms` : "-"}
                </td>
                <td className="px-4 py-3 font-mono text-xs text-gray-400">{svc.url}</td>
              </tr>
            ))}
          </tbody>
        </table>
        </div>
      </div>

      {/* Route stats */}
      {gwStats?.routes && Object.keys(gwStats.routes).length > 0 && (
        <div className="mt-6 rounded-xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 shadow-sm">
          <div className="border-b border-gray-200 p-4">
            <h2 className="text-sm font-semibold">Route Statistics</h2>
          </div>
          <table className="w-full">
            <thead className="border-b border-gray-100 bg-gray-50 dark:bg-gray-800">
              <tr>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Route</th>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Requests</th>
                <th scope="col" className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Errors</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {Object.entries(gwStats.routes).map(([route, stats]: any[]) => (
                <tr key={route} className="hover:bg-gray-50 dark:hover:bg-gray-700 dark:bg-gray-800">
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
    <div className="rounded-xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold">{value}</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">{label}</p>
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
