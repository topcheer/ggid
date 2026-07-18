"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { Search, Download, Filter, LogIn, LogOut, Shield, Key, UserCheck, FileEdit, AlertTriangle, ChevronLeft, Loader2, AlertCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ActivityEvent {
  id: string;
  timestamp: string;
  eventType: string;
  description: string;
  ip: string;
  device: string;
  result: "success" | "failure";
  details?: Record<string, string | number>;
}

const EVENT_TYPES = [
  { value: "all", label: "All Events" },
  { value: "login", label: "Login" },
  { value: "logout", label: "Logout" },
  { value: "mfa_challenge", label: "MFA Challenge" },
  { value: "token_refresh", label: "Token Refresh" },
  { value: "password_change", label: "Password Change" },
  { value: "role_assigned", label: "Role Assigned" },
  { value: "api_key_used", label: "API Key Usage" },
];

const getEventIcon = (type: string) => {
  const t = useTranslations();

  switch (type) {
    case "login": return LogIn;
    case "logout": return LogOut;
    case "mfa_challenge": return Shield;
    case "token_refresh": return Key;
    case "password_change": return Key;
    case "role_assigned": return UserCheck;
    case "api_key_used": return Key;
    default: return FileEdit;
  }
};

export default function UserActivityPage({ params }: { params: { id: string } }) {
  const { } = useApi();
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true); setError("");
    fetch(`/api/v1/identity/users/${params.id}/activity`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } })
      .then(async (res) => {
        if (res.ok) { const data = await res.json(); setEvents(Array.isArray(data.events) ? data.events : Array.isArray(data) ? data : []); }
        else { setError(`Failed to load activity: HTTP ${res.status}`); }
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Failed to load activity events"))
      .finally(() => setLoading(false));
  }, [params.id]);

  const exportCsv = () => {
    const header = "timestamp,eventType,description,ip,device,result\n";
    const rows = filtered.map((e: any) => `${e.timestamp},${e.eventType},${e.description},${e.ip},${e.device},${e.result}`).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = `user-${params.id}-activity.csv`;
    a.click(); URL.revokeObjectURL(url);
  };

  const filtered = events.filter((e: any) => {
    if (filter !== "all" && e.eventType !== filter) return false;
    if (!search) return true;
    const q = search.toLowerCase();
    return e.description.toLowerCase().includes(q) || e.ip.toLowerCase().includes(q) || e.device.toLowerCase().includes(q);
  });

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <button aria-label="Go back" onClick={() => history.back()} className="rounded-lg border border-gray-300 p-2 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800">
          <ChevronLeft className="w-5 h-5" />
        </button>
        <div>
          <h1 className="text-2xl font-bold dark:text-white">User Activity</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Timeline of events for user {params.id}</p>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" /> {error}
        </div>
      )}
      {loading && <div className="flex items-center gap-2 text-sm text-gray-500"><Loader2 className="h-4 w-4 animate-spin" /> Loading activity events...</div>}

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Total Events</p>
          <p className="text-xl font-bold dark:text-white">{events.length}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Logins</p>
          <p className="text-xl font-bold text-green-500">{events.filter(e => e.eventType === "login" && e.result === "success").length}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Failed Attempts</p>
          <p className="text-xl font-bold text-red-500">{events.filter(e => e.result === "failure").length}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Unique IPs</p>
          <p className="text-xl font-bold text-indigo-500">{new Set(events.map(e => e.ip)).size}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input aria-label="Search activity" type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search by description or IP..."
            className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white" />
        </div>
        <select aria-label="Filter event type" value={filter} onChange={(e) => setFilter(e.target.value)}
          className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white">
          {EVENT_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select>
        <button aria-label="Export activity as CSV" onClick={exportCsv} className="flex items-center gap-1.5 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition">
          <Download className="w-4 h-4" /> CSV
        </button>
      </div>

      {/* Timeline */}
      <div className="space-y-2">
        {filtered.map((event: any, idx: any) => {
          const Icon = getEventIcon(event.eventType);
          const isExpanded = expandedId === event.id;
          return (
            <div key={event.id}>
              <button aria-label={`${event.description} event`} onClick={() => setExpandedId(isExpanded ? null : event.id)}
                className={`w-full text-left flex items-center gap-4 p-3 rounded-xl border transition ${event.result === "failure" ? "border-red-200 dark:border-red-900 bg-red-50/50 dark:bg-red-950/20" : "border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-700"}`}>
                <div className="flex flex-col items-center">
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center ${event.result === "failure" ? "bg-red-100 dark:bg-red-900/40" : "bg-green-100 dark:bg-green-900/40"}`}>
                    <Icon className={`w-4 h-4 ${event.result === "failure" ? "text-red-500" : "text-green-500"}`} />
                  </div>
                  {idx < filtered.length - 1 && <div className="w-0.5 h-4 bg-gray-200 dark:bg-gray-700" />}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-900 dark:text-white">{event.description}</span>
                    {event.result === "failure" && <AlertTriangle className="w-3.5 h-3.5 text-red-500 flex-shrink-0" />}
                  </div>
                  <div className="flex items-center gap-3 mt-0.5 text-xs text-gray-400">
                    <span>{event.ip}</span>
                    <span>&middot;</span>
                    <span>{event.device}</span>
                  </div>
                </div>
                <div className="text-right flex-shrink-0">
                  <p className="text-xs text-gray-500">{new Date(event.timestamp).toLocaleDateString()}</p>
                  <p className="text-xs text-gray-400">{new Date(event.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</p>
                </div>
              </button>
              {isExpanded && event.details && (
                <div className="ml-12 mt-1 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-800">
                  <p className="text-xs font-semibold text-gray-500 mb-2">Additional Details</p>
                  <div className="grid grid-cols-2 gap-2">
                    {Object.entries(event.details).map(([k, v]: any[]) => (
                      <div key={k}>
                        <span className="text-xs text-gray-400">{k}:</span>
                        <span className="text-xs text-gray-700 dark:text-gray-300 ml-1 font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
      {filtered.length === 0 && !loading && (
        <div className="text-center py-12 text-gray-400 text-sm">No events match your filters.</div>
      )}
    </div>
  );
}
