"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Bell, Mail, MessageSquare, Shield, Clock, Save, Loader2,
  Check, AlertTriangle, FileText, Activity, Wrench,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

const CHANNELS = ["email", "slack", "teams", "webhook", "inApp"] as const;
type Channel = typeof CHANNELS[number];

const TYPES = [
  { id: "securityAlert", icon: Shield, color: "text-red-500", desc: "notificationPreferences.types.securityAlertDesc" },
  { id: "auditReport", icon: FileText, color: "text-blue-500", desc: "notificationPreferences.types.auditReportDesc" },
  { id: "complianceExpiry", icon: AlertTriangle, color: "text-orange-500", desc: "notificationPreferences.types.complianceExpiryDesc" },
  { id: "userActivity", icon: Activity, color: "text-green-500", desc: "notificationPreferences.types.userActivityDesc" },
  { id: "systemMaintenance", icon: Wrench, color: "text-purple-500", desc: "notificationPreferences.types.systemMaintenanceDesc" },
] as const;

const channelIcons: Record<string, typeof Mail> = { email: Mail, slack: MessageSquare, teams: MessageSquare, webhook: Bell, inApp: Bell };

export default function NotificationPreferencesPage() {
  const t = useTranslations();
  const [matrix, setMatrix] = useState<Record<string, Set<Channel>>>({});
  const [dndEnabled, setDndEnabled] = useState(false);
  const [dndStart, setDndStart] = useState("22:00");
  const [dndEnd, setDndEnd] = useState("08:00");
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  useEffect(() => {
    // Initialize defaults
    setMatrix({
      securityAlert: new Set<Channel>(["email", "inApp", "slack"]),
      auditReport: new Set<Channel>(["email"]),
      complianceExpiry: new Set<Channel>(["email", "inApp"]),
      userActivity: new Set<Channel>(["inApp"]),
      systemMaintenance: new Set<Channel>(["email", "inApp"]),
    });
  }, []);

  const toggle = (typeId: string, channel: Channel) => {
    setMatrix((prev) => {
      const next = { ...prev };
      const set = new Set(next[typeId] || []);
      if (set.has(channel)) set.delete(channel); else set.add(channel);
      next[typeId] = set;
      return next;
    });
  };

  const save = async () => {
    setSaving(true);
    try {
      await fetch(`${API_BASE}/api/v1/identity/notifications/preferences`, {
        method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ matrix: Object.fromEntries(Object.entries(matrix).map(([k, v]) => [k, [...v]])), dnd: { enabled: dndEnabled, start: dndStart, end: dndEnd } }),
      });
    } catch { /* ok */ }
    setSaving(false);
    setMsg(t("notificationPreferences.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Bell className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("notificationPreferences.title")}</h1>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">{t("notificationPreferences.description")}</p>
        </div>

        {/* Matrix Table */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 mb-4">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 dark:border-gray-800">
                  <th className="py-3 px-3 text-left font-medium text-gray-600 dark:text-gray-400 sticky left-0 bg-white dark:bg-gray-900">
                    {t("notificationPreferences.description")}
                  </th>
                  {CHANNELS.map((ch) => {
                    const Icon = channelIcons[ch];
                    return (
                      <th key={ch} className="py-3 px-2 text-center font-medium text-gray-600 dark:text-gray-400 min-w-[80px]">
                        <div className="flex flex-col items-center gap-1">
                          <Icon className="w-4 h-4 text-gray-400" />
                          <span className="text-xs">{t(`notificationPreferences.channels.${ch}`)}</span>
                        </div>
                      </th>
                    );
                  })}
                </tr>
              </thead>
              <tbody>
                {TYPES.map((type) => {
                  const Icon = type.icon;
                  const enabled = matrix[type.id] || new Set<Channel>();
                  return (
                    <tr key={type.id} className="border-b border-gray-100 dark:border-gray-800/50">
                      <td className="py-3 px-3 sticky left-0 bg-white dark:bg-gray-900">
                        <div className="flex items-center gap-2">
                          <Icon className={`w-4 h-4 ${type.color}`} />
                          <div>
                            <div className="font-medium text-gray-900 dark:text-white">{t(`notificationPreferences.types.${type.id}`)}</div>
                            <div className="text-xs text-gray-400">{t(type.desc)}</div>
                          </div>
                        </div>
                      </td>
                      {CHANNELS.map((ch) => {
                        const isChecked = enabled.has(ch);
                        return (
                          <td key={ch} className="py-3 px-2 text-center">
                            <button onClick={() => toggle(type.id, ch)}
                              className={`w-9 h-5 rounded-full transition-colors ${isChecked ? "bg-blue-600" : "bg-gray-200 dark:bg-gray-700"}`}>
                              <span className={`block w-4 h-4 bg-white rounded-full transition-transform ${isChecked ? "translate-x-[18px]" : "translate-x-0.5"}`} />
                            </button>
                          </td>
                        );
                      })}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>

        {/* DND */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 mb-4">
          <div className="flex items-center justify-between mb-3">
            <div>
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                <Clock className="w-4 h-4 text-purple-500" />{t("notificationPreferences.dnd.title")}
              </h3>
              <p className="text-xs text-gray-400">{t("notificationPreferences.dnd.desc")}</p>
            </div>
            <button onClick={() => setDndEnabled(!dndEnabled)}
              className={`relative w-10 h-6 rounded-full transition-colors ${dndEnabled ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
              <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${dndEnabled ? "translate-x-4" : ""}`} />
            </button>
          </div>
          {dndEnabled && (
            <div className="flex items-center gap-3">
              <div>
                <label className="text-xs text-gray-500 mr-2">{t("notificationPreferences.dnd.startTime")}</label>
                <input type="time" value={dndStart} onChange={(e) => setDndStart(e.target.value)}
                  className="px-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
              </div>
              <span className="text-gray-400">→</span>
              <div>
                <label className="text-xs text-gray-500 mr-2">{t("notificationPreferences.dnd.endTime")}</label>
                <input type="time" value={dndEnd} onChange={(e) => setDndEnd(e.target.value)}
                  className="px-3 py-1.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
              </div>
            </div>
          )}
        </div>

        {msg && (
          <div className="flex items-center gap-2 px-4 py-2 mb-4 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
            <Check className="w-4 h-4" />{msg}
          </div>
        )}

        <button onClick={save} disabled={saving}
          className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
          {t("notificationPreferences.save")}
        </button>
      </div>
    </div>
  );
}
