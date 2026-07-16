"use client";
import { useEffect, useState } from "react";
import { useAccessRequestLifecycleConfig, AccessRequestLifecycleConfig, LifecycleStage, AutoApprovalRule } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function AccessRequestLifecycleConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useAccessRequestLifecycleConfig();
  const [form, setForm] = useState<AccessRequestLifecycleConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("backend3.accessRequestLifecycle.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold"> {t("backend3.accessRequestLifecycle.title")}</h1>
      <p className="text-gray-600">Configure access request stages, SLAs, and auto-approval rules.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("backend3.accessRequestLifecycle.lifecycleStages")}</h2>
        <div className="flex flex-wrap items-center gap-2">
          {form.stages.map((s: LifecycleStage, i: number) => (
            <div key={i} className="flex items-center gap-2">
              <span className="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm font-medium">{s.stage}</span>
              <span className="text-xs text-gray-500">SLA: {s.sla_hours}h</span>
              {i < form.stages.length - 1 && <span className="text-gray-300">-&gt;</span>}
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("backend3.accessRequestLifecycle.globalLimits")}</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">{t("backend3.accessRequestLifecycle.maxDuration")}</label><input type="number" value={form.max_duration_days} onChange={(e) => setForm({ ...form, max_duration_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Renewal Reminder (days before)</label><input type="number" value={form.renewal_reminder_days} onChange={(e) => setForm({ ...form, renewal_reminder_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Auto-Approval Rules</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("backend3.accessRequestLifecycle.condition")}</th><th scope="col">{t("backend3.accessRequestLifecycle.targetRole")}</th><th>{t("backend3.accessRequestLifecycle.maxDuration")}</th></tr></thead><tbody>
          {form.auto_approval_rules.map((r: AutoApprovalRule, i: number) => (
            <tr key={i} className="border-b"><td className="py-2">{r.condition}</td><td className="font-medium">{r.target_role}</td><td>{r.max_duration_days} days</td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
