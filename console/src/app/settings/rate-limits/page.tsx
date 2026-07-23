"use client";

import { useState, useEffect, useCallback } from "react";
import {
  Gauge, Activity, Zap, Clock, TrendingUp, RefreshCw,
  Loader2, AlertCircle, Shield, Server,
} from "lucide-react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";

interface RateLimitConfig {
  login_rate_limit: number;
  login_window_seconds: number;
  register_rate_limit: number;
  register_window_seconds: number;
  ip_rate_limit: number;
  ip_window_seconds: number;
  tenant_rate_limit: number;
  tenant_window_seconds: number;
}

interface ClientRateLimit {
  client_id: string;
  requests_per_min: number;
  burst: number;
  strategy: string;
  current_usage?: number;
}

const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

export default function RateLimitPanelPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [authLimits, setAuthLimits] = useState<RateLimitConfig | null>(null);
  const [clientLimits, setClientLimits] = useState<ClientRateLimit[]>([]);

  const loadData = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true); else setLoading(true);
    setError(null);
    try {
      const [authRes, clientRes] = await Promise.allSettled([
        apiFetch<RateLimitConfig>("/api/v1/auth/rate-limits"),
        apiFetch<{ configs?: ClientRateLimit[] } | ClientRateLimit[]>("/api/v1/oauth/client-rate-limits"),
      ]);

      if (authRes.status === "fulfilled") setAuthLimits(authRes.value);
      if (clientRes.status === "fulfilled") {
        const val = clientRes.value;
        setClientLimits(Array.isArray(val) ? val : (val?.configs || []));
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load rate limit data");
    }
    if (isRefresh) setRefreshing(false); else setLoading(false);
  }, [apiFetch]);

  useEffect(() => { loadData(); }, [loadData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-brand-500" />
        <span className="ml-2 text-sm text-gray-500">Loading rate limit configuration...</span>
      </div>
    );
  }

  const limits = authLimits ? [
    { label: "Login Attempts", icon: Shield, value: authLimits.login_rate_limit, window: authLimits.login_window_seconds, color: "red" },
    { label: "Registration", icon: Activity, value: authLimits.register_rate_limit, window: authLimits.register_window_seconds, color: "amber" },
    { label: "Per IP", icon: Server, value: authLimits.ip_rate_limit, window: authLimits.ip_window_seconds, color: "blue" },
    { label: "Per Tenant", icon: Gauge, value: authLimits.tenant_rate_limit, window: authLimits.tenant_window_seconds, color: "purple" },
  ] : [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Gauge className="h-6 w-6 text-brand-600" /> Rate Limit Dashboard
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Authentication rate limiting configuration and per-client OAuth limits
          </p>
        </div>
        <button
          onClick={() => loadData(true)}
          disabled={refreshing}
          className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-700"
        >
          <RefreshCw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} /> Refresh
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-600 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" /> {error}
        </div>
      )}

      {/* Auth Rate Limits */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
          <Shield className="h-4 w-4" /> Authentication Rate Limits
        </h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {limits.map((lim, i) => {
            const Icon = lim.icon;
            const colorMap: Record<string, { bg: string; text: string; bar: string }> = {
              red: { bg: "bg-red-50 dark:bg-red-950/30", text: "text-red-600 dark:text-red-400", bar: "bg-red-500" },
              amber: { bg: "bg-amber-50 dark:bg-amber-950/30", text: "text-amber-600 dark:text-amber-400", bar: "bg-amber-500" },
              blue: { bg: "bg-blue-50 dark:bg-blue-950/30", text: "text-blue-600 dark:text-blue-400", bar: "bg-blue-500" },
              purple: { bg: "bg-purple-50 dark:bg-purple-950/30", text: "text-purple-600 dark:text-purple-400", bar: "bg-purple-500" },
            };
            const c = colorMap[lim.color];
            return (
              <div key={i} className={`${card} ${c.bg}`}>
                <div className="mb-3 flex items-center gap-2">
                  <Icon className={`h-4 w-4 ${c.text}`} />
                  <span className="text-xs font-semibold uppercase text-gray-400">{lim.label}</span>
                </div>
                <div className="flex items-baseline gap-1">
                  <span className={`text-3xl font-bold ${c.text}`}>{lim.value}</span>
                  <span className="text-sm text-gray-400">requests</span>
                </div>
                <div className="mt-2 flex items-center gap-1 text-xs text-gray-400">
                  <Clock className="h-3 w-3" />
                  per {lim.window < 60 ? `${lim.window}s` : `${lim.window / 60}min`}
                </div>
                {/* Visual gauge */}
                <div className="mt-3 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div
                    className={`h-full rounded-full ${c.bar}`}
                    style={{ width: `${Math.min((lim.value / Math.max(...limits.map(l => l.value))) * 100, 100)}%` }}
                  />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Client Rate Limits */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
          <Server className="h-4 w-4" /> OAuth Client Rate Limits
        </h2>
        {clientLimits.length === 0 ? (
          <div className={card}>
            <div className="py-8 text-center">
              <Server className="mx-auto mb-3 h-10 w-10 text-gray-300" />
              <p className="text-sm text-gray-400">No client rate limits configured</p>
              <p className="mt-1 text-xs text-gray-400">
                Configure per-client rate limits via <code className="text-xs">POST /api/v1/oauth/client-rate-limits</code>
              </p>
            </div>
          </div>
        ) : (
          <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-400">Client ID</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-400">Requests/min</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-400">Burst</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-400">Strategy</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase text-gray-400">Current Usage</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {clientLimits.map((cl, i) => (
                  <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <td className="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{cl.client_id}</td>
                    <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">{cl.requests_per_min}</td>
                    <td className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{cl.burst}</td>
                    <td className="px-4 py-3">
                      <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                        {cl.strategy}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {cl.current_usage !== undefined ? (
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                            <div
                              className={`h-full rounded-full ${cl.current_usage > cl.requests_per_min * 0.8 ? "bg-red-500" : cl.current_usage > cl.requests_per_min * 0.5 ? "bg-amber-500" : "bg-green-500"}`}
                              style={{ width: `${Math.min((cl.current_usage / cl.requests_per_min) * 100, 100)}%` }}
                            />
                          </div>
                          <span className="text-xs text-gray-500">{cl.current_usage}/{cl.requests_per_min}</span>
                        </div>
                      ) : (
                        <span className="text-xs text-gray-400">N/A</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Summary */}
      {authLimits && (
        <div className={`${card} bg-gradient-to-br from-brand-50 to-white dark:from-gray-800 dark:to-gray-900`}>
          <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <TrendingUp className="h-4 w-4 text-brand-600" /> Configuration Summary
          </h3>
          <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
            <div>
              <span className="text-gray-400">Max login attempts:</span>
              <span className="ml-2 font-medium text-gray-900 dark:text-white">{authLimits.login_rate_limit} per {authLimits.login_window_seconds}s</span>
            </div>
            <div>
              <span className="text-gray-400">Max registrations:</span>
              <span className="ml-2 font-medium text-gray-900 dark:text-white">{authLimits.register_rate_limit} per {authLimits.register_window_seconds / 60}min</span>
            </div>
            <div>
              <span className="text-gray-400">IP throttle:</span>
              <span className="ml-2 font-medium text-gray-900 dark:text-white">{authLimits.ip_rate_limit} per {authLimits.ip_window_seconds}s</span>
            </div>
            <div>
              <span className="text-gray-400">Tenant cap:</span>
              <span className="ml-2 font-medium text-gray-900 dark:text-white">{authLimits.tenant_rate_limit} per {authLimits.tenant_window_seconds / 60}min</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}