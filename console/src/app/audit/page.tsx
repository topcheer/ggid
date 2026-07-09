"use client";

import { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { ScrollText, RefreshCw } from "lucide-react";

interface AuditEvent {
  id: string;
  tenant_id: string;
  actor_type: string;
  actor_id: string;
  action: string;
  resource_type: string;
  result: string;
  created_at: string;
}

export default function AuditPage() {
  const { apiFetch } = useApi();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionFilter, setActionFilter] = useState("");

  const loadEvents = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (actionFilter) params.set("action", actionFilter);
      params.set("page_size", "20");
      const data = await apiFetch<{ events?: AuditEvent[]; items?: AuditEvent[] }>(
        `/api/v1/audit/events?${params}`,
      );
      setEvents(data.events || data.items || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load audit events");
      setEvents([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadEvents();
  }, []);

  const resultColor = (result: string) => {
    switch (result) {
      case "success": return "bg-green-100 text-green-700";
      case "failure": return "bg-yellow-100 text-yellow-700";
      case "denied": return "bg-red-100 text-red-700";
      default: return "bg-gray-100 text-gray-600";
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Audit Log</h1>
        <button
          onClick={loadEvents}
          className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50"
        >
          <RefreshCw className="h-4 w-4" />
          Refresh
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="mb-4 flex items-center gap-2">
        <input
          type="text"
          placeholder="Filter by action (e.g. user.login)"
          value={actionFilter}
          onChange={(e) => setActionFilter(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && loadEvents()}
          className="w-full max-w-sm rounded-lg border border-gray-300 px-3 py-2"
        />
      </div>

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : events.length === 0 ? (
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
                    {event.actor_id ? event.actor_id.substring(0, 8) : "system"}
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
    </div>
  );
}
