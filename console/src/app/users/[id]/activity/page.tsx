"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { Search, Download, Filter, LogIn, LogOut, Shield, Key, UserCheck, FileEdit, AlertTriangle, ChevronLeft } from "lucide-react";

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

const MOCK_EVENTS: ActivityEvent[] = [
  { id: "1", timestamp: "2024-07-11T10:30:00Z", eventType: "login", description: "Successful login from Chrome on macOS", ip: "192.168.1.100", device: "Chrome / macOS", result: "success", details: { session_id: "ses_abc", mfa_used: "totp" } },
  { id: "2", timestamp: "2024-07-11T09:15:00Z", eventType: "mfa_challenge", description: "TOTP verification challenge", ip: "192.168.1.100", device: "Chrome / macOS", result: "success" },
  { id: "3", timestamp: "2024-07-10T18:45:00Z", eventType: "logout", description: "Session terminated by user", ip: "192.168.1.100", device: "Chrome / macOS", result: "success" },
  { id: "4", timestamp: "2024-07-10T16:20:00Z", eventType: "api_key_used", description: "API key 'Production CI/CD' used for GET /api/v1/users", ip: "10.0.0.5", device: "curl / Linux", result: "success", details: { key_prefix: "ggid_pk_abc", endpoint: "/api/v1/users" } },
  { id: "5", timestamp: "2024-07-10T14:05:00Z", eventType: "role_assigned", description: "Role 'Editor' assigned by admin@example.com", ip: "192.168.1.50", device: "Firefox / Windows", result: "success", details: { assigned_by: "admin@example.com", role: "Editor" } },
  { id: "6", timestamp: "2024-07-10T11:30:00Z", eventType: "login", description: "Failed login attempt - wrong password", ip: "45.227.89.12", device: "Unknown / Unknown", result: "failure", details: { reason: "invalid_password" } },
  { id: "7", timestamp: "2024-07-10T10:00:00Z", eventType: "token_refresh", description: "Access token refreshed", ip: "192.168.1.100", device: "Chrome / macOS", result: "success" },
  { id: "8", timestamp: "2024-07-09T20:15:00Z", eventType: "password_change", description: "Password changed successfully", ip: "192.168.1.100", device: "Chrome / macOS", result: "success" },
  { id: "9", timestamp: "2024-07-09T15:30:00Z", eventType: "mfa_challenge", description: "Failed MFA - incorrect TOTP code (3rd attempt)", ip: "45.227.89.12", device: "Unknown / Unknown", result: "failure", details: { attempts: 3, method: "totp" } },
  { id: "10", timestamp: "2024-07-09T09:00:00Z", eventType: "login", description: "Successful login from Safari on iPhone", ip: "10.0.1.25", device: "Safari / iOS", result: "success" },
];

const getEventIcon = (type: string) => {
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
  const [filter, setFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const filtered = MOCK_EVENTS.filter(e => {
    if (filter !== "all" && e.eventType !== filter) return false;
    if (search && !e.description.toLowerCase().includes(search.toLowerCase()) && !e.ip.includes(search)) return false;
    return true;
  });

  const exportCsv = () => {
    const headers = "Timestamp,Event Type,Description,IP,Device,Result\n";
    const rows = filtered.map(e => `"${e.timestamp}","${e.eventType}","${e.description}","${e.ip}","${e.device}","${e.result}"`).join("\n");
    const blob = new Blob([headers + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = `user-${params.id}-activity.csv`; a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <a href="/users" className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition">
          <ChevronLeft className="w-5 h-5 text-gray-500" />
        </a>
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">User Activity Timeline</h1>
          <p className="text-sm text-gray-500 mt-1">User ID: <code className="text-xs bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">{params.id}</code></p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-4 gap-3">
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Total Events</p>
          <p className="text-xl font-bold text-gray-900 dark:text-white">{MOCK_EVENTS.length}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Logins</p>
          <p className="text-xl font-bold text-green-500">{MOCK_EVENTS.filter(e => e.eventType === "login" && e.result === "success").length}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Failed Attempts</p>
          <p className="text-xl font-bold text-red-500">{MOCK_EVENTS.filter(e => e.result === "failure").length}</p>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-3">
          <p className="text-xs text-gray-400">Unique IPs</p>
          <p className="text-xl font-bold text-indigo-500">{new Set(MOCK_EVENTS.map(e => e.ip)).size}</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="flex-1 relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search by description or IP..."
            className="w-full pl-9 pr-3 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white" />
        </div>
        <select value={filter} onChange={(e) => setFilter(e.target.value)}
          className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-lg bg-transparent text-gray-900 dark:text-white">
          {EVENT_TYPES.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
        </select>
        <button onClick={exportCsv} className="flex items-center gap-1.5 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition">
          <Download className="w-4 h-4" /> CSV
        </button>
      </div>

      {/* Timeline */}
      <div className="space-y-2">
        {filtered.map((event, idx) => {
          const Icon = getEventIcon(event.eventType);
          const isExpanded = expandedId === event.id;
          return (
            <div key={event.id}>
              <button onClick={() => setExpandedId(isExpanded ? null : event.id)}
                className={`w-full text-left flex items-center gap-4 p-3 rounded-xl border transition ${event.result === "failure" ? "border-red-200 dark:border-red-900 bg-red-50/50 dark:bg-red-950/20" : "border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 hover:border-gray-300 dark:hover:border-gray-700"}`}>
                {/* Timeline dot */}
                <div className="flex flex-col items-center">
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center ${event.result === "failure" ? "bg-red-100 dark:bg-red-900/40" : "bg-green-100 dark:bg-green-900/40"}`}>
                    <Icon className={`w-4 h-4 ${event.result === "failure" ? "text-red-500" : "text-green-500"}`} />
                  </div>
                  {idx < filtered.length - 1 && <div className="w-0.5 h-4 bg-gray-200 dark:bg-gray-700" />}
                </div>
                {/* Content */}
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
                {/* Timestamp */}
                <div className="text-right flex-shrink-0">
                  <p className="text-xs text-gray-500">{new Date(event.timestamp).toLocaleDateString()}</p>
                  <p className="text-xs text-gray-400">{new Date(event.timestamp).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</p>
                </div>
              </button>
              {/* Expanded details */}
              {isExpanded && event.details && (
                <div className="ml-12 mt-1 p-3 bg-gray-50 dark:bg-gray-800/50 rounded-lg border border-gray-200 dark:border-gray-800">
                  <p className="text-xs font-semibold text-gray-500 mb-2">Additional Details</p>
                  <div className="grid grid-cols-2 gap-2">
                    {Object.entries(event.details).map(([k, v]) => (
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
      {filtered.length === 0 && (
        <div className="text-center py-12 text-gray-400 text-sm">No events match your filters.</div>
      )}
    </div>
  );
}
