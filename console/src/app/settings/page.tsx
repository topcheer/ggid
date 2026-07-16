"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import { useTheme } from "@/lib/theme";
import {
  Save, Shield, Key, Lock, Globe, Server, Mail, Palette, Moon, Sun, Monitor,
  User, Clock, Smartphone, LogOut, Fingerprint, Link2, Trash2, Check, Loader2,
  ChevronRight, ShieldCheck, Users, Building2, FileText, Bell, Network,
  AlertTriangle, Eye, Activity,
} from "lucide-react";

type View = "hub" | "profile" | "account" | "security" | "ldap" | "oidc" | "saml" | "smtp" | "branding" | "general";

interface CategoryLink {
  href: string;
  label: string;
  desc?: string;
}

interface Category {
  id: string;
  title: string;
  icon: typeof Shield;
  color: string;
  links: CategoryLink[];
}

const CATEGORIES: Category[] = [
  {
    id: "security",
    title: "Security",
    icon: Shield,
    color: "text-red-600 bg-red-50 dark:bg-red-950/30",
    links: [
      { href: "/settings/security", label: "Security Center" },
      { href: "/settings/mfa-enrollment", label: "MFA Enrollment" },
      { href: "/settings/password-policy", label: "Password Policy" },
      { href: "/settings/brute-force", label: "Brute Force Protection" },
      { href: "/settings/anomaly-detect-dashboard", label: "Anomaly Detection" },
      { href: "/settings/threat-hunting-workbench", label: "Threat Hunting" },
      { href: "/settings/risk-posture-dashboard", label: "Risk Posture" },
      { href: "/settings/session-token-forgery", label: "Session Token Forgery" },
    ],
  },
  {
    id: "auth",
    title: "Authentication & Integration",
    icon: Key,
    color: "text-blue-600 bg-blue-50 dark:bg-blue-950/30",
    links: [
      { href: "/settings/oauth-clients-config", label: "OAuth Clients" },
      { href: "/settings/saml-sp-config", label: "SAML Configuration" },
      { href: "/settings/scim", label: "SCIM Provisioning" },
      { href: "/settings/ldap-config", label: "LDAP Configuration" },
      { href: "/settings/ldap-sync-config", label: "LDAP Sync" },
      { href: "/settings/idp-config", label: "Identity Provider" },
      { href: "/settings/sso-providers", label: "SSO Providers" },
      { href: "/settings/webauthn", label: "WebAuthn" },
    ],
  },
  {
    id: "identity",
    title: "User Identity",
    icon: Users,
    color: "text-green-600 bg-green-50 dark:bg-green-950/30",
    links: [
      { href: "/settings/user-provisioning", label: "User Provisioning" },
      { href: "/settings/user-provisioning-center", label: "Provisioning Center" },
      { href: "/settings/role-templates", label: "Role Templates" },
      { href: "/settings/access-requests", label: "Access Requests" },
      { href: "/settings/access-review-center", label: "Access Review Center" },
      { href: "/settings/user-attestation", label: "User Attestation" },
      { href: "/settings/entitlement-review", label: "Entitlement Review" },
      { href: "/settings/recertification", label: "Access Recertification" },
    ],
  },
  {
    id: "org",
    title: "Organization",
    icon: Building2,
    color: "text-purple-600 bg-purple-50 dark:bg-purple-950/30",
    links: [
      { href: "/settings/org-tree", label: "Organization Tree" },
      { href: "/settings/org-hierarchy", label: "Org Hierarchy" },
      { href: "/settings/department-analytics", label: "Department Analytics" },
      { href: "/settings/membership-graph", label: "Membership Graph" },
      { href: "/settings/group-analytics", label: "Group Analytics" },
      { href: "/settings/tenant", label: "Tenant Settings" },
      { href: "/settings/budget-tracking", label: "Budget Tracking" },
      { href: "/settings/reporting-structure", label: "Reporting Structure" },
    ],
  },
  {
    id: "audit",
    title: "Audit & Compliance",
    icon: FileText,
    color: "text-amber-600 bg-amber-50 dark:bg-amber-950/30",
    links: [
      { href: "/settings/audit-log-viewer", label: "Audit Log Viewer" },
      { href: "/settings/audit-export-center", label: "Audit Export Center" },
      { href: "/settings/siem-forwarder-dashboard", label: "SIEM Forwarder" },
      { href: "/settings/compliance-reports", label: "Compliance Reports" },
      { href: "/settings/audit-gdpr-requests", label: "GDPR Requests" },
      { href: "/settings/evidence-collection", label: "Evidence Collection" },
      { href: "/settings/hash-chain-verification", label: "Hash Chain Verification" },
      { href: "/settings/forensics-timeline", label: "Forensics Timeline" },
    ],
  },
  {
    id: "system",
    title: "System Configuration",
    icon: Server,
    color: "text-cyan-600 bg-cyan-50 dark:bg-cyan-950/30",
    links: [
      { href: "/monitoring", label: "System Monitoring" },
      { href: "/settings/notification-templates", label: "Notification Templates" },
      { href: "/settings/alert-webhook-config", label: "Alert Webhooks" },
      { href: "/settings/webhook-subscription-config", label: "Webhook Subscriptions" },
      { href: "/settings/api-gateway-config", label: "API Gateway Config" },
      { href: "/settings/feature-flag-architecture-config", label: "Feature Flags" },
      { href: "/settings/encryption-config", label: "Encryption Config" },
      { href: "/settings/nats-jetstream", label: "NATS JetStream" },
    ],
  },
];

export default function SettingsPage() {
  const t = useTranslations();
  const { apiFetch, API_BASE, TENANT_ID } = useApi();
  const { mode, setMode, theme } = useTheme();
  const [view, setView] = useState<View>("hub");
  const [msg, setMsg] = useState<string | null>(null);
  const [oidcConfig, setOidcConfig] = useState<Record<string, unknown> | null>(null);

  const [profile, setProfile] = useState({
    display_name: "",
    email: "",
    avatar_url: "",
    locale: "en",
    timezone: "UTC",
  });

  const [pwForm, setPwForm] = useState({ current: "", new: "", confirm: "" });
  const [pwError, setPwError] = useState<string | null>(null);
  const [sessions, setSessions] = useState<{ id: string; device: string; ip: string; last_active: string; current?: boolean }[]>([]);

  useEffect(() => {
    const fetchProfile = async () => {
      try {
        const data = await apiFetch<Record<string, unknown>>("/api/v1/users/me");
        setProfile({
          display_name: (data.display_name as string) || (data.username as string) || "",
          email: (data.email as string) || "",
          avatar_url: (data.avatar_url as string) || "",
          locale: (data.locale as string) || "en",
          timezone: (data.timezone as string) || "UTC",
        });
      } catch {
        const email = localStorage.getItem("ggid_user_email") || "admin@ggid.dev";
        const name = localStorage.getItem("ggid_user_name") || email.split("@")[0];
        setProfile((prev) => ({ ...prev, display_name: name, email, locale: "en", timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC" }));
      }
    };
    fetchProfile();
  }, [apiFetch]);

  useEffect(() => {
    apiFetch<{ sessions?: typeof sessions } | typeof sessions>("/api/v1/users/me/sessions")
      .then((data) => { const list = Array.isArray(data) ? data : data.sessions || []; setSessions(list); })
      .catch(() => setSessions([]));
  }, [apiFetch]);

  useEffect(() => {
    fetch(`${API_BASE}/oauth/.well-known/openid-configuration`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setOidcConfig(d))
      .catch(() => setOidcConfig(null));
  }, [API_BASE]);

  useEffect(() => {
    if (msg) { const timer = setTimeout(() => setMsg(null), 3000); return () => clearTimeout(timer); }
  }, [msg]);

  const changePassword = async () => {
    setPwError(null);
    if (!pwForm.current || !pwForm.new) { setPwError("All fields required"); return; }
    if (pwForm.new !== pwForm.confirm) { setPwError("Passwords don't match"); return; }
    if (pwForm.new.length < 8) { setPwError("Minimum 8 characters"); return; }
    try {
      await apiFetch("/api/v1/auth/change-password", { method: "POST", body: JSON.stringify({ current_password: pwForm.current, new_password: pwForm.new }) });
      setMsg("Password changed successfully"); setPwForm({ current: "", new: "", confirm: "" });
    } catch { setMsg("Password change failed"); }
  };

  const revokeSession = async (id: string) => {
    try { await apiFetch(`/api/v1/users/me/sessions/${id}`, { method: "DELETE" }); } catch { /* ignore */ }
    setSessions((prev) => prev.filter((s) => s.id !== id));
    setMsg("Session revoked");
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";

  // ===== HUB VIEW =====
  if (view === "hub") {
    return (
      <div>
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t("settings.title")}</h1>
            <p className="mt-1 text-sm text-gray-500">Configure security, authentication, compliance, and system settings.</p>
          </div>
          <button
            onClick={() => setView("profile")}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
          >
            <User className="h-4 w-4" /> My Profile
          </button>
        </div>

        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {CATEGORIES.map((cat) => (
            <div key={cat.id} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
              <div className="mb-4 flex items-center gap-3">
                <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${cat.color}`}>
                  <cat.icon className="h-5 w-5" />
                </div>
                <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100">{cat.title}</h2>
              </div>
              <ul className="space-y-1">
                {cat.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="flex items-center justify-between rounded-lg px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-700/50 dark:hover:text-gray-200"
                    >
                      <span>{link.label}</span>
                      <ChevronRight className="h-3.5 w-3.5 opacity-0 group-hover:opacity-100" />
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Quick personal settings bar */}
        <div className="mt-8 flex flex-wrap gap-3">
          {[
            { id: "profile", label: "Profile", icon: User },
            { id: "security", label: "Password & Security", icon: Lock },
            { id: "general", label: "Theme & System", icon: Globe },
            { id: "saml", label: "SAML Config", icon: Globe },
            { id: "ldap", label: "LDAP Config", icon: Server },
            { id: "branding", label: "Branding", icon: Palette },
          ].map((item) => (
            <button
              key={item.id}
              onClick={() => setView(item.id as View)}
              className="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-700/50"
            >
              <item.icon className="h-3.5 w-3.5" /> {item.label}
            </button>
          ))}
        </div>
      </div>
    );
  }

  // ===== PERSONAL SETTINGS VIEWS =====
  return (
    <div>
      <div className="mb-6 flex items-center gap-3">
        <button onClick={() => setView("hub")} className="flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400">
          <ChevronRight className="h-4 w-4 rotate-180" /> Settings Hub
        </button>
        <span className="text-gray-300">/</span>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 capitalize">{view}</h1>
      </div>

      {msg && <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950">{msg}</div>}

      {/* PROFILE */}
      {view === "profile" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold"><User className="mr-2 inline h-5 w-5 text-brand-600" /> Profile</h2>
            <button onClick={async () => { try { await apiFetch("/api/v1/users/me", { method: "PUT", body: JSON.stringify({ display_name: profile.display_name, avatar_url: profile.avatar_url, locale: profile.locale, timezone: profile.timezone }) }); setMsg("Profile saved"); } catch { setMsg("Saved locally"); } }} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700">
              <Save className="h-4 w-4" /> Save
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div><label className={labelCls}>Display Name</label><input value={profile.display_name} onChange={(e) => setProfile({ ...profile, display_name: e.target.value })} className={inputCls} /></div>
            <div><label className={labelCls}>Email</label><input value={profile.email} disabled className={`${inputCls} cursor-not-allowed opacity-60`} /></div>
            <div><label className={labelCls}>Avatar URL</label><input value={profile.avatar_url} onChange={(e) => setProfile({ ...profile, avatar_url: e.target.value })} className={inputCls} /></div>
            <div><label className={labelCls}>Locale</label><select value={profile.locale} onChange={(e) => setProfile({ ...profile, locale: e.target.value })} className={inputCls}><option value="en">English</option><option value="zh">中文</option></select></div>
          </div>
        </div>
      )}

      {/* SECURITY / PASSWORD */}
      {view === "security" && (
        <div className="space-y-6">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold"><Lock className="mr-2 inline h-5 w-5 text-brand-600" /> Change Password</h2>
            {pwError && <div className="mb-3 rounded-lg border border-red-200 bg-red-50 p-2 text-sm text-red-700">{pwError}</div>}
            <div className="grid gap-4 sm:grid-cols-3">
              <div><label className={labelCls}>Current</label><input autoComplete="current-password" type="password" value={pwForm.current} onChange={(e) => setPwForm({ ...pwForm, current: e.target.value })} className={inputCls} /></div>
              <div><label className={labelCls}>New</label><input autoComplete="current-password" type="password" value={pwForm.new} onChange={(e) => setPwForm({ ...pwForm, new: e.target.value })} className={inputCls} /></div>
              <div><label className={labelCls}>Confirm</label><input autoComplete="current-password" type="password" value={pwForm.confirm} onChange={(e) => setPwForm({ ...pwForm, confirm: e.target.value })} className={inputCls} /></div>
            </div>
            <button onClick={changePassword} className="mt-4 flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"><Save className="h-4 w-4" /> Change Password</button>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold"><Smartphone className="mr-2 inline h-5 w-5 text-brand-600" /> Active Sessions</h2>
            {sessions.length === 0 ? (
              <p className="text-sm text-gray-400">No active sessions found.</p>
            ) : (
              <div className="space-y-2">
                {sessions.map((s) => (
                  <div key={s.id} className="flex items-center justify-between rounded-lg border border-gray-100 p-3 dark:border-gray-700">
                    <div><p className="text-sm font-medium">{s.device} {s.current && <span className="ml-1 text-xs text-green-600">(current)</span>}</p><p className="text-xs text-gray-400">{s.ip} · {s.last_active}</p></div>
                    {!s.current && <button onClick={() => revokeSession(s.id)} className="text-xs text-red-600 hover:underline">Revoke</button>}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* GENERAL / THEME */}
      {view === "general" && (
        <div className="space-y-6">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold">Appearance</h2>
            <div className="flex items-center gap-1 rounded-lg border border-gray-300 p-1 dark:border-gray-600">
              <button onClick={() => setMode("light")} className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium ${mode === "light" ? "bg-brand-600 text-white" : "text-gray-600 dark:text-gray-400"}`}><Sun className="h-3.5 w-3.5" /> Light</button>
              <button onClick={() => setMode("dark")} className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium ${mode === "dark" ? "bg-brand-600 text-white" : "text-gray-600 dark:text-gray-400"}`}><Moon className="h-3.5 w-3.5" /> Dark</button>
              <button onClick={() => setMode("system")} className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium ${mode === "system" ? "bg-brand-600 text-white" : "text-gray-600 dark:text-gray-400"}`}><Monitor className="h-3.5 w-3.5" /> System</button>
            </div>
            {mode === "system" && <p className="mt-2 text-xs text-gray-400">Currently using {theme} theme</p>}
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold">Tenant Information</h2>
            <div className="space-y-3">
              {[{ label: "Tenant ID", value: TENANT_ID }, { label: "Plan", value: "Enterprise" }, { label: "Status", value: "Active" }, { label: "Version", value: "1.0.0-dev" }, { label: "API Gateway", value: API_BASE || "default" }].map((item) => (
                <div key={item.label} className="flex items-center justify-between border-b border-gray-100 dark:border-gray-700 pb-3 last:border-0">
                  <span className="text-sm text-gray-500">{item.label}</span>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* OIDC */}
      {view === "oidc" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold">OIDC Discovery</h2>
          {oidcConfig ? (
            <div className="space-y-2">{Object.entries(oidcConfig).map(([key, value]) => (<div key={key}><span className="text-xs font-medium text-gray-500">{key}</span><p className="break-all text-sm">{String(value)}</p></div>))}</div>
          ) : <p className="text-sm text-gray-400">OIDC discovery endpoint not available</p>}
        </div>
      )}

      {/* LDAP */}
      {view === "ldap" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold"><Server className="mr-2 inline h-5 w-5 text-brand-600" /> LDAP Configuration</h2>
          <p className="text-sm text-gray-500">LDAP is configured via environment variables. See <code className="rounded bg-gray-100 px-1 dark:bg-gray-900">LDAP_URL</code>, <code className="rounded bg-gray-100 px-1 dark:bg-gray-900">LDAP_BIND_DN</code>, etc.</p>
          <div className="mt-4"><Link href="/settings/ldap-config" className="text-sm text-brand-600 hover:underline">Go to LDAP Configuration page →</Link></div>
          <div className="mt-2"><Link href="/settings/ldap-sync-config" className="text-sm text-brand-600 hover:underline">Go to LDAP Sync Config page →</Link></div>
        </div>
      )}

      {/* SAML */}
      {view === "saml" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold">SAML Configuration</h2>
          <p className="text-sm text-gray-500">SAML SP configuration is available in the detailed settings.</p>
          <div className="mt-4"><Link href="/settings/saml-sp-config" className="text-sm text-brand-600 hover:underline">Go to SAML SP Config page →</Link></div>
        </div>
      )}

      {/* BRANDING */}
      {view === "branding" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold"><Palette className="mr-2 inline h-5 w-5 text-brand-600" /> Branding</h2>
          <p className="text-sm text-gray-500">Customize the login page and console appearance.</p>
          <div className="mt-4"><Link href="/settings/branding-config" className="text-sm text-brand-600 hover:underline">Go to Branding Config page →</Link></div>
          <div className="mt-2"><Link href="/settings/branding-custom" className="text-sm text-brand-600 hover:underline">Go to Branding Customization page →</Link></div>
        </div>
      )}
    </div>
  );
}
