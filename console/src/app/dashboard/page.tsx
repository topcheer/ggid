"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Users,
  Shield,
  Activity,
  AlertTriangle,
  TrendingUp,
  Clock,
  Loader2,
  CheckCircle2,
  XCircle,
  ShieldCheck,
  Radio,
} from "lucide-react";

interface DashboardStats {
  total_users: number;
  active_sessions: number;
  failed_logins_24h: number;
  successful_logins_24h: number;
  mfa_enrollment_rate: number;
  audit_events_24h: number;
  pending_access_requests: number;
}

interface ActivityItem {
  id: string;
  action: string;
  actor_name: string;
  result: "success" | "failure" | "denied";
  created_at: string;
}

interface ServiceHealth {
  name: string;
  status: "healthy" | "degraded" | "down";
  latency_ms: number;
}

interface ComplianceInfo {
  score: number;
  grade: string;
  frameworks: { name: string; status: string; score: number }[];
}

interface SoDInfo {
  total_violations: number;
  critical: number;
  warning: number;
}

interface AccessReviewInfo {
  pending: number;
  overdue: number;
  completed_30d: number;
}

export default function DashboardPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [activity, setActivity] = useState<ActivityItem[]>([]);
  const [health, setHealth] = useState<ServiceHealth[]>([]);
  const [compliance, setCompliance] = useState<ComplianceInfo | null>(null);
  const [sod, setSod] = useState<SoDInfo | null>(null);
  const [accessReview, setAccessReview] = useState<AccessReviewInfo | null>(null);
  const [liveEvents, setLiveEvents] = useState<ActivityItem[]>([]);
  const [isLive, setIsLive] = useState(false);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [statsRes, actRes, healthRes, compRes, sodRes, reviewRes] = await Promise.all([
        apiFetch<DashboardStats>("/api/v1/dashboard/stats").catch(() => null),
        apiFetch<{ events?: ActivityItem[] }>("/api/v1/audit/events?page_size=8").catch(() => ({ events: [] })),
        apiFetch<{ services?: ServiceHealth[] }>("/api/v1/health/services").catch(() => ({ services: [] })),
        apiFetch<ComplianceInfo>("/api/v1/audit/compliance/posture").catch(() => null),
        apiFetch<SoDInfo>("/api/v1/policy/sod/violations/summary").catch(() => null),
        apiFetch<AccessReviewInfo>("/api/v1/audit/access-reviews/summary").catch(() => null),
      ]);
      if (statsRes) setStats(statsRes);
      setActivity(actRes?.events ?? []);
      setHealth(healthRes?.services ?? []);
      if (compRes) setCompliance(compRes);
      if (sodRes) setSod(sodRes);
      if (reviewRes) setAccessReview(reviewRes);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  // Live audit event stream via SSE
  useEffect(() => {
    const tok = localStorage.getItem("ggid_access_token");
    if (!tok || typeof window === "undefined" || !window.EventSource) return;
    const apiBase = window.location.origin;
    const tenantId = localStorage.getItem("ggid_tenant_id") || "00000000-0000-0000-0000-000000000001";
    const url = `${apiBase}/api/v1/audit/stream?token=${encodeURIComponent(tok)}&tenant_id=${encodeURIComponent(tenantId)}`;
    const es = new EventSource(url);
    es.onopen = () => setIsLive(true);
    es.onmessage = (msg) => {
      try {
        const event: ActivityItem = JSON.parse(msg.data);
        setLiveEvents((prev) => [event, ...prev].slice(0, 12));
      } catch { /* ignore */ }
    };
    es.onerror = () => setIsLive(false);
    return () => es.close();
  }, []);

  useEffect(() => {
    load();
    const interval = setInterval(load, 30000);
    return () => clearInterval(interval);
  }, [load]);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const resultIcon = (result: string) => {
    if (result === "success") return <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />;
    if (result === "failure") return <XCircle className="h-3.5 w-3.5 text-red-500" />;
    return <AlertTriangle className="h-3.5 w-3.5 text-yellow-500" />;
  };

  const healthColor = (status: string) =>
    status === "healthy" ? "text-green-500" : status === "degraded" ? "text-yellow-500" : "text-red-500";

  const complianceColor = compliance
    ? compliance.score >= 90 ? "text-green-600" : compliance.score >= 70 ? "text-yellow-600" : "text-red-600"
    : "text-gray-400";

  const statCards = stats ? [
    { label: t("dashboard.totalusers"), value: stats.total_users, icon: Users, color: "text-indigo-600" },
    { label: t("dashboard.activesessions"), value: stats.active_sessions, icon: Shield, color: "text-green-600" },
    { label: t("dashboard.logins24h"), value: stats.successful_logins_24h, icon: TrendingUp, color: "text-blue-600" },
    { label: t("dashboard.failedlogins"), value: stats.failed_logins_24h, icon: AlertTriangle, color: "text-red-600" },
    { label: t("dashboard.mfaenrollment"), value: `${stats.mfa_enrollment_rate}%`, icon: Shield, color: "text-purple-600" },
    { label: t("dashboard.auditevents"), value: stats.audit_events_24h, icon: Activity, color: "text-orange-600" },
    { label: t("dashboard.accessrequests"), value: stats.pending_access_requests, icon: Clock, color: "text-cyan-600" },
    { label: t("dashboard.servicesup"), value: `${health.filter((s) => s.status === "healthy").length}/${health.length}`, icon: CheckCircle2, color: "text-green-600" },
  ] : [];

  if (loading && !stats) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("dashboard.dashboard")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Real-time overview. Auto-refreshes every 30s.
        </p>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {statCards.map((card) => {
          const Icon = card.icon;
          return (
            <div key={card.label} className={cardCls}>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-xs font-medium text-gray-400">{card.label}</p>
                  <p className={`mt-1 text-2xl font-bold ${card.color}`}>{card.value}</p>
                </div>
                <Icon className={`h-8 w-8 ${card.color} opacity-50`} />
              </div>
            </div>
          );
        })}
      </div>

      {/* SoD violations + Access review widgets */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {sod && (
          <div className={cardCls}>
            <h3 className="mb-2 flex items-center gap-2 text-xs font-semibold text-gray-700 dark:text-gray-300">
              <AlertTriangle className="h-4 w-4 text-amber-500" /> SoD Violations
            </h3>
            <div className="flex items-baseline gap-2">
              <span className="text-2xl font-bold text-amber-600">{sod.total_violations}</span>
              {sod.critical > 0 && (
                <span className="text-xs font-medium text-red-500">{sod.critical} critical</span>
              )}
            </div>
          </div>
        )}
        {accessReview && (
          <div className={cardCls}>
            <h3 className="mb-2 flex items-center gap-2 text-xs font-semibold text-gray-700 dark:text-gray-300">
              <Clock className="h-4 w-4 text-cyan-500" /> Access Reviews
            </h3>
            <div className="flex items-baseline gap-2">
              <span className="text-2xl font-bold text-cyan-600">{accessReview.pending}</span>
              <span className="text-xs text-gray-400">pending</span>
              {accessReview.overdue > 0 && (
                <span className="text-xs font-medium text-red-500">{accessReview.overdue} overdue</span>
              )}
            </div>
          </div>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Compliance score card */}
        {compliance && (
          <div className={cardCls}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
              <ShieldCheck className="h-4 w-4" /> Compliance Score
            </h3>
            <div className="flex items-center gap-4">
              <div className={`text-4xl font-bold ${complianceColor}`}>{compliance.score}%</div>
              <div>
                <span className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">
                  Grade {compliance.grade}
                </span>
              </div>
            </div>
            <div className="mt-4 space-y-2">
              {compliance.frameworks.map((fw) => (
                <div key={fw.name} className="flex items-center justify-between text-xs">
                  <span className="text-gray-600 dark:text-gray-300">{fw.name}</span>
                  <div className="flex items-center gap-2">
                    <div className="h-1.5 w-20 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <div className={`h-full rounded-full ${fw.score >= 90 ? "bg-green-500" : fw.score >= 70 ? "bg-yellow-500" : "bg-red-500"}`} style={{ width: `${fw.score}%` }} />
                    </div>
                    <span className={complianceColor}>{fw.score}%</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Recent activity */}
        <div className={compliance ? "lg:col-span-1" : "lg:col-span-2"}>
          <div className={cardCls}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
              <Activity className="h-4 w-4" /> Recent Activity
              {isLive && <span className="flex items-center gap-1 text-xs text-green-500"><Radio className="h-3 w-3 animate-pulse" /> LIVE</span>}
            </h3>
            {(liveEvents.length > 0 ? liveEvents : activity).length === 0 ? (
              <p className="py-6 text-center text-sm text-gray-400">{t("dashboard.norecentactivity")}</p>
            ) : (
              <div className="space-y-2">
                {(liveEvents.length > 0 ? liveEvents : activity).map((item) => (
                  <div key={item.id} className="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/30">
                    <div className="flex items-center gap-2">
                      {resultIcon(item.result)}
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-200">{item.action}</span>
                      <span className="text-xs text-gray-400">by {item.actor_name || "system"}</span>
                    </div>
                    <span className="text-xs text-gray-400">
                      {new Date(item.created_at).toLocaleTimeString()}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Service health */}
        <div>
          <div className={cardCls}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
              <Shield className="h-4 w-4" /> Service Health
            </h3>
            {health.length === 0 ? (
              <p className="py-6 text-center text-sm text-gray-400">{t("dashboard.nohealthdata")}</p>
            ) : (
              <div className="space-y-2">
                {health.map((svc) => (
                  <div key={svc.name} className="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/30">
                    <span className="text-sm font-medium capitalize text-gray-700 dark:text-gray-200">{svc.name}</span>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-400">{svc.latency_ms}ms</span>
                      <span className={`h-2 w-2 rounded-full ${svc.status === "healthy" ? "bg-green-500" : svc.status === "degraded" ? "bg-yellow-500" : "bg-red-500"}`} />
                      <span className={`text-xs font-medium ${healthColor(svc.status)}`}>{svc.status}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
