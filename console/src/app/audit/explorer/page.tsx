"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Search, Filter, Download, ChevronRight, ChevronDown, Loader2,
  FileJson, FileText, Calendar, Eye, Activity, X,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

type TabId = "events" | "details" | "export";

interface AuditEvent {
  id: string; timestamp: string; type: string; severity: "info" | "warning" | "error" | "critical";
  user: string; action: string; resource: string; status: string;
  ip_address: string; user_agent: string; payload: Record<string, unknown>;
  metadata: Record<string, unknown>; correlation_id: string;
}

const SEVERITY_COLORS: Record<string, string> = {
  info: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  warning: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
  error: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
  critical: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
};

const EVENT_TYPES = ["auth.login", "auth.logout", "auth.token.refresh", "auth.mfa.verify", "user.create", "user.update", "user.delete", "role.assign", "policy.change", "api.access"];

export default function AuditExplorerPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("events");
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [filterType, setFilterType] = useState("all");
  const [filterSeverity, setFilterSeverity] = useState("all");
  const [filterRange, setFilterRange] = useState("24h");
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (filterType !== "all") params.set("type", filterType);
      if (filterSeverity !== "all") params.set("severity", filterSeverity);
      params.set("range", filterRange);
      if (search) params.set("q", search);
      const res = await fetch(`${API_BASE}/api/v1/audit/events?${params}`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setEvents(d.events || d || []); return; }
    } catch { /* mock */ }
    setEvents([
      { id: "e1", timestamp: "2025-07-18T09:35:00Z", type: "auth.login", severity: "info", user: "alice@company.com", action: "login", resource: "/api/v1/auth/login", status: "success", ip_address: "192.168.1.100", user_agent: "Chrome/125", payload: { method: "passkey", mfa_used: true }, metadata: { session_id: "s123" }, correlation_id: "c-001" },
      { id: "e2", timestamp: "2025-07-18T09:32:00Z", type: "auth.mfa.verify", severity: "warning", user: "bob@company.com", action: "mfa_verify", resource: "/api/v1/auth/mfa/verify", status: "failed", ip_address: "10.0.1.5", user_agent: "Firefox/120", payload: { method: "totp", attempts: 2 }, metadata: {}, correlation_id: "c-002" },
      { id: "e3", timestamp: "2025-07-18T09:28:00Z", type: "user.create", severity: "info", user: "admin@company.com", action: "create", resource: "user:carol@company.com", status: "success", ip_address: "192.168.1.50", user_agent: "Chrome/125", payload: { email: "carol@company.com", role: "engineer" }, metadata: { source: "bulk_import" }, correlation_id: "c-003" },
      { id: "e4", timestamp: "2025-07-18T09:20:00Z", type: "role.assign", severity: "critical", user: "admin@company.com", action: "assign", resource: "role:superadmin → user:dave", status: "success", ip_address: "192.168.1.50", user_agent: "Chrome/125", payload: { role: "superadmin", target_user: "dave@company.com" }, metadata: { approved_by: "cto@company.com" }, correlation_id: "c-004" },
      { id: "e5", timestamp: "2025-07-18T09:15:00Z", type: "policy.change", severity: "error", user: "system", action: "update", resource: "policy:password-strength", status: "failed", ip_address: "127.0.0.1", user_agent: "ggid-cli/1.0", payload: { old_min: 8, new_min: 12 }, metadata: { error: "validation_failed" }, correlation_id: "c-005" },
    ]);
  }, [filterType, filterSeverity, filterRange, search]);

  useEffect(() => { load(); }, [load]);

  const toggleRow = (id: string) => {
    const next = new Set(expandedRows);
    if (next.has(id)) next.delete(id); else next.add(id);
    setExpandedRows(next);
  };

  const tabs: { id: TabId; label: string; icon: typeof Search }[] = [
    { id: "events", label: t("auditExplorer.tabs.events"), icon: Search },
    { id: "details", label: t("auditExplorer.tabs.details"), icon: Eye },
    { id: "export", label: t("auditExplorer.tabs.export"), icon: Download },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Activity className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("auditExplorer.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("auditExplorer.description")}</p>
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

        {tab === "events" && (
          <div className="space-y-4">
            {/* Filters */}
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
              <div className="flex flex-wrap items-center gap-3">
                <div className="relative flex-1 min-w-[200px]">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input type="text" value={search} onChange={(e) => setSearch(e.target.value)}
                    placeholder={t("auditExplorer.events.search")}
                    className="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
                </div>
                <select value={filterType} onChange={(e) => setFilterType(e.target.value)}
                  className="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
                  <option value="all">{t("auditExplorer.events.allTypes")}</option>
                  {EVENT_TYPES.map((tp: any) => <option key={tp} value={tp}>{tp}</option>)}
                </select>
                <select value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value)}
                  className="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
                  <option value="all">{t("auditExplorer.events.allSeverities")}</option>
                  {["info", "warning", "error", "critical"].map((s: any) => <option key={s} value={s}>{t(`auditExplorer.events.severity${s.replace(/^./, (m: any) => m.toUpperCase())}`)}</option>)}
                </select>
                <select value={filterRange} onChange={(e) => setFilterRange(e.target.value)}
                  className="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
                  <option value="1h">{t("auditExplorer.events.range1h")}</option>
                  <option value="24h">{t("auditExplorer.events.range24h")}</option>
                  <option value="7d">{t("auditExplorer.events.range7d")}</option>
                  <option value="30d">{t("auditExplorer.events.range30d")}</option>
                </select>
              </div>
            </div>

            {/* Events Table */}
            {loading ? (
              <Spinner />
            ) : events.length === 0 ? (
              <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
                <Search className="w-12 h-12 mx-auto mb-3 text-gray-300" />
                <p className="text-sm text-gray-500">{t("auditExplorer.events.noEvents")}</p>
              </div>
            ) : (
              <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
                        <th className="py-2 px-3 w-8"></th>
                        <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("auditExplorer.events.timestamp")}</th>
                        <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("auditExplorer.events.type")}</th>
                        <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("auditExplorer.events.severity")}</th>
                        <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("auditExplorer.events.user")}</th>
                        <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("auditExplorer.events.action")}</th>
                        <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("auditExplorer.events.status")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {events.map((e: any) => (
                        <>
                          <tr key={e.id} className="border-b border-gray-100 dark:border-gray-800/50 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/30"
                            onClick={() => toggleRow(e.id)}>
                            <td className="py-3 px-3">
                              {expandedRows.has(e.id) ? <ChevronDown className="w-4 h-4 text-gray-400" /> : <ChevronRight className="w-4 h-4 text-gray-400" />}
                            </td>
                            <td className="py-3 px-3 text-xs text-gray-500">{new Date(e.timestamp).toLocaleString()}</td>
                            <td className="py-3 px-3"><code className="text-xs text-gray-900 dark:text-white">{e.type}</code></td>
                            <td className="py-3 px-3">
                              <span className={`px-2 py-0.5 text-xs rounded-full ${SEVERITY_COLORS[e.severity]}`}>
                                {t(`auditExplorer.events.severity${e.severity.replace(/^./, (m: any) => m.toUpperCase())}`)}
                              </span>
                            </td>
                            <td className="py-3 px-3 text-gray-900 dark:text-white">{e.user}</td>
                            <td className="py-3 px-3 text-gray-600 dark:text-gray-400">{e.action}</td>
                            <td className="py-3 px-3 text-xs">
                              <span className={e.status === "success" ? "text-green-600" : "text-red-600"}>{e.status}</span>
                            </td>
                          </tr>
                          {expandedRows.has(e.id) && (
                            <tr className="bg-gray-50 dark:bg-gray-800/30">
                              <td></td>
                              <td colSpan={6} className="py-4 px-3">
                                <div className="space-y-2">
                                  <DetailRow label={t("auditExplorer.details.resource")} value={e.resource} />
                                  <DetailRow label={t("auditExplorer.details.ipAddress")} value={e.ip_address} />
                                  <DetailRow label={t("auditExplorer.details.correlationId")} value={e.correlation_id} />
                                  <div>
                                    <span className="text-xs font-medium text-gray-500">{t("auditExplorer.details.payload")}:</span>
                                    <pre className="mt-1 p-3 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-x-auto text-gray-700 dark:text-gray-300">
                                      {JSON.stringify(e.payload, null, 2)}
                                    </pre>
                                  </div>
                                </div>
                                <button onClick={() => { setSelectedEvent(e); setTab("details"); }}
                                  className="mt-2 flex items-center gap-1 text-xs text-blue-600 hover:underline">
                                  <Eye className="w-3 h-3" />
                                  {t("auditExplorer.tabs.details")}
                                </button>
                              </td>
                            </tr>
                          )}
                        </>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        {tab === "details" && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
            {selectedEvent ? (
              <div className="space-y-4">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("auditExplorer.details.title")}</h3>
                  <button onClick={() => setSelectedEvent(null)} className="text-gray-400 hover:text-gray-600"><X className="w-4 h-4" /></button>
                </div>
                <DetailRow label={t("auditExplorer.details.eventId")} value={selectedEvent.id} />
                <DetailRow label={t("auditExplorer.details.timestamp")} value={new Date(selectedEvent.timestamp).toLocaleString()} />
                <DetailRow label={t("auditExplorer.details.type")} value={selectedEvent.type} />
                <DetailRow label={t("auditExplorer.details.severity")} value={selectedEvent.severity} />
                <DetailRow label={t("auditExplorer.details.user")} value={selectedEvent.user} />
                <DetailRow label={t("auditExplorer.details.ipAddress")} value={selectedEvent.ip_address} />
                <DetailRow label={t("auditExplorer.details.userAgent")} value={selectedEvent.user_agent} />
                <DetailRow label={t("auditExplorer.details.resource")} value={selectedEvent.resource} />
                <DetailRow label={t("auditExplorer.details.action")} value={selectedEvent.action} />
                <DetailRow label={t("auditExplorer.details.result")} value={selectedEvent.status} />
                <DetailRow label={t("auditExplorer.details.correlationId")} value={selectedEvent.correlation_id} />
                <div>
                  <span className="text-xs font-medium text-gray-500">{t("auditExplorer.details.payload")}:</span>
                  <pre className="mt-1 p-3 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-x-auto text-gray-700 dark:text-gray-300">
                    {JSON.stringify(selectedEvent.payload, null, 2)}
                  </pre>
                </div>
                <div>
                  <span className="text-xs font-medium text-gray-500">{t("auditExplorer.details.metadata")}:</span>
                  <pre className="mt-1 p-3 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-x-auto text-gray-700 dark:text-gray-300">
                    {JSON.stringify(selectedEvent.metadata, null, 2)}
                  </pre>
                </div>
              </div>
            ) : (
              <div className="text-center py-12">
                <Eye className="w-12 h-12 mx-auto mb-3 text-gray-300" />
                <p className="text-sm text-gray-500">{t("auditExplorer.details.selectEvent")}</p>
                <button onClick={() => setTab("events")} className="mt-3 text-sm text-blue-600 hover:underline">
                  {t("auditExplorer.tabs.events")}
                </button>
              </div>
            )}
          </div>
        )}

        {tab === "export" && (
          <ExportTab events={events} />
        )}
      </div>
    </div>
  );
}

// ============ Export Tab ============

function ExportTab({ events }: { events: AuditEvent[] }) {
  const t = useTranslations();
  const [format, setFormat] = useState<"json" | "csv">("json");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [filterType, setFilterType] = useState("all");
  const [filterSeverity, setFilterSeverity] = useState("all");
  const [exporting, setExporting] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const doExport = () => {
    setExporting(true);
    const data = events.filter((e: any) => {
      if (filterType !== "all" && e.type !== filterType) return false;
      if (filterSeverity !== "all" && e.severity !== filterSeverity) return false;
      return true;
    });

    setTimeout(() => {
      let content: string;
      if (format === "json") {
        content = JSON.stringify(data, null, 2);
      } else {
        const headers = ["id", "timestamp", "type", "severity", "user", "action", "resource", "status", "ip_address"];
        const rows = data.map((e: any) => headers.map((h: any) => `"${String(e[h] || "").replace(/"/g, '""')}"`).join(","));
        content = [headers.join(","), ...rows].join("\n");
      }
      const blob = new Blob([content], { type: format === "json" ? "application/json" : "text/csv" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `audit-export-${Date.now()}.${format}`;
      a.click();
      setExporting(false);
      setMsg(t("auditExplorer.export.exportSuccess"));
      setTimeout(() => setMsg(null), 3000);
    }, 800);
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("auditExplorer.export.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t("auditExplorer.export.description")}</p>
        <p className="text-xs text-gray-400 mt-1">{t("auditExplorer.export.maxEvents")}</p>
      </div>

      {/* Format */}
      <div>
        <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">{t("auditExplorer.export.format")}</label>
        <div className="flex gap-2">
          <button onClick={() => setFormat("json")}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg border-2 text-sm transition-all ${format === "json" ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700" : "border-gray-200 dark:border-gray-700"}`}>
            <FileJson className="w-4 h-4" /> {t("auditExplorer.export.formatJson")}
          </button>
          <button onClick={() => setFormat("csv")}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg border-2 text-sm transition-all ${format === "csv" ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700" : "border-gray-200 dark:border-gray-700"}`}>
            <FileText className="w-4 h-4" /> {t("auditExplorer.export.formatCsv")}
          </button>
        </div>
      </div>

      {/* Date Range */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("auditExplorer.export.startDate")}</label>
          <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("auditExplorer.export.endDate")}</label>
          <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
      </div>

      {/* Filters */}
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("auditExplorer.export.filterType")}</label>
          <select value={filterType} onChange={(e) => setFilterType(e.target.value)}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
            <option value="all">{t("auditExplorer.events.allTypes")}</option>
            {EVENT_TYPES.map((tp: any) => <option key={tp} value={tp}>{tp}</option>)}
          </select>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("auditExplorer.export.filterSeverity")}</label>
          <select value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value)}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
            <option value="all">{t("auditExplorer.events.allSeverities")}</option>
            {["info", "warning", "error", "critical"].map((s: any) => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
      </div>

      {/* Estimated count */}
      <div className="flex items-center gap-2 text-sm">
        <Calendar className="w-4 h-4 text-gray-400" />
        <span className="text-gray-500">{t("auditExplorer.export.estimatedEvents")}:</span>
        <span className="font-medium text-gray-900 dark:text-white">{events.length}</span>
      </div>

      {msg && (
        <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
          <FileJson className="w-4 h-4" />{msg}
        </div>
      )}

      <button onClick={doExport} disabled={exporting || events.length === 0}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {exporting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
        {exporting ? t("auditExplorer.export.exporting") : t("auditExplorer.export.export")}
      </button>
    </div>
  );
}

// ============ Shared ============

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-1 border-b border-gray-100 dark:border-gray-800/50">
      <span className="text-xs text-gray-500 dark:text-gray-400">{label}</span>
      <span className="text-sm text-gray-900 dark:text-white font-mono">{value}</span>
    </div>
  );
}
