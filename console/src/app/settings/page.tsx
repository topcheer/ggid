"use client";

import { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { useTheme } from "@/lib/theme";
import { Save, Shield, Key, Lock, Globe, Server, Mail, Palette, Moon, Sun } from "lucide-react";

type Tab = "general" | "security" | "ldap" | "oidc" | "saml" | "smtp" | "branding";

export default function SettingsPage() {
  const { apiFetch, API_BASE, TENANT_ID } = useApi();
  const { theme, toggle } = useTheme();
  const [tab, setTab] = useState<Tab>("security");
  const [msg, setMsg] = useState<string | null>(null);
  const [oidcConfig, setOidcConfig] = useState<Record<string, unknown> | null>(null);

  // Security config
  const [jwtConfig, setJwtConfig] = useState({
    issuer: "https://ggid.dev",
    access_token_ttl: "15m",
    refresh_token_ttl: "168h",
    algorithm: "RS256",
  });
  const [passwordPolicy, setPasswordPolicy] = useState({
    min_length: "12",
    require_uppercase: true,
    require_lowercase: true,
    require_digit: true,
    require_special: true,
    history_count: "5",
    expiry_days: "90",
  });

  // LDAP config
  const [ldapConfig, setLdapConfig] = useState({
    url: "ldap://ldap:389",
    bind_dn: "cn=admin,dc=corp,dc=local",
    base_dn: "dc=corp,dc=local",
    user_filter: "(uid={username})",
    start_tls: false,
    auto_provision: true,
  });

  const [smtpConfig, setSmtpConfig] = useState({
    host: "smtp.gmail.com",
    port: "587",
    username: "",
    password: "",
    from_email: "noreply@ggid.dev",
    use_tls: true,
  });

  const [branding, setBranding] = useState({
    logo_url: "",
    primary_color: "#6366f1",
    login_title: "GGID Console",
    login_subtitle: "Identity & Access Management",
  });

  const [samlConfig, setSamlConfig] = useState({
    entity_id: "https://ggid.dev/saml/metadata",
    acs_url: "https://ggid.dev/saml/acs",
    slo_url: "https://ggid.dev/saml/slo",
    idp_entity_id: "",
    idp_sso_url: "",
    idp_slo_url: "",
    idp_cert: "",
    name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    signing_cert: "",
    want_assertions_signed: true,
    want_responses_signed: true,
  });

  useEffect(() => {
    fetch(`${API_BASE}/oauth/.well-known/openid-configuration`)
      .then((r) => (r.ok ? r.json() : null))
      .then((d) => setOidcConfig(d))
      .catch(() => setOidcConfig(null));
  }, [API_BASE]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
    { id: "security", label: "Security", icon: Shield },
    { id: "ldap", label: "LDAP / AD", icon: Server },
    { id: "smtp", label: "SMTP", icon: Mail },
    { id: "branding", label: "Branding", icon: Palette },
    { id: "general", label: "General", icon: Globe },
    { id: "oidc", label: "OIDC", icon: Key },
    { id: "saml", label: "SAML SP", icon: Globe },
  ];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold text-gray-900 dark:text-gray-100">Settings</h1>

      {msg && <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-gray-200 dark:border-gray-700">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${
              tab === t.id ? "border-b-2 border-brand-600 text-brand-600" : "text-gray-500 hover:text-gray-700"
            }`}
          >
            <t.icon className="h-4 w-4" /> {t.label}
          </button>
        ))}
      </div>

      {tab === "security" && (
        <div className="space-y-6">
          {/* JWT Configuration */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold">
                <Key className="h-5 w-5 text-brand-600" /> JWT Configuration
              </h2>
              <button onClick={() => setMsg("JWT config saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700">
                <Save className="h-4 w-4" /> Save
              </button>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Issuer</label>
                <input value={jwtConfig.issuer} onChange={(e) => setJwtConfig({ ...jwtConfig, issuer: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Signing Algorithm</label>
                <select value={jwtConfig.algorithm} onChange={(e) => setJwtConfig({ ...jwtConfig, algorithm: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm">
                  <option>RS256</option><option>RS384</option><option>RS512</option><option>ES256</option><option>HS256</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Access Token TTL</label>
                <input value={jwtConfig.access_token_ttl} onChange={(e) => setJwtConfig({ ...jwtConfig, access_token_ttl: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Refresh Token TTL</label>
                <input value={jwtConfig.refresh_token_ttl} onChange={(e) => setJwtConfig({ ...jwtConfig, refresh_token_ttl: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" />
              </div>
            </div>
          </div>

          {/* Password Policy */}
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="flex items-center gap-2 text-lg font-semibold">
                <Lock className="h-5 w-5 text-brand-600" /> Password Policy
              </h2>
              <button onClick={() => setMsg("Password policy saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700">
                <Save className="h-4 w-4" /> Save
              </button>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Minimum Length</label>
                <input type="number" value={passwordPolicy.min_length} onChange={(e) => setPasswordPolicy({ ...passwordPolicy, min_length: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Password History</label>
                <input type="number" value={passwordPolicy.history_count} onChange={(e) => setPasswordPolicy({ ...passwordPolicy, history_count: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
                <p className="mt-1 text-xs text-gray-400">Prevent reusing last N passwords</p>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Expiry (days)</label>
                <input type="number" value={passwordPolicy.expiry_days} onChange={(e) => setPasswordPolicy({ ...passwordPolicy, expiry_days: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" />
              </div>
            </div>
            <div className="mt-4 grid gap-2 sm:grid-cols-2">
              {[
                { key: "require_uppercase", label: "Require uppercase (A-Z)" },
                { key: "require_lowercase", label: "Require lowercase (a-z)" },
                { key: "require_digit", label: "Require digit (0-9)" },
                { key: "require_special", label: "Require special character (!@#$)" },
              ].map((req) => (
                <label key={req.key} className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={passwordPolicy[req.key as keyof typeof passwordPolicy] as boolean}
                    onChange={(e) => setPasswordPolicy({ ...passwordPolicy, [req.key]: e.target.checked })}
                    className="rounded"
                  />
                  <span className="text-sm text-gray-700">{req.label}</span>
                </label>
              ))}
            </div>
          </div>
        </div>
      )}

      {tab === "ldap" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Server className="h-5 w-5 text-brand-600" /> LDAP / Active Directory
            </h2>
            <button onClick={() => setMsg("LDAP config saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700">
              <Save className="h-4 w-4" /> Save
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">LDAP URL</label>
              <input value={ldapConfig.url} onChange={(e) => setLdapConfig({ ...ldapConfig, url: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Base DN</label>
              <input value={ldapConfig.base_dn} onChange={(e) => setLdapConfig({ ...ldapConfig, base_dn: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Bind DN</label>
              <input value={ldapConfig.bind_dn} onChange={(e) => setLdapConfig({ ...ldapConfig, bind_dn: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">User Filter</label>
              <input value={ldapConfig.user_filter} onChange={(e) => setLdapConfig({ ...ldapConfig, user_filter: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" />
            </div>
          </div>
          <div className="mt-4 flex gap-6">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={ldapConfig.start_tls} onChange={(e) => setLdapConfig({ ...ldapConfig, start_tls: e.target.checked })} className="rounded" />
              <span className="text-sm text-gray-700">Start TLS</span>
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={ldapConfig.auto_provision} onChange={(e) => setLdapConfig({ ...ldapConfig, auto_provision: e.target.checked })} className="rounded" />
              <span className="text-sm text-gray-700">Auto-provision users on first login</span>
            </label>
          </div>
          <div className="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs text-gray-500">
            <p>Environment variables: LDAP_URL, LDAP_BIND_DN, LDAP_BIND_PASSWORD, LDAP_BASE_DN, LDAP_USER_FILTER, LDAP_START_TLS, LDAP_AUTO_PROVISION</p>
          </div>
        </div>
      )}

      {tab === "smtp" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Mail className="h-5 w-5 text-brand-600" /> SMTP Configuration
            </h2>
            <div className="flex gap-2">
              <button onClick={() => setMsg("Test email sent")} className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50">Test Send</button>
              <button onClick={() => setMsg("SMTP config saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"><Save className="h-4 w-4" /> Save</button>
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div><label className="mb-1 block text-xs font-medium text-gray-500">SMTP Host</label><input value={smtpConfig.host} onChange={(e) => setSmtpConfig({ ...smtpConfig, host: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div><label className="mb-1 block text-xs font-medium text-gray-500">Port</label><input value={smtpConfig.port} onChange={(e) => setSmtpConfig({ ...smtpConfig, port: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div><label className="mb-1 block text-xs font-medium text-gray-500">Username</label><input value={smtpConfig.username} onChange={(e) => setSmtpConfig({ ...smtpConfig, username: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div><label className="mb-1 block text-xs font-medium text-gray-500">Password</label><input type="password" value={smtpConfig.password} onChange={(e) => setSmtpConfig({ ...smtpConfig, password: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div><label className="mb-1 block text-xs font-medium text-gray-500">From Email</label><input value={smtpConfig.from_email} onChange={(e) => setSmtpConfig({ ...smtpConfig, from_email: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div><label className="mb-1 flex items-center gap-2 pt-6"><input type="checkbox" checked={smtpConfig.use_tls} onChange={(e) => setSmtpConfig({ ...smtpConfig, use_tls: e.target.checked })} className="rounded" /><span className="text-sm text-gray-700">Use TLS</span></label></div>
          </div>
        </div>
      )}

      {tab === "branding" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Palette className="h-5 w-5 text-brand-600" /> Brand Customization
            </h2>
            <button onClick={() => setMsg("Branding saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"><Save className="h-4 w-4" /> Save</button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2"><label className="mb-1 block text-xs font-medium text-gray-500">Logo URL</label><input value={branding.logo_url} onChange={(e) => setBranding({ ...branding, logo_url: e.target.value })} placeholder="https://example.com/logo.png" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div><label className="mb-1 block text-xs font-medium text-gray-500">Primary Color</label><div className="flex items-center gap-2"><input type="color" value={branding.primary_color} onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })} className="h-9 w-12 rounded border border-gray-300" /><input value={branding.primary_color} onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })} className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" /></div></div>
            <div><label className="mb-1 block text-xs font-medium text-gray-500">Login Title</label><input value={branding.login_title} onChange={(e) => setBranding({ ...branding, login_title: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
            <div className="sm:col-span-2"><label className="mb-1 block text-xs font-medium text-gray-500">Login Subtitle</label><input value={branding.login_subtitle} onChange={(e) => setBranding({ ...branding, login_subtitle: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" /></div>
          </div>
          <div className="mt-4 rounded-lg border border-gray-200 p-4 text-center">
            <div className="mx-auto mb-2 flex h-10 w-10 items-center justify-center rounded-lg text-white font-bold" style={{ backgroundColor: branding.primary_color }}>G</div>
            <p className="font-semibold">{branding.login_title}</p>
            <p className="text-sm text-gray-500">{branding.login_subtitle}</p>
          </div>
        </div>
      )}

      {tab === "general" && (
        <div className="space-y-6">
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">Appearance</h2>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-700 dark:text-gray-300">Theme</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">Switch between light and dark mode</p>
              </div>
              <button
                onClick={toggle}
                className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {theme === "dark" ? (
                  <>
                    <Sun className="h-4 w-4" /> Light Mode
                  </>
                ) : (
                  <>
                    <Moon className="h-4 w-4" /> Dark Mode
                  </>
                )}
              </button>
            </div>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">Tenant Information</h2>
            <div className="space-y-3">
              {[
                { label: "Tenant ID", value: TENANT_ID },
                { label: "Plan", value: "Enterprise" },
                { label: "Status", value: "Active" },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between border-b border-gray-100 pb-3 last:border-0">
                  <span className="text-sm text-gray-500">{item.label}</span>
                  <span className="text-sm font-medium">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">System Information</h2>
            <div className="space-y-3">
              {[
                { label: "Version", value: "1.0.0-dev" },
                { label: "License", value: "Apache 2.0" },
                { label: "API Gateway", value: API_BASE },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between border-b border-gray-100 pb-3 last:border-0">
                  <span className="text-sm text-gray-500">{item.label}</span>
                  <span className="text-sm font-medium">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {tab === "oidc" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold">OIDC Discovery</h2>
          {oidcConfig ? (
            <div className="space-y-2">
              {Object.entries(oidcConfig).map(([key, value]) => (
                <div key={key} className="flex flex-col gap-1">
                  <span className="text-xs font-medium text-gray-500">{key}</span>
                  <span className="break-all text-sm text-gray-800">{String(value)}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-400">OIDC discovery endpoint not available</p>
          )}
        </div>
      )}

      {tab === "saml" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold">SAML Service Provider Configuration</h2>
          <div className="space-y-4">
            <div>
              <h3 className="mb-2 text-sm font-semibold text-gray-700">Service Provider</h3>
              <div className="grid grid-cols-2 gap-4">
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">Entity ID</span>
                  <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.entity_id} onChange={(e) => setSamlConfig({ ...samlConfig, entity_id: e.target.value })} />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">ACS URL</span>
                  <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.acs_url} onChange={(e) => setSamlConfig({ ...samlConfig, acs_url: e.target.value })} />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">SLO URL</span>
                  <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.slo_url} onChange={(e) => setSamlConfig({ ...samlConfig, slo_url: e.target.value })} />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">Name ID Format</span>
                  <select className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.name_id_format} onChange={(e) => setSamlConfig({ ...samlConfig, name_id_format: e.target.value })}>
                    <option value="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">Email Address</option>
                    <option value="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified">Unspecified</option>
                    <option value="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent">Persistent</option>
                    <option value="urn:oasis:names:tc:SAML:2.0:nameid-format:transient">Transient</option>
                  </select>
                </label>
              </div>
            </div>
            <div>
              <h3 className="mb-2 text-sm font-semibold text-gray-700">Identity Provider</h3>
              <div className="grid grid-cols-2 gap-4">
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">IdP Entity ID</span>
                  <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.idp_entity_id} onChange={(e) => setSamlConfig({ ...samlConfig, idp_entity_id: e.target.value })} placeholder="https://idp.example.com/entity" />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">IdP SSO URL</span>
                  <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.idp_sso_url} onChange={(e) => setSamlConfig({ ...samlConfig, idp_sso_url: e.target.value })} placeholder="https://idp.example.com/sso" />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">IdP SLO URL</span>
                  <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm" value={samlConfig.idp_slo_url} onChange={(e) => setSamlConfig({ ...samlConfig, idp_slo_url: e.target.value })} placeholder="https://idp.example.com/slo" />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-gray-500">IdP Certificate (PEM)</span>
                  <textarea className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" rows={3} value={samlConfig.idp_cert} onChange={(e) => setSamlConfig({ ...samlConfig, idp_cert: e.target.value })} placeholder="-----BEGIN CERTIFICATE-----" />
                </label>
              </div>
            </div>
            <div>
              <h3 className="mb-2 text-sm font-semibold text-gray-700">Security</h3>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={samlConfig.want_assertions_signed} onChange={(e) => setSamlConfig({ ...samlConfig, want_assertions_signed: e.target.checked })} />
                  <span className="text-sm text-gray-700">Require signed assertions</span>
                </label>
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={samlConfig.want_responses_signed} onChange={(e) => setSamlConfig({ ...samlConfig, want_responses_signed: e.target.checked })} />
                  <span className="text-sm text-gray-700">Require signed responses</span>
                </label>
              </div>
            </div>
            <label className="block">
              <span className="mb-1 block text-xs font-medium text-gray-500">SP Signing Certificate (PEM)</span>
              <textarea className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono" rows={3} value={samlConfig.signing_cert} onChange={(e) => setSamlConfig({ ...samlConfig, signing_cert: e.target.value })} placeholder="-----BEGIN CERTIFICATE-----" />
            </label>
            <button className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700" onClick={() => setMsg("SAML configuration saved (demo)")}>
              <Save className="mr-2 inline-block h-4 w-4" />
              Save SAML Config
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
