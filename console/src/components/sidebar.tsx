"use client";

import { useState, useEffect, useRef } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Users,
  Shield,
  Building2,
  ScrollText,
  KeyRound,
  Settings,
  LayoutDashboard,
  Webhook,
  Sun,
  Moon,
  Globe,
  Server,
  Send,
  Monitor,
  BookOpen,
  TrendingUp,
  X,
  Menu,
  AlertCircle,
  Loader2,
} from "lucide-react";
import { useTheme } from "@/lib/theme";
import { useI18n } from "@/lib/i18n";
import { checkApiHealthDetailed, type HealthResult } from "@/lib/api-config";

export function Sidebar() {
  const pathname = usePathname();
  const { mode, toggle } = useTheme();
  const { locale, setLocale, t } = useI18n();
  const [collapsed, setCollapsed] = useState(false);
  const [health, setHealth] = useState<HealthResult | null>(null);
  const [reconnecting, setReconnecting] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const doHealthCheck = async () => {
    const result = await checkApiHealthDetailed();
    setHealth(result);
    // If we were reconnecting and now we're back online, clear the flag
    if (result.online && reconnecting) setReconnecting(false);
    // If offline, mark reconnecting and switch to fast polling (5s)
    if (!result.online) setReconnecting(true);
    scheduleNext(result.online);
  };

  const scheduleNext = (online: boolean) => {
    if (intervalRef.current) clearTimeout(intervalRef.current);
    // Online: poll every 30s. Offline: retry every 5s.
    const delay = online ? 30000 : 5000;
    intervalRef.current = setTimeout(doHealthCheck, delay);
  };

  useEffect(() => {
    doHealthCheck();
    return () => {
      if (intervalRef.current) clearTimeout(intervalRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const navGroups = [
    {
      label: "Overview",
      items: [
        { href: "/", label: t("nav.dashboard"), icon: LayoutDashboard },
      ],
    },
    {
      label: "Management",
      items: [
        { href: "/users", label: t("nav.users"), icon: Users },
        { href: "/roles", label: t("nav.roles"), icon: Shield },
        { href: "/policies", label: t("nav.policies"), icon: Shield },
        { href: "/organizations", label: t("nav.organizations"), icon: Building2 },
        { href: "/organizations/analytics", label: t("nav.orgAnalytics"), icon: TrendingUp },
      ],
    },
    {
      label: "Security",
      items: [
        { href: "/audit", label: t("nav.audit"), icon: ScrollText },
        { href: "/oauth-clients", label: t("nav.oauthClients"), icon: KeyRound },
        { href: "/webhooks", label: t("nav.webhooks"), icon: Webhook },
        { href: "/sessions", label: t("nav.sessions"), icon: Monitor },
        { href: "/scim", label: t("nav.scim"), icon: BookOpen },
      ],
    },
    {
      label: "System",
      items: [
        { href: "/monitoring", label: t("nav.monitoring"), icon: Server },
        { href: "/api-explorer", label: t("nav.apiExplorer"), icon: Send },
        { href: "/settings", label: t("nav.settings"), icon: Settings },
      ],
    },
  ];

  return (
    <>
      {collapsed && (
        <>
          <button
            onClick={() => setCollapsed(false)}
            className="fixed left-4 top-4 z-50 rounded-lg border border-gray-200 bg-white p-2 shadow-md dark:border-gray-700 dark:bg-gray-800 md:hidden"
          >
            <Menu className="h-5 w-5" />
          </button>
          {/* Mobile backdrop */}
          <div
            className="fixed inset-0 z-40 bg-black/40 md:hidden"
            onClick={() => setCollapsed(false)}
          />
        </>
      )}
    <aside
      className={`${collapsed ? "hidden" : "flex"} flex-col border-r border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900 md:flex lg:flex`}
      style={{ width: "var(--sidebar-width)" }}
    >
      <div className="flex h-16 items-center gap-2 border-b border-gray-200 px-6 dark:border-gray-800">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-white font-bold">
          G
        </div>
        <span className="text-lg font-semibold text-gray-900 dark:text-gray-100">GGID</span>
        <button onClick={() => setCollapsed(true)} className="ml-auto md:hidden text-gray-400">
          <X className="h-5 w-5" />
        </button>
      </div>

      <nav className="flex-1 space-y-3 overflow-y-auto p-3">
        {navGroups.map((group) => (
          <div key={group.label}>
            <p className="mb-1 px-3 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
              {group.label}
            </p>
            <div className="space-y-1">
              {group.items.map((item) => {
                const Icon = item.icon;
                const active =
                  pathname === item.href ||
                  (item.href !== "/" && pathname.startsWith(item.href));
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                      active
                        ? "bg-brand-50 text-brand-700 font-medium dark:bg-brand-900/30 dark:text-brand-400"
                        : "text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
                    }`}
                  >
                    <Icon className="h-4 w-4 shrink-0" />
                    <span className="truncate">{item.label}</span>
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* Controls: theme + locale */}
      <div className="border-t border-gray-200 p-3 dark:border-gray-800">
        <div className="flex items-center gap-2">
          <button
            onClick={toggle}
            className="flex h-8 w-8 items-center justify-center rounded-lg border border-gray-200 text-gray-500 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-800"
            title={`Theme: ${mode} (click to cycle)`}
          >
            {mode === "light" ? (
              <Sun className="h-4 w-4" />
            ) : mode === "dark" ? (
              <Moon className="h-4 w-4" />
            ) : (
              <Monitor className="h-4 w-4" />
            )}
          </button>
          <button
            onClick={() => setLocale(locale === "en" ? "zh" : "en")}
            className="flex h-8 items-center gap-1 rounded-lg border border-gray-200 px-2 text-xs font-medium text-gray-500 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-800"
            title="Switch language"
          >
            <Globe className="h-3.5 w-3.5" />
            {locale === "en" ? "EN" : "中"}
          </button>
        </div>
      </div>

      <div className="border-t border-gray-200 p-4 dark:border-gray-800">
        {/* API Health Indicator with latency tooltip */}
        <div
          className="mb-3 flex items-center gap-2 text-xs"
          title={
            health?.online
              ? `Gateway: ${health.latencyMs ?? "?"}ms`
              : reconnecting
                ? "Reconnecting..."
                : "Offline"
          }
        >
          {reconnecting ? (
            <Loader2 className="h-3 w-3 animate-spin text-amber-500" />
          ) : health?.online ? (
            <span className="inline-block h-2 w-2 rounded-full bg-green-500 animate-pulse" />
          ) : (
            <AlertCircle className="h-3 w-3 text-red-500" />
          )}
          <span className="text-gray-500 dark:text-gray-400">
            {health === null
              ? t("sidebar.checking")
              : reconnecting
                ? t("sidebar.reconnecting")
                : health.online
                  ? `API: ${health.latencyMs ?? "?"}ms`
                  : t("sidebar.offline")}
          </span>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium dark:bg-gray-700 dark:text-gray-300">
            A
          </div>
          <div className="flex-1 min-w-0">
            <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-200">admin@ggid.dev</p>
            <p className="truncate text-xs text-gray-500 dark:text-gray-500">{t("sidebar.administrator")}</p>
          </div>
        </div>
      </div>
    </aside>
    </>
  );
}
