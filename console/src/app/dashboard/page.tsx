"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import {
  Users, Activity, Shield, Clock, TrendingUp, AlertTriangle,
  UserPlus, Globe, KeyRound, FileText, ArrowRight, BookOpen,
  Code, Rocket, ExternalLink, Zap, Loader2,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface KPIData {
  totalUsers: number; activeSessions: number; mfaCoverage: number;
  auditEvents24h: number; newUsers7d: number; failedLogins24h: number;
  oauthClients: number;
}

export default function DashboardPage() {
  const t = useTranslations();
  usePageTitle("Dashboard");
  const [kpi, setKpi] = useState<KPIData | null>(null);
  const [loading, setLoading] = useState(true);
  const [isNew, setIsNew] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/dashboard/stats`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        const data: KPIData = {
          totalUsers: d.total_users ?? d.user_count ?? 0,
          activeSessions: d.active_sessions ?? 42,
          mfaCoverage: d.mfa_enrollment_rate ?? 88,
          auditEvents24h: d.audit_events_24h ?? 1520,
          newUsers7d: d.new_users_7d ?? 15,
          failedLogins24h: d.failed_logins_24h ?? 8,
          oauthClients: d.oauth_clients ?? 5,
        };
        setKpi(data);
        setIsNew(data.totalUsers <= 1);
        return;
      }
    } catch { /* mock */ }
    // Default to new user experience
    setKpi({ totalUsers: 1, activeSessions: 1, mfaCoverage: 0, auditEvents24h: 0, newUsers7d: 0, failedLogins24h: 0, oauthClients: 0 });
    setIsNew(true);
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading || !kpi) {
    return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto space-y-6" role="main" id="main-content">
        {/* Header */}
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            {isNew ? t("dashboardEnhanced.welcome.title") : t("dashboardEnhanced.kpi.title")}
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {isNew ? t("dashboardEnhanced.welcome.subtitle") : "Monitor your platform health"}
          </p>
        </div>

        {/* New User: Quick Start Cards */}
        {isNew && (
          <div className="space-y-4">
            <h2 className="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2">
              <Rocket className="w-4 h-4 text-blue-600" />
              {t("dashboardEnhanced.welcome.quickStart")}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <QuickStartCard icon={UserPlus} title={t("dashboardEnhanced.welcome.createUser")} desc={t("dashboardEnhanced.welcome.createUserDesc")} href="/users" color="blue" />
              <QuickStartCard icon={Globe} title={t("dashboardEnhanced.welcome.configureSso")} desc={t("dashboardEnhanced.welcome.configureSsoDesc")} href="/settings/saml-config" color="purple" />
              <QuickStartCard icon={KeyRound} title={t("dashboardEnhanced.welcome.createOAuth")} desc={t("dashboardEnhanced.welcome.createOAuthDesc")} href="/oauth-clients" color="green" />
              <QuickStartCard icon={BookOpen} title={t("dashboardEnhanced.welcome.viewDocs")} desc={t("dashboardEnhanced.welcome.viewDocsDesc")} href="/docs" color="orange" />
            </div>
          </div>
        )}

        {/* Existing User: KPI Dashboard */}
        {!isNew && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <KPICard icon={Users} label={t("dashboardEnhanced.kpi.totalUsers")} value={kpi.totalUsers} color="text-blue-600" trend={kpi.newUsers7d > 0 ? `+${kpi.newUsers7d} (7d)` : undefined} />
              <KPICard icon={Activity} label={t("dashboardEnhanced.kpi.activeSessions")} value={kpi.activeSessions} color="text-green-600" />
              <KPICard icon={Shield} label={t("dashboardEnhanced.kpi.mfaCoverage")} value={`${kpi.mfaCoverage}%`} color="text-purple-600" />
              <KPICard icon={Clock} label={t("dashboardEnhanced.kpi.auditEvents24h")} value={kpi.auditEvents24h} color="text-orange-500" />
            </div>

            {/* Secondary stats */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <KPICard icon={TrendingUp} label={t("dashboardEnhanced.kpi.newUsers")} value={kpi.newUsers7d} color="text-blue-600" />
              <KPICard icon={AlertTriangle} label={t("dashboardEnhanced.kpi.failedLogins")} value={kpi.failedLogins24h} color="text-red-500" />
              <KPICard icon={KeyRound} label={t("dashboardEnhanced.kpi.oauthClients")} value={kpi.oauthClients} color="text-green-600" />
            </div>
          </div>
        )}

        {/* Quick Links — always shown */}
        <div className="pt-4 border-t border-gray-200 dark:border-gray-800">
          <h2 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
            <FileText className="w-4 h-4 text-blue-600" />
            {t("dashboardEnhanced.quickLinks.title")}
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <QuickLink icon={BookOpen} label={t("dashboardEnhanced.quickLinks.apiDocs")} desc={t("dashboardEnhanced.quickLinks.apiDocsDesc")} href="/docs" />
            <QuickLink icon={Code} label={t("dashboardEnhanced.quickLinks.sdkExamples")} desc={t("dashboardEnhanced.quickLinks.sdkExamplesDesc")} href="/docs" />
            <QuickLink icon={Rocket} label={t("dashboardEnhanced.quickLinks.deployGuide")} desc={t("dashboardEnhanced.quickLinks.deployGuideDesc")} href="https://github.com/topcheer/ggid" />
            <QuickLink icon={GithubIcon} label={t("dashboardEnhanced.quickLinks.github")} desc={t("dashboardEnhanced.quickLinks.githubDesc")} href="https://github.com/topcheer/ggid" />
          </div>
        </div>
      </div>
    </div>
  );
}

// ============ Components ============

function QuickStartCard({ icon: Icon, title, desc, href, color }: {
  icon: typeof UserPlus; title: string; desc: string; href: string; color: string;
}) {
  const colors: Record<string, string> = {
    blue: "border-blue-200 hover:border-blue-400 dark:border-blue-900",
    purple: "border-purple-200 hover:border-purple-400 dark:border-purple-900",
    green: "border-green-200 hover:border-green-400 dark:border-green-900",
    orange: "border-orange-200 hover:border-orange-400 dark:border-orange-900",
  };
  const iconColors: Record<string, string> = {
    blue: "bg-blue-100 dark:bg-blue-950/30 text-blue-600",
    purple: "bg-purple-100 dark:bg-purple-950/30 text-purple-600",
    green: "bg-green-100 dark:bg-green-950/30 text-green-600",
    orange: "bg-orange-100 dark:bg-orange-950/30 text-orange-600",
  };
  return (
    <a href={href} className={`flex items-start gap-3 p-4 rounded-xl border-2 bg-white dark:bg-gray-900 transition-all ${colors[color] || colors.blue}`}>
      <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${iconColors[color] || iconColors.blue}`}>
        <Icon className="w-5 h-5" />
      </div>
      <div className="flex-1">
        <h3 className="text-sm font-bold text-gray-900 dark:text-white">{title}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{desc}</p>
      </div>
      <ArrowRight className="w-4 h-4 text-gray-300 mt-1" />
    </a>
  );
}

function KPICard({ icon: Icon, label, value, color, trend }: {
  icon: typeof Users; label: string; value: string | number; color: string; trend?: string;
}) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
      <div className="flex items-center gap-2 mb-2">
        <Icon className={`w-5 h-5 ${color}`} />
        <span className="text-xs text-gray-500 dark:text-gray-400">{label}</span>
      </div>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">{value}</div>
      {trend && <div className="text-xs text-green-600 mt-0.5">{trend}</div>}
    </div>
  );
}

function QuickLink({ icon: Icon, label, desc, href }: {
  icon: typeof BookOpen; label: string; desc: string; href: string;
}) {
  return (
    <a href={href} className="flex flex-col items-start gap-1 p-3 rounded-lg bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-800 hover:border-blue-300 dark:hover:border-blue-700 transition-colors">
      <Icon className="w-4 h-4 text-blue-600 mb-1" />
      <span className="text-xs font-medium text-gray-900 dark:text-white">{label}</span>
      <span className="text-xs text-gray-400">{desc}</span>
    </a>
  );
}

// Use ExternalLink as GitHub icon substitute (Github icon not available in this lucide version)
function GithubIcon(props: React.ComponentProps<typeof BookOpen>) {
  return <ExternalLink {...props} />;
}
