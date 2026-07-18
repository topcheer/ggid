"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Activity, Globe, TrendingUp, Clock, Zap, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface VelocityData {
  user_id: string;
  username: string;
  total_attempts: number;
  failed_attempts: number;
  successful_attempts: number;
  success_rate: number;
  unique_ips: number;
  unique_countries: number;
  geo_spread: GeoPoint[];
  recent_events: VelocityEvent[];
  window_minutes: number;
}

interface GeoPoint {
  country: string;
  country_code: string;
  city: string;
  count: number;
  lat: number;
  lng: number;
}

interface VelocityEvent {
  timestamp: string;
  ip: string;
  country: string;
  success: boolean;
  method: string;
}

export default function LoginVelocityPage() {
  const t = useTranslations();

  const [data, setData] = useState<VelocityData | null>(null);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const fetchVelocity = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/auth/login-velocity?user=${encodeURIComponent(user)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const json = await res.json();
        setData(json);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchVelocity(search);
  }, [search, fetchVelocity]);

  // Auto-refresh every 10s
  useEffect(() => {
    if (!autoRefresh || !search) return;
    const interval = setInterval(() => fetchVelocity(search), 10000);
    return () => clearInterval(interval);
  }, [autoRefresh, search, fetchVelocity]);

  const gaugeColor = data ? (data.success_rate >= 90 ? "text-green-500" : data.success_rate >= 70 ? "text-yellow-500" : "text-red-500") : "";
  const gaugePct = data ? Math.min(100, data.success_rate) : 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><Zap className="w-6 h-6 text-yellow-500" /> {t("loginVelocity.title")}</h1>
          <p className="text-sm text-gray-500 mt-1">Monitor real-time login patterns and detect anomalies.</p>
        </div>
        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <RefreshCw className={`w-4 h-4 ${autoRefresh ? "text-blue-500 animate-spin" : "text-gray-400"}`} style={autoRefresh ? { animationDuration: "3s" } : {}} />
          <input aria-label="Auto refresh" type="checkbox" checked={autoRefresh} onChange={(e) => setAutoRefresh(e.target.checked)} className="sr-only" />
          <span className={autoRefresh ? "text-blue-600" : "text-gray-500"}>Auto-refresh</span>
        </label>
      </div>

      {/* User search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input
          type="text"
          placeholder="Search by username or user ID..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"
        />
      </div>

      {loading && !data && <p className="text-sm text-gray-500">Loading...</p>}

      {data && (
        <div className="space-y-4">
          {/* Velocity cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Attempts gauge */}
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-500">Total Attempts</span>
                <Activity className="w-5 h-5 text-gray-400" />
              </div>
              <p className="text-3xl font-bold mt-2">{data.total_attempts}</p>
              <div className="mt-2 flex items-center gap-2 text-xs">
                <span className="text-green-600">{data.successful_attempts} success</span>
                <span className="text-red-600">{data.failed_attempts} failed</span>
              </div>
            </div>

            {/* Success rate gauge */}
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-500">Success Rate</span>
                <TrendingUp className="w-5 h-5 text-gray-400" />
              </div>
              <div className="mt-2 flex items-center gap-3">
                <div className="relative w-16 h-16">
                  <svg className="w-16 h-16 -rotate-90" viewBox="0 0 64 64">
                    <circle cx="32" cy="32" r="28" fill="none" stroke="currentColor" strokeWidth="6" className="text-gray-200 dark:text-gray-800" />
                    <circle cx="32" cy="32" r="28" fill="none" stroke="currentColor" strokeWidth="6" strokeDasharray={`${gaugePct * 1.76} 176`} strokeLinecap="round" className={gaugeColor} />
                  </svg>
                  <span className={`absolute inset-0 flex items-center justify-center text-sm font-bold ${gaugeColor}`}>{Math.round(data.success_rate)}%</span>
                </div>
              </div>
            </div>

            {/* Unique IPs */}
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-500">Unique IPs</span>
                <Globe className="w-5 h-5 text-gray-400" />
              </div>
              <p className="text-3xl font-bold mt-2">{data.unique_ips}</p>
              <p className="text-xs text-gray-500 mt-1">across {data.unique_countries} countries</p>
            </div>

            {/* Window */}
            <div className="rounded-lg border p-4 dark:border-gray-800">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-500">Time Window</span>
                <Clock className="w-5 h-5 text-gray-400" />
              </div>
              <p className="text-3xl font-bold mt-2">{data.window_minutes}<span className="text-base font-normal text-gray-500">min</span></p>
            </div>
          </div>

          {/* Geo spread map */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Globe className="w-4 h-4" /> Geographic Spread</h3>
            </div>
            <div className="p-4 space-y-2">
              {data.geo_spread.map((geo: any, i: number) => (
                <div key={i} className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-lg">{geo.country_code === "US" ? "\ud83c\uddfa\ud83c\uddf8" : geo.country_code === "CN" ? "\ud83c\udde8\ud83c\uddf3" : geo.country_code === "GB" ? "\ud83c\uddec\ud83c\udde7" : geo.country_code === "DE" ? "\ud83c\udde9\ud83c\uddea" : geo.country_code === "JP" ? "\ud83c\uddef\ud83c\uddf5" : "\ud83c\udfdf"}</span>
                    <span>{geo.city}, {geo.country}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-24 h-2 rounded-full bg-gray-200 dark:bg-gray-800 overflow-hidden">
                      <div className="h-full bg-blue-500" style={{ width: `${Math.min(100, (geo.count / data.total_attempts) * 100)}%` }} />
                    </div>
                    <span className="text-xs text-gray-500 w-8 text-right">{geo.count}</span>
                  </div>
                </div>
              ))}
              {data.geo_spread.length === 0 && <p className="text-sm text-gray-500">No geo data available.</p>}
            </div>
          </div>

          {/* Recent events */}
          <div className="rounded-lg border dark:border-gray-800">
            <div className="px-4 py-3 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><Activity className="w-4 h-4" /> Recent Events</h3>
            </div>
            <div className="divide-y dark:divide-gray-800 max-h-64 overflow-y-auto">
              {data.recent_events.map((evt: any, i: number) => (
                <div key={i} className="px-4 py-2 flex items-center justify-between text-sm">
                  <div className="flex items-center gap-3">
                    <span className={`w-2 h-2 rounded-full ${evt.success ? "bg-green-500" : "bg-red-500"}`} />
                    <span className="text-gray-500">{evt.timestamp}</span>
                    <span className="font-mono text-xs">{evt.ip}</span>
                    <span className="text-gray-400">{evt.method}</span>
                  </div>
                  <span className="text-xs text-gray-400">{evt.country}</span>
                </div>
              ))}
              {data.recent_events.length === 0 && <p className="px-4 py-3 text-sm text-gray-500">No recent events.</p>}
            </div>
          </div>
        </div>
      )}

      {!data && !loading && search && <p className="text-sm text-gray-500">No data found.</p>}
      {!data && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to view login velocity data.</p>}
    </div>
  );
}
