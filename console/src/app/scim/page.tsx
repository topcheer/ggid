"use client";

import { useState, useEffect, useCallback, Fragment } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  RefreshCw,
  Users,
  FolderTree,
  Settings2,
  Plus,
  Trash2,
  ChevronDown,
  ChevronRight,
  CheckCircle,
  XCircle,
  AlertCircle,
  Clock,
  Zap,
  X,
  ArrowRight,
  Loader2,
  Link2,
} from "lucide-react";

interface ScimApp {
  id: string;
  name: string;
  icon: string;
  status: "connected" | "error" | "pending";
  last_sync: string;
  user_count: number;
  group_count: number;
  total_users?: number;
  synced_users?: number;
  total_groups?: number;
  synced_groups?: number;
}

interface SyncEvent {
  id: string;
  app: string;
  type: "user_created" | "user_updated" | "user_deactivated" | "group_created" | "group_updated" | "group_deactivated";
  entity: string;
  timestamp: string;
  status: "success" | "failed";
}

interface SyncHistoryEntry {
  id: string;
  timestamp: string;
  app: string;
  type: "full" | "incremental";
  users_processed: number;
  status: "success" | "partial" | "failed";
  duration: string;
  details?: string;
}

interface AttributeMapping {
  id: string;
  source: string;
  target: string;
}

const MOCK_APPS: ScimApp[] = [
  {
    id: "slack",
    name: "Slack",
    icon: "💬",
    status: "connected",
    last_sync: new Date(Date.now() - 300000).toISOString(),
    user_count: 142,
    group_count: 8,
    total_users: 150,
    synced_users: 142,
    total_groups: 10,
    synced_groups: 8,
  },
  {
    id: "google",
    name: "Google Workspace",
    icon: "🌐",
    status: "connected",
    last_sync: new Date(Date.now() - 900000).toISOString(),
    user_count: 198,
    group_count: 15,
    total_users: 200,
    synced_users: 198,
    total_groups: 15,
    synced_groups: 15,
  },
  {
    id: "okta",
    name: "Okta",
    icon: "🔐",
    status: "error",
    last_sync: new Date(Date.now() - 3600000).toISOString(),
    user_count: 0,
    group_count: 0,
    total_users: 120,
    synced_users: 0,
    total_groups: 6,
    synced_groups: 0,
  },
  {
    id: "custom",
    name: "Custom App",
    icon: "⚙️",
    status: "pending",
    last_sync: "—",
    user_count: 0,
    group_count: 0,
  },
];

const SOURCE_ATTRIBUTES = [
  "email",
  "displayName",
  "firstName",
  "lastName",
  "department",
  "title",
  "phone",
  "groups",
  "active",
  "userName",
];

const TARGET_ATTRIBUTES = [
  "emails[0].value",
  "displayName",
  "name.givenName",
  "name.familyName",
  "department",
  "title",
  "phoneNumbers[0].value",
  "groups",
  "active",
  "userName",
];

const DEFAULT_MAPPINGS: AttributeMapping[] = [
  { id: "m1", source: "email", target: "emails[0].value" },
  { id: "m2", source: "displayName", target: "displayName" },
  { id: "m3", source: "firstName", target: "name.givenName" },
  { id: "m4", source: "lastName", target: "name.familyName" },
  { id: "m5", source: "active", target: "active" },
];

const MOCK_SYNC_EVENTS: SyncEvent[] = [
  { id: "e1", app: "Slack", type: "user_created", entity: "john.doe@ggid.dev", timestamp: new Date(Date.now() - 120000).toISOString(), status: "success" },
  { id: "e2", app: "Slack", type: "user_updated", entity: "jane.smith@ggid.dev", timestamp: new Date(Date.now() - 300000).toISOString(), status: "success" },
  { id: "e3", app: "Google Workspace", type: "user_deactivated", entity: "old.user@ggid.dev", timestamp: new Date(Date.now() - 600000).toISOString(), status: "success" },
  { id: "e4", app: "Slack", type: "group_created", entity: "engineering-team", timestamp: new Date(Date.now() - 900000).toISOString(), status: "success" },
  { id: "e5", app: "Google Workspace", type: "user_updated", entity: "bob.wilson@ggid.dev", timestamp: new Date(Date.now() - 1200000).toISOString(), status: "success" },
  { id: "e6", app: "Okta", type: "user_created", entity: "test.user@ggid.dev", timestamp: new Date(Date.now() - 1800000).toISOString(), status: "failed" },
];

const MOCK_HISTORY: SyncHistoryEntry[] = [
  { id: "h1", timestamp: new Date(Date.now() - 300000).toISOString(), app: "Slack", type: "incremental", users_processed: 3, status: "success", duration: "2.1s" },
  { id: "h2", timestamp: new Date(Date.now() - 900000).toISOString(), app: "Google Workspace", type: "incremental", users_processed: 5, status: "success", duration: "4.3s" },
  { id: "h3", timestamp: new Date(Date.now() - 3600000).toISOString(), app: "Okta", type: "full", users_processed: 120, status: "failed", duration: "0.5s", details: "Connection refused: SCIM endpoint unreachable" },
  { id: "h4", timestamp: new Date(Date.now() - 7200000).toISOString(), app: "Slack", type: "full", users_processed: 150, status: "success", duration: "12.8s" },
  { id: "h5", timestamp: new Date(Date.now() - 86400000).toISOString(), app: "Google Workspace", type: "full", users_processed: 200, status: "partial", duration: "18.2s", details: "2 users failed: missing required attributes" },
];

function statusConfig(status: ScimApp["status"]) {
  switch (status) {
    case "connected":
      return { color: "text-green-600 dark:text-green-400", bgColor: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle, label: "Connected" };
    case "error":
      return { color: "text-red-600 dark:text-red-400", bgColor: "bg-red-100 dark:bg-red-900/30", icon: XCircle, label: "Error" };
    case "pending":
      return { color: "text-yellow-600 dark:text-yellow-400", bgColor: "bg-yellow-100 dark:bg-yellow-900/30", icon: AlertCircle, label: "Pending" };
  }
}

function formatTime(dateStr: string): string {
  if (dateStr === "—") return "Never";
  const d = new Date(dateStr);
  const diffMs = Date.now() - d.getTime();
  if (diffMs < 60000) return "Just now";
  if (diffMs < 3600000) return `${Math.floor(diffMs / 60000)}m ago`;
  if (diffMs < 86400000) return `${Math.floor(diffMs / 3600000)}h ago`;
  return d.toLocaleString("en-US", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

let mappingSeq = 100;

export default function ScimPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [apps, setApps] = useState<ScimApp[]>(MOCK_APPS);
  const [syncEvents] = useState<SyncEvent[]>(MOCK_SYNC_EVENTS);
  const [history] = useState<SyncHistoryEntry[]>(MOCK_HISTORY);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [mappings, setMappings] = useState<AttributeMapping[]>(DEFAULT_MAPPINGS);
  const [expandedHistory, setExpandedHistory] = useState<string | null>(null);
  const [syncTarget, setSyncTarget] = useState<ScimApp | null>(null);
  const [syncType, setSyncType] = useState<"full" | "incremental">("incremental");
  const [syncing, setSyncing] = useState(false);

  const loadApps = useCallback(async () => {
    try {
      const data = await apiFetch<{ apps?: ScimApp[] } | ScimApp[]>("/api/v1/scim/apps").catch(() => null);
      if (data) {
        const list = Array.isArray(data) ? data : data.apps || [];
        if (list.length > 0) {
          setApps(list);
        }
      }
    } catch {
      // Keep mock data
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadApps();
  }, [loadApps]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleSyncNow = async () => {
    if (!syncTarget) return;
    setSyncing(true);
    try {
      await apiFetch(`/api/v1/scim/apps/${syncTarget.id}/sync`, {
        method: "POST",
        body: JSON.stringify({ type: syncType }),
      }).catch(() => null);
      setMsg(`${syncType === "full" ? "Full" : "Incremental"} sync started for ${syncTarget.name}`);
      // Update last_sync
      setApps((prev) =>
        prev.map((a) =>
          a.id === syncTarget.id ? { ...a, last_sync: new Date().toISOString() } : a,
        ),
      );
    } catch (err) {
      setMsg(err instanceof Error ? err.message : "Sync failed");
    } finally {
      setSyncing(false);
      setSyncTarget(null);
    }
  };

  const addMapping = () => {
    setMappings([...mappings, { id: `m${mappingSeq++}`, source: "", target: "" }]);
  };

  const updateMapping = (id: string, field: "source" | "target", value: string) => {
    setMappings(mappings.map((m) => (m.id === id ? { ...m, [field]: value } : m)));
  };

  const deleteMapping = (id: string) => {
    setMappings(mappings.filter((m) => m.id !== id));
  };

  const inputCls =
    "w-full rounded-md border border-gray-300 px-2 py-1.5 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  const getEventLabel = (type: SyncEvent["type"]) => {
    const labels: Record<SyncEvent["type"], string> = {
      user_created: "User Created",
      user_updated: "User Updated",
      user_deactivated: "User Deactivated",
      group_created: "Group Created",
      group_updated: "Group Updated",
      group_deactivated: "Group Deactivated",
    };
    return labels[type] || type;
  };

  const getHistoryStatus = (status: SyncHistoryEntry["status"]) => {
    switch (status) {
      case "success":
        return { color: "text-green-600 dark:text-green-400", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle };
      case "partial":
        return { color: "text-yellow-600 dark:text-yellow-400", bg: "bg-yellow-100 dark:bg-yellow-900/30", icon: AlertCircle };
      case "failed":
        return { color: "text-red-600 dark:text-red-400", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle };
    }
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {t("scim.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("scim.subtitle")}
          </p>
        </div>
        <button
          onClick={loadApps}
          className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          <RefreshCw className="h-4 w-4" /> {t("common.refresh")}
        </button>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-700 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-400">
          {msg}
        </div>
      )}

      {loading ? (
        <div className="py-12 text-center text-gray-500">{t("scim.loadingApps")}</div>
      ) : (
        <>
          {/* Connected Apps */}
          <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {apps.map((app) => {
              const sc = statusConfig(app.status);
              const StatusIcon = sc.icon;
              const userProgress = app.total_users ? Math.round((app.synced_users || 0) / app.total_users * 100) : 0;
              const groupProgress = app.total_groups ? Math.round((app.synced_groups || 0) / app.total_groups * 100) : 0;

              return (
                <div
                  key={app.id}
                  className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800"
                >
                  <div className="mb-3 flex items-start justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-2xl">{app.icon}</span>
                      <div>
                        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{app.name}</h3>
                        <span className={`inline-flex items-center gap-1 rounded-full ${sc.bgColor} px-2 py-0.5 text-xs font-medium ${sc.color}`}>
                          <StatusIcon className="h-3 w-3" />
                          {sc.label}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Stats */}
                  <div className="mb-3 grid grid-cols-2 gap-2 text-xs">
                    <div className="rounded-lg bg-gray-50 p-2 dark:bg-gray-900/40">
                      <div className="flex items-center gap-1 text-gray-400">
                        <Users className="h-3 w-3" /> Users
                      </div>
                      <p className="mt-0.5 font-semibold text-gray-900 dark:text-gray-100">
                        {app.synced_users ?? app.user_count}
                        {app.total_users ? ` / ${app.total_users}` : ""}
                      </p>
                    </div>
                    <div className="rounded-lg bg-gray-50 p-2 dark:bg-gray-900/40">
                      <div className="flex items-center gap-1 text-gray-400">
                        <FolderTree className="h-3 w-3" /> Groups
                      </div>
                      <p className="mt-0.5 font-semibold text-gray-900 dark:text-gray-100">
                        {app.synced_groups ?? app.group_count}
                        {app.total_groups ? ` / ${app.total_groups}` : ""}
                      </p>
                    </div>
                  </div>

                  {/* Progress bars */}
                  {app.total_users && (
                    <div className="mb-2">
                      <div className="mb-0.5 flex justify-between text-xs text-gray-400">
                <span>{t("scim.userSync")}</span>
                        <span>{userProgress}%</span>
                      </div>
                      <div className="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                        <div
                          className={`h-full rounded-full transition-all ${userProgress === 100 ? "bg-green-500" : "bg-brand-500"}`}
                          style={{ width: `${userProgress}%` }}
                        />
                      </div>
                    </div>
                  )}
                  {app.total_groups && (
                    <div className="mb-2">
                      <div className="mb-0.5 flex justify-between text-xs text-gray-400">
                <span>{t("scim.groupSync")}</span>
                        <span>{groupProgress}%</span>
                      </div>
                      <div className="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                        <div
                          className={`h-full rounded-full transition-all ${groupProgress === 100 ? "bg-green-500" : "bg-brand-500"}`}
                          style={{ width: `${groupProgress}%` }}
                        />
                      </div>
                    </div>
                  )}

                  <div className="mt-2 flex items-center justify-between">
                    <span className="flex items-center gap-1 text-xs text-gray-400">
                      <Clock className="h-3 w-3" />
                      {formatTime(app.last_sync)}
                    </span>
                    <button
                      onClick={() => {
                        setSyncTarget(app);
                        setSyncType("incremental");
                      }}
                      disabled={app.status === "pending"}
                      className="flex items-center gap-1 rounded-md bg-brand-600 px-2.5 py-1 text-xs font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                    >
                      <Zap className="h-3 w-3" /> {t("scim.syncNow")}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Two-column: Sync Events + Group Sync */}
          <div className="mb-6 grid gap-4 lg:grid-cols-2">
            {/* User Sync Events */}
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                <Users className="h-4 w-4 text-gray-400" /> {t("scim.recentUserSync")}
              </h2>
              <div className="space-y-2">
                {syncEvents.filter((e) => e.type.startsWith("user")).map((event) => (
                  <div
                    key={event.id}
                    className="flex items-center justify-between rounded-lg border border-gray-100 p-2 dark:border-gray-700"
                  >
                    <div className="flex items-center gap-2">
                      <div
                        className={`flex h-7 w-7 items-center justify-center rounded-full ${
                          event.status === "success"
                            ? "bg-green-100 dark:bg-green-900/30"
                            : "bg-red-100 dark:bg-red-900/30"
                        }`}
                      >
                        {event.status === "success" ? (
                          <CheckCircle className="h-3.5 w-3.5 text-green-500" />
                        ) : (
                          <XCircle className="h-3.5 w-3.5 text-red-500" />
                        )}
                      </div>
                      <div>
                        <p className="text-xs font-medium text-gray-900 dark:text-gray-200">
                          {getEventLabel(event.type)}
                        </p>
                        <p className="font-mono text-xs text-gray-500 dark:text-gray-400">
                          {event.entity}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-gray-400">{event.app}</p>
                      <p className="text-xs text-gray-400">{formatTime(event.timestamp)}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Group Sync Events */}
            <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                <FolderTree className="h-4 w-4 text-gray-400" /> {t("scim.recentGroupSync")}
              </h2>
              <div className="space-y-2">
                {syncEvents.filter((e) => e.type.startsWith("group")).map((event) => (
                  <div
                    key={event.id}
                    className="flex items-center justify-between rounded-lg border border-gray-100 p-2 dark:border-gray-700"
                  >
                    <div className="flex items-center gap-2">
                      <div
                        className={`flex h-7 w-7 items-center justify-center rounded-full ${
                          event.status === "success"
                            ? "bg-green-100 dark:bg-green-900/30"
                            : "bg-red-100 dark:bg-red-900/30"
                        }`}
                      >
                        {event.status === "success" ? (
                          <CheckCircle className="h-3.5 w-3.5 text-green-500" />
                        ) : (
                          <XCircle className="h-3.5 w-3.5 text-red-500" />
                        )}
                      </div>
                      <div>
                        <p className="text-xs font-medium text-gray-900 dark:text-gray-200">
                          {getEventLabel(event.type)}
                        </p>
                        <p className="font-mono text-xs text-gray-500 dark:text-gray-400">
                          {event.entity}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-gray-400">{event.app}</p>
                      <p className="text-xs text-gray-400">{formatTime(event.timestamp)}</p>
                    </div>
                  </div>
                ))}
                {syncEvents.filter((e) => e.type.startsWith("group")).length === 0 && (
                  <p className="py-4 text-center text-xs text-gray-400">{t("scim.noGroupSyncEvents")}</p>
                )}
              </div>
            </div>
          </div>

          {/* Attribute Mappings */}
          <div className="mb-6 rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
              <Link2 className="h-4 w-4 text-gray-400" /> {t("scim.attributeMappings")}
              <span className="text-xs font-normal text-gray-400">
                {t("scim.sourceToTarget")}
              </span>
            </h2>
            <div className="space-y-2">
              {mappings.map((mapping) => (
                <div key={mapping.id} className="flex items-center gap-2">
                  <select
                    value={mapping.source}
                    onChange={(e) => updateMapping(mapping.id, "source", e.target.value)}
                    className={inputCls}
                  >
                    <option value="">{t("scim.sourceAttr")}</option>
                    {SOURCE_ATTRIBUTES.map((a) => (
                      <option key={a} value={a}>{a}</option>
                    ))}
                  </select>
                  <ArrowRight className="h-4 w-4 flex-shrink-0 text-gray-400" />
                  <select
                    value={mapping.target}
                    onChange={(e) => updateMapping(mapping.id, "target", e.target.value)}
                    className={inputCls}
                  >
                    <option value="">{t("scim.targetAttr")}</option>
                    {TARGET_ATTRIBUTES.map((a) => (
                      <option key={a} value={a}>{a}</option>
                    ))}
                  </select>
                  <button
                    onClick={() => deleteMapping(mapping.id)}
                    className="flex-shrink-0 rounded-md p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
              <button
                onClick={addMapping}
                className="flex items-center gap-1.5 rounded-lg border border-dashed border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-500 hover:border-brand-400 hover:text-brand-600 dark:border-gray-600 dark:text-gray-400"
              >
                <Plus className="h-3.5 w-3.5" /> {t("scim.addMapping")}
              </button>
            </div>
          </div>

          {/* Sync History */}
          <div className="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="border-b border-gray-100 p-4 text-sm font-semibold text-gray-900 dark:border-gray-700 dark:text-gray-100">
              {t("scim.syncHistory")}
            </h2>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-100 text-xs text-gray-400 dark:border-gray-700">
                    <th scope="col" className="px-4 py-2 text-left font-medium">{t("scim.timestamp")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium">{t("scim.app")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium">{t("scim.type")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium">{t("scim.users")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium">{t("scim.status")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium">{t("scim.duration")}</th>
                    <th scope="col" className="px-4 py-2 text-left font-medium"></th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((entry) => {
                    const sc = getHistoryStatus(entry.status);
                    const StatusIcon = sc.icon;
                    const isExpanded = expandedHistory === entry.id;
                    return (
                      <Fragment key={entry.id}>
                        <tr className="border-b border-gray-50 dark:border-gray-700/50">
                          <td className="px-4 py-2.5 text-xs text-gray-600 dark:text-gray-400">
                            {formatTime(entry.timestamp)}
                          </td>
                          <td className="px-4 py-2.5 text-xs font-medium text-gray-900 dark:text-gray-200">
                            {entry.app}
                          </td>
                          <td className="px-4 py-2.5">
                            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                              {entry.type}
                            </span>
                          </td>
                          <td className="px-4 py-2.5 text-xs text-gray-600 dark:text-gray-400">
                            {entry.users_processed}
                          </td>
                          <td className="px-4 py-2.5">
                            <span className={`inline-flex items-center gap-1 rounded-full ${sc.bg} px-2 py-0.5 text-xs font-medium ${sc.color}`}>
                              <StatusIcon className="h-3 w-3" />
                              {entry.status}
                            </span>
                          </td>
                          <td className="px-4 py-2.5 font-mono text-xs text-gray-600 dark:text-gray-400">
                            {entry.duration}
                          </td>
                          <td className="px-4 py-2.5">
                            {entry.details && (
                              <button
                                onClick={() => setExpandedHistory(isExpanded ? null : entry.id)}
                                className="text-xs text-brand-600 hover:underline dark:text-brand-400"
                              >
                                {isExpanded ? (
                                  <ChevronDown className="h-4 w-4" />
                                ) : (
                                  <ChevronRight className="h-4 w-4" />
                                )}
                              </button>
                            )}
                          </td>
                        </tr>
                        {isExpanded && entry.details && (
                          <tr>
                            <td colSpan={7} className="bg-gray-50 px-4 py-3 dark:bg-gray-900/30">
                              <div className="flex items-start gap-2">
                                <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-yellow-500" />
                                <p className="text-xs text-gray-600 dark:text-gray-400">
                                  {entry.details}
                                </p>
                              </div>
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      {/* Sync Confirmation Modal */}
      {syncTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-800">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-bold text-gray-900 dark:text-gray-100">
                <span className="text-xl">{syncTarget.icon}</span>
                {t("scim.sync")} {syncTarget.name}
              </h2>
              <button
                onClick={() => setSyncTarget(null)}
                className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <p className="mb-4 text-sm text-gray-500 dark:text-gray-400">
              {t("scim.syncDesc")}
            </p>

            <div className="mb-4 space-y-2">
              <label className="flex cursor-pointer items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <input
                  type="radio"
                  name="syncType"
                  value="incremental"
                  checked={syncType === "incremental"}
                  onChange={() => setSyncType("incremental")}
                  className="h-4 w-4"
                />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t("scim.incrementalSync")}</p>
                  <p className="text-xs text-gray-400">{t("scim.incrementalDesc")}</p>
                </div>
              </label>
              <label className="flex cursor-pointer items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                <input
                  type="radio"
                  name="syncType"
                  value="full"
                  checked={syncType === "full"}
                  onChange={() => setSyncType("full")}
                  className="h-4 w-4"
                />
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t("scim.fullSync")}</p>
                  <p className="text-xs text-gray-400">{t("scim.fullDesc")}</p>
                </div>
              </label>
            </div>

            <div className="flex justify-end gap-2">
              <button
                onClick={() => setSyncTarget(null)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              >
                {t("scim.cancel")}
              </button>
              <button
                onClick={handleSyncNow}
                disabled={syncing}
                className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                {syncing ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" /> {t("scim.syncing")}
                  </>
                ) : (
                  <>
                    <Zap className="h-4 w-4" /> {t("scim.startSync")}
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
