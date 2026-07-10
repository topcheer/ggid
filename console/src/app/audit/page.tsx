"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  ScrollText,
  RefreshCw,
  Download,
  Activity,
  AlertTriangle,
  TrendingUp,
  Users,
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
} from "recharts";

interface AuditEvent {
  id: string;
  tenant_id: string;
  actor_type: string;
  actor_id: string;
  actor_name: string;
  action: string;
  resource_type: string;
  result: string;
  created_at: string;
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
  const [tab, setTab] = useState<Tab>("dashboard");
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionFilter, setActionFilter] = useState("");

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
        params.set("page_size", "50");
        const data = await apiFetch<{ events?: AuditEvent[] }>(
          `/api/v1/audit/events?${params}`,
        );
        setEvents(data.events || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch, tab, actionFilter]);

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

  // Prepare chart data
  const actionData = stats
    ? Object.entries(stats.events_by_action)
        .map(([action, count]) => ({ name: action, value: count }))
        .sort((a, b) => b.value - a.value)
        .slice(0, 8)
    : [];

  const hourlyData = stats
    ? stats.hourly_distribution.map((h) => ({
        time: new Date(h.hour).toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" }),
        events: h.count,
      }))
    : [];

  const actorData = stats
    ? stats.top_actors
        .map((a) => ({ name: a.actor_name || a.actor_id.slice(0, 8), count: a.count }))
        .sort((a, b) => b.count - a.count)
    : [];

  // Action bar chart data (top actions as bars)
  const actionBarData = actionData.slice(0, 6);

  // Failed vs success comparison
  const failedVsSuccess = stats
    ? [
        { name: "Success", count: stats.total_events_24h - stats.failed_logins_24h, fill: "#10b981" },
        { name: "Failed Logins", count: stats.failed_logins_24h, fill: "#ef4444" },
      ].filter((d) => d.count > 0)
    : [];

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Audit Log</h1>
        <div className="flex gap-2">
          <button
            onClick={() => handleExport("csv")}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50"
          >
            <Download className="h-4 w-4" /> CSV
          </button>
          <button
            onClick={() => handleExport("json")}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50"
          >
            <Download className="h-4 w-4" /> JSON
          </button>
          <button
            onClick={loadData}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50"
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
                label="Events (24h)"
                value={stats.total_events_24h}
                color="indigo"
              />
              <StatCard
                icon={TrendingUp}
                label="Unique Event Types"
                value={Object.keys(stats.events_by_action).length}
                color="green"
              />
              <StatCard
                icon={AlertTriangle}
                label="Failed Logins (24h)"
                value={stats.failed_logins_24h}
                color="red"
              />
            </div>

            {/* Charts row */}
            <div className="grid gap-6 lg:grid-cols-2">
              {/* Hourly distribution */}
              <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
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
              <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
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
              <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
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

            {/* Action bar chart + Failed logins */}
            <div className="grid gap-6 lg:grid-cols-2">
              {actionBarData.length > 0 && (
                <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
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
                <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
                  <h3 className="mb-4 text-sm font-semibold">Success vs Failed Logins</h3>
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={failedVsSuccess}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#f3f4f6" />
                      <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                      <YAxis tick={{ fontSize: 11 }} allowDecimals={false} />
                      <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                      <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                        {failedVsSuccess.map((entry, i) => (
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
      ) : (
        /* ===== Event Log Table ===== */
        <>
          <div className="mb-4 flex items-center gap-2">
            <input
              type="text"
              placeholder="Filter by action (e.g. user.login)"
              value={actionFilter}
              onChange={(e) => setActionFilter(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && loadData()}
              className="w-full max-w-sm rounded-lg border border-gray-300 px-3 py-2"
            />
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
            <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
              <table className="w-full">
                <thead className="border-b border-gray-200 bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Time</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Action</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Actor</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Resource</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Result</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {events.map((event) => (
                    <tr key={event.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm text-gray-500">
                        {event.created_at ? new Date(event.created_at).toLocaleString() : "-"}
                      </td>
                      <td className="px-4 py-3">
                        <span className="font-mono text-xs">{event.action}</span>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">
                        {event.actor_name || (event.actor_id ? event.actor_id.substring(0, 8) : "system")}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">{event.resource_type || "-"}</td>
                      <td className="px-4 py-3">
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${resultColor(event.result)}`}>
                          {event.result}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
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
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold">{value.toLocaleString()}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  );
}
