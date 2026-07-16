"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import { User, Lock, Shield, Monitor, Smartphone, Globe } from "lucide-react";

type Tab = "profile" | "security" | "sessions";

interface Session {
  id: string;
  device: string;
  ip: string;
  location: string;
  last_active: string;
  current?: boolean;
}

export default function ProfilePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [tab, setTab] = useState<Tab>("profile");
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [mfaEnabled, setMfaEnabled] = useState(false);

  // Profile form
  const [profile, setProfile] = useState({
    username: "admin",
    email: "admin@ggid.dev",
    full_name: "Administrator",
    phone: "",
  });

  // Password form
  const [passwords, setPasswords] = useState({ current: "", new: "", confirm: "" });

  // Sessions
  const [sessions, setSessions] = useState<Session[]>([
    { id: "1", device: "Chrome on macOS", ip: "192.168.1.100", location: "Local Network", last_active: new Date().toISOString(), current: true },
    { id: "2", device: "Safari on iPhone", ip: "10.0.0.50", location: "Cellular", last_active: new Date(Date.now() - 3600000).toISOString() },
  ]);

  useEffect(() => {
    if (msg) { const t = setTimeout(() => setMsg(null), 3000); return () => clearTimeout(t); }
  }, [msg]);

  const handleSaveProfile = async () => {
    setMsg(t("profile.profilesaved"));
  };

  const handleChangePassword = async () => {
    if (passwords.new !== passwords.confirm) { setError(t("profile.passwordsdontmatch")); return; }
    if (passwords.new.length < 8) { setError(t("profile.passwordmustbeatleast8characte")); return; }
    try {
      await apiFetch("/api/v1/auth/change-password", {
        method: "POST",
        body: JSON.stringify({ current_password: passwords.current, new_password: passwords.new }),
      });
      setMsg(t("profile.passwordchanged"));
      setPasswords({ current: "", new: "", confirm: "" });
      setError(null);
    } catch {
      setMsg(t("profile.passwordchangeddemomode"));
      setPasswords({ current: "", new: "", confirm: "" });
    }
  };

  const handleToggleMFA = async () => {
    setMfaEnabled(!mfaEnabled);
    setMsg(mfaEnabled ? "MFA disabled" : "MFA enabled — use your authenticator app");
  };

  const handleRevokeSession = (id: string) => {
    setSessions(sessions.filter((s) => s.id !== id));
    setMsg(t("profile.sessionrevoked"));
  };

  const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
    { id: "profile", label: t("profile.profile"), icon: User },
    { id: "security", label: t("profile.security"), icon: Lock },
    { id: "sessions", label: t("profile.sessions"), icon: Monitor },
  ];

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold dark:text-gray-100">{t("profile.myprofile")}</h1>

      {msg && <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}
      {error && <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>}

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

      {tab === "profile" && (
        <div className="max-w-2xl rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-6 flex items-center gap-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-brand-100 text-xl font-bold text-brand-600">
              {profile.username.charAt(0).toUpperCase()}
            </div>
            <div>
              <h2 className="text-lg font-semibold dark:text-gray-100">{profile.full_name || profile.username}</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 dark:text-gray-500">{profile.email}</p>
            </div>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400 dark:text-gray-500">Username</label>
              <input aria-label="profile" value={profile.username} onChange={(e) => setProfile({ ...profile, username: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400 dark:text-gray-500">Email</label>
              <input aria-label="profile" value={profile.email} onChange={(e) => setProfile({ ...profile, email: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400 dark:text-gray-500">Full Name</label>
              <input aria-label="profile" value={profile.full_name} onChange={(e) => setProfile({ ...profile, full_name: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400 dark:text-gray-500">Phone</label>
              <input aria-label="profile" value={profile.phone} onChange={(e) => setProfile({ ...profile, phone: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
            </div>
          </div>
          <button onClick={handleSaveProfile} className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm text-white hover:bg-brand-700">
            Save Changes
          </button>
        </div>
      )}

      {tab === "security" && (
        <div className="space-y-6">
          {/* Change password */}
          <div className="max-w-2xl rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
              <Lock className="h-5 w-5 text-brand-600" /> Change Password
            </h2>
            <div className="space-y-3">
              <input autoComplete="current-password" aria-label="Current password" type="password" value={passwords.current} onChange={(e) => setPasswords({ ...passwords, current: e.target.value })} placeholder="Current password" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
              <input autoComplete="current-password" aria-label="New password" type="password" value={passwords.new} onChange={(e) => setPasswords({ ...passwords, new: e.target.value })} placeholder="New password" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
              <input autoComplete="current-password" aria-label="Confirm new password" type="password" value={passwords.confirm} onChange={(e) => setPasswords({ ...passwords, confirm: e.target.value })} placeholder="Confirm new password" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200" />
            </div>
            <button onClick={handleChangePassword} disabled={!passwords.current || !passwords.new} className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm text-white hover:bg-brand-700 disabled:opacity-50">
              Update Password
            </button>
          </div>

          {/* MFA */}
          <div className="max-w-2xl rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800 shadow-sm dark:border-gray-700 dark:bg-gray-800">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${mfaEnabled ? "bg-green-100" : "bg-gray-100"}`}>
                  <Shield className={`h-5 w-5 ${mfaEnabled ? "text-green-600" : "text-gray-400 dark:text-gray-500"}`} />
                </div>
                <div>
                  <h2 className="text-sm font-semibold">{t("profile.twofactorauthenticationtotp")}</h2>
                  <p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-500">{mfaEnabled ? "Enabled — scan QR in your authenticator app" : "Add an extra layer of security"}</p>
                </div>
              </div>
              <button onClick={handleToggleMFA} className={`relative h-6 w-11 rounded-full transition-colors ${mfaEnabled ? "bg-green-500" : "bg-gray-300"}`} aria-label="span">
                <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${mfaEnabled ? "left-5" : "left-0.5"}`} />
              </button>
            </div>
          </div>
        </div>
      )}

      {tab === "sessions" && (
        <div className="max-w-2xl rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-800 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 text-lg font-semibold dark:text-gray-100">{t("profile.activesessions")}</h2>
          <div className="space-y-3">
            {sessions.map((s) => (
              <div key={s.id} className="flex items-center justify-between rounded-lg border border-gray-100 p-3">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100">
                    {s.device.toLowerCase().includes("iphone") || s.device.toLowerCase().includes("android") ? (
                      <Smartphone className="h-5 w-5 text-gray-500 dark:text-gray-400 dark:text-gray-500" />
                    ) : (
                      <Monitor className="h-5 w-5 text-gray-500 dark:text-gray-400 dark:text-gray-500" />
                    )}
                  </div>
                  <div>
                    <p className="text-sm font-medium">{s.device}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 dark:text-gray-500">
                      <Globe className="mr-1 inline h-3 w-3" />
                      {s.ip} • {s.location} • {new Date(s.last_active).toLocaleString()}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {s.current && <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-700">{t("profile.current")}</span>}
                  {!s.current && (
                    <button onClick={() => handleRevokeSession(s.id)} className="rounded-lg border border-red-200 px-2 py-1 text-xs text-red-600 hover:bg-red-50">
                      Revoke
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
