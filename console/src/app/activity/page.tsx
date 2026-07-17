"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { useApi } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import {
  Activity,
  Download,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  Filter,
  CheckCircle,
  XCircle,
} from "lucide-react";

interface ActivityEvent {
  id: string;
  timestamp: string;
  event_type: string;
  ip_address: string;
  user_agent: string;
  result: "success" | "failure";
}

const EVENT_TYPES = [
  { value: "login", label: "Login" },
  { value: "logout", label: "Logout" },
  { value: "mfa_challenge", label: "MFA Challenge" },
  { value: "token_refresh", label: "Token Refresh" },
  { value: "api_call", label: "API Call" },
  { value: "password_change", label: "Password Change" },
  { value: "role_assigned", label: "Role Assigned" },
];

const PAGE_SIZE = 20;

export default function ActivityLogPage() {
  const { apiFetch } = useApi();
  const { t } = useI18n();
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [eventTypeFilter, setEventTypeFilter] = useState("");
  const [resultFilter, setResultFilter] = useState("");

  // Pagination
  const [page, setPage] = useState(1);

  // Load events
  const loadEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      params.set("actor", "me");
      params.set("page_size", "200");
      if (dateFrom) params.set("from", `${dateFrom}T00:00:00Z`);
      if (dateTo) params.set("to", `${dateTo}T23:59:59Z`);
      if (eventTypeFilter) params.set("action", eventTypeFilter);
      if (resultFilter) params.set("result", resultFilter);

      const data = await apiFetch<
        { events?: ActivityEvent[]; items?: ActivityEvent[] } | ActivityEvent[]
      >(`/api/v1/audit/events?${params}`).catch(() => null);

      if (data) {
        const list = Array.isArray(data) ? data : data.events || data.items || [];
        setEvents(list);
      } else {
        // Try alternate endpoint
        const altData = await apiFetch<{ activity?: ActivityEvent[] } | ActivityEvent[]>(
          "/api/v1/activity",
        ).catch(() => null);
        if (altData) {
          const list = Array.isArray(altData) ? altData : altData.activity || [];
          setEvents(list);
        } else {
          setEvents([]);
        }
      }
    } catch {
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch, dateFrom, dateTo, eventTypeFilter, resultFilter]);

  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  // Reset page when filters change
  useEffect(() => {
    setPage(1);
  }, [dateFrom, dateTo, eventTypeFilter, resultFilter]);

  // Client-side filtering (in case server doesn't support all filters)
  const filteredEvents = useMemo(() => {
    return events.filter((e) => {
      if (dateFrom) {
        const eventDate = new Date(e.timestamp);
        const fromDate = new Date(dateFrom);
        if (eventDate < fromDate) return false;
      }
      if (dateTo) {
        const eventDate = new Date(e.timestamp);
        const toDate = new Date(dateTo);
        toDate.setHours(23, 59, 59, 999);
        if (eventDate > toDate) return false;
      }
      if (eventTypeFilter && e.event_type !== eventTypeFilter) return false;
      if (resultFilter && e.result !== resultFilter) return false;
      return true;
    });
  }, [events, dateFrom, dateTo, eventTypeFilter, resultFilter]);

  const totalPages = Math.max(1, Math.ceil(filteredEvents.length / PAGE_SIZE));
  const currentPage = Math.min(page, totalPages);
  const paginatedEvents = filteredEvents.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE,
  );

  // CSV Export
  const handleExportCSV = () => {
    const headers = ["Timestamp", "Event Type", "IP Address", "User Agent", "Result"];
    const rows = filteredEvents.map((e) => [
      new Date(e.timestamp).toISOString(),
      e.event_type,
      e.ip_address || "",
      e.user_agent || "",
      e.result,
    ]);
    const csv = [headers, ...rows]
      .map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(","))
      .join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `activity-log-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const eventTypeLabel = (type: string): string => {
    const found = EVENT_TYPES.find((t) => t.value === type);
    return found ? found.label : type;
  };

  const resetFilters = () => {
    setDateFrom("");
    setDateTo("");
    setEventTypeFilter("");
    setResultFilter("");
  };

  const hasActiveFilters = dateFrom || dateTo || eventTypeFilter || resultFilter;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Activity className="h-7 w-7 text-brand-600" />
          <div>
            <h1 className="text-2xl font-bold dark:text-gray-100">{t("activity.title")}</h1>
            <p className="text-sm text-gray-500 dark:text-gray-400">{t("activity.subtitle")}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadEvents}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </button>
          <button
            onClick={handleExportCSV}
            disabled={filteredEvents.length === 0}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            <Download className="h-4 w-4" />
            Export CSV
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="mb-4 rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <div className="mb-3 flex items-center gap-1.5 text-sm font-medium text-gray-600">
          <Filter className="h-4 w-4" /> Filters
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <label className="mb-1 block text-xs text-gray-400">{t("activity.fromDate")}</label>
            <input
              type="date"
              value={dateFrom}
              onChange={(e) => setDateFrom(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-gray-400">{t("activity.toDate")}</label>
            <input
              type="date"
              value={dateTo}
              onChange={(e) => setDateTo(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            />
          </div>
          <div>
            <label className="mb-1 block text-xs text-gray-400">{t("activity.eventType")}</label>
            <select
              value={eventTypeFilter}
              onChange={(e) => setEventTypeFilter(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            >
              <option value="">{t("activity.allEvents")}</option>
              {EVENT_TYPES.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-xs text-gray-400">{t("activity.result")}</label>
            <select
              value={resultFilter}
              onChange={(e) => setResultFilter(e.target.value)}
              className="w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500"
            >
              <option value="">{t("activity.allResults")}</option>
              <option value="success">{t("activity.success")}</option>
              <option value="failure">{t("activity.failure")}</option>
            </select>
          </div>
        </div>
        {hasActiveFilters && (
          <button
            onClick={resetFilters}
            className="mt-3 text-xs font-medium text-brand-600 hover:text-brand-700"
          >
            Clear all filters
          </button>
        )}
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Table */}
      <div className="overflow-x-auto overflow-hidden rounded-xl border border-gray-200 bg-white shadow dark:border-gray-700 dark:bg-gray-800-sm dark:border-gray-700 dark:bg-gray-900">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50 text-left text-xs font-semibold uppercase tracking-wider text-gray-500">
              <th scope="col" className="px-4 py-3">{t("activity.timestamp")}</th>
              <th scope="col" className="px-4 py-3">{t("activity.eventType")}</th>
              <th scope="col" className="px-4 py-3">{t("activity.ipAddress")}</th>
              <th scope="col" className="hidden px-4 py-3 lg:table-cell">{t("activity.userAgent")}</th>
              <th scope="col" className="px-4 py-3">{t("activity.result")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {loading && (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-sm text-gray-400">
                  <RefreshCw className="mx-auto mb-2 h-5 w-5 animate-spin text-gray-300" />
                  Loading activity...
                </td>
              </tr>
            )}
            {!loading && paginatedEvents.length === 0 && (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-sm text-gray-400">
                  {t("activity.noEvents")}
                </td>
              </tr>
            )}
            {!loading &&
              paginatedEvents.map((event) => (
                <tr key={event.id} className="hover:bg-gray-50">
                  <td className="whitespace-nowrap px-4 py-3 text-sm text-gray-600">
                    {new Date(event.timestamp).toLocaleString()}
                  </td>
                  <td className="px-4 py-3">
                    <span className="inline-flex rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">
                      {eventTypeLabel(event.event_type)}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-sm font-mono text-gray-600">
                    {event.ip_address || "—"}
                  </td>
                  <td className="hidden max-w-xs truncate px-4 py-3 text-sm text-gray-500 lg:table-cell">
                    {event.user_agent || "—"}
                  </td>
                  <td className="px-4 py-3">
                    {event.result === "success" ? (
                      <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
                        <CheckCircle className="h-3 w-3" /> {t("activity.success")}
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
                        <XCircle className="h-3 w-3" /> {t("activity.failure")}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {filteredEvents.length > 0 && (
        <div className="mt-4 flex items-center justify-between">
          <p className="text-sm text-gray-500">
            Showing {(currentPage - 1) * PAGE_SIZE + 1}–
            {Math.min(currentPage * PAGE_SIZE, filteredEvents.length)} of{" "}
            {filteredEvents.length} events
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(Math.max(1, currentPage - 1))}
              disabled={currentPage <= 1}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
              <ChevronLeft className="h-4 w-4" /> Prev
            </button>
            <span className="text-sm text-gray-600">
              Page {currentPage} of {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages, currentPage + 1))}
              disabled={currentPage >= totalPages}
              className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
              Next <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
