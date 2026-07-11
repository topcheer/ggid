"use client";

import { useState, useEffect } from "react";
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
} from "lucide-react";
import { useTheme } from "@/lib/theme";
import { useI18n } from "@/lib/i18n";
import { checkApiHealth, API_BASE_URL } from "@/lib/api-config";

export function Sidebar() {
  const pathname = usePathname();
  const { mode, toggle } = useTheme();
  const { locale, setLocale, t } = useI18n();
  const [collapsed, setCollapsed] = useState(false);
  const [apiOnline, setApiOnline] = useState<boolean | null>(null);

  useEffect(() => {
    const check = () => checkApiHealth().then(setApiOnline);
    check();
    const interval = setInterval(check, 30000);
    return () => clearInterval(interval);
  }, []);

  const navItems = [
    { href: "/", label: t("nav.dashboard"), icon: LayoutDashboard },
    { href: "/users", label: t("nav.users"), icon: Users },
    { href: "/roles", label: t("nav.roles"), icon: Shield },
    { href: "/policies", label: "Policies", icon: Shield },
    { href: "/organizations", label: t("nav.organizations"), icon: Building2 },
    { href: "/audit", label: t("nav.audit"), icon: ScrollText },
    { href: "/oauth-clients", label: t("nav.oauthClients"), icon: KeyRound },
    { href: "/webhooks", label: t("nav.webhooks"), icon: Webhook },
    { href: "/sessions", label: "Sessions", icon: Monitor },
    { href: "/scim", label: "SCIM", icon: BookOpen },
    { href: "/organizations/analytics", label: "Org Analytics", icon: TrendingUp },
    { href: "/monitoring", label: "Monitoring", icon: Server },
    { href: "/api-explorer", label: "API Explorer", icon: Send },
    { href: "/settings", label: t("nav.settings"), icon: Settings },
  ];

  return (
    <>
      {collapsed && (
        <button
          onClick={() => setCollapsed(false)}
          className="fixed left-4 top-4 z-50 rounded-lg border border-gray-200 bg-white p-2 shadow-md dark:border-gray-700 dark:bg-gray-800 md:hidden"
        >
          <Menu className="h-5 w-5" />
        </button>
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

      <nav className="flex-1 space-y-1 p-3">
        {navItems.map((item) => {
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
              <Icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
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
        {/* API Health Indicator */}
        <div className="mb-3 flex items-center gap-2 text-xs">
          <span className={`inline-block h-2 w-2 rounded-full ${apiOnline === null ? "bg-gray-400" : apiOnline ? "bg-green-500" : "bg-red-500"}`} />
          <span className="text-gray-500 dark:text-gray-400">API: {apiOnline === null ? "Checking..." : apiOnline ? "Connected" : "Offline"}</span>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-sm font-medium dark:bg-gray-700 dark:text-gray-300">
            A
          </div>
          <div className="flex-1 min-w-0">
            <p className="truncate text-sm font-medium text-gray-900 dark:text-gray-200">admin@ggid.dev</p>
            <p className="truncate text-xs text-gray-500 dark:text-gray-500">Administrator</p>
          </div>
        </div>
      </div>
    </aside>
    </>
  );
}
