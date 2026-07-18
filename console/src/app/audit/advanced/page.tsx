"use client";

import { Fragment, useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Search,
  Save,
  Trash2,
  Plus,
  ChevronDown,
  ChevronRight,
  RefreshCw,
  AlertTriangle,
  Activity,
  Globe,
  ShieldAlert,
  Calendar,
  Mail,
  X,
  Clock,
  Download,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

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
  service?: string;
  severity?: string;
  metadata?: Record<string, unknown>;
}

// ===== Query Builder Types =====

type FieldType = "user" | "action" | "ip" | "service" | "severity" | "result" | "timestamp";
type Operator = "equals" | "contains" | "starts_with" | "in" | "between";

interface Condition {
  id: string;
  field: FieldType;
  operator: Operator;
  value: string;
  value2?: string; // for "between" operator
}

interface ConditionGroup {
  id: string;
  logic: "AND" | "OR";
  conditions: Condition[];
  groups: ConditionGroup[];
}

interface SavedSearch {
  id: string;
  name: string;
  group: ConditionGroup;
  createdAt: string;
}

interface ScheduledReport {
  id: string;
  name: string;
  frequency: "daily" | "weekly";
  recipients: string;
  time: string;
  enabled: boolean;
  group: ConditionGroup;
}

const FIELD_LABELS: Record<FieldType, string> = {
  user: "User",
  action: "Action",
  ip: "IP Address",
  service: "Service",
  severity: "Severity",
  result: "Result",
  timestamp: "Timestamp",
};

const OPERATOR_LABELS: Record<Operator, string> = {
  equals: "equals",
  contains: "contains",
  starts_with: "starts with",
  in: "in (comma-sep)",
  between: "between",
};

const SERVICES = ["gateway", "auth", "identity", "policy", "org", "audit", "oauth"];
const SEVERITIES = ["info", "warning", "critical", "error"];
const RESULTS = ["success", "failure", "denied"];

function genId() {
  const t = useTranslations();

  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function emptyGroup(logic: "AND" | "OR" = "AND"): ConditionGroup {
  return { id: genId(), logic, conditions: [], groups: [] };
}

function emptyCondition(): Condition {
  return { id: genId(), field: "user", operator: "equals", value: "" };
}

function cloneGroup(g: ConditionGroup): ConditionGroup {
  return {
    ...g,
    id: genId(),
    conditions: g.conditions.map((c: any) => ({ ...c, id: genId() })),
    groups: g.groups.map((sub: any) => cloneGroup(sub)),
  };
}

// ===== Main Component =====

export default function AuditAdvancedPage() {
  const { apiFetch, API_BASE, TENANT_ID } = useApi();
  const [rootGroup, setRootGroup] = useState<ConditionGroup>(() => ({
    ...emptyGroup(),
    conditions: [emptyCondition()],
  }));
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  // Saved searches
  const [savedSearches, setSavedSearches] = useState<SavedSearch[]>([]);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [newSearchName, setNewSearchName] = useState("");

  // Scheduled reports
  const [scheduledReports, setScheduledReports] = useState<ScheduledReport[]>([]);
  const [showScheduleDialog, setShowScheduleDialog] = useState(false);
  const [newReport, setNewReport] = useState({
    name: "",
    frequency: "daily" as "daily" | "weekly",
    recipients: "",
    time: "09:00",
  });

  // Anomaly summary
  const [failedLogins24h, setFailedLogins24h] = useState(0);
  const [uniqueIps24h, setUniqueIps24h] = useState(0);
  const [anomaliesDetected, setAnomaliesDetected] = useState(0);

  // Load saved searches from localStorage
  useEffect(() => {
    const stored = localStorage.getItem("ggid_saved_searches");
    if (stored) {
      try { setSavedSearches(JSON.parse(stored)); } catch { /* noop */ }
    }
    const reports = localStorage.getItem("ggid_scheduled_reports");
    if (reports) {
      try { setScheduledReports(JSON.parse(reports)); } catch { /* noop */ }
    }
  }, []);

  useEffect(() => {
    localStorage.setItem("ggid_saved_searches", JSON.stringify(savedSearches));
  }, [savedSearches]);

  useEffect(() => {
    localStorage.setItem("ggid_scheduled_reports", JSON.stringify(scheduledReports));
  }, [scheduledReports]);

  // ===== Group/Condition manipulation =====

  const addCondition = (groupId: string) => {
    const updater = (g: ConditionGroup): ConditionGroup => {
      if (g.id === groupId) return { ...g, conditions: [...g.conditions, emptyCondition()] };
      return { ...g, groups: g.groups.map(updater) };
    };
    setRootGroup((prev) => updater(prev));
  };

  const addGroup = (parentId: string) => {
    const updater = (g: ConditionGroup): ConditionGroup => {
      if (g.id === parentId) return { ...g, groups: [...g.groups, emptyGroup("AND")] };
      return { ...g, groups: g.groups.map(updater) };
    };
    setRootGroup((prev) => updater(prev));
  };

  const removeCondition = (groupId: string, condId: string) => {
    const updater = (g: ConditionGroup): ConditionGroup => {
      if (g.id === groupId) return { ...g, conditions: g.conditions.filter((c: any) => c.id !== condId) };
      return { ...g, groups: g.groups.map(updater) };
    };
    setRootGroup((prev) => updater(prev));
  };

  const removeGroup = (parentId: string, groupId: string) => {
    const updater = (g: ConditionGroup): ConditionGroup => {
      if (g.id === parentId) return { ...g, groups: g.groups.filter((sub: any) => sub.id !== groupId) };
      return { ...g, groups: g.groups.map(updater) };
    };
    setRootGroup((prev) => updater(prev));
  };

  const updateCondition = (groupId: string, condId: string, patch: Partial<Condition>) => {
    const updater = (g: ConditionGroup): ConditionGroup => {
      if (g.id === groupId)
        return { ...g, conditions: g.conditions.map((c: any) => (c.id === condId ? { ...c, ...patch } : c)) };
      return { ...g, groups: g.groups.map(updater) };
    };
    setRootGroup((prev) => updater(prev));
  };

  const toggleGroupLogic = (groupId: string) => {
    const updater = (g: ConditionGroup): ConditionGroup => {
      if (g.id === groupId) return { ...g, logic: g.logic === "AND" ? "OR" : "AND" };
      return { ...g, groups: g.groups.map(updater) };
    };
    setRootGroup((prev) => updater(prev));
  };

  // ===== Query evaluation =====

  const matchesCondition = (event: AuditEvent, cond: Condition): boolean => {
    let eventVal = "";
    switch (cond.field) {
      case "user": eventVal = event.actor_name || event.actor_id || ""; break;
      case "action": eventVal = event.action || ""; break;
      case "ip": eventVal = event.ip_address || ""; break;
      case "service": eventVal = event.service || event.action?.split(".")[0] || ""; break;
      case "severity": eventVal = event.severity || "info"; break;
      case "result": eventVal = event.result || ""; break;
      case "timestamp": eventVal = event.created_at || ""; break;
    }

    switch (cond.operator) {
      case "equals":
        return cond.field === "timestamp"
          ? eventVal.startsWith(cond.value)
          : eventVal.toLowerCase() === cond.value.toLowerCase();
      case "contains":
        return eventVal.toLowerCase().includes(cond.value.toLowerCase());
      case "starts_with":
        return eventVal.toLowerCase().startsWith(cond.value.toLowerCase());
      case "in":
        return cond.value.split(",").map((v: any) => v.trim().toLowerCase()).includes(eventVal.toLowerCase());
      case "between":
        if (cond.field === "timestamp") {
          const t = new Date(eventVal).getTime();
          const from = new Date(cond.value).getTime();
          const to = new Date(cond.value2 || cond.value).getTime();
          return t >= from && t <= to;
        }
        return eventVal >= cond.value && eventVal <= (cond.value2 || cond.value);
      default:
        return true;
    }
  };

  const matchesGroup = (event: AuditEvent, group: ConditionGroup): boolean => {
    const condResults = group.conditions.filter((c: any) => c.value).map((c: any) => matchesCondition(event, c));
    const subResults = group.groups.map((g: any) => matchesGroup(event, g));
    const all = [...condResults, ...subResults];
    if (all.length === 0) return true;
    return group.logic === "AND" ? all.every(Boolean) : all.some(Boolean);
  };

  // ===== Anomaly detection =====

  const getAnomalyType = (event: AuditEvent, allEvents: AuditEvent[]): "failed_spike" | "unusual_ip" | "new_device" | null => {
    // Failed login spike
    if (event.action === "user.login" && event.result !== "success") {
      const fails = allEvents.filter(
        (e) => e.actor_id === event.actor_id && e.action === "user.login" && e.result !== "success",
      ).length;
      if (fails >= 3) return "failed_spike";
    }
    // Unusual IP (first occurrence from this IP for this user)
    if (event.ip_address) {
      const userEventsFromIP = allEvents.filter(
        (e) => e.actor_id === event.actor_id && e.ip_address === event.ip_address,
      );
      if (userEventsFromIP.length === 1 && allEvents.filter((e: any) => e.ip_address === event.ip_address).length <= 2) {
        return "unusual_ip";
      }
    }
    // New device (first occurrence of user-agent for this user)
    if (event.user_agent) {
      const userAgents = allEvents.filter(
        (e) => e.actor_id === event.actor_id && e.user_agent === event.user_agent,
      );
      if (userAgents.length === 1) return "new_device";
    }
    return null;
  };

  const anomalyRowClass = (event: AuditEvent): string => {
    const a = getAnomalyType(event, events);
    switch (a) {
      case "failed_spike": return "bg-red-50 dark:bg-red-950/20";
      case "unusual_ip": return "bg-orange-50 dark:bg-orange-950/20";
      case "new_device": return "bg-blue-50 dark:bg-blue-950/20";
      default: return "";
    }
  };

  // ===== Data loading =====

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      params.set("page_size", "100");
      const data = await apiFetch<{ events?: AuditEvent[]; total?: number; total_count?: number }>(
        `/api/v1/audit/events?${params}`,
      );
      let fetched = data.events || [];
      // Apply client-side query filtering
      fetched = fetched.filter((e: any) => matchesGroup(e, rootGroup));
      setEvents(fetched);

      // Compute anomaly summary stats
      const now = Date.now();
      const dayAgo = now - 24 * 60 * 60 * 1000;
      const recentEvents = fetched.filter((e: any) => new Date(e.created_at).getTime() > dayAgo);
      setFailedLogins24h(
        recentEvents.filter((e: any) => e.action === "user.login" && e.result !== "success").length,
      );
      setUniqueIps24h(new Set(recentEvents.map((e: any) => e.ip_address).filter(Boolean)).size);
      setAnomaliesDetected(
        recentEvents.filter((e: any) => getAnomalyType(e, recentEvents) !== null).length,
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [apiFetch, rootGroup]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // ===== Saved searches =====

  const saveSearch = () => {
    if (!newSearchName.trim()) return;
    const search: SavedSearch = {
      id: genId(),
      name: newSearchName,
      group: cloneGroup(rootGroup),
      createdAt: new Date().toISOString(),
    };
    setSavedSearches((prev) => [...prev, search]);
    setShowSaveDialog(false);
    setNewSearchName("");
  };

  const loadSearch = (search: SavedSearch) => {
    setRootGroup(cloneGroup(search.group));
  };

  const deleteSearch = (id: string) => {
    setSavedSearches((prev) => prev.filter((s: any) => s.id !== id));
  };

  // ===== Scheduled reports =====

  const addScheduledReport = () => {
    if (!newReport.name.trim() || !newReport.recipients.trim()) return;
    const report: ScheduledReport = {
      id: genId(),
      name: newReport.name,
      frequency: newReport.frequency,
      recipients: newReport.recipients,
      time: newReport.time,
      enabled: true,
      group: cloneGroup(rootGroup),
    };
    setScheduledReports((prev) => [...prev, report]);
    setShowScheduleDialog(false);
    setNewReport({ name: "", frequency: "daily", recipients: "", time: "09:00" });
  };

  const toggleReport = (id: string) => {
    setScheduledReports((prev) =>
      prev.map((r: any) => (r.id === id ? { ...r, enabled: !r.enabled } : r)),
    );
  };

  const deleteReport = (id: string) => {
    setScheduledReports((prev) => prev.filter((r: any) => r.id !== id));
  };

  const handleExport = (format: "csv" | "json") => {
    const params = new URLSearchParams({ tenant_id: TENANT_ID, format });
    window.open(`${API_BASE}/api/v1/audit/export?${params}`, "_blank");
  };

  const toggleExpand = (id: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const resultBadge = (result: string) => {
    switch (result) {
      case "success": return "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400";
      case "failure": return "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400";
      case "denied": return "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400";
      default: return "bg-gray-100 text-gray-600";
    }
  };

  const severityBadge = (sev: string | undefined) => {
    switch (sev) {
      case "critical": return "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400";
      case "error": return "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-400";
      case "warning": return "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400";
      default: return "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400";
    }
  };

  // ===== Render condition row =====

  const renderCondition = (cond: Condition, groupId: string) => (
    <div key={cond.id} className="flex flex-wrap items-center gap-2">
      <select
        value={cond.field}
        onChange={(e) => updateCondition(groupId, cond.id, { field: e.target.value as FieldType })}
        className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
      >
        {Object.entries(FIELD_LABELS).map(([val, label]: any[]) => (
          <option key={val} value={val}>{label}</option>
        ))}
      </select>
      <select
        value={cond.operator}
        onChange={(e) => updateCondition(groupId, cond.id, { operator: e.target.value as Operator })}
        className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
      >
        {Object.entries(OPERATOR_LABELS).map(([val, label]: any[]) => (
          <option key={val} value={val}>{label}</option>
        ))}
      </select>
      {cond.field === "timestamp" ? (
        <>
          <input
            type="date"
            value={cond.value}
            onChange={(e) => updateCondition(groupId, cond.id, { value: e.target.value })}
            className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
          />
          {cond.operator === "between" && (
            <input
              type="date"
              value={cond.value2 || ""}
              onChange={(e) => updateCondition(groupId, cond.id, { value2: e.target.value })}
              className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
            />
          )}
        </>
      ) : cond.field === "service" ? (
        <select
          value={cond.value}
          onChange={(e) => updateCondition(groupId, cond.id, { value: e.target.value })}
          className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          <option value="">-- Select --</option>
          {SERVICES.map((s: any) => <option key={s} value={s}>{s}</option>)}
        </select>
      ) : cond.field === "severity" ? (
        <select
          value={cond.value}
          onChange={(e) => updateCondition(groupId, cond.id, { value: e.target.value })}
          className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          <option value="">-- Select --</option>
          {SEVERITIES.map((s: any) => <option key={s} value={s}>{s}</option>)}
        </select>
      ) : cond.field === "result" ? (
        <select
          value={cond.value}
          onChange={(e) => updateCondition(groupId, cond.id, { value: e.target.value })}
          className="rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        >
          <option value="">-- Select --</option>
          {RESULTS.map((r: any) => <option key={r} value={r}>{r}</option>)}
        </select>
      ) : (
        <input
          type="text"
          value={cond.value}
          onChange={(e) => updateCondition(groupId, cond.id, { value: e.target.value })}
          placeholder="Enter value..."
          className="min-w-[120px] flex-1 rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
        />
      )}
      <button
        onClick={() => removeCondition(groupId, cond.id)}
        className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-950"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );

  // ===== Render group recursively =====

  const renderGroup = (group: ConditionGroup, depth = 0): React.ReactNode => {
    const isRoot = depth === 0;
    return (
      <div
        key={group.id}
        className={`rounded-lg border ${isRoot ? "border-gray-300 bg-white p-4 dark:border-gray-600 dark:bg-gray-800" : "border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900/50"}`}
        style={{ marginLeft: depth > 0 ? 12 : 0 }}
      >
        {/* Group header */}
        <div className="mb-3 flex items-center gap-2">
          {!isRoot && (
            <button
              onClick={() => isRoot ? null : removeGroup(rootGroup.id, group.id)}
              className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-950"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
          <button
            onClick={() => toggleGroupLogic(group.id)}
            className={`rounded-md px-2 py-1 text-xs font-bold ${
              group.logic === "AND"
                ? "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400"
                : "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-400"
            }`}
          >
            {group.logic}
          </button>
          <span className="text-xs text-gray-400">Match {group.logic === "AND" ? "all" : "any"} of the following</span>
        </div>

        {/* Conditions */}
        <div className="space-y-2">
          {group.conditions.map((cond: any) => renderCondition(cond, group.id))}
          {group.groups.map((sub: any) => renderGroup(sub, depth + 1))}
        </div>

        {/* Add buttons */}
        <div className="mt-3 flex gap-2">
          <button
            onClick={() => addCondition(group.id)}
            className="flex items-center gap-1 rounded-md border border-gray-300 px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <Plus className="h-3 w-3" /> Condition
          </button>
          <button
            onClick={() => addGroup(group.id)}
            className="flex items-center gap-1 rounded-md border border-gray-300 px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <Plus className="h-3 w-3" /> Group
          </button>
        </div>
      </div>
    );
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold dark:text-gray-100">Advanced Audit Analysis</h1>
        <div className="flex gap-2">
          <button
            onClick={() => handleExport("csv")}
            className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <Download className="h-4 w-4" /> CSV
          </button>
          <button
            onClick={loadData}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            <RefreshCw className="h-4 w-4" /> Run Query
          </button>
        </div>
      </div>

      {/* Anomaly Summary Cards */}
      <div className="mb-6 grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-red-200 bg-red-50 p-5 dark:border-red-800 dark:bg-red-950/30">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-red-100 dark:bg-red-900/50">
              <AlertTriangle className="h-5 w-5 text-red-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-red-700 dark:text-red-400">{failedLogins24h}</p>
              <p className="text-xs text-red-600 dark:text-red-400">Failed Logins (24h)</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-orange-200 bg-orange-50 p-5 dark:border-orange-800 dark:bg-orange-950/30">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-orange-100 dark:bg-orange-900/50">
              <Globe className="h-5 w-5 text-orange-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-orange-700 dark:text-orange-400">{uniqueIps24h}</p>
              <p className="text-xs text-orange-600 dark:text-orange-400">Unique IPs (24h)</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-blue-200 bg-blue-50 p-5 dark:border-blue-800 dark:bg-blue-950/30">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/50">
              <ShieldAlert className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <p className="text-2xl font-bold text-blue-700 dark:text-blue-400">{anomaliesDetected}</p>
              <p className="text-xs text-blue-600 dark:text-blue-400">Anomalies Detected</p>
            </div>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_250px]">
        {/* ===== Main content ===== */}
        <div className="space-y-6">
          {/* Query Builder */}
          <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-sm font-semibold">
                <Search className="h-4 w-4 text-brand-600" /> Query Builder
              </h2>
              <div className="flex gap-2">
                <button
                  onClick={() => setShowSaveDialog(true)}
                  className="flex items-center gap-1.5 rounded-md border border-gray-300 px-2.5 py-1 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  <Save className="h-3.5 w-3.5" /> Save Search
                </button>
                <button
                  onClick={() => setShowScheduleDialog(true)}
                  className="flex items-center gap-1.5 rounded-md border border-gray-300 px-2.5 py-1 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  <Calendar className="h-3.5 w-3.5" /> Schedule Report
                </button>
              </div>
            </div>
            {renderGroup(rootGroup)}
          </div>

          {/* Error */}
          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-950">
              {error}
            </div>
          )}

          {/* Results Table */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-gray-700">
              <h3 className="text-sm font-semibold">
                Results {events.length > 0 && <span className="text-gray-400">({events.length})</span>}
              </h3>
            </div>
            {loading ? (
              <p className="p-8 text-center text-sm text-gray-500">Loading...</p>
            ) : events.length === 0 ? (
              <div className="p-12 text-center">
                <Activity className="mx-auto mb-4 h-12 w-12 text-gray-300" />
                <p className="text-gray-500">No events match your query</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead className="border-b border-gray-100 bg-gray-50 dark:border-gray-700 dark:bg-gray-900/50">
                    <tr>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Timestamp</th>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">User</th>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Action</th>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">IP</th>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Service</th>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Severity</th>
                      <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Result</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                    {events.map((event: any) => {
                      const anomaly = getAnomalyType(event, events);
                      const isExpanded = expandedRows.has(event.id);
                      return (
                        <Fragment key={event.id}>
                          <tr
                            onClick={() => toggleExpand(event.id)}
                            className={`cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50 ${anomalyRowClass(event)}`}
                          >
                            <td className="whitespace-nowrap px-4 py-3 text-xs text-gray-500">
                              <div className="flex items-center gap-1.5">
                                {isExpanded ? (
                                  <ChevronDown className="h-3.5 w-3.5 shrink-0 text-gray-400" />
                                ) : (
                                  <ChevronRight className="h-3.5 w-3.5 shrink-0 text-gray-400" />
                                )}
                                {event.created_at ? new Date(event.created_at).toLocaleString() : "-"}
                              </div>
                            </td>
                            <td className="px-4 py-3 text-xs">
                              <span className="font-medium">{event.actor_name || event.actor_id?.substring(0, 8) || "system"}</span>
                            </td>
                            <td className="px-4 py-3 text-xs font-mono text-gray-600 dark:text-gray-400">{event.action}</td>
                            <td className="px-4 py-3 text-xs font-mono text-gray-500">{event.ip_address || "-"}</td>
                            <td className="px-4 py-3 text-xs text-gray-500">{event.service || event.action?.split(".")[0] || "-"}</td>
                            <td className="px-4 py-3">
                              <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${severityBadge(event.severity)}`}>
                                {event.severity || "info"}
                              </span>
                            </td>
                            <td className="px-4 py-3">
                              <div className="flex items-center gap-2">
                                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${resultBadge(event.result)}`}>
                                  {event.result}
                                </span>
                                {anomaly && (
                                  <span
                                    className="flex items-center gap-0.5 text-xs"
                                    title={anomaly}
                                  >
                                    <AlertTriangle className={`h-3 w-3 ${
                                      anomaly === "failed_spike" ? "text-red-500" :
                                      anomaly === "unusual_ip" ? "text-orange-500" : "text-blue-500"
                                    }`} />
                                  </span>
                                )}
                              </div>
                            </td>
                          </tr>
                          {isExpanded && (
                            <tr>
                              <td colSpan={7} className="bg-gray-50 px-4 py-3 dark:bg-gray-900/50">
                                <div className="grid gap-3 sm:grid-cols-2">
                                  <div className="space-y-1 text-xs">
                                    <p><span className="font-semibold text-gray-600 dark:text-gray-400">Resource:</span> {event.resource_type} {event.resource_id && `(${event.resource_id.substring(0, 8)})`}</p>
                                    <p><span className="font-semibold text-gray-600 dark:text-gray-400">User Agent:</span> {event.user_agent || "-"}</p>
                                    <p><span className="font-semibold text-gray-600 dark:text-gray-400">Actor Type:</span> {event.actor_type || "user"}</p>
                                  </div>
                                  <div className="space-y-1 text-xs">
                                    <p><span className="font-semibold text-gray-600 dark:text-gray-400">Anomaly:</span> {anomaly || "none"}</p>
                                    <p><span className="font-semibold text-gray-600 dark:text-gray-400">Tenant:</span> <span className="font-mono">{event.tenant_id?.substring(0, 8)}</span></p>
                                  </div>
                                </div>
                                {event.metadata && Object.keys(event.metadata).length > 0 && (
                                  <pre className="mt-3 overflow-x-auto rounded-lg bg-gray-900 p-3 text-xs text-green-400">
                                    {JSON.stringify(event.metadata, null, 2)}
                                  </pre>
                                )}
                              </td>
                            </tr>
                          )}
                        </Fragment>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>

        {/* ===== Sidebar: Saved Searches & Scheduled Reports ===== */}
        <div className="space-y-4 lg:sticky lg:top-4 lg:self-start">
          {/* Saved Searches */}
          <div className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h3 className="mb-3 flex items-center gap-1.5 text-xs font-semibold uppercase text-gray-500">
              <Save className="h-3.5 w-3.5" /> Saved Searches
            </h3>
            {savedSearches.length === 0 ? (
              <p className="text-xs text-gray-400">No saved searches yet</p>
            ) : (
              <div className="space-y-1.5">
                {savedSearches.map((s: any) => (
                  <div
                    key={s.id}
                    className="group flex items-center justify-between rounded-lg border border-gray-200 p-2 hover:border-gray-300 dark:border-gray-700"
                  >
                    <button onClick={() => loadSearch(s)} className="flex-1 text-left">
                      <p className="truncate text-xs font-medium">{s.name}</p>
                      <p className="text-[10px] text-gray-400">{new Date(s.createdAt).toLocaleDateString()}</p>
                    </button>
                    <button
                      onClick={() => deleteSearch(s.id)}
                      className="ml-1 text-gray-300 opacity-0 group-hover:opacity-100 hover:text-red-500"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Scheduled Reports */}
          <div className="rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h3 className="mb-3 flex items-center gap-1.5 text-xs font-semibold uppercase text-gray-500">
              <Clock className="h-3.5 w-3.5" /> Scheduled Reports
            </h3>
            {scheduledReports.length === 0 ? (
              <p className="text-xs text-gray-400">No scheduled reports</p>
            ) : (
              <div className="space-y-2">
                {scheduledReports.map((r: any) => (
                  <div key={r.id} className="group rounded-lg border border-gray-200 p-2 dark:border-gray-700">
                    <div className="flex items-center justify-between">
                      <p className="truncate text-xs font-medium">{r.name}</p>
                      <button
                        onClick={() => toggleReport(r.id)}
                        className={`relative inline-flex h-4 w-7 shrink-0 rounded-full transition-colors ${
                          r.enabled ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
                        }`}
                      >
                        <span className={`inline-block h-3 w-3 transform rounded-full bg-white shadow transition ${r.enabled ? "translate-x-3.5" : "translate-x-0.5"} mt-0.5`} />
                      </button>
                    </div>
                    <div className="mt-1 flex items-center gap-2 text-[10px] text-gray-400">
                      <span className="rounded bg-gray-100 px-1 py-0.5 uppercase dark:bg-gray-700">{r.frequency}</span>
                      <span>{r.time}</span>
                      <Mail className="h-2.5 w-2.5" />
                      <span className="truncate">{r.recipients.split(",")[0]}{r.recipients.split(",").length > 1 ? "..." : ""}</span>
                    </div>
                    <button
                      onClick={() => deleteReport(r.id)}
                      className="mt-1 text-[10px] text-gray-300 opacity-0 group-hover:opacity-100 hover:text-red-500"
                    >
                      Delete
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ===== Save Search Dialog ===== */}
      {showSaveDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowSaveDialog(false)}>
          <div className="rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-4 text-lg font-semibold">Save Search</h3>
            <input
              value={newSearchName}
              onChange={(e) => setNewSearchName(e.target.value)}
              placeholder="Search name..."
              className="mb-4 w-72 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowSaveDialog(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={saveSearch}
                disabled={!newSearchName.trim()}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== Schedule Report Dialog ===== */}
      {showScheduleDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowScheduleDialog(false)}>
          <div className="rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <h3 className="mb-4 text-lg font-semibold">Schedule Report</h3>
            <div className="space-y-3">
              <input
                value={newReport.name}
                onChange={(e) => setNewReport({ ...newReport, name: e.target.value })}
                placeholder="Report name..."
                className="w-80 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <div className="flex gap-3">
                <select
                  value={newReport.frequency}
                  onChange={(e) => setNewReport({ ...newReport, frequency: e.target.value as "daily" | "weekly" })}
                  className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                >
                  <option value="daily">Daily</option>
                  <option value="weekly">Weekly</option>
                </select>
                <input
                  type="time"
                  value={newReport.time}
                  onChange={(e) => setNewReport({ ...newReport, time: e.target.value })}
                  className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
                />
              </div>
              <input
                value={newReport.recipients}
                onChange={(e) => setNewReport({ ...newReport, recipients: e.target.value })}
                placeholder="email@example.com, admin@example.com"
                className="w-80 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => setShowScheduleDialog(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={addScheduledReport}
                disabled={!newReport.name.trim() || !newReport.recipients.trim()}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                Schedule
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
