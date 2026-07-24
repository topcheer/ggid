"use client";

import { useState, useEffect, useRef, useMemo } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Users, Shield, Building2, ScrollText, KeyRound, Settings,
  LayoutDashboard, Webhook, Sun, Moon, Globe, Server, Send,
  Monitor, BookOpen, TrendingUp, Bot, FileCheck, Cloud, Network,
  RefreshCw, X, Menu, AlertCircle, Loader2, LogOut, Gauge, Radar,
  Fingerprint, Share2, ArrowUpCircle, Search, ChevronDown, ChevronRight,
  Activity, PieChart, Database, Lock, Crown, Zap, ShieldCheck,
  Scroll, FileText, Terminal, Building, HelpCircle, AlertTriangle,
  Layers, Grid3x3, CalendarClock, GitBranch, ExternalLink,
  Rocket, Info,
} from "lucide-react";
import { useTheme } from "@/lib/theme";
import { useI18n } from "@/lib/i18n";
import { LanguageSwitcher } from "@/components/LanguageSwitcher";
import { checkApiHealthDetailed, type HealthResult } from "@/lib/api-config";
import { useUserRole, useUserPermissions, NAV_PERMISSION_MAP } from "@/lib/api";
import { CommandPalette } from "@/components/CommandPalette";
import { TenantSwitcher } from "@/components/TenantSwitcher";

type LucideIcon = typeof Shield;

interface NavItem {
  href: string; label: string; icon: LucideIcon;
  requiredScope?: string;
}
interface NavGroup {
  label: string; icon: LucideIcon; items: NavItem[];
  requiredScope?: string;
}

export function Sidebar() {
  const pathname = usePathname();
  const { mode, toggle } = useTheme();
  const { t } = useI18n();
  const { isPlatformAdmin, isTenantAdmin, scopes } = useUserRole();
  const { hasPermission } = useUserPermissions();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set());
  const [health, setHealth] = useState<HealthResult | null>(null);
  const [reconnecting, setReconnecting] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const doHealthCheck = async () => {
    const result = await checkApiHealthDetailed();
    setHealth(result);
    if (result.online && reconnecting) setReconnecting(false);
    if (!result.online) setReconnecting(true);
    scheduleNext(result.online);
  };
  const scheduleNext = (online: boolean) => {
    if (intervalRef.current) clearTimeout(intervalRef.current);
    intervalRef.current = setTimeout(doHealthCheck, online ? 30000 : 5000);
  };
  useEffect(() => { doHealthCheck(); return () => { if (intervalRef.current) clearTimeout(intervalRef.current); }; /* eslint-disable-next-line */ }, []);

  const navGroups: NavGroup[] = useMemo(() => [
    {
      label: "Overview", icon: LayoutDashboard, items: [
        { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
        { href: "/sessions", label: "My Sessions", icon: Monitor },
        { href: "/access-requests", label: "Access Requests", icon: FileCheck },
      ],
    },
    {
      label: "Identity", icon: Users, requiredScope: "manager", items: [
        { href: "/users", label: "Users", icon: Users },
        { href: "/roles", label: "Roles", icon: Shield },
        { href: "/organizations", label: "Organizations", icon: Building2 },
        { href: "/settings/import-wizard", label: "Import", icon: Cloud },
        { href: "/settings/review-schedules", label: "Access Reviews", icon: CalendarClock },
      ],
    },
    {
      label: "Security", icon: ShieldCheck, requiredScope: "manager", items: [
        { href: "/security/itdr", label: "ITDR Dashboard", icon: Shield },
        { href: "/security/session-detail", label: "Sessions", icon: Monitor },
        { href: "/security/cae-monitor", label: "CAE Monitor", icon: Activity },
        { href: "/security/risk-score", label: "Risk Score", icon: Gauge },
        { href: "/security/posture", label: "Posture", icon: ShieldCheck },
        { href: "/settings/conditional-access", label: "Conditional Access", icon: Shield },
        { href: "/settings/password-policy", label: "Password Policy", icon: KeyRound },
        { href: "/settings/mfa", label: "MFA", icon: Fingerprint },
        { href: "/settings/passkey-management", label: "Passkeys", icon: Fingerprint },
      ],
    },
    {
      label: "Audit", icon: ScrollText, requiredScope: "manager", items: [
        { href: "/audit", label: "Audit Log", icon: ScrollText },
        { href: "/policies", label: "Policies", icon: Shield },
      ],
    },
    {
      label: "Applications", icon: Settings, requiredScope: "manager", items: [
        { href: "/oauth-clients", label: "OAuth Clients", icon: KeyRound },
        { href: "/webhooks", label: "Webhooks", icon: Webhook },
        { href: "/api-keys", label: "API Keys", icon: KeyRound },
        { href: "/settings/scim", label: "SCIM", icon: BookOpen },
        { href: "/settings/ldap-config", label: "LDAP", icon: Network },
        { href: "/settings/rate-limits", label: "Rate Limits", icon: Gauge },
        { href: "/api-explorer", label: "API Explorer", icon: Terminal },
      ],
    },
    {
      label: "Platform", icon: Building, requiredScope: "admin", items: [
        { href: "/admin/tenants", label: "Tenants", icon: Building2 },
        { href: "/admin/audit/global", label: "Global Audit", icon: FileText },
        { href: "/admin/threats", label: "Threat Dashboard", icon: Shield },
        { href: "/settings/branding", label: "Branding", icon: Globe },
        { href: "/settings/feature-flags", label: "Feature Flags", icon: Zap },
      ],
    },
  ], [t]);

  // Filter nav groups/items by user role + dynamic permissions
  const roleFilteredGroups = useMemo(() => {
    const hasScope = (required?: string) => {
      if (!required) return true;
      if (isPlatformAdmin) return true;
      if (required === "manager") return isTenantAdmin;
      if (required === "admin") return false;
      return scopes.includes(required);
    };

    const hasRoutePermission = (href: string) => {
      const requiredPerms = NAV_PERMISSION_MAP[href];
      if (!requiredPerms || requiredPerms.length === 0) return true; // No permission needed
      return requiredPerms.every(p => hasPermission(p));
    };

    return navGroups
      .filter((g) => hasScope(g.requiredScope))
      .map((g) => ({
        ...g,
        items: g.items.filter((i) => hasScope(i.requiredScope) && hasRoutePermission(i.href)),
      }))
      .filter((g) => g.items.length > 0);
  }, [navGroups, isPlatformAdmin, isTenantAdmin, scopes, hasPermission]);

  // Search filter
  const filtered = search.trim()
    ? roleFilteredGroups.map((g) => ({ ...g, items: g.items.filter((i) => i.label.toLowerCase().includes(search.toLowerCase())) })).filter((g) => g.items.length > 0)
    : roleFilteredGroups;

  const toggleGroup = (label: string) => {
    const next = new Set(collapsedGroups);
    if (next.has(label)) next.delete(label); else next.add(label);
    setCollapsedGroups(next);
  };

  // Auto-expand groups when searching
  const isGroupCollapsed = (label: string) => search.trim() ? false : collapsedGroups.has(label);

  return (
    <>
      {/* Mobile hamburger */}
      {!mobileOpen && (
        <button onClick={() => setMobileOpen(true)} className="fixed left-4 top-4 z-50 rounded-lg border border-gray-200 bg-white p-2 shadow-md dark:border-gray-700 dark:bg-gray-800 md:hidden">
          <Menu className="h-5 w-5" />
        </button>
      )}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 bg-black/40 md:hidden" onClick={() => setMobileOpen(false)} />
      )}

      <aside className={`${mobileOpen ? "flex" : "hidden"} md:flex flex-col border-r border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900`} style={{ width: "260px" }}>
        {/* Header */}
        <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6 dark:border-gray-800">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-white font-bold">G</div>
          <span className="text-lg font-semibold text-gray-900 dark:text-gray-100">GGID</span>
          <button onClick={() => setMobileOpen(false)} className="ml-auto md:hidden text-gray-400"><X className="h-5 w-5" /></button>
        </div>

        {/* Search + Command Palette */}
        <div className="p-3 border-b border-gray-100 dark:border-gray-800 space-y-2">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t("nav.searchPlaceholder")}
              aria-label={t("nav.searchPlaceholder")}
              role="searchbox"
              className="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
          </div>
          <CommandPalette />
        </div>

        {/* Nav */}
        <nav className="flex-1 overflow-y-auto p-3 space-y-1" aria-label="Main navigation">
          {filtered.length === 0 && <p className="text-xs text-gray-400 px-3 py-4 text-center">{t("nav.searchNoResults")}</p>}
          {filtered.map((group) => {
            const GroupIcon = group.icon;
            const collapsed = isGroupCollapsed(group.label);
            return (
              <div key={group.label}>
                <button onClick={() => toggleGroup(group.label)} className="flex items-center gap-2 w-full px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300">
                  {collapsed ? <ChevronRight className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                  <GroupIcon className="w-3.5 h-3.5" />
                  {group.label}
                </button>
                {!collapsed && (
                  <div className="space-y-0.5 mt-0.5">
                    {group.items.map((item) => {
                      const Icon = item.icon;
                      const active = pathname === item.href || (item.href !== "/" && item.href !== "/dashboard" && pathname.startsWith(item.href));
                      return (
                        <Link key={item.href} href={item.href}
                          className={`flex items-center gap-3 rounded-lg px-3 py-1.5 text-sm transition-colors ${
                            active ? "bg-brand-50 text-brand-700 font-medium dark:bg-brand-900/30 dark:text-brand-400" : "text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
                          }`}>
                          <Icon className="h-4 w-4 shrink-0" />
                          <span className="truncate">{item.label}</span>
                        </Link>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
        </nav>

        {/* Footer: theme + locale + health + user */}
        <div className="border-t border-gray-200 p-3 dark:border-gray-800">
          <div className="flex items-center gap-2 mb-2">
            <button onClick={toggle} className="flex h-8 w-8 items-center justify-center rounded-lg border border-gray-200 text-gray-500 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-800" title={`Theme: ${mode}`}>
              {mode === "light" ? <Sun className="h-4 w-4" /> : mode === "dark" ? <Moon className="h-4 w-4" /> : <Monitor className="h-4 w-4" />}
            </button>
            <LanguageSwitcher compact />
            {isPlatformAdmin && <TenantSwitcher />}
            <div className="flex-1" />
            {/* Help dropdown */}
            <HelpDropdown t={t} />
          </div>

          {/* Health */}
          <div className="flex items-center gap-2 text-xs mb-2 px-1" title={health?.online ? `Gateway: ${health.latencyMs ?? "?"}ms` : "Offline"}>
            {reconnecting ? <Loader2 className="h-3 w-3 animate-spin text-amber-500" /> : health?.online ? <span className="inline-block h-2 w-2 rounded-full bg-green-500 animate-pulse" /> : <AlertCircle className="h-3 w-3 text-red-500" />}
            <span className="text-gray-500 dark:text-gray-400">{health?.online ? `API: ${health.latencyMs ?? "?"}ms` : t("sidebar.offline")}</span>
          </div>

          {/* User */}
          <div className="flex items-center gap-3 pt-2 border-t border-gray-100 dark:border-gray-800">
            <Link href="/profile" className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600">A</Link>
            <div className="flex-1 min-w-0">
              <Link href="/profile" className="block truncate text-sm font-medium text-gray-900 hover:text-blue-600 dark:text-gray-200 dark:hover:text-blue-400">admin@ggid.dev</Link>
              <p className="truncate text-xs text-gray-500">{t("sidebar.administrator")}</p>
            </div>
            <button onClick={() => { localStorage.removeItem("ggid_access_token"); localStorage.removeItem("ggid_refresh_token"); localStorage.removeItem("ggid_tenant_id"); window.location.href = "/login"; }}
              className="flex-shrink-0 rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-red-600 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-red-400" title="Sign Out">
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}

// ============ Help Dropdown ============

const BUILD_VERSION = "v1.0-stable";
const BUILD_DATE = new Date().toISOString().split("T")[0];
const COMMIT_HASH = process.env.NEXT_PUBLIC_GIT_SHA?.slice(0, 7) || "dev";

function HelpDropdown({ t }: { t: (key: string) => string }) {
  const [open, setOpen] = useState(false);

  const items = [
    { label: t("nav.helpQuickStart"), icon: Rocket, href: "/docs" },
    { label: t("nav.helpApiDocs"), icon: BookOpen, href: "/docs" },
    { label: t("nav.helpSwagger"), icon: Terminal, href: "/docs/swagger", external: false },
    { label: t("nav.helpGithubIssues"), icon: ExternalLink, href: "https://github.com/topcheer/ggid/issues", external: true },
  ];

  return (
    <div className="relative">
      <button onClick={() => setOpen(!open)}
        className="flex h-8 w-8 items-center justify-center rounded-lg text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800"
        title={t("nav.helpMenu")}>
        {open ? <ChevronDown className="h-4 w-4" /> : <HelpCircle className="h-4 w-4" />}
      </button>

      {open && (
        <>
          {/* Backdrop */}
          <div className="fixed inset-0 z-40" onClick={() => setOpen(false)} />

          {/* Dropdown */}
          <div className="absolute bottom-10 right-0 z-50 w-56 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 shadow-xl py-1">
            <div className="px-3 py-2 border-b border-gray-100 dark:border-gray-800">
              <span className="text-xs font-semibold uppercase tracking-wider text-gray-400">{t("nav.helpMenu")}</span>
            </div>

            {items.map((item, i) => {
              const Icon = item.icon;
              return item.external ? (
                <a key={i} href={item.href} target="_blank" rel="noopener noreferrer"
                  className="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800"
                  onClick={() => setOpen(false)}>
                  <Icon className="w-4 h-4 text-gray-400" />{item.label}
                </a>
              ) : (
                <Link key={i} href={item.href}
                  className="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-800"
                  onClick={() => setOpen(false)}>
                  <Icon className="w-4 h-4 text-gray-400" />{item.label}
                </Link>
              );
            })}

            {/* Version info */}
            <div className="px-3 py-2 border-t border-gray-100 dark:border-gray-800 mt-1">
              <div className="flex items-center gap-2 text-xs text-gray-400">
                <Info className="w-3 h-3" />
                <span>{BUILD_VERSION}</span>
                <span className="font-mono text-gray-300 dark:text-gray-600">{COMMIT_HASH}</span>
              </div>
              <div className="text-xs text-gray-400 mt-0.5 pl-5">
                {t("nav.helpBuild")} {BUILD_DATE}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
