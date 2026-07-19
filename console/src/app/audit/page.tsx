"use client";

import { Fragment, useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { PermissionGuard } from "@/components/PermissionGuard";
import {
  ScrollText,
  RefreshCw,
  Download,
  Activity,
  AlertTriangle,
  TrendingUp,
  Users,
  ChevronRight,
  ChevronDown,
} from "lucide-react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  AreaChart,
  Area,
  Legend,
} from "@/components/charts/lazy-charts";

interface AuditEvent {
  id: string;
  tenant_id: string;
  actor_type: string;
  actor_id: string;
  actor_name: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  result: string;
  created_at: string;
  ip_address?: string;
  user_agent?: string;
  metadata?: Record<string, unknown>;
}

interface Stats {
  total_events_24h: number;
  events_by_action: Record<string, number>;
  hourly_distribution: { hour: string; count: number }[];
  top_actors: { actor_id: string; actor_name: string; count: number }[];
  failed_logins_24h: number;
}

const PIE_COLORS = ["#6366f1", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4", "#ec4899"];

type Tab = "events" | "dashboard";

export default function AuditPage() {
  const { apiFetch, API_BASE, TENANT_ID } = useApi();
  const { t } = useI18n();
  const [tab, setTab] = useState<Tab>("dashboard");
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionFilter, setActionFilter] = useState("");
  const [actorFilter, setActorFilter] = useState("");
  const [resultFilter, setResultFilter] = useState("");
  const [ipFilter, setIpFilter] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  // Sync filters to URL query params
  useEffect(() => {
    const params = new URLSearchParams();
    if (actionFilter) params.set("action", actionFilter);
    if (actorFilter) params.set("actor", actorFilter);
    if (resultFilter) params.set("result", resultFilter);
    if (ipFilter) params.set("ip", ipFilter);
    if (dateFrom) params.set("from", dateFrom);
    if (dateTo) params.set("to", dateTo);
    const qs = params.toString();
    const newUrl = qs ? `?${qs}` : window.location.pathname;
    window.history.replaceState(null, "", newUrl);
  }, [actionFilter, actorFilter, resultFilter, ipFilter, dateFrom, dateTo]);

  // Reset page when filters change
  useEffect(() => {
    setPage(1);
  }, [actionFilter, actorFilter, resultFilter, ipFilter, dateFrom, dateTo]);

  // Read filters from URL on mount
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("action")) setActionFilter(params.get("action")!);
    if (params.get("actor")) setActorFilter(params.get("actor")!);
    if (params.get("result")) setResultFilter(params.get("result")!);
    if (params.get("ip")) setIpFilter(params.get("ip")!);
    if (params.get("from")) setDateFrom(params.get("from")!);
    if (params.get("to")) setDateTo(params.get("to")!);
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (tab === "dashboard") {
        const data = await apiFetch<Stats>("/api/v1/audit/stats").catch(() => null);
        setStats(data);
      } else {
        const params = new URLSearchParams();
        if (actionFilter) params.set("action", actionFilter);
        if (actorFilter) params.set("actor_id", actorFilter);
        if (resultFilter) params.set("result", resultFilter);
        if (ipFilter) params.set("ip_address", ipFilter);
        if (dateFrom) params.set("from", dateFrom + "T00:00:00Z");
        if (dateTo) params.set("to", dateTo + "T23:59:59Z");
        params.set("page_size", "20");
        params.set("page", String(page));
        const data = await apiFetch<{ events?: AuditEvent[]; total?: number; total_count?: number }>(
          `/api/v1/audit/events?${params}`,
        );
        const totalCount = data.total || data.total_count || 0;
        setTotalPages(totalCount > 0 ? Math.ceil(totalCount / 20) : 1);
        let filtered = data.events || [];
        // Client-side IP filter fallback if API doesn't support it
        if (ipFilter && filtered.length > 0) {
          filtered = filtered.filter((e: any) => e.ip_address?.includes(ipFilter));
        }
        setEvents(filtered);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch, tab, actionFilter, actorFilter, resultFilter, ipFilter, dateFrom, dateTo, page]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleExport = (format: "csv" | "json") => {
    const params = new URLSearchParams({ tenant_id: TENANT_ID, format });
    if (actionFilter) params.set("action", actionFilter);
    window.open(`${API_BASE}/api/v1/audit/export?${params}`, "_blank");
  };

  const resultColor = (result: string) => {
    switch (result) {
      case "success": return "bg-green-100 text-green-700";
      case "failure": return "bg-yellow-100 text-yellow-700";
      case "denied": return "bg-red-100 text-red-700";
      default: return "bg-gray-100 text-gray-600";
    }
  };

  // Check if a specific event row should be highlighted as anomalous
  const isAnomalousEvent = (event: AuditEvent): string | null => {
    // Failed login from same actor > 3 times
    if (event.action === "user.login" && event.result !== "success") {
      const failuresFromActor = events.filter(
        (e) => e.actor_id === event.actor_id && e.action === "user.login" && e.result !== "success",
      ).length;
      if (failuresFromActor >= 3) {
        return "Brute force: " + failuresFromActor + " failed logins";
      }
    }
    // Denied access from unusual action type
    if (event.result === "denied") {
      const deniedFromActor = events.filter(
        (e) => e.actor_id === event.actor_id && e.result === "denied",
      ).length;
      if (deniedFromActor >= 3) {
        return "Repeated access denied: " + deniedFromActor + " attempts";
      }
    }
    // Off-hours activity (between 2am-5am UTC)
    const hour = new Date(event.created_at).getUTCHours();
    if (hour >= 2 && hour < 5 && event.result !== "success") {
      return "Off-hours suspicious activity";
    }
    return null;
  };

  // Detect anomalies in the current event set
  const anomalyAlerts = detectAnomalies(events);

  // Prepare chart data
  const actionData = stats
    ? Object.entries(stats.events_by_action)
        .map(([action, count]: any[]) => ({ name: action, value: count }))
        .sort((a: any, b: any) => b.value - a.value)
        .slice(0, 8)
    : [];

  const hourlyData = stats
    ? stats.hourly_distribution.map((h: any) => ({
        time: new Date(h.hour).toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" }),
        events: h.count,
      }))
    : [];

  const actorData = stats
    ? Object.values(
        stats.top_actors.reduce((acc: Record<string, { name: string; count: number }>, a: any) => {
          const name = a.actor_name || (a.actor_id !== "00000000-0000-0000-0000-000000000000" ? a.actor_id.slice(0, 8) : "system");
          if (!acc[name]) acc[name] = { name, count: 0 };
          acc[name].count += a.count;
          return acc;
        }, {})
      )
        .sort((a: any, b: any) => b.count - a.count)
    : [];

  // Deduplicated top actors for the table
  const topActorsDeduped = stats
    ? Object.values(
        stats.top_actors.reduce((acc: Record<string, { actor_id: string; actor_name: string; count: number }>, a: any) => {
          const name = a.actor_name || (a.actor_id !== "00000000-0000-0000-0000-000000000000" ? a.actor_id.slice(0, 8) : "system");
          if (!acc[name]) acc[name] = { ...a, actor_name: name, count: 0 };
          acc[name].count += a.count;
          return acc;
        }, {})
      )
        .sort((a: any, b: any) => b.count - a.count)
    : [];

  // Action bar chart data (top actions as bars)
  const actionBarData = actionData.slice(0, 6);

  // Failed vs success comparison
  const failedVsSuccess = stats
    ? [
        { name: "Success", count: stats.total_events_24h - stats.failed_logins_24h, fill: "#10b981" },
        { name: "Failed Logins", count: stats.failed_logins_24h, fill: "#ef4444" },
      ].filter((d: any) => d.count > 0)
    : [];

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold dark:text-gray-100">Audit Log</h1>
        <div className="flex gap-2">
          <button
            onClick={() => handleExport("csv")}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <Download className="h-4 w-4" /> CSV
          </button>
          <button
            onClick={() => handleExport("json")}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <Download className="h-4 w-4" /> JSON
          </button>
          <button
            onClick={loadData}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
          <p className="mt-1 text-xs">Make sure Audit Service (:8072) is running.</p>
        </div>
      )}

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-gray-200">
        <button
          onClick={() => setTab("dashboard")}
          className={`px-4 py-2 text-sm font-medium ${
            tab === "dashboard"
              ? "border-b-2 border-brand-600 text-brand-600"
              : "text-gray-500 hover:text-gray-700"
          }`}
        >
          Dashboard
        </button>
        <button
          onClick={() => setTab("events")}
          className={`px-4 py-2 text-sm font-medium ${
            tab === "events"
              ? "border-b-2 border-brand-600 text-brand-600"
              : "text-gray-500 hover:text-gray-700"
          }`}
        >
          Event Log
        </button>
      </div>

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : tab === "dashboard" ? (
        /* ===== Dashboard ===== */
        !stats ? (
          <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
            <Activity className="mx-auto mb-4 h-12 w-12 text-gray-300" />
            <p className="text-gray-500">No stats available</p>
            <p className="mt-1 text-xs text-gray-400">Stats are generated from events in the last 24 hours.</p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* Stat cards */}
            <div className="grid gap-4 sm:grid-cols-3">
              <StatCard
                icon={Activity}
                label={t("audit.events24h")}
                value={stats.total_events_24h}
                color="indigo"
              />
              <StatCard
                icon={TrendingUp}
                label={t("audit.uniqueEventTypes")}
                value={Object.keys(stats.events_by_action).length}
                color="green"
              />
              <StatCard
                icon={AlertTriangle}
                label={t("audit.failedLogins24h")}
                value={stats.failed_logins_24h}
                color="red"
              />
            </div>

            {/* Charts row */}
            <div className="grid gap-6 lg:grid-cols-2">
              {/* Hourly distribution */}
              <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <h3 className="mb-4 text-sm font-semibold">Event Timeline (24h)</h3>
                {hourlyData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={250}>
                    <AreaChart data={hourlyData}>
                      <defs>
                        <linearGradient id="colorEvents" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#6366f1" stopOpacity={0.8} />
                          <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                      <XAxis dataKey="time" tick={{ fontSize: 11 }} interval="preserveStartEnd" />
                      <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                      <Tooltip
                        contentStyle={{ fontSize: 12, borderRadius: 8 }}
                        labelStyle={{ fontWeight: 600 }}
                      />
                      <Area
                        type="monotone"
                        dataKey="events"
                        stroke="#6366f1"
                        strokeWidth={2}
                        fill="url(#colorEvents)"
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="flex h-[250px] items-center justify-center text-sm text-gray-400">
                    No hourly data
                  </div>
                )}
              </div>

              {/* Events by action pie */}
              <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <h3 className="mb-4 text-sm font-semibold">Events by Action Type</h3>
                {actionData.length > 0 ? (
                  <ResponsiveContainer width="100%" height={250}>
                    <PieChart>
                      <Pie
                        data={actionData}
                        cx="50%"
                        cy="50%"
                        outerRadius={80}
                        dataKey="value"
                        label={(entry: { name?: string; percent?: number }) => {
                          const n = entry.name ? entry.name.split(".").pop() : "";
                          const pct = entry.percent ? (entry.percent * 100).toFixed(0) : "0";
                          return `${n} ${pct}%`;
                        }}
                        labelLine={false}
                      >
                        {actionData.map((_, i) => (
                          <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                        ))}
                      </Pie>
                      <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                    </PieChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="flex h-[250px] items-center justify-center text-sm text-gray-400">
                    No action data
                  </div>
                )}
              </div>
            </div>

            {/* Top actors bar chart */}
            {actorData.length > 0 && (
              <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold">
                  <Users className="h-4 w-4 text-brand-600" />
                  Top Active Users (24h)
                </h3>
                <ResponsiveContainer width="100%" height={200}>
                  <BarChart data={actorData} layout="vertical">
                    <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" horizontal={false} />
                    <XAxis type="number" tick={{ fontSize: 11 }} allowDecimals={false} />
                    <YAxis
                      type="category"
                      dataKey="name"
                      tick={{ fontSize: 11 }}
                      width={120}
                    />
                    <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                    <Bar dataKey="count" fill="#8b5cf6" radius={[0, 4, 4, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
              )}

              {/* Top 10 active users table */}
              {stats?.top_actors && stats.top_actors.length > 0 && (
                <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <h3 className="mb-4 text-sm font-semibold">Top 10 Active Users (24h)</h3>
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-gray-100 dark:border-gray-700">
                        <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">#</th>
                        <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">User</th>
                        <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Events</th>
                        <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-500 dark:text-gray-400">Activity</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-50">
                      {topActorsDeduped.slice(0, 10).map((actor: any, i: any) => (
                        <tr key={actor.actor_id + i} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                          <td className="px-3 py-2 text-sm text-gray-400">{i + 1}</td>
                          <td className="px-3 py-2 text-sm font-medium">
                            {actor.actor_name}
                          </td>
                          <td className="px-3 py-2 text-sm">{actor.count}</td>
                          <td className="px-3 py-2">
                            <div className="h-2 w-full max-w-[120px] rounded-full bg-gray-100">
                              <div
                                className="h-2 rounded-full bg-purple-500"
                                style={{ width: `${Math.min(100, (actor.count / Math.max(...topActorsDeduped.map((a: any) => a.count))) * 100)}%` }}
                              />
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

            {/* Action bar chart + Failed logins */}
            <div className="grid gap-6 lg:grid-cols-2">
              {actionBarData.length > 0 && (
                <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <h3 className="mb-4 text-sm font-semibold">Top Event Actions</h3>
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={actionBarData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                      <XAxis dataKey="name" tick={{ fontSize: 10 }} angle={-30} textAnchor="end" height={50} />
                      <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                      <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                      <Bar dataKey="value" fill="#6366f1" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
              {failedVsSuccess.length > 0 && (
                <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <h3 className="mb-4 text-sm font-semibold">Success vs Failed Logins</h3>
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={failedVsSuccess}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                      <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                      <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                      <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                      <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                        {failedVsSuccess.map((entry: any, i: any) => (
                          <Cell key={i} fill={entry.fill} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </div>
          </div>
        )
      ) : tab === "events" ? (
        /* ===== Event Log Table ===== */
        <>
          <div className="mb-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-6">
            <select
              value={actionFilter}
              onChange={(e) => setActionFilter(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            >
              <option value="">{t("audit.allActions")}</option>
              <option value="user.login">user.login</option>
              <option value="user.register">user.register</option>
              <option value="user.logout">user.logout</option>
              <option value="role.create">role.create</option>
              <option value="role.update">role.update</option>
              <option value="role.delete">role.delete</option>
              <option value="org.create">org.create</option>
              <option value="org.update">org.update</option>
              <option value="org.delete">org.delete</option>
              <option value="member.add">member.add</option>
              <option value="member.remove">member.remove</option>
              <option value="policy.evaluate">policy.evaluate</option>
            </select>
            <input
              type="text"
              placeholder={t("audit.actorPlaceholder")}
              value={actorFilter}
              onChange={(e) => setActorFilter(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            />
            <input
              type="text"
              placeholder="IP Address"
              value={ipFilter}
              onChange={(e) => setIpFilter(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 font-mono"
            />
            <select
              value={resultFilter}
              onChange={(e) => setResultFilter(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            >
              <option value="">{t("audit.allResults")}</option>
              <option value="success">Success</option>
              <option value="failure">Failure</option>
              <option value="denied">Denied</option>
            </select>
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            />
            <input
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            />
            <button
              onClick={loadData}
              className="flex items-center justify-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
            >
              <RefreshCw className="h-4 w-4" />
              Apply Filters
            </button>
          </div>
          {events.length === 0 ? (
            <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
              <ScrollText className="mx-auto mb-4 h-12 w-12 text-gray-300" />
              <p className="text-gray-500">No audit events found</p>
              <p className="mt-1 text-xs text-gray-400">
                Audit events will appear here when services start sending them via NATS
              </p>
            </div>
          ) : (
            <>
            {/* Anomaly Alerts */}
            {anomalyAlerts.length > 0 && (
              <div className="mb-4 space-y-2">
                {anomalyAlerts.map((alert: any, i: any) => (
                  <div
                    key={i}
                    className={`flex items-center gap-3 rounded-lg border px-4 py-3 ${
                      alert.severity === "critical"
                        ? "border-red-300 bg-red-50"
                        : alert.severity === "warning"
                          ? "border-amber-300 bg-amber-50"
                          : "border-blue-300 bg-blue-50"
                    }`}
                  >
                    <AlertTriangle
                      className={`h-5 w-5 ${
                        alert.severity === "critical"
                          ? "text-red-500"
                          : alert.severity === "warning"
                            ? "text-amber-500"
                            : "text-blue-500"
                      }`}
                    />
                    <div>
                      <p className={`text-sm font-semibold ${
                        alert.severity === "critical" ? "text-red-700" : alert.severity === "warning" ? "text-amber-700" : "text-blue-700"
                      }`}>
                        {alert.title}
                      </p>
                      <p className="text-xs text-gray-600">{alert.description}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}
            <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
              <table className="w-full">
                <thead className="border-b border-gray-200 bg-gray-50">
                  <tr>
                    <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Time</th>
                    <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Action</th>
                    <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Actor</th>
                    <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Resource</th>
                    <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Result</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {events.map((event: any) => {
                    const anomaly = isAnomalousEvent(event);
                    const isExpanded = expandedRow === event.id;
                    return (
                    <Fragment key={event.id}>
                      <tr
                        onClick={() => setExpandedRow(isExpanded ? null : event.id)}
                        className={`cursor-pointer hover:bg-gray-50 ${anomaly ? "border-l-4 border-l-red-400 bg-red-50/40" : ""} ${isExpanded ? "bg-gray-50" : ""}`}
                      >
                        <td className="px-4 py-3 text-sm text-gray-500">
                          <div className="flex items-center gap-1.5">
                            {isExpanded ? (
                              <ChevronDown className="h-3.5 w-3.5 shrink-0 text-gray-400" />
                            ) : (
                              <ChevronRight className="h-3.5 w-3.5 shrink-0 text-gray-400" />
                            )}
                            {event.created_at ? new Date(event.created_at).toLocaleString() : "-"}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs">{event.action}</span>
                            {anomaly && (
                              <span className="flex items-center gap-0.5 rounded-full bg-red-100 px-1.5 py-0.5 text-xs text-red-600" title={anomaly}>
                                <AlertTriangle className="h-3 w-3" />
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                          <div>{event.actor_name || (event.actor_id && event.actor_id !== "00000000-0000-0000-0000-000000000000" ? event.actor_id.substring(0, 8) : "system")}</div>
                          {event.ip_address && (
                            <div className="font-mono text-xs text-gray-400">{event.ip_address.replace(/\/\d+$/, "")}</div>
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                          {event.resource_type && event.resource_type !== "-" ? (
                            <span>{event.resource_type}{event.resource_id && event.resource_id !== "00000000-0000-0000-0000-000000000000" && <span className="ml-1 font-mono text-xs text-gray-400">{event.resource_id.substring(0, 8)}</span>}</span>
                          ) : (
                            <span className="text-gray-300">—</span>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${resultColor(event.result)}`}>
                            {event.result}
                          </span>
                        </td>
                      </tr>
                      {isExpanded && (
                        <tr>
                          <td colSpan={5} className="bg-gray-50 px-4 py-3">
                            <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-xs text-green-400">
                              {JSON.stringify(event, null, 2)}
                            </pre>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-gray-500">
                Page {page} of {totalPages}
              </p>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                >
                  Previous
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                >
                  Next
                </button>
              </div>
            </div>
            </>
          )}
        </>
      )
      : null}
    </div>
  );
}

function StatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType;
  label: string;
  value: number;
  color: "indigo" | "green" | "red";
}) {
  const colorMap = {
    indigo: { bg: "bg-indigo-100", text: "text-indigo-600" },
    green: { bg: "bg-green-100", text: "text-green-600" },
    red: { bg: "bg-red-100", text: "text-red-600" },
  };
  const c = colorMap[color];
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold dark:text-gray-100">{value.toLocaleString()}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  );
}

// --- Anomaly Detection ---

interface AnomalyAlert {
  severity: "critical" | "warning" | "info";
  title: string;
  description: string;
}

// Detect anomalies in a set of audit events
function detectAnomalies(events: AuditEvent[]): AnomalyAlert[] {
  const alerts: AnomalyAlert[] = [];
  if (events.length === 0) return alerts;

  // 1. Brute force: same actor with >= 5 failed logins
  const failedLoginsByActor: Record<string, AuditEvent[]> = {};
  events.forEach((e: any) => {
    if (e.action === "user.login" && e.result !== "success") {
      const key = e.actor_id || e.actor_name || "unknown";
      if (!failedLoginsByActor[key]) failedLoginsByActor[key] = [];
      failedLoginsByActor[key].push(e);
    }
  });
  Object.entries(failedLoginsByActor).forEach(([actor, fails]) => {
    if (fails.length >= 5) {
      alerts.push({
        severity: "critical",
        title: `Brute Force Suspected: ${fails[0].actor_name || actor.substring(0, 8)}`,
        description: `${fails.length} failed login attempts detected. IP: ${fails[0].ip_address || "unknown"}.`,
      });
    } else if (fails.length >= 3) {
      alerts.push({
        severity: "warning",
        title: `Login Failures: ${fails[0].actor_name || actor.substring(0, 8)}`,
        description: `${fails.length} failed login attempts in current view.`,
      });
    }
  });

  // 2. Unusual IP: same actor from > 3 unique IPs
  const ipsByActor: Record<string, Set<string>> = {};
  events.forEach((e: any) => {
    if (e.ip_address) {
      const key = e.actor_id || e.actor_name || "unknown";
      if (!ipsByActor[key]) ipsByActor[key] = new Set();
      ipsByActor[key].add(e.ip_address);
    }
  });
  Object.entries(ipsByActor).forEach(([actor, ips]) => {
    if (ips.size > 3) {
      alerts.push({
        severity: "warning",
        title: `Multiple IP Addresses: ${events.find((e: any) => e.actor_id === actor)?.actor_name || actor.substring(0, 8)}`,
        description: `Activity from ${ips.size} unique IP addresses: ${[...ips].join(", ")}.`,
      });
    }
  });

  // 3. Access denied spike
  const deniedCount = events.filter((e: any) => e.result === "denied").length;
  if (deniedCount >= 5) {
    alerts.push({
      severity: "warning",
      title: "Access Denied Spike",
      description: `${deniedCount} access denied events in current view. Possible permission misconfiguration or unauthorized access attempt.`,
    });
  }

  // 4. Off-hours activity (2am-5am UTC)
  const offHours = events.filter((e: any) => {
    const h = new Date(e.created_at).getUTCHours();
    return h >= 2 && h < 5 && e.result !== "success";
  });
  if (offHours.length >= 3) {
    alerts.push({
      severity: "info",
      title: "Off-Hours Suspicious Activity",
      description: `${offHours.length} failed/denied events between 2-5 AM UTC. Potential unauthorized access attempt.`,
    });
  }

  return alerts;
}
