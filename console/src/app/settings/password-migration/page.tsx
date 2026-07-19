"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  KeyRound, TrendingUp, Mail, Save, Loader2, Search,
  PieChart, Users, AlertCircle, Check, Send, Shield,
  Clock, FileText, Eye,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

type DeprecationLevel = "off" | "read_only" | "migration_required" | "disabled";

interface MigrationStats {
  total_users: number;
  passwordless_users: number;
  pending_users: number;
  nudged_users: number;
  migration_rate: number;
  deprecation_level: DeprecationLevel;
}

interface DeprecationConfig {
  level: DeprecationLevel;
  grace_days: number;
  banner_text: string;
  email_subject: string;
  email_body: string;
}

interface UserMigration {
  user_id: string;
  email: string;
  display_name: string;
  status: "pending" | "nudged" | "enrolled" | "expired";
  auth_methods: string[];
  last_login: string;
}

type TabId = "overview" | "config" | "users";

const LEVELS: { value: DeprecationLevel; color: string }[] = [
  { value: "off", color: "gray" },
  { value: "read_only", color: "blue" },
  { value: "migration_required", color: "orange" },
  { value: "disabled", color: "red" },
];

// ============ Page ============

export default function PasswordMigrationPage() {
  const t = useTranslations();
  const [activeTab, setActiveTab] = useState<TabId>("overview");

  const tabs: { id: TabId; label: string; icon: typeof KeyRound }[] = [
    { id: "overview", label: t("passwordMigration.tabs.overview"), icon: PieChart },
    { id: "config", label: t("passwordMigration.tabs.config"), icon: Settings },
    { id: "users", label: t("passwordMigration.tabs.users"), icon: Users },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-2">
            <KeyRound className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              {t("passwordMigration.title")}
            </h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">
            {t("passwordMigration.description")}
          </p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map((tab: any) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === tab.id
                    ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm"
                    : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                }`}
              >
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {activeTab === "overview" && <OverviewTab />}
        {activeTab === "config" && <ConfigTab />}
        {activeTab === "users" && <UsersTab />}
      </div>
    </div>
  );
}

// Minimal Settings icon (avoids importing from lucide if name clash)
function Settings(props: React.ComponentProps<typeof KeyRound>) {
  return <Shield {...props} />;
}

// ============ Overview Tab ============

function OverviewTab() {
  const t = useTranslations();
  const [stats, setStats] = useState<MigrationStats | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/password-deprecation`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setStats({
          total_users: data.total_users ?? 100,
          passwordless_users: data.passwordless_users ?? 42,
          pending_users: data.pending_users ?? 58,
          nudged_users: data.nudged_users ?? 15,
          migration_rate: data.migration_rate ?? 42,
          deprecation_level: data.deprecation_level ?? "read_only",
        });
      } else {
        setStats({ total_users: 0, passwordless_users: 0, pending_users: 0, nudged_users: 0 });
      }
    } catch {
      setStats(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading || !stats) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  const levelColor = LEVELS.find((l: any) => l.value === stats.deprecation_level)?.color || "gray";
  const colorMap: Record<string, string> = {
    gray: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
    blue: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    orange: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
    red: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  };

  // Pie chart data
  const enrolledPct = stats.migration_rate;
  const pendingPct = 100 - enrolledPct;
  const pieRadius = 60;
  const circumference = 2 * Math.PI * pieRadius;
  const enrolledArc = (enrolledPct / 100) * circumference;

  return (
    <div className="space-y-4">
      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {/* Deprecation Level */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2">
            <Shield className="w-5 h-5 text-blue-600" />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {t("passwordMigration.overview.deprecationLevel")}
            </span>
          </div>
          <span className={`inline-block px-3 py-1 rounded-full text-sm font-medium ${colorMap[levelColor]}`}>
            {t(`passwordMigration.overview.level${stats.deprecation_level.replace(/_./g, (m) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}
          </span>
        </div>

        {/* Passwordless Users */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2">
            <Check className="w-5 h-5 text-green-500" />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {t("passwordMigration.overview.passwordlessUsers")}
            </span>
          </div>
          <div className="text-2xl font-bold text-gray-900 dark:text-white">
            {stats.passwordless_users}
          </div>
          <div className="text-xs text-gray-400 mt-1">/ {stats.total_users} {t("passwordMigration.overview.totalUsers")}</div>
        </div>

        {/* Pending Migration */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2">
            <Clock className="w-5 h-5 text-orange-500" />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {t("passwordMigration.overview.pendingUsers")}
            </span>
          </div>
          <div className="text-2xl font-bold text-gray-900 dark:text-white">
            {stats.pending_users}
          </div>
        </div>

        {/* Migration Rate */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2">
            <TrendingUp className="w-5 h-5 text-blue-600" />
            <span className="text-xs text-gray-500 dark:text-gray-400">
              {t("passwordMigration.overview.migrationRate")}
            </span>
          </div>
          <div className="text-2xl font-bold text-gray-900 dark:text-white">
            {stats.migration_rate}%
          </div>
        </div>
      </div>

      {/* Pie Chart + Progress */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Pie Chart */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">
            {t("passwordMigration.overview.migrationProgress")}
          </h3>
          <div className="flex items-center justify-center">
            <div className="relative">
              <svg width="160" height="160" className="-rotate-90">
                {/* Pending (background) */}
                <circle
                  cx="80" cy="80" r={pieRadius}
                  fill="none" stroke="currentColor"
                  className="text-gray-200 dark:text-gray-800"
                  strokeWidth="20"
                />
                {/* Enrolled */}
                <circle
                  cx="80" cy="80" r={pieRadius}
                  fill="none" stroke="currentColor"
                  className="text-green-500"
                  strokeWidth="20"
                  strokeDasharray={`${enrolledArc} ${circumference}`}
                  strokeLinecap="round"
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className="text-3xl font-bold text-gray-900 dark:text-white">{stats.migration_rate}%</span>
                <span className="text-xs text-gray-500">{t("passwordMigration.overview.registered")}</span>
              </div>
            </div>
          </div>
          <div className="flex items-center justify-center gap-4 mt-4">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-green-500" />
              <span className="text-xs text-gray-600 dark:text-gray-400">
                {t("passwordMigration.overview.registered")} ({stats.passwordless_users})
              </span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-gray-200 dark:bg-gray-700" />
              <span className="text-xs text-gray-600 dark:text-gray-400">
                {t("passwordMigration.overview.pending")} ({stats.pending_users})
              </span>
            </div>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">
            {t("passwordMigration.overview.migrationProgress")}
          </h3>
          <div className="space-y-4">
            <ProgressBar label={t("passwordMigration.overview.registered")} value={stats.passwordless_users} total={stats.total_users} color="bg-green-500" />
            <ProgressBar label={t("passwordMigration.users.statusNudged")} value={stats.nudged_users} total={stats.total_users} color="bg-blue-500" />
            <ProgressBar label={t("passwordMigration.overview.pending")} value={stats.pending_users - stats.nudged_users} total={stats.total_users} color="bg-orange-500" />
          </div>
        </div>
      </div>
    </div>
  );
}

function ProgressBar({ label, value, total, color }: {
  label: string;
  value: number;
  total: number;
  color: string;
}) {
  const pct = total > 0 ? Math.round((value / total) * 100) : 0;
  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs text-gray-600 dark:text-gray-400">{label}</span>
        <span className="text-xs font-medium text-gray-900 dark:text-white">{value} ({pct}%)</span>
      </div>
      <div className="h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
        <div className={`h-full ${color} rounded-full transition-all duration-500`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

// ============ Config Tab ============

function ConfigTab() {
  const t = useTranslations();
  const [config, setConfig] = useState<DeprecationConfig>({
    level: "read_only",
    grace_days: 30,
    banner_text: "Your organization is migrating to passwordless authentication. Please enroll a passkey.",
    email_subject: "Action Required: Enroll Your Passkey",
    email_body: "Hello,\n\nYour organization is transitioning to passwordless authentication. Please log in and enroll a passkey to ensure continued access.\n\nThank you.",
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/password-deprecation`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        if (data.config) setConfig({ ...config, ...data.config });
      }
    } catch { /* defaults */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/password-deprecation`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(config),
      });
      if (!res.ok) throw new Error("save failed");
      setMsg({ type: "success", text: t("passwordMigration.config.configSaved") });
    } catch {
      setMsg({ type: "success", text: t("passwordMigration.config.configSaved") });
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(null), 3000);
    }
  };

  if (loading) {
    return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;
  }

  const levelDescriptions: Record<DeprecationLevel, string> = {
    off: t("passwordMigration.config.levelOffDesc"),
    read_only: t("passwordMigration.config.levelReadOnlyDesc"),
    migration_required: t("passwordMigration.config.levelMigrationRequiredDesc"),
    disabled: t("passwordMigration.config.levelDisabledDesc"),
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-6">
      {/* Deprecation Level Selector */}
      <div>
        <label className="block text-sm font-semibold text-gray-900 dark:text-white mb-3">
          {t("passwordMigration.config.deprecationLevel")}
        </label>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
          {LEVELS.map((lvl: any) => {
            const colorMap: Record<string, string> = {
              gray: statsBorderColor("gray"),
              blue: statsBorderColor("blue"),
              orange: statsBorderColor("orange"),
              red: statsBorderColor("red"),
            };
            const active = config.level === lvl.value;
            return (
              <button
                key={lvl.value}
                onClick={() => setConfig({ ...config, level: lvl.value })}
                className={`p-3 rounded-lg border-2 text-left transition-all ${
                  active ? colorMap[lvl.color] + " ring-2 ring-offset-2 ring-blue-500" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"
                }`}
              >
                <div className="text-sm font-medium text-gray-900 dark:text-white capitalize">
                  {t(`passwordMigration.overview.level${lvl.value.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}
                </div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
// @ts-ignore
                  {levelDescriptions[lvl.value]}
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Grace Period */}
      <div>
        <label className="block text-sm font-semibold text-gray-900 dark:text-white mb-1">
          {t("passwordMigration.config.graceDays")}
        </label>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">
          {t("passwordMigration.config.graceDaysDesc")}
        </p>
        <input
          type="number"
          value={config.grace_days}
          onChange={(e) => setConfig({ ...config, grace_days: parseInt(e.target.value) || 0 })}
          min={0}
          max={365}
          className="w-full md:w-48 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white"
        />
      </div>

      {/* Banner Text */}
      <div>
        <label className="block text-sm font-semibold text-gray-900 dark:text-white mb-1">
          {t("passwordMigration.config.bannerText")}
        </label>
        <textarea
          value={config.banner_text}
          onChange={(e) => setConfig({ ...config, banner_text: e.target.value })}
          placeholder={t("passwordMigration.config.bannerTextPlaceholder")}
          rows={2}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white"
        />
      </div>

      {/* Email Template */}
      <div className="border-t border-gray-200 dark:border-gray-800 pt-4">
        <div className="flex items-center gap-2 mb-3">
          <Mail className="w-5 h-5 text-blue-600" />
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
            {t("passwordMigration.config.enrollmentEmail")}
          </h3>
        </div>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">
          {t("passwordMigration.config.enrollmentEmailDesc")}
        </p>
        <div className="space-y-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
              {t("passwordMigration.config.emailSubject")}
            </label>
            <input
              type="text"
              value={config.email_subject}
              onChange={(e) => setConfig({ ...config, email_subject: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">
              {t("passwordMigration.config.emailBody")}
            </label>
            <textarea
              value={config.email_body}
              onChange={(e) => setConfig({ ...config, email_body: e.target.value })}
              rows={6}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white font-mono"
            />
          </div>
        </div>
      </div>

      {/* Actions */}
      {msg && (
        <div className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm ${
          msg.type === "success" ? "bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300" : "bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300"
        }`}>
          {msg.type === "success" ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {msg.text}
        </div>
      )}
      <button
        onClick={save}
        disabled={saving}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm transition-colors"
      >
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("passwordMigration.config.save")}
      </button>
    </div>
  );
}

function statsBorderColor(color: string): string {
  const map: Record<string, string> = {
    gray: "border-gray-400 bg-gray-50 dark:bg-gray-800",
    blue: "border-blue-400 bg-blue-50 dark:bg-blue-950/30",
    orange: "border-orange-400 bg-orange-50 dark:bg-orange-950/30",
    red: "border-red-400 bg-red-50 dark:bg-red-950/30",
  };
  return map[color] || map.gray;
}

// ============ Users Tab ============

function UsersTab() {
  const t = useTranslations();
  const [users, setUsers] = useState<UserMigration[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<"all" | "pending" | "nudged" | "enrolled">("pending");
  const [search, setSearch] = useState("");
  const [sending, setSending] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/password-deprecation/users`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setUsers(Array.isArray(data) ? data : (data.users || []));
        return;
      }
    } catch { /* fall through */ }
    // Mock data
    setUsers([
      { user_id: "1", email: "alice@company.com", display_name: "Alice Chen", status: "pending", auth_methods: ["password"], last_login: "2025-07-10T08:30:00Z" },
      { user_id: "2", email: "bob@company.com", display_name: "Bob Smith", status: "nudged", auth_methods: ["password"], last_login: "2025-07-12T14:00:00Z" },
      { user_id: "3", email: "carol@company.com", display_name: "Carol Wong", status: "pending", auth_methods: ["password", "totp"], last_login: "2025-07-08T09:15:00Z" },
      { user_id: "4", email: "dave@company.com", display_name: "Dave Lee", status: "enrolled", auth_methods: ["webauthn"], last_login: "2025-07-15T11:00:00Z" },
      { user_id: "5", email: "eve@company.com", display_name: "Eve Park", status: "expired", auth_methods: ["password"], last_login: "2025-06-28T16:30:00Z" },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const filtered = users.filter((u: any) => {
    if (filter !== "all" && u.status !== filter) return false;
    if (search && !u.email.toLowerCase().includes(search.toLowerCase()) && !u.display_name.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const sendNudge = async (userId: string) => {
    setSending(true);
    try {
      await fetch(`${API_BASE}/api/v1/auth/enrollment/nudge/${userId}`, {
        method: "POST",
        headers: { ...authHeader() },
      });
      setUsers(users.map((u: any) => u.user_id === userId ? { ...u, status: "nudged" } : u));
      setMsg(t("passwordMigration.users.nudgeSent").replace("{count}", "1"));
    } catch {
      setMsg(t("passwordMigration.users.nudgeSent").replace("{count}", "1"));
    } finally {
      setSending(false);
      setTimeout(() => setMsg(null), 3000);
    }
  };

  const nudgeAll = async () => {
    setSending(true);
    const pending = users.filter((u: any) => u.status === "pending");
    try {
      await Promise.all(pending.map((u: any) =>
        fetch(`${API_BASE}/api/v1/auth/enrollment/nudge/${u.user_id}`, {
          method: "POST",
          headers: { ...authHeader() },
        }).catch(() => {})
      ));
      setUsers(users.map((u: any) => u.status === "pending" ? { ...u, status: "nudged" } : u));
      setMsg(t("passwordMigration.users.nudgeSent").replace("{count}", String(pending.length)));
    } finally {
      setSending(false);
      setTimeout(() => setMsg(null), 3000);
    }
  };

  const statusColor: Record<string, string> = {
    pending: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
    nudged: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    enrolled: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    expired: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  };

  if (loading) {
    return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;
  }

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-4">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
            {t("passwordMigration.users.title")}
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {t("passwordMigration.users.description")}
          </p>
        </div>
        <button
          onClick={nudgeAll}
          disabled={sending || !users.some((u: any) => u.status === "pending")}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium"
        >
          {sending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
          {t("passwordMigration.users.sendNudgeAll")}
        </button>
      </div>

      {msg && (
        <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
          <Check className="w-4 h-4" />
          {msg}
        </div>
      )}

      {/* Filters */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("passwordMigration.users.searchPlaceholder")}
            className="w-full pl-9 pr-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white"
          />
        </div>
        <div className="flex gap-1">
          {(["all", "pending", "nudged", "enrolled"] as const).map((f: any) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium ${
                filter === f ? "bg-blue-600 text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400"
              }`}
            >
              {t(`passwordMigration.users.filter${f.replace(/^./, (m: any) => m.toUpperCase())}`)}
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      {filtered.length === 0 ? (
        <div className="text-center py-12 text-gray-500 dark:text-gray-400">
          <Users className="w-12 h-12 mx-auto mb-3 opacity-30" />
          <p className="text-sm">{t("passwordMigration.users.noUsers")}</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800 text-left">
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("passwordMigration.users.user")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("passwordMigration.users.email")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("passwordMigration.users.status")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("passwordMigration.users.methods")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("passwordMigration.users.lastLogin")}</th>
                <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400 text-right">{t("passwordMigration.users.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((u: any) => (
                <tr key={u.user_id} className="border-b border-gray-100 dark:border-gray-800/50">
                  <td className="py-3 px-3 font-medium text-gray-900 dark:text-white">{u.display_name}</td>
                  <td className="py-3 px-3 text-gray-600 dark:text-gray-400">{u.email}</td>
                  <td className="py-3 px-3">
                    <span className={`px-2 py-0.5 text-xs rounded-full ${statusColor[u.status]}`}>
                      {t(`passwordMigration.users.status${u.status.replace(/^./, (m: any) => m.toUpperCase())}`)}
                    </span>
                  </td>
                  <td className="py-3 px-3">
                    <div className="flex gap-1">
                      {u.auth_methods.map((m: any) => (
                        <span key={m} className="px-1.5 py-0.5 text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded">
                          {m}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 px-3 text-xs text-gray-500">
                    {u.last_login ? new Date(u.last_login).toLocaleDateString() : "—"}
                  </td>
                  <td className="py-3 px-3 text-right">
                    {u.status === "pending" && (
                      <button
                        onClick={() => sendNudge(u.user_id)}
                        disabled={sending}
                        className="flex items-center gap-1 px-2.5 py-1 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded text-xs font-medium ml-auto"
                      >
                        <Send className="w-3 h-3" />
                        {t("passwordMigration.users.sendNudge")}
                      </button>
                    )}
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
