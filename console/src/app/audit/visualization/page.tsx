"use client";

import { useEffect, useState, useCallback, useRef, Fragment } from "react";
import { useApi } from "@/lib/api";
import {
  LogIn,
  LogOut,
  Shield,
  Key,
  FileEdit,
  UserCheck,
  Globe,
  Search,
  Download,
  Pause,
  Play,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  Activity,
  Code2,
  Clock,
  MapPin,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AuditEvent {
  id: string;
  tenant_id?: string;
  actor_type?: string;
  actor_id?: string;
  actor_name?: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  result: string;
  created_at: string;
  ip_address?: string;
  user_agent?: string;
  request_id?: string;
  session_id?: string;
  metadata?: Record<string, unknown>;
}

interface EventGroup {
  key: string;
  events: AuditEvent[];
  startTime: string;
}

const ACTION_TYPES = [
  "login",
  "logout",
  "mfa_challenge",
  "token_refresh",
  "password_change",
  "role_assigned",
  "api_call",
  "config_change",
];

const ACTION_ICONS: Record<string, React.ElementType> = {
  login: LogIn,
  "user.login": LogIn,
  logout: LogOut,
  "user.logout": LogOut,
  mfa_challenge: Shield,
  "mfa.challenge": Shield,
  token_refresh: Key,
  "token.refresh": Key,
  password_change: Key,
  "password.change": Key,
  role_assigned: UserCheck,
  "role.assign": UserCheck,
  api_call: Globe,
  config_change: FileEdit,
  "config.change": FileEdit,
};

function getActionIcon(action: string): React.ElementType {
  const t = useTranslations();

  return (
    ACTION_ICONS[action] ||
    (action.includes("login")
      ? LogIn
      : action.includes("logout")
        ? LogOut
        : action.includes("mfa")
          ? Shield
          : action.includes("token") || action.includes("password")
            ? Key
            : action.includes("role")
              ? UserCheck
              : action.includes("config") || action.includes("policy")
                ? FileEdit
                : action.includes("api") || action.includes("call")
                  ? Globe
                  : Activity)
  );
}

function getInitials(name?: string): string {
  if (!name) return "?";
  const parts = name.trim().split(/[\s@._-]+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

function groupEvents(events: AuditEvent[]): EventGroup[] {
  if (events.length === 0) return [];
  const groups: EventGroup[] = [];
  let currentGroup: AuditEvent[] = [events[0]];
  let groupStart = new Date(events[0].created_at).getTime();

  for (let i = 1; i < events.length; i++) {
    const eventTime = new Date(events[i].created_at).getTime();
    // Group events within 5 minutes (300000ms) of each other
    if (eventTime - groupStart <= 300000) {
      currentGroup.push(events[i]);
    } else {
      groups.push({
        key: `group-${groups.length}`,
        events: currentGroup,
        startTime: events[Math.max(0, i - currentGroup.length)].created_at,
      });
      currentGroup = [events[i]];
      groupStart = eventTime;
    }
  }
  groups.push({
    key: `group-${groups.length}`,
    events: currentGroup,
    startTime: currentGroup[0]?.created_at || "",
  });
  return groups;
}

function formatTime(dateStr: string): string {
  const d = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  if (diffMs < 60000) return "Just now";
  if (diffMs < 3600000) return `${Math.floor(diffMs / 60000)}m ago`;
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export default function AuditVisualizationPage() {
  const { apiFetch } = useApi();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [userSearch, setUserSearch] = useState("");
  const [actionFilter, setActionFilter] = useState("all");
  const [ipSearch, setIpSearch] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [resultFilter, setResultFilter] = useState("all");

  // Real-time polling
  const [isLive, setIsLive] = useState(true);
  const [expandedEvent, setExpandedEvent] = useState<string | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set());
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadEvents = useCallback(async () => {
    try {
      const params = new URLSearchParams();
      params.set("page_size", "100");
      if (actionFilter !== "all") params.set("action", actionFilter);
      if (resultFilter !== "all") params.set("result", resultFilter);
      if (userSearch) params.set("actor_id", userSearch);
      if (dateFrom) params.set("from", dateFrom + "T00:00:00Z");
      if (dateTo) params.set("to", dateTo + "T23:59:59Z");

      const data = await apiFetch<{ events?: AuditEvent[] } | AuditEvent[]>(
        `/api/v1/audit/events?${params}`,
      );
      let list: AuditEvent[];
      if (Array.isArray(data)) {
        list = data;
      } else {
        list = data.events || [];
      }
      // Client-side IP filter fallback
      if (ipSearch) {
        list = list.filter((e: any) => (e.ip_address || "").includes(ipSearch));
      }
      setEvents(list);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load events");
    } finally {
      setLoading(false);
    }
  }, [apiFetch, actionFilter, resultFilter, userSearch, dateFrom, dateTo, ipSearch]);

  // Initial load
  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  // Polling for real-time updates
  useEffect(() => {
    if (isLive) {
      intervalRef.current = setInterval(() => {
        loadEvents();
      }, 5000);
    }
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [isLive, loadEvents]);

  const handleExportCSV = () => {
    const headers = ["timestamp", "user", "action", "ip", "result", "details"];
    const rows = filteredEvents.map((e: any) => [
      e.created_at,
      e.actor_name || e.actor_id || "",
      e.action,
      e.ip_address || "",
      e.result,
      e.metadata ? JSON.stringify(e.metadata) : "",
    ]);
    const csv = [headers, ...rows]
      .map((row: any) => row.map((cell: any) => `"${String(cell).replace(/"/g, '""')}"`).join(","))
      .join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `audit-events-${new Date().toISOString().split("T")[0]}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const filteredEvents = events.filter((e: any) => {
    if (userSearch) {
      const u = userSearch.toLowerCase();
      if (
        !(e.actor_name || "").toLowerCase().includes(u) &&
        !(e.actor_id || "").toLowerCase().includes(u)
      )
        return false;
    }
    if (actionFilter !== "all") {
      const actionMatch =
        e.action === actionFilter ||
        e.action.includes(actionFilter) ||
        e.action.replace(/^(user|token|mfa|role|config|policy)\./, "") === actionFilter;
      if (!actionMatch) return false;
    }
    if (ipSearch && !(e.ip_address || "").includes(ipSearch)) return false;
    if (resultFilter !== "all" && e.result !== resultFilter) return false;
    if (dateFrom) {
      if (new Date(e.created_at) < new Date(dateFrom + "T00:00:00Z")) return false;
    }
    if (dateTo) {
      if (new Date(e.created_at) > new Date(dateTo + "T23:59:59Z")) return false;
    }
    return true;
  });

  const eventGroups = groupEvents(filteredEvents);
  const expandedGroups = eventGroups.filter((g: any) => !collapsedGroups.has(g.key));

  const toggleGroup = (key: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const toggleEvent = (id: string) => {
    setExpandedEvent((prev) => (prev === id ? null : id));
  };

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Audit Timeline Visualization
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Real-time chronological event feed with grouping and details
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Live indicator */}
          {isLive && (
            <div className="flex items-center gap-2 rounded-lg bg-green-50 px-3 py-2 dark:bg-green-900/20">
              <span className="relative flex h-2.5 w-2.5">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
                <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-green-500" />
              </span>
              <span className="text-sm font-medium text-green-700 dark:text-green-400">Live</span>
            </div>
          )}
          <button
            onClick={() => setIsLive(!isLive)}
            className={`flex items-center gap-1.5 rounded-lg border px-3 py-2 text-sm transition-colors ${
              isLive
                ? "border-yellow-300 bg-yellow-50 text-yellow-700 hover:bg-yellow-100 dark:border-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400"
                : "border-green-300 bg-green-50 text-green-700 hover:bg-green-100 dark:border-green-700 dark:bg-green-900/20 dark:text-green-400"
            }`}
          >
            {isLive ? (
              <>
                <Pause className="h-4 w-4" /> Pause
              </>
            ) : (
              <>
                <Play className="h-4 w-4" /> Resume
              </>
            )}
          </button>
          <button
            onClick={handleExportCSV}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <Download className="h-4 w-4" /> CSV
          </button>
          <button
            onClick={loadEvents}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400">
          {error}
          <p className="mt-1 text-xs">Make sure the Audit Service (:8072) is running.</p>
        </div>
      )}

      {/* Filters Bar */}
      <div className="mb-6 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search user..."
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
              className={`${inputCls} pl-9`}
            />
          </div>
          <select aria-label="Filter" value={actionFilter} onChange={(e) => setActionFilter(e.target.value)} className={inputCls}>
            <option value="all">All Actions</option>
            {ACTION_TYPES.map((a: any) => (
              <option key={a} value={a}>
                {a.replace(/_/g, " ")}
              </option>
            ))}
          </select>
          <div className="relative">
            <MapPin className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="IP address..."
              value={ipSearch}
              onChange={(e) => setIpSearch(e.target.value)}
              className={`${inputCls} pl-9`}
            />
          </div>
          <input
            type="date"
            placeholder="From"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            className={inputCls}
          />
          <input
            type="date"
            placeholder="To"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            className={inputCls}
          />
          <select aria-label="Filter" value={resultFilter} onChange={(e) => setResultFilter(e.target.value)} className={inputCls}>
            <option value="all">All Results</option>
            <option value="success">Success</option>
            <option value="failure">Failure</option>
            <option value="denied">Denied</option>
          </select>
        </div>
      </div>

      {/* Event count */}
      <div className="mb-3 flex items-center justify-between text-sm text-gray-500 dark:text-gray-400">
        <span>
          {filteredEvents.length} event{filteredEvents.length !== 1 ? "s" : ""}
          {eventGroups.length > 1 && `, ${eventGroups.length} groups`}
        </span>
      </div>

      {loading ? (
        <div className="py-12 text-center text-gray-500">Loading events...</div>
      ) : filteredEvents.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Activity className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">No events found</p>
          <p className="mt-1 text-xs text-gray-400">Try adjusting filters or wait for events to arrive.</p>
        </div>
      ) : (
        <div className="relative">
          {/* Vertical timeline line */}
          <div className="absolute left-5 top-0 bottom-0 w-0.5 bg-gradient-to-b from-gray-200 via-gray-300 to-gray-200 dark:from-gray-700 dark:via-gray-600 dark:to-gray-700" />

          <div className="space-y-2">
            {eventGroups.map((group: any) => {
              const isCollapsed = collapsedGroups.has(group.key);
              const isMulti = group.events.length > 1;

              return (
                <Fragment key={group.key}>
                  {/* Group header for collapsed multi-event groups */}
                  {isMulti && (
                    <div className="relative pl-14">
                      <button
                        onClick={() => toggleGroup(group.key)}
                        className="flex items-center gap-2 rounded-lg bg-gray-50 px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700"
                      >
                        {isCollapsed ? (
                          <ChevronRight className="h-3.5 w-3.5" />
                        ) : (
                          <ChevronDown className="h-3.5 w-3.5" />
                        )}
                        <span className="h-2 w-2 rounded-full bg-indigo-400" />
                        {group.events.length} events
                        <span className="text-gray-400">
                          around {formatTime(group.startTime)}
                        </span>
                      </button>
                    </div>
                  )}

                  {/* Render events in group (hidden if collapsed) */}
                  {!isCollapsed &&
                    group.events.map((event: any, idx: any) => {
                      const Icon = getActionIcon(event.action);
                      const initials = getInitials(event.actor_name || event.actor_id);
                      const isSuccess = event.result === "success";
                      const isExpanded = expandedEvent === event.id;

                      return (
                        <div key={event.id || `${group.key}-${idx}`} className="relative pl-14">
                          {/* Timeline dot with avatar */}
                          <div className="absolute left-0 top-3 flex h-10 w-10 items-center justify-center">
                            <div
                              className={`flex h-10 w-10 items-center justify-center rounded-full border-2 border-white text-xs font-bold shadow-sm dark:border-gray-800 ${
                                isSuccess
                                  ? "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300"
                                  : "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300"
                              }`}
                            >
                              {initials}
                            </div>
                          </div>

                          {/* Event card */}
                          <button
                            onClick={() => toggleEvent(event.id)}
                            className={`w-full overflow-hidden rounded-xl border text-left shadow-sm transition-all hover:shadow-md ${
                              isExpanded
                                ? "border-indigo-300 ring-1 ring-indigo-200 dark:border-indigo-700 dark:ring-indigo-800"
                                : isSuccess
                                  ? "border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800"
                                  : "border-red-200 bg-red-50/30 dark:border-red-900/50 dark:bg-gray-800"
                            }`}
                          >
                            <div className="p-4">
                              <div className="flex items-start justify-between gap-3">
                                <div className="flex items-start gap-3">
                                  {/* Action icon */}
                                  <div
                                    className={`mt-0.5 flex h-8 w-8 items-center justify-center rounded-lg ${
                                      isSuccess
                                        ? "bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400"
                                        : "bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-400"
                                    }`}
                                  >
                                    <Icon className="h-4 w-4" />
                                  </div>
                                  <div>
                                    <div className="flex flex-wrap items-center gap-2">
                                      <span className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">
                                        {event.action}
                                      </span>
                                      {/* Result badge */}
                                      <span
                                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                                          isSuccess
                                            ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                                            : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
                                        }`}
                                      >
                                        {event.result}
                                      </span>
                                    </div>
                                    <div className="mt-1 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                                      <span className="flex items-center gap-1 font-medium text-gray-700 dark:text-gray-300">
                                        {event.actor_name || event.actor_id?.substring(0, 8) || "system"}
                                      </span>
                                      <span className="flex items-center gap-1">
                                        <Clock className="h-3 w-3" />
                                        {formatTime(event.created_at)}
                                      </span>
                                      {event.ip_address && (
                                        <span className="flex items-center gap-1 font-mono">
                                          <MapPin className="h-3 w-3" />
                                          {event.ip_address}
                                        </span>
                                      )}
                                    </div>
                                  </div>
                                </div>
                                <div className="flex items-center gap-2">
                                  {isExpanded && (
                                    <ChevronDown className="h-4 w-4 text-gray-400" />
                                  )}
                                  {!isExpanded && (
                                    <ChevronRight className="h-4 w-4 text-gray-400" />
                                  )}
                                </div>
                              </div>

                              {/* Expandable details */}
                              {isExpanded && (
                                <div className="mt-3 border-t border-gray-100 pt-3 dark:border-gray-700">
                                  <div className="grid gap-3 sm:grid-cols-2">
                                    {event.user_agent && (
                                      <div>
                                        <label className="text-xs font-medium uppercase tracking-wide text-gray-400">
                                          User Agent
                                        </label>
                                        <p className="mt-0.5 break-all font-mono text-xs text-gray-600 dark:text-gray-300">
                                          {event.user_agent}
                                        </p>
                                      </div>
                                    )}
                                    {event.tenant_id && (
                                      <div>
                                        <label className="text-xs font-medium uppercase tracking-wide text-gray-400">
                                          Tenant ID
                                        </label>
                                        <p className="mt-0.5 font-mono text-xs text-gray-600 dark:text-gray-300">
                                          {event.tenant_id}
                                        </p>
                                      </div>
                                    )}
                                    {event.request_id && (
                                      <div>
                                        <label className="text-xs font-medium uppercase tracking-wide text-gray-400">
                                          Request ID
                                        </label>
                                        <p className="mt-0.5 font-mono text-xs text-gray-600 dark:text-gray-300">
                                          {event.request_id}
                                        </p>
                                      </div>
                                    )}
                                    {event.session_id && (
                                      <div>
                                        <label className="text-xs font-medium uppercase tracking-wide text-gray-400">
                                          Session ID
                                        </label>
                                        <p className="mt-0.5 font-mono text-xs text-gray-600 dark:text-gray-300">
                                          {event.session_id}
                                        </p>
                                      </div>
                                    )}
                                  </div>
                                  {event.metadata && Object.keys(event.metadata).length > 0 && (
                                    <div className="mt-3">
                                      <label className="flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-gray-400">
                                        <Code2 className="h-3 w-3" />
                                        Additional Context
                                      </label>
                                      <pre className="mt-1 overflow-x-auto rounded-lg bg-gray-50 p-3 text-xs text-gray-700 dark:bg-gray-900 dark:text-gray-300">
                                        {JSON.stringify(event.metadata, null, 2)}
                                        </pre>
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          </button>
                        </div>
                      );
                    })}

                  {/* Show collapsed group summary */}
                  {isCollapsed && (
                    <div className="relative pl-14 pb-2">
                      <div className="rounded-lg border border-dashed border-gray-200 p-3 text-center text-xs text-gray-400 dark:border-gray-700">
                        {group.events.length} events hidden — click header to expand
                      </div>
                    </div>
                  )}
                </Fragment>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
