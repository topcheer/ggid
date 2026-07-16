"use client";

import { useState, useEffect, useCallback } from "react";
import { Search, Save, Sliders, Bell, Palette, Code } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface UserPreferences {
  user_id: string;
  locale: string;
  timezone: string;
  theme: "light" | "dark" | "system";
  notification_channels: {
    email: boolean;
    push: boolean;
    sms: boolean;
    webhook: boolean;
  };
  dashboard_layout: string;
}

const locales = ["en", "zh", "es", "fr", "de", "ja", "ko", "pt", "ru"];
const timezones = ["UTC", "America/New_York", "America/Los_Angeles", "Europe/London", "Europe/Paris", "Asia/Shanghai", "Asia/Tokyo", "Australia/Sydney"];

export default function UserPreferencesPage() {
  const t = useTranslations();

  const [search, setSearch] = useState("");
  const [prefs, setPrefs] = useState<UserPreferences | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [layoutError, setLayoutError] = useState(false);

  const fetchPrefs = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/preferences?user=${encodeURIComponent(user)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setPrefs({
          user_id: data.user_id || user,
          locale: data.locale || "en",
          timezone: data.timezone || "UTC",
          theme: data.theme || "system",
          notification_channels: data.notification_channels || { email: true, push: false, sms: false, webhook: false },
          dashboard_layout: typeof data.dashboard_layout === "string" ? data.dashboard_layout : JSON.stringify(data.dashboard_layout || {}, null, 2),
        });
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!search) return;
    fetchPrefs(search);
  }, [search, fetchPrefs]);

  const save = async () => {
    if (!prefs) return;
    setSaving(true);
    try {
      await fetch(`/api/v1/identity/preferences/${prefs.user_id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(prefs),
      });
    } catch {
      /* noop */
    } finally {
      setSaving(false);
    }
  };

  const validateLayout = (val: string) => {
    setPrefs(prefs ? { ...prefs, dashboard_layout: val } : null);
    try { JSON.parse(val); setLayoutError(false); } catch { setLayoutError(true); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Sliders className="w-6 h-6 text-blue-500" /> {t("userPreferences.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Manage locale, timezone, theme, notification channels, and dashboard layout.</p>
      </div>

      {/* User search */}
      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input aria-label="Search by username or user ID..." type="text" placeholder="Search by username or user ID..." value={search} onChange={(e) => setSearch(e.target.value)} className="w-full pl-9 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
      </div>

      {loading && <p className="text-sm text-gray-500">Loading...</p>}

      {prefs && (
        <div className="space-y-4">
          {/* Locale & Timezone */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3 flex items-center gap-2"><Palette className="w-4 h-4" /> Locale & Display</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="text-sm font-medium">Locale</label>
                <select value={prefs.locale} onChange={(e) => setPrefs({ ...prefs, locale: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">
                  {locales.map((l) => <option key={l} value={l}>{l}</option>)}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">Timezone</label>
                <select value={prefs.timezone} onChange={(e) => setPrefs({ ...prefs, timezone: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">
                  {timezones.map((tz) => <option key={tz} value={tz}>{tz}</option>)}
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">Theme</label>
                <select value={prefs.theme} onChange={(e) => setPrefs({ ...prefs, theme: e.target.value as UserPreferences["theme"] })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">
                  <option value="light">Light</option>
                  <option value="dark">Dark</option>
                  <option value="system">System</option>
                </select>
              </div>
            </div>
          </div>

          {/* Notification channels */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3 flex items-center gap-2"><Bell className="w-4 h-4" /> Notification Channels</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              {(["email", "push", "sms", "webhook"] as const).map((ch) => (
                <label key={ch} className="flex items-center gap-2 text-sm cursor-pointer">
                  <input type="checkbox" checked={prefs.notification_channels[ch]} onChange={(e) => setPrefs({ ...prefs, notification_channels: { ...prefs.notification_channels, [ch]: e.target.checked } })} className="rounded" />
                  <span className="capitalize">{ch}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Dashboard layout JSON */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-3 flex items-center gap-2"><Code className="w-4 h-4" /> Dashboard Layout</h3>
            <textarea aria-label="Text input" value={prefs.dashboard_layout} onChange={(e) => validateLayout(e.target.value)} rows={8} spellCheck={false} className={`w-full px-3 py-2 rounded-lg border text-sm font-mono dark:bg-gray-800 ${layoutError ? "border-red-400" : "dark:border-gray-700"}`} />
            {layoutError && <p className="text-xs text-red-500 mt-1">Invalid JSON</p>}
          </div>

          <div className="flex justify-end">
            <button onClick={save} disabled={saving || layoutError} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save Preferences"}</button>
          </div>
        </div>
      )}

      {!prefs && !loading && search && <p className="text-sm text-gray-500">No preferences found.</p>}
      {!prefs && !search && <p className="text-sm text-gray-500 text-center py-8">Search for a user to manage their preferences.</p>}
    </div>
  );
}
