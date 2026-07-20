"use client";
import { useState, useMemo } from "react";
import Link from "next/link";
import {
  Settings, Shield, Lock, Globe, ArrowRightLeft, Bell, Gauge,
  Database, KeyRound, Server, FileText, Activity, Cpu, Zap,
  CheckCircle2, ChevronRight, Search, Fingerprint, Crown,
  Grid3x3, CalendarClock, Bot, Palette, Terminal, Building2,
  Rocket, Monitor, AlertTriangle, X,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { usePageTitle } from "@/lib/usePageTitle";

interface SettingsCard {
  href: string; icon: typeof Shield; label: string; desc: string;
  category: string; color: string; badge?: string; keywords: string[];
}

const categoryColors: Record<string, string> = {
  Security: "border-red-200 dark:border-red-900",
  Identity: "border-blue-200 dark:border-blue-900",
  Governance: "border-purple-200 dark:border-purple-900",
  Audit: "border-orange-200 dark:border-orange-900",
  Integration: "border-cyan-200 dark:border-cyan-900",
  System: "border-gray-200 dark:border-gray-700",
  Admin: "border-indigo-200 dark:border-indigo-900",
};

export default function SettingsHubPage() {
  const t = useTranslations();
  usePageTitle("Settings");
  const [search, setSearch] = useState("");
  const [focused, setFocused] = useState(false);

  const cards: SettingsCard[] = [
    // Security
    { href: "/settings/security", icon: Shield, label: "Security Center", desc: "Overall security posture and threats", category: "Security", color: "text-red-500", keywords: ["security", "threat", "risk", "posture"] },
    { href: "/settings/security-policy", icon: Lock, label: "Security Policy", desc: "Password, lockout, method policies", category: "Security", color: "text-red-500", keywords: ["password", "lockout", "policy", "mfa", "authentication"] },
    { href: "/settings/password-policy", icon: Lock, label: "Password Policy", desc: "Password requirements and expiration", category: "Security", color: "text-orange-500", keywords: ["password", "policy", "strength", "expiry"] },
    { href: "/settings/password-strength", icon: Fingerprint, label: "Password Strength", desc: "Real-time password evaluation", category: "Security", color: "text-orange-500", keywords: ["password", "strength", "zxcvbn", "score"] },
    { href: "/settings/password-migration", icon: KeyRound, label: "Password Migration", desc: "Passwordless transition dashboard", category: "Security", color: "text-orange-500", keywords: ["password", "migration", "passkey", "passwordless"] },
    { href: "/settings/conditional-access", icon: Shield, label: "Conditional Access", desc: "IF-THEN access policy builder", category: "Security", color: "text-red-500", keywords: ["conditional", "access", "policy", "mfa", "risk"] },
    { href: "/settings/rate-limits", icon: Zap, label: "Rate Limits", desc: "API and login rate limiting", category: "Security", color: "text-yellow-500", keywords: ["rate", "limit", "throttle", "api"] },
    { href: "/settings/enrollment-campaign", icon: Zap, label: "Enrollment Campaign", desc: "Passkey enrollment campaigns", category: "Security", color: "text-orange-500", keywords: ["passkey", "enrollment", "mfa", "campaign"] },
    // Identity
    { href: "/settings/nhi", icon: Bot, label: "NHI Inventory", desc: "Non-human identity management", category: "Identity", color: "text-blue-500", keywords: ["nhi", "service", "account", "machine", "api"] },
    { href: "/settings/migration", icon: Database, label: "Migration", desc: "Legacy system migration", category: "Identity", color: "text-blue-500", keywords: ["migration", "legacy", "import"] },
    { href: "/settings/import-wizard", icon: Server, label: "Import Wizard", desc: "Bulk import users from CSV/JSON", category: "Identity", color: "text-blue-500", keywords: ["import", "bulk", "csv", "json", "users"] },
    { href: "/settings/import-enhanced", icon: Rocket, label: "Enhanced Import", desc: "Import with dry-run and progress", category: "Identity", color: "text-blue-500", keywords: ["import", "dry-run", "progress", "bulk"] },
    { href: "/settings/import-monitor", icon: Activity, label: "Import Monitor", desc: "Monitor async import jobs", category: "Identity", color: "text-blue-500", keywords: ["import", "monitor", "job", "async"] },
    { href: "/settings/attribute-mapping", icon: Globe, label: "Attribute Mapping", desc: "Map IdP attributes to GGID fields", category: "Identity", color: "text-teal-500", keywords: ["attribute", "mapping", "idp", "ldap", "saml"] },
    { href: "/settings/review-schedules", icon: CalendarClock, label: "Review Schedules", desc: "Automate access certifications", category: "Identity", color: "text-blue-500", keywords: ["review", "access", "certification", "schedule"] },
    // Governance
    { href: "/settings/sod-matrix", icon: Grid3x3, label: "SoD Matrix", desc: "Separation of duties rules", category: "Governance", color: "text-purple-500", keywords: ["sod", "separation", "duties", "conflict", "role"] },
    { href: "/settings/delegations", icon: ArrowRightLeft, label: "Delegations", desc: "Manage access delegations", category: "Governance", color: "text-purple-500", keywords: ["delegation", "delegate", "access", "scope"] },
    { href: "/settings/platform-access", icon: Shield, label: "Platform Access", desc: "Grant/revoke platform admin access to your tenant", category: "Governance", color: "text-red-500", keywords: ["platform", "access", "consent", "impersonate", "support"] },
    { href: "/policies", icon: Shield, label: "Policies", desc: "Access policy management", category: "Governance", color: "text-purple-500", keywords: ["policy", "abac", "rbac", "access"] },
    { href: "/access-requests", icon: FileText, label: "Access Requests", desc: "Request and approve access", category: "Governance", color: "text-purple-500", keywords: ["access", "request", "approval"] },
    // Audit
    { href: "/audit", icon: FileText, label: "Audit Log", desc: "View audit events", category: "Audit", color: "text-orange-500", keywords: ["audit", "log", "event", "trail"] },
    { href: "/audit/explorer", icon: Search, label: "Audit Explorer", desc: "Search and export audit events", category: "Audit", color: "text-orange-500", keywords: ["audit", "search", "export", "filter"] },
    { href: "/audit/ccm", icon: CheckCircle2, label: "Compliance Monitor", desc: "Continuous compliance checks", category: "Audit", color: "text-emerald-500", keywords: ["compliance", "ccm", "dora", "hipaa", "sox"] },
    { href: "/settings/audit-export-center", icon: Database, label: "Audit Export", desc: "Export audit data", category: "Audit", color: "text-purple-500", keywords: ["audit", "export", "download"] },
    // Integration
    { href: "/settings/scim-provisioning", icon: ArrowRightLeft, label: "SCIM Provisioning", desc: "User provisioning via SCIM 2.0", category: "Integration", color: "text-cyan-500", badge: "SCIM 2.0", keywords: ["scim", "provisioning", "sync"] },
    { href: "/settings/webhooks", icon: Bell, label: "Webhooks", desc: "Event webhook configuration", category: "Integration", color: "text-pink-500", keywords: ["webhook", "event", "notification"] },
    { href: "/settings/integration-playground", icon: Terminal, label: "Integration Playground", desc: "Test your integration", category: "Integration", color: "text-cyan-500", keywords: ["test", "api", "token", "webhook", "playground"] },
    { href: "/settings/oauth-clients/new", icon: KeyRound, label: "Create OAuth Client", desc: "Register a new application", category: "Integration", color: "text-cyan-500", keywords: ["oauth", "client", "register", "app"] },
    // System
    { href: "/settings/branding-config", icon: Palette, label: "Branding", desc: "Customize Console appearance", category: "System", color: "text-purple-500", keywords: ["branding", "logo", "color", "theme", "customize"] },
    { href: "/settings/quotas", icon: Gauge, label: "Quotas", desc: "Tenant quotas and limits", category: "System", color: "text-blue-500", keywords: ["quota", "limit", "tenant"] },
    { href: "/settings/observability", icon: Activity, label: "Observability", desc: "System health and metrics", category: "System", color: "text-green-500", keywords: ["health", "metrics", "monitoring"] },
    { href: "/settings/api-keys", icon: KeyRound, label: "API Keys", desc: "Manage API keys", category: "System", color: "text-indigo-500", keywords: ["api", "key", "token"] },
    // Admin
    { href: "/admin/tenants", icon: Building2, label: "Tenants", desc: "Multi-tenant management", category: "Admin", color: "text-indigo-500", keywords: ["tenant", "multi", "plan"] },
    { href: "/admin/health", icon: CheckCircle2, label: "System Health", desc: "Service health status", category: "Admin", color: "text-green-500", keywords: ["health", "status", "service"] },
  ];

  const popularKeywords = ["MFA", "Password", "OAuth", "Audit", "Policy", "SSO", "SCIM"];

  const filtered = useMemo(() => {
    if (!search.trim()) return cards;
    const q = search.toLowerCase();
    return cards.filter(c =>
      c.label.toLowerCase().includes(q) ||
      c.desc.toLowerCase().includes(q) ||
      c.category.toLowerCase().includes(q) ||
      c.keywords.some(k => k.includes(q))
    );
  }, [search, cards]);

  const categories = useMemo(() => {
    const cats = [...new Set(filtered.map(c => c.category))];
    return cats;
  }, [filtered]);

  const highlight = (text: string, query: string) => {
    if (!query.trim()) return text;
    const idx = text.toLowerCase().indexOf(query.toLowerCase());
    if (idx === -1) return text;
    return (
      <>
        {text.substring(0, idx)}
        <mark className="bg-yellow-200 dark:bg-yellow-900 text-gray-900 dark:text-yellow-200 rounded px-0.5">{text.substring(idx, idx + query.length)}</mark>
        {text.substring(idx + query.length)}
      </>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Settings className="h-6 w-6 text-gray-500" /> {t("settingsHub.title") || "Settings"}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("settingsHub.subtitle") || "Configure your GGID platform"}</p>
        </div>
      </div>

      {/* Enhanced Search */}
      <div className="relative max-w-lg">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => setTimeout(() => setFocused(false), 200)}
          placeholder={t("settingsSearch.searchPlaceholder")}
          aria-label="Search settings"
          role="searchbox"
          className="w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 pl-10 pr-10 py-2.5 text-sm shadow-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-100 dark:focus:ring-blue-900/30 outline-none"
        />
        {search && (
          <button onClick={() => setSearch("")} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
            <X className="h-4 w-4" />
          </button>
        )}

        {/* Popular keywords when focused and empty */}
        {focused && !search && (
          <div className="absolute top-full mt-2 left-0 right-0 bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 shadow-xl p-3 z-20">
            <span className="text-xs font-medium text-gray-400 mb-2 block">{t("settingsSearch.popular")}</span>
            <div className="flex flex-wrap gap-1.5">
              {popularKeywords.map(kw => (
                <button key={kw} onClick={() => setSearch(kw.toLowerCase())}
                  className="px-2.5 py-1 rounded-lg bg-gray-100 dark:bg-gray-800 text-xs text-gray-600 dark:text-gray-400 hover:bg-blue-100 dark:hover:bg-blue-950 hover:text-blue-600">
                  {kw}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Results count */}
      {search && (
        <p className="text-xs text-gray-400">
          {filtered.length} {filtered.length === 1 ? "result" : "results"} for "{search}"
        </p>
      )}

      {/* No results */}
      {filtered.length === 0 && search && (
        <div className="text-center py-12">
          <Search className="w-12 h-12 mx-auto mb-3 text-gray-300" />
          <p className="text-sm text-gray-500">{t("settingsSearch.noResults")}</p>
          <p className="text-xs text-gray-400 mt-1">Try: MFA, password, OAuth, audit, policy</p>
        </div>
      )}

      {/* Category sections */}
      {categories.map(cat => {
        const catCards = filtered.filter(c => c.category === cat);
        return (
          <div key={cat}>
            <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-gray-400">{cat}</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {catCards.map(c => {
                const Icon = c.icon;
                return (
                  <Link key={c.href} href={c.href}
                    className={`group rounded-xl border bg-white p-4 shadow-sm transition hover:shadow-md hover:border-gray-300 dark:bg-gray-800 dark:hover:border-gray-600 ${categoryColors[c.category] || "border-gray-200 dark:border-gray-700"}`}>
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700">
                        <Icon className={`h-5 w-5 ${c.color}`} />
                      </div>
                      {c.badge && <span className="px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-600">{c.badge}</span>}
                    </div>
                    <h3 className="text-sm font-semibold text-gray-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400">
                      {highlight(c.label, search)}
                    </h3>
                    <p className="mt-1 text-xs text-gray-400">
                      {highlight(c.desc, search)}
                    </p>
                    <div className="mt-3 flex items-center gap-1 text-xs text-blue-500 opacity-0 transition group-hover:opacity-100">
                      Open <ChevronRight className="h-3 w-3" />
                    </div>
                  </Link>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
