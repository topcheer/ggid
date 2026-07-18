"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Activity, Clock, BarChart3, Search, Loader2, ChevronDown,
  ChevronRight, AlertTriangle, Zap, KeyRound, Shield, UserCog,
  TrendingUp, Users, Crown, Layers,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "timeline" | "sessions" | "stats";

interface PrivilegedOp {
  id: string; operator: string; action: string; target: string;
  elevated_role: string; duration_ms: number; scopes_delta: string[];
  timestamp: string; source: string; session_id: string; result: string;
}

interface Session {
  session_id: string; operator: string; call_count: number;
  first_call: string; last_call: string;
  ops: { seq: number; action: string; target: string; result: string; duration_ms: number }[];
}

const ACTION_LABELS: Record<string, string> = {
  break_glass: "actionBreakGlass", jit_elevate: "actionJitElevate",
  role_assign: "actionRoleAssign", policy_change: "actionPolicyChange",
  user_modify: "actionUserModify", config_change: "actionConfigChange",
  key_access: "actionKeyAccess",
};

const actionIcons: Record<string, typeof Shield> = {
  break_glass: AlertTriangle, jit_elevate: Zap, role_assign: UserCog,
  policy_change: Shield, user_modify: Users, config_change: Layers, key_access: KeyRound,
};

const actionColors: Record<string, string> = {
  break_glass: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  jit_elevate: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
  role_assign: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  policy_change: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
  user_modify: "bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300",
  config_change: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
  key_access: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
};

export default function PrivilegedActivityPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("timeline");
  const [ops, setOps] = useState<PrivilegedOp[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [filterAction, setFilterAction] = useState("all");
  const [filterRange, setFilterRange] = useState("24h");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (filterAction !== "all") params.set("action", filterAction);
      params.set("range", filterRange);
      if (search) params.set("q", search);
      const res = await fetch(`${API_BASE}/api/v1/identity/privileged-operations?${params}`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setOps(d.operations || d || []); return; }
    } catch { /* mock */ }
    setOps([
      { id: "1", operator: "admin@company.com", action: "break_glass", target: "production-db", elevated_role: "superadmin", duration_ms: 15000, scopes_delta: ["+all"], timestamp: "2025-07-18T09:35:00Z", source: "break_glass", session_id: "s-abc123", result: "success" },
      { id: "2", operator: "devops@company.com", action: "jit_elevate", target: "k8s-prod-cluster", elevated_role: "cluster-admin", duration_ms: 3600000, scopes_delta: ["+k8s:exec", "+k8s:logs"], timestamp: "2025-07-18T09:28:00Z", source: "jit", session_id: "s-def456", result: "success" },
      { id: "3", operator: "admin@company.com", action: "role_assign", target: "user:eve@company.com", elevated_role: "auditor", duration_ms: 0, scopes_delta: ["+audit:read"], timestamp: "2025-07-18T09:20:00Z", source: "ui", session_id: "s-abc123", result: "success" },
      { id: "4", operator: "sec-team@company.com", action: "policy_change", target: "policy:conditional-access-1", elevated_role: "policy-admin", duration_ms: 0, scopes_delta: ["+policy:write"], timestamp: "2025-07-18T09:10:00Z", source: "api", session_id: "s-ghi789", result: "success" },
      { id: "5", operator: "admin@company.com", action: "config_change", target: "config:auth-password-policy", elevated_role: "admin", duration_ms: 0, scopes_delta: [], timestamp: "2025-07-18T08:45:00Z", source: "ui", session_id: "s-abc123", result: "success" },
      { id: "6", operator: "devops@company.com", action: "key_access", target: "vault:prod-secrets", elevated_role: "vault-reader", duration_ms: 5000, scopes_delta: ["+vault:read"], timestamp: "2025-07-18T08:30:00Z", source: "cli", session_id: "s-def456", result: "failed" },
      { id: "7", operator: "admin@company.com", action: "user_modify", target: "user:dave@company.com", elevated_role: "admin", duration_ms: 0, scopes_delta: [], timestamp: "2025-07-18T08:15:00Z", source: "ui", session_id: "s-abc123", result: "success" },
      { id: "8", operator: "sec-team@company.com", action: "jit_elevate", target: "siem-platform", elevated_role: "siem-admin", duration_ms: 1800000, scopes_delta: ["+siem:read", "+siem:export"], timestamp: "2025-07-17T22:00:00Z", source: "jit", session_id: "s-jkl012", result: "success" },
    ]);
  }, [filterAction, filterRange, search]);

  useEffect(() => { load(); }, [load]);

  const tabs: { id: TabId; label: string; icon: typeof Activity }[] = [
    { id: "timeline", label: t("privilegedActivity.tabs.timeline"), icon: Activity },
    { id: "sessions", label: t("privilegedActivity.tabs.sessions"), icon: Layers },
    { id: "stats", label: t("privilegedActivity.tabs.stats"), icon: BarChart3 },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Shield className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("privilegedActivity.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("privilegedActivity.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "timeline" && <TimelineTab ops={ops} loading={loading} search={search} setSearch={setSearch}
          filterAction={filterAction} setFilterAction={setFilterAction}
          filterRange={filterRange} setFilterRange={setFilterRange} />}
        {tab === "sessions" && <SessionsTab ops={ops} loading={loading} />}
        {tab === "stats" && <StatsTab ops={ops} loading={loading} />}
      </div>
    </div>
  );
}

// ============ Timeline Tab ============

function TimelineTab({ ops, loading, search, setSearch, filterAction, setFilterAction, filterRange, setFilterRange }: {
  ops: PrivilegedOp[]; loading: boolean; search: string; setSearch: (v: string) => void;
  filterAction: string; setFilterAction: (v: string) => void;
  filterRange: string; setFilterRange: (v: string) => void;
}) {
  const t = useTranslations();

  if (loading) return <Spinner />;

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input type="text" value={search} onChange={(e) => setSearch(e.target.value)}
              placeholder={t("privilegedActivity.timeline.searchPlaceholder")}
              className="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
          </div>
          <select value={filterAction} onChange={(e) => setFilterAction(e.target.value)}
            className="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
            <option value="all">{t("privilegedActivity.timeline.filterAction")}</option>
            {Object.keys(ACTION_LABELS).map((a: any) => (
              <option key={a} value={a}>{t(`privilegedActivity.timeline.${ACTION_LABELS[a]}`)}</option>
            ))}
          </select>
          <select value={filterRange} onChange={(e) => setFilterRange(e.target.value)}
            className="px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
            <option value="1h">{t("privilegedActivity.timeline.range1h")}</option>
            <option value="24h">{t("privilegedActivity.timeline.range24h")}</option>
            <option value="7d">{t("privilegedActivity.timeline.range7d")}</option>
          </select>
        </div>
      </div>

      {/* Timeline */}
      {ops.length === 0 ? (
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
          <Activity className="w-12 h-12 mx-auto mb-3 text-gray-300" />
          <p className="text-sm text-gray-500">{t("privilegedActivity.timeline.noOperations")}</p>
        </div>
      ) : (
        <div className="relative">
          <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />
          <div className="space-y-3">
            {ops.map((op: any) => {
              const Icon = actionIcons[op.action] || Shield;
              const isBreakGlass = op.action === "break_glass";
              const isFailed = op.result === "failed";
              return (
                <div key={op.id} className="relative pl-12">
                  <div className={`absolute left-2 top-3 w-5 h-5 rounded-full flex items-center justify-center ring-4 ring-gray-50 dark:ring-gray-950 ${
                    isBreakGlass ? "bg-red-500" : isFailed ? "bg-orange-500" : "bg-blue-500"
                  }`}>
                    <Icon className="w-3 h-3 text-white" />
                  </div>
                  <div className={`bg-white dark:bg-gray-900 rounded-lg border p-3 ${
                    isBreakGlass ? "border-red-200 dark:border-red-900" : "border-gray-200 dark:border-gray-800"
                  }`}>
                    <div className="flex items-center gap-2 flex-wrap mb-1">
                      <span className={`px-1.5 py-0.5 text-xs rounded ${actionColors[op.action] || actionColors.config_change}`}>
                        {t(`privilegedActivity.timeline.${ACTION_LABELS[op.action] || "actionConfigChange"}`)}
                      </span>
                      <span className="text-xs text-gray-400">{new Date(op.timestamp).toLocaleString()}</span>
                      <span className="text-xs px-1.5 py-0.5 bg-gray-100 dark:bg-gray-800 text-gray-500 rounded">{t(`privilegedActivity.timeline.source${op.source.charAt(0).toUpperCase() + op.source.slice(1)}`)}</span>
                      {isFailed && <span className="text-xs px-1.5 py-0.5 bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300 rounded">FAILED</span>}
                    </div>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                      <Field label={t("privilegedActivity.timeline.operator")} value={op.operator} />
                      <Field label={t("privilegedActivity.timeline.target")} value={op.target} mono />
                      <Field label={t("privilegedActivity.timeline.elevatedRole")} value={op.elevated_role}>
                        <span className="inline-flex items-center gap-1"><Crown className="w-3 h-3 text-yellow-500" />{op.elevated_role}</span>
                      </Field>
                      <Field label={t("privilegedActivity.timeline.duration")} value={op.duration_ms > 0 ? formatDuration(op.duration_ms) : "—"} />
                    </div>
                    {op.scopes_delta.length > 0 && (
                      <div className="flex items-center gap-1 mt-2 flex-wrap">
                        <span className="text-xs text-gray-500">{t("privilegedActivity.timeline.scopesDelta")}:</span>
                        {op.scopes_delta.map((s: any) => (
                          <span key={s} className={`px-1.5 py-0.5 text-xs rounded ${s.startsWith("+") ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300"}`}>{s}</span>
                        ))}
                      </div>
                    )}
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

// ============ Sessions Tab ============

function SessionsTab({ ops, loading }: { ops: PrivilegedOp[]; loading: boolean }) {
  const t = useTranslations();
  const [expanded, setExpanded] = useState<string | null>(null);

  if (loading) return <Spinner />;

  // Group by session
  const sessions: Record<string, PrivilegedOp[]> = {};
  ops.forEach((op: any) => {
    if (!sessions[op.session_id]) sessions[op.session_id] = [];
    sessions[op.session_id].push(op);
  });
  const sessionList: Session[] = Object.entries(sessions).map(([sid, sOps]) => ({
    session_id: sid, operator: sOps[0].operator, call_count: sOps.length,
    first_call: sOps[sOps.length - 1].timestamp, last_call: sOps[0].timestamp,
    ops: sOps.map((o: any, i: number) => ({ seq: i + 1, action: o.action, target: o.target, result: o.result, duration_ms: o.duration_ms })),
  }));

  if (sessionList.length === 0) {
    return <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
      <Layers className="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p className="text-sm text-gray-500">{t("privilegedActivity.sessions.noSessions")}</p>
    </div>;
  }

  return (
    <div className="space-y-2">
      {sessionList.map((s: any) => (
        <div key={s.session_id} className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
          <button onClick={() => setExpanded(expanded === s.session_id ? null : s.session_id)}
            className="w-full flex items-center gap-3 p-4 hover:bg-gray-50 dark:hover:bg-gray-800/30">
            {expanded === s.session_id ? <ChevronDown className="w-4 h-4 text-gray-400" /> : <ChevronRight className="w-4 h-4 text-gray-400" />}
            <div className="flex-1 text-left">
              <span className="text-sm font-medium text-gray-900 dark:text-white">{s.operator}</span>
              <span className="text-xs text-gray-400 ml-2">{s.session_id}</span>
            </div>
            <span className="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300 rounded-full">{s.call_count} {t("privilegedActivity.sessions.callCount")}</span>
            <span className="text-xs text-gray-500">{new Date(s.first_call).toLocaleTimeString()} → {new Date(s.last_call).toLocaleTimeString()}</span>
          </button>
          {expanded === s.session_id && (
            <div className="border-t border-gray-200 dark:border-gray-800 p-4 bg-gray-50 dark:bg-gray-800/30">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left border-b border-gray-200 dark:border-gray-700">
                    <th className="py-2 px-3 font-medium text-gray-500 w-12">#</th>
                    <th className="py-2 px-3 font-medium text-gray-500">{t("privilegedActivity.sessions.actionColumn")}</th>
                    <th className="py-2 px-3 font-medium text-gray-500">{t("privilegedActivity.sessions.targetColumn")}</th>
                    <th className="py-2 px-3 font-medium text-gray-500">{t("privilegedActivity.sessions.resultColumn")}</th>
                    <th className="py-2 px-3 font-medium text-gray-500 text-right">{t("privilegedActivity.sessions.durationColumn")}</th>
                  </tr>
                </thead>
                <tbody>
                  {s.ops.map((op: any) => {
                    const Icon = actionIcons[op.action] || Shield;
                    return (
                      <tr key={op.seq} className="border-b border-gray-100 dark:border-gray-800/50">
                        <td className="py-2 px-3 text-gray-400">{op.seq}</td>
                        <td className="py-2 px-3">
                          <div className="flex items-center gap-2">
                            <Icon className="w-3.5 h-3.5 text-gray-400" />
                            <span className={`px-1.5 py-0.5 text-xs rounded ${actionColors[op.action] || actionColors.config_change}`}>
                              {t(`privilegedActivity.timeline.${ACTION_LABELS[op.action] || "actionConfigChange"}`)}
                            </span>
                          </div>
                        </td>
                        <td className="py-2 px-3 font-mono text-xs text-gray-900 dark:text-white">{op.target}</td>
                        <td className="py-2 px-3">
                          <span className={op.result === "success" ? "text-green-600 text-xs" : "text-red-600 text-xs"}>{op.result}</span>
                        </td>
                        <td className="py-2 px-3 text-right text-xs text-gray-500">{op.duration_ms > 0 ? formatDuration(op.duration_ms) : "—"}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// ============ Stats Tab ============

function StatsTab({ ops, loading }: { ops: PrivilegedOp[]; loading: boolean }) {
  const t = useTranslations();

  if (loading) return <Spinner />;
  if (ops.length === 0) {
    return <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
      <BarChart3 className="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p className="text-sm text-gray-500">{t("privilegedActivity.stats.noData")}</p>
    </div>;
  }

  // Compute stats
  const operators = new Map<string, number>();
  const roles = new Map<string, number>();
  const byDay = new Map<string, number>();
  const actionCounts = new Map<string, number>();

  ops.forEach((op: any) => {
    operators.set(op.operator, (operators.get(op.operator) || 0) + 1);
    roles.set(op.elevated_role, (roles.get(op.elevated_role) || 0) + 1);
    const day = op.timestamp.split("T")[0];
    byDay.set(day, (byDay.get(day) || 0) + 1);
    actionCounts.set(op.action, (actionCounts.get(op.action) || 0) + 1);
  });

  const breakGlassCount = actionCounts.get("break_glass") || 0;
  const jitCount = actionCounts.get("jit_elevate") || 0;
  const topOperators = [...operators.entries()].sort((a, b) => b[1] - a[1]).slice(0, 5);
  const topRoles = [...roles.entries()].sort((a, b) => b[1] - a[1]).slice(0, 5);
  const days = [...byDay.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  const maxDayCount = Math.max(...days.map((d: any) => d[1]), 1);

  return (
    <div className="space-y-4">
      {/* Top stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard icon={Activity} label={t("privilegedActivity.stats.totalOperations")} value={ops.length} color="text-blue-600" />
        <StatCard icon={Users} label={t("privilegedActivity.stats.uniqueOperators")} value={operators.size} color="text-green-600" />
        <StatCard icon={AlertTriangle} label={t("privilegedActivity.stats.breakGlassCount")} value={breakGlassCount} color="text-red-500" />
        <StatCard icon={Zap} label={t("privilegedActivity.stats.jitElevationCount")} value={jitCount} color="text-orange-500" />
      </div>

      {/* Operations by Day */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("privilegedActivity.stats.operationsByDay")}</h3>
        <div className="flex items-end gap-2 h-32">
          {days.map(([day, count]) => (
            <div key={day} className="flex-1 flex flex-col items-center gap-1">
              <div className="w-full bg-blue-500 rounded-t transition-all" style={{ height: `${(count / maxDayCount) * 100}%`, minHeight: "4px" }} />
              <span className="text-xs text-gray-400">{new Date(day).getDate()}</span>
              <span className="text-xs font-medium text-gray-700 dark:text-gray-300">{count}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Top Operators + Roles */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
            <TrendingUp className="w-4 h-4 text-blue-600" />
            {t("privilegedActivity.stats.topOperators")}
          </h3>
          <div className="space-y-2">
            {topOperators.map(([name, count]) => (
              <div key={name} className="flex items-center gap-3">
                <span className="text-sm text-gray-700 dark:text-gray-300 flex-1 truncate">{name}</span>
                <div className="w-20 h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                  <div className="h-full bg-blue-500 rounded-full" style={{ width: `${(count / topOperators[0][1]) * 100}%` }} />
                </div>
                <span className="text-xs font-medium text-gray-900 dark:text-white w-6 text-right">{count}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
            <Crown className="w-4 h-4 text-yellow-500" />
            {t("privilegedActivity.stats.mostElevatedRoles")}
          </h3>
          <div className="space-y-2">
            {topRoles.map(([role, count]) => (
              <div key={role} className="flex items-center gap-3">
                <span className="text-sm text-gray-700 dark:text-gray-300 flex-1 truncate">{role}</span>
                <div className="w-20 h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                  <div className="h-full bg-yellow-500 rounded-full" style={{ width: `${(count / topRoles[0][1]) * 100}%` }} />
                </div>
                <span className="text-xs font-medium text-gray-900 dark:text-white w-6 text-right">{count}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ============ Shared ============

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.round(ms / 60000)}m`;
  return `${(ms / 3600000).toFixed(1)}h`;
}

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }

function StatCard({ icon: Icon, label, value, color }: { icon: typeof Activity; label: string; value: number; color: string }) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
      <div className="flex items-center gap-2 mb-2"><Icon className={`w-5 h-5 ${color}`} /><span className="text-xs text-gray-500">{label}</span></div>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">{value}</div>
    </div>
  );
}

function Field({ label, value, mono, children }: { label: string; value: string; mono?: boolean; children?: React.ReactNode }) {
  return (
    <div>
      <span className="text-gray-400 block">{label}</span>
      <span className={`text-gray-900 dark:text-white ${mono ? "font-mono" : ""}`}>{children || value}</span>
    </div>
  );
}
