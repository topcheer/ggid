"use client";
import { useState, useEffect, useCallback } from "react";
import { Shield, ShieldCheck, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface SuperAdmin { user_id: string; username: string; added_at: string; added_by: string; }
interface AdminActivity { id: string; admin: string; action: string; target: string; timestamp: string; }
interface AdminConfig { super_admins: SuperAdmin[]; permissions: Record<string, boolean>; restricted_actions: string[]; require_mfa: boolean; activity_log: AdminActivity[]; }

const permLabels: Record<string, string> = { view_users: "adminSettings.viewUsers", manage_users: "adminSettings.manageUsers", view_policies: "adminSettings.viewPolicies", manage_policies: "adminSettings.managePolicies", view_audit: "adminSettings.viewAudit", manage_audit: "adminSettings.manageAudit", view_oauth: "adminSettings.viewOauth", manage_oauth: "adminSettings.manageOauth" };

export default function AdminSettingsPage() {
  const [config, setConfig] = useState<AdminConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const t = useTranslations();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/admin/settings", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setConfig(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  if (!config) return <p className="text-sm text-gray-500 text-center py-8">{t("adminSettings.loading")}</p>;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-red-500" /> Admin Settings</h1><p className="text-sm text-gray-500 mt-1">{t("adminSettings.subtitle")}</p></div>

      <div className="flex items-center justify-between rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center gap-3"><ShieldCheck className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">{t("adminSettings.requireMfa")}</span><p className="text-xs text-gray-400">{t("adminSettings.requireMfaDesc")}</p></div></div><label className="relative inline-flex items-center cursor-pointer"><input type="checkbox" checked={config.require_mfa} onChange={(e) => setConfig({ ...config, require_mfa: e.target.checked })} className="sr-only peer" /><div className="w-11 h-6 bg-gray-200 rounded-full peer dark:bg-gray-700 peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-5" /></label></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("adminSettings.superAdmins")} ({config.super_admins.length})</h3><div className="space-y-1">{config.super_admins.map((a) => (<div key={a.user_id} className="flex items-center justify-between text-sm py-1"><div><span className="font-medium">{a.username}</span><p className="text-xs text-gray-400 font-mono">{a.user_id}</p></div><div className="text-right"><span className="text-xs text-gray-500">{t("adminSettings.added")} {a.added_at}</span><p className="text-xs text-gray-400">by {a.added_by}</p></div></div>))}</div></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("adminSettings.permissionsMatrix")}</h3><div className="grid grid-cols-2 md:grid-cols-4 gap-2">{Object.entries(permLabels).map(([key, label]) => (<label key={key} className="flex items-center gap-2 text-sm rounded-lg border dark:border-gray-700 p-2"><input type="checkbox" checked={config.permissions[key] || false} onChange={(e) => setConfig({ ...config, permissions: { ...config.permissions, [key]: e.target.checked } })} className="rounded" /> {t(label)}</label>))}</div></div>

      {config.restricted_actions.length > 0 && (<div className="rounded-lg border border-red-200 dark:border-red-900 p-4"><h3 className="text-sm font-semibold mb-2 text-red-600">{t("adminSettings.restrictedActions")}</h3><div className="space-y-1">{config.restricted_actions.map((a, i) => (<div key={i} className="text-sm font-mono text-xs text-red-600">{a}</div>))}</div></div>)}

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><Activity className="w-4 h-4 text-gray-400" /> Admin Activity Log</h3><div className="space-y-1">{config.activity_log.slice(0, 10).map((a) => (<div key={a.id} className="flex items-center justify-between text-sm py-1"><div><span className="font-medium">{a.admin}</span><span className="text-gray-500"> {a.action}</span><span className="text-xs text-gray-400"> {a.target}</span></div><span className="text-xs text-gray-400">{a.timestamp}</span></div>))}{config.activity_log.length === 0 && <p className="text-xs text-gray-500">{t("adminSettings.noActivity")}</p>}</div></div>
    </div>
  );
}
