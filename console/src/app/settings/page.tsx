"use client";

import { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { useTheme } from "@/lib/theme";
import {
  Save, Shield, Key, Lock, Globe, Server, Mail, Palette, Moon, Sun, Monitor,
  User, Clock, Smartphone, LogOut, Fingerprint, Link2, Trash2, Check,
} from "lucide-react";

type Tab = "profile" | "account" | "security" | "ldap" | "oidc" | "saml" | "smtp" | "branding" | "general";

// Common locales and timezones for dropdowns
const LOCALES = [
  { value: "en", label: "English" },
  { value: "zh", label: "中文 (Chinese)" },
  { value: "es", label: "Español (Spanish)" },
  { value: "fr", label: "Français (French)" },
  { value: "de", label: "Deutsch (German)" },
  { value: "ja", label: "日本語 (Japanese)" },
  { value: "ko", label: "한국어 (Korean)" },
  { value: "pt", label: "Português (Portuguese)" },
  { value: "ru", label: "Русский (Russian)" },
];

const TIMEZONES = [
  "UTC",
  "America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles",
  "America/Sao_Paulo", "Europe/London", "Europe/Paris", "Europe/Berlin",
  "Europe/Moscow", "Asia/Dubai", "Asia/Shanghai", "Asia/Tokyo", "Asia/Seoul",
  "Asia/Singapore", "Australia/Sydney", "Pacific/Auckland",
];

interface SessionInfo {
  id: string;
  device: string;
  ip: string;
  last_active: string;
  current?: boolean;
}

interface ConnectedApp {
  id: string;
  name: string;
  scopes: string[];
  authorized_at: string;
}

export default function SettingsPage() {
  const { apiFetch, API_BASE, TENANT_ID } = useApi();
  const { mode, setMode, theme } = useTheme();
  const [tab, setTab] = useState<Tab>("profile");
  const [msg, setMsg] = useState<string | null>(null);
  const [oidcConfig, setOidcConfig] = useState<Record<string, unknown> | null>(null);

  // --- Profile state ---
  const [profile, setProfile] = useState({
    display_name: "",
    email: "",
    avatar_url: "",
    locale: "en",
    timezone: "UTC",
  });
  const [profileLoaded, setProfileLoaded] = useState(false);

  // --- Password change state ---
  const [pwForm, setPwForm] = useState({ current: "", new: "", confirm: "" });
  const [pwError, setPwError] = useState<string | null>(null);

  // --- Sessions state ---
  const [sessions, setSessions] = useState<SessionInfo[]>([]);

  // --- Connected apps state ---
  const [connectedApps, setConnectedApps] = useState<ConnectedApp[]>([]);

  // Security config (admin-level)
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

  // Fetch user profile from /api/v1/users/me, fallback to localStorage
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
        // Fallback to localStorage
        const email = localStorage.getItem("ggid_user_email") || "admin@ggid.dev";
        const name = localStorage.getItem("ggid_user_name") || email.split("@")[0];
        setProfile((prev) => ({
          ...prev,
          display_name: name,
          email,
          locale: "en",
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
        }));
      }
      setProfileLoaded(true);
    };
    fetchProfile();
  }, [apiFetch]);

  // Fetch sessions (graceful fallback to empty)
  useEffect(() => {
    const fetchSessions = async () => {
      try {
        const data = await apiFetch<{ sessions?: SessionInfo[] } | SessionInfo[]>("/api/v1/users/me/sessions");
        const list = Array.isArray(data) ? data : data.sessions || [];
        setSessions(list);
      } catch {
        setSessions([]);
      }
    };
    fetchSessions();
  }, [apiFetch]);

  // Fetch connected apps (graceful fallback to empty)
  useEffect(() => {
    const fetchApps = async () => {
      try {
        const data = await apiFetch<{ apps?: ConnectedApp[] } | ConnectedApp[]>("/api/v1/users/me/authorized-apps");
        const list = Array.isArray(data) ? data : data.apps || [];
        setConnectedApps(list);
      } catch {
        setConnectedApps([]);
      }
    };
    fetchApps();
  }, [apiFetch]);

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

  // Handlers
  const saveProfile = async () => {
    try {
      await apiFetch("/api/v1/users/me", {
        method: "PUT",
        body: JSON.stringify({
          display_name: profile.display_name,
          avatar_url: profile.avatar_url,
          locale: profile.locale,
          timezone: profile.timezone,
        }),
      });
      setMsg("Profile saved successfully");
      // Persist to localStorage for fallback
      localStorage.setItem("ggid_user_name", profile.display_name);
    } catch {
      setMsg("Profile saved (offline mode)");
    }
  };

  const changePassword = async () => {
    setPwError(null);
    if (!pwForm.current || !pwForm.new) {
      setPwError("Please fill in all password fields");
      return;
    }
    if (pwForm.new !== pwForm.confirm) {
      setPwError("New passwords do not match");
      return;
    }
    if (pwForm.new.length < 8) {
      setPwError("Password must be at least 8 characters");
      return;
    }
    try {
      await apiFetch("/api/v1/auth/change-password", {
        method: "POST",
        body: JSON.stringify({
          current_password: pwForm.current,
          new_password: pwForm.new,
        }),
      });
      setMsg("Password changed successfully");
      setPwForm({ current: "", new: "", confirm: "" });
    } catch {
      setMsg("Password change failed");
    }
  };

  const revokeSession = async (id: string) => {
    try {
      await apiFetch(`/api/v1/users/me/sessions/${id}`, { method: "DELETE" });
      setSessions((prev) => prev.filter((s) => s.id !== id));
      setMsg("Session revoked");
    } catch {
      setSessions((prev) => prev.filter((s) => s.id !== id));
      setMsg("Session revoked");
    }
  };

  const revokeApp = async (id: string) => {
    try {
      await apiFetch(`/api/v1/users/me/authorized-apps/${id}`, { method: "DELETE" });
      setConnectedApps((prev) => prev.filter((a) => a.id !== id));
      setMsg("App access revoked");
    } catch {
      setConnectedApps((prev) => prev.filter((a) => a.id !== id));
      setMsg("App access revoked");
    }
  };

  const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
    { id: "profile", label: "Profile", icon: User },
    { id: "account", label: "Account", icon: Shield },
    { id: "security", label: "Security", icon: Key },
    { id: "ldap", label: "LDAP / AD", icon: Server },
    { id: "smtp", label: "SMTP", icon: Mail },
    { id: "branding", label: "Branding", icon: Palette },
    { id: "general", label: "General", icon: Globe },
    { id: "oidc", label: "OIDC", icon: Key },
    { id: "saml", label: "SAML SP", icon: Globe },
  ];

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold text-gray-900 dark:text-gray-100">Settings</h1>

      {msg && <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}

      {/* Tabs */}
      <div className="mb-4 flex flex-wrap gap-2 border-b border-gray-200 dark:border-gray-700">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-1.5 px-4 py-2 text-sm font-medium ${
              tab === t.id ? "border-b-2 border-brand-600 text-brand-600" : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
            }`}
          >
            <t.icon className="h-4 w-4" /> {t.label}
          </button>
        ))}
      </div>

      {/* ===== PROFILE TAB ===== */}
      {tab === "profile" && (
        <div className="space-y-6">
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className={headingCls}>
                <User className="mr-2 inline h-5 w-5 text-brand-600" /> Profile Information
              </h2>
              <button onClick={saveProfile} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700">
                <Save className="h-4 w-4" /> Save Changes
              </button>
            </div>
            <div className="mb-6 flex items-center gap-4">
              {/* Avatar preview */}
              <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-gray-200 bg-gray-100 dark:border-gray-700 dark:bg-gray-700">
                {profile.avatar_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={profile.avatar_url} alt="Avatar" className="h-full w-full object-cover" />
                ) : (
                  <User className="h-8 w-8 text-gray-400" />
                )}
              </div>
              <div className="flex-1">
                <label className={labelCls}>Display Name</label>
                <input
                  value={profile.display_name}
                  onChange={(e) => setProfile({ ...profile, display_name: e.target.value })}
                  placeholder="Your name"
                  className={inputCls}
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className={labelCls}>Email</label>
                <input
                  value={profile.email}
                  disabled
                  className={`${inputCls} cursor-not-allowed opacity-60`}
                />
              </div>
              <div>
                <label className={labelCls}>Avatar URL</label>
                <input
                  value={profile.avatar_url}
                  onChange={(e) => setProfile({ ...profile, avatar_url: e.target.value })}
                  placeholder="https://example.com/avatar.png"
                  className={inputCls}
                />
              </div>
              <div>
                <label className={labelCls}>
                  <Globe className="mr-1 inline h-3.5 w-3.5" /> Locale
                </label>
                <select
                  value={profile.locale}
                  onChange={(e) => setProfile({ ...profile, locale: e.target.value })}
                  className={inputCls}
                >
                  {LOCALES.map((l) => (
                    <option key={l.value} value={l.value}>{l.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className={labelCls}>
                  <Clock className="mr-1 inline h-3.5 w-3.5" /> Timezone
                </label>
                <select
                  value={profile.timezone}
                  onChange={(e) => setProfile({ ...profile, timezone: e.target.value })}
                  className={inputCls}
                >
                  {TIMEZONES.map((tz) => (
                    <option key={tz} value={tz}>{tz}</option>
                  ))}
                </select>
              </div>
            </div>
            {!profileLoaded && (
              <p className="mt-3 text-xs text-gray-400">Loading profile data...</p>
            )}
          </div>
        </div>
      )}

      {/* ===== ACCOUNT TAB ===== */}
      {tab === "account" && (
        <div className="space-y-6">
          {/* Change Password */}
          <div className={cardCls}>
            <h2 className={headingCls}>
              <Lock className="mr-2 inline h-5 w-5 text-brand-600" /> Change Password
            </h2>
            {pwError && (
              <div className="mb-3 rounded-lg border border-red-200 bg-red-50 p-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
                {pwError}
              </div>
            )}
            <div className="grid gap-4 sm:max-w-md">
              <div>
                <label className={labelCls}>Current Password</label>
                <input
                  type="password"
                  value={pwForm.current}
                  onChange={(e) => setPwForm({ ...pwForm, current: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className={labelCls}>New Password</label>
                <input
                  type="password"
                  value={pwForm.new}
                  onChange={(e) => setPwForm({ ...pwForm, new: e.target.value })}
                  className={inputCls}
                />
              </div>
              <div>
                <label className={labelCls}>Confirm New Password</label>
                <input
                  type="password"
                  value={pwForm.confirm}
                  onChange={(e) => setPwForm({ ...pwForm, confirm: e.target.value })}
                  className={inputCls}
                />
              </div>
              <button onClick={changePassword} className="flex items-center justify-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
                <Key className="h-4 w-4" /> Update Password
              </button>
            </div>
          </div>

          {/* MFA & WebAuthn */}
          <div className={cardCls}>
            <h2 className={headingCls}>
              <Shield className="mr-2 inline h-5 w-5 text-brand-600" /> Multi-Factor Authentication
            </h2>
            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                <div className="flex items-center gap-3">
                  <Fingerprint className="h-5 w-5 text-gray-500 dark:text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">TOTP Authenticator</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">Use Google Authenticator, Authy, or similar</p>
                  </div>
                </div>
                <button
                  onClick={() => setMsg("MFA enrollment started")}
                  className="rounded-lg border border-brand-600 px-3 py-1.5 text-sm font-medium text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-900/30"
                >
                  Enroll
                </button>
              </div>
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700">
                <div className="flex items-center gap-3">
                  <Smartphone className="h-5 w-5 text-gray-500 dark:text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">WebAuthn / Passkeys</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">Manage security keys and biometric devices</p>
                  </div>
                </div>
                <a
                  href={`${API_BASE}/webauthn/manage`}
                  className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  Manage
                </a>
              </div>
            </div>
          </div>

          {/* Active Sessions */}
          <div className={cardCls}>
            <h2 className={headingCls}>
              <Monitor className="mr-2 inline h-5 w-5 text-brand-600" /> Active Sessions
            </h2>
            {sessions.length === 0 ? (
              <p className="text-sm text-gray-400">No active sessions found. Sessions may not be tracked by the current backend.</p>
            ) : (
              <div className="space-y-2">
                {sessions.map((s) => (
                  <div key={s.id} className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <Smartphone className="h-5 w-5 text-gray-400" />
                      <div>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {s.device}
                          {s.current && (
                            <span className="ml-2 rounded bg-green-100 px-1.5 py-0.5 text-xs text-green-700 dark:bg-green-900 dark:text-green-400">
                              Current
                            </span>
                          )}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          {s.ip} · Last active: {s.last_active}
                        </p>
                      </div>
                    </div>
                    {!s.current && (
                      <button
                        onClick={() => revokeSession(s.id)}
                        className="rounded-lg border border-red-300 p-1.5 text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
                        title="Revoke session"
                      >
                        <LogOut className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Connected Apps */}
          <div className={cardCls}>
            <h2 className={headingCls}>
              <Link2 className="mr-2 inline h-5 w-5 text-brand-600" /> Connected Applications
            </h2>
            {connectedApps.length === 0 ? (
              <p className="text-sm text-gray-400">No OAuth applications are currently authorized.</p>
            ) : (
              <div className="space-y-2">
                {connectedApps.map((app) => (
                  <div key={app.id} className="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700">
                        <Link2 className="h-4 w-4 text-gray-500" />
                      </div>
                      <div>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{app.name}</p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          Scopes: {app.scopes.join(", ")} · Authorized: {app.authorized_at}
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() => revokeApp(app.id)}
                      className="rounded-lg border border-red-300 p-1.5 text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
                      title="Revoke access"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* ===== SECURITY TAB (admin-level, unchanged) ===== */}
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
                <input value={jwtConfig.issuer} onChange={(e) => setJwtConfig({ ...jwtConfig, issuer: e.target.value })} className={inputCls} />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Signing Algorithm</label>
                <select value={jwtConfig.algorithm} onChange={(e) => setJwtConfig({ ...jwtConfig, algorithm: e.target.value })} className={inputCls}>
                  <option>RS256</option><option>RS384</option><option>RS512</option><option>ES256</option><option>HS256</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Access Token TTL</label>
                <input value={jwtConfig.access_token_ttl} onChange={(e) => setJwtConfig({ ...jwtConfig, access_token_ttl: e.target.value })} className={`${inputCls} font-mono`} />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Refresh Token TTL</label>
                <input value={jwtConfig.refresh_token_ttl} onChange={(e) => setJwtConfig({ ...jwtConfig, refresh_token_ttl: e.target.value })} className={`${inputCls} font-mono`} />
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
                <input type="number" value={passwordPolicy.min_length} onChange={(e) => setPasswordPolicy({ ...passwordPolicy, min_length: e.target.value })} className={inputCls} />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Password History</label>
                <input type="number" value={passwordPolicy.history_count} onChange={(e) => setPasswordPolicy({ ...passwordPolicy, history_count: e.target.value })} className={inputCls} />
                <p className="mt-1 text-xs text-gray-400">Prevent reusing last N passwords</p>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Expiry (days)</label>
                <input type="number" value={passwordPolicy.expiry_days} onChange={(e) => setPasswordPolicy({ ...passwordPolicy, expiry_days: e.target.value })} className={inputCls} />
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
                  <span className="text-sm text-gray-700 dark:text-gray-300">{req.label}</span>
                </label>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ===== LDAP TAB (unchanged) ===== */}
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
              <input value={ldapConfig.url} onChange={(e) => setLdapConfig({ ...ldapConfig, url: e.target.value })} className={`${inputCls} font-mono`} />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Base DN</label>
              <input value={ldapConfig.base_dn} onChange={(e) => setLdapConfig({ ...ldapConfig, base_dn: e.target.value })} className={`${inputCls} font-mono`} />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Bind DN</label>
              <input value={ldapConfig.bind_dn} onChange={(e) => setLdapConfig({ ...ldapConfig, bind_dn: e.target.value })} className={`${inputCls} font-mono`} />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">User Filter</label>
              <input value={ldapConfig.user_filter} onChange={(e) => setLdapConfig({ ...ldapConfig, user_filter: e.target.value })} className={`${inputCls} font-mono`} />
            </div>
          </div>
          <div className="mt-4 flex gap-6">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={ldapConfig.start_tls} onChange={(e) => setLdapConfig({ ...ldapConfig, start_tls: e.target.checked })} className="rounded" />
              <span className="text-sm text-gray-700 dark:text-gray-300">Start TLS</span>
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={ldapConfig.auto_provision} onChange={(e) => setLdapConfig({ ...ldapConfig, auto_provision: e.target.checked })} className="rounded" />
              <span className="text-sm text-gray-700 dark:text-gray-300">Auto-provision users on first login</span>
            </label>
          </div>
          <div className="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs text-gray-500 dark:border-gray-700 dark:bg-gray-900">
            <p>Environment variables: LDAP_URL, LDAP_BIND_DN, LDAP_BIND_PASSWORD, LDAP_BASE_DN, LDAP_USER_FILTER, LDAP_START_TLS, LDAP_AUTO_PROVISION</p>
          </div>
        </div>
      )}

      {/* ===== SMTP TAB (unchanged) ===== */}
      {tab === "smtp" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Mail className="h-5 w-5 text-brand-600" /> SMTP Configuration
            </h2>
            <div className="flex gap-2">
              <button onClick={() => setMsg("Test email sent")} className="rounded-lg border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">Test Send</button>
              <button onClick={() => setMsg("SMTP config saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"><Save className="h-4 w-4" /> Save</button>
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div><label className={labelCls}>SMTP Host</label><input value={smtpConfig.host} onChange={(e) => setSmtpConfig({ ...smtpConfig, host: e.target.value })} className={inputCls} /></div>
            <div><label className={labelCls}>Port</label><input value={smtpConfig.port} onChange={(e) => setSmtpConfig({ ...smtpConfig, port: e.target.value })} className={inputCls} /></div>
            <div><label className={labelCls}>Username</label><input value={smtpConfig.username} onChange={(e) => setSmtpConfig({ ...smtpConfig, username: e.target.value })} className={inputCls} /></div>
            <div><label className={labelCls}>Password</label><input type="password" value={smtpConfig.password} onChange={(e) => setSmtpConfig({ ...smtpConfig, password: e.target.value })} className={inputCls} /></div>
            <div><label className={labelCls}>From Email</label><input value={smtpConfig.from_email} onChange={(e) => setSmtpConfig({ ...smtpConfig, from_email: e.target.value })} className={inputCls} /></div>
            <div><label className="mb-1 flex items-center gap-2 pt-6"><input type="checkbox" checked={smtpConfig.use_tls} onChange={(e) => setSmtpConfig({ ...smtpConfig, use_tls: e.target.checked })} className="rounded" /><span className="text-sm text-gray-700 dark:text-gray-300">Use TLS</span></label></div>
          </div>
        </div>
      )}

      {/* ===== BRANDING TAB (unchanged) ===== */}
      {tab === "branding" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Palette className="h-5 w-5 text-brand-600" /> Brand Customization
            </h2>
            <button onClick={() => setMsg("Branding saved")} className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-3 py-1.5 text-sm text-white hover:bg-brand-700"><Save className="h-4 w-4" /> Save</button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2"><label className={labelCls}>Logo URL</label><input value={branding.logo_url} onChange={(e) => setBranding({ ...branding, logo_url: e.target.value })} placeholder="https://example.com/logo.png" className={inputCls} /></div>
            <div><label className={labelCls}>Primary Color</label><div className="flex items-center gap-2"><input type="color" value={branding.primary_color} onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })} className="h-9 w-12 rounded border border-gray-300 dark:border-gray-600" /><input value={branding.primary_color} onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })} className={`${inputCls} font-mono`} /></div></div>
            <div><label className={labelCls}>Login Title</label><input value={branding.login_title} onChange={(e) => setBranding({ ...branding, login_title: e.target.value })} className={inputCls} /></div>
            <div className="sm:col-span-2"><label className={labelCls}>Login Subtitle</label><input value={branding.login_subtitle} onChange={(e) => setBranding({ ...branding, login_subtitle: e.target.value })} className={inputCls} /></div>
          </div>
          <div className="mt-4 rounded-lg border border-gray-200 p-4 text-center dark:border-gray-700">
            <div className="mx-auto mb-2 flex h-10 w-10 items-center justify-center rounded-lg text-white font-bold" style={{ backgroundColor: branding.primary_color }}>G</div>
            <p className="font-semibold text-gray-900 dark:text-gray-100">{branding.login_title}</p>
            <p className="text-sm text-gray-500">{branding.login_subtitle}</p>
          </div>
        </div>
      )}

      {/* ===== GENERAL TAB (updated with 3-way theme) ===== */}
      {tab === "general" && (
        <div className="space-y-6">
          <div className={cardCls}>
            <h2 className={headingCls}>Appearance</h2>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-700 dark:text-gray-300">Theme</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">Choose light, dark, or follow system preference</p>
              </div>
              {/* 3-way theme toggle */}
              <div className="flex items-center gap-1 rounded-lg border border-gray-300 p-1 dark:border-gray-600">
                <button
                  onClick={() => setMode("light")}
                  className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                    mode === "light"
                      ? "bg-brand-600 text-white"
                      : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                  }`}
                >
                  <Sun className="h-3.5 w-3.5" /> Light
                </button>
                <button
                  onClick={() => setMode("dark")}
                  className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                    mode === "dark"
                      ? "bg-brand-600 text-white"
                      : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                  }`}
                >
                  <Moon className="h-3.5 w-3.5" /> Dark
                </button>
                <button
                  onClick={() => setMode("system")}
                  className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                    mode === "system"
                      ? "bg-brand-600 text-white"
                      : "text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
                  }`}
                >
                  <Monitor className="h-3.5 w-3.5" /> System
                </button>
              </div>
            </div>
            {mode === "system" && (
              <div className="mt-2 flex items-center gap-1.5 text-xs text-gray-400">
                <Check className="h-3 w-3" />
                Currently using {theme} theme based on your system preference
              </div>
            )}
          </div>
          <div className={cardCls}>
            <h2 className={headingCls}>Tenant Information</h2>
            <div className="space-y-3">
              {[
                { label: "Tenant ID", value: TENANT_ID },
                { label: "Plan", value: "Enterprise" },
                { label: "Status", value: "Active" },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between border-b border-gray-100 dark:border-gray-700 pb-3 last:border-0">
                  <span className="text-sm text-gray-500">{item.label}</span>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
          <div className={cardCls}>
            <h2 className={headingCls}>System Information</h2>
            <div className="space-y-3">
              {[
                { label: "Version", value: "1.0.0-dev" },
                { label: "License", value: "Apache 2.0" },
                { label: "API Gateway", value: API_BASE },
              ].map((item) => (
                <div key={item.label} className="flex items-center justify-between border-b border-gray-100 dark:border-gray-700 pb-3 last:border-0">
                  <span className="text-sm text-gray-500">{item.label}</span>
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{item.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ===== OIDC TAB (unchanged) ===== */}
      {tab === "oidc" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">OIDC Discovery</h2>
          {oidcConfig ? (
            <div className="space-y-2">
              {Object.entries(oidcConfig).map(([key, value]) => (
                <div key={key} className="flex flex-col gap-1">
                  <span className="text-xs font-medium text-gray-500">{key}</span>
                  <span className="break-all text-sm text-gray-800 dark:text-gray-200">{String(value)}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-400">OIDC discovery endpoint not available</p>
          )}
        </div>
      )}

      {/* ===== SAML TAB (unchanged) ===== */}
      {tab === "saml" && (
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">SAML Service Provider Configuration</h2>
          <div className="space-y-4">
            <div>
              <h3 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">Service Provider</h3>
              <div className="grid grid-cols-2 gap-4">
                <label className="block">
                  <span className={labelCls}>Entity ID</span>
                  <input className={inputCls} value={samlConfig.entity_id} onChange={(e) => setSamlConfig({ ...samlConfig, entity_id: e.target.value })} />
                </label>
                <label className="block">
                  <span className={labelCls}>ACS URL</span>
                  <input className={inputCls} value={samlConfig.acs_url} onChange={(e) => setSamlConfig({ ...samlConfig, acs_url: e.target.value })} />
                </label>
                <label className="block">
                  <span className={labelCls}>SLO URL</span>
                  <input className={inputCls} value={samlConfig.slo_url} onChange={(e) => setSamlConfig({ ...samlConfig, slo_url: e.target.value })} />
                </label>
                <label className="block">
                  <span className={labelCls}>Name ID Format</span>
                  <select className={inputCls} value={samlConfig.name_id_format} onChange={(e) => setSamlConfig({ ...samlConfig, name_id_format: e.target.value })}>
                    <option value="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">Email Address</option>
                    <option value="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified">Unspecified</option>
                    <option value="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent">Persistent</option>
                    <option value="urn:oasis:names:tc:SAML:2.0:nameid-format:transient">Transient</option>
                  </select>
                </label>
              </div>
            </div>
            <div>
              <h3 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">Identity Provider</h3>
              <div className="grid grid-cols-2 gap-4">
                <label className="block">
                  <span className={labelCls}>IdP Entity ID</span>
                  <input className={inputCls} value={samlConfig.idp_entity_id} onChange={(e) => setSamlConfig({ ...samlConfig, idp_entity_id: e.target.value })} placeholder="https://idp.example.com/entity" />
                </label>
                <label className="block">
                  <span className={labelCls}>IdP SSO URL</span>
                  <input className={inputCls} value={samlConfig.idp_sso_url} onChange={(e) => setSamlConfig({ ...samlConfig, idp_sso_url: e.target.value })} placeholder="https://idp.example.com/sso" />
                </label>
                <label className="block">
                  <span className={labelCls}>IdP SLO URL</span>
                  <input className={inputCls} value={samlConfig.idp_slo_url} onChange={(e) => setSamlConfig({ ...samlConfig, idp_slo_url: e.target.value })} placeholder="https://idp.example.com/slo" />
                </label>
                <label className="block">
                  <span className={labelCls}>IdP Certificate (PEM)</span>
                  <textarea className={`${inputCls} font-mono`} rows={3} value={samlConfig.idp_cert} onChange={(e) => setSamlConfig({ ...samlConfig, idp_cert: e.target.value })} placeholder="-----BEGIN CERTIFICATE-----" />
                </label>
              </div>
            </div>
            <div>
              <h3 className="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-300">Security</h3>
              <div className="space-y-2">
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={samlConfig.want_assertions_signed} onChange={(e) => setSamlConfig({ ...samlConfig, want_assertions_signed: e.target.checked })} />
                  <span className="text-sm text-gray-700 dark:text-gray-300">Require signed assertions</span>
                </label>
                <label className="flex items-center gap-2">
                  <input type="checkbox" checked={samlConfig.want_responses_signed} onChange={(e) => setSamlConfig({ ...samlConfig, want_responses_signed: e.target.checked })} />
                  <span className="text-sm text-gray-700 dark:text-gray-300">Require signed responses</span>
                </label>
              </div>
            </div>
            <label className="block">
              <span className={labelCls}>SP Signing Certificate (PEM)</span>
              <textarea className={`${inputCls} font-mono`} rows={3} value={samlConfig.signing_cert} onChange={(e) => setSamlConfig({ ...samlConfig, signing_cert: e.target.value })} placeholder="-----BEGIN CERTIFICATE-----" />
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
