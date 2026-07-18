"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Search,
  Filter,
  ChevronDown,
  ChevronRight,
  Loader2,
  RefreshCw,
  Globe,
  User,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Download,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AuditEvent {
  id: string;
  action: string;
  actor_id: string;
  actor_name: string;
  resource_type: string;
  resource_id: string;
  result: "success" | "failure" | "denied";
  tenant_id: string;
  created_at: string;
  metadata?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
}

const EVENT_TYPES = [
  "user.login",
  "user.login.success",
  "user.login.failed",
  "user.register",
  "user.update",
  "user.delete",
  "role.create",
  "role.update",
  "role.delete",
  "org.create",
  "org.update",
  "policy.evaluate",
  "token.refresh",
  "token.revoke",
  "session.create",
  "session.revoke",
];

const RESOURCE_TYPES = [
  "user",
  "role",
  "organization",
  "policy",
  "session",
  "token",
  "webhook",
  "agent",
];

export default function AuditEventsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Filters
  const [actionFilter, setActionFilter] = useState("");
  const [resourceFilter, setResourceFilter] = useState("");
  const [actorFilter, setActorFilter] = useState("");
  const [resultFilter, setResultFilter] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");

  // Pagination
  const [page, setPage] = useState(1);
  const [pageSize] = useState(25);
  const [total, setTotal] = useState(0);

  const loadEvents = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      params.set("page_size", String(pageSize));
      params.set("page", String(page));
      if (actionFilter) params.set("action", actionFilter);
      if (resourceFilter) params.set("resource_type", resourceFilter);
      if (actorFilter) params.set("actor_id", actorFilter);
      if (resultFilter) params.set("result", resultFilter);
      if (dateFrom) params.set("date_from", dateFrom);
      if (dateTo) params.set("date_to", dateTo);

      const data = await apiFetch<{ events?: AuditEvent[]; total?: number; total_pages?: number }>(
        `/api/v1/audit/events?${params.toString()}`
      );
      setEvents(data.events ?? []);
      setTotal(data.total ?? 0);
    } catch {
      setError("Failed to load audit events");
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch, page, pageSize, actionFilter, resourceFilter, actorFilter, resultFilter, dateFrom, dateTo]);

  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  const handleResetFilters = () => {
    setActionFilter("");
    setResourceFilter("");
    setActorFilter("");
    setResultFilter("");
    setDateFrom("");
    setDateTo("");
    setPage(1);
  };

  const handleExport = () => {
    const csv = [
      "ID,Action,Actor,Resource,Result,Timestamp,IP",
      ...events.map((e: any) =>
        `${e.id},${e.action},${e.actor_name || ""},${e.resource_type}/${e.resource_id},${e.result},${e.created_at},${e.ip_address || ""}`
      ),
    ].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `audit-events-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const totalPages = Math.ceil(total / pageSize);

  const inputCls =
    "rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const selectCls =
    "rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const resultIcon = (result: string) => {
    if (result === "success")
      return <CheckCircle2 className="h-4 w-4 text-green-500" />;
    if (result === "failure")
      return <XCircle className="h-4 w-4 text-red-500" />;
    return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Search className="h-7 w-7 text-indigo-600" />
            Audit Events Explorer
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Search and explore audit events in real time. {total} events found.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleExport}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <Download className="mr-1 inline h-4 w-4" /> Export CSV
          </button>
          <button
            onClick={loadEvents}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            <RefreshCw className="mr-1 inline h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className={`${cardCls} space-y-4`}>
        <div className="flex items-center gap-2 text-sm font-medium text-gray-600 dark:text-gray-300">
          <Filter className="h-4 w-4" /> Filters
        </div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-3 lg:grid-cols-6">
          <select
            className={selectCls}
            value={actionFilter}
            onChange={(e) => { setActionFilter(e.target.value); setPage(1); }}
          >
            <option value="">All Actions</option>
            {EVENT_TYPES.map((t: any) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          <select
            className={selectCls}
            value={resourceFilter}
            onChange={(e) => { setResourceFilter(e.target.value); setPage(1); }}
          >
            <option value="">All Resources</option>
            {RESOURCE_TYPES.map((r: any) => (
              <option key={r} value={r}>{r}</option>
            ))}
          </select>
          <input
            className={inputCls}
            placeholder="Actor ID..."
            value={actorFilter}
            onChange={(e) => { setActorFilter(e.target.value); setPage(1); }}
          />
          <select
            className={selectCls}
            value={resultFilter}
            onChange={(e) => { setResultFilter(e.target.value); setPage(1); }}
          >
            <option value="">All Results</option>
            <option value="success">Success</option>
            <option value="failure">Failure</option>
            <option value="denied">Denied</option>
          </select>
          <input
            type="date"
            className={inputCls}
            value={dateFrom}
            onChange={(e) => { setDateFrom(e.target.value); setPage(1); }}
          />
          <input
            type="date"
            className={inputCls}
            value={dateTo}
            onChange={(e) => { setDateTo(e.target.value); setPage(1); }}
          />
        </div>
        <div className="flex justify-between text-xs text-gray-400">
          <button onClick={handleResetFilters} className="text-indigo-500 hover:underline">
            Reset filters
          </button>
          <span>
            Page {page} of {totalPages || 1}
          </span>
        </div>
      </div>

      {/* Events table */}
      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : events.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <Search className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">No events found matching your filters.</p>
        </div>
      ) : (
        <div className={`${cardCls} overflow-hidden p-0`}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs uppercase text-gray-400 dark:border-gray-700">
                  <th scope="col" className="px-4 py-3"></th>
                  <th scope="col" className="px-4 py-3">Action</th>
                  <th scope="col" className="px-4 py-3">Actor</th>
                  <th scope="col" className="px-4 py-3">Resource</th>
                  <th scope="col" className="px-4 py-3">Result</th>
                  <th scope="col" className="px-4 py-3">IP</th>
                  <th scope="col" className="px-4 py-3">Time</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event: any) => (
                  <>
                    <tr
                      key={event.id}
                      className="cursor-pointer border-b border-gray-100 hover:bg-gray-50 dark:border-gray-700/50 dark:hover:bg-gray-700/30"
                      onClick={() => setExpandedId(expandedId === event.id ? null : event.id)}
                    >
                      <td className="px-4 py-3">
                        {expandedId === event.id ? (
                          <ChevronDown className="h-4 w-4 text-gray-400" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-gray-400" />
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <code className="rounded bg-gray-100 px-1.5 py-0.5 text-xs dark:bg-gray-700">
                          {event.action}
                        </code>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          <User className="h-3.5 w-3.5 text-gray-400" />
                          <span className="truncate">{event.actor_name || event.actor_id}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-gray-600 dark:text-gray-300">
                        {event.resource_type}/{event.resource_id.slice(0, 12)}
                      </td>
                      <td className="px-4 py-3">
                        <span className="flex items-center gap-1">
                          {resultIcon(event.result)}
                          <span className={
                            event.result === "success" ? "text-green-600" :
                            event.result === "failure" ? "text-red-600" : "text-yellow-600"
                          }>
                            {event.result}
                          </span>
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <span className="flex items-center gap-1 text-xs text-gray-400">
                          <Globe className="h-3 w-3" />
                          {event.ip_address || "—"}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs text-gray-400">
                        {new Date(event.created_at).toLocaleString()}
                      </td>
                    </tr>
                    {expandedId === event.id && (
                      <tr className="bg-gray-50 dark:bg-gray-800/50">
                        <td colSpan={7} className="px-8 py-4">
                          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <div>
                              <h4 className="mb-2 text-xs font-semibold uppercase text-gray-400">Details</h4>
                              <dl className="space-y-1 text-xs">
                                <div className="flex gap-2">
                                  <dt className="font-mono text-gray-500">event_id:</dt>
                                  <dd className="font-mono text-gray-700 dark:text-gray-300">{event.id}</dd>
                                </div>
                                <div className="flex gap-2">
                                  <dt className="font-mono text-gray-500">tenant_id:</dt>
                                  <dd className="font-mono text-gray-700 dark:text-gray-300">{event.tenant_id}</dd>
                                </div>
                                <div className="flex gap-2">
                                  <dt className="font-mono text-gray-500">resource_id:</dt>
                                  <dd className="font-mono text-gray-700 dark:text-gray-300">{event.resource_id}</dd>
                                </div>
                                {event.user_agent && (
                                  <div className="flex gap-2">
                                    <dt className="font-mono text-gray-500">user_agent:</dt>
                                    <dd className="font-mono break-all text-gray-700 dark:text-gray-300">{event.user_agent}</dd>
                                  </div>
                                )}
                              </dl>
                            </div>
                            {event.metadata && Object.keys(event.metadata).length > 0 && (
                              <div>
                                <h4 className="mb-2 text-xs font-semibold uppercase text-gray-400">Metadata (JSON)</h4>
                                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400">
                                  {JSON.stringify(event.metadata, null, 2)}
                                </pre>
                              </div>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-between border-t border-gray-200 px-4 py-3 dark:border-gray-700">
            <span className="text-xs text-gray-400">
              Showing {(page - 1) * pageSize + 1}–{Math.min(page * pageSize, total)} of {total}
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Previous
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
