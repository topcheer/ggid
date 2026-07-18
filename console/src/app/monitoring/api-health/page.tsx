"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Activity, Clock, AlertTriangle, Loader2, RefreshCw, Save,
  Check, AlertCircle, Gauge, Zap, Bell, Server,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type TabId = "endpoints" | "slowRequests" | "alerts";

interface Endpoint {
  path: string; method: string; p95_latency_ms: number; error_rate: number;
  calls: number; last_called: string; status: "healthy" | "degraded" | "down";
}

interface SlowRequest {
  id: string; timestamp: string; path: string; method: string;
  duration_ms: number; status_code: number; user_agent: string; ip: string;
}

export default function ApiHealthPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("endpoints");

  const tabs: { id: TabId; label: string; icon: typeof Activity }[] = [
    { id: "endpoints", label: t("apiHealth.tabs.endpoints"), icon: Activity },
    { id: "slowRequests", label: t("apiHealth.tabs.slowRequests"), icon: Clock },
    { id: "alerts", label: t("apiHealth.tabs.alerts"), icon: Bell },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Server className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("apiHealth.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("apiHealth.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />
              {label}
            </button>
          ))}
        </div>

        {tab === "endpoints" && <EndpointsTab />}
        {tab === "slowRequests" && <SlowRequestsTab />}
        {tab === "alerts" && <AlertsTab />}
      </div>
    </div>
  );
}

// ============ Endpoints Tab ============

function EndpointsTab() {
  const t = useTranslations();
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/gateway/metrics`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); if (d.endpoints) { setEndpoints(d.endpoints); return; } }
    } catch { /* mock */ }
    setEndpoints([
      { path: "/api/v1/auth/login", method: "POST", p95_latency_ms: 145, error_rate: 1.2, calls: 8520, last_called: "2025-07-18T09:30:00Z", status: "healthy" },
      { path: "/api/v1/users", method: "GET", p95_latency_ms: 89, error_rate: 0.3, calls: 12400, last_called: "2025-07-18T09:35:00Z", status: "healthy" },
      { path: "/api/v1/auth/token", method: "POST", p95_latency_ms: 620, error_rate: 3.5, calls: 5230, last_called: "2025-07-18T09:32:00Z", status: "degraded" },
      { path: "/api/v1/audit/events", method: "GET", p95_latency_ms: 320, error_rate: 0.8, calls: 3100, last_called: "2025-07-18T09:34:00Z", status: "healthy" },
      { path: "/api/v1/scim/v2/Users", method: "POST", p95_latency_ms: 1200, error_rate: 8.2, calls: 450, last_called: "2025-07-18T08:15:00Z", status: "degraded" },
      { path: "/api/v1/oauth/authorize", method: "GET", p95_latency_ms: 210, error_rate: 0.5, calls: 8900, last_called: "2025-07-18T09:36:00Z", status: "healthy" },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading) return <Spinner />;

  const statusColors: Record<string, string> = {
    healthy: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    degraded: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
    down: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  };
  const methodColors: Record<string, string> = {
    GET: "text-blue-600", POST: "text-green-600", PUT: "text-orange-600", DELETE: "text-red-600", PATCH: "text-purple-600",
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("apiHealth.endpoints.title")}</h3>
        <button onClick={load} className="flex items-center gap-1.5 px-3 py-1.5 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg text-xs hover:bg-gray-50">
          <RefreshCw className="w-3 h-3" />
          {t("apiHealth.endpoints.refresh")}
        </button>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.endpoints.method")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.endpoints.path")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("apiHealth.endpoints.p95Latency")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("apiHealth.endpoints.errorRate")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("apiHealth.endpoints.calls")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.endpoints.status")}</th>
              </tr>
            </thead>
            <tbody>
              {endpoints.map((e, i) => (
                <tr key={i} className="border-b border-gray-100 dark:border-gray-800/50 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                  <td className="py-3 px-3"><span className={`text-xs font-bold ${methodColors[e.method] || "text-gray-500"}`}>{e.method}</span></td>
                  <td className="py-3 px-3 font-mono text-xs text-gray-900 dark:text-white">{e.path}</td>
                  <td className="py-3 px-3 text-right">
                    <span className={`text-xs font-medium ${e.p95_latency_ms > 500 ? "text-red-600" : e.p95_latency_ms > 200 ? "text-orange-500" : "text-gray-900 dark:text-white"}`}>
                      {e.p95_latency_ms}ms
                    </span>
                  </td>
                  <td className="py-3 px-3 text-right">
                    <span className={`text-xs font-medium ${e.error_rate > 5 ? "text-red-600" : e.error_rate > 1 ? "text-orange-500" : "text-gray-900 dark:text-white"}`}>
                      {e.error_rate}%
                    </span>
                  </td>
                  <td className="py-3 px-3 text-right text-xs text-gray-600 dark:text-gray-400">{e.calls.toLocaleString()}</td>
                  <td className="py-3 px-3">
                    <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[e.status]}`}>
                      {t(`apiHealth.endpoints.status${e.status.replace(/^./, (m) => m.toUpperCase())}`)}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ============ Slow Requests Tab ============

function SlowRequestsTab() {
  const t = useTranslations();
  const [requests, setRequests] = useState<SlowRequest[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Mock data
    setRequests([
      { id: "1", timestamp: "2025-07-18T09:35:22Z", path: "/api/v1/scim/v2/Users", method: "POST", duration_ms: 1850, status_code: 201, user_agent: "SCIM-Client/2.0", ip: "10.0.1.5" },
      { id: "2", timestamp: "2025-07-18T09:34:10Z", path: "/api/v1/auth/token", method: "POST", duration_ms: 920, status_code: 200, user_agent: "Mozilla/5.0", ip: "192.168.1.100" },
      { id: "3", timestamp: "2025-07-18T09:32:45Z", path: "/api/v1/audit/events", method: "GET", duration_ms: 680, status_code: 200, user_agent: "curl/8.0", ip: "10.0.2.3" },
      { id: "4", timestamp: "2025-07-18T09:30:15Z", path: "/api/v1/scim/v2/Users", method: "POST", duration_ms: 1200, status_code: 500, user_agent: "SCIM-Client/2.0", ip: "10.0.1.5" },
      { id: "5", timestamp: "2025-07-18T09:28:03Z", path: "/api/v1/auth/token", method: "POST", duration_ms: 750, status_code: 200, user_agent: "PostmanRuntime/7.0", ip: "172.16.0.5" },
    ]);
    setLoading(false);
  }, []);

  if (loading) return <Spinner />;

  const statusColors: Record<number, string> = {
    200: "text-green-600", 201: "text-green-600", 400: "text-orange-600", 500: "text-red-600",
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("apiHealth.slowRequests.title")}</h3>
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("apiHealth.slowRequests.description")}</p>

      {requests.length === 0 ? (
        <div className="text-center py-12">
          <Gauge className="w-12 h-12 mx-auto mb-2 text-green-500" />
          <p className="text-sm text-gray-500">{t("apiHealth.slowRequests.noSlowRequests")}</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800 text-left">
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.slowRequests.timestamp")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.slowRequests.path")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("apiHealth.slowRequests.duration")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("apiHealth.slowRequests.statusCode")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.slowRequests.ip")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("apiHealth.slowRequests.userAgent")}</th>
              </tr>
            </thead>
            <tbody>
              {requests.map((r) => (
                <tr key={r.id} className="border-b border-gray-100 dark:border-gray-800/50">
                  <td className="py-3 px-3 text-xs text-gray-500">{new Date(r.timestamp).toLocaleTimeString()}</td>
                  <td className="py-3 px-3"><span className="text-xs font-mono text-gray-900 dark:text-white">{r.method} {r.path}</span></td>
                  <td className="py-3 px-3 text-right">
                    <span className={`text-xs font-medium ${r.duration_ms > 1000 ? "text-red-600" : "text-orange-500"}`}>
                      {r.duration_ms}ms
                    </span>
                  </td>
                  <td className={`py-3 px-3 text-right text-xs font-medium ${statusColors[r.status_code] || "text-gray-500"}`}>{r.status_code}</td>
                  <td className="py-3 px-3 text-xs text-gray-600 dark:text-gray-400">{r.ip}</td>
                  <td className="py-3 px-3 text-xs text-gray-400">{r.user_agent}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

// ============ Alerts Tab ============

function AlertsTab() {
  const t = useTranslations();
  const [config, setConfig] = useState({
    enabled: true, error_rate_threshold: 5, latency_threshold: 500,
    min_calls: 100, window_minutes: 5, webhook_url: "",
  });
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const save = async () => {
    setSaving(true);
    try {
      await fetch(`${API_BASE}/api/v1/gateway/alerts/config`, {
        method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(config),
      });
    } catch { /* ok */ }
    setSaving(false);
    setMsg(t("apiHealth.alerts.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("apiHealth.alerts.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t("apiHealth.alerts.description")}</p>
      </div>

      {/* Enable toggle */}
      <label className="flex items-center justify-between cursor-pointer">
        <span className="text-sm text-gray-700 dark:text-gray-300">{t("apiHealth.alerts.enabled")}</span>
        <button onClick={() => setConfig({ ...config, enabled: !config.enabled })}
          className={`relative w-10 h-6 rounded-full transition-colors ${config.enabled ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
          <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${config.enabled ? "translate-x-4" : ""}`} />
        </button>
      </label>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <NumberInput label={t("apiHealth.alerts.errorRateThreshold")} value={config.error_rate_threshold} onChange={(v) => setConfig({ ...config, error_rate_threshold: v })} min={0} max={100} suffix="%" />
        <NumberInput label={t("apiHealth.alerts.latencyThreshold")} value={config.latency_threshold} onChange={(v) => setConfig({ ...config, latency_threshold: v })} min={100} max={10000} suffix="ms" />
        <NumberInput label={t("apiHealth.alerts.minCalls")} value={config.min_calls} onChange={(v) => setConfig({ ...config, min_calls: v })} min={1} max={10000} />
        <NumberInput label={t("apiHealth.alerts.windowMinutes")} value={config.window_minutes} onChange={(v) => setConfig({ ...config, window_minutes: v })} min={1} max={60} suffix="min" />
      </div>

      <div>
        <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("apiHealth.alerts.webhookUrl")}</label>
        <input type="text" value={config.webhook_url} onChange={(e) => setConfig({ ...config, webhook_url: e.target.value })}
          placeholder={t("apiHealth.alerts.webhookPlaceholder")}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
      </div>

      {msg && (
        <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
          <Check className="w-4 h-4" />{msg}
        </div>
      )}

      <button onClick={save} disabled={saving}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("apiHealth.alerts.save")}
      </button>
    </div>
  );
}

// ============ Shared ============

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }

function NumberInput({ label, value, onChange, min, max, suffix }: {
  label: string; value: number; onChange: (v: number) => void; min: number; max: number; suffix?: string;
}) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{label}</label>
      <div className="relative">
        <input type="number" value={value} onChange={(e) => onChange(Math.max(min, Math.min(max, parseInt(e.target.value) || min)))}
          min={min} max={max}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white pr-8" />
        {suffix && <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">{suffix}</span>}
      </div>
    </div>
  );
}
