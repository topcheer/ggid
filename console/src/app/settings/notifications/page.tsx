"use client";
import { useState, useEffect, useCallback } from "react";
import { Bell, Loader2, AlertCircle, CheckCircle2, Save } from "lucide-react";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

interface NotificationPrefs { email_enabled: boolean; sms_enabled: boolean; push_enabled: boolean; security_alerts: boolean; audit_reports: boolean; system_updates: boolean; }

export default function NotificationPreferencesPage() {
  usePageTitle("Notification Preferences");
  const [prefs, setPrefs] = useState<NotificationPrefs>({ email_enabled: true, sms_enabled: false, push_enabled: false, security_alerts: true, audit_reports: false, system_updates: true });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/notification/preferences`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setPrefs({ ...prefs, ...d }); }
    } catch { /* use defaults */ }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleSave = async () => {
    setSaving(true); setError(""); setSuccess(false);
    try {
      const res = await fetch(`${API_BASE}/api/v1/notification/preferences`, { method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() }, body: JSON.stringify(prefs) });
      if (res.ok) { setSuccess(true); setTimeout(() => setSuccess(false), 3000); } else { setError("Failed to save preferences"); }
    } catch { setError("Network error"); }
    setSaving(false);
  };

  const toggle = (key: keyof NotificationPrefs) => setPrefs(prev => ({ ...prev, [key]: !prev[key] }));

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  const channels: { key: keyof NotificationPrefs; label: string; desc: string }[] = [
    { key: "email_enabled", label: "Email Notifications", desc: "Receive notifications via email" },
    { key: "sms_enabled", label: "SMS Notifications", desc: "Receive notifications via SMS" },
    { key: "push_enabled", label: "Push Notifications", desc: "Receive push notifications on your device" },
  ];
  const categories: { key: keyof NotificationPrefs; label: string; desc: string }[] = [
    { key: "security_alerts", label: "Security Alerts", desc: "Critical security events (failed logins, MFA changes)" },
    { key: "audit_reports", label: "Audit Reports", desc: "Weekly audit summary reports" },
    { key: "system_updates", label: "System Updates", desc: "System maintenance and updates notifications" },
  ];

  return (
    <div className="mx-auto max-w-2xl p-6">
      <h1 className="mb-1 text-2xl font-bold text-gray-900 dark:text-white dark:text-white">Notification Preferences</h1>
      <p className="mb-6 text-sm text-gray-500">Choose how and when you want to receive notifications.</p>

      {error && <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4 shrink-0" /> {error}</div>}
      {success && <div className="mb-4 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950"><CheckCircle2 className="h-4 w-4 shrink-0" /> Preferences saved.</div>}

      <div className="mb-6 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 dark:border-gray-800 dark:bg-gray-900">
        <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">Channels</h3>
        <div className="space-y-3">
          {channels.map(c => (
            <div key={c.key} className="flex items-center justify-between">
              <div><p className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{c.label}</p><p className="text-xs text-gray-500">{c.desc}</p></div>
              <button onClick={() => toggle(c.key)} className={`relative h-6 w-11 rounded-full transition-colors ${prefs[c.key] ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-700"}`}>
                <span className={`absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white transition-transform ${prefs[c.key] ? "translate-x-5" : ""}`}></span>
              </button>
            </div>
          ))}
        </div>
      </div>

      <div className="mb-6 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 dark:border-gray-800 dark:bg-gray-900">
        <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">Categories</h3>
        <div className="space-y-3">
          {categories.map(c => (
            <div key={c.key} className="flex items-center justify-between">
              <div><p className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{c.label}</p><p className="text-xs text-gray-500">{c.desc}</p></div>
              <button onClick={() => toggle(c.key)} className={`relative h-6 w-11 rounded-full transition-colors ${prefs[c.key] ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-700"}`}>
                <span className={`absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white transition-transform ${prefs[c.key] ? "translate-x-5" : ""}`}></span>
              </button>
            </div>
          ))}
        </div>
      </div>

      <div className="flex justify-end">
        <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save Preferences
        </button>
      </div>
    </div>
  );
}
