"use client";
import { useState } from "react";
import {
  Settings, Shield, Lock, Globe, ArrowRightLeft, Bell, Gauge,
  Database, KeyRound, Server, FileText, Activity, Cpu, Zap,
  CheckCircle2, ChevronRight, Search,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { usePageTitle } from "@/lib/usePageTitle";

interface SettingsCard { href: string; icon: typeof Shield; label: string; desc: string; category: string; color: string; badge?: string; }

export default function SettingsHubPage() {
  const t = useTranslations();
  usePageTitle("Settings");
  const [search, setSearch] = useState("");

  const cards: SettingsCard[] = [
    // Security
    { href: "/settings/security", icon: Shield, label: t("settingsHub.security"), desc: t("settingsHub.securityDesc"), category: t("settingsHub.catSecurity"), color: "text-red-500" },
    { href: "/settings/password-policy", icon: Lock, label: t("settingsHub.passwordPolicy"), desc: t("settingsHub.passwordPolicyDesc"), category: t("settingsHub.catSecurity"), color: "text-orange-500" },
    { href: "/settings/rate-limits", icon: Zap, label: t("settingsHub.rateLimits"), desc: t("settingsHub.rateLimitsDesc"), category: t("settingsHub.catSecurity"), color: "text-yellow-500" },
    // Compliance
    { href: "/settings/compliance-dashboard", icon: FileText, label: t("settingsHub.compliance"), desc: t("settingsHub.complianceDesc"), category: t("settingsHub.catCompliance"), color: "text-emerald-500" },
    { href: "/settings/alerting", icon: Bell, label: t("settingsHub.alerting"), desc: t("settingsHub.alertingDesc"), category: t("settingsHub.catCompliance"), color: "text-blue-500" },
    { href: "/settings/audit-export-center", icon: Database, label: t("settingsHub.auditExport"), desc: t("settingsHub.auditExportDesc"), category: t("settingsHub.catCompliance"), color: "text-purple-500" },
    // Integrations
    { href: "/settings/scim-provisioning", icon: ArrowRightLeft, label: t("settingsHub.scim"), desc: t("settingsHub.scimDesc"), category: t("settingsHub.catIntegrations"), color: "text-cyan-500", badge: "SCIM 2.0" },
    { href: "/settings/webhooks", icon: Bell, label: t("settingsHub.webhooks"), desc: t("settingsHub.webhooksDesc"), category: t("settingsHub.catIntegrations"), color: "text-pink-500" },
    { href: "/settings/federation", icon: Globe, label: t("settingsHub.federation"), desc: t("settingsHub.federationDesc"), category: t("settingsHub.catIntegrations"), color: "text-teal-500" },
    { href: "/settings/graphql", icon: Cpu, label: t("settingsHub.graphql"), desc: t("settingsHub.graphqlDesc"), category: t("settingsHub.catIntegrations"), color: "text-violet-500", badge: "Beta" },
    // System
    { href: "/settings/quotas", icon: Gauge, label: t("settingsHub.quotas"), desc: t("settingsHub.quotasDesc"), category: t("settingsHub.catSystem"), color: "text-blue-500" },
    { href: "/settings/observability", icon: Activity, label: t("settingsHub.observability"), desc: t("settingsHub.observabilityDesc"), category: t("settingsHub.catSystem"), color: "text-green-500" },
    { href: "/settings/api-keys", icon: KeyRound, label: t("settingsHub.apiKeys"), desc: t("settingsHub.apiKeysDesc"), category: t("settingsHub.catSystem"), color: "text-indigo-500" },
    { href: "/settings/import-wizard", icon: Server, label: t("settingsHub.importWizard"), desc: t("settingsHub.importWizardDesc"), category: t("settingsHub.catSystem"), color: "text-gray-500" },
    { href: "/settings/delegation", icon: ArrowRightLeft, label: t("settingsHub.delegation"), desc: t("settingsHub.delegationDesc"), category: t("settingsHub.catSystem"), color: "text-purple-500" },
    { href: "/settings/preference-center", icon: Settings, label: t("settingsHub.preferences"), desc: t("settingsHub.preferencesDesc"), category: t("settingsHub.catSystem"), color: "text-teal-500" },
    // Admin
    { href: "/admin/settings", icon: Server, label: t("settingsHub.adminSettings"), desc: t("settingsHub.adminSettingsDesc"), category: t("settingsHub.catAdmin"), color: "text-gray-600" },
    { href: "/admin/backup", icon: Database, label: t("settingsHub.backup"), desc: t("settingsHub.backupDesc"), category: t("settingsHub.catAdmin"), color: "text-indigo-500" },
    { href: "/admin/secrets", icon: KeyRound, label: t("settingsHub.secrets"), desc: t("settingsHub.secretsDesc"), category: t("settingsHub.catAdmin"), color: "text-amber-500" },
    { href: "/admin/health", icon: CheckCircle2, label: t("settingsHub.health"), desc: t("settingsHub.healthDesc"), category: t("settingsHub.catAdmin"), color: "text-green-500" },
  ];

  const categories = [...new Set(cards.map(c => c.category))];
  const filtered = search ? cards.filter(c => c.label.toLowerCase().includes(search.toLowerCase()) || c.desc.toLowerCase().includes(search.toLowerCase())) : cards;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Settings className="h-6 w-6 text-gray-500" /> {t("settingsHub.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("settingsHub.subtitle")}</p></div>
      </div>

      <div className="relative max-w-md"><Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder={t("settingsHub.search")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-2 text-sm" /></div>

      {categories.map(cat => {
        const catCards = filtered.filter(c => c.category === cat);
        if (catCards.length === 0) return null;
        return (
          <div key={cat}>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-400">{cat}</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {catCards.map(c => {
                const Icon = c.icon;
                return (
                  <a key={c.href} href={c.href} className="group rounded-xl border border-gray-200 bg-white p-4 shadow-sm transition hover:shadow-md hover:border-gray-300 dark:border-gray-700 dark:bg-gray-800 dark:hover:border-gray-600">
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><Icon className={`h-5 w-5 ${c.color}`} /></div>
                      {c.badge && <span className="px-1.5 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-600">{c.badge}</span>}
                    </div>
                    <h3 className="text-sm font-semibold text-gray-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400">{c.label}</h3>
                    <p className="mt-1 text-xs text-gray-400">{c.desc}</p>
                    <div className="mt-3 flex items-center gap-1 text-xs text-blue-500 opacity-0 transition group-hover:opacity-100">{t("settingsHub.open")} <ChevronRight className="h-3 w-3" /></div>
                  </a>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
