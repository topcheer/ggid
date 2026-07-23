"use client";

import { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  Search, ArrowRight, Command, CornerDownLeft, ArrowUp, ArrowDown,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface CommandItem {
  id: string;
  label: string;
  href: string;
  group: string;
  icon?: string;
}

const ALL_COMMANDS: CommandItem[] = [
  // Overview
  { id: "dashboard", label: "Dashboard", href: "/dashboard", group: "Overview" },
  { id: "sessions", label: "My Sessions", href: "/sessions", group: "Overview" },
  { id: "access-requests", label: "Access Requests", href: "/access-requests", group: "Overview" },
  // Identity
  { id: "users", label: "Users", href: "/users", group: "Identity" },
  { id: "roles", label: "Roles & Permissions", href: "/roles", group: "Identity" },
  { id: "organizations", label: "Organizations", href: "/organizations", group: "Identity" },
  { id: "org-tree", label: "Organization Tree", href: "/organizations/tree", group: "Identity" },
  { id: "api-keys", label: "API Keys", href: "/api-keys", group: "Identity" },
  { id: "import", label: "Import Users", href: "/settings/import-wizard", group: "Identity" },
  // Security
  { id: "itdr", label: "ITDR Dashboard", href: "/security/itdr", group: "Security" },
  { id: "security-sessions", label: "Security → Sessions", href: "/security/session-detail", group: "Security" },
  { id: "risk-score", label: "Risk Score", href: "/security/risk-score", group: "Security" },
  { id: "posture", label: "Security Posture", href: "/security/posture", group: "Security" },
  { id: "conditional-access", label: "Conditional Access", href: "/settings/conditional-access", group: "Security" },
  { id: "password-policy", label: "Password Policy", href: "/settings/password-policy", group: "Security" },
  { id: "mfa", label: "MFA Configuration", href: "/settings/mfa", group: "Security" },
  { id: "passkeys", label: "Passkey Management", href: "/settings/passkey-management", group: "Security" },
  { id: "webauthn", label: "WebAuthn Configuration", href: "/settings/webauthn-config", group: "Security" },
  // Audit
  { id: "audit", label: "Audit Log", href: "/audit", group: "Audit" },
  { id: "audit-alerts", label: "Audit Alerts", href: "/audit/alerts", group: "Audit" },
  { id: "audit-anomalies", label: "Anomaly Detection", href: "/audit/anomalies", group: "Audit" },
  { id: "audit-advanced", label: "Advanced Audit Search", href: "/audit/advanced", group: "Audit" },
  { id: "policies", label: "Policies", href: "/policies", group: "Audit" },
  { id: "access-reviews", label: "Access Reviews", href: "/audit/access-reviews", group: "Audit" },
  // Applications
  { id: "oauth-clients", label: "OAuth Clients", href: "/oauth-clients", group: "Applications" },
  { id: "webhooks", label: "Webhooks", href: "/webhooks", group: "Applications" },
  { id: "scim", label: "SCIM Configuration", href: "/settings/scim", group: "Applications" },
  { id: "ldap", label: "LDAP Configuration", href: "/settings/ldap-config", group: "Applications" },
  { id: "saml", label: "SAML Configuration", href: "/settings/saml-config", group: "Applications" },
  // Settings
  { id: "settings", label: "All Settings", href: "/settings", group: "Settings" },
  { id: "branding", label: "Branding", href: "/settings/branding", group: "Settings" },
  { id: "feature-flags", label: "Feature Flags", href: "/settings/feature-flags", group: "Settings" },
  { id: "notifications", label: "Notifications", href: "/settings/notifications", group: "Settings" },
  // Analytics
  { id: "analytics-iam", label: "IAM Metrics", href: "/analytics/iam-metrics", group: "Analytics" },
  { id: "analytics-identity", label: "Identity Analytics", href: "/analytics/identity", group: "Analytics" },
  { id: "analytics-login", label: "Login Security Analytics", href: "/analytics/login-security", group: "Analytics" },
  // Platform
  { id: "admin", label: "Platform → Tenants", href: "/admin", group: "Platform" },
  { id: "admin-audit", label: "Global Audit", href: "/admin/audit/global", group: "Platform" },
  { id: "admin-threats", label: "Threat Dashboard", href: "/admin/threats", group: "Platform" },
  { id: "admin-health", label: "System Health", href: "/admin/health", group: "Platform" },
  // Profile
  { id: "profile", label: "My Profile", href: "/profile", group: "Account" },
  { id: "activity", label: "My Activity", href: "/activity", group: "Account" },
  // Onboarding
  { id: "onboarding", label: "Create New Organization", href: "/onboarding", group: "Account" },
];

export function CommandPalette() {
  const router = useRouter();
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  // Toggle with Ctrl+K / Cmd+K
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen(prev => !prev);
      }
      if (e.key === "Escape") {
        setOpen(false);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  // Focus input when opened
  useEffect(() => {
    if (open) {
      setQuery("");
      setSelected(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  // Filter commands
  const filtered = useMemo(() => {
    const q = query.toLowerCase().trim();
    if (!q) return ALL_COMMANDS;
    return ALL_COMMANDS.filter(c =>
      c.label.toLowerCase().includes(q) ||
      c.group.toLowerCase().includes(q) ||
      c.href.toLowerCase().includes(q)
    );
  }, [query]);

  // Group filtered results
  const grouped = useMemo(() => {
    const groups: Record<string, CommandItem[]> = {};
    for (const item of filtered) {
      if (!groups[item.group]) groups[item.group] = [];
      groups[item.group].push(item);
    }
    return groups;
  }, [filtered]);

  // Flat list for keyboard navigation
  const flatList = useMemo(() => Object.values(grouped).flat(), [grouped]);

  const navigate = useCallback((item: CommandItem) => {
    setOpen(false);
    router.push(item.href);
  }, [router]);

  // Keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelected(prev => Math.min(prev + 1, flatList.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelected(prev => Math.max(prev - 1, 0));
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (flatList[selected]) navigate(flatList[selected]);
    }
  };

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        aria-label="Open quick navigation (Ctrl+K)"
        className="hidden md:flex items-center gap-2 rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-3 py-1.5 text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition"
        title="Quick navigation (Ctrl+K)"
      >
        <Command className="h-3.5 w-3.5" />
        <span>Quick Jump</span>
        <kbd className="rounded border border-gray-300 dark:border-gray-600 px-1 py-0.5 font-mono text-[10px]">⌘K</kbd>
      </button>
    );
  }

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm"
        onClick={() => setOpen(false)}
      />

      {/* Palette */}
      <div className="fixed left-1/2 top-[15%] z-50 w-full max-w-xl -translate-x-1/2">
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-2xl dark:border-gray-700 dark:bg-gray-800">
          {/* Search input */}
          <div className="flex items-center gap-3 border-b border-gray-200 px-4 py-3 dark:border-gray-700">
            <Search className="h-5 w-5 text-gray-400" />
            <input
              ref={inputRef}
              type="text"
              aria-label="Search pages and settings"
              role="combobox"
              aria-expanded="true"
              aria-controls="command-palette-results"
              value={query}
              onChange={(e) => { setQuery(e.target.value); setSelected(0); }}
              onKeyDown={handleKeyDown}
              placeholder="Search pages, settings, tools..."
              className="flex-1 bg-transparent text-sm text-gray-900 dark:text-white placeholder:text-gray-400 focus:outline-none"
            />
            <kbd className="rounded border border-gray-300 dark:border-gray-600 px-1.5 py-0.5 font-mono text-[10px] text-gray-400">ESC</kbd>
          </div>

          {/* Results */}
          <div className="max-h-96 overflow-y-auto p-2">
            {flatList.length === 0 && (
              <div className="py-8 text-center text-sm text-gray-400">No results found</div>
            )}
            {Object.entries(grouped).map(([group, items]) => (
              <div key={group}>
                <div className="px-2 py-1.5 text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                  {group}
                </div>
                {items.map((item) => {
                  const idx = flatList.indexOf(item);
                  const isSelected = idx === selected;
                  return (
                    <button
                      key={item.id}
                      onClick={() => navigate(item)}
                      onMouseEnter={() => setSelected(idx)}
                      className={`flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition ${
                        isSelected
                          ? "bg-brand-50 text-brand-700 dark:bg-brand-950/30 dark:text-brand-300"
                          : "text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700/50"
                      }`}
                    >
                      <span className="flex items-center gap-2">
                        <span className="text-gray-400 dark:text-gray-500">{item.label}</span>
                      </span>
                      {isSelected && <CornerDownLeft className="h-3.5 w-3.5 text-brand-500" />}
                    </button>
                  );
                })}
              </div>
            ))}
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between border-t border-gray-200 px-4 py-2 dark:border-gray-700">
            <div className="flex items-center gap-3 text-xs text-gray-400">
              <span className="flex items-center gap-1"><ArrowUp className="h-3 w-3" /><ArrowDown className="h-3 w-3" /> Navigate</span>
              <span className="flex items-center gap-1"><CornerDownLeft className="h-3 w-3" /> Open</span>
            </div>
            <span className="text-xs text-gray-400">{flatList.length} results</span>
          </div>
        </div>
      </div>
    </>
  );
}