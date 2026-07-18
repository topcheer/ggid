"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  LogIn, LogOut, Shield, FileEdit, UserCheck, Key,
  RefreshCw, Download, Search, Filter,
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
  service?: string;
  severity?: string;
  description?: string;
  created_at: string;
  ip_address?: string;
  user_agent?: string;
  metadata?: Record<string, unknown>;
}

type Severity = "info" | "warning" | "error" | "critical";

const SEVERITY_CONFIG: Record<Severity, { color: string; border: string; dot: string; badge: string; line: string }> = {
  info:     { color: "text-blue-700",     border: "border-blue-300",     dot: "bg-blue-500",     badge: "bg-blue-100 text-blue-700",     line: "bg-blue-300" },
  warning:  { color: "text-yellow-700",   border: "border-yellow-300",   dot: "bg-yellow-500",   badge: "bg-yellow-100 text-yellow-700", line: "bg-yellow-300" },
  error:    { color: "text-red-700",      border: "border-red-300",      dot: "bg-red-500",      badge: "bg-red-100 text-red-700",       line: "bg-red-300" },
  critical: { color: "text-purple-700",   border: "border-purple-300",   dot: "bg-purple-500",   badge: "bg-purple-100 text-purple-700", line: "bg-purple-300" },
};

const ACTION_ICONS: Record<string, React.ElementType> = {
  login: LogIn,
  "user.login": LogIn,
  "user.register": UserCheck,
  logout: LogOut,
  "user.logout": LogOut,
  mfa: Shield,
  "mfa.challenge": Shield,
  "mfa.verify": Shield,
  policy: FileEdit,
  "policy.create": FileEdit,
  "policy.update": FileEdit,
  "policy.evaluate": FileEdit,
  role: UserCheck,
  "role.create": UserCheck,
  "role.assign": UserCheck,
  "role.update": UserCheck,
  token: Key,
  "token.issue": Key,
  "token.refresh": Key,
  "token.revoke": Key,
  "oauth.authorize": Key,
};

function getEventIcon(action: string): React.ElementType {
  const t = useTranslations();

  return ACTION_ICONS[action] || (action.includes("login") ? LogIn : action.includes("logout") ? LogOut : action.includes("mfa") ? Shield : action.includes("policy") ? FileEdit : action.includes("role") ? UserCheck : action.includes("token") || action.includes("oauth") ? Key : FileEdit);
}

function inferSeverity(event: AuditEvent): Severity {
  if (event.severity) return event.severity as Severity;
  if (event.result === "denied") return "critical";
  if (event.result === "failure") return "error";
  if (event.action?.includes("mfa") || event.action?.includes("denied")) return "warning";
  return "info";
}

function inferService(event: AuditEvent): string {
  if (event.service) return event.service;
  if (event.action?.startsWith("user.") || event.action?.startsWith("auth.")) return "auth";
  if (event.action?.startsWith("oauth.") || event.action?.startsWith("token.")) return "oauth";
  if (event.action?.startsWith("policy.") || event.action?.startsWith("role.")) return "policy";
  if (event.action?.startsWith("org.") || event.action?.startsWith("member.")) return "identity";
  return "audit";
}

export default function AuditTimelinePage() {
  const { apiFetch } = useApi();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [userSearch, setUserSearch] = useState("");
  const [serviceFilter, setServiceFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  const loadEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      params.set("page_size", "50");
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
      setEvents(list);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load events");
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch, userSearch, dateFrom, dateTo]);

  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  // Client-side filtering for service and severity (may not be server-side filterable)
  const filteredEvents = events.filter((e: any) => {
    if (serviceFilter !== "all" && inferService(e) !== serviceFilter) return false;
    if (severityFilter !== "all" && inferSeverity(e) !== severityFilter) return false;
    if (userSearch) {
      const u = userSearch.toLowerCase();
      const matches =
        (e.actor_name || "").toLowerCase().includes(u) ||
        (e.actor_id || "").toLowerCase().includes(u);
      if (!matches) return false;
    }
    return true;
  });

  const handlePrint = () => {
    window.print();
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  return (
    <div>
      <style jsx global>{`
        @media print {
          body * { visibility: hidden; }
          #audit-timeline, #audit-timeline * { visibility: visible; }
          #audit-timeline { position: absolute; left: 0; top: 0; width: 100%; }
          .no-print { display: none !important; }
          .timeline-card { break-inside: avoid; page-break-inside: avoid; }
        }
      `}</style>

      <div className="mb-6 flex items-center justify-between no-print">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Audit Timeline</h1>
        <div className="flex gap-2">
          <button
            onClick={handlePrint}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <Download className="h-4 w-4" /> Export PDF
          </button>
          <button
            onClick={loadEvents}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 no-print">
          {error}
          <p className="mt-1 text-xs">Make sure the Audit Service (:8072) is running.</p>
        </div>
      )}

      {/* Filter Bar */}
      <div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-5 no-print">
        <div className="relative lg:col-span-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search user..."
            value={userSearch}
            onChange={(e) => setUserSearch(e.target.value)}
            className={`${inputCls} pl-9`}
          />
        </div>
        <select
          value={serviceFilter}
          onChange={(e) => setServiceFilter(e.target.value)}
          className={inputCls}
        >
          <option value="all">All Services</option>
          <option value="auth">Auth</option>
          <option value="oauth">OAuth</option>
          <option value="identity">Identity</option>
          <option value="policy">Policy</option>
          <option value="audit">Audit</option>
        </select>
        <select
          value={severityFilter}
          onChange={(e) => setSeverityFilter(e.target.value)}
          className={inputCls}
        >
          <option value="all">All Severities</option>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="error">Error</option>
          <option value="critical">Critical</option>
        </select>
        <input
          type="date"
          value={dateFrom}
          onChange={(e) => setDateFrom(e.target.value)}
          className={inputCls}
        />
        <input
          type="date"
          value={dateTo}
          onChange={(e) => setDateTo(e.target.value)}
          className={inputCls}
        />
      </div>

      {loading ? (
        <div className="py-12 text-center text-gray-500">Loading events...</div>
      ) : filteredEvents.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Filter className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No events found</p>
          <p className="mt-1 text-xs text-gray-400">
            Try adjusting filters or wait for audit events to arrive via NATS.
          </p>
        </div>
      ) : (
        <div id="audit-timeline" className="relative">
          {/* Vertical line */}
          <div className="absolute left-5 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-700" />

          <div className="space-y-4">
            {filteredEvents.map((event: any, idx: any) => {
              const sev = inferSeverity(event);
              const cfg = SEVERITY_CONFIG[sev];
              const Icon = getEventIcon(event.action);
              const service = inferService(event);
              const time = event.created_at
                ? new Date(event.created_at).toLocaleString("en-US", {
                    month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit",
                  })
                : "Unknown";

              return (
                <div key={event.id || idx} className="timeline-card relative pl-14">
                  {/* Timeline dot */}
                  <div
                    className={`absolute left-2 top-4 flex h-7 w-7 items-center justify-center rounded-full border-2 border-white dark:border-gray-800 ${cfg.dot} shadow-sm`}
                  >
                    <Icon className="h-3.5 w-3.5 text-white" />
                  </div>

                  {/* Event card */}
                  <div className={`rounded-xl border ${cfg.border} bg-white p-4 shadow-sm dark:bg-gray-800`}>
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex-1">
                        <div className="mb-1 flex items-center gap-2">
                          <span className="font-mono text-sm font-semibold text-gray-900 dark:text-gray-100">
                            {event.action}
                          </span>
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cfg.badge}`}>
                            {sev}
                          </span>
                          <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                            {service}
                          </span>
                        </div>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {event.description || event.resource_type || "No description available"}
                        </p>
                        <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                          <span className="flex items-center gap-1">
                            <UserCheck className="h-3 w-3" />
                            {event.actor_name || event.actor_id?.substring(0, 8) || "system"}
                          </span>
                          <span>{time}</span>
                          {event.ip_address && (
                            <span className="font-mono">{event.ip_address}</span>
                          )}
                          <span className={`font-medium ${
                            event.result === "success" ? "text-green-600" : "text-red-500"
                          }`}>
                            {event.result}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
